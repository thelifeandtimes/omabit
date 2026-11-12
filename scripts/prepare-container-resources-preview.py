#!/usr/bin/env python3
"""Prepare a fresh full-panel resource preview; never overwrite human data."""
import argparse,hashlib,importlib.util,json,re,shutil,subprocess
from pathlib import Path
ROOT=Path(__file__).resolve().parents[1]
def main():
 p=argparse.ArgumentParser(description=__doc__);p.add_argument('--output',type=Path,required=True);out=p.parse_args().output.absolute()
 spec=importlib.util.spec_from_file_location('storage',ROOT/'scripts/qualification/app-development/storage.py');storage=importlib.util.module_from_spec(spec);spec.loader.exec_module(storage);storage.admit(32<<20)
 subprocess.run(['/usr/bin/python3','-B',str(ROOT/'scripts/prepare-app-development-preview.py'),'--output',str(out)],check=True)
 shutil.copy2(ROOT/'scripts/container-resources-preview/backend.py',out/'resources-backend.py')
 s=(out/'PreviewService.qml').read_text().replace('import QtQuick','import QtQuick\nimport Quickshell.Io\nimport "Plugin/Model.js" as Model',1)
 s,count=re.subn(r'  function inventory\(\) \{.*?\n  \}', '  function inventory(){return resourceInventory()}',s,count=1,flags=re.S);assert count==1
 for before,after in [
  ('function canConfigure(){return false}','function canConfigure(target){return resourceCanConfigure(target)}'),
  ('function configure(){return unavailable()}','function configure(target,port,loom,memory,cpus){return resourceConfigure(target,port,loom,memory,cpus)}'),
  ('function operate(){return unavailable()}','function operate(connection,instance,kind){return resourceOperate(connection,instance,kind)}'),
  ('supports application-policy editing only.','supports simulated permissions and resource settings only.')]:
  assert before in s;s=s.replace(before,after)
 s=s.rsplit('}',1)[0]+(ROOT/'scripts/container-resources-preview/service.qml.txt').read_text()+'}\n';(out/'PreviewService.qml').write_text(s)
 shell=(out/'shell.qml').read_text().replace('ShellRoot {','''ShellRoot {
  id:preview
  function find(o,name){if(o.objectName===name)return o;var children=o.children||[];for(var i=0;i<children.length;i++){var v=find(children[i],name);if(v)return v}return null}
  function openResources(){window.page="overview";Qt.callLater(function(){var edit=find(window.contentItem,"edit-ship-ports");if(edit && edit.enabled)edit.clicked();var card=find(window.contentItem,"connectivity-runtime-card");if(card)window.revealItem(card)})}
''',1).replace('title:"Omabit 0.4.0 · Setup Preview"','title:"Omabit 0.4.0 · Isolated Preview"').replace('page:"setup"','page:"overview"')
 shell=shell.replace('interval:1000;running:!!window.setupPane.appDevelopmentPane.policyView;repeat:false','interval:1000;running:service.resourceReady;repeat:false\n    onTriggered:{preview.openResources();capture.start()}\n  }\n  Timer {\n    id:capture;interval:350;repeat:false')
 (out/'shell.qml').write_text(shell)
 readme=(out/'README.md').read_text();readme='''# RAM and CPU manual review

Open **Omabit 0.4.0 Preview** from the app launcher. The full panel starts with
`preview-local → ~zod` and its runtime editor visible. Open runtime configuration on the ship's Overview page. Choose RAM/CPU, Save, then use the
normal Start button at the top of the Overview page. Stop and reload to edit
again. The remote `~bus` fixture starts running, so its resource controls remain
disabled until Stop. Its Offline/observer switch also disables editing.

All Start/Stop/configuration actions are simulated. No collector, Docker, Urbit,
runner service or production plugin is connected. Choices persist separately in
`data/state/resources.json`; existing permission choices stay in
`data/state/policy.json`. Project member `~nec` retains its fixed contract.
Setup and the sidebar remain available. The earlier Setup review guide follows.

'''+readme
 readme=readme.replace('with the fleet sidebar and **Setup → App\ndevelopment and publishing** selected.','with the fleet sidebar; **Setup → App\ndevelopment and publishing** remains available.')
 readme=readme.replace('with host/ship mutations disabled by the inert service adapter.','with only simulated Start/Stop/resource edits enabled by the inert adapter.')
 (out/'README.md').write_text(readme)
 source=json.loads((out/'source.json').read_text());source['preview']='full-panel resource simulation; no real backend';source['files']={str(p.relative_to(out)):hashlib.sha256(p.read_bytes()).hexdigest() for p in sorted(out.rglob('*')) if p.is_file() and p.name!='source.json' and 'data' not in p.relative_to(out).parts};(out/'source.json').write_text(json.dumps(source,indent=2)+'\n')
 print(out)
if __name__=='__main__':main()
