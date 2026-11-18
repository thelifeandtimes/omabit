package service

import (
	_ "embed"
	"encoding/json"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/buildmeta"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"sort"
)

//go:embed legacy-inventory.json
var legacyInventory []byte

type CLIBinding struct {
	CLI             string   `json:"cli"`
	Effects         []string `json:"effects"`
	Operations      []string `json:"operations"`
	MCPStatus       string   `json:"mcp_status"`
	ExclusionReason string   `json:"exclusion_reason,omitempty"`
}
type Catalog struct {
	ClientBuild          *protocol.ExecutableBuild `json:"client_build"`
	Version              int                       `json:"version"`
	Operations           []Descriptor              `json:"operations"`
	CLIBindings          []CLIBinding              `json:"cli_bindings"`
	MutationToolsEnabled bool                      `json:"mutation_tools_enabled"`
	Transport            string                    `json:"transport"`
}

func (s *Service) Discovery() Catalog {
	var inventory struct {
		Bindings []CLIBinding `json:"bindings"`
	}
	// The embedded inventory is checked in CI; it contains no executable inputs.
	if err := json.Unmarshal(legacyInventory, &inventory); err != nil {
		panic(err)
	}
	operations := s.Catalog()
	for i, b := range inventory.Bindings {
		b.Operations = []string{}
		b.MCPStatus = "excluded"
		for _, op := range operations {
			for _, cli := range op.CLI {
				if cli == b.CLI {
					b.Operations = append(b.Operations, op.ID)
					b.ExclusionReason = ""
					b.MCPStatus = "shared-core"
					b.Effects = op.Effects
				}
			}
		}
		inventory.Bindings[i] = b
	}
	for _, op := range operations {
		inventory.Bindings = append(inventory.Bindings, CLIBinding{CLI: "omabit call " + op.ID, Effects: op.Effects, Operations: []string{op.ID}, MCPStatus: "shared-core"})
	}
	sort.Slice(inventory.Bindings, func(i, j int) bool { return inventory.Bindings[i].CLI < inventory.Bindings[j].CLI })
	return Catalog{ClientBuild: buildmeta.Current(""), Version: 1, Operations: operations, CLIBindings: inventory.Bindings, MutationToolsEnabled: s.options.EnableMutations, Transport: "local-stdio; HTTP not implemented"}
}
