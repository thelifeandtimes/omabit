#!/usr/bin/env python3
"""Actual remote wizard/service controls with inert CLI responses and private XDG."""
import importlib.util
import json
from pathlib import Path
import tempfile
from desktop_fixture import ROOT, run_qs

spec = importlib.util.spec_from_file_location("panel_fixture", Path(__file__).with_name("test-panel.py"))
fixture = importlib.util.module_from_spec(spec)
spec.loader.exec_module(fixture)
with tempfile.TemporaryDirectory(prefix="omabit-remote-ui-") as temporary:
    directory = Path(temporary)
    for name in ("Commons", "Ui", "services"):
        (directory / name).symlink_to(Path("/usr/share/omarchy/shell") / name)
    (directory / "Plugin").symlink_to(ROOT / "plugins/omabit")
    helper = fixture.write_fixture(directory)
    qml = '''import QtQuick
import Quickshell
import "Plugin" as Plugin
ShellRoot {
  id: test
  property int phase:0
  property var settings:({collectorPath:HELPER,omabitPath:HELPER,notifications:false})
  Plugin.Service {id:service;settings:test.settings}
  function control(object,name,visited) {
    visited=visited || [];if(!object||visited.indexOf(object)>=0)return null;visited.push(object)
    if(object.objectName===name)return object
    for(var key of ["children","data","contentItem"]) {
      var children=object[key];if(!children)continue;if(typeof children.length!=="number")children=[children]
      for(var i=0;i<children.length;i++){var found=control(children[i],name,visited);if(found)return found}
    }
    return null
  }
  Timer {interval:100;repeat:true;running:true;onTriggered:{
    if(phase===0&&service.registrationsLoaded){service.openSetup("runner");phase=1}
    else if(phase===1&&service.remoteSetup&&service.remoteSetup.phase==="idle") {
      var pane=control(service.monitorWindow,"remote-runner-setup")
      if(!pane.visible||control(pane,"remote-setup-continue").enabled)throw new Error("remote setup was not initially gated on a host")
      var host=control(pane,"remote-setup-host")
      host.text="custom.example"
      host.acceptSuggestions({hosts:[{host:"sourcebox",label:"sourcebox:2200",port:2200,source:"Known host"}]})
      if(!host.editable||host.text!=="custom.example")throw new Error("host discovery overwrote custom input")
      host.currentIndex=0;host.activated(0)
      if(host.text!=="sourcebox"||control(pane,"remote-setup-port").text!=="2200")throw new Error("host dropdown lost hostname/port selection")
      if(!control(pane,"remote-setup-continue").enabled)throw new Error("normal SSH alias cannot continue")
      control(pane,"remote-setup-continue").clicked();control(pane,"remote-setup-continue").clicked();phase=2
    } else if(phase===2&&service.remoteSetupBusy) {
      var pane=control(service.monitorWindow,"remote-runner-setup")
      if(service.reloadSafe||control(pane,"remote-setup-continue").enabled||control(pane,"remote-setup-host").enabled)throw new Error("pending setup allowed duplicate dispatch/target editing or unsafe reload")
      phase=3
    } else if(phase===3&&service.remoteSetup&&service.remoteSetup.phase==="connected"&&!service.busy) {
      var pane=control(service.monitorWindow,"remote-runner-setup")
      if(control(pane,"remote-setup-progress").text.indexOf("Found 2 ships")<0||!control(pane,"remote-setup-view").visible)throw new Error("verified remote completion was not shown")
      control(pane,"remote-setup-revoke").clicked();phase=4
    } else if(phase===4&&service.remoteSetup&&service.remoteSetup.phase==="revoked"&&!service.busy) {
      if(service.remoteSetupError)throw new Error(service.remoteSetupError)
      console.log("PASS: real remote wizard accepts an ordinary SSH alias, blocks duplicate setup, polls progress and shows verified completion/revocation")
      service.monitorWindow.visible=false;Qt.quit()
    }
  }}
}'''.replace("HELPER", json.dumps(str(helper)))
    (directory / "shell.qml").write_text(qml)
    result = run_qs(directory, timeout=25)
    text = result.stdout + result.stderr
    assert result.returncode == 0 and "PASS: real remote wizard" in text, text
    assert not any(e in text for e in ("Error:", "Unable to assign", "Binding loop", "is not defined")), text
    calls = [json.loads(line) for line in (directory / "calls.jsonl").read_text().splitlines()]
    starts = [c for c in calls if "remote-setup-start" in c]
    assert len(starts) == 2 and "--host" in starts[0] and "sourcebox" in starts[0], starts
    assert "--port" in starts[0] and "2200" in starts[0], starts
    assert "--id" in starts[1] and "--revoke" in starts[1], starts
    assert not any("operate" in c or "setup" in c for c in calls), calls
    for line in text.splitlines():
        if "PASS:" in line:
            print(line)
    print("PASS: CLI audit confirms one setup and one bound revocation, with no ship operations")
