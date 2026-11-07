"""Actual receiver consumers; only Theme's desktop effects are inert in this lab."""
import contextlib
import copy
import json
import os
from pathlib import Path
import sys

assert Path.home() == Path('/lab/home') and not Path('/home').exists()
assert os.environ.get('OMABIT_REAL_RECONNECT') == 'private-v1'
root = Path(sys.argv[1])
sys.path.insert(0, str(root))
if root.name.startswith('urbit.dms.'):
    import bridge
    raise SystemExit(bridge.main())

from client import sync
from client.support import StateStore

# Enabling this isolated account is an explicit fixture choice, independent of
# granting an owner session. The receiver itself must not enable desktop sync.
accounts = StateStore()
with accounts.locked(wait=True):
    record = accounts.load()
    assert len(record['ships']) == 1 and not record['ships'][0]['automatic']
    record['ships'][0]['automatic'] = True
    accounts.save(record)

profile = json.loads(Path('/lab/profile.json').read_text())


class Ownership:
    def notification_current(self, notice):
        return False


class Desktop:
    ownership = Ownership()

    def session(self):
        return contextlib.nullcontext()

    def adopt_legacy(self, values):
        pass

    def capture(self, device):
        return copy.deepcopy(profile)

    def needs_repair(self):
        return False

    def apply(self, value):
        raise AssertionError('Unexpected desktop mutation in reconnect fixture')


class Checkpoints:
    def __init__(self, desktop):
        pass

    def assert_no_pending(self):
        pass


sync.Desktop = Desktop
sync.Checkpoints = Checkpoints
sync.ensure_hooks = lambda **kwargs: None
# Coordinator, Watch, Talon bridge, saved state, keyring and HTTP are unchanged.
raise SystemExit(sync.main() if sys.argv[2:] else sync.serve())
