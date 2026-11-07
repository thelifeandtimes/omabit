#!/usr/bin/env python3
"""Exercise the real policy editor with an inert helper in a private desktop."""
import argparse, json, os, re, shutil, subprocess
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
COMPONENTS = ('AppDevelopment.qml', 'AppDevelopmentPane.qml', 'AppPolicy.js', 'ReaderLifecycle.qml', 'Field.qml', 'ActionButton.qml', 'SelectField.qml', 'Label.qml', 'Hint.qml', 'Design.qml')
HELPER = r'''
import json, signal, sys, time
from pathlib import Path
scenario, profile, command, *args = sys.argv[1:]
assert command in ('app-development-status','app-development-save','app-development-retry')
path=Path('/lab/policy.json')
v=json.loads(path.read_text())
with Path('/lab/calls.jsonl').open('a') as f:f.write(json.dumps({'profile':profile,'command':command,'args':args})+'\n')
if command=='app-development-save':
    assert args[0]=='--request-json'
    r=json.loads(args[1]);u=r['update'];p=v['runners'][0]['status']['policy']
    if u['scope']=='global':
        assert r['revision']==v['global']['revision']
        v['global'].update(revision=r['revision']+1,gates=u['gates'],runners=r['runners'])
        p['global']=u['gates'];v['global']['runners'][0]['applied']=True
    else:
        assert r['runner_id']=='runner-lab' and u['expected_revision']==p['revision']
        if u['scope']=='runner':p['runner']=u['gates']
        else:p['entries']=[e for e in p['entries'] if e['target']!=u['target']]+[{'target':u['target'],'gates':u['gates']}]
    p['revision']+=1;path.write_text(json.dumps(v))
if scenario=='delay':
    signal.signal(signal.SIGTERM,lambda *_:None)
    time.sleep(.6);Path('/lab/late-finished').write_text(profile)
if scenario=='malformed':print('{');sys.exit(0)
if scenario=='mismatch':v['runners'][0]['status']['runner_id']='another-runner'
if scenario=='offline':v['runners'][0].update(status=None,can_edit=False,error='Runner unavailable',rollout='pending')
if scenario=='observer':v['runners'][0]['can_edit']=False
print(json.dumps(v))
'''

def main():
    p=argparse.ArgumentParser(description=__doc__);p.add_argument('--output',type=Path,required=True);a=p.parse_args();out=a.output.absolute()
    if out.parent!=Path('/tmp') or not re.fullmatch('omabit-policy-ui-[A-Za-z0-9-]+',out.name) or out.exists():p.error('use a fresh /tmp/omabit-policy-ui-NAME')
    binary=shutil.which('quickshell');assert binary
    out.mkdir(mode=0o700)
    for n in ('home','config','state','data','cache','runtime'):(out/n).mkdir(mode=0o700)
    plugin=out/'Plugin';plugin.mkdir()
    for n in COMPONENTS:shutil.copy2(ROOT/'plugins/omabit'/n,plugin/n)
    (plugin/'qmldir').write_text('singleton Design 1.0 Design.qml\n'+''.join(n[:-4]+' 1.0 '+n+'\n' for n in COMPONENTS if n.endswith('.qml') and n!='Design.qml'))
    commons=out/'Commons';commons.mkdir();(commons/'qmldir').write_text('singleton Style 1.0 Style.qml\nsingleton Color 1.0 Color.qml\n')
    (commons/'Style.qml').write_text('pragma Singleton\nimport QtQuick\nQtObject { function space(n) {return n} property var font: ({family:"sans-serif",caption:12,bodySmall:14,body:14}) }\n')
    (commons/'Color.qml').write_text('pragma Singleton\nimport QtQuick\nQtObject {property color background:"#171a1e";property color foreground:"#e7e9eb";property color accent:"#8ab4f8";property color urgent:"#ed8994"}\n')
    blocked={'commit':'block','publish':'block'};inherit={'commit':'inherit','publish':'inherit'}
    v={'version':1,'global':{'version':1,'revision':0,'gates':blocked,'runners':[]},'runners':[{'name':'lab','runner_id':'runner-lab','can_edit':True,'rollout':'not enrolled in global rollout','status':{'version':1,'runner_id':'runner-lab','mode':'simulation','read_at':'2026-11-06T00:00:00Z','desk_operations':False,'policy':{'version':1,'revision':0,'global':blocked,'runner':inherit,'entries':[]},'targets':[{'target':{'instance_id':'publisher'},'ship':'~zod','label':'Publisher','ready':True}]}}]}
    (out/'policy.json').write_text(json.dumps(v));(out/'backend.py').write_text(HELPER)
    shutil.copy2(ROOT/'scripts/test-app-development-offscreen.qml',out/'shell.qml')
    env={'PATH':'/usr/bin:/bin','LANG':'C.UTF-8','HOME':'/lab/home','XDG_CONFIG_HOME':'/lab/config','XDG_STATE_HOME':'/lab/state','XDG_DATA_HOME':'/lab/data','XDG_CACHE_HOME':'/lab/cache','XDG_RUNTIME_DIR':'/lab/runtime','QT_QPA_PLATFORM':'offscreen','QT_QUICK_BACKEND':'software','QSG_RHI_BACKEND':'software','QT_QUICK_CONTROLS_STYLE':'Basic','QML_DISABLE_DISK_CACHE':'1','NO_COLOR':'1'}
    command=['bwrap','--unshare-all','--die-with-parent','--clearenv','--ro-bind','/usr','/usr','--symlink','usr/bin','/bin','--symlink','usr/lib','/lib','--symlink','usr/lib','/lib64','--ro-bind','/etc/fonts','/etc/fonts','--proc','/proc','--dev','/dev','--tmpfs','/tmp','--bind',str(out),'/lab','--chdir','/lab']
    for k,v in env.items():command+=['--setenv',k,v]
    q=subprocess.run(command+['--',binary,'--no-color','-p','/lab/shell.qml'],capture_output=True,text=True,timeout=35)
    log=q.stdout+q.stderr;(out/'qml.log').write_text(log)
    assert q.returncode==0 and 'PASS: app policy editor' in log,log
    assert not any(s in log for s in ('Error:','FAIL:','Unable to assign','Cannot assign','is not defined','Binding loop')),log
    calls=[json.loads(s) for s in (out/'calls.jsonl').read_text().splitlines()]
    assert len([c for c in calls if c['command']=='app-development-save'])==4,calls
    assert (out/'late-finished').read_text()=='dev'
    assert (out/'policy.png').exists()
    print('PASS: four policy scopes, defaults, inheritance, observer/offline refusal, malformed identity and stale-profile response rejection')
    print(out)

if __name__=='__main__':main()
