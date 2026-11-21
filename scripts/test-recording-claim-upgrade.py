#!/usr/bin/env python3
"""Rehearse actual original claim1 writes, byte-preserved legacy lookup and claim2 refusal.

Source/filesystem tests only. Requires --run, a clean candidate commit and a
fresh /tmp output. Never opens a runner, engine, existing recording or ship.
"""
import argparse
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import re
import subprocess

ROOT = Path(__file__).resolve().parents[1]
ORIGINAL = '85ac7c6892dd4934b64e5dacc98846001ab3b01d'


def digest(path):
    with path.open('rb') as stream:
        return hashlib.file_digest(stream, 'sha256').hexdigest()


def selected(path):
    return (path in ('go.mod', 'go.sum')
            or path.startswith('internal/') and path.endswith('.go') and (
                not path.endswith('_test.go') or path.startswith('internal/runtimequalify/'))
            or any(path.startswith(prefix) for prefix in (
                'internal/appdev/frameworks/', 'internal/appdev/samples/',
                'internal/rootless/assets/'))
            or path in ('internal/service/legacy-inventory.json',
                        'internal/runtimequalify/private_gateway_probe.cjs',
                        'internal/runtimequalify/private_member_key_probe.cjs'))


def snapshot(directory):
    files = {}
    for path in sorted(directory.rglob('*')):
        if path.is_symlink():
            raise RuntimeError('ordinary qualification files required')
        if path.is_file():
            files[str(path.relative_to(directory))] = digest(path)
    return files


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--output', type=Path)
    parser.add_argument('--storage-budget-gib', type=int, choices=(40, 48), default=40)
    parser.add_argument('--run', action='store_true')
    args = parser.parse_args()
    if not args.run:
        parser.print_help()
        return
    out = args.output
    if out is None or out.parent != Path('/tmp') or not re.fullmatch('omabit-recording-claim-upgrade-[0-9]{2}', out.name) or out.exists() or out.is_symlink():
        parser.error('fresh physical /tmp/omabit-recording-claim-upgrade-NN output required')
    if subprocess.check_output(['git', 'status', '--porcelain'], cwd=ROOT):
        raise RuntimeError('freeze a clean candidate before qualification')
    candidate = subprocess.check_output(['git', 'rev-parse', 'HEAD'], cwd=ROOT, text=True).strip()
    spec = importlib.util.spec_from_file_location('budget', ROOT/'scripts/qualification/issues-recovery/run.py')
    budget = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(budget)
    admission = budget.charge(256*1024**2, args.storage_budget_gib*1024**3)
    out.mkdir(mode=0o700)
    (out/'work').mkdir(mode=0o700)
    (out/'admission.json').write_text(json.dumps(admission)+'\n')
    source = out/'original-source'
    source.mkdir(mode=0o700)
    entries = subprocess.check_output(['git', 'ls-tree', '-rl', ORIGINAL, '--', 'internal', 'go.mod', 'go.sum'], cwd=ROOT, text=True).splitlines()
    inputs = {}
    for entry in entries:
        metadata, name = entry.split('\t', 1)
        mode, kind, blob, size = metadata.split()
        if not selected(name):
            continue
        if kind != 'blob' or mode not in ('100644', '100755') or Path(name).is_absolute() or '..' in Path(name).parts:
            raise RuntimeError('unexpected original source object')
        data = subprocess.check_output(['git', 'cat-file', 'blob', blob], cwd=ROOT)
        if len(data) != int(size) or hashlib.sha1(('blob '+str(len(data))+'\0').encode()+data).hexdigest() != blob:
            raise RuntimeError('original Git object differs')
        path = source/name
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_bytes(data)
        inputs[name] = {'git_blob': blob, 'sha256': digest(path), 'bytes': len(data)}
    helper = ROOT/'scripts/qualification/runtime-recordings/original_claim_v1_test.go.txt'
    (source/'internal/runtimequalify/qualification_original_test.go').write_bytes(helper.read_bytes())
    (out/'original-source.json').write_text(json.dumps({'source': ORIGINAL, 'scope': 'unchanged original runtime Go/embedded files and runtimequalify test sources; declared synthetic filesystem helper only', 'inputs': inputs, 'helper_sha256': digest(helper)}, indent=2)+'\n')
    env = dict(os.environ, GOCACHE=str(Path.home()/'.cache/go-build'), GOTELEMETRY='off', GOPROXY='off', GOMAXPROCS='2', GOTMPDIR=str(out/'work'), TMPDIR=str(out/'work'))
    commands = []

    def run(command, cwd=ROOT, extra=None):
        result = subprocess.run(command, cwd=cwd, env=dict(env, **(extra or {})), capture_output=True, text=True, timeout=120)
        commands.append({'command': [str(x) for x in command], 'cwd': str(cwd), 'exit': result.returncode, 'stdout': result.stdout, 'stderr': result.stderr})
        (out/'commands.json').write_text(json.dumps(commands, indent=2)+'\n')
        if result.returncode:
            raise RuntimeError('recording qualification command failed; original output retained')

    old = out/'original-writer.test'
    new = out/'candidate.test'
    run(['go', 'test', '-p=2', '-c', '-ldflags=-s -w', '-o', old, './internal/runtimequalify'], cwd=source)
    run(['go', 'test', '-p=2', '-c', '-ldflags=-s -w', '-o', new, './internal/runtimequalify'])
    run([old, '-test.run=^TestOriginalRecordingClaimWriter$', '-test.v'], extra={'OMABIT_RECORDING_CLAIM_UPGRADE_ROOT': str(out)})
    original = snapshot(out/'original')
    run([new, '-test.run=^TestRecordingActualOriginalClaimWriterQualification$', '-test.v'], extra={'OMABIT_RECORDING_CLAIM_UPGRADE_ROOT': str(out)})
    if snapshot(out/'original') != original:
        raise RuntimeError('candidate changed actual original writer files')
    successor = snapshot(out/'successor')
    run([old, '-test.run=^TestOriginalRecordingClaimReaderRefusesSuccessor$', '-test.v'], extra={'OMABIT_RECORDING_CLAIM_UPGRADE_ROOT': str(out)})
    if snapshot(out/'successor') != successor:
        raise RuntimeError('old reader modified successor files')
    if any(digest(source/name) != value['sha256'] for name, value in inputs.items()):
        raise RuntimeError('original runtime source changed')
    if subprocess.check_output(['git', 'rev-parse', 'HEAD'], cwd=ROOT, text=True).strip() != candidate or subprocess.check_output(['git', 'status', '--porcelain'], cwd=ROOT):
        raise RuntimeError('candidate source changed during qualification')
    report = {'version': 1, 'passed': True, 'original_source': ORIGINAL, 'candidate_source': candidate,
              'original_writer_sha256': digest(old), 'candidate_sha256': digest(new),
              'declared_helper_sha256': digest(helper), 'original_source_manifest_sha256': digest(out/'original-source.json'),
              'original_writer_files': original, 'successor_files': successor,
              'original_reads_byte_preserved': True, 'old_reader_refused_successor_unchanged': True,
              'actual_claim1_omissions_remain_unknown': True,
              'legacy_closed_claim_not_replayed': True,
              'scope': 'source-only original claim1 and candidate claim2 filesystem lookup/refusal with synthetic job/metadata; no live worker, Docker, lifecycle, power-loss or installed upgrade qualification'}
    (out/'result.json').write_text(json.dumps(report, indent=2)+'\n')
    print(json.dumps({key: report[key] for key in ('passed', 'original_source', 'candidate_source', 'original_writer_sha256', 'candidate_sha256')}))


if __name__ == '__main__':
    main()
