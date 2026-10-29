#!/usr/bin/env python3
"""Format2 native guard with explicit linked-authority reader support."""
import argparse
import json
import os
import selectors
import sys

from artifacts import object_json
import journal as journal
import continuation_admission as admission


def output(value):
    print(json.dumps(value, sort_keys=True, separators=(',', ':')), flush=True)


def run(root):
    gate = claim = None
    try:
        raw = sys.stdin.buffer.raw.readline(16385)
        journal.require(raw.endswith(b'\n') and len(raw) <= 16384)
        request = object_json(raw)
        journal.exact(request, ('expected', 'role', 'instance'))
        gate = admission.Gate(root)
        claim = gate.claim(request['expected'], request['role'], request['instance'])
        status = claim.status()
        journal.require(status['admitted'] and status['state'] == 'open')
        output({'ok': True, 'admission': status})
        previous = status
        with selectors.DefaultSelector() as poll:
            poll.register(sys.stdin, selectors.EVENT_READ)
            while True:
                if poll.select(.25):
                    journal.require(not os.read(sys.stdin.fileno(), 1), 'unexpected guard input')
                    return 0
                status = claim.status()
                if status != previous:
                    output({'ok': True, 'admission': status})
                    previous = status
                if not status['admitted']:
                    return 1
    except (OSError, ValueError, KeyError):
        try:
            output({'ok': False, 'reason': 'shared-admission-unavailable'})
        except (OSError, BrokenPipeError):
            pass
        return 1
    finally:
        if claim is not None and claim.fd is not None:
            try:
                claim.end()
            except (OSError, ValueError, KeyError):
                pass
        if gate is not None:
            gate.close()


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--root', required=True)
    args = parser.parse_args()
    raise SystemExit(run(args.root))
