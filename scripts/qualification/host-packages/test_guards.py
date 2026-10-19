"""CI-safe guard/CLI tests: no root, native package tools, network or VM required."""
import contextlib
import hashlib
import io
import json
from pathlib import Path
import stat
import subprocess
import sys
import tempfile
import unittest
from unittest import mock

sys.path.insert(0, str(Path(__file__).parent))
import guards
import qualify

SHA = hashlib.sha256(b'\x7fELFtest candidate, never executed').hexdigest()
UUID = '12345678-1234-1234-1234-123456789abc'


def arguments(action='validate', provider='apt'):
    args = [action, '--hostname', 'hostcompat-qual-ci', '--vm-uuid', UUID,
            '--workspace', '/opt/hostcompat-qualification', '--candidate', '/opt/inputs/runner',
            '--candidate-sha256', SHA, '--candidate-version', 'development-ci',
            '--candidate-schema', '21', '--hooks', '/opt/inputs/hooks', '--provider', provider]
    if provider != 'apt':
        args += ['--pacman-root', '/opt/hostcompat-qualification/arch']
    return args


class FakeProbe:
    """Only synthetic observations; mutations/native tools are never mocked as success."""
    def __init__(self, provider='apt'):
        self.calls = []
        self.fail = None
        self.host = 'hostcompat-qual-ci'
        self.uuid = UUID
        self.candidate = b'\x7fELFtest candidate, never executed'
        self.data = {'schema': 1, 'purpose': guards.PURPOSE, 'hostname': self.host,
                     'vm_uuid': UUID, 'workspace': '/opt/hostcompat-qualification',
                     'candidate_sha256': SHA, 'candidate_version': 'development-ci',
                     'candidate_schema': '21', 'provider': provider,
                     'pacman_root': '/opt/hostcompat-qualification/arch' if provider != 'apt' else None,
                     'hook_sha256': {n: hashlib.sha256(n.encode()).hexdigest() for n in guards.HOOKS}}

    def call(self, name):
        self.calls.append(name)
        if self.fail == name:
            raise guards.Refusal('injected unsafe ' + name)

    def identity(self): self.call('identity'); return self.host
    def marker(self): self.call('marker'); return self.data
    def vm_uuid(self): self.call('uuid'); return self.uuid
    def virtualization(self): self.call('vm'); return 'kvm'
    def processes(self): self.call('processes')
    def clean_root(self, root, pacman=False): self.call('arch-clean' if pacman else 'owner-clean')
    def directory(self, path): self.call('directory')
    def absent(self, path): self.call('absent')
    def tool(self, path): self.call('tool')
    def apt_identity(self): self.call('apt-identity')
    def file(self, path, limit=None):
        self.call('file')
        if str(path).endswith('/runner'):
            return self.candidate
        if str(path).endswith('.hostcompat-clean-root.json'):
            return json.dumps({'schema': 1, 'purpose': 'omabit-disposable-pacman-root',
                               'vm_uuid': UUID, 'root': self.data['pacman_root']}).encode()
        return Path(path).name.encode()


class AdmissionTests(unittest.TestCase):
    def validate(self, probe, args=None):
        return guards.validate(qualify.parser().parse_args(args or arguments()), probe)

    def test_positive_validate_is_read_only_and_does_not_execute_candidate(self):
        probe = FakeProbe()
        out = io.StringIO()
        with mock.patch('subprocess.run', side_effect=AssertionError('no subprocess in fake validation')), contextlib.redirect_stdout(out):
            self.assertEqual(qualify.main(arguments(), probe=probe), 0)
        data = json.loads(out.getvalue())
        self.assertTrue(data['validated'])
        self.assertFalse(data['mutated'])
        self.assertIn('run checks', data['schema_execution'])

    def test_each_safety_boundary_refuses_before_mutation_dispatch(self):
        for name in ('identity', 'marker', 'uuid', 'vm', 'processes', 'owner-clean', 'directory', 'absent', 'file', 'apt-identity', 'tool'):
            with self.subTest(boundary=name):
                p = FakeProbe(); p.fail = name
                mutate = mock.Mock(side_effect=AssertionError('mutator reached'))
                with contextlib.redirect_stderr(io.StringIO()):
                    code = qualify.main(arguments('run') + ['--acknowledge-disposable-guest'], probe=p, execute=mutate)
                self.assertEqual(code, 2)
                mutate.assert_not_called()

    def test_real_identity_rejects_nonroot_before_any_read(self):
        with mock.patch('guards.platform.system', return_value='Linux'), mock.patch('guards.os.geteuid', return_value=1000), mock.patch('guards.trusted_bytes', side_effect=AssertionError('read before euid gate')):
            with self.assertRaisesRegex(guards.Refusal, 'root required'):
                guards.Probe().identity()

    def test_nonlinux_refuses_before_identity_or_fs(self):
        with mock.patch('guards.platform.system', return_value='Darwin'), mock.patch('guards.os.geteuid', side_effect=AssertionError('identity reached')):
            with self.assertRaisesRegex(guards.Refusal, 'Linux required'):
                guards.Probe().identity()

    def test_marker_and_identity_pins_fail_closed(self):
        for field, value in [('purpose', 'normal-host'), ('hostname', 'sourceport'), ('vm_uuid', '0' * 36),
                             ('candidate_sha256', 'a' * 64), ('candidate_version', 'other'), ('candidate_schema', '22'),
                             ('workspace', '/home/operator'), ('extra', True)]:
            with self.subTest(field=field):
                p = FakeProbe(); p.data[field] = value
                with self.assertRaises(guards.Refusal): self.validate(p)
        for attr, value in [('host', 'production'), ('uuid', '0' * 36), ('candidate', b'tampered')]:
            p = FakeProbe(); setattr(p, attr, value)
            with self.assertRaises(guards.Refusal): self.validate(p)

    def test_hook_digest_and_missing_version_refuse(self):
        p = FakeProbe(); p.data['hook_sha256']['refresh-host-packages'] = '0' * 64
        with self.assertRaisesRegex(guards.Refusal, 'hook digest'): self.validate(p)
        args = arguments(); i = args.index('--candidate-version'); del args[i:i+2]
        with self.assertRaisesRegex(guards.Refusal, 'version label'): self.validate(FakeProbe(), args)

    def test_no_ack_never_probes_guest_or_mutates(self):
        p = FakeProbe(); mutate = mock.Mock()
        with contextlib.redirect_stderr(io.StringIO()):
            self.assertEqual(qualify.main(arguments('run'), probe=p, execute=mutate), 2)
        self.assertEqual(p.calls, []); mutate.assert_not_called()

    def test_schema_mismatch_refuses_before_loading_mutator(self):
        with mock.patch('qualify.digest', return_value=SHA), mock.patch('qualify.subprocess.run', return_value=subprocess.CompletedProcess([], 0, b'22\n', b'')), contextlib.redirect_stderr(io.StringIO()):
            self.assertEqual(qualify.main(arguments('run') + ['--acknowledge-disposable-guest'], probe=FakeProbe()), 2)

    def test_arch_must_be_clean_marked_subroot(self):
        args = arguments(provider='both'); p = FakeProbe('both')
        self.assertTrue(self.validate(p, args)['validated'])
        p.fail = 'arch-clean'
        with self.assertRaises(guards.Refusal): self.validate(p, args)
        for root in ('/', '/usr', '/opt/other/arch', '/opt/hostcompat-qualification/run/arch', '/opt/hostcompat-qualification/../arch'):
            values = list(args); values[values.index('--pacman-root') + 1] = root
            with self.subTest(root=root), self.assertRaises(guards.Refusal): self.validate(FakeProbe('both'), values)

    def test_real_process_probe_rejects_urbit_and_engine_without_killing(self):
        pid = mock.Mock(); pid.name = '123'
        comm = mock.Mock(); comm.read_text.return_value = 'urbit\n'
        pid.__truediv__ = mock.Mock(return_value=comm)
        with mock.patch('guards.Path.iterdir', return_value=iter([pid])), mock.patch('guards.os.readlink', return_value='/bin/urbit'):
            with self.assertRaisesRegex(guards.Refusal, 'process refused'): guards.Probe().processes()

    def test_existing_owner_or_pier_boundary_is_not_disposable(self):
        with mock.patch('guards.os.path.lexists', side_effect=lambda p: str(p) == '/etc/urbit-runner'):
            with self.assertRaisesRegex(guards.Refusal, 'owner/state/pier'): guards.Probe().clean_root(Path('/'))

    def test_vm_detection_rejects_container_and_bare_metal(self):
        with mock.patch('guards.trusted_path'), mock.patch('guards.subprocess.run', return_value=subprocess.CompletedProcess([], 0, b'docker\n', b'')):
            with self.assertRaisesRegex(guards.Refusal, 'containers'): guards.Probe().virtualization()
        with mock.patch('guards.trusted_path'), mock.patch('guards.subprocess.run', return_value=subprocess.CompletedProcess([], 1, b'none\n', b'')):
            with self.assertRaisesRegex(guards.Refusal, 'VM is required'): guards.Probe().virtualization()

    def test_links_and_writable_parent_fail_before_file_open(self):
        good = type('S', (), {'st_mode': stat.S_IFDIR | 0o755, 'st_uid': 0})()
        link = type('S', (), {'st_mode': stat.S_IFLNK | 0o777, 'st_uid': 0, 'st_nlink': 1})()
        with mock.patch('guards.Path.lstat', side_effect=[good, good, link]), mock.patch('guards.os.open', side_effect=AssertionError('opened unsafe marker')):
            with self.assertRaises(guards.Refusal): guards.trusted_bytes('/etc/marker', 4096)
        bad = type('S', (), {'st_mode': stat.S_IFDIR | 0o777, 'st_uid': 0})()
        with mock.patch('guards.Path.lstat', return_value=bad):
            with self.assertRaisesRegex(guards.Refusal, 'unsafe parent'): guards.trusted_path('/etc/marker')

    def test_duplicate_marker_keys_refused(self):
        with self.assertRaisesRegex(guards.Refusal, 'duplicate'): guards.decode_marker('{"schema":1,"schema":1}')

    def test_python_optimization_refused_before_filesystem_checks(self):
        with mock.patch('guards.platform.system', return_value='Linux'), mock.patch('guards.platform.machine', return_value='x86_64'), mock.patch('guards.os.geteuid', return_value=0), mock.patch.object(guards.sys, 'flags', mock.Mock(optimize=1)):
            with self.assertRaisesRegex(guards.Refusal, 'optimization'): guards.Probe().identity()

    def test_candidate_drift_after_admission_still_precedes_first_write(self):
        from fixtures import Fixtures
        options = qualify.parser().parse_args(arguments())
        f = Fixtures(options, {'hook_sha256': FakeProbe().data['hook_sha256']})
        with mock.patch('fixtures.trusted_bytes', return_value=b'changed after validate'), mock.patch('pathlib.Path.mkdir', side_effect=AssertionError('mutation before digest check')):
            with self.assertRaisesRegex(guards.Refusal, 'candidate changed'): f.setup()


class CLITests(unittest.TestCase):
    def test_fixture_and_suite_imports_are_inert(self):
        import importlib
        with mock.patch('pathlib.Path.mkdir', side_effect=AssertionError('import mkdir')), mock.patch('pathlib.Path.write_bytes', side_effect=AssertionError('import write')), mock.patch('pathlib.Path.write_text', side_effect=AssertionError('import write')), mock.patch('subprocess.run', side_effect=AssertionError('import command')), mock.patch('threading.Thread.start', side_effect=AssertionError('import thread')):
            importlib.import_module('fixtures')
            importlib.import_module('suite')

    def test_harmless_help_plan_and_validate_formatting(self):
        script = str(Path(__file__).with_name('qualify.py'))
        with tempfile.TemporaryDirectory() as tmp:
            for args in ([], ['--help'], ['plan'], ['plan', '--format', 'text'], ['validate']):
                with self.subTest(args=args):
                    r = subprocess.run([sys.executable, '-B', script] + args, cwd=tmp, capture_output=True, text=True)
                    self.assertEqual(list(Path(tmp).iterdir()), [])
                    self.assertNotIn('Traceback', r.stderr)
                    if args == ['validate']:
                        self.assertEqual(r.returncode, 2)
                        self.assertFalse(json.loads(r.stderr)['validated'])
                    elif args == ['plan']:
                        self.assertEqual(r.returncode, 0)
                        self.assertFalse(json.loads(r.stdout)['mutated'])
                    else:
                        self.assertEqual(r.returncode, 0)
                        self.assertTrue(r.stdout.strip())


if __name__ == '__main__':
    unittest.main()
