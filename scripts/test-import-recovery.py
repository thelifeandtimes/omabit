#!/usr/bin/env python3
"""Mapped recovery review, immutable selection and one-shot dispatch fixture."""
import datetime
import argparse
import importlib.util
import json
import os
from pathlib import Path
import subprocess
import tempfile
from desktop_fixture import ROOT, NATIVE, HARNESS_ROOT, run_qs

spec = importlib.util.spec_from_file_location("panel_fixture", HARNESS_ROOT / "scripts/test-panel.py")
fixture = importlib.util.module_from_spec(spec)
spec.loader.exec_module(fixture)

parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument("--static", action="store_true", help="verify the generated CLI scenario without QML or a compositor")
parser.add_argument("--native", action="store_true", help="exercise the canonical release entry-point wrapper")
args = parser.parse_args()

with tempfile.TemporaryDirectory(prefix="omabit-recovery-") as temporary:
    directory = Path(temporary)
    helper = fixture.write_fixture(directory, "collector", extra_features=["recover-import"], import_recovery=True)
    observation = fixture.assert_fixture_capabilities(helper, {"recover-import", "pier-storage", "storage-locations", "runtime-settings"})
    ship = observation["snapshot"]["instances"][0]
    assert ship["runtime_status"]["import_recovery_available"] and ship["external_import"]["stage"] == "enrolling"
    if args.static:
        command = [str(helper), "--runner", "fixture", "--expected-runner-id", "runner-fixture", "operate", "alpha", "recover-import", "--revision", "2", "--request-id"]
        env = dict(os.environ, OMABIT_FIXTURE_TIME=datetime.datetime.now(datetime.timezone.utc).isoformat())
        refused = subprocess.run(command + ["refused"], env=env, capture_output=True, text=True, timeout=5)
        assert refused.returncode == 1 and json.loads(refused.stderr)["error"]["code"] == "revision_conflict"
        accepted = json.loads(subprocess.check_output(command + ["retry"], env=env, text=True, timeout=5))
        assert accepted["request_id"] == "retry" and accepted["status"] == "queued"
        completed = fixture.assert_fixture_capabilities(helper, {"recover-import"})
        assert completed["snapshot"]["operations"][-1]["status"] == "succeeded"
        assert completed["snapshot"]["instances"][0]["observed_state"] == "stopped"
        # Reproduce the original missing-injection case as a negative control.
        baseline_dir = directory / "baseline"
        baseline_dir.mkdir()
        baseline = fixture.write_fixture(baseline_dir, import_recovery=True)
        try:
            fixture.assert_fixture_capabilities(baseline, {"recover-import"})
        except AssertionError:
            pass
        else:
            raise AssertionError("missing fixture capability was silently accepted")
        print("PASS: explicit recovery capabilities, pre-mapping prerequisites, refusal/retry and stopped receipt fixture")
        raise SystemExit(0)
    for name in ("Commons", "Ui", "services"):
        (directory / name).symlink_to(Path("/usr/share/omarchy/shell") / name)
    (directory / "Plugin").symlink_to(ROOT / "plugins/omabit")
    qml = '''import QtQuick
import Quickshell
import "Plugin" as Plugin
ShellRoot {
  id: test
  property int phase:0
  property int settle:0
  property double deadline:Date.now()+10000
  Plugin.Service { id:service;settings:({collectorPath:HELPER,omabitPath:HELPER,notifications:false}) }
  function control(object,name,visited) {
    visited=visited || [];if(!object || visited.indexOf(object)>=0)return null;visited.push(object)
    if(object.objectName===name)return object
    for(var key of ["children","data","contentItem"]) {
      var children=object[key];if(!children)continue;if(typeof children.length!=="number")children=[children]
      for(var i=0;i<children.length;i++){var found=control(children[i],name,visited);if(found)return found}
    }
    return null
  }
  Timer { interval:100;repeat:true;running:true;onTriggered:{
    if(Date.now()>deadline){console.error("Error: recovery phase "+phase+" "+service.feedError+" "+service.lastError+" "+JSON.stringify(service.connections));Qt.quit();return}
    if(phase===0 && service.observing && service.connections.length){service.openSetup("storage");phase=1}
    else if(phase===1){
      var pane=control(service.monitorWindow,"import-recovery-alpha")
      if(!pane || !pane.visible || !pane.available)throw new Error("Eligible import did not expose recovery")
      control(pane,"review-import-recovery").clicked()
      phase=11
    } else if(phase===11){
      if(++settle<6)return
      settle=0
      var pane=control(service.monitorWindow,"import-recovery-alpha")
      var target=pane.target
      if(!target)throw new Error("Inventory refresh erased the review")
      if(service.recoverImport(Object.assign({},target,{revision:99})))throw new Error("Stale review dispatched")
      if(service.recoverImport(Object.assign({},target,{holding:"/replacement"})))throw new Error("Changed evidence dispatched")
      var submit=control(pane,"submit-import-recovery")
      if(!submit.enabled)throw new Error("Current review is disabled")
      submit.clicked();submit.clicked();phase=2
    } else if(phase===2 && !service.busy){
      var pane=control(service.monitorWindow,"import-recovery-alpha")
      if(pane.receipt || !pane.outcome || pane.outcome.status!=="failed")throw new Error("Known rejection was not distinguished")
      if(++settle<6)return
      settle=0
      var retry=control(pane,"review-import-recovery-again")
      if(!retry.visible || !retry.enabled)throw new Error("Known refusal cannot be reviewed again")
      retry.clicked();control(pane,"review-import-recovery").clicked();control(pane,"submit-import-recovery").clicked();phase=3
    } else if(phase===3 && !service.busy){
      var pane=control(service.monitorWindow,"import-recovery-alpha")
      if(!pane.receipt)return
      if(pane.receipt.status!=="succeeded" || pane.available || control(pane,"submit-import-recovery").enabled || control(pane,"import-recovery-receipt").text.indexOf("ship is stopped")<0)throw new Error("Recovery receipt or stop boundary missing")
      control(pane,"submit-import-recovery").clicked()
      if(++settle<5)return
      console.log("PASS: recovery review rejects stale/evidence-changed targets, dispatches once, and reports the stopped receipt")
      service.monitorWindow.visible=false;Qt.quit()
    }
  } }
}'''.replace("HELPER", json.dumps(str(helper)))
    if args.native:
        qml = qml.replace('import "Plugin" as Plugin', 'import "Plugin" as Plugin\nimport ' + json.dumps(NATIVE.as_uri()) + ' as Native').replace("Plugin.Service {", "Native.Service {")
    (directory / "shell.qml").write_text(qml)
    result = run_qs(directory, timeout=20)
    text = result.stdout + result.stderr
    assert result.returncode == 0 and text.count("PASS:") == 1 and not any(e in text for e in ("Error:", "Unable to assign", "Binding loop")), text
    calls = [json.loads(line) for line in (directory / "calls.jsonl").read_text().splitlines()]
    mutations = [c for c in calls if "operate" in c]
    assert len(mutations) == 2 and all("recover-import" in c for c in mutations), mutations
    assert mutations[0][mutations[0].index("--request-id")+1] != mutations[1][mutations[1].index("--request-id")+1], mutations
    for line in text.splitlines():
        if "PASS:" in line:
            print(line)
    print("PASS: CLI audit confirms one refused request, one explicit reviewed retry, and no runtime Start or automatic retry")
