#!/usr/bin/env python3
"""Prepare a fresh private-chain VM appliance from explicit verified inputs.

This builds code and disk inputs only. It never starts Docker, a VM or a ship,
fetches dependencies, reads a pier, or generates keys. Use the common VM launcher
only after reviewing the resulting build.json. Never reuse a partial output.
"""
import argparse
import gzip
import hashlib
import json
import os
from pathlib import Path,PurePosixPath
import re
import shutil
import stat
import subprocess
import sys
import tarfile

HERE=Path(__file__).resolve().parent
REPO=HERE.parents[2]
IMPORTS=HERE.parent/'imports'
sys.path.insert(0,str(IMPORTS))
import storage

NODE_SHA='6dcdb9bbe2bf22bd053ba06d5ba7b8a2d92b2d3d8c714f3b2670be3d2728c9c8'
VERE_SHA='b5c54439ae5b9f0ff94d75319e67e04f04369776fa5e2180a5ab1a5ebd4c2ff0'
MAX_FILES=40000
MAX_BYTES=1024**3

def sha(path):
    with Path(path).open('rb') as stream:return hashlib.file_digest(stream,'sha256').hexdigest()

def read_json(path,limit=4*1024**2):
    with Path(path).open('rb') as stream:raw=stream.read(limit+1)
    if len(raw)>limit:raise ValueError('metadata exceeds its size bound')
    return json.loads(raw)

def write_json(path,value):path.write_text(json.dumps(value,indent=2)+'\n')

def relative(name):
    if not isinstance(name,str) or not name or name=='.' or '\\' in name or name!=str(PurePosixPath(name)) or PurePosixPath(name).is_absolute() or '..' in PurePosixPath(name).parts:
        raise ValueError('canonical relative input path required')
    return name

def checked_file(root,name,digest=None):
    relative(name);path=root/name;real=path.resolve(strict=True)
    if not real.is_relative_to(root) or not stat.S_ISREG(real.stat().st_mode):raise ValueError('input leaves its verified tree or is not regular')
    if digest is not None and (not re.fullmatch('[0-9a-f]{64}',digest) or sha(real)!=digest):raise ValueError('input hash differs: '+name)
    return real

class PayloadWriter:
    def __init__(self,tar):self.tar=tar;self.inventory={};self.bytes=0;self.active=set()
    def add(self,path,name,root):
        relative(name);real=path.resolve(strict=True)
        if not real.is_relative_to(root):raise ValueError('payload input escapes its selected tree')
        mode=real.stat().st_mode
        if stat.S_ISDIR(mode):
            if real in self.active:raise ValueError('payload directory cycle')
            self.active.add(real)
            try:
                for child in sorted(real.iterdir()):self.add(child,name+'/'+child.name,root)
            finally:self.active.remove(real)
            return
        if not stat.S_ISREG(mode):raise ValueError('payload accepts only regular file inputs')
        if name in self.inventory:raise ValueError('duplicate payload destination')
        size=real.stat().st_size
        if len(self.inventory)>=MAX_FILES or size<0 or self.bytes+size>MAX_BYTES:raise ValueError('payload input budget exceeded')
        info=self.tar.gettarinfo(str(real),arcname=name);info.uid=info.gid=0;info.uname=info.gname='';info.mtime=0;info.mode=0o755 if mode&0o111 else 0o644
        if not info.isfile() or info.size!=size:raise ValueError('payload input changed during collection')
        digest=sha(real)
        with real.open('rb') as stream:self.tar.addfile(info,stream)
        self.inventory[name]={'bytes':size,'sha256':digest};self.bytes+=size

def verify_payload(path,inventory):
    seen=set();total=0
    with tarfile.open(path,mode='r|gz') as tar:
        for entry in tar:
            relative(entry.name)
            if not entry.isfile() or entry.name in seen or entry.name not in inventory:raise ValueError('sealed payload member differs')
            meta=inventory[entry.name]
            if entry.size!=meta['bytes']:raise ValueError('sealed payload size differs')
            with tar.extractfile(entry) as stream:digest=hashlib.file_digest(stream,'sha256').hexdigest()
            if digest!=meta['sha256']:raise ValueError('input changed while sealing payload')
            seen.add(entry.name);total+=entry.size
    if seen!=set(inventory) or total>MAX_BYTES or len(seen)>MAX_FILES:raise ValueError('sealed payload inventory differs')


def cpio_entries(path):
    if path.stat().st_size>64*1024**2:raise ValueError('pristine initramfs exceeds its bound')
    raw=path.read_bytes();offset=0;seen=set();entries=[]
    while True:
        if raw[offset:offset+6]!=b'070701':raise ValueError('unsupported pristine initramfs')
        try:fields=[int(raw[offset+6+i*8:offset+14+i*8],16) for i in range(13)]
        except ValueError:raise ValueError('invalid initramfs header') from None
        length=fields[11]
        if not 1<length<=4096 or offset+110+length>len(raw):raise ValueError('initramfs name bound')
        encoded=raw[offset+110:offset+110+length]
        if encoded[-1:]!=b'\0':raise ValueError('initramfs name terminator missing')
        name=encoded[:-1].decode();start=(offset+110+length+3)//4*4;end=start+fields[6]
        if end>len(raw):raise ValueError('initramfs body truncated')
        offset=(end+3)//4*4
        if name=='TRAILER!!!':break
        relative(name)
        if name in seen or name=='payload.tar.gz' or stat.S_IFMT(fields[1]) not in (stat.S_IFREG,stat.S_IFDIR):raise ValueError('unexpected pristine initramfs entry')
        seen.add(name);entries.append((name,raw[start:end],fields[1]))
        if len(entries)>4096:raise ValueError('initramfs entry bound')
    if 'init' not in seen:raise ValueError('pristine init missing')
    return entries

def write_cpio(out,name,data,mode,inode):
    encoded=name.encode()+b'\0';fields=(inode,mode,0,0,1,0,len(data),0,0,0,0,len(encoded),0)
    out.write(b'070701'+b''.join(f'{n:08x}'.encode() for n in fields)+encoded);out.write(b'\0'*(-(110+len(encoded))%4));out.write(data);out.write(b'\0'*(-len(data)%4))


def gateway_sources(resume,peers=False,publication=False,isolation=False,initial_publication=False):
    if type(resume) is not bool or type(peers) is not bool or peers and not resume:raise ValueError('explicit boolean gateway-resume selection required')
    if type(publication) is not bool or publication and not peers:raise ValueError('publication requires explicit retained-peer recovery')
    if type(isolation) is not bool or isolation and (resume or peers or publication):raise ValueError('project isolation requires its own fresh two-project case')
    if type(initial_publication) is not bool or initial_publication and (resume or peers or publication or isolation):raise ValueError('initial publication requires its own fresh private-chain case')
    files={'guest.py':HERE/('docker_resume_guest.py' if resume else 'docker_guest.py'),
           **{name:HERE/name for name in ('docker_gateway.cjs','docker_peer.py','docker_case.py','docker_runtime.py')}}
    if resume:files.update({name:HERE/name for name in ('docker_resume_case.py','docker_gateway_resume.cjs','gateway_resume_contract.cjs')})
    if peers:files.update({'guest.py':HERE/'docker_gateway_peers_guest.py','docker_gateway_peers.py':HERE/'docker_gateway_peers.py'})
    if publication:files.update({'guest.py':HERE/'docker_publication_guest.py',**{name:HERE/name for name in ('docker_publication.py','docker_published_client.py')}})
    if isolation:files.update({'guest.py':HERE/'docker_isolation_guest.py','docker_isolation.py':HERE/'docker_isolation.py'})
    if initial_publication:files.update({'guest.py':HERE/'docker_initial_publication_guest.py',**{name:HERE/name for name in ('docker_initial_publication.py','docker_publication.py','docker_published_client.py','docker_gateway_peers.py')}})
    return files

def validate_inputs(args):
    gateway_sources(getattr(args,'gateway_resume',False),getattr(args,'gateway_peer_recovery',False),getattr(args,'http_publication',False),getattr(args,'project_isolation',False),getattr(args,'initial_http_publication',False))
    roots={key:getattr(args,key).resolve(strict=True) for key in
           ('base_appliance','tools','sources','compiled','deployment','node_runtime')}
    if any(not root.is_dir() for root in roots.values()):raise ValueError('input directories required')
    out=args.output.absolute()
    parent=out.parent.resolve(strict=True);out=parent/out.name
    if out.exists() or out.is_symlink():raise ValueError('fresh output required; retain any partial preparation')
    if any(out.is_relative_to(root) for root in roots.values()):raise ValueError('output must be outside every input tree')
    roots['output']=out
    base=read_json(roots['base_appliance']/'build.json')
    if base.get('root_disk_gib')!=12 or base.get('base_source') or base.get('exercise_private_chain'):
        raise ValueError('pristine 12GiB foundation appliance required')
    # A pristine build may declare a different exercise: no exercise has run in
    # base.raw. Actual launches always write a separate qcow2 overlay.
    for name,digest in base['artifacts'].items():
        # Base files may have been relocated to a verified read-only evidence
        # directory; resolve that explicit file and bind its digest, not its path.
        relative(name);path=(roots['base_appliance']/'artifacts'/name).resolve(strict=True)
        if not path.is_file() or sha(path)!=digest:raise ValueError('pristine artifact differs: '+name)
    for name in ('base.raw','vmlinuz','initramfs.cpio','init'):
        if name not in base['artifacts']:raise ValueError('pristine artifact missing: '+name)
    if (roots['base_appliance']/'artifacts/base.raw').stat().st_size!=12*1024**3:raise ValueError('pristine disk size differs')
    old_guest=base['source_sha256']['scripts/qualification/imports/guest.py'];old_runner=base['runner_sha256']
    if not all(isinstance(v,str) and re.fullmatch('[0-9a-f]{64}',v) for v in (old_guest,old_runner)):
        raise ValueError('pristine replacement bindings missing')
    pins=read_json(HERE/'inputs.json')
    for name,meta in pins['sources'].items():
        path=checked_file(roots['sources'],name,meta['sha256'])
        if path.stat().st_size!=meta['bytes']:raise ValueError('pinned source size differs')
    checked_file(roots['tools'],'package-lock.json',pins['package_lock_sha256'])
    for name,version in pins['packages'].items():
        if read_json(checked_file(roots['tools'],'node_modules/'+name+'/package.json'))['version']!=version:
            raise ValueError('package version differs: '+name)
    compiled=read_json(checked_file(roots['compiled'],'result.json'))
    if compiled.get('passed') is not True:raise ValueError('successful offline compilation required')
    checked_file(roots['compiled'],'compiler-output.json',compiled['compilerOutputSha256'])
    if any(compiled['sources'].get(name)!=meta['sha256'] for name,meta in pins['sources'].items() if name.endswith('.sol')):
        raise ValueError('compilation source pins differ')
    deployment=read_json(checked_file(roots['deployment'],'result.json'))
    outer=read_json(checked_file(roots['deployment'],'outer-result.json'))
    if deployment.get('passed') is not True or outer!={'exit':0,'scope_admitted':True,'scope_inactive':True}:
        raise ValueError('stopped verified foundation deployment required')
    before=deployment['before']
    if before!=deployment['after'] or before['chainId']!='0x539' or before['block']!='0xd':
        raise ValueError('original unkeyed foundation chain required')
    if deployment['compiler']!=compiled['compiler'] or deployment['node']!=compiled['node']:
        raise ValueError('compiler/runtime provenance differs')
    chain=roots['deployment']/'chain'
    if chain.is_symlink() or not chain.is_dir():raise ValueError('foundation chain directory required')
    closure=read_json(checked_file(roots['node_runtime'],'closure.json'))
    if closure.get('extra_libraries_copied') is not True:raise ValueError('complete Node/addon library set required')
    node=(roots['node_runtime']/'node').resolve(strict=True)
    if not node.is_relative_to(roots['node_runtime']):raise ValueError('Node runtime leaves selected bundle')
    node_files={}
    for group in ('original_node_closure','addon_libraries'):
        for name,meta in closure[group].items():
            path=checked_file(node,name,meta['sha256'])
            if path.stat().st_size!=meta['bytes']:raise ValueError('Node library size differs')
            if name in node_files and node_files[name]!=meta['sha256']:raise ValueError('conflicting Node library binding')
            node_files[name]=meta['sha256']
    checked_file(node,'usr/bin/node',NODE_SHA)
    if len(closure['addons'])!=14 or len({a['path'] for a in closure['addons']})!=14:raise ValueError('qualified addon inventory required')
    for addon in closure['addons']:
        checked_file(roots['tools'],addon['path'],addon['sha256'])
        if any(name not in node_files for name in addon['libraries']):raise ValueError('addon library missing')
    runner=args.runner.resolve(strict=True);vere=args.vere.resolve(strict=True)
    for path in (runner,vere):
        if not path.is_file() or not 0<path.stat().st_size<=256*1024**2:raise ValueError('bounded regular executable input required')
        with path.open('rb') as stream:
            if stream.read(4)!=b'\x7fELF':raise ValueError('ELF executable input required')
    if sha(vere)!=VERE_SHA:raise ValueError('pinned Vere differs')
    roots.update(runner=runner,vere=vere,node=node)
    expected={'src/urbit-runner':sha(runner),'src/node-closure.json':sha(roots['node_runtime']/'closure.json'),
              'tools/package-lock.json':pins['package_lock_sha256'],
              'source/bridge/package.json':pins['sources']['bridge/package.json']['sha256'],
              'compiled/result.json':sha(roots['compiled']/'result.json'),
              'compiled/compiler-output.json':compiled['compilerOutputSha256'],
              'deployment/result.json':sha(roots['deployment']/'result.json'),'vere':VERE_SHA}
    expected.update({'node/'+name:digest for name,digest in node_files.items()})
    expected.update({'tools/'+addon['path']:addon['sha256'] for addon in closure['addons']})
    return roots,base,node_files,expected


def seal_payload(path,items):
    with path.open('xb') as out:
        with gzip.GzipFile(filename='',mode='wb',fileobj=out,compresslevel=3,mtime=0) as gz:
            with tarfile.open(fileobj=gz,mode='w|',dereference=True) as tar:
                writer=PayloadWriter(tar)
                for source,name,root in items:writer.add(source,name,root)
    verify_payload(path,writer.inventory)
    return writer.inventory


def prepare(args):
    roots,base,node_files,expected=validate_inputs(args)
    out=roots['output']
    admission={'temporary':storage.admit('/tmp',8*1024**3),'output':storage.admit(out.parent,8*1024**3)}
    entries=cpio_entries(roots['base_appliance']/'artifacts/initramfs.cpio')
    os.umask(0o077);out.mkdir(mode=0o700);artifacts=out/'artifacts';artifacts.mkdir();source=out/'source';source.mkdir()
    # Frozen source inputs are the only code passed into the fresh guest.
    files={'urbit-runner':roots['runner'],'node-closure.json':roots['node_runtime']/'closure.json',
           **gateway_sources(getattr(args,'gateway_resume',False),getattr(args,'gateway_peer_recovery',False),getattr(args,'http_publication',False),getattr(args,'project_isolation',False),getattr(args,'initial_http_publication',False))}
    for name in ('network_case.py','state_case.py','omabit-state.hoon'):
        path=HERE.parent/'fake-networks'/name
        if sha(path)!=base['source_sha256']['scripts/qualification/fake-networks/'+name]:
            raise ValueError('foundation networking fixture differs: '+name)
        files[name]=path
    for name,path in files.items():
        expected['src/'+name]=expected.get('src/'+name,sha(path))
        shutil.copyfile(path,source/name)
    items=[(source,'src',source),(roots['tools'],'tools',roots['tools']),
           (roots['sources']/'bridge/package.json','source/bridge/package.json',roots['sources']),
           (roots['deployment']/'result.json','deployment/result.json',roots['deployment']),
           (roots['deployment']/'chain','deployment/chain',roots['deployment']),
           (roots['vere'],'vere',roots['vere'].parent)]
    items.extend((roots['compiled']/name,'compiled/'+name,roots['compiled']) for name in ('result.json','compiler-output.json'))
    items.extend((roots['node']/name,'node/'+name,roots['node']) for name in sorted(node_files))
    payload=out/'payload.tar.gz';inventory=seal_payload(payload,items)
    if any(inventory[name]['sha256']!=digest for name,digest in expected.items()):
        raise ValueError('verified input changed before payload publication')
    write_json(out/'payload-inventory.json',inventory);payload_sha=sha(payload)
    print(json.dumps({'stage':'payload_sealed','files':len(inventory),'bytes':sum(m['bytes'] for m in inventory.values())}),flush=True)
    init=(IMPORTS/'guest-init/main.go').read_text();anchor='\t// The host builds'
    if init.count(anchor)!=1:raise ValueError('guest admission insertion point differs')
    (source/'main.go').write_text(init.replace(anchor,'\tinstallPayload()\n'+anchor))
    helper=(HERE/'payload/payload.go').read_text()
    if not helper.startswith('package payload\n'):raise ValueError('payload package declaration differs')
    (source/'payload.go').write_text(helper.replace('package payload\n','package main\n',1))
    (source/'payload_config.go').write_text('''package main
var payloadSHA, previousGuestSHA, previousRunnerSHA, candidateRunnerSHA string
func installPayload() {
    must(Install(Config{PayloadSHA:payloadSHA, PreviousGuestSHA:previousGuestSHA,
        PreviousRunnerSHA:previousRunnerSHA, RunnerSHA:candidateRunnerSHA}))
}
''')
    bindings={'payloadSHA':payload_sha,'previousGuestSHA':base['source_sha256']['scripts/qualification/imports/guest.py'],
              'previousRunnerSHA':base['runner_sha256'],'candidateRunnerSHA':inventory['src/urbit-runner']['sha256']}
    env=dict(os.environ,CGO_ENABLED='0',GOOS='linux',GOARCH='amd64',GOTOOLCHAIN='local',GOPROXY='off',GOSUMDB='off',GOFLAGS='',GOENV='off')
    subprocess.run(['go','build','-trimpath','-ldflags',' '.join('-X main.'+key+'='+value for key,value in bindings.items()),
                    '-o',str(artifacts/'init'),str(source/'main.go'),str(source/'payload.go'),str(source/'payload_config.go')],
                   cwd=REPO,env=env,check=True,timeout=180)
    with (artifacts/'initramfs.cpio').open('xb') as stream:
        for i,(name,data,mode) in enumerate(entries,1):
            write_cpio(stream,name,(artifacts/'init').read_bytes() if name=='init' else data,mode,i)
        write_cpio(stream,'payload.tar.gz',payload.read_bytes(),0o100644,len(entries)+1)
        write_cpio(stream,'TRAILER!!!',b'',0,len(entries)+2)
    shutil.copyfile(roots['base_appliance']/'artifacts/vmlinuz',artifacts/'vmlinuz')
    (artifacts/'base.raw').touch(exist_ok=False)
    # A successful manifest is the publication boundary; an incomplete output is
    # preserved and cannot be reused. The common launcher rechecks these hashes.
    build={'exercise_private_chain':True,'import_mode':'rename','root_disk_gib':12,
           'pristine_parent':str(roots['base_appliance']),'base_source':str(roots['base_appliance']/'artifacts/base.raw'),
           'base_source_sha256':base['artifacts']['base.raw'],
           'pristine_parent_build_sha256':sha(roots['base_appliance']/'build.json'),
           'payload_sha256':payload_sha,'payload_inventory_sha256':sha(out/'payload-inventory.json'),
           'input_files':len(inventory),'expanded_input_bytes':sum(m['bytes'] for m in inventory.values()),
           'artifacts':{p.name:sha(p) for p in artifacts.iterdir()},'source_sha256':{p.name:sha(p) for p in source.iterdir()},
           'preparer_sha256':sha(Path(__file__)),'pins_sha256':sha(HERE/'inputs.json'),'storage_admission':admission}
    if getattr(args,'gateway_resume',False):build['gateway_resume_qualification']=True
    if getattr(args,'gateway_peer_recovery',False):build['gateway_peer_recovery_qualification']=True
    if getattr(args,'http_publication',False):build['http_publication_qualification']=True
    if getattr(args,'project_isolation',False):build['private_project_isolation_qualification']=True
    if getattr(args,'initial_http_publication',False):build['initial_http_publication_qualification']=True
    write_json(out/'build.json',build);payload.unlink()
    print(json.dumps({'stage':'prepared','output':str(out),'build_sha256':sha(out/'build.json')}),flush=True)


def main():
    parser=argparse.ArgumentParser(description=__doc__)
    for key,help_text in (
        ('base-appliance','Pristine 12GiB import appliance; never a runtime overlay'),
        ('tools','Pinned offline JavaScript tools'),('sources','Pinned Bridge/Azimuth source tree'),
        ('compiled','Passed offline compiler output'),('deployment','Stopped original unkeyed foundation deployment'),
        ('node-runtime','Complete verified Node/addon runtime bundle'),('runner','Candidate ELF runner; never executed on host'),
        ('vere','Pinned official standalone Vere'),('output','Fresh owner-only preparation directory')):
        parser.add_argument('--'+key,type=Path,required=True,help=help_text)
    parser.add_argument('--gateway-resume',action='store_true',help='Append the retained-state gateway experiment after the complete fresh peer case')
    parser.add_argument('--gateway-peer-recovery',action='store_true',help='With --gateway-resume, recover retained peers, client sessions and subscription intent')
    parser.add_argument('--http-publication',action='store_true',help='With both recovery flags, publish only two private guest-loopback HTTP ports and use a direct client')
    parser.add_argument('--project-isolation',action='store_true',help='Keep two fresh keyed projects live together; cannot combine with gateway recovery/publication')
    parser.add_argument('--initial-http-publication',action='store_true',help='Publish only two private guest-loopback HTTP ports from first initialization; excludes recovery/isolation modes')
    prepare(parser.parse_args())

if __name__=='__main__':main()
