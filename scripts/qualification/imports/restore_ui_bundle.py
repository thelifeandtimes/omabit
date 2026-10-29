"""Freeze shipped QML/callbacks with a guest-only subprocess adapter."""
import hashlib
import importlib.util
from pathlib import Path
import re
import shutil


def prepare(repo, target):
    target.mkdir(mode=0o755)
    plugin=target/'Plugin';plugin.mkdir()
    sources=[]
    for source in sorted((repo/'plugins/omabit').glob('*')):
        if source.suffix in ('.qml','.js'):
            shutil.copyfile(source,plugin/source.name);sources.append(source)
    (plugin/'qmldir').write_text('singleton Design 1.0 Design.qml\n')
    commons=target/'qs/Commons';commons.mkdir(parents=True)
    (commons/'qmldir').write_text('module qs.Commons\nsingleton Style 1.0 Style.qml\nsingleton Color 1.0 Color.qml\n')
    (commons/'Style.qml').write_text('pragma Singleton\nimport QtQuick\nQtObject {function space(n){return n} property var font:({family:"DejaVu Sans",caption:12,bodySmall:14,body:14})}\n')
    (commons/'Color.qml').write_text('pragma Singleton\nimport QtQuick\nQtObject {property color background:"#171a1e";property color foreground:"#e7e9eb";property color accent:"#8ab4f8";property color urgent:"#ed8994"}\n')
    helper=repo/'scripts/test-import-ui-offscreen.py'
    spec=importlib.util.spec_from_file_location('inert_import_fixture',helper)
    module=importlib.util.module_from_spec(spec);spec.loader.exec_module(module)
    prior=repo/'scripts/test-rc11-panel-offscreen.qml';text=prior.read_text()
    fixture=text[text.index('  QtObject {\n    id: svc'):text.index('  function check(')]
    fixture=fixture.replace('    property var externalImportPreview: null','    property var pendingExternalPreview: null\n    property var externalImportPreview: null\n    readonly property P.ImportRestoration importRestoration: P.ImportRestoration {service:svc}')
    old='calls=calls.concat([{args:args,kind:kind}]);currentAction=kind;return true'
    assert fixture.count(old)==1
    fixture=fixture.replace(old,'calls=calls.concat([{args:args,kind:kind}]);currentAction=kind;bridge.submit(kind,JSON.stringify(args));return true')
    service=repo/'plugins/omabit/Service.qml';text=service.read_text()
    fixture=fixture.replace('@@SERVICE_METHODS@@','\n'.join(module.method(text,n) for n in module.METHODS))
    fixture=fixture.replace('@@CONNECTIONS_CHANGED@@',re.search(r'^  onConnectionsChanged: .*$',text,re.M)[0])
    template=repo/'scripts/qualification/imports/restore_ui.qml'
    (target/'main.qml').write_text(template.read_text().replace('@@SERVICE_FIXTURE@@',fixture))
    bridge=repo/'scripts/qualification/imports/restore_ui_guest.py'
    shutil.copyfile(bridge,target/'run.py')
    sources.extend([helper,prior,service,template,bridge,Path(__file__)])
    return {str(p.relative_to(repo)):hashlib.sha256(p.read_bytes()).hexdigest() for p in sources}
