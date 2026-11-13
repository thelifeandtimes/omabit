#!/usr/bin/env python3
"""Full-panel Agent Help with actual helpers, private skills and an inert agent."""
import argparse
import importlib.util
import json
from pathlib import Path
import re
import shutil
import subprocess

ROOT=Path(__file__).resolve().parents[1]
def main():
    parser=argparse.ArgumentParser();parser.add_argument('--output',type=Path,required=True)
    parser.add_argument('--persistent-agent',action='store_true',help='Check interactive sessions past both helper deadlines and UI suspension')
    args=parser.parse_args();out=args.output.absolute()
    if out.parent!=Path('/tmp') or not re.fullmatch('omabit-agent-ui-[a-zA-Z0-9-]+',out.name) or out.exists():
        parser.error('Use a fresh /tmp/omabit-agent-ui-NAME fixture')
    subprocess.run(['/usr/bin/python3','-B',str(ROOT/'scripts/prepare-app-development-preview.py'),'--output',str(out)],check=True)
    for name in ('agent-help.py','manifest.json'):
        shutil.copy2(ROOT/'plugins/omabit'/name,out/'Plugin'/name)
    shutil.copytree(ROOT/'skills',out/'skills')
    spec=importlib.util.spec_from_file_location('packager',ROOT/'scripts/package-release.py')
    packager=importlib.util.module_from_spec(spec);spec.loader.exec_module(packager)
    metadata={'version':json.loads((out/'Plugin/manifest.json').read_text())['version'],'source_commit':'a'*40,'development':{'label':'agent-ui-fixture'}}
    (out/'BUILD.json').write_text(json.dumps(metadata));packager.write_skill_manifest(out,metadata)
    (out/'bin').mkdir()
    (out/'bin/omabit').write_text('#!/bin/sh\nexit 77\n')
    (out/'bin/omarchy').write_text('''#!/usr/bin/python3
import json,os,sys
from pathlib import Path
if sys.argv[1:]==['default','agent']:
    if os.environ.get('OMABIT_AGENT_FIXTURE_MODE')!='offline':print('opencode')
elif sys.argv[1:3]==['agent','prompt'] and len(sys.argv)==4:
    with Path('/data/state/agent-prompts.jsonl').open('a') as log:log.write(json.dumps(sys.argv[1:])+'\\n')
else:raise SystemExit(78)
''');(out/'bin/omarchy').chmod(0o755)
    if args.persistent_agent:
        fake=out/'bin/omarchy';text=fake.read_text().replace('import json,os,sys','import json,os,sys,time')
        text=text.replace('else:raise SystemExit(78)','''    start=time.monotonic()
    while time.monotonic()-start < 45:
        state={"pid":os.getpid(),"session":os.getsid(0),"age":time.monotonic()-start,
               "stdio":[os.readlink("/proc/self/fd/"+str(n)) for n in range(3)]}
        path=Path("/data/state/agent-session-"+str(os.getpid())+".json")
        temp=path.with_suffix(".tmp");temp.write_text(json.dumps(state));temp.replace(path)
        time.sleep(0.1)
else:raise SystemExit(78)''')
        fake.write_text(text)
    launcher=out/'launch.py';launcher.write_text(launcher.read_text().replace('"PATH": "/usr/bin:/bin"','"PATH": "/preview/bin:/usr/bin:/bin"'))
    service=out/'PreviewService.qml';source=service.read_text().replace("function checkRemoteSetup(){return unavailable()}","function checkRemoteSetup(){return true}").replace('import QtQuick','import QtQuick\nimport "Plugin" as Plugin',1)
    source=source.replace('  id:root','''  id:root
  readonly property alias agentHelp:agentHelper
  Plugin.AgentHelp {id:agentHelper;service:root;commandPrefix:["/usr/bin/env","OMABIT_AGENT_FIXTURE_MODE="+root.remoteMode,"/usr/bin/python3","-B","/preview/Plugin/agent-help.py"]}
  function requestAgentHelp(topic,setup){var context={topic:topic,profile:"preview",environment:{XDG_CONFIG_HOME:"/data/config"}};if(setup){context.setup_id=setup.id;context.phase=setup.phase}return agentHelper.launch(context)}
''',1);service.write_text(source)
    shell=out/'shell.qml';shell.write_text(shell.read_text().replace('ShellRoot {','ShellRoot {\n AgentChecks { window:window }',1).replace('setupPane.mode="development"','setupPane.mode="agent"').replace('if(Quickshell.env("OMABIT_PREVIEW_SMOKE")==="1")Qt.quit()',''))
    shutil.copy2(ROOT/'scripts/test-agent-help-ui.qml',out/'AgentChecks.qml')
    if args.persistent_agent:
        shell.write_text(shell.read_text().replace('import QtQuick','import QtQuick\nimport QtQuick.Window',1).replace('AgentChecks { window:window }','Window {visible:true;width:1;height:1}\n AgentChecks { window:window;persistentAgent:true }').replace('onClosing:Qt.quit()','').replace('onVisibleChanged:if(!visible)Qt.quit()',''))
        launcher.write_text(launcher.read_text().replace('timeout=25 if smoke else None','timeout=40 if smoke else None'))
    result=subprocess.run(['/usr/bin/python3','-B',str(launcher),'smoke'],capture_output=True,text=True,timeout=45 if args.persistent_agent else 30)
    log=(out/'data/state/window.log').read_text()
    assert result.returncode==0 and 'PASS: agent help' in log,result.stdout+result.stderr+log
    assert not any(s in log for s in ('WARN','ERROR','Error:','FAIL:','Binding loop')),log
    prompts=[json.loads(line) for line in (out/'data/state/agent-prompts.jsonl').read_text().splitlines()]
    assert len(prompts)==2,prompts
    assert 'remote-setup-status --id '+'1'*32 in prompts[1][2],prompts
    assert all(p[:2]==['agent','prompt'] and len(p)==3 and '/preview/skills/omabit-operator/SKILL.md' in p[2] for p in prompts)
    for name in ('omabit-operator','omabit-plugin-developer'):
        assert (out/'data/config/opencode/skills'/name/'SKILL.md').read_bytes()==(ROOT/'skills'/name/'SKILL.md').read_bytes()
    if args.persistent_agent:
        sessions=[json.loads(path.read_text()) for path in sorted((out/'data/state').glob('agent-session-*.json'))]
        assert len(sessions)==2,sessions
        assert all(s['age']>=21 and s['pid']==s['session'] and s['stdio']==['/dev/null']*3 for s in sessions),sessions
        print('PASS: both detached agent sessions stayed alive beyond21s with private stdio, after helper exit and panel suspension')
    print(log)
if __name__=='__main__':main()
