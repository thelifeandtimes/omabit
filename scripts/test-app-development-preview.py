#!/usr/bin/env python3
"""Exercise the full native Setup tree with a fresh, networkless fixture."""
import argparse
import re
import shutil
import subprocess
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]

def main():
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--output',type=Path,required=True)
    out=parser.parse_args().output.absolute()
    if out.parent!=Path('/tmp') or not re.fullmatch(r'omabit-policy-tree-test-[A-Za-z0-9-]+',out.name) or out.exists():
        parser.error('Use a fresh /tmp/omabit-policy-tree-test-NAME; never a human preview')
    subprocess.run(['/usr/bin/python3','-B',str(ROOT/'scripts/prepare-app-development-preview.py'),'--output',str(out)],check=True)
    shutil.copy2(ROOT/'scripts/test-app-development-preview.qml',out/'PreviewChecks.qml')
    shell=(out/'shell.qml').read_text().replace('ShellRoot {','ShellRoot {\n  PreviewChecks { window:window }',1)
    shell=shell.replace('if(Quickshell.env("OMABIT_PREVIEW_SMOKE")==="1")Qt.quit()','')
    (out/'shell.qml').write_text(shell)
    q=subprocess.run(['/usr/bin/python3','-B',str(out/'launch.py'),'smoke'],capture_output=True,text=True,timeout=30)
    log=(out/'data/state/window.log').read_text()
    assert q.returncode==0 and 'PASS: full Setup navigation' in log,q.stdout+q.stderr+log
    assert not any(word in log for word in ('WARN','ERROR','Error:','FAIL:','Unable to assign','Binding loop')),log
    print(log)

if __name__=='__main__':main()
