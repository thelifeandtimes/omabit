"""Secret-free evidence for real runner code discovery in the private import VM."""
from datetime import datetime, timezone
import time
import http.client
from http.cookies import SimpleCookie
import json
import os
from pathlib import Path
import re
import subprocess
from urllib.parse import urlencode

ROOT=Path('/opt/omabit-lab')

def require_guest():
    if os.geteuid()!=0 or (ROOT/'guest-marker').read_text()!='offline-import-guest-v1\n' or Path('/sys/class/dmi/id/sys_vendor').read_text().strip()!='QEMU':raise RuntimeError('private code-discovery guest required')
    if sorted(p.name for p in Path('/sys/class/net').iterdir())!=['lo'] or not json.loads((ROOT/'case.json').read_text()).get('exercise_login_code'):raise RuntimeError('wrong code-discovery boundary')

def select(pinned, environment, instance, runner_id, container, record, deadline_seconds=45):
    """Observe the runner's existing fresh-selection contract before revealing once."""
    require_guest()
    deadline=time.monotonic()+deadline_seconds;attempts=[]
    while time.monotonic()<deadline:
        response=subprocess.run(pinned+['snapshot'],env=environment,capture_output=True,timeout=10)
        if response.returncode or response.stderr or len(response.stdout)>1024*1024:raise RuntimeError('fresh runtime selection unavailable')
        try:value=json.loads(response.stdout)
        except Exception:raise RuntimeError('invalid runtime selection') from None
        if value.get('runner_id')!=runner_id:raise RuntimeError('runtime selection runner changed')
        rows=[v for v in value.get('instances',[]) if v.get('id')==instance['id']]
        if len(rows)!=1:raise RuntimeError('runtime selection instance changed')
        current=rows[0];resources=current.get('resources') or {}
        if current.get('config_revision')!=instance['config_revision'] or current.get('ship')!=instance['ship'] or current.get('endpoint')!=instance['endpoint']:raise RuntimeError('runtime selection binding changed')
        if resources.get('container_id') not in (None,'',container):raise RuntimeError('runtime selection container changed')
        try:age=(datetime.now(timezone.utc)-datetime.fromisoformat(current.get('observed_at','').replace('Z','+00:00'))).total_seconds()
        except (TypeError,ValueError):age=None
        ready=(current.get('observed_state')=='running' and not current.get('active_operation_id') and not current.get('recovery_required') and resources.get('container_id')==container and age is not None and -5<=age<=30)
        attempts.append({'state':current.get('observed_state'),'resource_status':resources.get('status'),'container_selected':resources.get('container_id')==container,'fresh_observation':age is not None and -5<=age<=30,'ready':ready})
        record(attempts.copy())
        if ready:return current
        time.sleep(1)
    raise RuntimeError('fresh runtime selection did not become available before deadline')

def verify(pinned, environment, instance, runner_id, container):
    require_guest()
    match=re.fullmatch(r'http://127\.0\.0\.1:([1-9][0-9]{3,4})',instance['endpoint'])
    if not match or not 1024<=int(match[1])<=65535 or instance['ship']!='~zod' or (instance.get('resources') or {}).get('container_id')!=container:raise RuntimeError('unexpected disposable endpoint')
    # Do not use the harness command logger: this stdout intentionally contains
    # a secret. It stays in memory and never enters a receipt or diagnostic.
    response=subprocess.run(pinned+['login-code',instance['id'],'--revision',str(instance['config_revision']),'--container-id',container,'--reveal'],env=environment,capture_output=True,timeout=25)
    if response.returncode or len(response.stdout)>8192 or response.stderr:raise RuntimeError('selected runtime code unavailable (output redacted)')
    try:revealed=json.loads(response.stdout)
    except Exception:raise RuntimeError('invalid code response (output redacted)') from None
    del response
    for key,expected in [('runner_id',runner_id),('instance_id',instance['id']),('config_revision',instance['config_revision']),('container_id',container)]:
        if revealed.get(key)!=expected:raise RuntimeError('code response selection mismatch')
    code=revealed.get('code');del revealed
    if not isinstance(code,str) or not re.fullmatch(r'[a-z]{6}(?:-[a-z]{6}){3}',code):raise RuntimeError('invalid code shape (redacted)')
    def request(method,path,body=None,cookie=None):
        connection=http.client.HTTPConnection('127.0.0.1',int(match[1]),timeout=5)
        headers={'Content-Type':'application/x-www-form-urlencoded'} if body is not None else {}
        if cookie:headers['Cookie']=cookie
        try:
            connection.request(method,path,body,headers)
            result=connection.getresponse();data=result.read(65537)
            if len(data)>65536:raise RuntimeError('private Eyre response bound')
            cookies=result.getheaders();status=result.status
            return status,cookies,data
        finally:connection.close()
    status,headers,body=request('POST','/~/login',urlencode({'password':code}));del code,body
    if status not in (200,204):raise RuntimeError('discovered code did not authenticate')
    jar=SimpleCookie()
    for key,value in headers:
        if key.lower()=='set-cookie':jar.load(value)
    del headers
    values=[v for key,v in jar.items() if key.startswith('urbauth-')]
    if len(values)!=1:raise RuntimeError('missing unique fixture session')
    cookie=values[0].key+'='+values[0].value;del jar,values
    try:
        status,_,body=request('GET','/~/scry/hood/kiln/pikes.json',cookie=cookie)
        if status!=200 or not isinstance(json.loads(body),(dict,list)):raise RuntimeError('discovered code failed authenticated scry')
    finally:
        status,_,_=request('POST','/~/logout',cookie=cookie)
        if status not in (200,204,303):raise RuntimeError('fixture session revocation failed')
    status,_,_=request('GET','/~/scry/hood/kiln/pikes.json',cookie=cookie);del cookie
    if status not in (401,403):raise RuntimeError('fixture session survived logout')
    return {'container_id':container,'instance_id':instance['id'],'config_revision':instance['config_revision'],'code_returned_for_exact_selection':True,'authenticated_scry':True,'session_revoked':True,'no_secret_recorded':True}
