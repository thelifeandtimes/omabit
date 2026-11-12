package managed

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestContainerResourcesOriginalSchema58Preserved(t *testing.T) {
	sqlBytes, err := os.ReadFile("testdata/resources-schema58.sql")
	g02Check(t, err)
	metaBytes, err := os.ReadFile("testdata/resources-schema58.json")
	g02Check(t, err)
	var meta struct {
		Schema int    `json:"schema"`
		Source string `json:"source_commit"`
		SHA    string `json:"sql_sha256"`
	}
	g02Check(t, json.Unmarshal(metaBytes, &meta))
	h := sha256.Sum256(sqlBytes)
	if meta.Schema != 58 || meta.Source != "66c641bb41771c03391de3057cbf62c096f3ae51" || meta.SHA != hex.EncodeToString(h[:]) {
		t.Fatal("original writer fixture identity")
	}
	dir := privateDir(t)
	g02Check(t, os.WriteFile(filepath.Join(dir, "runner.sqlite"), nil, 0600))
	db := g02RawDB(t, dir)
	_, err = db.Exec(string(sqlBytes))
	g02Check(t, err)
	before, metadata := originalShipRouteRows(t, db), originalDeskRows(t, db)
	var route, history []byte
	g02Check(t, db.QueryRow("SELECT body FROM ship_routes").Scan(&route))
	g02Check(t, db.QueryRow("SELECT body FROM ship_routes_history").Scan(&history))
	g02Check(t, db.Close())
	for n := 0; n < 2; n++ {
		s, err := testOpenStore(dir)
		g02Check(t, err)
		if !reflect.DeepEqual(before, originalShipRouteRows(t, s.db)) || !reflect.DeepEqual(metadata, originalDeskRows(t, s.db)) {
			t.Fatal("migration changed original receipts")
		}
		var afterRoute, afterHistory []byte
		g02Check(t, s.db.QueryRow("SELECT body FROM ship_routes").Scan(&afterRoute))
		g02Check(t, s.db.QueryRow("SELECT body FROM ship_routes_history").Scan(&afterHistory))
		if string(route) != string(afterRoute) || string(history) != string(afterHistory) {
			t.Fatal("migration changed URL choices")
		}
		g02Check(t, s.close())
	}
}
