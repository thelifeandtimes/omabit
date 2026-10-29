import json
import unittest
from console_events import guest_events


class ConsoleEventsTests(unittest.TestCase):
    def test_small_and_wrapped_same_process(self):
        value = {'stage': 'result', 'ok': True, 'values': ['escaped\\n"' * 10000]}
        raw = json.dumps(value)
        chunks = [raw[:49000], raw[49000:98000], raw[98000:]]
        output = 'OMABIT_GUEST {"stage":"start"}\n'
        output += '\n'.join('[ 2.000000] python3[200]: ' + ('OMABIT_GUEST ' if i == 0 else '') + part for i, part in enumerate(chunks))
        self.assertEqual(guest_events(output), [{'stage': 'start'}, value])

    def test_unrelated_or_unframed_continuation_refused(self):
        start = '[ 2.0] python3[200]: OMABIT_GUEST {"stage":"result","ok":'
        for continuation in ('[ 2.1] python3[201]: true}', 'true}', '[ 2.1] other[200]: true}'):
            self.assertEqual(guest_events(start + '\n' + continuation), [])
        self.assertEqual(guest_events(start + '\nnoise\n[ 2.1] python3[200]: true}'), [])

    def test_bounds_and_new_event(self):
        self.assertEqual(guest_events('OMABIT_GUEST '+json.dumps({'stage':'result','x':'a'*262144})), [])
        self.assertEqual(guest_events('OMABIT_GUEST {"stage":\nOMABIT_GUEST {"stage":"failed"}'), [{'stage':'failed'}])
        self.assertEqual(guest_events('prefix OMABIT_GUEST {"stage":"result"}'), [])
        self.assertEqual(guest_events('OMABIT_GUEST [1,2]\nOMABIT_GUEST {"ok":true}'), [])

if __name__ == '__main__':
    unittest.main()
