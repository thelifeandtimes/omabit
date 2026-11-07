#!/usr/bin/python3
"""Fleet permission/connection interactions inside the normal full panel."""
import hashlib
import importlib.util
import json
from pathlib import Path
import re
import shutil
import subprocess
import sys

root = Path(__file__).resolve().parents[1]
out = Path(sys.argv[1]).absolute()
assert out.parent == Path('/tmp') and re.fullmatch('omabit-fleet-access-ui-[a-z0-9-]+', out.name) and not out.exists()
spec = importlib.util.spec_from_file_location('storage', root / 'scripts/qualification/app-development/storage.py')
storage = importlib.util.module_from_spec(spec)
spec.loader.exec_module(storage)
storage.admit(64 << 20)
subprocess.run(['/usr/bin/python3', '-B', str(root / 'scripts/prepare-app-development-preview.py'), '--output', str(out)], check=True)
shutil.copy2(root / 'scripts/fleet-access-preview/backend.py', out / 'access-backend.py')
shutil.copy2(root / 'scripts/ship-routes-preview/backend.py', out / 'routes-backend.py')
shutil.copy2(root / 'scripts/test-fleet-access-preview.qml', out / 'AccessChecks.qml')
service = (out / 'PreviewService.qml').read_text().replace('import QtQuick', 'import QtQuick\nimport "Plugin" as Plugin', 1)
service = service.replace('property string remoteMode:"online"', 'property string remoteMode:"online"\n  property string applicationScenario:"normal"')
service = service.replace('observed_state:"stopped",desired_state:"stopped"', 'observed_state:"running",desired_state:"running",resources:{container_id:"a".repeat(64)}')
service = service.replace('features:[]', 'features:["ship-routes-v1","ship-route-code-v1"]')
service = service.replace('function command(args){return ["/usr/bin/python3","-B","/preview/backend.py",remoteMode,"preview-unavailable"]}', 'function command(args){return ["/usr/bin/python3","-B","/preview/access-backend.py",args.length ? remoteMode : "online",args.length ? applicationScenario : "normal"].concat(args)}')
service = re.sub(r'  property QtObject applicationConnections:QtObject \{.*?\n  \}\n', '  property alias applicationConnections:access\n  Plugin.ApplicationConnections {id:access;service:root}\n', service, flags=re.S)
(out / 'PreviewService.qml').write_text(service)
shell = (out / 'shell.qml').read_text().replace('ShellRoot {', 'ShellRoot {\n  AccessChecks {window:window}', 1)
shell = shell.replace('setupPane.mode="development"', 'setupPane.mode="apps"').replace('if(Quickshell.env("OMABIT_PREVIEW_SMOKE")==="1")Qt.quit()', '')
(out / 'shell.qml').write_text(shell)
launcher = (out / 'launch.py').read_text().replace('timeout=25', 'timeout=55')
(out / 'launch.py').write_text(launcher)
# The preparation manifest is supplemented after fixture adaptation, before launch.
(out / 'fixture-source.json').write_text(json.dumps({str(p.relative_to(out)): hashlib.sha256(p.read_bytes()).hexdigest() for p in sorted(out.rglob('*')) if p.is_file() and 'data' not in p.relative_to(out).parts}, indent=2) + '\n')
p = subprocess.run(['/usr/bin/python3', '-B', str(out / 'launch.py'), 'smoke'], capture_output=True, text=True, timeout=60)
log = (out / 'data/state/window.log').read_text()
print(log)
assert p.returncode == 0 and 'PASS: standing fleet permission' in log, p.stdout + p.stderr + log
assert not any(word in log for word in ('WARN', 'ERROR', 'Error:', 'FAIL:', 'Unable to assign', 'Binding loop')), log
calls = [json.loads(s) for s in (out / 'data/state/access-calls.jsonl').read_text().splitlines()]
connects = [c for c in calls if 'app-fleet-connect' in c]
assert len(connects) == 2, connects
maintains = [c for c in calls if 'app-fleet-maintain' in c]
assert len(maintains) == 3, maintains
assert not any('app-managed-connect' in c or '--url' in c for c in calls)
for c in calls:
    if 'app-ship-approve' in c:
        q = json.loads(c[c.index('--request-json') + 1])
        assert q['allowed'] == ('--allow-owner-session' in c)
        assert q['allowed'] == ('--runner' in c)
print('PASS: full-panel standing permission, independent disconnect, offline revoke, conflict and lost reply; no automatic replay')
