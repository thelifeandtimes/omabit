"""Immutable runtime generations for the supported native plugin collection.

Preparation never publishes a top-level native manifest or enables a plugin. The
caller supplies a pinned, explicit runtime inventory (not a repository scan).
Loaded identity comes from generated QML at a unique generation URL; a registry
manifest is selection metadata only. These are same-user integrity checks, not a
sandbox or a cryptographic signature from the publisher.
"""
import copy
import fcntl
import hashlib
import json
import os
from pathlib import Path, PurePosixPath
import re
import stat
import uuid

import journal as j

FORMAT = 1
MAX_FILES = 4096
MAX_BYTES = 512 << 20
META = 'collection-artifact.json'
WRAPPER = 'CollectionRuntime.qml'
ROLES = {
    'omabit': ('thelifeandtimes.omabit', 'service', 'Service.qml'),
    'theme': ('thelifeandtimes.urbit-theme', 'service', 'Service.qml'),
    'dms': ('urbit.dms', 'overlay', 'Dms.qml'),
}
PROFILE = re.compile(r'[a-z][a-z0-9-]{0,31}\Z')


def relative(name):
    j.require(type(name) is str and len(name) <= 512 and '\\' not in name and '\x00' not in name)
    p = PurePosixPath(name)
    j.require(not p.is_absolute() and str(p) == name and all(x not in ('..', '.', '.git', '__pycache__') for x in p.parts))
    j.require(bool(p.parts) and all(not x.startswith('.') for x in p.parts), 'hidden runtime inputs are unsupported')
    return p


def directory(path):
    """Open without following any ancestor symlink, including the final root."""
    p = j.absolute(str(path)); fd = os.open('/', os.O_RDONLY | os.O_DIRECTORY | os.O_CLOEXEC)
    try:
        for part in p.parts[1:]:
            nxt = os.open(part, os.O_RDONLY | os.O_DIRECTORY | os.O_NOFOLLOW | os.O_CLOEXEC, dir_fd=fd)
            os.close(fd); fd = nxt
        return fd
    except BaseException:
        os.close(fd); raise


def read_file(root, name):
    p = relative(name); fd = directory(Path(root).joinpath(*p.parts[:-1]))
    try:
        file = os.open(p.name, os.O_RDONLY | os.O_NONBLOCK | os.O_NOFOLLOW | os.O_CLOEXEC, dir_fd=fd)
    finally:
        os.close(fd)
    with os.fdopen(file, 'rb') as stream:
        before = os.fstat(stream.fileno())
        j.require(stat.S_ISREG(before.st_mode) and before.st_nlink == 1 and before.st_size <= MAX_BYTES,
                  'runtime input must be a bounded, unlinked regular file')
        data = stream.read(MAX_BYTES + 1); after = os.fstat(stream.fileno())
        j.require(len(data) == before.st_size and (before.st_size, before.st_mtime_ns, before.st_ctime_ns) ==
                  (after.st_size, after.st_mtime_ns, after.st_ctime_ns), 'runtime input changed while reading')
        j.require(not before.st_mode & (stat.S_ISUID | stat.S_ISGID), 'privileged runtime files are unsupported')
        return data, bool(before.st_mode & 0o111)


def inventory(files):
    return {name: {'sha256': hashlib.sha256(data).hexdigest(), 'bytes': len(data), 'executable': executable}
            for name, (data, executable) in sorted(files.items())}


def snapshot(root, names):
    j.require(type(names) is list and 0 < len(names) <= MAX_FILES and len(set(names)) == len(names))
    files = {str(relative(name)): read_file(root, name) for name in sorted(names)}
    j.require(sum(len(data) for data, _ in files.values()) <= MAX_BYTES, 'runtime inventory exceeds size limit')
    return files


def object_json(data):
    def unique(pairs):
        result = {}
        for key, value in pairs:
            j.require(key not in result, 'duplicate artifact field'); result[key] = value
        return result
    return json.loads(data, object_pairs_hook=unique)


def settings(role, profile, environment):
    j.require(role in ROLES); j.token(profile, PROFILE)
    j.require(profile != 'default', 'collection preparation requires an explicit independent profile')
    if role != 'omabit':
        j.require(environment is None, 'Theme/DMs use ambient XDG roots plus their declared profile namespace')
        return None
    j.exact(environment, ('XDG_CONFIG_HOME', 'XDG_STATE_HOME', 'XDG_DATA_HOME', 'XDG_CACHE_HOME'))
    roots = list(environment.values())
    for path in roots: j.absolute(path)
    j.require(all(not j.overlaps(a,b) for i,a in enumerate(roots) for b in roots[i+1:]), 'profile roots overlap')
    return copy.deepcopy(environment)


def wrapper(descriptor):
    role, profile = descriptor['role'], descriptor['profile']
    _, _, entry = ROLES[role]; identity = j.digest(descriptor)
    plugin = descriptor['plugin_id']; env = descriptor['environment']
    attributes = ''
    if role == 'omabit':
        attributes = ('  profileName: ' + json.dumps(profile) + '\n  profileEnvironment: ' + j.canonical(env).decode() +
                      '\n  localSetupAllowed: false\n  toolsDirectory: Qt.resolvedUrl("bin")\n')
    template = '''import QtQuick
import Quickshell.Io
import "." as Runtime
Runtime.TYPE {
  id: collectionRoot
ATTRIBUTES  readonly property string collectionArtifact: ARTIFACT
  readonly property string collectionProfile: PROFILE
  readonly property string collectionPlugin: PLUGIN
  readonly property string collectionInstance: {
    var value = ""
    for (var i = 0; i < 4; i++) value += ("00000000" + Math.floor(Math.random()*4294967296).toString(16)).slice(-8)
    return value
  }
  function collectionObservation() {
    return {format:1, plugin_id:collectionPlugin, profile:collectionProfile,
      artifact_sha256:collectionArtifact, instance:collectionInstance,
      source:Qt.resolvedUrl("CollectionRuntime.qml").toString()}
  }
  IpcHandler {
    target: PLUGIN + ".collection"
    function identity(): string { return JSON.stringify(collectionRoot.collectionObservation()) }
    function drain(): string { return String(collectionRoot.stopTransport()) }
    function transport(): string { return JSON.stringify(collectionRoot.transportState) }
  }
}
'''
    values = {'TYPE':Path(entry).stem, 'ATTRIBUTES':attributes, 'ARTIFACT':json.dumps(identity),
              'PROFILE':json.dumps(profile), 'PLUGIN':json.dumps(plugin)}
    # Substitute once: literal path/name contents must never become template tokens.
    return re.sub(r'\b(TYPE|ATTRIBUTES|ARTIFACT|PROFILE|PLUGIN)\b', lambda m:values[m[0]], template).encode()


def transform(files, role, profile, environment):
    environment = settings(role, profile, environment)
    j.require(META not in files and WRAPPER not in files and 'application-profile.json' not in files,
              'input is already prepared or has a profile declaration')
    canonical_id, kind, entry = ROLES[role]
    j.require('manifest.json' in files and entry in files, 'missing supported runtime entry')
    manifest = object_json(files['manifest.json'][0])
    j.require(type(manifest.get('schemaVersion')) is int and manifest['schemaVersion'] == 1 and manifest.get('id') == canonical_id)
    j.require(type(manifest.get('name')) is str and 0 < len(manifest['name']) <= 160)
    j.require(type(manifest.get('version')) is str and 0 < len(manifest['version']) <= 80)
    j.require(type(manifest.get('kinds')) is list and kind in manifest['kinds']
              and set(manifest['kinds']) <= {kind, 'bar-widget'})
    j.require(manifest.get('entryPoints', {}).get(kind) == entry, 'unsupported bootstrap/runtime layout')
    for ep in manifest['entryPoints'].values():
        relative(ep); j.require(ep in files and ep.endswith('.qml') and '..' not in ep, 'unbundled or unsafe native entry point')
    if role == 'omabit':
        for tool in ('omabit', 'urbit-collector', 'urbit-runner'):
            j.require('bin/'+tool in files and files['bin/'+tool][1], 'Omabit requires paired tools')
        contract = object_json(files['DESKTOP.json'][0]) if 'DESKTOP.json' in files else {}
        j.require(contract.get('bootstrap_api') == 1 and contract.get('profile_state_schema') == 1,
                  'Omabit requires a compatible desktop/profile contract')
    payload = dict(files); manifest = copy.deepcopy(manifest)
    manifest['id'] = canonical_id + '.' + profile
    manifest['name'] += ' (' + profile + ')'
    if isinstance(manifest.get('barWidget'), dict):
        manifest['barWidget']['displayName'] = manifest['name']
    payload['manifest.json'] = (j.canonical(manifest)+b'\n', False)
    if role != 'omabit': payload['application-profile.json'] = (j.canonical({'version':1,'profile':profile})+b'\n', False)
    descriptor = {'format':FORMAT, 'role':role, 'plugin_id':manifest['id'], 'profile':profile,
                  'environment':environment, 'source':inventory(files), 'payload':inventory(payload)}
    return descriptor, payload


def _sync(path):
    fd = directory(path)
    try: os.fsync(fd)
    finally: os.close(fd)


def _publish(stage, final):
    # Caller holds the private store's lock. No replacing an existing generation.
    j.require(not os.path.lexists(final), 'generation appeared during preparation')
    os.rename(stage, final)
    _sync(final.parent)


def verify(root, expected):
    """Verify bytes and loaded wrapper against the separately pinned artifact ID."""
    j.token(expected, j.HASH); root = Path(root)
    descriptor = object_json(read_file(root, META)[0])
    j.exact(descriptor, ('format','role','plugin_id','profile','environment','source','payload'))
    j.require(type(descriptor['format']) is int and descriptor['format'] == FORMAT and j.digest(descriptor) == expected,
              'unknown or changed artifact descriptor')
    settings(descriptor['role'],descriptor['profile'],descriptor['environment'])
    j.require(descriptor['plugin_id'] == ROLES[descriptor['role']][0]+'.'+descriptor['profile'])
    names = []
    for directory_path, dirs, files in os.walk(root, followlinks=False):
        j.require(all(not (Path(directory_path)/d).is_symlink() for d in dirs), 'linked generation directory')
        names.extend(str((Path(directory_path)/n).relative_to(root)) for n in files)
        j.require(len(names) <= MAX_FILES+3)
    j.require(set(names) == set(descriptor['payload']) | {META, WRAPPER}, 'generation inventory changed')
    payload = snapshot(root, list(descriptor['payload']))
    j.require(inventory(payload) == descriptor['payload'], 'generation payload changed')
    j.require(read_file(root, WRAPPER) == (wrapper(descriptor), False), 'generation identity wrapper changed')
    manifest = object_json(payload['manifest.json'][0]); kind = ROLES[descriptor['role']][1]
    # All entry URLs change with the artifact, including nested QML/JS imports.
    selected = copy.deepcopy(manifest)
    selected['entryPoints'][kind] = WRAPPER
    selected['entryPoints'] = {key:'generations/'+expected+'/'+value for key,value in selected['entryPoints'].items()}
    selected['keepLoaded'] = True
    return {'artifact_sha256':expected, 'descriptor':descriptor, 'manifest':selected, 'path':str(root)}


def observation(root, expected, value):
    """Bind actual loaded IPC identity to verified bytes and the exact source URL.

    Call via the coordinator's independently bound shell process. This function
    neither locates a shell nor establishes process ownership or readiness.
    """
    result = verify(root, expected); descriptor = result['descriptor']
    j.exact(value, ('format','plugin_id','profile','artifact_sha256','instance','source'))
    j.require(type(value['format']) is int and value['format'] == 1)
    j.require(value['source'] == (Path(root)/WRAPPER).as_uri(), 'loaded component URL differs from the prepared generation')
    member = {key: value[key] for key in ('plugin_id','profile','artifact_sha256','instance')}
    expected_member = {'plugin_id':descriptor['plugin_id'], 'profile':descriptor['profile'], 'artifact_sha256':expected}
    j.binding(member,expected_member)
    return member


def prepare(source, expected_inventory, store, *, role, profile, environment):
    """Snapshot pinned inputs into an explicit private store; never install/activate.

    ``store`` is a new or previously created generations directory, mode0700.
    Exact retry verifies and re-syncs existing bytes. Interrupted hidden staging
    directories are retained; a retry publishes only a complete new generation.
    Omabit input must be its paired runtime tree, not the mutable bootstrap.
    """
    files = snapshot(Path(source), list(expected_inventory))
    j.require(inventory(files) == expected_inventory, 'source differs from pinned inventory')
    descriptor, payload = transform(files,role,profile,environment); identity = j.digest(descriptor)
    store = Path(store); fd = directory(store)
    try:
        info = os.fstat(fd)
        j.require(info.st_uid == os.geteuid() and stat.S_IMODE(info.st_mode) == 0o700, 'private generation store required')
        lock = os.open('.prepare.lock', os.O_RDWR|os.O_CREAT|os.O_NOFOLLOW|os.O_NONBLOCK|os.O_CLOEXEC, 0o600, dir_fd=fd)
        with os.fdopen(lock, 'r+b') as stream:
            info = os.fstat(stream.fileno())
            j.require(stat.S_ISREG(info.st_mode) and info.st_nlink == 1 and info.st_uid == os.geteuid() and stat.S_IMODE(info.st_mode) == 0o600)
            fcntl.flock(stream, fcntl.LOCK_EX)
            final = store/identity
            if not os.path.lexists(final):
                stage = store/('.preparing-'+uuid.uuid4().hex); stage.mkdir(mode=0o700)
                output = {**payload, META:(j.canonical(descriptor)+b'\n',False), WRAPPER:(wrapper(descriptor),False)}
                for name,(data,executable) in sorted(output.items()):
                    path = stage/relative(name); path.parent.mkdir(parents=True,exist_ok=True,mode=0o700)
                    with path.open('xb') as file:
                        file.write(data); file.flush(); os.fchmod(file.fileno(),0o555 if executable else 0o444); os.fsync(file.fileno())
                for path,_,_ in os.walk(stage,topdown=False):
                    os.chmod(path,0o555); _sync(path)
                verify(stage,identity)
                _publish(stage,final)
            result = verify(final,identity)
            # Re-sync a generation whose prior rename/reply may have succeeded.
            _sync(store)
            return result
    finally:
        os.close(fd)
