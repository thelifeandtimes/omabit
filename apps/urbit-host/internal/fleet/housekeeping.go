package fleet

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Housekeeping does no heavy metrics collection. It completes one-time birth
// bookkeeping and checks image notices hourly, without replacing running images.
func (m *Manager) Housekeeping(ctx context.Context) {
	tick := time.NewTicker(15 * time.Second)
	defer tick.Stop()
	updates := time.NewTicker(time.Hour)
	defer updates.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			m.CompleteBirth(ctx)
		case <-updates.C:
			check, cancel := context.WithTimeout(ctx, time.Minute)
			_, err := m.CheckUpdates(check)
			cancel()
			if err != nil {
				log.Printf("runtime notice check failed: %v", err)
			}
		}
	}
}
func (m *Manager) CompleteBirth(ctx context.Context) {
	if !m.serial.TryLock() {
		return
	}
	defer m.serial.Unlock()
	instances, _ := m.Snapshot()
	for _, i := range instances {
		if i.Busy != "" || i.Archived != "" || i.Lifecycle != "active" || i.Kind == "fake" {
			continue
		}
		keyPath, e := CheckPath(m.Engine.Host.DataRoot, filepath.Join(i.RelativeDir(), ".boot-secret", "network.key"))
		if e != nil {
			continue
		}
		_, keyErr := os.Stat(keyPath)
		if i.Identity != "" && os.IsNotExist(keyErr) {
			continue
		}
		probe, cancel := context.WithTimeout(ctx, 25*time.Second)
		identity, e := m.Engine.Control(probe, i, "our")
		cancel()
		if e != nil {
			continue
		}
		identity = strings.TrimPrefix(strings.TrimSpace(identity), "~")
		if len(identity) < 3 || len(identity) > 127 || strings.Trim(identity, "abcdefghijklmnopqrstuvwxyz-_") != "" {
			continue
		}
		if i.Identity != "" && i.Identity != identity {
			log.Printf("instance %s identity verification failed; boot secret retained", i.ID)
			continue
		}
		if i.Identity == "" {
			i.Identity = identity
			if e = m.setInstance(i); e != nil {
				continue
			}
		}
		if keyErr == nil {
			if e = os.Remove(keyPath); e != nil {
				log.Printf("instance %s boot-key cleanup failed", i.ID)
			}
		}
	}
}
