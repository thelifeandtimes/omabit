import importlib.util,json,os,shutil,subprocess,sys,time
from pathlib import Path
repo=Path(__file__).resolve().parents[3]
if '--inside' not in sys.argv:
 spec=importlib.util.spec_from_file_location('base',repo/'scripts/qualification/runtime-profiles/full_shell.py');m=importlib.util.module_from_spec(spec);spec.loader.exec_module(m)
 # Reuse the proven isolated compositor launcher; add only private DEV-shaped mounts.
 original=m.subprocess.run
 def launch(args,**kw):
  if args[0]=='bwrap':
   lab=Path(args[args.index('--bind')+1]);dev=lab/'dev';(dev/'desktop').mkdir(parents=True)
   for n in ['config','data','state','cache']:(dev/'desktop'/n).mkdir()
   driver=lab/'dev-driver.py';shutil.copyfile(__file__,driver)
   extra=['--dir','/srv','--dir','/srv/dev','--bind',str(dev),'/srv/dev/urbit-040','--ro-bind',str(Path.home()/'.config/omarchy/plugins/thelifeandtimes.omabit.review040-v050'),'/oldplugin','--ro-bind',str(repo/'build/review040-update-060rc3/plugin'),'/candidateplugin']
   i=args.index('--');args=args[:i]+extra+args[i:];args=[str(driver).replace(str(lab),'/lab') if a=='/lab/driver.py' else a for a in args]
  return original(args,**kw)
 m.subprocess.run=launch
 sys.argv=[__file__,'--output','/tmp/omabit-060-shell-dev01'];m.main();sys.exit()
lab=Path('/lab');home=lab/'home';pluginid='thelifeandtimes.omabit.review040';plugins=home/'.config/omarchy/plugins';plugins.mkdir(parents=True)
old=plugins/'thelifeandtimes.omabit.review040-v050';new=plugins/'thelifeandtimes.omabit.review040-v060rc3'
shutil.copytree('/oldplugin',old)
disabled=[json.loads(p.read_text())['id'] for p in Path('/usr/share/omarchy/shell/plugins').glob('*/manifest.json') if json.loads(p.read_text())['id']!='omarchy.bar']
config=home/'.config/omarchy/shell.json';config.write_text(json.dumps({'version':1,'idle':{'screensaver':0,'lock':0},'bar':{'position':'top','layout':{k:([{'id':pluginid}] if k=='right' else []) for k in ['left','centerBegin','center','centerEnd','right','tray']}},'plugins':[],'disabledPlugins':disabled}))
original=config.read_bytes();env=dict(os.environ,LD_LIBRARY_PATH='/tools/usr/lib',WLR_BACKENDS='headless',WLR_RENDERER='pixman',WLR_HEADLESS_OUTPUTS='1');compositor=shell=None
report={'passed':False,'host_access':False,'checks':{}}
def ipc(target,method,*args):return subprocess.check_output(['omarchy','shell',target,method,*args],env=env,text=True,stderr=subprocess.DEVNULL,timeout=8).strip()
def wait(version):
 end=time.monotonic()+60
 while time.monotonic()<end:
  try:
   s=json.loads(ipc(pluginid+'.review','status'))
   if s['loaded'] and s['version']==version:return s
  except (ValueError,subprocess.SubprocessError):pass
  time.sleep(.5)
 raise RuntimeError('UI activation deadline')
try:
 compositor=subprocess.Popen(['/tools/usr/bin/sway','--config','/lab/sway.conf'],env=env,stdout=(lab/'sway.log').open('w'),stderr=subprocess.STDOUT)
 deadline=time.monotonic()+15
 while not list((lab/'runtime').glob('wayland-*')):
  assert compositor.poll() is None and time.monotonic()<deadline;time.sleep(.1)
 env['WAYLAND_DISPLAY']=str(next(p for p in (lab/'runtime').glob('wayland-*') if p.is_socket()))
 shell=subprocess.Popen(['quickshell','--no-color','-p','/usr/share/omarchy/shell'],env=env,stdout=(lab/'shell.log').open('w'),stderr=subprocess.STDOUT)
 before=wait('0.5.0');assert before['profile']=='review-040' and not before['local_setup'];report['checks']['previous_wrapper_loads']=True
 assert ipc('shell','setPluginEnabled',pluginid,'false')=='ok'
 deadline=time.monotonic()+10
 while True:
  try:ipc(pluginid+'.review','status')
  except subprocess.SubprocessError:break
  assert time.monotonic()<deadline;time.sleep(.2)
 report['checks']['targeted_disable_unloads']=True;disabled_bytes=config.read_bytes()
 old.rename(lab/'old-retained');shutil.copytree('/candidateplugin',new)
 ipc('shell','rescanPlugins');assert config.read_bytes()==disabled_bytes
 config.write_bytes(original);ipc('shell','reloadConfig');ipc('shell','rescanPlugins')
 after=wait('0.6.0-rc.3');assert after['profile']=='review-040' and not after['local_setup'];assert after['collector']==str(new/'bin/urbit-collector')
 assert shell.poll() is None and config.read_bytes()==original
 report['checks'].update(candidate_versioned_wrapper_loads=True,profile_and_local_setup_boundary_preserved=True,shell_config_restored_exactly=True,no_shell_restart=True)
 report.update(before={k:before[k] for k in ['version','profile','collector','local_setup']},after={k:after[k] for k in ['version','profile','collector','local_setup']},passed=True)
finally:
 for child in [shell,compositor]:
  if child and child.poll() is None:child.terminate();child.wait(timeout=15)
 (lab/'result.json').write_text(json.dumps(report,indent=2)+'\n')
