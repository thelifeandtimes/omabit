#!/usr/bin/env python3
"""One fresh offline development upgrade/recovery VM; no work without --run.

Uses published0.4.0 inputs and an exact clean source snapshot, not a new release.
Never opens an existing overlay, host runner socket, Docker socket or ship.
"""
import argparse
import fcntl
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import re
import shutil
import subprocess
import sys
import time

ROOT = Path(__file__).resolve().parents[3]
sys.path.insert(0, str(ROOT / 'scripts/qualification/app-development'))
import storage

spec = importlib.util.spec_from_file_location('upgrade_util', ROOT / 'scripts/qualification/patch-upgrade/run.py')
util = importlib.util.module_from_spec(spec)
spec.loader.exec_module(util)
GIB = 1024**3
OLD = {
    'omabit-linux-amd64': '237bb08256a4b077546885dd9306ac94fae0d3ecfb6998475f70d82202aa7a20',
    'urbit-collector-linux-amd64': '781decb3285ede2cf7bec317041d64599690bf3cb416f0c7b722846c41ca217b',
    'urbit-runner-linux-amd64': '5d273dfa82e2464f94c2ae5e8366cd314edf2917514b4baf3c7131824d195752',
}
PILL = 'f8b08c53dd6d0aa9c4f7654f0a7bcd11e94089222888a3c99c0a10eedcbb377b'
OCI = '25d86c8984acdc031380a897ccbf7dd2289fc1e19c10f81ba8cbe216009938ec'
RECOMMENDED = '85635c0b17a138a6473580ca362475faadf281e09cc9fa3746cfc698cc054f62'


def ordinary(path):
    if path.is_symlink() or not path.is_file() or path.resolve() != path.absolute():
        raise RuntimeError('ordinary physical input required: ' + str(path))
    return path


def worktree_build_roots():
    # A release continuation may move worktrees while retaining its earlier
    # frozen executables. Count every existing checkout's build directory so
    # changing ROOT cannot make those retained artifacts disappear from budget.
    records = subprocess.check_output(
        ['git', 'worktree', 'list', '--porcelain', '-z'], cwd=ROOT, text=True)
    paths = []
    for field in records.split('\0'):
        if field.startswith('worktree '):
            path = Path(field.removeprefix('worktree ')) / 'build'
            if path.is_symlink() or (path.exists() and not path.is_dir()):
                raise RuntimeError('unsafe worktree build directory')
            if path.is_dir():
                paths.append(path)
    if ROOT / 'build' not in paths:
        raise RuntimeError('current physical build directory missing from worktree inventory')
    return paths


def charge(additional, cap):
    roots = storage.roots()
    for p in (Path.home()/'.cache/go-build', Path.home()/'go/pkg/mod', *worktree_build_roots()):
        if p.is_symlink() or not p.is_dir():
            raise RuntimeError('missing physical shared build cache')
        if not any(p == root or p.is_relative_to(root) for root in roots):
            roots.append(p)
    measurements = [storage.measure(p) for p in roots]
    if any(row.get('unmeasured_reserve', 0) for row in measurements):
        raise RuntimeError('issues qualification requires complete namespace measurement; estimated reserves are not admission')
    return storage.admit(additional, measurements, budget_bytes=cap)


def verify_upgrade_section(prior,commit,builds):
    section=prior.get('installed_upgrade',{})
    if prior.get('source_commit')!=commit or prior.get('schema_before')!=59 or not section.get('passed') or not all(section.get(k) is True for k in ('running_stopped_intent_and_history_preserved','engine_and_container_processes_preserved','access_preserved','old_reader_refused_unchanged')) or any(prior.get('inputs',{}).get('candidate',{}).get(n,{}).get('sha256')!=builds[n]['sha256'] for n in builds):
        raise RuntimeError('recovery prerequisite is not the exact passed installed-upgrade section')


def main():
    p = argparse.ArgumentParser(description=__doc__)
    for name in ('portable', 'previous', 'candidate', 'rootless', 'runtime', 'helper', 'output'):
        p.add_argument('--'+name, type=Path)
    p.add_argument('--storage-budget-gib', type=int, choices=(40, 48), default=40)
    p.add_argument('--case',choices=('upgrade-recovery','recovery-only'),default='upgrade-recovery')
    p.add_argument('--artifact-source',help='explicit frozen artifact commit, independent of the clean qualification-harness commit')
    p.add_argument('--upgrade-evidence',type=Path,help='required passed exact-artifact upgrade section for recovery-only')
    p.add_argument('--run', action='store_true')
    a = p.parse_args()
    if not a.run:
        p.print_help()
        return
    if os.geteuid() == 0 or any(getattr(a, n) is None for n in ('portable','previous','candidate','rootless','runtime','helper','output')):
        p.error('ordinary owner and all explicit inputs required')
    if a.output.parent.resolve() != Path('/tmp') or a.output.exists() or a.output.is_symlink() or not re.fullmatch(r'omabit-issues-vm-[0-9]{2}', a.output.name):
        p.error('fresh /tmp/omabit-issues-vm-NN output required; existing capsules never resume')
    if subprocess.check_output(['git','status','--porcelain'], cwd=ROOT, text=True):
        raise RuntimeError('freeze a clean source commit before qualification')
    harness_commit = subprocess.check_output(['git','rev-parse','HEAD'], cwd=ROOT, text=True).strip()
    commit = a.artifact_source or harness_commit
    if not re.fullmatch('[0-9a-f]{40}',commit):
        p.error('exact frozen artifact source commit required')
    subprocess.run(['git','merge-base','--is-ancestor',commit,harness_commit],cwd=ROOT,check=True)
    lock = os.open('/tmp/omabit-issues-vm-runtime.lock', os.O_RDWR|os.O_CREAT|os.O_NOFOLLOW, 0o600)
    fcntl.flock(lock, fcntl.LOCK_EX|fcntl.LOCK_NB)
    cap = a.storage_budget_gib*GIB
    scratch_limit=(3 if a.case=='recovery-only' else 4)*GIB
    admission = charge(scratch_limit, cap)
    mem = {line.split(':')[0]: int(line.split()[1])*1024 for line in Path('/proc/meminfo').read_text().splitlines() if line.startswith('MemAvailable:')}
    if mem['MemAvailable'] < 12*GIB:
        raise RuntimeError('12GiB available memory required for the bounded8GiB guest')
    portable = a.portable.resolve(strict=True)
    meta = json.loads(ordinary(portable/'bundle.json').read_text())
    for name, expected in meta['sha256'].items():
        relative = Path(name)
        if relative.is_absolute() or '..' in relative.parts or util.digest(ordinary(portable/relative)) != expected:
            raise RuntimeError('pristine input inventory differs: '+name)
    if meta['sha256']['base.qcow2'] != 'e778801a4d1366004bbb0700b3920c2bc9cf4d1b0c07137357adfbabb1b7ec32':
        raise RuntimeError('wrong pristine not-onboarded base')
    for name, expected in OLD.items():
        if util.digest(ordinary(a.previous/name)) != expected:
            raise RuntimeError('published0.4.0 input differs')
    if util.digest(ordinary(a.previous/'omabit-recommended-apps-0.4.0.pill')) != RECOMMENDED:
        raise RuntimeError('published Recommended Apps pill differs')
    if util.digest(ordinary(a.runtime/'boot.pill')) != PILL or util.digest(ordinary(a.runtime/'runtime.oci.tar')) != OCI:
        raise RuntimeError('pinned runtime/pill differs')
    source_builds = {}
    for name, path in [(n,a.candidate/n) for n in ('omabit','urbit-collector','urbit-runner')] + [('managed.test',a.helper)]:
        info = subprocess.check_output(['go','version','-m',ordinary(path)], text=True)
        if name=='managed.test':
            # Go test omits automatic VCS settings. The frozen helper exposes
            # only its link-time source pin through this read-only test and
            # checks it again inside the guarded guest before opening state.
            identity = subprocess.check_output([path,'-test.run','^TestIssuesStopVMBuildIdentity$','-test.v'],text=True)
            if 'OMABIT_ISSUES_HELPER_SOURCE '+commit not in identity:
                raise RuntimeError('unfrozen/wrong test helper source')
        elif '\tvcs.revision='+commit not in info or '\tvcs.modified=false' not in info:
            raise RuntimeError('source snapshot provenance differs: '+name)
        source_builds[name] = {'sha256': util.digest(path), 'bytes': path.stat().st_size, 'build_info': info}
    prerequisite=None
    if a.case=='recovery-only':
        if a.upgrade_evidence is None:
            p.error('recovery-only requires passed exact-artifact upgrade evidence')
        prior=json.loads(ordinary(a.upgrade_evidence).read_text())
        verify_upgrade_section(prior,commit,source_builds)
        prerequisite={'original_report_sha256':util.digest(a.upgrade_evidence),'passed_section':'installed_upgrade','whole_original_scenario_passed':prior.get('passed',False),'source_commit':commit,'runner_sha256':source_builds['urbit-runner']['sha256']}
    out = a.output
    out.mkdir(mode=0o700)
    (out/'control').mkdir(mode=0o700)
    payload = out/'payload'
    payload.mkdir()
    for label in ('previous','candidate'):
        dest = payload/label
        dest.mkdir()
        for name in ('omabit','urbit-collector','urbit-runner'):
            shutil.copy2(a.previous/(name+'-linux-amd64') if label=='previous' else a.candidate/name, dest/name)
            (dest/name).chmod(0o755)
    shutil.copytree(a.rootless, payload/'previous/rootless-bundle')
    for name in ('boot.pill','runtime.oci.tar'):
        util.stage_immutable(a.runtime/name, payload/name)
    util.stage_immutable(a.previous/'omabit-recommended-apps-0.4.0.pill', payload/'recommended.pill')
    shutil.copy2(a.helper, payload/'managed.test')
    shutil.copy2(Path(__file__).with_name('guest.py'), payload/'guest.py')
    scenario = {'kind': 'issues16-21-development-v1', 'source_commit': commit, 'development_snapshot': True,
                'harness_commit':harness_commit,'case':a.case,'prerequisite_upgrade':prerequisite,
                'previous_version': '0.4.0', 'previous_schema': 59, 'target_schema': 60,
                'previous': OLD, 'candidate': source_builds, 'pill_sha256': PILL, 'recommended_sha256': RECOMMENDED, 'oci_sha256': OCI}
    (payload/'scenario.json').write_text(json.dumps(scenario,indent=2)+'\n')
    disk = out/'payload.ext4'
    with disk.open('xb') as f:
        f.truncate(768*1024**2)
    util.run(['mkfs.ext4','-q','-F','-d',payload,disk], stdout=subprocess.DEVNULL)
    shutil.rmtree(payload)  # Only this invocation's completed inert staging.
    init = out/'init'
    util.run(['go','build','-trimpath','-o',init,'./scripts/qualification/release-upgrade/init'], cwd=ROOT, env={**os.environ,'CGO_ENABLED':'0'})
    boot = out/'boot'
    boot.mkdir()
    util.run(['bsdtar','-xf',portable/'boot/initramfs.cpio','-C',boot,'guest-modules','modules'])
    with (out/'initramfs.cpio').open('xb') as f:
        ino = 0
        for name in ('proc','sys','dev','newroot','modules'):
            ino += 1
            util.cpio(f,name,b'',0o40755,ino)
        for path in [init,boot/'guest-modules',*sorted((boot/'modules').glob('*.ko'))]:
            ino += 1
            name = 'init' if path==init else str(path.relative_to(boot))
            util.cpio(f,name,path.read_bytes(),0o100755 if name=='init' else 0o100644,ino)
        util.cpio(f,'TRAILER!!!',b'',0,ino+1)
    tools = portable/'tools/qemu'
    env = {**os.environ,'LD_LIBRARY_PATH':str(tools/'usr/lib'),'QEMU_MODULE_DIR':str(tools/'usr/lib/qemu')}
    info = json.loads(subprocess.check_output([tools/'usr/bin/qemu-img','info','--output=json',portable/'base.qcow2'],env=env,text=True))
    if info['format']!='qcow2' or info.get('backing-filename'):
        raise RuntimeError('standalone immutable base required')
    util.run([tools/'usr/bin/qemu-img','create','-q','-u','-f','qcow2','-F','qcow2','-b','/base.qcow2',out/'root.qcow2',str(info['virtual-size'])],env=env)
    unit = 'omabit-'+out.name
    command = ['systemd-run','--user','--scope','--unit='+unit,'-p','MemoryMax=10G','-p','MemorySwapMax=0','-p','CPUQuota=100%','-p','TasksMax=512','--',
        'bwrap','--unshare-all','--die-with-parent','--clearenv','--ro-bind','/usr','/usr','--symlink','usr/bin','/bin','--symlink','usr/lib','/lib','--symlink','usr/lib','/lib64',
        '--proc','/proc','--dev','/dev','--dev-bind','/dev/kvm','/dev/kvm','--tmpfs','/tmp','--dir','/run',
        '--ro-bind',str(tools),'/tools','--ro-bind',str(portable/'base.qcow2'),'/base.qcow2','--ro-bind',str(portable/'boot/vmlinuz'),'/vmlinuz','--bind',str(out),'/lab',
        '--chdir','/lab','--setenv','HOME','/lab','--setenv','PATH','/usr/bin:/bin','--setenv','LD_LIBRARY_PATH','/tools/usr/lib','--setenv','QEMU_MODULE_DIR','/tools/usr/lib/qemu','--',
        '/tools/usr/bin/qemu-system-x86_64','-accel','kvm','-cpu','host','-machine','q35','-m','8G','-smp','3','-nodefaults','-no-user-config','-display','none','-monitor','none','-nic','none','-no-reboot',
        '-L','/tools/usr/share/qemu','-bios','/tools/usr/share/qemu/bios-256k.bin','-kernel','/vmlinuz','-initrd','/lab/initramfs.cpio',
        '-append','console=ttyS0 panic=-1 quiet rdinit=/init omabit.disposable-guest=release-upgrade-v1 systemd.unit=multi-user.target',
        '-drive','file=/lab/root.qcow2,format=qcow2,if=none,id=rootdisk','-device','virtio-blk-pci,drive=rootdisk',
        '-drive','file=/lab/payload.ext4,format=raw,if=none,id=payload,readonly=on','-device','virtio-blk-pci,drive=payload',
        '-qmp','unix:/lab/control/qmp.sock,server=on,wait=off','-serial','stdio']
    receipt = {'scenario':scenario,'admission':admission,'memory_available_before':mem['MemAvailable'],
        'host_mounts':'read-only OS/QEMU/pristine inputs; only fresh capsule writable; no host HOME/run/sockets',
        'network':'none','scratch_limit':scratch_limit,'command':command,'harness_sha256':{str(path.relative_to(ROOT)):util.digest(path) for path in (Path(__file__),Path(__file__).with_name('guest.py'),ROOT/'internal/managed/issues_stop_vm_test.go')}}
    def save():
        (out/'host.json').write_text(json.dumps(receipt,indent=2)+'\n')
    save()
    with (out/'serial.log').open('wb') as log:
        proc = subprocess.Popen(command,stdout=log,stderr=subprocess.STDOUT)
        started = time.monotonic()
        try:
            while proc.poll() is None:
                time.sleep(5)
                receipt['latest_storage'] = charge(0,cap)
                if storage.measure(out)['allocated_bytes']>scratch_limit or (out/'serial.log').stat().st_size>32*1024**2 or time.monotonic()-started>2400:
                    raise RuntimeError('VM storage/log/deadline exceeded; retain failure capsule')
        except BaseException as error:
            receipt['interruption'] = type(error).__name__+': '+str(error)
            save()
            if proc.poll() is None:
                # The uniquely created scope contains only this new VM.
                subprocess.run(['systemctl','--user','kill','--signal=TERM',unit+'.scope'],check=True)
                proc.wait(timeout=15)
            raise
        receipt['exit'] = proc.returncode
    receipt['after'] = charge(0,cap)
    result = None
    for line in (out/'serial.log').read_text(errors='replace').splitlines():
        if 'OMABIT_ISSUES_VM_RESULT ' in line:
            result = json.loads(line.split('OMABIT_ISSUES_VM_RESULT ',1)[1])
    (out/'result.json').write_text(json.dumps(result,indent=2)+'\n')
    save()
    print(json.dumps({'passed':bool(result and result.get('passed')),'output':str(out),'error':result.get('error') if result else 'No completed guest report'}))
    if not result or not result.get('passed'):
        raise SystemExit(1)


if __name__ == '__main__':
    main()
