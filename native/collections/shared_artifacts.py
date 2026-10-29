"""Format3 generations require shared-authority admission before construction.

Historical generations remain readable for cleanup; none is relabeled or claimed
to enforce the successor contract. Preparation never installs/enables a plugin.
"""
from pathlib import Path

import artifacts as a
import guarded_artifacts as g
import journal as j

FORMAT = 3
HOST, WIDGET = g.HOST, g.WIDGET
SUPPORT = g.SUPPORT + ('SharedHost.qml', 'runtime_authority.py', 'shared_admission.py', 'shared_admission_guard.py')


def generated(descriptor):
    files = g.generated(descriptor)
    primary, executable = files[HOST]
    old = b'Admission.Host {'
    guard = b'collection/admission_guard.py'
    j.require(primary.count(old) == 1 and primary.count(guard) == 1)
    files[HOST] = (primary.replace(old, b'Admission.SharedHost {').replace(guard, b'collection/shared_admission_guard.py'), executable)
    files['collection/qmldir'] = (g.QMLDIR + b'SharedHost 1.0 SharedHost.qml\n', False)
    return files


def verify(root, expected):
    descriptor = a.object_json(a.read_file(root, a.META)[0])
    if type(descriptor.get('format')) is int and descriptor['format'] in (1, 2):
        return dict(g.verify(root, expected), shared_authority_capable=False)
    result = g._verify(root, expected, contract_format=FORMAT, support_names=SUPPORT, derive=generated)
    return dict(result, shared_authority_capable=True)


def observation(root, expected, value):
    record = verify(root, expected)
    j.require(record['shared_authority_capable'], 'historical generation lacks shared-authority admission')
    descriptor = record['descriptor']
    j.exact(value, ('format', 'plugin_id', 'profile', 'artifact_sha256', 'instance', 'source'))
    j.require(type(value['format']) is int and value['format'] == 1)
    j.require(value['source'] == (Path(root) / a.WRAPPER).as_uri(), 'loaded URL differs from generation')
    member = {key: value[key] for key in ('plugin_id', 'profile', 'artifact_sha256', 'instance')}
    j.binding(member, {'plugin_id': descriptor['plugin_id'], 'profile': descriptor['profile'], 'artifact_sha256': expected})
    return member


def prepare(source, expected_inventory, store, *, role, profile, environment, admission_source, expected_admission):
    return g._prepare(source, expected_inventory, store, role=role, profile=profile, environment=environment,
                      admission_source=admission_source, expected_admission=expected_admission,
                      contract_format=FORMAT, support_names=SUPPORT, derive=generated, verifier=verify)
