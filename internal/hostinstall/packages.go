package hostinstall

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/hostsetup"
)

// Called only by explicitly authorized owner setup, after validating the bundle,
// key, owner and existing installation and taking the setup lock. The root helper
// recomputes its plan; no desktop-supplied package list crosses this boundary.
func (i *installer) installNativePrerequisites(ctx context.Context, owner string, progress func(string)) error {
	if i.euid != 0 || !i.installPrerequisites || i.inspectOwner == nil {
		return errors.New("native package installation requires authorized owner setup")
	}
	report := i.inspectOwner(owner)
	for _, c := range report.Checks {
		if hostsetup.RequiredSetupCheck(c) && c.Status != "pass" && !hostsetup.InstallableCheck(report, c) {
			return fmt.Errorf("%s: %s", c.ID, c.Detail)
		}
	}
	packages := hostsetup.NativePackages(report)
	if len(packages) == 0 {
		return nil
	}
	// A missing executable in an installed package is damage/configuration, not
	// authority to upgrade/reinstall it. Query the whole local inventory so a
	// failing query cannot be mistaken for an absent package.
	data, err := i.command(ctx, "/usr/bin/pacman", "-Qq")
	if err != nil {
		return fmt.Errorf("read installed native packages: %w", err)
	}
	installed := strings.Fields(string(data))
	for _, name := range packages {
		if slices.Contains(installed, name) {
			return fmt.Errorf("native package %s is installed but a required executable is missing; repair that package before retrying setup", name)
		}
	}
	if progress != nil {
		progress("packages")
	}
	// Use the configured repositories and their signature policy. Never refresh
	// databases without a system upgrade, remove conflicts, or bypass pacman's lock.
	args := append([]string{"-S", "--needed", "--noconfirm", "--"}, packages...)
	i.mutated = true // a failed transaction may already have installed some packages
	if _, err := i.commandWithTimeout(ctx, 10*time.Minute, "/usr/bin/pacman", args...); err != nil {
		return fmt.Errorf("native package installation did not finish: %w. Resolve the package-manager error and retry setup; if mirrors no longer have these builds, update Omarchy first", err)
	}
	// Exit status is not readiness. Re-read actual paths, ownership metadata and
	// engine version before proceeding to authenticated payload verification.
	for _, c := range i.inspectOwner(owner).Checks {
		if hostsetup.RequiredSetupCheck(c) && c.Status != "pass" {
			return fmt.Errorf("native package installation finished but %s still needs attention: %s", c.ID, c.Detail)
		}
	}
	return nil
}
