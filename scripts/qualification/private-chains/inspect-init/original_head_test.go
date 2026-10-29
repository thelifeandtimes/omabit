package main

import (
	"strings"
	"testing"
)

func TestResumeAuditDistinguishesPredecessorHeadHashCoverage(t *testing.T) {
	s, r, g, p, h := syntheticResume()
	if auditResume(s, r, g, p, h)["original_head_hash_matched"] != false {
		t.Fatal("legacy proof overstated")
	}
	head := obj(obj(r["result"])["before"])["head_hash"]
	obj(p["gateway"])["headHash"] = head
	obj(obj(s["gateway_proof"])["gateway"])["headHash"] = head
	if auditResume(s, r, g, p, h)["original_head_hash_matched"] != true {
		t.Fatal("predecessor head hash missing")
	}
	changed := "0x" + strings.Repeat("e", 64)
	obj(p["gateway"])["headHash"] = changed
	obj(obj(s["gateway_proof"])["gateway"])["headHash"] = changed
	defer func() {
		if recover() == nil {
			t.Fatal("changed predecessor head admitted")
		}
	}()
	auditResume(s, r, g, p, h)
}
