"""Direct guest-loopback Eyre client for the opt-in no-NIC fake publication lab."""
import http.client
from http.cookies import SimpleCookie
import re
import socket
import time
from urllib.parse import urlencode


class PublishedClient:
    """Keep session cookies in memory; receipts never contain request/response data."""
    def __init__(self, bindings, record, save):
        if bindings not in ({8081: 18281, 8082: 18282}, {8081: 18291, 8082: 18292}):
            raise RuntimeError('unbound published fake endpoint set')
        self.bindings = dict(bindings)
        self.cookies = {}
        self.record = record
        self.save = save

    @staticmethod
    def cookie_key(value):
        if value not in ('/tmp/omabit-zod-cookie', '/tmp/omabit-nec-cookie', '/tmp/omabit-cross-zod-cookie', '/tmp/omabit-cross-nec-cookie'):
            raise RuntimeError('unbound published session handle')
        return value

    def request(self, port, path, data=None, cookie=None, jar=None, content=None,
                accept=None, timeout=30, form=False, method=None, cursor=None, stream=False):
        if port not in self.bindings or not path.startswith('/') or path.startswith('//') or '\r' in path or '\n' in path:
            raise RuntimeError('unbound published request')
        headers = {}
        if cookie:
            key = self.cookie_key(cookie)
            if key not in self.cookies: raise RuntimeError('published session unavailable')
            headers['Cookie'] = self.cookies[key]
        if jar:
            self.cookie_key(jar)
            if not form or path != '/~/login' or jar not in ('/tmp/omabit-zod-cookie','/tmp/omabit-nec-cookie'): raise RuntimeError('session can only be captured during login')
        if form:
            if not isinstance(data, bytes): raise RuntimeError('invalid fixture login body')
            data = urlencode({'password': data.decode()}).encode()
            content = 'application/x-www-form-urlencoded'
        if content: headers['Content-Type'] = content
        if accept: headers['Accept'] = accept
        if cursor is not None:
            if type(cursor) is not int or cursor < 0: raise RuntimeError('invalid stream cursor')
            headers['Last-Event-ID'] = str(cursor)
        verb = method or ('POST' if data is not None else 'GET')
        if verb not in ('GET', 'POST', 'PUT'): raise RuntimeError('invalid published method')
        connection = http.client.HTTPConnection('127.0.0.1', self.bindings[port], timeout=timeout)
        try:
            connection.request(verb, path, data, headers)
            response = connection.getresponse()
            if stream:
                chunks = []; size = 0; end = time.monotonic() + timeout
                while time.monotonic() < end:
                    try: chunk = response.read1(min(4096, 65537-size))
                    except (TimeoutError, socket.timeout): break
                    if not chunk: break
                    chunks.append(chunk); size += len(chunk)
                    if size > 65536: raise RuntimeError('published stream bound')
                body = b''.join(chunks)
            else:
                body = response.read(65537)
                if len(body) > 65536: raise RuntimeError('published response bound')
            if jar and response.status in (200, 204):
                parsed = SimpleCookie()
                for key, value in response.getheaders():
                    if key.lower() == 'set-cookie': parsed.load(value)
                values = [(k,v.value) for k,v in parsed.items() if re.fullmatch('urbauth-~(?:zod|nec)', k)]
                expected = 'urbauth-~' + ('zod' if jar == '/tmp/omabit-zod-cookie' else 'nec')
                if len(values) != 1 or values[0][0] != expected or not values[0][1]: raise RuntimeError('published session identity mismatch')
                self.cookies[jar] = values[0][0] + '=' + values[0][1]
            self.record.append({'port':port, 'published_port':self.bindings[port],
                'method':verb, 'path':path, 'status':response.status, 'stream':stream,
                'authenticated':cookie is not None, 'response_bytes':len(body)})
            if len(self.record) > 1000: raise RuntimeError('published receipt bound')
            self.save()
            return response.status, body
        except (OSError, http.client.HTTPException):
            # A failed mutation stays uncertain. Never replay or expose secret output.
            raise RuntimeError('direct published HTTP failed (request not replayed)') from None
        finally: connection.close()

    def events(self, port, channel, cookie, cursor):
        status, body = self.request(port, channel, cookie=cookie, cursor=cursor,
            accept='text/event-stream', timeout=3, stream=True)
        if status != 200: raise RuntimeError('published subscription refused')
        return body

    def reject_foreign_session(self, source, ship):
        """Try a separate lab project's same-identity session without file transfer."""
        if ship not in ('zod','nec') or source is self or source.bindings == self.bindings:
            raise RuntimeError('foreign session requires distinct bound lab projects')
        origin='/tmp/omabit-'+ship+'-cookie';temporary='/tmp/omabit-cross-'+ship+'-cookie'
        if origin not in source.cookies or temporary in self.cookies:
            raise RuntimeError('foreign session fixture unavailable')
        self.cookies[temporary]=source.cookies[origin]
        try:
            status,_=self.request(8081 if ship=='zod' else 8082,'/~/scry/hood/kiln/pikes.json',cookie=temporary)
            if status not in (401,403):raise RuntimeError('same-identity session crossed published project boundary')
            return status
        finally:del self.cookies[temporary]

    def logout(self):
        checks=[]
        for ship,port in [('zod',8081),('nec',8082)]:
            key='/tmp/omabit-'+ship+'-cookie'
            if key not in self.cookies: continue
            status,_=self.request(port,'/~/logout',cookie=key,method='POST')
            if status not in (200,204,303): raise RuntimeError('published session logout refused')
            status,_=self.request(port,'/~/scry/hood/kiln/pikes.json',cookie=key)
            if status not in (401,403): raise RuntimeError('published session survived logout')
            del self.cookies[key]
            checks.append({'ship':ship,'revoked':True})
        return checks
