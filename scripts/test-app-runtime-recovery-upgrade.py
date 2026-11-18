#!/usr/bin/env python3
"""Rehearse actual62 writer preservation and actual62 reader refusal of63.

Source/SQLite only. Requires --run, a clean candidate and fresh /tmp output.
Never opens a runner service, engine, native runtime or existing user store.
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
ORIGINAL = '3e7808791c6af9357eb0bc808620b1ec1daca349'


def digest(path):
    with path.open('rb') as stream:
        return hashlib.file_digest(stream, 'sha256').hexdigest()


def snapshot(directory):
    files = {}
    for path in sorted(directory.rglob('*')):
        if path.is_symlink():
            raise RuntimeError('ordinary physical qualification files required')
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
    if out is None or out.parent != Path('/tmp') or not re.fullmatch('omabit-app-runtime-upgrade-[0-9]{2}', out.name) or out.exists() or out.is_symlink():
        parser.error('fresh physical /tmp/omabit-app-runtime-upgrade-NN output required')
    if subprocess.check_output(['git', 'status', '--porcelain'], cwd=ROOT):
        raise RuntimeError('freeze a clean candidate before qualification')
    candidate = subprocess.check_output(['git', 'rev-parse', 'HEAD'], cwd=ROOT, text=True).strip()
    spec = importlib.util.spec_from_file_location('budget', ROOT/'scripts/qualification/issues-recovery/run.py')
    budget = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(budget)
    admission = budget.charge(256*1024**2, args.storage_budget_gib*1024**3)
    out.mkdir(mode=0o700)
    (out/'admission.json').write_text(json.dumps(admission, indent=2)+'\n')
    source = out/'original-source'
    source.mkdir(mode=0o700)
    entries = subprocess.check_output(['git', 'ls-tree', '-rl', ORIGINAL, '--', 'internal', 'go.mod', 'go.sum'], cwd=ROOT, text=True).splitlines()
    inputs = {}
    for entry in entries:
        metadata, name = entry.split('\t', 1)
        mode, kind, blob, size = metadata.split()
        if kind != 'blob' or mode not in ('100644', '100755') or Path(name).is_absolute() or '..' in Path(name).parts:
            raise RuntimeError('unexpected original source object')
        data = subprocess.check_output(['git', 'cat-file', 'blob', blob], cwd=ROOT)
        if len(data) != int(size) or hashlib.sha1(('blob '+str(len(data))+'\0').encode()+data).hexdigest() != blob:
            raise RuntimeError('original Git object differs')
        path = source/name
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_bytes(data)
        inputs[name] = {'git_blob': blob, 'sha256': digest(path), 'bytes': len(data)}
    helper = ROOT/'scripts/qualification/app-runtime-recovery/original62-helper.go.txt'
    injected = source/'internal/managed/qualification_original62_test.go'
    injected.write_bytes(helper.read_bytes())
    (out/'original-source.json').write_text(json.dumps({'source_commit': ORIGINAL, 'scope': 'unchanged original Go/embedded files, original tests and fixtures; declared metadata-only qualification helper', 'inputs': inputs, 'declared_helper_sha256': digest(helper)}, indent=2)+'\n')
    env = dict(os.environ, GOCACHE=str(Path.home()/'.cache/go-build'), GOTELEMETRY='off', GOPROXY='off', GOMAXPROCS='2')
    commands = []

    def run(command, cwd, extra=None):
        result = subprocess.run(command, cwd=cwd, env=dict(env, **(extra or {})), capture_output=True, text=True, timeout=240)
        commands.append({'command': [str(x) for x in command], 'cwd': str(cwd), 'exit': result.returncode, 'stdout': result.stdout, 'stderr': result.stderr})
        (out/'commands.json').write_text(json.dumps(commands, indent=2)+'\n')
        if result.returncode:
            raise RuntimeError('source upgrade command failed; original output retained')

    old, new = out/'original62.test', out/'candidate.test'
    run(['go', 'test', '-p=2', '-c', '-ldflags=-s -w', '-o', old, './internal/managed'], source)
    run(['go', 'test', '-p=2', '-c', '-ldflags=-s -w', '-o', new, './internal/managed'], ROOT)
    qualify = {'OMABIT_APP_RUNTIME_UPGRADE_ROOT': str(out)}
    run([old, '-test.run=^TestOriginal62AppRuntimeWriter$', '-test.v'], source/'internal/managed', qualify)
    original = snapshot(out/'original')
    if set(original) != {'runner.sqlite'}:
        raise RuntimeError('closed original database required')
    run([new, '-test.run=^TestAppRuntimeOriginal62Qualification$', '-test.v'], ROOT/'internal/managed', qualify)
    if snapshot(out/'original') != original:
        raise RuntimeError('candidate changed original62 fixture files')
    successor = snapshot(out/'successor')
    if set(successor) != {'runner.sqlite'}:
        raise RuntimeError('closed successor database required')
    run([old, '-test.run=^TestOriginal62ReaderRefusesAppRuntime63Unchanged$', '-test.v'], source/'internal/managed', qualify)
    if snapshot(out/'successor') != successor:
        raise RuntimeError('old reader changed successor files')
    if any(digest(source/name) != value['sha256'] for name, value in inputs.items()) or digest(injected) != digest(helper):
        raise RuntimeError('original source/helper changed')
    if subprocess.check_output(['git', 'rev-parse', 'HEAD'], cwd=ROOT, text=True).strip() != candidate or subprocess.check_output(['git', 'status', '--porcelain'], cwd=ROOT):
        raise RuntimeError('candidate source changed during qualification')
    final = budget.charge(0, args.storage_budget_gib*1024**3)
    (out/'final-storage.json').write_text(json.dumps(final, indent=2)+'\n')
    report = {'version': 1, 'passed': True, 'original_source': ORIGINAL, 'candidate_source': candidate,
              'original_writer_sha256': digest(old), 'candidate_sha256': digest(new),
              'declared_helper_sha256': digest(helper), 'original_source_manifest_sha256': digest(out/'original-source.json'),
              'original_writer_files': original, 'successor_files': successor,
              'original_rows_ddl_access_and_uncertainty_preserved': True,
              'repeated_migration_and_reopen': True, 'metadata_only_audit_preserved_original_rows': True,
              'actual_old_reader_refused_successor_unchanged': True,
              'scope': 'source-only actual62/candidate63 SQLite migration and reader refusal; no native process rebinding, app write, installed upgrade, DEV or stable qualification'}
    (out/'result.json').write_text(json.dumps(report, indent=2)+'\n')
    print(json.dumps({key: report[key] for key in ('passed', 'original_source', 'candidate_source', 'original_writer_sha256', 'candidate_sha256')}))


if __name__ == '__main__':
    main()
