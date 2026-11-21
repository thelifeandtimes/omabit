package recording

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"testing"
)

func TestOrdinaryRecordingJSONDoesNotExportRawSecretBytes(t *testing.T) {
	data := []byte("Authorization: private-example\r\x1b[2K\xff\x00")
	var storage bytes.Buffer
	encoder, err := NewEncoder(&storage, testHeader())
	if err != nil {
		t.Fatal(err)
	}
	frame, err := encoder.Append(Stdout, data, testHeader().StartedUnixNano, 17)
	if err != nil {
		t.Fatal(err)
	}
	for _, value := range []any{frame, struct{ Frames []Frame }{[]Frame{frame}}} {
		encoded, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Contains(encoded, []byte("private-example")) || bytes.Contains(encoded, []byte(base64.StdEncoding.EncodeToString(data))) {
			t.Fatal("ordinary recording metadata exported raw bytes")
		}
	}
	exported, err := ExportOriginalFrame(frame)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(exported)
	if err != nil {
		t.Fatal(err)
	}
	var decoded OriginalFrameExport
	if json.Unmarshal(encoded, &decoded) != nil || !bytes.Equal(decoded.OriginalBytes, data) || decoded.Frame.Data != nil || decoded.Frame.Digest != frame.Digest || decoded.Frame.ElapsedNano != 17 {
		t.Fatal("explicit raw export changed octets or timing/digest metadata")
	}
	exported.OriginalBytes[0] = 'X'
	if !bytes.Equal(frame.Data, data) {
		t.Fatal("export aliased the original frame")
	}
	reader, err := NewDecoder(bytes.NewReader(storage.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	original, err := reader.Next()
	if err != nil || !bytes.Equal(original.Data, data) || original.Digest != frame.Digest {
		t.Fatal("JSON boundary changed retained original bytes", err)
	}
}

func TestOriginalFrameExportRefusesInvalidOrUnboundedFrames(t *testing.T) {
	var storage bytes.Buffer
	encoder, err := NewEncoder(&storage, testHeader())
	if err != nil {
		t.Fatal(err)
	}
	frame, err := encoder.Append(Stdout, []byte("original"), testHeader().StartedUnixNano, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, change := range []func(*Frame){func(f *Frame) { f.Sequence = 0 }, func(f *Frame) { f.Digest = [32]byte{} }, func(f *Frame) { f.Kind = 0 }, func(f *Frame) { f.Data = nil }, func(f *Frame) { f.Data = make([]byte, MaxFrameBytes+1) }, func(f *Frame) { f.Kind = Exit; f.Data = []byte(`{"version":1}`) }} {
		invalid := frame
		change(&invalid)
		if _, err := ExportOriginalFrame(invalid); !errors.Is(err, ErrInvalid) {
			t.Fatal("invalid original export accepted", err)
		}
	}
}
