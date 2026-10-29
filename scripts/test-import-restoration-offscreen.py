#!/usr/bin/env python3
"""Run the actual storage/restoration QML with inert dispatch in a private namespace."""
import argparse,hashlib,importlib.util,json,re,shutil,subprocess
from pathlib import Path
ROOT=Path(__file__).resolve().parents[1]

def main():
 p=argparse.ArgumentParser(description=__doc__);p.add_argument('--output',type=Path,required=True);args=p.parse_args();out=args.output.absolute()
 if out.parent!=Path('/tmp') or not re.fullmatch('omabit-restore-ui-[A-Za-z0-9-]+',out.name) or out.exists():p.error('use a fresh /tmp/omabit-restore-ui-NAME directory')
 qml=shutil.which('qml6')
 if not qml:p.error('Qt 6 qml6 is required')
 out.mkdir(mode=0o700)
 for leaf in ['home','config','data','state','cache','runtime']:(out/leaf).mkdir(mode=0o700)
 plugin=out/'Plugin';plugin.mkdir()
 sources=[]
 for source in sorted((ROOT/'plugins/omabit').glob('*')):
  if source.suffix in ['.qml','.js']:shutil.copyfile(source,plugin/source.name);sources.append(source)
 (plugin/'qmldir').write_text('singleton Design 1.0 Design.qml\n')
 commons=out/'qs/Commons';commons.mkdir(parents=True)
 (commons/'qmldir').write_text('module qs.Commons\nsingleton Style 1.0 Style.qml\nsingleton Color 1.0 Color.qml\n')
 (commons/'Style.qml').write_text('pragma Singleton\nimport QtQuick\nQtObject {function space(n){return n} property var font:({family:"sans-serif",caption:12,bodySmall:14,body:14})}\n')
 (commons/'Color.qml').write_text('pragma Singleton\nimport QtQuick\nQtObject {property color background:"#171a1e";property color foreground:"#e7e9eb";property color accent:"#8ab4f8";property color urgent:"#ed8994"}\n')
 # This source helper has no initialization side effects. Never import the live
 # test-panel.py or desktop_fixture.py modules in this qualification harness.
 helper=ROOT/'scripts/test-import-ui-offscreen.py';spec=importlib.util.spec_from_file_location('inert_import_fixture',helper);module=importlib.util.module_from_spec(spec);spec.loader.exec_module(module)
 prior=(ROOT/'scripts/test-rc11-panel-offscreen.qml').read_text()
 fixture=prior[prior.index('  QtObject {\n    id: svc'):prior.index('  function check(')]
 fixture=fixture.replace('    property var externalImportPreview: null','    property var pendingExternalPreview: null\n    property var externalImportPreview: null\n    readonly property P.ImportRestoration importRestoration: P.ImportRestoration {service:svc}')
 service=(ROOT/'plugins/omabit/Service.qml').read_text()
 fixture=fixture.replace('@@SERVICE_METHODS@@','\n'.join(module.method(service,n) for n in module.METHODS))
 fixture=fixture.replace('@@CONNECTIONS_CHANGED@@',re.search(r'^  onConnectionsChanged: .*$',service,re.M)[0])
 template=ROOT/'scripts/test-import-restoration-offscreen.qml'
 (out/'main.qml').write_text(template.read_text().replace('@@SERVICE_FIXTURE@@',fixture).replace('@@OUTPUT@@',json.dumps(str(out))))
 env={'PATH':'/usr/bin:/bin','LANG':'C.UTF-8','HOME':str(out/'home'),'XDG_CONFIG_HOME':str(out/'config'),'XDG_STATE_HOME':str(out/'state'),'XDG_DATA_HOME':str(out/'data'),'XDG_CACHE_HOME':str(out/'cache'),'XDG_RUNTIME_DIR':str(out/'runtime'),'QT_QPA_PLATFORM':'offscreen','QT_QUICK_BACKEND':'software','QSG_RHI_BACKEND':'software','QT_QUICK_CONTROLS_STYLE':'Basic','QML_DISABLE_DISK_CACHE':'1','QT_LOGGING_RULES':'*.warning=true;*.critical=true;qml.debug=true','QT_FORCE_STDERR_LOGGING':'1'}
 command=['bwrap','--unshare-all','--die-with-parent','--clearenv','--ro-bind','/usr','/usr','--symlink','usr/bin','/bin','--symlink','usr/lib','/lib','--symlink','usr/lib','/lib64','--ro-bind','/etc/fonts','/etc/fonts','--proc','/proc','--dev','/dev','--tmpfs','/tmp','--dir','/run','--bind',str(out),str(out),'--chdir',str(out)]
 for name,value in env.items():command+=['--setenv',name,value]
 command+=['--',qml,'-I',str(out),str(out/'main.qml')]
 (out/'command.json').write_text(json.dumps(command,indent=2)+'\n')
 result=subprocess.run(command,capture_output=True,text=True,timeout=40)
 log=result.stdout+result.stderr;(out/'qml.log').write_text(log)
 assert result.returncode==0 and 'PASS: actual restoration review' in log,log
 assert not any(v in log for v in ['Error:','FAIL','Unable to assign','Binding loop','Cannot assign','is not defined']),log
 rows=[json.loads(line.split('CALLS ',1)[1]) for line in log.splitlines() if 'CALLS ' in line];assert len(rows)==1;calls=rows[0]
 assert [c['kind'] for c in calls]==['preview-import-restore','preview-import-restore','preview-import-restore','restore-import','preview-import-restore','restore-import','preview-import-restore','restore-import','preview-pier'],calls
 for c in calls:
  a=c['args'];assert a[:4]==['--runner','restore-lab','--expected-runner-id','runner-restore-lab'],a
  if c['kind']!='preview-pier':assert a[a.index('--expected-actor')+1]=='local-uid:1000'
  if c['kind']=='restore-import':assert '--confirm-restore' in a and a[a.index('--restore-digest')+1]=='a'*64
 for name in ['refusal','review','uncertain','restored','interrupted','continuation']:assert (out/(name+'.png')).is_file(),name
 sources += [helper,template,Path(__file__),ROOT/'scripts/test-rc11-panel-offscreen.qml']
 evidence={'ok':True,'exit':result.returncode,'calls':calls,'source_sha256':{str(s.relative_to(ROOT)):hashlib.sha256(s.read_bytes()).hexdigest() for s in sources},'evidence_sha256':{f.name:hashlib.sha256(f.read_bytes()).hexdigest() for f in out.glob('*.png')},'scope':'Actual storage and restoration QML and verbatim service finish/import callbacks. Inert dispatch, private HOME/XDG and Bubblewrap user/network/mount/PID/IPC namespace; no host home, run, desktop, engine, keyring or real runner.'}
 evidence['evidence_sha256']['qml.log']=hashlib.sha256(log.encode()).hexdigest()
 (out/'result.json').write_text(json.dumps(evidence,indent=2)+'\n');print('PASS: actual restoration UI; '+str(out))
if __name__=='__main__':main()
