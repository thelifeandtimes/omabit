"""Offline direct transport checks; no sockets or process dispatch."""
import http.client
import json
import unittest
from unittest.mock import patch
from published_client import PublishedClient
from state_case import parse_sse

class Response:
    status=200
    def __init__(self, body=b'{}', headers=(), chunks=()):
        self.body=body;self.headers=headers;self.chunks=iter(chunks)
    def read(self,n):return self.body[:n]
    def read1(self,n):
        value=next(self.chunks,b'')
        if isinstance(value,Exception):raise value
        return value
    def getheaders(self):return self.headers

class Connection:
    def __init__(self,response):self.response=response;self.sent=[];self.closed=False
    def request(self,*args):self.sent.append(args)
    def getresponse(self):
        if isinstance(self.response,Exception):raise self.response
        return self.response
    def close(self):self.closed=True

class PublishedTests(unittest.TestCase):
    def setUp(self):
        self.record=[];self.client=PublishedClient({8081:18281,8082:18282},self.record,lambda:None)

    def invoke(self,response,*args,**kwargs):
        connection=Connection(response)
        with patch('published_client.http.client.HTTPConnection',return_value=connection) as factory:
            result=self.client.request(*args,**kwargs)
        self.assertTrue(connection.closed)
        return result,connection,factory

    def test_exact_endpoint_memory_cookie_and_redacted_receipt(self):
        response=Response(headers=[('Set-Cookie','urbauth-~zod=private-fixture-cookie; Path=/')])
        _,connection,factory=self.invoke(response,8081,'/~/login',b'private-fixture-code',jar='/tmp/omabit-zod-cookie',form=True)
        factory.assert_called_once_with('127.0.0.1',18281,timeout=30)
        self.assertEqual(connection.sent[0][0],'POST')
        self.assertIn(b'password=private-fixture-code',connection.sent[0][2])
        _,connection,_=self.invoke(Response(),8081,'/~/scry/hood/kiln/pikes.json',cookie='/tmp/omabit-zod-cookie')
        self.assertEqual(connection.sent[0][3]['Cookie'],'urbauth-~zod=private-fixture-cookie')
        self.assertNotIn('private-fixture',json.dumps(self.record))

    def test_wrong_ship_and_ambiguous_sessions_refused(self):
        for headers in ([('Set-Cookie','urbauth-~nec=wrong')],[('Set-Cookie','urbauth-~zod=one'),('Set-Cookie','urbauth-~nec=two')],[('Set-Cookie','urbauth-~zod=')]):
            with self.subTest(headers=headers),self.assertRaisesRegex(RuntimeError,'identity'):
                self.invoke(Response(headers=headers),8081,'/~/login',b'code',jar='/tmp/omabit-zod-cookie',form=True)
        self.assertEqual(self.client.cookies,{})

    def test_unbound_requests_never_open_socket(self):
        with patch('published_client.http.client.HTTPConnection') as factory:
            for port,path,kwargs in [(12321,'/',{}),(8081,'//elsewhere',{}),(8081,'/~/login',{'jar':'/tmp/other','form':True,'data':b'x'}),(8081,'/path',{'cookie':'/tmp/omabit-zod-cookie'})]:
                with self.subTest(port=port,path=path),self.assertRaises(RuntimeError):self.client.request(port,path,**kwargs)
            factory.assert_not_called()

    def test_redirect_is_returned_without_following(self):
        response=Response(headers=[('Location','http://elsewhere/')]);response.status=302
        (status,_),connection,_=self.invoke(response,8081,'/~/host')
        self.assertEqual(status,302);self.assertEqual(len(connection.sent),1)

    def test_uncertain_write_is_not_replayed_or_exposed(self):
        connection=Connection(http.client.RemoteDisconnected('private-fixture-code'))
        with patch('published_client.http.client.HTTPConnection',return_value=connection) as factory:
            with self.assertRaisesRegex(RuntimeError,'not replayed') as error:self.client.request(8081,'/poke',b'private-fixture-code')
        factory.assert_called_once();self.assertEqual(len(connection.sent),1);self.assertTrue(connection.closed)
        self.assertNotIn('private-fixture',str(error.exception));self.assertEqual(self.record,[])

    def test_bounded_stream_preserves_complete_events_and_cursor(self):
        self.client.cookies['/tmp/omabit-nec-cookie']='urbauth-~nec=private-fixture-cookie'
        body=b'id: 5\ndata: {"id":1,"response":"subscribe","ok":"ok"}\n\n'
        response=Response(chunks=[body[:19],body[19:],TimeoutError()])
        (_,data),connection,_=self.invoke(response,8082,'/~/channel/test',cookie='/tmp/omabit-nec-cookie',stream=True,timeout=3,cursor=4)
        self.assertEqual(connection.sent[0][3]['Last-Event-ID'],'4')
        self.assertEqual(list(parse_sse(data.splitlines(keepends=True)))[0][0],5)
        self.assertNotIn('private-fixture',json.dumps(self.record))

    def test_oversized_response_refused(self):
        with self.assertRaisesRegex(RuntimeError,'bound'):self.invoke(Response(body=b'x'*65537),8081,'/~/host')

    def test_logout_requires_revocation(self):
        self.client.cookies['/tmp/omabit-zod-cookie']='urbauth-~zod=private-fixture-cookie'
        response=Response();response.status=401
        connections=[Connection(Response()),Connection(response)]
        with patch('published_client.http.client.HTTPConnection',side_effect=connections):
            self.assertEqual(self.client.logout(),[{'ship':'zod','revoked':True}])
        self.assertEqual(self.client.cookies,{})

class ForeignPublishedTests(unittest.TestCase):
    def test_foreign_cookie_is_ephemeral_and_never_replaces_target_session(self):
        source=PublishedClient({8081:18281,8082:18282},[],lambda:None)
        records=[];target=PublishedClient({8081:18291,8082:18292},records,lambda:None)
        key='/tmp/omabit-zod-cookie';source.cookies[key]='urbauth-~zod=source-private';target.cookies[key]='urbauth-~zod=target-private'
        response=Response();response.status=403;connection=Connection(response)
        with patch('published_client.http.client.HTTPConnection',return_value=connection) as factory:
            self.assertEqual(target.reject_foreign_session(source,'zod'),403)
        factory.assert_called_once_with('127.0.0.1',18291,timeout=30)
        self.assertEqual(connection.sent[0][3]['Cookie'],'urbauth-~zod=source-private')
        self.assertEqual(target.cookies,{key:'urbauth-~zod=target-private'})
        self.assertEqual(source.cookies,{key:'urbauth-~zod=source-private'})
        self.assertNotIn('private',json.dumps(records))

    def test_foreign_cookie_removed_on_acceptance_or_transport_failure(self):
        for response in [Response(),http.client.RemoteDisconnected('private')]:
            source=PublishedClient({8081:18281,8082:18282},[],lambda:None)
            target=PublishedClient({8081:18291,8082:18292},[],lambda:None)
            source.cookies['/tmp/omabit-nec-cookie']='urbauth-~nec=private'
            with patch('published_client.http.client.HTTPConnection',return_value=Connection(response)),self.assertRaises(RuntimeError):
                target.reject_foreign_session(source,'nec')
            self.assertEqual(target.cookies,{})

    def test_foreign_fixture_refuses_same_endpoint_missing_session_and_capture(self):
        a=PublishedClient({8081:18281,8082:18282},[],lambda:None)
        b=PublishedClient({8081:18291,8082:18292},[],lambda:None)
        with patch('published_client.http.client.HTTPConnection') as factory:
            for target,source,ship in [(a,a,'zod'),(b,a,'bus'),(b,a,'zod')]:
                with self.assertRaises(RuntimeError):target.reject_foreign_session(source,ship)
            with self.assertRaises(RuntimeError):a.request(8081,'/~/login',b'code',jar='/tmp/omabit-cross-zod-cookie',form=True)
            factory.assert_not_called()

if __name__=='__main__':unittest.main()
