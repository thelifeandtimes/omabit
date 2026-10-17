package managed

import (
	"context"
	"errors"
	"fmt"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/hostports"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func (s *store) suggestHTTPPort(ctx context.Context) (uint16, error) {
	all, err := instances(s.db)
	if err != nil {
		return 0, err
	}
	reserved := map[uint16]bool{}
	for _, v := range all {
		reserved[v.Public.Config.HTTPPort] = true
		if v.Runtime.Prepared {
			reserved[v.Runtime.Port] = true
		}
	}
	free := s.portAvailable
	if free == nil {
		free = portAvailable
	}
	for p := 18080; p <= 65535; p++ {
		if err := ctx.Err(); err != nil {
			return 0, err
		}
		if !reserved[uint16(p)] && free("tcp", uint16(p)) {
			return uint16(p), nil
		}
	}
	return 0, conflict("port_unavailable", "no available host HTTP port")
}

func portAvailable(kind string, port uint16) bool {
	return hostports.Available(kind, port)
}

func (s *store) availablePort(kind string, port uint16) error {
	free := s.portAvailable
	if free == nil {
		free = portAvailable
	}
	if !free(kind, port) {
		return conflict("port_conflict", fmt.Sprintf("host %s port %d is already in use; choose an unused port", kind, port))
	}
	return nil
}

func reserveAmes(q querier, id string, port uint16) error {
	all, err := instances(q)
	if err != nil {
		return err
	}
	for _, v := range all {
		if v.Public.ID == id || v.Runtime.Boot == nil {
			continue
		}
		if v.Public.Config.AmesPort == port || v.Runtime.AmesPort == port {
			return conflict("port_conflict", "Ames UDP port is reserved by another ship")
		}
	}
	return nil
}

func (s *store) choosePort(q querier, id, kind string, wanted uint16) (uint16, error) {
	free := s.portAvailable
	if free == nil {
		free = portAvailable
	}
	reserved := func(port uint16) error {
		if kind == "udp" {
			return reserveAmes(q, id, port)
		}
		all, err := instances(q)
		if err != nil {
			return err
		}
		for _, v := range all {
			if v.Public.ID != id && (v.Public.Config.HTTPPort == port || v.Runtime.Prepared && v.Runtime.Port == port) {
				return conflict("port_conflict", "HTTP port is reserved by another ship")
			}
		}
		return nil
	}
	if wanted != 0 {
		if wanted < 1024 {
			return 0, bad("choose unprivileged HTTP/Ames ports, 1024–65535")
		}
		if err := reserved(wanted); err != nil {
			return 0, err
		}
		if !free(kind, wanted) {
			return 0, conflict("port_conflict", "selected host port is already in use")
		}
		return wanted, nil
	}
	start := 18080
	if kind == "udp" {
		start = 34343
	}
	for p := start; p <= 65535; p++ {
		if err := reserved(uint16(p)); err != nil {
			var problem *fault
			if !errors.As(err, &problem) {
				return 0, err
			}
			continue
		}
		if free(kind, uint16(p)) {
			return uint16(p), nil
		}
	}
	return 0, conflict("port_unavailable", "no available host port in the automatic range")
}

func validateOwnedConfig(c protocol.Configuration) error {
	if c.HTTPPort < 1024 || c.AmesPort < 1024 || !protocol.ValidLoom(c.Loom) {
		return bad("owned runtime requires unprivileged HTTP/Ames ports and loom 20..34")
	}
	return nil
}
