"""Promoted native cases from fixed_suite.py; inert until guarded execute()."""
import contextlib
import hashlib
import http.server
import json
import os
from pathlib import Path
import re
import threading
import time
import traceback
from types import SimpleNamespace

from fixtures import Fixtures, RUNNER
from guards import require

a = p = R = E = I = T = None
B = RUNNER
WR = '/usr/local/lib/urbit-runner/package-hooks/refresh-host-packages'
SRC = Path('/etc/apt/sources.list.d/hc.sources')
CACHE = Path('/var/lib/omabit')
HTTP = []
DENY = False
APT_UID = APT_GID = None


def save(name, data):
    a.write(E / name, json.dumps(data, indent=2) + '\n', 0o600)


def preserve(base, label):
    if base.exists():
        base.rename(T / label)


def receipt(base=Path('/')):
    return base / 'var/lib/omabit/hostcompat/receipts-v2.json'


def report(r):
    data = json.loads(r.stdout)
    assert data['qualification'] == 'unqualified' and data.get('engine') is None
    return data


def permissions(label, base=Path('/')):
    expected = {'var/lib/omabit': 0o755, 'var/lib/omabit/hostcompat': 0o755,
                'var/lib/omabit/hostcompat/receipts-v2.json': 0o644,
                'var/lib/omabit/hostcompat/evidence-v2': 0o700}
    out = {}
    for name, mode in expected.items():
        s = (base / name).stat()
        out[name] = {'uid': s.st_uid, 'gid': s.st_gid, 'mode': oct(s.st_mode & 0o777), 'inode': s.st_ino}
    for f in (base / 'var/lib/omabit/hostcompat/evidence-v2').iterdir():
        s = f.stat()
        assert s.st_uid == 0 and s.st_mode & 0o777 == 0o600, (f, s)
    save(label + '.permissions.json', out)
    for name, mode in expected.items():
        assert out[name]['uid'] == 0 and int(out[name]['mode'], 8) == mode, (name, out[name])
    return out


def observe(stop, out):
    seen = {}
    while not stop.wait(0.008):
        for d in Path('/proc').iterdir():
            if not d.name.isdigit():
                continue
            try:
                exe = os.readlink(d / 'exe')
                if exe not in ['/usr/bin/apt-get', '/usr/lib/apt/methods/http', '/usr/lib/apt/methods/gpgv', '/usr/bin/gpgv']:
                    continue
                seen[d.name] = {'pid': d.name, 'exe': exe,
                                'cmdline': (d / 'cmdline').read_bytes().replace(b'\0', b' ').decode(errors='replace'),
                                'status': (d / 'status').read_text()}
            except OSError:
                pass
        for stage in Path('/var/lib/omabit/hostcompat').glob('.fetch-*'):
            for f in [stage, stage / 'apt', stage / 'apt/apt.conf', stage / 'apt/archives', stage / 'apt/lists', stage / 'apt-evidence']:
                try:
                    st = f.stat()
                    item = {'path': str(f), 'uid': st.st_uid, 'gid': st.st_gid, 'mode': oct(st.st_mode & 0o777)}
                    if f.name == 'apt.conf':
                        item['contents'] = f.read_text()
                    seen[str(f)] = item
                except OSError:
                    pass
    out.extend(seen.values())


def verify(label, fetch=False, expected=0, arch=False, env=None):
    args = ['chroot', str(p.A), '/usr/local/bin/urbit-runner'] if arch else [str(B)]
    args += ['verify-host-packages', '--acknowledge-local-verification'] + (['--fetch-missing'] if fetch else [])
    args = ['strace', '-f', '-tt', '-s', '2048', '-e', 'trace=%process,%creds', '-o', E / (label + '.strace')] + args
    seen, stop = [], threading.Event()
    thread = threading.Thread(target=observe, args=(stop, seen), daemon=True)
    if fetch and not arch:
        thread.start()
    old = os.umask(0o077)
    try:
        r = a.run(args, label, check=False, env=env)
    finally:
        os.umask(old)
        stop.set()
        if thread.is_alive():
            thread.join()
        if fetch and not arch:
            save(label + '.sandbox.json', seen)
    data = report(r)
    save(label + '.summary.json', {'exit': r.returncode, 'provenance': data['provenance'], 'packages': data['packages']})
    f = receipt(p.A if arch else Path('/'))
    if f.exists():
        a.write(E / (label + '.receipt.json'), f.read_bytes(), 0o600)
    assert r.returncode == expected, (label, r.returncode, expected)
    assert (data['provenance'] == 'verified') == (expected == 0), (label, data['provenance'])
    if fetch:
        base = p.A if arch else Path('/')
        assert not list((base / 'var/lib/omabit/hostcompat').glob('.fetch-*'))
    return data


def reader(label, expected='verified', arch=False, env=None):
    args = ['chroot', '--userspec=1000:1000', str(p.A), '/usr/local/bin/urbit-runner'] if arch else ['setpriv', '--reuid=1000', '--regid=1000', '--clear-groups', str(B)]
    r = a.run(['strace', '-f', '-tt', '-s', '1024', '-e', 'trace=openat,execve,%creds', '-o', E / (label + '.reader.strace')] + args + ['host-check'], label, env=env)
    data = report(r)
    assert data['provenance'] == expected, (label, data['provenance'])
    return data


class Handler(http.server.SimpleHTTPRequestHandler):
    def __init__(self, *args, **kw):
        super().__init__(*args, directory=str(R / 'repo'), **kw)

    def do_GET(self):
        HTTP.append({'time': time.time(), 'path': self.path, 'denied': DENY and '/engine/pool/' in self.path})
        time.sleep(0.04)
        if DENY and '/engine/pool/' in self.path:
            self.send_error(403, 'disposable denied download')
            return
        super().do_GET()

    def log_message(self, *args):
        pass


@contextlib.contextmanager
def server(arch=False):
    s = p.server() if arch else http.server.ThreadingHTTPServer(('127.0.0.1', 80), Handler)
    if not arch:
        threading.Thread(target=s.serve_forever, daemon=True).start()
    try:
        yield
    finally:
        s.shutdown()
        s.server_close()
        save('http-requests.json', HTTP)


def check_sandbox(label):
    seen = json.loads((E / (label + '.sandbox.json')).read_text())
    processes = [x for x in seen if x.get('exe') == '/usr/bin/apt-get']
    assert any(' update' in x['cmdline'] for x in processes), processes
    assert any(' download ' in x['cmdline'] for x in processes), processes
    for proc in processes:
        fields = {line.split(':', 1)[0]: line.split(':', 1)[1].strip() for line in proc['status'].splitlines() if ':' in line}
        assert fields['Uid'].split() == [str(APT_UID)] * 4, proc
        assert fields['Gid'].split() == [str(APT_GID)] * 4, proc
        assert not fields['Groups'], proc
    config = [x for x in seen if 'contents' in x]
    assert config and all('APT::Sandbox::User "_apt";' in x['contents'] for x in config)
    assert any(x.get('path', '').endswith('/apt/archives') and x['uid'] == APT_UID and x['mode'] == '0o700' for x in seen)
    save(label + '.sandbox-assertions.json', {'uid': APT_UID, 'gid': APT_GID, 'supplementary_groups': [], 'apt_processes': processes})


def baseline():
    a.clean_os_cache()
    verify('a01-empty-offline', expected=1)
    a.write('/etc/apt/apt.conf.d/99-hc-fetch-canary', f'APT::Update::Post-Invoke {{ "touch {T}/GLOBAL-HOOK-RAN"; }};\n')
    before = {'lists': a.digest_tree('/var/lib/apt/lists'), 'status': hashlib.sha256(Path('/var/lib/dpkg/status').read_bytes()).hexdigest()}
    try:
        with server():
            verify('a02-nonroot-fetch', fetch=True)
    finally:
        Path('/etc/apt/apt.conf.d/99-hc-fetch-canary').unlink()
    after = {'lists': a.digest_tree('/var/lib/apt/lists'), 'status': hashlib.sha256(Path('/var/lib/dpkg/status').read_bytes()).hexdigest()}
    save('a02-system-state.json', {'before': before, 'after': after})
    assert before == after and not (T / 'GLOBAL-HOOK-RAN').exists()
    check_sandbox('a02-nonroot-fetch')
    permissions('a03-first-root-refresh-077'); reader('a03-owner')
    a.clean_os_cache(); verify('a04-retained-offline'); reader('a04-owner')
    x = Path('/usr/bin/dockerd'); old = x.read_bytes(); mtime = x.stat().st_mtime_ns
    try:
        x.write_bytes(old.replace(b'exit 97', b'exit 96')); os.utime(x, ns=(x.stat().st_atime_ns, mtime))
        reader('a05-samesize-restored-mtime-tamper', 'unverified'); verify('a05-refresh-tamper', expected=1)
    finally:
        x.write_bytes(old)
    verify('a06-tamper-recover')
    src = SRC.read_text()
    try:
        SRC.write_text(src.replace('hc-engine.gpg', 'hc-support.gpg'))
        reader('a07-wrong-source-key', 'unverified'); verify('a07-wrong-source-key', expected=1)
    finally:
        SRC.write_text(src)
    verify('a08-source-recover')
    try:
        SRC.write_text(src.replace('hc-engine.gpg', 'hc-engine.asc'))
        verify('a09-armor'); reader('a09-armor-owner')
    finally:
        SRC.write_text(src)
    verify('a10-original-source')


def hook_assets():
    a.write(WR, a.hook_assets['refresh-host-packages'], 0o755)
    a.write('/etc/apt/apt.conf.d/00-omabit-host-packages', a.hook_assets['00-omabit-host-packages'])
    observer = T / 'cleanup-observer.py'
    a.write(observer, '''import pathlib,json
r=pathlib.Path('/var/lib/omabit/hostcompat/receipts-v2.json')
out={'receipt':json.loads(r.read_text()),'archives':[str(x) for x in pathlib.Path('/var/cache/apt/archives').glob('*.deb')],'mode':oct(r.stat().st_mode&511),'cache_mode':oct((r.parent/'evidence-v2').stat().st_mode&511),'uid':r.stat().st_uid}
label=pathlib.Path(''' + repr(str(T / 'observer-label')) + ''').read_text().strip()
(pathlib.Path(''' + repr(str(E)) + ''')/(label+'.before-cleanup.json')).write_text(json.dumps(out,indent=2))
for x in pathlib.Path('/var/cache/apt/archives').glob('hc-*.deb'):x.unlink()
''')
    a.write('/etc/apt/apt.conf.d/99-hc-cleanup', f'DPkg::Post-Invoke {{ "/usr/bin/python3 {observer}"; }};\nAPT::Keep-Downloaded-Packages "false";\n')


def apt_transaction(label, first=False):
    version = a.run(['dpkg-query', '-W', '-f=${Version}', 'hc-engine']).stdout.decode().strip()
    rev = int(version.rsplit('-', 1)[1]) + 1
    a.make_package('hc-engine', rev); a.publish('engine'); a.publish('support')
    if first:
        preserve(CACHE, label + '-prior-proof')
    a.write(T / 'observer-label', label)
    with server():
        a.run(['apt-get', 'update'], label + '-update')
        args = ['apt-get', '-y'] + (['--reinstall'] if first else []) + ['install', f'hc-engine=5:29.7.2-{rev}'] + (['hc-support=5:29.7.2-1'] if first else [])
        old = os.umask(0o077)
        try:
            a.run(['strace', '-f', '-tt', '-s', '2048', '-e', 'trace=%process,%creds,umask,fchmod', '-o', E / (label + '.transaction.strace')] + args, label)
        finally:
            os.umask(old)
    obs = json.loads((E / (label + '.before-cleanup.json')).read_text())
    assert obs['archives'] and obs['mode'] == '0o644' and obs['cache_mode'] == '0o700' and obs['uid'] == 0, obs
    assert len(obs['receipt']['packages']) == 8 and all(x['authorized'] for x in obs['receipt']['packages'])
    assert all(x['version'] == (f'5:29.7.2-{rev}' if x['name'] == 'hc-engine' else '5:29.7.2-1') for x in obs['receipt']['packages'])
    assert not list(Path('/var/cache/apt/archives').glob('*.deb'))
    permissions(label); reader(label + '-owner-after-cleanup')
    a.clean_os_cache(); verify(label + '-offline'); reader(label + '-owner-offline')


def apt_hooks():
    hook_assets(); a.run(['apt-config', 'dump'], 'a11-effective-config')
    apt_transaction('a12-first-use-native-hook', True)
    inode = receipt().stat().st_ino
    apt_transaction('a13-replacement-native-hook')
    assert receipt().stat().st_ino != inode
    for f in [CACHE, CACHE / 'hostcompat']:
        f.chmod(0o700)
    receipt().chmod(0o600)
    reader('a14-legacy-private-layout', 'unverified'); a.run([WR], 'a14-hook-repair')
    permissions('a14-recovered-public-layout'); reader('a14-owner-recovered')
    tree = T / 'unrelated'
    a.write(tree / 'DEBIAN/control', 'Package: hc-unrelated\nVersion: 1.0-3\nArchitecture: all\nMaintainer: Fixture <fixture@invalid.test>\nDescription: inert dpkg metadata change\n')
    a.write(tree / 'usr/share/hc-unrelated/marker', 'new fixture\n')
    a.run(['dpkg-deb', '--build', tree, T / 'hc-unrelated.deb'], 'a15-unrelated-build')
    a.run(['dpkg', '-i', T / 'hc-unrelated.deb'], 'a15-unrelated-install'); reader('a15-unrelated-owner')


def negative_fetch():
    global DENY
    a.clean_os_cache(); preserve(CACHE, 'before-negative-fetch')
    key = Path('/etc/apt/keyrings/hc-engine.gpg')
    try:
        key.chmod(0o600)
        with server():
            verify('a16-rootonly-key-fetch', True, 1)
        seen = json.loads((E / 'a16-rootonly-key-fetch.sandbox.json').read_text())
        uid_line = 'Uid:\t' + '\t'.join([str(APT_UID)] * 4)
        assert any(x.get('exe') == '/usr/bin/apt-get' and uid_line in x['status'] for x in seen)
    finally:
        key.chmod(0o644)
    preserve(CACHE, 'rootonly-key-failure')
    passwd = Path('/etc/passwd'); old = passwd.read_bytes()
    try:
        passwd.write_bytes(b'\n'.join(line for line in old.split(b'\n') if not line.startswith(b'_apt:')))
        with server():
            verify('a17-no-apt-account', True, 1)
    finally:
        passwd.write_bytes(old)
    assert 'execve("/usr/bin/apt-get"' not in (E / 'a17-no-apt-account.strace').read_text()
    preserve(CACHE, 'no-account-failure')
    try:
        DENY = True
        with server():
            verify('a18-native-download-http403', True, 1)
    finally:
        DENY = False
    assert any(x['denied'] for x in HTTP)
    reader('a18-denied-owner', 'unverified')
    with server():
        verify('a19-fetch-recovery', True)
    check_sandbox('a19-fetch-recovery'); permissions('a19-recovered'); reader('a19-owner')


def authority():
    hidden = SRC.with_name('.hc.sources'); SRC.rename(hidden)
    try:
        a.run(['apt-get', 'indextargets', '--format', '$(SITE)'], 'a20-native-hidden-targets')
        assert not (E / 'a20-native-hidden-targets.stdout').read_text().strip()
        reader('a20-hidden-source-owner', 'unverified'); verify('a20-hidden-source-refresh', expected=1)
    finally:
        hidden.rename(SRC)
    verify('a21-source-visible'); reader('a21-source-visible-owner')
    malicious = SRC.with_name('.ignored-insecure.sources')
    a.write(malicious, 'Types: deb\nURIs: http://invalid.test\nSuites: stable\nComponents: main\nTrusted: yes\n')
    try:
        reader('a22-hidden-insecure-ignored'); verify('a22-hidden-insecure-ignored')
    finally:
        malicious.unlink()
    override = Path('/etc/apt/apt.conf.d/98-hc-selection')
    empty = T / 'empty-sources'; empty.mkdir(exist_ok=True)
    for label, text in [('a23-dir-override', f'Dir::Etc::sourcelist "{T}/missing.list"; Dir::Etc::sourceparts "{empty}";\n'), ('a24-binary-override', f'Binary::apt-get {{ Dir::Etc::sourceparts "{empty}"; }};\n')]:
        a.write(override, text)
        try:
            a.run(['apt-config', 'dump'], label + '-native-config')
            reader(label + '-owner', 'unverified'); verify(label + '-refresh', expected=1)
        finally:
            override.unlink()
        verify(label + '-recovery')
    conf = T / 'invocation.conf'; a.write(conf, 'Acquire::Retries "0";\n')
    env = {'APT_CONFIG': str(conf)}
    reader('a25-aptconfig-owner', 'unverified', env=env); verify('a25-aptconfig-refresh', expected=1, env=env)
    verify('a26-authority-recovery')


def multisigner():
    home = T / 'multi-keys'; home.mkdir(mode=0o700)
    for key in ('engine', 'support'):
        secret = a.run(['gpg', '--homedir', R / ('key-' + key), '--batch', '--export-secret-keys']).stdout
        a.run(['gpg', '--homedir', home, '--batch', '--import'], input=secret)
    a.run(['gpg', '--homedir', home, '--batch', '--pinentry-mode', 'loopback', '--passphrase', '', '--quick-generate-key', 'third synthetic signer <third@invalid.test>', 'rsa2048', 'sign', '0'], 'a27-third-key')
    engine, support = a.fingerprint('engine'), a.fingerprint('support')
    lines = a.run(['gpg', '--homedir', home, '--with-colons', '--list-keys']).stdout.decode().splitlines()
    third = next(x.split(':')[9] for x in lines if x.startswith('fpr:') and x.split(':')[9] not in (engine, support))
    release = R / 'repo/engine/dists/stable/Release'; inrelease = release.parent / 'InRelease'; old = inrelease.read_bytes()
    key = Path('/etc/apt/keyrings/hc-engine.gpg'); oldkey = key.read_bytes()
    base = ['gpg', '--homedir', home, '--batch', '--yes', '--digest-algo', 'SHA256']
    try:
        a.run(base + ['--local-user', engine, '--local-user', support, '--local-user', third, '--clearsign', '--output', inrelease, release], 'a27-multi-sign')
        a.write(I / 'three-signer.InRelease', inrelease.read_bytes())
        g = a.run(['gpgv', '--keyring', key, '--status-fd', '1', inrelease], 'a27-native-gpgv', check=False)
        assert g.returncode == 2 and g.stdout.count(b'VALIDSIG') == 1 and g.stdout.count(b'NO_PUBKEY') == 2
        preserve(CACHE, 'before-multisigner'); a.clean_os_cache()
        with server():
            verify('a28-three-signers-one-selected-fetch', True)
        check_sandbox('a28-three-signers-one-selected-fetch'); reader('a28-owner')
        verify('a29-multisigner-retained-offline')
        with server():
            a.run(['apt-get', 'update'], 'a30-native-multi-update')
            a.run(['apt-get', '-y', '--download-only', '--reinstall', 'install', 'hc-engine', 'hc-support'], 'a30-native-cache')
        cached = Path('/var/lib/apt/lists/127.0.0.1_engine_dists_stable_InRelease')
        preserve(CACHE, 'before-invalid-only')
        a.run(base + ['--local-user', support, '--clearsign', '--output', inrelease, release], 'a31-wrong-key-sign')
        a.write(I / 'wrong-key.InRelease', inrelease.read_bytes()); cached.write_bytes(inrelease.read_bytes())
        verify('a31-invalid-only-signature', expected=1); reader('a31-owner', 'unverified')
        a.run(base + ['--local-user', engine, '--local-user', third, '--clearsign', '--output', inrelease, release], 'a32-two-selected-sign')
        rev = (R / 'key-engine/openpgp-revocs.d' / f'{engine}.rev').read_text()
        armor = rev[rev.index(':-----BEGIN PGP PUBLIC KEY BLOCK-----') + 1:]
        a.run(['gpg', '--homedir', home, '--batch', '--import'], 'a32-import-revocation', input=armor.encode())
        a.write(key, a.run(['gpg', '--homedir', home, '--export', engine, third]).stdout)
        a.write(I / 'revoked-plus-good.InRelease', inrelease.read_bytes()); a.write(I / 'revoked-public.gpg', key.read_bytes())
        cached.write_bytes(inrelease.read_bytes()); preserve(CACHE, 'before-revoked-selected')
        g = a.run(['gpgv', '--keyring', key, '--status-fd', '1', inrelease], 'a32-native-gpgv', check=False)
        assert b'REVKEYSIG' in g.stdout and g.stdout.count(b'VALIDSIG') == 2
        verify('a32-revoked-selected-plus-good', expected=1); reader('a32-owner', 'unverified')
    finally:
        key.write_bytes(oldkey); key.chmod(0o644); inrelease.write_bytes(old)
    a.clean_os_cache()
    with server():
        verify('a33-signature-recovery', True)
    reader('a33-owner')


def colon_authority():
    """Focused colon_guest.py logic appended without rerunning the other groups."""
    auth = Path('/etc/apt/apt.conf.d/99:authority.conf')
    benign = Path('/etc/apt/apt.conf.d/99:unrelated.conf')
    def targets(label):
        with server():
            a.run(['apt-get', 'update'], label + '-update')
        return a.run(['apt-get', 'indextargets', '--format', '$(SITE)'], label + '-targets').stdout
    before = targets('colon-baseline')
    assert b'/engine' in before and b'/support' in before
    try:
        a.write(benign, 'Acquire::Retries "7";\n')
        assert b'Acquire::Retries "7";' in a.run(['apt-config', 'dump'], 'colon-unrelated-config').stdout
        assert targets('colon-unrelated') == before
        reader('colon-unrelated-owner'); verify('colon-unrelated-refresh')
        empty = T / 'colon-empty'; empty.mkdir()
        emptylist = T / 'colon-empty.list'; a.write(emptylist, '# empty\n')
        a.write(auth, f'Dir::Etc::sourcelist "{emptylist}"; Dir::Etc::sourceparts "{empty}";\n')
        a.write(I / '99:authority.conf', auth.read_bytes())
        a.run(['apt-config', 'dump'], 'colon-authority-config')
        assert not targets('colon-authority').strip()
        reader('colon-authority-owner', 'unverified'); verify('colon-authority-refresh', expected=1)
        auth.unlink()
        assert targets('colon-recovery') == before
        verify('colon-recovery'); reader('colon-recovery-owner')
    finally:
        for file in (auth, benign):
            if file.exists():
                file.unlink()


def pacman():
    p.publish()
    with server(True):
        p.arun(['/usr/bin/pacman', '-Syy', '--noconfirm'], 'p01-native-sync')
        p.arun(['/usr/bin/pacman', '-S', '--noconfirm', 'hc-engine', 'hc-support'], 'p01-native-install')
    for f in (p.A / 'var/cache/pacman/pkg').glob('*.sig'):
        f.unlink()
    save('p02-cache-without-sig.json', [x.name for x in (p.A / 'var/cache/pacman/pkg').iterdir()])
    verify('p02-cache-repo-pgpsig', arch=True); permissions('p02-first-use-077', p.A); reader('p02-owner', arch=True)
    for d in ('var/cache/pacman/pkg', 'var/lib/pacman/sync'):
        for f in (p.A / d).iterdir():
            if f.is_file():
                f.unlink()
    verify('p03-retained-offline', arch=True); reader('p03-owner', arch=True)
    version = p.arun(['/usr/bin/pacman', '-Q', 'hc-engine']).stdout.decode().split()[1]
    rev = int(version.rsplit('-', 1)[1]) + 1
    p.package('hc-engine', rev); p.publish()
    p.arun(['/usr/bin/pacman', '-U', '--noconfirm', f'/repo/hc-engine-5:29.7.2-{rev}-x86_64.pkg.tar.zst'], 'p04-native-update')
    verify('p04-no-exact-evidence', arch=True, expected=1)
    with server(True):
        verify('p05-native-fetch-exact', fetch=True, arch=True)
    verify('p06-offline-after-fetch', arch=True); reader('p06-owner', arch=True)
    x = p.A / 'usr/bin/dockerd'; old = x.read_bytes()
    try:
        x.write_bytes(old + b'#tamper\n'); reader('p07-owner-tamper', 'unverified', True); verify('p07-refresh-tamper', arch=True, expected=1)
    finally:
        x.write_bytes(old)
    verify('p08-recovery', arch=True)
    revlist = p.A / 'usr/share/pacman/keyrings/hc-revoked'; old = revlist.read_bytes()
    try:
        revlist.write_text(a.fingerprint('engine') + '\n')
        reader('p09-owner-revoked', 'unverified', True); verify('p09-signature-revoked', arch=True, expected=1)
    finally:
        revlist.write_bytes(old)
    verify('p10-recovery', arch=True)


def pacman_hooks():
    a.write(p.A / 'usr/local/lib/urbit-runner/current/urbit-runner', B.read_bytes(), 0o755)
    a.write(p.A / WR.lstrip('/'), a.hook_assets['refresh-host-packages'], 0o755)
    a.write(p.A / 'etc/pacman.d/hooks/00-omabit-host-packages.hook', a.hook_assets['00-omabit-host-packages.hook'])
    for count in (1, 2):
        label = f'p{10 + count}-native-hook'
        version = p.arun(['/usr/bin/pacman', '-Q', 'hc-engine']).stdout.decode().split()[1]
        rev = int(version.rsplit('-', 1)[1]) + 1
        p.package('hc-engine', rev); p.publish()
        if count == 1:
            preserve(p.A / 'var/lib/omabit', 'pacman-proof-before-first-hook')
        out = '/hook-fixed/' + label
        (p.A / 'hook-fixed').mkdir(exist_ok=True)
        a.write(p.A / 'cleanup-fixed.sh', f'#!/bin/sh\nset -eu\ncp /var/lib/omabit/hostcompat/receipts-v2.json {out}.receipt.json\nls -l /var/cache/pacman/pkg > {out}.archives\nrm -f /var/cache/pacman/pkg/hc-*.pkg.tar.zst /var/cache/pacman/pkg/hc-*.pkg.tar.zst.sig\n', 0o755)
        a.write(p.A / 'etc/pacman.d/hooks/99-hc-cleanup.hook', '[Trigger]\nOperation = Upgrade\nOperation = Install\nType = Package\nTarget = hc-engine\n\n[Action]\nWhen = PostTransaction\nExec = /usr/bin/sh /cleanup-fixed.sh\n')
        with server(True):
            p.arun(['/usr/bin/pacman', '-Syy', '--noconfirm'], label + '-sync')
            args = ['chroot', p.A, '/usr/bin/pacman', '-S', '--noconfirm', 'hc-engine'] + (['hc-support'] if count == 1 else [])
            old = os.umask(0o077)
            try:
                a.run(['strace', '-f', '-tt', '-s', '2048', '-e', 'trace=%process,%creds,umask,fchmod', '-o', E / (label + '.transaction.strace')] + args, label)
            finally:
                os.umask(old)
        rec = json.loads((p.A / (out.lstrip('/') + '.receipt.json')).read_text())
        save(label + '.before-cleanup.json', rec)
        assert len(rec['packages']) == 8 and all(x['authorized'] for x in rec['packages'])
        assert all(x['version'] == (f'5:29.7.2-{rev}' if x['name'] == 'hc-engine' else '5:29.7.2-1') for x in rec['packages'])
        assert 'hc-engine' in (p.A / (out.lstrip('/') + '.archives')).read_text()
        assert not list((p.A / 'var/cache/pacman/pkg').glob('hc-*.pkg.tar.zst'))
        permissions(label, p.A); reader(label + '-owner-after-cleanup', arch=True)
        for file in (p.A / 'var/lib/pacman/sync').iterdir():
            if file.is_file():
                file.unlink()
        verify(label + '-retained-offline', arch=True)


def audit():
    forbidden = []
    for file in E.glob('*.strace'):
        text = file.read_text()
        for tool in re.findall(r'execve\("([^"]+)"', text):
            assert Path(tool).name not in ('dockerd', 'docker', 'containerd', 'runc', 'rootlesskit', 'urbit', 'vere'), (file, tool)
        if file.name.endswith('.reader.strace'):
            forbidden += [line for line in text.splitlines() if 'openat(' in line and any(s in line for s in ('evidence-v2', '"archives"', '"lists"', '"gnupg"'))]
    assert not forbidden, forbidden
    assert hashlib.sha256(B.read_bytes()).hexdigest() == a.options.candidate_sha256
    save('audit.json', {'private_reader_opens': [], 'engine_launches': [], 'candidate_sha256': a.options.candidate_sha256})


def execute(options, admission):
    global a, p, R, T, E, I, APT_UID, APT_GID
    # This callable is internal; only qualify.py exposes a mutating entry point.
    a = Fixtures(options, admission)
    R = T = a.root; E = a.evidence; I = a.inputs
    p = SimpleNamespace(A=a.arch, arun=a.arun, package=a.arch_package, publish=a.arch_publish, server=a.arch_server)
    if options.provider in ('apt', 'both'):
        rows = [line.split(':') for line in Path('/etc/passwd').read_text().splitlines() if line.startswith('_apt:')]
        require(len(rows) == 1 and len(rows[0]) == 7, 'native _apt identity changed after admission')
        APT_UID, APT_GID = int(rows[0][2]), int(rows[0][3])
        require(APT_UID > 0 and APT_GID > 0, 'unsafe native _apt identity')
    phases = []
    if options.provider in ('apt', 'both'):
        phases += [('apt-base', baseline), ('apt-hooks', apt_hooks), ('apt-fetch-denials', negative_fetch),
                   ('apt-authority', authority), ('apt-multisigner', multisigner), ('apt-colon', colon_authority)]
    if a.arch:
        phases += [('pacman', pacman), ('pacman-hooks', pacman_hooks)]
    results = {}
    try:
        a.setup()
        for name, fn in phases:
            try:
                fn(); results[name] = 'PASS'
            except Exception as error:
                results[name] = 'FAIL ' + repr(error)
                a.write(E / (name + '.failure.txt'), traceback.format_exc(), 0o600)
            save('results.json', results)
        audit()
    finally:
        if E.exists():
            if a.arch:
                a.arun(['/usr/bin/gpgconf', '--homedir', '/etc/pacman.d/gnupg', '--kill', 'all'], check=False)
                for name in ('proc', 'dev'):
                    if os.path.ismount(a.arch / name):
                        a.run(['umount', a.arch / name], 'unmount-' + name, check=False)
            for home in [R / 'key-engine', R / 'key-support', R / 'multi-keys']:
                if home.exists():
                    a.run(['gpgconf', '--homedir', home, '--kill', 'all'], check=False)
    print(json.dumps(results, indent=2))
    return int(any(value != 'PASS' for value in results.values()))


if __name__ == '__main__':
    raise SystemExit('Use qualify.py --help; direct suite execution is refused.')
