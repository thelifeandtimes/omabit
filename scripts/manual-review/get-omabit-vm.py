#!/usr/bin/env python3
"""Pull the sealed review bundle over your SSH alias and run it locally."""
import argparse,json,os,re,shutil,subprocess,sys
from pathlib import Path
BUILD='20261026-onboarding-v1'
SOURCE='/home/thelifeandtimes/Documents/ChatGPT/omabit-private-docker-labs/manual-review-20261026-01/portable'
def main():
 p=argparse.ArgumentParser(description=__doc__)
 p.add_argument('--from',dest='host',default='sourcebox',help='Your normal SSH alias for sourcebox')
 p.add_argument('--source-path',default=SOURCE)
 p.add_argument('--to',type=Path,default=Path.home()/'.local/share/omabit-review'/BUILD)
 p.add_argument('--memory',type=int,default=8,choices=range(6,13))
 p.add_argument('--cpus',type=int,default=4,choices=range(2,9))
 p.add_argument('--no-start',action='store_true');p.add_argument('--no-open',action='store_true')
 a=p.parse_args()
 if os.geteuid()==0:raise RuntimeError('Run as your normal user, not sudo/root.')
 if not re.fullmatch(r'[A-Za-z0-9_][A-Za-z0-9_.@-]*',a.host):raise RuntimeError('Use a normal SSH alias, without shell arguments.')
 if not re.fullmatch(r'/[A-Za-z0-9_./-]+',a.source_path):raise RuntimeError('Source path must be an absolute path without shell metacharacters.')
 for n in ['ssh','rsync','python3']:
  if not shutil.which(n):raise RuntimeError('Missing '+n+'. Install its native Omarchy package and retry.')
 dest=a.to.expanduser().absolute()
 if dest.is_symlink():raise RuntimeError('Choose a real destination directory, not a symlink.')
 marker=dest/'.omabit-pull.json'
 identity={'build':BUILD,'host':a.host,'source':a.source_path}
 resumable=marker.is_file() and json.loads(marker.read_text())==identity
 if dest.exists() and any(dest.iterdir()) and not (dest/'bundle.json').is_file() and not resumable:raise RuntimeError('Destination is not empty and is not this Omabit review transfer; nothing overwritten.')
 if (dest/'bundle.json').is_file() and json.loads((dest/'bundle.json').read_text()).get('build')!=BUILD:raise RuntimeError('Existing bundle is a different build; choose another destination.')
 if (dest/'state/session.qcow2').exists():
  print('Existing session found. Preserving its base and all review progress; no re-download.')
  subprocess.run([str(dest/'omabit-vm'),'verify'],check=True)
 else:
  if not dest.exists():dest.mkdir(parents=True,mode=0o700)
  if shutil.disk_usage(dest).free<12*1024**3:raise RuntimeError('Keep at least 12 GiB free for the bundle and persistent review session.')
  remote=subprocess.check_output(['ssh','-o','ConnectTimeout=15',a.host,'cat '+a.source_path+'/bundle.json'],text=True)
  if json.loads(remote)['build']!=BUILD:raise RuntimeError('Unexpected source bundle version; nothing copied.')
  marker.write_text(json.dumps(identity)+'\n')
  subprocess.run(['rsync','-a','--protect-args','--partial','--checksum','--sparse','--info=progress2','--exclude=/state/','--exclude=/prepare-copy.log','--exclude=/prepare-serial.log','--exclude=/prepare-check.json',a.host+':'+a.source_path+'/',str(dest)+'/'],check=True)
  subprocess.run([str(dest/'omabit-vm'),'verify'],check=True)
  marker.unlink(missing_ok=True)
 print('Review directory: '+str(dest))
 subprocess.run([str(dest/'omabit-vm'),'doctor'],check=True)
 if not a.no_start:
  subprocess.run([str(dest/'omabit-vm'),'start','--memory',str(a.memory),'--cpus',str(a.cpus)],check=True)
  if not a.no_open and shutil.which('xdg-open') and (os.environ.get('DISPLAY') or os.environ.get('WAYLAND_DISPLAY')):
   # Opening early is safe; refresh after the short viewer initialization.
   subprocess.run(['xdg-open','http://127.0.0.1:6081/vnc.html?autoconnect=1&resize=scale'],check=False)
 print('Guest F3: onboarding walkthrough. Stop later with: '+str(dest/'omabit-vm')+' stop')
if __name__=='__main__':
 try:main()
 except (RuntimeError,OSError,subprocess.CalledProcessError,ValueError) as e:print('Omabit VM: '+str(e),file=sys.stderr);sys.exit(1)
