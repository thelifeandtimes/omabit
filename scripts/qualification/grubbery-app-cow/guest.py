#!/usr/bin/python3
"""New COW guest, fixed task-only predecessor, one fresh ~nec; no installer claim."""
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import sqlite3
import subprocess
import sys
import time
I=Path('/opt/omabit-release-input')
O=Path('/opt/omabit-issues-nec-results')
HOME=Path('/home/review')
report={'version':1,'passed':False,'scope':'actual source service Docker/curl and CLI/stdio on new task-only COW guest; fresh ~nec only; not installed upgrade/DEV/production qualification'}
PIN='a23f10ddc89657489d8e55eaf5e4362603848366a447dbf9f51ce192837a08f9'

def sha(path):
 with Path(path).open('rb') as f:return hashlib.file_digest(f,'sha256').hexdigest()
def guard():
 if os.geteuid()!=0 or sys.argv[1:]:raise RuntimeError('root in fixed offline guest only')
 if Path('/sys/class/dmi/id/sys_vendor').read_text().strip()!='QEMU' or 'omabit.disposable-guest=release-upgrade-v1' not in Path('/proc/cmdline').read_text().split():raise RuntimeError('not isolated guest')
 value=json.loads((I/'scenario.json').read_text())
 if value.get('kind')!='issues18-product-grubbery-v1' or value.get('case')!='cow-fresh-nec-v1' or value.get('predecessor_root_sha256')!=PIN or value.get('target_schema')!=62 or value.get('development_snapshot') is not True:raise RuntimeError('wrong source fixture')
 return value

def original_rows():
 # Diagnostic read on a closed original task fixture, never a repair or admission.
 with sqlite3.connect('file:/home/review/.local/share/urbit-runner/managed/runner.sqlite?mode=ro',uri=True) as db:
  db.execute('BEGIN')
  if db.execute('pragma user_version').fetchone()[0]!=62:raise RuntimeError('wrong original schema')
  tables={row[0] for row in db.execute("select name from sqlite_master where type='table'")}
  names=['grubbery_app_events','grubbery_app_leases','grubbery_app_operations']
  if not set(names+['grubbery_app_clock']).issubset(tables):raise RuntimeError('wrong original app journal layout')
  return {name:list(db.execute('select * from '+name)) for name in names}

def main():
 scenario=guard()
 if O.exists():raise RuntimeError('fresh derived results required')
 O.mkdir(mode=0o700)
 for name,pin in scenario['candidate'].items():
  if sha(I/('candidate/'+name if name!='managed.test' else name))!=pin['sha256']:raise RuntimeError('candidate input differs')
 # Cache comes from the fixed disposable predecessor, not a host mount. The
 # public fake-creation API binds this verified descriptor without re-upload.
 cache=HOME/'.local/share/urbit-runner/managed/boot-pills'/('f8b08c53dd6d0aa9c4f7654f0a7bcd11e94089222888a3c99c0a10eedcbb377b.pill')
 if cache.is_symlink() or sha(cache)!='f8b08c53dd6d0aa9c4f7654f0a7bcd11e94089222888a3c99c0a10eedcbb377b':raise RuntimeError('fixed cached standard pill missing')
 before=original_rows();report['original_app_rows_sha256']=hashlib.sha256(repr(before).encode()).hexdigest()
 spec=importlib.util.spec_from_file_location('common',I/'issues-common.py');common=importlib.util.module_from_spec(spec);spec.loader.exec_module(common)
 common.O=O;common.report=report;common.scenario=scenario
 report.update(inputs=scenario,source_commit=scenario['source_commit'])
 common.run(['systemctl','start','user@1000.service'])
 # Systemd condition and no-resume ExecStart were installed by PID1 before user
 # services. No installer is exercised or its override checks bypassed.
 properties=common.run(['systemctl','--user','show','urbit-runner.service','-p','ActiveState,ExecStart'],True).stdout
 if 'ActiveState=active' in properties or '--resume-running' in properties:raise RuntimeError('old runtime could auto-resume')
 Path('/run/omabit-nec-runner-ready').touch(mode=0o600)
 common.run(['systemctl','--user','start','urbit-runner.service'],True)
 hello=common.wait(common.hello,60)
 if hello['build']['sha256']!=scenario['candidate']['urbit-runner']['sha256']:raise RuntimeError('wrong actual service executable')
 report['runner_id']=hello['runner_id']
 old_proof=json.loads((HOME/'omabit-app-product/proof.json').read_text())
 old_container=old_proof['source-1-plan']['binding']['runtime_id']
 old_inspect=common.inspect(old_container)
 if any(old_inspect['State'].get(k) is not False for k in ('Running','Paused','Restarting')) or old_inspect['HostConfig']['RestartPolicy']['Name']!='no':raise RuntimeError('original disposable container is not safely stopped')
 report['old_container_stopped_before']=True
 helper=subprocess.Popen(common.owner_args(['env','OMABIT_APP_PRODUCT_VM=source-state-restart-nec-v1',I/'managed.test','-test.run=^TestIssuesGrubberyProductDisposableVM$','-test.v','-test.timeout=35m']))
 deadline=time.monotonic()+2150
 try:
  while helper.poll() is None:
   if time.monotonic()>deadline:raise RuntimeError('fresh peer helper deadline')
   common.run(['fstrim','/']);time.sleep(5)
 finally:
  if helper.poll() is None:
   helper.terminate()
   try:helper.wait(timeout=15)
   except subprocess.TimeoutExpired:helper.kill();helper.wait()
 report['helper_exit']=helper.returncode
 # Old app rows must still be exact; new peer app rows can be added only.
 after=original_rows()
 if any(not all(row in after[name] for row in rows) for name,rows in before.items()):raise RuntimeError('old pending app rows changed')
 report['original_app_rows_preserved']=True
 old_after=common.inspect(old_container)
 if any(old_after['State'].get(k) is not False for k in ('Running','Paused','Restarting')) or old_after['State']['StartedAt']!=old_inspect['State']['StartedAt']:raise RuntimeError('old container restarted during fresh peer test')
 report['old_container_stopped_unchanged']=True
 if helper.returncode:raise RuntimeError('source helper failed; private invocation envelopes retained')
 proof=json.loads((HOME/'omabit-app-product-nec/proof.json').read_text())
 if proof.get('original-pending-app-receipt-unchanged') is not True or not all(proof.get('qualification',{}).get(k) is True for k in ('actual_product_docker_transport','native_cli_and_stdio','two_source_versions','state_and_new_code','same_pier_container_replacement','state_after_replacement','manager_restart_original_replay')):raise RuntimeError('not every declared product gate passed')
 report['product']=proof['qualification'];report['proof_sha256']=sha(HOME/'omabit-app-product-nec/proof.json')
 common.operation('candidate','grubbery-nec','stop','product-nec-final-clean-stop')
 report['fresh_peer_final_clean_stop']=True;report['passed']=True

if __name__=='__main__':
 try:main()
 except BaseException as e:report['error']=type(e).__name__+': '+str(e)
 finally:
  if O.is_dir():(O/'result.json').write_text(json.dumps(report,indent=2)+'\n')
  print('OMABIT_GRUBBERY_VM_RESULT '+json.dumps(report,separators=(',',':')),flush=True)
  try:guard();subprocess.run(['fstrim','/'],timeout=60,check=False);subprocess.run(['systemctl','poweroff'],check=True)
  except BaseException:pass
