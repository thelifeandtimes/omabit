import contextlib
import hashlib
import importlib.util
import io
import json
from pathlib import Path
import struct
import sys
import tempfile
import unittest
from unittest.mock import patch


def module(name):
    spec = importlib.util.spec_from_file_location('pressure_'+name, Path(__file__).with_name(name+'.py'))
    value = importlib.util.module_from_spec(spec); spec.loader.exec_module(value); return value


class PressureGuards(unittest.TestCase):
    def test_no_run_is_inert(self):
        host = module('run')
        with patch.object(sys, 'argv', ['run.py']), patch.object(host.subprocess, 'check_output', side_effect=AssertionError('process')), patch.object(host.budget, 'charge', side_effect=AssertionError('storage')), contextlib.redirect_stdout(io.StringIO()): host.main()

    def test_host_guest_refuses_before_paths(self):
        guest = module('guest')
        with patch.object(guest.os, 'geteuid', return_value=1000), patch.object(Path, 'read_text', side_effect=AssertionError('host paths')):
            with self.assertRaisesRegex(RuntimeError, 'refused outside'): guest.guard()

    def test_foreign_scenario_refuses(self):
        guest = module('guest')
        scenario = {'kind': 'issues20-recording-pressure-v1', 'target_schema': 65, 'development_snapshot': True}
        values = {'/sys/class/dmi/id/sys_vendor': 'QEMU\n', '/proc/cmdline': 'omabit.disposable-guest=release-upgrade-v1', str(guest.I/'scenario.json'): json.dumps(scenario)}
        with patch.object(guest.os, 'geteuid', return_value=0), patch.object(sys, 'argv', ['guest.py']), patch.object(Path, 'read_text', lambda p: values[str(p)]):
            self.assertEqual(guest.guard(), scenario)
            for key, value in [('kind', 'issues20-recording-policy-upgrade-v1'), ('target_schema', 64), ('development_snapshot', False)]:
                values[str(guest.I/'scenario.json')] = json.dumps({**scenario, key: value})
                with self.assertRaises(RuntimeError): guest.guard()

    def test_decoder_checks_chain_identity_torn_tail_and_original_bytes(self):
        guest = module('guest')
        with tempfile.TemporaryDirectory(prefix='omabit-pressure-decoder-') as d:
            root = Path(d); metadata = b'{"session_id":"11111111111111111111111111111111"}'
            (root/'metadata.json').write_bytes(metadata)
            header = b'OMABIT-RECORDING/1\0'+bytes.fromhex('11'*16)+hashlib.sha256(metadata).digest()+struct.pack('>QQq', 1, 1, 1)+bytes(32)
            previous = hashlib.sha256(header).digest(); body = bytearray(header)
            for seq, kind, data in [(1, 1, b'original\x00\xff'), (2, 2, b'stderr\n')]:
                frame = struct.pack('>QqQBI', seq, seq+1, seq, kind, len(data)); previous = hashlib.sha256(previous+frame+data).digest(); body.extend(frame+data+previous)
            p = root/'frames.obr'; p.write_bytes(body)
            result = guest.decode_session(root)
            self.assertEqual(result['stdout_sha256'], hashlib.sha256(b'original\x00\xff').hexdigest()); self.assertEqual(result['frames'], 2)
            for bad in [bytes(body[:-1]), bytes(body[:18])+bytes(16)+bytes(body[34:]), bytes(body[:-1])+bytes([body[-1]^1])]:
                p.write_bytes(bad)
                with self.assertRaises(RuntimeError): guest.decode_session(root)

    def test_emitter_refuses_foreign_row_before_exec(self):
        from types import SimpleNamespace
        guest = module('guest')
        for row in (None, {'ship':'~nec','id':'recording-zod'}, {'ship':'~zod','id':'recording-zod','project_id':'foreign'}, {'ship':'~zod','id':'recording-zod'}):
            guest.common = SimpleNamespace(instance=lambda *args: row, cid=lambda value: 'foreign', docker=lambda *a, **k: self.fail('foreign exec'))
            with self.assertRaises(RuntimeError): guest.emit('selected')

    def test_capacity_refusal_uses_native_inventory_and_exact_failed_request(self):
        guest = module('guest')
        receipt = {'status': 'failed', 'request_id': 'pressure-capacity-refusal', 'message': 'runtime action failed: recording admission budget is exhausted or cannot be established; preserve saved jobs and inspect recording capacity'}
        jobs = {'jobs': {'one': {}, 'two': {}, 'three': {}}}
        self.assertTrue(guest.capacity_refusal_preserved(jobs, jobs, [], [], receipt))
        self.assertFalse(guest.capacity_refusal_preserved(jobs, {'jobs': {**jobs['jobs'], 'new': {}}}, [], [], receipt))
        self.assertFalse(guest.capacity_refusal_preserved(jobs, jobs, [], ['unexpected'], receipt))
        self.assertFalse(guest.capacity_refusal_preserved(jobs, jobs, [], [], {**receipt, 'message': 'transport unavailable'}))
        self.assertFalse(guest.capacity_refusal_preserved(jobs, jobs, [], [], {**receipt, 'request_id': 'other'}))

    def test_real_job_layout_separates_lock_and_preserves_contents(self):
        guest = module('guest')
        with tempfile.TemporaryDirectory(prefix='omabit-pressure-inventory-') as d:
            root = Path(d); (root/'admission.lock').touch()
            for name in ('1'*32, '2'*32, '3'*32):
                (root/name).mkdir(); (root/name/'worker.lock').touch()
                for leaf in ('job.json', 'claim.json', 'session.json', 'ready.json', 'result.json'):
                    (root/name/leaf).write_text('{}')
            before = guest.job_inventory(root)
            self.assertEqual(len(before['jobs']), 3)
            self.assertEqual(before['admission_lock_sha256'], hashlib.sha256(b'').hexdigest())
            self.assertEqual(before['jobs']['1'*32]['worker.lock'], hashlib.sha256(b'').hexdigest())
            (root/('1'*32)/'worker.lock').write_text('invalid')
            with self.assertRaisesRegex(RuntimeError, 'unexpected recording job file'): guest.job_inventory(root)
            (root/('1'*32)/'worker.lock').write_bytes(b'')
            (root/('1'*32)/'job.json').write_text('{"changed":true}')
            self.assertNotEqual(guest.job_inventory(root), before)
            (root/'admission.lock').write_text('bad')
            with self.assertRaisesRegex(RuntimeError, 'admission lock'): guest.job_inventory(root)
            (root/'admission.lock').unlink(); (root/'admission.lock').symlink_to(root/('1'*32)/'job.json')
            with self.assertRaisesRegex(RuntimeError, 'admission lock'): guest.job_inventory(root)

    def test_issues_budget_refuses_estimated_measurement(self):
        budget = module('run').budget
        with patch.object(budget.storage, 'roots', return_value=[]), patch.object(budget, 'worktree_build_roots', return_value=[budget.ROOT/'build']), patch.object(Path, 'is_dir', return_value=True), patch.object(Path, 'is_symlink', return_value=False), patch.object(budget.storage, 'measure', return_value={'allocated_bytes': 0, 'unmeasured_reserve': 1024**3}), patch.object(budget.storage, 'admit', side_effect=AssertionError('incomplete admission')):
            with self.assertRaisesRegex(RuntimeError, 'complete namespace measurement'):
                budget.charge(0, 48*1024**3)

    def test_native_limits_and_final_storage_are_bounded(self):
        guest = module('guest'); host = Path(__file__).with_name('run.py').read_text(); source = Path(__file__).with_name('guest.py').read_text()
        self.assertEqual(len(guest.STDOUT), 1572921)
        self.assertIn('head -c 1572864 /dev/zero', guest.EMIT)
        self.assertIn('"$i" -lt 64', guest.EMIT)
        self.assertIn('fill < 160*MIB', source)
        for gate in ('budget.charge(scratch, cap)', 'fcntl.LOCK_EX|fcntl.LOCK_NB', "'-nic', 'none'", "'MemorySwapMax=0'", '2500'):
            self.assertIn(gate, host)


if __name__ == '__main__': unittest.main()
