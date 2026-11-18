import os
from pathlib import Path
import signal
import subprocess
import sys
import tempfile
import unittest


class SupervisorSignalTests(unittest.TestCase):
    def test_termination_reaches_exception_and_finally_in_owned_child(self):
        with tempfile.TemporaryDirectory(prefix='omabit-signal-fixture-') as name:
            output = Path(name) / 'result'
            program = """
import importlib.util,signal,sys,time
from pathlib import Path
spec=importlib.util.spec_from_file_location('probe_fixture',sys.argv[1])
probe=importlib.util.module_from_spec(spec);spec.loader.exec_module(probe)
signal.signal(signal.SIGTERM,probe._termination_signal)
try:
 print('ready',flush=True)
 time.sleep(20)
except InterruptedError:
 Path(sys.argv[2]).write_text('interrupted')
finally:
 with Path(sys.argv[2]).open('a') as f:f.write(':finally')
"""
            child = subprocess.Popen([sys.executable, '-B', '-c', program,
                                      str(Path(__file__).with_name('probe.py')), str(output)],
                                     stdout=subprocess.PIPE, stderr=subprocess.PIPE,
                                     text=True)
            try:
                self.assertEqual(child.stdout.readline().strip(), 'ready')
                os.kill(child.pid, signal.SIGTERM)
                stdout, stderr = child.communicate(timeout=5)
                self.assertEqual(child.returncode, 0, stderr)
                self.assertEqual(output.read_text(), 'interrupted:finally')
            finally:
                if child.poll() is None:
                    child.kill()
                    child.wait()
