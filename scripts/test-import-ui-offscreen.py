#!/usr/bin/env python3
"""Actual import/recovery QML and shipped callbacks with an inert runner adapter."""
import argparse
import hashlib
import json
from pathlib import Path
import re
import shutil
import subprocess

ROOT=Path(__file__).resolve().parents[1]
COMPONENTS=('Model.js','StorageLocationSelector.qml','SelectField.qml','Label.qml','Design.qml','PierImportPane.qml','ActionButton.qml','Field.qml','CheckField.qml','Hint.qml','ImportRecoveryPane.qml')
METHODS=('previewExternalPier','rememberInspection','reconcileInspection','rememberImport','reconcileImports','storageConnection','importExternalPier','recoverImport','finish','clearEpochPreview','canMaintain','canSaveRoutine','canReviewTarget','invalidateEpochPreview','invalidateImportCatalogue')

def method(source,name):
    found=re.search(r'^  function '+name+r'\([^\n]*\) \{[^\n]*\}$',source,re.M)
    if not found:found=re.search(r'^  function '+name+r'\([^\n]*\) \{[\s\S]*?^  \}',source,re.M)
    assert found,'missing shipped method '+name
    return found[0]

def main():
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--output',type=Path,required=True,help='fresh /tmp/omabit-import-ui-NAME evidence directory')
    args=parser.parse_args()
    directory=args.output.absolute()
    if directory.parent!=Path('/tmp') or not re.fullmatch('omabit-import-ui-[A-Za-z0-9-]+',directory.name) or directory.exists():parser.error('use a fresh /tmp/omabit-import-ui-NAME directory')
    executable=shutil.which('qml6')
    if not executable:parser.error('Qt 6 qml6 is required')
    directory.mkdir(mode=0o700)
    for leaf in ('home','config','data','state','cache','runtime'):(directory/leaf).mkdir(mode=0o700)
    plugin=directory/'Plugin';plugin.mkdir()
    for name in COMPONENTS:shutil.copyfile(ROOT/'plugins/omabit'/name,plugin/name)
    (plugin/'qmldir').write_text('singleton Design 1.0 Design.qml\n')
    commons=directory/'qs/Commons';commons.mkdir(parents=True)
    (commons/'qmldir').write_text('module qs.Commons\nsingleton Style 1.0 Style.qml\nsingleton Color 1.0 Color.qml\n')
    (commons/'Style.qml').write_text('pragma Singleton\nimport QtQuick\nQtObject { function space(n) {return n} property var font: ({family:"sans-serif",caption:12,bodySmall:14,body:14}) }\n')
    (commons/'Color.qml').write_text('pragma Singleton\nimport QtQuick\nQtObject {property color background:"#171a1e";property color foreground:"#e7e9eb";property color accent:"#8ab4f8";property color urgent:"#ed8994"}\n')
    # Reuse only the inert QtObject adapter from the existing regression template.
    # No imports or execution of test-panel.py / desktop_fixture.py.
    prior=(ROOT/'scripts/test-rc11-panel-offscreen.qml').read_text()
    fixture=prior[prior.index('  QtObject {\n    id: svc'):prior.index('  function check(')]
    fixture=fixture.replace('    property var externalImportPreview: null','    property var pendingExternalPreview: null\n    property var externalImportPreview: null')
    service=(ROOT/'plugins/omabit/Service.qml').read_text()
    fixture=fixture.replace('@@SERVICE_METHODS@@','\n'.join(method(service,name) for name in METHODS))
    fixture=fixture.replace('@@CONNECTIONS_CHANGED@@',re.search(r'^  onConnectionsChanged: .*$',service,re.M)[0])
    qml=(ROOT/'scripts/test-import-ui-offscreen.qml').read_text().replace('@@SERVICE_FIXTURE@@',fixture).replace('@@OUTPUT@@',json.dumps(str(directory)))
    (directory/'main.qml').write_text(qml)
    env={'PATH':'/usr/bin:/bin','LANG':'C.UTF-8','HOME':str(directory/'home'),'XDG_CONFIG_HOME':str(directory/'config'),'XDG_STATE_HOME':str(directory/'state'),'XDG_DATA_HOME':str(directory/'data'),'XDG_CACHE_HOME':str(directory/'cache'),'XDG_RUNTIME_DIR':str(directory/'runtime'),'QT_QPA_PLATFORM':'offscreen','QT_QUICK_BACKEND':'software','QSG_RHI_BACKEND':'software','QT_QUICK_CONTROLS_STYLE':'Basic','QML_DISABLE_DISK_CACHE':'1','QT_LOGGING_RULES':'*.warning=true;*.critical=true;qml.debug=true','QT_FORCE_STDERR_LOGGING':'1'}
    result=subprocess.run([executable,'-I',str(directory),str(directory/'main.qml')],env=env,capture_output=True,text=True,timeout=35)
    log=result.stdout+result.stderr
    (directory/'qml.log').write_text(log)
    assert result.returncode==0 and 'PASS: guided import and recovery UI' in log,log
    assert not any(value in log for value in ('Error:','FAIL:','Unable to assign','Binding loop','Cannot assign','is not defined')),log
    rows=[json.loads(line.split('CALLS ',1)[1]) for line in log.splitlines() if 'CALLS ' in line]
    assert len(rows)==1
    calls=rows[0]
    assert [v['kind'] for v in calls if v['kind']!='preview-pier']==['import-pier','recover-import','recover-import','import-pier'],calls
    assert all(v['kind'] in ('preview-pier','import-pier','recover-import') for v in calls),calls
    for call in calls:
        args=call['args']
        assert args[:4]==['--runner','import-lab','--expected-runner-id','runner-import-lab'],args
        if call['kind']=='import-pier':
            for flag,value in (('--path','@disk/fakeships/friendly/zod'),('--ship','~zod'),('--fingerprint','a'*64),('--http-port','18090')):
                assert args[args.index(flag)+1]==value,args
            assert '--confirm-stopped' in args and '--identify' in args
        if call['kind']=='recover-import':
            assert args[4:9]==['operate','friendly','recover-import','--revision','2'],args
    for name in ('diagnostic','plan','copying','recovered','complete'):assert (directory/(name+'.png')).is_file(),name
    sources=['plugins/omabit/'+name for name in COMPONENTS]+['plugins/omabit/Service.qml','scripts/test-import-ui-offscreen.py','scripts/test-import-ui-offscreen.qml','scripts/test-rc11-panel-offscreen.qml']
    evidence={'ok':True,'exit':result.returncode,'methods':METHODS,'calls':calls,'source_sha256':{name:hashlib.sha256((ROOT/name).read_bytes()).hexdigest() for name in sources},'evidence_sha256':{p.name:hashlib.sha256(p.read_bytes()).hexdigest() for p in directory.glob('*.png')},'environment':env,'scope':'Actual copied QML and verbatim service callbacks; inert dispatch/receipts, private HOME/XDG, offscreen Qt. No Quickshell, desktop socket, Docker or real runner.'}
    evidence['evidence_sha256']['qml.log']=hashlib.sha256(log.encode()).hexdigest()
    (directory/'result.json').write_text(json.dumps(evidence,indent=2)+'\n')
    print('PASS: guided import and recovery UI; explicit reviewed operations, uncertain result retained, no automatic Start')
    print(directory)

if __name__=='__main__':main()
