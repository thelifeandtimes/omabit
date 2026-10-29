#!/usr/bin/python3
"""One-time cleanup of a separately copied, disposable onboarding disk only."""
from pathlib import Path
import json,os,shutil,subprocess
r=Path('/opt/omabit-review')
assert os.geteuid()==0
assert Path('/sys/class/dmi/id/sys_vendor').read_text().strip()=='QEMU'
assert 'omabit.onboarding=prepare-v1' in Path('/proc/cmdline').read_text().split()
assert r.joinpath('marker').read_text().strip()=='omabit-manual-review-v1'
assert not (r/'fresh-onboarding.json').exists(), 'Preparation is never a reset command.'
assert subprocess.run(['pgrep','-x','dockerd'],stdout=subprocess.DEVNULL).returncode==1
paths=['/etc/urbit-runner','/usr/local/lib/urbit-runner','/etc/systemd/user/urbit-runner.service','/etc/systemd/user/urbit-docker.service','/etc/systemd/system/urbit-runner-ssh.service','/etc/systemd/system/multi-user.target.wants/urbit-runner-ssh.service','/etc/systemd/system/user@1000.service.d/omabit-shutdown.conf','/etc/pacman.d/hooks/00-omabit-host-packages.hook','/srv/urbit/piers','/home/review/.local/share/urbit-runner','/home/review/.config/urbit-collector','/home/review/.config/omabit','/home/review/.local/state/omabit','/home/review/.local/state/omarchy-urbit-theme.review','/home/review/.local/state/omarchy-urbit-dms.review','/home/review/.local/state/omarchy-urbit-theme-ownership','/home/review/.local/share/keyrings','/home/review/.ssh','/home/review/.config/chromium','/home/review/.cache/chromium','/opt/omabit-review/rootless-bundle','/opt/omabit-review/runtime/runtime.oci.tar','/opt/omabit-review/after-restart.json','/opt/omabit-review/before-restart.json']
for p in Path('/home/review/.config/systemd/user/default.target.wants').glob('urbit-*'):paths.append(str(p))
removed=[]
for name in paths:
 p=Path(name)
 if p.is_symlink() or p.is_file():p.unlink();removed.append(name)
 elif p.is_dir():shutil.rmtree(p);removed.append(name)
# Preserve OS-assigned subordinate-ID prerequisites; these are not Omabit state.
p=Path('/home/review/.ssh');p.mkdir(mode=0o700);os.chown(p,1000,1000)
shutil.copyfile(r/'transfers/ONBOARDING.html',r/'START-HERE.html')
# Internet-only guest egress; local loopback remains usable for the fake ships.
p=Path('/etc/systemd/network/80-review.network');p.parent.mkdir(exist_ok=True)
p.write_text('[Match]\nDriver=virtio_net\n[Network]\nAddress=192.0.2.15/24\nGateway=192.0.2.2\nDNS=192.0.2.3\nIPv6AcceptRA=no\nLinkLocalAddressing=no\n')
p=Path('/etc/resolv.conf');p.unlink(missing_ok=True);p.write_text('nameserver 192.0.2.3\n')
p=Path('/opt/omabit-review/egress.sh');p.write_text('''#!/bin/sh
set -eu
/usr/bin/iptables -N OMABIT_REVIEW_EGRESS
/usr/bin/iptables -A OMABIT_REVIEW_EGRESS -o lo -j ACCEPT
for proto in udp tcp; do /usr/bin/iptables -A OMABIT_REVIEW_EGRESS -d 192.0.2.3 -p "$proto" --dport 53 -j ACCEPT; done
for net in 0.0.0.0/8 10.0.0.0/8 100.64.0.0/10 127.0.0.0/8 169.254.0.0/16 172.16.0.0/12 192.168.0.0/16 192.0.0.0/24 192.0.2.0/24 198.18.0.0/15 224.0.0.0/4 240.0.0.0/4; do /usr/bin/iptables -A OMABIT_REVIEW_EGRESS -d "$net" -j DROP; done
/usr/bin/iptables -A OUTPUT -j OMABIT_REVIEW_EGRESS
/usr/bin/sysctl -q -w net.ipv6.conf.all.disable_ipv6=1
''');p.chmod(0o755)
p=Path('/etc/systemd/system/omabit-review-egress.service');p.write_text('[Unit]\nDescription=Review guest internet egress boundaries\nBefore=systemd-networkd.service network-pre.target\nAfter=local-fs.target\n[Service]\nType=oneshot\nExecStart=/opt/omabit-review/egress.sh\nRemainAfterExit=yes\n[Install]\nWantedBy=multi-user.target\n')
p=Path('/etc/systemd/system/systemd-networkd.service.d/review.conf');p.parent.mkdir(exist_ok=True);p.write_text('[Unit]\nRequires=omabit-review-egress.service\nAfter=omabit-review-egress.service\n')
subprocess.run(['systemctl','enable','systemd-networkd.service','omabit-review-egress.service'],check=True)
# These records describe absence of setup, not a fabricated setup success.
(r/'fresh-onboarding.json').write_text(json.dumps({'version':1,'state':'not-onboarded','removed_generated_paths':removed,'native_prerequisites':'preinstalled','development_plugins':'preinstalled with review IDs','ships_enrolled':0,'network':'public internet; guest private/LAN/tailnet egress blocked'},indent=2)+'\n')
subprocess.run(['sync'],check=True)
print('FRESH_ONBOARDING_PREPARED',flush=True)
subprocess.run(['systemctl','--no-block','poweroff'],check=True)
