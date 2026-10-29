"""Reject unrelated, duplicated and truncated plugin evidence without a runtime."""
import io
import json
import unittest
from plugin_case import StreamTap, assert_history, matched_message, MemoryStore

class PluginEvidenceTests(unittest.TestCase):
    def test_history_requires_each_unique_text_and_author(self):
        row={'text':'nonce','author':'~zod','id':'~zod/123'}
        self.assertEqual(assert_history({'messages':[row]},[('nonce','zod')]),[row])
        for rows in [[],[row,row],[dict(row,author='~nec')],[dict(row,text='unrelated')]]:
            with self.subTest(rows=rows),self.assertRaisesRegex(RuntimeError,'history'):
                assert_history({'messages':rows},[('nonce','zod')])

    def test_live_delivery_requires_kind_peer_and_exact_body(self):
        row={'event':'message','ship':'~zod','message':{'text':'nonce'}}
        self.assertEqual(matched_message([row],'message','zod','nonce'),[row])
        for field,value in [('event','sent'),('ship','~nec'),('message',{'text':'wrong'})]:
            self.assertEqual(matched_message([dict(row,**{field:value})],'message','zod','nonce'),[])

    def test_sse_tap_preserves_split_bytes_and_records_full_event(self):
        raw=b'id: 42\ndata: {"response":"diff",\ndata: "json":{"fake":true}}\n\n'
        rows=[];tap=StreamTap(io.BytesIO(raw),lambda identity,event:rows.append((identity,event)))
        received=b''
        while chunk:=tap.read1(3):received+=chunk
        self.assertEqual(received,raw)
        self.assertEqual(rows,[('42',{'response':'diff','json':{'fake':True}})])

    def test_frame_observation_is_bounded(self):
        tap=StreamTap(io.BytesIO(b'a'*262145),lambda *_:None)
        with self.assertRaisesRegex(RuntimeError,'bound'):tap.read1(262145)

    def test_memory_adapter_checks_account_binding_and_copies(self):
        store=MemoryStore();session={'url':'http://127.0.0.1:18081','ship':'~zod','cookieValue':'fixture'}
        store.save(session);session['ship']='~nec'
        self.assertEqual(store.lookup(session['url'],'~zod')['ship'],'~zod')
        with self.assertRaises(ValueError):store.lookup(session['url'],'~nec')
        store.forget();self.assertIsNone(store.account())

if __name__=='__main__':unittest.main()
