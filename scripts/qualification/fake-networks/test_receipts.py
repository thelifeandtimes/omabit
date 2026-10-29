"""Qualification receipt matching; no runtime or network access."""
import json
import unittest
from network_case import poke_receipt, thread_poke, thread_receipt, jam, cue_atom


def sse(event, identity='0'):
    return ('id: '+identity+'\ndata: '+json.dumps(event)+'\n\n').encode()


class ReceiptTests(unittest.TestCase):
    def setUp(self):
        self.accepted = sse({'ok': 'ok', 'id': 1, 'response': 'poke'})
        self.rejected = sse({'id': 1, 'response': 'poke', 'err': 'poke-hi-fail\n'})

    def test_application_outcomes_are_distinct(self):
        self.assertEqual(poke_receipt(self.accepted)[0], 0)
        self.assertEqual(poke_receipt(self.rejected, reject=True)[0], 0)
        for stream, reject in [(self.accepted, True), (self.rejected, False)]:
            with self.assertRaises(ValueError):
                poke_receipt(stream, reject=reject)

    def test_only_exact_complete_receipt_is_accepted(self):
        for stream in [self.accepted.rstrip(), self.accepted+self.accepted,
                       sse({'id': 2, 'response': 'poke', 'ok': 'ok'}),
                       sse({'id': True, 'response': 'poke', 'ok': 'ok'}),
                       sse({'id': 1, 'response': 'poke', 'ok': 'ok'}, '-1'),
                       b'data: not json\n\n', b'x'*65537]:
            with self.subTest(stream=stream[:100]), self.assertRaises(ValueError):
                poke_receipt(stream)

    def test_transport_or_conversion_failure_is_not_application_rejection(self):
        for event in [{'id': 1, 'response': 'poke', 'err': 'missing mark'},
                      {'id': 1, 'response': 'poke', 'err': 'poke-hi-fail', 'ok': 'ok'}]:
            with self.assertRaises(ValueError):
                poke_receipt(sse(event), reject=True)


class ThreadReceiptTests(unittest.TestCase):
    receipt='omabit-receipt-'+'a'*32

    def test_positive_receipt_is_bound_and_complete(self):
        body=jam(int.from_bytes(self.receipt.encode(),'little'))
        self.assertEqual(thread_receipt(200,body,self.receipt),self.receipt)
        for status,data,expected in [(500,body,self.receipt),(200,body,'different'),
            (200,body+b'\xff',self.receipt),(200,body[:-1],self.receipt),(200,b'x'*65537,self.receipt)]:
            with self.subTest(status=status,data=data[:40]),self.assertRaises(ValueError):
                thread_receipt(status,data,expected)

    def test_negative_receipt_requires_actual_gall_failure(self):
        value={'term':'child-failed','tang':[['thread-fail'],['poke-fail'],['poke-hi-fail']]}
        body=json.dumps(value).encode()
        self.assertEqual(thread_receipt(500,body,self.receipt,True),value)
        for status,data in [(200,body),(500,b'crud!'),(500,b'{"term":"child-failed","tang":[["missing mark"]]}'),
            (500,b'{"term":"child-failed","tang":[["thread-fail","poke-fail","not-poke-hi-fail"]]}'),
            (500,b'{"term":"child-failed","tang":"poke-hi-fail"}')]:
            with self.subTest(data=data),self.assertRaises(ValueError):thread_receipt(status,data,self.receipt,True)

    def test_request_rejects_uncontrolled_hoon_strings(self):
        nonce='omabit-first-'+'b'*24
        self.assertIn(b'~nec %hood',cue_atom(thread_poke('nec',nonce,self.receipt)).to_bytes(512,'little'))
        for receiver,message,receipt in [('other',nonce,self.receipt),('nec',"'; !!",self.receipt),('nec',nonce,"'; !!")]:
            with self.assertRaises(ValueError):thread_poke(receiver,message,receipt)


if __name__ == '__main__':
    unittest.main()
