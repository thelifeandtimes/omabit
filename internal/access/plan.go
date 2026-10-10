package access

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const ShellPath = "/usr/local/lib/urbit-runner/current/urbit-runner"

// Prepare writes an inert review bundle, never host configuration. Installation
// and changing the service account to the restricted shell require OS authority.
func Prepare(dir string, public []byte, role string, port uint) error {
	if !filepath.IsAbs(dir) || filepath.Clean(dir) != dir || port < 1024 || port > 65535 {
		return errors.New("require a new absolute output directory and port 1024..65535")
	}
	line, err := AuthorizedKey(public, role)
	if err != nil {
		return err
	}
	if err := os.Mkdir(dir, 0700); err != nil {
		return err
	}
	for name, data := range map[string]string{"authorized_keys": line + "\n", "sshd_config": SSHDConfig(port)} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(data), 0600); err != nil {
			return err
		}
	}
	return nil
}

// SSHDConfig is the fixed, loopback-only dedicated transport configuration.
func SSHDConfig(port uint) string {
	return fmt.Sprintf(`Port %d
ListenAddress 127.0.0.1
HostKey /etc/urbit-runner/ssh_host_ed25519_key
PidFile /run/urbit-runner-sshd.pid
AuthorizedKeysFile /etc/urbit-runner/authorized_keys
AuthorizedKeysCommand none
AuthorizedPrincipalsFile none
TrustedUserCAKeys none
AllowUsers urbit-runner
AuthenticationMethods publickey
PasswordAuthentication no
KbdInteractiveAuthentication no
PermitEmptyPasswords no
UsePAM yes
PermitRootLogin no
DisableForwarding yes
PermitTTY no
PermitUserEnvironment no
PermitUserRC no
X11Forwarding no
MaxSessions 4
LoginGraceTime 30
ClientAliveInterval 15
ClientAliveCountMax 2
LogLevel VERBOSE
`, port)
}
