#!/usr/bin/env python3
"""Real layer-shell picker input on private headless Sway; no host desktop access.

Requires an extracted Sway/wlroots tool root (--tools), plus installed Quickshell,
wtype and grim. No packages are installed. --baseline reproduces the old separate
window behavior using only a copied component. Evidence is retained in --output.
"""
import argparse
import hashlib
import json
import os
from pathlib import Path
import shutil
import socket
import struct
import subprocess
import sys
import time

ROOT = Path(__file__).resolve().parents[1]


class PrivatePointer:
    """Minimal wlr-virtual-pointer v1 client, restricted to the private lab socket.

    Sway's cursor IPC alone does not advertise pointer capability on an empty
    headless seat, so Qt never creates a pointer. Keep a virtual device alive.
    """
    def __init__(self, path):
        assert path.parent == Path('/lab/runtime') and path.is_socket()
        self.socket = socket.socket(socket.AF_UNIX)
        self.socket.settimeout(5)
        self.socket.connect(str(path))
        self.next_id = 3
        self.globals = {}
        self.send(1, 1, struct.pack('=I', 2))  # wl_display.get_registry
        self.sync()
        name = 'zwlr_virtual_pointer_manager_v1'
        encoded = name.encode() + b'\0'
        encoded += b'\0' * (-len(encoded) % 4)
        self.send(2, 0, struct.pack('=II', self.globals[name], len(name)+1)
                  + encoded + struct.pack('=II', 1, 4))
        self.send(4, 0, struct.pack('=II', 0, 5))  # create_virtual_pointer
        self.next_id = 6
        self.sync()

    def send(self, object_id, opcode, payload=b''):
        self.socket.sendall(struct.pack('=II', object_id, (len(payload)+8) << 16 | opcode) + payload)

    def receive(self, size):
        result = b''
        while len(result) < size:
            chunk = self.socket.recv(size-len(result))
            assert chunk, 'private Wayland connection closed'
            result += chunk
        return result

    def sync(self):
        callback = self.next_id
        self.next_id += 1
        self.send(1, 0, struct.pack('=I', callback))
        while True:
            object_id, header = struct.unpack('=II', self.receive(8))
            payload = self.receive((header >> 16)-8)
            opcode = header & 0xffff
            assert not (object_id == 1 and opcode == 0), repr(payload)
            if object_id == 2 and opcode == 0:
                name, length = struct.unpack('=II', payload[:8])
                self.globals[payload[8:8+length-1].decode()] = name
            if object_id == callback:
                return

    def click(self, x, y, twice=False):
        timestamp = lambda: int(time.monotonic()*1000) & 0xffffffff
        self.send(5, 1, struct.pack('=IIIII', timestamp(), round(x), round(y), 1280, 960))
        self.send(5, 4)
        self.sync()
        time.sleep(.05)
        for _ in range(2 if twice else 1):
            for pressed in (1, 0):
                self.send(5, 2, struct.pack('=III', timestamp(), 272, pressed))
                self.send(5, 4)
                self.sync()
                time.sleep(.04)


def inside():
    lab=Path('/lab')
    assert Path.home()==lab/'home' and not Path('/run/user').exists()
    env=dict(os.environ, LD_LIBRARY_PATH='/tools/usr/lib', WLR_BACKENDS='headless',
             WLR_RENDERER='pixman', WLR_HEADLESS_OUTPUTS='1')
    def run(*args):
        return subprocess.check_output(args, env=env, text=True, stderr=subprocess.STDOUT, timeout=5).strip()
    def wait(check, seconds=8):
        deadline=time.monotonic()+seconds
        while time.monotonic()<deadline:
            result=check()
            if result:return result
            time.sleep(.08)
        raise AssertionError('private fixture deadline')
    def children(node):
        result=[node]
        for child in node.get('nodes',[])+node.get('floating_nodes',[]):result+=children(child)
        return result
    compositor=None
    shell=None
    pointer=None
    keyboard=None
    report={'passed':False,'baseline':(lab/'baseline').exists(),'host_desktop_access':False}
    try:
        with (lab/'sway.log').open('w') as log:
            compositor=subprocess.Popen(['/tools/usr/bin/sway','--config','/lab/sway.conf'],env=env,stdout=log,stderr=log)
        def ready():
            assert compositor.poll() is None,'compositor exited: '+(lab/'sway.log').read_text()
            displays=list((lab/'runtime').glob('wayland-*'))
            sockets=list((lab/'runtime').glob('sway-ipc.*.sock'))
            displays=[p for p in displays if p.is_socket()]
            return (displays[0],sockets[0]) if displays and sockets else None
        display,ipc=wait(ready)
        env.update(WAYLAND_DISPLAY=display.name,SWAYSOCK=str(ipc))
        pointer=PrivatePointer(display)
        def sway(*args):
            result=run('/tools/usr/bin/swaymsg','-r',*args)
            if '-t' not in args:
                assert all(row.get('success') for row in json.loads(result)),result
            return result
        keyboard=subprocess.Popen(['wtype','-s','45000'],env=env,stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL)
        wait(lambda:any(s['capabilities'] & 2 for s in json.loads(sway('-t','get_seats'))))
        def call(method,*args):return run('qs','ipc','--pid',str(shell.pid),'call','--','picker-fixture',method,*args)
        def state():return json.loads(call('status'))
        def click(rect, twice=False):
            assert rect and rect['width']>0 and rect['height']>0,rect
            pointer.click(rect['x'],rect['y'],twice)
        with (lab/'shell.log').open('w') as log:
            shell=subprocess.Popen(['qs','--no-color','-p','/lab/shell.qml'],env=env,stdout=log,stderr=log)
        def shell_ready():
            assert shell.poll() is None,'shell exited: '+(lab/'shell.log').read_text()
            try:return state()
            except subprocess.CalledProcessError:return None
        wait(shell_ready)
        assert call('openMenu')=='true';time.sleep(.4)
        call('revealBrowse');time.sleep(.1)
        report['opened']=state()
        (lab/'tree-before.json').write_text(sway('-t','get_tree','-r')+'\n')
        browse=json.loads(call('rectangle','key-file-browse'))
        report['browse']=browse
        run('grim','/lab/before-browse.png')
        click(browse)
        report['after_browse']=state()
        (lab/'seats.json').write_text(sway('-t','get_seats')+'\n')
        run('grim','/lab/after-browse.png')
        wait(lambda:state()['visible']);time.sleep(.4)
        tree=json.loads(sway('-t','get_tree','-r'))
        (lab/'tree-open.json').write_text(json.dumps(tree,indent=2)+'\n')
        run('grim','/lab/picker-open.png')
        normal=[n for n in children(tree) if n.get('app_id') or n.get('window')]
        report['normal_window_titles']=[n.get('name') for n in normal]
        if report['baseline']:
            chooser=next(n for n in normal if n.get('name')=='Select Urbit boot key')
            rect=chooser['rect']
            pointer.click(rect['x']+rect['width']*.55,rect['y']+chooser['window_rect']['y']+80,twice=True)
            time.sleep(.2)
            report['after_pointer_selection']=state()
            assert state()['visible'] and state()['path']=='/previous/key.key','baseline unexpectedly accepted pointer selection'
        run('wtype','-k','Escape');time.sleep(.4)
        report['after_escape']=state()
        if report['baseline']:
            assert len(normal)>1,'baseline did not create a separate chooser window'
            report['reproduced']=True
        else:
            assert len(normal)==1,report['normal_window_titles']
            assert state()['open'] and not state()['visible'] and state()['path']=='/previous/key.key',state()
            click(json.loads(call('rectangle','key-file-browse')))
            wait(lambda:state()['visible'])
            entry=wait(lambda:json.loads(call('rectangle','fileDialogDelegate0')))
            assert entry['fileName']=='owned key λ #1.key',entry
            click(entry,twice=True)
            wait(lambda:not state()['visible'])
            assert state()['path']=='/lab/files/owned key λ #1.key' and state()['browseEnabled'],state()
            # Reopen/cancel by actual key input and return input to another window.
            click(json.loads(call('rectangle','key-file-browse')))
            wait(lambda:state()['visible']);run('wtype','-k','Escape')
            wait(lambda:not state()['visible'])
            assert state()['path']=='/lab/files/owned key λ #1.key'
            call('hide');time.sleep(.4)
            sway('[title="Picker fixture background"]','focus')
            run('wtype','input-restored')
            wait(lambda:state()['backgroundText']=='input-restored')
            assert state()['submissions']==0
            report['final']=state()
            run('grim','/lab/input-restored.png')
        report['passed']=True
        print('PASS: '+('separate-window baseline reproduces the blocked chooser' if report['baseline'] else 'real Wayland overlay Browse, Escape, pointer selection and restored background input'))
    finally:
        (lab/'result.json').write_text(json.dumps(report,indent=2)+'\n')
        if pointer is not None:pointer.socket.close()
        for proc in (shell,keyboard,compositor):
            if proc is not None and proc.poll() is None:
                proc.terminate()
                try:proc.wait(timeout=5)
                except subprocess.TimeoutExpired:proc.kill();proc.wait(timeout=5)


def main():
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--inside',action='store_true')
    parser.add_argument('--tools',type=Path)
    parser.add_argument('--output',type=Path)
    parser.add_argument('--baseline',action='store_true')
    args=parser.parse_args()
    if args.inside:return inside()
    if not args.tools or not args.output:parser.error('--tools and --output required')
    out=args.output.absolute();tools=args.tools.resolve(strict=True)
    assert out.parent==Path('/tmp/opencode') and not out.exists()
    out.mkdir(mode=0o700)
    for name in ('home','runtime','config','state','data','cache','files','Commons','etc'):(out/name).mkdir(mode=0o700)
    (out/'etc/fonts').mkdir()
    (out/'files/owned key λ #1.key').write_text('synthetic path fixture; never boot\n')
    if args.baseline:(out/'baseline').touch()
    shutil.copytree(ROOT/'plugins/omabit',out/'Plugin')
    if args.baseline:
        path=out/'Plugin/KeyBootPane.qml'
        path.write_text(path.read_text().replace('    parentWindow: root.Window.window\n','').replace('    popupType: Controls.Popup.Item\n',''))
    native=Path('/usr/share/omarchy/shell')
    shutil.copytree(native/'Ui',out/'Ui')
    for name in ('Style.qml','Util.qml','Border.qml','BorderGeometry.js'):shutil.copy2(native/'Commons'/name,out/'Commons'/name)
    (out/'Commons/qmldir').write_text('module qs.Commons\nsingleton Style 1.0 Style.qml\nsingleton Color 1.0 Color.qml\nsingleton Util 1.0 Util.qml\nsingleton Border 1.0 Border.qml\nShellIpc 1.0 ShellIpc.qml\n')
    (out/'Commons/ShellIpc.qml').write_text('import Quickshell.Io\nIpcHandler {}\n')
    (out/'Commons/I18n.qml').write_text('pragma Singleton\nimport QtQuick\nQtObject { property bool rightToLeft: false }\n')
    with (out/'Commons/qmldir').open('a') as f:f.write('singleton I18n 1.0 I18n.qml\n')
    (out/'Commons/Color.qml').write_text('pragma Singleton\nimport QtQuick\nQtObject { property color background:"#171a1e";property color foreground:"#e7e9eb";property color accent:"#8ab4f8";property color urgent:"#ed8994";property var shellValues:({});property var menu:({background:"#171a1e",text:"#e7e9eb",border:"#8ab4f8",scrim:"#88000000"});property var tooltip:menu;property var popups:menu }\n')
    shutil.copy2(ROOT/'scripts/test-key-picker-wayland.qml',out/'shell.qml')
    shutil.copy2(__file__,out/'driver.py')
    (out/'sway.conf').write_text('xwayland disable\nswaybg_command -\noutput HEADLESS-1 mode 1280x960\nfont monospace 12\nfocus_follows_mouse no\nseat seat0 fallback true\nfor_window [title="Select Urbit boot key"] floating enable, resize set 550 px 500 px, move position 25 px 70 px\n')
    (out/'etc/passwd').write_text(f'lab:x:{os.getuid()}:{os.getgid()}:Lab:/lab/home:/bin/sh\n')
    (out/'etc/group').write_text(f'lab:x:{os.getgid()}:\n')
    (out/'etc/nsswitch.conf').write_text('passwd: files\ngroup: files\n')
    (out/'sources.json').write_text(json.dumps({str(p):hashlib.sha256(p.read_bytes()).hexdigest() for p in [ROOT/'plugins/omabit/KeyBootPane.qml',native/'Ui/KeyboardPanel.qml']},indent=2)+'\n')
    command=['bwrap','--unshare-all','--die-with-parent','--clearenv','--ro-bind','/usr','/usr','--symlink','usr/bin','/bin','--symlink','usr/lib','/lib','--symlink','usr/lib','/lib64','--proc','/proc','--dev','/dev','--tmpfs','/tmp','--ro-bind',str(out/'etc'),'/etc','--ro-bind','/etc/fonts','/etc/fonts','--ro-bind',str(tools),'/tools','--bind',str(out),'/lab','--chdir','/lab']
    env={'PATH':'/usr/bin:/bin','LANG':'C.UTF-8','HOME':'/lab/home','XDG_RUNTIME_DIR':'/lab/runtime','XDG_CONFIG_HOME':'/lab/config','XDG_DATA_HOME':'/lab/data','XDG_STATE_HOME':'/lab/state','XDG_CACHE_HOME':'/lab/cache','QT_QPA_PLATFORM':'wayland','QT_QUICK_BACKEND':'software','QT_QUICK_CONTROLS_STYLE':'Basic','QT_QPA_PLATFORMTHEME':'','QML_DISABLE_DISK_CACHE':'1'}
    for key,value in env.items():command+=['--setenv',key,value]
    result=subprocess.run(command+['--','python3','-B','/lab/driver.py','--inside'],text=True,capture_output=True,timeout=60)
    (out/'driver.log').write_text(result.stdout+result.stderr)
    print(result.stdout+result.stderr);return result.returncode


if __name__=='__main__':sys.exit(main())
