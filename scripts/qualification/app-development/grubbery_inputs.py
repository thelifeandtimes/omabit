"""Source-only admission for one explicitly pinned native Grubbery capsule."""
import hashlib
import json
import os
from pathlib import Path
import stat

DEFINITION_SHA = 'ad2ea8615c95483232eb2a07c019fade42787775e9520b158e26c6cbc43cce77'


def verify_framework_source(directory, definition):
    directory = Path(directory)
    raw = Path(definition).read_bytes()
    if hashlib.sha256(raw).hexdigest() != DEFINITION_SHA:
        raise ValueError('unexpected framework definition')
    frozen = json.loads(raw)
    if not directory.is_absolute() or directory.is_symlink() or not directory.is_dir():
        raise ValueError('explicit ordinary framework desk required')
    expected = {row['path']: row for row in frozen['source']['files']}
    seen = set()
    entries = 0
    for parent, folders, files in os.walk(directory, followlinks=False):
        entries += len(folders) + len(files)
        if entries > 8192 or len(Path(parent).relative_to(directory).parts) > 16:
            raise ValueError('framework traversal bound exceeded')
        for name in folders:
            if (Path(parent)/name).is_symlink():
                raise ValueError('framework directory link refused')
        for name in files:
            path = Path(parent)/name
            relative = path.relative_to(directory).as_posix()
            before = path.lstat()
            if not stat.S_ISREG(before.st_mode) or before.st_nlink != 1 or relative not in expected:
                raise ValueError('unexpected framework source object')
            row = expected[relative]
            if before.st_size != row['bytes']:
                raise ValueError('framework source size changed')
            # Each admitted file is independently bounded before reading.
            if before.st_size > 32*1024**2:
                raise ValueError('framework file exceeds source bound')
            with path.open('rb') as stream:
                opened = os.fstat(stream.fileno())
                digest = hashlib.file_digest(stream, 'sha256').hexdigest()
                after = os.fstat(stream.fileno())
            current = path.lstat()
            attributes = ('st_dev', 'st_ino', 'st_mode', 'st_nlink', 'st_size', 'st_mtime_ns', 'st_ctime_ns')
            if any(getattr(before, key) != getattr(value, key) for value in (opened, after, current) for key in attributes) or digest != row['sha256']:
                raise ValueError('framework source changed during observation')
            seen.add(relative)
    if seen != set(expected):
        raise ValueError('incomplete framework source')
    return {'commit': frozen['commit'], 'files': len(seen), 'bytes': sum(row['bytes'] for row in expected.values()),
            'source_digest': frozen['source']['digest'], 'definition_sha256': DEFINITION_SHA}
