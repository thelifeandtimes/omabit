#!/usr/bin/bash
# Install the app-owned service files on an already configured development host.
set -euo pipefail
export PATH=/usr/bin:/bin

if [[ $EUID != 0 || $# != 0 ]]; then
  printf '%s\n' 'Run this script as root with no arguments after building the runner.' >&2
  exit 2
fi
ROOT="$(dirname "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")")"
if [[ -f /etc/urbit-runner/owner.json ]]; then
  printf '%s\n' 'Owner-user hosts are updated with urbit-runner install-host --owner USER --bundle DIR --public-key FILE --acknowledge-host-changes. This historical service-only installer is for legacy dedicated-account hosts.' >&2
  exit 2
fi
UID_RUNNER="$(/usr/bin/id -u urbit-runner)"
[[ $UID_RUNNER != 0 && -x "$ROOT/build/urbit-runner" ]]
[[ -f /etc/systemd/user/urbit-docker.service && -f /etc/urbit-runner/sshd_config ]]
/usr/bin/sshd -t -f /etc/urbit-runner/sshd_config

# The dedicated Docker config was created by Omabit. Refuse unrelated edits.
/usr/bin/python3 -I -c 'import json, sys
with open(sys.argv[1]) as f: value = json.load(f)
if value not in ({}, {"shutdown-timeout": 310}):
    sys.exit("Review the existing dedicated Docker configuration before installing services.")
' /usr/local/lib/urbit-runner/rootless/29.7.2/daemon.json

user_manager() { /usr/bin/systemctl --user --machine=urbit-runner@.host "$@"; }
/usr/bin/loginctl enable-linger urbit-runner
/usr/bin/systemctl start "user@${UID_RUNNER}.service"

# Stop previously installed services, then the exact foreground commands used by
# the development walkthrough. Never signal Docker or ship processes here.
if user_manager is-active --quiet urbit-runner.service; then user_manager stop urbit-runner.service; fi
if /usr/bin/systemctl is-active --quiet urbit-runner-ssh.service; then /usr/bin/systemctl stop urbit-runner-ssh.service; fi
mapfile -t manager_pids < <(/usr/bin/pgrep -u urbit-runner -f '^/usr/local/lib/urbit-runner/current/urbit-runner serve-development( |$)' || true)
mapfile -t ssh_pids < <(/usr/bin/pgrep -u root -f '^sshd: /usr/bin/sshd -D -e -f /etc/urbit-runner/sshd_config \[listener\]' || true)
pids=("${manager_pids[@]}" "${ssh_pids[@]}")
if ((${#pids[@]})); then
  /usr/bin/kill -TERM -- "${pids[@]}"
  for pid in "${pids[@]}"; do
    for ((attempt=0; attempt<60; attempt++)); do
      /usr/bin/kill -0 "$pid" 2>/dev/null || break
      /usr/bin/sleep 1
    done
    if /usr/bin/kill -0 "$pid" 2>/dev/null; then
      printf 'Process %s has not exited; inspect it before retrying.\n' "$pid" >&2
      exit 1
    fi
  done
fi

/usr/bin/install -D -o root -g root -m 0755 "$ROOT/build/urbit-runner" /usr/local/lib/urbit-runner/current/urbit-runner
/usr/bin/install -D -o root -g root -m 0644 "$ROOT/deploy/systemd/urbit-runner.service" /etc/systemd/user/urbit-runner.service
/usr/bin/install -D -o root -g root -m 0644 "$ROOT/deploy/systemd/urbit-runner-ssh.service" /etc/systemd/system/urbit-runner-ssh.service
/usr/bin/install -D -o root -g root -m 0644 "$ROOT/deploy/systemd/urbit-docker.service.d/shutdown.conf" /etc/systemd/user/urbit-docker.service.d/shutdown.conf
/usr/bin/install -D -o root -g root -m 0644 "$ROOT/deploy/systemd/user-manager-shutdown.conf" "/etc/systemd/system/user@${UID_RUNNER}.service.d/omabit-shutdown.conf"
/usr/bin/install -o root -g root -m 0644 "$ROOT/internal/rootless/assets/daemon.json" /usr/local/lib/urbit-runner/rootless/29.7.2/daemon.json
/usr/bin/systemctl daemon-reload
user_manager daemon-reload
user_manager enable --now urbit-runner.service
/usr/bin/systemctl enable --now urbit-runner-ssh.service
printf '%s\n' 'Management and SSH services enabled and started.' 'Docker shutdown-timeout takes effect at its next engine start; see docs/services.md for validation.'
