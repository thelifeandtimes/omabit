"""A fixed private tmux pipe receiver, used only inside the inert capsule."""
import json
import os
from pathlib import Path
import time

root = Path('/lab/tmux')
os.umask(0o077)
origin = time.monotonic_ns()
(root/'capture-start.json').write_text(json.dumps({'unix_ns': time.time_ns(), 'monotonic_ns': origin, 'basis': 'recorder receive clock, not process write time'}))
with (root/'raw.bin').open('xb', buffering=0) as raw, (root/'timing.ndjson').open('x') as timing:
    (root/'capture-ready').write_text('ready')
    offset = 0
    while True:
        data = os.read(0, 65536)
        if not data:
            break
        raw.write(data)
        timing.write(json.dumps({'offset': offset, 'length': len(data), 'elapsed_ns': time.monotonic_ns()-origin, 'stream': 'pty-combined'})+'\n')
        timing.flush()
        offset += len(data)
    os.fsync(raw.fileno()); os.fsync(timing.fileno())
(root/'capture-closed').write_text('closed')
