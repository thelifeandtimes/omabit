# Brave launch failure caused by system D-Bus exhaustion

Date: 2026-10-09

## Symptom

Brave stopped running at about 16:59 PDT and every subsequent launch aborted almost immediately. Launch attempts at 17:42, 17:43, and during diagnosis produced new Brave Crashpad minidumps.

## What was established

- Brave 1.93.138 aborts with `SIGABRT` rather than being killed by the OOM killer.
- The immediate failure is a libdbus assertion:

  ```text
  Failed to add match rule ... org.freedesktop.UPower
  org.freedesktop.DBus.Error.NoReply
  dbus_pending_call_set_notify(): assertion "pending != NULL" failed
  ```

- The system journal records the D-Bus broker disconnecting peers because they do not have resources to receive replies or signals. These messages coincide exactly with Brave's failed launches.
- A completely fresh temporary Brave profile fails in the same way with exit status 134. This rules out Brave's normal profile, tabs, session data, and extensions as the cause.
- The machine is not short of RAM: approximately 69 GiB was available, and there was no OOM event.
- Brave's profile and user data remain intact. A stale `SingletonLock` is a consequence of the abort, not the cause.
- The Brave package has not just changed; the installed build dates from August 21. This does not look like a new Brave package regression.

## Likely mechanism

The system D-Bus broker is unhealthy/resource-exhausted. It has been running since September 29, uses substantially more memory than the user-session broker, and has many system-bus sockets with backed-up send queues. New Brave processes make several system-bus requests during startup; the broker disconnects them instead of delivering replies, and Brave aborts on the resulting libdbus error.

The strongest correlated source of D-Bus pressure is `urbit-runner.service` (running as user `urbit-runner`, UID 954):

- It is running `urbit-runner serve-development --acknowledge-development --resume-running`.
- It generated about 30,029 localhost SSH/PAM/logind sessions on October 9, roughly one every two seconds.
- Each login/logout causes systemd-logind D-Bus traffic.
- Numerous long-running `urbit` processes have system-bus connections with backed-up receive queues.

This strongly suggests that repeated runner SSH sessions, logind signal traffic, and Urbit processes that are not draining that traffic combined to exhaust the broker. The evidence does not prove that this is the only contributor, but the correlation and socket state are compelling.

## Immediate recovery

Reboot the machine. This is safer than attempting to restart the system D-Bus broker in place and will clear the broker and socket queues. Active Urbit workloads may be interrupted. Brave should launch normally after the reboot without deleting or resetting its profile.

## Follow-up

If the problem returns, investigate why the development runner opens a new localhost SSH/login session every two seconds. Reducing that polling/session churn or changing it so health checks do not create full PAM/logind sessions should prevent the system bus from accumulating the same pressure. Also inspect why the long-running Urbit processes open system-bus connections but leave data queued on them.

Useful evidence commands:

```bash
journalctl -u dbus-broker.service --since today
journalctl --since today -g 'New session.*urbit-runner'
busctl --system list
ss -xnpH 'src /run/dbus/system_bus_socket'
free -h
```
