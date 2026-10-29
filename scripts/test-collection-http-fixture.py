#!/usr/bin/env python3
"""Private synthetic server distinguishes idle resets from request failures."""
from pathlib import Path
import http.client,os,socket,struct,sys,threading,time,types,unittest
from unittest.mock import patch
ROOT=Path(__file__).resolve().parents[1]
sys.path.insert(0,str(ROOT/'scripts/qualification/collections'))
from appearance_fixture import AppearanceHub,AppearanceHandler
from connected_fixture import Handler

class IdleResetTests(unittest.TestCase):
 def setUp(self):
  self.assertEqual(Path.home(),Path('/lab/home'));self.assertFalse(Path('/home').exists())
  self.assertEqual([n for _,n in socket.if_nameindex()],['lo'])
 def stub(self):
  handler=AppearanceHandler.__new__(AppearanceHandler);handler.server=types.SimpleNamespace(cond=threading.Condition(),idle_disconnects=[]);return handler
 def test_real_keepalive_reset_does_not_poison_next_request(self):
  hub=AppearanceHub()
  try:
   c=http.client.HTTPConnection('127.0.0.1',hub.server_port,timeout=3)
   c.request('GET','/~/scry/settings/desk/talon.json',headers={'Cookie':'urbauth-~zod=stable-theme'})
   response=c.getresponse();self.assertEqual(response.status,200);response.read()
   self.assertIsNotNone(c.sock);c.sock.setsockopt(socket.SOL_SOCKET,socket.SO_LINGER,struct.pack('ii',1,0));c.close()
   deadline=time.monotonic()+3
   while not hub.idle_disconnects and time.monotonic()<deadline:time.sleep(.02)
   self.assertEqual(hub.idle_disconnects,[{'stage':'before-request-parse','errno':104}]);self.assertEqual(hub.errors,[])
   c=http.client.HTTPConnection('127.0.0.1',hub.server_port,timeout=3)
   try:
    c.request('GET','/~/scry/settings/desk/talon.json',headers={'Cookie':'urbauth-~zod=stable-theme','Connection':'close'})
    response=c.getresponse();self.assertEqual(response.status,200);response.read()
   finally:c.close()
   self.assertEqual(hub.errors,[]);self.assertEqual(len(hub.requests),2)
  finally:hub.close()
 def test_reset_after_parse_begins_still_propagates(self):
  handler=self.stub()
  def dispatched(this):
   self.assertTrue(this.parse_request());raise ConnectionResetError(104,'after request parse')
  with patch.object(Handler,'parse_request',return_value=True),patch.object(Handler,'handle_one_request',dispatched),self.assertRaises(ConnectionResetError):handler.handle_one_request()
  self.assertEqual(handler.server.idle_disconnects,[])
 def test_non_reset_errors_before_parse_still_propagate(self):
  handler=self.stub()
  with patch.object(Handler,'handle_one_request',side_effect=ValueError('fixture bug')),self.assertRaises(ValueError):handler.handle_one_request()
  self.assertEqual(handler.server.idle_disconnects,[])
 def test_reset_before_parse_is_recorded_and_connection_closed(self):
  handler=self.stub()
  with patch.object(Handler,'handle_one_request',side_effect=ConnectionResetError(104,'idle reset')):handler.handle_one_request()
  self.assertTrue(handler.close_connection);self.assertEqual(handler.server.idle_disconnects,[{'stage':'before-request-parse','errno':104}])

if __name__=='__main__':unittest.main()
