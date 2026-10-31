#!/usr/bin/python3
"""Actual sealed-archive loader, entirely inside the disposable guest."""
import json,sys
from pathlib import Path
I=Path('/opt/omabit-release-input')
assert 'omabit.disposable-guest=release-upgrade-v1' in Path('/proc/cmdline').read_text().split()
sys.path.insert(0,str(I/'bootstrap'));import desktop
old=sys.argv[1];assert old == '0.3.3'
pins=Path.home()/'rehearsal-pins.json'
pins.write_text(json.dumps(desktop.descriptor_from_release(json.loads((I/old/'release.json').read_text()))))
d=desktop.Desktop(pins)
records=[]
with d.lease() as session:
 for version in (old,'0.3.4'):
  release=d.import_release(I/version/'release.json',I/version/('omabit-'+version+'-linux-amd64.tar.gz'))['release']
  d.select('default',release);plan=d.resolve();prepared=d.prepare()
  d.activation('begin',session,plan['request']);d.activation('commit',session,plan['request'])
  records.append({'version':version,'release':release,'source_commit':prepared['source_commit']})
print(json.dumps({'records':records,'directory':prepared['directory'],'loader':'actual native sealed archive importer; activation acknowledgements driven by guest harness'}))
