#!/usr/bin/env python3
"""Load a closed panel with isolated CLI fixtures on the current Wayland session."""
import json
import os
import pathlib
import subprocess
import tempfile

ROOT = pathlib.Path(__file__).resolve().parents[1]

FAKE = '''#!/usr/bin/env python3
import datetime,json,sys,time
if "inventory" in sys.argv:
    while True:
        snapshot={"runner_id":"runner-fixture","mode":"development-managed-fake","instances":[{"id":"alpha","config_revision":2,"applied_revision":2,"desired_state":"running","observed_state":"running","health":"healthy","config":{"http_port":18081,"ames_port":34343,"loom":31},"endpoint":"http://127.0.0.1:18081"}],"operations":[],"incidents":[]}
        print(json.dumps({"name":"fixture","runner_id":"runner-fixture","mode":"development-managed-fake","access_role":"operator","features":["create-fake"],"status":"available","checked_at":datetime.datetime.now(datetime.timezone.utc).isoformat(),"snapshot":snapshot}),flush=True)
        time.sleep(.2)
elif "ships" in sys.argv:
    print(json.dumps([{"id":"a"*32,"name":"app-fixture","ship":"~zod","origin":"http://127.0.0.1:18085","status":"login_required"}]))
elif "ship-login" in sys.argv:
    assert "--expected-ship-id" in sys.argv
    assert "fixture-only-secret" not in sys.argv
    assert sys.stdin.read() == "fixture-only-secret"
    print(json.dumps({"id":"a"*32,"name":"app-fixture","status":"authenticated"}))
elif "theme-status" in sys.argv:
    print(json.dumps({"version":1,"ships":{}}))
elif "theme-preview" in sys.argv:
    print(json.dumps({"id":"omarchy-urbit-theme","name":"Fixture","dark":True,"primary":"#112233","secondary":"#112233","tertiary":"#112233","background":"#112233","surface":"#112233"}))
elif "theme-sync" in sys.argv:
    print(json.dumps({"complete":True,"failed":[]}))
else:
    assert "--expected-runner-id" in sys.argv
    print(json.dumps({"id":"op-fixture","status":"succeeded"}))
'''


def main():
    with tempfile.TemporaryDirectory(prefix="omabit-panel-") as temporary:
        directory = pathlib.Path(temporary)
        for name in ("Commons", "Ui", "services"):
            (directory / name).symlink_to(pathlib.Path("/usr/share/omarchy/shell") / name)
        (directory / "Plugin").symlink_to(ROOT / "plugins/omabit")
        helper = directory / "collector-fixture"
        helper.write_text(FAKE)
        helper.chmod(0o700)
        runtime = directory / "runtime"
        runtime.mkdir(mode=0o700)
        (directory / "shell.qml").write_text('''import QtQuick
import Quickshell
import "Plugin" as Plugin
ShellRoot {
  property var testSettings: ({collectorPath: %s, omabitPath: %s, notifications: false})
  Plugin.Service { id: service; settings: testSettings }
  Plugin.Panel { id: panel; service: service; settings: testSettings; manageIpc: false; bar: null }
  property int phase: 0
  Timer {
    interval: 300; repeat: true; running: true
    onTriggered: {
      if (phase === 0 && service.connections.length === 1) {
        if (service.connections[0].snapshot.instances[0].health !== "healthy") throw new Error("bad fixture state")
        service.operate(service.connections[0], service.connections[0].snapshot.instances[0], "stop")
        phase = 1
      } else if (phase === 1 && service.lastResult.indexOf("op-fixture") >= 0) {
        if (!service.ships.length) return
        service.shipAction(service.ships[0], "ship-login", "fixture-only-secret")
        phase = 2
      } else if (phase === 2 && service.shipStatus["aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"] && service.shipStatus["aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"].status === "authenticated") {
        if (service.pendingInput !== "") throw new Error("secret retained after handoff")
        console.log("PASS: real QML panel/service load, inventory, selected mutation, and secret stdin handoff")
        Qt.quit()
      }
    }
  }
}''' % (json.dumps(str(helper)),json.dumps(str(helper))))
        # Quickshell PanelWindow has no offscreen backend. Connect to the current
        # compositor but keep the panel closed; use a separate IPC/log runtime.
        display = os.environ.get("WAYLAND_DISPLAY", "")
        if not display:
            raise RuntimeError("a Wayland session is required for the closed-panel load test")
        if not display.startswith("/"):
            display = str(pathlib.Path(os.environ["XDG_RUNTIME_DIR"]) / display)
        env = dict(os.environ, QT_QPA_PLATFORM="wayland", WAYLAND_DISPLAY=display, XDG_RUNTIME_DIR=str(runtime), NO_COLOR="1")
        result = subprocess.run(["qs", "--no-color", "-p", str(directory)], env=env, capture_output=True, text=True, timeout=15)
        text = result.stdout + result.stderr
        if result.returncode != 0 or "PASS: real QML" not in text:
            raise RuntimeError(text)
        for line in text.splitlines():
            if "PASS:" in line:
                print(line)
        if any(message in text for message in ("ReferenceError:", "TypeError:", "Unable to assign", "is not a type", "Cannot assign to")):
            raise RuntimeError(text)


if __name__ == "__main__":
    main()
