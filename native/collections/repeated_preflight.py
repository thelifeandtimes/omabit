"""Read-only compatibility check for every immutable member of a collection.

A ready result binds the caller's specification and exact artifact hashes. It
neither admits a loader nor proves ship/backend readiness, process isolation or
runtime cleanup. Recheck at each effect boundary; do not cache readiness after
source changes or infer it from a selected native manifest.
"""
import copy

import journal as j
import repeated_artifacts as artifacts

REQUIRED = {'admission': 2, 'authority': 4, 'reservation': 3, 'journal': 3}


def requirements(value):
    j.exact(value, REQUIRED)
    j.require(all(type(version) is int and version > 0 for version in value.values()),
              'protocol requirements must be positive integer versions')
    return copy.deepcopy(value)


def check(specification, generations, required=None):
    spec = j.specification(specification)
    required = requirements(REQUIRED if required is None else required)
    j.exact(generations, j.ROLES)
    for root in generations.values():j.absolute(root)
    members = {}
    for role in j.ROLES:
        member = spec['members'][role]
        row = {'plugin_id': member['plugin_id'], 'artifact_sha256': member['artifact_sha256'],
               'path': generations[role], 'ready': False, 'artifact_format': None, 'runtime_protocol': None,
               'reason': None, 'next_action': None}
        try:
            artifact = artifacts.verify(generations[role], member['artifact_sha256'])
            descriptor = artifact['descriptor'];row['artifact_format'] = descriptor['format']
            j.require(descriptor['role'] == role and descriptor['plugin_id'] == member['plugin_id'] and
                      descriptor['profile'] == member['profile'], 'generation differs from selected collection member')
            supported = artifact['runtime_protocol'];row['runtime_protocol'] = supported
            if supported is None:
                row.update(reason='runtime-protocol-undeclared', next_action='prepare-compatible-generation')
            elif (supported['admission'] != required['admission'] or required['authority'] not in (supported['authorities'] if supported['format'] == 2 else [supported['authority']]) or
                  required['reservation'] not in supported['reservations'] or required['journal'] not in supported['journals']):
                row.update(reason='runtime-protocol-unsupported', next_action='use-compatible-runtime-and-generation')
            else:row['ready'] = True
        except (OSError, ValueError, KeyError, TypeError):
            row.update(reason='artifact-invalid', next_action='review-pinned-generation')
        members[role] = row
    return {'format': 1, 'collection': j.digest(spec), 'required': required,
            'ready': all(row['ready'] for row in members.values()), 'members': members}


def require_ready(specification, generations, required=None):
    result = check(specification, generations, required)
    blocked = [role + ':' + row['reason'] for role, row in result['members'].items() if not row['ready']]
    j.require(not blocked, 'collection runtime compatibility refused: ' + ', '.join(blocked))
    return result
