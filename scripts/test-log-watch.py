#!/usr/bin/env python3
"""Mapped log-pane lifecycle/stream regression with an isolated collector fixture."""
import json
import os
from pathlib import Path
import subprocess
import tempfile

ROOT = Path(__file__).resolve().parents[1]

HELPER = r'''#!/usr/bin/env python3
import datetime,json,os,pathlib,signal,sys,time
directory=pathlib.Path(__file__).parent
name=sys.argv[sys.argv.index("logs")+1]
revision=int(sys.argv[sys.argv.index("--revision")+1])
watch="--watch" in sys.argv
def event(kind):
    with (directory/"events.jsonl").open("a") as f:
        f.write(json.dumps(dict(kind=kind,pid=os.getpid(),name=name,revision=revision,watch=watch))+"\n")
def tail():
    return dict(runner_id="runner-test",instance_id=name,config_revision=revision,
                source="docker",container_id="a"*64,observed_at=datetime.datetime.now(datetime.timezone.utc).isoformat(),
                lines=["boot: "+name]+["x"*4000]*8,truncated=False)
def record():
    return dict(runner_id="runner-test",instance_id=name,config_revision=revision,logs=tail())
def stop(*_):
    # A late old-selection record during graceful cancellation must be discarded.
    print(json.dumps(record() if watch else tail()),flush=True)
    sys.exit(0)
signal.signal(signal.SIGTERM,stop)
event("start")
try:
    if name=="oversized":
        print("x"*524289,flush=True)
        time.sleep(10)
    elif name=="mismatch":
        value=record();value["logs"]["instance_id"]="other"
        print(json.dumps(value),flush=True)
        time.sleep(10)
    elif name=="denied":
        value=record();del value["logs"]
        value["error"]=dict(code="logs_access_denied",message="Review operator access.")
        print(json.dumps(value),flush=True)
        sys.exit(1)
    elif name=="failed":
        sys.exit(1)
    elif name=="retry":
        value=record();del value["logs"]
        value.update(error=dict(code="connection_unavailable",message="Reconnecting with backoff."),retry_after_ms=30000)
        print(json.dumps(value),flush=True)
        time.sleep(30)
    elif not watch:
        time.sleep(.3)
        print(json.dumps(tail()))
    else:
        while True:
            # Fragment a record, then concatenate two records in one write.
            line=json.dumps(record())+"\n"
            sys.stdout.write(line[:17]);sys.stdout.flush();time.sleep(.005)
            sys.stdout.write(line[17:]+line);sys.stdout.flush()
            time.sleep(.05)
finally:
    event("exit")
'''

QML = r'''import QtQuick
import QtQuick.Window
import Quickshell
import "Plugin" as Plugin
ShellRoot {
  id: test
  property int phase: 0
  property double since: Date.now()
  property bool overview: true
  // Hyprland does not implement native minimization. Inject the Qt visibility
  // state to exercise the same gate used by MonitorWindow on supporting WMs.
  property int viewVisibility: window.visibility
  function advance(n) { phase=n;since=Date.now() }
  function select(name,revision) { pane.instance={id:name,mode:"fake",config_revision:revision || 1} }
  QtObject {
    id: fixtureService
    property bool observing: true
    property string collector: HELPER
    property var settings: ({})
    property var registrations: [{name:"test",runner_id:"runner-test",revision:1}]
    property double now: Date.now()
    function command(args) { return [collector,"--format","json"].concat(args) }
  }
  Window {
    id: window
    visible:true;width:800;height:700
    Plugin.RuntimePane {
      id: pane
      width:parent.width
      service:fixtureService
      connection:({name:"test",runner_id:"runner-test",mode:"development-managed",status:"available",access_role:"operator",features:["runtime-logs"],snapshot:{},checked_at:new Date(fixtureService.now).toISOString()})
      instance:({id:"alpha",mode:"fake",config_revision:1})
      active:window.visible && test.viewVisibility!==Window.Minimized && test.overview
    }
  }
  Timer { interval:50;repeat:true;running:true;onTriggered: {
    fixtureService.now=Date.now()
    var elapsed=Date.now()-test.since
    if (phase===0 && pane.logs && elapsed>6500) {
      if (!pane.running || pane.blocked || pane.output.length>524288) throw new Error("Healthy follow stopped or accumulated stream output")
      console.log("PASS: one persistent watcher accepts fragmented/multiple records beyond the per-record lifetime limit")
      pane.following=false;advance(1)
    } else if (phase===1 && !pane.running) {
      if (!pane.logs) throw new Error("Pause lost the displayed tail")
      pane.refresh();pane.refresh();advance(2)
    } else if (phase===2 && !pane.running && elapsed>400) {
      if (!pane.logs || pane.blocked || pane.following) throw new Error("One-shot refresh while paused failed")
      pane.following=true;advance(3)
    } else if (phase===3 && pane.logs && pane.running && elapsed>150) {
      select("beta");advance(4)
    } else if (phase===4 && pane.logs && pane.logs.instance_id==="beta") {
      if (pane.logs.lines[0]!=="boot: beta") throw new Error("Old selection output crossed cancellation")
      select("beta",2);advance(5)
    } else if (phase===5 && pane.logs && pane.logs.config_revision===2) {
      fixtureService.registrations=[{name:"test",runner_id:"runner-test",revision:2}];advance(6)
    } else if (phase===6 && pane.logs && elapsed>250) {
      viewVisibility=Window.Minimized;advance(7)
    } else if (phase===7 && !pane.running && elapsed>150) {
      if (pane.active || pane.logs) throw new Error("Minimized pane kept working")
      viewVisibility=Window.Windowed;advance(8)
    } else if (phase===8 && pane.logs && pane.running) {
      overview=false;advance(9)
    } else if (phase===9 && !pane.running) {
      overview=true;advance(10)
    } else if (phase===10 && pane.logs && pane.running) {
      window.visible=false;advance(11)
    } else if (phase===11 && !pane.running) {
      window.visible=true;advance(12)
    } else if (phase===12 && pane.logs && pane.running) {
      console.log("PASS: pause/refresh, selection/revision/enrollment changes, minimize, navigation and hide reap before restarting")
      select("retry");advance(13)
    } else if (phase===13 && pane.error && elapsed>250) {
      if (!pane.running || pane.logs || pane.blocked) throw new Error("Transient failure did not stay in collector backoff")
      pane.connection={name:"test",runner_id:"runner-test",mode:"development-managed",status:"unavailable"};advance(14)
    } else if (phase===14 && elapsed>350) {
      if (!pane.running) throw new Error("Inventory failure defeated log backoff")
      pane.connection={name:"test",runner_id:"runner-test",mode:"development-managed",status:"available",access_role:"operator",features:["runtime-logs"],snapshot:{},checked_at:new Date(fixtureService.now).toISOString()}
      advance(15)
    } else if (phase===15 && elapsed>350) {
      if (!pane.running) throw new Error("Inventory recovery respawned log watcher")
      select("denied");advance(16)
    } else if (phase===16 && elapsed>1000 && !pane.running) {
      if (!pane.blocked || pane.logs || pane.error.indexOf("operator")<0) throw new Error("Denied watcher not stopped")
      select("mismatch");advance(17)
    } else if (phase===17 && elapsed>700 && !pane.running) {
      if (!pane.blocked || pane.logs) throw new Error("Mismatched record accepted")
      select("oversized");advance(18)
    } else if (phase===18 && elapsed>700 && !pane.running) {
      if (!pane.blocked || pane.logs || pane.output.length) throw new Error("Oversized record retained")
      select("failed");advance(19)
    } else if (phase===19 && elapsed>1000 && !pane.running) {
      if (!pane.blocked || pane.logs) throw new Error("Unexpected exit restarted automatically")
      console.log("PASS: backoff survives inventory changes; access, mismatched/oversized output and unexpected exit fail closed without respawn")
      select("shutdown");advance(20)
    } else if (phase===20 && pane.logs && pane.running) {
      Qt.quit()
    }
    if (elapsed>10000) { console.error("Error: phase timed out: "+phase+" "+pane.error);Qt.quit() }
  } }
}'''

with tempfile.TemporaryDirectory(prefix="omabit-log-watch-") as temporary:
    directory = Path(temporary)
    for name in ("Commons", "Ui", "services"):
        (directory / name).symlink_to(Path("/usr/share/omarchy/shell") / name)
    (directory / "Plugin").symlink_to(ROOT / "plugins/omabit")
    helper = directory / "collector"
    helper.write_text(HELPER)
    helper.chmod(0o700)
    (directory / "shell.qml").write_text(QML.replace("HELPER", json.dumps(str(helper))))
    runtime = directory / "runtime"
    runtime.mkdir(mode=0o700)
    display = os.environ["WAYLAND_DISPLAY"]
    if not display.startswith("/"):
        display = str(Path(os.environ["XDG_RUNTIME_DIR"]) / display)
    env = dict(os.environ, WAYLAND_DISPLAY=display, XDG_RUNTIME_DIR=str(runtime), XDG_STATE_HOME=str(directory / "state"))
    try:
        result = subprocess.run(["qs", "--no-color", "-p", str(directory)], env=env, capture_output=True, text=True, timeout=45)
    except subprocess.TimeoutExpired as e:
        raise AssertionError((e.stdout or b"").decode() + (e.stderr or b"").decode()) from e
    text = result.stdout + result.stderr
    assert result.returncode == 0 and text.count("PASS:") == 3 and not any(e in text for e in ("Error:", "Unable to assign", "Binding loop")), text
    events = [json.loads(line) for line in (directory / "events.jsonl").read_text().splitlines()]
    starts = [e for e in events if e["kind"] == "start"]
    assert len([e for e in starts if e["name"] == "alpha" and e["watch"]]) == 2, events
    assert len([e for e in starts if not e["watch"]]) == 1, events
    for name in ("retry", "denied", "mismatch", "oversized", "failed", "shutdown"):
        assert len([e for e in starts if e["name"] == name]) == 1, events
    live = set()
    for event in events:
        if event["kind"] == "start":
            assert not live, ("overlapping collectors", events)
            live.add(event["pid"])
        else:
            live.discard(event["pid"])
    # Quickshell may kill its final child on destruction instead of delivering
    # SIGTERM. Either way, no collector may survive the shell.
    for event in starts:
        assert not Path(f"/proc/{event['pid']}").exists(), ("orphan collector", events)
    for line in text.splitlines():
        if "PASS:" in line:
            print(line)
    print("PASS: process audit confirms bounded launches, no overlap, and no collector surviving shell exit")
