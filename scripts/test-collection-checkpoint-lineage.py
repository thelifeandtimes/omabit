#!/usr/bin/env python3
"""Read actual retained legacy checkpoints with bounded, non-leasing ancestry."""
import copy
import importlib.util
import json
from pathlib import Path
import sys
import unittest
from unittest.mock import patch

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT/'native/collections'))
import checkpoint_lineage as lineage
import journal as j
import runtime_authority as authority
import session_successor as successor

spec = importlib.util.spec_from_file_location('ancestry_fixture', Path(__file__).with_name('test-collection-successor-checkpoint.py'))
fixture = importlib.util.module_from_spec(spec)
spec.loader.exec_module(fixture)


class LineageTests(unittest.TestCase):
    def setUp(self):
        fixture.LinkedCheckpointTests.setUp(self)
        self.clean_linked()
        self.saved_checkpoint = self.capture_linked()
        self.head_reference = successor.reference(self.linked_checkpoint_path())

    def read_chain(self, **kwargs):
        return lineage.read(self.head_reference, self.store.root, **kwargs)

    def refuse_chain(self, message=None, **kwargs):
        before = self.snapshot()
        with self.assertRaisesRegex((ValueError, OSError), message or '.'):
            self.read_chain(**kwargs)
        self.assertEqual(before, self.snapshot())

    def test_original_and_linked_records_match_legacy_readers(self):
        before = self.snapshot()
        value = self.read_chain()
        self.assertEqual(value['head'], self.read_linked())
        self.assertEqual(value['records'], [self.saved_checkpoint, self.archived])
        self.assertEqual(value['references'], [self.head_reference, self.saved_checkpoint['ancestor_checkpoint']])
        self.assertEqual((value['events'], value['state']), self.store._load())
        self.assertEqual(value['usage']['nodes'], 2)
        first = lineage.read(value['references'][1], self.store.root)
        self.assertEqual(first['head'], self.archived)
        self.assertEqual(first['usage']['nodes'], 1)
        self.assertEqual(before, self.snapshot())
        if ROOT == Path('/lab/source'):
            fixture.save(Path('/lab/checkpoint-lineage-result.json'), {
                'passed': True, 'legacy_formats': [1, 2], 'usage': value['usage'],
                'canonical_events': len(value['events']), 'read_only': True,
                'new_persistent_format': False, 'runtime_activation': False})

    def test_existing_lease_does_not_require_a_second_lease(self):
        with self.store.lease():
            with patch.object(j.Store, 'lease', side_effect=AssertionError('second lease')):
                self.assertEqual(self.read_chain()['head'], self.saved_checkpoint)

    def test_retired_runtimes_are_never_opened(self):
        self.lose_linked_runtime()
        original = lineage.inspection.Reader.read
        def only_canonical(reader, path, **kwargs):
            self.assertEqual(Path(path).parent, self.store.root)
            return original(reader, path, **kwargs)
        with patch.object(lineage.inspection.Reader, 'read', only_canonical), \
             patch.object(authority.Authority, '__init__', side_effect=AssertionError('live authority')):
            self.assertEqual(self.read_chain()['head'], self.saved_checkpoint)

    def test_later_atomic_journal_appends_are_accepted(self):
        first = self.read_chain()
        changed = copy.deepcopy(self.next_spec)
        for role in j.ROLES:
            changed['members'][role]['artifact_sha256'] = j.digest(['lineage-append', role])
        before_inode = (self.store.root/'journal.json').stat().st_ino
        self.record({'type': 'register', 'spec': changed})
        self.assertNotEqual(before_inode, (self.store.root/'journal.json').stat().st_ino)
        after = self.read_chain()
        self.assertEqual(first['records'], after['records'])
        self.assertEqual(after['events'][:-1], first['events'])

    def test_missing_parent_is_preserved_as_a_refusal(self):
        self.ancestor_checkpoint.unlink()
        self.refuse_chain()

    def test_equal_parent_bytes_with_another_inode_are_refused(self):
        path = self.ancestor_checkpoint
        raw = path.read_bytes()
        path.rename(path.with_suffix('.retained'))
        path.write_bytes(raw)
        path.chmod(0o600)
        self.refuse_chain()

    def test_missing_canonical_metadata_refuses_even_after_retirement(self):
        self.lose_linked_runtime()
        self.record_path().unlink()
        self.refuse_chain()

    def test_changed_canonical_prefix_is_refused(self):
        path = self.store.root/'journal.json'
        value = json.loads(path.read_text())
        value['events'] = value['events'][:1]
        fixture.save(path, value)
        self.refuse_chain('canonical prefix changed')

    def test_unknown_checkpoint_format_is_preserved(self):
        path = self.linked_checkpoint_path()
        value = copy.deepcopy(self.saved_checkpoint)
        value['format'] = 99
        fixture.save(path, fixture.redigest(value))
        self.head_reference = successor.reference(path)
        self.refuse_chain('unknown checkpoint ancestry format')

    def test_other_coordinator_is_refused_before_checkpoint_read(self):
        other = j.Store(self.root/'foreign-coordinator', create=True)
        try:
            with other.lease():
                pass
            with patch.object(lineage._Evidence, 'bound', side_effect=AssertionError('read foreign checkpoint')):
                with self.assertRaisesRegex(ValueError, 'outside its coordinator'):
                    lineage.read(self.head_reference, other.root)
        finally:
            other.close()

    def test_node_limit_refuses_before_reading_parent(self):
        original = lineage._Evidence.bound
        parent = self.saved_checkpoint['ancestor_checkpoint']['object']['path']
        def no_parent(reader, reference, **kwargs):
            self.assertNotEqual(reference['object']['path'], parent)
            return original(reader, reference, **kwargs)
        with patch.object(lineage._Evidence, 'bound', no_parent):
            self.refuse_chain('node limit', limits=lineage.Limits(nodes=1))

    def test_exact_file_and_byte_limits_succeed(self):
        usage = self.read_chain()['usage']
        limits = lineage.Limits(nodes=2, files=usage['files'],
                                bytes=usage['immutable_bytes'] + usage['journal_byte_reserve'])
        self.assertEqual(self.read_chain(limits=limits)['usage'], usage)

    def test_one_less_file_or_byte_refuses_without_writing(self):
        usage = self.read_chain()['usage']
        self.refuse_chain('file limit', limits=lineage.Limits(files=usage['files']-1))
        self.refuse_chain('byte limit', limits=lineage.Limits(bytes=usage['immutable_bytes']+j.MAX_BYTES-1))

    def test_budget_refusal_happens_before_json_allocation(self):
        with patch.object(lineage.inspection.Reader, 'read', side_effect=AssertionError('allocated body')):
            self.refuse_chain('byte limit', limits=lineage.Limits(bytes=j.MAX_BYTES))

    def test_repeated_reference_is_refused_before_second_read(self):
        # The real legacy format cannot encode a cycle. Inject the traversal
        # boundary alone to prove future formats cannot cause an unbounded walk.
        original = lineage._node
        count = []
        def cycle(record):
            count.append(record['id'])
            name, _ = original(record)
            return name, self.head_reference
        with patch.object(lineage, '_node', cycle):
            self.refuse_chain('repeated checkpoint ancestry node')
        self.assertEqual(count, [self.saved_checkpoint['id']])

    def test_conflicting_cached_reference_is_refused(self):
        reader = lineage._Evidence(lineage.Limits())
        reader.bound(self.head_reference, maximum=lineage.first.MAX_BYTES)
        changed = copy.deepcopy(self.head_reference)
        changed['sha256'] = '0'*64
        with self.assertRaisesRegex(ValueError, 'conflicting checkpoint ancestry reference'):
            reader.bound(changed, maximum=lineage.first.MAX_BYTES)

    def test_final_verification_catches_canonical_change(self):
        original = lineage.inspection.Reader.verify
        target = self.cleanup_path()
        def change_then_verify(reader):
            target.write_bytes(b'{}\n')
            return original(reader)
        with patch.object(lineage.inspection.Reader, 'verify', change_then_verify):
            with self.assertRaises(ValueError):
                self.read_chain()

    def test_missing_lock_is_never_created(self):
        (self.store.root/'coordinator.lock').unlink()
        self.refuse_chain()


for name in dir(fixture.LinkedCheckpointTests):
    value = getattr(fixture.LinkedCheckpointTests, name)
    if callable(value) and not name.startswith(('test', '_')) and not hasattr(LineageTests, name):
        setattr(LineageTests, name, value)


class LimitTests(unittest.TestCase):
    def test_limits_cannot_expand_or_disable_the_safety_bound(self):
        for values in ({'nodes': 0}, {'nodes': True}, {'nodes': lineage.MAX_NODES+1},
                       {'files': -1}, {'files': lineage.MAX_FILES+1},
                       {'bytes': lineage.MAX_BYTES+1}, {'bytes': 0}):
            with self.subTest(values=values), self.assertRaises(ValueError):
                lineage.Limits(**values).check()


if __name__ == '__main__':
    unittest.main()
