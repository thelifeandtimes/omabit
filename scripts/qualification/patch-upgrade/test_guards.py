import ast
import contextlib
import hashlib
import importlib.util
import io
import json
from pathlib import Path
import sqlite3
import sys
import tempfile
import unittest
from unittest.mock import patch


class ReleaseUpgradeGuards(unittest.TestCase):
    def host(self):
        spec=importlib.util.spec_from_file_location('upgrade_guard_host',Path(__file__).with_name('run.py'))
        host=importlib.util.module_from_spec(spec);spec.loader.exec_module(host);return host

    def test_no_run_performs_no_admission_or_process(self):
        host=self.host()
        with patch.object(sys,'argv',['run.py']),patch.object(host,'full_charge',side_effect=AssertionError('admission')),patch.object(host.subprocess,'check_output',side_effect=AssertionError('process')),contextlib.redirect_stdout(io.StringIO()):host.main()

    def test_wrong_boot_image_refuses_before_execution(self):
        host=self.host()
        with tempfile.TemporaryDirectory(prefix='omabit-upgrade-guard-') as directory:
            p=Path(directory)/'init';p.write_bytes(b'unrelated executable')
            with patch.object(host.subprocess,'check_output',side_effect=AssertionError('must not execute wrong image')):
                with self.assertRaisesRegex(RuntimeError,'exact separately frozen'):host.verify_boot(p)

    def projections(self):
        # Load only inert read functions. Importing the full guest must keep its
        # early QEMU/root refusal and must never select a live host database.
        tree=ast.parse(Path(__file__).with_name('guest.py').read_text())
        nodes=[x for x in tree.body if isinstance(x,ast.FunctionDef) and x.name in ('db_projection','app_projection')]
        self.assertEqual(len(nodes),2)
        ns=dict(contextlib=contextlib,sqlite3=sqlite3,hashlib=hashlib,json=json)
        exec(compile(ast.Module(body=nodes,type_ignores=[]),'read-only-projections','exec'),ns)
        return ns

    def test_projection_closes_readers_and_preserves_original_table_selection(self):
        functions=self.projections();connect=sqlite3.connect
        with tempfile.TemporaryDirectory(prefix='omabit-upgrade-projection-') as directory:
            path=Path(directory)/'original.sqlite';writer=connect(path)
            self.addCleanup(writer.close)
            writer.executescript('pragma journal_mode=wal; pragma user_version=59; create table desk_operations(id text primary key,body blob); create table unrelated(value text);')
            writer.execute('insert into desk_operations values (?,?)',('original',b'original\0\xff'));writer.commit()
            readers=[]
            def local(filename,**kwargs):
                self.assertEqual(filename,'file:/home/review/.local/share/urbit-runner/managed/runner.sqlite?mode=ro');self.assertEqual(kwargs,{'uri':True})
                db=connect('file:'+str(path)+'?mode=ro',uri=True);readers.append(db);return db
            with patch.object(sqlite3,'connect',side_effect=local):
                before=functions['app_projection']();self.assertEqual(before['desk_operations']['rows'],1)
                self.assertEqual(functions['db_projection']()['schema'],59)
                writer.execute('create table grubbery_new (body text)');writer.commit()
                self.assertEqual(functions['app_projection'](before),before)
                writer.execute('update desk_operations set body=?',(b'changed',));writer.commit()
                self.assertNotEqual(functions['app_projection'](before),before)
                with self.assertRaisesRegex(RuntimeError,'missing'):functions['app_projection'](['desk_missing'])
            for db in readers:
                with self.assertRaises(sqlite3.ProgrammingError):db.execute('select 1')
            writer.close();self.assertFalse(Path(str(path)+'-wal').exists())


if __name__=='__main__':unittest.main()
