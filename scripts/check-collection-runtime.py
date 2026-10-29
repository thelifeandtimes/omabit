#!/usr/bin/env python3
"""Check explicit prepared collection artifacts without enabling plugins or touching ships."""
import argparse
import json
from pathlib import Path
import sys
sys.dont_write_bytecode = True
sys.path.insert(0, str(Path(__file__).resolve().parents[1] / 'native/collections'))
import artifacts
import repeated_preflight as runtime_preflight


def main():
    parser = argparse.ArgumentParser(description=__doc__, epilog='A ready result checks bundled reader compatibility only. It does not authorize activation, install a backend, or prove application readiness.')
    parser.add_argument('--specification', type=Path, required=True, help='collection specification JSON with pinned member hashes')
    parser.add_argument('--generations', type=Path, required=True, help='JSON object mapping omabit/theme/dms to explicit absolute generation directories')
    parser.add_argument('--authority-format', type=int, default=1, help='desktop authority format required by the intended runtime (default: 1; linked successor: 2; checkpoint continuation: 3; repeated sessions: 4)')
    parser.add_argument('--reservation-format', type=int, default=3, help='reservation format required by the intended runtime (default: 3)')
    parser.add_argument('--journal-format', type=int, default=3, help='collection journal format required by the intended runtime (default: 3)')
    args = parser.parse_args()
    try:
        spec = artifacts.object_json(args.specification.read_bytes())
        roots = artifacts.object_json(args.generations.read_bytes())
        required = dict(runtime_preflight.REQUIRED, authority=args.authority_format, reservation=args.reservation_format, journal=args.journal_format)
        result = runtime_preflight.check(spec, roots, required)
    except (OSError, ValueError, KeyError, TypeError) as error:
        print(json.dumps({'error': str(error)}), file=sys.stderr);return 1
    print(json.dumps(result, indent=2))
    return 0 if result['ready'] else 2


if __name__ == '__main__':raise SystemExit(main())
