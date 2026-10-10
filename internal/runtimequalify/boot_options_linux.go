//go:build linux && amd64

package runtimequalify

import (
	"context"
	"errors"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/bootpill"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	candidate "git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimecandidate"
)

func (q *qualifier) selectedPill() *protocol.BootPill {
	if q.d.bootstrap != nil {
		return q.d.bootstrap.Pill
	}
	if q.d.boot != nil {
		return q.d.boot.Pill
	}
	return nil
}
func (q *qualifier) canonicalPill() bool {
	return (q.d.bootstrap != nil || q.d.boot != nil) && q.selectedPill() == nil
}
func (q *qualifier) pillSource() string {
	if q.canonicalPill() {
		return "runtime-canonical"
	}
	if p := q.selectedPill(); p != nil {
		return "sha256:" + p.SHA256
	}
	return candidate.PillURL
}
func (q *qualifier) usesPillFile(kind string) bool {
	return (kind == "boot" || kind == "http-boot" || kind == "comet-boot") && !q.canonicalPill()
}
func (q *qualifier) publishAmes(kind string) bool {
	return q.d.boot != nil && q.d.amesPort >= 1024 && isCometKind(kind) && kind != "comet-verify"
}
func (q *qualifier) copySelectedPill(ctx context.Context) error {
	p := q.selectedPill()
	if p == nil {
		return errors.New("selected pill is missing")
	}
	f, err := bootpill.Open(bootpill.CachePath(q.base, *p))
	if err != nil {
		return errors.New("selected boot pill is unavailable")
	}
	defer f.Close()
	return bootpill.Publish(ctx, f, *p, q.r.PillPath)
}
