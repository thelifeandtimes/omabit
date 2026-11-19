// Package checkplan exposes finite local-owner checkpoint plans and explicit
// digest-bound historical report summaries. Discovery never executes a test.
package checkplan

import (
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/rootless"
	candidate "git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimecandidate"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

func LegacyRuntime() map[string]any {
	return map[string]any{
		"profile": runtimequalify.ProfileLifecycle,
		"mode":    "disposable-fake-only", "account": rootless.AccountName, "home": rootless.HomeDir,
		"engine_version": rootless.EngineVersion, "image": candidate.Image, "manifest_digest": candidate.ManifestDigest,
		"config_digest": candidate.ConfigDigest, "platform": candidate.Platform, "vere_version": candidate.VereVersion,
		"pill_default_url": candidate.PillDefaultURL, "pill_generation": candidate.PillGeneration,
		"pill":                        candidate.Artifact{URL: candidate.PillURL, SHA256: candidate.PillSHA256, Bytes: candidate.PillBytes},
		"pill_pin_basis":              "Observed HTTPS content SHA-256 on 2026-09-23; not an independent publisher signature",
		"minimum_available_ram_bytes": int64(5 << 30), "minimum_available_disk_bytes": int64(8 << 30),
		"container_memory_bytes": int64(4 << 30), "container_swap_bytes": 0, "cpus": 2, "pids": 256, "loom_exponent": 31,
		"network": "none; no published ports", "control_socket": "disabled with --no-conn", "runtime_qualified": false,
		"steps":           []string{"engine verification", "pinned image pull and identity check", "verified upstream URL pill", "live cgroup and Vere version probe", "fake boot and graceful stop", "replacement-container restart", "offline pack", "post-pack restart", "owned-container cleanup; retain report and pier"},
		"remaining_gates": []string{"desk inventory/application health", "Ames and host HTTP forwarding", "meld", "reboot", "failure recovery and production runner"},
	}
}

func LegacyHTTP() map[string]any {
	plan := LegacyRuntime()
	plan["profile"] = runtimequalify.ProfileHTTP
	plan["mode"] = "disposable-fake-host-http"
	plan["network"] = "one run-owned IPv4 NAT bridge with normal outbound access; only 127.0.0.1:ephemeral -> 8080/tcp is published"
	plan["host_probe"] = "credential-free HTTP 200 with public identity ~zod at /~/host; no proxies or redirects"
	plan["steps"] = []string{"engine verification", "pinned image and pill", "isolated limits/version probe", "dedicated NAT bridge creation", "fake boot and host HTTP identity", "replacement and rediscovered host HTTP endpoint", "owned-container and network cleanup; retain data/report"}
	plan["remaining_gates"] = []string{"application authentication and desk inventory", "Ames and remote/tailnet connectivity", "meld", "reboot", "failure recovery and production runner"}
	return plan
}
