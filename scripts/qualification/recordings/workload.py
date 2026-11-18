"""Fixed inert output/signal fixture. Never launches Urbit or reads user files."""
import json
import os
from pathlib import Path
import signal
import sys
import time

root = Path(sys.argv[1])
if len(sys.argv) != 2 or str(root) not in ('/lab/pipe', '/lab/pty', '/lab/script', '/lab/tmux'):
    raise RuntimeError('fixed fixture root required')
os.umask(0o077)
(root/'pid').write_text(str(os.getpid()))
(root/'startup-signal-mask.json').write_text(json.dumps(sorted(int(s) for s in signal.pthread_sigmask(signal.SIG_BLOCK, []))))

def stop(number, frame):
    result = {'signal': number, 'pid': os.getpid(), 'planned_exit': 7 if number == signal.SIGTERM else 128 + number}
    try:
        os.write(1, b'FINAL-OWNED-SIGNAL\n'); result['final_write'] = 'accepted'
    except OSError as error:
        result['final_write'] = 'failed'; result['write_errno'] = error.errno
    (root/'signal.json').write_text(json.dumps(result))
    raise SystemExit(result['planned_exit'])

signal.signal(signal.SIGTERM, stop)
signal.signal(signal.SIGHUP, stop)
if root.name == 'tmux':
    end = time.monotonic() + 10
    while not (root/'capture-ready').exists():
        if time.monotonic() > end:
            raise RuntimeError('private tmux pipe readiness deadline')
        time.sleep(.01)
os.write(1, b'BEGIN-OWNED-OUTPUT\n')
for i in range(350):
    os.write(1, ('trace-line-%04d\n' % i).encode())
os.write(1, b'long-line:' + b'x'*12288 + b'\n')
os.write(1, b'0wdiagnostic.atom Authorization: fake-fixture-token Cookie: fake-fixture-cookie\n')
os.write(2, b'STDERR-OWNED-MARKER\n')
for i in range(4):
    os.write(1, b'\x1b[2K\rspinner-' + str(i).encode())
    time.sleep(.04)
os.write(1, b'\nBURST-COMPLETE\n')
(root/'ready').write_text('ready')
while True:
    signal.pause()
