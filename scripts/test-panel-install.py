#!/usr/bin/env python3
"""Verify snapshot replacement refuses local changes and unrelated files."""
import contextlib
import importlib.util
import io
import pathlib
import tempfile

spec = importlib.util.spec_from_file_location("panel_install", pathlib.Path(__file__).with_name("install-panel.py"))
installer = importlib.util.module_from_spec(spec)
spec.loader.exec_module(installer)

with tempfile.TemporaryDirectory(prefix="omabit-install-test-") as directory:
    destination = pathlib.Path(directory) / "plugin"
    with contextlib.redirect_stdout(io.StringIO()):
        installer.install(destination)
        installer.install(destination, replace=True)
    original = (destination / "Model.js").read_text()
    (destination / "Model.js").write_text(original + "\n// user change\n")
    try:
        installer.install(destination, replace=True)
        raise AssertionError("overwrote user work")
    except RuntimeError:
        assert (destination / "Model.js").read_text().endswith("// user change\n")
    (destination / "Model.js").write_text(original)
    (destination / "extra.txt").write_text("unrelated user file")
    try:
        installer.install(destination, replace=True)
        raise AssertionError("removed unrelated file")
    except RuntimeError:
        assert (destination / "extra.txt").read_text() == "unrelated user file"
print("PASS: snapshot install/update, local modifications preserved, unrelated files preserved")
