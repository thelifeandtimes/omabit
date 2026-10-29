#!/usr/bin/env python3
"""Download a signed, offline Arch guest package set without installing on the host."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import re
import shutil
import subprocess
import urllib.request

PACKAGES=('base','docker','rootlesskit','slirp4netns','python','sudo','openssh','iptables-nft','e2fsprogs','omarchy-keyring')

def digest(path):
    with path.open('rb') as f:return hashlib.file_digest(f,'sha256').hexdigest()

def main():
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--output',type=Path,required=True)
    parser.add_argument('--cache',type=Path,help='optional previous private package directory; every copied signature is reverified')
    parser.add_argument('--restore-ui',action='store_true',help='include signed Qt/PySide6 and font packages for the guest-only offscreen UI')
    args=parser.parse_args();work=args.output.absolute()
    if work.parent!=Path('/tmp') or not re.fullmatch('omabit-import-guest-packages-[A-Za-z0-9-]+',work.name) or work.exists():parser.error('use a fresh /tmp/omabit-import-guest-packages-NAME')
    os.umask(0o077);work.mkdir(mode=0o700)
    packages=work/'packages';packages.mkdir()
    database=work/'database';(database/'sync').mkdir(parents=True);(database/'local').mkdir()
    gnupg=work/'gnupg';gnupg.mkdir()
    config=work/'pacman.conf'
    config.write_text('[options]\nArchitecture = x86_64\nSigLevel = Required DatabaseOptional\n'+''.join('\n['+name+']\nServer = '+('https://pkgs.omarchy.org/stable/$arch' if name=='omarchy' else 'https://stable-mirror.omarchy.org/'+name+'/os/$arch')+'\n' for name in ('core','extra','omarchy')))
    for name in ('core','extra','omarchy'):shutil.copyfile(Path('/var/lib/pacman/sync')/(name+'.db'),database/'sync'/(name+'.db'))
    command=['pacman','--config',str(config),'--dbpath',str(database),'--cachedir',str(packages),'--logfile',str(work/'pacman.log'),'-Sp','--print-format','%l',*PACKAGES,*(('pyside6','ttf-dejavu') if args.restore_ui else ())]
    result=subprocess.run(command,capture_output=True,text=True,check=True)
    urls=result.stdout.splitlines()
    if not 1<=len(urls)<=256 or any(not url.startswith(('https://stable-mirror.omarchy.org/','https://pkgs.omarchy.org/stable/')) or not url.endswith('.pkg.tar.zst') for url in urls):raise ValueError('unexpected package resolution')
    (work/'resolution.json').write_text(json.dumps({'command':command,'urls':urls,'sync_sha256':{p.name:digest(p) for p in (database/'sync').iterdir()}},indent=2)+'\n')
    keyrings=[];revoked=set()
    for name in ('archlinux','omarchy'):
        source=Path('/usr/share/pacman/keyrings')/(name+'.gpg');data=source.read_bytes();target=gnupg/(name+'.gpg')
        if data.startswith(b'-----BEGIN'):data=subprocess.check_output(['gpg','--homedir',str(gnupg),'--dearmor'],input=data)
        target.write_bytes(data);keyrings+=['--keyring',str(target)]
        revoked.update((source.parent/(name+'-revoked')).read_text().split())
    evidence=[];total=0
    for index,url in enumerate(urls,1):
        name=url.rsplit('/',1)[1]
        print(f'[{index}/{len(urls)}] {name}',flush=True)
        archive=packages/name
        cached=args.cache/name if args.cache else None
        for suffix in ('','.sig'):
            target=Path(str(archive)+suffix)
            if cached and cached.is_file() and Path(str(cached)+'.sig').is_file():
                shutil.copyfile(Path(str(cached)+suffix),target);total+=target.stat().st_size
                if total>2147483648:raise ValueError('package bound exceeded')
                continue
            with urllib.request.urlopen(url+suffix,timeout=120) as response,target.open('xb') as out:
                size=0
                while chunk:=response.read(1048576):
                    size+=len(chunk);total+=len(chunk)
                    if size>536870912 or total>2147483648:raise ValueError('package download bound exceeded')
                    out.write(chunk)
        verified=subprocess.run(['gpgv','--homedir',str(gnupg),'--status-fd=1',*keyrings,str(archive)+'.sig',str(archive)],capture_output=True,text=True,check=True)
        signatures=[line.split()[2:] for line in verified.stdout.splitlines() if line.startswith('[GNUPG:] VALIDSIG ')]
        if not signatures or any(value in revoked for fields in signatures for value in fields):raise ValueError('missing or revoked package signer')
        evidence.append({'url':url,'filename':name,'sha256':digest(archive),'signature_status':verified.stdout,'signature':verified.stderr})
    (work/'packages.json').write_text(json.dumps(evidence,indent=2)+'\n')
    print(json.dumps({'packages':len(evidence),'bytes':total,'output':str(work)}),flush=True)

if __name__=='__main__':main()
