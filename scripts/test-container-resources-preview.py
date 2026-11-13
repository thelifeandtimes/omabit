#!/usr/bin/env python3
"""Exercise a disposable full-panel resource preview and persistence, never a human preview."""
import argparse,hashlib,importlib.util,json,re,shutil,subprocess
from pathlib import Path
ROOT=Path(__file__).resolve().parents[1]
p=argparse.ArgumentParser(description=__doc__);p.add_argument('--output',type=Path,required=True);out=p.parse_args().output.absolute()
if out.parent!=Path('/tmp') or not re.fullmatch(r'omabit-resources-preview-test-[A-Za-z0-9-]+',out.name) or out.exists():p.error('Use a fresh /tmp/omabit-resources-preview-test-NAME')
subprocess.run(['/usr/bin/python3','-B',str(ROOT/'scripts/prepare-container-resources-preview.py'),'--output',str(out)],check=True)
spec=importlib.util.spec_from_file_location('backend',ROOT/'scripts/app-development-preview/backend.py');backend=importlib.util.module_from_spec(spec);spec.loader.exec_module(backend)
policy=out/'data/state/policy.json';policy.write_text(json.dumps(backend.initial_state(),indent=2)+'\n');before=hashlib.sha256(policy.read_bytes()).hexdigest()
shutil.copy2(ROOT/'scripts/test-container-resources-preview.qml',out/'ResourceChecks.qml')
shell=(out/'shell.qml').read_text().replace('ShellRoot {','ShellRoot {\n ResourceChecks {window:window}',1).replace('if(Quickshell.env("OMABIT_PREVIEW_SMOKE")==="1")Qt.quit()','')
for reloading in (False,True):
 (out/'shell.qml').write_text(shell.replace('ResourceChecks {window:window}','ResourceChecks {window:window;reloadOnly:true}') if reloading else shell)
 v=subprocess.run(['/usr/bin/python3','-B',str(out/'launch.py'),'smoke'],capture_output=True,text=True,timeout=30)
 log=(out/'data/state/window.log').read_text();print(log)
 marker='PASS: resource preview persisted' if reloading else 'PASS: full-panel resource save'
 assert v.returncode==0 and marker in log,v.stdout+v.stderr+log
 assert not any(word in log for word in ('WARN','ERROR','Error:','FAIL:','Unable to assign','Binding loop')),log
 assert hashlib.sha256(policy.read_bytes()).hexdigest()==before,'resource review changed existing permission bytes'
print('PASS: original permission file byte-preserved')
