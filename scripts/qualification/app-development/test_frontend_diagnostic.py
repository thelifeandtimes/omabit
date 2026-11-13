import json
from pathlib import Path
import tempfile
from types import SimpleNamespace
import unittest
from unittest.mock import patch

import frontend_diagnostic
import probe


class FrontendDiagnosticTests(unittest.TestCase):
    def test_diagnostic_cannot_launch_with_a_mutating_workflow(self):
        for change in ({'resume_from': None}, {'workflow_binary': Path('/unused')},
                       {'release_source': Path('/unused')}, {'receivers': True},
                       {'recover_receiver_boot': Path('/unused')}):
            with self.subTest(change=change):
                values = dict(frontend_diagnostic=True, resume_from=Path('/unused'),
                              workflow_binary=None, release_source=None,
                              receivers=False, recover_receiver_boot=None)
                values.update(change)
                with patch('probe.subprocess.Popen') as spawn, patch('probe.shutil.copyfile') as copy:
                    with self.assertRaisesRegex(ValueError, 'only one clean stopped fixture'):
                        probe.host(SimpleNamespace(**values))
                    spawn.assert_not_called()
                    copy.assert_not_called()

    def test_asset_inspection_never_reuploads_and_cleans_its_session(self):
        calls = []

        class Response:
            status = 200
            def read(self, limit):
                return b'fixture'
            def getheader(self, name, default=None):
                return 'urbauth-~zod=fixture-secret' if name == 'Set-Cookie' else 'text/plain'

        class Connection:
            def request(self, method, path, body=None, headers=None):
                calls.append((method, path))
            def getresponse(self):
                return Response()
            def close(self):
                pass

        def read(name, expression, **kwargs):
            self.assertNotIn('app', kwargs)
            self.assertTrue('/docket/' in expression or '/multipart/' in expression)
            return 'fixture metadata'

        with tempfile.TemporaryDirectory() as directory, patch('frontend_diagnostic.http.client.HTTPConnection', return_value=Connection()):
            result = frontend_diagnostic.inspect_frontend(Path(directory), 'omabit-test-01234567', read)
            saved = (Path(directory) / 'frontend-diagnostic.json').read_text()
            self.assertTrue(result['session_cleanup'])
            self.assertEqual(result['app_mutations'], 0)
            self.assertEqual(calls[0], ('POST', '/~/login'))
            self.assertEqual(calls[-1], ('POST', '/~/logout'))
            self.assertTrue(all(method == 'GET' for method, _ in calls[1:-1]))
            self.assertNotIn('fixture-secret', saved)
            self.assertEqual(len(json.loads(saved)['assets']), 3)


if __name__ == '__main__':
    unittest.main()
