"""Public CLI and real approval pane in the guarded fresh-fake namespace only."""
import json
import os
from pathlib import Path
import re
import selectors
import secrets
import shutil
import subprocess
import sys
import time

from qualify import check_namespace

ORIGIN='http://127.0.0.1:18081'
CODE=b'lidlut-tabwed-pillex-ridrup'  # Public tutorial code; only the fresh fake.
CANONICAL={'theme':'thelifeandtimes.urbit-theme','dms':'urbit.dms'}
events=[]


def stage(profile):
    root=Path('/lab')/profile
    root.mkdir(mode=0o700)
    env=dict(os.environ)
    for name,leaf in (('HOME','home'),('XDG_CONFIG_HOME','config'),('XDG_STATE_HOME','state'),('XDG_DATA_HOME','data'),('XDG_CACHE_HOME','cache'),('XDG_RUNTIME_DIR','runtime')):
        path=root/leaf;path.mkdir(mode=0o700);env[name]=str(path)
    for name in ('WAYLAND_DISPLAY','DISPLAY','HYPRLAND_INSTANCE_SIGNATURE','QML_IMPORT_PATH','QML2_IMPORT_PATH'):
        env.pop(name,None)
    install=Path(env['HOME'])/'.config/omarchy/plugins'
    install.mkdir(parents=True,mode=0o700)
    for kind,canonical in CANONICAL.items():
        destination=install/(canonical+'.'+profile)
        shutil.copytree(Path('/artifacts/plugins')/kind,destination)
        manifest=json.loads((destination/'manifest.json').read_text())
        assert manifest['id']==canonical+'.auth-lab'
        manifest['id']=canonical+'.'+profile
        manifest['name'] += ' / '+profile
        if 'barWidget' in manifest:manifest['barWidget']['displayName']=manifest['name']
        (destination/'manifest.json').write_text(json.dumps(manifest))
        (destination/'application-profile.json').write_text(json.dumps({'version':1,'profile':profile}))
    return root,env,install


def command(root,env,args,expected=0,private=None):
    result=subprocess.run(['/artifacts/urbit-collector','--config-dir',str(root/'collector'),'--format','json']+args,
                          input=private,stdout=subprocess.PIPE,stderr=subprocess.PIPE,env=env,timeout=85)
    # Never emit arbitrary captured output or credentials on failure.
    assert result.returncode==expected, 'CLI command failed: '+args[0]
    for value in (CODE,b'cookie_value',b'session_ref'):
        assert value not in result.stdout+result.stderr,'private material in public CLI output'
    events.append({'profile':root.name,'command':args[0],'exit':result.returncode})
    data=result.stdout if result.stdout else result.stderr
    return json.loads(data)


def probe(root,env,install,kind,expected=0):
    result=subprocess.run(['/usr/bin/python3','-B','/artifacts/plugin_probe.py',kind,'read',str(install/(CANONICAL[kind]+'.'+root.name)),root.name],
                          env=env,stdout=subprocess.PIPE,stderr=subprocess.PIPE,timeout=25)
    assert result.returncode==expected,'direct plugin probe failed: '+kind
    if expected==0:assert json.loads(result.stdout).get('ok') is True,'direct plugin probe did not confirm success'
    events.append({'profile':root.name,'direct_plugin':kind,'expected_success':expected==0})


def cleanup(root,env,install):
    attempts=command(root,env,['app-connections'])
    assert len(attempts)==2 and all(a['phase']=='connected' for a in attempts)
    parent=command(root,env,['ships'])[0]
    by_kind={kind:next(a for a in attempts if a['target']['plugin_id']==canonical+'.'+root.name) for kind,canonical in CANONICAL.items()}
    # No approval UI, collector service or owner client is running during probes.
    probe(root,env,install,'theme');probe(root,env,install,'dms')
    assert command(root,env,['ship-status','lab'])['status']=='authenticated'
    for index,kind in enumerate(('theme','dms')):
        row=by_kind[kind]
        # A harmless installed helper update must not replace the approved code
        # used for receiver cleanup; the retained generation is authoritative.
        declaration=json.loads((install/row['target']['plugin_id']/'application-connection.json').read_text())
        entry=install/row['target']['plugin_id']/declaration['entry']
        entry.write_text(entry.read_text()+'\n# New installed generation after approval.\n')
        value=command(root,env,['app-disconnect',row['request_id'],'--expected-revision',str(row['revision'])])
        closed=value['connection']
        assert closed['phase']=='closed' and closed['receiver_reconciliation']
        probe(root,env,install,kind,expected=1)  # Remotely revoked cookie cannot read.
        value=command(root,env,['app-reconcile',row['request_id'],'--expected-revision',str(closed['revision'])])
        assert not value['connection']['receiver_reconciliation'] and value['connection']['receiver_outcome']=='removed'
        if index==0:
            probe(root,env,install,'dms')
            assert command(root,env,['ship-status','lab'])['status']=='authenticated'
            command(root,env,['ship-logout','lab'])
            command(root,env,['ship-forget','lab','--url',ORIGIN])
            assert command(root,env,['ships'])==[]
            probe(root,env,install,'dms')  # Survives parent logout and enrollment removal.
    final=command(root,env,['app-connections'])
    assert len(final)==2 and all(a['phase']=='closed' and not a['receiver_reconciliation'] and a['next_actions']==['app-archive'] for a in final)
    receipts=[]
    for row in final:
        args=['app-archive',row['request_id'],'--expected-revision',str(row['revision'])]
        stale=args[:-1]+[str(row['revision']-1)]
        assert command(root,env,stale,expected=1)['code']=='selection_changed'
        archived=command(root,env,args)['connection']
        assert archived['archived'] and archived['next_actions']==[]
        assert command(root,env,args)['connection']==archived
        assert command(root,env,['app-connection',row['request_id']])==archived
        target=row['target']
        replay=['app-connect','lab','--expected-ship-id',parent['id'],'--expected-ship-revision',str(parent['revision']),
                '--plugin-id',target['plugin_id'],'--profile',target['profile'],'--artifact-sha256',target['artifact_sha256'],
                '--request-id',row['request_id'],'--approve-owner-session']
        assert command(root,env,replay,expected=1)['code']=='archived'
        receipts.append(archived)
    assert command(root,env,['app-connections'])==[]
    # Each real receiver compacted only its reconciled attempt. Read no secrets.
    journals=list(root.rglob('application-attempts.json'))
    assert len(journals)==2
    for path in journals:
        assert json.loads(path.read_text())=={'version':1,'attempts':[]}
        history=list((path.parent/'application-history').glob('*.json'))
        assert len(history)==1
        assert json.loads(history[0].read_text())['attempt']['phase']=='reconciled'
    # Keep only nonsecret lifecycle evidence; no Secret Service export.
    (root/'attempts-final.json').write_text(json.dumps(receipts,indent=2)+'\n')


def cli_gate():
    root,env,install=stage('cli-lab')
    discovered=command(root,env,['app-plugins'])
    assert len(discovered)==2 and all(p['status']=='available' and p['target']['profile']=='cli-lab' for p in discovered)
    command(root,env,['ship-enroll','lab','--url',ORIGIN,'--ship','~zod'])
    command(root,env,['ship-login','lab','--code-stdin'],private=CODE)
    ship=command(root,env,['ships'])[0]
    for plugin in discovered:
        target=plugin['target'];request=secrets.token_hex(16)
        args=['app-connect','lab','--expected-ship-id',ship['id'],'--expected-ship-revision',str(ship['revision']),
              '--plugin-id',target['plugin_id'],'--profile',target['profile'],'--artifact-sha256',target['artifact_sha256'],'--request-id',request]
        command(root,env,args,expected=2)  # Missing authority approval cannot issue.
        args+=['--approve-owner-session']
        stale=args.copy();stale[stale.index('--artifact-sha256')+1]='0'*64
        assert command(root,env,stale,expected=1)['code']=='selection_changed'
        first=command(root,env,args)['connection']
        assert first['phase']=='connected'
        duplicate=command(root,env,args)['connection']
        assert (duplicate['request_id'],duplicate['revision'])==(first['request_id'],first['revision'])
        recovered=command(root,env,['app-recover',request,'--expected-revision',str(first['revision'])])['connection']
        assert recovered['phase']=='connected' and recovered['revision']==first['revision']
    cleanup(root,env,install)
    print('PASS: real public CLI, declared receivers, private keyring, independent direct transports and retained-generation cleanup',flush=True)


def ui_gate():
    root,env,install=stage('ui-lab')
    env.update(QT_QPA_PLATFORM='offscreen',QT_QUICK_BACKEND='software',QSG_RHI_BACKEND='software',QT_QUICK_CONTROLS_STYLE='Basic',QML_DISABLE_DISK_CACHE='1',NO_COLOR='1',FONTCONFIG_FILE='/artifacts/ui/fonts.conf')
    bridge=subprocess.Popen(['/usr/bin/python3','-B','/artifacts/running_bridge.py'],env=env,stdin=subprocess.PIPE,stdout=subprocess.PIPE,stderr=subprocess.DEVNULL)
    def expect(stage):
        with selectors.DefaultSelector() as selector:
            selector.register(bridge.stdout,selectors.EVENT_READ)
            assert selector.select(30),'running bridge response timed out'
            value=json.loads(bridge.stdout.readline(4097))
            assert value=={'stage':stage,'ok':True},'running bridge failed at '+stage
    try:
        expect('empty')
        result=subprocess.run(['/usr/bin/quickshell','--no-color','-p','/artifacts/ui/shell.qml'],env=env,stdout=subprocess.PIPE,stderr=subprocess.STDOUT,timeout=85)
        log=result.stdout
        (root/'qml.log').write_bytes(log)
        assert result.returncode==0 and b'PASS: real panel, collector and both declared receivers connected' in log,'real panel did not complete'
        assert CODE not in log,'UI exposed login code'
        assert not any(token in log for token in (b'Error:',b'Unable to assign',b'is not defined',b'Cannot assign to',b'Binding loop detected')),'real panel reported binding errors'
        calls=[json.loads(line.split('CLI ',1)[1]) for line in log.decode().splitlines() if ' qml: CLI ' in line]
        mutations=[call for call in calls if call[0] not in ('app-plugins','ships','app-connections')]
        assert [call[0] for call in mutations]==['ship-enroll','ship-login','app-connect','app-connect'],'unexpected real UI mutations'
        assert len({call[call.index('--request-id')+1] for call in mutations if call[0]=='app-connect'})==2
        (root/'ui-calls.json').write_text(json.dumps(calls,indent=2)+'\n')
        bridge.stdin.write(b'{"action":"refresh"}\n');bridge.stdin.flush()
        expect('restored')
        bridge.stdin.write(b'{"action":"stop"}\n');bridge.stdin.flush()
        expect('stopped')
        assert bridge.wait(timeout=10)==0,'running bridge did not stop'
        events.append({'profile':root.name,'running_dms_bridge':'adopted-after-approval'})
        events.append({'profile':root.name,'running_theme_service':'refreshed-after-approval-without-publication'})
    finally:
        if bridge.poll() is None:
            bridge.terminate()
            try:bridge.wait(timeout=3)
            except subprocess.TimeoutExpired:
                bridge.kill();bridge.wait()
    cleanup(root,env,install)
    print('PASS: actual approval panel + real collector + both real receivers; plugins work after panel exits',flush=True)


def main():
    check_namespace()  # Must run before creating state or contacting loopback.
    started=time.monotonic()
    result={'ok':False}
    try:
        cli_gate();ui_gate();result['ok']=True
    finally:
        result.update(seconds=round(time.monotonic()-started,2),events=events)
        Path('/lab/public-flow-result.json').write_text(json.dumps(result,indent=2)+'\n')
    return 0


if __name__=='__main__':
    try:sys.exit(main())
    except (AssertionError,OSError,ValueError,subprocess.SubprocessError) as error:
        # Assertion text is fixed by this driver; other errors may have output.
        print('FAIL: '+(str(error) if isinstance(error,AssertionError) else type(error).__name__),flush=True)
        sys.exit(1)
