package managed

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

type IdentityRepairRequest struct {
	RunnerID     string `json:"runner_id"`
	InstanceID   string `json:"instance_id"`
	Revision     uint64 `json:"revision"`
	ExpectedShip string `json:"expected_ship"`
	Ship         string `json:"ship"`
	ContainerID  string `json:"container_id"`
	RequestID    string `json:"request_id"`
}

type identityRepairJournal struct {
	Version   int                                `json:"version"`
	Request   IdentityRepairRequest              `json:"request"`
	Before    storedInstance                     `json:"before"`
	Progress  *runtimequalify.BootIdentityRepair `json:"progress"`
	Operation protocol.Operation                 `json:"operation"`
}

type identityRepairDriver func(context.Context, runtimequalify.ManagedState, string, string, string, *runtimequalify.BootIdentityRepair, func(runtimequalify.BootIdentityRepair) error) (runtimequalify.ManagedState, error)

// RepairBootIdentity requires the manager's exclusive, existing ownership lock.
// Schema47 readers preserve the opaque meta journal and already quarantine the
// unresolved boot. No instance fields change until correction commits atomically.
func (l *OfflineStateLock) RepairBootIdentity(ctx context.Context, req IdentityRepairRequest) (protocol.Operation, error) {
	return l.repairBootIdentity(ctx, req, runtimequalify.RepairBootIdentity)
}

func (l *OfflineStateLock) repairBootIdentity(ctx context.Context, req IdentityRepairRequest, driver identityRepairDriver) (protocol.Operation, error) {
	if !protocol.ValidID(req.InstanceID) || !protocol.ValidID(req.RequestID) || req.Revision == 0 || req.Revision == ^uint64(0) {
		return protocol.Operation{}, errors.New("identity repair requires explicit instance, request ID and revision")
	}
	db, err := l.openDB(true)
	if err != nil {
		return protocol.Operation{}, err
	}
	defer db.Close()
	identity, err := inspectOfflineIdentity(db)
	if err != nil {
		return protocol.Operation{}, err
	}
	if identity.Schema != StateSchemaVersion || identity.RunnerID != req.RunnerID || identity.Quiesced {
		return protocol.Operation{}, errors.New("identity repair runner/schema mismatch or management update in progress")
	}
	fingerprint, _ := json.Marshal(req)
	const actor = "local-owner-identity-repair"
	if op, found, err := replay(db, actor, req.RequestID, string(fingerprint)); found || err != nil {
		return op, err
	}
	v, err := loadInstance(db, req.InstanceID)
	if err != nil {
		return protocol.Operation{}, err
	}
	if err = admitIdentityRepair(db, v, req); err != nil {
		return protocol.Operation{}, err
	}
	key := "offline-boot-identity-repair-v1/" + req.InstanceID
	j := identityRepairJournal{Version: 1, Request: req, Before: v}
	var data []byte
	err = db.QueryRow("SELECT value FROM meta WHERE key=?", key).Scan(&data)
	if err == nil {
		var retained identityRepairJournal
		if json.Unmarshal(data, &retained) != nil || retained.Version != 1 || retained.Request != req || !reflect.DeepEqual(retained.Before, v) || retained.Progress == nil || retained.Operation.Status != "running" {
			return protocol.Operation{}, errors.New("repair journal binding changed; retain evidence and review intervening manager activity")
		}
		j = retained
	} else if !errors.Is(err, sql.ErrNoRows) {
		return protocol.Operation{}, err
	} else {
		id, err := randomID()
		if err != nil {
			return protocol.Operation{}, err
		}
		j.Operation = protocol.Operation{ID: "op-" + id, RequestID: req.RequestID, InstanceID: req.InstanceID, Kind: "repair-boot-identity", Actor: actor, Status: "running", ConfigRevision: req.Revision, CreatedAt: time.Now().UTC()}
	}
	// Every callback commits only the repair journal, preserving the old runtime
	// and its quarantine. Any intervening ordinary manager activity invalidates it.
	save := func(progress runtimequalify.BootIdentityRepair) error {
		if err := l.Verify(); err != nil {
			return err
		}
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		defer tx.Rollback()
		current, err := loadInstance(tx, req.InstanceID)
		if err != nil {
			return err
		}
		if !reflect.DeepEqual(current, j.Before) {
			return errors.New("instance changed during offline identity repair")
		}
		j.Progress = &progress
		body, err := json.Marshal(j)
		if err != nil {
			return err
		}
		if _, err = tx.Exec("INSERT INTO meta(key,value) VALUES(?,?) ON CONFLICT(key) DO UPDATE SET value=excluded.value", key, body); err != nil {
			return err
		}
		return tx.Commit()
	}
	// Separate pointer-backed runtime state from the immutable before image.
	raw, _ := json.Marshal(v.Runtime)
	var original runtimequalify.ManagedState
	if err := json.Unmarshal(raw, &original); err != nil {
		return protocol.Operation{}, err
	}
	state, err := driver(ctx, original, req.ExpectedShip, req.Ship, req.ContainerID, j.Progress, save)
	if err != nil {
		return protocol.Operation{}, err
	}
	if j.Progress == nil || j.Progress.Stage != "removed" || state.ID != v.Runtime.ID || state.NativeShip != req.Ship || !state.Initialized || !state.MaintenanceQualified() || state.UncertainProvisioning() || state.UncertainMaintenance() || state.Storage == nil || state.Storage.Path != v.Runtime.Storage.Path || state.Storage.MountIdentity != v.Runtime.Storage.MountIdentity || state.Storage.Identity.Inode == 0 {
		return protocol.Operation{}, errors.New("identity repair did not produce verified same-pier completion")
	}
	if err := l.Verify(); err != nil {
		return protocol.Operation{}, err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return protocol.Operation{}, err
	}
	defer tx.Rollback()
	current, err := loadInstance(tx, req.InstanceID)
	if err != nil {
		return protocol.Operation{}, err
	}
	if !reflect.DeepEqual(current, j.Before) {
		return protocol.Operation{}, errors.New("instance changed before repair commit")
	}
	if err = admitIdentityRepair(tx, current, req); err != nil {
		return protocol.Operation{}, err
	}
	v.Runtime, v.RuntimeRevision = state, v.RuntimeRevision+1
	v.Public.ConfigRevision++
	v.RuntimeConfigRevision = v.Public.ConfigRevision
	v.Public.Ship = req.Ship
	if v.Public.Label == strings.TrimPrefix(req.ExpectedShip, "~") {
		v.Public.Label = strings.TrimPrefix(req.Ship, "~")
	}
	now := time.Now().UTC()
	applyObservation(&v, runtimequalify.ManagedObservation{State: "stopped", Health: "unknown"}, now)
	op := j.Operation
	op.Status, op.ConfigRevision, op.FinishedAt = "succeeded", v.Public.ConfigRevision, &now
	op.Message = "Corrected verified keyed boot identity from " + req.ExpectedShip + " to " + req.Ship + "; same pier retained, gracefully stopped; normal Start required"
	if err = saveInstance(tx, v); err != nil {
		return op, err
	}
	if err = insertOperation(tx, op, string(fingerprint)); err != nil {
		return op, err
	}
	j.Operation = op
	data, err = json.Marshal(j)
	if err != nil {
		return op, err
	}
	if _, err = tx.Exec("UPDATE meta SET value=? WHERE key=?", data, key); err != nil {
		return op, err
	}
	return op, tx.Commit()
}

func admitIdentityRepair(q querier, v storedInstance, req IdentityRepairRequest) error {
	if v.Public.ConfigRevision != req.Revision || v.Public.Ship != req.ExpectedShip || v.Public.DesiredState != "stopped" || v.Public.ActiveOperationID != "" || !v.Public.RecoveryRequired {
		return errors.New("identity repair requires the pinned revision, old identity, stopped intent and idle recovery state")
	}
	if err := runtimequalify.BootIdentityRepairEligible(v.Runtime, req.ExpectedShip, req.Ship, req.ContainerID); err != nil {
		return err
	}
	all, err := instances(q)
	if err != nil {
		return err
	}
	for _, other := range all {
		if other.Public.ActiveOperationID != "" {
			return errors.New("another managed operation is active")
		}
		if other.Public.ID != req.InstanceID && (other.Public.Mode != "fake" && other.Public.Ship == req.Ship || other.Runtime.NativeShip == req.Ship || other.Runtime.Comet != nil && other.Runtime.Comet.Ship == req.Ship) {
			return errors.New("corrected identity is already enrolled")
		}
	}
	return nil
}
