#!/usr/bin/env python3
"""Real Quickshell/Qt release activation in private HOME/XDG roots, offscreen.

Exercises actual bootstrap/process/cache code with inert candidate UI/tool bundles.
Optional application loading uses the desktop Wayland socket for PanelWindow type
registration, with every writable root private and all windows kept closed.
No live shell IPC, credentials, host services or runner operations are used.
"""
import argparse
import importlib.util
import json
import os
from pathlib import Path
import shutil
import signal
import subprocess
import sys
import tempfile
import time
from unittest.mock import patch

ROOT = Path(__file__).resolve().parents[1]
spec = importlib.util.spec_from_file_location("bootstrap_fixtures", ROOT / "scripts/test-desktop-bootstrap.py")
fixtures = importlib.util.module_from_spec(spec)
spec.loader.exec_module(fixtures)

APPLICATION = b'''import QtQuick
import "Model.js" as Model
Item {
  readonly property int bootstrapAPI: 1
  readonly property string applicationVersion: Model.release()
  property var shell; property var manifest; property var settings
  property url toolsDirectory
  property string profileName: "default"
  property var profileEnvironment: ({})
  property bool localSetupAllowed: true
  property var bootstrap
  property bool activationEnabled: true
  property bool forcedBusy: false
  readonly property bool busy: forcedBusy || !activationEnabled
  property bool copyingCode: false
  property bool checkingSetup: false
  property var pendingFirstBoot: null
  readonly property bool reloadSafe: !busy
  property bool registrationsLoaded: true
  property bool observing: true
  property var registrations: profileName==="default" ? [{name:"existing",runner_id:"runner-existing"}] : []
  property var connections: profileName==="default" ? [{snapshot:{instances:[{id:"one"},{id:"two"}]}}] : []
  property string implementationVersion: Model.release()
  property QtObject monitorWindow: QtObject {property bool visible:false;property string page:"overview";property string selectedKey:"runner-existing/one"}
  function openMonitor(c,i,p) {monitorWindow.visible=true;monitorWindow.page=p}
}
'''


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--require-quickshell", action="store_true")
    parser.add_argument("--application-assets", type=Path, help="also load the actual UI and binaries from a built development/release bundle")
    parser.add_argument("--previous-application-assets", type=Path, help="activate this actual bundle first, then hot-upgrade in the same shell")
    args = parser.parse_args()
    if args.previous_application_assets and not args.application_assets:
        parser.error("previous application assets require target application assets")
    qs = shutil.which("quickshell")
    if not qs:
        if args.require_quickshell:
            parser.error("Quickshell is required")
        print("SKIP: real bootstrap lifecycle needs Quickshell")
        return
    with tempfile.TemporaryDirectory(prefix="omabit-bootstrap-qml-") as temporary:
        root = Path(temporary)
        env = dict(os.environ)
        for key, leaf in (("HOME", "home"), ("XDG_CONFIG_HOME", "config"), ("XDG_CACHE_HOME", "cache"),
                          ("XDG_STATE_HOME", "state"), ("XDG_DATA_HOME", "data"), ("XDG_RUNTIME_DIR", "runtime")):
            directory = root / leaf
            directory.mkdir(mode=0o700)
            env[key] = str(directory)
        env.update(QT_QPA_PLATFORM="offscreen", QT_QUICK_BACKEND="software", QSG_RHI_BACKEND="software", QT_QPA_PLATFORMTHEME="", NO_COLOR="1")
        for key in ("WAYLAND_DISPLAY", "DISPLAY", "DBUS_SESSION_BUS_ADDRESS", "HYPRLAND_INSTANCE_SIGNATURE", "QML_IMPORT_PATH", "QML2_IMPORT_PATH"):
            env.pop(key, None)
        if args.application_assets:
            display = os.environ.get("WAYLAND_DISPLAY", "")
            if not display:
                parser.error("actual Omarchy PanelWindow loading requires a Wayland session")
            env["WAYLAND_DISPLAY"] = display if display.startswith("/") else str(Path(os.environ["XDG_RUNTIME_DIR"]) / display)
            env["QT_QPA_PLATFORM"] = "wayland"
        # Production's release cache is outside the shell configuration tree.
        # Keeping it inside lets Quickshell discover undeclared QML types and
        # masks missing qmldir entries during a real installed-plugin update.
        configuration = root / "shell-config"
        configuration.mkdir()
        shutil.copytree(ROOT / "native", configuration / "native")
        for name in ("Commons", "Ui", "services"):
            stock = Path("/usr/share/omarchy/shell") / name
            if stock.exists():
                (configuration / name).symlink_to(stock)
        if not (configuration / "Commons").exists():
            if args.application_assets:
                parser.error("actual application qualification requires the installed stock Omarchy QML modules")
            commons = configuration / "Commons"
            commons.mkdir()
            (commons / "qmldir").write_text("singleton Style 1.0 Style.qml\nsingleton Color 1.0 Color.qml\n")
            (commons / "Style.qml").write_text('pragma Singleton\nimport QtQuick\nQtObject {function space(n){return n} property var font: ({family:"sans-serif"})}\n')
            (commons / "Color.qml").write_text('pragma Singleton\nimport QtQuick\nQtObject {property color foreground:"white"}\n')
        first_metadata, first_archive, pins = fixtures.bundle(root / "one", qml=APPLICATION)
        pins_path = configuration / "native/bootstrap/release.json"
        pins_path.write_text(json.dumps(pins))
        with patch.dict(os.environ, env, clear=True):
            desktop = fixtures.desktop.Desktop(pins_path)
            first = desktop.import_release(first_metadata, first_archive)["release"]
        shell = configuration / "shell.qml"
        shell.write_text('''import QtQuick
import Quickshell
import Quickshell.Io
import "native/v1" as Bootstrap
ShellRoot {
  Bootstrap.Service {id:host}
  Bootstrap.Panel {id:widget;bootstrap:host;manageIpc:false;bar:null}
  IpcHandler {
    target:"bootstrap-fixture"
    function busy(value: bool): void {host.payload.forcedBusy=value}
    function view(page: string, selection: string): void {host.payload.monitorWindow.page=page;host.payload.monitorWindow.selectedKey=selection}
    function details(): string {return JSON.stringify({diagnostic:host.diagnostic(),implementation:host.payload && "implementationVersion" in host.payload ? host.payload.implementationVersion : "",tools:host.payload ? host.payload.toolsDirectory.toString() : "",environment:host.payload ? host.payload.profileEnvironment : {},widget_ready:!!widget.implementation,widget_version:widget.implementation && "implementationVersion" in widget.implementation ? widget.implementation.implementationVersion : "",view:host.payload ? {page:host.payload.monitorWindow.page,selection:host.payload.monitorWindow.selectedKey} : {},session:host.session})}
  }
}
''')
        process = None
        with (root / "quickshell.log").open("w+") as log:
            def launch():
                return subprocess.Popen([qs, "--no-color", "-p", str(shell)], env=env, stdout=log, stderr=subprocess.STDOUT, start_new_session=True)

            def ipc(*arguments):
                return subprocess.check_output([qs, "ipc", "-p", str(shell), "call", "bootstrap-fixture", *arguments], env=env, text=True, stderr=subprocess.DEVNULL, timeout=3)

            def wait(predicate, timeout=35):
                deadline = time.monotonic() + timeout
                last = None
                while time.monotonic() < deadline:
                    if process.poll() is not None:
                        break
                    try:
                        last = json.loads(ipc("details"))
                        if predicate(last):
                            return last
                    except (subprocess.SubprocessError, ValueError):
                        pass
                    time.sleep(.1)
                log.flush()
                raise AssertionError(str(last) + "\n" + (root / "quickshell.log").read_text()[-12000:])

            def stop():
                if process and process.poll() is None:
                    os.killpg(process.pid, signal.SIGTERM)
                    process.wait(timeout=10)

            try:
                process = launch()
                live = wait(lambda r: r["diagnostic"]["loaded_release"] == first and r["widget_ready"])
                assert live["implementation"] == "0.1.0-rc.13" and live["diagnostic"]["ships"] == 2
                ipc("busy", "true")
                second_metadata, second_archive, _ = fixtures.bundle(root / "two", "0.1.0-rc.14", qml=APPLICATION)
                with patch.dict(os.environ, env, clear=True):
                    second = desktop.import_release(second_metadata, second_archive)["release"]
                    desktop.select("default", second)
                waiting = wait(lambda r: r["diagnostic"]["phase"] == "waiting")
                assert waiting["diagnostic"]["loaded_release"] == first and waiting["session"] == live["session"]
                ipc("busy", "false")
                switched = wait(lambda r: r["diagnostic"]["loaded_release"] == second and r["widget_ready"])
                assert switched["implementation"] == "0.1.0-rc.14" and second in switched["tools"]
                assert switched["widget_version"] == switched["implementation"]
                print("PASS: actual Qt loads fresh nested JS and paired tools, defers busy activation and retains its lease")
                bad_metadata, bad_archive, _ = fixtures.bundle(root / "bad", "0.1.0-rc.15", qml=b"invalid QML candidate")
                with patch.dict(os.environ, env, clear=True):
                    bad = desktop.import_release(bad_metadata, bad_archive)["release"]
                    desktop.select("default", bad)
                recovered = wait(lambda r: r["diagnostic"]["requested_release"] == bad and r["diagnostic"]["loaded_release"] == second and bool(r["diagnostic"]["error"]))
                assert recovered["implementation"] == "0.1.0-rc.14"
                print("PASS: invalid candidate rolls back the UI without relabeling requested/loaded versions")
                widget_meta, widget_archive, _ = fixtures.bundle(root / "bad-widget", "0.1.0-rc.16", qml=APPLICATION, panel=b"invalid widget QML")
                with patch.dict(os.environ, env, clear=True):
                    bad_widget = desktop.import_release(widget_meta, widget_archive)["release"]
                    desktop.select("default", bad_widget)
                wait(lambda r: r["diagnostic"]["requested_release"] == bad_widget and r["diagnostic"]["loaded_release"] == second and bool(r["diagnostic"]["error"]))
                with patch.dict(os.environ, env, clear=True):
                    assert desktop.resolve()["active"] == second
                mismatch_meta, mismatch_archive, _ = fixtures.bundle(root / "wrong-version", "0.1.0-rc.17", qml=APPLICATION.replace(b"applicationVersion: Model.release()", b'applicationVersion: "0.1.0-rc.1"'))
                with patch.dict(os.environ, env, clear=True):
                    mismatch = desktop.import_release(mismatch_meta, mismatch_archive)["release"]
                    desktop.select("default", mismatch)
                wait(lambda r: r["diagnostic"]["requested_release"] == mismatch and r["diagnostic"]["loaded_release"] == second and bool(r["diagnostic"]["error"]))
                print("PASS: broken widgets and runtime version mismatches never become the committed release")
                with patch.dict(os.environ, env, clear=True):
                    desktop.create_profile("fresh")
                    desktop.select("fresh", second)
                fresh = wait(lambda r: r["diagnostic"]["profile"] == "fresh" and r["diagnostic"]["phase"] == "ready")
                assert fresh["environment"]["XDG_CONFIG_HOME"] != env["XDG_CONFIG_HOME"] and fresh["diagnostic"]["runners"] == 0
                stop()
                process = launch()
                restarted = wait(lambda r: r["diagnostic"]["profile"] == "fresh" and r["diagnostic"]["phase"] == "ready")
                assert restarted["diagnostic"]["loaded_release"] == second
                print("PASS: isolated fresh profile and selected release survive a real bootstrap process restart")
                if args.application_assets:
                    with patch.dict(os.environ, env, clear=True):
                        desktop.create_profile("application")
                    sequence = ([args.previous_application_assets] if args.previous_application_assets else []) + [args.application_assets]
                    for assets in sequence:
                        metadata = assets / "release.json"
                        document = json.loads(metadata.read_text())
                        archive = assets / ("omabit-" + document["version"] + "-linux-amd64.tar.gz")
                        with patch.dict(os.environ, env, clear=True):
                            actual = desktop.import_release(metadata, archive)["release"]
                            desktop.select("application", actual)
                        loaded = wait(lambda r: r["diagnostic"]["profile"] == "application" and r["diagnostic"]["loaded_release"] == actual and r["widget_ready"])
                        assert loaded["diagnostic"]["loaded_version"] == document["version"]
                    if args.previous_application_assets:
                        print("PASS: actual previous-to-target bundle hot activation in the same shell process")
                    assert loaded["diagnostic"]["runners"] == 0 and loaded["environment"]["XDG_CONFIG_HOME"] != env["XDG_CONFIG_HOME"]
                    ipc("view", "setup", "runner-isolated/ship-test")
                    preference = Path(loaded["environment"]["XDG_STATE_HOME"]) / "omabit/panel.ini"
                    assert "desktopPage=setup" in preference.read_text(), ("view was not saved before shutdown", preference.read_text())
                    stop()
                    process = launch()
                    restored = wait(lambda r: r["diagnostic"]["profile"] == "application" and r["widget_ready"])
                    assert restored["view"] == {"page": "setup", "selection": "runner-isolated/ship-test"}, (restored, {str(p.relative_to(root)):p.read_text() for p in root.rglob("panel.ini")}, (root / "quickshell.log").read_text()[-6000:])
                    print("PASS: actual packaged application and real helpers load a fresh isolated desktop profile")
                    print("PASS: the actual application's selected page/ship survive a profile-scoped process restart")
            finally:
                stop()


if __name__ == "__main__":
    main()
