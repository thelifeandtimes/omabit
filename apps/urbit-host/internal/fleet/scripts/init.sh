#!/bin/bash
# PID 1 supervisor. A GroundSeg-style tmux console survives client detach.
set -euo pipefail
umask 077
export HOME=/tmp TMUX_TMPDIR=/tmp
for tool in urbit tmux curl; do command -v "$tool" >/dev/null || { echo "missing runtime dependency: $tool" >&2; exit 69; }; done
mkdir -p /tmp/runner
printf 'set-option -g remain-on-exit on\nset-option -g history-limit 10000\nset-option -g default-shell /bin/bash\n' > /tmp/runner/tmux.conf
mkfifo /tmp/runner/console.pipe
cat /tmp/runner/console.pipe &
logger_pid=$!
# Run a waiting shell first so output piping is installed before Vere starts.
printf -v command '%q ' /bin/bash /opt/omarchy-urbit/boot.sh "$@"
tmux -f /tmp/runner/tmux.conf new-session -d -s urbit -x 120 -y 40 "$command"
tmux pipe-pane -o -t urbit 'cat > /tmp/runner/console.pipe'
stop() {
  if [[ -f /tmp/runner/vere.pid ]]; then kill -TERM "$(cat /tmp/runner/vere.pid)" 2>/dev/null || true; fi
}
trap stop TERM INT
touch /tmp/runner/go
while tmux has-session -t urbit 2>/dev/null; do
  dead=$(tmux display-message -p -t urbit '#{pane_dead}' 2>/dev/null || echo 1)
  if [[ "$dead" == 1 ]]; then
    status=$(tmux display-message -p -t urbit '#{pane_dead_status}' 2>/dev/null || echo 1)
    tmux kill-server 2>/dev/null || true
    kill "$logger_pid" 2>/dev/null || true
    [[ "$status" =~ ^[0-9]+$ ]] || status=1
    exit "$status"
  fi
  sleep 1 & wait $! || true
done
exit 1
