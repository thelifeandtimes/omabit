package managed

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"syscall"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/access"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/collection"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

// These are logical SQLite exports of original historical writers, not current
// Go objects with old version markers. Blob cells preserve original bytes. The
// generator, pinned source commits, source tree IDs and coverage gaps live beside
// the fixtures; ordinary tests require neither Git nor historical compilation.
type g02Fixture struct {
	SourceSnapshotSHA256     string `json:"source_snapshot_sha256"`
	UnreleasedSourceSnapshot bool   `json:"unreleased_source_snapshot"`
	SourceBase               string `json:"source_base"`
	Format                   int
	Name                     string
	Schema                   int
	Synthetic                bool
	SourceCommit             string `json:"source_commit"`
	SourceTree               string `json:"source_tree"`
	SourceSupplementCommit   string `json:"source_supplement_commit"`
	SourceSupplementTree     string `json:"source_supplement_tree"`
	SourceBundleSHA256       string `json:"source_bundle_sha256"`
	WriterSHA256             string `json:"writer_sha256"`
	RunnerID                 string `json:"runner_id"`
	CommitmentBytes          []byte `json:"commitment_bytes"`
	Tables                   []g02Table
	Hashes                   map[string]string
	Files                    []struct {
		Path   string
		Mode   os.FileMode
		Body   []byte
		SHA256 string
	}
}

type g02Table struct {
	Name    string
	SQL     string
	Columns []string
	Rows    [][]any
}

type g02Release struct {
	Release      string
	Schema       int
	Fixture      string
	SourceCommit string `json:"source_commit"`
	HostUpgrade  string `json:"host_upgrade"`
}

type g02Target struct {
	Release     string
	Channel     string
	StateSchema int `json:"state_schema"`
	Evidence    string
}

type g02Manifest struct {
	BaselineFixtures      []string `json:"baseline_fixtures"`
	Format                int
	QualificationTarget   g02Target    `json:"qualification_target"`
	DevelopmentTestTarget g02Target    `json:"development_test_target"`
	AcceptedSchemaWindow  []int        `json:"accepted_schema_window"`
	SupportedReleaseScope string       `json:"supported_release_scope"`
	SupportedReleases     []g02Release `json:"supported_releases"`
	HostUpgrade           struct {
		InitialSchemas []int `json:"initial_schemas"`
		Deployment     string
		Procedure      string
		Evidence       string
	} `json:"host_upgrade"`
	PrototypeSchemas   []int    `json:"prototype_schemas"`
	PreviewFixtures    []string `json:"preview_fixtures"`
	Fixtures           []string
	UnavailableWriters map[string]string `json:"unavailable_writers"`
}

func g02ManifestData(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile("testdata/historical-upgrades/manifest.json")
	g02Check(t, err)
	return data
}

// Every supported RELEASE needs its own original writer, even when several
// releases share a schema. Reader acceptance of a prototype is not qualification
// of that prototype, and database coverage is not a host-install support promise.
func g02ValidateManifest(m g02Manifest, fixtures map[string]g02Fixture) error {
	target := m.QualificationTarget
	if m.Format != 2 || target.Release != "v0.1.0-rc.12" || target.Channel != "prerelease" || target.StateSchema != 20 || target.Evidence != "source-tests-only" {
		return errors.New("preserve historical RC12/schema20 source qualification separately from current development tests")
	}
	development := m.DevelopmentTestTarget
	if currentSchema != 47 || development.StateSchema != currentSchema || development.Release != "unreleased" || development.Channel != "development" || development.Evidence != "source-tests-only" {
		return errors.New("review the development schema target explicitly; source tests are not release/artifact qualification")
	}
	if !slices.Equal(m.AcceptedSchemaWindow, []int{11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37, 38, 39, 40, 41, 42, 43, 44, 45}) || !slices.Equal(m.PrototypeSchemas, []int{11, 12, 14, 17, 18}) || m.SupportedReleaseScope != "database-reader" {
		return errors.New("reader window, release support and unpublished prototypes must remain distinct")
	}
	if !slices.Equal(m.HostUpgrade.InitialSchemas, []int{19}) || m.HostUpgrade.Deployment != "owner-user-linux-amd64" || m.HostUpgrade.Procedure != "docs/management-offline-upgrade.md" || m.HostUpgrade.Evidence != "source-tests-only" {
		return errors.New("offline host support requires original schema19 and the reviewed owner-user procedure")
	}
	listed := map[string]bool{}
	covered := map[int]bool{}
	for _, name := range m.Fixtures {
		f, ok := fixtures[name]
		if !ok || listed[name] || name != f.Name+".json" || !slices.Contains(m.AcceptedSchemaWindow, f.Schema) {
			return fmt.Errorf("missing, duplicate or invalid original fixture %s", name)
		}
		listed[name], covered[f.Schema] = true, true
	}
	if len(listed) != len(fixtures) {
		return errors.New("unlisted historical fixture")
	}
	// Freeze the published RC.1-12 inventory, not merely the number of schemas.
	// A future support-window change must explicitly review this release gate.
	required := map[string]string{}
	for n := 1; n <= 12; n++ {
		required[fmt.Sprintf("v0.1.0-rc.%d", n)] = fmt.Sprintf("rc%d.json", n)
	}
	used := map[string]bool{}
	sources := map[string]bool{}
	for _, release := range m.SupportedReleases {
		name, ok := required[release.Release]
		f := fixtures[release.Fixture]
		if !ok || name != release.Fixture || !listed[name] || used[name] || sources[release.SourceCommit] || f.Schema != release.Schema || f.SourceCommit != release.SourceCommit || len(release.SourceCommit) != 40 || slices.Contains(m.PrototypeSchemas, release.Schema) {
			return fmt.Errorf("supported release %s lacks its own pinned original fixture", release.Release)
		}
		if name == "rc12.json" && (f.Schema != 20 || f.SourceCommit != "1529c1335754d15c6efd03d77ce2af7ede3d48f0" || f.SourceTree != "c623bda87c95ecfa9a8808945a49520ac7ec93b2") {
			return errors.New("RC12 must use its published schema20 writer, not relabeled development state")
		}
		wantHost := "unsupported"
		if slices.Contains(m.HostUpgrade.InitialSchemas, release.Schema) {
			wantHost = "owner-user-schema19-offline"
		} else if release.Schema == 20 {
			wantHost = "owner-user-quiescence-online"
		}
		if release.HostUpgrade != wantHost {
			return fmt.Errorf("%s overstates or omits host support", release.Release)
		}
		used[name] = true
		sources[release.SourceCommit] = true
		delete(required, release.Release)
	}
	if len(required) != 0 {
		return fmt.Errorf("published supported releases missing original coverage: %v", required)
	}
	if !slices.Equal(m.PreviewFixtures, []string{"preview14.json", "preview18.json"}) {
		return errors.New("retain the available original previews separately from releases")
	}
	for _, name := range m.PreviewFixtures {
		if !listed[name] || used[name] || !slices.Contains(m.PrototypeSchemas, fixtures[name].Schema) {
			return fmt.Errorf("invalid preview classification %s", name)
		}
		used[name] = true
	}
	if !slices.Equal(m.BaselineFixtures, []string{"schema21.json", "schema22.json", "schema23.json", "schema24.json", "schema25.json", "schema26.json", "schema27.json", "schema28.json", "schema29.json", "schema30.json", "schema31.json", "schema32.json", "schema33.json", "schema34.json", "schema35.json", "schema36.json", "schema37.json", "schema38.json", "schema39.json", "schema40.json", "schema41.json", "schema42.json", "schema43.json", "schema44.json", "schema45.json"}) {
		return errors.New("retain the pinned schema21 development baseline separately from release claims")
	}
	baseline := fixtures["schema21.json"]
	if !listed["schema21.json"] || baseline.Schema != 21 || baseline.SourceCommit != "b23312876c611e1d083bdcc9468469f7e716d749" {
		return errors.New("schema21 baseline lacks its original writer")
	}
	used["schema21.json"] = true
	for version, digest := range map[int]string{22: "4f5029ac1469d6cb8b74c99e3fe88eb6932ee2f561460e7a48dd02834b6f8ed5", 23: "4dcd826f1d3268cfe1eb66abd586cb599266b04aff9b543677cc647db1bfb2dc", 24: "69b23e268b73cb87db73b825d43c9692feeced18b30e5c8674d0c2b05f5edf54", 25: "ea439418502436ac149c48d01316135834f762b67e629b84c0b410ef89639dd9", 26: "a5f0af92d9ab4d504879a732f315e8902d67d0e59312e2b504aaec9c32db5ecd", 27: "221f4e7ba757c65339759423c8d5f43232b65cd309755e9f4378ceeeca441904", 28: "8386021e437e4d586b06695d69f3877c65d9cb5a7d81105ae04a61c1e3d13be3", 29: "8527e8465d3fc2330bc6ba661577b6d3bc776b75da36c301f4557be6550a0331", 30: "b8aa477ce568abee0c607b3375686b143fa04e69f62a90c389631b383114dea8", 31: "74dcd3b8847474f3d2772af0177e0c85573f5a231fdab958b9a3f9af4829ebf9", 32: "3af14bd5559188c9293a23ada1124f246606447935445cd438e6a5fb4305d7eb", 33: "ce58e7e73ec6312b34f68ae09d8348a1edcdbd562b61b420d8a09dd408fff30b", 34: "0e4893ae8fe8e9fbb486e3a34ccea1a5ec0d6717cfafa2a174331c7a519dceb4", 35: "ea67857247fccc7b175e6bb4c54fc7110604b0f195850475781fca6b54cc3eba", 36: "17340ebdb296525407a95e004b4779935df2e25f3acec3ae47f861b7bd651c2b", 37: "57ed6461b9cd12f595a20139c3643e76cd79bdfd1d336b3d69c8a86547feec04", 38: "8b363b4c25c8f256b017abbeaedb0bb5b9be5e4d85f52d53778b1df90eea692b", 39: "e9afaa95aa2d4ad0531baee741925c4fb8e823ce2644118888bcb7633fb3d356", 40: "052de08d5f4aaec592af98396bf457c3b21bc16847e14d59ae1ae6843e58e14f", 41: "eb4bd69f6ac1bb692a83768d84f58ea00afb4c1ba3096a637f215c200b50beb2", 42: "5f743bba5b7df16e4bbaa6ccb3f96582e357c12c313882c154c959a64947ccf3", 43: "ff7e9860bc98dbb765e4e2f98535a77a4e8b0898d79327494c1d17b912876222", 44: "80e34af6c6569e9c2bd37a0589cb674261524f6dc1eec176f480afca2aece1a6", 45: "b366caf7df4c488816cb69b24fbeefb082d6b2618283ab76fe2d721a06496da9"} {
		name := fmt.Sprintf("schema%d.json", version)
		snapshot := fixtures[name]
		if !listed[name] || snapshot.Schema != version || !snapshot.UnreleasedSourceSnapshot || snapshot.SourceBase != baseline.SourceCommit || snapshot.SourceCommit != "" || snapshot.SourceTree != "" || snapshot.SourceSnapshotSHA256 != digest {
			return fmt.Errorf("schema%d requires its frozen unreleased source, not a relabeled release", version)
		}
		used[name] = true
	}
	if len(used) != len(listed) {
		return errors.New("every fixture must identify a supported release or an unpublished preview")
	}
	for _, version := range m.AcceptedSchemaWindow {
		gap := m.UnavailableWriters[fmt.Sprint(version)]
		if covered[version] == (strings.TrimSpace(gap) != "") {
			return fmt.Errorf("schema%d must have original coverage or an explicit unavailable-writer gap", version)
		}
	}
	for version, reason := range m.UnavailableWriters {
		if version != "11" && version != "12" && version != "17" || !strings.HasPrefix(reason, "Unpublished prototype:") {
			return fmt.Errorf("unreviewed unavailable-writer classification %s", version)
		}
	}
	return nil
}

func g02Check(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func g02Fixtures(t *testing.T) []g02Fixture {
	t.Helper()
	dir := "testdata/historical-upgrades"
	var manifest g02Manifest
	g02Check(t, json.Unmarshal(g02ManifestData(t), &manifest))
	var out []g02Fixture
	fixtures := map[string]g02Fixture{}
	for _, name := range manifest.Fixtures {
		if filepath.Base(name) != name {
			t.Fatal("invalid fixture name")
		}
		data, err := os.ReadFile(filepath.Join(dir, name))
		g02Check(t, err)
		limit := 96 << 10
		if name == "schema37.json" || name == "schema38.json" || name == "schema39.json" || name == "schema40.json" || name == "schema41.json" || name == "schema42.json" || name == "schema43.json" || name == "schema44.json" || name == "schema45.json" {
			limit = 1536 << 10
		}
		if name == "schema36.json" {
			limit = 1024 << 10
		}
		if name == "schema35.json" {
			limit = 768 << 10
		} // Repeated replacements and retained earlier mirrors.
		if name == "schema34.json" {
			limit = 384 << 10
		} // Complete current Stop plus original lifecycle mirrors.
		if name == "schema33.json" {
			limit = 320 << 10
		} // Complete replacement readiness and original lifecycle mirrors.
		if name == "schema32.json" {
			limit = 320 << 10
		} // Complete replacement and original retirement/exit/Stop mirrors.
		if name == "schema31.json" {
			limit = 224 << 10
		} // Complete retirement and immutable prior exit/Stop mirrors.
		if name == "schema30.json" {
			limit = 192 << 10
		} // Complete exit proof and immutable original Stop mirrors.
		if name == "schema29.json" {
			limit = 160 << 10
		} // Original uncertain Stop plus complete readiness mirrors.
		if name == "schema28.json" {
			limit = 160 << 10 // Original readiness proofs and receipt mirrors.
		}
		if len(data) > limit {
			t.Fatal("historical fixture exceeds its per-schema budget", name)
		}
		var f g02Fixture
		decoder := json.NewDecoder(bytes.NewReader(data))
		decoder.UseNumber()
		g02Check(t, decoder.Decode(&f))
		if f.Format != 1 || !f.Synthetic || (!f.UnreleasedSourceSnapshot && (len(f.SourceCommit) != 40 || len(f.SourceTree) != 40)) || len(f.WriterSHA256) != 64 || f.Schema < 11 || f.Schema >= currentSchema || len(f.Hashes) < 20 {
			t.Fatal("missing provenance/coverage", name)
		}
		if f.UnreleasedSourceSnapshot {
			bundle, err := os.ReadFile(filepath.Join(dir, f.Name+"-source.tar.gz"))
			g02Check(t, err)
			if fmt.Sprintf("%x", sha256.Sum256(bundle)) != f.SourceSnapshotSHA256 {
				t.Fatal("schema22 source snapshot digest mismatch")
			}
		}
		if f.Schema == 14 {
			// This is an original local pre-release snapshot, not a published
			// schema14 release. Both stash trees are necessary to compile it.
			if f.Name != "preview14" || f.SourceCommit != "9e956bbab517dc9debe8161c09100e3211df0786" || f.SourceTree != "9170c9feea349a678f8153fe106d12e20657353a" || f.SourceSupplementCommit != "043e1b971800f4e4543e6884d20ef625ec55abf4" || f.SourceSupplementTree != "e0e58b7f91a9637920e124191d7e662feb1bb584" {
				t.Fatal("schema14 original stash provenance changed")
			}
			bundle, err := os.ReadFile(filepath.Join(dir, "schema14-source.bundle"))
			g02Check(t, err)
			if f.SourceBundleSHA256 != "09e2f733ae992826ee21aeee32cc2826bed5b46cd8f9db613b747ecf0ca86556" || fmt.Sprintf("%x", sha256.Sum256(bundle)) != f.SourceBundleSHA256 {
				t.Fatal("schema14 original source bundle changed")
			}
		}
		fixtures[name] = f
		out = append(out, f)
	}
	g02Check(t, g02ValidateManifest(manifest, fixtures))
	return out
}

func TestG02HistoricalSupportManifestRejectsLostReleaseOrInventedEvidence(t *testing.T) {
	originals := g02Fixtures(t)
	for _, tc := range []struct {
		name   string
		change func(*g02Manifest, map[string]g02Fixture)
	}{
		{"missing-release-with-shared-schema", func(m *g02Manifest, _ map[string]g02Fixture) {
			m.SupportedReleases = append(m.SupportedReleases[:2], m.SupportedReleases[3:]...)
		}},
		{"missing-original-file", func(_ *g02Manifest, f map[string]g02Fixture) { delete(f, "rc3.json") }},
		{"substitute-same-schema-release", func(m *g02Manifest, _ map[string]g02Fixture) { m.SupportedReleases[2].Fixture = "rc4.json" }},
		{"changed-source", func(_ *g02Manifest, f map[string]g02Fixture) {
			rc := f["rc11.json"]
			rc.SourceCommit = strings.Repeat("0", 40)
			f["rc11.json"] = rc
		}},
		{"reused-source-for-distinct-release", func(m *g02Manifest, f map[string]g02Fixture) {
			rc := f["rc3.json"]
			rc.SourceCommit = f["rc4.json"].SourceCommit
			f["rc3.json"] = rc
			m.SupportedReleases[2].SourceCommit = rc.SourceCommit
		}},
		{"missing-rc12", func(m *g02Manifest, _ map[string]g02Fixture) { m.SupportedReleases = m.SupportedReleases[:11] }},
		{"substitute-rc12-source-and-manifest", func(m *g02Manifest, f map[string]g02Fixture) {
			rc := f["rc12.json"]
			rc.SourceCommit = strings.Repeat("0", 40)
			f["rc12.json"] = rc
			m.SupportedReleases[11].SourceCommit = rc.SourceCommit
		}},
		{"preview-as-release", func(m *g02Manifest, _ map[string]g02Fixture) {
			m.SupportedReleases[0].Fixture, m.SupportedReleases[0].Schema = "preview14.json", 14
		}},
		{"missing-prototype-gap", func(m *g02Manifest, _ map[string]g02Fixture) { delete(m.UnavailableWriters, "17") }},
		{"covered-schema-as-gap", func(m *g02Manifest, _ map[string]g02Fixture) {
			m.UnavailableWriters["14"] = "Unpublished prototype: unavailable"
		}},
		{"unsupported-host-upgrade", func(m *g02Manifest, _ map[string]g02Fixture) {
			m.SupportedReleases[2].HostUpgrade = "owner-user-schema19-offline"
		}},
		{"reader-window-as-host-window", func(m *g02Manifest, _ map[string]g02Fixture) { m.HostUpgrade.InitialSchemas = m.AcceptedSchemaWindow }},
		{"stable-claim", func(m *g02Manifest, _ map[string]g02Fixture) { m.QualificationTarget.Channel = "stable" }},
		{"relabel-rc12-schema", func(m *g02Manifest, _ map[string]g02Fixture) { m.QualificationTarget.StateSchema = currentSchema }},
		{"development-as-release", func(m *g02Manifest, _ map[string]g02Fixture) { m.DevelopmentTestTarget.Release = "v0.1.0-rc.12" }},
		{"development-as-stable", func(m *g02Manifest, _ map[string]g02Fixture) { m.DevelopmentTestTarget.Channel = "stable" }},
		{"development-as-artifact-evidence", func(m *g02Manifest, _ map[string]g02Fixture) { m.DevelopmentTestTarget.Evidence = "artifact-qualified" }},
		{"stale-development-schema", func(m *g02Manifest, _ map[string]g02Fixture) { m.DevelopmentTestTarget.StateSchema = 20 }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var m g02Manifest
			g02Check(t, json.Unmarshal(g02ManifestData(t), &m))
			fixtures := map[string]g02Fixture{}
			for _, f := range originals {
				fixtures[f.Name+".json"] = f
			}
			tc.change(&m, fixtures)
			if err := g02ValidateManifest(m, fixtures); err == nil {
				t.Fatal("invalid qualification claim passed")
			}
		})
	}
}

func TestG02HistoricalOfflineSupportBoundary(t *testing.T) {
	var m g02Manifest
	g02Check(t, json.Unmarshal(g02ManifestData(t), &m))
	for _, f := range g02Fixtures(t) {
		t.Run(f.Name, func(t *testing.T) {
			dir := g02Materialize(t, f)
			g02Check(t, os.WriteFile(filepath.Join(dir, ".lock"), nil, 0600))
			lock, err := LockOfflineState(dir, uint32(os.Geteuid()))
			g02Check(t, err)
			defer lock.Close()
			path := filepath.Join(dir, "runner.sqlite")
			before, err := os.ReadFile(path)
			g02Check(t, err)
			for pass := 0; pass < 2; pass++ {
				db, err := lock.openDB(false)
				g02Check(t, err)
				state, identityErr := inspectOfflineIdentity(db)
				g02Check(t, db.Close())
				eligible := slices.Contains(m.HostUpgrade.InitialSchemas, f.Schema)
				if eligible != (identityErr == nil) || eligible && (state.RunnerID != f.RunnerID || state.Schema != f.Schema) {
					t.Fatal("original state disagrees with offline support matrix", state, identityErr)
				}
				// These originals deliberately contain queued/interrupted work.
				// Even schema19 must be refused intact until original-runtime
				// preparation; this is not an end-to-end host activation test.
				if _, err := lock.Inspect(context.Background()); err == nil {
					t.Fatal("offline inspection accepted unresolved original state")
				}
				if _, err := lock.Fence(context.Background(), f.RunnerID, strings.Repeat("e", 64)); err == nil {
					t.Fatal("offline migration fenced unsupported or unresolved original state")
				}
				after, err := os.ReadFile(path)
				g02Check(t, err)
				if !bytes.Equal(before, after) {
					t.Fatal("offline refusal rewrote original database")
				}
			}
		})
	}
}

func g02Cell(t *testing.T, value any) any {
	t.Helper()
	switch v := value.(type) {
	case map[string]any:
		encoded, ok := v["blob"].(string)
		if !ok || len(v) != 1 {
			t.Fatal("invalid blob cell")
		}
		data, err := base64.StdEncoding.DecodeString(encoded)
		g02Check(t, err)
		return data
	case json.Number:
		n, err := v.Int64()
		g02Check(t, err)
		return n
	case nil, string:
		return v
	default:
		t.Fatalf("invalid SQLite fixture cell %T", value)
		return nil
	}
}

func g02RawDB(t *testing.T, dir string) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", filepath.Join(dir, "runner.sqlite"))
	g02Check(t, err)
	db.SetMaxOpenConns(1)
	return db
}

func g02Materialize(t *testing.T, f g02Fixture) string {
	t.Helper()
	dir := t.TempDir()
	g02Check(t, os.Chmod(dir, 0700))
	path := filepath.Join(dir, "runner.sqlite")
	g02Check(t, os.WriteFile(path, nil, 0600))
	db := g02RawDB(t, dir)
	tx, err := db.Begin()
	g02Check(t, err)
	defer tx.Rollback()
	for _, table := range f.Tables {
		_, err := tx.Exec(table.SQL) // the historical sqlite_master DDL
		g02Check(t, err)
		for _, row := range table.Rows {
			args := make([]any, len(row))
			for i, cell := range row {
				args[i] = g02Cell(t, cell)
			}
			_, err = tx.Exec("INSERT INTO "+table.Name+" VALUES ("+strings.TrimSuffix(strings.Repeat("?,", len(args)), ",")+")", args...)
			g02Check(t, err)
		}
	}
	// Restore the version read from the ORIGINAL writer alongside its original
	// DDL/rows. No current store/model participates in fixture construction.
	_, err = tx.Exec(fmt.Sprintf("PRAGMA user_version=%d", f.Schema))
	g02Check(t, err)
	g02Check(t, tx.Commit())
	g02AssertRows(t, db, f, nil)
	g02Check(t, db.Close())
	for _, file := range f.Files {
		if filepath.IsAbs(file.Path) || strings.Contains(file.Path, "..") {
			t.Fatal("unsafe fixture file path")
		}
		if fmt.Sprintf("%x", sha256.Sum256(file.Body)) != file.SHA256 {
			t.Fatal("fixture file digest mismatch", file.Path)
		}
		path := filepath.Join(dir, file.Path)
		g02Check(t, os.MkdirAll(filepath.Dir(path), 0700))
		g02Check(t, os.WriteFile(path, file.Body, file.Mode))
	}
	return dir
}

// except contains only rows reconciliation is explicitly expected to update.
func g02AssertRows(t *testing.T, db *sql.DB, f g02Fixture, except map[string]bool) {
	t.Helper()
	for _, table := range f.Tables {
		rows, err := db.Query("SELECT * FROM " + table.Name + " ORDER BY rowid")
		g02Check(t, err)
		columns, err := rows.Columns()
		g02Check(t, err)
		if !reflect.DeepEqual(columns, table.Columns) {
			t.Fatal("historical columns changed", table.Name)
		}
		n := 0
		for rows.Next() {
			if n >= len(table.Rows) {
				t.Fatal("unexpected new row", table.Name)
			}
			got := make([]any, len(columns))
			ptrs := make([]any, len(columns))
			for i := range got {
				ptrs[i] = &got[i]
			}
			g02Check(t, rows.Scan(ptrs...))
			rowKey := fmt.Sprint(got[0])
			for i, cell := range table.Rows[n] {
				want := g02Cell(t, cell)
				key := table.Name + "/" + rowKey + "/" + columns[i]
				if except[key] {
					continue
				}
				if !reflect.DeepEqual(want, got[i]) {
					t.Fatalf("historical bytes/value changed: %s", key)
				}
				if digest, ok := f.Hashes[key]; ok {
					data, ok := got[i].([]byte)
					if !ok {
						data = []byte(got[i].(string))
					}
					if fmt.Sprintf("%x", sha256.Sum256(data)) != digest {
						t.Fatal("historical receipt/body hash changed", key)
					}
				}
			}
			n++
		}
		g02Check(t, rows.Err())
		g02Check(t, rows.Close())
		if n != len(table.Rows) {
			t.Fatal("historical rows lost", table.Name)
		}
	}
}

func g02Open(t *testing.T, dir string) *store {
	t.Helper()
	s, err := openStore(dir) // actual validation, transaction, migration and WAL setup
	g02Check(t, err)
	t.Cleanup(func() { s.close() })
	s.mode = DevelopmentMode
	s.portAvailable = func(string, uint16) bool { t.Fatal("replay probed a host port"); return false }
	s.validateStorage = func(string) error {
		t.Fatal("replay inspected host storage")
		return errors.New("unexpected storage access")
	}
	var version int
	g02Check(t, s.db.QueryRow("PRAGMA user_version").Scan(&version))
	if version != currentSchema {
		t.Fatal("real open did not migrate to current schema", version)
	}
	return s
}

func g02Replay(t *testing.T, s *store, f g02Fixture) {
	t.Helper()
	for _, table := range f.Tables {
		if table.Name != "operations" {
			continue
		}
		for _, row := range table.Rows {
			id, actor, request, fingerprint := row[0].(string), row[1].(string), row[2].(string), row[3].(string)
			op, hit, err := replay(s.db, actor, request, fingerprint)
			g02Check(t, err)
			if !hit || op.ID != id {
				t.Fatal("historical receipt lost", request)
			}
			_, hit, err = replay(s.db, actor, request, fingerprint+" changed")
			var fault *fault
			if !hit || !errors.As(err, &fault) || fault.problem.Code != "request_conflict" {
				t.Fatal("changed-input replay did not conflict", request, err)
			}
			_, hit, err = replay(s.db, "unrelated-synthetic-actor", request, fingerprint)
			g02Check(t, err)
			if hit {
				t.Fatal("receipt crossed actor boundary", request)
			}
			switch op.Kind {
			case "register-fake":
				var wire struct{ Request protocol.CreateFakeRequest }
				g02Check(t, json.Unmarshal([]byte(fingerprint), &wire))
				got, hit, err := s.register(actor, wire.Request)
				g02Check(t, err)
				if !hit || got.ID != id {
					t.Fatal("registration entrypoint did not replay", request)
				}
				wire.Request.Label += " changed"
				_, _, err = s.register(actor, wire.Request)
				if !errors.As(err, &fault) || fault.problem.Code != "request_conflict" {
					t.Fatal("registration entrypoint missed historical conflict", err)
				}
			case "start", "stop":
				var wire struct {
					ID      string
					Request protocol.OperationRequest
				}
				g02Check(t, json.Unmarshal([]byte(fingerprint), &wire))
				got, hit, err := s.accept(actor, wire.ID, wire.Request)
				g02Check(t, err)
				if !hit || got.ID != id || got.Status != op.Status {
					t.Fatal("operation entrypoint did not replay", request)
				}
				wire.Request.ExpectedRevision++
				_, _, err = s.accept(actor, wire.ID, wire.Request)
				if !errors.As(err, &fault) || fault.problem.Code != "request_conflict" {
					t.Fatal("operation entrypoint missed historical conflict", err)
				}
			}
		}
	}
}

func TestG02HistoricalUpgradePreservesOriginalBytesAndReplay(t *testing.T) {
	for _, f := range g02Fixtures(t) {
		t.Run(f.Name, func(t *testing.T) {
			dir := g02Materialize(t, f)
			for pass := 0; pass < 3; pass++ {
				s := g02Open(t, dir)
				if s.runnerID != f.RunnerID {
					t.Fatal("runner identity changed")
				}
				g02AssertRows(t, s.db, f, nil)
				g02Replay(t, s, f)
				g02AssertRows(t, s.db, f, nil)
				snapshot, err := s.snapshot()
				g02Check(t, err)
				if len(snapshot.Instances) != 8 {
					t.Fatal("inventory lost historical instances")
				}
				enrolled, err := loadInstance(s.db, "enrolled")
				g02Check(t, err)
				if enrolled.Public.Ship != "~sampel-palnet" || enrolled.Public.DesiredState != "stopped" || enrolled.Runtime.Storage.Identity.Inode != 500 || !enrolled.Runtime.Adoption.Verified || enrolled.Public.AmesPublished != (enrolled.Runtime.Boot != nil) {
					t.Fatal("historical enrollment/network contract changed", enrolled.Public)
				}
				for _, id := range []string{"running", "stopped"} {
					v, err := loadInstance(s.db, id)
					g02Check(t, err)
					if v.Public.DesiredState != id || v.Public.ObservedState != id || v.Public.ConfigRevision != 1 || v.Public.AppliedRevision != 1 || v.Runtime.Storage.Identity.Inode == 0 {
						t.Fatal("historical config/intent/identity changed", id, v.Public)
					}
					p, err := loadRoutinePolicy(s.db, id)
					g02Check(t, err)
					if p == nil || !p.Config.Enabled || p.Config.StopRunning != (id == "stopped") || p.Config.Resume != (id == "stopped") {
						t.Fatal("historical downtime consent changed", p)
					}
					if f.Schema < 18 && (v.Runtime.Loom != 0 || v.Runtime.NativeControl || v.Public.Config.Loom != 31) {
						t.Fatal("legacy launch defaults were rewritten")
					}
					for _, c := range v.Runtime.Runtime.Containers {
						if c.EngineVersion != "" {
							t.Fatal("historical container acquired invented engine-version evidence", id)
						}
					}
				}
				moved, err := loadInstance(s.db, "moved")
				g02Check(t, err)
				h := moved.Runtime.Handoff
				if h == nil || !h.Fenced() || !h.ValidCommitment(h.Commitment) || !bytes.Equal(h.CommitmentBytes(), f.CommitmentBytes) {
					t.Fatal("original signature bytes or permanent source fence lost")
				}
				key, err := hex.DecodeString(h.VerifyKey)
				g02Check(t, err)
				if len(key) != 32 || (h.RuntimeSettings != nil) != (f.Schema >= 19) {
					t.Fatal("historical signature contract changed")
				}
				g02AssertRows(t, s.db, f, nil)
				g02Check(t, s.close())
			}
			g02Sidecars(t, dir, f)
		})
	}
}

func g02Sidecars(t *testing.T, dir string, f g02Fixture) {
	t.Helper()
	info, err := os.Stat(filepath.Join(dir, "runner.sqlite"))
	g02Check(t, err)
	if info.Mode().Perm() != 0600 || info.Sys().(*syscall.Stat_t).Uid != uint32(os.Geteuid()) {
		t.Fatal("upgraded database owner/mode changed")
	}
	registry, err := collection.Open(filepath.Join(dir, "collector"))
	g02Check(t, err)
	entries, err := registry.List(context.Background())
	g02Check(t, err)
	if len(entries) != 1 || entries[0].RunnerID != f.RunnerID || entries[0].Revision != 3 || entries[0].SSH.Host != "g02.invalid" {
		t.Fatal("historical enrollment/pin changed", entries)
	}
	if f.Schema >= 14 {
		roots, err := pierstore.Roots(dir)
		g02Check(t, err)
		if len(roots) != 2 || roots[1].Path != "/synthetic/g02/custom-piers" || roots[1].Identity.Inode != 90 {
			t.Fatal("custom root mapping changed", roots)
		}
	}
	for _, file := range f.Files {
		path := filepath.Join(dir, file.Path)
		data, err := os.ReadFile(path)
		g02Check(t, err)
		if !bytes.Equal(file.Body, data) {
			t.Fatal("historical sidecar bytes changed", file.Path)
		}
		info, err := os.Stat(path)
		g02Check(t, err)
		if info.Mode().Perm() != file.Mode || info.Sys().(*syscall.Stat_t).Uid != uint32(os.Geteuid()) {
			t.Fatal("private owner/mode changed", file.Path)
		}
		if file.Path == "authorized_keys" {
			g, err := access.ParseLine(strings.TrimSpace(string(data)))
			g02Check(t, err)
			loaded, err := access.Load(path, g.Fingerprint)
			g02Check(t, err)
			if loaded.Role != "operator" {
				t.Fatal("historical access grant lost")
			}
		}
	}
}

// Opt-in driver bridge for testing original readers against real upgraded state.
// Ordinary Go test never writes outside t.TempDir. The driver supplies its own
// private temporary export directory and compiles the unchanged original reader.
func TestG02HistoricalExportForOldReaders(t *testing.T) {
	out := os.Getenv("G02_EXPORT_DIR")
	if out == "" {
		t.Skip("only used by scripts/test-historical-upgrades.py")
	}
	info, err := os.Lstat(out)
	g02Check(t, err)
	if !info.IsDir() || info.Mode().Perm() != 0700 || info.Sys().(*syscall.Stat_t).Uid != uint32(os.Geteuid()) {
		t.Fatal("export requires an owned private directory")
	}
	for _, f := range g02Fixtures(t) {
		dir := g02Materialize(t, f)
		s := g02Open(t, dir)
		if f.Schema >= 22 {
			// The immediate predecessor must refuse actual project ownership,
			// not just an empty successor with a different schema marker.
			s.portAvailable = func(string, uint16) bool { return true } // Synthetic new reservation, no host I/O.
			port := uint16(28080)
			if f.Schema >= 25 {
				port = 29080
			} // The original schema25/26 fixtures reserve 28080/28081.
			if f.Schema == 45 {
				port = 31080
			} // Leave 29080/29081 for the captured private successor.
			projectReceipt, _, err := s.registerProject("synthetic-project-actor", projectRequest(t, "project-reservation", "new-project", port))
			g02Check(t, err)
			reservations, err := projectReservations(s.db)
			g02Check(t, err)
			if len(reservations.Instances) < 2 {
				t.Fatal("missing successor project reservations")
			}
			if f.Schema == 42 {
				populatePrivateLifecycle43(t, s)
			}
			if f.Schema == 45 {
				populatePrivateCapture46(t, s)
			}
			if f.Schema == 44 {
				populatePrivateBirth45(t, s)
			}
			if f.Schema == 43 {
				populatePrivateMember44(t, s)
			}
			if f.Schema == 41 {
				var project string
				g02Check(t, s.db.QueryRow("SELECT id FROM private_projects ORDER BY id LIMIT 1").Scan(&project))
				private, err := loadPrivateProject(s.db, project)
				g02Check(t, err)
				digest, err := private.Plan.Digest()
				g02Check(t, err)
				op, hit, err := s.acceptPrivateWorkspace("synthetic-workspace-actor", project, privateWorkspaceRequest{RequestID: "workspace-preparation", ExpectedRevision: private.Revision, ReviewDigest: digest}, privateWorkspaceParent())
				g02Check(t, err)
				if hit {
					t.Fatal("successor workspace unexpectedly replayed")
				}
				w, err := s.beginPrivateWorkspace(project, op.ID)
				g02Check(t, err)
				for _, phase := range []string{"root-dispatched", "root-created", "layout-dispatched"} {
					root := w.Root
					if phase == "root-created" {
						root = privateWorkspaceParent()
						root.Inode = 10
					}
					next, err := w.Advance(private.Plan, w.Phase, phase, root, nil)
					g02Check(t, err)
					g02Check(t, s.advancePrivateWorkspace(project, op.ID, w.Phase, next))
					w = next
				}
				g02Check(t, s.interruptPrivateWorkspace(project, op.ID))
				g02Check(t, validatePrivateWorkspaces(s.db))
			}
			if f.Schema == 40 {
				op, hit, err := s.registerPrivateProject("synthetic-private-chain-actor", privateProjectRequest(t, "private-reservation", "private-project", 30080))
				g02Check(t, err)
				if hit || op.ProjectID == "" {
					t.Fatal("successor private ownership missing")
				}
				_, err = projectReservations(s.db)
				g02Check(t, err)
			}
			if f.Schema == 39 {
				jobs, err := projectAnchorRecoveryJobs(s.db)
				g02Check(t, err)
				populated := false
				for _, parent := range jobs {
					owner, err := loadOperation(s.db, parent.OperationID)
					g02Check(t, err)
					if owner.Status != "interrupted" {
						continue
					}
					p, err := loadProject(s.db, parent.ProjectID)
					g02Check(t, err)
					op, hit, err := s.acceptProjectRecoveryContinuation("synthetic-continuation-actor", p.ID, observationRequest(p, parent, "continue-observed-checkpoint"))
					g02Check(t, err)
					if hit {
						t.Fatal("successor continuation unexpectedly replayed")
					}
					_, err = s.beginProjectRecoveryContinuation(op.ID)
					g02Check(t, err)
					populated = true
				}
				if !populated {
					t.Fatal("original39 lacks observed interrupted recovery")
				}
				g02Check(t, validateProjectRecoveryContinuations(s.db))
			}
			if f.Schema == 38 {
				jobs, err := projectAnchorRecoveryJobs(s.db)
				g02Check(t, err)
				populated := false
				for _, parent := range jobs {
					owner, err := loadOperation(s.db, parent.OperationID)
					g02Check(t, err)
					if owner.Status != "interrupted" {
						continue
					}
					p, err := loadProject(s.db, parent.ProjectID)
					g02Check(t, err)
					op, hit, err := s.acceptProjectRecoveryObservation("synthetic-observer", p.ID, observationRequest(p, parent, "observe-retained-checkpoint"))
					g02Check(t, err)
					if hit {
						t.Fatal("successor observation unexpectedly replayed")
					}
					_, err = s.beginProjectRecoveryObservation(op.ID)
					g02Check(t, err)
					populated = true
				}
				if !populated {
					t.Fatal("original schema38 lacks interrupted recovery")
				}
				g02Check(t, validateProjectRecoveryObservations(s.db))
			}
			if f.Schema == 37 {
				var stopID string
				g02Check(t, s.db.QueryRow("SELECT operation_id FROM project_member_stops WHERE json_extract(body,'$.stop.phase')='stopped' ORDER BY operation_id LIMIT 1").Scan(&stopID))
				stop, err := loadProjectMemberStop(s.db, stopID)
				g02Check(t, err)
				request := retireStoppedGenerationFixture(t, s, stop)
				resumed, _, err := s.acceptProjectMemberResume("actor", stop.ProjectID, stop.InstanceID, request)
				g02Check(t, err)
				resumeThroughFixture(t, s, resumed.ID, "running")
				g02Check(t, s.finishProjectMemberResume(resumed.ID))
				p, err := loadProject(s.db, stop.ProjectID)
				g02Check(t, err)
				h, _, _, err := projectRecoveryHistory(s.db, p.ID, true)
				g02Check(t, err)
				digest, err := p.Plan.Digest()
				g02Check(t, err)
				op, _, err := s.acceptProjectAnchorRecovery("synthetic-recovery-actor", p.ID, projectAnchorRecoveryRequest{RequestID: "recover-anchor", ExpectedRevision: p.Revision, ReviewDigest: digest, HistoryDigest: recoveryHistoryDigest(h)})
				g02Check(t, err)
				j, err := loadProjectAnchorRecovery(s.db, op.ID)
				g02Check(t, err)
				for _, phase := range []string{"reviewing", "reviewed", "draining", "stop-dispatched"} {
					j = advanceRecoveryStoreFixture(t, s, j, phase)
				}
				g02Check(t, validateProjectAnchorRecoveries(s.db))
			}
			if f.Schema == 36 {
				var readyID string
				g02Check(t, s.db.QueryRow("SELECT operation_id FROM project_member_context_readiness WHERE json_extract(body,'$.context.member.generation')=1 ORDER BY operation_id LIMIT 1").Scan(&readyID))
				j, err := loadProjectMemberContextReadiness(s.db, readyID)
				g02Check(t, err)
				p, err := loadProject(s.db, j.ProjectID)
				g02Check(t, err)
				digest, err := p.Plan.Digest()
				g02Check(t, err)
				op, _, err := s.acceptProjectMemberStop("synthetic-context-stop-actor", p.ID, j.InstanceID, projectMemberStopRequest{ReadinessOperationID: readyID, RequestID: "context-stop", ExpectedRevision: p.Revision, ReviewDigest: digest})
				g02Check(t, err)
				_, err = s.beginProjectMemberStop(op.ID)
				g02Check(t, err)
			}
			if f.Schema == 35 {
				var readyID, projectID, instance string
				g02Check(t, s.db.QueryRow("SELECT r.operation_id,r.project_id,r.instance_id FROM project_member_readiness r WHERE EXISTS (SELECT 1 FROM project_member_resumes x WHERE x.instance_id=r.instance_id) ORDER BY r.operation_id LIMIT 1").Scan(&readyID, &projectID, &instance))
				src, err := projectContextReadinessSource(s.db, projectID, instance, readyID, nil)
				g02Check(t, err)
				digest, err := src.Project.Plan.Digest()
				g02Check(t, err)
				op, _, err := s.acceptProjectMemberContextReadiness("synthetic-current-readiness-actor", projectID, instance, projectMemberContextReadinessRequest{OriginalReadinessOperationID: readyID, CreatorOperationID: src.Context.Member.OperationID, RequestID: "current-readiness", ExpectedRevision: src.Project.Revision, ReviewDigest: digest})
				g02Check(t, err)
				_, err = s.beginProjectMemberContextReadiness(op.ID)
				g02Check(t, err)
			}
			if f.Schema == 34 {
				// Real second replacement after the original schema34 stopped generation2.
				var stopID string
				g02Check(t, s.db.QueryRow("SELECT operation_id FROM project_member_stops WHERE json_extract(body,'$.stop.phase')='stopped' ORDER BY operation_id LIMIT 1").Scan(&stopID))
				stop, err := loadProjectMemberStop(s.db, stopID)
				g02Check(t, err)
				req := retireStoppedGenerationFixture(t, s, stop)
				op, _, err := s.acceptProjectMemberResume("actor", stop.ProjectID, stop.InstanceID, req)
				g02Check(t, err)
				resumeThroughFixture(t, s, op.ID, "create-dispatched")
				var n int
				g02Check(t, s.db.QueryRow("SELECT count(*) FROM project_member_resumes WHERE instance_id=?", stop.InstanceID).Scan(&n))
				if n != 2 {
					t.Fatal("successor lacks two retained replacements")
				}
			}
			if f.Schema == 33 {
				var project, instance, ready string
				g02Check(t, s.db.QueryRow("SELECT project_id,instance_id,operation_id FROM project_member_resumed_readiness ORDER BY operation_id LIMIT 1").Scan(&project, &instance, &ready))
				p, err := loadProject(s.db, project)
				g02Check(t, err)
				digest, err := p.Plan.Digest()
				g02Check(t, err)
				_, _, err = s.acceptProjectMemberStop("synthetic-current-stop-actor", project, instance, projectMemberStopRequest{RequestID: "stop-resumed-member", ExpectedRevision: p.Revision, ReviewDigest: digest, ReadinessOperationID: ready})
				g02Check(t, err)
			}
			if f.Schema == 32 {
				var project, instance, resumed string
				g02Check(t, s.db.QueryRow("SELECT project_id,instance_id,operation_id FROM project_member_resumes ORDER BY operation_id LIMIT 1").Scan(&project, &instance, &resumed))
				p, err := loadProject(s.db, project)
				g02Check(t, err)
				digest, err := p.Plan.Digest()
				g02Check(t, err)
				op, _, err := s.acceptProjectMemberResumedReadiness("synthetic-resumed-readiness-actor", project, instance, projectMemberResumedReadinessRequest{RequestID: "observe-resumed-member", ExpectedRevision: 1, ReviewDigest: digest, ResumeOperationID: resumed})
				g02Check(t, err)
				_, err = s.beginProjectMemberResumedReadiness(op.ID)
				g02Check(t, err)
				g02Check(t, s.completeProjectMemberResumedReadiness(op.ID, resumedReadinessProof(t, s, project, instance, resumed)))
			}
			if f.Schema == 31 {
				var project, instance, retirement string
				g02Check(t, s.db.QueryRow("SELECT project_id,instance_id,operation_id FROM project_member_retirements ORDER BY operation_id LIMIT 1").Scan(&project, &instance, &retirement))
				p, err := loadProject(s.db, project)
				g02Check(t, err)
				digest, err := p.Plan.Digest()
				g02Check(t, err)
				op, _, err := s.acceptProjectMemberResume("synthetic-resume-actor", project, instance, projectMemberResumeRequest{RequestID: "resume-member", ExpectedRevision: 1, ReviewDigest: digest, RetirementOperationID: retirement})
				g02Check(t, err)
				_, err = s.beginProjectMemberResume(op.ID, initialResumeFixture(t, s, op.ID))
				g02Check(t, err)
			}
			if f.Schema == 30 {
				var project, instance, exit string
				g02Check(t, s.db.QueryRow("SELECT project_id,instance_id,operation_id FROM project_member_stop_exits ORDER BY operation_id LIMIT 1").Scan(&project, &instance, &exit))
				p, err := loadProject(s.db, project)
				g02Check(t, err)
				digest, err := p.Plan.Digest()
				g02Check(t, err)
				op, _, err := s.acceptProjectMemberRetirement("synthetic-retirement-actor", project, instance, projectMemberRetirementRequest{RequestID: "retire-member", ExpectedRevision: 1, ReviewDigest: digest, ExitOperationID: exit})
				g02Check(t, err)
				_, err = s.beginProjectMemberRetirement(op.ID)
				g02Check(t, err)
			}
			if f.Schema == 29 {
				var project, instance, stop string
				g02Check(t, s.db.QueryRow("SELECT project_id,instance_id,operation_id FROM project_member_stops ORDER BY operation_id LIMIT 1").Scan(&project, &instance, &stop))
				p, err := loadProject(s.db, project)
				g02Check(t, err)
				digest, err := p.Plan.Digest()
				g02Check(t, err)
				op, _, err := s.acceptProjectMemberStopExit("synthetic-stop-exit-actor", project, instance, projectMemberStopExitRequest{RequestID: "observe-stop-exit", ExpectedRevision: 1, ReviewDigest: digest, StopOperationID: stop})
				g02Check(t, err)
				_, err = s.beginProjectMemberStopExit(op.ID)
				g02Check(t, err)
			}
			if f.Schema == 28 {
				var project, instance, readiness string
				g02Check(t, s.db.QueryRow("SELECT project_id,instance_id,operation_id FROM project_member_readiness ORDER BY operation_id LIMIT 1").Scan(&project, &instance, &readiness))
				p, err := loadProject(s.db, project)
				g02Check(t, err)
				digest, err := p.Plan.Digest()
				g02Check(t, err)
				op, _, err := s.acceptProjectMemberStop("synthetic-stop-actor", project, instance, projectMemberStopRequest{RequestID: "member-stop", ExpectedRevision: 1, ReviewDigest: digest, ReadinessOperationID: readiness})
				g02Check(t, err)
				_, err = s.beginProjectMemberStop(op.ID)
				g02Check(t, err)
			}
			if f.Schema == 27 {
				var project string
				g02Check(t, s.db.QueryRow("SELECT project_id FROM project_member_launches LIMIT 1").Scan(&project))
				p, err := loadProject(s.db, project)
				g02Check(t, err)
				digest, err := p.Plan.Digest()
				g02Check(t, err)
				op, _, err := s.acceptProjectMemberReadiness("synthetic-readiness-actor", project, p.Members[0].InstanceID, projectMemberReadinessRequest{RequestID: "member-readiness", ExpectedRevision: 1, ReviewDigest: digest})
				g02Check(t, err)
				_, err = s.beginProjectMemberReadiness(op.ID)
				g02Check(t, err)
			}
			if f.Schema == 26 {
				var project string
				g02Check(t, s.db.QueryRow("SELECT project_id FROM project_anchor_starts LIMIT 1").Scan(&project))
				p, err := loadProject(s.db, project)
				g02Check(t, err)
				digest, err := p.Plan.Digest()
				g02Check(t, err)
				member := p.Members[0].InstanceID
				op, _, err := s.acceptProjectMemberLaunch("synthetic-launch-actor", project, member, projectMemberLaunchRequest{RequestID: "member-launch", ExpectedRevision: 1, ReviewDigest: digest})
				g02Check(t, err)
				initial := memberLaunchSelection(t, s, op)
				_, err = s.beginProjectMemberLaunch(member, op.ID, initial)
				g02Check(t, err)
				g02Check(t, s.advanceProjectMemberLaunch(member, op.ID, "planned", "workspace-dispatched", projectnetwork.MemberLaunchResult{}))
			}
			if f.Schema == 25 {
				s.validateStorage = func(string) error { return nil } // Synthetic new claim; no filesystem authority.
				p, err := loadProject(s.db, projectReceipt.ProjectID)
				g02Check(t, err)
				digest, err := p.Plan.Digest()
				g02Check(t, err)
				op, _, err := s.acceptProjectMemberStorage("synthetic-project-actor", p.ID, "new-project-zod", projectMemberStorageRequest{RequestID: "member-storage", ExpectedRevision: 1, ReviewDigest: digest})
				g02Check(t, err)
				_, err = s.beginProjectMemberStorage("new-project-zod", op.ID)
				g02Check(t, err)
				g02Check(t, s.advanceProjectMemberStorage("new-project-zod", op.ID, "planned", "dispatched", pierstore.DirectoryIdentity{}))
			}
			if f.Schema == 24 {
				prepared := projectPreparationFixture(t, s, projectReceipt.ProjectID, "pending")
				op, _, err := s.acceptProjectPreparation("synthetic-project-actor", projectReceipt.ProjectID, projectPrepareRequest{RequestID: "prepare-project", ExpectedRevision: 1, ReviewDigest: prepared.PlanDigest})
				g02Check(t, err)
				completeProjectPreparation(t, s, op)
				start, _, err := s.acceptProjectAnchorStart("synthetic-project-actor", projectReceipt.ProjectID, projectAnchorStartRequest{RequestID: "start-anchor", ExpectedRevision: 1, ReviewDigest: prepared.PlanDigest})
				g02Check(t, err)
				_, err = s.beginProjectAnchorStart(projectReceipt.ProjectID, start.ID)
				g02Check(t, err)
				g02Check(t, s.advanceProjectAnchorStart(projectReceipt.ProjectID, start.ID, "planned", "dispatched", nil))
			}
			if f.Schema == 23 {
				prepared := projectPreparationFixture(t, s, projectReceipt.ProjectID, "pending")
				op, _, err := s.acceptProjectPreparation("synthetic-project-actor", projectReceipt.ProjectID, projectPrepareRequest{RequestID: "prepare-project", ExpectedRevision: 1, ReviewDigest: prepared.PlanDigest})
				g02Check(t, err)
				prepared.OperationID = op.ID
				g02Check(t, s.beginProjectPreparation(projectReceipt.ProjectID, op.ID, prepared))
				g02Check(t, s.advanceProjectPreparation(projectReceipt.ProjectID, op.ID, 0, "bridge", "dispatched", ""))
			}
		}
		g02Check(t, s.close())
		body, err := os.ReadFile(filepath.Join(dir, "runner.sqlite"))
		g02Check(t, err)
		target := filepath.Join(out, f.Name)
		g02Check(t, os.Mkdir(target, 0700))
		g02Check(t, os.WriteFile(filepath.Join(target, "runner.sqlite"), body, 0600))
	}
	t.Logf("exported real schema%d migrations for historical-reader checks", currentSchema)
}

func TestG02HistoricalInterruptedOperationsReconcileOnce(t *testing.T) {
	for _, f := range g02Fixtures(t) {
		t.Run(f.Name, func(t *testing.T) {
			dir := g02Materialize(t, f)
			except := map[string]bool{}
			after := map[string][]byte{}
			for pass := 0; pass < 3; pass++ {
				s := g02Open(t, dir)
				for _, id := range []string{"pending", "queued"} {
					v, err := loadInstance(s.db, id)
					g02Check(t, err)
					if pass == 0 {
						except["instances/"+id+"/body"] = true
						except["operations/"+v.Public.ActiveOperationID+"/body"] = true
					}
				}
				g02Check(t, s.reconcileInterrupted())
				for _, id := range []string{"pending", "queued", "interrupted"} {
					v, err := loadInstance(s.db, id)
					g02Check(t, err)
					if v.Public.ActiveOperationID != "" || v.Public.DesiredState != "stopped" || !v.Public.RecoveryRequired {
						t.Fatal("interrupted stop lost stopped intent/recovery fence", v.Public)
					}
					var body []byte
					g02Check(t, s.db.QueryRow("SELECT body FROM operations WHERE request_id=?", "stop-"+id).Scan(&body))
					var op protocol.Operation
					g02Check(t, json.Unmarshal(body, &op))
					if op.Status != "interrupted" || op.FinishedAt == nil {
						t.Fatal("unsettled historical receipt", op)
					}
					if pass > 0 && !bytes.Equal(body, after[id]) {
						t.Fatal("repeated reconciliation rewrote receipt", id)
					}
					after[id] = body
					g02Check(t, s.db.QueryRow("SELECT body FROM instances WHERE id=?", id).Scan(&body))
					if pass > 0 && !bytes.Equal(body, after["instance/"+id]) {
						t.Fatal("repeated reconciliation rewrote instance", id)
					}
					after["instance/"+id] = body
				}
				g02Replay(t, s, f)
				g02AssertRows(t, s.db, f, except)
				g02Check(t, s.close())
			}
		})
	}
}

func TestG02HistoricalConfigurationOmissionAndSmallLoom(t *testing.T) {
	for _, f := range g02Fixtures(t) {
		t.Run(f.Name, func(t *testing.T) {
			dir := g02Materialize(t, f)
			s := g02Open(t, dir)
			s.portAvailable = func(string, uint16) bool { return true }
			before, err := loadInstance(s.db, "stopped")
			g02Check(t, err)
			r := &Server{s: s, uid: uint32(os.Geteuid())}
			for n, field := range []string{"", `,"native_control":false`} {
				wire := fmt.Sprintf(`{"expected_revision":%d,"config":{"http_port":%d,"ames_port":34343,"loom":%d%s}}`, n+1, 18280+n, before.Public.Config.Loom, field)
				request := httptest.NewRequest("PUT", "/v1/instances/stopped/configuration", strings.NewReader(wire))
				request.Header.Set("Content-Type", "application/json")
				request.Header.Set(protocol.Header, "1")
				request = request.WithContext(context.WithValue(request.Context(), peerKey{}, r.uid))
				response := httptest.NewRecorder()
				r.ServeHTTP(response, request)
				if response.Code != 200 {
					t.Fatal("historical configuration edit refused", response.Code, response.Body.String())
				}
				after, err := loadInstance(s.db, "stopped")
				g02Check(t, err)
				wantNative := n == 0 && before.Public.Config.NativeControl
				if after.Public.Config.NativeControl != wantNative || after.Public.ConfigRevision != uint64(n+2) || after.Public.DesiredState != "stopped" || !reflect.DeepEqual(before.Runtime, after.Runtime) {
					t.Fatal("legacy omission/explicit false changed launch record or stopped intent", after.Public)
				}
			}
			if f.Name == "rc10" || f.Name == "preview18" {
				v, err := loadInstance(s.db, "running")
				g02Check(t, err)
				if v.Public.Config.Loom != 29 || v.Runtime.Loom != 29 || !v.Public.Config.NativeControl {
					t.Fatal("genuine historical small loom was clamped", v.Public)
				}
				// A fresh request with the preserved historical settings is still
				// subject to the current bounds; replay above used the old receipt.
				s.validateStorage = func(string) error { return nil }
				_, _, err = s.register("new-synthetic-actor", protocol.CreateFakeRequest{ID: "new-small", RequestID: "new-small", Config: v.Public.Config})
				var fault *fault
				if !errors.As(err, &fault) || fault.problem.Code != "invalid_request" {
					t.Fatal("new request bypassed current loom bounds", err)
				}
			}
			g02Check(t, s.close())
		})
	}
}

func TestG02HistoricalMigrationFailureAndFutureRefusalPreserveState(t *testing.T) {
	for _, f := range g02Fixtures(t) {
		for _, cause := range []string{"invalid-identity", "future-schema"} {
			t.Run(f.Name+"/"+cause, func(t *testing.T) {
				dir := g02Materialize(t, f)
				db := g02RawDB(t, dir)
				version := f.Schema
				var err error
				if cause == "future-schema" {
					version = currentSchema + 1 // explicit negative input, not a historical fixture
					_, err = db.Exec(fmt.Sprintf("PRAGMA user_version=%d", version))
				} else {
					_, err = db.Exec("UPDATE meta SET value='invalid' WHERE key='runner_id'")
				}
				g02Check(t, err)
				g02Check(t, db.Close())
				path := filepath.Join(dir, "runner.sqlite")
				before, err := os.ReadFile(path)
				g02Check(t, err)
				for pass := 0; pass < 2; pass++ {
					s, err := openStore(dir)
					if err == nil {
						s.close()
						t.Fatal("invalid/future database accepted")
					}
					after, err := os.ReadFile(path)
					g02Check(t, err)
					if !bytes.Equal(before, after) {
						t.Fatal("failed open changed database bytes")
					}
				}
				db = g02RawDB(t, dir)
				var got int
				g02Check(t, db.QueryRow("PRAGMA user_version").Scan(&got))
				if got != version {
					t.Fatal("failed migration committed schema marker")
				}
				if cause == "invalid-identity" {
					_, err = db.Exec("UPDATE meta SET value=? WHERE key='runner_id'", f.RunnerID)
					g02Check(t, err)
				}
				g02AssertRows(t, db, f, nil)
				g02Check(t, db.Close())
				if cause == "invalid-identity" {
					s := g02Open(t, dir) // the rolled-back historical migration is retryable
					g02AssertRows(t, s.db, f, nil)
					g02Check(t, s.close())
				}
			})
		}
	}
}
