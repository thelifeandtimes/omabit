#!/usr/bin/env python3
"""Private original60 -> current app-journal rehearsal and actual reader refusal.

Requires the frozen original writer recorded in the fixture manifest. Builds the
candidate from this checkout, records all Go input hashes, and runs no ship or
installed service. This is source migration evidence, not an installed upgrade.
"""
import argparse
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import subprocess

ROOT = Path(__file__).resolve().parents[1]
FIXTURE = ROOT / 'internal/managed/testdata/grubbery-app-original60'


def sha(path):
    with path.open('rb') as f:
        return hashlib.file_digest(f, 'sha256').hexdigest()


def ordinary(path):
    path = path.absolute()
    if path.is_symlink() or not path.is_file() or path.resolve() != path:
        raise ValueError('ordinary physical input required')
    return path


def source_inputs():
    return {str(p.relative_to(ROOT)): sha(p) for p in sorted(ROOT.rglob('*.go'))
            if '.git' not in p.parts and 'build' not in p.parts} | {
                name: sha(ROOT / name) for name in ('go.mod', 'go.sum')}


CAPSULE = r'''
import hashlib,json,sqlite3,subprocess
from contextlib import closing
from pathlib import Path
p=Path('/tmp/schema-state/runner.sqlite')
def run(binary,name):
 subprocess.run([binary,'-test.run','^'+name+'$','-test.count=1','-test.timeout=120s'],check=True,cwd='/source60/internal/managed')
def snapshot():
 # All Go writers have closed. Immutable read avoids creating empty WAL/SHM
 # coordination files merely to verify an otherwise closed fixture.
 assert not (p.parent/'runner.sqlite-wal').exists() or (p.parent/'runner.sqlite-wal').stat().st_size==0
 with closing(sqlite3.connect(p.as_uri()+'?mode=ro&immutable=1',uri=True)) as db:
  assert db.execute('pragma quick_check').fetchone()==('ok',)
  names=[r[0] for r in db.execute("select name from sqlite_master where type='table' and name not like 'sqlite_%' order by name")]
  return db.execute('pragma user_version').fetchone()[0],{n:sorted(json.dumps([{'hex':v.hex()} if isinstance(v,bytes) else v for v in row],sort_keys=True) for row in db.execute('select * from "'+n+'"')) for n in names}
def hashes():
 return {f.name:hashlib.sha256(f.read_bytes()).hexdigest() for f in p.parent.iterdir() if f.is_file()}
run('/writer60','TestAppOriginal60Writer')
version,before=snapshot();assert version==60
original_hashes=hashes()
run('/candidate','TestGrubberyOriginal60MigrationRehearsal')
version,after=snapshot();assert version==61
for table,rows in before.items():
 assert all(row in after[table] for row in rows),'original row changed: '+table
assert len(after['grubbery_app_operations'])==1
assert len(after['grubbery_app_events'])==2
assert len(after['grubbery_app_leases'])==1
frozen=hashes()
Path('/tmp/before-old-reader.json').write_text(json.dumps(frozen,indent=2)+'\n')
run('/writer60','TestAppOriginal60ReaderRefuses')
after_reader=hashes()
Path('/tmp/after-old-reader.json').write_text(json.dumps(after_reader,indent=2)+'\n')
assert after_reader==frozen,'old reader changed successor files'
Path('/tmp/verification.json').write_text(json.dumps({'passed':True,'original_schema':60,'target_schema':61,'original_tables_preserved':sorted(before),'populated_app_journal':True,'original_uncertainty_preserved_across_reopen':True,'actual_old_reader_refused_unchanged':True,'original_file_hashes':original_hashes,'successor_file_hashes':frozen,'scope':'synthetic source journal migration only; no native dispatch, installed service or production state'},indent=2)+'\n')
'''


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument('--old-writer', type=Path, required=True)
    p.add_argument('--output', type=Path, required=True)
    p.add_argument('--budget-gib', type=int, choices=(40, 48), default=40)
    a = p.parse_args()
    old = ordinary(a.old_writer)
    manifest = json.loads((FIXTURE / 'manifest.json').read_text())
    if sha(old) != manifest['writer_binary_sha256']:
        raise ValueError('require the exact frozen actual60 writer')
    # Its historical fixture dependencies are byte-identical, independently of
    # the candidate reader and the current test helper implementation.
    dependency = ROOT / 'internal/managed/testdata/project-stop-recovery'
    for name in ('manifest.json', 'schema59.sql.gz', 'schema59-writer-test.go.txt'):
        original = subprocess.check_output(['git', 'show', manifest['source_commit'] + ':internal/managed/testdata/project-stop-recovery/' + name], cwd=ROOT)
        if hashlib.sha256(original).hexdigest() != sha(dependency / name):
            raise ValueError('historical fixture dependency changed')
    out = a.output.absolute()
    if out.parent != Path('/tmp') or out.exists() or out.is_symlink() or not out.name.startswith('omabit-issues-grubbery-upgrade-'):
        raise ValueError('fresh task-owned /tmp/omabit-issues-grubbery-upgrade-* output required')
    spec = importlib.util.spec_from_file_location('app_upgrade_storage', ROOT / 'scripts/qualification/issues-recovery/run.py')
    storage = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(storage)
    admission = storage.charge(96 * 1024**2, a.budget_gib * 1024**3)
    out.mkdir(mode=0o700)
    scratch = out / 'scratch'
    scratch.mkdir(mode=0o700)
    before = source_inputs()
    candidate = out / 'candidate.test'
    build = ['go', 'test', '-race', '-c', '-o', str(candidate), './internal/managed']
    subprocess.run(build, cwd=ROOT, check=True)
    if source_inputs() != before:
        raise ValueError('source changed during build; preserve output and rebuild separately')
    capsule = out / 'capsule.py'
    capsule.write_text(CAPSULE)
    command = ['bwrap', '--unshare-all', '--die-with-parent', '--clearenv', '--ro-bind', '/usr', '/usr', '--symlink', 'usr/bin', '/bin', '--symlink', 'usr/lib', '/lib', '--symlink', 'usr/lib', '/lib64', '--proc', '/proc', '--dev', '/dev', '--dir', '/etc', '--tmpfs', '/run', '--bind', str(scratch), '/tmp', '--ro-bind', str(old), '/writer60', '--ro-bind', str(candidate), '/candidate', '--ro-bind', str(capsule), '/capsule.py', '--ro-bind', str(dependency), '/source60/internal/managed/testdata/project-stop-recovery', '--setenv', 'HOME', '/tmp', '--setenv', 'PATH', '/usr/bin:/bin', '--setenv', 'LANG', 'C.UTF-8', '--setenv', 'OMABIT_TEST_GRUBBERY_UPGRADE', 'networkless-original60', '--chdir', '/tmp']
    for name in ('passwd', 'group', 'nsswitch.conf', 'hosts'):
        command += ['--ro-bind', '/etc/' + name, '/etc/' + name]
    command += ['--', '/usr/bin/python3', '-B', '/capsule.py']
    report = {'original_source': manifest['source_commit'], 'candidate_base': subprocess.check_output(['git', 'rev-parse', 'HEAD'], cwd=ROOT, text=True).strip(), 'candidate_go_inputs': before, 'writer_sha256': sha(old), 'candidate_sha256': sha(candidate), 'build_command': build, 'command': command, 'capsule_sha256': sha(capsule), 'storage_admission': admission}
    (out / 'launch.json').write_text(json.dumps(report, indent=2) + '\n')
    with (out / 'test.log').open('w') as log:
        result = subprocess.run(command, stdout=log, stderr=subprocess.STDOUT, timeout=300)
    report['exit'] = result.returncode
    report['source_inputs_unchanged'] = source_inputs() == before
    report['storage_after'] = storage.charge(0, a.budget_gib * 1024**3)
    (out / 'result.json').write_text(json.dumps(report, indent=2) + '\n')
    print((out / 'test.log').read_text())
    print('Evidence:', out)
    return result.returncode if report['source_inputs_unchanged'] else 1


if __name__ == '__main__':
    raise SystemExit(main())
