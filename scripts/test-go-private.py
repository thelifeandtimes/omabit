#!/usr/bin/env python3
"""Build a Go test binary, then run it without host profiles, homes or sockets."""
import argparse
import json
import os
from pathlib import Path
import re
import subprocess
import tempfile

ROOT=Path(__file__).resolve().parents[1]
def main():
    p=argparse.ArgumentParser(description=__doc__)
    p.add_argument('--package',required=True)
    p.add_argument('--run',default='.')
    p.add_argument('--race',action='store_true')
    p.add_argument('--with-go',action='store_true',help='read-only Go toolchain/modules for tests that compile their own fixture worker')
    p.add_argument('--timeout',default='5m')
    p.add_argument('--output',type=Path)
    args=p.parse_args()
    if not re.fullmatch(r'\./(?:internal|cmd)/[a-z0-9_-]+',args.package):p.error('one repository package required')
    match=re.fullmatch(r'([1-9][0-9]?)([sm])',args.timeout)
    if not match or int(match[1])*(60 if match[2]=='m' else 1)>2400:p.error('timeout must be between 1s and 40m')
    output=args.output or Path(tempfile.mkdtemp(prefix='omabit-go-private-'))
    if args.output:output.mkdir(mode=0o700)
    output=output.resolve();scratch=output/'scratch';scratch.mkdir(mode=0o700)
    for name in ['home','config','state','cache','run']:(scratch/name).mkdir(mode=0o700)
    binary=output/'package.test'
    build=['go','test','-c','-o',str(binary)]
    if args.race:build.append('-race')
    build.append(args.package)
    subprocess.run(build,cwd=ROOT,check=True)
    command=['bwrap','--unshare-all','--die-with-parent','--clearenv',
      '--ro-bind','/usr','/usr','--symlink','usr/bin','/bin',
      '--symlink','usr/lib','/lib','--symlink','usr/lib','/lib64',
      '--proc','/proc','--dev','/dev','--dir','/etc','--tmpfs','/run',
      '--bind',str(scratch),'/tmp','--ro-bind',str(ROOT),str(ROOT),
      '--ro-bind',str(binary),'/test','--chdir',str(ROOT/args.package),
      '--setenv','HOME','/tmp/home','--setenv','XDG_CONFIG_HOME','/tmp/config',
      '--setenv','XDG_STATE_HOME','/tmp/state','--setenv','XDG_CACHE_HOME','/tmp/cache',
      '--setenv','XDG_RUNTIME_DIR','/tmp/run','--setenv','PATH','/usr/bin:/bin',
      '--setenv','LANG','C.UTF-8']
    for name in ['passwd','group','nsswitch.conf','hosts']:
        file=Path('/etc')/name
        if file.is_file():command+=['--ro-bind',str(file),str(file)]
    if args.with_go:
        paths=json.loads(subprocess.check_output(['go','env','-json','GOROOT','GOMODCACHE'],text=True,cwd=ROOT))
        goroot=Path(paths['GOROOT']).resolve();modules=Path(paths['GOMODCACHE']).resolve()
        if not goroot.is_dir() or not modules.is_dir():raise ValueError('cached toolchain/modules required')
        command+=['--ro-bind',str(goroot),'/opt/go','--ro-bind',str(modules),'/opt/modules',
          '--setenv','PATH','/opt/go/bin:/usr/bin:/bin','--setenv','GOROOT','/opt/go',
          '--setenv','GOMODCACHE','/opt/modules','--setenv','GOCACHE','/tmp/go-build',
          '--setenv','GOPATH','/tmp/gopath','--setenv','GOTOOLCHAIN','local',
          '--setenv','GOPROXY','off','--setenv','GOSUMDB','off','--setenv','GOMAXPROCS','2']
    command+=['--','/test','-test.run',args.run,'-test.timeout',args.timeout,'-test.count=1']
    (output/'command.json').write_text(json.dumps(command,indent=2)+'\n')
    with (output/'test.log').open('w') as log:
        try:
            result=subprocess.run(command,cwd=ROOT,stdout=log,stderr=subprocess.STDOUT,timeout=int(match[1])*(60 if match[2]=='m' else 1)+30)
            status=result.returncode
        except subprocess.TimeoutExpired:
            status=124
    print((output/'test.log').read_text()[-16000:])
    receipt={'package':args.package,'run':args.run,'race':args.race,'fixture_compilation':args.with_go,'exit':status,'output':str(output),'isolation':'private user/network/PID/mount/IPC/UTS; no owner profile, host home/run or Docker socket'}
    (output/'result.json').write_text(json.dumps(receipt,indent=2)+'\n');print(json.dumps(receipt))
    return status
if __name__=='__main__':raise SystemExit(main())
