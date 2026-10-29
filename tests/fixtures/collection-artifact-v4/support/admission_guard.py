#!/usr/bin/env python3
"""One native component's admission lease; never issue or reopen a permit."""
import argparse
import json
import os
import selectors
import sys

import admission as a
from artifacts import object_json
import journal as j


def output(value):
    print(json.dumps(value,sort_keys=True,separators=(',',':')),flush=True)


def run(root):
    gate=claim=None
    try:
        # QML sends only non-secret binding metadata here. Reading a bounded line
        # before a selector avoids consuming/buffering later commands accidentally.
        raw=sys.stdin.buffer.raw.readline(16385)
        j.require(raw.endswith(b'\n') and len(raw)<=16384,'invalid admission request')
        request=object_json(raw);j.exact(request,('expected','role','instance'))
        gate=a.Gate(root);claim=gate.claim(request['expected'],request['role'],request['instance'])
        status=claim.status();j.require(status['admitted'] and status['state']=='open','admission closed before grant')
        output({'ok':True,'admission':status});previous=status
        with selectors.DefaultSelector() as poll:
            poll.register(sys.stdin,selectors.EVENT_READ)
            while True:
                events=poll.select(.25)
                if events:
                    # No commands are needed to renew admission. EOF only ends
                    # this guard; it never proves application/descendant closure.
                    byte=os.read(sys.stdin.fileno(),1)
                    j.require(not byte,'unexpected input after admission request')
                    return 0
                status=claim.status()
                if status!=previous:
                    output({'ok':True,'admission':status});previous=status
                if not status['admitted']:return 1
    except (OSError,ValueError,KeyError):
        try:output({'ok':False,'reason':'admission-unavailable'})
        except (OSError,BrokenPipeError):pass
        return 1
    finally:
        if claim is not None and claim.fd is not None:
            try:claim.end()
            except (OSError,ValueError,KeyError):pass
        if gate is not None:gate.close()


if __name__=='__main__':
    parser=argparse.ArgumentParser(description=__doc__);parser.add_argument('--root',required=True)
    args=parser.parse_args();raise SystemExit(run(args.root))
