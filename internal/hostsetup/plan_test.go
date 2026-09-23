package hostsetup

import (
	"bytes"
	"encoding/json"
	"reflect"
	"slices"
	"strings"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/rootless"
)

func TestPlanStableOrderedJSON(t *testing.T) {
	p, _ := fixture()
	r := p.inspect()
	plan := Plan(r)
	if plan.SchemaVersion != 1 || plan.Deployment != deployment || plan.Account != rootless.AccountName || plan.Home != rootless.HomeDir || plan.DockerVersion != rootless.EngineVersion || !plan.RequiresAuthorization || plan.RuntimeQualified {
		t.Fatalf("unsafe plan metadata: %+v", plan)
	}
	var ids, privileges []string
	for _, step := range plan.Steps {
		ids = append(ids, step.ID)
		privileges = append(privileges, step.Privilege)
	}
	if !reflect.DeepEqual(ids, []string{"review-preflight", "authorize", "install-packages", "verify-artifacts", "dedicated-account", "install-service", "enable-linger", "start-user-engine", "verify-rootless-engine"}) {
		t.Fatalf("unexpected step order: %v", ids)
	}
	if !reflect.DeepEqual(privileges, []string{"none", "none", "root", "root", "root", "root", "root", rootless.AccountName, rootless.AccountName}) {
		t.Fatalf("unexpected privilege boundaries: %v", privileges)
	}
	if !reflect.DeepEqual(plan.Preserves, []string{"existing rootful/system Docker services and sockets", "existing Docker contexts and configuration", "desktop account and group membership", "existing piers and ship data"}) {
		t.Fatalf("preservation contract changed: %v", plan.Preserves)
	}
	data, err := json.Marshal(plan)
	if err != nil {
		t.Fatal(err)
	}
	fields := []string{"schema_version", "deployment", "account", "home", "docker_version", "requires_authorization", "preserves", "steps", "runtime_qualified"}
	previous := -1
	for _, field := range fields {
		index := bytes.Index(data, []byte(`"`+field+`":`))
		if index <= previous {
			t.Fatalf("JSON field absent or unordered: %s", field)
		}
		previous = index
	}
	slices.Reverse(r.Checks)
	for range 3 {
		again, err := json.Marshal(Plan(r))
		if err != nil || !bytes.Equal(data, again) {
			t.Fatalf("plan JSON is unstable: %s (%v)", again, err)
		}
	}
	var decoded SetupPlan
	if err := json.Unmarshal(data, &decoded); err != nil || !reflect.DeepEqual(decoded, plan) {
		t.Fatalf("JSON roundtrip failed: %+v, %v", decoded, err)
	}
}

func TestPlanNeverTrustsReadinessOrInterpolatesInput(t *testing.T) {
	for _, report := range []Report{
		{},
		{SchemaVersion: 999, Deployment: "$(private-injection)", RuntimeQualified: true, Checks: []Check{{"private-id", "pass", "private-secret"}}},
		{Checks: []Check{{"private-id", "unrecognized-status", "private-secret"}}},
	} {
		plan := Plan(report)
		if !plan.RequiresAuthorization || plan.RuntimeQualified || plan.Deployment != deployment || len(plan.Steps) != 9 {
			t.Fatalf("unsafe plan: %+v", plan)
		}
		data, err := json.Marshal(plan)
		if err != nil {
			t.Fatal(err)
		}
		for _, forbidden := range []string{"private-injection", "private-id", "private-secret", "unrecognized-status", "999"} {
			if bytes.Contains(data, []byte(forbidden)) {
				t.Fatalf("report input interpolated: %s", data)
			}
		}
		if !strings.Contains(plan.Steps[0].Description, "absent evidence remains unknown") {
			t.Fatal("missing evidence treated as ready")
		}
	}
	first := Plan(Report{})
	first.Steps[0].Description = "changed"
	first.Preserves[0] = "changed"
	second := Plan(Report{})
	if second.Steps[0].Description == "changed" || second.Preserves[0] == "changed" {
		t.Fatal("plans share mutable state")
	}
}

func TestPlanEnablesDedicatedUserServiceForReboot(t *testing.T) {
	for _, step := range Plan(Report{}).Steps {
		if step.ID != "start-user-engine" {
			continue
		}
		if step.Privilege != rootless.AccountName {
			t.Fatalf("service enable/start runs outside dedicated account: %s", step.Privilege)
		}
		for _, required := range []string{
			"Enable and start only " + rootless.UnitName,
			"for " + rootless.AccountName + " in its systemd user manager",
			"reboot startup with lingering",
			"do not enable it globally for other users",
		} {
			if !strings.Contains(step.Description, required) {
				t.Fatalf("missing service startup requirement %q: %s", required, step.Description)
			}
		}
		return
	}
	t.Fatal("missing dedicated-user service enable/start step")
}
