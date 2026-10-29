#!/usr/bin/env python3
"""Offscreen QML process bridge, admitted only to the disposable guest collector."""
import argparse
import json
import os
from pathlib import Path
import sys

from PySide6.QtCore import QObject, Signal, Slot, Property, QProcess, QTimer, QUrl
from PySide6.QtGui import QGuiApplication
from PySide6.QtQml import QQmlApplicationEngine

ROOT=Path('/opt/omabit-lab')

class Bridge(QObject):
    inventoryReady=Signal(str)
    commandDone=Signal(str,int,str,str)

    def __init__(self,args,app):
        super().__init__();self.args=args;self.app=app;self.jobs={};self.history=[];self.inventories=[];self.finished=False
        self.expected={} if args.preflight else json.loads((args.output/'input.json').read_text())
        self.collector=[str(ROOT/'urbit-collector'),'--config-dir','/home/lab/import-collector']

    @Property(str,constant=True)
    def expectedReview(self):return json.dumps(self.expected)
    @Property(str,constant=True)
    def outputDirectory(self):return str(self.args.output)
    @Property(bool,constant=True)
    def preflight(self):return self.args.preflight

    def start(self,key,arguments):
        if self.finished or key in self.jobs:return
        process=QProcess(self);timer=QTimer(process);timer.setSingleShot(True)
        record={'kind':key,'args':arguments,'stdout':'','stderr':''}
        self.jobs[key]=(process,timer,record)
        def read(channel):
            raw=bytes(process.readAllStandardOutput() if channel=='stdout' else process.readAllStandardError())
            record[channel]+=raw.decode(errors='replace')
            if len(record[channel])>2*1024*1024:self.fail('collector output limit')
        process.readyReadStandardOutput.connect(lambda:read('stdout'))
        process.readyReadStandardError.connect(lambda:read('stderr'))
        def done(code,status):
            read('stdout');read('stderr');timer.stop();self.jobs.pop(key,None)
            record['exit']=code;record['normal_exit']=status==QProcess.ExitStatus.NormalExit
            self.history.append(record)
            if self.finished:return
            if not record['normal_exit']:self.fail('collector process crashed');return
            if key=='inventory':
                if code:self.fail('guest inventory failed');return
                try:
                    inventory=json.loads(record['stdout']);connections=inventory['connections']
                    assert len(connections)==1 and connections[0]['name']=='guest'
                    assert connections[0]['runner_id']==self.expected['runner_id'] and connections[0]['status']=='available'
                    self.inventories.append(inventory);self.inventoryReady.emit(record['stdout'])
                except Exception as error:self.fail('invalid inventory: '+str(error))
            else:self.commandDone.emit(key,code,record['stdout'],record['stderr'])
            process.deleteLater()
        process.finished.connect(done)
        process.errorOccurred.connect(lambda error:self.fail('collector process error: '+str(error)))
        timer.timeout.connect(lambda:self.fail('collector process timed out: '+key))
        process.start(self.collector[0],self.collector[1:]+arguments);timer.start(60000)

    @Slot()
    def refresh(self):
        if self.args.preflight:return
        self.start('inventory',['inventory'])

    @Slot(str,str)
    def submit(self,kind,raw):
        if self.args.preflight:self.fail('preflight attempted a collector command');return
        try:
            arguments=json.loads(raw)
            prefix=['--runner','guest','--expected-runner-id',self.expected['runner_id'],kind,self.expected['instance_id'],'--revision',str(self.expected['config_revision']),'--expected-actor',self.expected['actor']]
            assert kind in ('preview-import-restore','restore-import') and arguments[:len(prefix)]==prefix
            tail=arguments[len(prefix):]
            if kind=='preview-import-restore':assert tail==[]
            else:
                assert len(tail)==5 and tail[:3]==['--restore-digest',self.expected['review_digest'],'--request-id'] and tail[4]=='--confirm-restore'
                assert isinstance(tail[3],str) and 1<=len(tail[3])<=128
            assert not any(item['kind']==kind for item in self.history) and kind not in self.jobs
        except Exception as error:self.fail('unapproved test dispatch: '+str(error));return
        self.start(kind,arguments)

    @Slot(str)
    def fail(self,message):
        if not self.finished:self.finish({'ok':False,'error':message},1)

    @Slot(str)
    def complete(self,raw):
        if self.finished:return
        try:
            report=json.loads(raw);assert report['ok']
            if self.args.preflight:assert report['calls']==[]
            else:
                mutations=[item for item in self.history if item['kind']=='restore-import']
                assert len(mutations)==1 and mutations[0]['exit']==0
                report['restore_args']=mutations[0]['args'];report['ack']=json.loads(mutations[0]['stdout'])
                assert report['ack']['id']==report['outcome']['receipt']['id']
        except Exception as error:self.fail('completion evidence invalid: '+str(error));return
        self.finish(report,0)

    def finish(self,report,code):
        self.finished=True
        for process,timer,record in list(self.jobs.values()):
            timer.stop();process.terminate()
            if not process.waitForFinished(2000):process.kill();process.waitForFinished(2000)
        report['processes']=self.history;report['inventories']=self.inventories
        (self.args.output/'result.json').write_text(json.dumps(report,indent=2)+'\n')
        self.app.exit(code)

def main():
    parser=argparse.ArgumentParser(description=__doc__);parser.add_argument('--output',type=Path,required=True);parser.add_argument('--preflight',action='store_true');args=parser.parse_args()
    if os.environ.get('QT_QPA_PLATFORM')!='offscreen':raise RuntimeError('offscreen Qt required')
    if not args.preflight:
        if os.getuid()!=1000 or (ROOT/'guest-marker').read_text()!='offline-import-guest-v1\n' or Path('/sys/class/dmi/id/sys_vendor').read_text().strip()!='QEMU':raise RuntimeError('fresh guest owner required')
        if sorted(p.name for p in Path('/sys/class/net').iterdir())!=['lo']:raise RuntimeError('guest must have no network card')
        if args.output!=ROOT/'results/ui-restore' or not json.loads((ROOT/'case.json').read_text()).get('restore_ui'):raise RuntimeError('wrong restoration case')
    app=QGuiApplication([]);bridge=Bridge(args,app);engine=QQmlApplicationEngine()
    bundle=Path(__file__).resolve().parent;engine.addImportPath(str(bundle));engine.rootContext().setContextProperty('bridge',bridge)
    engine.quit.connect(app.quit);engine.load(QUrl.fromLocalFile(str(bundle/'main.qml')))
    if not engine.rootObjects():bridge.fail('QML failed to load');return 1
    code=app.exec()
    if not bridge.finished:bridge.fail('QML exited without evidence');return 1
    return code

if __name__=='__main__':sys.exit(main())
