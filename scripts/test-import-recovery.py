#!/usr/bin/env python3
"""Mapped recovery review, immutable selection and one-shot dispatch fixture."""
import datetime
import importlib.util
import json
import os
from pathlib import Path
import subprocess
import tempfile

ROOT = Path(__file__).resolve().parents[1]
spec = importlib.util.spec_from_file_location("panel_fixture", ROOT / "scripts/test-panel.py")
fixture = importlib.util.module_from_spec(spec)
spec.loader.exec_module(fixture)

with tempfile.TemporaryDirectory(prefix="omabit-recovery-") as temporary:
    directory = Path(temporary)
    for name in ("Commons", "Ui", "services"):
        (directory / name).symlink_to(Path("/usr/share/omarchy/shell") / name)
    (directory / "Plugin").symlink_to(ROOT / "plugins/omabit")
    helper = directory / "collector"
    code = fixture.FAKE.replace('features += ["host-port-suggestion",', 'features += ["recover-import","host-port-suggestion",')
    code = code.replace('if "capture" in sys.argv:', '''if "operate" in sys.argv and sys.argv[sys.argv.index("operate")+2]=="recover-import":
    assert sys.argv[sys.argv.index("--runner")+1]=="fixture"
    assert sys.argv[sys.argv.index("--expected-runner-id")+1]=="runner-fixture"
    assert sys.argv[sys.argv.index("operate")+1]=="alpha"
    assert sys.argv[sys.argv.index("--revision")+1]=="2"
    if not (directory/"refused-once").exists():
        (directory/"refused-once").touch()
        print(json.dumps(dict(error=dict(code="revision_conflict",message="Review the current selection."))),file=sys.stderr)
        sys.exit(1)
    op=dict(id="op-recovered",kind="recover-import",instance_id="alpha",request_id=sys.argv[sys.argv.index("--request-id")+1],config_revision=2,status="succeeded")
    (directory/"recovered.json").write_text(json.dumps(op))
    time.sleep(.4)
    print(json.dumps(dict(op,status="queued")))
elif "capture" in sys.argv:''')
    code = code.replace('snapshot["instances"][0]["runtime_status"]={"initialized":True}', '''recovered=(directory/"recovered.json").exists()
        ship=snapshot["instances"][0]
        ship.update(desired_state="stopped",observed_state="stopped" if recovered else "unknown",recovery_required=not recovered,identity_pending=True)
        ship["runtime_status"]={"initialized":recovered,"import_recovery_available":not recovered}
        ship["external_import"]={"stage":"complete" if recovered else "enrolling","source":"/fixture/old-source","holding_path":"/fixture/.omabit-import-old/pier"}
        if recovered: snapshot["operations"].append(json.loads((directory/"recovered.json").read_text()))''')
    compile(code,str(helper),"exec")
    helper.write_text(code)
    helper.chmod(0o700)
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
    (directory / "shell.qml").write_text(qml)
    runtime = directory / "runtime"
    runtime.mkdir(mode=0o700)
    (runtime / "hypr").mkdir()
    (runtime / "hypr/f").symlink_to(Path(os.environ["XDG_RUNTIME_DIR"]) / "hypr" / os.environ["HYPRLAND_INSTANCE_SIGNATURE"])
    display = os.environ["WAYLAND_DISPLAY"]
    if not display.startswith("/"):
        display = str(Path(os.environ["XDG_RUNTIME_DIR"]) / display)
    env = dict(os.environ, WAYLAND_DISPLAY=display, XDG_RUNTIME_DIR=str(runtime), HYPRLAND_INSTANCE_SIGNATURE="f", XDG_STATE_HOME=str(directory / "state"), OMABIT_FIXTURE_TIME=datetime.datetime.now(datetime.timezone.utc).isoformat())
    try:
        result = subprocess.run(["qs", "--no-color", "-p", str(directory)], env=env, capture_output=True, text=True, timeout=20)
    except subprocess.TimeoutExpired as e:
        raise AssertionError((e.stdout or b"").decode() + (e.stderr or b"").decode()) from e
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
