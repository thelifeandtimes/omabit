"""Format2 supported generations: inert primary and widget admission hosts.

The original format1 writer remains independent. Its immutable bytes are readable
here for history, but explicitly report admission_capable=False. Preparation takes
separately pinned application and admission inventories; it never installs/enables.
"""
import copy
import fcntl
import os
from pathlib import Path
import stat
import uuid

import artifacts as a
import journal as j

FORMAT=2
HOST='CollectionHost.qml'
WIDGET='CollectionWidget.qml'
SUPPORT=('Host.qml','WidgetHost.qml','Registry.qml','admission.py','admission_guard.py','artifacts.py','journal.py')
QMLDIR=b'singleton Registry 1.0 Registry.qml\nHost 1.0 Host.qml\nWidgetHost 1.0 WidgetHost.qml\n'


def generated(d):
    plugin=d['plugin_id'];identity=j.digest(d)
    fields={'pluginId':plugin,'profile':d['profile'],'artifact':identity,'role':d['role']}
    primary='import QtQuick\nimport "collection" as Admission\nAdmission.Host {\n'
    primary+=''.join('  '+key+': '+j.canonical(value).decode()+'\n' for key,value in fields.items())
    primary+='  runtimeSource: Qt.resolvedUrl("CollectionRuntime.qml")\n  guardSource: Qt.resolvedUrl("collection/admission_guard.py")\n  collectionRegistry: Admission.Registry\n}\n'
    out={HOST:(primary.encode(),False),a.WRAPPER:(a.wrapper(d),False),'collection/qmldir':(QMLDIR,False)}
    # The descriptor pins the exact source widget path independently of registry
    # metadata. No static import of that path appears in this inert entry point.
    if d['widget'] is not None:
        widget='import QtQuick\nimport "collection" as Admission\nAdmission.WidgetHost {\n'
        for key,value in {'pluginId':plugin,'artifact':identity,'role':d['role']}.items():widget+='  '+key+': '+j.canonical(value).decode()+'\n'
        widget+='  widgetSource: Qt.resolvedUrl('+j.canonical(d['widget']).decode()+')\n  collectionRegistry: Admission.Registry\n}\n'
        out[WIDGET]=(widget.encode(),False)
    return out


def verify(root,expected):
    """Historical readability never upgrades an old runtime's admission contract."""
    root=Path(root);j.token(expected,j.HASH);d=a.object_json(a.read_file(root,a.META)[0])
    if type(d.get('format')) is int and d['format']==1:
        return dict(a.verify(root,expected),admission_capable=False)
    return _verify(root,expected,contract_format=FORMAT,support_names=SUPPORT,derive=generated)


def _verify(root,expected,*,contract_format,support_names,derive):
    root=Path(root);j.token(expected,j.HASH);d=a.object_json(a.read_file(root,a.META)[0])
    j.exact(d,('format','role','plugin_id','profile','environment','source','payload','admission','widget'))
    j.require(type(d['format']) is int and d['format']==contract_format and j.digest(d)==expected,'unknown or changed guarded artifact')
    a.settings(d['role'],d['profile'],d['environment'])
    j.require(d['plugin_id']==a.ROLES[d['role']][0]+'.'+d['profile'])
    j.require(type(d['admission']) is dict and set(d['admission'])==set(support_names),'unsupported admission bundle')
    payload=a.snapshot(root,list(d['payload']));j.require(a.inventory(payload)==d['payload'],'application payload changed')
    support=a.snapshot(root/'collection',list(support_names));j.require(a.inventory(support)==d['admission'],'admission bundle changed')
    manifest=a.object_json(payload['manifest.json'][0]);kind=a.ROLES[d['role']][1]
    j.require(d['widget']==manifest['entryPoints'].get('barWidget'),'widget entry differs')
    derived=derive(d)
    names=[]
    for path,dirs,files in os.walk(root,followlinks=False):
        j.require(all(not (Path(path)/name).is_symlink() for name in dirs),'linked generation directory')
        names.extend(str((Path(path)/name).relative_to(root)) for name in files)
        j.require(len(names)<=a.MAX_FILES+len(support_names)+5)
    j.require(set(names)==set(payload)|{a.META}|set(derived)|{'collection/'+n for n in support_names},'guarded generation inventory changed')
    for name,value in derived.items():j.require(a.read_file(root,name)==value,'generated entry changed')
    selected=copy.deepcopy(manifest);selected['entryPoints'][kind]=HOST
    if d['widget'] is not None:selected['entryPoints']['barWidget']=WIDGET
    selected['entryPoints']={key:'generations/'+expected+'/'+value for key,value in selected['entryPoints'].items()}
    selected['keepLoaded']=True
    return {'artifact_sha256':expected,'descriptor':d,'manifest':selected,'path':str(root),'admission_capable':True}


def observation(root,expected,value):
    record=verify(root,expected);j.require(record['admission_capable'],'legacy runtime has no guarded admission')
    d=record['descriptor'];j.exact(value,('format','plugin_id','profile','artifact_sha256','instance','source'))
    j.require(type(value['format']) is int and value['format']==1)
    j.require(value['source']==(Path(root)/a.WRAPPER).as_uri(),'loaded URL differs from generation')
    member={key:value[key] for key in ('plugin_id','profile','artifact_sha256','instance')}
    j.binding(member,{'plugin_id':d['plugin_id'],'profile':d['profile'],'artifact_sha256':expected});return member


def prepare(source,expected_inventory,store,*,role,profile,environment,admission_source,expected_admission):
    return _prepare(source,expected_inventory,store,role=role,profile=profile,environment=environment,
                    admission_source=admission_source,expected_admission=expected_admission,
                    contract_format=FORMAT,support_names=SUPPORT,derive=generated,verifier=verify)


def _prepare(source,expected_inventory,store,*,role,profile,environment,admission_source,expected_admission,contract_format,support_names,derive,verifier):
    files=a.snapshot(source,list(expected_inventory));j.require(a.inventory(files)==expected_inventory,'source differs from pinned inventory')
    j.require(set(expected_admission)==set(support_names),'explicit complete admission pin required')
    support=a.snapshot(admission_source,list(support_names));j.require(a.inventory(support)==expected_admission,'admission source differs from pin')
    j.require(not any(n in (HOST,WIDGET) or n.split('/')[0]=='collection' for n in files),'reserved collection paths')
    d,payload=a.transform(files,role,profile,environment);d['format']=contract_format;d['admission']=a.inventory(support)
    manifest=a.object_json(payload['manifest.json'][0]);d['widget']=manifest['entryPoints'].get('barWidget')
    j.require(set(manifest['entryPoints'])<= {a.ROLES[role][1],'barWidget'},'unsupported unguarded entry point')
    identity=j.digest(d);store=Path(store);fd=a.directory(store)
    try:
        info=os.fstat(fd);j.require(info.st_uid==os.geteuid() and stat.S_IMODE(info.st_mode)==0o700,'private generation store required')
        lock=os.open('.prepare.lock',os.O_RDWR|os.O_CREAT|os.O_NOFOLLOW|os.O_NONBLOCK|os.O_CLOEXEC,0o600,dir_fd=fd)
        with os.fdopen(lock,'r+b') as stream:
            info=os.fstat(stream.fileno());j.require(stat.S_ISREG(info.st_mode) and info.st_nlink==1 and info.st_uid==os.geteuid() and stat.S_IMODE(info.st_mode)==0o600)
            fcntl.flock(stream,fcntl.LOCK_EX);final=store/identity
            if not os.path.lexists(final):
                stage=store/('.preparing-'+uuid.uuid4().hex);stage.mkdir(mode=0o700)
                output={**payload,**{'collection/'+n:v for n,v in support.items()},**derive(d),a.META:(j.canonical(d)+b'\n',False)}
                for name,(data,executable) in sorted(output.items()):
                    path=stage/a.relative(name);path.parent.mkdir(parents=True,exist_ok=True,mode=0o700)
                    with path.open('xb') as file:
                        file.write(data);file.flush();os.fchmod(file.fileno(),0o555 if executable else 0o444);os.fsync(file.fileno())
                for path,_,_ in os.walk(stage,topdown=False):os.chmod(path,0o555);a._sync(path)
                verifier(stage,identity);a._publish(stage,final)
            result=verifier(final,identity);a._sync(store);return result
    finally:os.close(fd)
