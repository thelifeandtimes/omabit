#!/usr/bin/env python3
"""Exercise the shipped observation schedule and stdin writer in private Qt."""
import importlib.util
import os
from pathlib import Path
import shutil
import subprocess
import tempfile

ROOT = Path(__file__).resolve().parents[1]
spec = importlib.util.spec_from_file_location("callbacks", ROOT / "scripts/test-rc11-panel-offscreen.py")
callbacks = importlib.util.module_from_spec(spec)
spec.loader.exec_module(callbacks)


def main():
    qml = shutil.which(os.environ.get("QT_QML_EXECUTABLE", "qml6"))
    if not qml:
        raise SystemExit("Qt qml6 is required")
    source = (ROOT / "plugins/omabit/Service.qml").read_text()
    methods = "\n".join(callbacks.extract_method(source, name) for name in ("setPanelOpen", "updateObservationSchedule"))
    with tempfile.TemporaryDirectory(prefix="omabit-observation-schedule-") as tmp:
        root = Path(tmp)
        shutil.copyfile(ROOT / "plugins/omabit/ObservationSchedule.qml", root / "ObservationSchedule.qml")
        for name in ("home", "runtime", "config", "state", "cache", "data"):
            (root / name).mkdir(mode=0o700)
        (root / "main.qml").write_text('''import QtQuick
Item {
 id: root
 property bool transportStopping: false
 property var writes: []
 property bool wantMonitor: false
 property bool wantWork: false
 property QtObject feed: QtObject {
   property bool running: true
   function write(value) { root.writes=root.writes.concat([JSON.parse(value)]) }
 }
 ObservationSchedule {
   id: observationSchedule
   monitorOpen: root.wantMonitor
   workPending: root.wantWork
   onChanged: function(refresh) { root.updateObservationSchedule(refresh) }
 }
 @@METHODS@@
 function check(value,message) { if(!value) throw new Error(message) }
 function last() { return writes[writes.length-1] }
 function run() {
   check(!observationSchedule.active && observationSchedule.interval==="30s", "initial hidden cadence")
   var a={},b={}
   setPanelOpen(a,true)
   check(observationSchedule.foreground && last().interval==="2s" && last().refresh, "opening did not wake watcher")
   var count=writes.length
   setPanelOpen(a,true)
   check(writes.length===count && observationSchedule.panels.length===1,"duplicate panel registration")
   setPanelOpen(b,true);setPanelOpen(a,false)
   check(observationSchedule.active && writes.length===count,"closing one of two panels idled watcher")
   setPanelOpen(b,false)
   check(!observationSchedule.active && last().interval==="30s" && !last().refresh,"last panel did not idle watcher")
   wantWork=true
   check(observationSchedule.active && last().refresh,"pending action was not observed promptly")
   count=writes.length;wantMonitor=true
   check(writes.length>count && last().refresh,"opening during action did not refresh")
   wantWork=false
   check(observationSchedule.active,"action completion idled visible monitor")
   wantMonitor=false
   check(last().interval==="30s" && !last().refresh,"closed monitor kept active cadence")
   transportStopping=true;count=writes.length;setPanelOpen(a,true)
   check(writes.length===count,"draining service restarted observation")
   transportStopping=false;feed.running=false;setPanelOpen(a,false);setPanelOpen(a,true)
   check(writes.length===count,"schedule wrote to exited collector")
   feed.running=true;updateObservationSchedule(false)
   check(last().interval==="2s" && !last().refresh,"new collector missed current schedule")
   console.log("PASS: hidden/visible cadence, shared panels, pending actions, immediate wake, no-op updates and drain safety")
 }
 Component.onCompleted: { try { run(); Qt.quit() } catch(e) { console.error("FAIL:",e); Qt.exit(1) } }
}
'''.replace('@@METHODS@@', methods))
        env = {"PATH": "/usr/bin:/bin", "LANG": "C.UTF-8", "QT_QPA_PLATFORM": "offscreen",
               "QT_QUICK_BACKEND": "software", "QSG_RHI_BACKEND": "software", "QML_DISABLE_DISK_CACHE": "1",
               "HOME": str(root / "home"), **{"XDG_" + key + "_HOME": str(root / leaf) for key, leaf in
               (("CONFIG", "config"), ("STATE", "state"), ("CACHE", "cache"), ("DATA", "data"))},
               "XDG_RUNTIME_DIR": str(root / "runtime"), "QT_FORCE_STDERR_LOGGING":"1",
               "QT_LOGGING_RULES":"*.warning=true;*.critical=true;qml.debug=true"}
        result = subprocess.run([qml, str(root / "main.qml")], env=env, text=True, capture_output=True, timeout=10)
        output = result.stdout + result.stderr
        assert result.returncode == 0 and "PASS:" in output, output
        assert not any(token in output for token in ("Error:", "FAIL:", "Binding loop", "Cannot assign", "is not defined")), output
        print(next(line for line in output.splitlines() if "PASS:" in line))

if __name__ == "__main__":
    main()
