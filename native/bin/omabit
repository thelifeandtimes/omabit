#!/usr/bin/env python3
"""Dispatch a plugin command through the pinned, per-user binary cache."""
import sys
from pathlib import Path

sys.dont_write_bytecode = True
sys.path.insert(0, str(Path(__file__).resolve().parent.parent / "bootstrap"))
from runtime import main

if __name__ == "__main__":
    sys.exit(main(Path(__file__).name, sys.argv[1:]))
