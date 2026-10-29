#!/usr/bin/env python3
"""Coordinator admission/receipt tests using fresh journal fixtures only."""
import importlib.util
from pathlib import Path
import sys
import unittest

ROOT=Path(__file__).resolve().parents[1]
sys.path.insert(0,str(ROOT/'native/collections'))
import journal as j
import coordinator as c
spec=importlib.util.spec_from_file_location('journal_tests',ROOT/'scripts/test-collection-journal.py')
m=importlib.util.module_from_spec(spec);spec.loader.exec_module(m)


class Tests(unittest.TestCase):
    def setUp(self):
        self.case=m.JournalTests();self.case.setUp();self.addCleanup(self.case.doCleanups)
        self.store=self.case.store
    def run_step(self,fn):
        class Adapter:
            def perform(inner,*args):return fn(*args)
        with self.store.lease() as lease:return c.step(self.store,lease,Adapter())
    def begin(self):self.case.write(self.case.begin)
    def forbid(self,*args):self.fail('adapter called without admission')
    def test_no_intent_means_no_runtime_effects(self):self.run_step(self.forbid)
    def test_closed_lease_refuses_before_effect(self):
        self.begin()
        class Adapter:
            perform=self.forbid
        with self.store.lease() as lease:pass
        with self.assertRaises(j.Conflict):c.step(self.store,lease,Adapter())
    def test_busy_and_unexpected_error_preserve_exact_journal(self):
        self.begin();before=(self.case.root/'journal.json').read_bytes()
        def busy(*_):raise c.Busy()
        self.run_step(busy);self.assertEqual((self.case.root/'journal.json').read_bytes(),before)
        def failed(*_):raise RuntimeError('connection lost after effect')
        with self.assertRaises(RuntimeError):self.run_step(failed)
        self.assertEqual((self.case.root/'journal.json').read_bytes(),before)
    def test_lineage_problem_is_durable_and_prevents_future_adapter_calls(self):
        self.begin()
        def failed(*_):raise c.Problem('helper-descendants-unconfirmed')
        self.run_step(failed);self.case.reopen();self.store=self.case.store
        with self.assertRaises(j.Conflict):self.run_step(self.forbid)
        self.assertEqual(self.case.pending()['uncertain'][0]['reason'],'helper-descendants-unconfirmed')
    def test_one_receipt_per_step_and_adapter_cannot_rewrite_store_context(self):
        self.begin()
        def drain(phase,request,specification,role):
            self.assertEqual(phase,'drain-previous');self.assertEqual(role,'omabit')
            observation=self.case.next_events()[0]['observation']
            request['previous']['epoch']='f'*32;specification['name']='changed'
            return observation
        self.run_step(drain)
        p=self.case.pending();self.assertEqual(p['previous'],self.case.old);self.assertEqual(p['phase'],'drain-previous')
        self.assertEqual(len(p['receipts']),1)
        def next_role(phase,request,specification,role):
            self.assertEqual(role,'theme');return self.case.next_events()[1]['observation']
        self.run_step(next_role);self.assertEqual(len(self.case.pending()['receipts']),2)
    def test_invalid_observation_never_advances(self):
        self.begin();before=(self.case.root/'journal.json').read_bytes()
        with self.assertRaises(j.Conflict):self.run_step(lambda *_:{'closed':True})
        self.assertEqual((self.case.root/'journal.json').read_bytes(),before)

if __name__=='__main__':unittest.main(verbosity=2)
