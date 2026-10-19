package hostsetup

import (
	"fmt"
	"os/user"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/ownerhost"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/rootless"
)

type Step struct {
	ID          string `json:"id"`
	Privilege   string `json:"privilege"`
	Description string `json:"description"`
}

type SetupPlan struct {
	SchemaVersion int    `json:"schema_version"`
	Deployment    string `json:"deployment"`
	Account       string `json:"account"`
	Home          string `json:"home"`
	// DockerVersion is the historical baseline for older clients; current engine
	// admission comes from Report.Compatibility, not exact baseline equality.
	DockerVersion         string   `json:"docker_version"`
	RequiresAuthorization bool     `json:"requires_authorization"`
	Preserves             []string `json:"preserves"`
	Steps                 []Step   `json:"steps"`
	RuntimeQualified      bool     `json:"runtime_qualified"`
}

// Plan is descriptive, never executable or an authorization decision. All steps
// remain necessary: a report is partial, untrusted evidence, not install state.
func Plan(report Report) SetupPlan {
	packages := packageGuidance(report)
	if report.Deployment == "owner-user-rootless" {
		name, home := "selected user", "selected home"
		if u, e := user.Current(); e == nil {
			name, home = u.Username, u.HomeDir
		}
		plan := SetupPlan{SchemaVersion: 1, Deployment: report.Deployment, Account: name, Home: home, DockerVersion: rootless.EngineVersion, RequiresAuthorization: true,
			Preserves: []string{"login account and shell", "personal SSH configuration", "unrelated Docker services and contexts", "existing piers"},
			Steps:     []Step{{"prerequisites", "none", "Review platform, owner and rootless engine prerequisites."}, {"authorize", "root", "Authorize owner-user setup; allocate missing subordinate IDs without replacing existing ranges."}, {"storage", "root", "Create " + ownerhost.DefaultPiers + " owned by " + name + "; do not recursively change an unrelated existing directory."}, {"services", "root", "Install owner-bound rootless Docker, runner and restricted management SSH services; preserve personal SSH and login shell."}, {"verify", name, "Verify the explicit rootless socket, same-UID filesystem access and pinned management identity."}}}
		plan.Steps[0].Description += " " + packages
		return plan
	}
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
			{"install-packages", "root", packages},
			{"verify-artifacts", "root", "Revalidate native package provenance and helper privileges after OS authorization; require a supported Docker Engine API compatibility policy. Package metadata and MD5 alone do not authenticate payloads. Independently verify the pinned rootless launcher source and digest."},
			{"dedicated-account", "root", "Create a locked dedicated system account " + rootless.AccountName + " with private home " + rootless.HomeDir + ", no wheel/docker membership, and allocated non-overlapping contiguous subuid/subgid ranges of at least 65536 IDs each; review existing accounts rather than repurpose them automatically."},
			{"install-service", "root", "Install the verified launcher root-owned under " + rootless.InstallDir + " outside plugin checkouts and a root-owned systemd user unit " + rootless.UnitName + "; explicitly use the dedicated account's private /run/user/<uid>/" + rootless.SocketName + " without inherited Docker overrides."},
			{"enable-linger", "root", "Enable lingering and dedicated-account user-manager startup only after authorization; preserve existing system Docker and desktop settings."},
			{"start-user-engine", rootless.AccountName, "Enable and start only " + rootless.UnitName + " for " + rootless.AccountName + " in its systemd user manager, enabling reboot startup with lingering; do not enable it globally for other users or connect desktop clients to its engine socket."},
			{"verify-rootless-engine", rootless.AccountName, "Separately verify the explicit private engine endpoint, rootless mode, account/socket isolation, cgroup resource limits and unattended restart/reboot behavior. Stop on failure; runtime and disposable-ship qualification remain outstanding."},
		},
	}
}

func packageGuidance(report Report) string {
	text := "Use the host's configured native OS repositories for Docker, rootlesskit, slirp4netns, uidmap helpers, iptables, systemd and account tools; resolve unknown provider evidence before setup."
	if report.Compatibility != nil {
		switch report.Compatibility.Distribution.Provider {
		case "pacman":
			text = "Use configured Pacman repositories for docker, rootlesskit, slirp4netns, shadow (newuidmap/newgidmap/getsubids), iptables and systemd."
		case "deb", "dpkg", "debian", "apt":
			text = "Use configured Debian/Ubuntu APT repositories for a compatible Docker engine (docker.io or docker-ce), rootlesskit, slirp4netns, uidmap, iptables and systemd; review package availability and user-namespace/AppArmor policy."
		}
	}
	return text + " Do not install a second static engine, mutate repositories or enable system Docker. Authorized setup may download missing native authentication artifacts into a private verified cache before executing Docker or changing services. After trusted management activation, host setup installs offline native transaction hooks to refresh authenticated package evidence; verification failures do not abort OS updates or restart the engine."
}
