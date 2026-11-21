"""Whole-message bounded serial status; full private proof remains on the guest disk."""
import json
import re

KIND = 'issues16-project-transition-compact-v1'
MARKER = 'OMABIT_PROJECT_TRANSITION_RESULT '
MAX_BYTES = 2048
GATES = ('native_transition', 'single_sigterm', 'terminal_stop_succeeded', 'receipt_replay_after_restart', 'same_pier_start', 'peer_anchor_preserved', 'base_stopped_app_preserved', 'final_clean_stops')


def serialize(report):
    source = report.get('source_commit', '')
    proof = report.get('proof_sha256', '')
    gates = {key: report.get(key) is True for key in GATES}
    valid = isinstance(source, str) and re.fullmatch('[0-9a-f]{40}', source) and isinstance(proof, str) and re.fullmatch('[0-9a-f]{64}', proof)
    passed = report.get('passed') is True and bool(valid) and all(gates.values())
    value = {'kind': KIND, 'passed': passed, 'source_commit': source if isinstance(source, str) and re.fullmatch('[0-9a-f]{40}', source) else '', 'proof_sha256': proof if isinstance(proof, str) and re.fullmatch('[0-9a-f]{64}', proof) else '', 'gates': gates, 'error': '' if passed else 'guest_checks_failed; inspect retained complete report'}
    raw = json.dumps(value, separators=(',', ':'), ensure_ascii=True)
    if len(raw.encode('ascii')) > MAX_BYTES:
        raise RuntimeError('compact serial status exceeds bound')
    return raw


def read_log(text, source):
    matches = [line.split(MARKER, 1)[1] for line in text.splitlines() if MARKER in line]
    if not matches:
        return None
    if len(matches) != 1 or len(matches[0].encode('utf-8')) > MAX_BYTES:
        raise RuntimeError('duplicate or oversized compact status')
    v = json.loads(matches[0])
    if not isinstance(v, dict) or set(v) != {'kind', 'passed', 'source_commit', 'proof_sha256', 'gates', 'error'} or v['kind'] != KIND or type(v['passed']) is not bool:
        raise RuntimeError('wrong compact status protocol')
    if v['source_commit'] not in ('', source) or not isinstance(v['gates'], dict) or set(v['gates']) != set(GATES) or any(type(x) is not bool for x in v['gates'].values()):
        raise RuntimeError('foreign or malformed compact status')
    if v['passed'] and (v['source_commit'] != source or not isinstance(v['proof_sha256'], str) or not re.fullmatch('[0-9a-f]{64}', v['proof_sha256']) or not all(v['gates'].values()) or v['error'] != ''):
        raise RuntimeError('incomplete compact success')
    return v
