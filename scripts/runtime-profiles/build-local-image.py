#!/usr/bin/env python3
"""Package a local Vere build or an explicit upstream channel without running it.

The output profile is content-addressed. Installation and ship selection are
separate owner commands. Compatibility flags are assertions, not test evidence.
"""
import argparse, hashlib, importlib.util, json, re, tempfile, urllib.request
from pathlib import Path

HERE=Path(__file__).resolve().parent
spec=importlib.util.spec_from_file_location('vere_image',HERE/'build-jsonfix-image.py')
image=importlib.util.module_from_spec(spec);spec.loader.exec_module(image)
VERSION=re.compile(r'^[A-Za-z0-9][A-Za-z0-9._+-]{0,95}$')

def profile_id(p):
    fields=['format','kind','platform','vere_version','image','manifest_digest','config_digest','binary_sha256','source_base','patch_revision','requirement','review_url','archive_sha256','archive_bytes','channel','max_loom','compatible_versions']
    return 'local-'+hashlib.sha256('\n'.join(str(p.get(k,'')) for k in fields).encode()).hexdigest()[:56]

def download(url,limit):
    with urllib.request.urlopen(url,timeout=60) as r:
        if not r.url.startswith('https://'):raise ValueError('insecure channel redirect')
        data=r.read(limit+1)
    if len(data)>limit:raise ValueError('channel response exceeds size bound')
    return data

def build(args):
    if not 30<=args.max_loom<=40:raise ValueError('max loom must be in 30..40')
    for value in (args.source_revision,args.patch_revision,args.requirement,' '.join(sorted(set(args.compatible_vere)))):
        if len(value)>512 or any(c in value for c in '\x00\n\r'):raise ValueError('invalid runtime provenance')
    for v in args.compatible_vere:
        if not VERSION.fullmatch(v):raise ValueError('invalid compatible Vere version')
    with tempfile.TemporaryDirectory(prefix='omabit-vere-input-') as temporary:
        url=''
        if args.channel:
            channel_version=download('https://bootstrap.urbit.org/vere/'+args.channel+'/last',128).decode().strip()
            version=args.version or channel_version
            if not VERSION.fullmatch(channel_version) or not VERSION.fullmatch(version):raise ValueError('invalid upstream channel version')
            url=f'https://bootstrap.urbit.org/vere/{args.channel}/v{channel_version}/vere{args.bits}-v{channel_version}-linux-x86_64'
            data=download(url,image.MAX_BLOB)
            binary=Path(temporary)/'vere';binary.write_bytes(data)
        else:
            version=args.version
            if not version or not VERSION.fullmatch(version):raise ValueError('--version must be the exact local Vere banner version')
            binary=args.binary
            if binary.is_symlink() or not binary.is_file() or binary.stat().st_size>image.MAX_BLOB:raise ValueError('invalid local executable input')
            data=binary.read_bytes()
        digest=image.sha(data)
        if args.sha256 and args.sha256!=digest:raise ValueError('Vere executable checksum mismatch')
        metadata={'binary_sha256':digest,'vere_version':version,'source_base':args.source_revision,'patch_revision':args.patch_revision}
        report=image.build(binary,args.cache,args.output,metadata)
    p={'format':1,'kind':'local','platform':'linux/amd64','vere_version':version,
       'image':'omabit/vere@sha256:'+report['image_manifest_sha256'],
       'manifest_digest':'sha256:'+report['image_manifest_sha256'],'config_digest':'sha256:'+report['image_config_sha256'],
       'binary_sha256':digest,'source_base':args.source_revision,'patch_revision':args.patch_revision,
       'requirement':args.requirement,'review_url':url,'archive_sha256':report['archive_sha256'],'archive_bytes':report['archive_bytes'],
       'channel':args.channel or '', 'max_loom':args.max_loom,'compatible_versions':' '.join(sorted(set(args.compatible_vere)))}
    p['id']=profile_id(p)
    (args.output/'profile.json').write_text(json.dumps(p,indent=2)+'\n')
    print(json.dumps({'profile':p['id'],'manifest':str(args.output/'profile.json'),'archive':str(args.output/report['archive']),'qualification':'packaging only; executable was not run'},indent=2))

def parser():
    p=argparse.ArgumentParser(description=__doc__)
    choice=p.add_mutually_exclusive_group(required=True)
    choice.add_argument('--binary',type=Path);choice.add_argument('--channel',choices=['live','edge','dev'])
    p.add_argument('--version',help='exact Vere banner version; defaults to the channel version for upstream builds');p.add_argument('--bits',choices=['32','64'],default='32')
    p.add_argument('--sha256');p.add_argument('--source-revision',default='');p.add_argument('--patch-revision',default='')
    p.add_argument('--max-loom',type=int,default=34);p.add_argument('--compatible-vere',action='append',default=[])
    p.add_argument('--requirement',default='',help='replay correction this build actually retains (owner assertion)')
    p.add_argument('--cache',type=Path,required=True);p.add_argument('--output',type=Path,required=True)
    return p

if __name__=='__main__':
    build(parser().parse_args())
