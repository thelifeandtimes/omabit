"""Read-only disk/quota admission for the private Linux x86-64 VM harness."""
import ctypes
import errno
import os
from pathlib import Path
import platform


class Quota(ctypes.Structure):
    # linux/quota.h: if_dqblk. Limits are 1024-byte units; usage is bytes.
    _fields_ = [(name, ctypes.c_uint64) for name in (
        'hard_blocks', 'soft_blocks', 'used_bytes', 'hard_inodes', 'soft_inodes',
        'used_inodes', 'block_time', 'inode_time')] + [('valid', ctypes.c_uint32)]


def available(directory):
    directory = Path(directory)
    fs = os.statvfs(directory)
    result = {'filesystem_available_bytes': fs.f_bavail * fs.f_frsize}
    if platform.system() != 'Linux' or platform.machine() != 'x86_64':
        raise RuntimeError('VM quota admission currently requires Linux x86-64')
    libc = ctypes.CDLL(None, use_errno=True)
    quota = Quota()
    fd = os.open(directory, os.O_RDONLY | os.O_DIRECTORY)
    try:
        # x86-64 unistd_64.h: quotactl_fd=443. QCMD(Q_GETQUOTA, USRQUOTA).
        status = libc.syscall(443, fd, ctypes.c_uint(0x80000700), os.getuid(), ctypes.byref(quota))
        error = ctypes.get_errno()
    finally:
        os.close(fd)
    if status:
        if error not in (errno.ESRCH, errno.EOPNOTSUPP):
            raise OSError(error, 'cannot establish temporary-storage user quota')
        result['quota'] = 'not_enabled'
    else:
        if quota.valid & 3 != 3:
            raise RuntimeError('incomplete temporary-storage quota evidence')
        result['quota'] = {name: getattr(quota, name) for name, _ in Quota._fields_}
        limits = [value * 1024 for value in (quota.hard_blocks, quota.soft_blocks) if value]
        if limits:
            result['quota_available_bytes'] = max(0, min(limits) - quota.used_bytes)
    result['available_bytes'] = min(result['filesystem_available_bytes'], result.get('quota_available_bytes', result['filesystem_available_bytes']))
    return result


def admit(directory, required_bytes):
    result = available(directory)
    result['required_bytes'] = required_bytes
    if result['available_bytes'] < required_bytes:
        raise RuntimeError('insufficient private-lab storage headroom: ' + str(result))
    return result
