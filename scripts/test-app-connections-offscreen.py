#!/usr/bin/env python3
"""Actual QML + Quickshell Process flow with inert backend and private desktop state."""
import argparse
import json
import os
from pathlib import Path
import shutil
import signal
import subprocess
import tempfile

ROOT = Path(__file__).resolve().parents[1]
COMPONENTS = ('HostedUrbitsPane.qml','Card.qml','Model.js','ApplicationConnections.qml','ApplicationsPane.qml','ActionButton.qml','SelectField.qml','Field.qml','CheckField.qml','Label.qml','Hint.qml','Design.qml')

HELPER = r'''
import json,sys
from pathlib import Path
root=Path(__file__).parent
args=sys.argv[1:];cmd=args[0]
with (root/'calls.jsonl').open('a') as f:f.write(json.dumps(args)+'\n')
p=root/'state.json'
state=json.loads(p.read_text()) if p.exists() else {'login':False,'attempts':[]}
plugin={'name':'Theme (Auth Lab)','status':'available','target':{'plugin_id':'fixture.theme.auth-lab','profile':'auth-lab','artifact_sha256':'a'*64}}
ship={'name':'lab','id':'b'*32,'revision':2,'origin':'http://127.0.0.1:18081','ship':'~zod','status':'session_stored' if state['login'] else 'login_required'}
exitcode=0
if cmd=='app-plugins':
    second=json.loads(json.dumps(plugin));second['name']='DMs (Auth Lab)';second['target']['plugin_id']='fixture.dms.auth-lab'
    result=[plugin,second]
elif cmd=='ships':result=[ship]
elif cmd=='app-connections':result=state['attempts']
elif cmd=='ship-login':
    assert '--code-stdin' in args and '--expected-ship-id' in args
    assert sys.stdin.read()=='fixture-only-secret'
    state['login']=True;result={'status':'authenticated'}
elif cmd=='app-connect':
    def arg(k):return args[args.index(k)+1]
    assert arg('--expected-ship-id')==ship['id'] and arg('--expected-ship-revision')=='2'
    assert arg('--artifact-sha256')=='a'*64 and arg('--profile')=='auth-lab'
    assert '--approve-owner-session' in args and not state['attempts']
    row={'request_id':arg('--request-id'),'revision':5,'origin':ship['origin'],'ship':ship['ship'],'target':plugin['target'],'phase':'connected','next_actions':['app-disconnect'],'receiver_reconciliation':True}
    state['attempts']=[row];result={'code':'ok','connection':row}
elif cmd in ('app-disconnect','app-recover','app-reconcile','app-archive'):
    row=state['attempts'][0]
    assert args[1]==row['request_id'] and args[2:]==['--expected-revision',str(row['revision'])]
    row['revision']+=1
    if cmd=='app-disconnect':
        row.update(phase='revoking',next_actions=['app-recover']);exitcode=1
        result={'code':'cleanup_pending','message':'Session cleanup is pending. Refresh and retry recovery; do not approve another session.','connection':row}
    elif cmd=='app-recover':
        row.update(phase='closed',next_actions=['app-reconcile']);result={'code':'ok','connection':row}
    elif cmd=='app-reconcile':
        row.update(receiver_reconciliation=False,next_actions=['app-archive']);result={'code':'ok','connection':row}
    else:
        assert row['phase']=='closed' and not row['receiver_reconciliation']
        row.update(archived=True,next_actions=[]);state['attempts']=[];result={'code':'ok','connection':row}
else:raise AssertionError('unexpected fixture command '+cmd)
p.write_text(json.dumps(state))
print(json.dumps(result))
sys.exit(exitcode)
'''


def main():
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--output',type=Path,help='fresh /tmp/omabit-app-ui-NAME directory to retain evidence')
    args=parser.parse_args()
    executable=shutil.which('quickshell')
    if not executable:parser.error('Quickshell is required')
    temporary=None
    if args.output:
        directory=args.output.absolute()
        if directory.parent!=Path('/tmp') or not directory.name.startswith('omabit-app-ui-') or directory.exists():parser.error('use a fresh /tmp/omabit-app-ui-NAME')
        directory.mkdir(mode=0o700)
    else:
        temporary=tempfile.TemporaryDirectory(prefix='omabit-app-ui-');directory=Path(temporary.name)
    try:
        for name in ('home','config','state','data','cache','runtime'):(directory/name).mkdir(mode=0o700)
        plugin=directory/'Plugin';plugin.mkdir()
        for name in COMPONENTS:shutil.copyfile(ROOT/'plugins/omabit'/name,plugin/name)
        (plugin/'qmldir').write_text('singleton Design 1.0 Design.qml\n' + ''.join(name[:-4]+' 1.0 '+name+'\n' for name in COMPONENTS if name != 'Design.qml' and name.endswith('.qml')))
        commons=directory/'Commons';commons.mkdir()
        (commons/'qmldir').write_text('singleton Style 1.0 Style.qml\nsingleton Color 1.0 Color.qml\n')
        (commons/'Style.qml').write_text('pragma Singleton\nimport QtQuick\nQtObject { function space(n) {return n} property var font: ({family:"sans-serif",caption:12,bodySmall:14,body:14}) }\n')
        (commons/'Color.qml').write_text('pragma Singleton\nimport QtQuick\nQtObject {property color background:"#171a1e";property color foreground:"#e7e9eb";property color accent:"#8ab4f8";property color urgent:"#ed8994"}\n')
        helper=directory/'backend.py';helper.write_text(HELPER)
        qml=(ROOT/'scripts/test-app-connections-offscreen.qml').read_text().replace('@@HELPER@@',json.dumps(str(helper))).replace('@@SNAPSHOT@@',json.dumps(str(directory/'approval.png')))
        shell=directory/'shell.qml';shell.write_text(qml)
        env={'PATH':'/usr/bin:/bin','LANG':'C.UTF-8','HOME':str(directory/'home'),'XDG_CONFIG_HOME':str(directory/'config'),'XDG_STATE_HOME':str(directory/'state'),'XDG_DATA_HOME':str(directory/'data'),'XDG_CACHE_HOME':str(directory/'cache'),'XDG_RUNTIME_DIR':str(directory/'runtime'),'QT_QPA_PLATFORM':'offscreen','QT_QUICK_BACKEND':'software','QSG_RHI_BACKEND':'software','QT_QUICK_CONTROLS_STYLE':'Basic','QML_DISABLE_DISK_CACHE':'1','NO_COLOR':'1'}
        # No inherited display, compositor, DBus/keyring, plugin or runner socket.
        process=subprocess.Popen([executable,'--no-color','-p',str(shell)],env=env,stdout=subprocess.PIPE,stderr=subprocess.STDOUT,text=True,start_new_session=True)
        try:text,_=process.communicate(timeout=45)
        except subprocess.TimeoutExpired:
            os.killpg(process.pid,signal.SIGKILL);text,_=process.communicate();raise AssertionError(text)
        (directory/'qml.log').write_text(text)
        assert process.returncode==0 and 'PASS: actual application connection panel' in text,text
        assert not any(token in text for token in ('Error:','Unable to assign','is not defined','Cannot assign to','Binding loop detected')),text
        calls=[json.loads(line) for line in (directory/'calls.jsonl').read_text().splitlines()]
        mutations=[c for c in calls if c[0] not in ('app-plugins','ships','app-connections')]
        assert [c[0] for c in mutations]==['ship-login','app-connect','app-disconnect','app-recover','app-reconcile','app-archive'],mutations
        assert 'fixture-only-secret' not in json.dumps(calls),calls
        assert (directory/'approval.png').is_file(),'approval UI snapshot missing'
        print('PASS: actual application connection panel, explicit approval, private stdin, revision binding, failed disconnect, manual recovery, completed disconnect archival, and separate hosted login; no automatic approval')
        if args.output:print(directory)
    finally:
        if temporary:temporary.cleanup()


if __name__=='__main__':main()
