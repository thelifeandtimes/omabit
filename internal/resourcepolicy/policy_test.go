package resourcepolicy

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestMinimumAndRecommendations(t *testing.T) {
	p := Default()
	if err := p.Admit(false, 5<<30, 8<<30); err != nil {
		t.Fatal(err)
	}
	if err := p.Admit(false, 5<<30-1, 8<<30); err == nil {
		t.Fatal("boot admitted below minimum")
	}
	if err := p.Admit(true, 1<<30, 2<<30); err != nil {
		t.Fatal("recommendation incorrectly gates existing start", err)
	}
	if err := p.Admit(true, 1<<30, 2<<30-1); err == nil {
		t.Fatal("existing start admitted below disk minimum")
	}
	p.Existing.Recommended.RAMMiB = 1
	if p.Validate() == nil {
		t.Fatal("invalid recommendations accepted")
	}
}
func TestPolicyPersistenceAndConcurrentPhaseChanges(t *testing.T) {
	home := t.TempDir()
	p, err := Load(home)
	if err != nil || p != Default() {
		t.Fatal(p, err)
	}
	if _, err := os.Stat(filepath.Dir(Path(home))); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("read created state")
	}
	var wg sync.WaitGroup
	for _, existing := range []bool{false, true} {
		wg.Add(1)
		go func(existing bool) {
			defer wg.Done()
			_, err := Update(home, func(p *Policy) error {
				if existing {
					p.Existing.Minimum.RAMMiB = 768
				} else {
					p.New.Minimum.DiskMiB = 4096
				}
				return nil
			})
			if err != nil {
				t.Error(err)
			}
		}(existing)
	}
	wg.Wait()
	p, err = Load(home)
	if err != nil || p.Existing.Minimum.RAMMiB != 768 || p.New.Minimum.DiskMiB != 4096 {
		t.Fatal(p, err)
	}
	before := p
	if _, err = Update(home, func(p *Policy) error { p.New.Minimum.RAMMiB = 0; return nil }); err == nil {
		t.Fatal("zero accepted")
	}
	p, err = Load(home)
	if err != nil || p != before {
		t.Fatal("failed edit changed policy", p, err)
	}
	i, _ := os.Stat(Path(home))
	if i.Mode().Perm() != 0600 {
		t.Fatal(i.Mode())
	}
}
func TestMalformedAndLinkedPoliciesRejected(t *testing.T) {
	for _, fault := range []string{"json", "unknown", "trailing", "version", "symlink", "directory", "permissions", "hardlink"} {
		t.Run(fault, func(t *testing.T) {
			home := t.TempDir()
			if _, err := Update(home, func(*Policy) error { return nil }); err != nil {
				t.Fatal(err)
			}
			p := Path(home)
			switch fault {
			case "json":
				os.WriteFile(p, []byte("{"), 0600)
			case "unknown":
				os.WriteFile(p, []byte(`{"version":1,"unexpected":true}`), 0600)
			case "trailing":
				f, _ := os.OpenFile(p, os.O_APPEND|os.O_WRONLY, 0)
				f.WriteString("{}")
				f.Close()
			case "version":
				os.WriteFile(p, []byte(`{"version":2}`), 0600)
			case "symlink":
				os.Remove(p)
				os.Symlink(filepath.Join(home, "elsewhere"), p)
			case "directory":
				os.Remove(p)
				os.Mkdir(p, 0700)
			case "permissions":
				os.Chmod(p, 0644)
			case "hardlink":
				os.Link(p, filepath.Join(home, "alias"))
			}
			if _, err := Load(home); err == nil {
				t.Fatal("unsafe policy accepted")
			}
		})
	}
}
