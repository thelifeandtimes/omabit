#!/usr/bin/env python3
"""Actual Omabit bodies in guarded native loaders; other roles remain synthetic.

All runner/SSH tools and window focus are inert. Native UI controls are copied
with private color/window values; no installed plugin or service runs. Optional
--actual-pair requires explicit --theme-source and --dms-source checkout paths.
"""
import argparse,importlib.util,json,os,shutil,sys
from pathlib import Path

def module(name,path):
 spec=importlib.util.spec_from_file_location(name,path);m=importlib.util.module_from_spec(spec);spec.loader.exec_module(m);return m

def main(*,wayland=False,artifact_module=None):
 if '--inside' in sys.argv:
  return module('admission_fixture',Path('/lab/admission-fixture.py')).inside()
 parser=argparse.ArgumentParser(add_help=False)
 parser.add_argument('--actual-pair',action='store_true');parser.add_argument('--theme-source',type=Path);parser.add_argument('--dms-source',type=Path)
 options,remaining=parser.parse_known_args();sys.argv[1:]=remaining;pair=options.actual_pair
 if pair and (not options.theme_source or not options.dms_source):parser.error('--actual-pair requires --theme-source and --dms-source')
 pair_sources={role:path.resolve(strict=True) for role,path in (('theme',options.theme_source),('dms',options.dms_source)) if path is not None}
 source=Path(__file__).resolve().parents[1];scripts=source/'scripts'
 fixture=module('admission_fixture',scripts/'test-collection-admission-native.py')
 fixture_path=Path(fixture.__file__)
 if '--guarded' not in sys.argv:sys.argv.append('--guarded')
 helpers=module('omabit_inert_tools',scripts/'test-omabit-service-drain-offscreen.py')
 sys.path.insert(0,str(source/'native/collections'));import artifacts as a;import guarded_artifacts as ga
 if artifact_module is not None:ga=artifact_module
 original_prepare=ga.prepare;original_run=fixture.subprocess.run;originals={}
 def prepare(root,pin,store,**kw):
  root=Path(root)
  if kw['role']=='omabit':
   # Replace only synthetic fixture input with copied actual product sources.
   # Repin the resulting test adaptation before calling the real preparer.
   assert a.inventory(a.snapshot(root,list(pin)))==pin
   names=[]
   for p in sorted((source/'plugins/omabit').iterdir()):
    if p.suffix in ('.qml','.js') or p.name in ('qmldir','manifest.json','omabit.svg'):
     shutil.copy2(p,root/p.name);originals[str(p)]=fixture.sha(p);names.append(p.name)
   (root/'WindowActivation.qml').write_text(helpers.FOCUS)
   (root/'ssh-hosts.py').write_text('print(\'{"hosts":[]}\\n\')\n');names+=['ssh-hosts.py']
   for name in ('urbit-collector','omabit','urbit-runner'):
    p=root/'bin'/name;p.write_text(helpers.HELPER if name!='urbit-runner' else '#!/bin/sh\nexit 69\n');p.chmod(0o700);names.append('bin/'+name)
   names+=['DESKTOP.json']
   service=root/'Service.qml';text=service.read_text()
   assert text.count('  id: root')==1 and text.count('  Component.onCompleted: {')==1
   text=text.replace('  id: root','  id: root\n  readonly property int fixtureVersion: 1\n  readonly property string fixtureHelper: registrationsLoaded && observing ? "1" : ""')
   text=text.replace('  Component.onCompleted: {','  Component.onCompleted: {\n    console.log("APPLICATION_CREATED actual-omabit "+moduleName)\n    if(moduleName.indexOf(".qualification-")<0)console.log("APPLICATION_BAD_ID")')
   service.write_text(text)
   panel=root/'Panel.qml';text=panel.read_text();assert 'Component.onCompleted' not in text
   text=text.replace('  id: root','  id: root\n  Component.onCompleted: {console.log("WIDGET_CREATED "+moduleName);if(moduleName.indexOf(".qualification-")<0 || !service || service.moduleName!==moduleName)console.log("WIDGET_BAD_ID")}')
   panel.write_text(text)
   # The files not listed here are prior synthetic inputs and are never copied.
   pin=a.inventory(a.snapshot(root,sorted(set(names))))
  elif pair:
   role=kw['role'];actual=pair_sources[role]
   assert a.inventory(a.snapshot(root,list(pin)))==pin
   names=[]
   files=[p for p in actual.iterdir() if p.is_file() and not p.name.startswith('test_') and (p.suffix in ('.qml','.js','.py') or p.name in ('qmldir','manifest.json'))]
   if role=='theme':files += [p for p in (actual/'client').rglob('*.py') if '__pycache__' not in p.parts]
   for p in files:
    rel=p.relative_to(actual);dest=root/rel;dest.parent.mkdir(parents=True,exist_ok=True);shutil.copy2(p,dest);names.append(str(rel));originals[str(p)]=fixture.sha(p)
   if role=='theme':
    target=root/'Service.qml';text=target.read_text()
    text=text.replace('  id: root','  id: root\n  readonly property int fixtureVersion: 1\n  readonly property string fixtureHelper: loaded && desktop.running ? "1" : ""')
    text=text.replace('  Component.onCompleted: {','  Component.onCompleted: { console.log("APPLICATION_CREATED actual-theme "+moduleName);if(moduleName.indexOf(".qualification-")<0)console.log("APPLICATION_BAD_ID");')
    target.write_text(text);widget=root/'Panel.qml'
   else:
    target=root/'Dms.qml';text=target.read_text();assert text.count('  PanelWindow {')==1
    if not wayland:
     text=text.replace('import Quickshell.Wayland\n','').replace('  PanelWindow {','  Rectangle {\n    width: 1024; height: 768').replace('    anchors { top: true; bottom: true; left: true; right: true }\n','')
     text='\n'.join(line for line in text.split('\n') if not line.strip().startswith(('WlrLayershell.','exclusionMode:')))
    text=text.replace('  id: root','  id: root\n  readonly property int fixtureVersion: 1\n  property string fixtureHelper: ""')
    text=text.replace('  Component.onCompleted: {','  Component.onCompleted: {\n    console.log("APPLICATION_CREATED actual-dms "+moduleName);if(moduleName.indexOf(".qualification-")<0)console.log("APPLICATION_BAD_ID")')
    text=text.replace('    if (data.event === "transport-state") {','    if (data.event === "ready") fixtureHelper="1"\n    if (data.event === "transport-state") {')
    target.write_text(text);widget=root/'BarWidget.qml'
   text=widget.read_text();assert 'Component.onCompleted' not in text
   check='!service || service.moduleName!==moduleName' if role=='theme' else '!Dm.State.sendHandler'
   text=text.replace('  id: root','  id: root\n  Component.onCompleted:{console.log("WIDGET_CREATED "+moduleName);if(moduleName.indexOf(".qualification-")<0 || '+check+')console.log("WIDGET_BAD_ID")}')
   widget.write_text(text);pin=a.inventory(a.snapshot(root,sorted(set(names))))
  return original_prepare(root,pin,store,**kw)
 ga.prepare=prepare
 def launch(command,*args,**kw):
  assert command[0]=='bwrap';out=Path(command[command.index('--bind')+1])
  native=Path('/usr/share/omarchy/shell')
  ui_names=['Panel','PanelController','BarIconButton','WidgetButton','OpticalGlyph','PanelToolTip','BorderSurface','BorderOverlay','PanelKeyCatcher']
  if pair:
   ui_fixture=module('pair_ui_fixture',scripts/'test-real-plugin-lifecycle-offscreen.py')
   ui_names=sorted(set(ui_names+ui_fixture.UI))
   originals[str(scripts/'test-real-plugin-lifecycle-offscreen.py')]=fixture.sha(scripts/'test-real-plugin-lifecycle-offscreen.py')
  ui_path=out/'imports/qs/Ui';ui_path.mkdir(parents=True,exist_ok=True)
  for name in ui_names:
   src=native/'Ui'/(name+'.qml');shutil.copy2(src,ui_path/src.name);originals[str(src)]=fixture.sha(src)
  (ui_path/'KeyboardPanel.qml').write_text('import QtQuick\nItem { property Item anchorItem;property var owner;property var bar;property bool open:false;property Item focusTarget;property int contentWidth:440;property int contentHeight:720;width:contentWidth;height:contentHeight;visible:open;function fittedContentWidth(n){return n} function fittedContentHeight(n,cap){return Math.min(n,cap)} }\n')
  (ui_path/'qmldir').write_text('module qs.Ui\n'+''.join(name+' 1.0 '+name+'.qml\n' for name in ui_names+['KeyboardPanel']))
  commons=out/'imports/qs/Commons'
  for name in ('Style.qml','Util.qml','Border.qml','BorderGeometry.js'):
   src=native/'Commons'/name;shutil.copy2(src,commons/name);originals[str(src)]=fixture.sha(src)
  (commons/'qmldir').write_text('module qs.Commons\nsingleton Style 1.0 Style.qml\nsingleton Color 1.0 Color.qml\nsingleton Util 1.0 Util.qml\nsingleton Border 1.0 Border.qml\n')
  (commons/'Color.qml').write_text('pragma Singleton\nimport QtQuick\nQtObject {property color background:"#171a1e";property color foreground:"#e7e9eb";property color accent:"#8ab4f8";property color urgent:"#ed8994";property var shellValues:({});property var menu:({background:"#171a1e",text:Qt.rgba(0.9,0.9,0.9,1),border:"#8ab4f8",scrim:"#88000000"});property var tooltip:({background:"#171a1e",text:Qt.rgba(0.9,0.9,0.9,1),border:"#8ab4f8"});property var popups:({background:"#171a1e",text:Qt.rgba(0.9,0.9,0.9,1),border:"#8ab4f8"})}\n')
  if pair:
   # No credentials or user configuration are mounted; supply only private
   # account metadata needed by ordinary Python libraries.
   etc=out/'etc';etc.mkdir(mode=0o700)
   (etc/'passwd').write_text('lab:x:'+str(os.getuid())+':'+str(os.getgid())+':Lab:/lab/home:/bin/sh\n')
   (etc/'group').write_text('lab:x:'+str(os.getgid())+':\n')
   command[1:1]=['--ro-bind',str(etc),'/etc']
   private_bin=out/'bin';private_bin.mkdir(mode=0o700)
   for name in ('omarchy','omarchy-shell','hyprctl','notify-send','xdg-open'):
    tool=private_bin/name;tool.write_text('#!/bin/sh\nexit 69\n');tool.chmod(0o700)
   at=command.index('PATH');assert command[at+1]=='/usr/bin:/bin';command[at+1]='/lab/bin:/usr/bin:/bin'
   fixture.save(out/'actual-pair-mode.json',{'actual_theme_dms':True,'accounts':'empty','theme_desktop_helper':True,'private_desktop_shims':True,'dms_window_offscreen':not wayland})
  shutil.copy2(fixture_path,out/'admission-fixture.py')
  hashes=json.loads((out/'source.json').read_text());hashes.update(originals)
  hashes[str(fixture_path)]=fixture.sha(fixture_path);hashes[str(scripts/'test-omabit-service-drain-offscreen.py')]=fixture.sha(scripts/'test-omabit-service-drain-offscreen.py')
  fixture.save(out/'source.json',hashes)
  fixture.save(out/'actual-omabit-mode.json',{'actual_service':True,'actual_panel':True,'inert_tools':True,'private_ui_source':'copied native controls with private values/window','private_ui_sha256':{str(p.relative_to(out)):fixture.sha(p) for folder in ('Ui','Commons') for p in (out/'imports/qs'/folder).iterdir()},'no_runner_operations':True,'other_roles_synthetic':not pair})
  return original_run(command,*args,**kw)
 fixture.subprocess.run=launch
 # The base preparation copies this driver; its private inner path dispatches
 # the unchanged common native admission scenario above.
 fixture.__file__=__file__
 return fixture.main(artifact_module=artifact_module)

if __name__=='__main__':main()
