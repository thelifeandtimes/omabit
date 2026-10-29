package main

import (
	"bytes"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/resourcepolicy"
	"testing"
)

func TestResourcePolicyCLI(t *testing.T) {
	home := t.TempDir()
	var out, errout bytes.Buffer
	for _, args := range [][]string{
		{"show"},
		{"set", "--phase", "existing", "--minimum-ram-mib", "768", "--minimum-disk-mib", "1024", "--recommended-ram-mib", "2048", "--recommended-disk-mib", "4096"},
		{"set", "--phase", "new", "--minimum-ram-mib", "4096"},
	} {
		out.Reset()
		errout.Reset()
		if rc := resourcePolicyAtHome(home, args, &out, &errout); rc != 0 {
			t.Fatal(args, rc, errout.String())
		}
	}
	p, err := resourcepolicy.Load(home)
	if err != nil || p.New.Minimum.RAMMiB != 4096 || p.Existing.Minimum.RAMMiB != 768 || p.Existing.Recommended.DiskMiB != 4096 {
		t.Fatal(p, err)
	}
	for _, args := range [][]string{
		{"set", "--phase", "existing", "--minimum-ram-mib", "0"},
		{"set", "--phase", "existing", "--minimum-ram-mib", "9000"},
		{"set", "--phase", "other", "--minimum-ram-mib", "1"},
		{"set", "--phase", "existing"},
		{"show", "--phase", "existing"},
		{"set", "--phase", "existing", "--minimum-ram-mib", "-1"},
	} {
		if rc := resourcePolicyAtHome(home, args, &out, &errout); rc == 0 {
			t.Fatal("invalid arguments accepted", args)
		}
	}
	after, err := resourcepolicy.Load(home)
	if err != nil || after != p {
		t.Fatal("invalid edit changed policy", err)
	}
}
