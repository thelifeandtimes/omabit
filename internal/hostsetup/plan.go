package hostsetup

import (
	"fmt"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/rootless"
)

type Step struct {
	ID          string `json:"id"`
	Privilege   string `json:"privilege"`
	Description string `json:"description"`
}

type SetupPlan struct {
	SchemaVersion         int      `json:"schema_version"`
	Deployment            string   `json:"deployment"`
	Account               string   `json:"account"`
	Home                  string   `json:"home"`
	DockerVersion         string   `json:"docker_version"`
	RequiresAuthorization bool     `json:"requires_authorization"`
	Preserves             []string `json:"preserves"`
	Steps                 []Step   `json:"steps"`
	RuntimeQualified      bool     `json:"runtime_qualified"`
}

// Plan is descriptive, never executable or an authorization decision. All steps
// remain necessary: a report is partial, untrusted evidence, not install state.
func Plan(report Report) SetupPlan {
	unresolved := 0
	for _, check := range report.Checks {
		if check.Status != "pass" && check.Status != "info" {
			unresolved++
		}
	}
	return SetupPlan{
		SchemaVersion: 1, Deployment: deployment, Account: rootless.AccountName,
		Home: rootless.HomeDir, DockerVersion: rootless.EngineVersion,
		RequiresAuthorization: true,
		Preserves:             []string{"existing rootful/system Docker services and sockets", "existing Docker contexts and configuration", "desktop account and group membership", "existing piers and ship data"},
		Steps: []Step{
			{"review-preflight", "none", fmt.Sprintf("Review preflight evidence (%d unresolved reported checks); absent evidence remains unknown. Resolve blocked/unknown prerequisites and unsupported platforms before setup. No automatic rootful fallback.", unresolved)},
			{"authorize", "none", "Review this non-executable plan and obtain explicit OS authorization before host changes. The development-only install-rootless helper requires --acknowledge-host-changes and elevation; it refuses existing resources and is not an updater or production package installer."},
			{"install-packages", "root", "Use configured Omarchy repositories for package-managed docker, rootlesskit, slirp4netns, uidmap helpers (newuidmap/newgidmap/getsubids), iptables, systemd and account tools; do not install a second static engine or enable system Docker."},
			{"verify-artifacts", "root", "Verify package integrity, helper privileges and Docker upstream version " + rootless.EngineVersion + "; verify the reviewed pinned rootless launcher source and digest before installation. Stop on mismatch."},
			{"dedicated-account", "root", "Create a locked dedicated system account " + rootless.AccountName + " with private home " + rootless.HomeDir + ", no wheel/docker membership, and allocated non-overlapping contiguous subuid/subgid ranges of at least 65536 IDs each; review existing accounts rather than repurpose them automatically."},
			{"install-service", "root", "Install the verified launcher root-owned under " + rootless.InstallDir + " outside plugin checkouts and a root-owned systemd user unit " + rootless.UnitName + "; explicitly use the dedicated account's private /run/user/<uid>/" + rootless.SocketName + " without inherited Docker overrides."},
			{"enable-linger", "root", "Enable lingering and dedicated-account user-manager startup only after authorization; preserve existing system Docker and desktop settings."},
			{"start-user-engine", rootless.AccountName, "Enable and start only " + rootless.UnitName + " for " + rootless.AccountName + " in its systemd user manager, enabling reboot startup with lingering; do not enable it globally for other users or connect desktop clients to its engine socket."},
			{"verify-rootless-engine", rootless.AccountName, "Separately verify the explicit private engine endpoint, rootless mode, account/socket isolation, cgroup resource limits and unattended restart/reboot behavior. Stop on failure; runtime and disposable-ship qualification remain outstanding."},
		},
	}
}
