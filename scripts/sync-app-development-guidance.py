#!/usr/bin/env python3
"""Keep independently installable app guidance bound to canonical operator text."""
import argparse
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
NAMES = ('app-development.md', 'operations.md', 'urgit.md')

def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--check', action='store_true')
    args = parser.parse_args()
    for name in NAMES:
        source = ROOT / 'skills/omabit-operator/references' / name
        target = ROOT / 'skills/omabit-app-developer/references' / name
        expected = source.read_bytes()
        if args.check:
            if not target.is_file() or target.is_symlink() or target.read_bytes() != expected:
                raise SystemExit('Generated application guidance differs: ' + name)
        else:
            if target.is_symlink():
                raise SystemExit('Refusing linked generated guidance')
            target.write_bytes(expected)

if __name__ == '__main__':
    main()
