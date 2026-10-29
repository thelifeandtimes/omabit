"""Inert isolation fixture boundaries; Docker and HTTP are always replaced."""
import os
from pathlib import Path
import subprocess
import sys
import unittest
from unittest.mock import Mock,patch

HERE=Path(__file__).resolve().parent
sys.path.insert(0,str(HERE.parent/'fake-networks'))
import docker_isolation as case
import docker_appliance as prep
from docker_runtime import Ship,receipt_name

class IsolationTests(unittest.TestCase):
    def contexts(self):
        return [dict(engine=Mock(environment={}),specs={s:Ship(p*32,s,a*64,25432) for s in ('zod','nec')}) for p,a in [('a','b'),('c','d')]]
    def test_host_refuses_before_state_engine_or_transport(self):
        with patch.dict(os.environ,{},clear=True),patch.object(case,'save') as save,patch.object(case.docker_case,'exercise') as engine,patch.object(case.subprocess,'run') as run:
            for invoke in (lambda:case.exercise({}),lambda:case.verify({}, {},None),lambda:case.snapshot({}),lambda:case.reject_foreign_session({}, {},'zod')):
                with self.assertRaisesRegex(RuntimeError,'guest admission'):invoke()
            save.assert_not_called();engine.assert_not_called();run.assert_not_called()
    def test_receipt_labels_are_fixed_and_disjoint(self):
        names=[receipt_name(k,l) for k in ('chain','peers','log') for l in (None,'a','b')]
        self.assertEqual(len(set(names)),9);self.assertEqual(receipt_name('chain'),'private-chain.json')
        for label in ('../a','a/b','',False,1,'c'):
            with self.subTest(label=label),self.assertRaises(ValueError):receipt_name('chain',label)
    def test_isolation_selection_is_exclusive_and_explicit(self):
        files=prep.gateway_sources(False,isolation=True)
        self.assertEqual(files['guest.py'],HERE/'docker_isolation_guest.py')
        self.assertEqual(set(files)-set(prep.gateway_sources(False)),{'docker_isolation.py'})
        for args in [(True,False,False,True),(True,True,False,True),(True,True,True,True),(False,False,False,1)]:
            with self.subTest(args=args),self.assertRaises(ValueError):prep.gateway_sources(*args)
        old=(HERE/'docker_guest.py').read_text()
        new=(HERE/'docker_isolation_guest.py').read_text().replace('private-chain-isolation.json','private-chain.json').replace('import docker_isolation','import docker_case').replace('docker_isolation.exercise','docker_case.exercise')
        self.assertEqual(new,old)
    def test_foreign_cookie_is_stdin_only_and_rejected_for_both_ships(self):
        for ship in ('zod','nec'):
            source,target=self.contexts();source['engine'].execute.return_value=b'private-cookie'
            reply=subprocess.CompletedProcess([],0,stdout=b'refused\n403',stderr=b'')
            with patch.object(case,'require_guest'),patch.object(case.subprocess,'run',return_value=reply) as run:
                self.assertEqual(case.reject_foreign_session(source,target,ship),403)
            args=run.call_args.args[0];self.assertEqual(run.call_args.kwargs['input'],b'private-cookie')
            self.assertNotIn('private-cookie',' '.join(args));self.assertIn('/dev/stdin',args)
            self.assertEqual(args[-1],'http://127.0.0.1:'+str(18081+(ship=='nec'))+'/~/scry/hood/kiln/pikes.json')
            target['engine'].execute.assert_not_called()
    def test_crossed_session_success_and_unbounded_responses_refuse(self):
        replies=[subprocess.CompletedProcess([],0,stdout=b'accepted\n200',stderr=b''),subprocess.CompletedProcess([],28,stdout=b'\n403',stderr=b''),subprocess.CompletedProcess([],0,stdout=b'x'*65537+b'\n403',stderr=b''),subprocess.CompletedProcess([],0,stdout=b'403',stderr=b'')]
        for reply in replies:
            source,target=self.contexts();source['engine'].execute.return_value=b'private-cookie'
            with patch.object(case,'require_guest'),patch.object(case.subprocess,'run',return_value=reply),self.assertRaises(RuntimeError):case.reject_foreign_session(source,target,'zod')
        source,target=self.contexts()
        with patch.object(case,'require_guest'),patch.object(case.subprocess,'run') as run:
            with self.assertRaises(ValueError):case.reject_foreign_session(source,source,'zod')
            source['engine'].execute.assert_not_called();run.assert_not_called()
    def test_completed_recovery_transcript_is_not_mutated_by_extension(self):
        recovery=Mock(report={'ok':True},entries=['a','b','c','d'],channel=None)
        ctx=case.context(Mock(),{},Mock(),recovery,{},lambda:None)
        ctx['recovery'].entries.append('new')
        self.assertEqual(recovery.entries,['a','b','c','d']);self.assertEqual(recovery.report,{'ok':True})
        for update in ({'report':{'ok':False}},{'entries':['a']},{'channel':'pending'}):
            failed=Mock(report={'ok':True},entries=['a','b','c','d'],channel=None)
            for k,v in update.items():setattr(failed,k,v)
            with self.subTest(update=update),self.assertRaises(RuntimeError):case.context(None,{},None,failed,{},lambda:None)

if __name__=='__main__':unittest.main()
