"""Guard recovery of a disabled UI1 widget without changing live state."""
import importlib.util
import json
import os
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch

spec = importlib.util.spec_from_file_location('ui1', Path(__file__).with_name('update_review040_ui1.py'))
m = importlib.util.module_from_spec(spec)
spec.loader.exec_module(m)


class RecoveryGuards(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        root = Path(self.tmp.name)
        self.work, self.old, self.new, self.config = [root / n for n in ('work', 'old', 'new', 'shell.json')]
        for p in (self.old, self.work / 'plugin'):
            p.mkdir(parents=True)
            (p / 'payload').write_text(str(p))
        for context in (patch.multiple(m, WORK=self.work, OLD=self.old, NEW=self.new), patch.object(m.base, 'CONFIG', self.config)):
            context.start()
            self.addCleanup(context.stop)
        original = {'bar': {'layout': {'right': [{'id': 'other', 'setting': 7}, {'id': m.base.PLUGIN_ID}]}}, 'custom': {'keep': True}}
        disabled = {'bar': {'layout': {'right': [{'id': 'other', 'setting': 7}], 'left': [], 'centerBegin': [], 'center': [], 'centerEnd': [], 'tray': []}}, 'custom': {'keep': True}, 'plugins': []}
        self.original = json.dumps(original).encode()
        self.disabled = json.dumps(disabled).encode()
        self.config.write_bytes(self.disabled)
        (self.work / 'shell-before.json').write_bytes(self.original)
        (self.work / 'shell-disabled.json').write_bytes(self.disabled)
        m.base.write(self.work / 'plan.json', {'version': m.VERSION, 'script_sha256': m.FAILED_UPDATER_SHA,
            'base_script_sha256': m.base.sha(m.base.__file__), 'before': m.base.tree(self.old), 'candidate': m.base.tree(self.work / 'plugin')})
        m.base.write(self.work / 'apply-started.json', {'version': m.VERSION, 'old_plugin': str(self.old), 'new_plugin': str(self.new)})
        m.base.write(self.work / 'failed.json', {'phase': 'disable-dev-widget', 'error': 'shell settings changed while disabling the DEV widget'})

    def test_missing_sections_recover_without_writes(self):
        before = m.base.tree(Path(self.tmp.name))
        _, original, disabled = m.recovery_checkpoint()
        self.assertEqual((original, disabled), (self.original, self.disabled))
        self.assertEqual(m.base.tree(Path(self.tmp.name)), before)

    def test_unrelated_checkpoint_change_refused(self):
        changed = json.loads(self.disabled)
        changed['custom']['keep'] = False
        raw = json.dumps(changed).encode()
        (self.work / 'shell-disabled.json').write_bytes(raw)
        self.config.write_bytes(raw)
        with self.assertRaisesRegex(RuntimeError, 'unrelated shell'):
            m.recovery_checkpoint()

    def test_new_user_setting_refused(self):
        self.config.write_bytes(self.disabled + b'\n')
        with self.assertRaisesRegex(RuntimeError, 'live shell settings changed'):
            m.recovery_checkpoint()

    def test_partial_or_repeated_recovery_refused(self):
        (self.work / 'recovery-started.json').write_text('{}')
        with self.assertRaisesRegex(RuntimeError, 'cannot resume'):
            m.recovery_checkpoint()

    def test_changed_candidate_refused(self):
        (self.work / 'plugin/payload').write_text('unexpected')
        with self.assertRaisesRegex(RuntimeError, 'staged plugin changed'):
            m.recovery_checkpoint()

    def test_wrong_failure_phase_refused(self):
        (self.work / 'failed.json').write_text(json.dumps({'phase': 'activate-ui-preview', 'error': 'other'}))
        with self.assertRaisesRegex(RuntimeError, 'not the recoverable'):
            m.recovery_checkpoint()


class DeliveryLock(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.lock = Path(self.tmp.name) / 'delivery.lock'
        context = patch.object(m, 'LOCK', self.lock)
        context.start()
        self.addCleanup(context.stop)

    def test_missing_lock_is_created_and_reused_without_truncation(self):
        with m.delivery_lock():
            self.assertEqual(self.lock.stat().st_mode & 0o777, 0o600)
            inode = self.lock.stat().st_ino
        self.lock.write_bytes(b'retain')
        with m.delivery_lock():
            self.assertEqual(self.lock.stat().st_ino, inode)
            self.assertEqual(self.lock.read_bytes(), b'retain')

    def test_concurrent_holder_is_not_bypassed(self):
        with m.delivery_lock():
            with self.assertRaisesRegex(RuntimeError, 'another Omabit'):
                with m.delivery_lock():
                    self.fail('second lock admitted')
        with m.delivery_lock():
            pass

    def test_symlink_is_not_followed(self):
        target = self.lock.with_name('target')
        target.write_bytes(b'untouched')
        self.lock.symlink_to(target)
        with self.assertRaises(OSError):
            with m.delivery_lock():
                self.fail('symlink admitted')
        self.assertEqual(target.read_bytes(), b'untouched')

    def test_fifo_refuses_without_waiting(self):
        os.mkfifo(self.lock, 0o600)
        with self.assertRaisesRegex(RuntimeError, 'unsafe delivery lock'):
            with m.delivery_lock():
                self.fail('FIFO admitted')


if __name__ == '__main__':
    unittest.main()
