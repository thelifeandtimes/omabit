package hostsetup

import (
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"
)

func TestSubordinateIDs(t *testing.T) {
	for _, tc := range []struct{ name, text, uid, status string }{
		{"named", "urbit-runner:100000:65536\n", "942", "pass"},
		{"numeric", "942:100000:65536\n", "942", "pass"},
		{"numeric-leading-zero", "000942:100000:65536\n", "942", "unknown"},
		{"unrelated-numeric-leading-zero", "urbit-runner:100000:65536\n001000:200000:65536\n", "942", "unknown"},
		{"numeric-zero-leading-zero", "00:200000:65536\nurbit-runner:100000:65536\n", "942", "unknown"},
		{"contiguous-mixed-owners", "942:132768:32768\nurbit-runner:100000:32768\n", "942", "pass"},
		{"non-contiguous", "urbit-runner:100000:32768\n942:132769:32768\n", "942", "missing"},
		{"too-small", "urbit-runner:100000:65535\n", "942", "missing"},
		{"missing", "private-owner:100000:65536\n", "942", "missing"},
		{"empty", "", "942", "missing"},
		{"comments", "# private comment\n\nurbit-runner:100000:65536\n", "942", "pass"},
		{"unknown-uid", "urbit-runner:100000:65536\n", "", "unknown"},
		{"touching-not-overlapping", "private-owner:34464:65536\nurbit-runner:100000:65536\nother:165536:65536\n", "942", "pass"},
		{"other-before-target", "private-owner:99999:2\nurbit-runner:100000:65536\n", "942", "blocked"},
		{"other-after-target", "urbit-runner:100000:65536\nprivate-owner:165535:2\n", "942", "blocked"},
		{"numeric-owner-overlap", "private-owner:100000:65536\n942:100001:65536\n", "942", "blocked"},
		{"target-name-numeric-overlap", "urbit-runner:100000:65536\n942:100000:65536\n", "942", "blocked"},
		{"duplicate-target", strings.Repeat("urbit-runner:100000:65536\n", 2), "942", "blocked"},
		{"overlapping-small-target-ranges", "urbit-runner:100000:32768\n942:120000:32768\n", "942", "blocked"},
		{"nested-other-before-target", "private-owner:100000:200000\nother:110000:100\nurbit-runner:200000:65536\n", "942", "blocked"},
		{"unrelated-overlaps", "private-owner:1:100\nother:1:1000\nurbit-runner:100000:65536\n", "942", "pass"},
		{"malformed-unrelated", "urbit-runner:100000:65536\nprivate-malformed-entry\n", "942", "unknown"},
		{"extra-field", "urbit-runner:100000:65536:extra\n", "942", "unknown"},
		{"empty-owner", ":100000:65536\n", "942", "unknown"},
		{"owner-whitespace", "urbit-runner x:100000:65536\n", "942", "unknown"},
		{"numeric-owner-overflow", "4294967296:100000:65536\n", "942", "unknown"},
		{"negative-start", "urbit-runner:-1:65536\n", "942", "unknown"},
		{"signed-start", "urbit-runner:+100000:65536\n", "942", "unknown"},
		{"negative-count", "urbit-runner:100000:-65536\n", "942", "unknown"},
		{"zero-count", "urbit-runner:100000:0\n", "942", "unknown"},
		{"nonnumeric", "urbit-runner:unknown:65536\n", "942", "unknown"},
		{"uint32-start-overflow", "urbit-runner:4294967296:65536\n", "942", "unknown"},
		{"uint64-overflow", "urbit-runner:18446744073709551616:65536\n", "942", "unknown"},
		{"sum-overflow", "urbit-runner:4294960000:65536\n", "942", "unknown"},
		{"count-overflow", "urbit-runner:100000:4294967296\n", "942", "unknown"},
		{"reserved-id", "urbit-runner:4294967295:1\n", "942", "unknown"},
		{"last-valid-range", "urbit-runner:4294901759:65536\n", "942", "pass"},
		{"root-id", "urbit-runner:0:1\n", "942", "blocked"},
		{"own-id", "urbit-runner:942:65536\n", "942", "blocked"},
		{"desktop-id", "942:1000:65536\n", "942", "blocked"},
		{"small-extra-host-collision", "urbit-runner:100000:65536\n942:942:1\n", "942", "blocked"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			status, detail := subids(tc.text, tc.uid, []uint64{0, 942, 1000})
			if status != tc.status {
				t.Fatalf("got %s (%s), want %s", status, detail, tc.status)
			}
			if strings.Contains(detail, "private-") || strings.Contains(detail, "100000") {
				t.Fatal("raw subordinate-ID data leaked")
			}
		})
	}
	if status, _ := subids("urbit-runner:100000:65536\n", "942", nil); status != "unknown" {
		t.Fatal("missing host ID inventory accepted")
	}
	if status, _ := subids("urbit-runner:0:65536\n", "942", []uint64{}); status != "blocked" {
		t.Fatal("root mapping accepted without a local root entry")
	}
	for _, path := range []string{"etc/subuid", "etc/subgid"} {
		t.Run("inspect-"+path, func(t *testing.T) {
			p, f := fixture()
			f.files[path].Data = []byte("942:100000:65536\nprivate-owner:100001:1\n")
			check(t, p.inspect(), strings.TrimPrefix(path, "etc/"), "blocked")
		})
	}
}

func TestInspectSubordinateHostIDCollisions(t *testing.T) {
	for _, tc := range []struct{ name, kind, span, passwd, group, status string }{
		{"root-uid", "subuid", "0:65536", "", "", "blocked"},
		{"root-gid", "subgid", "0:65536", "", "", "blocked"},
		{"own-uid", "subuid", "942:65536", "", "", "blocked"},
		{"own-gid", "subgid", "942:65536", "", "", "blocked"},
		{"desktop-uid", "subuid", "1000:65536", "", "", "blocked"},
		{"desktop-primary-gid-without-group-entry", "subgid", "1000:65536", "", "", "blocked"},
		{"group-only-gid", "subgid", "200000:65536", "", "private-group:x:240000:\n", "blocked"},
		{"passwd-only-primary-gid", "subgid", "200000:65536", "private-login:x:700000:240000::/private:/bin/false\n", "", "blocked"},
		{"uid-not-gid", "subgid", "200000:65536", "private-login:x:240000:700000::/private:/bin/false\n", "", "pass"},
		{"gid-not-uid", "subuid", "200000:65536", "", "private-group:x:240000:\n", "pass"},
		{"last-uid-in-range", "subuid", "200000:65536", "private-login:x:265535:700000::/private:/bin/false\n", "", "blocked"},
		{"uid-at-exclusive-end", "subuid", "200000:65536", "private-login:x:265536:700000::/private:/bin/false\n", "", "pass"},
		{"uid-before-start", "subuid", "200000:65536", "private-login:x:199999:700000::/private:/bin/false\n", "", "pass"},
		{"last-gid-in-range", "subgid", "200000:65536", "", "private-group:x:265535:\n", "blocked"},
		{"gid-at-exclusive-end", "subgid", "200000:65536", "", "private-group:x:265536:\n", "pass"},
		{"gid-before-start", "subgid", "200000:65536", "", "private-group:x:199999:\n", "pass"},
	} {
		for _, owner := range []string{"urbit-runner", "942"} {
			t.Run(tc.name+"/"+owner, func(t *testing.T) {
				p, f := fixture()
				f.files["etc/passwd"].Data = append(f.files["etc/passwd"].Data, tc.passwd...)
				f.files["etc/group"].Data = append(f.files["etc/group"].Data, tc.group...)
				f.files["etc/"+tc.kind].Data = []byte(owner + ":" + tc.span + "\n")
				c := check(t, p.inspect(), tc.kind, tc.status)
				if strings.Contains(c.Detail, "private-") || strings.Contains(c.Detail, tc.span) {
					t.Fatal("host identity or range contents leaked")
				}
				for _, path := range []string{"etc/passwd", "etc/group"} {
					reads := 0
					for _, opened := range f.opened {
						if opened == path {
							reads++
						}
					}
					if reads != 1 {
						t.Fatalf("%s: got %d reads, want one shared bounded read", path, reads)
					}
				}
			})
		}
	}
	t.Run("own-primary-gid-distinct-from-uid", func(t *testing.T) {
		p, f := fixture()
		f.files["etc/passwd"].Data = []byte("urbit-runner:x:942:1942::/var/lib/urbit-runner:/usr/bin/nologin\n")
		f.files["etc/subgid"].Data = []byte("942:1942:65536\n")
		check(t, p.inspect(), "subgid", "blocked")
	})
	for _, kind := range []string{"subuid", "subgid"} {
		t.Run(kind+"/noncanonical-owner", func(t *testing.T) {
			p, f := fixture()
			f.files["etc/"+kind].Data = []byte("000942:200000:65536\n")
			check(t, p.inspect(), kind, "unknown")
		})
		t.Run(kind+"/root-without-local-root-entry", func(t *testing.T) {
			p, f := fixture()
			f.files["etc/passwd"].Data = []byte("urbit-runner:x:942:1942::/var/lib/urbit-runner:/usr/bin/nologin\n")
			f.files["etc/group"].Data = []byte("urbit-runner:x:1942:\n")
			f.files["etc/"+kind].Data = []byte("urbit-runner:0:1\nurbit-runner:200000:65536\n")
			check(t, p.inspect(), kind, "blocked")
		})
	}
}

func TestSubordinateIDsRequireCompleteHostInventories(t *testing.T) {
	for _, path := range []string{"etc/passwd", "etc/group"} {
		for _, failure := range []string{"absent", "denied", "malformed-tail", "invalid-id", "oversized", "special-file"} {
			t.Run(path+"/"+failure, func(t *testing.T) {
				p, f := fixture()
				switch failure {
				case "absent":
					delete(f.files, path)
				case "denied":
					f.denied[path] = true
				case "malformed-tail":
					f.files[path].Data = append(f.files[path].Data, "private-malformed-tail\n"...)
				case "invalid-id":
					line := "private-login:x:700000:invalid::/private:/bin/false\n"
					if path == "etc/group" {
						line = "private-group:x:invalid:\n"
					}
					f.files[path].Data = append(f.files[path].Data, line...)
				case "oversized":
					f.files[path].Data = []byte(strings.Repeat("x", maxRead+1))
				case "special-file":
					f.files[path].Mode = fs.ModeNamedPipe | 0600
				}
				r := p.inspect()
				check(t, r, "subgid", "unknown")
				status := "pass"
				if path == "etc/passwd" {
					status = "unknown"
				}
				check(t, r, "subuid", status)
			})
		}
	}
}

func TestPackageVersion(t *testing.T) {
	for _, tc := range []struct{ name, text, status string }{
		{"epoch-match", "%NAME%\ndocker\n\n%VERSION%\n1:29.7.2-1\n", "pass"},
		{"plain-match", "%VERSION%\n29.7.2-2\n\n%NAME%\ndocker\n", "pass"},
		{"decimal-release", "%NAME%\ndocker\n\n%VERSION%\n29.7.2-1.1\n", "pass"},
		{"compatible-older", "%NAME%\ndocker\n\n%VERSION%\n1:28.5.0-1\n", "pass"},
		{"compatible-newer", "%NAME%\ndocker\n\n%VERSION%\n1:29.9.1-1\n", "pass"},
		{"unsupported", "%NAME%\ndocker\n\n%VERSION%\n1:30.0.0-1\n", "blocked"},
		{"prerelease", "%NAME%\ndocker\n\n%VERSION%\n29.9.1-rc1-1\n", "unknown"},
		{"missing-name", "%VERSION%\n1:29.7.2-1\n", "unknown"},
		{"missing-version", "%NAME%\ndocker\n", "unknown"},
		{"duplicate-name", "%NAME%\ndocker\n\n%NAME%\ndocker\n\n%VERSION%\n29.7.2-1\n", "unknown"},
		{"duplicate-version", "%NAME%\ndocker\n\n%VERSION%\n29.7.2-1\n\n%VERSION%\n29.7.2-1\n", "unknown"},
		{"truncated", "%NAME%\ndocker\n\n%VERSION%", "unknown"},
		{"empty", "", "unknown"},
		{"multiple-values", "%NAME%\ndocker\nunexpected\n\n%VERSION%\n29.7.2-1\n", "unknown"},
		{"bad-epoch", "%NAME%\ndocker\n\n%VERSION%\nepoch:29.7.2-1\n", "unknown"},
		{"long-version", "%NAME%\ndocker\n\n%VERSION%\n" + strings.Repeat("0", 129) + ".7.2-1\n", "unknown"},
		{"bad-upstream", "%NAME%\ndocker\n\n%VERSION%\nprivate.7.2-1\n", "unknown"},
		{"non-version", "%NAME%\ndocker\n\n%VERSION%\nprivate-1\n", "unknown"},
		{"too-large", strings.Repeat("x", maxRead+1), "unknown"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p, f := fixture()
			f.files["var/lib/pacman/local/docker-29.7.2-1/desc"].Data = []byte(tc.text)
			status, detail := p.engineVersion()
			if status != tc.status {
				t.Fatalf("got %s (%s), want %s", status, detail, tc.status)
			}
			if strings.Contains(detail, "private") || strings.Contains(detail, "secret") {
				t.Fatal("unvalidated package data leaked")
			}
		})
	}
	for _, tc := range []string{"absent-database", "denied-database", "special-database", "duplicate", "missing-desc", "special-desc"} {
		t.Run(tc, func(t *testing.T) {
			p, f := fixture()
			switch tc {
			case "absent-database":
				f.files = fstest.MapFS{}
			case "denied-database":
				f.denied["var/lib/pacman/local"] = true
			case "special-database":
				f.files["var/lib/pacman/local"] = &fstest.MapFile{Mode: fs.ModeNamedPipe | 0600}
			case "duplicate":
				f.files["var/lib/pacman/local/docker-29.7.2-2/desc"] = f.files["var/lib/pacman/local/docker-29.7.2-1/desc"]
				f.files["var/lib/pacman/local/docker-29.7.2-2/files"] = f.files["var/lib/pacman/local/docker-29.7.2-1/files"]
			case "missing-desc":
				delete(f.files, "var/lib/pacman/local/docker-29.7.2-1/desc")
				f.files["var/lib/pacman/local/docker-29.7.2-1"] = &fstest.MapFile{Mode: fs.ModeDir | 0755}
			case "special-desc":
				f.files["var/lib/pacman/local/docker-29.7.2-1/desc"].Mode = fs.ModeNamedPipe | 0600
			}
			status, detail := p.engineVersion()
			if status != "unknown" {
				t.Fatalf("got %s (%s), want unknown", status, detail)
			}
			for _, path := range f.opened {
				if tc == "special-database" && path == "var/lib/pacman/local" {
					t.Fatal("special file opened as package database directory")
				}
			}
		})
	}
}
