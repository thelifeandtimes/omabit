//go:build linux && amd64 && omabit_private_member_boot_inspection

package main

import (
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/privatechain"
	"testing"
	"time"
)

func TestMemberBootAuditReopen(t *testing.T) {
	first := memberBootProof{Version: 1, ProjectID: "project", InstanceID: "member", OperationID: "op", OperationSHA256: "operation", LaunchSHA256: "launch"}
	reopened := first
	reopened.ManagerReopened = true
	reopened.ReplayPreserved = true
	if !memberBootContinuation(first, reopened) {
		t.Fatal("valid retained receipt rejected")
	}
	for _, change := range []func(*memberBootProof){func(p *memberBootProof) { p.OperationID = "other" }, func(p *memberBootProof) { p.LaunchSHA256 = "other" }, func(p *memberBootProof) { p.OperationSHA256 = "other" }, func(p *memberBootProof) { p.ReplayPreserved = false }, func(p *memberBootProof) { p.ManagerReopened = false }} {
		v := reopened
		change(&v)
		if memberBootContinuation(first, v) {
			t.Fatal("changed receipt admitted")
		}
	}
}
func TestMemberBootAuditShutdownSequence(t *testing.T) {
	makeRows := func() []memberBootReceipt {
		rows := []memberBootReceipt{}
		for i, phase := range []string{"ready", "stop-dispatched", "stop-acknowledged", "stopped"} {
			r := memberBootReceipt{Version: 1, Phase: phase, LaunchSHA256: "launch", ContainerID: "container", Ship: "~zod", PublicHTTPIdentity: true, LensIdentity: true, GatewayBefore: &privatechain.GatewayObservation{}, StopDispatched: i >= 1, StopAcknowledged: i >= 2}
			if i == 3 {
				r.GatewayAfter = &privatechain.GatewayObservation{}
				r.FinishedAt = time.Unix(100, 0)
				r.StoppedMetadata = true
			}
			rows = append(rows, r)
		}
		return rows
	}
	if !memberBootSequence(makeRows()) {
		t.Fatal("acknowledged sequence rejected")
	}
	changes := map[string]func([]memberBootReceipt){
		"ack-before-dispatch":   func(r []memberBootReceipt) { r[1].StopAcknowledged = true },
		"missing-ack":           func(r []memberBootReceipt) { r[3].StopAcknowledged = false },
		"container-replaced":    func(r []memberBootReceipt) { r[3].ContainerID = "other" },
		"missing-readiness":     func(r []memberBootReceipt) { r[0].PublicHTTPIdentity = false },
		"failed-exit":           func(r []memberBootReceipt) { r[3].ExitCode = 1 },
		"missing-exit-time":     func(r []memberBootReceipt) { r[3].FinishedAt = time.Time{} },
		"missing-final-gateway": func(r []memberBootReceipt) { r[3].GatewayAfter = nil },
		"premature-stopped":     func(r []memberBootReceipt) { r[2].StoppedMetadata = true },
	}
	for name, change := range changes {
		t.Run(name, func(t *testing.T) {
			rows := makeRows()
			change(rows)
			if memberBootSequence(rows) {
				t.Fatal("changed shutdown evidence admitted")
			}
		})
	}
}
