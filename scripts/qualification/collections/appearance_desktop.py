"""Inert Omarchy/Hyprland executable substitutes, only in the guarded private lab."""
import base64,fcntl,json,os,re,signal,subprocess,sys,tomllib
from pathlib import Path
r=Path('/lab');h=r/'home';current=h/'.local/state/omarchy/current'
assert os.environ.get('OMABIT_LIFECYCLE_APPEARANCE')=='1' and Path.home()==h and not Path('/home').exists()
name,args=Path(sys.argv[0]).name,sys.argv[1:]
def save(v):
 p=r/'desktop.next';p.write_text(json.dumps(v));p.replace(r/'desktop.json')
def events(v):
 with (r/'desktop-events.jsonl').open('a') as f:f.write(json.dumps(v)+'\n')
with (r/'desktop-command.lock').open('a') as lock:
 fcntl.flock(lock,fcntl.LOCK_EX)
 value=json.loads((r/'desktop.json').read_text());stage=None
 if name=='fc-list': print('Stable Mono,Candidate Mono,Fixture Mono')
 elif name=='omarchy':
  if args==['theme','color','--all']:print('\n'.join(k+'\t'+v for k,v in value['colors'].items()))
  elif args==['theme','color','--file',str(current/'theme/colors.toml'),'--all']:
   colors=tomllib.loads((current/'theme/colors.toml').read_text());print('\n'.join(k+'\t'+v for k,v in colors.items()))
  elif args==['font','current']:print(value['font'])
  elif args[:2]==['font','set']:
   assert os.environ['OMARCHY_URBIT_THEME_REMOTE']=='1';value['font']=args[2];stage='font'
  elif args[:2]==['theme','set']:
   assert os.environ['OMARCHY_URBIT_THEME_REMOTE']=='1' and (args[2].startswith('urbit-synced.qualification-') or re.fullmatch('urbit-restored-[a-f0-9]{64}',args[2]))
   if args[2].startswith('urbit-restored-'):assert os.environ.get('OMARCHY_THEME_SKIP_BACKGROUND')=='1'
   theme=h/'.config/omarchy/themes'/args[2]
   value['colors']=tomllib.loads((theme/'colors.toml').read_text());value['rounding']=0
   value['shell']=tomllib.loads((theme/'shell.toml').read_text());value['shellApplied']=value['shell']
   if os.environ.get('OMABIT_NATIVE_THEME_STAGING')=='1':
    env=dict(os.environ,OMARCHY_PATH='/lab/native',OMARCHY_THEME_HEADLESS='1',OMARCHY_THEME_SKIP_BACKGROUND='1',PATH='/lab/native/bin:/usr/bin:/bin')
    q=subprocess.run(['/usr/bin/bash','/lab/native/bin/omarchy-theme-set',args[2]],env=env,capture_output=True,text=True,timeout=12)
    assert q.returncode==0,('native theme staging',q.returncode,q.stderr[-500:])
    assert (current/'theme.name').read_text().strip()==args[2]
   else:
    for p in (current/'theme').iterdir():
     assert p.is_file() and not p.is_symlink();p.unlink()
    for p in theme.iterdir():
     if p.is_file() and not p.name.startswith('.'):(current/'theme'/p.name).write_bytes(p.read_bytes())
   (current/'theme.name').write_text(args[2]+'\n');stage='theme'
  else:raise AssertionError((name,args))
 elif name=='hyprctl':
  if args==['reload']:
   # Execute actual generated Lua; never call a compositor or substitute a
   # hand-parsed approximation of its owner/name guards.
   code='hl={config=function(t) if t.decoration and t.decoration.rounding then print(t.decoration.rounding) end end,curve=function() end,animation=function() end}\ndofile(os.getenv("HOME").."/.config/hypr/hyprland.lua")'
   q=subprocess.run(['/usr/bin/lua','-e',code],capture_output=True,text=True,timeout=4);assert q.returncode==0,q.stderr
   rounds=[int(x) for x in q.stdout.splitlines()];assert len(rounds)<=1,'multiple active appearance includes'
   if rounds:value['rounding']=rounds[0]
   stage='windows'
  elif args==['-j','configerrors']:print('[]')
  elif args==['-j','animations']:print('[[],[]]')
  elif args and args[0]=='--batch':
   for key in args[1].split(';'):
    if key=='j/getoption decoration:rounding':print(json.dumps({'option':'decoration:rounding','int':value['rounding'],'set':True})+'\n')
    else:print('no such option\n')
  else:raise AssertionError((name,args))
 elif name=='omarchy-shell':
  assert args[:2]==['shell','applyTheme'];value['shellApplied']=tomllib.loads(base64.b64decode(args[3]).decode());stage='shell'
 else:raise AssertionError('No real desktop command fallback: '+name)
 if stage:
  save(value)
  owner=(h/'.local/state/omarchy-urbit-theme-ownership/active').read_text().splitlines()[1].split('.qualification-')[-1]
  fault=json.loads((r/'fault.json').read_text())
  fail=owner==fault.get('profile') and stage==fault.get('stage')
  events({'profile':owner,'stage':stage,'failed_after_effect':fail,'skip_background':os.environ.get('OMARCHY_THEME_SKIP_BACKGROUND')=='1','rounding':value['rounding'],'font':value['font'],'background':value['colors']['background']})
  if fail and fault.get('kill_parent'):
   parent=os.getppid()
   command=Path('/proc/'+str(parent)+'/cmdline').read_bytes().split(b'\0')
   assert b'/lab/artifacts/stable/theme/client/sync.py' in command and os.environ.get('OMABIT_LIFECYCLE_APPEARANCE_OFFLINE')=='1'
   os.kill(parent,signal.SIGKILL)  # Exact fresh helper parent, inside the guarded PID namespace.
  if fail:sys.exit(73)
