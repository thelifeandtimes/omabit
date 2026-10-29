"""Explicit private linked-authority snapshots; no admission or cleanup."""
from pathlib import Path
import json,os,subprocess,sys
sys.path.insert(0,'/lab/source')
import successor_inspection as inspection
import runtime_authority as authority
from journal_runtime_fixture import ROOT,guard,read,save,sha


def snapshot():
 roots=[ROOT/'collection-journal',ROOT/'successor-runtime'/authority.ANCHOR,ROOT/'gates',ROOT/'runtime-observations',ROOT/'receipts']
 files=[ROOT/'successor-runtime'/authority.ANCHOR_SEAL,ROOT/'successor-runtime'/authority.PROVISION_CLAIM]
 files+=[p for root in roots for p in root.rglob('*') if p.is_file()]
 return {str(p.relative_to(ROOT)):{'inode':p.stat().st_ino,'sha256':sha(p)} for p in files}


def observe(label):
 guard();before=snapshot();root=ROOT/'successor-runtime'/authority.ANCHOR;store=ROOT/'collection-journal'
 value=inspection.inspect(root,store);assert snapshot()==before and value['format']==2
 assert not value['activation_authorized'] and not value['transport_cleanup_verified']
 command=[sys.executable,'-B','/lab/inspect-collection-authority.py','--authority',str(root),'--coordinator',str(store),'--details']
 q=subprocess.run(command,env=dict(os.environ,PYTHONPATH='/lab/source'),capture_output=True,text=True,timeout=12)
 assert q.returncode==0,q.stderr;assert json.loads(q.stdout)==value and snapshot()==before
 path=ROOT/'successor-inspection-result.json';report=read(path) if path.exists() else {'passed':True,'observations':{}}
 assert label not in report['observations'];report['observations'][label]={'inspection':value,'unchanged':before,'cli_exit':q.returncode}
 save(path,report);return value
