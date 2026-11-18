#!/usr/bin/env python3
"""One fixed read-only-backed disposable pending-app recovery VM with bounded supervision.

No work without --run. Fixed disposable predecessor and inputs only; no installed host service,
engine, runner socket, keyring or user pier is opened or changed.
"""
import argparse
import fcntl
import importlib.util
import json
import os
from pathlib import Path
import re
import shutil
import signal
import subprocess
import sys
import time
import uuid

ROOT = Path(__file__).resolve().parents[3]
spec = importlib.util.spec_from_file_location('issues_budget', ROOT/'scripts/qualification/issues-recovery/run.py')
budget = importlib.util.module_from_spec(spec)
spec.loader.exec_module(budget)
util = budget.util
GIB = 1024**3
# Share the fixed ended task fixture read-only. Recover its copied pending app
# through supported lifecycle and audited APIs; no bootstrap or full input copies.
# The completed first source used218,238,976 allocated root bytes (no fresh
# bootstrap). Keep more than3x that measured space behind a hard file bound
# while reserving every input/log/report byte under the aggregate cap.
SCRATCH = 896*1024**2
HOST_BUILD_RESERVE = 64*1024**2
ROOT_FILE_MAX = 768*1024**2
VM_DEADLINE_SECONDS = 6100
SUPERVISOR_DEADLINE_SECONDS = 6300
PREDECESSOR = Path('/tmp/omabit-issues-app-vm-05/root.qcow2')
PREDECESSOR_SHA = 'a23f10ddc89657489d8e55eaf5e4362603848366a447dbf9f51ce192837a08f9'


def artifact_change_allowed(path):
    relative = Path(path)
    if relative.is_absolute() or '..' in relative.parts or path.endswith('.go'):
        return False
    return path.startswith('docs/') or path in {
        'scripts/qualification/grubbery-runtime-cow/run.py',
        'scripts/qualification/grubbery-runtime-cow/guest.py',
        'scripts/qualification/grubbery-runtime-cow/test_guards.py',
        'scripts/qualification/grubbery-runtime-cow/README.md',
    }


def ordinary(path, directory=False):
    if path.is_symlink() or path.resolve() != path.absolute() or not (path.is_dir() if directory else path.is_file()):
        raise RuntimeError('ordinary physical input required')
    return path


def terminate(number, frame):
    raise InterruptedError('product VM supervisor terminated: '+str(number))


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ('portable', 'candidate', 'rootless', 'runtime', 'helper', 'output'):
        parser.add_argument('--'+name, type=Path)
    parser.add_argument('--storage-budget-gib', type=int, choices=(40, 48), default=40)
    parser.add_argument('--artifact-source', help='exact frozen artifact commit when only harness/documentation changed')
    parser.add_argument('--run', action='store_true')
    parser.add_argument('--worker', action='store_true', help=argparse.SUPPRESS)
    args = parser.parse_args()
    if not args.run:
        parser.print_help()
        return
    names = ('portable', 'candidate', 'rootless', 'runtime', 'helper', 'output')
    if os.geteuid() == 0 or any(getattr(args, name) is None for name in names):
        parser.error('ordinary owner and every explicit input required')
    out = args.output
    if out.parent != Path('/tmp') or out.exists() or out.is_symlink() or not re.fullmatch('omabit-issues-runtime-cow-[0-9]{2}', out.name):
        parser.error('fresh physical /tmp/omabit-issues-runtime-cow-NN required; never resume a capsule')
    if subprocess.check_output(['git', 'status', '--porcelain'], cwd=ROOT):
        raise RuntimeError('freeze clean matching source before qualification')
    harness_commit = subprocess.check_output(['git', 'rev-parse', 'HEAD'], cwd=ROOT, text=True).strip()
    commit = args.artifact_source or harness_commit
    if not re.fullmatch('[0-9a-f]{40}',commit):
        raise RuntimeError('exact artifact source commit required')
    subprocess.run(['git','merge-base','--is-ancestor',commit,harness_commit],cwd=ROOT,check=True)
    changed = subprocess.check_output(['git','diff','--name-only',commit],cwd=ROOT,text=True).splitlines()
    if any(not artifact_change_allowed(path) for path in changed):
        raise RuntimeError('runtime or embedded inputs changed after the frozen artifacts')
    portable = ordinary(args.portable, True)
    meta = json.loads(ordinary(portable/'bundle.json').read_text())
    for name, pin in meta['sha256'].items():
        relative = Path(name)
        if relative.is_absolute() or '..' in relative.parts or util.digest(ordinary(portable/relative)) != pin:
            raise RuntimeError('pristine immutable input differs')
    if meta['sha256']['base.qcow2'] != 'e778801a4d1366004bbb0700b3920c2bc9cf4d1b0c07137357adfbabb1b7ec32':
        raise RuntimeError('wrong pristine not-onboarded base')
    if util.digest(ordinary(PREDECESSOR)) != PREDECESSOR_SHA:
        raise RuntimeError('fixed task-only predecessor changed')
    prior = json.loads((PREDECESSOR.parent/'host.json').read_text())
    if prior.get('exit') != 0 or prior['scenario']['source_commit'] != '5eeae9a2c9c15c764aaa6534b6a9fd6d745f6bb4' or json.loads((PREDECESSOR.parent/'result.json').read_text()).get('passed') is not False:
        raise RuntimeError('wrong ended predecessor result')
    for unit in ('omabit-omabit-issues-app-vm-05.scope', 'omabit-issues-app-vm-supervisor-0fd9e321b4814441b9fbc72c7814692a.service','omabit-issues-evidence-appvm05-01.scope'):
        state = subprocess.run(['systemctl','--user','is-active',unit],capture_output=True,text=True,timeout=10).stdout.strip()
        if state not in ('inactive','failed','unknown'):
            raise RuntimeError('original owned scope still active')
    builds = {}
    for name, path in [(name, args.candidate/name) for name in ('omabit', 'urbit-collector', 'urbit-runner')]+[('managed.test', args.helper)]:
        info = subprocess.check_output(['go', 'version', '-m', ordinary(path)], text=True)
        if name == 'managed.test':
            identity = subprocess.check_output([path, '-test.run=^TestIssuesStopVMBuildIdentity$', '-test.v'], text=True)
            if 'OMABIT_ISSUES_HELPER_SOURCE '+commit not in identity:
                raise RuntimeError('helper source pin does not match clean source')
        elif '\tvcs.revision='+commit not in info or '\tvcs.modified=false' not in info:
            raise RuntimeError('candidate source identity differs')
        builds[name] = {'sha256': util.digest(path), 'bytes': path.stat().st_size, 'build_info': info}
    runtime = ordinary(args.runtime, True)
    if util.digest(ordinary(runtime/'boot.pill')) != budget.PILL or util.digest(ordinary(runtime/'runtime.oci.tar')) != budget.OCI:
        raise RuntimeError('frozen runtime/pill differs')
    ordinary(args.rootless, True)
    if not args.worker:
        # This unit owns only the host-side supervisor. The fresh QEMU scope is
        # separate and has its own stronger memory/CPU/no-swap boundary.
        budget.charge(SCRATCH+HOST_BUILD_RESERVE, args.storage_budget_gib*GIB)
        token = uuid.uuid4().hex
        unit = 'omabit-issues-runtime-cow-supervisor-'+token
        log = Path('/tmp')/(unit+'.log')
        with log.open('x'):
            pass
        log.chmod(0o600)
        command = ['systemd-run', '--user', '--collect', '--unit='+unit,
                   '--property=Type=exec', '--property=RuntimeMaxSec='+str(SUPERVISOR_DEADLINE_SECONDS),
                   '--property=MemoryMax=2G', '--property=MemorySwapMax=0',
                   '--property=CPUQuota=25%', '--property=UMask=0077',
                   '--property=WorkingDirectory='+str(ROOT),
                   '--property=StandardOutput=append:'+str(log),
                   '--property=StandardError=append:'+str(log),
                   '--setenv=OMABIT_PRODUCT_VM_WORKER=1', '--setenv=GOPROXY=off', '--setenv=GOTELEMETRY=off',
                   '--setenv=GOCACHE='+str(Path.home()/'.cache/go-build'), '--', '/usr/bin/python3', '-B',
                   str(Path(__file__).resolve()), *sys.argv[1:], '--worker']
        report = {'unit': unit+'.service', 'source_commit': commit, 'harness_commit':harness_commit,'candidate': builds,
                  'output': str(out), 'log': str(log), 'command': command,
                  'scope': 'new read-only-backed task fixture; supported same-pier Stop/Start and explicit audited recovery of its original pending app; no host deployment changes'}
        receipt = Path('/tmp')/(unit+'.json')
        receipt.write_text(json.dumps(report, indent=2)+'\n')
        result = subprocess.run(command, capture_output=True, text=True, timeout=20)
        report.update(submit_exit=result.returncode, submit_stdout=result.stdout, submit_stderr=result.stderr)
        if result.returncode == 0:
            observed = subprocess.run(['systemctl', '--user', 'show', unit+'.service', '-p', 'ActiveState,SubState,ControlGroup,MemoryMax,MemorySwapMax,CPUQuotaPerSecUSec,RuntimeMaxUSec'], capture_output=True, text=True, timeout=10)
            report['live_supervisor_observation'] = {'exit': observed.returncode, 'properties': observed.stdout}
        receipt.write_text(json.dumps(report, indent=2)+'\n')
        print(json.dumps({'unit': unit+'.service', 'receipt': str(receipt), 'log': str(log), 'output': str(out)}))
        if result.returncode:
            raise RuntimeError('fresh supervisor submission failed')
        return
    if os.environ.get('OMABIT_PRODUCT_VM_WORKER') != '1':
        raise RuntimeError('bounded worker environment required')
    signal.signal(signal.SIGTERM, terminate)
    signal.signal(signal.SIGINT, terminate)
    lock = os.open('/tmp/omabit-issues-vm-runtime.lock', os.O_RDWR|os.O_CREAT|os.O_NOFOLLOW, 0o600)
    fcntl.flock(lock, fcntl.LOCK_EX|fcntl.LOCK_NB)
    cap, scratch = args.storage_budget_gib*GIB, SCRATCH
    admission = budget.charge(scratch+HOST_BUILD_RESERVE, cap)
    available = next(int(line.split()[1])*1024 for line in Path('/proc/meminfo').read_text().splitlines() if line.startswith('MemAvailable:'))
    if available < 14*GIB:
        raise RuntimeError('14GiB available memory required for bounded8GiB guest and2GiB staging supervisor')
    out.mkdir(mode=0o700)
    (out/'control').mkdir(mode=0o700)
    (out/'preparation.json').write_text(json.dumps({'source_commit':commit,'harness_commit':harness_commit,
        'phase':'immutable-payload-staging','guest_created':False,'candidate':builds,'admission':admission},indent=2)+'\n')
    payload = out/'payload'
    payload.mkdir(mode=0o700)
    (payload/'candidate').mkdir()
    import gzip, hashlib
    for name, path in [(n, args.candidate/n) for n in ('omabit', 'urbit-collector', 'urbit-runner')]+[('managed.test', args.helper)]:
        target = payload/('candidate/'+name+'.gz' if name!='managed.test' else name+'.gz')
        target.write_bytes(gzip.compress(path.read_bytes(), compresslevel=6, mtime=0))
        if hashlib.sha256(gzip.decompress(target.read_bytes())).hexdigest()!=builds[name]['sha256']:
            raise RuntimeError('compressed exact input differs')
    shutil.copy2(Path(__file__).with_name('guest.py'), payload/'guest.py')
    shutil.copy2(ROOT/'scripts/qualification/issues-recovery/guest.py', payload/'issues-common.py')
    scenario = {'kind': 'issues18-product-grubbery-runtime-v1', 'development_snapshot': True,
                'source_commit': commit, 'harness_commit':harness_commit,'original_schema': 62, 'target_schema': 63, 'candidate': builds,
                'pill_sha256': budget.PILL, 'oci_sha256': budget.OCI,
                'case': 'cow-recover-pending-zod-v1','predecessor_root_sha256':PREDECESSOR_SHA}
    (payload/'scenario.json').write_text(json.dumps(scenario, indent=2)+'\n')
    disk = out/'payload.ext4'
    with disk.open('xb') as stream:
        stream.truncate(64*1024**2)
    util.run(['mkfs.ext4', '-q', '-F', '-d', payload, disk], stdout=subprocess.DEVNULL)
    shutil.rmtree(payload)  # Only this invocation's completed inert staging.
    (out/'preparation.json').write_text(json.dumps({'source_commit':commit,'harness_commit':harness_commit,
        'phase':'payload-built','guest_created':False,'candidate':builds,'admission':admission},indent=2)+'\n')
    init = out/'init'
    util.run(['go', 'build', '-trimpath', '-o', init, './scripts/qualification/grubbery-runtime-cow/init'], cwd=ROOT, env={**os.environ, 'CGO_ENABLED': '0','GOPROXY':'off','GOTELEMETRY':'off'})
    boot = out/'boot'
    boot.mkdir()
    util.run(['bsdtar', '-xf', portable/'boot/initramfs.cpio', '-C', boot, 'guest-modules', 'modules'])
    with (out/'initramfs.cpio').open('xb') as stream:
        ino = 0
        for name in ('proc', 'sys', 'dev', 'newroot', 'modules'):
            ino += 1
            util.cpio(stream, name, b'', 0o40755, ino)
        for path in [init, boot/'guest-modules', *sorted((boot/'modules').glob('*.ko'))]:
            ino += 1
            name = 'init' if path == init else str(path.relative_to(boot))
            util.cpio(stream, name, path.read_bytes(), 0o100755 if name == 'init' else 0o100644, ino)
        util.cpio(stream, 'TRAILER!!!', b'', 0, ino+1)
    tools = portable/'tools/qemu'
    env = {**os.environ, 'LD_LIBRARY_PATH': str(tools/'usr/lib'), 'QEMU_MODULE_DIR': str(tools/'usr/lib/qemu')}
    info = json.loads(subprocess.check_output([tools/'usr/bin/qemu-img', 'info', '--output=json', portable/'base.qcow2'], env=env, text=True))
    if info['format'] != 'qcow2' or info.get('backing-filename'):
        raise RuntimeError('standalone immutable base required')
    util.run([tools/'usr/bin/qemu-img', 'create', '-q', '-u', '-f', 'qcow2', '-F', 'qcow2', '-b', '/predecessor.qcow2', out/'root.qcow2', str(info['virtual-size'])], env=env)

    staged_bytes = budget.storage.measure(out)['allocated_bytes']
    if staged_bytes+ROOT_FILE_MAX+32*1024**2+8*1024**2 > SCRATCH:
        raise RuntimeError('fixed staged inputs plus hard root/log/report bounds exceed scratch; no VM launched')
    unit = 'omabit-'+out.name
    command = ['systemd-run', '--user', '--scope', '--unit='+unit, '-p', 'MemoryMax=10G', '-p', 'MemorySwapMax=0', '-p', 'CPUQuota=100%', '-p', 'TasksMax=512', '--',
               'bwrap', '--unshare-all', '--die-with-parent', '--clearenv', '--ro-bind', '/usr', '/usr', '--symlink', 'usr/bin', '/bin', '--symlink', 'usr/lib', '/lib', '--symlink', 'usr/lib', '/lib64',
               '--proc', '/proc', '--dev', '/dev', '--dev-bind', '/dev/kvm', '/dev/kvm', '--tmpfs', '/tmp', '--dir', '/run', '--ro-bind', str(tools), '/tools', '--ro-bind', str(portable/'base.qcow2'), '/base.qcow2', '--ro-bind', str(PREDECESSOR), '/predecessor.qcow2',
               '--ro-bind', str(portable/'boot/vmlinuz'), '/vmlinuz', '--bind', str(out), '/lab', '--chdir', '/lab', '--setenv', 'HOME', '/lab', '--setenv', 'PATH', '/usr/bin:/bin',
               '--setenv', 'LD_LIBRARY_PATH', '/tools/usr/lib', '--setenv', 'QEMU_MODULE_DIR', '/tools/usr/lib/qemu', '--', '/usr/bin/prlimit', '--fsize='+str(ROOT_FILE_MAX)+':'+str(ROOT_FILE_MAX), '--', '/tools/usr/bin/qemu-system-x86_64',
               '-accel', 'kvm', '-cpu', 'host', '-machine', 'q35', '-m', '8G', '-smp', '3', '-nodefaults', '-no-user-config', '-display', 'none', '-monitor', 'none', '-nic', 'none', '-no-reboot',
               '-L', '/tools/usr/share/qemu', '-bios', '/tools/usr/share/qemu/bios-256k.bin', '-kernel', '/vmlinuz', '-initrd', '/lab/initramfs.cpio',
               '-append', 'console=ttyS0 panic=-1 quiet rdinit=/init omabit.disposable-guest=release-upgrade-v1 systemd.unit=multi-user.target',
               '-drive', 'file=/lab/root.qcow2,format=qcow2,discard=unmap,if=none,id=rootdisk', '-device', 'virtio-blk-pci,drive=rootdisk',
               '-drive', 'file=/lab/payload.ext4,format=raw,if=none,id=payload,readonly=on', '-device', 'virtio-blk-pci,drive=payload',
               '-serial', 'stdio']
    report = {'scenario': scenario, 'admission': admission, 'scratch_limit': scratch, 'root_logical_file_max': ROOT_FILE_MAX, 'host_build_reserve': HOST_BUILD_RESERVE, 'staged_allocated_bytes': staged_bytes,
              'host_mounts': 'readonly OS/QEMU/pristine base/fixed task-only predecessor plus fresh COW capsule; no host HOME/run/sockets',
              'network': 'none', 'command': command, 'memory_available_before': available,
              'predecessor_root_sha256':PREDECESSOR_SHA, 'harness_sha256': {str(p.relative_to(ROOT)): util.digest(p) for p in (Path(__file__), Path(__file__).with_name('guest.py'), Path(__file__).with_name('init')/'main.go', ROOT/'scripts/qualification/issues-recovery/guest.py', ROOT/'internal/managed/issues_grubbery_runtime_vm_test.go')}}

    def save():
        (out/'host.json').write_text(json.dumps(report, indent=2)+'\n')

    save()
    with (out/'serial.log').open('xb') as log:
        child = subprocess.Popen(command, stdout=log, stderr=subprocess.STDOUT)
        started = time.monotonic()
        try:
            while child.poll() is None:
                time.sleep(1)
                report['latest_storage'] = budget.charge(0, cap)
                if budget.storage.measure(out)['allocated_bytes'] > scratch or (out/'serial.log').stat().st_size > 32*1024**2 or time.monotonic()-started > VM_DEADLINE_SECONDS:
                    raise RuntimeError('product VM storage/log/deadline exceeded; preserve failure capsule')
        except BaseException as error:
            report['interruption'] = type(error).__name__+': '+str(error)
            save()
            if child.poll() is None:
                subprocess.run(['systemctl', '--user', 'kill', '--signal=TERM', unit+'.scope'], check=True)
                child.wait(timeout=15)
            raise
        report['exit'] = child.returncode
    if util.digest(PREDECESSOR) != PREDECESSOR_SHA:
        raise RuntimeError('original predecessor changed; retain all evidence')
    report['original_predecessor_unchanged'] = True
    result = None
    for line in (out/'serial.log').read_text(errors='replace').splitlines():
        if 'OMABIT_GRUBBERY_VM_RESULT ' in line:
            result = json.loads(line.split('OMABIT_GRUBBERY_VM_RESULT ', 1)[1])
    report['after'] = budget.charge(0, cap)
    (out/'result.json').write_text(json.dumps(result, indent=2)+'\n')
    save()
    print(json.dumps({'passed': bool(result and result.get('passed')), 'output': str(out), 'error': result.get('error') if result else 'No completed guest result'}))
    if not result or not result.get('passed'):
        raise SystemExit(1)


if __name__ == '__main__':
    main()
