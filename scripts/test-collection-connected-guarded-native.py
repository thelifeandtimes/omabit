#!/usr/bin/env python3
"""Private connected native profiles; optional original-guardian recovery scenario."""
import importlib.util,json,shutil,sys
from pathlib import Path

def module(name,path):
 s=importlib.util.spec_from_file_location(name,path);m=importlib.util.module_from_spec(s);s.loader.exec_module(m);return m

QML=r'''
 IpcHandler {
  target:"connectedProbe"
  function state():string {
   var value={}
   for(var id in shell._services) {
    var h=shell._services[id],p=h.payload
    if(p && h.role==="theme")value.theme={loaded:p.loaded,ships:p.account.ships.length,desktop:p.desktopState.status,error:p.desktopState.error,bridgeBusy:!!p.desktopState.bridgeBusy,busy:p.busy}
   }
   for(var id in shell.panels) {
    var h=shell.panels[id].item,p=h?h.payload:null
    if(p)value.dms={ship:p.ship,busy:p.busy,outboundReadable:p.outboundReadable,unresolved:p.unresolvedOutbound.map(function(x){return {id:x.id,phase:x.phase}})}
   }
   return JSON.stringify(value)
  }
  function send():string {
   for(var id in shell.panels) {var h=shell.panels[id].item;if(h && h.payload)return h.payload.send("~nec","synthetic guarded native unresolved message","bar")}
   return "absent"
  }
 }
'''

def main():
 if '--inside' in sys.argv:
  if Path('/lab/shared-recovery-mode.json').exists():
   sys.path.insert(0,'/lab');sys.path.insert(0,'/lab/source')
   from wayland_runtime import Runtime
   import native_shared_runtime as shared
   display=Runtime()
   try:
    display.start()
    if Path('/lab/compositor-loss-mode.json').exists():
     module('native_compositor_loss_fixture',Path('/lab/native_compositor_loss_fixture.py')).run(display)
    elif Path('/lab/post-adopt-loss-mode.json').exists():
     module('native_recovery_fixture',Path('/lab/native_recovery_fixture.py')).run(runtime=shared,unbound_initial=True);shared.verify_result()
    elif Path('/lab/initial-owner-mode.json').exists():module('native_initial_owner_fixture',Path('/lab/native_initial_owner_fixture.py')).run()
    elif Path('/lab/initial-recovery-mode.json').exists():module('native_initial_fixture',Path('/lab/native_initial_fixture.py')).run()
    elif Path('/lab/partial-candidate-mode.json').exists():module('native_partial_fixture',Path('/lab/native_partial_fixture.py')).run()
    else:module('native_recovery_fixture',Path('/lab/native_recovery_fixture.py')).run(runtime=shared);shared.verify_result()
    if Path('/lab/provisioning-mode.json').exists() and not Path('/lab/compositor-loss-mode.json').exists():
     from native_provisioning_fixture import verify_completed
     verify_completed()
    if Path('/lab/authority-inspection-mode.json').exists():
     from native_authority_inspection_fixture import observe
     observe('final')
    if Path('/lab/session-checkpoint-mode.json').exists():
     from native_session_checkpoint_fixture import retire_runtime
     retire_runtime(display)
   finally:display.close()
   return
  if Path('/lab/native-recovery-mode.json').exists():return module('native_recovery_fixture',Path('/lab/native_recovery_fixture.py')).run()
  common=module('admission_fixture',Path('/lab/admission-fixture.py'))
  return module('native_connected_fixture',Path('/lab/native_connected_fixture.py')).run(common)
 repeat_sessions='--repeat-sessions' in sys.argv
 if repeat_sessions:
  sys.argv.remove('--repeat-sessions');assert '--session-checkpoint' in sys.argv
  assert not any(x in sys.argv for x in ('--session-successor','--successor-activation','--successor-upgrade','--continue-session'))
 cleanup_continuation='--cleanup-continuation' in sys.argv
 if cleanup_continuation:sys.argv.remove('--cleanup-continuation');assert '--inspect-continuation' in sys.argv
 inspect_continuation='--inspect-continuation' in sys.argv
 if inspect_continuation:sys.argv.remove('--inspect-continuation');assert '--continuation-activation' in sys.argv
 continuation_activation='--continuation-activation' in sys.argv
 if continuation_activation:
  sys.argv.remove('--continuation-activation');assert '--continue-session' in sys.argv and '--successor-upgrade' in sys.argv
 continue_session='--continue-session' in sys.argv
 if continue_session:sys.argv.remove('--continue-session');assert '--checkpoint-successor' in sys.argv
 checkpoint_successor='--checkpoint-successor' in sys.argv
 if checkpoint_successor:sys.argv.remove('--checkpoint-successor');assert '--cleanup-successor' in sys.argv
 cleanup_successor='--cleanup-successor' in sys.argv
 if cleanup_successor:
  sys.argv.remove('--cleanup-successor');assert all(x in sys.argv for x in ('--inspect-successor','--successor-activation','--successor-upgrade'))
 inspect_successor='--inspect-successor' in sys.argv
 if inspect_successor:sys.argv.remove('--inspect-successor');assert '--successor-activation' in sys.argv and '--inspect-retained-authority' in sys.argv
 successor_upgrade='--successor-upgrade' in sys.argv
 if successor_upgrade:sys.argv.remove('--successor-upgrade');assert '--successor-activation' in sys.argv
 successor_activation='--successor-activation' in sys.argv
 if successor_activation:sys.argv.remove('--successor-activation');assert '--session-successor' in sys.argv
 session_successor='--session-successor' in sys.argv
 if session_successor:sys.argv.remove('--session-successor');assert '--session-checkpoint' in sys.argv
 session_checkpoint='--session-checkpoint' in sys.argv
 if session_checkpoint:sys.argv.remove('--session-checkpoint');assert '--session-cleanup' in sys.argv
 session_cleanup='--session-cleanup' in sys.argv
 if session_cleanup:sys.argv.remove('--session-cleanup');assert '--compositor-loss' in sys.argv
 inspect_authority='--inspect-retained-authority' in sys.argv
 if inspect_authority:sys.argv.remove('--inspect-retained-authority');assert '--shared-authority' in sys.argv
 compositor_loss='--compositor-loss' in sys.argv
 if compositor_loss:
  sys.argv.remove('--compositor-loss');assert all(x in sys.argv for x in ('--shared-authority','--provisioning-recovery','--runtime-protocol'))
  assert not any(x in sys.argv for x in ('--post-adopt-loss','--initial-owner-loss','--initial-recovery','--partial-candidate'))
 provisioning='--provisioning-recovery' in sys.argv
 if provisioning:sys.argv.remove('--provisioning-recovery');assert '--shared-authority' in sys.argv
 protocol='--runtime-protocol' in sys.argv
 if protocol:sys.argv.remove('--runtime-protocol');assert '--shared-authority' in sys.argv
 cohort='all'
 if '--initial-cohort' in sys.argv:
  at=sys.argv.index('--initial-cohort');cohort=sys.argv[at+1];del sys.argv[at:at+2]
  assert '--initial-owner-loss' in sys.argv and cohort in ('before-issue','after-issue','omabit-only','theme-only','dms-only')
 post_adopt='--post-adopt-loss' in sys.argv
 if post_adopt:
  sys.argv.remove('--post-adopt-loss');assert '--shared-authority' in sys.argv and not any(x in sys.argv for x in ('--partial-candidate','--initial-recovery','--initial-owner-loss'))
 owner_loss='--initial-owner-loss' in sys.argv
 if owner_loss:
  sys.argv.remove('--initial-owner-loss');assert '--shared-authority' in sys.argv and '--partial-candidate' not in sys.argv and '--initial-recovery' not in sys.argv
 initial='--initial-recovery' in sys.argv
 if initial:
  sys.argv.remove('--initial-recovery');assert '--shared-authority' in sys.argv and '--partial-candidate' not in sys.argv
 partial=None
 if '--partial-candidate' in sys.argv:
  at=sys.argv.index('--partial-candidate');partial=sys.argv[at+1];del sys.argv[at:at+2]
  assert partial in ('before-issue','dms-only','dms-cleanup-refused','theme-cleanup-refused','all-cleanup-refused') and '--shared-authority' in sys.argv
 shared='--shared-authority' in sys.argv;bundle=None
 if shared:
  sys.argv.remove('--shared-authority');at=sys.argv.index('--wayland-runtime');bundle=Path(sys.argv[at+1]);del sys.argv[at:at+2]
  assert '--recovery' in sys.argv,'shared qualification requires original-guardian recovery'
 recovery='--recovery' in sys.argv
 if recovery:sys.argv.remove('--recovery')
 source=Path(__file__).resolve().parents[1];scripts=source/'scripts';artifact_module=None
 if shared:
  sys.path.insert(0,str(source/'native/collections'))
  if repeat_sessions:import repeated_artifacts as artifact_module
  elif continuation_activation:import continuation_artifacts as artifact_module
  elif successor_activation:import successor_artifacts as artifact_module
  elif protocol:import recovery_artifacts as artifact_module
  else:import shared_artifacts as artifact_module
 successor_pairs={};continuation_pairs={}
 if successor_upgrade:
  import recovery_artifacts as original_artifacts
  new_prepare=artifact_module.prepare
  def prepare_pair(*args,**kwargs):
   newer=new_prepare(*args,**kwargs)
   if continuation_activation:
    import successor_artifacts as middle_artifacts
    middle=middle_artifacts.prepare(*args,**dict(kwargs,expected_admission={n:kwargs['expected_admission'][n] for n in middle_artifacts.SUPPORT}))
    continuation_pairs[middle['artifact_sha256']]=newer;newer=middle
   older=original_artifacts.prepare(*args,**dict(kwargs,expected_admission={n:kwargs['expected_admission'][n] for n in original_artifacts.SUPPORT}))
   successor_pairs[older['artifact_sha256']]=newer
   return older
  artifact_module.prepare=prepare_pair
 actual=module('actual_native_fixture',scripts/'test-collection-omabit-guarded-native.py')
 if '--actual-pair' not in sys.argv:sys.argv.append('--actual-pair')
 original_run=actual.module('native_base_for_run',scripts/'test-collection-admission-native.py').subprocess.run
 def launch(command,*args,**kwargs):
  assert command[0]=='bwrap';out=Path(command[command.index('--bind')+1])
  common=module('hash_helpers',scripts/'test-collection-admission-native.py');prepared=common.read(out/'prepared.json')
  hashes=common.read(out/'source.json')
  if successor_upgrade:
   common.save(out/'successor-upgrade-prepared.json',{profile:{role:successor_pairs[versions['1']['artifact_sha256']] for role,versions in roles.items()} for profile,roles in prepared.items()})
   common.save(out/'successor-upgrade-mode.json',{'from_artifact':4,'to_artifact':5})
   if continuation_activation:
    common.save(out/'continuation-upgrade-prepared.json',{profile:{role:continuation_pairs[successor_pairs[versions['1']['artifact_sha256']]['artifact_sha256']] for role,versions in roles.items()} for profile,roles in prepared.items()})
    common.save(out/'continuation-activation-mode.json',{'from_artifact':5,'to_artifact':6,'authority':3})
  for profile,roles in prepared.items():
   for role in ('theme','dms'):
    dest=out/'artifacts'/profile/role;dest.parent.mkdir(parents=True,exist_ok=True)
    shutil.copytree(roles[role]['1']['path'],dest)
  names=['native_connected_fixture.py','connected_fixture.py','appearance_fixture.py','appearance_desktop.py']
  if recovery:
   names+=['native_recovery_runtime.py','native_recovery_fixture.py','journal_runtime_fixture.py','recovery_runtime.py','supervised_shell.py','process_supervisor.py']
   src=source/'native/collections/coordinator.py';shutil.copy2(src,out/'source/coordinator.py');hashes[str(src)]=common.sha(src)
   common.save(out/'native-recovery-mode.json',{'native_guarded_recovery':True})
  if shared:
   names+=['native_shared_runtime.py','wayland_runtime.py']
   if repeat_sessions:
    names+=['native_repeated_sessions_fixture.py'];common.save(out/'repeated-sessions-mode.json',{'sessions':3,'unchanged_artifact_format':7})
    for name in (*artifact_module.SUPPORT,'successor_artifacts.py','continuation_artifacts.py','repeated_artifacts.py','repeated_preflight.py'):
     src=source/'native/collections'/name;shutil.copy2(src,out/'source'/name);hashes[str(src)]=common.sha(src)
   if cleanup_continuation:
    names+=['native_continuation_cleanup_fixture.py'];common.save(out/'continuation-cleanup-mode.json',{'explicit_transport_cleanup':True})
    src=source/'native/collections/continuation_cleanup.py';shutil.copy2(src,out/'source/continuation_cleanup.py');hashes[str(src)]=common.sha(src)
   if inspect_continuation:
    names+=['native_continuation_inspection_fixture.py'];common.save(out/'continuation-inspection-mode.json',{'read_only':True})
   if continuation_activation:
    names+=['native_continuation_activation_fixture.py']
    for name in ('continuation_artifacts.py','continuation_preflight.py'):
     src=source/'native/collections'/name;shutil.copy2(src,out/'source'/name);hashes[str(src)]=common.sha(src)
   if continue_session:
    names+=['native_session_continuation_fixture.py'];common.save(out/'session-continuation-mode.json',{'linked_checkpoint_ownership_only':True})
    src=source/'native/collections/session_continuation.py';shutil.copy2(src,out/'source/session_continuation.py');hashes[str(src)]=common.sha(src)
   if checkpoint_successor:
    names+=['native_successor_checkpoint_fixture.py'];common.save(out/'successor-checkpoint-mode.json',{'non_admitting_linked_checkpoint':True})
    src=source/'native/collections/successor_checkpoint.py';shutil.copy2(src,out/'source/successor_checkpoint.py');hashes[str(src)]=common.sha(src)
   if cleanup_successor:
    names+=['native_successor_cleanup_fixture.py'];common.save(out/'successor-cleanup-mode.json',{'linked_transport_cleanup':True})
    src=source/'native/collections/successor_cleanup.py';shutil.copy2(src,out/'source/successor_cleanup.py');hashes[str(src)]=common.sha(src)
   if inspect_successor:
    names+=['native_successor_inspection_fixture.py'];common.save(out/'successor-inspection-mode.json',{'read_only':True})
   if successor_activation:
    names+=['native_successor_activation_fixture.py'];common.save(out/'successor-activation-mode.json',{'artifact_format':5,'linked_authority':2})
    for name in (*[n for n in artifact_module.SUPPORT if n.endswith(('.py','.json'))],'successor_artifacts.py','successor_preflight.py'):
     src=source/'native/collections'/name;shutil.copy2(src,out/'source'/name);hashes[str(src)]=common.sha(src)
   if session_successor:
    names+=['native_session_successor_fixture.py'];common.save(out/'session-successor-mode.json',{'linked_ownership_only':True})
    src=source/'native/collections/session_successor.py';shutil.copy2(src,out/'source/session_successor.py');hashes[str(src)]=common.sha(src)
   if session_checkpoint:
    names+=['native_session_checkpoint_fixture.py'];common.save(out/'session-checkpoint-mode.json',{'non_admitting_checkpoint':True})
    src=source/'native/collections/session_checkpoint.py';shutil.copy2(src,out/'source/session_checkpoint.py');hashes[str(src)]=common.sha(src)
   if session_cleanup:
    names+=['native_session_cleanup_fixture.py','native_candidate_recovery.py'];common.save(out/'session-cleanup-mode.json',{'explicit_transport_cleanup':True})
    for name in ('session_cleanup.py','authority_inspection.py'):
     src=source/'native/collections'/name;shutil.copy2(src,out/'source'/name);hashes[str(src)]=common.sha(src)
   if inspect_authority:
    names+=['native_authority_inspection_fixture.py'];common.save(out/'authority-inspection-mode.json',{'read_only':True})
    for relative,target in [('native/collections/authority_inspection.py','source/authority_inspection.py'),('native/collections/successor_inspection.py','source/successor_inspection.py'),('native/collections/continuation_inspection.py','source/continuation_inspection.py'),('scripts/inspect-collection-authority.py','inspect-collection-authority.py')]:
     src=source/relative;shutil.copy2(src,out/target);hashes[str(src)]=common.sha(src)
   if compositor_loss:
    names+=['native_compositor_loss_fixture.py'];common.save(out/'compositor-loss-mode.json',{'physical_compositor_replacement':True})
   if provisioning:
    names+=['native_provisioning_fixture.py'];common.save(out/'provisioning-mode.json',{'same_compositor_setup_recovery':True})
    src=source/'native/collections/authority_provisioning.py';shutil.copy2(src,out/'source/authority_provisioning.py');hashes[str(src)]=common.sha(src)
   if protocol:
    common.save(out/'runtime-protocol-mode.json',{'artifact_format':7 if repeat_sessions else 5 if successor_activation else 4})
    for name in ('recovery_artifacts.py','runtime_preflight.py','runtime-protocol.json'):
     src=source/'native/collections'/name;shutil.copy2(src,out/'source'/name);hashes[str(src)]=common.sha(src)
    frozen=source/'tests/fixtures/collection-artifact-v3';meta=common.read(frozen/'provenance.json')
    target=out/'home/.config/omarchy/plugins'/prepared['stable']['theme']['1']['manifest']['id']/'generations'/meta['artifact_sha256']
    shutil.copytree(frozen/'generation',target);common.save(out/'legacy-preflight.json',{'artifact_sha256':meta['artifact_sha256']})
    for file in (frozen/'generation').rglob('*'):
     if file.is_file():hashes[str(file)]=common.sha(file)

   if post_adopt:
    names+=['native_initial_fixture.py'];common.save(out/'post-adopt-loss-mode.json',{'unbound_initial_adoption':True})
   if owner_loss:
    names+=['native_initial_owner_fixture.py','native_initial_cleanup.py','native_initial_fixture.py','native_candidate_recovery.py']
    common.save(out/'initial-owner-mode.json',{'original_owner_loss':True,'cohort':cohort})
    src=source/'native/collections/startup_recovery.py';shutil.copy2(src,out/'source/startup_recovery.py');hashes[str(src)]=common.sha(src)
   if initial:
    names+=['native_initial_fixture.py'];common.save(out/'initial-recovery-mode.json',{'same_original_shell':True})
   if partial:
    names+=['native_partial_fixture.py','native_candidate_recovery.py'];common.save(out/'partial-candidate-mode.json',{'case':partial})
   for name in ('shared_artifacts.py','runtime_authority.py','shared_admission.py','shared_admission_guard.py'):
    src=source/'native/collections'/name;shutil.copy2(src,out/'source'/name);hashes[str(src)]=common.sha(src)
   common.save(out/'shared-recovery-mode.json',{'shared_authority':True,'reservation_format':2})
  for name in names:
   src=scripts/'qualification/collections'/name;shutil.copy2(src,out/name);hashes[str(src)]=common.sha(src)
  hashes[str(scripts/'test-collection-omabit-guarded-native.py')]=common.sha(scripts/'test-collection-omabit-guarded-native.py')
  common.save(out/'source.json',hashes);common.save(out/'native-connected-mode.json',{'synthetic_eyre':True,'private_keyring':True})
  path=out/'shell.qml';body=path.read_text().rstrip();assert body.endswith('}');path.write_text(body[:-1]+QML+'\n}\n')
  at=command.index('--')
  command[at:at]=['--setenv','OMABIT_REAL_PLUGIN_LIFECYCLE','private-v1','--setenv','OMABIT_LIFECYCLE_APPEARANCE','1']
  if recovery:
   at=command.index('--');command[at:at]=['--setenv','OMABIT_COLLECTION_JOURNAL','private-v1','--setenv','OMABIT_COLLECTION_EXPLICIT_RECOVERY','1','--setenv','OMABIT_SUPERVISOR_LAB','private-v1']
  at=command.index('--');command[at+1:at+1]=['/usr/bin/dbus-run-session','--']
  if shared:
   wayland=module('private_wayland_runtime',scripts/'qualification/collections/wayland_runtime.py');wayland.prepare(out,bundle,command,hashes)
  kwargs['timeout']=780 if repeat_sessions else 500 if cleanup_continuation else 450 if continuation_activation else 390 if recovery else 240
  common.save(out/'connected-command.json',command)
  return original_run(command,*args,**kwargs)
 # All three fixture modules share this subprocess module; the final callback
 # sees only the bounded bwrap command after actual source preparation is done.
 import subprocess
 subprocess.run=launch
 actual.__file__=__file__
 return actual.main(wayland=shared,artifact_module=artifact_module)

if __name__=='__main__':main()
