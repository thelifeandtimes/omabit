"""One durable collection step through an explicitly supplied runtime adapter.

This is an internal driver, not a desktop command. The adapter owns verification
and reconciliation of effects after lost replies. It must live outside the shell,
bind observations to exact loaded instances, and refuse uncertain cleanup.
"""
import copy
import os
import journal as j


class Busy(Exception):
    """No receipt yet; retain the pending phase and retry observation later."""


class Problem(Exception):
    def __init__(self, reason):
        j.require(reason in j.PROBLEMS)
        self.reason = reason
        super().__init__(reason)


def step(store, lease, adapter):
    """Perform at most one member/phase, always after durable transition intent.

    perform(phase, request, specification, role) returns a verified observation.
    Its effect identity is j.effect(request, phase); repeated calls after process
    loss must observe/reconcile that effect, never assume it did not happen.
    Busy performs no journal write. Problems persist, and lineage uncertainty
    prevents subsequent adapter calls. Unexpected failures retain the old phase.
    Rollback is an explicit journal event, never inferred from an exception here.
    """
    j.require(lease.store is store and lease.live and lease.pid == os.getpid(),
              "a live coordinator lease is required before runtime effects")
    state = store.status()
    if state['pending'] is None:
        return state
    request = state['requests'][state['pending']]
    j.require(not request['uncertain'], "unconfirmed helper lineage prevents runtime effects")
    phase = request['phase']; effect = j.effect(request, phase)
    role = next((r for r in j.ROLES if effect+'/'+r not in request['receipts']), None) if phase.startswith('drain-') else None
    event = dict(type='drained' if role else 'completed', request=request['id'], phase=phase, effect=effect)
    if role: event['role'] = role
    try:
        observation = adapter.perform(phase, copy.deepcopy(request), copy.deepcopy(state['collections'][j.target(request, phase)]), role)
    except Busy:
        return state
    except Problem as error:
        event = dict(type='problem', request=request['id'], phase=phase, effect=effect, reason=error.reason)
    else:
        event['observation'] = observation
    return store.record(lease, event)


def recover_step(store, lease, adapter, request_id, recovery_id):
    """Prepare or verify one explicit recovery; no automatic restart cleanup.

    prepare_recovery(request, specification, recovery_id) is read-only toward
    runtime/application state and returns a verified frozen cohort. The adapter
    must exclude competing plugin work. recover(request, plan) may reconcile only
    the original guardian and exact transport channels, preserving application
    intent and credentials. It must revalidate admission/cohort on every retry.
    Only a separately durable prepared event permits recover(). Busy/exception
    leaves intent and uncertainty intact. The adapter supplies proof, not a user.
    """
    j.require(lease.store is store and lease.live and lease.pid == os.getpid(),
              "a live coordinator lease is required before recovery")
    j.token(request_id); j.token(recovery_id)
    state = store.status()
    j.require(request_id in state['requests'], "unknown collection transition")
    request = state['requests'][request_id]; saved = request['recovery']
    if saved is not None:
        j.require(saved['id'] == recovery_id, "a different recovery owns this transition")
        if saved['verified'] is not None:
            return store.record(lease, request['receipts']['recovery/'+recovery_id+'/recovery-verified'])
    j.require(state['pending'] == request_id and request['phase'] == 'drain-previous' and bool(request['uncertain']),
              "explicit recovery requires a pending fenced old activation")
    event = dict(request=request_id, phase=request['phase'], effect=j.effect(request, request['phase']), recovery=recovery_id)
    try:
        if saved is None:
            specification = state['collections'][request['previous']['collection']]
            plan = adapter.prepare_recovery(copy.deepcopy(request), copy.deepcopy(specification), recovery_id)
            event.update(type='recovery-prepared', plan=plan)
        else:
            j.recovery_plan(saved['plan'], request)
            # Retry the prepared event's directory fsync before any effect. A
            # previous rename may have succeeded while its fsync/response failed.
            store.record(lease, request['receipts']['recovery/'+recovery_id+'/recovery-prepared'])
            observation = adapter.recover(copy.deepcopy(request), copy.deepcopy(saved['plan']))
            event.update(type='recovery-verified', observation=observation)
    except Busy:
        return state
    return store.record(lease, event)
