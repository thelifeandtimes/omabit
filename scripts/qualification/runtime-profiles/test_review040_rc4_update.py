"""Failure tests for the owner adapter; all state and command effects are private."""
import hashlib
import importlib.util
import json
from pathlib import Path
import sqlite3
import subprocess
import tempfile
import tarfile
import unittest
from unittest.mock import patch

spec = importlib.util.spec_from_file_location('update_review040_rc4', Path(__file__).with_name('update_review040_rc4.py'))
m = importlib.util.module_from_spec(spec)
spec.loader.exec_module(m)


class UpdateGuards(unittest.TestCase):
    def test_repeated_attempt_refuses_before_touching_environment(self):
        with tempfile.TemporaryDirectory() as tmp:
            work = Path(tmp)
            (work / 'plan.json').write_text(json.dumps({'phase': 'prepared-only', 'script_sha256': m.sha(m.__file__)}))
            (work / 'apply-started.json').write_text('{}')
            with patch.object(m, 'preflight', side_effect=AssertionError('must not inspect or mutate host')):
                for operation in (m.apply, m.refresh_plan):
                    with self.assertRaisesRegex(RuntimeError, 'already started'):
                        operation(work)

    def test_changed_candidate_refuses_before_service_commands(self):
        with tempfile.TemporaryDirectory() as tmp:
            work = Path(tmp);old = work / 'old';old.mkdir();candidate = work / 'plugin';candidate.mkdir()
            (candidate / 'file').write_text('original')
            plan = {'phase': 'prepared-only', 'script_sha256': m.sha(m.__file__), 'plugin_before': m.tree(old), 'plugin_candidate': m.tree(candidate)}
            (work / 'plan.json').write_text(json.dumps(plan));(candidate / 'file').write_text('changed')
            with patch.object(m, 'OLD', old), patch.object(m, 'preflight', return_value={}), patch.object(m, 'run', side_effect=AssertionError('must not run a command')):
                with self.assertRaisesRegex(RuntimeError, 'inputs changed'):
                    m.apply(work)

    def test_large_boot_pills_are_counted_without_a_fixed_directory_ceiling(self):
        with tempfile.TemporaryDirectory() as tmp:
            root=Path(tmp);dev=root/'dev';work=root/'work';managed=dev/'managed'
            for p in (managed/'boot-pills', managed/'instance', dev/'bin', work/'plugin/bin'):
                p.mkdir(parents=True)
            for name in ('config','data','state'):(dev/'desktop'/name).mkdir(parents=True)
            for name in m.TOOLS:
                (dev/'bin'/name).write_bytes(b'old');(work/'plugin/bin'/name).write_bytes(b'new')
            (managed/'runner.sqlite').write_bytes(b'state')
            (managed.parent/'runtime-profiles').mkdir()
            (managed.parent/'runtime-profiles/archive').write_bytes(b'preserved runtime')
            for path in (managed/'boot-pills/retained.pill', managed/'instance/retained.pill'):
                with path.open('wb') as f:f.truncate(217421172)
            with tarfile.open(work/m.ARCHIVE,'w:gz') as archive:
                archive.add(work/'plugin/bin/urbit-runner',arcname='layer.tar')
            with patch.multiple(m,DEV=dev,MANAGED=managed):
                result=m.storage_plan(work)
                self.assertGreater(result['breakdown']['management_backup'],2*217421172)
                self.assertLess(result['breakdown']['sqlite_analysis_copy'],1024**2)
                self.assertGreaterEqual(result['required_scratch_bytes'],sum(result['breakdown'].values()))
                self.assertEqual(result['aggregate_cap_bytes'],48*1024**3)
                (managed/'unsafe-link').symlink_to(root/'outside')
                with self.assertRaisesRegex(RuntimeError,'unexpected backup input'):
                    m.storage_plan(work)

    def test_late_ui_failure_retains_migrated_manager_and_checkpoint(self):
        with tempfile.TemporaryDirectory() as tmp:
            root=Path(tmp);dev=root/'dev';work=root/'work';old=root/'plugins/old';new=root/'plugins/new';config=root/'shell.json'
            managed=dev/'state/.local/share/urbit-runner/managed'
            for p in (managed,dev/'bin',work/'plugin/bin',old):p.mkdir(parents=True)
            for name in ('config','data','state'):(dev/'desktop'/name).mkdir(parents=True)
            (managed.parent/'runtime-profiles').mkdir()
            (managed.parent/'runtime-profiles/archive').write_bytes(b'preserved runtime')
            config.write_text('{"preserve":"original"}')
            with sqlite3.connect(managed/'runner.sqlite') as db:db.execute('pragma journal_mode=WAL');db.execute('pragma user_version=66');db.execute('create table history(value text)');db.execute("insert into history values('retain')")
            before_pins={};after_pins={}
            for name in m.TOOLS:
                (dev/'bin'/name).write_text('old '+name);before_pins[name]=m.sha(dev/'bin'/name)
                (work/'plugin/bin'/name).write_text('new '+name);after_pins[name]=m.sha(work/'plugin/bin'/name)
            with tarfile.open(work/m.ARCHIVE, 'w:gz') as archive:
                archive.add(work/'plugin/bin/urbit-runner', arcname='layer.tar')
            plan={'phase':'prepared-only','script_sha256':m.sha(m.__file__),'plugin_before':m.tree(old),'plugin_candidate':m.tree(work/'plugin'),
                  'work':str(work),'version':m.VERSION,'source':m.SOURCE,'backup':str(dev/'state/upgrade-060rc4-owner')}
            (work/'plan.json').write_text(json.dumps(plan))
            calls=[]
            def command(args,timeout=30):
                args=list(map(str,args));calls.append(args)
                if args[0]=='omarchy' and args[-1]=='false':config.write_text('{"disabled":true}');return 'ok'
                if args[:3]==['systemctl','--user','start']:
                    with sqlite3.connect(managed/'runner.sqlite') as db:db.execute('pragma user_version=67')
                    return ''
                if 'runtime-profile-install' in args:
                    self.assertEqual(args[1:3],['--development-root',str(dev)])
                    return '{}'
                if args[0]=='omarchy' and args[-1]=='status':raise RuntimeError('simulated late UI activation failure')
                return ''
            with patch.multiple(m,DEV=dev,OLD=old,NEW=new,CONFIG=config,MANAGED=managed,TOOLS=after_pins,OLD_TOOLS=before_pins,PINS={m.ARCHIVE:m.sha(work/m.ARCHIVE)}), \
                 patch.object(m,'preflight',return_value={}),patch.object(m,'budget',return_value=0),patch.object(m,'snapshot',return_value={'preserved':'state'}), \
                 patch.object(m,'engine_state',return_value={'untouched':'engine'}),patch.object(m,'properties',return_value={'ActiveState':'inactive'}), \
                 patch.object(m,'hello',return_value={}),patch.object(m,'wait_for',side_effect=lambda fn,*args:fn()),patch.object(m,'run',side_effect=command), \
                 patch.object(m.subprocess,'run',return_value=subprocess.CompletedProcess([],1)):
                with patch.object(m,'budget',side_effect=RuntimeError('aggregate storage budget exceeded')):
                    with self.assertRaisesRegex(RuntimeError,'aggregate storage'):
                        m.apply(work)
                    self.assertFalse(calls)
                    self.assertFalse((work/'apply-started.json').exists())
                    self.assertFalse((dev/'state/upgrade-060rc4-owner').exists())
                with self.assertRaisesRegex(RuntimeError,'late UI'):
                    m.apply(work)
                self.assertTrue((work/'apply-started.json').exists())
                self.assertEqual(json.loads((work/'failed.json').read_text())['phase'],'activate-versioned-dev-widget')
                with sqlite3.connect(managed/'runner.sqlite') as db:
                    self.assertEqual(db.execute('pragma user_version').fetchone()[0],67)
                    self.assertEqual(db.execute('select value from history').fetchall(),[('retain',)])
                for name,digest in after_pins.items():self.assertEqual(m.sha(dev/'bin'/name),digest)
                self.assertEqual(config.read_text(),'{"preserve":"original"}')
                self.assertTrue((work/'plugin-before').is_dir())
                backup=dev/'state/upgrade-060rc4-owner'
                self.assertEqual(m.tree(backup/'managed-before'),json.loads((backup/'stopped.json').read_text())['management_backup'])
                self.assertEqual(m.tree(backup/'runtime-profiles-before'), m.tree(managed.parent/'runtime-profiles'))
                self.assertFalse(any(c[0]=='systemctl' and m.ENGINE in c for c in calls))
                with self.assertRaisesRegex(RuntimeError,'already started'):m.apply(work)


if __name__=='__main__':unittest.main()
