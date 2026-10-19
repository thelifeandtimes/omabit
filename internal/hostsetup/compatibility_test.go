package hostsetup

import (
	"encoding/json"
	"strings"
	"testing"
	"testing/fstest"
)

func TestPlatformNativePreflightAndGuidance(t *testing.T) {
	for _, tc := range []struct{ id, version, upstream, want string }{
		{"debian", "28.2.2+dfsg1-1", "28.2.2", "pass"},
		{"ubuntu", "5:29.9.1-1~ubuntu.24.04~noble", "29.9.1", "pass"},
		{"debian", "26.1.5+dfsg1-1", "26.1.5", "blocked"},
	} {
		t.Run(tc.id+tc.version, func(t *testing.T) {
			p, f := fixture()
			for name := range f.files {
				if strings.HasPrefix(name, "var/lib/pacman/") {
					delete(f.files, name)
				}
			}
			f.files["etc/os-release"] = &fstest.MapFile{Mode: 0644, Data: []byte("ID=" + tc.id + "\nID_LIKE=debian\n")}
			f.files["var/lib/dpkg/status"] = &fstest.MapFile{Mode: 0644, Data: []byte("Package: docker.io\nStatus: install ok installed\nArchitecture: amd64\nVersion: " + tc.version + "\n")}
			f.files["var/lib/dpkg/info/docker.io.list"] = &fstest.MapFile{Mode: 0644, Data: []byte("/usr/sbin/dockerd\n/usr/bin/docker\n")}
			for _, name := range []string{"dockerd", "useradd", "runuser", "iptables"} {
				f.files["usr/sbin/"+name] = f.files["usr/bin/"+name]
				delete(f.files, "usr/bin/"+name)
			}
			r := p.inspect()
			check(t, r, "docker-version", tc.want)
			for _, name := range []string{"dockerd", "useradd", "runuser", "iptables"} {
				check(t, r, "binary-"+name, "pass")
			}
			if r.Compatibility == nil || r.Compatibility.Distribution.Provider != "deb" || r.Compatibility.InstalledEngineVersion != tc.upstream || r.Compatibility.Provenance == "verified" || r.RuntimeQualified {
				t.Fatalf("unsafe native evidence: %+v", r)
			}
			plan, _ := json.Marshal(Plan(r))
			if !strings.Contains(string(plan), "APT repositories") || strings.Contains(string(plan), "Omarchy repositories") || strings.Contains(string(plan), "upstream version 29.7.2") {
				t.Fatal(string(plan))
			}
			encoded, _ := json.Marshal(r)
			var legacy struct {
				SchemaVersion int     `json:"schema_version"`
				Deployment    string  `json:"deployment"`
				Checks        []Check `json:"checks"`
			}
			if json.Unmarshal(encoded, &legacy) != nil || legacy.SchemaVersion != 1 || len(legacy.Checks) != 28 || !strings.Contains(string(encoded), `"compatibility":`) {
				t.Fatal("old CLI report compatibility lost")
			}
		})
	}
}
