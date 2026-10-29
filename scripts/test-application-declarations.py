#!/usr/bin/env python3
"""Check declared helper inventories using private copies and invalid offers."""
import argparse,json,os,shutil,subprocess,tempfile
from pathlib import Path

def main():
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument('sources',nargs='+',type=Path)
    args=parser.parse_args()
    for source in args.sources:
        declaration=json.loads((source/'application-connection.json').read_text())
        with tempfile.TemporaryDirectory(prefix='omabit-declaration-') as temporary:
            root=Path(temporary);plugin=root/'plugin';plugin.mkdir()
            for name in declaration['files']:
                relative=Path(name)
                assert not relative.is_absolute() and '..' not in relative.parts
                target=plugin/relative;target.parent.mkdir(parents=True,exist_ok=True)
                shutil.copyfile(source/relative,target)
            manifest=json.loads((plugin/'manifest.json').read_text())
            manifest['id']+='.auth-lab';manifest['name']+=' (Auth Lab)'
            (plugin/'manifest.json').write_text(json.dumps(manifest))
            (plugin/'application-profile.json').write_text('{"version":1,"profile":"auth-lab"}')
            env={'PATH':'/usr/bin:/bin','HOME':str(root/'home'),'XDG_STATE_HOME':str(root/'state'),'XDG_CONFIG_HOME':str(root/'config'),'XDG_RUNTIME_DIR':str(root/'runtime')}
            for path in env.values():
                if path.startswith(str(root)):Path(path).mkdir(mode=0o700)
            bootstrap="import runpy,sys;sys.path.insert(0,sys.argv[1]);entry=sys.argv[2];sys.argv=[entry]+sys.argv[3:];runpy.run_path(entry,run_name='__main__')"
            entry=plugin/declaration['entry']
            for action in ([],['--reconcile']):
                result=subprocess.run(['/usr/bin/python3','-I','-S','-B','-c',bootstrap,str(entry.parent),str(entry)]+action,input=b'{}',capture_output=True,env=env,timeout=30)
                assert result.returncode==1 and not result.stderr,(source,action,result)
                response=json.loads(result.stdout)
                assert response['status']=='rejected',response
            assert not list((root/'state').iterdir()),'invalid offer wrote account state'
            print('PASS: complete declared inventory and profiled imports for '+manifest['id'])

if __name__=='__main__':main()
