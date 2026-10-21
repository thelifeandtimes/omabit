package remotesetup

import (
	"encoding/base64"
	"encoding/binary"
	"strings"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/hostcompat"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/ownerhost"
)

func testRequest() Request {
	key := make([]byte, 51)
	binary.BigEndian.PutUint32(key, 11)
	copy(key[4:], "ssh-ed25519")
	binary.BigEndian.PutUint32(key[15:], 32)
	key[50] = 1
	return Request{Version: 1, ID: strings.Repeat("a", 32), PublicKey: "ssh-ed25519 " + base64.StdEncoding.EncodeToString(key), Role: "operator"}
}

func TestRemoteRequestAndResumeJournalBinding(t *testing.T) {
	r := testRequest()
	if err := r.Validate(); err != nil {
		t.Fatal(err)
	}
	j := &installJournal{Version: 1, ID: r.ID, Fingerprint: "fingerprint", Role: r.Role, Owner: ownerhost.Profile{Version: 1, Name: "owner", UID: 1001, GID: 1001, Home: "/home/owner", Piers: "/srv/urbit/piers"}, HelperSHA256: "original", Stage: "installing"}
	if !matchesJournal(j, r, "fingerprint", "original") {
		t.Fatal("matching interruption refused")
	}
	for _, change := range []func(*Request){func(r *Request) { r.ID = strings.Repeat("b", 32) }, func(r *Request) { r.Owner = "another" }, func(r *Request) { r.Role = "observer" }, func(r *Request) { r.RunnerID = "runner-other" }} {
		other := r
		change(&other)
		if matchesJournal(j, other, "fingerprint", "original") {
			t.Fatal("journal lost binding", other)
		}
	}
	if matchesJournal(j, r, "changed", "original") || matchesJournal(j, r, "fingerprint", "new-helper") {
		t.Fatal("key/helper drift accepted")
	}
	r.Revoke = true
	if r.Validate() == nil {
		t.Fatal("unbound revocation accepted")
	}
	r.RunnerID = "runner-original"
	if r.Validate() != nil {
		t.Fatal("bound revocation refused")
	}
	j.Stage = "installed"
	j.RunnerID = r.RunnerID
	if !matchesJournal(j, r, "fingerprint", "new-helper") {
		t.Fatal("completed installation cannot be paired/revoked by a newer helper")
	}
}

func TestDependencyCandidateUsesSharedNativeEnginePolicy(t *testing.T) {
	for _, version := range []string{"1:29.7.2-1", "5:28.2.2-1~ubuntu.24.04~noble", "28.2.2+dfsg1-1"} {
		if hostcompat.EnginePackageVersionStatus(version) != "compatible" {
			t.Fatal(version)
		}
	}
	for _, version := range []string{"20.10.24+dfsg1-1", "1:30.0.0-1", "29.1.0-rc.1", "(none)", ""} {
		if hostcompat.EnginePackageVersionStatus(version) == "compatible" {
			t.Fatal("incompatible candidate", version)
		}
	}
	var output commandOutput
	output.Write([]byte(strings.Repeat("x", 100000)))
	output.Write([]byte("end"))
	if output.Len() != 65536 || !strings.HasSuffix(output.String(), "end") {
		t.Fatal("native output was not bounded")
	}
}
