"""Offline binding, redaction and logout tests; all HTTP/process IO is inert."""
import contextlib
import io
import json
import subprocess
import unittest
from unittest.mock import patch
import login_case

SECRET='lidlut-tabwed-pillex-ridrup'
INSTANCE={'id':'zod','config_revision':2,'endpoint':'http://127.0.0.1:18080','ship':'~zod','resources':{'container_id':'a'*64}}

class LoginCaseTests(unittest.TestCase):
    def exercise(self,variant='ok'):
        reply=dict(runner_id='runner',instance_id='zod',config_revision=2,container_id='a'*64,code=SECRET)
        if variant=='wrong-binding':reply['container_id']='b'*64
        if variant=='bad-code':reply['code']='PRIVATE-INVALID'
        raw=json.dumps(reply).encode() if variant!='malformed' else b'PRIVATE-INVALID'
        process=subprocess.CompletedProcess([],1 if variant=='process-error' else 0,raw,b'PRIVATE-ERROR' if variant=='process-error' else b'')
        requests=[]
        statuses=[200,500 if variant=='bad-scry' else 200,303,403]
        class HTTP:
            def __init__(self,host,port,timeout):
                assert (host,port,timeout)==('127.0.0.1',18080,5)
            def request(self,method,path,body,headers):requests.append((method,path,body,headers))
            def getresponse(self):
                status=statuses.pop(0)
                class Reply:
                    def read(self,n):return b'{}'
                    def getheaders(self):return [('Set-Cookie','urbauth-~zod=PRIVATE-COOKIE; Path=/')]
                reply=Reply();reply.status=status;return reply
            def close(self):pass
        captured=io.StringIO()
        with patch.object(login_case,'require_guest'),patch.object(login_case.subprocess,'run',return_value=process) as run,patch.object(login_case.http.client,'HTTPConnection',HTTP),contextlib.redirect_stdout(captured),contextlib.redirect_stderr(captured):
            if variant=='ok':
                result=login_case.verify(['private-collector'],{},INSTANCE,'runner','a'*64)
                self.assertTrue(result['authenticated_scry'] and result['session_revoked'])
                self.assertNotIn(SECRET,json.dumps(result));self.assertNotIn('PRIVATE',json.dumps(result))
            else:
                with self.assertRaises(Exception) as error:login_case.verify(['private-collector'],{},INSTANCE,'runner','a'*64)
                self.assertNotIn('PRIVATE',str(error.exception));self.assertNotIn(SECRET,str(error.exception))
            self.assertTrue(run.call_args.kwargs['capture_output'])
            self.assertNotIn(SECRET,str(run.call_args))
        self.assertEqual(captured.getvalue(),'')
        if variant in ('wrong-binding','bad-code','malformed','process-error'):self.assertEqual(requests,[])
        elif variant=='bad-scry':self.assertEqual([r[1] for r in requests],['/~/login','/~/scry/hood/kiln/pikes.json','/~/logout'])
        else:self.assertEqual([r[1] for r in requests],['/~/login','/~/scry/hood/kiln/pikes.json','/~/logout','/~/scry/hood/kiln/pikes.json'])

    def test_exact_selection_authenticates_and_revokes_without_recording_secret(self):self.exercise()
    def test_wrong_binding_never_authenticates(self):self.exercise('wrong-binding')
    def test_bad_code_never_authenticates(self):self.exercise('bad-code')
    def test_malformed_response_is_redacted(self):self.exercise('malformed')
    def test_failed_process_is_redacted(self):self.exercise('process-error')
    def test_session_revoked_after_failed_scry(self):self.exercise('bad-scry')

class SelectionTests(unittest.TestCase):
    def sample(self,container=None):
        from datetime import datetime,timezone
        value=dict(INSTANCE,observed_state='running',observed_at=datetime.now(timezone.utc).isoformat(),resources={'status':'fresh' if container else 'unavailable'})
        if container:value['resources']['container_id']=container
        return {'runner_id':'runner','instances':[value]}
    def run_selection(self,values,deadline=45):
        responses=[subprocess.CompletedProcess([],0,json.dumps(v).encode(),b'') for v in values];observations=[]
        with patch.object(login_case,'require_guest'),patch.object(login_case.subprocess,'run',side_effect=responses) as run,patch.object(login_case.time,'sleep'):
            result=login_case.select(['collector'],{},INSTANCE,'runner','a'*64,observations.append,deadline_seconds=deadline)
        self.assertTrue(all(v.args[0]==['collector','snapshot'] for v in run.call_args_list))
        return result,observations
    def test_waits_for_current_container_observation_before_code_selection(self):
        result,rows=self.run_selection([self.sample(),self.sample('a'*64)])
        self.assertEqual(result['resources']['container_id'],'a'*64)
        self.assertEqual([v['ready'] for v in rows[-1]],[False,True])
    def test_replaced_container_is_not_a_wait_or_reveal(self):
        with self.assertRaisesRegex(RuntimeError,'container changed'):self.run_selection([self.sample('b'*64)])
    def test_changed_binding_is_refused(self):
        for key,value in [('config_revision',3),('ship','~nec'),('endpoint','http://127.0.0.1:18081')]:
            sample=self.sample('a'*64);sample['instances'][0][key]=value
            with self.subTest(key=key),self.assertRaisesRegex(RuntimeError,'binding changed'):self.run_selection([sample])
    def test_expired_deadline_never_dispatches(self):
        with patch.object(login_case,'require_guest'),patch.object(login_case.subprocess,'run') as run:
            with self.assertRaisesRegex(RuntimeError,'deadline'):login_case.select(['collector'],{},INSTANCE,'runner','a'*64,lambda rows:None,deadline_seconds=0)
            run.assert_not_called()

if __name__=='__main__':unittest.main()
