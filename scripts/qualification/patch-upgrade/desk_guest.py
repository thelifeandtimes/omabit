"""Exercise published agent commands against only the release guest's fresh fake."""
import json
import os
from pathlib import Path
import time


def exercise(run, tool, version, runner_id):
    assert os.geteuid() == 0
    assert 'omabit.disposable-guest=release-upgrade-v1' in Path('/proc/cmdline').read_text().split()
    root = Path('/home/review/desk-qualification')
    assert not root.exists()
    root.mkdir(mode=0o700)
    source = root/'source'
    source.mkdir(mode=0o700)
    (source/'hello.txt').write_text('Omabit release qualification\n')
    run(['chown', '-R', 'review:review', root])
    report = {'passed': False, 'steps': {}}
    target = {'instance_id': 'upgrade-zod', 'desk': 'omabit-qualification'}
    selection = ['--instance', target['instance_id'], '--desk', target['desk']]

    def save(name, value):
        report['steps'][name] = value
        (Path('/opt/omabit-release-results')/'desk.json').write_text(json.dumps(report, indent=2)+'\n')
        return value

    def command(name, *args, setup=False):
        pin = [] if setup else ['--runner', 'local', '--expected-runner-id', runner_id]
        return json.loads(run([tool(version, 'omabit'), name, *pin, *args], True, timeout=180).stdout)

    def receipt(plan, name, want):
        intent = plan['intent']
        operation_id = intent.get('intent', intent)['id']
        end = time.monotonic()+90
        while time.monotonic() < end:
            value = command('desk-operation', '--id', operation_id)
            steps = value['receipt']['steps']
            state = steps[-1]['state'] if steps else 'pending'
            if state in ('observed', 'blocked', 'uncertain', 'failed'):
                save(name, value)
                assert state == want, value
                return value
            time.sleep(.25)
        raise RuntimeError('desk receipt deadline: '+name)

    def policy(scope, gates):
        current = command('app-development-status', setup=True)
        row = next(r for r in current['runners'] if r['runner_id'] == runner_id)
        request = {'name': 'local', 'runner_id': runner_id, 'update': {
            'scope': scope, 'target': target if scope == 'desk' else {'instance_id': target['instance_id']},
            'expected_revision': row['status']['policy']['revision'], 'gates': gates}}
        # This is the test operator's explicit Setup choice for a fresh fake,
        # separate from the agent commands whose default refusal is checked.
        return command('app-development-save', '--request-json', json.dumps(request), setup=True)

    def create(name, want):
        plan = save(name+'-plan', command('desk-plan-create', *selection))
        command('desk-create', '--plan-json', json.dumps(plan))
        receipt(plan, name, want)

    try:
        inventory = command('desk-list', '--instance', target['instance_id'])
        assert target['desk'] not in inventory['desks']
        create('default-create', 'blocked')
        policy('ship', {'create_desk': 'allow', 'commit': 'inherit', 'publish': 'inherit'})
        create('allowed-create', 'observed')
        assert target['desk'] in command('desk-list', '--instance', target['instance_id'])['desks']
        stage = save('stage-plan', command('desk-plan-stage', *selection, '--source-dir', source))
        save('stage', command('desk-stage', '--plan-json', json.dumps(stage), '--source-dir', source))
        stage_id = stage['intent']['id']
        plan = save('default-commit-plan', command('desk-plan-commit', '--stage-id', stage_id))
        command('desk-commit', '--plan-json', json.dumps(plan))
        receipt(plan, 'default-commit', 'blocked')
        policy('desk', {'create_desk': 'inherit', 'commit': 'allow', 'publish': 'block'})
        plan = save('allowed-commit-plan', command('desk-plan-commit', '--stage-id', stage_id))
        assert plan['review']['change_count'] == 1
        command('desk-commit', '--plan-json', json.dumps(plan))
        receipt(plan, 'allowed-commit', 'observed')
        setting = command('desk-settings', *selection)['setting']
        desk = dict(setting['desk'], publication='source')
        request = {'name': 'local', 'runner_id': runner_id, 'target': target, 'update': {
            'target': target, 'expected_revision': setting['revision'], 'desk': desk}}
        command('app-development-configure', '--request-json', json.dumps(request), setup=True)
        for name, want in [('default-publish', 'blocked'), ('allowed-publish', 'observed')]:
            if want == 'observed':
                policy('desk', {'create_desk': 'inherit', 'commit': 'allow', 'publish': 'allow'})
            plan = save(name+'-plan', command('desk-plan-release', *selection, '--action', 'publish-source', '--checks-json', '[]'))
            command('desk-release', '--plan-json', json.dumps(plan))
            receipt(plan, name, want)
        inspected = command('desk-inspect', *selection)
        assert inspected['observation']['observation']['publication'] == 'published'
        save('published', inspected)
        policy('desk', {'create_desk': 'inherit', 'commit': 'block', 'publish': 'block'})
        policy('ship', {'create_desk': 'block', 'commit': 'block', 'publish': 'block'})
        report.update(passed=True, boundary='actual frozen CLI, restricted SSH, installed runner, Docker exec and native Clay; test operator explicitly opens only this fresh fake/desk gates; no Treaty/frontend claim from this scenario')
    except BaseException as error:
        report['error'] = type(error).__name__+': '+str(error)
        raise
    finally:
        save('completed', report['passed'])
        print('UPGRADE_DESK_RESULT '+json.dumps(report), flush=True)
    return report
