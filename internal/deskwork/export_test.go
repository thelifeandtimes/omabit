package deskwork

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func exportDisplay(value any) string {
	data, _ := json.Marshal(value)
	for i, j := 0, len(data)-1; i < j; i, j = i+1, j-1 {
		data[i], data[j] = data[j], data[i]
	}
	return "0x" + hex.EncodeToString(data)
}

func TestSourceExportPreservesBoundedRawBytesAndExactSelection(t *testing.T) {
	paths := []string{"empty.txt", "nested/raw.bin"}
	raw := []byte{0, 255, 27, 10, 0, 128}
	rows := []map[string]string{{"path": paths[0], "content": ""}, {"path": paths[1], "content": base64.StdEncoding.EncodeToString(raw)}}
	files, err := DecodeExportedFiles(exportDisplay(rows), paths)
	if err != nil || len(files) != 2 || ValidateExportedFiles(paths, files) != nil {
		t.Fatal(files, err)
	}
	empty, e := files[0].Decode()
	decoded, e2 := files[1].Decode()
	if e != nil || e2 != nil || empty == nil || !reflect.DeepEqual(decoded, raw) {
		t.Fatal("binary/empty source changed")
	}
	for _, change := range []string{"path", "duplicate", "extra", "newline-base64", "oversized", "null"} {
		t.Run(change, func(t *testing.T) {
			copyRows := []map[string]string{{"path": paths[0], "content": ""}, {"path": paths[1], "content": rows[1]["content"]}}
			var display string
			switch change {
			case "path":
				copyRows[1]["path"] = "other.bin"
			case "duplicate":
				copyRows[1]["path"] = paths[0]
			case "extra":
				copyRows[1]["unreviewed"] = "ignored"
			case "newline-base64":
				copyRows[1]["content"] += "\n"
			case "oversized":
				copyRows[1]["content"] = base64.StdEncoding.EncodeToString(make([]byte, MaxExportBytes+1))
			case "null":
				display = exportDisplay(nil)
			}
			if display == "" {
				display = exportDisplay(copyRows)
			}
			if _, err := DecodeExportedFiles(display, paths); err == nil {
				t.Fatal("invalid export accepted")
			}
		})
	}
	files[1].ContentBase64 = base64.StdEncoding.EncodeToString([]byte("changed"))
	if ValidateExportedFiles(paths, files) == nil {
		t.Fatal("changed exported bytes retained trusted hash")
	}
}

func TestSourceExportIsPureHistoricalReadAndRefusesInjection(t *testing.T) {
	q, err := ExportFiles("~zod", "grubbery", 3, []string{"mar/hoon.hoon", "sys.kelvin"})
	if err != nil {
		t.Fatal(err)
	}
	body, err := q.Body()
	if err != nil {
		t.Fatal(err)
	}
	var envelope struct {
		Source struct {
			Hoon struct {
				Code string `json:"code"`
				Next struct {
					Data string `json:"data"`
				} `json:"next"`
			} `json:"hoon"`
		} `json:"source"`
		Sink map[string]any `json:"sink"`
	}
	if json.Unmarshal(body, &envelope) != nil || len(envelope.Sink) != 1 {
		t.Fatal("export is not stdout-only")
	}
	if _, ok := envelope.Sink["stdout"]; !ok {
		t.Fatal("export effect sink")
	}
	var rows [][]any
	if json.Unmarshal([]byte(envelope.Source.Hoon.Next.Data), &rows) != nil || len(rows) != 2 || !reflect.DeepEqual(rows[0][1], []any{"mar", "hoon", "hoon"}) {
		t.Fatal("selected Clay path changed", rows)
	}
	if !strings.Contains(envelope.Source.Hoon.Code, "/~zod/grubbery/3") {
		t.Fatal("historical desk binding missing")
	}
	for _, paths := range [][]string{nil, {}, {"../private.txt"}, {"/raw.txt"}, {"raw.txt", "raw.txt"}, {"z.txt", "a.txt"}, {"bad'.txt"}, {"raw.txt?x"}} {
		if _, err := ExportFiles("~zod", "grubbery", 3, paths); err == nil {
			t.Fatal("unsafe source selection", paths)
		}
	}
	if _, err := ExportFiles("~zod]", "grubbery", 3, []string{"a.txt"}); err == nil {
		t.Fatal("ship injection accepted")
	}
	if _, err := ExportFiles("~zod", "grubbery", 1<<53, []string{"a.txt"}); err == nil {
		t.Fatal("inexact revision accepted")
	}
}
