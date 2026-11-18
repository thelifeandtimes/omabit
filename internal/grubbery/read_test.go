package grubbery_test

import (
	"strings"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/grubbery"
)

func TestNativeReadsRemainBoundAndNeedNoSourceCache(t *testing.T) {
	p, _ := plan(t)
	for _, step := range p.Steps {
		reads, err := p.AfterReads(step.ID)
		if err != nil || len(reads) == 0 {
			t.Fatal("saved after-read unavailable", err)
		}
		for _, r := range reads {
			q, err := r.Request()
			if err != nil || q.Method != "GET" || len(q.Body) != 0 || !strings.HasPrefix(q.Path, "/grubbery/api/") {
				t.Fatal("read gained a mutation or URL", q, err)
			}
			d, err := r.Descriptor()
			if err != nil || d.PlanDigest != p.Digest || d.StepID != step.ID || d.BindingDigest != hashValue(p.Binding) {
				t.Fatal("after-read lost original binding", d, err)
			}
			if strings.Contains(q.Path, "/cull/") || strings.Contains(q.Path, "/sand/") || strings.Contains(q.Query, "force") {
				t.Fatal("read acquired destructive authority")
			}
		}
	}
	sources, err := p.StagedReads()
	if err != nil || len(sources) != len(p.Source.Manifest.Files) {
		t.Fatal("staged read inventory changed", err)
	}
	root, _ := p.Binding.ReleaseRoot(p.RequestID)
	for i, r := range sources {
		q, err := r.Request()
		if err != nil || q.Path != "/grubbery/api/peek"+root+"/"+p.Source.Manifest.Files[i].Path || q.Query != "" {
			t.Fatal("staged source read escaped its original manifest", err)
		}
	}
	if _, err = p.AfterReads(strings.Repeat("0", 64)); err == nil {
		t.Fatal("foreign step read admitted")
	}
	changed := p
	changed.Digest = strings.Repeat("f", 64)
	if _, err = changed.StagedReads(); err == nil {
		t.Fatal("changed metadata gained source read")
	}
}
func TestNativeReadSelectorsDoNotAcceptArbitraryPaths(t *testing.T) {
	b := binding()
	for _, purpose := range []grubbery.ReadPurpose{grubbery.ReadSourceRoot, grubbery.ReadSourceOwner, grubbery.ReadDeskRoot, grubbery.ReadDeskOwner, grubbery.ReadSourceConfig, grubbery.ReadFollowedVersion} {
		r, err := b.Read(purpose)
		if err != nil {
			t.Fatal(err)
		}
		q, err := r.Request()
		if err != nil || q.Method != "GET" || q.Body != nil {
			t.Fatal(err)
		}
	}
	for _, purpose := range []grubbery.ReadPurpose{"", "../../secret", "https://other.invalid", "sand", "owner?force=1"} {
		if _, err := b.Read(purpose); err == nil {
			t.Fatal("arbitrary native read accepted", purpose)
		}
	}
	if _, err := (grubbery.ReadRequest{}).Request(); err == nil {
		t.Fatal("zero read accepted")
	}
}
