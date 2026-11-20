#!/usr/bin/python3
"""Native recording limits on one new disposable fake; never installed state."""
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import shutil
import struct
import subprocess
import sys
import time

I = Path('/opt/omabit-release-input')
O = Path('/opt/omabit-issues-results')
HOME = Path('/home/review')
FS = Path('/mnt/omabit-recording-pressure')
BASE = HOME/'.local/share/urbit-runner'
MIB = 1024**2
RESERVE = 512*MIB
STDOUT = b'OMABIT_PRESSURE_STDOUT_BEGIN\n' + bytes(1536*1024) + b'\nOMABIT_PRESSURE_STDOUT_END\n'
STDERR = b'OMABIT_PRESSURE_STDERR_BURST\n'
# This fixture writes only finite known bytes to its own newly selected
# container's output pipes. It does not supply runtime commands or touch a pier.
EMIT = "printf 'OMABIT_PRESSURE_STDOUT_BEGIN\\n' > /proc/1/fd/1; head -c 1572864 /dev/zero > /proc/1/fd/1; printf '\\nOMABIT_PRESSURE_STDOUT_END\\n' > /proc/1/fd/1; i=0; while [ \"$i\" -lt 64 ]; do printf 'OMABIT_PRESSURE_STDERR_BURST\\n' > /proc/1/fd/2; i=$((i+1)); done"
report = {'format': 1, 'passed': False, 'scope': 'fresh ordinary fake, installed opt-in worker; synthetic bounded native output, actual session limit and separate-filesystem free-space pressure; no existing-process adoption or automatic retention'}
common = pilot = None


def sha(path):
    with Path(path).open('rb') as f:
        return hashlib.file_digest(f, 'sha256').hexdigest()


def guard():
    if os.geteuid() != 0 or Path('/sys/class/dmi/id/sys_vendor').read_text().strip() != 'QEMU' or 'omabit.disposable-guest=release-upgrade-v1' not in Path('/proc/cmdline').read_text().split() or sys.argv[1:]:
        raise RuntimeError('refused outside fresh offline recording pressure guest')
    value = json.loads((I/'scenario.json').read_text())
    if value.get('kind') != 'issues20-recording-pressure-v1' or value.get('target_schema') != 65 or not value.get('development_snapshot'):
        raise RuntimeError('wrong exact pressure scenario')
    return value


def decode_session(directory):
    """Independent bounded decoder: verify every original header/frame/link."""
    metadata = (directory/'metadata.json').read_bytes()
    m = json.loads(metadata)
    magic = b'OMABIT-RECORDING/1\0'
    header_size = len(magic)+16+32+24+32
    paths = [directory/'frames.obr', *sorted(directory.glob('frames-*.obr'))]
    if not 1 <= len(paths) <= 128 or sum(p.stat().st_size for p in paths) > 8*MIB:
        raise RuntimeError('bounded native recording scan exceeded')
    sequence, previous, elapsed, total, frames = 1, bytes(32), 0, 0, 0
    streams = {1: bytearray(), 2: bytearray()}
    lifecycle = []
    for number, path in enumerate(paths, 1):
        raw = path.read_bytes(); total += len(raw)
        if len(raw) < header_size or raw[:len(magic)] != magic or raw[len(magic):len(magic)+16].hex() != m['session_id'] or raw[len(magic)+16:len(magic)+48] != hashlib.sha256(metadata).digest():
            raise RuntimeError('recording metadata/header binding differs')
        segment, first, epoch = struct.unpack('>QQq', raw[len(magic)+48:len(magic)+72])
        if (segment, first) != (number, sequence) or raw[header_size-32:header_size] != previous or epoch <= 0:
            raise RuntimeError('recording segment/sequence/chain differs')
        previous = hashlib.sha256(raw[:header_size]).digest(); offset = header_size
        while offset < len(raw):
            if len(raw)-offset < 61:
                raise RuntimeError('unexpected incomplete native frame')
            head = raw[offset:offset+29]; seq, wall, monotonic, kind, length = struct.unpack('>QqQBI', head)
            end = offset+29+length
            if seq != sequence or wall <= 0 or monotonic < elapsed or kind not in (1, 2, 4, 5) or not 1 <= length <= 65536 or end+32 > len(raw):
                raise RuntimeError('recording frame bounds/order differ')
            data = raw[offset+29:end]; digest = hashlib.sha256(previous+head+data).digest()
            if raw[end:end+32] != digest:
                raise RuntimeError('recording frame digest differs')
            if kind in streams:
                streams[kind].extend(data)
            else:
                lifecycle.append({'kind': kind, 'event': json.loads(data)})
            previous, elapsed, sequence, offset = digest, monotonic, sequence+1, end+32
            frames += 1
    return {'session_id': m['session_id'], 'segments': len(paths), 'encoded_bytes': total, 'frames': frames, 'last_digest': previous.hex(), 'stdout_bytes': len(streams[1]), 'stderr_bytes': len(streams[2]), 'stdout_sha256': hashlib.sha256(streams[1]).hexdigest(), 'stderr_sha256': hashlib.sha256(streams[2]).hexdigest(), 'burst_and_long_line_exact': bytes(streams[1]).count(STDOUT) == 1 and bytes(streams[2]).count(STDERR) == 64, 'lifecycle': lifecycle}


def configure(name, **changes):
    state = pilot.core('runtime.recording-controls.v1', {'include_private_data': True})['state']
    policy = dict(state['policy']); policy.update(changes)
    request = {'runner_id': report['runner_id'], 'request_id': name, 'expected_revision': state['revision'], 'expected_digest': state['digest'], 'kind': 'configure', 'policy': policy}
    applied = pilot.core('runtime.recording-control-apply.v1', {'request': request})
    replay = pilot.core('runtime.recording-control-apply.v1', {'request': request})
    if applied['replayed'] or not replay['replayed'] or applied['receipt'] != replay['receipt']:
        raise RuntimeError('policy replay changed original receipt')
    return applied['receipt']['state']


def process(container):
    v = common.inspect(container)
    row = common.instance('candidate', 'recording-zod', 'running')
    if not row or row['ship'] != '~zod' or common.cid(row) != container or row.get('project_id'):
        raise RuntimeError('fresh fixture container binding changed')
    return {k: v['State'][k] for k in ('Pid', 'StartedAt', 'Running')}


def emit(container):
    row = common.instance('candidate', 'recording-zod', 'running')
    if not row or row['ship'] != '~zod' or common.cid(row) != container or row.get('project_id'):
        raise RuntimeError('only this new ordinary fake may emit test bytes')
    before = process(container)
    common.docker('exec', '--user', '0:0', container, '/bin/sh', '-ec', EMIT, timeout=30)
    if process(container) != before or not before['Running']:
        raise RuntimeError('emission/recording failure disturbed native process')
    return before


def selected(label, container):
    rows = pilot.snapshot(label)
    matching = [x for x in rows if (x['status'].get('metadata') or {}).get('container_id') == container]
    if len(matching) != 1:
        raise RuntimeError('missing exact selected capture')
    return matching[0]


def incomplete(label, container):
    def finished():
        row = selected(label, container)
        return row if row['status'].get('result') and not row['status']['writer_present'] else None
    row = common.wait(finished, 30)
    capture = row['status']['result']['capture']
    if row['status']['state'] != 'capture-incomplete' or capture.get('gap_reason') != 'disk-pressure' or not capture['closed_durably'] or capture['stats']['KnownUnrecordedBytes'] <= 0:
        raise RuntimeError('bounded refusal lost explicit incomplete coverage')
    return row


def preserve(label, row):
    directory = BASE/'recordings'/row['status']['session_id']
    decoded = decode_session(directory)
    if any(decoded[k] != row[k] for k in ('stdout_bytes', 'stderr_bytes', 'stdout_sha256', 'stderr_sha256', 'frames', 'last_digest')):
        raise RuntimeError('independent decoder differs from actual helper')
    target = O/'recordings'/label
    target.parent.mkdir(exist_ok=True)
    shutil.copytree(directory, target)
    return {'snapshot': row, 'decoded': decoded, 'original_files': pilot.retained_session(row), 'exported_files': {str(p.relative_to(target)): sha(p) for p in target.rglob('*') if p.is_file()}}


def main():
    global common, pilot
    scenario = guard()
    if O.exists() or FS.exists() or any(Path(p).exists() for p in ('/etc/urbit-runner', '/srv/urbit/piers', str(BASE))):
        raise RuntimeError('pristine guest required; never resume')
    O.mkdir(mode=0o700)
    for name in ('omabit', 'urbit-collector', 'urbit-runner', 'managed.test'):
        path = I/name if name == 'managed.test' else I/'candidate'/name
        if sha(path) != scenario['candidate'][name]['sha256']:
            raise RuntimeError('candidate input differs')
    for name, pin in (('boot.pill', scenario['pill_sha256']), ('runtime.oci.tar', scenario['oci_sha256'])):
        if sha(I/name) != pin:
            raise RuntimeError('runtime input differs')
    def load(name, filename):
        spec = importlib.util.spec_from_file_location(name, I/filename); value = importlib.util.module_from_spec(spec); spec.loader.exec_module(value); return value
    common = load('pressure_common', 'issues-common.py'); pilot = load('pressure_pilot', 'recording-common.py')
    common.O, common.report, common.scenario = O, report, scenario
    pilot.O, pilot.report, pilot.common = O, report, common
    report.update(inputs=scenario, source_commit=scenario['source_commit'])
    common.run(['loginctl', 'enable-linger', 'review']); common.run(['systemctl', 'start', 'user@1000.service'])
    common.run(['ssh-keygen', '-q', '-t', 'ed25519', '-N', '', '-f', HOME/'operator'], True)
    host = common.install('candidate'); report['runner_id'] = host['runner_id']
    common.docker('load', '--input', I/'runtime.oci.tar')
    common.unit('stop')
    FS.mkdir(mode=0o700)
    common.run(['mount', '-t', 'tmpfs', '-o', 'size=640M,mode=0700,uid=1000,gid=1000,nosuid,nodev,noexec', 'omabit-recording-pressure', FS])
    (FS/'recordings').mkdir(mode=0o700); os.chown(FS/'recordings', 1000, 1000)
    destination = BASE/'recordings'
    destination.mkdir(mode=0o700); os.chown(destination, 1000, 1000)
    common.run(['mount', '--bind', FS/'recordings', destination])
    if destination.stat().st_dev == BASE.stat().st_dev or destination.stat().st_dev != FS.stat().st_dev:
        raise RuntimeError('pressure filesystem not isolated from job/state/pier filesystem')
    override = HOME/'.config/systemd/user/urbit-runner.service.d'; override.mkdir(mode=0o700, parents=True)
    (override/'recording-pilot.conf').write_text('[Service]\nExecStart=\nExecStart=/usr/bin/env -i HOME=/home/review PATH=/usr/bin:/bin LANG=C XDG_RUNTIME_DIR=/run/user/1000 /usr/local/lib/urbit-runner/current/urbit-runner serve-development --acknowledge-development --record-runtime\n')
    for p in (override, override.parent, override.parent.parent, override/'recording-pilot.conf'): os.chown(p, 1000, 1000)
    common.run(['systemctl', '--user', 'daemon-reload'], True); common.unit('start')
    hello = common.hello()
    if hello['build']['sha256'] != scenario['candidate']['urbit-runner']['sha256'] or 'runtime-recording-pilot-v1' not in hello['features']:
        raise RuntimeError('actual pilot build differs')
    common.run([I/'candidate/urbit-collector', 'enroll', 'local', '--socket', common.SOCK], True)
    configure('pressure-initial-policy', max_jobs=3, max_session_bytes=8*MIB, max_segment_bytes=64*1024, reserve_free_bytes=RESERVE)
    pilot.start_guest_trim()
    common.collector('candidate', 'create-fake', 'recording-zod', '--ship', '~zod', '--http-port', '19480', '--request-id', 'pressure-create', '--bootstrap', 'recommended', '--pill', str(I/'boot.pill'))
    original = None; proofs = {}
    for label in ('burst', 'session-limit', 'free-space'):
        if label != 'burst': configure('pressure-policy-'+label, max_session_bytes=MIB if label == 'session-limit' else 8*MIB)
        common.operation('candidate', 'recording-zod', 'start', 'pressure-start-'+label)
        current = common.wait(lambda: common.instance('candidate', 'recording-zod', 'running'), 600)
        container = common.cid(current)
        if original is None: original = current
        elif (current['ship'], current['pier_path']) != (original['ship'], original['pier_path']) or container == common.cid(original): raise RuntimeError('same pier was not retained')
        row = selected(label+'-ready', container); pilot.worker(row)
        if label == 'free-space':
            fs = os.statvfs(FS); free = fs.f_bavail*fs.f_frsize; fill = free-(RESERVE-MIB)
            if not 0 < fill < 160*MIB: raise RuntimeError('bounded isolated pressure allocation unavailable')
            with (FS/'filler').open('xb') as f: os.posix_fallocate(f.fileno(), 0, fill); os.fsync(f.fileno())
            after = os.statvfs(FS); report['pressure'] = {'allocated_filler_bytes': fill, 'free_before': free, 'free_after': after.f_bavail*after.f_frsize, 'reserve_bytes': RESERVE}
            if report['pressure']['free_after'] >= RESERVE: raise RuntimeError('actual filesystem pressure not established')
        native = emit(container)
        if label != 'burst':
            row = incomplete(label+'-incomplete', container)
            if label == 'free-space' and row['status']['result']['capture']['gap_saved']: raise RuntimeError('gap unexpectedly saved below actual reserve')
            saved = pilot.retained_session(row)
            if label == 'free-space': (FS/'filler').unlink()
            if process(container) != native or pilot.retained_session(row) != saved: raise RuntimeError('space recovery changed runtime or original prefix')
        common.operation('candidate', 'recording-zod', 'stop', 'pressure-stop-'+label)
        row = selected(label+'-stopped', container)
        if label == 'burst':
            pilot.closed(row); pilot.verify_api_prefix(row)
        elif pilot.retained_session(row) != saved: raise RuntimeError('Stop rewrote incomplete original recording')
        proofs[label] = preserve(label, row)
        if label == 'burst' and not proofs[label]['decoded']['burst_and_long_line_exact']: raise RuntimeError('native long line/burst bytes changed')
        if label == 'session-limit' and proofs[label]['decoded']['encoded_bytes'] > MIB: raise RuntimeError('session encoded limit exceeded')
        if common.docker('inspect', container, ok=False).returncode == 0: raise RuntimeError('terminal drain did not permit ordinary ephemeral cleanup')
        for old in proofs.values():
            if pilot.retained_session(old['snapshot']) != old['original_files']: raise RuntimeError('later session changed earlier bytes')
        report['cases'] = proofs; (O/'partial.json').write_text(json.dumps(report, indent=2)+'\n')
    # Retained jobs consume their immutable commitments; no silent eviction.
    before = sorted(p.name for p in (BASE/'recording-jobs').iterdir())
    row = common.instance('candidate', 'recording-zod', 'stopped')
    common.collector('candidate', 'operate', 'recording-zod', 'start', '--revision', str(row['config_revision']), '--request-id', 'pressure-capacity-refusal')
    def terminal():
        op = next((x for x in common.collector('candidate', 'snapshot')['operations'] if x['request_id'] == 'pressure-capacity-refusal'), None)
        return op if op and op['status'] not in ('queued', 'running', 'pending') else None
    refusal = common.wait(terminal, 30)
    if refusal['status'] != 'failed': raise RuntimeError('full job capacity accepted Start')
    report['capacity_refusal_receipt'] = refusal
    if sorted(p.name for p in (BASE/'recording-jobs').iterdir()) != before or not common.instance('candidate', 'recording-zod', 'stopped'): raise RuntimeError('capacity refusal created another job/runtime')
    common.operation('candidate', 'recording-zod', 'stop', 'pressure-capacity-final-stop')
    final = common.instance('candidate', 'recording-zod', 'stopped')
    if final['desired_state'] != 'stopped': raise RuntimeError('final saved intent not stopped')
    controls = pilot.core('runtime.recording-controls.v1', {'include_private_data': True})
    if controls['capacity']['jobs'] != 3 or controls['capacity']['committed_bytes'] != 17*MIB: raise RuntimeError('closed jobs lost immutable commitment')
    report.update(passed=True, qualification={'exact_native_burst_long_line': True, 'session_limit': True, 'actual_free_space_refusal': True, 'explicit_incomplete_coverage': True, 'capture_failure_preserves_runtime': True, 'same_pier_sessions': True, 'earlier_recordings_immutable': True, 'job_capacity_refusal_without_eviction': True, 'final_clean_stop': True}, controls=controls)


if __name__ == '__main__':
    try: main()
    except BaseException as error: report['error'] = type(error).__name__+': '+str(error)
    finally:
        if pilot is not None:
            pilot.trim_stop.set()
            if pilot.trim_thread is not None: pilot.trim_thread.join(timeout=12)
        if O.is_dir():
            try:
                jobs = BASE/'recording-jobs'
                report['saved_jobs'] = {p.parent.name+'/'+p.name: json.loads(p.read_text()) for p in jobs.glob('*/*.json') if p.is_file() and p.stat().st_size <= 65536}
                # tmpfs disappears on poweroff. Retain bounded diagnostic bytes
                # even on failure; only explicit closed-case checks qualify them.
                captures = BASE/'recordings'
                files = [p for p in captures.rglob('*') if p.is_file()]
                if sum(p.stat().st_size for p in files) > 24*MIB: raise RuntimeError('diagnostic recording copy exceeds bound')
                if captures.is_dir(): shutil.copytree(captures, O/'final-recording-files')
            except BaseException as error:
                report['diagnostic_error'] = type(error).__name__+': '+str(error)
                report['passed'] = False
            (O/'result.json').write_text(json.dumps(report, indent=2)+'\n')
        # Keep native worker/status/raw evidence private on the ended disk. The
        # host result is bounded and carries a content hash, never partial JSON.
        compact = {'kind': 'issues20-recording-pressure-compact-v1', 'passed': report['passed'], 'source_commit': report.get('source_commit'), 'gates': report.get('qualification', {}), 'proof_sha256': sha(O/'result.json') if (O/'result.json').is_file() else None, 'error': report.get('error', '')}
        print('OMABIT_RECORDING_PRESSURE_VM_RESULT '+json.dumps(compact, separators=(',', ':')), flush=True)
        try: guard(); subprocess.run(['fstrim', '/'], timeout=60, check=False); subprocess.run(['systemctl', 'poweroff'], check=True)
        except BaseException: pass
