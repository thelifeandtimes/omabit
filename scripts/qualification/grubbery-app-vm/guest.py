#!/usr/bin/python3
"""One fresh offline guest; actual installed source transport and CLI/stdio."""
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import subprocess
import sys

I = Path('/opt/omabit-release-input')
O = Path('/opt/omabit-issues-results')
HOME = Path('/home/review')
report = {'format': 1, 'passed': False,
          'scope': 'fresh offline Linux/AMD64 product transport; one disposable fake; not production/DEV/stable qualification'}


def sha(path):
    with Path(path).open('rb') as stream:
        return hashlib.file_digest(stream, 'sha256').hexdigest()


def guard():
    if (os.geteuid() != 0 or Path('/sys/class/dmi/id/sys_vendor').read_text().strip() != 'QEMU'
            or 'omabit.disposable-guest=release-upgrade-v1' not in Path('/proc/cmdline').read_text().split()
            or sys.argv[1:]):
        raise RuntimeError('refused outside fresh offline product VM')
    value = json.loads((I/'scenario.json').read_text())
    if value.get('kind') != 'issues18-product-grubbery-v1' or not value.get('development_snapshot') or value.get('target_schema') != 62:
        raise RuntimeError('wrong frozen product scenario')
    return value


def main():
    scenario = guard()
    if O.exists():
        raise RuntimeError('never resume or reset an existing guest')
    O.mkdir(mode=0o700)
    for name in ('/etc/urbit-runner', '/srv/urbit/piers', '/home/review/.local/share/urbit-runner'):
        if Path(name).exists():
            raise RuntimeError('guest is not pristine')
    for name in ('omabit', 'urbit-collector', 'urbit-runner'):
        if sha(I/'candidate'/name) != scenario['candidate'][name]['sha256']:
            raise RuntimeError('frozen candidate bytes changed')
    for name, pin in (('managed.test', scenario['candidate']['managed.test']['sha256']),
                      ('runtime.oci.tar', scenario['oci_sha256']), ('boot.pill', scenario['pill_sha256'])):
        if sha(I/name) != pin:
            raise RuntimeError('frozen helper/runtime bytes changed')
    spec = importlib.util.spec_from_file_location('common', I/'issues-common.py')
    common = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(common)
    # Reuse only its fixed owner/CLI/install helpers, never its scenario main,
    # old schema assertions, process fault injection or reset/cleanup flow.
    common.O = O
    common.report = report
    common.scenario = scenario
    report.update(source_commit=scenario['source_commit'], inputs=scenario)
    common.run(['loginctl', 'enable-linger', 'review'])
    common.run(['systemctl', 'start', 'user@1000.service'])
    common.run(['ssh-keygen', '-q', '-t', 'ed25519', '-N', '', '-f', HOME/'operator'], True)
    host = common.install('candidate')
    report['runner_id'] = host['runner_id']
    common.run([I/'candidate/urbit-collector', 'enroll', 'local', '--socket', common.SOCK], True)
    common.docker('load', '--input', I/'runtime.oci.tar')
    hello = common.hello()
    if hello['build']['sha256'] != scenario['candidate']['urbit-runner']['sha256'] or common.db_projection()['schema'] != 62:
        raise RuntimeError('installed build/schema does not match frozen source')
    # Historical "recommended" selects the explicit immutable uploaded pill;
    # this case deliberately uses the standard f8b08c pill, not the Apps pill.
    common.collector('candidate', 'create-fake', 'grubbery-zod', '--ship', '~zod',
                     '--http-port', '19380', '--request-id', 'product-grubbery-create',
                     '--bootstrap', 'recommended', '--pill', str(I/'boot.pill'))
    common.operation('candidate', 'grubbery-zod', 'start', 'product-initial-start')
    common.wait(lambda: common.instance('candidate', 'grubbery-zod', 'running'), 600)
    helper = subprocess.run(common.owner_args([
        'env', 'OMABIT_APP_PRODUCT_VM=source-state-restart-v1', I/'managed.test',
        '-test.run=^TestIssuesGrubberyProductDisposableVM$', '-test.v', '-test.timeout=35m']),
        timeout=2150)
    report['helper_exit'] = helper.returncode
    if helper.returncode:
        raise RuntimeError('product helper failed; original phase proof and serial log retained')
    proof = json.loads((HOME/'omabit-app-product/proof.json').read_text())
    if not all(proof.get('qualification', {}).get(key) is True for key in (
            'actual_product_docker_transport', 'native_cli_and_stdio', 'two_source_versions',
            'state_and_new_code', 'same_pier_container_replacement', 'state_after_replacement',
            'manager_restart_original_replay')):
        raise RuntimeError('product helper did not complete every declared gate')
    report['product'] = proof['qualification']
    report['proof_sha256'] = sha(HOME/'omabit-app-product/proof.json')
    common.operation('candidate', 'grubbery-zod', 'stop', 'product-final-clean-stop')
    common.wait(lambda: common.instance('candidate', 'grubbery-zod', 'stopped'), 180)
    report['final_clean_ship_stop'] = True
    report['passed'] = True


if __name__ == '__main__':
    try:
        main()
    except BaseException as error:
        report['error'] = type(error).__name__+': '+str(error)
    finally:
        if O.is_dir():
            (O/'result.json').write_text(json.dumps(report, indent=2)+'\n')
        print('OMABIT_GRUBBERY_VM_RESULT '+json.dumps(report, separators=(',', ':')), flush=True)
        try:
            guard()
            subprocess.run(['systemctl', 'poweroff'], check=True)
        except BaseException:
            pass
