"""Read-only, fail-closed admission for an explicitly provisioned disposable VM."""
import hashlib
import json
import os
from pathlib import Path
import platform
import re
import socket
import stat
import subprocess
import sys

MARKER = Path('/etc/omabit-host-packages-qualification.json')
PURPOSE = 'omabit-disposable-native-package-auth'
HOOKS = ('refresh-host-packages', '00-omabit-host-packages', '00-omabit-host-packages.hook')
ROLES = ('dockerd', 'docker', 'containerd', 'runc', 'rootlesskit', 'slirp4netns', 'newuidmap', 'newgidmap')
STATE_PATHS = ('etc/urbit-runner', 'usr/local/lib/urbit-runner', 'var/lib/omabit',
               'var/lib/urbit-runner', 'srv/urbit',
               'etc/apt/apt.conf.d/00-omabit-host-packages', 'etc/apt/apt.conf.d/99-hc-cleanup',
               'etc/apt/apt.conf.d/99-hc-fetch-canary', 'etc/apt/apt.conf.d/98-hc-selection',
               'etc/apt/apt.conf.d/99:authority.conf', 'etc/apt/apt.conf.d/99:unrelated.conf',
               'etc/apt/sources.list.d/hc.sources', 'etc/apt/keyrings/hc-engine.gpg',
               'etc/apt/keyrings/hc-engine.asc', 'etc/apt/keyrings/hc-support.gpg',
               'etc/apt/keyrings/hc-support.asc', 'etc/pacman.d/hooks/00-omabit-host-packages.hook',
               'etc/pacman.d/hooks/99-hc-cleanup.hook')


class Refusal(Exception):
    pass


def require(condition, message):
    if not condition:
        raise Refusal(message)


def absolute_path(value):
    require(bool(re.fullmatch(r'/[A-Za-z0-9_./-]+', str(value))), 'paths must be absolute and shell-safe')
    p = Path(value)
    require(str(p) == str(value) and '..' not in p.parts and p != Path('/'), 'noncanonical or root path refused')
    return p


def trusted_path(path, directory=False):
    """No links, foreign ownership, writable parents, hardlinks or special files."""
    path = Path(path)
    for parent in reversed(path.parents):
        s = parent.lstat()
        require(stat.S_ISDIR(s.st_mode) and s.st_uid == 0 and not s.st_mode & 0o022,
                f'unsafe parent: {parent}')
    s = path.lstat()
    require(s.st_uid == 0 and not s.st_mode & 0o022, f'unsafe ownership/mode: {path}')
    if directory:
        require(stat.S_ISDIR(s.st_mode), f'not a real directory: {path}')
    else:
        require(stat.S_ISREG(s.st_mode) and s.st_nlink == 1, f'not an exclusively linked regular file: {path}')
    return s


def trusted_bytes(path, limit):
    before = trusted_path(path)
    require(before.st_size <= limit, f'input too large: {path}')
    fd = os.open(path, os.O_RDONLY | os.O_NOFOLLOW | os.O_NONBLOCK)
    with os.fdopen(fd, 'rb') as f:
        opened = os.fstat(f.fileno())
        data = f.read(limit + 1)
        after = os.fstat(f.fileno())
    current = Path(path).lstat()
    identity = lambda s: (s.st_dev, s.st_ino, s.st_mode, s.st_uid, s.st_gid,
                          s.st_nlink, s.st_size, s.st_mtime_ns, s.st_ctime_ns)
    require(len(data) <= limit and identity(before) == identity(opened) == identity(after) == identity(current),
            f'input changed while reading: {path}')
    return data


def digest(path):
    return hashlib.sha256(trusted_bytes(path, 128 << 20)).hexdigest()


def decode_marker(data):
    def unique(pairs):
        result = {}
        for key, value in pairs:
            require(key not in result, 'duplicate marker key')
            result[key] = value
        return result
    return json.loads(data, object_pairs_hook=unique)


class Probe:
    """Production observations. Tests inject a fake probe, never an alternate CLI root."""
    def identity(self):
        require(platform.system() == 'Linux', 'Linux required')
        require(os.geteuid() == 0, 'root required inside the disposable VM')
        require(platform.machine() == 'x86_64', 'this retained fixture targets Linux/amd64')
        require(not sys.flags.optimize, 'Python optimization disables case assertions; refused')
        for name in ('APT_CONFIG', 'LD_PRELOAD', 'LD_LIBRARY_PATH', 'PYTHONPATH', 'GNUPGHOME'):
            require(not os.environ.get(name), f'inherited {name} is refused')
        return socket.gethostname()

    def marker(self):
        return decode_marker(trusted_bytes(MARKER, 16384))

    def vm_uuid(self):
        # DMI identity must match a separately provisioned marker and CLI pin.
        return Path('/sys/class/dmi/id/product_uuid').read_text().strip().lower()

    def virtualization(self):
        tool = '/usr/bin/systemd-detect-virt'
        trusted_path(tool)
        env = {'PATH': '/usr/bin:/usr/sbin:/bin:/sbin', 'LC_ALL': 'C'}
        container = subprocess.run([tool, '--container'], capture_output=True, env=env, timeout=10)
        require(container.returncode == 1, 'containers or unknown container detection are refused')
        vm = subprocess.run([tool, '--vm'], capture_output=True, env=env, timeout=10)
        require(vm.returncode == 0, 'a positively identified VM is required')
        kind = vm.stdout.decode().strip()
        require(kind in ('kvm', 'qemu'), 'only documented QEMU/KVM guests are admitted')
        return kind

    def processes(self):
        # Inspect exe/comm rather than arbitrary argv: a harness filename may itself
        # contain "urbit". Unreadable live processes fail closed; exiting PIDs do not.
        for entry in Path('/proc').iterdir():
            if not entry.name.isdigit():
                continue
            try:
                comm = (entry / 'comm').read_text().strip()
                try:
                    executable = os.readlink(entry / 'exe').removesuffix(' (deleted)')
                except FileNotFoundError:
                    executable = ''  # kernel thread or exiting process
            except FileNotFoundError:
                continue
            name = Path(executable).name
            require(not (name in ROLES + ('urbit', 'vere', 'urbit-runner') or
                         comm in ('urbit', 'vere', 'urbit-runner', 'dockerd', 'containerd') or
                         name.startswith(('urbit', 'vere', 'dockerd', 'containerd-shim')) or
                         comm.startswith(('urbit', 'vere'))),
                    f'preexisting runtime/manager process refused: PID {entry.name}')

    def clean_root(self, root, pacman=False):
        for name in STATE_PATHS:
            require(not os.path.lexists(root / name), f'existing owner/state/pier boundary: {root / name}')
        for name in ROLES:
            if pacman and name in ('newuidmap', 'newgidmap'):
                continue  # native bootstrap shadow removal, only in the clean subroot
            for directory in ('usr/bin', 'usr/sbin'):
                require(not os.path.lexists(root / directory / name), f'preexisting runtime executable: {name}')
        for relative in ('etc/apt/auth.conf', 'etc/apt/auth.conf.d'):
            p = root / relative
            if p.is_file():
                require(not p.read_bytes().strip(), 'native credentials are not permitted in this fixture guest')
            elif p.is_dir():
                require(not any(p.iterdir()), 'native credentials directory is not empty')
        if pacman:
            require(not os.path.lexists(root / 'repo'), 'Arch fixture repository already exists')
            for line in Path('/proc/self/mountinfo').read_text().splitlines():
                mount = line.split()[4]
                require(not (mount == str(root) or mount.startswith(str(root) + '/')),
                        'prepare an unmounted clean Arch extraction, not a bind-mounted host root')

    def directory(self, path):
        trusted_path(path, directory=True)

    def absent(self, path):
        require(not os.path.lexists(path), f'output already exists; use a fresh guest: {path}')

    def file(self, path, limit=128 << 20):
        return trusted_bytes(path, limit)

    def apt_identity(self):
        rows = [line.split(':') for line in self.file('/etc/passwd', 1 << 20).decode().splitlines() if line.startswith('_apt:')]
        require(len(rows) == 1 and len(rows[0]) == 7 and rows[0][2].isdigit() and rows[0][3].isdigit(), 'existing native _apt identity required')
        require(int(rows[0][2]) > 0 and int(rows[0][3]) > 0, 'unsafe native _apt identity')

    def tool(self, path):
        # Fixed native tool paths may be normal distro symlinks inside the guest.
        if str(path) == '/usr/bin/chroot' and not Path(path).exists():
            path = '/usr/sbin/chroot'
        target = Path(path).resolve(strict=True)
        s = trusted_path(target)
        # Distro mount/umount may conventionally be setuid-root. They run only
        # after this harness has already established guest-root identity. Native
        # acquisition/validation tools receive no such exception.
        mount_tool = Path(path).name in ('mount', 'umount')
        require(s.st_mode & 0o111 and (mount_tool or not s.st_mode & 0o6000), f'unsafe native tool: {path}')


def validate(options, probe=None):
    """Read-only gate. No directory creation, candidate execution or key generation."""
    p = probe or Probe()
    hostname = p.identity()  # must precede even marker/candidate reads on a live host
    require(bool(re.fullmatch(r'hostcompat-qual-[a-z0-9-]+', options.hostname or '')), 'dedicated expected hostname required')
    require(hostname == options.hostname, 'guest hostname does not match the explicit pin')
    require(bool(re.fullmatch(r'[0-9a-f]{64}', options.candidate_sha256 or '')), 'exact candidate SHA-256 required')
    require(bool(re.fullmatch(r'[A-Za-z0-9_.+-]{1,128}', options.candidate_version or '')), 'exact development version label required')
    require(bool(re.fullmatch(r'[1-9][0-9]*', options.candidate_schema or '')), 'exact candidate state-schema required')
    require(bool(re.fullmatch(r'[0-9a-fA-F-]{36}', options.vm_uuid or '')), 'explicit VM UUID required')
    workspace = absolute_path(options.workspace)
    require(workspace.is_relative_to('/opt') and len(workspace.parts) >= 3, 'workspace must be a dedicated child of /opt')
    candidate = absolute_path(options.candidate)
    hooks = absolute_path(options.hooks)
    arch = absolute_path(options.pacman_root) if options.pacman_root else None
    require((options.provider in ('pacman', 'both')) == (arch is not None), 'Pacman requires an explicit clean Arch subroot')
    if arch:
        require(arch.is_relative_to(workspace) and arch != workspace and not arch.is_relative_to(workspace / 'run'),
                'Arch root must be a separate child of the dedicated workspace')
    marker = p.marker()
    expected = {'schema': 1, 'purpose': PURPOSE, 'hostname': options.hostname,
                'vm_uuid': options.vm_uuid.lower(), 'workspace': str(workspace),
                'candidate_sha256': options.candidate_sha256, 'candidate_version': options.candidate_version,
                'candidate_schema': options.candidate_schema, 'provider': options.provider,
                'pacman_root': str(arch) if arch else None}
    require(isinstance(marker, dict) and set(marker) == set(expected) | {'hook_sha256'}, 'marker schema/fields do not match')
    require(all(marker[k] == v for k, v in expected.items()), 'marker does not authorize this exact invocation')
    require(p.vm_uuid() == options.vm_uuid.lower(), 'VM UUID does not match marker and invocation')
    virtualization = p.virtualization()
    p.processes()
    p.clean_root(Path('/'))
    p.directory(workspace)
    p.absent(workspace / 'run')
    p.directory(hooks)
    if arch:
        p.directory(arch)
        sub = decode_marker(p.file(arch / '.hostcompat-clean-root.json', 4096))
        require(sub == {'schema': 1, 'purpose': 'omabit-disposable-pacman-root',
                        'vm_uuid': options.vm_uuid.lower(), 'root': str(arch)}, 'Arch clean-root marker mismatch')
        p.clean_root(arch, pacman=True)
    candidate_bytes = p.file(candidate)
    require(hashlib.sha256(candidate_bytes).hexdigest() == options.candidate_sha256, 'candidate digest mismatch')
    require(candidate_bytes.startswith(b'\x7fELF'), 'candidate must be the static Linux ELF runner')
    require(isinstance(marker['hook_sha256'], dict) and set(marker['hook_sha256']) == set(HOOKS), 'all generated hook hashes must be pinned')
    for name in HOOKS:
        data = p.file(hooks / name, 65536)
        require(hashlib.sha256(data).hexdigest() == marker['hook_sha256'][name], f'hook digest mismatch: {name}')
    tools = ['gpg', 'gpg-agent', 'gpgconf', 'strace', 'prlimit', 'setpriv', 'chroot', 'mount', 'umount', 'bsdtar']
    if options.provider in ('apt', 'both'):
        p.apt_identity()
        tools += ['apt-get', 'apt-config', 'apt-ftparchive', 'dpkg', 'dpkg-deb', 'dpkg-query', 'gpgv', 'python3']
    for name in tools:
        p.tool('/usr/bin/' + name)
    if options.provider in ('apt', 'both'):
        p.tool('/usr/lib/apt/apt-helper')
    if arch:
        for name in ('pacman', 'pacman-conf', 'repo-add', 'gpg', 'gpgconf', 'bsdtar', 'curl', 'prlimit'):
            p.tool(arch / 'usr/bin' / name)
    return {'validated': True, 'mutated': False, **expected, 'virtualization': virtualization,
            'candidate': str(candidate), 'hooks': str(hooks), 'hook_sha256': marker['hook_sha256'],
            'schema_execution': 'run checks state-schema after admission and before the first write'}
