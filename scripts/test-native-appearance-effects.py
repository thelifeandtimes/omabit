#!/usr/bin/env python3
"""Private native-command lifecycle characterization; never an installed desktop."""
import argparse,hashlib,json,os,re,shutil,signal,socket,subprocess,sys,time
from pathlib import Path
POST=['omarchy-restart-terminal','omarchy-restart-hyprctl','omarchy-restart-btop','omarchy-restart-opencode','omarchy-restart-helix','omarchy-theme-set-foot','omarchy-theme-set-tmux','omarchy-theme-set-gnome','omarchy-theme-set-pi','omarchy-theme-set-claude','omarchy-theme-set-hermes','omarchy-theme-set-t3code','omarchy-theme-set-browser','omarchy-theme-set-vscode','omarchy-theme-set-obsidian','omarchy-theme-set-keyboard']
NATIVE=['omarchy-theme-set','omarchy-theme-set-templates','omarchy-theme-color','omarchy-font-set','omarchy-restart-shell']
QML='''import QtQuick
import Quickshell
import Quickshell.Io
ShellRoot {
 Process {
  running: Quickshell.env("OMABIT_FONT_CHILD")==="1"
  command: ["/usr/bin/python3","-B","/lab/worker.py"]
 }
 IpcHandler {
  target: "nativeEffects"
  function ping():string {return "ready"}
  function stop():string {Qt.callLater(Qt.quit);return "stopping"}
 }
}
'''
WORKER='''import json,os,sys,time
from pathlib import Path
sys.path.insert(0,'/lab/theme')
from client.support import command
root=Path('/lab');case=os.environ['OMABIT_FONT_CASE']
(root/(case+'-worker.json')).write_text(json.dumps({'pid':os.getpid(),'phase':'applying'}))
deadline=time.monotonic()+25
while not (root/(case+'-go')).exists():
    assert time.monotonic()<deadline
    time.sleep(.025)
code,out=command(['/usr/bin/bash','/lab/native/bin/omarchy-font-set','Fixture Mono'],timeout=30)
(root/(case+'-worker-returned.json')).write_text(json.dumps({'exit':code,'font_file':(root/'home/.config/fontconfig/fonts.conf').read_text()}))
'''
SHIM='''#!/usr/bin/python3 -B
import json,os,signal,subprocess,sys,time
from pathlib import Path
r=Path('/lab');assert os.environ.get('OMABIT_NATIVE_EFFECTS')=='private-v1' and Path.home()==r/'home' and not Path('/home').exists()
name=Path(sys.argv[0]).name;args=sys.argv[1:];case=os.environ.get('OMABIT_FONT_CASE','theme')
def event(action,extra=None):
 v={'name':name,'action':action,'case':case,'pid':os.getpid(),'ppid':os.getppid(),'args':args};v.update(extra or {})
 with (r/'events.jsonl').open('a') as f:f.write(json.dumps(v)+'\\n')
event('start')
if name=='fc-list':print('Fixture Mono')
elif name=='omarchy-cmd-present':sys.exit(1)
elif name in ('pgrep','pkill'):
 assert args[-1] in ('kitty','ghostty','foot')
 sys.exit(1)
elif name=='systemctl':
 if args==['--user','show-environment']:print('OMARCHY_PATH=/lab/native')
 else:assert args==['--user','try-restart','omarchy-*-invitation.service']
elif name=='omarchy-hyprland-session-locked':sys.exit(1)
elif name=='hyprctl':
 assert args==['dispatch','hl.dsp.exec_cmd("omarchy-launch-shell")']
 env=dict(os.environ,OMABIT_FONT_CHILD='0')
 with (r/(case+'-replacement-shell.log')).open('wb') as out:
  p=subprocess.Popen(['/usr/bin/quickshell','--no-color','--path','/lab/native/shell'],env=env,stdout=out,stderr=subprocess.STDOUT,start_new_session=True)
 (r/(case+'-replacement.json')).write_text(json.dumps({'pid':p.pid}))
elif name=='omarchy-shell':
 assert args[:2]==['shell','ping'] or args[:2]==['shell','applyTheme']
 if args[:2]==['shell','ping']:
  file=r/(case+'-replacement.json')
  if not file.exists():sys.exit(1)
  pid=json.loads(file.read_text())['pid']
  q=subprocess.run(['/usr/bin/quickshell','ipc','--pid',str(pid),'call','--','nativeEffects','ping'],capture_output=True,text=True,timeout=2)
  if q.returncode or q.stdout.strip()!='ready':sys.exit(1)
  print('ready')
elif name=='omarchy-theme-bg-cache':
 (r/'cache-child.json').write_text(json.dumps({'pid':os.getpid()}))
 while not (r/'release-cache').exists():time.sleep(.025)
 event('cache-finished')
elif name=='omarchy-theme-switcher':assert args==['--preload']
elif name=='omarchy-hook':
 assert args[0] in ('theme-set','font-set')
 if args[0]=='font-set':(r/(case+'-font-hook')).write_text('complete')
else:
 assert name in POST
 if (r/'fail-retint').exists() and name=='omarchy-theme-set-vscode':event('failed',{'exit':73});sys.exit(73)
 time.sleep(.02)
event('exit')
'''
def sha(p):return hashlib.sha256(p.read_bytes()).hexdigest()
def identity(pid):
 try:
  fields=Path('/proc/'+str(pid)+'/stat').read_text().split(') ',1)[1].split();return {'pid':pid,'start':fields[19],'state':fields[0]}
 except FileNotFoundError:return None
def gone(old):
 now=identity(old['pid']);return not now or now['state']=='Z' or now['start']!=old['start']
def inside():
 root=Path('/lab')
 if os.environ.get('OMABIT_NATIVE_EFFECTS')!='private-v1' or Path.home()!=root/'home' or Path('/home').exists() or Path('/run/user').exists():raise RuntimeError('private native-effects namespace required')
 ns=json.loads((root/'host-namespaces.json').read_text());assert all(os.readlink('/proc/self/ns/'+n)!=ns[n] for n in ns);assert [n for _,n in socket.if_nameindex()]==['lo']
 sys.path.insert(0,'/lab/theme');from client.support import command
 report={'passed':False,'copied_native_scripts':True,'real_theme_command_runner':True,'real_private_quickshell':True,'real_plugins':False,'actual_theme_restore_cli':os.environ.get('OMABIT_OFFLINE_NATIVE_RESTORE')=='1','actual_compositor':False,'retint_effects_inert':True,'font_availability_stub':True,'cases':[]}
 def save():(root/'result.json').write_text(json.dumps(report,indent=2)+'\n')
 def wait(fn,seconds=15):
  end=time.monotonic()+seconds
  while time.monotonic()<end:
   result=fn()
   if result:return result
   time.sleep(.04)
  raise AssertionError('private native effects deadline')
 def ipc(pid,method):
  q=subprocess.run(['/usr/bin/quickshell','ipc','--pid',str(pid),'call','--','nativeEffects',method],capture_output=True,text=True,timeout=3)
  return q.returncode,q.stdout.strip()
 save()
 # Positive control: the native setter returns while its detached cache lives.
 raw=subprocess.run(['/usr/bin/bash','/lab/native/bin/omarchy-theme-set','fixture'],capture_output=True,text=True,start_new_session=True,timeout=30)
 assert raw.returncode==0
 wait(lambda:(root/'cache-child.json').exists())
 cache=identity(json.loads((root/'cache-child.json').read_text())['pid']);assert cache and cache['state']!='Z'
 report['native_control']={'exit':raw.returncode,'returned_with_background_cache_alive':True};save()
 (root/'release-cache').write_text('finish');wait(lambda:gone(cache))
 (root/'cache-child.json').unlink();(root/'release-cache').unlink()
 cleanup=[];killpg=os.killpg
 def observed_killpg(group,sig):
  cleanup.append({'group':group,'signal':int(sig)})
  return killpg(group,sig)
 os.killpg=observed_killpg
 # The script dispatch is real; application-specific effects are inert probes.
 for failing in (False,True):
  if failing:(root/'fail-retint').write_text('yes')
  start=len((root/'events.jsonl').read_text().splitlines())
  code,out=command(['/usr/bin/bash','/lab/native/bin/omarchy-theme-set','fixture'],timeout=30)
  assert cleanup[-1]['signal']==signal.SIGKILL
  marker=root/'cache-child.json'
  child=identity(json.loads(marker.read_text())['pid']) if marker.exists() else None
  if child:wait(lambda:gone(child))
  events=[json.loads(x) for x in (root/'events.jsonl').read_text().splitlines()[start:]]
  assert code==0 and not any(x['action']=='cache-finished' for x in events)
  assert any(x['name']=='omarchy-theme-set-vscode' and x['action']=='failed' for x in events)==failing
  report['cases'].append({'kind':'native-theme-fanout','early_retint_failure':failing,'native_exit':code,'command_group_cleanup':cleanup[-1],'background_cache_started':marker.exists(),'background_cache_completed':False,'command_zero_does_not_prove_all_retints':failing});save()
  if marker.exists():marker.unlink()
 os.killpg=killpg
 # Native restart kills the real owning shell; the dispatcher is an inert
 # Hyprland replacement spawning only another private fixture shell.
 for case,owned in [('owned',True),('external',False)]:
  env=dict(os.environ,OMABIT_FONT_CASE=case,OMABIT_FONT_CHILD='1' if owned else '0')
  worker=None;replacement=None
  with (root/(case+'-shell.log')).open('wb') as log:
   shell=subprocess.Popen(['/usr/bin/quickshell','--no-color','--path','/lab/native/shell'],env=env,stdout=log,stderr=subprocess.STDOUT,start_new_session=True)
   try:
    wait(lambda:ipc(shell.pid,'ping')==(0,'ready'))
    if not owned:worker=subprocess.Popen(['/usr/bin/python3','-B','/lab/worker.py'],env=env,stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL)
    wait(lambda:(root/(case+'-worker.json')).exists());worker_id=identity(json.loads((root/(case+'-worker.json')).read_text())['pid']);assert worker_id
    (root/(case+'-go')).write_text('start')
    wait(lambda:shell.poll() is not None,35)
    wait(lambda:(root/(case+'-replacement.json')).exists(),35)
    replacement=identity(json.loads((root/(case+'-replacement.json')).read_text())['pid']);assert replacement
    wait(lambda:ipc(replacement['pid'],'ping')==(0,'ready'))
    wait(lambda:(root/(case+'-font-hook')).exists(),35)
    wait(lambda:gone(worker_id),35)
    if worker:assert worker.wait(timeout=3)==0
    font=(root/'home/.config/fontconfig/fonts.conf').read_text();assert '<string>Fixture Mono</string>' in font
    returned=root/(case+'-worker-returned.json')
    if not owned:assert json.loads(returned.read_text())['exit']==0
    else:assert not returned.exists(),'Shell-owned helper unexpectedly completed; update the lifecycle characterization'
    assert shell.returncode==0
    report['cases'].append({'kind':'native-font-restart','helper_owned_by_shell':owned,'old_shell_exit':shell.returncode,'old_helper_exited':True,'font_file_written':True,'new_private_shell_ready':True,'font_hook_completed_after_restart':True,'worker_recorded_result':returned.exists(),'replacement':replacement,'replacement_stopped':False});save()
    assert ipc(replacement['pid'],'stop')==(0,'stopping');wait(lambda:gone(replacement));replacement=None
    report['cases'][-1]['replacement_stopped']=True;save()
   finally:
    if shell.poll() is None:shell.terminate();shell.wait(timeout=4)
    if worker and worker.poll() is None:worker.terminate();worker.wait(timeout=4)
    if replacement and not gone(replacement):ipc(replacement['pid'],'stop');wait(lambda:gone(replacement),4)
 if os.environ.get('OMABIT_OFFLINE_NATIVE_RESTORE')=='1':
  # Compose the actual saved-appearance CLI with native font/shell restart.
  # No ship or keyring is present; the fixture seeds local verified state only.
  (root/'restart-control.py').write_text((root/'bin/hyprctl').read_text())
  (root/'home/.local/state/omarchy/current').rename(root/'home/.local/state/omarchy/before-offline-restore')
  os.environ['OMABIT_LIFECYCLE_APPEARANCE']='1';os.environ['OMABIT_NATIVE_THEME_STAGING']='1'
  from appearance_fixture import prepare,profile
  prepare(root)
  from client.profile import Desktop
  from client.checkpoints import Checkpoints
  from client.sync import Disk,fresh
  d=Desktop();store=Disk();desired=profile('stable');d.apply(desired)
  state=fresh();state.update(observed=d.capture(state['deviceId']),pending=profile('edge'),attempted=True,baseId='e'*32);store.save(state)
  checkpoint=Checkpoints(d).capture(state)['id']
  # A later candidate changes appearance; its pending application is retained.
  d.apply(profile('dev'));state['applying']=profile('dev');store.save(state)
  journal=sha(store.path)
  font=root/'home/.config/fontconfig/fonts.conf';font.write_text('<?xml version="1.0"?><fontconfig><match><edit><string>Candidate Mono</string></edit></match></fontconfig>')
  env=dict(os.environ,OMABIT_FONT_CASE='restore',OMABIT_FONT_CHILD='0',OMABIT_NATIVE_FONT_RESTART='1')
  with (root/'restore-shell.log').open('wb') as log:
   shell=subprocess.Popen(['/usr/bin/quickshell','--no-color','--path','/lab/native/shell'],env=env,stdout=log,stderr=subprocess.STDOUT,start_new_session=True)
   replacement=None
   try:
    wait(lambda:ipc(shell.pid,'ping')==(0,'ready'))
    (root/'fault.json').write_text(json.dumps({'profile':'effects','stage':'font'}))
    guard="""import runpy,sys
from pathlib import Path
script=sys.argv[1];sys.argv=sys.argv[1:];sys.path.insert(0,str(Path(script).parent))
def audit(event,args):
    if event in ('socket.connect','socket.getaddrinfo','socket.bind') or event=='subprocess.Popen' and Path(args[0]).name in ('secret-tool','busctl'):
        Path('/lab/restore-forbidden-access').write_text(event)
        raise RuntimeError('Offline restore attempted network or credentials')
sys.addaudithook(audit)
runpy.run_path(script,run_name='__main__')
"""
    request='f'*32;args=['/usr/bin/python3','-B','-c',guard,'/lab/theme/client/sync.py','restore-appearance',checkpoint,request]
    first=subprocess.run(args,env=env,capture_output=True,text=True,timeout=70)
    assert first.returncode==1 and json.loads(first.stdout)['status']=='Needs attention',first.stdout
    assert shell.wait(timeout=4)==0
    wait(lambda:(root/'restore-replacement.json').exists())
    replacement=identity(json.loads((root/'restore-replacement.json').read_text())['pid']);assert replacement
    wait(lambda:ipc(replacement['pid'],'ping')==(0,'ready'))
    receipt=d.appearance/'restore-receipts'/(request+'.json');first_receipt=json.loads(receipt.read_text());assert first_receipt['phase']=='applying'
    assert (root/'restore-font-hook').exists() and sha(store.path)==journal and '<string>Stable Mono</string>' in font.read_text()
    (root/'fault.json').write_text('{}')
    second=subprocess.run(args,env=env,capture_output=True,text=True,timeout=70)
    assert second.returncode==0 and json.loads(second.stdout)['appearanceRestoration']['phase']=='verified',second.stdout
    events=sha(root/'desktop-events.jsonl')
    third=subprocess.run(args,env=env,capture_output=True,text=True,timeout=15)
    assert third.returncode==0 and sha(root/'desktop-events.jsonl')==events and sha(store.path)==journal
    assert not (root/'restore-forbidden-access').exists()
    assert (root/'home/.local/state/omarchy/current/theme.name').read_text().strip()=='fixture-stable'
    report['actual_offline_restore']={'checkpoint':checkpoint,'request':request,'exits':[first.returncode,second.returncode,third.returncode],'native_font_file_written':True,'native_old_shell_exited':True,'new_private_shell_ready':True,'failed_after_native_restart_retains_applying':True,'fresh_retry_verified':True,'success_replay_no_effects':True,'pending_sync_journal_unchanged':True,'journal_sha256_before':journal,'journal_sha256_after':sha(store.path),'failed_receipt':first_receipt,'cli_results':[json.loads(first.stdout),json.loads(second.stdout),json.loads(third.stdout)],'network_and_credentials_forbidden':True,'replacement_stopped':False};save()
    assert ipc(replacement['pid'],'stop')==(0,'stopping');wait(lambda:gone(replacement));replacement=None
    report['actual_offline_restore']['replacement_stopped']=True;save()
   finally:
    if shell.poll() is None:shell.terminate();shell.wait(timeout=4)
    if replacement and not gone(replacement):ipc(replacement['pid'],'stop');wait(lambda:gone(replacement),4)
 report['passed']=True;save();print('PASS: actual offline Theme restoration survives native font/shell restart and failed-effect retry' if os.environ.get('OMABIT_OFFLINE_NATIVE_RESTORE')=='1' else 'PASS: native command fanout and shell-owned versus external font restart characterized')
def main():
 p=argparse.ArgumentParser(description=__doc__);p.add_argument('--inside',action='store_true');p.add_argument('--output',type=Path);p.add_argument('--theme-source',type=Path);p.add_argument('--offline-restore',action='store_true',help='also qualify actual offline Theme restore through native font/shell restart');p.add_argument('--collections-source',type=Path,default=Path(__file__).parent/'qualification/collections');a=p.parse_args()
 if a.inside:return inside()
 work=a.output
 if not work or work.parent!=Path('/tmp') or work.exists() or not re.fullmatch('omabit-native-effects-[a-zA-Z0-9-]+',work.name):p.error('use a fresh /tmp/omabit-native-effects-NAME')
 if not a.theme_source:p.error('--theme-source is required outside the private entrypoint')
 source=a.theme_source.resolve();assert (source/'client/support.py').is_file()
 work.mkdir(mode=0o700)
 for n in ('home','runtime','cache','etc','bin','native/bin','native/shell','theme/client'):(work/n).mkdir(parents=True,mode=0o700)
 for n in ('passwd','group','nsswitch.conf'):(work/'etc'/n).write_text('')
 (work/'etc/passwd').write_text('lab:x:'+str(os.getuid())+':'+str(os.getgid())+':Lab:/lab/home:/bin/sh\n');(work/'etc/group').write_text('lab:x:'+str(os.getgid())+':\n');(work/'etc/nsswitch.conf').write_text('passwd: files\ngroup: files\n')
 (work/'host-namespaces.json').write_text(json.dumps({n:os.readlink('/proc/self/ns/'+n) for n in ('pid','net','user','mnt')}));shutil.copy2(__file__,work/'driver.py')
 (work/'native/shell/shell.qml').write_text(QML);(work/'worker.py').write_text(WORKER)
 sources={}
 for n in NATIVE:
  src=Path('/usr/share/omarchy/bin')/n;shutil.copy2(src,work/'native/bin'/n);sources[str(src)]=sha(src)
 declared=re.search(r'post_theme_commands=\((.*?)\n\)',(work/'native/bin/omarchy-theme-set').read_text(),re.S)
 assert declared and declared[1].split()==POST,'Native theme fanout changed; review fixture adapters before running'
 shutil.copytree('/usr/share/omarchy/default/themed',work/'native/default/themed')
 for src in Path('/usr/share/omarchy/default/themed').iterdir():assert src.is_file() and not src.is_symlink();sources[str(src)]=sha(src)
 for src in (source/'client').glob('*.py'):shutil.copy2(src,work/'theme/client'/src.name);sources[str(src)]=sha(src)
 for src in (source/'manifest.json',):shutil.copy2(src,work/'theme'/src.name);sources[str(src)]=sha(src)
 meta=json.loads((work/'theme/manifest.json').read_text());meta['id']+='.qualification-effects';meta['name']+=' (qualification effects)';(work/'theme/manifest.json').write_text(json.dumps(meta));(work/'theme/application-profile.json').write_text(json.dumps({'version':1,'profile':'qualification-effects'}))
 for n in POST+['omarchy-shell','omarchy-hook','omarchy-theme-switcher','omarchy-theme-bg-cache','fc-list','omarchy-cmd-present','pgrep','pkill','systemctl','omarchy-hyprland-session-locked','hyprctl']:
  f=work/'bin'/n;f.write_text(SHIM.replace("r=Path('/lab');",'POST='+repr(POST)+"\nr=Path('/lab');"));f.chmod(0o700)
 theme=work/'home/.config/omarchy/themes/fixture';theme.mkdir(parents=True);(theme/'colors.toml').write_text('mode="dark"\nbackground="#112233"\nforeground="#EEEEEE"\nblue="#123ABC"\ngreen="#33CC66"\nred="#EE3333"\n')
 (work/'home/.local/state/omarchy/current').mkdir(parents=True)
 if a.offline_restore:
  for name in ('appearance_fixture.py','appearance_desktop.py','connected_fixture.py'):
   src=a.collections_source/name;shutil.copy2(src,work/name);sources[str(src)]=sha(src)
  shim=work/'appearance_desktop.py';value=shim.read_text()
  value=value.replace("name,args=Path(sys.argv[0]).name,sys.argv[1:]", "name,args=Path(sys.argv[0]).name,sys.argv[1:]\nif name=='hyprctl' and args[:1]==['dispatch'] or name=='omarchy-shell' and args[:2]==['shell','ping']:\n exec((r/'restart-control.py').read_text());sys.exit(0)")
  value=value.replace("elif args==['font','current']:print(value['font'])", "elif args==['font','current']:\n   if os.environ.get('OMABIT_NATIVE_FONT_RESTART')=='1':\n    import xml.etree.ElementTree as ET\n    print(ET.parse(h/'.config/fontconfig/fonts.conf').findtext('./match/edit/string'))\n   else:print(value['font'])")
  needle="assert os.environ['OMARCHY_URBIT_THEME_REMOTE']=='1';value['font']=args[2];stage='font'"
  replacement=needle+"\n   if os.environ.get('OMABIT_NATIVE_FONT_RESTART')=='1':\n    fcntl.flock(lock,fcntl.LOCK_UN)\n    q=subprocess.run(['/usr/bin/bash','/lab/native/bin/omarchy-font-set',args[2]],capture_output=True,text=True,timeout=45)\n    fcntl.flock(lock,fcntl.LOCK_EX)\n    assert q.returncode==0,('native font restart',q.returncode,q.stderr[-500:])"
  assert value.count(needle)==1;value=value.replace(needle,replacement);shim.write_text(value)
 (work/'source.json').write_text(json.dumps({'source_sha256':sources,'native_scripts':NATIVE,'application_commands_inert':POST,'no_production_access':True},indent=2)+'\n')
 cmd=['bwrap','--unshare-all','--die-with-parent','--clearenv','--ro-bind','/usr','/usr','--symlink','usr/bin','/bin','--symlink','usr/lib','/lib','--symlink','usr/lib','/lib64','--proc','/proc','--dev','/dev','--tmpfs','/tmp','--ro-bind',str(work/'etc'),'/etc','--bind',str(work),'/lab','--ro-bind',str(work/'native'),'/lab/native','--ro-bind',str(work/'theme'),'/lab/theme','--chdir','/lab']
 for k,v in {'HOME':'/lab/home','XDG_RUNTIME_DIR':'/lab/runtime','XDG_CACHE_HOME':'/lab/cache','PATH':'/lab/bin:/lab/native/bin:/usr/bin:/bin','LANG':'C.UTF-8','QT_QPA_PLATFORM':'offscreen','QT_QUICK_BACKEND':'software','QML_DISABLE_DISK_CACHE':'1','OMARCHY_PATH':'/lab/native','OMARCHY_THEME_SKIP_BACKGROUND':'1','HYPRLAND_INSTANCE_SIGNATURE':'private-fixture','OMABIT_NATIVE_EFFECTS':'private-v1','OMABIT_OFFLINE_NATIVE_RESTORE':'1' if a.offline_restore else '0','PYTHONDONTWRITEBYTECODE':'1'}.items():cmd+=['--setenv',k,v]
 cmd+=['--','/usr/bin/python3','-B','/lab/driver.py','--inside'];(work/'command.json').write_text(json.dumps(cmd,indent=2)+'\n')
 q=subprocess.run(cmd,capture_output=True,text=True,timeout=190);(work/'driver.log').write_text(q.stdout+q.stderr);print(q.stdout+q.stderr);return q.returncode
if __name__=='__main__':sys.exit(main())
