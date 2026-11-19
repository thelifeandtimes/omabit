import contextlib
import configparser
import importlib.util
import io
import json
import re
import sqlite3
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest
from unittest.mock import patch


def module(name):
    spec = importlib.util.spec_from_file_location('runtime_cow_'+name, Path(__file__).with_name(name+'.py'))
    value = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(value)
    return value


class RecoveryGuards(unittest.TestCase):
    def test_frozen_boot_refuses_wrong_bytes_before_provenance_probe(self):
        host=module('run')
        with tempfile.TemporaryDirectory(prefix='omabit-cow-boot-pin-') as root:
            init=Path(root)/'init';init.write_bytes(b'unqualified')
            with patch.object(host.subprocess,'check_output',side_effect=AssertionError('wrong bytes must refuse first')):
                with self.assertRaisesRegex(RuntimeError,'boot helper or source differs'):host.frozen_boot(init)

    def test_cloned_store_readers_close_on_success_and_schema_refusal(self):
        guest=module('guest')
        connect=sqlite3.connect
        with tempfile.TemporaryDirectory(prefix='omabit-cow-read-lifetime-') as root:
            path=Path(root)/'fixture.sqlite'
            with contextlib.closing(connect(path)) as db:
                db.execute('pragma user_version=65')
                for table in ('grubbery_app_operations','grubbery_app_events','grubbery_app_leases'):
                    db.execute('create table '+table+' (id text)')
                db.execute('create table instances (body text)')
                db.execute('insert into instances values (?)',(json.dumps({'public':{'id':'grubbery-zod','ship':'~zod'},'runtime':{'prepared':True,'initialized':True,'storage':{'fixture':'only'}}}),))
                db.commit()
            readers=[]
            def read_only(filename,**kwargs):
                self.assertEqual(filename,'file:/home/review/.local/share/urbit-runner/managed/runner.sqlite?mode=ro')
                self.assertEqual(kwargs,{'uri':True})
                db=connect('file:'+str(path)+'?mode=ro',uri=True);readers.append(db);self.addCleanup(db.close);return db
            with patch.object(guest.sqlite3,'connect',side_effect=read_only):
                self.assertEqual(len(guest.original_rows(65)),3)
                with self.assertRaisesRegex(RuntimeError,'unexpected task store schema'):guest.original_rows(63)
                self.assertEqual(len(guest.initialized_fixture()),64)
            self.assertEqual(len(readers),3)
            for db in readers:
                with self.assertRaises(sqlite3.ProgrammingError):db.execute('select 1')

    def test_artifact_boundary_rejects_embedded_and_other_runtime_inputs(self):
        host = module('run')
        for path in ('internal/service/legacy-inventory.json', 'internal/appdev/samples/grubbery-state-v2/nex/state.hoon', 'contents/ui/Panel.qml', 'go.mod', 'go.sum', 'internal/managed/server.go', 'docs/example.go', 'docs/../internal/service/legacy-inventory.json', '/docs/result.json'):
            self.assertFalse(host.artifact_change_allowed(path), path)
        for path in ('docs/diagnostics/result.json', 'docs/development-memory.md', 'scripts/qualification/grubbery-runtime-cow/run.py', 'scripts/qualification/grubbery-runtime-cow/guest.py', 'scripts/qualification/grubbery-runtime-cow/test_guards.py', 'scripts/qualification/grubbery-runtime-cow/README.md'):
            self.assertTrue(host.artifact_change_allowed(path), path)

    def test_changed_embedded_inventory_refuses_before_artifact_reads_or_admission(self):
        host = module('run')
        args = ['run.py', '--run', '--artifact-source', 'a'*40, '--output', '/tmp/omabit-issues-runtime-cow-99']
        for key in ('portable', 'candidate', 'rootless', 'runtime', 'helper', 'init'):
            args += ['--'+key, '/missing']

        def git(command, **kwargs):
            if command == ['git', 'status', '--porcelain']:
                return b''
            if command == ['git', 'rev-parse', 'HEAD']:
                return 'b'*40+'\n'
            if command == ['git', 'diff', '--name-only', 'a'*40]:
                return 'internal/service/legacy-inventory.json\n'
            raise AssertionError('unexpected command before refusal')

        with patch.object(sys, 'argv', args), patch.object(host.os, 'geteuid', return_value=1000), patch.object(Path, 'exists', return_value=False), patch.object(host.subprocess, 'check_output', git), patch.object(host.subprocess, 'run') as ancestry, patch.object(host, 'ordinary', side_effect=AssertionError('no artifact read')), patch.object(host.budget, 'charge', side_effect=AssertionError('no storage admission')):
            with self.assertRaisesRegex(RuntimeError, 'runtime or embedded inputs changed'):
                host.main()
            self.assertEqual(ancestry.call_args.args[0], ['git', 'merge-base', '--is-ancestor', 'a'*40, 'b'*40])
            self.assertEqual(ancestry.call_count, 1)

    def test_outer_deadlines_leave_time_for_inner_failure_and_cleanup(self):
        host, guest = module('run'), module('guest')
        self.assertGreater(guest.HELPER_PROCESS_TIMEOUT_SECONDS, 90*60)
        self.assertGreater(guest.HELPER_DEADLINE_SECONDS, guest.HELPER_PROCESS_TIMEOUT_SECONDS)
        self.assertGreater(host.VM_DEADLINE_SECONDS, guest.HELPER_DEADLINE_SECONDS+120)
        self.assertGreater(host.SUPERVISOR_DEADLINE_SECONDS, host.VM_DEADLINE_SECONDS+120)

    def test_actual_guest_service_deadline_encloses_helper_and_final_checks(self):
        host, guest = module('run'), module('guest')
        source = (Path(__file__).parent/'init/main.go').read_text()
        # Inspect the actual emitted systemd unit, not another Python copy of
        # its bound. An earlier 2400s unit killed the parent while its runuser
        # helper continued in a separate guest user session and later passed.
        unit = re.search(r'service := `([^`]+)`', source)
        self.assertIsNotNone(unit)
        config = configparser.ConfigParser(interpolation=None)
        config.read_string(unit.group(1))
        self.assertEqual(config['Service']['Type'], 'oneshot')
        start = config.getint('Service', 'TimeoutStartSec')
        self.assertGreater(start, guest.HELPER_DEADLINE_SECONDS+120)
        self.assertGreater(host.VM_DEADLINE_SECONDS, start+60)

    def test_no_run_has_no_source_storage_or_process_effect(self):
        host = module('run')
        with patch.object(sys, 'argv', ['run.py']), patch.object(host.subprocess, 'check_output', side_effect=AssertionError('no process')), patch.object(host.budget, 'charge', side_effect=AssertionError('no admission')), contextlib.redirect_stdout(io.StringIO()):
            host.main()

    def test_existing_capsule_is_not_an_output(self):
        host = module('run')
        args = ['run.py', '--run', '--output', '/tmp/omabit-issues-app-vm-05']
        for key in ('portable', 'candidate', 'rootless', 'runtime', 'helper'):
            args += ['--'+key, '/missing']
        with patch.object(sys, 'argv', args), patch.object(host.os, 'geteuid', return_value=1000), patch.object(host.subprocess, 'check_output', side_effect=AssertionError('refuse before inputs')), contextlib.redirect_stderr(io.StringIO()):
            with self.assertRaises(SystemExit) as stopped:
                host.main()
        self.assertEqual(stopped.exception.code, 2)

    def test_guest_refuses_host_wrong_predecessor_case_and_schema(self):
        guest = module('guest')
        with patch.object(guest.os, 'geteuid', return_value=1000), patch.object(Path, 'read_text', side_effect=AssertionError('host read forbidden')):
            with self.assertRaises(RuntimeError):
                guest.guard()
        scenario = {'kind': 'issues18-product-grubbery-runtime-v1', 'case': guest.CASE, 'predecessor_root_sha256': guest.PIN, 'original_schema': 62, 'target_schema': 65, 'development_snapshot': True}
        values = {'/sys/class/dmi/id/sys_vendor': 'QEMU', '/proc/cmdline': 'omabit.disposable-guest=release-upgrade-v1', '/opt/omabit-release-input/scenario.json': json.dumps(scenario)}
        with patch.object(guest.os, 'geteuid', return_value=0), patch.object(sys, 'argv', ['guest.py']), patch.object(Path, 'read_text', lambda path, **kwargs: values[str(path)]):
            self.assertEqual(guest.guard(), scenario)
            for change in ({'predecessor_root_sha256': '0'*64}, {'case': 'cow-fresh-nec-v1'}, {'original_schema': 63}, {'target_schema': 62}, {'target_schema': 63}, {'development_snapshot': False}):
                values['/opt/omabit-release-input/scenario.json'] = json.dumps({**scenario, **change})
                with self.assertRaises(RuntimeError):
                    guest.guard()

    def test_compressed_input_refuses_wrong_bytes_in_fresh_private_output(self):
        import gzip
        guest = module('guest')
        with tempfile.TemporaryDirectory() as root:
            path = Path(root)
            (path/'input/candidate').mkdir(parents=True)
            (path/'input/candidate/omabit.gz').write_bytes(gzip.compress(b'wrong'))
            with patch.object(guest, 'I', path/'input'), patch.object(guest, 'R', path/'runtime'):
                with self.assertRaises(RuntimeError):
                    guest.inflate({'candidate': {'omabit': {'bytes': 5, 'sha256': '0'*64}}})
                self.assertEqual((path/'runtime/candidate/omabit').read_bytes(), b'wrong')

    def test_actual_kernel_file_limit_refuses_growth_and_preserves_first_bytes(self):
        host = module('run')
        self.assertLess(host.ROOT_FILE_MAX+64*1024**2+32*1024**2+16*1024**2, host.SCRATCH)
        self.assertEqual(host.SCRATCH, 896*1024**2)
        with tempfile.TemporaryDirectory() as root:
            target = Path(root)/'bounded-disposable-file'
            code = "import errno,pathlib,sys; p=pathlib.Path(sys.argv[1]); f=p.open('xb',buffering=0); f.write(b'A'*1048576)\ntry: f.write(b'B')\nexcept OSError as e: assert e.errno==errno.EFBIG\nelse: raise RuntimeError('kernel file bound was ignored')\nf.close(); assert p.stat().st_size==1048576 and p.read_bytes()==b'A'*1048576"
            result = subprocess.run(['/usr/bin/prlimit', '--fsize=1048576:1048576', '--', sys.executable, '-B', '-c', code, str(target)], capture_output=True, text=True, timeout=10)
            self.assertEqual(result.returncode, 0, result.stderr)


if __name__ == '__main__':
    unittest.main()
