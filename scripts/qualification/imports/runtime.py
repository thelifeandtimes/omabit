#!/usr/bin/env python3
"""Stage Omabit's pinned official runtime as a verified offline OCI archive."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import re
import shutil
import tarfile
import urllib.request

REPO=Path(__file__).resolve().parents[3]

def sha(path):
    with path.open('rb') as f:return hashlib.file_digest(f,'sha256').hexdigest()

def main():
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--output',type=Path,required=True)
    parser.add_argument('--pill',type=Path,required=True,help='previously downloaded pinned pill; verified again')
    args=parser.parse_args();work=args.output.absolute()
    if work.parent!=Path('/tmp') or not re.fullmatch('omabit-import-runtime-[A-Za-z0-9-]+',work.name) or work.exists():parser.error('use a fresh /tmp/omabit-import-runtime-NAME')
    source=(REPO/'internal/runtimecandidate/metadata.go').read_text()
    def pin(name):return re.search(r'\b'+name+r'\s*=\s*"([^"]+)"',source)[1]
    digest=pin('ManifestDigest');config_digest=pin('ConfigDigest');pill_sha=pin('PillSHA256')
    pill_bytes=int(re.search(r'PillBytes\s*=\s*int64\((\d+)\)',source)[1])
    if args.pill.stat().st_size!=pill_bytes or sha(args.pill)!=pill_sha:raise ValueError('pill pin mismatch')
    os.umask(0o077);work.mkdir(mode=0o700)
    blobs=work/'oci/blobs/sha256';blobs.mkdir(parents=True)
    token_url='https://auth.docker.io/token?service=registry.docker.io&scope=repository:tloncorp/vere:pull'
    with urllib.request.urlopen(token_url,timeout=30) as response:token=json.loads(response.read(32768))['token']
    evidence=[];total=0
    def fetch(kind,wanted,expected=None):
        nonlocal total
        if not re.fullmatch('sha256:[0-9a-f]{64}',wanted):raise ValueError('invalid digest')
        path=blobs/wanted.split(':')[1]
        if path.exists():return path
        url='https://registry-1.docker.io/v2/tloncorp/vere/'+kind+'/'+wanted
        request=urllib.request.Request(url,headers={'Authorization':'Bearer '+token,'Accept':'application/vnd.docker.distribution.manifest.v2+json, application/vnd.oci.image.manifest.v1+json'})
        size=0
        with urllib.request.urlopen(request,timeout=120) as response,path.open('xb') as out:
            while chunk:=response.read(1048576):
                size+=len(chunk);total+=len(chunk)
                if size>1024**3 or total>2*1024**3:raise ValueError('runtime download bound exceeded')
                out.write(chunk)
        if sha(path)!=wanted.split(':')[1] or (expected is not None and size!=expected):raise ValueError('runtime digest/size mismatch')
        evidence.append({'digest':wanted,'size':size,'url':url});print('Verified '+wanted,flush=True)
        return path
    manifest_path=fetch('manifests',digest)
    manifest=json.loads(manifest_path.read_text())
    if manifest.get('schemaVersion')!=2 or manifest['config']['digest']!=config_digest:raise ValueError('unexpected runtime manifest')
    if not 1<=len(manifest['layers'])<=32:raise ValueError('unexpected layer count')
    config=manifest['config'];config_path=fetch('blobs',config['digest'],config['size'])
    metadata=json.loads(config_path.read_text())
    if metadata.get('os')!='linux' or metadata.get('architecture')!='amd64':raise ValueError('wrong image platform')
    for layer in manifest['layers']:fetch('blobs',layer['digest'],layer['size'])
    image='docker.io/tloncorp/vere@'+digest
    index={'schemaVersion':2,'manifests':[{'mediaType':manifest['mediaType'],'digest':digest,'size':manifest_path.stat().st_size,'platform':{'os':'linux','architecture':'amd64'},'annotations':{'io.containerd.image.name':image,'org.opencontainers.image.ref.name':image}}]}
    (work/'oci/index.json').write_text(json.dumps(index)+'\n')
    (work/'oci/oci-layout').write_text('{"imageLayoutVersion":"1.0.0"}\n')
    with tarfile.open(work/'runtime.oci.tar','w') as tar:
        for path in sorted((work/'oci').rglob('*')):
            if path.is_file():tar.add(path,arcname=str(path.relative_to(work/'oci')),recursive=False)
    shutil.copyfile(args.pill,work/'boot.pill')
    report={'image':image,'config_digest':config_digest,'platform':'linux/amd64','verified_blobs':evidence,'bytes':total,'archive_sha256':sha(work/'runtime.oci.tar'),'pill_sha256':sha(work/'boot.pill'),'pill_bytes':pill_bytes,'pin_source_sha256':sha(REPO/'internal/runtimecandidate/metadata.go'),'driver_sha256':sha(Path(__file__))}
    (work/'runtime.json').write_text(json.dumps(report,indent=2)+'\n')
    print(json.dumps({'output':str(work),'bytes':total,'verified_blobs':len(evidence)}),flush=True)

if __name__=='__main__':main()
