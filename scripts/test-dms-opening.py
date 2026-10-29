#!/usr/bin/env python3
"""Exercise the shipped DM opening callback with inert Qt items and no desktop."""
import argparse
from pathlib import Path
import shutil
import subprocess
import tempfile


def main():
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--source',type=Path,required=True)
    args=parser.parse_args()
    qml=shutil.which('qml6')
    if not qml:parser.error('Qt 6 qml6 is required')
    source=(args.source/'Dms.qml').read_text()
    callback=source[source.index('  function open(payloadJson) {'):source.index('\n  function close()')]
    template='''import QtQuick
Item {
  id: root
  property string mode: ""
  property bool opened: false
  property string status: ""
  property bool statusError: false
  property string ship: ""
  property string quickTarget: ""
  property var quickMatches: []
  property int quickCursor: -1
  property var calls: []
  function command(action) { calls.push(action) }
  function rebuildResults() {}
  function check(expected) { if(JSON.stringify(calls)!==JSON.stringify(expected))throw new Error("wrong opening commands");calls=[] }
  Item { id: quickRecipient; property string text: "" }
  Item { id: quickText; property string text: "" }
  Item { id: searchInput }
  Item { id: loginUrl }
@@CALLBACK@@
  Component.onCompleted: {
    try {
      root.open('{"mode":"quick"}');check(["apps","restore"])
      root.ship="~zod"
      root.open('{"mode":"quick"}');check(["apps","contacts"])
      root.ship=""
      root.open('{}');check(["apps","refresh"])
      console.log("PASS: shipped DMs opening callback restores an approved account without replacing an active session")
      Qt.callLater(Qt.quit)
    } catch(error) {console.error("FAIL: "+error.message);Qt.exit(1)}
  }
  Timer { interval:5000;running:true;onTriggered:Qt.exit(1) }
}
'''
    with tempfile.TemporaryDirectory(prefix='omabit-dms-opening-') as temporary:
        root=Path(temporary)
        env={'PATH':'/usr/bin:/bin','LANG':'C.UTF-8','QT_QPA_PLATFORM':'offscreen','QT_QUICK_BACKEND':'software','QML_DISABLE_DISK_CACHE':'1','QT_FORCE_STDERR_LOGGING':'1','QT_LOGGING_RULES':'*.warning=true;*.critical=true;qml.debug=true'}
        for name,leaf in (('HOME','home'),('XDG_RUNTIME_DIR','runtime'),('XDG_CONFIG_HOME','config'),('XDG_STATE_HOME','state'),('XDG_CACHE_HOME','cache'),('XDG_DATA_HOME','data')):
            path=root/leaf;path.mkdir(mode=0o700);env[name]=str(path)
        script=root/'main.qml';script.write_text(template.replace('@@CALLBACK@@',callback))
        result=subprocess.run([qml,str(script)],env=env,capture_output=True,text=True,timeout=15)
        output=result.stdout+result.stderr
        assert result.returncode==0 and 'PASS:' in output,(result.returncode,output)
        assert not any(value in output for value in ('Error:','is not defined','Cannot assign','Unable to assign')),output
        print(next(line for line in output.splitlines() if 'PASS:' in line))


if __name__=='__main__':main()
