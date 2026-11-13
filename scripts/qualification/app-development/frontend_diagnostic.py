"""Read an already-uploaded synthetic frontend; never upload or edit its desk."""
import hashlib
import http.client
import http.cookies
import json


def inspect_frontend(lab, desk, lens):
    # The outer capsule checks the exact transferred fake's identity and stopped
    # receipts. This selection is its original random desk, never caller Hoon.
    result = {'desk': desk, 'app_mutations': 0, 'assets': []}
    # probe.lens substitutes this fixed placeholder once with the selected desk.
    inner = ("=/  row  (~(got by charges) %omabit-test)  "
             "?>  ?=(%glob -.chad.row)  "
             "(turn ~(tap by glob.chad.row) |=([p=path q=mime] [p p.q p.q.q]))")
    expression = ("=/  state  .^(vase %gx /~zod/docket/(scot %da now)/dbug/state/noun)  "
                  "q:(slap (slop state !>(..zuse)) (ream '" + inner + "'))")
    result['native_paths_types_lengths'] = lens('frontend-metadata-diagnostic', expression, required=False)
    cookie = None
    connection = http.client.HTTPConnection('127.0.0.1', 18081, timeout=10)
    try:
        connection.request('POST', '/~/login', body=b'password=lidlut-tabwed-pillex-ridrup',
                           headers={'Content-Type': 'application/x-www-form-urlencoded'})
        response = connection.getresponse()
        response.read(1048577)
        if response.status not in (200, 204):
            raise RuntimeError('private diagnostic login failed')
        parsed = http.cookies.SimpleCookie()
        parsed.load(response.getheader('Set-Cookie', ''))
        if 'urbauth-~zod' not in parsed:
            raise RuntimeError('private diagnostic session absent')
        cookie = 'urbauth-~zod=' + parsed['urbauth-~zod'].value
        for name in ('assets/Upper_case.file.png', 'empty.txt', 'index.html'):
            connection.request('GET', '/apps/' + desk + '/' + name, headers={'Cookie': cookie})
            response = connection.getresponse()
            data = response.read(1025)
            if len(data) > 1024:
                raise RuntimeError('synthetic asset exceeds diagnostic bound')
            result['assets'].append({'path': name, 'status': response.status,
                                     'content_type': response.getheader('Content-Type'),
                                     'bytes': len(data), 'sha256': hashlib.sha256(data).hexdigest(),
                                     'synthetic_body_hex': data.hex()})
    finally:
        if cookie:
            connection.request('POST', '/~/logout', body=b'', headers={'Cookie': cookie})
            response = connection.getresponse()
            response.read(1048577)
            result['session_cleanup'] = response.status in (200, 204, 303)
        connection.close()
        (lab / 'frontend-diagnostic.json').write_text(json.dumps(result, indent=2) + '\n')
    return result
