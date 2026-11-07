package protocol

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
)

func TestSourceStatusBoundsFileDetailWithoutLosingCount(t *testing.T) {
	files := []deskwork.File{}
	for i := 0; i < 300; i++ {
		files = append(files, deskwork.File{Path: fmt.Sprintf("src/file-%03d.txt", i), SHA256: strings.Repeat("a", 64)})
	}
	unix, _ := deskwork.NewManifest(files)
	clay, _ := deskwork.NewManifest([]deskwork.File{})
	comparison, _ := deskwork.Compare(clay, unix, clay)
	comparison.State = "different"
	status := deskwork.SourceStatus{Version: 1, ReadAt: time.Now().UTC(), Observation: deskwork.Observation{Exists: true, Revision: 1, ClayHash: "0v123", Publication: "private"}, Unix: unix, CanonicalUnix: unix, Clay: clay, Comparison: comparison, Differences: comparison.UnixChanges, Autocommit: "unknown"}
	q := DeskSourceRequest{Selection: DeskSelection{Target: apppolicy.Target{InstanceID: "alpha", Desk: "example"}, RuntimeID: strings.Repeat("a", 64), ConfigRevision: 1}}
	out := NewDeskSourceStatus("runner-fixture", "simulation", q, "example", status, nil)
	if out.Validate() != nil || out.DifferenceCount != 300 || len(out.Differences) != MaxSourceDetails {
		t.Fatal(out.Validate(), out)
	}
	encoded, _ := json.Marshal(out)
	if len(encoded) > 1<<20 {
		t.Fatal("response exceeds client bound")
	}
	for _, change := range []func(*DeskSourceStatus){
		func(s *DeskSourceStatus) { s.UnixChangeCount = 0 },
		func(s *DeskSourceStatus) { s.State = "synchronized" },
		func(s *DeskSourceStatus) { s.State = "unix-changes" },
		func(s *DeskSourceStatus) { s.Differences[1] = s.Differences[0] },
		func(s *DeskSourceStatus) { s.Differences[0].Path = "../private/key" },
		func(s *DeskSourceStatus) { s.Differences = s.Differences[:1] },
	} {
		var copy DeskSourceStatus
		json.Unmarshal(encoded, &copy)
		change(&copy)
		if copy.Validate() == nil {
			t.Fatal("unsafe or false status accepted", copy)
		}
	}
}
