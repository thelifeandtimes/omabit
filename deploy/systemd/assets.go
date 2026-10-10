// Package serviceunits provides the same reviewed units for CLI installation and
// source-checkout deployment. The privileged installer does not execute bundle code.
package serviceunits

import "embed"

//go:embed urbit-runner.service urbit-runner-ssh.service urbit-docker.service.d/shutdown.conf user-manager-shutdown.conf
var Files embed.FS
