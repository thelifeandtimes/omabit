"""Format4 immutable generations declare their bundled runtime protocols.

Older generations remain readable for cleanup. Their missing declaration never
implies support for a newer recovery history. All inspection is static; no
application or bundled Python is imported to discover compatibility.
"""
import ast
import copy
from pathlib import Path

import artifacts as a
import guarded_artifacts as g
import shared_artifacts as previous
import journal as j

FORMAT = 4
HOST, WIDGET = g.HOST, g.WIDGET
PROTOCOL_FILE = 'runtime-protocol.json'
SUPPORT = previous.SUPPORT + (PROTOCOL_FILE,)
PROTOCOL = {'format': 1, 'admission': 2, 'authority': 1,
            'reservations': [1, 2, 3], 'journals': [1, 2, 3]}


def literal(source, name):
    """Read one literal module declaration without executing the source."""
    try:
        tree = ast.parse(source)
        declarations = [node.value for node in tree.body if isinstance(node, ast.Assign)
                        and any(isinstance(target, ast.Name) and target.id == name for target in node.targets)]
        j.require(len(declarations) == 1, 'missing or ambiguous runtime protocol declaration: ' + name)
        return ast.literal_eval(declarations[0])
    except (SyntaxError, TypeError, ValueError) as error:
        raise j.Conflict('invalid static runtime protocol declaration: ' + name) from error


def protocol(support):
    declared = a.object_json(support[PROTOCOL_FILE][0])
    j.exact(declared, PROTOCOL)
    # JSON booleans must not masquerade as protocol integers.
    j.require(j.canonical(declared) == j.canonical(PROTOCOL), 'unsupported generation runtime protocol')
    actual = {'format': 1,
              'admission': literal(support['shared_admission.py'][0], 'FORMAT'),
              'authority': literal(support['runtime_authority.py'][0], 'FORMAT'),
              'reservations': list(literal(support['runtime_authority.py'][0], 'RESERVATION_READ_FORMATS')),
              'journals': list(literal(support['journal.py'][0], 'READ_FORMATS'))}
    j.require(j.canonical(actual) == j.canonical(declared), 'runtime protocol differs from bundled readers')
    return copy.deepcopy(declared)


def verify(root, expected):
    descriptor = a.object_json(a.read_file(root, a.META)[0])
    if type(descriptor.get('format')) is int and descriptor['format'] in (1, 2, 3):
        return dict(previous.verify(root, expected), runtime_protocol=None)
    result = g._verify(root, expected, contract_format=FORMAT, support_names=SUPPORT, derive=previous.generated)
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
    j.require(set(expected_admission) == set(SUPPORT), 'explicit complete runtime protocol pin required')
    support = a.snapshot(admission_source, list(SUPPORT))
    j.require(a.inventory(support) == expected_admission, 'runtime support differs from pinned inventory')
    protocol(support)  # Refuse unsupported/mixed readers before creating a stage.
    return g._prepare(source, expected_inventory, store, role=role, profile=profile, environment=environment,
                      admission_source=admission_source, expected_admission=expected_admission,
                      contract_format=FORMAT, support_names=SUPPORT, derive=previous.generated, verifier=verify)
