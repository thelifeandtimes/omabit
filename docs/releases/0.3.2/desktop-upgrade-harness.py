#!/usr/bin/env python3
"""Extend the frozen desktop fixture with saved preferences and a failed candidate.

No host state, services, IPC or ships are used. The failure is synthetic; it does
not claim reproduction of the user's long-lived shell's type-resolution failure.
Pass the same previous/target asset arguments as test-desktop-bootstrap-qml.py.
"""
from pathlib import Path
from configparser import ConfigParser

def preferences(path):
    config=ConfigParser()
    config.read_string(path.read_text())
    return dict(config.items("General"))

repo=Path(__file__).resolve().parents[3]
path=repo/'scripts/test-desktop-bootstrap-qml.py'
source=path.read_text()
anchor='''                        assert loaded["diagnostic"]["loaded_version"] == document["version"]'''
addition=anchor+'''
                        if assets == args.previous_application_assets:
                            ipc("view", "setup", "runner-isolated/preserved-selection")
                            before_preference = Path(loaded["environment"]["XDG_STATE_HOME"]) / "omabit/panel.ini"
                            before_values = preferences(before_preference)
                            assert before_values["desktopselection"] == "runner-isolated/preserved-selection"
                            # Exercise recovery from an explicitly broken type in this
                            # actual profile; this is not the observed host-only cause.
                            failed_meta, failed_archive, _ = fixtures.bundle(root / "failed-actual-profile", "0.3.1", qml=b"import QtQuick\\nItem { MissingServiceUpgradePane {} }")
                            with patch.dict(os.environ, env, clear=True):
                                failed = desktop.import_release(failed_meta, failed_archive)["release"]
                                desktop.select("application", failed)
                            retained = wait(lambda r: r["diagnostic"]["requested_release"] == failed and r["diagnostic"]["loaded_release"] == actual and bool(r["diagnostic"]["error"]) and r["widget_ready"])
                            assert all(preferences(before_preference).get(k) == v for k,v in before_values.items()), (before_values, preferences(before_preference))
                            print("PASS: actual previous desktop remains usable after synthetic type failure")
                        elif args.previous_application_assets:
                            assert loaded["view"] == {"page":"setup", "selection":"runner-isolated/preserved-selection"}, loaded
                            assert all(preferences(before_preference).get(k) == v for k,v in before_values.items()), (before_values, preferences(before_preference))
                            assert not loaded["diagnostic"]["error"], loaded
                            print("PASS: target activates after failure and preserves actual prior desktop preferences")'''
assert source.count(anchor)==1
exec(compile(source.replace(anchor,addition),str(path),'exec'),{'__file__':str(path),'__name__':'__main__','preferences':preferences})
