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
    return {'ok':receipt['ok'],'receipt':{'path':'results/private-chain.json','bytes':len(raw),'sha256':hashlib.sha256(raw).hexdigest()}}


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
        # Explicit offline verification records per-role evidence before installation.
        # No retry or relaxed deadline: a refusal ends this fresh guest.
        verified=json.loads(run('verify-packages',[str(ROOT/'urbit-runner'),'verify-host-packages','--acknowledge-local-verification'],180))
        if verified.get('provenance')!='verified' or len(verified.get('packages',[]))!=8 or any(p.get('provenance')!='verified' for p in verified['packages']):raise RuntimeError('native package roles are not all verified')
        report['package_verification']={'provenance':verified['provenance'],'roles':[p['role'] for p in verified['packages']]}
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
        if case != {'exercise_private_chain': True}:raise RuntimeError('private-chain case required')
        os.environ['OMABIT_PRIVATE_CHAIN_DOCKER']='guest-v1'
        import sys
        sys.path.insert(0,str(ROOT/'private-chain/src'))
        import docker_case
        report['ship_operations']=True
        import docker_initial_publication
        proof=docker_case.exercise(environment,peer_checkpoint=docker_initial_publication.capture,publication_ports=(18381,18382))
        report['private_chain']=project_receipt_summary(ROOT/'results/private-chain.json')
        if not proof['ok']:raise RuntimeError('private keyed Docker proof failed')
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
