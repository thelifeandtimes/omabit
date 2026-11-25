"""Exact release bundles in an unmodified Omarchy shell on a private compositor."""
import argparse,hashlib,importlib.util,json,os,shutil,subprocess,sys,time,traceback
from pathlib import Path
REPO=Path('/repo') if '--inside' in sys.argv else Path(__file__).resolve().parents[3]
PLUGIN='thelifeandtimes.omabit.review060shell'

def run(args,**kw):return subprocess.run(list(map(str,args)),check=True,**kw)
def sha(p):
 with p.open('rb') as f:return hashlib.file_digest(f,'sha256').hexdigest()
def inside():
 lab=Path('/lab');assert Path.home()==lab/'home' and not Path('/run/user').exists()
 report={'passed':False,'full_omarchy_shell':True,'host_desktop_access':False,'network':False,'service_upgrade':False,'checks':{}}
 env=dict(os.environ,LD_LIBRARY_PATH='/tools/usr/lib',WLR_BACKENDS='headless',WLR_RENDERER='pixman',WLR_HEADLESS_OUTPUTS='1')
 compositor=None;shell=None
 try:
  compositor=subprocess.Popen(['/tools/usr/bin/sway','--config','/lab/sway.conf'],env=env,stdout=(lab/'sway.log').open('w'),stderr=subprocess.STDOUT)
  end=time.monotonic()+15
  while True:
   assert compositor.poll() is None,'compositor exited'
   displays=[p for p in (lab/'runtime').glob('wayland-*') if p.is_socket()]
   if displays:break
   assert time.monotonic()<end,'compositor deadline';time.sleep(.1)
  env['WAYLAND_DISPLAY']=str(displays[0]);os.environ.update(env)
  plugin=lab/'home/.config/omarchy/plugins'/PLUGIN;plugin.mkdir(parents=True)
  shutil.copytree('/repo/native',plugin/'native',ignore=shutil.ignore_patterns('__pycache__'))
  metadata={key:json.loads((Path('/'+key)/'release.json').read_text()) for key in ('previous','candidate')}
  sys.path.insert(0,str(plugin/'native/bootstrap'));import desktop
  pins=plugin/'native/bootstrap/release.json';pins.write_text(json.dumps(desktop.descriptor_from_release(metadata['previous'])))
  d=desktop.Desktop(pins);d.create_profile('review060');d.select('review060')
  imports={}
  for key in ('previous','candidate'):
   assets=Path('/'+key);v=metadata[key]['version'];imports[key]=d.import_release(assets/'release.json',assets/('omabit-'+v+'-linux-amd64.tar.gz'))['release']
  # Namespace adapters inject only review metadata/properties; sealed UI/tool bytes stay exact.
  manifest=json.loads(Path('/repo/manifest.json').read_text());manifest.update(id=PLUGIN,name='Omabit 0.6 REVIEW',entryPoints={'service':'ReviewService.qml','barWidget':'ReviewPanel.qml'})
  (plugin/'manifest.json').write_text(json.dumps(manifest))
  (plugin/'ReviewService.qml').write_text('''import QtQuick
import Quickshell.Io
import "native/v1" as Native
Native.Service {
 id: review
 onPayloadChanged: if(payload) payload.manifest=Object.assign({},payload.manifest,{id:"'''+PLUGIN+'''",name:"Omabit 0.6 REVIEW"})
 IpcHandler {
  target:"'''+PLUGIN+'''.qualification"
  function status(): string { return JSON.stringify({bootstrap:review.diagnostic(),session:review.session,profile:review.payload?review.payload.profileName:"",local_setup:review.payload?review.payload.localSetupAllowed:true,page:review.payload?review.payload.monitorWindow.page:"",selection:review.payload?review.payload.monitorWindow.selectedKey:"",visible:review.payload?review.payload.monitorWindow.visible:false,module:review.payload?review.payload.moduleName:""}) }
  function view(page:string, selection:string): void { if(review.payload) { review.payload.monitorWindow.page=page;review.payload.monitorWindow.selectedKey=selection;review.payload.openMonitor(null,null,page) } }
  function close(): void { if(review.payload)review.payload.monitorWindow.visible=false }
 }
}
''')
  (plugin/'ReviewPanel.qml').write_text('''import QtQuick
import Quickshell.Io
import "native/v1" as Native
Native.Panel {
 id: review
 moduleName:"'''+PLUGIN+'''"
 onImplementationChanged: if(implementation)implementation.moduleName=moduleName
 IpcHandler {target:"'''+PLUGIN+'''.widget"
  function status(): string { return JSON.stringify({ready:!!review.implementation,opened:review.opened,paired:!!review.service && review.bootstrap.payload===review.service}) }
  function open(): void {review.open()}
  function close(): void {review.close()}
 }
}
''')
  disabled=[json.loads(p.read_text())['id'] for p in Path('/usr/share/omarchy/shell/plugins').glob('*/manifest.json') if json.loads(p.read_text())['id']!='omarchy.bar']
  cfg={'version':1,'idle':{'screensaver':0,'lock':0},'bar':{'position':'top','layout':{k:([{'id':PLUGIN}] if k=='right' else []) for k in ['left','centerBegin','center','centerEnd','right','tray']}},'plugins':[],'disabledPlugins':disabled}
  (lab/'home/.config/omarchy/shell.json').write_text(json.dumps(cfg))
  def ipc(target,method,*args):
   r=subprocess.run(['omarchy','shell',target,method,*args],env=env,capture_output=True,text=True,timeout=8)
   if r.returncode:raise RuntimeError(r.stderr[-500:])
   return r.stdout.strip()
  def status():return json.loads(ipc(PLUGIN+'.qualification','status'))
  def wait(version):
   deadline=time.monotonic()+60;last=None
   while time.monotonic()<deadline:
    assert shell.poll() is None,'full shell exited'
    try:
     last=status()
     if last['bootstrap']['loaded_version']==version and last['bootstrap']['phase']=='ready' and not last['bootstrap']['error'] and json.loads(ipc(PLUGIN+'.widget','status'))['ready']:return last
    except (RuntimeError,ValueError):pass
    time.sleep(.5)
   raise RuntimeError('activation deadline: '+str(last))
  def start(label):return subprocess.Popen(['quickshell','--no-color','-p','/usr/share/omarchy/shell'],env=env,stdout=(lab/(label+'.log')).open('w'),stderr=subprocess.STDOUT)
  shell=start('shell-cold');before=wait(metadata['previous']['version'])
  assert before['module']==PLUGIN and before['profile']=='review060' and before['local_setup'] is False
  report['checks']['cold_previous_full_shell']=True
  ipc(PLUGIN+'.widget','open');assert json.loads(ipc(PLUGIN+'.widget','status'))['opened'];ipc(PLUGIN+'.widget','close')
  ipc(PLUGIN+'.qualification','view','setup','review-runner/preserved-selection');time.sleep(1)
  before=status();report['before']=before;pid=shell.pid
  pins.with_suffix('.new').write_text(json.dumps(desktop.descriptor_from_release(metadata['candidate'])));pins.with_suffix('.new').replace(pins)
  after=wait(metadata['candidate']['version']);assert shell.pid==pid and before['session']==after['session']
  assert after['page']=='setup' and after['selection']=='review-runner/preserved-selection' and after['visible']
  assert after['bootstrap']['loaded_release']==imports['candidate']
  report['after']=after;report['checks']['hot_update_same_shell_preferences_and_pin']=True
  ipc('shell','rescanPlugins');time.sleep(2);rescanned=wait(metadata['candidate']['version']);assert rescanned['session']==after['session']
  assert json.loads(ipc(PLUGIN+'.widget','status'))['paired'];report['checks']['rescan_preserves_paired_generation']=True
  ipc(PLUGIN+'.qualification','close');shell.terminate();shell.wait(timeout=15)
  shell=start('shell-restarted');restarted=wait(metadata['candidate']['version']);assert restarted['bootstrap']['loaded_release']==imports['candidate'] and restarted['session']!=after['session']
  assert restarted['selection']=='review-runner/preserved-selection'
  report['restarted']=restarted;report['checks']['cold_restart_retains_candidate_and_preferences']=True
  report['artifact_sources']={k:v['source_commit'] for k,v in metadata.items()};report['package_hashes']={k:sha(Path('/'+k)/('omabit-'+v['version']+'-linux-amd64.tar.gz')) for k,v in metadata.items()}
  report['namespace_adapters']={p.name:sha(p) for p in [plugin/'ReviewService.qml',plugin/'ReviewPanel.qml']}
  report['shell_sha256']=sha(Path('/usr/share/omarchy/shell/shell.qml'));report['shell_version']=subprocess.check_output(['quickshell','--version'],text=True).strip()
  report['passed']=True
 except Exception as e:report.update(error=str(e),traceback=traceback.format_exc());raise
 finally:
  for child in [shell,compositor]:
   if child and child.poll() is None:
    child.terminate()
    try:child.wait(timeout=10)
    except subprocess.TimeoutExpired:child.kill();child.wait(timeout=5)
  (lab/'result.json').write_text(json.dumps(report,indent=2)+'\n')

def main():
 p=argparse.ArgumentParser();p.add_argument('--inside',action='store_true');p.add_argument('--output',type=Path)
 p.add_argument('--previous',type=Path,default=Path('/tmp/omabit-050-public-downloads/assets'))
 p.add_argument('--candidate',type=Path,default=Path('/tmp/omabit-runtime-profiles-review-assets/0.6.0-rc.4'));a=p.parse_args()
 if a.inside:return inside()
 if a.output is None:p.print_help();return
 out=a.output;assert out.parent==Path('/tmp') and not out.exists() and out.name.startswith('omabit-060-shell-')
 spec=importlib.util.spec_from_file_location('budget',REPO/'scripts/qualification/runtime-profiles/run.py');budget=importlib.util.module_from_spec(spec);spec.loader.exec_module(budget)
 import fcntl
 lock=open('/tmp/omabit-issues-vm-runtime.lock','rb');fcntl.flock(lock,fcntl.LOCK_EX|fcntl.LOCK_NB)
 admission=budget.full_charge(1024**3,48*1024**3)
 out.mkdir(mode=0o700)
 for name in ['home','runtime','config','state','data','cache','etc']:(out/name).mkdir(mode=0o700)
 (out/'etc/fonts').mkdir();(out/'etc/passwd').write_text(f'lab:x:{os.getuid()}:{os.getgid()}:Lab:/lab/home:/bin/sh\n');(out/'etc/group').write_text(f'lab:x:{os.getgid()}:\n');(out/'etc/nsswitch.conf').write_text('passwd: files\ngroup: files\n')
 (out/'sway.conf').write_text('xwayland disable\nswaybg_command -\noutput HEADLESS-1 mode 1280x960\n');shutil.copyfile(__file__,out/'driver.py')
 tools=Path.home()/'Documents/ChatGPT/omabit-qualification-evidence/reboot-preservation-20261120-050/tmp/omabit-040-wayland-tools/root'
 cmd=['bwrap','--unshare-all','--die-with-parent','--clearenv','--ro-bind','/usr','/usr','--symlink','usr/bin','/bin','--symlink','usr/lib','/lib','--symlink','usr/lib','/lib64','--proc','/proc','--dev','/dev','--tmpfs','/tmp','--dir','/run','--ro-bind',out/'etc','/etc','--ro-bind','/etc/fonts','/etc/fonts','--ro-bind',tools,'/tools','--ro-bind',REPO,'/repo','--ro-bind',a.previous.resolve(),'/previous','--ro-bind',a.candidate.resolve(),'/candidate','--bind',out,'/lab','--chdir','/repo']
 for k,v in {'PATH':'/usr/bin:/bin','LANG':'C.UTF-8','HOME':'/lab/home','XDG_RUNTIME_DIR':'/lab/runtime','XDG_CONFIG_HOME':'/lab/home/.config','XDG_STATE_HOME':'/lab/state','XDG_DATA_HOME':'/lab/data','XDG_CACHE_HOME':'/lab/cache','QT_QUICK_BACKEND':'software','QT_QPA_PLATFORMTHEME':'','QML_DISABLE_DISK_CACHE':'1','OMARCHY_PATH':'/usr/share/omarchy','XDG_SESSION_TYPE':'wayland'}.items():cmd+=['--setenv',k,v]
 (out/'host.json').write_text(json.dumps({'admission':admission,'driver_sha256':sha(Path(__file__))},indent=2))
 r=subprocess.run(list(map(str,cmd+['--','dbus-run-session','--','python3','-B','/lab/driver.py','--inside'])),timeout=270)
 print(json.dumps({'exit':r.returncode,'output':str(out)}));raise SystemExit(r.returncode)
if __name__=='__main__':main()
