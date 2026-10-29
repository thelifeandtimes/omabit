//go:build linux && amd64

package runtimequalify

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
)

// Revalidate a retained VM receipt against current pure containment checks.
// This optional offline test reads one supplied file; it makes no engine calls.
func TestProjectRecordedPreparation(t *testing.T) {
	path := os.Getenv("OMABIT_TEST_PROJECT_RECEIPT")
	if path == "" {
		t.Skip("requires a retained disposable VM receipt")
	}
	data, err := os.ReadFile(path)
	if err != nil || len(data) > 1<<20 {
		t.Fatal("missing or excessive retained receipt", err)
	}
	var receipt struct {
		OK             bool                       `json:"ok"`
		Stage          string                     `json:"stage"`
		ShipOperations bool                       `json:"ship_operations"`
		RunnerID       string                     `json:"runner_id"`
		OperationID    string                     `json:"operation_id"`
		Prepared       projectnetwork.Preparation `json:"prepared"`
		Project        struct {
			ID       string              `json:"id"`
			Revision uint64              `json:"revision"`
			Plan     projectnetwork.Plan `json:"plan"`
		} `json:"project"`
		Journal struct {
			Preparation *projectnetwork.Preparation `json:"preparation"`
		} `json:"journal"`
		Bridge []bridge    `json:"bridge_inspection"`
		Anchor []container `json:"anchor_inspection"`
	}
	if json.Unmarshal(data, &receipt) != nil || !receipt.OK || receipt.Stage != "complete" || receipt.ShipOperations || len(receipt.Bridge) != 1 || len(receipt.Anchor) != 1 || receipt.Journal.Preparation == nil {
		t.Fatal("not a complete bounded preparation receipt")
	}
	c, err := newProjectResourceContract(projectSelection{RunnerID: receipt.RunnerID, ProjectID: receipt.Project.ID, Revision: receipt.Project.Revision, OperationID: receipt.OperationID, Generation: 1, Plan: receipt.Project.Plan}, receipt.Prepared.EngineID, receipt.Anchor[0].Image)
	if err != nil || !reflect.DeepEqual(c.Preparation, receipt.Prepared) {
		t.Fatal("recorded contract differs from current fixed creation arguments", err)
	}
	p := receipt.Journal.Preparation
	if p.Validate(c.Selection.Plan) != nil || p.Sequence != 6 || p.Bridge.Phase != "observed" || p.Anchor.Phase != "observed" {
		t.Fatal("recorded resource preparation incomplete")
	}
	if err = c.validateBridge(receipt.Bridge[0], p.Bridge.ID, p.Anchor.ID); err != nil {
		t.Fatal("recorded bridge rejected", err)
	}
	if err = c.validateStoppedAnchor(receipt.Anchor[0], p.Anchor.ID, p.Bridge.ID); err != nil {
		t.Fatal("recorded never-started anchor rejected", err)
	}
	t.Log("retained bridge/anchor match current fixed contracts and pure containment checks; no engine call")
}

// Failed initial-Start trials may still carry valid fixed resource inspections.
// Replaying those fields is not proof of namespace completion or overall success.
func TestProjectRecordedRunningAnchorContracts(t *testing.T) {
	path := os.Getenv("OMABIT_TEST_ANCHOR_RECEIPT")
	if path == "" {
		t.Skip("requires an exact retained disposable anchor receipt")
	}
	data, err := os.ReadFile(path)
	if err != nil || len(data) > 1<<20 {
		t.Fatal("missing/excessive retained receipt", err)
	}
	var r struct {
		ShipOperations bool                       `json:"ship_operations"`
		RunnerID       string                     `json:"runner_id"`
		OperationID    string                     `json:"operation_id"`
		Prepared       projectnetwork.Preparation `json:"prepared"`
		Project        struct {
			ID       string              `json:"id"`
			Revision uint64              `json:"revision"`
			Plan     projectnetwork.Plan `json:"plan"`
		} `json:"project"`
		Start struct {
			Journal projectnetwork.AnchorStart `json:"journal"`
			Bridge  []bridge                   `json:"bridge_inspection"`
			Anchor  []container                `json:"anchor_inspection"`
		} `json:"anchor_start"`
	}
	if json.Unmarshal(data, &r) != nil || r.ShipOperations || len(r.Start.Anchor) != 1 || len(r.Start.Bridge) != 1 {
		t.Fatal("invalid bounded anchor receipt")
	}
	c, err := newProjectResourceContract(projectSelection{RunnerID: r.RunnerID, ProjectID: r.Project.ID, Revision: r.Project.Revision, OperationID: r.OperationID, Generation: 1, Plan: r.Project.Plan}, r.Prepared.EngineID, r.Start.Anchor[0].Image)
	if err != nil || !reflect.DeepEqual(c.Preparation, r.Prepared) || r.Start.Journal.Validate(r.Project.Plan) != nil {
		t.Fatal("recorded fixed contract changed", err)
	}
	p := r.Start.Journal.Prepared
	if err = c.validateBridge(r.Start.Bridge[0], p.Bridge.ID, p.Anchor.ID); err != nil {
		t.Fatal("recorded bridge rejected", err)
	}
	if err = c.validateAnchor(r.Start.Anchor[0], p.Anchor.ID, p.Bridge.ID, true); err != nil {
		t.Fatal("recorded running anchor rejected", err)
	}
	t.Log("retained running anchor/bridge fixed contracts pass; no engine call or namespace completion claim")
}
