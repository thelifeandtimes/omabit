"""Disposable software Wayland compositor; never attaches to a host desktop."""
from pathlib import Path
import hashlib,json,os,socket,subprocess,time

def prepare(out,bundle,command,hashes):
 bundle=Path(bundle).resolve(strict=True)
 record=json.loads((bundle/'prepared.json').read_text())
 assert record['host_installation'] is False
 for name,digest in record['tools'].items():
  assert name.startswith('usr/') and '..' not in Path(name).parts
  assert hashlib.sha256((bundle/name).read_bytes()).hexdigest()==digest
 (out/'wayland-runtime.json').write_text(json.dumps(record,indent=2)+'\n')
 (out/'sway.conf').write_text('xwayland disable\nswaybg_command -\noutput HEADLESS-1 mode 1280x720\nfont monospace 12\nfocus_follows_mouse no\n')
 (out/'fonts.conf').write_text('<fontconfig><dir>/tools/usr/share/fonts</dir><dir>/usr/share/fonts</dir><cachedir>/lab/cache/fonts</cachedir></fontconfig>')
 command[1:1]=['--ro-bind',str(bundle),'/tools']
 at=command.index('--');command[at:at]=['--setenv','FONTCONFIG_FILE','/lab/fonts.conf']
 at=command.index('QT_QPA_PLATFORM');assert command[at+1]=='offscreen';command[at+1]='wayland'

class Runtime:
 def __init__(self):self.proc=None;self.log=None
 def start(self):
  assert Path.home()==Path('/lab/home') and not any(Path(p).exists() for p in ('/home','/run/user','/dev/dri'))
  assert [n for _,n in socket.if_nameindex()]==['lo']
  assert all(os.readlink('/proc/self/ns/'+n)!=v for n,v in json.loads(Path('/lab/host-namespaces.json').read_text()).items())
  assert os.environ.get('FONTCONFIG_FILE')=='/lab/fonts.conf'
  assert os.environ['QT_QPA_PLATFORM']=='wayland' and not os.environ.get('WAYLAND_DISPLAY')
  self.env=dict(os.environ,WLR_BACKENDS='headless',WLR_RENDERER='pixman',WLR_HEADLESS_OUTPUTS='1',XKB_CONFIG_ROOT='/tools/usr/share/X11/xkb')
  self.loader=['/tools/usr/lib/ld-linux-x86-64.so.2','--library-path','/tools/usr/lib']
  self.log=Path('/lab/sway.log').open('wb')
  self.proc=subprocess.Popen(self.loader+['/tools/usr/bin/sway','--config','/lab/sway.conf'],env=self.env,stdout=self.log,stderr=subprocess.STDOUT)
  end=time.monotonic()+15
  while time.monotonic()<end:
   assert self.proc.poll() is None,'private compositor exited'
   displays=[p for p in Path('/lab/runtime').glob('wayland-*') if p.is_socket()]
   ipcs=list(Path('/lab/runtime').glob('sway-ipc.*.sock'))
   if len(displays)==len(ipcs)==1:
    os.environ['WAYLAND_DISPLAY']=displays[0].name
    self.env.update(WAYLAND_DISPLAY=displays[0].name,SWAYSOCK=str(ipcs[0]))
    output=json.loads(self.tool('swaymsg','-t','get_outputs','-r').stdout)
    assert len(output)==1 and output[0]['name']=='HEADLESS-1'
    Path('/lab/wayland-outputs.json').write_text(json.dumps(output,indent=2)+'\n');return
   time.sleep(.05)
  raise AssertionError('private compositor readiness deadline')
 def tool(self,name,*args):
  assert name in ('swaymsg','grim')
  return subprocess.run(self.loader+['/tools/usr/bin/'+name,*args],env=self.env,capture_output=True,text=True,timeout=8,check=True)
 def snapshot(self,name):
  assert name.replace('-','').isalnum()
  path=Path('/lab')/(name+'.png');time.sleep(.25);self.tool('grim',str(path))
  return {'file':path.name,'sha256':hashlib.sha256(path.read_bytes()).hexdigest()}
 def close(self):
  if self.proc:
   if self.proc.poll() is None:
    self.proc.terminate()
    try:self.proc.wait(timeout=8)
    except subprocess.TimeoutExpired:self.proc.kill();self.proc.wait(timeout=4)
   Path('/lab/compositor-exit.json').write_text(json.dumps({'pid':self.proc.pid,'exit':self.proc.returncode,'renderer':'pixman','backend':'headless'})+'\n')
  if self.log:self.log.close()
