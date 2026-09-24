package client

import (
	"slices"
	"testing"
)

func TestSSHRejectsCommandInjectionAndUnsafeOverrides(t *testing.T) {
	for _, host := range []string{"", "-oProxyCommand=id", "user@host", "host;id", "host\n", "host name", "$(id)"} {
		if err := (SSHConfig{Host: host}).Validate(); err == nil {
			t.Fatalf("unsafe SSH target accepted: %q", host)
		}
	}
	args := sshArguments(SSHConfig{Host: "runner.example", ConfigFile: "/private/ssh.conf"})
	for _, option := range []string{"-oBatchMode=yes", "-oStrictHostKeyChecking=yes", "-oControlPath=none", "-oForwardAgent=no", "-oClearAllForwardings=yes", "-oPermitLocalCommand=no", "-oRemoteCommand=none"} {
		if !slices.Contains(args, option) {
			t.Fatalf("missing invariant %s", option)
		}
	}
	if args[len(args)-1] != SSHCommand || args[len(args)-2] != "runner.example" {
		t.Fatal("remote command is not fixed")
	}
}
