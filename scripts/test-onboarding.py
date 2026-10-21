#!/usr/bin/env python3
"""Fresh empty collector and guided provisioning regression on mapped QML."""
import datetime
import importlib.util
import json
import os
from pathlib import Path
import subprocess
import tempfile
from desktop_fixture import ROOT, HARNESS_ROOT, run_qs

spec = importlib.util.spec_from_file_location("panel_fixture", HARNESS_ROOT / "scripts/test-panel.py")
fixture = importlib.util.module_from_spec(spec)
spec.loader.exec_module(fixture)

with tempfile.TemporaryDirectory(prefix="omabit-onboarding-") as temporary:
    directory = Path(temporary)
    for name in ("Commons", "Ui", "services"):
        (directory / name).symlink_to(Path("/usr/share/omarchy/shell") / name)
    (directory / "Plugin").symlink_to(ROOT / "plugins/omabit")
    helper = directory / "collector-fixture"
    assert fixture.FAKE.count('elif "capture" in sys.argv:') == 1
    code = fixture.FAKE.replace('elif "capture" in sys.argv:', '''elif "fixture-enroll" in sys.argv:
    (directory/"enrolled").touch()
elif "runners" in sys.argv and not (directory/"enrolled").exists():
    print("[]")
elif "create-comet" in sys.argv and sys.argv[sys.argv.index("create-comet")+1]=="ambiguous-comet":
    print("unreadable registration acknowledgement")
elif "create-comet" in sys.argv and sys.argv[sys.argv.index("create-comet")+1]=="uncertain-start":
    print(json.dumps({"id":"op-confirmed-registration","instance_id":"uncertain-start","kind":"register-comet","request_id":sys.argv[sys.argv.index("--request-id")+1],"config_revision":1,"status":"succeeded"}))
elif "operate" in sys.argv and sys.argv[sys.argv.index("operate")+1]=="uncertain-start":
    print("unreadable start acknowledgement")
elif "capture" in sys.argv:''')
    helper.write_text(code)
    helper.chmod(0o700)
    qml = '''import QtQuick
import Quickshell
import Quickshell.Io
import "Plugin" as Plugin
ShellRoot {
  id: test
  property int phase: 0
  property int settle: 0
  property var logGeometry: null
  property real portFormHeight: 0
  Plugin.Service { id: service; settings: ({collectorPath:HELPER,omabitPath:HELPER,notifications:false}) }
  Plugin.Panel { id: panel; service: service; settings: service.settings; manageIpc:false; bar:null }
  function control(object,name,visited) {
    visited=visited || []
    if (!object || visited.indexOf(object)>=0) return null
    visited.push(object)
    if (object.objectName===name) return object
    for (var key of ["children","data","contentItem"]) {
      var children=object[key];if(!children)continue
      if(typeof children.length!=="number")children=[children]
      for(var i=0;i<children.length;i++){var found=control(children[i],name,visited);if(found)return found}
    }
    return null
  }
  Process { id: enroll; command:[HELPER,"fixture-enroll"]; onExited:service.refresh() }
  Timer { interval:100;repeat:true;running:true;onTriggered: {
    try {
    if (phase===0 && service.firstRun && service.localSetupStatus) {
      if(service.feedError || service.observing || service.machineUse!=="" || service.setupCheckRequested)throw new Error("Fresh collector is not an empty welcome state")
      if(control(panel,"open-monitor").text!=="Start Setup →" || control(panel,"fleet-welcome").text.indexOf('An "urbit ship"')!==0)throw new Error("Missing requested welcome/CTA")
      control(panel,"open-monitor").clicked();if(panel.pane!=="setup")throw new Error("Start Setup did not navigate")
      service.openSetup("local");phase=1
    } else if(phase===1) {
      var setup=service.monitorWindow.setupPane
      var details=control(setup,"setup-install-details")
      if(details.visible || control(setup,"setup-install-approval").checked || service.setUpLocal(false))throw new Error("Install details expanded or setup approved by default")
      if(control(setup,"local-setup-primary").enabled)throw new Error("Installation bypassed its approval checkbox")
      control(setup,"setup-install-approval").checked=true
      if(!control(setup,"local-setup-primary").enabled)throw new Error("Approved setup is unavailable")
      control(setup,"setup-install-approval").checked=false
      control(setup,"setup-view-details").clicked()
      if(!details.visible || control(setup,"setup-binaries").text.indexOf("urbit-collector")<0 || !control(setup,"setup-source") || !control(setup,"setup-build-guide"))throw new Error("Expandable software/source details missing")
      control(setup,"setup-view-details").clicked()
      if(control(setup,"local-check-again").visible)throw new Error("Unchecked first setup offered recheck")
      var machine=control(setup,"machine-use-toggle");machine.checked=false;machine.toggled()
      if(service.machineUse!=="laptop")throw new Error("Laptop choice not retained")
      service.localSetupStatus=Object.assign({},service.localSetupStatus,{can_setup:false})
      if(!control(setup,"local-setup-primary").enabled || control(setup,"local-setup-primary").text!=="Check prerequisites")throw new Error("Automatic prerequisite failure left no first-check action")
      control(setup,"local-setup-primary").clicked();phase=2
    } else if(phase===2 && !service.checkingSetup) {
      if(!control(service.monitorWindow.setupPane,"local-check-again").visible)throw new Error("Completed explicit check cannot be repeated")
      console.log("PASS: fresh welcome, Start Setup, first-check gating and persisted laptop choice")
      enroll.running=true;phase=3
    } else if(phase===3 && service.connections.length && !service.busy) {
      if(service.firstRun || control(panel,"open-monitor").text.indexOf("Monitor")!==0)throw new Error("Enrolled fleet stayed in first run")
      control(panel,"setup-remote").clicked();phase=4
    } else if(phase===4) {
      if(!service.monitorWindow.visible || service.monitorWindow.page!=="setup" || service.monitorWindow.setupPane.mode!=="remote")throw new Error("Remote setup CTA failed")
      control(panel,"fleet-monitor-alpha").clicked();phase=5
    } else if(phase===5) {
      if(service.monitorWindow.selectedKey!=="runner-fixture/alpha" || service.monitorWindow.page!=="overview")throw new Error("Ship chevron lost selection")
      if(control(panel,"fleet-http-port-alpha").text!=="HTTP\\n18081" || control(panel,"fleet-open-ship-alpha").text!=="Open →")throw new Error("Fleet HTTP presentation incorrect")
      var rows=JSON.parse(JSON.stringify(service.connections));rows[0].snapshot.instances[0].health="starting";service.connections=rows
      if(control(panel,"fleet-open-ship-alpha").enabled || control(panel,"fleet-copy-code-alpha").enabled || control(service.monitorWindow,"monitor-open-ship").enabled)throw new Error("Starting ship exposed browser/code actions")
      console.log("PASS: laptop remote setup, ship-specific monitor navigation, and health-gated access")
      phase=50
    } else if(phase===50) {
      var runtime=control(service.monitorWindow,"runtime-pane"),button=control(runtime,"runtime-logs-refresh")
      if(!runtime.logs)return
      runtime.following=false
      if(runtime.running)return
      logGeometry={width:button.width,height:button.height,row:button.parent.height}
      button.clicked();button.clicked();phase=51
    } else if(phase===51) {
      var runtime=control(service.monitorWindow,"runtime-pane"),button=control(runtime,"runtime-logs-refresh")
       if(button.text!=="↻" || !button.enabled || button.width!==logGeometry.width || button.height!==logGeometry.height || button.parent.height!==logGeometry.row)throw new Error("Log polling changed refresh control geometry/state")
      if(runtime.reading)return
      console.log("PASS: log refresh retains its label, enabled state and row geometry during a slow request")
      service.openSetup("ship");service.monitorWindow.setupPane.shipKind="comet";phase=6
    } else if(phase===6 && !service.busy) {
      var form=control(service.monitorWindow.setupPane,"registration-form")
      var choice=control(form,"bootstrap-choice")
       if(choice.currentIndex!==1 || control(form,"bootstrap-selector").kind!=="recommended")throw new Error("Recommended Apps is not the default for a new comet")
      if(control(form,"bootstrap-selector").source!=="https://urbit-foundation.s3.us-east-2.amazonaws.com/omabit.pill")throw new Error("Omitted inline setting lost the packaged Recommended Apps default")
      choice.currentIndex=0;choice.activated(0)
      if(control(form,"bootstrap-selector").source!=="")throw new Error("Bare bootstrap inherited Recommended Apps")
      if(control(form,"registration-name").text!=="" || control(form,"registration-name").placeholderText!=="comet-1" || control(form,"registration-pier").visible)throw new Error("Comet name/path defaults incorrect")
      control(form,"registration-name").text="my-required-name"
      phase=61
    } else if(phase===61 && !service.busy) {
      var form=control(service.monitorWindow.setupPane,"registration-form")
      portFormHeight=form.implicitHeight
      control(form,"suggest-host-port").clicked();phase=62
    } else if(phase===62) {
      var form=control(service.monitorWindow.setupPane,"registration-form")
       if(control(service.monitorWindow,"monitor-action-progress").visible || control(panel,"fleet-action-progress").visible || form.implicitHeight!==portFormHeight)throw new Error("Port lookup moved the pane or showed global progress: "+JSON.stringify({before:portFormHeight,after:form.implicitHeight,monitor:control(service.monitorWindow,"monitor-action-progress").visible,fleet:control(panel,"fleet-action-progress").visible,problem:form.problem,action:service.currentAction}))
      if(service.checkingPort && control(form,"port-check-progress").text!=="Checking host ports…")throw new Error("Missing inline port progress")
      if(service.busy)return
      phase=7
    } else if(phase===7 && !service.busy) {
      var form=control(service.monitorWindow.setupPane,"registration-form")
       if(control(form,"registration-port").text!=="18123" || control(form,"registration-name").text!=="my-required-name")throw new Error("Host suggestion altered name or ignored host reply")
       if(form.advanced || control(form,"storage-location-selector").visible)throw new Error("Storage controls should start under Advanced")
       control(form,"registration-advanced-toggle").clicked()
       var location=control(form,"storage-location-selector")
      location.currentIndex=location.count-1;location.activated(location.currentIndex)
      if(!control(form,"registration-pier").visible)throw new Error("Custom selection did not reveal pier path")
      location.currentIndex=0;location.activated(0)
      if(control(form,"registration-pier").visible || control(form,"register-instance").enabled)throw new Error("Default path editable or boot skipped confirmation")
      console.log("PASS: required comet name, runner-checked port, custom-only path and explicit boot confirmation")
      if(!service.registerComet(service.connections[0],"ambiguous-comet",18124,true,undefined,"bare",""))throw new Error("ambiguous registration fixture refused")
      phase=8
    } else if(phase===8 && !service.busy) {
      if(service.registrationOutcome.status!=="uncertain" || service.pendingFirstBoot || service.bootTarget)throw new Error("Uncertain registration authorized first boot")
      service.editRegistration()
      if(!service.registerComet(service.connections[0],"uncertain-start",18125,true,undefined,"bare",""))throw new Error("start uncertainty fixture refused")
      phase=9
    } else if(phase===9 && !service.busy && !service.pendingFirstBoot && !service.bootTarget) {
      if(service.registrationOutcome.message.indexOf("boot result needs review")<0)throw new Error("Lost boot acknowledgement hidden")
      if(++settle<5)return
      console.log("PASS: uncertain registration never boots, and uncertain first Start is not automatically retried")
      service.monitorWindow.visible=false;Qt.quit()
    }
    } catch(error) {
      console.error("Fixture phase "+phase+": "+error)
      running=false;Qt.exit(1)
    }
  } }
}'''.replace("HELPER", json.dumps(str(helper)))
    (directory / "shell.qml").write_text(qml)
    result = run_qs(directory, timeout=35)
    text = result.stdout + result.stderr
    assert result.returncode == 0 and text.count("PASS:") == 5 and not any(e in text for e in ("Error:", "Unable to assign", "Binding loop")), text
    for line in text.splitlines():
        if "PASS:" in line:
            print(line)
    assert "machineUse=laptop" in (directory / "state/omabit/panel.ini").read_text()
    calls = [json.loads(line) for line in (directory / "calls.jsonl").read_text().splitlines()]
    assert any("suggest-port" in c for c in calls)
    assert not any(any(a in c for a in ("create-ship", "setup", "copy-code")) for c in calls)
    assert [c[c.index("create-comet")+1] for c in calls if "create-comet" in c] == ["ambiguous-comet", "uncertain-start"]
    starts = [c for c in calls if "operate" in c]
    assert [c[c.index("operate")+1] for c in starts] == ["uncertain-start"], starts
    print("PASS: preference persisted; CLI audit confirms no first-run mutations and exactly one authorized Start despite response loss")
