"""Durable phase bookkeeping for the disposable runner app probe; no runtime access."""
import copy
import re

PHASES=('setup-before-first','after-first','before-second','after-second')
PEER_PHASES=PHASES+('before-peer','after-peer')
ANCHOR_PHASES=PEER_PHASES+('before-anchor','after-anchor')

def phase_order(state):
    mode=state.get('peer_cycle',False)
    if type(mode) is not bool:raise ValueError('invalid peer cycle mode')
    anchor=state.get('anchor_cycle',False)
    if type(anchor) is not bool or anchor and not mode:raise ValueError('invalid anchor cycle mode')
    return ANCHOR_PHASES if anchor else PEER_PHASES if mode else PHASES

def begin_phase(state, phase):
    phases=phase_order(state)
    if phase not in phases or type(state.get('format')) is not int or state['format']!=1:
        raise ValueError('invalid runner application fixture phase')
    attempts=state.get('attempts')
    if not isinstance(attempts,list) or len(attempts)>=len(phases):
        raise ValueError('fixture has no next phase')
    for expected,attempt in zip(phases,attempts):
        if not isinstance(attempt,dict) or attempt.get('phase')!=expected or attempt.get('status')!='complete':
            raise ValueError('retain uncertain fixture phase; never repeat its mutations')
    if phase!=phases[len(attempts)]:
        raise ValueError('fixture phase would replay or skip work')
    result=copy.deepcopy(state)
    result['attempts'].append({'phase':phase,'status':'running'})
    result['ok']=False
    return result

def restore_recovery(state, recovery):
    saved=state.get('recovery')
    if saved is None:
        if len(state['attempts'])!=1 or state['attempts'][0]['phase']!=PHASES[0]:
            raise ValueError('missing retained recovery state')
        return
    if not isinstance(saved,dict):raise ValueError('invalid retained recovery state')
    entries=saved.get('entries');channel=saved.get('channel');cursor=saved.get('cursor')
    if not isinstance(entries,list) or len(entries)>16 or any(not isinstance(x,str) or not re.fullmatch('omabit-state-[0-9a-f]{24}',x) for x in entries) or len(set(entries))!=len(entries):
        raise ValueError('invalid retained fixture entries')
    if channel is not None and (not isinstance(channel,str) or not re.fullmatch('/~/channel/omabit-state-[0-9a-f]{32}',channel)):
        raise ValueError('invalid retained fixture channel')
    if cursor is not None and (type(cursor) is not int or not 0<=cursor<10**16):
        raise ValueError('invalid retained fixture cursor')
    report=saved.get('report')
    if not isinstance(report,dict) or not isinstance(report.get('writes'),list) or len(report['writes'])!=len(entries) or any(not isinstance(w,dict) or w.get('state')!='acknowledged' or w.get('entry')!=entry or not isinstance(w.get('receipt'),str) or not re.fullmatch('omabit-receipt-[0-9a-f]{32}',w['receipt']) for w,entry in zip(report['writes'],entries)):
        raise ValueError('retain uncertain write; no fixture mutation replay')
    if len({w['receipt'] for w in report['writes']})!=len(entries):
        raise ValueError('duplicate fixture receipt')
    recovery.entries=copy.deepcopy(entries);recovery.channel=channel;recovery.cursor=cursor;recovery.report=copy.deepcopy(report)

def save_recovery(state,recovery):
    state['recovery']={'entries':copy.deepcopy(recovery.entries),'channel':recovery.channel,'cursor':recovery.cursor,'report':copy.deepcopy(recovery.report)}

# Main transport will admit the exact disposable VM before reading the private
# config or phase file. A phase is persisted BEFORE invoking Recovery actions.
# Separate invocations restore only entries/channel/cursor. Session files use the
# selected transport's private tmpfs storage; later phases never retrieve +code.


def execute_phase(state, phase, commit, work):
    """Persist once before effects; a lost completion never authorizes replay.

    The caller admits the disposable guest and owns private durable I/O. Work may
    update the recovery report but cannot rewrite phase history. A failed commit
    before work performs no effects; a failed final commit retains uncertain work.
    """
    current=begin_phase(state,phase)
    expected=copy.deepcopy(current['attempts'])
    commit(copy.deepcopy(current))
    work(current)
    if type(current.get('format')) is not int or current['format']!=1 or current.get('attempts')!=expected or phase_order(current)!=phase_order(state):
        raise ValueError('fixture work changed its phase binding')
    current['attempts'][-1]['status']='complete'
    current['ok']=phase==phase_order(state)[-1]
    commit(copy.deepcopy(current))
    return current
