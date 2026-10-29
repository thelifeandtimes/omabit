#!/usr/bin/env python3
"""Fresh no-NIC VM: first managed member, HTTP identity and fixture Lens exit."""
import sys
sys.path.insert(0, '/opt/omabit-lab/private-chain/src')
from managed_gateway_guest import main
if __name__ == '__main__':
    main(member_boot=True)
