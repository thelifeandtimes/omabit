package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestObserverTraceProjectsOnlyFixedMetadata(t *testing.T) {
	text := "SECRET raw payload\nOMABIT_PRIVATE_OBSERVER observe-inputs-after begin 0 false false\nOMABIT_PRIVATE_OBSERVER observe-inputs-after end 25000 false true\n"
	got := observerTraceProjection(text)
	if got["count"] != 2 || got["retained_count"] != 2 || got["diagnostic_only"] != true {
		t.Fatal(got)
	}
	rows := got["records"].([]observerTraceEvent)
	if rows[1].Milliseconds != 25000 || rows[1].Succeeded || !rows[1].Expired {
		t.Fatal(rows)
	}
	raw, err := json.Marshal(got)
	if err != nil || strings.Contains(string(raw), "SECRET") {
		t.Fatal("raw data exported")
	}
}

func TestObserverTraceRefusesInjectedOrMalformedFields(t *testing.T) {
	for _, line := range []string{
		"OMABIT_PRIVATE_OBSERVER secret-value end 1 true false",
		"OMABIT_PRIVATE_OBSERVER observe-rpc end 900001 true false",
		"OMABIT_PRIVATE_OBSERVER observe-rpc end -1 true false",
		"OMABIT_PRIVATE_OBSERVER observe-rpc begin 1 false false",
		"OMABIT_PRIVATE_OBSERVER observe-rpc begin 0 true false",
		"OMABIT_PRIVATE_OBSERVER observe-rpc end 1 true false SECRET",
		"prefix OMABIT_PRIVATE_OBSERVER observe-rpc end 1 true false",
		"OMABIT_PRIVATE_OBSERVER observe-rpc end 1 null false",
	} {
		got := observerTraceProjection(line)
		if got["count"] != 0 {
			t.Fatal("accepted malformed diagnostic", got)
		}
	}
}

func TestObserverTraceBoundsRepeatedDiagnostics(t *testing.T) {
	line := "OMABIT_PRIVATE_OBSERVER observe-rpc end 1 true false\n"
	got := observerTraceProjection(strings.Repeat(line, 200))
	if got["count"] != 200 || got["retained_count"] != 128 || got["over_limit"] != false {
		t.Fatal(got)
	}
	got = observerTraceProjection(strings.Repeat(line, 1025))
	if got["over_limit"] != true || got["records"] != nil {
		t.Fatal("over-limit trace exported", got)
	}
}
