"""Read bounded JSON guest events, including journald LineMax continuations.

Only immediately adjacent records from the same process tag can continue a JSON
value. This is transport framing, not authority; offline disk receipts still need
independent equality/hash checks before runtime qualification.
"""
import json
import re

_PREFIX = re.compile(r'^\[\s*\d+\.\d+\] (?P<tag>[A-Za-z0-9_.-]+\[\d+\]): (?P<body>.*)$')
_MARKER = 'OMABIT_GUEST '
_LIMIT = 256 * 1024


def guest_events(output):
    events = []
    pending = ''
    tag = None
    for line in output.splitlines():
        match = _PREFIX.fullmatch(line)
        current_tag = match['tag'] if match else None
        body = match['body'] if match else line
        if body.startswith(_MARKER):
            pending, tag = body[len(_MARKER):], current_tag
        elif pending and tag is not None and current_tag == tag:
            pending += body
        else:
            pending, tag = '', None
            continue
        if len(pending) > _LIMIT:
            pending, tag = '', None
            continue
        try:
            value = json.loads(pending)
        except ValueError:
            continue
        if isinstance(value, dict) and isinstance(value.get('stage'), str):
            events.append(value)
        pending, tag = '', None
    return events
