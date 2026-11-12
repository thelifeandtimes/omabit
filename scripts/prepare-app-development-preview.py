#!/usr/bin/env python3
"""Create a fresh, self-contained manual policy preview; preserve existing installs."""
import argparse
import hashlib
import json
import os
import shutil
import subprocess
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
COMPONENTS = tuple(p.name for p in sorted((ROOT / "plugins/omabit").iterdir())
                   if p.suffix in (".qml", ".js", ".svg") and p.name not in ("Service.qml", "Panel.qml", "WindowActivation.qml"))



def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--output", type=Path, required=True)
    parser.add_argument("--desktop-entry", type=Path, help="optional new, separately named .desktop launcher")
    args = parser.parse_args()
    output = args.output.absolute()
    if output.exists() or output.is_symlink():
        parser.error("Preview destination already exists; refusing to overwrite manual-review state")
    if args.desktop_entry and (args.desktop_entry.exists() or args.desktop_entry.is_symlink()):
        parser.error("Desktop launcher already exists; refusing to replace it")
    os.umask(0o077)
    output.mkdir(mode=0o700, parents=True)
    (output / "Plugin").mkdir()
    for name in COMPONENTS:
        shutil.copy2(ROOT / "plugins/omabit" / name, output / "Plugin" / name)
    # Use the shipped registry, so a preview cannot hide missing native types.
    registry = (ROOT / "plugins/omabit/qmldir").read_text().splitlines()
    qml = {name for name in COMPONENTS if name.endswith(".qml")} | {"WindowActivation.qml"}
    declared = {line.split()[-1] for line in registry if line.strip()}
    if not qml <= declared:
        raise ValueError("Preview components are missing from the native QML registry: " + str(sorted(qml - declared)))
    (output / "Plugin/qmldir").write_text("\n".join(line for line in registry if line.strip() and line.split()[-1] in qml) + "\n")
    # Focus only this fixture window, without exposing compositor control IPC.
    (output / "Plugin/WindowActivation.qml").write_text('import QtQuick\nItem { required property var targetWindow;signal failed();function activate(){targetWindow.visible=true;targetWindow.requestActivate()} }\n')
    for name in ("backend.py", "launch.py", "shell.qml", "PreviewService.qml"):
        shutil.copy2(ROOT / "scripts/app-development-preview" / name, output / name)
    commons = output / "Commons"
    commons.mkdir()
    (commons / "qmldir").write_text("singleton Style 1.0 Style.qml\nsingleton Color 1.0 Color.qml\n")
    (commons / "Style.qml").write_text('pragma Singleton\nimport QtQuick\nQtObject { function space(n) {return Math.round(n)} property var font: ({family:"monospace",caption:10,bodySmall:11,body:12}) }\n')
    (commons / "Color.qml").write_text('''pragma Singleton
import QtQuick
QtObject {
  property color background:"#101315"
  property color foreground:"#cacccc"
  property color accent:"#94b4a4"
  property color urgent:"#cc7777"
}
''')
    for name in ("home", "config", "state", "share", "cache"):
        (output / "data" / name).mkdir(parents=True, mode=0o700)
    source = subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=ROOT, text=True).strip()
    hashes = {str(path.relative_to(output)): hashlib.sha256(path.read_bytes()).hexdigest()
              for path in sorted(output.rglob("*")) if path.is_file()}
    (output / "source.json").write_text(json.dumps({"base_commit": source, "files": hashes}, indent=2) + "\n")
    shutil.copy2(ROOT / "docs/app-development-preview.md", output / "README.md")
    if args.desktop_entry:
        # Desktop Exec has its own escaping rules, distinct from shell quoting.
        path = str(output / "launch.py")
        escaped = path.replace("\\", "\\\\").replace('"', '\\"').replace("`", "\\`").replace("$", "\\$").replace("%", "%%")
        args.desktop_entry.parent.mkdir(parents=True, exist_ok=True)
        with args.desktop_entry.open("x") as launcher:
            launcher.write('[Desktop Entry]\nType=Application\nName=Omabit 0.4.0 Preview\n'
                           'Comment=Isolated Setup review with simulated runners\n'
                           f'Exec=/usr/bin/python3 -B "{escaped}" start\n'
                           'Icon=applications-development\nTerminal=false\nCategories=Development;\n')
    print(output)


if __name__ == "__main__":
    main()
