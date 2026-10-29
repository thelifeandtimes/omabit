#!/usr/bin/env python3
"""Fresh no-NIC VM entry point for managed gateway plus member-key qualification."""
import sys

# The inert source module is delivered inside the same reviewed guest payload.
# main() checks the fresh guest boundary before installation or runtime access.
sys.path.insert(0, '/opt/omabit-lab/private-chain/src')
from managed_gateway_guest import main

if __name__ == '__main__':
    main(member_keys=True)
