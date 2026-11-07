#!/usr/bin/env python3
"""Exercise Ship URLs inside a fresh, networkless full-panel preview."""
import argparse
import importlib.util
from pathlib import Path
import re
import shutil
import subprocess

ROOT = Path(__file__).resolve().parents[1]


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--output", type=Path, required=True)
    out = parser.parse_args().output.absolute()
    if out.parent != Path("/tmp") or not re.fullmatch(r"omabit-ship-routes-ui-[A-Za-z0-9-]+", out.name) or out.exists():
        parser.error("Use a new /tmp/omabit-ship-routes-ui-NAME directory")
    spec = importlib.util.spec_from_file_location("storage", ROOT / "scripts/qualification/app-development/storage.py")
    storage = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(storage)
    storage.admit(64 << 20)
    subprocess.run(["/usr/bin/python3", "-B", str(ROOT / "scripts/prepare-app-development-preview.py"), "--output", str(out)], check=True)
    shutil.copy2(ROOT / "scripts/test-ship-routes-preview.qml", out / "RouteChecks.qml")
    shutil.copy2(ROOT / "scripts/ship-routes-preview/backend.py", out / "routes-backend.py")
    service = (out / "PreviewService.qml").read_text()
    service = service.replace('"/preview/backend.py",remoteMode].concat(args)', '(args.length && args[0].indexOf("ship-routes")===0 ? "/preview/routes-backend.py" : "/preview/backend.py"),remoteMode].concat(args)')
    service = service.replace('property string remoteMode:"online"', 'property string remoteMode:"online"\n  property string routeScenario:"normal"')
    service = service.replace('property string routeScenario:"normal"', '''property string routeScenario:"normal"
  property var agentHelp:({busy:false,value:null,target:"",error:"",message:""})
  property var lastHelp:null
  property int helpCalls:0
  function requestAgentHelp(topic,context){lastHelp={topic:topic,context:JSON.parse(JSON.stringify(context))};helpCalls++;return true}''')
    service = service.replace('remoteMode].concat(args)', '(args.length && args[0].indexOf("ship-routes")===0 && routeScenario!=="normal" ? routeScenario : remoteMode)].concat(args)')
    service = service.replace('observed_state:"stopped",desired_state:"stopped"', 'observed_state:"running",desired_state:"running",resources:{container_id:"a".repeat(64)}')
    service = service.replace('features:[]', 'features:["ship-routes-v1"]')
    (out / "PreviewService.qml").write_text(service)
    shell = (out / "shell.qml").read_text().replace('ShellRoot {', 'ShellRoot {\n  RouteChecks {window:window}', 1)
    shell = shell.replace('setupPane.mode="development"', 'setupPane.mode="routes"')
    shell = shell.replace('if(Quickshell.env("OMABIT_PREVIEW_SMOKE")==="1")Qt.quit()', '')
    (out / "shell.qml").write_text(shell)
    q = subprocess.run(["/usr/bin/python3", "-B", str(out / "launch.py"), "smoke"], capture_output=True, text=True, timeout=35)
    log = (out / "data/state/window.log").read_text()
    print(log)
    assert q.returncode == 0 and "PASS: ship URL Setup" in log, q.stdout + q.stderr + log
    assert not any(word in log for word in ("WARN", "ERROR", "Error:", "FAIL:", "Unable to assign", "Binding loop")), log


if __name__ == "__main__":
    main()
