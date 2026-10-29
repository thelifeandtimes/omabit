#!/usr/bin/env python3
"""Fresh VM-only owner installation, with optional disposable fake import case."""
import hashlib
import json
import os
from pathlib import Path
import shutil
import subprocess
import time

ROOT=Path('/opt/omabit-lab')

def event(stage,**values):print('OMABIT_GUEST '+json.dumps(dict(stage=stage,**values)),flush=True)

def project_receipt_summary(path):
    # The full lifecycle receipt remains on the private guest disk. Serial events
    # carry a bounded reference whose bytes/hash must be checked after extraction.
    with path.open('rb') as stream:
        raw=stream.read(4*1024*1024+1)
    if len(raw)>4*1024*1024:raise ValueError('project receipt exceeds retained-file budget')
    receipt=json.loads(raw)
    if not isinstance(receipt,dict) or type(receipt.get('ok')) is not bool:raise ValueError('invalid project receipt status')
    return {'ok':receipt['ok'],'receipt':{'path':'results/project-preparation.json','bytes':len(raw),'sha256':hashlib.sha256(raw).hexdigest()}}


def run(name,args,timeout=120,env=None):
    event(name,status='running')
    result=subprocess.run(args,stdout=subprocess.PIPE,stderr=subprocess.STDOUT,timeout=timeout,env=env)
    (ROOT/'results'/(name+'.log')).write_bytes(result.stdout[:8*1024*1024])
    if result.returncode:
        event(name,status='failed',exit=result.returncode,tail=result.stdout[-3000:].decode(errors='replace'))
        raise RuntimeError(name+' failed')
    event(name,status='complete')
    return result.stdout

def main():
    # Admission precedes package, account, key and service mutations.
    if os.geteuid()!=0 or (ROOT/'guest-marker').read_text()!='offline-import-guest-v1\n':raise RuntimeError('wrong guest marker/uid')
    if Path('/sys/class/dmi/id/sys_vendor').read_text().strip()!='QEMU' or Path('/proc/1/comm').read_text().strip()!='systemd':raise RuntimeError('real QEMU/systemd required')
    if sorted(p.name for p in Path('/sys/class/net').iterdir())!=['lo']:raise RuntimeError('guest has a network device')
    if Path('/etc/urbit-runner/owner.json').exists():raise RuntimeError('guest already has an owner installation')
    (ROOT/'results').mkdir()
    report={'ok':False,'ship_operations':False}
    try:
        run('keyring-init',['pacman-key','--init'],240)
        run('keyring-populate',['pacman-key','--populate','archlinux','omarchy'],240)
        manifest=json.loads((ROOT/'packages.json').read_text())
        packages=[str(Path('/var/cache/pacman/pkg')/entry['filename']) for entry in manifest]
        run('native-packages',['pacman','-U','--noconfirm','--overwrite','*',*packages],600)
        shutil.copyfile(ROOT/'pacman.conf','/etc/pacman.conf')
        run('reload',['systemctl','daemon-reload'])
        run('owner',['useradd','--create-home','--uid','1000','--shell','/bin/bash','lab'])
        case=json.loads((ROOT/'case.json').read_text())
        if case.get('exercise_import') and case.get('import_mode')=='copy':
            import import_case
            import_case.prepare_source_disk(run)
        run('linger',['loginctl','enable-linger','lab'])
        run('user-manager',['systemctl','start','user@1000.service'])
        os.chmod('/home/lab',0o700)
        run('key',['runuser','-u','lab','--','ssh-keygen','-q','-t','ed25519','-N','','-f','/home/lab/lab-operator'])
        run('host-install',[str(ROOT/'urbit-runner'),'install-host','--owner','lab','--bundle',str(ROOT/'rootless-bundle'),'--public-key','/home/lab/lab-operator.pub','--acknowledge-host-changes'],240)
        environment=dict(os.environ,HOME='/home/lab',XDG_RUNTIME_DIR='/run/user/1000',DBUS_SESSION_BUS_ADDRESS='unix:path=/run/user/1000/bus')
        info=json.loads(run('engine',['runuser','-u','lab','--','docker','--host','unix:///run/user/1000/urbit-docker.sock','info','--format','{{json .}}'],env=environment))
        if 'name=rootless' not in info.get('SecurityOptions',[]) or info.get('CgroupVersion')!='2' or info.get('CgroupDriver')!='systemd' or info.get('DockerRootDir')!='/home/lab/.local/share/urbit-runner/docker':raise RuntimeError('unexpected engine identity')
        report.update(engine={key:info.get(key) for key in ('ServerVersion','CgroupVersion','CgroupDriver','DockerRootDir','SecurityOptions')})
        compatibility=json.loads(run('host-check',['runuser','-u','lab','--',str(ROOT/'urbit-runner'),'host-check','--live'],env=environment))
        if compatibility.get('can_mutate') is not True or compatibility.get('provenance')!='verified':raise RuntimeError('host compatibility does not admit mutations')
        report['compatibility']={key:compatibility.get(key) for key in ('compatibility','provenance','can_mutate','qualification')}
        if (ROOT/'runtime/runtime.json').exists():
            runtime=json.loads((ROOT/'runtime/runtime.json').read_text())
            docker=['runuser','-u','lab','--','docker','--host','unix:///run/user/1000/urbit-docker.sock']
            run('load-image',docker+['image','load','--input',str(ROOT/'runtime/runtime.oci.tar')],timeout=180,env=environment)
            inspected=json.loads(run('image-identity',docker+['image','inspect',runtime['image']],env=environment))
            if len(inspected)!=1:raise RuntimeError('wrong runtime inspection count')
            image=inspected[0]
            if image.get('Os')!='linux' or image.get('Architecture')!='amd64' or image.get('Id') not in (runtime['config_digest'],runtime['image'].split('@')[1]) or runtime['image'].removeprefix('docker.io/') not in image.get('RepoDigests',[]):raise RuntimeError('wrong runtime image identity')
            version=run('runtime-version',docker+['run','--rm','--network','none','--read-only','--cap-drop','ALL','--security-opt','no-new-privileges','--memory=512m','--pids-limit=128','--cpus=1','--entrypoint','/bin/urbit',runtime['image'],'--version'],env=environment)
            if b'4.6-5a7614f' not in version:raise RuntimeError('unexpected runtime version')
            report['runtime']={'image':runtime['image'],'id':image['Id'],'version':version.decode().strip()}
        if json.loads((ROOT/'case.json').read_text()).get('exercise_import'):
            if info.get('NCPU',0)<2:raise RuntimeError('managed runtime needs two exposed guest CPUs')
            network=run('network-preflight-create',docker+['network','create','--driver','bridge','omabit-import-preflight'],env=environment).decode().strip()
            probe=run('network-preflight-container',docker+['create','--name','omabit-import-preflight','--network',network,'--init','--entrypoint','/bin/sleep','--pull','never','--user','0:0','--cap-drop','ALL','--security-opt','no-new-privileges','--cgroupns','private','--ipc','private','--restart','no','--stop-signal','SIGTERM','--stop-timeout','15','--memory=4g','--memory-swap=4g','--pids-limit=256','--cpus=2','--publish','127.0.0.1:18079:8080/tcp',runtime['image'],'120'],env=environment).decode().strip()
            run('network-preflight-start',docker+['start',probe],env=environment)
            observed=json.loads(run('network-preflight-inspect',docker+['inspect',probe],env=environment))[0]
            if not observed['State']['Running'] or observed['State']['OOMKilled'] or observed['HostConfig']['NanoCpus']!=2000000000:raise RuntimeError('network/resource preflight did not start')
            run('network-preflight-stop',docker+['stop','--timeout','15',probe],env=environment)
            run('network-preflight-remove',docker+['rm',probe],env=environment)
            run('network-preflight-network-remove',docker+['network','rm',network],env=environment)
            report['network_preflight']=True
            import import_case
            report['ship_operations']=True
            report['import_case']=import_case.exercise(run,environment,runtime)
        if case.get('exercise_fake_network'):
            if info.get('NCPU',0)<2:raise RuntimeError('two fake galaxies need two exposed guest CPUs')
            import network_case
            report['ship_operations']=True
            report['fake_network']=network_case.exercise(run,environment,runtime)
        if case.get('exercise_project_preparation') or case.get('exercise_project_anchor_start') or case.get('exercise_project_member_storage') or case.get('exercise_project_member_launch'):
            start_anchor=case.get('exercise_project_anchor_start',False)
            project_env=dict(environment,OMABIT_TEST_PROJECT_VM='offline-project-anchor-start-v1' if start_anchor else 'offline-project-preparation-v1')
            test='TestProjectAnchorStartDisposableVM' if start_anchor else 'TestProjectPreparationDisposableVM'
            if case.get('exercise_project_member_storage'):
                project_env['OMABIT_TEST_PROJECT_VM']='offline-project-member-storage-v1'
                test='TestProjectMemberStorageDisposableVM'
                if case.get('exercise_project_public_initial'):
                    project_env['OMABIT_TEST_PROJECT_VM']='offline-project-public-initial-v1'
                    test='TestProjectPublicInitialDisposableVM'
            member_launch=case.get('exercise_project_member_launch',False)
            if member_launch:
                if info.get('NCPU',0)<2:raise RuntimeError('member launch needs two exposed guest CPUs')
                report['ship_operations']=True
                project_env['OMABIT_TEST_PROJECT_VM']='offline-project-member-launch-v1'
                test='TestProjectMemberLaunchDisposableVM'
                if case.get('exercise_project_app_state'):
                    project_env['OMABIT_TEST_PROJECT_VM']='offline-project-application-state-v1'
                    test='TestProjectApplicationStateDisposableVM'
                    if case.get('exercise_project_live_peer'):
                        project_env['OMABIT_TEST_PROJECT_VM']='offline-project-live-peer-v1'
                        test='TestProjectLivePeerDisposableVM'
                        if case.get('exercise_project_current_readiness'):
                            project_env['OMABIT_TEST_PROJECT_VM']='offline-project-current-context-v1'
                            test='TestProjectCurrentContextDisposableVM'
                        if case.get('exercise_project_context_lifecycle'):
                            project_env['OMABIT_TEST_PROJECT_VM']='offline-project-context-lifecycle-v1'
                            test='TestProjectContextLifecycleDisposableVM'
                        if case.get('exercise_project_anchor_review'):
                            project_env['OMABIT_TEST_PROJECT_VM']='offline-project-anchor-review-v1'
                            test='TestProjectAnchorReviewDisposableVM'
                        if case.get('exercise_project_anchor_recovery'):
                            project_env['OMABIT_TEST_PROJECT_VM']='offline-project-anchor-recovery-v1'
                            test='TestProjectAnchorRecoveryDisposableVM'
                        if case.get('exercise_project_anchor_continuation'):
                            project_env['OMABIT_TEST_PROJECT_VM']='offline-project-anchor-continuation-v1'
                            test='TestProjectAnchorRecoveryContinuationDisposableVM'
                        if case.get('exercise_project_public_recovery'):
                            project_env['OMABIT_TEST_PROJECT_VM']='offline-project-public-recovery-v1'
                            test='TestProjectPublicRecoveryDisposableVM'
            if case.get('exercise_project_public_members'):
                project_env['OMABIT_TEST_PROJECT_VM']='offline-project-public-members-v1'
                test='TestProjectPublicMembersDisposableVM'
            if case.get('exercise_project_service_restart'):
                project_env['OMABIT_TEST_PROJECT_VM']='offline-project-service-restart-v1'
                test='TestProjectServiceRestartDisposableVM'
            try:
                run('project-preparation',['runuser','-u','lab','--',str(ROOT/'project-preparation-test'),f'-test.run=^{test}$','-test.v','-test.timeout=3750s' if member_launch else '-test.timeout=240s'],timeout=3810 if member_launch else 260,env=project_env)
            finally:
                result_path=Path('/home/lab/project-preparation.json')
                if result_path.exists():
                    shutil.copyfile(result_path,ROOT/'results/project-preparation.json')
                    report['project_preparation']=project_receipt_summary(ROOT/'results/project-preparation.json')
            if not report.get('project_preparation',{}).get('ok'):raise RuntimeError('project preparation has no successful receipt')
        if case.get('exercise_endpoints'):
            import endpoints_case
            report['endpoints']=endpoints_case.exercise(environment,runtime)
        report['ok']=True
    except Exception as error:
        report['error']=str(error);event('failure',error=str(error))
        # Capture this admitted guest's own service diagnostics before shutdown.
        # These files remain private evidence; do not print raw journal contents.
        for unit in ('urbit-docker.service','urbit-runner.service'):
            try:
                diagnostic=subprocess.run(['journalctl','--no-pager','-n','150','_SYSTEMD_USER_UNIT='+unit],capture_output=True,timeout=15)
                (ROOT/'results'/('journal-'+unit+'.log')).write_bytes(diagnostic.stdout[:8*1024*1024])
            except Exception:pass
    finally:
        (ROOT/'results/result.json').write_text(json.dumps(report,indent=2)+'\n')
        event('result',**report)
        # This is the admitted fresh guest, not the host. Stop its own services
        # through systemd's normal shutdown ordering and retain its private disk.
        subprocess.run(['systemctl','poweroff'],check=False)

if __name__=='__main__':main()
