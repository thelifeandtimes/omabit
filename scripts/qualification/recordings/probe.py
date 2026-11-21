#!/usr/bin/env python3
"""Inert, private comparison of pipe/PTY/script/tmux recording lifecycles."""
import argparse
import ctypes
import errno
import fcntl
import hashlib
import json
import os
from pathlib import Path
import pty
import selectors
import signal
import socket
import subprocess
import sys
import termios
import time

def wait_file(path, timeout=8):
    end = time.monotonic() + timeout
    while not path.is_file():
        if time.monotonic() > end:
            raise RuntimeError('fixed fixture readiness deadline')
        time.sleep(.01)

def inspect(root):
    raw = (root/'raw.bin').read_bytes()
    result = {'raw_bytes': len(raw), 'raw_sha256': hashlib.sha256(raw).hexdigest(),
            'burst_350_lines': all(('trace-line-%04d'%i).encode() in raw for i in range(350)),
            'long_line_12288': b'x'*12288 in raw,
            'original_diagnostic_atom': b'0wdiagnostic.atom' in raw,
            'original_fake_credentials': b'fake-fixture-token' in raw and b'fake-fixture-cookie' in raw,
            'terminal_redraw_bytes': b'\x1b[2K\rspinner-3' in raw,
            'stderr_marker': b'STDERR-OWNED-MARKER' in raw,
            'final_signal_marker': b'FINAL-OWNED-SIGNAL' in raw,
            'signal': json.loads((root/'signal.json').read_text()) if (root/'signal.json').exists() else None}
    if root.name!='script':
        events=[json.loads(line) for line in (root/'timing.ndjson').read_text().splitlines()]
        offset=0;elapsed=0
        for event in events:
            if event['offset']!=offset or event['length']<1 or event['elapsed_ns']<elapsed:raise RuntimeError('capture timing/offset inconsistency')
            offset+=event['length'];elapsed=event['elapsed_ns']
        if offset!=len(raw):raise RuntimeError('raw bytes and timing coverage differ')
        result.update(timing_covers_all_raw_bytes=True,timing_events=len(events),timing_basis='recorder receive clock; cross-stream order is observation order')
    else:
        events=[line.split() for line in (root/'script-timing.log').read_text().splitlines()]
        if not events or any(len(row)!=2 or float(row[0])<0 or int(row[1])<0 for row in events):raise RuntimeError('invalid native script timing')
        result.update(timing_events=len(events),script_timed_bytes=sum(int(row[1]) for row in events),timing_basis='script classic timing; raw file also has untimed wrapper header/footer')
    return result

def adopted_exit(pid):
    end=time.monotonic()+5
    while time.monotonic()<end:
        try:
            observed, status=os.waitpid(pid,os.WNOHANG)
            if observed==pid:return os.waitstatus_to_exitcode(status)
        except ChildProcessError:
            pass
        time.sleep(.01)
    raise RuntimeError('owned adopted-child exit unconfirmed')

def direct(kind):
    root = Path('/lab')/kind; root.mkdir()
    command = ['/usr/bin/python3', '/artifacts/workload.py', str(root)]
    master = None
    if kind == 'pty':
        master, slave = pty.openpty()
        process = subprocess.Popen(command, stdin=slave, stdout=slave, stderr=slave, start_new_session=True, preexec_fn=lambda: fcntl.ioctl(0, termios.TIOCSCTTY, 0))
        os.close(slave)
        streams = [(master, 'pty-combined')]
    elif kind == 'script':
        command = ['/usr/bin/script', '--quiet', '--return', '--flush', '--echo', 'never', '--log-out', str(root/'raw.bin'), '--log-timing', str(root/'script-timing.log'), '--'] + command
        process = subprocess.Popen(command, stdin=subprocess.PIPE, stdout=subprocess.DEVNULL, stderr=subprocess.PIPE, start_new_session=True)
        streams = [(process.stderr, 'script-diagnostic')]
    else:
        process = subprocess.Popen(command, stdin=subprocess.DEVNULL, stdout=subprocess.PIPE, stderr=subprocess.PIPE, start_new_session=True)
        streams = [(process.stdout, 'stdout'), (process.stderr, 'stderr')]
    selector = selectors.DefaultSelector()
    for stream, name in streams:
        selector.register(stream, selectors.EVENT_READ, name)
    origin = time.monotonic_ns(); offset = 0; signalled = False
    (root/'capture-start.json').write_text(json.dumps({'unix_ns': time.time_ns(), 'monotonic_ns': origin, 'basis': 'recorder receive clock, not process write time'}))
    end = time.monotonic() + 15
    output = root/('wrapper-diagnostic.bin' if kind == 'script' else 'raw.bin')
    with output.open('xb', buffering=0) as raw, (root/'timing.ndjson').open('x') as timing:
        try:
            while selector.get_map() or process.poll() is None:
                if time.monotonic() > end:
                    raise RuntimeError('fixed recording fixture deadline')
                if (root/'ready').exists() and not signalled:
                    owned_status=Path('/proc/'+str(process.pid)+'/status').read_text()
                    (root/'stop-dispatched.json').write_text(json.dumps({'target_pid':process.pid,'workload_pid':int((root/'pid').read_text()),'signals':[line for line in owned_status.splitlines() if line.startswith(('SigBlk:','SigIgn:','SigCgt:'))]}))
                    process.send_signal(signal.SIGTERM); signalled = True
                for key, _ in selector.select(.02):
                    fd = key.fileobj if isinstance(key.fileobj, int) else key.fileobj.fileno()
                    try: data = os.read(fd, 65536)
                    except OSError as error:
                        if master == fd and error.errno == errno.EIO: data = b''
                        else: raise
                    if not data:
                        selector.unregister(key.fileobj); continue
                    raw.write(data)
                    timing.write(json.dumps({'offset': offset, 'length': len(data), 'elapsed_ns': time.monotonic_ns()-origin, 'stream': key.data})+'\n')
                    timing.flush(); offset += len(data)
            code = process.wait(timeout=2)
        finally:
            selector.close()
            if process.poll() is None:
                # Only the freshly created private process group, never a host
                # process inventory or an existing terminal/ship.
                os.killpg(process.pid, signal.SIGKILL); process.wait()
            if master is not None: os.close(master)
            for stream in (process.stdin, process.stdout, process.stderr):
                if stream: stream.close()
        os.fsync(raw.fileno()); os.fsync(timing.fileno())
    result = inspect(root)
    result.update(wrapper_exit=code, stop_target='owned workload' if kind != 'script' else 'owned script wrapper', separate_stream_identity=kind == 'pipe')
    result['observed_workload_exit']=adopted_exit(int((root/'pid').read_text())) if kind=='script' else code
    return result

def tmux_case():
    root = Path('/lab/tmux'); root.mkdir()
    config = root/'tmux.conf'; config.write_text('set -g remain-on-exit on\nset -g exit-empty off\n')
    prefix = ['/usr/bin/tmux', '-S', str(root/'private.sock'), '-f', str(config)]
    def run(args):
        return subprocess.run(prefix+args, check=True, capture_output=True, text=True, timeout=5).stdout.strip()
    try:
        run(['new-session', '-d', '-s', 'capture', '/usr/bin/python3 /artifacts/workload.py /lab/tmux'])
        run(['pipe-pane', '-O', '-t', 'capture:0.0', '/usr/bin/python3 /artifacts/pipe_capture.py'])
        wait_file(root/'ready')
        server_pid = int(run(['display-message', '-p', '-t', 'capture:0.0', '#{pid}']))
        workload_pid = int((root/'pid').read_text())
        if server_pid < 2 or workload_pid < 2: raise RuntimeError('private server identity unavailable')
        # This models a container/supervisor sending TERM to its tmux server,
        # not a custom wrapper selecting and signalling an inner pane process.
        os.kill(server_pid, signal.SIGTERM)
        wait_file(root/'signal.json')
        wait_file(root/'capture-closed')
        result = inspect(root)
        result.update(wrapper_exit=0, wrapper_exit_basis='new-session client exited after submission; not pane completion', stop_target='own private tmux server', separate_stream_identity=False)
        result['observed_workload_exit']=adopted_exit(workload_pid)
        return result
    finally:
        subprocess.run(prefix+['kill-server'], capture_output=True, timeout=5)

def inside():
    os.umask(0o077); os.nice(15)
    guard = json.loads(Path('/guard.json').read_text())
    for key in ('user', 'pid', 'mnt', 'net'):
        if os.readlink('/proc/self/ns/'+key) == guard['namespaces'][key]: raise RuntimeError('host namespace refused')
    if any(Path(p).exists() for p in ('/home', '/run/user', '/var/run/docker.sock')) or [name for _,name in socket.if_nameindex()] != ['lo']:
        raise RuntimeError('host access refused')
    wait_file(Path('/lab/admitted'))
    if Path('/lab/admitted').read_text()!=guard['unit']:raise RuntimeError('scope admission absent')
    # Observe exact owned grandchildren after wrappers exit. This applies only
    # within the new PID namespace; no host process is enumerated or adopted.
    if ctypes.CDLL(None,use_errno=True).prctl(36,1,0,0,0)!=0:raise RuntimeError('private subreaper unavailable')
    results = {'scope':'inert native workloads only; no Docker/Urbit/installed lifecycle qualification', 'probe_complete':False, 'versions':{}, 'cases':{}}
    for name, args in (('script',['script','--version']),('tmux',['tmux','-V'])):
        results['versions'][name]=subprocess.check_output(args,text=True).strip()
    for kind in ('pipe', 'pty', 'script', 'tmux'):
        try:results['cases'][kind] = tmux_case() if kind=='tmux' else direct(kind)
        except Exception as error:results['cases'][kind]={'failed':type(error).__name__+': '+str(error)}
        Path('/lab/result.json').write_text(json.dumps(results,indent=2)+'\n')
    results['probe_complete']=True
    Path('/lab/result.json').write_text(json.dumps(results,indent=2)+'\n')
    return 0

def host(args):
    import shutil
    import uuid
    sys.path.insert(0,str(Path(__file__).resolve().parents[1]/'app-development'))
    from storage import admit, roots, measure
    output=args.output.absolute()
    if output.parent!=Path('/tmp') or not output.name.startswith('omabit-recording-probe-') or output.exists() or output.is_symlink():raise ValueError('fresh private /tmp recording output required')
    paths=roots()
    for p in (Path.home()/'.cache/go-build',Path.home()/'go/pkg/mod',Path(__file__).resolve().parents[3]/'build'):
        if p.exists() and not any(p==r or p.is_relative_to(r) for r in paths):paths.append(p)
    admission=admit(16*1024**2,measurements=[measure(p) for p in paths],budget_bytes=args.storage_budget_gib*1024**3)
    os.umask(0o077);output.mkdir();lab=output/'lab';lab.mkdir();(lab/'home').mkdir();art=output/'artifacts';art.mkdir()
    for name in ('probe.py','workload.py','pipe_capture.py'):shutil.copyfile(Path(__file__).with_name(name),art/name)
    guard={'unit':'omabit-recording-probe-'+uuid.uuid4().hex,'namespaces':{k:os.readlink('/proc/self/ns/'+k) for k in ('user','pid','mnt','net')}}
    (output/'guard.json').write_text(json.dumps(guard));(output/'storage-admission.json').write_text(json.dumps(admission,indent=2))
    (output/'inputs.json').write_text(json.dumps({p.name:hashlib.sha256(p.read_bytes()).hexdigest() for p in art.iterdir()},indent=2))
    command=['systemd-run','--user','--scope','--unit='+guard['unit'],'-p','MemoryMax=268435456','-p','MemorySwapMax=0','-p','CPUQuota=100%','-p','TasksMax=64','--','bwrap','--unshare-all','--die-with-parent','--clearenv','--ro-bind','/usr','/usr','--symlink','usr/bin','/bin','--symlink','usr/lib','/lib','--symlink','usr/lib','/lib64','--proc','/proc','--dev','/dev','--tmpfs','/tmp','--dir','/run','--ro-bind',str(art),'/artifacts','--ro-bind',str(output/'guard.json'),'/guard.json','--bind',str(lab),'/lab','--setenv','HOME','/lab/home','--setenv','PATH','/usr/bin:/bin','--setenv','LANG','C.UTF-8','--','python3','-B','/artifacts/probe.py','--inside']
    position=command.index('--ro-bind')
    command[position:position]=['--dir','/etc','--ro-bind','/etc/passwd','/etc/passwd','--ro-bind','/etc/group','/etc/group','--ro-bind','/etc/nsswitch.conf','/etc/nsswitch.conf']
    with (output/'console.log').open('wb') as log:
        child=subprocess.Popen(command,stdout=log,stderr=subprocess.STDOUT,start_new_session=True)
        try:
            end=time.monotonic()+10
            while time.monotonic()<end and child.poll() is None:
                state=subprocess.run(['systemctl','--user','show',guard['unit']+'.scope','--property=ActiveState,MemoryMax,MemorySwapMax,CPUQuotaPerSecUSec,TasksMax'],capture_output=True,text=True,timeout=3)
                values=dict(line.split('=',1) for line in state.stdout.splitlines() if '=' in line)
                if values.get('ActiveState')=='active':
                    if any(values.get(k)!=v for k,v in {'MemoryMax':'268435456','MemorySwapMax':'0','CPUQuotaPerSecUSec':'1s','TasksMax':'64'}.items()):raise RuntimeError('resource limits differ')
                    (output/'admission.json').write_text(json.dumps(values));(lab/'admitted').write_text(guard['unit']);break
                time.sleep(.05)
            else:raise RuntimeError('scope admission failed')
            code=child.wait(timeout=90)
        finally:
            if child.poll() is None:os.killpg(child.pid,signal.SIGTERM);child.wait(timeout=5)
    (output/'exit.json').write_text(json.dumps({'exit':code}));print(json.dumps({'output':str(output),'exit':code}));return code

if __name__=='__main__':
    parser=argparse.ArgumentParser(description=__doc__);parser.add_argument('--inside',action='store_true');parser.add_argument('--output',type=Path);parser.add_argument('--storage-budget-gib',type=int,choices=(40,48),default=40)
    args=parser.parse_args();raise SystemExit(inside() if args.inside else host(args))
