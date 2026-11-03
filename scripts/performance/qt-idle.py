#!/usr/bin/env python3
"""Measure real Service/Panel/Monitor QML in an offline fixture, with inert helpers.

Requires a passing --panel service-drain fixture for its unchanged native UI and
Commons adapters. This measures offscreen Quickshell CPU, not desktop GPU/power.
"""
import argparse
import hashlib
import json
import os
from pathlib import Path
import re
import select
import shutil
import signal
import socket
import subprocess
import time

QML = '''import QtQuick
import Quickshell
import Quickshell.Io
ShellRoot {
 id: root
 property string variant: Quickshell.env("OMABIT_QT_PERF_VARIANT")
 property string label: Quickshell.env("OMABIT_QT_PERF_LABEL")
 property int clockChanges: 0
 property int observations: 0
 property int memoryChanges: 0
 property int cpuChanges: 0
 property int churnSteps: 0
 Loader {id: app; active: false}
 Loader {id: panel; active: false}
 Connections {
  target: app.item
  function onNowChanged() {root.clockChanges++}
  function onConnectionsChanged() {root.observations++}
  function onMemoryHistoryChanged() {root.memoryChanges++}
  function onCpuHistoryChanged() {root.cpuChanges++}
 }
 function snapshot() {
  var a=app.item
  return JSON.stringify({loaded:!!a,observing:a?a.observing:false,registered:a?a.registrationsLoaded:false,
    error:a?a.feedError:"",connections:a?a.connections.length:0,clockChanges:clockChanges,
    observations:observations,memoryChanges:memoryChanges,cpuChanges:cpuChanges,
    panelOpen:panel.item?panel.item.opened:false,monitorOpen:a?a.monitorWindow.visible:false,
    owners:a && a.observationSchedule ? a.observationSchedule.panels.length : null,
    interval:a && a.observationSchedule ? a.observationSchedule.interval : "2s",
    churnSteps:churnSteps,closed:a?a.transportState.closed:false})
 }
 Timer {
  id: churnTimer;interval:20;repeat:true
  onTriggered: {
   var step=root.churnSteps%4
   if(step===0)panel.item.open()
   else if(step===3)panel.item.close()
   else if(step===2)app.item.monitorWindow.present()
   else app.item.monitorWindow.visible=false
   root.churnSteps--
   if(root.churnSteps===0) {panel.item.close();app.item.monitorWindow.visible=false;stop()}
  }
 }
 IpcHandler {
  target:"omabitQtPerformance"
  function load():string {
   app.setSource("file:///lab/artifacts/"+root.variant+"/Service.qml",{profileName:root.label,
    profileEnvironment:{XDG_STATE_HOME:"/lab/state/"+root.label,XDG_CONFIG_HOME:"/lab/config/"+root.label,
      XDG_DATA_HOME:"/lab/data/"+root.label,XDG_CACHE_HOME:"/lab/cache/"+root.label},
    manifest:{id:"thelifeandtimes.omabit.qualification-"+root.label},settings:{notifications:false},
    localSetupAllowed:false,toolsDirectory:"file:///lab/artifacts/"+root.variant+"/bin"})
   app.active=true
   panel.setSource("file:///lab/artifacts/"+root.variant+"/Panel.qml",{service:app.item,
    moduleName:"thelifeandtimes.omabit.qualification-"+root.label,settings:app.item.settings})
   panel.active=true
   return root.snapshot()
  }
  function state():string {return root.snapshot()}
  function reset():string {root.clockChanges=0;root.observations=0;root.memoryChanges=0;root.cpuChanges=0;return root.snapshot()}
  function view(mode:string):string {
   if(mode==="panel")panel.item.open()
   else if(mode==="monitor")app.item.monitorWindow.present()
   else if(mode==="hidden") {panel.item.close();app.item.monitorWindow.visible=false}
   else throw new Error("unknown fixture view")
   return root.snapshot()
  }
  function churn(count:string):string {root.churnSteps=Number(count)*4;churnTimer.start();return root.snapshot()}
  function drain():string {return app.item.stopTransport()}
  function stop():string {Qt.callLater(Qt.quit);return "stopping"}
 }
}
'''

HELPER = '''#!/usr/bin/python3 -B
import datetime,json,os,select,signal,sys,time
from pathlib import Path
args=sys.argv[1:]
label=os.environ['OMABIT_QT_PERF_LABEL']
assert args[:2]==['--format','json'],args
args=args[2:]
def record(event,**fields):
 with Path('/lab/'+label+'-calls.jsonl').open('a') as f:f.write(json.dumps({'event':event,'pid':os.getpid(),'args':args,**fields})+'\\n')
record('start')
if args==['runners']:print(json.dumps([{'name':'fixture','runner_id':'fixture-runner','mode':'development-managed','transport':'unix'}]))
elif args==['remote-setup-status']:print('{}')
elif args in (['app-plugins'],['ships'],['app-connections']):print('[]')
elif args[:3]==['inventory','--watch','--interval'] and args[3] in ('2s','30s') and args[4:] in ([],['--control-stdin']):
 interval=float(args[3][:-1]);controlled=bool(args[4:]);due=time.monotonic();buffer=b''
 Path('/lab/'+label+'-feed').write_text(str(os.getpid()))
 def stop(*_):record('exit');raise SystemExit(0)
 signal.signal(signal.SIGTERM,stop)
 while True:
  if time.monotonic()>=due:
   stamp=datetime.datetime.now(datetime.timezone.utc).isoformat()
   ships=[{'id':'fixture-'+str(n),'ship':'~zod','mode':'fake','config_revision':1,'applied_revision':1,
    'config':{'http_port':18080+n,'ames_port':34343,'loom':31},'observed_state':'stopped','desired_state':'stopped','observed_at':stamp}
    for n in range(int(os.environ['OMABIT_QT_PERF_SHIPS']))]
   print(json.dumps({'name':'fixture','runner_id':'fixture-runner','mode':'development-managed','status':'available',
     'access_role':'observer','features':[],'checked_at':stamp,'snapshot':{'runner_id':'fixture-runner','mode':'development-managed',
     'instances':ships,'operations':[],'incidents':[]}}),flush=True)
   record('observation');last=time.monotonic();due=last+interval
  ready,_,_=select.select([sys.stdin] if controlled else [],[],[],max(0,due-time.monotonic()))
  if ready:
   data=os.read(sys.stdin.fileno(),4096)
   if not data:break
   buffer+=data;wake=False
   while b'\\n' in buffer:
    line,buffer=buffer.split(b'\\n',1)
    control=json.loads(line);assert set(control)=={'interval','refresh'} and control['interval'] in ('2s','30s')
    interval=float(control['interval'][:-1]);wake=wake or control['refresh'];record('control',control=control)
   due=time.monotonic() if wake else last+interval
else:raise AssertionError('Unexpected fixture command: '+repr(args))
record('exit')
'''


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def inside():
    root=Path('/lab')
    assert os.environ.get('OMABIT_QT_PERF')=='private-v1' and os.environ['HOME']=='/lab/home'
    assert not Path('/home').exists() and not Path('/run/user').exists()
    expected=json.loads((root/'host-namespaces.json').read_text())
    assert all(os.readlink('/proc/self/ns/'+n)!=expected[n] for n in expected)
    assert [n for _,n in socket.if_nameindex()]==['lo']
    assert not subprocess.check_output(['/usr/bin/ip','route','show'],text=True)
    options=json.loads((root/'options.json').read_text())
    report={'passed':False,'actual_qml':['Service','Panel','MonitorWindow'],'offscreen':True,
            'inert_collector':True,'actual_compositor':False,'cases':[]}
    def save(): (root/'result.json').write_text(json.dumps(report,indent=2)+'\n')
    save()
    def process(pid):
        fields=Path(f'/proc/{pid}/stat').read_text().split(') ',1)[1].split()
        status=Path(f'/proc/{pid}/status').read_text()
        return {'pid':pid,'start':fields[19],'cpu_seconds':(int(fields[11])+int(fields[12]))/os.sysconf('SC_CLK_TCK'),
                'main_thread_cpu_seconds':int(Path(f'/proc/{pid}/schedstat').read_text().split()[0])/1e9,
                'rss_kib':int(re.search(r'^VmRSS:\s+(\d+)',status,re.M)[1]),
                'fds':len(list(Path(f'/proc/{pid}/fd').iterdir())),
                'children':[int(x) for x in Path(f'/proc/{pid}/task/{pid}/children').read_text().split()]}
    for count in options['ships']:
      for variant in ('baseline','candidate'):
        label=variant+'-'+str(count)
        env={**os.environ,'OMABIT_QT_PERF_VARIANT':variant,'OMABIT_QT_PERF_LABEL':label,'OMABIT_QT_PERF_SHIPS':str(count)}
        with (root/(label+'-shell.log')).open('wb') as log:
          shell=subprocess.Popen(['/usr/bin/quickshell','--no-color','--path','/lab/shell.qml'],env=env,stdout=log,stderr=subprocess.STDOUT,start_new_session=True)
          def call(method,*args,check=True):
            reply=subprocess.run(['/usr/bin/quickshell','ipc','--pid',str(shell.pid),'call','--','omabitQtPerformance',method,*args],capture_output=True,text=True,timeout=3)
            if check: assert reply.returncode==0,(method,reply.stderr)
            return reply.stdout.strip()
          def state(): return json.loads(call('state'))
          def wait(fn,seconds=10):
            end=time.monotonic()+seconds
            while time.monotonic()<end:
              assert shell.poll() is None,(root/(label+'-shell.log')).read_text()[-4000:]
              value=fn()
              if value:return value
              time.sleep(.05)
            raise AssertionError('fixture deadline: '+label+' '+str(state()))
          def measure(name,seconds):
            call('reset');before=process(shell.pid);started=time.monotonic()
            time.sleep(seconds)
            after=process(shell.pid);elapsed=time.monotonic()-started;observed=state()
            assert observed['observing'] and not observed['error'],observed
            assert before['start']==after['start'] and before['children']==after['children'],(before,after)
            return {'view':name,'seconds':elapsed,'shell_cpu_seconds':after['cpu_seconds']-before['cpu_seconds'],
                    'shell_cpu_resolution_seconds':1/os.sysconf('SC_CLK_TCK'),
                    'main_thread_cpu_seconds':after['main_thread_cpu_seconds']-before['main_thread_cpu_seconds'],
                    'before':before,'after':after,'qml':observed}
          try:
            wait(lambda:call('state',check=False).startswith('{'))
            call('load')
            wait(lambda:state()['registered'] and state()['observing'] and state()['connections']==1)
            time.sleep(2)
            feed=int((root/(label+'-feed')).read_text())
            assert process(shell.pid)['children']==[feed],process(shell.pid)
            record={'variant':variant,'ships':count,'feed_pid':feed,'measurements':[]}
            hidden=measure('hidden',options['seconds']);record['measurements'].append(hidden)
            if options['seconds']>=30:
              if variant=='candidate':
                assert hidden['qml']['interval']=='30s' and hidden['qml']['clockChanges']<=3 and hidden['qml']['memoryChanges']==0,hidden
              else: assert hidden['qml']['clockChanges']>=options['seconds']-2,hidden
            before=state()['observations'];opened=time.monotonic();call('view','panel')
            wait(lambda:state()['observations']>before,3)
            record['panel_wake_seconds']=time.monotonic()-opened
            time.sleep(.2)
            visible=measure('panel',options['visible_seconds']);record['measurements'].append(visible)
            assert visible['qml']['interval']=='2s' and visible['qml']['observations']>=max(1,int(options['visible_seconds']/2)-1),visible
            call('view','hidden')
            if variant=='candidate' and count==options['ships'][-1]:
              call('view','monitor');time.sleep(.2);call('view','hidden');time.sleep(.3)
              prior=process(shell.pid)
              call('churn',str(options['churn']))
              wait(lambda:state()['churnSteps']==0,options['churn']*.2+10)
              time.sleep(.3)
              after=process(shell.pid);status=state()
              assert prior['children']==after['children']==[feed] and prior['fds']==after['fds'],(prior,after)
              assert status['owners']==0 and status['interval']=='30s' and not status['panelOpen'] and not status['monitorOpen'],status
              record['churn']={'cycles':options['churn'],'before':prior,'after':after,'qml':status}
            assert int((root/(label+'-feed')).read_text())==feed
            assert call('drain')=='requested';wait(lambda:state()['closed'])
            assert process(shell.pid)['children']==[]
            assert call('stop')=='stopping' and shell.wait(timeout=4)==0
            text=(root/(label+'-shell.log')).read_text()
            assert not any(x in text for x in ('ReferenceError','TypeError','Binding loop','ERROR:','Unable to assign','Cannot assign')),text[-4000:]
            report['cases'].append(record);save()
          except Exception as e:
            report['error']=type(e).__name__+': '+str(e);save();raise
          finally:
            if shell.poll() is None:
              os.killpg(shell.pid,signal.SIGTERM)
              try:shell.wait(timeout=3)
              except subprocess.TimeoutExpired:os.killpg(shell.pid,signal.SIGKILL);shell.wait(timeout=3)
    report['passed']=True;save();print(json.dumps(report))


def main():
    p=argparse.ArgumentParser(description=__doc__)
    p.add_argument('--inside',action='store_true')
    p.add_argument('--baseline',type=Path);p.add_argument('--candidate',type=Path)
    p.add_argument('--ui-fixture',type=Path);p.add_argument('--output',type=Path)
    p.add_argument('--seconds',type=float,default=32);p.add_argument('--visible-seconds',type=float,default=8)
    p.add_argument('--ships',type=int,nargs='+',default=[0,32]);p.add_argument('--churn',type=int,default=50)
    a=p.parse_args()
    if a.inside:return inside()
    assert 2<=a.seconds<=45 and 2<=a.visible_seconds<=15 and 1<=a.churn<=100
    assert a.ships and all(0<=n<=128 for n in a.ships)
    assert a.baseline and a.candidate and a.ui_fixture and a.output
    out=a.output.absolute();assert out.parent==Path('/tmp') and re.fullmatch(r'omabit-qt-performance-[A-Za-z0-9-]+',out.name) and not out.exists()
    native=json.loads((a.ui_fixture/'result.json').read_text());assert native['passed'] and native['native_panel_mode']
    out.mkdir(mode=0o700)
    for name in ('etc','home','runtime','config','state','cache','data','artifacts'):(out/name).mkdir(mode=0o700)
    (out/'etc/fonts').mkdir()
    (out/'etc/passwd').write_text(f'lab:x:{os.getuid()}:{os.getgid()}:Lab:/lab/home:/bin/sh\n')
    (out/'etc/group').write_text(f'lab:x:{os.getgid()}:\n');(out/'etc/nsswitch.conf').write_text('passwd: files\ngroup: files\n')
    sources={}
    for variant,source in [('baseline',a.baseline),('candidate',a.candidate)]:
      dest=out/'artifacts'/variant;dest.mkdir();(dest/'bin').mkdir()
      for src in sorted((source/'plugins/omabit').iterdir()):
        if src.suffix in ('.qml','.js','.svg') or src.name=='qmldir':
          shutil.copy2(src,dest/src.name);sources[str(src)]=sha(src)
      shutil.copy2(a.ui_fixture/'artifacts/stable/WindowActivation.qml',dest/'WindowActivation.qml')
      for name in ('urbit-collector','omabit'):
        tool=dest/'bin'/name;tool.write_text(HELPER);tool.chmod(0o700)
    for name in ('Commons','Ui'):shutil.copytree(a.ui_fixture/name,out/name)
    (out/'source.json').write_text(json.dumps({'sha256':sources,'native_fixture_source_sha256':sha(a.ui_fixture/'source.json'),
      'fixture_sha256':sha(Path(__file__)),'limitations':['inert collector','offscreen software rendering','inert KeyboardPanel/focus/Commons palette','no actual desktop compositor or runner']},indent=2)+'\n')
    (out/'host-namespaces.json').write_text(json.dumps({n:os.readlink('/proc/self/ns/'+n) for n in ('user','net','pid','mnt')}))
    (out/'options.json').write_text(json.dumps({'seconds':a.seconds,'visible_seconds':a.visible_seconds,'ships':a.ships,'churn':a.churn}))
    (out/'shell.qml').write_text(QML);shutil.copy2(__file__,out/'driver.py')
    command=['bwrap','--unshare-all','--die-with-parent','--clearenv','--ro-bind','/usr','/usr',
      '--symlink','usr/bin','/bin','--symlink','usr/lib','/lib','--symlink','usr/lib','/lib64','--proc','/proc','--dev','/dev',
      '--tmpfs','/tmp','--ro-bind',str(out/'etc'),'/etc','--ro-bind','/etc/fonts','/etc/fonts',
      '--bind',str(out),'/lab','--ro-bind',str(out/'artifacts'),'/lab/artifacts','--chdir','/lab']
    env={'HOME':'/lab/home','XDG_RUNTIME_DIR':'/lab/runtime','XDG_CONFIG_HOME':'/lab/config','XDG_DATA_HOME':'/lab/data',
      'XDG_STATE_HOME':'/lab/state','XDG_CACHE_HOME':'/lab/cache','PATH':'/usr/bin:/bin','LANG':'C.UTF-8',
      'QT_QPA_PLATFORM':'offscreen','QT_QUICK_BACKEND':'software','QML_DISABLE_DISK_CACHE':'1','QT_QUICK_CONTROLS_STYLE':'Basic',
      'QT_QPA_PLATFORMTHEME':'','PYTHONDONTWRITEBYTECODE':'1','OMABIT_QT_PERF':'private-v1'}
    for key,value in env.items():command+=['--setenv',key,value]
    command+=['--','/usr/bin/python3','-B','/lab/driver.py','--inside']
    (out/'command.json').write_text(json.dumps(command,indent=2)+'\n')
    result=subprocess.run(command,capture_output=True,text=True,timeout=len(a.ships)*2*(a.seconds+a.visible_seconds+15)+a.churn*.2+30)
    (out/'driver.log').write_text(result.stdout+result.stderr)
    if result.returncode:print(result.stderr[-6000:])
    else:print('PASS: private actual-QML idle/visible CPU measurement and view churn: '+str(out/'result.json'))
    raise SystemExit(result.returncode)

if __name__=='__main__':main()
