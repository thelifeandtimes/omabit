package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/rootless"
	candidate "git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimecandidate"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

func runtimePlan() map[string]any {
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

func httpPlan() map[string]any {
	plan := runtimePlan()
	plan["profile"] = runtimequalify.ProfileHTTP
	plan["mode"] = "disposable-fake-host-http"
	plan["network"] = "one run-owned IPv4 NAT bridge with normal outbound access; only 127.0.0.1:ephemeral -> 8080/tcp is published"
	plan["host_probe"] = "credential-free HTTP 200 with public identity ~zod at /~/host; no proxies or redirects"
	plan["steps"] = []string{"engine verification", "pinned image and pill", "isolated limits/version probe", "dedicated NAT bridge creation", "fake boot and host HTTP identity", "replacement and rediscovered host HTTP endpoint", "owned-container and network cleanup; retain data/report"}
	plan["remaining_gates"] = []string{"application authentication and desk inventory", "Ames and remote/tailnet connectivity", "meld", "reboot", "failure recovery and production runner"}
	return plan
}

func meldPlan() map[string]any {
	plan := runtimePlan()
	plan["profile"] = runtimequalify.ProfileMeld
	plan["mode"] = "disposable-fake-offline-meld"
	plan["management"] = "management must be stopped; its process lock is held for the entire run, including cleanup"
	plan["utility"] = "/bin/urbit meld --loom 31 /urbit/fake-zod"
	plan["measurements"] = "fresh admission before meld; before/after logical regular-file pier bytes, not allocated storage or loom utilization"
	plan["failure_policy"] = "no retry or post-meld boot on uncertain utility outcome; cleanup does not clear pier quarantine"
	plan["steps"] = []string{"engine verification", "pinned image and pill", "live cgroup and Vere version probe", "fresh isolated fake boot and clean stop", "fresh resource admission and measured offline meld", "same-pier replacement restart and clean stop", "owned-container cleanup; retain report and pier"}
	plan["remaining_gates"] = []string{"managed meld admission/replay/recovery", "larger-pier memory budgets", "live failure/reboot qualification", "epoch retention", "scheduling", "production runtime"}
	return plan
}

func epochsPlan() map[string]any {
	plan := runtimePlan()
	plan["profile"] = runtimequalify.ProfileEpochs
	plan["mode"] = "disposable-fake-epoch-lifecycle"
	plan["management"] = "management must be stopped; its process lock is held through cleanup"
	plan["utility"] = "/bin/urbit roll --loom 31 /urbit/fake-zod (three explicit invocations)"
	plan["measurements"] = "epoch inventories before/after each roll and restart; oldest-only preview retains three newer epochs"
	plan["failure_policy"] = "no retry or restart after an uncertain roll; zero exit without a new epoch fails the gate"
	plan["steps"] = []string{"engine verification", "pinned image and pill", "live limits/version probe", "fresh isolated fake boot and clean stop", "verify initial epoch 0i0", "three offline rolls, each with fresh admission, inventory checks and same-pier restart", "preview selecting only the oldest epoch, retaining three newer epochs; no deletion", "owned-container cleanup; retain report and multi-epoch pier"}
	plan["remaining_gates"] = []string{"exact selected-prefix deletion", "retained-suffix replay/recovery", "interrupted deletion recovery", "managed roll and pruning", "production runtime"}
	return plan
}

func prunePlan() map[string]any {
	plan := epochsPlan()
	plan["profile"] = runtimequalify.ProfilePrune
	plan["mode"] = "destructive-disposable-fake-only"
	plan["utility"] = "three offline rolls, then /bin/urbit play --full --loom 31 /urbit/fake-zod"
	plan["measurements"] = "revalidated oldest-only manifest; retained epoch fingerprints; checkpoint absence/restoration and HTTP restarts"
	plan["failure_policy"] = "partial deletion or interrupted recovery stays quarantined; no automatic retry or rollback"
	plan["steps"] = []string{"pinned engine/image/pill and resource verification", "fresh isolated fake with three qualified roll/restart pairs", "revalidate exact oldest-only preview against the stopped pier", "durably journal and unlink only epoch 0i0 files/directory; verify three retained epoch fingerprints", "ordinary same-pier restart and clean stop", "journal and remove only the disposable current checkpoint", "pinned play --full restores from the latest retained epoch; strict runtime replay checks", "post-recovery HTTP restart and retained epoch-set verification", "owned-container cleanup; retain report and three-epoch pier"}
	plan["remaining_gates"] = []string{"managed pruning", "older-retained-epoch recovery", "live interrupted-deletion recovery", "backup integrity", "production runtime"}
	return plan
}

func prefixPlan() map[string]any {
	plan := prunePlan()
	plan["profile"] = runtimequalify.ProfilePrefix
	plan["mode"] = "destructive-disposable-repeated-prefix-only"
	plan["utility"] = "four offline rolls; two pinned play --full recovery checks"
	plan["measurements"] = "five epochs -> delete oldest two -> retain three -> delete oldest remaining -> retain newest two; exact manifests and fingerprints"
	plan["steps"] = []string{"fresh isolated fake with pinned engine/image/pill and resource checks", "four explicit roll/restart pairs create five epochs", "select and remove genesis plus the first snapshot-bearing epoch; retain three", "ordinary restart, controlled checkpoint loss, latest-epoch replay and restart", "fresh selection removes the oldest remaining snapshot-bearing epoch; retain newest two", "repeat ordinary restart, checkpoint loss, latest-epoch replay and restart", "remove sixteen owned containers; retain report and two-epoch pier"}
	plan["remaining_gates"] = []string{"managed broader-prefix integration", "older-epoch recovery", "live interrupted-deletion recovery", "production runtime"}
	return plan
}

func importPlan() map[string]any {
	plan := runtimePlan()
	plan["profile"] = runtimequalify.ProfileImport
	plan["mode"] = "disposable-fake-pier-copy"
	plan["management"] = "management must be stopped; its lock is held through transfer, both resumes and cleanup"
	plan["measurements"] = "SHA-256 content/tree digest, file/byte counts, exact fake identity, and unchanged original after two copied-pier resumes"
	plan["failure_policy"] = "partial copy or uncertain resume stays quarantined; no overwrite, retry, source restart, or recursive deletion"
	plan["steps"] = []string{"verify pinned engine/image/pill and live limits", "boot a fresh isolated fake, verify ~zod and cleanly stop/remove its container", "require source logical size plus 8 GiB disk; reserve original under source/", "copy into fresh data/ with owner-only permissions, strict tree bounds and content verification", "resume the copied pier without bootstrap flags; verify ~zod and cleanly stop", "replace its runtime and verify another initialized-pier resume", "verify stopped original content unchanged; remove all four containers; retain both piers and report"}
	plan["remaining_gates"] = []string{"managed import staging and acceptance", "external-pier format/identity validation", "key/comet bootstrap", "Ames networking", "routine maintenance", "production runtime"}
	return plan
}

func cometPlan() map[string]any {
	plan := httpPlan()
	plan["profile"] = runtimequalify.ProfileComet
	plan["mode"] = "fresh-disposable-comet"
	plan["management"] = "management must be stopped; its lock is held throughout provisioning, resume and cleanup"
	plan["network"] = "run-owned NAT bridge with normal outbound access (including Ames); only ephemeral loopback HTTP published; no host Ames/UDP publication"
	plan["host_probe"] = "credential-free /~/host must report a comet-shaped identity and retain that exact identity on replacement"
	plan["failure_policy"] = "no creation retry; partial/new identity state stays quarantined even after resource cleanup"
	plan["steps"] = []string{"verify pinned engine/image/pill and live container limits", "create owned NAT bridge", "recheck resource admission; generate a fresh comet using -c, with no supplied identity or key", "observe public comet identity via inspected host-loopback HTTP; cleanly stop", "resume same pier without bootstrap flags or pill mount; verify unchanged identity and clean stop", "remove all three containers and bridge; retain private comet pier and report"}
	plan["remaining_gates"] = []string{"managed comet provisioning", "Ames peer connectivity and inbound/tailnet publication", "key-based bootstrap", "external-pier import", "routine maintenance", "production runtime"}
	return plan
}

func cometMaintenancePlan() map[string]any {
	p := cometPlan()
	p["profile"] = runtimequalify.ProfileCometMaintenance
	p["mode"] = "fresh-disposable-comet-maintenance"
	p["utility"] = "/bin/urbit pack|meld --loom 31 /urbit/comet; utilities run with network=none"
	p["measurements"] = "fresh RAM/disk admission, logical pier sizes before/after each utility, and exact public comet identity after each restart"
	p["steps"] = []string{"verify pinned engine/image/pill and live limits", "create owned NAT bridge and generate a fresh disposable comet", "cleanly stop; run isolated offline pack and verify unchanged-identity restart", "cleanly stop; run isolated offline meld and verify unchanged-identity restart", "verify an additional initialized-pier replacement resume", "remove all seven containers and bridge; retain private pier and outcome report"}
	p["remaining_gates"] = []string{"managed comet pack/meld and routine integration", "comet roll/retention/recovery", "key bootstrap and external import", "host Ames/tailnet publication and peering", "production runtime"}
	return p
}

func cometEpochsPlan() map[string]any {
	p := cometPlan()
	p["profile"] = runtimequalify.ProfileCometEpochs
	p["mode"] = "fresh-disposable-comet-epochs"
	p["utility"] = "/bin/urbit roll --loom 31 /urbit/comet (three network-isolated invocations)"
	p["measurements"] = "bounded epoch inventories before/after each roll and exact-identity restart; four epochs, preview retains three; no deletion"
	p["failure_policy"] = "no retry or restart after uncertain roll; missing snapshot or changed identity quarantines the run even after cleanup"
	p["steps"] = []string{"verify pinned engine/image/pill and live limits", "create owned NAT bridge and a fresh disposable comet; verify public identity and clean stop", "record canonical initial epoch 0i0", "three isolated offline rolls, each with fresh admission, exactly one new snapshot-bearing epoch and same-identity restart", "final initialized-pier replacement; recheck four epochs and preview oldest-only selection retaining three", "remove all nine containers and bridge; retain private pier and report"}
	p["remaining_gates"] = []string{"managed comet epoch inspection and roll", "comet prefix pruning and latest-epoch recovery", "key bootstrap and external import", "host Ames/tailnet publication and peering", "production runtime"}
	return p
}

func cometPrefixPlan() map[string]any {
	p := cometEpochsPlan()
	p["profile"] = runtimequalify.ProfileCometPrefix
	p["mode"] = "destructive-fresh-disposable-comet-prefix"
	p["utility"] = "four network-isolated rolls; two /bin/urbit play --full --loom 31 /urbit/comet recovery checks"
	p["measurements"] = "five -> three -> two epochs; exact prefix unlink manifests, unchanged retained metadata, restored checkpoint and same public identity after each restart"
	p["failure_policy"] = "partial deletion, failed recovery or changed identity stays quarantined after cleanup; no automatic retry or rollback"
	p["steps"] = []string{"verify pinned engine/image/pill, live limits, fresh RAM/disk admission and owned NAT bridge", "bootstrap a NEW disposable comet and pin its public identity", "four explicit roll/restart pairs create five epochs", "delete the oldest two epochs (genesis and one snapshot-bearing epoch); verify the three retained manifests", "restart with the same identity, then remove this disposable current checkpoint and restore it with isolated play --full; verify another identity-preserving restart", "delete the oldest remaining epoch, retain the protected newest two, and repeat restart/checkpoint-loss/recovery/restart checks", "final initialized-pier replacement verifies the same identity and two retained epochs", "remove all seventeen containers and bridge; retain private pier/report"}
	p["remaining_gates"] = []string{"managed comet confirmed pruning integration", "older-epoch and live interrupted-deletion recovery", "key bootstrap and external import", "host Ames/tailnet publication and peering", "production runtime"}
	return p
}

func runRuntime(ctx context.Context, args []string, stdout, stderr io.Writer, qualify func(context.Context, func(string)) (runtimequalify.Result, error)) int {
	format := "json"
	fail := func(err error, status int) int {
		if format == "text" {
			fmt.Fprintf(stderr, "Qualification error: %s\n", err)
		} else {
			_ = json.NewEncoder(stderr).Encode(protocol.ErrorResponse{Error: protocol.APIError{Code: "qualification_error", Message: err.Error()}})
		}
		return status
	}
	command := args[0]
	isPlan := command == "runtime-plan" || command == "http-plan" || command == "meld-plan" || command == "epochs-plan" || command == "prune-plan" || command == "prefix-plan" || command == "import-plan" || command == "comet-plan" || command == "comet-maintenance-plan" || command == "comet-epochs-plan" || command == "comet-prefix-plan"
	flags := flag.NewFlagSet(command, flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.Usage = func() {}
	flags.StringVar(&format, "format", "json", "output format: json or text")
	var acknowledged bool
	if !isPlan {
		flags.BoolVar(&acknowledged, "acknowledge-disposable-run", false, "authorize bounded downloads and the selected new disposable-ship workflow")
	}
	if err := flags.Parse(args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			if isPlan {
				fmt.Fprintf(stdout, "Usage: urbit-runner %s [--format json|text]\n\nPrints pinned candidate and checkpoint scope without inspecting the host or contacting Docker.\n", command)
			} else if command == "qualify-comet-prefix" {
				fmt.Fprint(stdout, `Usage: urbit-runner qualify-comet-prefix --acknowledge-disposable-run [--format json|text]

DESTRUCTIVE qualification confined to a NEW DISPOSABLE COMET, never an existing pier.
Management must be stopped. Runtime uses outbound NAT and loopback-only HTTP.
Four isolated rolls/restarts create five epochs. Delete the oldest two, retain three,
verify ordinary restart, then remove this disposable current checkpoint and run
network-isolated play --full to recover from the latest retained epoch. Verify an
identity-preserving restart, delete the oldest remaining epoch (retain newest two),
then repeat ordinary restart, checkpoint-loss/recovery and recovered restart.
A final replacement must preserve the same public identity and two retained epochs.
No existing path, key, identity, cutoff or image override. No managed pruning enabled.
Fresh admission requires 5 GiB available RAM / 8 GiB free disk before expensive work.
Limits: 4 GiB RAM, no extra swap, two CPUs, 256 PIDs, loom 31; 20-minute readiness and
10-minute utility bounds. Partial deletion/recovery stays quarantined without retry.
Seventeen containers and the bridge are removed; the private pier/report remain.
`)
			} else if command == "qualify-comet-epochs" {
				fmt.Fprint(stdout, `Usage: urbit-runner qualify-comet-epochs --acknowledge-disposable-run [--format json|text]

Creates a NEW DISPOSABLE COMET with outbound NAT and loopback-only HTTP.
Management must be stopped. Three network-isolated rolls must each create exactly
one newer snapshot-bearing epoch, with an unchanged-identity restart after each.
A final replacement rechecks all four epochs; an oldest-only preview retains three.
No epoch is deleted. No managed comet roll/retention capability is enabled here.
Fresh admission: 5 GiB available RAM / 8 GiB disk before expensive work.
Limits: 4 GiB RAM, no extra swap, two CPUs, 256 PIDs, loom 31. Readiness is bounded
at 20 minutes and each utility at 10 minutes. No existing path/key/image overrides.
Uncertain rolls or changed identity remain quarantined; no automatic retry.
Nine containers and the bridge are removed; the private pier/report are retained.
`)
			} else if command == "qualify-comet-maintenance" {
				fmt.Fprint(stdout, `Usage: urbit-runner qualify-comet-maintenance --acknowledge-disposable-run [--format json|text]

Creates a NEW DISPOSABLE COMET, with outbound NAT and loopback-only host HTTP.
Management must be stopped. Cleanly stops the comet, runs network-isolated pack,
verifies an unchanged-identity restart, then repeats with meld and one final
replacement resume. Every utility requires fresh 5 GiB available RAM / 8 GiB disk.
Fixed 4 GiB/no-extra-swap/two-CPU/256-PID/loom-31 containers; no path/key overrides.
Creation/readiness have 20-minute bounds; each offline utility has a 10-minute bound.
Uncertain utility/provisioning outcomes stay quarantined, without automatic retry.
Seven containers and the bridge are removed; the private pier/report are retained.
No managed comet maintenance or automatic pruning is enabled by this qualifier.
`)
			} else if command == "qualify-comet" {
				fmt.Fprint(stdout, `Usage: urbit-runner qualify-comet --acknowledge-disposable-run [--format json|text]

Creates a NEW DISPOSABLE COMET using the pinned Vere binary and pill. No existing
pier, supplied identity or key is accepted. Management must be stopped. The comet
can contact the network through its owned NAT bridge. Only host-loopback HTTP is
published; host Ames/UDP, clearweb and remote exposure remain unqualified.
Requires >=5 GiB available RAM and >=8 GiB disk, checked before creation and resume.
Limits: 4 GiB RAM, no extra swap, two CPUs, 256 PIDs, loom 31. Initial readiness and
resume each have a 20-minute bound. Public comet identity must survive replacement.
No automatic creation retry. Partial provisioning remains quarantined after cleanup.
Three containers and the bridge are removed; the private identity-bearing pier
and report are retained. This is development qualification, not production support.
`)
			} else if command == "qualify-import" {
				fmt.Fprint(stdout, `Usage: urbit-runner qualify-import --acknowledge-disposable-run [--format json|text]

DEVELOPMENT stopped-pier transfer qualification on a newly created isolated fake.
Management must be stopped. Boots and cleanly stops a fresh fake, reserves the
original source, copies it into fresh owner-only storage, then resumes the copied
pier twice without creation flags. Source content must remain unchanged.
Tree limits: 8192 entries, depth 32, 100 GB logical bytes; reject symlinks, hardlinked
files, special files, cross-device entries, ownership changes and concurrent writes.
Copy admission requires source logical bytes plus 8 GiB free disk; boot/resume
requires at least 5 GiB available RAM. No existing-pier, path, key or image input.
Four containers are removed; both piers and the report remain for review.
Interrupted copies/resumes remain quarantined; cleanup does not clear uncertainty.
`)
			} else if command == "qualify-prefix" {
				fmt.Fprint(stdout, `Usage: urbit-runner qualify-prefix --acknowledge-disposable-run [--format json|text]

DESTRUCTIVE DEVELOPMENT qualification on a newly created isolated fake only.
Management must be stopped. Four rolls/restarts create five epochs. First deletes
the oldest two (including one snapshot), retaining three. After restart and
latest-epoch checkpoint recovery, deletes the oldest remaining epoch, retaining
the newest two, then repeats restart/recovery checks. Both recoveries deliberately
remove the disposable current checkpoint before pinned play --full.
Requires >=5 GiB available RAM and >=8 GiB free disk before expensive work.
Fixed 4 GiB/no-extra-swap/two-CPU/256-PID containers; loom 31. No existing-pier,
cutoff, image or path override. Uncertain work is quarantined without retry.
Sixteen containers are cleaned up; report and remaining two-epoch pier are kept.
Use --format text for readable results; allow cancellation cleanup to finish.
`)
			} else if command == "qualify-prune" {
				fmt.Fprint(stdout, `Usage: urbit-runner qualify-prune --acknowledge-disposable-run [--format json|text]

DESTRUCTIVE DEVELOPMENT qualification, confined to a newly created isolated fake.
Run as the dedicated account with management stopped. Creates four epochs using
three rolls/restarts, then revalidates and deletes ONLY the oldest epoch 0i0.
Three newer epochs must retain their exact metadata fingerprints. After ordinary
restart, removes the disposable current checkpoint and runs pinned play --full,
which restores from the LATEST retained epoch, followed by another HTTP restart.
Requires 5 GiB available RAM / 8 GiB free disk; fixed 4 GiB/no-extra-swap containers.
No existing-pier, cutoff, path or image override. Partial deletion/recovery stays
quarantined without automatic retries. Owned containers are cleaned up; data and
report are retained. This does not qualify older-epoch recovery or managed prune.
Use --format text for readable progress/results. Let cancellation cleanup finish.
`)
			} else if command == "qualify-epochs" {
				fmt.Fprint(stdout, `Usage: urbit-runner qualify-epochs --acknowledge-disposable-run [--format json|text]

DEVELOPMENT multi-epoch qualification. Run as the dedicated account with management
stopped. Creates a fresh isolated fake, verifies its initial epoch, then performs
three explicit offline rolls, each followed by a same-pier restart and clean stop.
Requires 5 GiB available RAM and 8 GiB disk, rechecked before every roll.
Container limits: 4 GiB RAM, no additional swap, two CPUs, 256 PIDs, loom 31.
Each utility/readiness phase has a ten-minute deadline. Downloads the pinned image
and ~207 MiB pill. Snapshots/epochs and the private report are retained afterward.
An uncertain roll is quarantined; cleanup does not clear it or trigger a retry.
The final preview selects the oldest epoch and retains three; no epoch is deleted.
This does not qualify retained-suffix replay or backup recovery. Use --format text
for readable progress and results. Ctrl+C requests bounded resource cleanup.
`)
			} else if command == "qualify-meld" {
				fmt.Fprint(stdout, `Usage: urbit-runner qualify-meld --acknowledge-disposable-run [--format json|text]

DEVELOPMENT offline-meld checkpoint. Run as the dedicated urbit-runner account,
with management stopped. Holds the management lock through bounded cleanup.
Boots a fresh isolated fake zod, stops/removes its container, rechecks available
RAM/disk, runs pinned offline meld, then restarts the same pier and checks private
HTTP reachability. No existing pier, real identity, published port or control
socket is used. Limits: 4 GiB RAM, no additional swap, two CPUs, 256 PIDs, loom 31.
Requires >=5 GiB available RAM and >=8 GiB disk, including immediately before meld.
Downloads the pinned image and ~207 MiB pill. Meld and each readiness check have
ten-minute deadlines. Ctrl+C requests cleanup, which may take several minutes.
Retains private data and report; uncertain meld stays quarantined after cleanup.
Default output is JSON; --format text provides readable progress and results.
`)
			} else if command == "qualify-http" {
				fmt.Fprint(stdout, `Usage: urbit-runner qualify-http --acknowledge-disposable-run [--format json|text]

DEVELOPMENT host HTTP checkpoint. Run as the dedicated urbit-runner account,
never root. Creates a fresh fake zod, a per-run NAT bridge, and only an
ephemeral IPv4 loopback publication. Verifies public Eyre identity from the host,
replaces the container and reconnects, then cleans up owned containers/network.
The dedicated bridge permits normal outbound access; Ames remains disabled.
No real identity, existing pier, login cookie, Ames port or control socket is used.
Requires >=5 GiB available RAM and >=8 GiB free disk. Downloads ~207 MiB of pill
data plus the pinned image. Progress is JSON lines on stderr; the report is on
stdout and in private storage. Cancellation cleanup can take several minutes.
Use --format text for readable progress and results.
`)
			} else {
				fmt.Fprint(stdout, `Usage: urbit-runner qualify-runtime --acknowledge-disposable-run [--format json|text]

DEVELOPMENT checkpoint. Must execute as the dedicated urbit-runner account,
never as root or the desktop account. Uses only its verified private engine.
Downloads a pinned image and ~200 MiB pill, creates a fresh network-isolated
fake zod, tests limits, boot/restart/offline pack, and cleans up owned containers.
No identity keys, existing piers, public ports, or Vere control socket are used.
Requires >=5 GiB available RAM and >=8 GiB free disk. Data and reports are retained.
Progress is JSON lines on stderr; the final report is JSON on stdout. Ctrl+C
requests cancellation with bounded cleanup, which may take several minutes.
Uncertain or incomplete cleanup requires manual review, not an automatic retry.
Use --format text for readable progress and results.
`)
			}
			return 0
		}
		return fail(err, 2)
	}
	if flags.NArg() != 0 || (!isPlan && !acknowledged) {
		return fail(errors.New("unexpected arguments or missing --acknowledge-disposable-run; review the corresponding plan first"), 2)
	}
	if format != "json" && format != "text" {
		return fail(errors.New("format must be json or text"), 2)
	}
	encoder := json.NewEncoder(stdout)
	encoder.SetIndent("", "  ")
	if isPlan {
		plan := runtimePlan()
		if command == "http-plan" {
			plan = httpPlan()
		} else if command == "meld-plan" {
			plan = meldPlan()
		} else if command == "epochs-plan" {
			plan = epochsPlan()
		} else if command == "prune-plan" {
			plan = prunePlan()
		} else if command == "prefix-plan" {
			plan = prefixPlan()
		} else if command == "import-plan" {
			plan = importPlan()
		} else if command == "comet-plan" {
			plan = cometPlan()
		} else if command == "comet-maintenance-plan" {
			plan = cometMaintenancePlan()
		} else if command == "comet-epochs-plan" {
			plan = cometEpochsPlan()
		} else if command == "comet-prefix-plan" {
			plan = cometPrefixPlan()
		}
		var err error
		if format == "text" {
			err = writeRuntimePlan(stdout, plan)
		} else {
			err = encoder.Encode(plan)
		}
		if err != nil {
			return fail(err, 1)
		}
		return 0
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	var progressErr error
	result, err := qualify(ctx, func(phase string) {
		if progressErr == nil {
			if format == "text" {
				_, progressErr = fmt.Fprintf(stderr, "Phase: %s\n", phase)
			} else {
				progressErr = json.NewEncoder(stderr).Encode(struct {
					Phase string `json:"phase"`
				}{phase})
			}
			if progressErr != nil {
				cancel()
			}
		}
	})
	var writeErr error
	if format == "text" {
		writeErr = writeRuntimeResult(stdout, result)
	} else {
		writeErr = encoder.Encode(result)
	}
	if writeErr != nil {
		err = errors.Join(err, writeErr)
	}
	if err = errors.Join(err, progressErr); err != nil {
		return fail(err, 1)
	}
	return 0
}
