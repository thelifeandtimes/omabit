#!/usr/bin/env python3
"""Compare real Theme and DMs native popup geometry and render light/dark/narrow views.

Only copied QML/JS and inert process/file/IPC adapters enter a networkless fixture.
No backend helper, production HOME, keyring, Docker or compositor is accessible.
"""
import argparse
import hashlib
import json
from pathlib import Path
import re
import shutil
import subprocess

UI = ['Panel','PanelController','BarWidget','BarIconButton','WidgetButton','OpticalGlyph',
      'Button','TextField','ToggleSwitch','PanelHero','PanelSeparator','PanelSectionHeader',
      'BorderSurface','BorderOverlay','PanelKeyCatcher','PanelActionButton','PanelToolTip']
QML = (Path(__file__).parent / "qualification/desktop/plugin-cohesion.qml").read_text()

def main():
    p=argparse.ArgumentParser(description=__doc__)
    p.add_argument('--theme',type=Path,required=True)
    p.add_argument('--dms',type=Path,required=True)
    p.add_argument('--output',type=Path,required=True)
    args=p.parse_args()
    work=args.output.absolute()
    if work.parent!=Path('/tmp') or not re.fullmatch('omabit-plugin-cohesion-[a-zA-Z0-9-]+',work.name) or work.exists():
        p.error('use a fresh /tmp/omabit-plugin-cohesion-NAME')
    work.mkdir(mode=0o700)
    for name in ('home','runtime','tests'): (work/name).mkdir(mode=0o700)
    shutil.copytree(args.theme/'tests/qml/imports',work/'imports')
    native=Path('/usr/share/omarchy/shell')
    for name in UI: shutil.copyfile(native/'Ui'/(name+'.qml'),work/'imports/qs/Ui'/(name+'.qml'))
    (work/'imports/qs/Ui/qmldir').write_text('module qs.Ui\n'+''.join(n+' 1.0 '+n+'.qml\n' for n in [*UI,'KeyboardPanel']))
    for name in ('Style.qml','Util.qml','Border.qml','BorderGeometry.js'):
        shutil.copyfile(native/'Commons'/name,work/'imports/qs/Commons'/name)
    hashes={}
    for src,name in ((args.theme,'theme'),(args.dms,'dms')):
        dst=work/name;dst.mkdir()
        paths=[*src.glob('*.qml'),*src.glob('*.js')]
        if (src/'qmldir').exists(): paths.append(src/'qmldir')
        for path in paths:
            text=path.read_text();hashes[name+'/'+path.name]=hashlib.sha256(path.read_bytes()).hexdigest()
            if path.name=='Dms.qml':
                assert text.count('  PanelWindow {')==1
                text=text.replace('import Quickshell.Wayland\n','').replace('  PanelWindow {','  Rectangle {\n    width: 1024; height: 768')
                text=text.replace('    anchors { top: true; bottom: true; left: true; right: true }\n','')
                text='\n'.join(line for line in text.split('\n') if not line.strip().startswith(('WlrLayershell.','exclusionMode:')))
            (dst/path.name).write_text(text)
    for name in ('PluginHeader.qml','ConnectionChoices.qml'):
        if (args.theme/name).read_bytes()!=(args.dms/name).read_bytes(): raise RuntimeError('shared presentation drift: '+name)
    (work/'tests/tst_cohesion.qml').write_text(QML)
    cmd=['bwrap','--unshare-all','--die-with-parent','--clearenv','--ro-bind','/usr','/usr',
         '--symlink','usr/bin','/bin','--symlink','usr/lib','/lib','--symlink','usr/lib','/lib64',
         '--proc','/proc','--dev','/dev','--tmpfs','/tmp','--dir','/etc','--ro-bind','/etc/fonts','/etc/fonts',
         '--bind',str(work),'/lab','--chdir','/lab']
    for k,v in {'HOME':'/lab/home','XDG_RUNTIME_DIR':'/lab/runtime','PATH':'/usr/bin:/bin',
                'QT_QPA_PLATFORM':'offscreen','QT_QUICK_BACKEND':'software','QML_DISABLE_DISK_CACHE':'1',
                'QT_QUICK_CONTROLS_STYLE':'Basic'}.items():cmd+=['--setenv',k,v]
    cmd+=['--','/usr/lib/qt6/bin/qmltestrunner','-input','/lab/tests','-import','/lab/imports']
    result=subprocess.run(cmd,capture_output=True,text=True,timeout=30)
    output=result.stdout+result.stderr
    passed=result.returncode==0 and not any(s in output for s in ('QWARN','ReferenceError','TypeError','FAIL!'))
    passed=passed and all((work/name).is_file() for name in ('popups-dark.png','popups-light.png','popups-narrow.png','popups-connected.png','theme-sign-in.png','dms-sign-in.png'))
    (work/'results.log').write_text(output)
    (work/'evidence.json').write_text(json.dumps({'passed':passed,'source_sha256':hashes,'inert_io':True,'networkless':True,'native_compositor':False},indent=2)+'\n')
    print(output);print(work)
    return 0 if passed else 1

if __name__=='__main__': raise SystemExit(main())
