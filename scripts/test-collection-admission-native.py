#!/usr/bin/env python3
"""Native registry + guarded QML host, in a fresh no-NIC private namespace.

Synthetic application bodies expose creation and helper readiness. Real admission
code and complete native registry/service/panel loaders run. Generation-format
integration, real supported apps, widgets and the full shell remain separate.
"""
import argparse,hashlib,importlib.util,json,os,re,select,shutil,signal,socket,subprocess,sys,time,uuid
from pathlib import Path


def read(p):return json.loads(Path(p).read_text())
def save(p,v):Path(p).write_text(json.dumps(v,indent=2)+'\n')
def sha(p):return hashlib.sha256(Path(p).read_bytes()).hexdigest()


def inside(hooks=None):
 root=Path('/lab');assert Path.home()==root/'home' and os.environ.get('OMABIT_NATIVE_ARTIFACT_PROBE')=='private-v1'
 assert not Path('/home').exists() and not Path('/run/user').exists()
 assert all(os.readlink('/proc/self/ns/'+n)!=v for n,v in read(root/'host-namespaces.json').items())
 assert [n for _,n in socket.if_nameindex()]==['lo']
 sys.path.insert(0,str(root/'source'));import admission as a;import journal as j
 guarded=(root/'guarded-mode.json').exists()
 if guarded:import guarded_artifacts as ga
 prepared=read(root/'prepared.json');(root/'gates').mkdir(mode=0o700);shells=[];gates=[];events=[];owners=[]
 if hooks:hooks.begin(root,prepared)
 def start(profile=None):
  env=dict(os.environ)
  if profile:env['GATE_INITIAL_CONFIG']=json.dumps({'version':1,'plugins':[{'id':prepared[profile][role]['1']['manifest']['id']} for role in j.ROLES],'bar':{'layout':{'left':[],'center':[],'right':[]}}})
  log=(root/('shell-'+str(len(shells))+'.log')).open('wb')
  proc=subprocess.Popen(['/usr/bin/quickshell','--no-color','--path','/lab/shell.qml'],env=env,stdout=log,stderr=subprocess.STDOUT)
  shells.append((proc,log));return proc
 def call(proc,target,method,*args,check=True):
  q=subprocess.run(['/usr/bin/quickshell','ipc','--pid',str(proc.pid),'call','--',target,method,*args],capture_output=True,text=True,timeout=4)
  if check:assert q.returncode==0,(method,q.stderr)
  return q.stdout.strip()
 def state(proc):return json.loads(call(proc,'artifactProbe','state'))
 def wait(proc,fn):
  until=time.monotonic()+10
  while time.monotonic()<until:
   if hooks:hooks.check()
   assert proc.poll() is None,'bound shell exited'
   v=fn()
   if v:return v
   time.sleep(.04)
  save(root/'failure-snapshot.json',state(proc))
  raise AssertionError('admission native deadline')
 def ready_shell(proc):
  wait(proc,lambda:call(proc,'artifactProbe','state',check=False).startswith('{'))
  wait(proc,lambda:len(state(proc)['selected'])==9 and not state(proc)['scanning'])
 def ids(profile):return [prepared[profile][role]['1']['manifest']['id'] for role in j.ROLES]
 def enable(proc,profile,enabled):
  for id in ids(profile):assert call(proc,'shell','setPluginEnabled',id,str(enabled).lower())=='ok'
 def phases(proc,phase,loaded):
  rows=state(proc)['loaded']
  return rows if len(rows)==3 and all(v['host']['phase']==phase and (v['host']['loaded'] is not None)==loaded for v in rows.values()) else None
 def count_apps():return sum((root/('shell-'+str(i)+'.log')).read_text().count('APPLICATION_CREATED ') for i in range(len(shells)))
 def issue(proc,profile):
  members={}
  for role in j.ROLES:
   item=prepared[profile][role]['1'];d=item['descriptor']
   members[role]={'plugin_id':d['plugin_id'],'profile':d['profile'],'artifact_sha256':item['artifact_sha256'],
                  'endpoint_ref':None,'paths':{k:'/lab/app/'+profile+'/'+role+'/'+k for k in j.PATHS}}
  spec={'format':1,'name':profile,'members':members};epoch=uuid.uuid4().hex;path=root/'gates'/epoch;path.mkdir(mode=0o700)
  gate=a.Gate(path);gates.append(gate)
  permit={'id':uuid.uuid4().hex,'request':uuid.uuid4().hex,'epoch':epoch,'specification':spec,'shell':a.identity(proc.pid)}
  gate.issue(permit);return gate,permit
 def publish(proc,profile,gate,permit):
  for role in j.ROLES:
   item=prepared[profile][role]['1'];m=dict(item['manifest'])
   m['entryPoints']=dict(m['entryPoints']);kind=a.a.ROLES[role][1]
   if not guarded:m['entryPoints'][kind]='Admission-'+item['artifact_sha256']+'.qml'
   m['collectionActivation']={'root':str(gate.root),'permit':permit['id'],'epoch':permit['epoch'],'collection':j.digest(permit['specification'])}
   save(root/'home/.config/omarchy/plugins'/m['id']/'manifest.json',m)
  call(proc,'artifactProbe','rescan');wait(proc,lambda:not state(proc)['scanning'])
 def inspect(proc,profile,permit):
  rows=wait(proc,lambda:phases(proc,'loaded',True))
  wait(proc,lambda:all(row['helper']=='1' for row in state(proc)['loaded'].values()))
  for role in j.ROLES:
   member=permit['specification']['members'][role];host=rows[member['plugin_id']]['host'];grant=host['grant'];loaded=host['loaded']
   assert grant['permit']==permit['id'] and grant['epoch']==permit['epoch'] and grant['artifact_sha256']==member['artifact_sha256']
   assert a.identity(grant['owner']['pid'])==grant['owner']
   fields=(Path('/proc')/str(grant['owner']['pid'])/'stat').read_text().rsplit(') ',1)[1].split();assert int(fields[1])==proc.pid
   assert loaded['artifact_sha256']==member['artifact_sha256'] and loaded['plugin_id']==member['plugin_id']
   owners.append(grant['owner'])
   if guarded:
    generation=root/'home/.config/omarchy/plugins'/member['plugin_id']/'generations'/member['artifact_sha256']
    assert ga.verify(generation,member['artifact_sha256'])['admission_capable']
    assert ga.observation(generation,member['artifact_sha256'],loaded)['instance']==loaded['instance']
  if guarded:
   wait(proc,lambda:len(state(proc)['widgets'])==3 and all(w['phase']=='loaded' and w['loaded'] for w in state(proc)['widgets'].values()))
  if (root/'actual-omabit-mode.json').exists():
   id=prepared[profile]['omabit']['1']['manifest']['id']
   monitor=json.loads(call(proc,id+'.monitor','status'));assert monitor['module']==id
   shown=subprocess.run(['/usr/bin/quickshell','ipc','--pid',str(proc.pid),'show'],capture_output=True,text=True,timeout=4).stdout
   targets=re.findall(r'^target (.+)$',shown,re.M)
   assert all(target in targets for target in (id,id+'.monitor',id+'.setup'))
   assert not any(target in targets for target in ('thelifeandtimes.omabit','thelifeandtimes.omabit.monitor','thelifeandtimes.omabit.setup'))
   call(proc,id+'.setup','open')
   wait(proc,lambda:state(proc)['widgets'][id]['opened'])
   call(proc,id,'close');wait(proc,lambda:not state(proc)['widgets'][id]['opened'])
   save(root/('actual-ipc-'+profile+'.json'),{'targets':targets,'monitor':monitor,'setup_open_and_close':True})
  if hooks:hooks.admitted(profile,proc,call)
  return rows
 def closed_ui(proc,profile,only=None):
  if not (root/'actual-omabit-mode.json').exists():return
  targets=subprocess.run(['/usr/bin/quickshell','ipc','--pid',str(proc.pid),'show'],capture_output=True,text=True,timeout=4).stdout
  names=re.findall(r'^target (.+)$',targets,re.M)
  roles=[only] if only else ['omabit','theme']
  for role in roles:
   if role=='theme' and not (root/'actual-pair-mode.json').exists():continue
   id=prepared[profile][role]['1']['manifest']['id']
   assert id not in names,('closed widget still exposes IPC',id,names)
   assert call(proc,id,'open')=='Target not found.'
   if role=='omabit':
    assert id+'.setup' not in names
    call(proc,id+'.monitor','open')
    assert not json.loads(call(proc,id+'.monitor','status'))['visible'],'closed monitor reopened'
  save(root/('closed-ui-'+profile+('-'+only if only else '')+'.json'),{'targets':names,'closed':True})
  if hooks and hasattr(hooks,'closed_ui'):hooks.closed_ui(profile,proc,call,only)
 def drain(proc,profile,gate,permit):
  actual=(root/'actual-omabit-mode.json').exists()
  if actual:
   id=prepared[profile]['omabit']['1']['manifest']['id']
   assert call(proc,'artifactProbe','holdOmabit',id)=='true'
   wait(proc,lambda:(root/'hold-started').exists())
  if (root/'actual-pair-mode.json').exists():
   dms=prepared[profile]['dms']['1']['manifest']['id']
   assert call(proc,'artifactProbe','overlayOpen',dms)=='true'
  gate.stop_admission(permit['id'],uuid.uuid4().hex)
  wait(proc,lambda:phases(proc,'draining',True))
  if actual:
   assert not state(proc)['loaded'][id]['host']['transport']['closed']
   closed_ui(proc,profile)
   save(root/'busy-closing-ui.json',{'passed':True,'omabit_transport':state(proc)['loaded'][id]['host']['transport']})
   (root/'release-hold').touch()
  if (root/'actual-pair-mode.json').exists():assert call(proc,'artifactProbe','overlayOpen',dms)=='false'
  wait(proc,lambda:all(row['host']['transport']['closed'] for row in state(proc)['loaded'].values()))
  closed_ui(proc,profile)
  enable(proc,profile,False);wait(proc,lambda:not state(proc)['loaded'])
  if hooks and hasattr(hooks,'unloaded'):hooks.unloaded(profile,proc,call)
 try:
  first=start();ready_shell(first);enable(first,'stable',True)
  wait(first,lambda:phases(first,'waiting',False));assert count_apps()==0
  if guarded:
   wait(first,lambda:len(state(first)['widgets'])==3)
   assert all(not v['loaded'] for v in state(first)['widgets'].values())
   assert 'WIDGET_CREATED ' not in (root/'shell-0.log').read_text()
  if hooks:hooks.no_admission()
  events.append({'case':'enabled-without-permit','application_creations':0,'hosts':state(first)['loaded']})
  gate,p=issue(first,'stable');publish(first,'stable',gate,p);rows=inspect(first,'stable',p);assert count_apps()==3
  events.append({'case':'issued-before-construction','permit':p,'loaded':rows,'claims':gate.status()['record']['claims']})
  # Kill only the exact recorded Theme admission child of our private shell.
  theme=rows[ids('stable')[1]]['host']['grant']['owner'];fd=os.pidfd_open(theme['pid'])
  try:assert a.identity(theme['pid'])==theme;signal.pidfd_send_signal(fd,signal.SIGKILL)
  finally:os.close(fd)
  theme_id=ids('stable')[1]
  wait(first,lambda:state(first)['loaded'][theme_id]['host']['phase']=='fenced')
  wait(first,lambda:state(first)['loaded'][theme_id]['host']['transport']['closed'])
  closed_ui(first,'stable','theme')
  assert call(first,'shell','setPluginEnabled',theme_id,'false')=='ok';wait(first,lambda:theme_id not in state(first)['loaded'])
  assert call(first,'shell','setPluginEnabled',theme_id,'true')=='ok'
  wait(first,lambda:theme_id in state(first)['loaded'] and state(first)['loaded'][theme_id]['host']['phase']=='fenced')
  assert state(first)['loaded'][theme_id]['host']['loaded'] is None and count_apps()==3
  if guarded:
   assert state(first)['widgets'][theme_id]['loaded'] is False
   assert (root/'shell-0.log').read_text().count('WIDGET_CREATED '+theme_id+'\n')==1
  assert gate.status()['record']['claims']['theme']['owner']==theme
  events.append({'case':'guard-loss-and-native-recreation-refused','original_owner':theme,'hosts':state(first)['loaded'],'record':gate.status()['record'],'application_creations':count_apps()})
  gate.stop_admission(p['id'],uuid.uuid4().hex)
  wait(first,lambda:all(row['host']['phase'] in ('draining','fenced') for row in state(first)['loaded'].values()))
  wait(first,lambda:all(not row['host']['loaded'] or row['host']['transport']['closed'] for row in state(first)['loaded'].values()))
  closed_ui(first,'stable')
  enable(first,'stable',False);wait(first,lambda:not state(first)['loaded'])
  if hooks and hasattr(hooks,'unloaded'):hooks.unloaded('stable',first,call)
  # This permit is still unused when the shell exits. No previous claim exists
  # to explain the next refusal: the exact shell binding must prevent startup.
  stale_gate,stale=issue(first,'dev');publish(first,'dev',stale_gate,stale)
  assert not stale_gate.status()['record']['claims']
  call(first,'artifactProbe','stop');assert first.wait(timeout=4)==0
  second=start('dev');ready_shell(second);wait(second,lambda:phases(second,'fenced',False))
  assert count_apps()==3 and not stale_gate.status()['record']['claims'] and not stale_gate.status()['shell_live']
  if guarded:assert 'WIDGET_CREATED ' not in (root/'shell-1.log').read_text()
  if hooks:hooks.no_admission()
  events.append({'case':'new-shell-stale-enabled-config-refused','old_shell':stale['shell'],'new_shell':a.identity(second.pid),'record':stale_gate.status()['record'],'application_creations':count_apps(),'hosts':state(second)['loaded']})
  enable(second,'dev',False);wait(second,lambda:not state(second)['loaded'])
  fresh_gate,fresh=issue(second,'dev');publish(second,'dev',fresh_gate,fresh);enable(second,'dev',True);rows=inspect(second,'dev',fresh);assert count_apps()==6
  events.append({'case':'explicit-new-shell-activation','permit':fresh,'loaded':rows})
  drain(second,'dev',fresh_gate,fresh)
  if hooks:
   return_gate,returned=issue(second,'stable');publish(second,'stable',return_gate,returned);enable(second,'stable',True)
   rows=inspect(second,'stable',returned);assert count_apps()==9
   events.append({'case':'connected-stable-return','permit':returned,'loaded':rows})
   (root/'hold-started').unlink();(root/'release-hold').unlink()
   drain(second,'stable',return_gate,returned)
  call(second,'artifactProbe','stop');assert second.wait(timeout=4)==0
  assert all(a.identity(owner['pid'])!=owner for owner in owners)
  if guarded:
   for i,profile in enumerate(('stable','dev')):
    for id in ids(profile):assert (root/('shell-'+str(i)+'.log')).read_text().count('WIDGET_CREATED '+id+'\n')>=1
   for i in range(len(shells)):
    text=(root/('shell-'+str(i)+'.log')).read_text()
    assert 'WIDGET_BAD_ID' not in text and 'APPLICATION_BAD_ID' not in text
  for i in range(len(shells)):
   text=(root/('shell-'+str(i)+'.log')).read_text()
   assert not any(k in text for k in ('TypeError','ReferenceError','Binding loop','Unable to assign','Cannot assign','failed to load','service plugin load failed')),text[-3000:]
  save(root/'result.json',{'passed':True,'events':events,'application_creations':count_apps(),'guard_owners':owners,'all_recorded_guards_exited':True,'complete_native_registry':True,'actual_guard_and_host':True,'real_plugin_bodies':(root/'actual-pair-mode.json').exists(),'actual_omabit_service_panel':(root/'actual-omabit-mode.json').exists(),'full_native_shell':False,'artifact_host_format_integrated':guarded,'widget_fixture_loaders':guarded,'application_widget_creations':sum((root/('shell-'+str(i)+'.log')).read_text().count('WIDGET_CREATED ') for i in range(len(shells))) if guarded else None,'coordinator_recovery_integrated':False})
  print('PASS: native gated construction, guard-loss fence, stale enabled startup refusal and explicit new-shell activation')
 finally:
  for gate in gates:gate.close()
  for proc,log in shells:
   if proc.poll() is None:
    proc.terminate()
    try:proc.wait(timeout=4)
    except subprocess.TimeoutExpired:proc.kill();proc.wait(timeout=3)
   log.close()


def main(*,artifact_module=None):
 if '--inside' in sys.argv:return inside()
 guarded='--guarded' in sys.argv
 if guarded:sys.argv.remove('--guarded')
 path=Path(__file__).with_name('test-collection-artifacts-native.py')
 spec=importlib.util.spec_from_file_location('native_artifact_fixture',path);base=importlib.util.module_from_spec(spec);spec.loader.exec_module(base)
 original_source=Path(__file__).resolve().parents[1];base.__file__=__file__
 old='loaded[id]={identity:o.collectionObservation(),code:o.fixtureVersion,helper:o.fixtureHelper,selected:o.manifest.version}'
 assert base.QML.count(old)==2
 base.QML=base.QML.replace(old,'loaded[id]={host:o.diagnostic(),code:o.payload?o.payload.fixtureVersion:null,helper:o.payload?o.payload.fixtureHelper:null}')
 old='property var config: ({version:1,plugins:[],bar:{layout:{left:[],center:[],right:[]}}})'
 assert old in base.QML
 base.QML=base.QML.replace(old,'property var config: JSON.parse(Quickshell.env("GATE_INITIAL_CONFIG") || \'{"version":1,"plugins":[],"bar":{"layout":{"left":[],"center":[],"right":[]}}}\')')
 base.BODY=base.BODY.replace(' id: root',' id: root\n Component.onCompleted:console.log("APPLICATION_CREATED "+Qt.resolvedUrl("helper.py"))')
 base.BODY=base.BODY.replace(' property var shell: null',' property var shell: null\n property var settings: ({})\n property bool uiAdmissionOpen: true')
 if guarded:
  sys.path.insert(0,str(original_source/'native/collections'));import artifacts as artifact;import guarded_artifacts as ga
  if artifact_module is not None:ga=artifact_module
  admission_root=original_source/'native/collections';admission_pin=artifact.inventory(artifact.snapshot(admission_root,list(ga.SUPPORT)))
  def guarded_prepare(source,pin,store,**kwargs):
   # Synthetic widget inputs are fixture-owned and finalized before repinning.
   source=Path(source);assert artifact.inventory(artifact.snapshot(source,list(pin)))==pin
   m=read(source/'manifest.json');m['kinds'].append('bar-widget');m['entryPoints']['barWidget']='Widget.qml';save(source/'manifest.json',m)
   (source/'Widget.qml').write_text('import QtQuick\nItem {\n property QtObject bar: null\n property string moduleName: ""\n property var settings: ({})\n property var service: null\n property bool manageIpc: true\n property bool opened: false\n function open(){opened=true}\n function close(){opened=false}\n function toggle(){opened=!opened}\n Component.onCompleted:{console.log("WIDGET_CREATED "+moduleName);if(moduleName.indexOf(".qualification-")<0)console.log("WIDGET_BAD_ID")}\n}\n')
   updated=artifact.inventory(artifact.snapshot(source,list(pin)+['Widget.qml']))
   return ga.prepare(source,updated,store,**kwargs,admission_source=admission_root,expected_admission=admission_pin)
  artifact.prepare=guarded_prepare
  base.QML=base.QML.replace('  function state():string{return shell.describe()}', '  function state():string{return shell.describe()}\n  function holdOmabit(id:string):string{return String(shell._services[id].payload.submit(["fixture-hold"],"fixture-hold"))}\n  function overlayOpen(id:string):string{var o=shell.panels[id].item;o.open("{}");return String(o.opened)}')
  base.QML=base.QML.replace(' property var panels: ({})',' property var panels: ({})\n property var widgets: ({})')
  base.QML=base.QML.replace('return JSON.stringify({selected:selected,loaded:loaded,scanning:pluginRegistry.scanning})','var w=({});for(var id in widgets){var item=widgets[id];w[id]={phase:item.phase,loaded:!!item.payload,admitted:item.admitted,opened:item.opened}};return JSON.stringify({selected:selected,loaded:loaded,widgets:w,scanning:pluginRegistry.scanning})')
  base.QML=base.QML.replace(' METHODS',''' property var widgetEntries: {
   var installed=pluginRegistry.installedPlugins;var configuration=config;var entries=[]
   for(var id in installed)if(pluginRegistry.isEnabled(id))entries.push({id:id,source:pluginRegistry.entryPointUrl(installed[id],"barWidget")})
   return entries
 }
 Instantiator {
  model:shell.widgetEntries
  delegate:QtObject {
   id:entry
   required property var modelData
   property Loader widgetLoader:Loader {
    source:entry.modelData.source;asynchronous:true
    onLoaded:{item.moduleName=entry.modelData.id;var next=Object.assign({},shell.widgets);next[entry.modelData.id]=item;shell.widgets=next}
    Component.onDestruction:{var next=Object.assign({},shell.widgets);delete next[entry.modelData.id];shell.widgets=next}
   }
  }
 }
 METHODS''')
  base.BODY=base.BODY.replace(' Component.onCompleted:console.log("APPLICATION_CREATED "+Qt.resolvedUrl("helper.py"))',' Component.onCompleted:{console.log("APPLICATION_CREATED "+Qt.resolvedUrl("helper.py"));if(!manifest || manifest.id.indexOf(".qualification-")<0)console.log("APPLICATION_BAD_ID")}')
 original_run=base.subprocess.run
 def launch(command,*args,**kwargs):
  assert command[0]=='bwrap'
  out=Path(command[command.index('--bind')+1]);prepared=read(out/'prepared.json');hashes=read(out/'source.json')
  for name in ('admission.py','admission_guard.py'):
   src=original_source/'native/collections'/name;shutil.copy2(src,out/'source'/name);hashes[str(src)]=sha(src)
  src=original_source/'native/collections/Host.qml';shutil.copy2(src,out/'CollectionHost.qml');hashes[str(src)]=sha(src)
  hashes[str(path)]=sha(path);hashes[str(Path(__file__))]=sha(Path(__file__));shutil.copy2(path,out/'native-artifact-base.py')
  if guarded:
   for name in ('guarded_artifacts.py','WidgetHost.qml','Registry.qml'):
    src=original_source/'native/collections'/name;shutil.copy2(src,out/'source'/name);hashes[str(src)]=sha(src)
   save(out/'source.json',hashes);save(out/'guarded-mode.json',{'format':2,'bundle_pin':admission_pin})
   return original_run(command,*args,**kwargs)
  wrappers={}
  for roles in prepared.values():
   for role,versions in roles.items():
    for item in versions.values():
     manifest=dict(item['manifest']);plugin=out/'home/.config/omarchy/plugins'/manifest['id'];name='Admission-'+item['artifact_sha256']+'.qml'
     fields={'pluginId':manifest['id'],'profile':item['descriptor']['profile'],'artifact':item['artifact_sha256'],'role':role,
             'runtimeSource':'file:///lab/home/.config/omarchy/plugins/'+manifest['id']+'/generations/'+item['artifact_sha256']+'/CollectionRuntime.qml',
             'guardSource':'file:///lab/source/admission_guard.py'}
     body='import QtQuick\nimport "file:///lab" as Shared\nShared.CollectionHost {\n'+''.join('  '+k+': '+json.dumps(v)+'\n' for k,v in fields.items())+'}\n'
     (plugin/name).write_text(body);wrappers[str((plugin/name).relative_to(out))]=sha(plugin/name)
    item=versions['1'];manifest=dict(item['manifest']);manifest['entryPoints']=dict(manifest['entryPoints']);manifest['entryPoints'][{'omabit':'service','theme':'service','dms':'overlay'}[role]]='Admission-'+item['artifact_sha256']+'.qml'
    save(plugin/'manifest.json',manifest)
  save(out/'source.json',hashes);save(out/'host-wrappers.json',wrappers)
  return original_run(command,*args,**kwargs)
 base.subprocess.run=launch
 # Reuse only the already guarded artifact fixture's file preparation/namespace
 # runner. The copied inner driver above owns the new admission scenario.
 base.main()


if __name__=='__main__':main()
