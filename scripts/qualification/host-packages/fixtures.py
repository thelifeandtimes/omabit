"""Real signed inert .deb/Pacman fixtures. Call only after qualify.py admission."""
import datetime
import email.utils
import gzip
import hashlib
import http.server
import json
from pathlib import Path
import subprocess
import threading

from guards import require, trusted_bytes

ROLES = {'hc-engine': ['dockerd', 'docker'],
         'hc-support': ['containerd', 'runc', 'rootlesskit', 'slirp4netns', 'newuidmap', 'newgidmap']}
RUNNER = Path('/usr/local/lib/urbit-runner/current/urbit-runner')


class Fixtures:
    def __init__(self, options, admission):
        self.options, self.admission = options, admission
        self.root = Path(options.workspace) / 'run'
        self.evidence = self.root / 'evidence'
        self.inputs = self.root / 'inputs'
        self.arch = Path(options.pacman_root) if options.pacman_root else None
        self.hook_assets = {}

    @staticmethod
    def write(path, data, mode=0o644):
        path = Path(path)
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_bytes(data if isinstance(data, bytes) else data.encode())
        path.chmod(mode)

    def run(self, args, label=None, check=True, env=None, **kwargs):
        args = list(map(str, args))
        print('+', ' '.join(args), flush=True)
        clean = {'PATH': '/usr/bin:/usr/sbin:/bin:/sbin', 'HOME': '/root',
                 'LC_ALL': 'C', 'LANG': 'C', 'DEBIAN_FRONTEND': 'noninteractive'}
        if env:
            clean.update(env)
        r = subprocess.run(args, stdout=subprocess.PIPE, stderr=subprocess.PIPE,
                           env=clean, timeout=180, **kwargs)
        if label:
            self.write(self.evidence / (label + '.stdout'), r.stdout, 0o600)
            self.write(self.evidence / (label + '.stderr'), r.stderr, 0o600)
        # No command stdin is logged: fixture secret-key transfer stays in memory.
        with (self.evidence / 'commands.jsonl').open('a') as log:
            log.write(json.dumps({'argv': args, 'exit': r.returncode, 'label': label}) + '\n')
        require(not check or r.returncode == 0, f'native command failed ({r.returncode}): {label or args[0]}')
        return r

    def arun(self, args, label=None, check=True, **kwargs):
        return self.run(['chroot', self.arch] + list(args), label, check, **kwargs)

    def fingerprint(self, key):
        out = self.run(['gpg', '--homedir', self.root / ('key-' + key), '--with-colons', '--list-keys']).stdout
        return next(x.split(':')[9] for x in out.decode().splitlines() if x.startswith('fpr:'))

    def make_package(self, name, revision):
        tree = self.root / f'pkg-{name}-{revision}'
        version = f'5:29.7.2-{revision}'
        self.write(tree / 'DEBIAN/control', f'Package: {name}\nVersion: {version}\nArchitecture: amd64\nMaintainer: Disposable Fixture <fixture@invalid.test>\nDescription: NOT an engine; synthetic native authentication test payload\n')
        for exe in ROLES[name]:
            self.write(tree / 'usr/bin' / exe, f'#!/bin/sh\n# synthetic fixture {name} {version} {exe}; never an engine\nexit 97\n', 0o755)
        section = 'engine' if name == 'hc-engine' else 'support'
        archive = self.root / 'repo' / section / 'pool' / f'{name}_29.7.2-{revision}_amd64.deb'
        archive.parent.mkdir(parents=True, exist_ok=True)
        self.run(['dpkg-deb', '--build', '--root-owner-group', tree, archive], f'build-{name}-{revision}')
        return archive

    def publish(self, section):
        base = self.root / 'repo' / section
        dest = base / 'dists/stable/main/binary-amd64/Packages'
        data = self.run(['apt-ftparchive', 'packages', 'pool'], cwd=base).stdout
        self.write(dest, data)
        self.write(str(dest) + '.gz', gzip.compress(data, mtime=0))
        now = datetime.datetime.now(datetime.timezone.utc)
        text = f'Origin: Hostcompat native synthetic fixture\nLabel: {section}\nSuite: stable\nCodename: stable\nDate: {email.utils.format_datetime(now)}\nValid-Until: {email.utils.format_datetime(now + datetime.timedelta(days=7))}\nArchitectures: amd64\nComponents: main\nSHA256:\n'
        for file in (dest, Path(str(dest) + '.gz')):
            data = file.read_bytes()
            text += f' {hashlib.sha256(data).hexdigest()} {len(data)} {file.relative_to(base / "dists/stable")}\n'
        release = base / 'dists/stable/Release'
        self.write(release, text)
        self.run(['gpg', '--homedir', self.root / ('key-' + section), '--batch', '--yes', '--digest-algo', 'SHA256', '--clearsign', '--output', release.parent / 'InRelease', release], 'sign-' + section)

    def arch_package(self, name, revision):
        tree = self.arch / f'build-{name}-{revision}'
        for exe in ROLES[name]:
            self.write(tree / 'usr/bin' / exe, f'#!/bin/sh\n# ARCH SYNTHETIC FIXTURE {name} 5:29.7.2-{revision} {exe}\nexit 97\n', 0o755)
        self.write(tree / '.PKGINFO', f'pkgname = {name}\npkgbase = {name}\npkgver = 5:29.7.2-{revision}\npkgdesc = Synthetic nonfunctional auth fixture\nurl = https://invalid.test\nbuilddate = {int(datetime.datetime.now().timestamp())}\npackager = Disposable fixture\nsize = 4096\narch = x86_64\nlicense = MIT\n')
        dest = self.arch / f'repo/{name}-5:29.7.2-{revision}-x86_64.pkg.tar.zst'
        self.run(['bsdtar', '--zstd', '-cf', dest, '-C', tree, '.PKGINFO', 'usr'], f'arch-build-{name}-{revision}')
        self.run(['gpg', '--homedir', self.root / 'key-engine', '--batch', '--yes', '--digest-algo', 'SHA256', '--detach-sign', '--output', str(dest) + '.sig', dest], f'arch-sign-{name}-{revision}')

    def arch_publish(self):
        # Fresh one-shot suite has revisions1–4; native version order is numeric.
        files = sorted((self.arch / 'repo').glob('*.pkg.tar.zst'), key=lambda p: (p.name.split('-5:')[0], int(p.name.split('29.7.2-')[1].split('-')[0])))
        self.arun(['/usr/bin/repo-add', '--include-sigs', '/repo/hc.db.tar.gz'] + ['/repo/' + p.name for p in files], 'arch-repo-add')

    def arch_server(self):
        handler = lambda *a, **kw: http.server.SimpleHTTPRequestHandler(*a, directory=str(self.arch / 'repo'), **kw)
        server = http.server.ThreadingHTTPServer(('127.0.0.1', 18090), handler)
        threading.Thread(target=server.serve_forever, daemon=True).start()
        return server

    def clean_os_cache(self):
        self.run(['apt-get', 'clean'])
        for file in Path('/var/lib/apt/lists').iterdir():
            if file.is_file():
                file.unlink()

    @staticmethod
    def digest_tree(root):
        return {str(p): hashlib.sha256(p.read_bytes()).hexdigest() for p in Path(root).rglob('*') if p.is_file()}

    def setup(self):
        # Called once, only after all guards and schema check. Never infer a reset
        # from a missing enrollment: preexisting management state was refused.
        candidate = trusted_bytes(self.options.candidate, 128 << 20)
        require(hashlib.sha256(candidate).hexdigest() == self.options.candidate_sha256, 'candidate changed before staging')
        for name, expected in self.admission['hook_sha256'].items():
            data = trusted_bytes(Path(self.options.hooks) / name, 65536)
            require(hashlib.sha256(data).hexdigest() == expected, 'hook changed before staging')
            self.hook_assets[name] = data
        self.root.mkdir(mode=0o755)
        self.evidence.mkdir(mode=0o700)
        self.inputs.mkdir(mode=0o755)
        self.write(self.evidence / 'admission.json', json.dumps(self.admission, indent=2), 0o600)
        self.write(RUNNER, candidate, 0o755)
        for name in ('engine', 'support'):
            home = self.root / ('key-' + name)
            home.mkdir(mode=0o700)
            self.run(['gpg', '--homedir', home, '--batch', '--pinentry-mode', 'loopback', '--passphrase', '', '--quick-generate-key', f'{name} fixture <{name}@invalid.test>', 'rsa2048', 'sign', '0'], 'keygen-' + name)
        if self.options.provider in ('apt', 'both'):
            self.setup_apt()
        if self.arch:
            self.setup_arch()

    def setup_apt(self):
        for section in ('engine', 'support'):
            home = self.root / ('key-' + section)
            for ext, flags in [('gpg', []), ('asc', ['--armor'])]:
                self.write(f'/etc/apt/keyrings/hc-{section}.{ext}', self.run(['gpg', '--homedir', home] + flags + ['--export']).stdout)
        for name in ROLES:
            self.run(['dpkg', '-i', self.make_package(name, 1)], 'install-' + name)
        for section in ('engine', 'support'):
            self.publish(section)
        official = self.root / 'official-sources'
        official.mkdir()
        main = Path('/etc/apt/sources.list')
        if main.exists():
            main.rename(official / 'sources.list')
        parts = Path('/etc/apt/sources.list.d')
        for file in parts.iterdir():
            file.rename(official / file.name)
        text = ''.join(f'Types: deb\nURIs: http://127.0.0.1/{s}\nSuites: stable\nComponents: main\nArchitectures: amd64\nSigned-By: /etc/apt/keyrings/hc-{s}.gpg\n\n' for s in ('engine', 'support'))
        self.write(parts / 'hc.sources', text)
        self.run(['dpkg-query', '-W'], 'ubuntu-packages')
        for tool in ('apt-get', 'dpkg-deb', 'gpg', 'gpgv', 'prlimit', 'strace'):
            self.run([tool, '--version'], 'version-' + tool)

    def setup_arch(self):
        # The extraction was signature/digest checked by the guest provisioner.
        # Only /dev and /proc from this already isolated VM are mounted here.
        self.run(['mount', '--bind', '/dev', self.arch / 'dev'])
        self.run(['mount', '--make-private', self.arch / 'dev'])
        self.run(['mount', '-t', 'proc', 'proc', self.arch / 'proc'])
        self.write(self.arch / 'usr/local/bin/urbit-runner', RUNNER.read_bytes(), 0o755)
        for tool in ('pacman', 'pacman-conf', 'gpg', 'bsdtar', 'curl', 'prlimit'):
            self.arun(['/usr/bin/' + tool, '--version'], 'arch-version-' + tool)
        self.arun(['/usr/bin/pacman', '-Q'], 'arch-packages')
        if self.arun(['/usr/bin/pacman', '-Q', 'shadow'], check=False).returncode == 0:
            self.arun(['/usr/bin/pacman', '-Rdd', '--noconfirm', 'shadow'], 'arch-remove-bootstrap-shadow')
        private = self.arch / 'etc/pacman.d/gnupg'
        private.mkdir(mode=0o700, exist_ok=True)
        private.chmod(0o700)
        public = self.run(['gpg', '--homedir', self.root / 'key-engine', '--export']).stdout
        self.write(self.arch / 'usr/share/pacman/keyrings/hc.gpg', public)
        fp = self.fingerprint('engine')
        self.write(self.arch / 'usr/share/pacman/keyrings/hc-trusted', fp + ':6:\n')
        self.write(self.arch / 'usr/share/pacman/keyrings/hc-revoked', '')
        self.arun(['/usr/bin/gpg', '--homedir', '/etc/pacman.d/gnupg', '--batch', '--import', '/usr/share/pacman/keyrings/hc.gpg'], 'arch-import-fixture-key')
        self.arun(['/usr/bin/gpg', '--homedir', '/etc/pacman.d/gnupg', '--batch', '--import-ownertrust'], 'arch-ownertrust', input=(fp + ':6:\n').encode())
        self.arun(['/usr/bin/gpg', '--homedir', '/etc/pacman.d/gnupg', '--batch', '--check-trustdb'], 'arch-trustdb')
        self.write(self.arch / 'etc/pacman.conf', '[options]\nArchitecture = x86_64\nSigLevel = Required DatabaseOptional\nLocalFileSigLevel = Required\n[hc]\nServer = http://127.0.0.1:18090\n')
        (self.arch / 'repo').mkdir()
        for name in ROLES:
            self.arch_package(name, 1)
        self.arch_publish()


if __name__ == '__main__':
    raise SystemExit('Use qualify.py --help; direct fixture generation is refused.')
