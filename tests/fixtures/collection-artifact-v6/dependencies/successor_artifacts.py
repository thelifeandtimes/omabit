"""Format5 immutable generations with explicit authority1/2 reader support.

Historical writers and bundles remain unchanged. Static protocol and import
checks describe the pinned bundle; they are not a sandbox or a code audit.
"""
import ast
import copy
from pathlib import Path
import sys

import artifacts as a
import guarded_artifacts as g
import shared_artifacts as shared
import recovery_artifacts as previous
import journal as j

FORMAT = 5
HOST, WIDGET = g.HOST, g.WIDGET
PROTOCOL_FILE = 'successor-protocol.json'
SUPPORT = previous.SUPPORT + (
    'successor_authority.py', 'successor_admission.py', 'successor_admission_guard.py',
    'session_successor.py', 'session_checkpoint.py', 'session_cleanup.py',
    'authority_inspection.py', 'authority_provisioning.py', 'startup_recovery.py', PROTOCOL_FILE)
PROTOCOL = {'format': 2, 'admission': 2, 'authorities': [1, 2],
            'reservations': [1, 2, 3], 'journals': [1, 2, 3]}


def imports(support):
    """Require explicit local Python imports to resolve inside this bundle."""
    modules = {Path(name).stem for name in support if name.endswith('.py')}
    for name, (data, _) in support.items():
        if not name.endswith('.py'):continue
        try:tree = ast.parse(data)
        except (SyntaxError, ValueError) as error:
            raise j.Conflict('invalid bundled Python: ' + name) from error
        for node in ast.walk(tree):
            if isinstance(node, ast.Import):names = [alias.name for alias in node.names]
            elif isinstance(node, ast.ImportFrom):
                j.require(node.level == 0 and node.module, 'unsupported relative bundle import: ' + name)
                names = [node.module]
            else:continue
            for imported in names:
                root = imported.split('.')[0]
                j.require(root in modules or root in sys.stdlib_module_names,
                          'missing bundled dependency: ' + name + ' imports ' + imported)


def protocol(support):
    j.require(set(support) == set(SUPPORT), 'incomplete successor support bundle')
    # The original metadata still describes the original modules, without
    # relabeling their scalar authority declaration as a new reader list.
    legacy = previous.protocol(support)
    declared = a.object_json(support[PROTOCOL_FILE][0]);j.exact(declared, PROTOCOL)
    j.require(j.canonical(declared) == j.canonical(PROTOCOL), 'unsupported successor runtime protocol')
    literal = previous.literal
    j.require(literal(support['successor_authority.py'][0], 'FORMAT') == 2 and
              type(literal(support['successor_authority.py'][0], 'FORMAT')) is int,
              'unsupported linked authority writer')
    actual = {'format': 2,
              'admission': literal(support['successor_admission.py'][0], 'FORMAT'),
              'authorities': list(literal(support['successor_authority.py'][0], 'AUTHORITY_READ_FORMATS')),
              'reservations': legacy['reservations'], 'journals': legacy['journals']}
    j.require(j.canonical(actual) == j.canonical(declared), 'successor protocol differs from bundled readers')
    imports(support)
    return copy.deepcopy(declared)


def generated(descriptor):
    files = shared.generated(descriptor)
    data, executable = files[HOST];old = b'collection/shared_admission_guard.py'
    j.require(data.count(old) == 1, 'unexpected shared host guard')
    files[HOST] = (data.replace(old, b'collection/successor_admission_guard.py'), executable)
    return files


def verify(root, expected):
    descriptor = a.object_json(a.read_file(root, a.META)[0])
    if type(descriptor.get('format')) is int and descriptor['format'] in (1, 2, 3, 4):
        return previous.verify(root, expected)
    result = g._verify(root, expected, contract_format=FORMAT, support_names=SUPPORT, derive=generated)
    declared = protocol(a.snapshot(Path(root) / 'collection', list(SUPPORT)))
    return dict(result, shared_authority_capable=True, runtime_protocol=declared)


def observation(root, expected, value):
    record = verify(root, expected)
    j.require(record['shared_authority_capable'], 'historical generation lacks shared admission')
    descriptor = record['descriptor']
    j.exact(value, ('format', 'plugin_id', 'profile', 'artifact_sha256', 'instance', 'source'))
    j.require(type(value['format']) is int and value['format'] == 1)
    j.require(value['source'] == (Path(root) / a.WRAPPER).as_uri(), 'loaded URL differs from generation')
    member = {key: value[key] for key in ('plugin_id', 'profile', 'artifact_sha256', 'instance')}
    j.binding(member, {'plugin_id': descriptor['plugin_id'], 'profile': descriptor['profile'], 'artifact_sha256': expected})
    return member


def prepare(source, expected_inventory, store, *, role, profile, environment, admission_source, expected_admission):
    j.require(set(expected_admission) == set(SUPPORT), 'explicit complete successor support pin required')
    support = a.snapshot(admission_source, list(SUPPORT))
    j.require(a.inventory(support) == expected_admission, 'successor support differs from pinned inventory')
    protocol(support)
    return g._prepare(source, expected_inventory, store, role=role, profile=profile, environment=environment,
                      admission_source=admission_source, expected_admission=expected_admission,
                      contract_format=FORMAT, support_names=SUPPORT, derive=generated, verifier=verify)
