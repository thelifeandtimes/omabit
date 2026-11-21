#!/usr/bin/env python3
"""Package the reviewed #1145 backport, without a daemon or access to any pier.

Produces a deterministic OCI layout plus Docker-load manifest in one archive.
Only the frozen official base and exact reviewed executable are accepted.
No compilation or execution of the input binary occurs here.
"""
import argparse
import copy
import gzip
import hashlib
import io
import json
from pathlib import Path
import tarfile
import urllib.request

BASE = '47fbcf0710ded939931ce3c6408864cee5e8848c5765da0b51a90ca0b1ab1afd'
BASE_CONFIG = 'd6cc5107ed7fa92174e336af9c7f25ee433073bd9961e1c3ed23a225b866b245'
BINARY = '0774857cdb4460fc288bfa55e778249456a7d8bd7b6203922455d887a92fc96c'
NAME = 'omabit/vere:4.6-jsonfix-0774857c'
MAX_BLOB = 128 * 1024 * 1024


def sha(data):
    return hashlib.sha256(data).hexdigest()


def encode(value):
    return json.dumps(value, sort_keys=True, separators=(',', ':')).encode()


def add(tar, name, data, mode=0o644):
    entry = tarfile.TarInfo(name)
    entry.size, entry.mode = len(data), mode
    entry.uid = entry.gid = entry.mtime = 0
    tar.addfile(entry, io.BytesIO(data))


def build(binary, cache, output):
    data = binary.read_bytes()
    if sha(data) != BINARY or data[:6] != b'\x7fELF\x02\x01':
        raise ValueError('expected exact reviewed Linux x86-64 executable')
    if int.from_bytes(data[18:20], 'little') != 62:
        raise ValueError('wrong ELF machine')
    cache.mkdir(parents=True, exist_ok=True)
    output.mkdir(parents=True, exist_ok=False)
    token = None

    def fetch(digest, kind='blobs', size=None):
        nonlocal token
        path = cache / digest
        if path.exists():
            blob = path.read_bytes()
        else:
            if token is None:
                with urllib.request.urlopen('https://auth.docker.io/token?service=registry.docker.io&scope=repository:tloncorp/vere:pull', timeout=30) as response:
                    token = json.load(response)['token']
            req = urllib.request.Request('https://registry-1.docker.io/v2/tloncorp/vere/' + kind + '/sha256:' + digest,
                                         headers={'Authorization': 'Bearer ' + token, 'Accept': 'application/vnd.docker.distribution.manifest.v2+json'})
            with urllib.request.urlopen(req, timeout=60) as response:
                blob = response.read(MAX_BLOB + 1)
            if len(blob) > MAX_BLOB or sha(blob) != digest:
                raise ValueError('downloaded base blob checksum/size mismatch')
            path.write_bytes(blob)
        if sha(blob) != digest or size is not None and len(blob) != size:
            raise ValueError('base blob checksum/size mismatch')
        return blob

    base = json.loads(fetch(BASE, 'manifests'))
    if base['config']['digest'] != 'sha256:' + BASE_CONFIG:
        raise ValueError('wrong base config')
    config = json.loads(fetch(BASE_CONFIG, size=base['config']['size']))
    if config['os'] != 'linux' or config['architecture'] != 'amd64':
        raise ValueError('wrong base platform')
    blobs, unpacked, stock_sha = {}, [], None
    for index, desc in enumerate(base['layers']):
        digest = desc['digest'].removeprefix('sha256:')
        blob = fetch(digest, size=desc['size'])
        raw = gzip.decompress(blob)
        if 'sha256:' + sha(raw) != config['rootfs']['diff_ids'][index]:
            raise ValueError('base uncompressed layer mismatch')
        blobs[digest] = blob
        unpacked.append(raw)
        with tarfile.open(fileobj=io.BytesIO(raw)) as layer:
            for member in layer:
                if member.name.lstrip('./') == 'bin/urbit':
                    if not member.isfile():
                        raise ValueError('unexpected stock executable type')
                    stock_sha = sha(layer.extractfile(member).read())
    if stock_sha is None:
        raise ValueError('base contains no ordinary /bin/urbit')
    layer = io.BytesIO()
    with tarfile.open(fileobj=layer, mode='w', format=tarfile.USTAR_FORMAT) as tar:
        add(tar, 'bin/urbit', data, 0o755)
    raw = layer.getvalue()
    compressed = gzip.compress(raw, mtime=0)
    unpacked.append(raw)
    blobs[sha(compressed)] = compressed
    config['rootfs']['diff_ids'].append('sha256:' + sha(raw))
    config.setdefault('history', []).append({'created': '2026-11-19T02:00:00Z', 'created_by': 'Omabit reviewed Vere 4.6 PR1145 binary ' + BINARY})
    config['created'] = '2026-11-19T02:00:00Z'
    config.setdefault('config', {}).setdefault('Labels', {}).update({
        'org.opencontainers.image.source': 'https://github.com/thelifeandtimes/omabit',
        'org.omabit.vere.binary-sha256': BINARY,
        'org.omabit.vere.base': '5a7614f69bbc783736b08965fdc0a2534bb42e82',
        'org.omabit.vere.patch': 'fcc9dd214ae4af07e683490a6cc3f2ded1ab7d47',
    })
    config_blob = encode(config)
    config_sha = sha(config_blob)
    blobs[config_sha] = config_blob
    layers = copy.deepcopy(base['layers'])
    for desc in layers:
        desc['mediaType'] = 'application/vnd.oci.image.layer.v1.tar+gzip'
    layers.append({'mediaType': 'application/vnd.oci.image.layer.v1.tar+gzip', 'size': len(compressed), 'digest': 'sha256:' + sha(compressed)})
    manifest = encode({'schemaVersion': 2, 'mediaType': 'application/vnd.oci.image.manifest.v1+json',
                       'config': {'mediaType': 'application/vnd.oci.image.config.v1+json', 'digest': 'sha256:' + config_sha, 'size': len(config_blob)}, 'layers': layers})
    manifest_sha = sha(manifest)
    blobs[manifest_sha] = manifest
    index = encode({'schemaVersion': 2, 'manifests': [{'mediaType': 'application/vnd.oci.image.manifest.v1+json', 'digest': 'sha256:' + manifest_sha, 'size': len(manifest),
                      'platform': {'os': 'linux', 'architecture': 'amd64'}, 'annotations': {'org.opencontainers.image.ref.name': NAME}}]})
    archive = output / 'vere-4.6-jsonfix-0774857c.tar.gz'
    with archive.open('xb') as file, gzip.GzipFile(fileobj=file, mode='wb', filename='', mtime=0) as gz, tarfile.open(fileobj=gz, mode='w|', format=tarfile.USTAR_FORMAT) as tar:
        add(tar, 'oci-layout', encode({'imageLayoutVersion': '1.0.0'}))
        add(tar, 'index.json', index)
        for digest, blob in sorted(blobs.items()):
            add(tar, 'blobs/sha256/' + digest, blob)
        # Legacy Docker's load path binds the same config and uncompressed layers.
        # Extra OCI entries preserve the independently verified manifest identity.
        add(tar, config_sha + '.json', config_blob)
        paths = []
        for i, raw in enumerate(unpacked):
            path = 'layers/' + str(i) + '.tar'
            paths.append(path)
            add(tar, path, raw)
        add(tar, 'manifest.json', encode([{'Config': config_sha + '.json', 'RepoTags': [NAME], 'Layers': paths}]))
    report = {'format': 1, 'platform': 'linux/amd64', 'vere_version': '4.6-82f4558', 'binary_sha256': BINARY,
              'base_image_manifest_sha256': BASE, 'base_image_config_sha256': BASE_CONFIG, 'stock_binary_sha256': stock_sha,
              'image_manifest_sha256': manifest_sha, 'image_config_sha256': config_sha,
              'archive': archive.name, 'archive_sha256': sha(archive.read_bytes()), 'archive_bytes': archive.stat().st_size,
              'source_base': '5a7614f69bbc783736b08965fdc0a2534bb42e82', 'patch_revision': 'fcc9dd214ae4af07e683490a6cc3f2ded1ab7d47',
              'approval': 'https://github.com/nisfeb/talon/issues/112#issuecomment-6997450705',
              'qualification': 'packaging only; no executable or runtime lifecycle test performed'}
    (output / 'provenance.json').write_bytes(encode(report) + b'\n')
    print(json.dumps(report, indent=2))


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--binary', type=Path, required=True)
    parser.add_argument('--cache', type=Path, required=True)
    parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    build(args.binary, args.cache, args.output)
