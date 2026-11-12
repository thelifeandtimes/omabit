"""Real managed ship/SSH/receiver check, only inside the offline release VM."""
import json
import os
from pathlib import Path
import select
import shutil
import socket
import socketserver
import subprocess
import threading


def exercise(run, tool, target, runner_id, host_key):
    assert os.geteuid() == 0
    assert 'omabit.disposable-guest=release-upgrade-v1' in Path('/proc/cmdline').read_text().split()
    inputs = Path('/opt/omabit-release-input')
    root = Path('/home/review/fleet-qualification')
    assert not root.exists()
    root.mkdir(mode=0o700)
    for name in ('home', 'config', 'data', 'state', 'cache', 'runtime', 'evidence'):
        (root/name).mkdir(mode=0o700)
    plugins = root/'home/.config/omarchy/plugins'
    plugins.mkdir(parents=True)
    shutil.copytree(inputs/'fleet-plugins', plugins, dirs_exist_ok=True)
    key = root/'management-key'
    shutil.copy2('/home/review/.config/omabit/setup/id_ed25519', key)
    (root/'known_hosts').write_text('[10.77.40.1]:22222 '+host_key+'\n')
    (root/'ssh_config').write_text(f'Host qualification-runner\n  HostName 10.77.40.1\n  Port 22222\n  User review\n  IdentityFile {key}\n  IdentityAgent none\n  IdentitiesOnly yes\n  UserKnownHostsFile {root}/known_hosts\n  GlobalKnownHostsFile /dev/null\n  StrictHostKeyChecking yes\n  BatchMode yes\n')
    (root/'input.json').write_text(json.dumps({'target':target,'runner_id':runner_id,'runner_tool':str(tool(target,'urbit-collector')),'omabit_tool':str(tool(target,'omabit'))})+'\n')
    run(['chown','-R','review:review',root])
    namespace='omabit-release-client'
    run(['ip','netns','add',namespace])
    servers=[];threads=[]
    counters={'wrong_identity_requests':0,'wrong_identity_credentials':0,'loopback_decoy_requests':0}
    class Proxy(socketserver.BaseRequestHandler):
        def handle(self):
            with socket.create_connection(('127.0.0.1',19180),timeout=5) as upstream:
                peers=[self.request,upstream]
                while not (root/'route-one-offline').exists() or self.server.server_address[1]!=20081:
                    ready,_,_=select.select(peers,[],[],.25)
                    for peer in ready:
                        data=peer.recv(65536)
                        if not data:return
                        (upstream if peer is self.request else self.request).sendall(data)
    class WrongShip(socketserver.BaseRequestHandler):
        def handle(self):
            request=self.request.recv(16384).lower()
            counters['wrong_identity_requests']+=1
            if b'cookie:' in request or request.startswith(b'post '):counters['wrong_identity_credentials']+=1
            self.request.sendall(b'HTTP/1.1 200 OK\r\nContent-Length: 4\r\nConnection: close\r\n\r\n~nec')
    class Server(socketserver.ThreadingTCPServer):
        allow_reuse_address=False
        daemon_threads=True
    try:
        run(['ip','link','add','ofleet-host','type','veth','peer','name','ofleet-client'])
        run(['ip','link','set','ofleet-client','netns',namespace])
        run(['ip','addr','add','10.77.40.1/30','dev','ofleet-host'])
        run(['ip','link','set','ofleet-host','up'])
        for args in (['link','set','lo','up'],['addr','add','10.77.40.2/30','dev','ofleet-client'],['link','set','ofleet-client','up']):run(['ip','-n',namespace,*args])
        for port,handler in ((20080,WrongShip),(20081,Proxy),(20082,Proxy)):
            server=Server(('10.77.40.1',port),handler);servers.append(server)
            thread=threading.Thread(target=server.serve_forever,daemon=True);thread.start();threads.append(thread)
        env=['HOME='+str(root/'home'),'PATH=/usr/bin:/bin','XDG_CONFIG_HOME='+str(root/'config'),'XDG_DATA_HOME='+str(root/'data'),'XDG_STATE_HOME='+str(root/'state'),'XDG_CACHE_HOME='+str(root/'cache'),'XDG_RUNTIME_DIR='+str(root/'runtime')]
        # The desktop has its own network namespace, HOME, registry and Secret
        # Service. It reaches the real runner exclusively over restricted SSH.
        command=['ip','netns','exec',namespace,'runuser','-u','review','--','env',*env,'dbus-run-session','--','python3','-B',inputs/'fleet_client.py',root]
        run(command,timeout=600)
        result=json.loads((root/'evidence/result.json').read_text())
        assert result['passed']
        assert counters['wrong_identity_requests']>0 and counters['wrong_identity_credentials']==0
        result['identity_decoy']=counters
        result['boundary']='real managed Docker fake and restricted SSH; desktop on separate offline veth namespace; real Theme/DMs helpers and private Secret Service; explicit LAN routes, no live Tailscale or provider'
        return result
    finally:
        for server in servers:server.shutdown();server.server_close()
        for thread in threads:thread.join(timeout=5)
        run(['ip','netns','delete',namespace],ok=False)
        run(['ip','link','delete','ofleet-host'],ok=False)
