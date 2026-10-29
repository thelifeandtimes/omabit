#!/usr/bin/env python3
"""Host diagnostics in real Qt bindings; inert processes, no desktop session."""
import argparse
import importlib.util
import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile

ROOT = Path(__file__).resolve().parents[1]
COMPONENTS = (
    "ServiceUpgrade.qml", "ServiceUpgradePane.qml", "Label.qml", "Design.qml",
    "ActionButton.qml", "Hint.qml", "Card.qml", "CardTitle.qml",
)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--require-qt", action="store_true")
    args = parser.parse_args()
    executable = shutil.which(os.environ.get("QT_QML_EXECUTABLE", "qml6"))
    if not executable:
        if args.require_qt:
            parser.error("Qt 6 qml6 is required")
        print("SKIP: service upgrade offscreen check requires Qt 6 qml6")
        return
    with tempfile.TemporaryDirectory(prefix="omabit-upgrade-offscreen-") as temporary:
        directory = Path(temporary)
        plugin = directory / "Plugin"
        plugin.mkdir()
        for name in COMPONENTS:
            shutil.copy2(ROOT / "plugins/omabit" / name, plugin / name)
        (plugin / "qmldir").write_text("singleton Design 1.0 Design.qml\n")
        commons = directory / "qs/Commons"
        commons.mkdir(parents=True)
        (commons / "qmldir").write_text("module qs.Commons\nsingleton Style 1.0 Style.qml\nsingleton Color 1.0 Color.qml\n")
        (commons / "Style.qml").write_text('pragma Singleton\nimport QtQuick\nQtObject { function space(n) { return n } property var font: ({family:"sans-serif",caption:12,bodySmall:14,body:14}) }\n')
        (commons / "Color.qml").write_text('pragma Singleton\nimport QtQuick\nQtObject { property color background:"black";property color foreground:"white";property color accent:"blue";property color urgent:"red" }\n')
        io = directory / "Quickshell/Io"
        io.mkdir(parents=True)
        (io / "qmldir").write_text("module Quickshell.Io\nProcess 1.0 Process.qml\nSplitParser 1.0 SplitParser.qml\n")
        (io / "Process.qml").write_text('import QtQuick\nQtObject { property bool running:false;property var command:[];property var stdout;property var stderr;signal started();signal exited(int code,int status);function signal(n) { running=false } function finish(code) { running=false;exited(code,0) } }\n')
        (io / "SplitParser.qml").write_text('import QtQuick\nQtObject { property string splitMarker:"\\n";signal read(string data) }\n')
        qml = (ROOT / "scripts/test-service-upgrade-offscreen.qml").read_text()
        (directory / "main.qml").write_text(qml)
        runtime = directory / "runtime"
        runtime.mkdir(mode=0o700)
        env = dict(os.environ, QT_QPA_PLATFORM="offscreen", QT_QUICK_BACKEND="software",
                   QSG_RHI_BACKEND="software", QT_QUICK_CONTROLS_STYLE="Basic", QT_QPA_PLATFORMTHEME="",
                   QT_IM_MODULE="", HOME=str(directory / "home"), XDG_RUNTIME_DIR=str(runtime),
                   XDG_CONFIG_HOME=str(directory / "config"), XDG_CACHE_HOME=str(directory / "cache"),
                   XDG_STATE_HOME=str(directory / "state"), QML_DISABLE_DISK_CACHE="1",
                   QT_LOGGING_RULES="*.warning=true;*.critical=true;qml.debug=true", QT_FORCE_STDERR_LOGGING="1")
        for name in ("WAYLAND_DISPLAY", "DISPLAY", "HYPRLAND_INSTANCE_SIGNATURE", "DBUS_SESSION_BUS_ADDRESS", "QML_IMPORT_PATH", "QML2_IMPORT_PATH"):
            env.pop(name, None)
        try:
            result = subprocess.run([executable, "-I", str(directory), str(directory / "main.qml")], env=env, capture_output=True, text=True, timeout=25)
        except subprocess.TimeoutExpired as error:
            raise AssertionError((error.stdout or b"").decode() + (error.stderr or b"").decode()) from error
        text = result.stdout + result.stderr
        assert result.returncode == 0 and text.count("PASS:") == 3, text
        assert not any(error in text for error in ("Error:", "FAIL:", "Unable to assign", "Binding loop", "Cannot assign", "is not defined")), text
        for line in text.splitlines():
            if "PASS:" in line:
                print(line)


if __name__ == "__main__":
    main()
