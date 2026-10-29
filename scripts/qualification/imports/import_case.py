"""Actual collector import acceptance, called only inside the admitted fresh VM."""
import hashlib
import json
import os
import re
from pathlib import Path
import stat
import subprocess
import time
import urllib.request

ROOT=Path('/opt/omabit-lab')

def prepare_source_disk(run):
    # All formatting is inside the admitted fresh guest. The serial, exact size
    # and empty header bind it to the extra disk created by this boot driver.
    if os.geteuid()!=0 or (ROOT/'guest-marker').read_text()!='offline-import-guest-v1\n' or Path('/sys/class/dmi/id/sys_vendor').read_text().strip()!='QEMU':raise RuntimeError('source disk preparation requires the admitted guest')
    case=json.loads((ROOT/'case.json').read_text())
    if not case.get('exercise_import') or case.get('import_mode')!='copy':raise RuntimeError('source disk preparation requires the copy case')
    device=Path('/dev/vdb');parent=Path('/home/lab/import-source')
    if not stat.S_ISBLK(device.stat().st_mode) or Path('/sys/class/block/vdb/serial').read_text().strip()!='omabit-import-source':raise RuntimeError('unexpected source disk identity')
    size=int(run('source-disk-size',['blockdev','--getsize64',str(device)]).strip())
    with device.open('rb') as f:header=f.read(4096)
    if size!=4*1024**3 or header!=bytes(4096) or parent.exists():raise RuntimeError('source disk is not a fresh expected device')
    run('source-disk-format',['mkfs.ext4','-q','-m','0',str(device)])
    parent.mkdir(mode=0o700)
    run('source-disk-mount',['mount','-t','ext4','-o','nodev,nosuid',str(device),str(parent)])
    os.chown(parent,1000,1000);parent.chmod(0o700)
    if parent.stat().st_dev==Path('/').stat().st_dev:raise RuntimeError('copy source did not mount a separate filesystem')
    evidence={'device':str(device),'serial':'omabit-import-source','bytes':size,'mount':str(parent),'device_number':parent.stat().st_dev,'root_device_number':Path('/').stat().st_dev}
    (ROOT/'results/source-disk.json').write_text(json.dumps(evidence,indent=2)+'\n')

def exercise(run,environment,runtime):
    if os.geteuid()!=0 or (ROOT/'guest-marker').read_text()!='offline-import-guest-v1\n' or Path('/sys/class/dmi/id/sys_vendor').read_text().strip()!='QEMU':raise RuntimeError('import case requires the admitted fresh VM')
    case=json.loads((ROOT/'case.json').read_text());mode=case.get('import_mode','rename');interrupt_copy=case.get('interrupt_copy',False);restore_copy=case.get('restore_copy',False)
    if restore_copy and not interrupt_copy:raise RuntimeError('restoration requires the interruption case')
    if interrupt_copy and mode!='copy':raise RuntimeError('interruption requires the copy case')
    if mode not in ('rename','copy'):raise RuntimeError('unsupported import mode')
    owner=['runuser','-u','lab','--']
    docker=owner+['docker','--host','unix:///run/user/1000/urbit-docker.sock']
    collector=owner+[str(ROOT/'urbit-collector'),'--config-dir','/home/lab/import-collector']
    def call(name,args,timeout=120):return run(name,args,timeout=timeout,env=environment)
    def document(name,args,timeout=120):return json.loads(call(name,args,timeout))
    def emit(stage,**values):print('OMABIT_IMPORT '+json.dumps(dict(stage=stage,**values)),flush=True)
    def save(name,value):(ROOT/'results'/(name+'.json')).write_text(json.dumps(value,indent=2)+'\n')
    def tree(path):
        entries=[]
        for p in sorted(path.rglob('*')):
            s=p.lstat();item={'path':str(p.relative_to(path)),'mode':stat.S_IMODE(s.st_mode),'uid':s.st_uid,'gid':s.st_gid,'size':s.st_size,'mtime_ns':s.st_mtime_ns}
            if stat.S_ISREG(s.st_mode):
                with p.open('rb') as f:item['sha256']=hashlib.file_digest(f,'sha256').hexdigest()
            elif not stat.S_ISDIR(s.st_mode):raise RuntimeError('unexpected fresh source entry')
            entries.append(item)
        return entries
    enrollment=document('collector-enroll',collector+['enroll','guest','--socket','/run/user/1000/urbit-runner/runner.sock'])
    pinned=collector+['--runner','guest','--expected-runner-id',enrollment['runner_id']]
    initial=document('collector-empty',pinned+['snapshot'])
    if initial['instances'] or initial['operations']:raise RuntimeError('runner is not empty')
    # This directory and container are created by this invocation; no existing
    # pier is discovered, copied, opened or re-used as source material.
    parent=Path('/home/lab/import-source');source=parent/'unclassified-pier'
    if mode=='rename':call('source-directory',owner+['mkdir','-m','700',str(parent)])
    elif not parent.is_mount() or parent.stat().st_dev==Path('/srv/urbit/piers').stat().st_dev:raise RuntimeError('copy source must be on the prepared separate disk')
    if source.exists():raise RuntimeError('fresh source already exists')
    container=call('source-create',docker+['create','--name','omabit-import-source','--label','org.omabit.import-lab=source','--pull','never','--entrypoint','/bin/urbit','--network','none','--user','0:0','--cap-drop','ALL','--security-opt','no-new-privileges','--cgroupns','private','--ipc','private','--restart','no','--stop-signal','SIGTERM','--stop-timeout','300','--memory=4g','--memory-swap=4g','--pids-limit=256','--cpus=1','--log-driver','local','--log-opt','max-size=5m','--log-opt','max-file=2','--mount','type=bind,src='+str(parent)+',dst=/urbit','--mount','type=bind,src='+str(ROOT/'runtime/boot.pill')+',dst=/artifacts/boot.pill,readonly',runtime['image'],'-t','-L','--no-conn','--no-dock','--loom','31','--snap-time','60','-b','127.0.0.1','--http-port','8080','-F','zod','-B','/artifacts/boot.pill','-c','/urbit/unclassified-pier']).decode().strip()
    if len(container)!=64 or any(c not in '0123456789abcdef' for c in container):raise RuntimeError('bad source container ID')
    call('source-start',docker+['start',container])
    emit('source-boot',container=container)
    deadline=time.monotonic()+720;last_notice=0
    while True:
        probe=subprocess.run(docker+['exec',container,'/usr/bin/curl','--disable','--noproxy','*','--silent','--fail','--max-time','3','--max-filesize','128','http://127.0.0.1:8080/~/host'],capture_output=True,env=environment,timeout=15)
        if probe.returncode==0 and probe.stdout.strip()==b'~zod':break
        inspected=json.loads(subprocess.check_output(docker+['inspect',container],env=environment,timeout=15))[0]
        if not inspected['State']['Running'] or time.monotonic()>deadline:
            call('source-failure-logs',docker+['logs','--tail','100',container])
            raise RuntimeError('fresh fake did not become ready; source container and data retained')
        if time.monotonic()-last_notice>30:emit('source-boot',status='waiting');last_notice=time.monotonic()
        time.sleep(3)
    call('source-stop',docker+['stop','--timeout','300',container],timeout=330)
    stopped=document('source-stopped',docker+['inspect',container])[0]
    if stopped['State']['Running'] or stopped['State']['ExitCode']!=0 or stopped['State']['OOMKilled']:raise RuntimeError('source did not stop cleanly')
    call('source-container-remove',docker+['rm',container])
    before=tree(source);source_stat=source.stat();save('source-tree',before)
    preview=document('guided-preview',pinned+['preview-pier','--source',str(source),'--storage-root','default'])
    if (preview['ship'],preview['network'],preview['kind'],preview['instance_id'],preview['destination'],preview['in_place'])!=('~zod','fakenet','fake','zod','fakeships/zod/zod',False):raise RuntimeError('unexpected identity-based import preview')
    if tree(source)!=before:raise RuntimeError('preview changed source contents or metadata')
    destination=Path('/srv/urbit/piers')/preview['destination']
    if destination.exists():raise RuntimeError('preview published a destination')
    emit('preview-verified',ship=preview['ship'],destination=preview['destination'],source_unchanged=True)
    request=pinned+['import-pier',preview['instance_id'],'--source',str(source),'--path',preview['destination'],'--ship',preview['ship'],'--fingerprint',preview['source']['fingerprint'],'--http-port',str(preview['http_port']),'--request-id','lab-import-zod','--identify','--confirm-stopped']
    accepted=document('import-submit',request)
    restoration=None;retained=None;empty_holding=None
    def settled(operation,timeout,retired=False):
        deadline=time.monotonic()+timeout;previous=None
        while True:
            snapshot=json.loads(subprocess.check_output(pinned+['snapshot'],env=environment,timeout=30))
            ops=[op for op in snapshot['operations'] if op['id']==operation['id']]
            items=[v for v in snapshot['instances'] if v['id']==preview['instance_id']]
            state=(ops[0]['status'] if ops else 'missing',items[0].get('external_import',{}).get('stage') if items else None)
            if state!=previous:emit('operation',operation=operation['kind'],status=state[0],import_stage=state[1]);previous=state
            if ops and ops[0]['status'] in ('succeeded','failed','interrupted','canceled'):
                save(operation['kind']+'-snapshot',snapshot)
                if ops[0]['status']!='succeeded':raise RuntimeError('operation failed: '+json.dumps(ops[0]))
                if retired:
                    if items:raise RuntimeError('restoration retained the failed registration')
                    return snapshot,None
                if len(items)!=1:raise RuntimeError('completed operation missing its instance')
                return snapshot,items[0]
            if time.monotonic()>deadline:raise RuntimeError('operation remains uncertain: '+operation['id'])
            time.sleep(2)
    if interrupt_copy:
        deadline=time.monotonic()+90
        while True:
            snapshot=json.loads(subprocess.check_output(pinned+['snapshot'],env=environment,timeout=10))
            instance=next(v for v in snapshot['instances'] if v['id']==preview['instance_id'])
            operation=next(v for v in snapshot['operations'] if v['id']==accepted['id'])
            progress=instance.get('external_import',{})
            if operation['status'] not in ('pending','running'):raise RuntimeError('missed partial-copy interruption window')
            if progress.get('stage')=='copying' and 0<progress.get('copied_bytes',0)<progress.get('total_bytes',0):break
            if time.monotonic()>deadline:raise RuntimeError('partial-copy progress was not observed')
            time.sleep(0.1)
        save('copy-before-interruption',snapshot)
        emit('interrupting-copy',copied_bytes=progress['copied_bytes'],total_bytes=progress['total_bytes'])
        call('interrupt-runner-stop',owner+['systemctl','--user','stop','urbit-runner.service'])
        held=Path(progress.get('holding_path',''))
        if held.parent.parent!=parent or held.name!='pier' or not re.fullmatch(r'\.omabit-import-[0-9a-f]{32}',held.parent.name):raise RuntimeError('unexpected held source path')
        if source.exists() or not held.is_dir() or held.stat().st_dev!=source_stat.st_dev or held.stat().st_ino!=source_stat.st_ino or tree(held)!=before:raise RuntimeError('interruption did not retain the complete original source object')
        partial=tree(destination);save('interrupted-destination-tree',partial)
        partial_bytes=sum(e['size'] for e in partial if 'sha256' in e)
        if not 0<partial_bytes<sum(e['size'] for e in before if 'sha256' in e):raise RuntimeError('interruption did not leave a partial destination')
        call('restart-runner',owner+['systemctl','--user','start','urbit-runner.service'])
        deadline=time.monotonic()+30
        while True:
            observed=subprocess.run(pinned+['snapshot'],env=environment,capture_output=True,timeout=10)
            if observed.returncode==0:
                snapshot=json.loads(observed.stdout);break
            if time.monotonic()>deadline:raise RuntimeError('same runner did not return after restart')
            time.sleep(0.2)
        save('copy-after-restart',snapshot)
        instance=next(v for v in snapshot['instances'] if v['id']==preview['instance_id'])
        operation=next(v for v in snapshot['operations'] if v['id']==accepted['id'])
        if operation['status'] not in ('failed','interrupted') or instance.get('active_operation_id') or instance['desired_state']!='stopped' or not instance.get('recovery_required'):raise RuntimeError('restarted runner did not retain the import fence')
        if call('after-interruption-runtime-count',docker+['ps','--quiet']).strip():raise RuntimeError('restart implicitly started an incomplete pier')
        # The current recover-import contract excludes partial copies. Verify
        # that refusal preserves both trees; this is not a successful recovery.
        response=subprocess.run(pinned+['operate',instance['id'],'recover-import','--revision',str(instance['config_revision']),'--request-id','lab-recover-partial-copy'],env=environment,capture_output=True,text=True,timeout=30)
        save('partial-recovery-response',{'exit':response.returncode,'stdout':response.stdout,'stderr':response.stderr})
        if response.returncode==0 or json.loads(response.stderr)['error']['code']!='invalid_state':raise RuntimeError('partial-copy recovery behavior changed; review before proceeding')
        if source.exists() or tree(held)!=before or tree(destination)!=partial:raise RuntimeError('refused recovery changed retained pier data')
        if call('final-runtime-count',docker+['ps','--quiet']).strip():raise RuntimeError('runtime started during refused recovery')
        result={'checkpoint':'partial-copy-preservation','checkpoint_passed':True,'full_import_recovery':False,'recovery_supported':False,'ship':'~zod','source':str(source),'holding_path':str(held),'destination':str(destination),'import_operation':accepted['id'],'original_operation_status':operation['status'],'copied_bytes_at_interrupt':progress['copied_bytes'],'partial_destination_bytes':partial_bytes,'source_preserved':True,'restart_preserved_fence':True,'recovery_refusal':'invalid_state','no_runtime_started':True}
        save('copy-interruption',result);emit('interruption-checkpoint',**result)
        if not restore_copy:return result
        failed_request=request;failed_operation=operation
        failed_revision=instance['config_revision']
        partial_parent=tree(destination.parent);partial_parent_stat=destination.parent.stat()
        review=document('restore-preview',pinned+['preview-import-restore',instance['id'],'--revision',str(failed_revision)])
        retained=Path(review['retain_at']);empty_holding=held.parent
        if (review['runner_id'],review['instance_id'],review['config_revision'],review['stage'],review['continuation'])!=(enrollment['runner_id'],instance['id'],failed_revision,'review',False):raise RuntimeError('restoration preview binding mismatch')
        if (review['source'],review['restore_to'],review['destination'])!=(str(held),str(source),str(destination.parent)) or retained.parent!=Path('/srv/urbit/piers/archive') or retained.name!='import-'+held.parent.name.removeprefix('.omabit-import-'):raise RuntimeError('restoration preview paths mismatch')
        if not re.fullmatch('[0-9a-f]{64}',review['review_digest']) or retained.exists():raise RuntimeError('invalid restoration digest or occupied archive')
        if (review['source_files'],review['source_bytes'],review['destination_files'],review['destination_bytes'])!=(sum('sha256' in e for e in before),sum(e['size'] for e in before if 'sha256' in e),sum('sha256' in e for e in partial),partial_bytes):raise RuntimeError('restoration preview counts differ from retained trees')
        if source.exists() or tree(held)!=before or tree(destination.parent)!=partial_parent:raise RuntimeError('restoration preview changed either tree')
        restore_request=pinned+['restore-import',instance['id'],'--revision',str(failed_revision),'--restore-digest',review['review_digest'],'--request-id','lab-restore-partial-copy','--confirm-restore']
        if case.get('restore_ui'):
            ui=ROOT/'results/ui-restore';ui.mkdir(mode=0o700);os.chown(ui,1000,1000)
            (ui/'input.json').write_text(json.dumps(review)+'\n');(ui/'input.json').chmod(0o444)
            ui_env=dict(environment,QT_QPA_PLATFORM='offscreen',QT_QUICK_BACKEND='software',QSG_RHI_BACKEND='software',QT_QUICK_CONTROLS_STYLE='Basic',QML_DISABLE_DISK_CACHE='1',QT_FORCE_STDERR_LOGGING='1')
            call_result=run('restore-ui',owner+['/usr/bin/python',str(ROOT/'restore-ui/run.py'),'--output',str(ui)],timeout=210,env=ui_env)
            ui_result=json.loads((ui/'result.json').read_text())
            if not ui_result.get('ok') or ui_result['review']['plan']!=review or not ui_result.get('retirement_visible'):raise RuntimeError('UI restoration evidence failed')
            restore_request=collector+ui_result['restore_args']
            restored=ui_result['ack'];save('restore-submit',restored)
        else:
            restored=document('restore-submit',restore_request)
        snapshot,_=settled(restored,120,retired=True)
        terminal=next(op for op in snapshot['operations'] if op['id']==restored['id'])
        expected_receipt={key:review[key] for key in ('source','restore_to','destination','retain_at','review_digest')}
        if terminal.get('import_restoration')!=expected_receipt:raise RuntimeError('completed restoration receipt lost its reviewed paths')
        def restored_source():
            return source.is_dir() and (source.stat().st_dev,source.stat().st_ino)==(source_stat.st_dev,source_stat.st_ino) and tree(source)==before
        def retained_partial():
            return retained.is_dir() and (retained.stat().st_dev,retained.stat().st_ino)==(partial_parent_stat.st_dev,partial_parent_stat.st_ino) and tree(retained)==partial_parent and empty_holding.is_dir() and not list(empty_holding.iterdir())
        if not restored_source() or not retained_partial() or destination.parent.exists():raise RuntimeError('restoration did not preserve both reviewed filesystem objects')
        if call('after-restoration-runtime-count',docker+['ps','--quiet']).strip():raise RuntimeError('restoration implicitly started a runtime')
        if document('restore-terminal-replay',restore_request)!=terminal:raise RuntimeError('terminal restoration replay differs')
        replay=document('failed-import-terminal-replay',failed_request)
        if replay!=failed_operation:raise RuntimeError('failed import replay differs from its original receipt')
        snapshot=document('after-restoration-replays',pinned+['snapshot'])
        if snapshot['instances'] or not restored_source() or not retained_partial() or destination.parent.exists():raise RuntimeError('terminal replay changed retired registration or restored trees')
        emit('restoration-verified',source=str(source),archive=str(retained),same_source_inode=True,failed_registration_retired=True,terminal_replays_unchanged=True)
        preview=document('reimport-guided-preview',pinned+['preview-pier','--source',str(source),'--storage-root','default'])
        if (preview['ship'],preview['network'],preview['kind'],preview['instance_id'],preview['destination'],preview['in_place'])!=('~zod','fakenet','fake','zod','fakeships/zod/zod',False) or not restored_source() or not retained_partial():raise RuntimeError('fresh reimport preview changed identity/layout or retained data')
        request=pinned+['import-pier',preview['instance_id'],'--source',str(source),'--path',preview['destination'],'--ship',preview['ship'],'--fingerprint',preview['source']['fingerprint'],'--http-port',str(preview['http_port']),'--request-id','lab-reimport-restored-zod','--identify','--confirm-stopped']
        accepted=document('reimport-submit',request)
        if accepted['id']==failed_operation['id']:raise RuntimeError('fresh reimport reused the failed operation')
        restoration={'restore_operation':restored['id'],'failed_import_operation':failed_operation['id'],'failed_revision':failed_revision,'restored_source_same_inode':True,'retained_partial_same_inode':True,'archive':str(retained),'empty_holding':str(empty_holding),'terminal_replays_unchanged':True,'guided_ui':case.get('restore_ui',False)}
        save('source-restoration',restoration)
    snapshot,instance=settled(accepted,360)
    if instance['observed_state']!='stopped' or instance['desired_state']!='stopped' or not instance.get('identity_pending') or instance['pier_path']!='fakeships/zod/zod' or instance['pier_name']!='zod':raise RuntimeError('import did not settle at the required stopped named layout')
    after=tree(destination);save('destination-tree-before-start',after)
    holdings=set(parent.glob('.omabit-import-*'))
    if source.exists() or holdings!=({empty_holding} if empty_holding else set()):raise RuntimeError('successful import retained an old launch path or unexpected holding directory')
    if restoration:
        if instance['config_revision']!=restoration['failed_revision']+1 or not retained_partial():raise RuntimeError('reimport lost retirement revision or retained evidence')
        if document('restore-replay-after-reimport',restore_request)!=terminal or document('failed-import-replay-after-reimport',failed_request)!=failed_operation:raise RuntimeError('old request replay changed after reimport')
        current=document('reimport-after-old-replays',pinned+['snapshot'])
        selected=next(v for v in current['instances'] if v['id']==instance['id'])
        if selected['config_revision']!=instance['config_revision'] or selected['pier_path']!=instance['pier_path'] or selected['desired_state']!='stopped' or selected.get('active_operation_id') or tree(destination)!=after or not retained_partial():raise RuntimeError('old request replay affected the new registration')
    if mode=='rename':
        if after!=before or destination.stat().st_dev!=source_stat.st_dev or destination.stat().st_ino!=source_stat.st_ino:raise RuntimeError('same-filesystem import did not preserve the reviewed tree by rename')
    else:
        # Copy deliberately normalizes private modes and does not preserve times
        # or directory sizes. Compare every path/type and regular file's bytes.
        def contents(entries):return [{k:v for k,v in e.items() if k in ('path','sha256') or k=='size' and 'sha256' in e} for e in entries]
        if destination.stat().st_dev==source_stat.st_dev or contents(after)!=contents(before):raise RuntimeError('cross-filesystem copy did not preserve every path and byte')
        if any(e['uid']!=1000 or e['gid']!=1000 or e['mode']!=(0o600 if 'sha256' in e else 0o700) for e in after):raise RuntimeError('copied tree is not private and owner-bound')
        copied=instance.get('external_import',{})
        if copied.get('copied_bytes')!=sum(e['size'] for e in before if 'sha256' in e) or copied.get('copied_files')!=sum('sha256' in e for e in before):raise RuntimeError('copy receipt does not account for every file and byte')
    if [p.name for p in destination.parent.iterdir()]!=['zod']:raise RuntimeError('destination parent is not exclusive')
    # Deliberately re-present the exact already-completed request. This tests
    # durable replay only after a terminal result, never uncertain dispatch retry.
    replay=document('import-replay',request)
    if replay['id']!=accepted['id']:raise RuntimeError('completed import replay created another operation')
    active=call('before-explicit-start',docker+['ps','--quiet'])
    if active.strip():raise RuntimeError('import started a runtime implicitly')
    emit('import-verified',stopped=True,mode=mode,rename_preserved_inode=mode=='rename',copied_across_devices=mode=='copy',completed_replay_same_operation=True)
    started=document('explicit-start',pinned+['operate',instance['id'],'start','--revision',str(instance['config_revision']),'--request-id','lab-start-zod'])
    snapshot,instance=settled(started,600)
    if instance['observed_state']!='running' or instance.get('identity_pending') or instance.get('recovery_required') or instance.get('ship')!='~zod' or instance.get('health')!='healthy':raise RuntimeError('imported fake did not reach verified healthy state')
    ids=call('running-containers',docker+['ps','--quiet','--no-trunc']).decode().split()
    if len(ids)!=1:raise RuntimeError('unexpected runtime count after explicit Start')
    running=document('runtime-mounts',docker+['inspect',ids[0]])[0]
    binds=[m for m in running['Mounts'] if m['Type']=='bind']
    if len(binds)!=1 or binds[0]['Source']!=str(destination.parent) or binds[0]['Destination']!='/urbit' or not binds[0]['RW']:raise RuntimeError('runtime did not bind only the exclusive parent')
    if '/urbit/zod' not in running['Config']['Cmd']:raise RuntimeError('runtime did not select the named pier child')
    expected='http://127.0.0.1:'+str(preview['http_port'])
    if instance.get('endpoint')!=expected:raise RuntimeError('wrong published endpoint')
    with urllib.request.build_opener(urllib.request.ProxyHandler({})).open(expected+'/~/host',timeout=10) as response:
        if response.read(128).strip()!=b'~zod':raise RuntimeError('host endpoint identity mismatch')
    emit('running-verified',ship='~zod',parent_mount=str(destination.parent),pier='/urbit/zod')
    login_checks=[]
    if case.get('exercise_login_code'):
        import login_case
        instance=login_case.select(pinned,environment,instance,enrollment['runner_id'],ids[0],lambda rows:save('login-selection-before-restart',rows))
        login_checks.append(login_case.verify(pinned,environment,instance,enrollment['runner_id'],ids[0]))
        save('login-code-checks',login_checks);emit('login-code-verified',phase='before-restart')
        original_pier=(destination.stat().st_dev,destination.stat().st_ino)
        old_container=ids[0]
    stop=document('explicit-stop',pinned+['operate',instance['id'],'stop','--revision',str(instance['config_revision']),'--request-id','lab-stop-zod'])
    snapshot,instance=settled(stop,360)
    if instance['observed_state']!='stopped' or instance['desired_state']!='stopped' or instance.get('recovery_required'):raise RuntimeError('imported fake did not stop cleanly')
    if case.get('exercise_login_code'):
        restarted=document('explicit-restart-for-login',pinned+['operate',instance['id'],'start','--revision',str(instance['config_revision']),'--request-id','lab-login-restart-zod'])
        snapshot,instance=settled(restarted,600)
        if instance['observed_state']!='running' or instance.get('ship')!='~zod' or instance.get('health')!='healthy' or instance.get('recovery_required'):raise RuntimeError('code-discovery restart not healthy')
        resumed=call('login-restart-containers',docker+['ps','--quiet','--no-trunc']).decode().split()
        if len(resumed)!=1 or resumed[0]==old_container or (destination.stat().st_dev,destination.stat().st_ino)!=original_pier:raise RuntimeError('code-discovery restart did not preserve pier/replace container')
        instance=login_case.select(pinned,environment,instance,enrollment['runner_id'],resumed[0],lambda rows:save('login-selection-after-restart',rows))
        login_checks.append(login_case.verify(pinned,environment,instance,enrollment['runner_id'],resumed[0]))
        save('login-code-checks',login_checks);emit('login-code-verified',phase='after-restart')
        stop=document('explicit-login-final-stop',pinned+['operate',instance['id'],'stop','--revision',str(instance['config_revision']),'--request-id','lab-login-stop-zod'])
        snapshot,instance=settled(stop,360)
        if instance['observed_state']!='stopped' or instance.get('recovery_required'):raise RuntimeError('code-discovery final stop failed')
    if call('final-runtime-count',docker+['ps','--quiet']).strip():raise RuntimeError('runtime remains active')
    result={'passed':True,'ship':'~zod','source':str(source),'destination':str(destination),'import_operation':accepted['id'],'start_operation':started['id'],'stop_operation':stop['id'],'preview_source_unchanged':True,'mode':mode,'same_filesystem_rename':mode=='rename','cross_filesystem_copy':mode=='copy','parent_mount':str(destination.parent),'identity_verified':True,'final_state':'stopped'}
    if login_checks:result['login_code_checks']=login_checks;result['login_restart_same_pier']=True
    if restoration:
        if not retained_partial():raise RuntimeError('Start/Stop changed the retained partial copy')
        result.update(checkpoint='partial-copy-restore-reimport',restoration=restoration,old_replays_preserved_new_registration=True,retained_archive_unchanged=True)
    save('import-case',result);emit('complete',**result)
    return result
