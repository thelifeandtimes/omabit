#!/usr/bin/env python3
"""Inspect an explicitly selected retained authority without granting activation."""
import argparse,json
from pathlib import Path
import sys
sys.dont_write_bytecode=True
sys.path.insert(0,str(Path(__file__).resolve().parents[1]/'native/collections'))
import continuation_inspection as authority_inspection

def main():
    parser=argparse.ArgumentParser(description=__doc__,epilog='Evidence inspection never proves transport cleanup or permits restarting plugins. Original bound files must remain at their recorded locations; copied archives and missing runtime directories are refused.')
    parser.add_argument('--authority',required=True,type=Path,help='absolute existing .omabit-collection-authority directory')
    parser.add_argument('--coordinator',required=True,type=Path,help='absolute canonical collection journal directory')
    parser.add_argument('--details',action='store_true',help='include retained descriptor, reservation history and file hashes')
    args=parser.parse_args()
    try:
        result=authority_inspection.inspect(args.authority,args.coordinator)
        if not args.details:result={k:result[k] for k in ('format','mode','authority','context','journal','activations','missing_files','activation_authorized','transport_cleanup_verified')}
    except (OSError,ValueError,KeyError,TypeError) as error:
        print(json.dumps({'error':str(error),'activation_authorized':False}),file=sys.stderr);return 1
    print(json.dumps(result,indent=2));return 0

if __name__=='__main__':raise SystemExit(main())
