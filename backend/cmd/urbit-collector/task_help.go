package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

type helpCommand struct {
	Name   string `json:"name"`
	Effect string `json:"effect"`
}

type helpTopic struct {
	Name     string        `json:"name"`
	Summary  string        `json:"summary"`
	Commands []helpCommand `json:"commands,omitempty"`
	Guidance []string      `json:"guidance,omitempty"`
}

type helpDocument struct {
	SchemaVersion int         `json:"schema_version"`
	Notice        string      `json:"notice"`
	Topics        []helpTopic `json:"topics,omitempty"`
	Topic         *helpTopic  `json:"topic,omitempty"`
	Next          []string    `json:"next"`
}

// This is navigation for this client, never a claim about a selected runner.
// Keep command flags in the actual handlers' --help, where they are validated.
func taskHelpTopics() []helpTopic {
	return []helpTopic{
		{
			Name: "runners", Summary: "Select a runner and inspect its identity, authority and fleet.",
			Commands: []helpCommand{
				{"runners", "Read saved runner connections locally."},
				{"hello", "Contact one selected runner for identity, mode, role and features."},
				{"snapshot", "Read the selected runner's instances and operation receipts."},
				{"inventory", "Contact all enrolled runners; watch reports each independently."},
				{"enroll", "Verify and save a new runner connection."},
				{"runner-edit", "Verify the same runner at a new endpoint and update local enrollment."},
				{"forget", "Remove local enrollment; does not stop its ships."},
			},
			Guidance: []string{
				"Start with --config-dir REGISTRY runners. Then put --config-dir REGISTRY --runner NAME --expected-runner-id ID before hello and snapshot, using the saved runner_id.",
				"Unreachable is not stopped. A desktop profile does not isolate the runner's ships. Honor excluded ships and hosts without probing them.",
				"An identity mismatch requires resolving the intended runner; do not overwrite enrollment to suppress it. hello, not this guide, establishes supported features and authority.",
			},
		},
		{
			Name: "ships", Summary: "Observe and operate ships managed by a selected runner.",
			Commands: []helpCommand{
				{"snapshot", "Read desired/observed state, revisions, recovery status and operation receipts."},
				{"watch", "Poll snapshots until canceled; JSON output is NDJSON."},
				{"logs", "Read a bounded runtime log tail with operator authority."},
				{"login-code", "Reveal a secret only for an explicitly selected current runtime and authorized sign-in task."},
				{"operate", "Submit an explicit start, stop, maintenance or import-recovery operation."},
				{"configure", "Change requested runtime settings at an expected revision."},
				{"create-fake", "Register a fresh fake; boot occurs on explicit Start."},
				{"create-ship", "Stage a keyed identity and boot material; boot occurs on explicit Start."},
				{"create-comet", "Register a new comet configuration; mining/networking occurs on explicit Start."},
			},
			Guidance: []string{
				"Pin --runner NAME --expected-runner-id ID and inspect hello/snapshot. Bind actions to instance ID and current config_revision; the instance name need not be the Urbit identity.",
				"Read operate --help, then use operate INSTANCE start|stop --revision N --request-id ID for the authorized action. Submission success is not completion: follow that exact operation in snapshot until terminal, then check resulting state.",
				"After a revision conflict, refresh and review the intended effect. After a lost acknowledgement, retain the original request ID and reconcile its receipt; do not guess by retrying with a new ID.",
				"Current managed fakes have container-local Ames. A common project directory does not connect them; project networking remains experimental until the runner exposes that capability.",
				"If an authorized sign-in requires login-code, first select config_revision and resources.container_id from a fresh running snapshot. Completed Start can precede resource observation; wait for that selection instead of supplying a remembered container ID. Read login-code --help before revealing a secret.",
				"Logs may contain sensitive data. Ordinary status does not require login-code or credentials. For hosted Eyre sessions, see help connections.",
			},
		},
		{
			Name: "projects", Summary: "Review fresh fake-galaxy projects, run explicit lifecycle actions and read saved operation history.",
			Commands: []helpCommand{{"private-chain-inputs", "Verify a local sealed dependency generation against separately reviewed manifest/foundation hashes; no execution or runtime authority."}, {"private-chain-register", "Reserve a reviewed private-chain plan with a pinned operator runner; no keys, chain or ships created."}, {"private-chain-projects", "Read saved private-chain reservations with observer access; no live probe."}, {"private-chain-plan", "Review a distinct private local chain/genesis/contracts plan offline; runner private-chain lifecycle is not implemented."}, {"project-plan", "Build a deterministic offline plan and review digest; no runner connection or resource changes."}, {"project-register", "Reserve an explicitly reviewed complete fresh-fake plan on a pinned operator runner; starts nothing."}, {"projects", "Read selected runner reservations; no live runtime inspection or mutation."}, {"project-status", "Read one project's recent saved operation receipts; recorded status is not current runtime health."}, {"project-recovery-source", "Read saved recovery history and exact creator/readiness references; no runtime probe."}, {"project-recovery-checkpoint", "Read an exact saved recovery checkpoint and digests; hints grant no action permission."}, {"project-operate", "Explicit project/member lifecycle actions; each requires its capability and reviewed inputs."}, {"project-recover", "Explicit coordinated recovery, checkpoint observation or continuation; each requires operator access and exact reviewed inputs."}, {"hello", "Inspect a selected runner's advertised capabilities separately."}},
			Guidance: []string{"private-chain-inputs --help describes local read-only input verification with no runner selection. A matching hash is not publisher authentication or permission to initialize a chain. Current gateway inputs remain guest-only qualification candidates.", "private-chain-plan --help describes the separate local chain1337 ~zod/~nec review. Its public foundation metadata is not a key or database. It does not contact a runner, generate keys, reserve resources or launch anything; project-register does not accept it. private-chain-register requires its own capability, matching --plan/--review-digest/--request-id and --confirm-reservation; after a lost reply preserve inputs and use --replay-only. private-chain-projects reads saved reservations. Neither command grants runtime authority.", "Use project-plan --help to specify every fresh instance, canonical galaxy and runner-loopback HTTP port. The initial complete member set determines internal ports.", "Planning does not inspect host reservations, reserve ports or start ships. A review digest is not a project ID or runtime authority; do not assume a selected runner supports project operations.", "Use the intended pinned runner with projects to read saved reservation claims. project-reservations-v1 advertises this read only; it does not imply launch or recovery support. Runtime observation remains explicitly not-performed.", "Use project-status PROJECT with the same runner pin for recent operation history. It requires project-saved-status-v1. An interrupted receipt needs explicit recovery; reading it never repeats an action. has_older_operations means the returned 128 receipts are not the whole history.", "project-register requires project-registration-v1, the saved offline JSON plan, its review digest, a caller-chosen request ID and --confirm-reservation. It reserves membership/ports only. Keep the original inputs and use --replay-only after a lost reply; a missing receipt does not authorize a new request.", "project-operate exposes the actions listed by --help. Each uses the original actor/request/revision/digest and explicit --confirm-action; lookup-only replay starts no worker. Observe initial readiness after the complete member set is launched. Stop/exit/retire/resume/current observations require the exact --after receipt; observe-current also retains --original-readiness. Coordinated recovery uses the separate project-recover command.", "project-recovery-checkpoint PROJECT RECOVERY_OPERATION reads a saved coordinated recovery parent with observer access. It exposes exact history/checkpoint digests and a non-authorizing next-step hint. Unknown Create remains review-required; reading never probes Docker, adopts a container or repeats an effect.", "project-recover PROJECT start binds a new recovery to the saved project-recovery-source history; it may interrupt all members. For an interrupted parent, observe or continue binds to its exact project-recovery-checkpoint history, sequence and digest. Each requires its capability, request ID and --confirm-action. Lookup-only replay creates no worker; never replace preserved inputs or repeat unknown Create.", "Project members share administrative loopback access. Standalone fake directories remain separate from this explicitly reviewed project topology."},
		},
		{
			Name: "imports", Summary: "Review a stopped pier, import it and resolve retained recovery state.",
			Commands: []helpCommand{
				{"piers", "Inspect pier discovery on the selected runner."},
				{"preview-pier", "Inspect a stopped source and derive identity, destination and fingerprint."},
				{"import-pier", "Move/copy the reviewed source into managed storage; completion stays stopped."},
				{"snapshot", "Read import progress, terminal result and retained recovery evidence."},
				{"operate", "Finish retained enrollment with recover-import, or Start after reviewing stopped completion."},
				{"preview-import-restore", "Review restoring a held source and retaining a partial destination."},
				{"restore-import", "Apply the approved restoration and retire the failed registration without deleting either copy."},
				{"enroll-pier", "Register a discovered stopped pier or relink a moved registered pier."},
				{"move-pier", "Move a stopped registered pier without replacing a destination."},
			},
			Guidance: []string{
				"Pin the selected runner and use preview-pier --help. Source paths belong to that runner host, not the collector machine. Inspect only the authorized stopped source.",
				"Review the derived instance/ship/network, fingerprint, exclusive instance parent and named pier child. Use import-pier --help for the reviewed inputs and --confirm-stopped; that flag does not stop a running ship.",
				"Preserve a rejected source. Use its quoted entry/check to diagnose ownership, links, mounts or file types; do not recursively chmod/chown or delete lock files as a repair.",
				"Follow the exact operation receipt. Retained-source or recovery-required state needs its specific recovery action and current revision; do not delete the source, overwrite a destination or restart blindly.",
				"For an interrupted partial copy, require restore-import capability and read preview-import-restore INSTANCE --revision N. Review both moves, then use restore-import --help for digest-bound confirmation. A terminal failed attempt needs a new explicit continuation request; replaying its old ID only retrieves its receipt.",
				"Successful import stays stopped and may have identity_pending. Explicit Start is a separate operation that verifies the running identity.",
			},
		},
		{
			Name: "connections", Summary: "Sign in to an Eyre ship or approve an independent plugin session.",
			Commands: []helpCommand{
				{"ships", "Read locally enrolled Eyre connections; no runner is required."},
				{"ship-enroll", "Save an origin-bound Eyre connection and expected ship identity."},
				{"ship-login", "Read +code from stdin, verify identity and save a session."},
				{"ship-status", "Check the enrolled ship/session against Eyre."},
				{"ship-logout", "Revoke the selected collector session."},
				{"app-plugins", "Inspect locally installed connection-helper declarations without executing them."},
				{"app-connections", "Read durable application attempts locally."},
				{"app-connect", "Issue and hand off an independent session to the exact approved plugin/profile."},
				{"app-disconnect", "Revoke only the selected application attempt's session."},
				{"app-recover", "Recover an interrupted attempt without retrying issuance or delivery."},
				{"app-reconcile", "Clean the selected receiver state after confirmed revocation."},
				{"app-archive", "Archive a fully closed/reconciled attempt while retaining its replay receipt."},
				{"app-connection", "Read one active or archived request locally."},
			},
			Guidance: []string{
				"Use --config-dir REGISTRY, without --runner or --socket. Hosted ships work independently of runner enrollment. Put the login code on stdin with ship-login NAME --code-stdin; never in arguments or reports.",
				"Before app-connect, read ships, app-plugins and app-connections. Its --help specifies the exact ship ID/revision, plugin/profile/artifact, request ID and broad-session acknowledgement.",
				"Each plugin receives a separate ordinary Eyre session. Eyre does not isolate desks. Plugins send data directly to the ship and retain standalone URL/+code sign-in; Omabit is not their data proxy.",
				"On uncertain results, inspect the durable request before recovery. Do not replay approval. Disconnect/recover/reconcile require the attempt's current revision; archive retains permanent replay protection.",
			},
		},
		{
			Name: "hosting", Summary: "Inspect runner hosting evidence and separate repair from observation.",
			Commands: []helpCommand{
				{"hello", "Read the selected runner's capabilities and access role."},
				{"host-check", "Read cached native-package and live private-engine compatibility evidence."},
				{"snapshot", "Read runtime, operation and recovery status."},
			},
			Guidance: []string{
				"Use host-check --help with a pinned runner. The collector needs no local Docker access. A successful report does not itself qualify hosting or repair a failed check.",
				"Installation, local package verification/repair and qualification are separate urbit-runner commands with different effects; read that binary's help before using them.",
				"Qualification creates disposable runtime resources. Use only the authorized host/test boundary; do not substitute the active engine or inspect excluded ships. A profile alone does not isolate host services.",
			},
		},
	}
}

func runTaskHelp(args []string, format string, out io.Writer, fail func(error, int) int) int {
	if len(args) > 1 {
		return fail(errors.New("use help [TOPIC]; put --format json|text before help"), 2)
	}
	doc := helpDocument{SchemaVersion: 1, Notice: "Offline client guidance only: no registry, credentials or runner was inspected. Use hello for actual runner capabilities and authority."}
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		for _, topic := range taskHelpTopics() {
			doc.Topics = append(doc.Topics, helpTopic{Name: topic.Name, Summary: topic.Summary})
		}
		doc.Next = []string{"urbit-collector --format text help TOPIC", "urbit-collector help TOPIC", "urbit-collector COMMAND --help", "urbit-collector --help"}
	} else {
		for _, topic := range taskHelpTopics() {
			if topic.Name == args[0] {
				doc.Topic = &topic
				break
			}
		}
		if doc.Topic == nil {
			return fail(errors.New("unknown help topic; use help to list topics"), 2)
		}
		doc.Next = []string{"urbit-collector COMMAND --help", "urbit-collector help"}
	}
	var err error
	if format == "json" {
		encoder := json.NewEncoder(out)
		encoder.SetIndent("", "  ")
		err = encoder.Encode(doc)
	} else {
		var text strings.Builder
		fmt.Fprintln(&text, "Usage: urbit-collector [--format json|text] help [TOPIC]")
		fmt.Fprintln(&text, "\n"+doc.Notice)
		for _, topic := range doc.Topics {
			fmt.Fprintf(&text, "\n  %-12s %s", topic.Name, topic.Summary)
		}
		if topic := doc.Topic; topic != nil {
			fmt.Fprintf(&text, "\n%s: %s\n", topic.Name, topic.Summary)
			for _, command := range topic.Commands {
				fmt.Fprintf(&text, "\n  %-17s %s", command.Name, command.Effect)
			}
			for _, guidance := range topic.Guidance {
				fmt.Fprintln(&text, "\n\n"+guidance)
			}
		}
		fmt.Fprintln(&text, "\n\nNext:")
		for _, next := range doc.Next {
			fmt.Fprintln(&text, "  "+next)
		}
		_, err = io.WriteString(out, text.String())
	}
	if err != nil {
		return fail(err, 1)
	}
	return 0
}
