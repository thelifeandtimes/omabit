package deskwork

import (
	"errors"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
	"sort"
)

// AppState describes separate stages, not an all-purpose readiness assertion.
// In particular live Gall agents and a loaded glob do not establish app health.
type AppAgent struct {
	Name string `json:"name"`
	Live bool   `json:"live"`
}
type AppState struct {
	Zest               string     `json:"zest"`
	SourceRevision     uint64     `json:"source_revision"`
	SourceShip         string     `json:"source_ship"`
	SourceDesk         string     `json:"source_desk"`
	ExpectedAgents     []string   `json:"expected_agents"`
	Agents             []AppAgent `json:"agents"`
	TreatyAvailable    bool       `json:"treaty_available"`
	TreatyListed       bool       `json:"treaty_listed"`
	TreatyHash         string     `json:"treaty_hash"`
	DocketAvailable    bool       `json:"docket_available"`
	DocketHash         string     `json:"docket_hash"`
	FrontendMode       string     `json:"frontend_mode"`
	FrontendState      string     `json:"frontend_state"`
	FrontendDocketHash string     `json:"frontend_docket_hash"`
}

func (s *AppState) Normalize() error {
	sort.Strings(s.ExpectedAgents)
	sort.Slice(s.Agents, func(i, j int) bool { return s.Agents[i].Name < s.Agents[j].Name })
	return s.Validate()
}
func (s AppState) Validate() error {
	if s.SourceRevision == 0 || s.SourceRevision > apppolicy.MaxRevision || s.ExpectedAgents == nil || s.Agents == nil || len(s.ExpectedAgents) > 256 || len(s.Agents) > 256 {
		return errors.New("invalid application observation")
	}
	switch s.Zest {
	case "absent", "dead", "held", "live":
	default:
		return errors.New("invalid installation state")
	}
	if (s.SourceShip == "") != (s.SourceDesk == "") || s.SourceShip != "" && !validSelection(s.SourceShip, s.SourceDesk) {
		return errors.New("invalid application source")
	}
	for i, v := range s.ExpectedAgents {
		if !deskName.MatchString(v) || i > 0 && s.ExpectedAgents[i-1] >= v {
			return errors.New("invalid expected agents")
		}
	}
	for i, v := range s.Agents {
		if !deskName.MatchString(v.Name) || i > 0 && s.Agents[i-1].Name >= v.Name {
			return errors.New("invalid observed agents")
		}
	}
	validHash := func(h string) bool {
		return (Observation{Exists: true, Revision: 1, ClayHash: h, Publication: "private"}).Validate() == nil
	}
	if s.TreatyListed && !s.TreatyAvailable || (s.TreatyHash != "") != s.TreatyListed || s.TreatyHash != "" && !validHash(s.TreatyHash) || s.DocketHash != "" && !validHash(s.DocketHash) {
		return errors.New("invalid publication evidence")
	}
	switch s.FrontendMode {
	case "none", "site", "glob-ames", "glob-http":
	default:
		return errors.New("invalid frontend mode")
	}
	if (s.FrontendMode == "none") != (s.DocketHash == "") {
		return errors.New("inconsistent docket observation")
	}
	if s.FrontendDocketHash != "" && !validHash(s.FrontendDocketHash) {
		return errors.New("invalid realized docket hash")
	}
	switch s.FrontendState {
	case "unavailable":
		if s.DocketAvailable || s.FrontendDocketHash != "" {
			return errors.New("inconsistent docket availability")
		}
	case "absent", "site", "glob", "install", "suspend", "hung":
		if (s.FrontendState == "absent") != (s.FrontendDocketHash == "") {
			return errors.New("inconsistent realized docket")
		}
		if !s.DocketAvailable {
			return errors.New("inconsistent docket state")
		}
	default:
		return errors.New("invalid frontend state")
	}
	return nil
}
