#!/usr/bin/env python3
"""Private inert saved-recording UI; never selects an installed namespace."""
import argparse,json,re,shutil,subprocess
from pathlib import Path
ROOT=Path(__file__).resolve().parents[1]
PARTS=('Recordings.qml','RecordingsPane.qml','Recordings.js','Model.js','ReaderLifecycle.qml','SwitchField.qml','ActionButton.qml','Label.qml','Hint.qml','Design.qml','Card.qml','CardTitle.qml')
HELPER=r'''
import json,sys,time,signal
from pathlib import Path
scenario,profile,command,operation,*args=sys.argv[1:]
assert command=='call' and args[0]=='--input-json'
request=json.loads(args[1]);assert request['name']=='lab' and request['runner_id']=='runner-lab' and request['include_private_data']is True
with Path('/lab/calls.jsonl').open('a')as f:f.write(json.dumps({'profile':profile,'scenario':scenario,'operation':operation,'request':request})+'\n')
if scenario=='delay':signal.signal(signal.SIGTERM,lambda *_:None);time.sleep(.5)
s={'job_id':'1'*32,'job_sha256':'2'*64,'runner_id':'runner-lab','mode':'development-managed','instance_id':'one','runtime_id':'3'*32,'configuration_revision':1,'container_id':'4'*64,'operation_id':'start-original','request_id':'request-original','created_at':'2026-11-18T00:00:00Z'}
now='2026-11-18T00:01:00Z'
if operation=='runtime.recordings.v1':data={'version':1,'runner_id':'runner-lab','instance_id':'one','sessions':[s],'read_at':now,'basis':'saved fixture only'}
elif operation=='runtime.recording-controls.v1':data={'version':1,'state':{'version':1,'runner_id':'runner-lab','revision':0,'digest':'','pins':[],'policy':{'version':1,'max_jobs':16,'max_session_bytes':67108864,'host_max_raw_bytes':1073741824,'instance_max_raw_bytes':268435456,'reserve_free_bytes':2147483648}},'capacity_status':'unknown','pilot_enabled':True,'read_at':now}
elif operation=='runtime.recording-page.v1':
 assert request['job_id']==s['job_id']and request['job_sha256']==s['job_sha256']
 m={'runner_id':s['runner_id'],'instance_id':'one','container_id':s['container_id'],'operation_id':s['operation_id'],'request_id':s['request_id'],'session_id':'5'*32}
 data={'version':1,'session':s,'status':{'state':'attachment-ready','session_storage':'header-available','session_id_basis':'allocation','session_id':'5'*32,'writer_present':True},'representation':request['representation'],'from':request.get('cursor'),'metadata':m,'metadata_sha256':'6'*64,'frame_count':1,'read_at':now,'tail':'snapshot-eof','complete_export':False}
 frame={'Kind':1,'Sequence':1,'UnixNano':1794950400000000000}
 if request['representation']=='original':data['original']=[{'frame':frame,'original_bytes_base64':'UFJJVkFURS1PUklHSU5BTA=='}]
 else:data['redacted']={'representation':'escaped-literal-plain-text','frames':[{'frame':frame,'spans':[{'offset':0,'bytes':20,'escaped_text':'<b>literal</b>\\u001b[2K'}]}]}
 if scenario=='foreign':data['session']={**s,'job_id':'7'*32}
else:raise AssertionError('unexpected operation')
print(json.dumps({'version':1,'operation':operation,'data':data}))
'''
def main():
 p=argparse.ArgumentParser(description=__doc__);p.add_argument('--run',action='store_true');p.add_argument('--output',type=Path);a=p.parse_args()
 if not a.run:p.print_help();return
 out=a.output
 if out is None or out.parent!=Path('/tmp')or out.exists()or not re.fullmatch('omabit-recordings-ui-[A-Za-z0-9-]+',out.name):p.error('fresh private /tmp/omabit-recordings-ui-NAME required')
 out.mkdir(mode=0o700)
 for n in ('home','config','state','data','cache','runtime','Plugin','Commons'):(out/n).mkdir(mode=0o700)
 for n in PARTS:shutil.copy2(ROOT/'plugins/omabit'/n,out/'Plugin'/n)
 (out/'Plugin/qmldir').write_text('singleton Design 1.0 Design.qml\n'+''.join(n[:-4]+' 1.0 '+n+'\n'for n in PARTS if n.endswith('.qml')and n!='Design.qml'))
 (out/'Commons/qmldir').write_text('singleton Style 1.0 Style.qml\nsingleton Color 1.0 Color.qml\n')
 (out/'Commons/Style.qml').write_text('pragma Singleton\nimport QtQuick\nQtObject {function space(n){return n} property var font:({family:"monospace",caption:10,bodySmall:11,body:12})}\n')
 (out/'Commons/Color.qml').write_text('pragma Singleton\nimport QtQuick\nQtObject {property color background:"#101315";property color foreground:"#cacccc";property color accent:"#94b4a4";property color urgent:"#cc7777"}\n')
 (out/'backend.py').write_text(HELPER);shutil.copy2(ROOT/'scripts/test-recordings-offscreen.qml',out/'shell.qml')
 env={'PATH':'/usr/bin:/bin','LANG':'C.UTF-8','HOME':'/lab/home','XDG_CONFIG_HOME':'/lab/config','XDG_STATE_HOME':'/lab/state','XDG_DATA_HOME':'/lab/data','XDG_CACHE_HOME':'/lab/cache','XDG_RUNTIME_DIR':'/lab/runtime','QT_QPA_PLATFORM':'offscreen','QT_QUICK_BACKEND':'software','QSG_RHI_BACKEND':'software','QT_QUICK_CONTROLS_STYLE':'Basic','QML_DISABLE_DISK_CACHE':'1','NO_COLOR':'1'}
 command=['bwrap','--unshare-all','--die-with-parent','--clearenv','--ro-bind','/usr','/usr','--symlink','usr/bin','/bin','--symlink','usr/lib','/lib','--symlink','usr/lib','/lib64','--ro-bind','/etc/fonts','/etc/fonts','--proc','/proc','--dev','/dev','--tmpfs','/tmp','--bind',str(out),'/lab','--chdir','/lab']
 for k,v in env.items():command+=['--setenv',k,v]
 q=subprocess.run(command+['--',shutil.which('quickshell'),'--no-color','-p','/lab/shell.qml'],capture_output=True,text=True,timeout=25)
 log=q.stdout+q.stderr;(out/'qml.log').write_text(log)
 assert q.returncode==0 and 'PASS: private saved recording UI'in log,log
 assert not any(x in log for x in ('FAIL:','Unable to assign','Cannot assign','is not defined','Binding loop')),log
 calls=[json.loads(line)for line in (out/'calls.jsonl').read_text().splitlines()]
 assert len(calls)==7 and all(c['request']['include_private_data']is True for c in calls),calls
 assert len([c for c in calls if c['operation']=='runtime.recording-page.v1'and c['request']['representation']=='original'])==1,calls
 print(json.dumps({'passed':True,'calls':len(calls),'scope':'inert private desktop only; consent, explicit encoding, plain text, unknown storage and stale-profile rejection; no installed state'}))
if __name__=='__main__':main()
