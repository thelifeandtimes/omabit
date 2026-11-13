package deskwork

import (
	"reflect"
	"testing"
)

func TestClayPathsAreValidatedBeforeUnixUse(t *testing.T) {
	got, err := UnixPaths([][]string{{"app", "example", "hoon"}, {"sys", "kelvin"}, {"hello", "txt"}})
	if err != nil || !reflect.DeepEqual(got, []string{"app/example.hoon", "hello.txt", "sys.kelvin"}) {
		t.Fatal(got, err)
	}
	for _, paths := range [][][]string{nil, {{"single"}}, {{"..", "escape", "hoon"}}, {{"x/y", "hoon"}}, {{"foo", "bar'"}}, {{"hello", "txt"}, {"hello", "txt"}}, {{"foo", "txt"}, {"foo.txt", "child", "txt"}}} {
		if _, err = UnixPaths(paths); err == nil {
			t.Fatal("unsafe path accepted", paths)
		}
	}
	if _, err = ListDesks("~zod/other"); err == nil {
		t.Fatal("ship injection")
	}
	if _, err = FilePaths("~zod", "base", 0); err == nil {
		t.Fatal("mutable revision accepted")
	}
	if _, err = AutocommitActive("~zod\n+code"); err == nil {
		t.Fatal("ship injection")
	}
}

func TestDeskNamesBoundedAndCanonical(t *testing.T) {
	names := []string{"zeta", "base"}
	got, err := NormalizeDeskNames(names)
	if err != nil || !reflect.DeepEqual(got, []string{"base", "zeta"}) || names[0] != "zeta" {
		t.Fatal(got, err)
	}
	for _, bad := range [][]string{nil, {"base", "base"}, {"%base"}, {"bad/path"}, make([]string, MaxDesks+1)} {
		if _, err := NormalizeDeskNames(bad); err == nil {
			t.Fatal("invalid inventory accepted", len(bad))
		}
	}
	if got, err := NormalizeDeskNames([]string{}); err != nil || got == nil {
		t.Fatal("empty inventory is valid", got, err)
	}
}
