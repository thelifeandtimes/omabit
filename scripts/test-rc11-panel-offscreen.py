#!/usr/bin/env python3
"""Real Qt binding/queued-callback regressions, without Quickshell or a compositor.

Requires Qt 6's qml6 (or QT_QML_EXECUTABLE), QtQuick/Controls/Dialogs and the
offscreen platform plugin. Developer runs print SKIP if qml6 is absent; CI jobs
qualifying QML must install those dependencies and use --require-qt. Installed
Omarchy modules are replaced only by small theme-value stubs in the temp fixture.
The application components and service callbacks are loaded from this checkout.
"""
import argparse
import datetime
import importlib.util
import json
import os
from pathlib import Path
import re
import shutil
import subprocess
import tempfile

ROOT = Path(__file__).resolve().parents[1]
COMPONENTS = (
    "Model.js", "StorageLocationSelector.qml", "SelectField.qml", "Label.qml",
    "Design.qml", "PierImportPane.qml", "ActionButton.qml", "Field.qml",
    "CheckField.qml", "Hint.qml", "MaintenancePane.qml", "RoutinePane.qml",
    "RoutineCard.qml", "Card.qml", "CardTitle.qml", "StatusBadge.qml", "ImportRecoveryPane.qml",
)


def extract_method(source, name):
    match = re.search(r"^  function " + name + r"\([^\n]*\) \{[^\n]*\}$", source, re.M)
    if not match:
        match = re.search(r"^  function " + name + r"\([^\n]*\) \{[\s\S]*?^  \}", source, re.M)
    assert match, f"Missing shipped Service.{name} callback"
    return match[0]


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--require-qt", action="store_true", help="fail, rather than skip, when Qt's QML runtime is unavailable (CI)")
    args = parser.parse_args()
    executable = shutil.which(os.environ.get("QT_QML_EXECUTABLE", "qml6"))
    if not executable:
        if args.require_qt:
            parser.error("Qt 6 qml6 is required; install Qt declarative tools or set QT_QML_EXECUTABLE")
        print("SKIP: offscreen QML regression requires Qt 6 qml6; CI should use --require-qt")
        return
    spec = importlib.util.spec_from_file_location("panel_fixture", ROOT / "scripts/test-panel.py")
    fixture = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(fixture)
    with tempfile.TemporaryDirectory(prefix="omabit-rc11-offscreen-") as temporary:
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
        service = (ROOT / "plugins/omabit/Service.qml").read_text()
        methods = "\n".join(extract_method(service, name) for name in (
            "rememberInspection", "reconcileInspection", "rememberImport", "reconcileImports",
            "storageConnection", "importExternalPier", "recoverImport", "finish",
            "clearEpochPreview", "canMaintain", "canSaveRoutine", "canReviewTarget",
            "invalidateEpochPreview", "invalidateImportCatalogue",
        ))
        changed = re.search(r"^  onConnectionsChanged: .*$", service, re.M)
        assert changed, "Missing shipped connectionsChanged reconciliation"
        monitor = (ROOT / "plugins/omabit/MonitorWindow.qml").read_text()
        selections = []
        for name in ("fleet", "selected", "connection", "instance"):
            match = re.search(r"^  readonly property var " + name + r":.*$", monitor, re.M)
            assert match, f"Missing shipped MonitorWindow.{name} binding"
            selections.append(match[0].replace("root.", "test."))

        # G04: exercise the exact, explicitly configured collector fixture data in
        # the actual recovery component, with prerequisites checked before Qt loads.
        helper = fixture.write_fixture(directory, extra_features=["recover-import"], import_recovery=True)
        recovery = fixture.assert_fixture_capabilities(helper, {"recover-import", "pier-storage", "storage-locations"})
        env = dict(os.environ, OMABIT_FIXTURE_TIME=datetime.datetime.now(datetime.timezone.utc).isoformat())
        command = [str(helper), "--runner", "fixture", "--expected-runner-id", "runner-fixture", "operate", "alpha", "recover-import", "--revision", "2", "--request-id", "fixture-recovery"]
        refusal = subprocess.run(command, env=env, text=True, capture_output=True, timeout=5)
        assert refusal.returncode == 1
        assert json.loads(refusal.stderr)["error"]["code"] == "revision_conflict"
        acknowledgement = json.loads(subprocess.check_output(command, env=env, text=True, timeout=5))
        completed = fixture.assert_fixture_capabilities(helper, {"recover-import"})

        qml = (ROOT / "scripts/test-rc11-panel-offscreen.qml").read_text()
        for token, value in {
            "SERVICE_METHODS": methods, "CONNECTIONS_CHANGED": changed[0],
            "MONITOR_SELECTION": "\n".join(selections),
            "RECOVERY_INVENTORY": json.dumps(recovery), "RECOVERY_COMPLETED": json.dumps(completed),
            "RECOVERY_ACK": json.dumps(acknowledgement), "RECOVERY_REFUSAL": json.dumps(refusal.stderr),
        }.items():
            assert qml.count("@@" + token + "@@") == 1, token
            qml = qml.replace("@@" + token + "@@", value)
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
        result = subprocess.run([executable, "-I", str(directory), str(directory / "main.qml")], env=env, capture_output=True, text=True, timeout=25)
        text = result.stdout + result.stderr
        assert result.returncode == 0 and text.count("PASS:") == 4, text
        assert not any(error in text for error in ("Error:", "FAIL:", "Unable to assign", "Binding loop", "Cannot assign", "is not defined")), text
        for line in text.splitlines():
            if "PASS:" in line:
                print(line)


if __name__ == "__main__":
    main()
