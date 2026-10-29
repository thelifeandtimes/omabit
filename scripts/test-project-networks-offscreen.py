#!/usr/bin/env python3
"""Actual project QML/Process with an inert helper in a networkless desktop namespace."""
import argparse,hashlib,json,os,re,shutil,subprocess
from pathlib import Path
ROOT=Path(__file__).resolve().parents[1]
COMPONENTS=('ProjectCreatePane.qml','ProjectCreation.qml','Field.qml','CheckField.qml','ReaderLifecycle.qml','ProjectNetworks.qml','ProjectNetworksPane.qml','Model.js','ActionButton.qml','SelectField.qml','Label.qml','Hint.qml','Design.qml')
HELPER=r'''
import json,signal,sys,time
from pathlib import Path
scenario,profile,*args=sys.argv[1:]
assert args[:4]==['--runner','lab-a','--expected-runner-id','runner-a'],args
assert args[4] in ('projects','project-status') and (args[5:]==[] if args[4]=='projects' else args[5:]==['project-one']),args
with Path('/lab/calls.jsonl').open('a') as f:f.write(json.dumps({'scenario':scenario,'profile':profile,'args':args})+'\n')
p={'id':'project-one','label':'Messaging lab','revision':1,'created_at':'2026-10-24T00:00:00Z','review_digest':'a'*64,'members':[{'instance_id':'vm-zod','ship':'~zod','http_port':18381,'internal_http_port':8080},{'instance_id':'vm-nec','ship':'~nec','http_port':18382,'internal_http_port':8081}]}
v={'schema_version':1,'runner_id':'runner-a','mode':'development-managed','read_at':'2026-10-24T00:01:00Z','runtime_observation':'not-performed'}
if args[4]=='projects':v.update(basis='stored-reservations',projects=[] if scenario=='empty' else [p])
else:
    ops=[{'id':'op-'+str(i),'request_id':'req-'+str(i),'kind':'stop-project-member' if i==0 else 'launch-project-member','actor':'local-uid:1000','status':'interrupted' if i==0 else 'succeeded','config_revision':1,'instance_id':'vm-zod','created_at':'2026-10-24T00:00:01Z','finished_at':'2026-10-24T00:00:02Z'} for i in range(128 if scenario=='older' else 3)]
    v.update(basis='stored-operation-history',project=p,operations=ops,has_older_operations=scenario=='older')
if scenario=='mismatch':v['runner_id']='runner-b'
if scenario=='schema':v['schema_version']=2
if scenario=='malformed':print('{');sys.exit(0)
if scenario=='members':p['members']=None
if scenario=='changed' and args[4]=='project-status':p['revision']=2
if scenario=='delay' and args[4]=='project-status':
    signal.signal(signal.SIGTERM,lambda *_:None)
    Path('/lab/delay-started').touch();time.sleep(.5);Path('/lab/delay-completed').write_text(profile)
if scenario=='timeout':time.sleep(10)
if scenario=='oversize':print('x'*(2*1024*1024));sys.exit(0)
print(json.dumps(v))
'''
def main():
    p=argparse.ArgumentParser(description=__doc__);p.add_argument('--output',type=Path,required=True);a=p.parse_args();out=a.output.absolute()
    if out.parent!=Path('/tmp') or not re.fullmatch('omabit-project-ui-[A-Za-z0-9-]+',out.name) or out.exists():p.error('use a fresh /tmp/omabit-project-ui-NAME')
    binary=shutil.which('quickshell');assert binary
    out.mkdir(mode=0o700)
    for name in ('home','config','data','state','cache','runtime'):(out/name).mkdir(mode=0o700)
    plug=out/'Plugin';plug.mkdir()
    for name in COMPONENTS:shutil.copy2(ROOT/'plugins/omabit'/name,plug/name)
    (plug/'qmldir').write_text('singleton Design 1.0 Design.qml\n'+''.join(n[:-4]+' 1.0 '+n+'\n' for n in COMPONENTS if n.endswith('.qml') and n!='Design.qml'))
    commons=out/'Commons';commons.mkdir();(commons/'qmldir').write_text('singleton Style 1.0 Style.qml\nsingleton Color 1.0 Color.qml\n')
    (commons/'Style.qml').write_text('pragma Singleton\nimport QtQuick\nQtObject { function space(n) {return n} property var font: ({family:"sans-serif",caption:12,bodySmall:14,body:14}) }\n')
    (commons/'Color.qml').write_text('pragma Singleton\nimport QtQuick\nQtObject {property color background:"#171a1e";property color foreground:"#e7e9eb";property color accent:"#8ab4f8";property color urgent:"#ed8994"}\n')
    (out/'backend.py').write_text(HELPER);shutil.copy2(ROOT/'scripts/test-project-networks-offscreen.qml',out/'shell.qml')
    environment={'PATH':'/usr/bin:/bin','LANG':'C.UTF-8','HOME':'/lab/home','XDG_CONFIG_HOME':'/lab/config','XDG_STATE_HOME':'/lab/state','XDG_DATA_HOME':'/lab/data','XDG_CACHE_HOME':'/lab/cache','XDG_RUNTIME_DIR':'/lab/runtime','QT_QPA_PLATFORM':'offscreen','QT_QUICK_BACKEND':'software','QSG_RHI_BACKEND':'software','QT_QUICK_CONTROLS_STYLE':'Basic','QML_DISABLE_DISK_CACHE':'1','NO_COLOR':'1'}
    command=['bwrap','--unshare-all','--die-with-parent','--clearenv','--ro-bind','/usr','/usr','--symlink','usr/bin','/bin','--symlink','usr/lib','/lib','--symlink','usr/lib','/lib64','--ro-bind','/etc/fonts','/etc/fonts','--proc','/proc','--dev','/dev','--tmpfs','/tmp','--bind',str(out),'/lab','--chdir','/lab']
    for k,v in environment.items():command+=['--setenv',k,v]
    command+=['--',binary,'--no-color','-p','/lab/shell.qml']
    q=subprocess.run(command,capture_output=True,text=True,timeout=40);log=q.stdout+q.stderr;(out/'qml.log').write_text(log)
    assert q.returncode==0 and 'PASS: public project UI' in log,log
    assert not any(t in log for t in ('Error:','FAIL:','Unable to assign','Cannot assign','is not defined','Binding loop')),log
    calls=[json.loads(line) for line in (out/'calls.jsonl').read_text().splitlines()]
    assert calls and all(c['args'][:4]==['--runner','lab-a','--expected-runner-id','runner-a'] and c['args'][4] in ('projects','project-status') for c in calls)
    assert (out/'history.png').exists()
    assert (out/'delay-completed').read_text()=='dev','late old-profile helper did not finish'
    sources=['plugins/omabit/'+n for n in COMPONENTS]+['scripts/test-project-networks-offscreen.py','scripts/test-project-networks-offscreen.qml']
    result={'passed':True,'exit':q.returncode,'read_calls':len(calls),'mutations':0,'source_sha256':{n:hashlib.sha256((ROOT/n).read_bytes()).hexdigest() for n in sources},'scope':'actual copied QML and Quickshell Process; inert helper; private HOME/XDG; offscreen/no-NIC namespace; no host runner/desktop/DBus/engine sockets','evidence_sha256':{n:hashlib.sha256((out/n).read_bytes()).hexdigest() for n in ('history.png','qml.log','calls.jsonl')}}
    (out/'result.json').write_text(json.dumps(result,indent=2)+'\n');print('PASS: public project UI, read-only pinned commands and stale-response rejection');print(out)
if __name__=='__main__':main()
