#!/usr/bin/env python3
"""Build and boot an offline disposable systemd/rootless-engine guest."""
import storage
from console_events import guest_events
import argparse
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import re
import shutil
import signal
import stat
import subprocess
import time
import uuid

REPO=Path(__file__).resolve().parents[3]
MODULES=('virtio_pci','virtio_blk','ext4','overlay','tun','veth','bridge','br_netfilter','nf_tables','nft_chain_nat','nft_compat','ip_tables','ip6_tables','iptable_nat','iptable_filter','ip6table_nat','ip6table_filter','xt_MASQUERADE','xt_addrtype','xt_conntrack','xt_tcpudp','xt_nat','nft_nat','nft_masq','nf_conntrack_netlink')

def sha(path):
    with path.open('rb') as f:return hashlib.file_digest(f,'sha256').hexdigest()

def helper():
    spec=importlib.util.spec_from_file_location('private_vm_tools',REPO/'scripts/qualify-powercut.py')
    module=importlib.util.module_from_spec(spec);spec.loader.exec_module(module);return module

def fresh(path,prefix):
    path=path.absolute()
    if path.parent!=Path('/tmp') or not re.fullmatch(prefix+'[A-Za-z0-9-]+',path.name) or path.exists():raise ValueError('a fresh /tmp/'+prefix+'NAME is required')
    path.mkdir(mode=0o700);return path

def write(path,data,mode=0o644):
    path.parent.mkdir(parents=True,exist_ok=True)
    path.write_text(data);path.chmod(mode)

def prepare(args):
    if getattr(args,"exercise_project_service_restart",False) and (not getattr(args,"exercise_project_public_recovery",False) or getattr(args,"exercise_project_public_initial",False) or getattr(args,"exercise_project_public_members",False)):
        raise ValueError("--exercise-project-service-restart requires public-recovery and excludes public-initial/public-members")
    if getattr(args,"exercise_project_public_members",False) and (not getattr(args,"exercise_project_context_lifecycle",False) or getattr(args,"exercise_project_public_initial",False) or getattr(args,"exercise_project_anchor_review",False)):
        raise ValueError("--exercise-project-public-members requires --exercise-project-context-lifecycle and excludes public-initial/anchor-review")
    if getattr(args,"exercise_project_public_initial",False) and not getattr(args,"exercise_project_member_storage",False):raise ValueError("--exercise-project-public-initial requires --exercise-project-member-storage")
    if getattr(args,"exercise_project_public_recovery",False) and not args.exercise_project_anchor_continuation:raise ValueError("--exercise-project-public-recovery requires --exercise-project-anchor-continuation")
    if args.exercise_project_anchor_continuation and not args.exercise_project_anchor_recovery:raise ValueError("--exercise-project-anchor-continuation requires --exercise-project-anchor-recovery")
    if args.exercise_project_anchor_recovery and not args.exercise_project_anchor_review:raise ValueError("--exercise-project-anchor-recovery requires --exercise-project-anchor-review")
    if args.exercise_project_anchor_review and not args.exercise_project_context_lifecycle:raise ValueError("--exercise-project-anchor-review requires --exercise-project-context-lifecycle")
    if args.exercise_project_context_lifecycle and not args.exercise_project_current_readiness:raise ValueError("--exercise-project-context-lifecycle requires --exercise-project-current-readiness")
    if args.exercise_project_current_readiness and not args.exercise_project_live_peer:raise ValueError("--exercise-project-current-readiness requires --exercise-project-live-peer")
    if args.exercise_project_live_peer and not args.exercise_project_app_state:raise ValueError("--exercise-project-live-peer requires --exercise-project-app-state")
    if args.exercise_project_app_state and not args.exercise_project_member_launch:raise ValueError('--exercise-project-app-state requires --exercise-project-member-launch')
    if (args.exercise_import or args.exercise_fake_network or args.exercise_endpoints or args.exercise_project_preparation or args.exercise_project_anchor_start or args.exercise_project_member_storage or args.exercise_project_member_launch) and not args.runtime:raise ValueError('runtime cases require the verified runtime artifacts')
    if args.exercise_login_code and not args.exercise_import:raise ValueError('--exercise-login-code requires --exercise-import')
    if args.network_published and not args.exercise_fake_network:raise ValueError('--network-published requires --exercise-fake-network')
    if args.network_anchor_recovery:
        if not args.network_published or args.network_isolation:raise ValueError('--network-anchor-recovery requires a separate single published fake project')
        args.network_state=True
    if args.network_isolation:args.network_state=True
    if args.network_state and not args.exercise_fake_network:raise ValueError('--network-state requires --exercise-fake-network')
    if sum((args.exercise_import,args.exercise_fake_network,args.exercise_endpoints,args.exercise_project_preparation,args.exercise_project_anchor_start,args.exercise_project_member_storage,args.exercise_project_member_launch))>1:raise ValueError('choose one runtime qualification case')
    if args.import_mode!='rename' and not args.exercise_import:raise ValueError('copy mode requires --exercise-import')
    if args.interrupt_copy and (not args.exercise_import or args.import_mode!='copy'):raise ValueError('interruption requires --exercise-import --import-mode copy')
    if args.restore_copy and not args.interrupt_copy:raise ValueError('restoration requires --interrupt-copy')
    if args.restore_ui and not args.restore_copy:raise ValueError('UI restoration requires --restore-copy')
    root_disk_gib=24 if args.import_mode=='copy' else 12
    storage.admit(args.output.absolute().parent,8*1024**3)
    os.umask(0o022)
    work=fresh(args.output,'omabit-import-appliance-')
    root=work/'rootfs';root.mkdir(mode=0o755)
    artifacts=work/'artifacts';artifacts.mkdir(mode=0o700)
    package_root=args.packages.absolute()
    manifest=json.loads((package_root/'packages.json').read_text())
    if not 1<=len(manifest)<=256:raise ValueError('invalid package count')
    for entry in manifest:
        name=entry['filename']
        if '/' in name or not name.endswith('.pkg.tar.zst') or sha(package_root/'packages'/name)!=entry['sha256']:raise ValueError('package differs from signed preparation')
    # Archive expansion has no host writable mounts except this new root tree.
    extraction=['bwrap','--unshare-all','--die-with-parent','--clearenv','--ro-bind','/usr','/usr','--symlink','usr/bin','/bin','--symlink','usr/lib','/lib','--symlink','usr/lib','/lib64','--proc','/proc','--dev','/dev','--tmpfs','/tmp','--ro-bind',str(package_root/'packages'),'/packages','--bind',str(root),'/rootfs','--chdir','/rootfs','--','/usr/bin/bsdtar','--no-same-owner','--no-same-permissions','--no-xattrs','--no-acls','--exclude=.PKGINFO','--exclude=.BUILDINFO','--exclude=.MTREE','--exclude=.INSTALL','-xf']
    with (work/'extraction.log').open('wb') as log:
        for entry in manifest:subprocess.run(extraction+['/packages/'+entry['filename']],stdout=log,stderr=subprocess.STDOUT,check=True)
    # A few archive payloads are execute-only. The unprivileged disk builder
    # must read them; the guest subsequently reinstalls every signed package to
    # restore native ownership/modes before the host installer is tested.
    for parent, directories, files in os.walk(root,followlinks=False):
        for name in directories+files:
            path=Path(parent)/name;mode=path.lstat().st_mode
            if stat.S_ISLNK(mode):continue
            path.chmod(stat.S_IMODE(mode)|(0o700 if stat.S_ISDIR(mode) else 0o600))
    print('Package payloads staged in private root tree.',flush=True)
    lab=root/'opt/omabit-lab';lab.mkdir(parents=True,mode=0o755)
    for directory in ('proc','sys','dev','run','tmp','home','root','var/lib/pacman/local','var/lib/pacman/sync','var/cache/pacman/pkg'):(root/directory).mkdir(parents=True,exist_ok=True)
    (root/'tmp').chmod(0o1777)
    for source in (package_root/'database/sync').iterdir():shutil.copyfile(source,root/'var/lib/pacman/sync'/source.name)
    for entry in manifest:
        for suffix in ('','.sig'):
            target=root/'var/cache/pacman/pkg'/(entry['filename']+suffix)
            shutil.copyfile(package_root/'packages'/target.name,target);target.chmod(0o644)
    config=(package_root/'pacman.conf').read_text()+'\nLocalFileSigLevel = Required\n'
    # SigLevel from [options] already applies to local packages; put the explicit
    # local setting there rather than under the last repository section.
    config=config.replace('\nLocalFileSigLevel = Required\n','').replace('[options]\n','[options]\nLocalFileSigLevel = Required\n')
    write(root/'etc/pacman.conf',config);write(lab/'pacman.conf',config)
    write(lab/'packages.json',json.dumps(manifest,indent=2)+'\n')
    write(lab/'guest-marker','offline-import-guest-v1\n',0o444)
    write(root/'etc/hostname','omabit-disposable-import\n')
    write(root/'etc/machine-id','')
    write(root/'etc/fstab','/dev/vda / ext4 defaults 0 0\n')
    shutil.copyfile(REPO/'scripts/qualification/imports/guest.py',lab/'guest.py')
    shutil.copyfile(REPO/'scripts/qualification/imports/import_case.py',lab/'import_case.py')
    if args.exercise_project_app_state:
        for name in ('runner_app_probe.py','runner_app_state.py'):shutil.copyfile(REPO/'scripts/qualification/imports'/name,lab/name)
    shutil.copyfile(REPO/'scripts/qualification/fake-networks/network_case.py',lab/'network_case.py')
    for name in ('state_case.py','omabit-state.hoon','project_isolation.py'):
        shutil.copyfile(REPO/'scripts/qualification/fake-networks'/name,lab/name)
    write(lab/'case.json',json.dumps({'exercise_project_service_restart':getattr(args,'exercise_project_service_restart',False),'exercise_project_public_members':getattr(args,'exercise_project_public_members',False),'exercise_project_public_initial':getattr(args,'exercise_project_public_initial',False),'exercise_project_public_recovery':getattr(args,'exercise_project_public_recovery',False),'exercise_project_anchor_continuation':args.exercise_project_anchor_continuation,'exercise_project_anchor_recovery':args.exercise_project_anchor_recovery,'exercise_project_anchor_review':args.exercise_project_anchor_review,'exercise_project_context_lifecycle':args.exercise_project_context_lifecycle,'exercise_project_current_readiness':args.exercise_project_current_readiness,'exercise_project_live_peer':args.exercise_project_live_peer,'exercise_project_app_state':args.exercise_project_app_state,'exercise_project_member_launch':args.exercise_project_member_launch,'exercise_project_member_storage':args.exercise_project_member_storage,'exercise_project_anchor_start':args.exercise_project_anchor_start,'exercise_project_preparation':args.exercise_project_preparation,'network_anchor_recovery':args.network_anchor_recovery,'network_published':args.network_published,'exercise_login_code':args.exercise_login_code,'exercise_endpoints':args.exercise_endpoints,'endpoint_network':args.endpoint_network,'exercise_import':args.exercise_import,'exercise_fake_network':args.exercise_fake_network,'network_state':args.network_state,'network_isolation':args.network_isolation,'import_mode':args.import_mode,'interrupt_copy':args.interrupt_copy,'restore_copy':args.restore_copy,'restore_ui':args.restore_ui})+'\n')
    ui_sources={}
    if args.restore_ui:
        import restore_ui_bundle
        ui_sources=restore_ui_bundle.prepare(REPO,lab/'restore-ui')
    if args.runtime:
        runtime=args.runtime.absolute();manifest=json.loads((runtime/'runtime.json').read_text())
        if sha(runtime/'runtime.oci.tar')!=manifest['archive_sha256'] or sha(runtime/'boot.pill')!=manifest['pill_sha256']:raise ValueError('runtime artifacts changed')
        target=lab/'runtime';target.mkdir()
        for name in ('runtime.json','runtime.oci.tar','boot.pill'):shutil.copyfile(runtime/name,target/name)
    unit='''[Unit]
Description=Omabit disposable import guest preparation
After=systemd-user-sessions.service dbus.service
Wants=dbus.service
[Service]
Type=oneshot
ExecStart=/usr/bin/python3 /opt/omabit-lab/guest.py
ExecStopPost=/usr/bin/systemctl --no-block poweroff
TimeoutStartSec=2100
StandardOutput=journal+console
StandardError=journal+console
[Install]
WantedBy=multi-user.target
'''
    if args.exercise_project_member_launch:unit=unit.replace('TimeoutStartSec=2100','TimeoutStartSec=4200')
    if args.exercise_fake_network:unit=unit.replace('TimeoutStartSec=2100','TimeoutStartSec='+('6600' if args.network_isolation else '3300'))
    write(root/'etc/systemd/system/omabit-guest.service',unit)
    wants=root/'etc/systemd/system/multi-user.target.wants';wants.mkdir(parents=True,exist_ok=True)
    (wants/'omabit-guest.service').symlink_to('../omabit-guest.service')
    (root/'etc/systemd/system/default.target').symlink_to('/usr/lib/systemd/system/multi-user.target')
    # The system Docker daemon is never needed; only the installed owner's engine.
    for name in ('docker.service','docker.socket','containerd.service'):(root/'etc/systemd/system'/name).symlink_to('/dev/null')
    env=dict(os.environ,CGO_ENABLED='0',GOCACHE='/tmp/omabit-go-build',GOMODCACHE='/tmp/omabit-go-mod')
    for name,source in (('urbit-runner','./backend/cmd/urbit-runner'),('urbit-collector','./backend/cmd/urbit-collector')):
        subprocess.run(['go','build','-trimpath','-o',str(lab/name),source],cwd=REPO,env=env,check=True)
    if args.network_published:
        for name in ('published_client.py','endpoints_case.py'):
            shutil.copyfile(REPO/'scripts/qualification/fake-networks'/name,lab/name)
    if args.network_anchor_recovery:
        shutil.copyfile(REPO/'scripts/qualification/fake-networks/anchor_recovery.py',lab/'anchor_recovery.py')
    if args.exercise_login_code:
        shutil.copyfile(REPO/'scripts/qualification/imports/login_case.py',lab/'login_case.py')
    if args.exercise_endpoints:
        shutil.copyfile(REPO/'scripts/qualification/fake-networks/endpoints_case.py',lab/'endpoints_case.py')
        subprocess.run(['go','build','-trimpath','-o',str(lab/'endpoint-probe'),'./scripts/qualification/fake-networks/endpoint-probe'],cwd=REPO,env=env,check=True)
    if args.exercise_project_preparation or args.exercise_project_anchor_start or args.exercise_project_member_storage or args.exercise_project_member_launch:
        subprocess.run(['go','test','-c','-trimpath','-o',str(lab/'project-preparation-test'),'./internal/managed'],cwd=REPO,env=env,check=True)
    # This reviewed command only downloads/validates inert launcher artifacts.
    subprocess.run([str(lab/'urbit-runner'),'prepare-rootless','--output',str(lab/'rootless-bundle')],stdout=(work/'rootless-preparation.json').open('wb'),check=True)
    subprocess.run(['go','build','-trimpath','-o',str(artifacts/'init'),'./scripts/qualification/imports/guest-init'],cwd=REPO,env=env,check=True)
    kernel=args.kernel.absolute();release=kernel.parent.name
    shutil.copyfile(kernel,artifacts/'vmlinuz')
    entries={'init':((artifacts/'init').read_bytes(),0o100755)};seen=set();names=[]
    for name in MODULES:
        result=subprocess.check_output(['modprobe','--set-version',release,'--show-depends',name],text=True)
        for line in result.splitlines():
            if not line.startswith('insmod '):continue
            path=Path(line.split()[1])
            if path in seen:continue
            seen.add(path);target=f'modules/{len(seen):03d}.ko';names.append('/'+target)
            raw=subprocess.check_output(['zstd','-dc',str(path)]) if path.suffix=='.zst' else path.read_bytes()
            entries[target]=(raw,0o100644)
    entries['guest-modules']=(('\n'.join(names)+'\n').encode(),0o100644)
    module=helper()
    with (artifacts/'initramfs.cpio').open('wb') as out:
        inode=0
        for name in ('proc','sys','dev','newroot','modules'):inode+=1;module.cpio_file(out,name,b'',0o40755,inode)
        for name,(data,mode) in entries.items():inode+=1;module.cpio_file(out,name,data,mode,inode)
        module.cpio_file(out,'TRAILER!!!',b'',0,inode+1)
    disk=artifacts/'base.raw'
    with disk.open('xb') as f:f.truncate(root_disk_gib*1024**3)
    with (work/'mkfs.log').open('wb') as log:
        subprocess.run(['mkfs.ext4','-q','-F','-E','lazy_itable_init=0,lazy_journal_init=0','-d',str(root),str(disk)],stdout=log,stderr=subprocess.STDOUT,check=True)
    print('Fresh root disk and guest initialization image prepared.',flush=True)
    paths=[REPO/'scripts/qualification/imports'/name for name in ('appliance.py','console_events.py','guest.py','import_case.py','guest-init/main.go')]
    for name in ('network_case.py','state_case.py','omabit-state.hoon','project_isolation.py'):
        paths.append(REPO/'scripts/qualification/fake-networks'/name)
    if args.exercise_project_app_state:paths += [REPO/'scripts/qualification/imports'/name for name in ('runner_app_probe.py','runner_app_state.py')]
    if args.network_published:
        paths += [REPO/'scripts/qualification/fake-networks'/name for name in ('published_client.py','endpoints_case.py')]
    if args.network_anchor_recovery:paths += [REPO/'scripts/qualification/fake-networks/anchor_recovery.py']
    if args.exercise_login_code:paths += [REPO/'scripts/qualification/imports/login_case.py']
    if args.exercise_project_preparation or args.exercise_project_anchor_start or args.exercise_project_member_storage or args.exercise_project_member_launch:
        paths += [REPO/'internal/managed/project_vm_test.go']
        paths += [REPO/'internal/managed'/name for name in ('schema.go','store.go','storage.go','storage_actions.go')]
        paths += [REPO/'internal/runtimequalify'/name for name in ('docker_linux.go','compatibility_linux.go')]
        paths += sorted((REPO/'internal/runtimequalify').glob('project*.go'))
        paths += sorted((REPO/'internal/projectnetwork').glob('*.go'))
        paths += sorted((REPO/'internal/bootpill').glob('*.go'))
        paths += sorted((REPO/'internal/pierstore').glob('*.go'))
        paths += [REPO/'internal/runtimecandidate/metadata.go']
        paths += sorted((REPO/'internal/managed').glob('project*.go'))
    if args.exercise_project_anchor_continuation or getattr(args,"exercise_project_public_initial",False) or getattr(args,"exercise_project_public_members",False):
        # The public gates also calls the public collector/status route.
        # Bind every local Go input, including its transitive client/protocol and
        # test-helper sources, instead of only the earlier private project subset.
        paths += sorted(REPO.rglob('*.go'))
        paths += [REPO/'go.mod', REPO/'go.sum']
    if args.exercise_endpoints:
        paths += [REPO/'scripts/qualification/fake-networks/endpoints_case.py',REPO/'scripts/qualification/fake-networks/endpoint-probe/main.go']
    report={'exercise_project_service_restart':getattr(args,'exercise_project_service_restart',False),'exercise_project_public_members':getattr(args,'exercise_project_public_members',False),'exercise_project_public_initial':getattr(args,'exercise_project_public_initial',False),'exercise_project_public_recovery':getattr(args,'exercise_project_public_recovery',False),'exercise_project_anchor_continuation':args.exercise_project_anchor_continuation,'exercise_project_anchor_recovery':args.exercise_project_anchor_recovery,'exercise_project_anchor_review':args.exercise_project_anchor_review,'exercise_project_context_lifecycle':args.exercise_project_context_lifecycle,'exercise_project_current_readiness':args.exercise_project_current_readiness,'exercise_project_live_peer':args.exercise_project_live_peer,'exercise_project_app_state':args.exercise_project_app_state,'exercise_project_member_launch':args.exercise_project_member_launch,'exercise_project_member_storage':args.exercise_project_member_storage,'exercise_project_anchor_start':args.exercise_project_anchor_start,'exercise_project_preparation':args.exercise_project_preparation,'network_anchor_recovery':args.network_anchor_recovery,'network_published':args.network_published,'exercise_login_code':args.exercise_login_code,'exercise_endpoints':args.exercise_endpoints,'endpoint_network':args.endpoint_network,'exercise_import':args.exercise_import,'exercise_fake_network':args.exercise_fake_network,'network_state':args.network_state,'network_isolation':args.network_isolation,'import_mode':args.import_mode,'interrupt_copy':args.interrupt_copy,'restore_copy':args.restore_copy,'restore_ui':args.restore_ui,'ui_source_sha256':ui_sources,'root_disk_gib':root_disk_gib,'packages_manifest_sha256':sha(package_root/'packages.json'),'kernel_source':str(kernel),'modules':len(names),'artifacts':{p.name:sha(p) for p in artifacts.iterdir() if p.is_file()},'source_sha256':{str(p.relative_to(REPO)):sha(p) for p in paths},'runner_sha256':sha(lab/'urbit-runner'),'collector_sha256':sha(lab/'urbit-collector')}
    if args.exercise_project_preparation or args.exercise_project_anchor_start or args.exercise_project_member_storage or args.exercise_project_member_launch:report['project_test_sha256']=sha(lab/'project-preparation-test')
    write(work/'build.json',json.dumps(report,indent=2)+'\n',0o600)
    print(work,flush=True)

def private_chain_case(build):
    value=build.get('exercise_private_chain',False)
    if type(value) is not bool:raise ValueError('private-chain case flag must be boolean')
    if value and (build.get('import_mode','rename')!='rename' or build.get('root_disk_gib',12)!=12 or build.get('network_isolation',False) or any(v for k,v in build.items() if k.startswith('exercise_') and k!='exercise_private_chain')):
        raise ValueError('private-chain appliance must be its own twelve-GiB case')
    isolation=build.get('private_project_isolation_qualification',False)
    if type(isolation) is not bool or isolation and (not value or any(build.get(k,False) for k in ('gateway_resume_qualification','gateway_peer_recovery_qualification','http_publication_qualification'))):
        raise ValueError('private project isolation must be its own fresh private-chain case')
    initial=build.get('initial_http_publication_qualification',False)
    if type(initial) is not bool or initial and (not value or any(build.get(k,False) for k in ('gateway_resume_qualification','gateway_peer_recovery_qualification','http_publication_qualification','private_project_isolation_qualification'))):
        raise ValueError('initial HTTP publication must be its own fresh private-chain case')
    return value

def cpu_settings(build, cores):
    private_chain_case(build)
    if type(cores) is not int or cores not in (1,2):
        raise ValueError('disposable VM CPU cap must be one or two cores')
    runtime_case=any(build.get(k,False) for k in ('exercise_import','exercise_fake_network','exercise_project_member_launch','exercise_private_chain'))
    if cores==2 and not runtime_case:
        raise ValueError('two-core emulation requires an explicit ship runtime case')
    return {'cores':cores,'quota':str(cores*100)+'%','scope_quota':str(cores)+'s','tcg_thread':'multi' if cores==2 else 'single'}

def boot(args):
    appliance=args.appliance.absolute()
    build=json.loads((appliance/'build.json').read_text())
    cpu=cpu_settings(build,getattr(args,'cpu_cores',1))
    storage_admission=storage.admit(args.output.absolute().parent,8*1024**3)
    work=fresh(args.output,'omabit-import-guest-run-');lab=work/'lab';lab.mkdir()
    appliance=args.appliance.absolute();artifacts=appliance/'artifacts';bundle=args.tools.absolute()
    if any(sha(artifacts/name)!=digest for name,digest in build['artifacts'].items()):raise ValueError('appliance artifacts changed')
    if build.get('exercise_private_chain'):
        if sha(Path(build['base_source']))!=build['base_source_sha256']:raise ValueError('pristine private backing changed')
    qemu=helper().tool_evidence(bundle)
    mode=build.get('import_mode','rename');root_disk_gib=build.get('root_disk_gib',12)
    if mode not in ('rename','copy') or root_disk_gib not in (12,24) or mode=='copy' and not build.get('exercise_import'):raise ValueError('unsupported import disk case')
    # Create only fresh lab disks. The root backing path is valid in the sandbox.
    subprocess.run([str(bundle/'tools/usr/bin/qemu-img'),'create','-q','-u','-f','qcow2','-F','raw','-b','/artifacts/base.raw',str(lab/'root.qcow2'),str(root_disk_gib)+'G'],env=helper().tool_env(bundle),check=True)
    if mode=='copy':
        subprocess.run([str(bundle/'tools/usr/bin/qemu-img'),'create','-q','-f','qcow2',str(lab/'source.qcow2'),'4G'],env=helper().tool_env(bundle),check=True)
    unit='omabit-import-guest-'+uuid.uuid4().hex
    import_case=build.get('exercise_import',False);network_case=build.get('exercise_fake_network',False);member_launch=build.get('exercise_project_member_launch',False);runtime_case=import_case or network_case or member_launch or build.get('exercise_private_chain',False)
    isolation=build.get('network_isolation',False)
    private_isolation=build.get('private_project_isolation_qualification',False)
    four_ships=isolation or private_isolation
    memory_gib=24 if four_ships else 12 if runtime_case else 8
    memory=str(memory_gib)+'G';guest_ram='14336M' if four_ships else '7168M' if runtime_case else '3072M';deadline=7200 if four_ships else 3600 if build.get('exercise_private_chain') else 3600 if network_case else 4500 if member_launch else 2400 if import_case else 1500
    memory_available=None
    if build.get('exercise_private_chain'):
        memory_available=int(next(line.split()[1] for line in Path('/proc/meminfo').read_text().splitlines() if line.startswith('MemAvailable:')))*1024
        required=(32 if private_isolation else 16)*1024**3
        if memory_available<required:raise RuntimeError('private keyed VM has insufficient available host memory for its declared case')
    if isolation:
        if not network_case or not build.get('network_state'):raise ValueError('invalid isolation appliance')
        memory_available=int(next(line.split()[1] for line in Path('/proc/meminfo').read_text().splitlines() if line.startswith('MemAvailable:')))*1024
        if memory_available<32*1024**3:raise RuntimeError('four-fake VM requires 32 GiB available host memory before admission')
    command=['systemd-run','--user','--scope','--unit='+unit,'--property=MemoryMax='+memory,'--property=MemorySwapMax=0','--property=CPUQuota='+cpu['quota'],'--property=TasksMax=128','--','bwrap','--unshare-all','--die-with-parent','--clearenv','--ro-bind','/usr','/usr','--symlink','usr/bin','/bin','--symlink','usr/lib','/lib','--symlink','usr/lib','/lib64','--proc','/proc','--dev','/dev','--tmpfs','/tmp','--dir','/run','--ro-bind',str(bundle/'tools'),'/tools','--ro-bind',str(artifacts),'/artifacts','--bind',str(lab),'/lab','--chdir','/lab','--setenv','HOME','/lab','--setenv','PATH','/usr/bin:/bin','--setenv','LD_LIBRARY_PATH','/tools/usr/lib','--setenv','QEMU_MODULE_DIR','/tools/usr/lib/qemu','--','/tools/usr/bin/qemu-system-x86_64','-accel','tcg,thread='+cpu['tcg_thread'],'-cpu','max','-machine','q35','-m',guest_ram,'-smp','2' if runtime_case else '1','-nodefaults','-no-user-config','-display','none','-monitor','none','-serial','stdio','-nic','none','-no-reboot','-L','/tools/usr/share/qemu','-bios','/tools/usr/share/qemu/bios-256k.bin','-kernel','/artifacts/vmlinuz','-initrd','/artifacts/initramfs.cpio','-append','console=ttyS0 panic=-1 quiet rdinit=/init omabit.disposable-guest=offline-import-v1','-drive','file=/lab/root.qcow2,format=qcow2,if=none,id=rootdisk','-device','virtio-blk-pci,drive=rootdisk']
    if build.get('exercise_private_chain'):
        index=command.index('--bind')
        command[index:index]=['--ro-bind',build['base_source'],'/artifacts/base.raw']
    if mode=='copy':command+=['-drive','file=/lab/source.qcow2,format=qcow2,if=none,id=sourcedisk','-device','virtio-blk-pci,drive=sourcedisk,serial=omabit-import-source']
    write(work/'command.json',json.dumps(command,indent=2)+'\n')
    write(work/'inputs.json',json.dumps({'appliance':str(appliance),'build':build,'qemu':qemu,'unit':unit,'deadline_seconds':deadline,'memory_max':memory,'memory_available_at_start':memory_available,'guest_ram':guest_ram,'vcpus':2 if runtime_case else 1,'import_mode':mode,'source_disk_gib':4 if mode=='copy' else None,'storage_admission':storage_admission,'storage_driver_sha256':sha(Path(storage.__file__)),'boot_driver_sha256':sha(Path(__file__)),'cpu':cpu},indent=2)+'\n')
    print('Guest evidence: '+str(work),flush=True)
    reserve=work/'diagnostic-reserve'
    with reserve.open('xb') as f:os.posix_fallocate(f.fileno(),0,1024**2)
    start=time.monotonic();timed_out=False;log_limit=False;storage_problem=None;storage_checked=start;storage_latest=storage_admission
    with (work/'console.log').open('wb') as log:
        process=subprocess.Popen(command,stdout=log,stderr=subprocess.STDOUT,start_new_session=True)
        admission={};admission_deadline=time.monotonic()+8
        while process.poll() is None and time.monotonic()<admission_deadline:
            observed=subprocess.run(['systemctl','--user','show',unit+'.scope','--property=LoadState,ActiveState,MemoryMax,MemorySwapMax,TasksMax,CPUQuotaPerSecUSec'],capture_output=True,text=True,timeout=5)
            admission=dict(line.split('=',1) for line in observed.stdout.splitlines() if '=' in line)
            if admission.get('ActiveState')=='active':break
            time.sleep(.25)
        admitted=admission.get('LoadState')=='loaded' and admission.get('ActiveState')=='active' and admission.get('MemoryMax')==str(memory_gib*1024**3) and admission.get('MemorySwapMax')=='0' and admission.get('TasksMax')=='128' and admission.get('CPUQuotaPerSecUSec')==cpu['scope_quota']
        write(work/'scope-admission.json',json.dumps(admission,indent=2)+'\n')
        while admitted and process.poll() is None:
            timed_out=time.monotonic()-start>deadline
            log_limit=log.tell()>16*1024*1024
            if time.monotonic()-storage_checked>2:
                try:
                    storage_latest=storage.available(work)
                    minimum=(2 if build.get('exercise_private_chain') else 1)*1024**3
                    if storage_latest['available_bytes']<minimum:storage_problem='temporary storage below the '+str(minimum//1024**3)+' GiB reserve'
                except Exception as error:storage_problem=str(error)
                storage_checked=time.monotonic()
            if timed_out or log_limit or storage_problem:break
            time.sleep(0.25)
        if process.poll() is None:
            os.killpg(process.pid,signal.SIGTERM)
            try:process.wait(timeout=5)
            except subprocess.TimeoutExpired:os.killpg(process.pid,signal.SIGKILL);process.wait()
    reserve.unlink()
    output=(work/'console.log').read_text(errors='replace')
    events=guest_events(output)
    final=next((item for item in reversed(events) if item.get('stage')=='result'),None)
    report={'exit':process.returncode,'storage_admission':storage_admission,'storage_last':storage_latest,'storage_problem':storage_problem,'scope_admitted':admitted,'scope_admission':admission,'timed_out':timed_out,'log_limit':log_limit,'seconds':round(time.monotonic()-start,2),'events':events,'guest_result':final,'clean_shutdown':'Power down' in output or 'Powering off' in output,'console_sha256':sha(work/'console.log')}
    scope=subprocess.run(['systemctl','--user','show',unit+'.scope','--property=Result,MemoryPeak,MemoryMax'],capture_output=True,text=True,timeout=10)
    report['scope_status']=scope.stdout.strip()
    write(work/'result.json',json.dumps(report,indent=2)+'\n')
    print(json.dumps(report),flush=True)
    if not admitted or process.returncode or timed_out or log_limit or storage_problem or not final or not final.get('ok') or not report['clean_shutdown']:raise SystemExit(1)

def main():
    parser=argparse.ArgumentParser(description=__doc__);sub=parser.add_subparsers(dest='action',required=True)
    p=sub.add_parser('prepare');p.add_argument('--packages',type=Path,required=True);p.add_argument('--kernel',type=Path,required=True);p.add_argument('--runtime',type=Path);p.add_argument('--exercise-import',action='store_true');p.add_argument('--exercise-login-code',action='store_true');p.add_argument('--exercise-fake-network',action='store_true');p.add_argument('--exercise-endpoints',action='store_true');p.add_argument('--exercise-project-preparation',action='store_true');p.add_argument('--exercise-project-anchor-start',action='store_true');p.add_argument('--exercise-project-member-storage',action='store_true');p.add_argument('--exercise-project-member-launch',action='store_true');p.add_argument('--exercise-project-app-state',action='store_true');p.add_argument('--exercise-project-live-peer',action='store_true');p.add_argument('--exercise-project-current-readiness',action='store_true');p.add_argument('--exercise-project-context-lifecycle',action='store_true');p.add_argument('--exercise-project-anchor-review',action='store_true');p.add_argument('--exercise-project-anchor-recovery',action='store_true');p.add_argument('--exercise-project-anchor-continuation',action='store_true');p.add_argument('--exercise-project-public-recovery',action='store_true');p.add_argument('--exercise-project-service-restart',action='store_true');p.add_argument('--exercise-project-public-initial',action='store_true');p.add_argument('--exercise-project-public-members',action='store_true');p.add_argument('--endpoint-network',choices=['internal','egress'],default='internal');p.add_argument('--network-state',action='store_true');p.add_argument('--network-published',action='store_true');p.add_argument('--network-anchor-recovery',action='store_true');p.add_argument('--network-isolation',action='store_true');p.add_argument('--import-mode',choices=['rename','copy'],default='rename');p.add_argument('--interrupt-copy',action='store_true');p.add_argument('--restore-copy',action='store_true');p.add_argument('--restore-ui',action='store_true');p.add_argument('--output',type=Path,required=True)
    p=sub.add_parser('boot');p.add_argument('--cpu-cores',type=int,choices=[1,2],default=1,help='bounded host CPU cap; two enables multi-thread TCG for explicit ship runtime cases');p.add_argument('--appliance',type=Path,required=True);p.add_argument('--tools',type=Path,required=True);p.add_argument('--output',type=Path,required=True)
    args=parser.parse_args();prepare(args) if args.action=='prepare' else boot(args)

if __name__=='__main__':main()
