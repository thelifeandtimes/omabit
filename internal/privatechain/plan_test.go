package privatechain

import (
	"encoding/json"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"
)

func foundation(t *testing.T) Foundation {
	t.Helper()
	data, err := os.ReadFile("testdata/qualified-foundation.json")
	if err != nil {
		t.Fatal(err)
	}
	var f Foundation
	if err = json.Unmarshal(data, &f); err != nil {
		t.Fatal(err)
	}
	return f
}
func request(t *testing.T) Request {
	return Request{Label: "Local keyed development", Foundation: foundation(t), ConfirmSharedAdministration: true, Members: []MemberRequest{{"sender", "~zod", 18381}, {"receiver", "~nec", 18382}}}
}
func TestCanonicalPlanPreservesInputsAndQualifiedPorts(t *testing.T) {
	req := request(t)
	before := request(t)
	p, err := Build(req, Reservations{})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(req, before) {
		t.Fatal("build mutated request")
	}
	want := map[string][2]uint16{"~zod": {18081, 31337}, "~nec": {18082, 31338}}
	for _, m := range p.Members {
		if want[m.Ship] != [2]uint16{m.InternalHTTPPort, m.InternalAmesPort} || !m.FreshKeyRequired {
			t.Fatal(m)
		}
	}
	if p.Mode != Mode || p.BindAddress != "127.0.0.1" || p.GatewayPort != 25432 || p.PublishRPC || p.PublishAmes || p.FirstStop != FirstStop {
		t.Fatal(p)
	}
	d, err := p.Digest()
	if err != nil {
		t.Fatal(err)
	}
	slices.Reverse(req.Members)
	slices.Reverse(req.Foundation.Contracts)
	q, err := Build(req, Reservations{})
	if err != nil {
		t.Fatal(err)
	}
	other, err := q.Digest()
	if err != nil || d != other {
		t.Fatal("input order changed digest", err)
	}
	req.Foundation.Contracts[0].CodeSHA256 = "changed"
	req.Members[0].HTTPPort++
	if got, _ := p.Digest(); got != d {
		t.Fatal("output retained mutable input slices")
	}
}
func TestFoundationRejectsWrongChainAndCode(t *testing.T) {
	cases := map[string]func(*Foundation){
		"version": func(f *Foundation) { f.Version = 2 }, "profile": func(f *Foundation) { f.Profile = "mainnet" }, "chain": func(f *Foundation) { f.ChainID = 1 },
		"missing genesis": func(f *Foundation) { f.Genesis = "" }, "zero genesis": func(f *Foundation) { f.Genesis = "0x" + strings.Repeat("0", 64) },
		"uppercase genesis": func(f *Foundation) { f.Genesis = strings.ToUpper(f.Genesis) }, "short genesis": func(f *Foundation) { f.Genesis = "0x1234" },
		"missing contract": func(f *Foundation) { f.Contracts = f.Contracts[:8] }, "extra contract": func(f *Foundation) { f.Contracts = append(f.Contracts, f.Contracts[0]) },
		"duplicate contract": func(f *Foundation) { f.Contracts[0] = f.Contracts[1] }, "wrong address": func(f *Foundation) { f.Contracts[0].Address = "0x" + strings.Repeat("1", 40) },
		"wrong code": func(f *Foundation) { f.Contracts[0].CodeSHA256 = strings.Repeat("1", 64) }, "unknown contract": func(f *Foundation) { f.Contracts[0].Name = "Other" },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			f := foundation(t)
			mutate(&f)
			if d, err := f.Digest(); err == nil || d != "" {
				t.Fatal("accepted wrong foundation", d, err)
			}
		})
	}
}
func TestInvalidMembersCannotProducePartialPlan(t *testing.T) {
	cases := map[string]func(*Request){
		"trust": func(r *Request) { r.ConfirmSharedAdministration = false }, "label": func(r *Request) { r.Label = "" }, "label control": func(r *Request) { r.Label = "dev\nproduction" },
		"partial": func(r *Request) { r.Members = r.Members[:1] }, "extra": func(r *Request) { r.Members = append(r.Members, r.Members[0]) },
		"duplicate ID": func(r *Request) { r.Members[0].InstanceID = r.Members[1].InstanceID }, "path ID": func(r *Request) { r.Members[0].InstanceID = "../production" },
		"duplicate ship": func(r *Request) { r.Members[0].Ship = r.Members[1].Ship }, "unsupported galaxy": func(r *Request) { r.Members[0].Ship = "~bud" },
		"noncanonical ship": func(r *Request) { r.Members[0].Ship = "zod" }, "planet": func(r *Request) { r.Members[0].Ship = "~sampel-palnet" },
		"privileged port": func(r *Request) { r.Members[0].HTTPPort = 80 }, "shared port": func(r *Request) { r.Members[0].HTTPPort = r.Members[1].HTTPPort },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			r := request(t)
			mutate(&r)
			p, err := Build(r, Reservations{})
			if err == nil || !reflect.DeepEqual(p, Plan{}) {
				t.Fatal("partial plan escaped", p, err)
			}
		})
	}
	for name, reserved := range map[string]Reservations{"instance": {Instances: map[string]bool{"sender": true}}, "port": {HTTPPorts: map[uint16]bool{18382: true}}} {
		t.Run(name, func(t *testing.T) {
			if _, err := Build(request(t), reserved); err == nil {
				t.Fatal("reservation reused")
			}
		})
	}
}
func TestDigestRefusesRelabelingAndLifecycleChanges(t *testing.T) {
	cases := map[string]func(*Plan){
		"fake": func(p *Plan) { p.Mode = projectnetwork.FakeGalaxies }, "livenet": func(p *Plan) { p.Mode = "livenet" },
		"RPC publication": func(p *Plan) { p.PublishRPC = true }, "Ames publication": func(p *Plan) { p.PublishAmes = true },
		"wildcard HTTP": func(p *Plan) { p.BindAddress = "0.0.0.0" }, "gateway": func(p *Plan) { p.GatewayPort++ },
		"unacknowledged stop": func(p *Plan) { p.FirstStop = "docker-stop" }, "reused key": func(p *Plan) { p.Members[0].FreshKeyRequired = false },
		"internal HTTP": func(p *Plan) { p.Members[0].InternalHTTPPort++ }, "internal Ames": func(p *Plan) { p.Members[0].InternalAmesPort++ },
		"unbound foundation": func(p *Plan) { p.FoundationDigest = strings.Repeat("1", 64) }, "noncanonical order": func(p *Plan) { slices.Reverse(p.Members) },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			p, err := Build(request(t), Reservations{})
			if err != nil {
				t.Fatal(err)
			}
			mutate(&p)
			if d, err := p.Digest(); err == nil || d != "" {
				t.Fatal("changed lifecycle contract accepted", d, err)
			}
		})
	}
	p, _ := Build(request(t), Reservations{})
	data, _ := json.Marshal(p)
	var fake projectnetwork.Plan
	if err := json.Unmarshal(data, &fake); err != nil {
		t.Fatal(err)
	}
	if _, err := fake.Digest(); err == nil {
		t.Fatal("private plan admitted by fake project registration")
	}
}
func TestSameGenesisAndNamesDoNotIdentifyProject(t *testing.T) {
	f := foundation(t)
	d, err := f.Digest()
	if err != nil {
		t.Fatal(err)
	}
	expected := Binding{Mode: Mode, RunnerID: "runner-a", ProjectID: "project-a", Generation: 1, FoundationDigest: d}
	if err := Match(expected, expected, f); err != nil {
		t.Fatal(err)
	}
	for name, mutate := range map[string]func(*Binding){
		"project": func(b *Binding) { b.ProjectID = "project-b" }, "runner": func(b *Binding) { b.RunnerID = "runner-b" },
		"generation": func(b *Binding) { b.Generation = 2 }, "foundation": func(b *Binding) { b.FoundationDigest = strings.Repeat("1", 64) },
		"fake": func(b *Binding) { b.Mode = projectnetwork.FakeGalaxies }, "zero generation": func(b *Binding) { b.Generation = 0 },
		"absent project": func(b *Binding) { b.ProjectID = "" },
	} {
		t.Run(name, func(t *testing.T) {
			other := expected
			mutate(&other)
			if Match(expected, other, f) == nil {
				t.Fatal("wrong project observation accepted")
			}
		})
	}
	other := foundation(t)
	other.Genesis = "0x" + strings.Repeat("1", 64)
	if Match(expected, expected, other) == nil {
		t.Fatal("same chain ID hid different genesis")
	}
}
func TestChangedReviewedPortAndGenesisChangeDigest(t *testing.T) {
	p, _ := Build(request(t), Reservations{})
	original, _ := p.Digest()
	for _, field := range []string{"port", "genesis"} {
		t.Run(field, func(t *testing.T) {
			r := request(t)
			if field == "port" {
				r.Members[0].HTTPPort++
			} else {
				r.Foundation.Genesis = "0x" + strings.Repeat("1", 64)
			}
			// Choose an unused port; the original pair occupies consecutive ports.
			if field == "port" {
				r.Members[0].HTTPPort = 18390
			}
			p, err := Build(r, Reservations{})
			if err != nil {
				t.Fatal(err)
			}
			d, err := p.Digest()
			if err != nil || d == original {
				t.Fatal("review failed to bind change", err)
			}
		})
	}
}
