#!/usr/bin/env python3
"""Passive discovery preserves SSH files and never evaluates executable directives."""
import importlib.util
import os
from pathlib import Path
import tempfile
import unittest

spec = importlib.util.spec_from_file_location("ssh_hosts", Path(__file__).resolve().parents[1] / "plugins/omabit/ssh-hosts.py")
hosts = importlib.util.module_from_spec(spec)
spec.loader.exec_module(hosts)


class HostSuggestions(unittest.TestCase):
    def test_aliases_includes_known_hosts_and_ports_without_execution(self):
        with tempfile.TemporaryDirectory() as directory:
            home = Path(directory)
            ssh = home / ".ssh"
            (ssh / "conf.d").mkdir(parents=True)
            marker = home / "must-not-exist"
            config = ssh / "config"
            config.write_text('Host sourcebox rooftop\nHost *.example !excluded\nHost=equals\nInclude "conf.d/with space.conf"\n'
                              f'Match exec "touch {marker}"\nKnownHostsCommand "touch {marker}"\nUserKnownHostsFile %d/.ssh/custom_known\n')
            (ssh / "conf.d/with space.conf").write_text("Host included\nInclude config\n")
            (ssh / "known_hosts").write_text("sourcebox,plain.example ssh-ed25519 AAAA\n[remote.example]:2200 ssh-ed25519 AAAA\n|1|hash|hash ssh-ed25519 AAAA\n@revoked revoked.example ssh-ed25519 AAAA\n[::1]:22 ssh-ed25519 AAAA\n")
            (ssh / "custom_known").write_text("custom.example ssh-ed25519 AAAA\n")
            before = {p: p.read_bytes() for p in ssh.rglob("*") if p.is_file()}
            result = hosts.discover(home, system_config=None, system_known=[])["hosts"]
            by_name = {(r["host"], r["port"]): r for r in result}
            self.assertEqual(set(by_name), {("sourcebox", 0), ("rooftop", 0), ("equals", 0), ("included", 0), ("plain.example", 0), ("remote.example", 2200), ("custom.example", 0)})
            self.assertEqual(by_name[("sourcebox", 0)]["source"], "SSH config")
            self.assertEqual(by_name[("remote.example", 2200)]["label"], "remote.example:2200")
            self.assertFalse(marker.exists())
            self.assertEqual(before, {p: p.read_bytes() for p in ssh.rglob("*") if p.is_file()})

    def test_custom_configuration_and_bounded_nonregular_inputs(self):
        with tempfile.TemporaryDirectory() as directory:
            home = Path(directory)
            (home / ".ssh").mkdir()
            os.mkfifo(home / ".ssh/config")
            self.assertEqual(hosts.discover(home, system_config=None, system_known=[]), {"hosts": []})
            custom = home / "custom.conf"
            custom.write_text("Host " + " ".join("host" + str(n) for n in range(200)) + "\nHost " + " ".join("extra" + str(n) for n in range(200)) + "\n")
            result = hosts.discover(home, custom, system_config=None, system_known=[])["hosts"]
            self.assertEqual(len(result), hosts.MAX_HOSTS)
            self.assertTrue(all(hosts.HOST.fullmatch(r["host"]) for r in result))


if __name__ == "__main__":
    unittest.main(verbosity=2)
