#!/usr/bin/env python3
"""One fresh offline recording pilot VM, with bounded independent supervision.

No work without --run. Fixed disposable inputs only; no installed host service,
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
# One fresh ordinary fake, immutable payload and bounded private recordings.
# The host cap includes all retained evidence and shared build caches.
SCRATCH = 2*GIB


def ordinary(path, directory=False):
    if path.is_symlink() or path.resolve() != path.absolute() or not (path.is_dir() if directory else path.is_file()):
        raise RuntimeError('ordinary physical input required')
    return path


def terminate(number, frame):
    raise InterruptedError('recording pilot VM supervisor terminated: '+str(number))


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ('portable', 'candidate', 'predecessor', 'rootless', 'runtime', 'helper', 'output'):
        parser.add_argument('--'+name, type=Path)
    parser.add_argument('--storage-budget-gib', type=int, choices=(40, 48), default=40)
    parser.add_argument('--artifact-source', help='exact frozen artifact commit when only harness/documentation changed')
    parser.add_argument('--run', action='store_true')
    parser.add_argument('--worker', action='store_true', help=argparse.SUPPRESS)
    args = parser.parse_args()
    if not args.run:
        parser.print_help()
        return
    names = ('portable', 'candidate', 'predecessor', 'rootless', 'runtime', 'helper', 'output')
    if os.geteuid() == 0 or any(getattr(args, name) is None for name in names):
        parser.error('ordinary owner and every explicit input required')
    out = args.output
    if out.parent != Path('/tmp') or out.exists() or out.is_symlink() or not re.fullmatch('omabit-issues-recording-vm-[0-9]{2}', out.name):
        parser.error('fresh physical /tmp/omabit-issues-recording-vm-NN required; never resume a capsule')
    if subprocess.check_output(['git', 'status', '--porcelain'], cwd=ROOT):
        raise RuntimeError('freeze clean matching source before qualification')
    harness_commit = subprocess.check_output(['git', 'rev-parse', 'HEAD'], cwd=ROOT, text=True).strip()
    commit = args.artifact_source or harness_commit
    if not re.fullmatch('[0-9a-f]{40}',commit):
        raise RuntimeError('exact artifact source commit required')
    subprocess.run(['git','merge-base','--is-ancestor',commit,harness_commit],cwd=ROOT,check=True)
    if subprocess.check_output(['git','diff',commit,'--','*.go','go.mod','go.sum','internal/appdev'],cwd=ROOT):
        raise RuntimeError('Go/embedded inputs changed after the frozen artifacts')
    portable = ordinary(args.portable, True)
    meta = json.loads(ordinary(portable/'bundle.json').read_text())
    for name, pin in meta['sha256'].items():
        relative = Path(name)
        if relative.is_absolute() or '..' in relative.parts or util.digest(ordinary(portable/relative)) != pin:
            raise RuntimeError('pristine immutable input differs')
    if meta['sha256']['base.qcow2'] != 'e778801a4d1366004bbb0700b3920c2bc9cf4d1b0c07137357adfbabb1b7ec32':
        raise RuntimeError('wrong pristine not-onboarded base')
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
    predecessor=ordinary(args.predecessor)
    if util.digest(predecessor)!='98bfa21634dce830076f49b0b05a2422944fe55476734bd8e961b68c1ee60e6c':
        raise RuntimeError('exact previously qualified schema64 recording runner required')
    prior_info=subprocess.check_output(['go','version','-m',predecessor],text=True)
    if '\tvcs.revision=2d0281971cca0285e5aabc4b91279830a04ab9b1' not in prior_info or '\tvcs.modified=false' not in prior_info:
        raise RuntimeError('predecessor source identity differs')
    runtime = ordinary(args.runtime, True)
    if util.digest(ordinary(runtime/'boot.pill')) != budget.PILL or util.digest(ordinary(runtime/'runtime.oci.tar')) != budget.OCI:
        raise RuntimeError('frozen runtime/pill differs')
    ordinary(args.rootless, True)
    if not args.worker:
        # This unit owns only the host-side supervisor. The fresh QEMU scope is
        # separate and has its own stronger memory/CPU/no-swap boundary.
        budget.charge(SCRATCH, args.storage_budget_gib*GIB)
        token = uuid.uuid4().hex
        unit = 'omabit-issues-recording-vm-supervisor-'+token
        log = Path('/tmp')/(unit+'.log')
        with log.open('x'):
            pass
        log.chmod(0o600)
        command = ['systemd-run', '--user', '--collect', '--unit='+unit,
                   '--property=Type=exec', '--property=RuntimeMaxSec=2700',
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
                  'scope': 'one fresh offline recording pilot VM; no installed host service changes'}
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
    admission = budget.charge(scratch, cap)
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
    for name in ('omabit', 'urbit-collector', 'urbit-runner'):
        shutil.copy2(args.candidate/name, payload/'candidate'/name)
        (payload/'candidate'/name).chmod(0o755)
    shutil.copy2(predecessor,payload/'predecessor-runner')
    (payload/'predecessor-runner').chmod(0o755)
    shutil.copytree(args.rootless, payload/'previous/rootless-bundle')
    for name in ('boot.pill', 'runtime.oci.tar'):
        util.stage_immutable(runtime/name, payload/name)
    shutil.copy2(args.helper, payload/'managed.test')
    shutil.copy2(Path(__file__).with_name('guest.py'), payload/'guest.py')
    shutil.copy2(ROOT/'scripts/qualification/issues-recovery/guest.py', payload/'issues-common.py')
    scenario = {'kind': 'issues20-recording-policy-upgrade-v1', 'development_snapshot': True,
                'source_commit': commit, 'harness_commit':harness_commit,'target_schema': 65, 'candidate': builds,
                'pill_sha256': budget.PILL, 'oci_sha256': budget.OCI,
                'case': 'ordinary-capture-active-worker-upgrade-policy-api-v1',
                'predecessor':{'source':'2d0281971cca0285e5aabc4b91279830a04ab9b1','runner_sha256':util.digest(predecessor),'state_schema':64}}
    (payload/'scenario.json').write_text(json.dumps(scenario, indent=2)+'\n')
    disk = out/'payload.ext4'
    with disk.open('xb') as stream:
        stream.truncate(512*1024**2)
    util.run(['mkfs.ext4', '-q', '-F', '-d', payload, disk], stdout=subprocess.DEVNULL)
    shutil.rmtree(payload)  # Only this invocation's completed inert staging.
    (out/'preparation.json').write_text(json.dumps({'source_commit':commit,'harness_commit':harness_commit,
        'phase':'payload-built','guest_created':False,'candidate':builds,'admission':admission},indent=2)+'\n')
    init = out/'init'
    go_tmp=out/'go-tmp'
    go_tmp.mkdir(mode=0o700)
    util.run(['go', 'build', '-trimpath', '-o', init, './scripts/qualification/release-upgrade/init'], cwd=ROOT, env={**os.environ, 'CGO_ENABLED': '0','GOPROXY':'off','GOTELEMETRY':'off','GOTMPDIR':str(go_tmp),'TMPDIR':str(go_tmp)})
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
    util.run([tools/'usr/bin/qemu-img', 'create', '-q', '-u', '-f', 'qcow2', '-F', 'qcow2', '-b', '/base.qcow2', out/'root.qcow2', str(info['virtual-size'])], env=env)
    unit = 'omabit-'+out.name
    command = ['systemd-run', '--user', '--scope', '--unit='+unit, '-p', 'MemoryMax=10G', '-p', 'MemorySwapMax=0', '-p', 'CPUQuota=100%', '-p', 'TasksMax=512', '--',
               'bwrap', '--unshare-all', '--die-with-parent', '--clearenv', '--ro-bind', '/usr', '/usr', '--symlink', 'usr/bin', '/bin', '--symlink', 'usr/lib', '/lib', '--symlink', 'usr/lib', '/lib64',
               '--proc', '/proc', '--dev', '/dev', '--dev-bind', '/dev/kvm', '/dev/kvm', '--tmpfs', '/tmp', '--dir', '/run', '--ro-bind', str(tools), '/tools', '--ro-bind', str(portable/'base.qcow2'), '/base.qcow2',
               '--ro-bind', str(portable/'boot/vmlinuz'), '/vmlinuz', '--bind', str(out), '/lab', '--chdir', '/lab', '--setenv', 'HOME', '/lab', '--setenv', 'PATH', '/usr/bin:/bin',
               '--setenv', 'LD_LIBRARY_PATH', '/tools/usr/lib', '--setenv', 'QEMU_MODULE_DIR', '/tools/usr/lib/qemu', '--', '/tools/usr/bin/qemu-system-x86_64',
               '-accel', 'kvm', '-cpu', 'host', '-machine', 'q35', '-m', '8G', '-smp', '3', '-nodefaults', '-no-user-config', '-display', 'none', '-monitor', 'none', '-nic', 'none', '-no-reboot',
               '-L', '/tools/usr/share/qemu', '-bios', '/tools/usr/share/qemu/bios-256k.bin', '-kernel', '/vmlinuz', '-initrd', '/lab/initramfs.cpio',
               '-append', 'console=ttyS0 panic=-1 quiet rdinit=/init omabit.disposable-guest=release-upgrade-v1 systemd.unit=multi-user.target',
               '-drive', 'file=/lab/root.qcow2,format=qcow2,discard=unmap,if=none,id=rootdisk', '-device', 'virtio-blk-pci,drive=rootdisk',
               '-drive', 'file=/lab/payload.ext4,format=raw,if=none,id=payload,readonly=on', '-device', 'virtio-blk-pci,drive=payload',
               '-serial', 'stdio']
    report = {'scenario': scenario, 'admission': admission, 'scratch_limit': scratch,
              'host_mounts': 'only readonly OS/QEMU/pristine inputs plus fresh capsule; no host HOME/run/sockets',
              'network': 'none', 'command': command, 'memory_available_before': available,
              'harness_sha256': {str(p.relative_to(ROOT)): util.digest(p) for p in (Path(__file__), Path(__file__).with_name('guest.py'), ROOT/'scripts/qualification/issues-recovery/guest.py', ROOT/'internal/managed/issues_recording_vm_test.go')}}

    def save():
        (out/'host.json').write_text(json.dumps(report, indent=2)+'\n')

    save()
    with (out/'serial.log').open('xb') as log:
        child = subprocess.Popen(command, stdout=log, stderr=subprocess.STDOUT)
        started = time.monotonic()
        try:
            while child.poll() is None:
                time.sleep(5)
                report['latest_storage'] = budget.charge(0, cap)
                if budget.storage.measure(out)['allocated_bytes'] > scratch or (out/'serial.log').stat().st_size > 32*1024**2 or time.monotonic()-started > 2500:
                    raise RuntimeError('recording pilot VM storage/log/deadline exceeded; preserve failure capsule')
        except BaseException as error:
            report['interruption'] = type(error).__name__+': '+str(error)
            save()
            if child.poll() is None:
                subprocess.run(['systemctl', '--user', 'kill', '--signal=TERM', unit+'.scope'], check=True)
                child.wait(timeout=15)
            raise
        report['exit'] = child.returncode
    result = None
    for line in (out/'serial.log').read_text(errors='replace').splitlines():
        if 'OMABIT_RECORDING_VM_RESULT ' in line:
            result = json.loads(line.split('OMABIT_RECORDING_VM_RESULT ', 1)[1])
    report['after'] = budget.charge(0, cap)
    (out/'result.json').write_text(json.dumps(result, indent=2)+'\n')
    save()
    print(json.dumps({'passed': bool(result and result.get('passed')), 'output': str(out), 'error': result.get('error') if result else 'No completed guest result'}))
    if not result or not result.get('passed'):
        raise SystemExit(1)


if __name__ == '__main__':
    main()
