#!/usr/bin/env python3
"""Exercise monitor activation on the real compositor with isolated CLI fixtures."""
import argparse
import datetime
import importlib.util
import json
import os
from pathlib import Path
import subprocess
import tempfile
import time
from desktop_fixture import ROOT, HARNESS_ROOT, environment, stop_process, save_evidence, preserve_desktop

TITLE = "Omabit · Monitor & maintenance"


def run(*args, env=None):
    return subprocess.check_output(args, text=True, env=env, timeout=10).strip()


def hypr(command):
    return json.loads(run("hyprctl", "-j", command))


def dispatch(name, value, using_lua):
    if using_lua:
        quote = lambda s: json.dumps(s, ensure_ascii=False)
        if name == "focuswindow":
            expression = "hl.dsp.focus({ window = " + quote(value) + " })"
        elif name == "workspace":
            expression = "hl.dsp.focus({ workspace = " + quote(value) + " })"
        elif name == "movetoworkspacesilent":
            workspace, address = value.split(",address:", 1)
            expression = "hl.dsp.window.move({ workspace = " + quote(workspace) + ", window = " + quote("address:" + address) + ", follow = false })"
        else:
            raise AssertionError(name)
        return run("hyprctl", "dispatch", expression)
    return run("hyprctl", "dispatch", name, value)


def wait(check, message, seconds=12):
    deadline = time.monotonic() + seconds
    while time.monotonic() < deadline:
        result = check()
        if result:
            return result
        time.sleep(0.1)
    raise AssertionError(message)


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--plugin-dir", type=Path, default=ROOT / "plugins/omabit", help="test a prepared desktop snapshot instead of the source plugin")
    args = parser.parse_args()
    lock = json.loads(run("omarchy", "shell", "lock", "status"))
    assert not any(lock.get(k) for k in ("locked", "secure", "requested", "pending", "sessionLocked")), "Unlock before the workspace focus test"
    original = hypr("activewindow")
    original_workspace = hypr("activeworkspace")
    using_lua = hypr("status").get("configProvider") == "lua"
    spec = importlib.util.spec_from_file_location("panel_fixture", HARNESS_ROOT / "scripts/test-panel.py")
    fixtures = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(fixtures)
    process = None
    with tempfile.TemporaryDirectory(prefix="omabit-focus-") as temporary:
        directory = Path(temporary)
        for name in ("Commons", "Ui", "services"):
            (directory / name).symlink_to(Path("/usr/share/omarchy/shell") / name)
        (directory / "Plugin").symlink_to(args.plugin_dir.resolve())
        if args.plugin_dir.resolve().name == "native":
            (directory / "plugins").symlink_to(args.plugin_dir.resolve().parent / "plugins")
        helper = directory / "collector-fixture"
        helper.write_text(fixtures.FAKE)
        helper.chmod(0o700)
        env = environment(directory)
        mapped_windows = []
        qml = '''import QtQuick
import Quickshell
import Quickshell.Io
import "Plugin" as Plugin
ShellRoot {
  id: fixtureRoot
  // Match Omarchy's third-party loader: post-start createObject with no visual parent.
  property var service: null
  Timer { interval: 500; running: true; onTriggered: {
    var component=Qt.createComponent("Plugin/Service.qml")
    if (component.status!==Component.Ready) throw new Error(component.errorString())
    service=component.createObject(null,{settings:{collectorPath:%s,omabitPath:%s,notifications:false}})
  } }
  Component.onDestruction: if (service) service.destroy()
  Loader { id: panelLoader; active: fixtureRoot.service!==null
    sourceComponent: Component { Plugin.Panel { service: fixtureRoot.service; settings: fixtureRoot.service.settings; manageIpc:false; bar:null } }
  }
  function control(object,name,visited) {
    visited=visited || []
    if (!object || visited.indexOf(object)>=0) return null
    visited.push(object)
    if (object.objectName===name) return object
    for(var key of ["children","data","contentItem"]) {
      var children=object[key]
      if (!children) continue
      if (typeof children.length!=="number") children=[children]
      for(var i=0;i<children.length;i++) {var found=control(children[i],name,visited);if(found)return found}
    }
    return null
  }
  IpcHandler {
    target: "focus-test"
    function menu(): void {
      var button=control(panelLoader.item,"open-monitor")
      if (!button) throw new Error("Monitor menu button not found")
      button.clicked()
    }
    function close(): void { service.monitorWindow.visible=false }
    function state(): string {return JSON.stringify(service ? {visible:service.monitorWindow.visible,error:service.lastError,page:service.monitorWindow.page} : null)}
  }
}''' % (json.dumps(str(helper)), json.dumps(str(helper)))
        (directory / "shell.qml").write_text(qml)
        log_path = directory / "quickshell.log"
        try:
            with log_path.open("w") as log:
                process = subprocess.Popen(["qs", "--no-color", "-p", str(directory)], env=env, stdout=log, stderr=subprocess.STDOUT, start_new_session=True)

            def ipc(method):
                return run("qs", "ipc", "--pid", str(process.pid), "call", "focus-test", method, env=env)

            def ready():
                try:
                    return isinstance(json.loads(ipc("state")), dict)
                except (subprocess.SubprocessError, ValueError):
                    return False

            def windows():
                return [w for w in hypr("clients") if w["pid"] == process.pid and w["title"] == TITLE and w["mapped"]]

            def focused(address):
                return hypr("activewindow").get("address") == address

            wait(ready, "fixture IPC did not start")
            ipc("menu")
            window = wait(lambda: windows() and windows()[0], "monitor did not map")
            address = window["address"]
            mapped_windows.append(window)
            wait(lambda: focused(address), "first open did not focus the monitor")
            assert len(windows()) == 1
            print("PASS: actual menu-button handler opens and focuses one monitor window")

            destination = "omabit-focus-" + str(process.pid)
            away = destination + "-away"
            dispatch("movetoworkspacesilent", "name:" + destination + ",address:" + address, using_lua)
            dispatch("workspace", "name:" + away, using_lua)
            wait(lambda: hypr("activeworkspace")["name"] == away, "did not leave the monitor workspace")
            ipc("menu")
            wait(lambda: focused(address) and hypr("activeworkspace")["name"] == destination,
                 "reopening did not switch workspace and focus the existing window")
            assert len(windows()) == 1 and windows()[0]["address"] == address
            assert windows()[0]["workspace"]["name"] == destination
            mapped_windows.extend(windows())
            print("PASS: existing monitor keeps its address/workspace and receives focus from another workspace")

            ipc("menu")
            wait(lambda: focused(address), "same-workspace activation lost focus")
            assert len(windows()) == 1 and windows()[0]["address"] == address
            print("PASS: repeated activation reuses the existing window")

            special = "special:" + destination
            dispatch("movetoworkspacesilent", special + ",address:" + address, using_lua)
            dispatch("workspace", "name:" + away, using_lua)
            ipc("menu")
            wait(lambda: focused(address), "special-workspace monitor was not activated")
            assert windows()[0]["workspace"]["name"] == special
            assert any(m.get("specialWorkspace", {}).get("name") == special for m in hypr("monitors"))
            mapped_windows.extend(windows())
            print("PASS: activation also reveals a monitor on a special workspace")

            ipc("close")
            wait(lambda: not windows(), "closed monitor remained mapped")
            ipc("menu")
            reopened = wait(lambda: windows() and windows()[0], "closed monitor did not reopen")
            mapped_windows.append(reopened)
            wait(lambda: focused(reopened["address"]), "reopened monitor did not receive focus")
            assert len(windows()) == 1
            assert not json.loads(ipc("state"))["error"]
            print("PASS: close/reopen obtains and activates the new toplevel")
        except BaseException:
            print(log_path.read_text())
            raise
        finally:
            if process:
                stop_process(process)
            save_evidence(directory, log_path.read_text(), mapped_windows)
            # Only fixture windows were moved. Restore the user's original view.
            if original.get("address") and any(w["address"] == original["address"] for w in hypr("clients")):
                dispatch("focuswindow", "address:" + original["address"], using_lua)
            else:
                dispatch("workspace", "name:" + original_workspace["name"], using_lua)
        text = log_path.read_text()
        assert not any(error in text for error in ("TypeError:", "ReferenceError:", "Error:", "Binding loop detected")), text
        calls = [json.loads(line) for line in (directory / "calls.jsonl").read_text().splitlines()]
        assert not any(c for c in calls if any(action in c for action in ("operate", "configure", "create-comet", "delete-pier", "import-pier")))
        print("PASS: focus actions issued no ship mutations")


if __name__ == "__main__":
    with preserve_desktop():
        main()
