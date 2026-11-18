#!/usr/bin/env python3
"""Launch one fresh native source capsule under its own bounded user supervisor.

The supervisor survives a short-lived CLI/MCP/tool caller. It installs no unit,
changes no existing service, and still delegates namespace/resource/storage
admission to probe.py. Preserve interrupted receipts; never reuse an output.
"""
import argparse
import hashlib
import json
import os
from pathlib import Path
import re
import subprocess
import uuid

ROOT = Path(__file__).resolve().parents[3]
CASES = ('grubbery-bootstrap-v1', 'grubbery-app-contract-v1', 'grubbery-app-contract-v2', 'grubbery-managed-coordinator-v1', 'grubbery-source-layout-v1', 'grubbery-public-workflow-v1')


def ordinary(path, directory=False):
    path = path.absolute()
    if path.is_symlink() or path.resolve() != path or not (path.is_dir() if directory else path.is_file()):
        raise ValueError('ordinary physical input required')
    return path


def sha(path):
    with path.open('rb') as stream:
        return hashlib.file_digest(stream, 'sha256').hexdigest()


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument('--output', type=Path, required=True)
    for name in ('requests', 'workflow-binary', 'framework-source', 'vere', 'pill'):
        p.add_argument('--' + name, type=Path, required=True)
    p.add_argument('--workflow-case', choices=CASES, required=True)
    p.add_argument('--artifact-source', required=True)
    p.add_argument('--artifact-sha256', required=True)
    p.add_argument('--storage-budget-gib', type=int, choices=(40, 48), default=40)
    a = p.parse_args()
    if os.geteuid() == 0:
        p.error('ordinary owner required')
    out = a.output.absolute()
    if out.parent != Path('/tmp') or not re.fullmatch(r'omabit-desk-probe-[a-z0-9-]+', out.name) or out.exists() or out.is_symlink():
        p.error('fresh task-owned /tmp/omabit-desk-probe-* output required')
    inputs = {name: ordinary(getattr(a, name), name == 'framework_source')
              for name in ('requests', 'workflow_binary', 'framework_source', 'vere', 'pill')}
    if not re.fullmatch('[0-9a-f]{40}', a.artifact_source) or not re.fullmatch('[0-9a-f]{64}', a.artifact_sha256) or sha(inputs['workflow_binary']) != a.artifact_sha256:
        p.error('exact frozen artifact source/hash required')
    if subprocess.check_output(['git', 'status', '--porcelain'], cwd=ROOT):
        p.error('freeze clean harness source before launch')
    subprocess.run(['git', 'cat-file', '-e', a.artifact_source + '^{commit}'], cwd=ROOT, check=True)
    if subprocess.check_output(['git', 'diff', a.artifact_source, '--', '*.go', 'go.mod', 'go.sum', 'internal/appdev'], cwd=ROOT):
        p.error('Go/embedded inputs changed after this artifact; build a matching helper')
    supervisor_deadline = 1750 if a.workflow_case == 'grubbery-public-workflow-v1' else 1250
    token = uuid.uuid4().hex
    unit = 'omabit-issues-capsule-supervisor-' + token
    log = Path('/tmp') / (unit + '.log')
    fd = os.open(log, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
    os.close(fd)
    command = ['systemd-run', '--user', '--collect', '--unit=' + unit,
               '--property=Type=exec', '--property=RuntimeMaxSec=' + str(supervisor_deadline),
               '--property=MemoryMax=512M', '--property=MemorySwapMax=0',
               '--property=CPUQuota=25%', '--property=UMask=0077',
               '--property=WorkingDirectory=' + str(ROOT),
               '--property=StandardOutput=append:' + str(log),
               '--property=StandardError=append:' + str(log),
               '--', '/usr/bin/python3', '-B', str(Path(__file__).with_name('probe.py')),
               '--output', str(out), '--workflow-case', a.workflow_case,
               '--storage-budget-gib', str(a.storage_budget_gib)]
    for name, path in inputs.items():
        command += ['--' + name.replace('_', '-'), str(path)]
    report = {'unit': unit + '.service', 'output': str(out), 'log': str(log),
              'case': a.workflow_case, 'artifact_source': a.artifact_source,
              'artifact_sha256': a.artifact_sha256,
              'harness_source': subprocess.check_output(['git', 'rev-parse', 'HEAD'], cwd=ROOT, text=True).strip(),
              'harness_sha256': sha(Path(__file__).with_name('probe.py')),
              'command': command, 'limits': {'supervisor_memory_bytes': 512 * 1024**2,
              'supervisor_no_swap': True, 'supervisor_deadline_seconds': supervisor_deadline,
              'capsule': 'separate guarded4GiB/no-swap/one-core scope; fresh private namespaces; declared storage admission'}}
    receipt = Path('/tmp') / (unit + '.json')
    with receipt.open('x') as stream:
        json.dump(report, stream, indent=2)
        stream.write('\n')
    receipt.chmod(0o600)
    result = subprocess.run(command, capture_output=True, text=True, timeout=20)
    report['submit_exit'] = result.returncode
    report['submit_stdout'] = result.stdout
    report['submit_stderr'] = result.stderr
    if result.returncode == 0:
        observed = subprocess.run(['systemctl', '--user', 'show', report['unit'],
                                   '--property=ActiveState,SubState,ControlGroup,InvocationID,MemoryMax,MemorySwapMax,CPUQuotaPerSecUSec,RuntimeMaxUSec'],
                                  capture_output=True, text=True, timeout=10)
        report['supervisor_observation'] = {'exit': observed.returncode,
                                            'properties': observed.stdout,
                                            'error': observed.stderr,
                                            'live_observation': observed.returncode == 0 and 'ActiveState=active\n' in observed.stdout}
    receipt.write_text(json.dumps(report, indent=2) + '\n')
    if result.returncode:
        raise RuntimeError('owned supervisor submission failed; preserve its receipt')
    print(json.dumps({'unit': report['unit'], 'output': str(out), 'log': str(log), 'receipt': str(receipt)}))


if __name__ == '__main__':
    main()
