"""Offline checks for state/subscription evidence and mutation input boundaries."""
import json
import unittest
from state_case import matching_state, parse_sse, state_poke
from network_case import cue_atom


def event(value, identity=1):
    return ('id: '+str(identity)+'\ndata: '+json.dumps(value)+'\n\n').encode()


class StateEvidenceTests(unittest.TestCase):
    expected = {'revision': 2, 'entries': ['first', 'second']}

    def test_complete_snapshot_and_unrelated_response(self):
        value = {'id': 1, 'response': 'diff', 'json': self.expected}
        rows = list(parse_sse(event(value).splitlines(keepends=True)))
        self.assertEqual(rows, [(1, value)])
        self.assertTrue(matching_state(rows[0][1], self.expected))
        for unrelated in [{'id': 2, 'response': 'diff', 'json': self.expected},
                          {'id': True, 'response': 'diff', 'json': self.expected},
                          {'id': 1, 'response': 'subscribe', 'ok': 'ok'}]:
            self.assertFalse(matching_state(unrelated, self.expected))

    def test_missing_duplicate_or_reordered_mutation_cannot_pass(self):
        for state in [{'revision': 1, 'entries': ['first']},
                      {'revision': 3, 'entries': ['first', 'first', 'second']},
                      {'revision': 2, 'entries': ['second', 'first']},
                      {'revision': True, 'entries': ['first', 'second']}]:
            with self.subTest(state=state), self.assertRaises(ValueError):
                matching_state({'id': 1, 'response': 'diff', 'json': state}, self.expected)

    def test_broken_subscription_is_not_recovery(self):
        for response in [{'id': 1, 'response': 'quit'},
                         {'id': 1, 'response': 'subscribe', 'err': 'missing-app'}]:
            with self.assertRaises(ValueError):
                matching_state(response, self.expected)

    def test_partial_malformed_or_unbounded_stream_is_not_evidence(self):
        for raw in [event({}).rstrip(), b'data: {}\n\n', event({}, -1),
                    b'id: 1\ndata: []\n\n', b'id: 1\ndata: bad\n\n', b'x'*8193+b'\n']:
            with self.subTest(raw=raw[:40]), self.assertRaises(ValueError):
                list(parse_sse(raw.splitlines(keepends=True)))

    def test_fixed_mutation_does_not_accept_hoon_injection(self):
        token = 'omabit-state-'+'a'*24
        receipt = 'omabit-receipt-'+'b'*32
        encoded = state_poke('nec', token, receipt)
        code = cue_atom(encoded).to_bytes(512, 'little')
        self.assertIn(b'~nec %omabit-state', code)
        for receiver,entry,ack in [('other',token,receipt),('nec',"'; !!",receipt),('nec',token,"'; !!")]:
            with self.assertRaises(ValueError):
                state_poke(receiver,entry,ack)


if __name__ == '__main__':
    unittest.main()
