package recording

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
)

type viewInput struct {
	kind Kind
	data string
}

func verifiedViewFrames(t *testing.T, inputs ...viewInput) []Frame {
	t.Helper()
	var raw bytes.Buffer
	encoder, err := NewEncoder(&raw, testHeader())
	if err != nil {
		t.Fatal(err)
	}
	for i, input := range inputs {
		if _, err = encoder.Append(input.kind, []byte(input.data), testHeader().StartedUnixNano-int64(i), uint64(i)); err != nil {
			t.Fatal(err)
		}
	}
	decoder, err := NewDecoder(bytes.NewReader(raw.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	var frames []Frame
	for range inputs {
		frame, err := decoder.Next()
		if err != nil {
			t.Fatal(err)
		}
		frames = append(frames, frame)
	}
	return frames
}

func viewText(view RedactedView) string {
	var out strings.Builder
	for _, frame := range view.Frames {
		for _, span := range frame.Spans {
			out.WriteString(span.Text)
		}
	}
	return out.String()
}

func checkViewAccounting(t *testing.T, frames []Frame, view RedactedView) {
	t.Helper()
	var total, omitted uint64
	if len(frames) != len(view.Frames) {
		t.Fatal("frame ordering lost")
	}
	for i, got := range view.Frames {
		original := frames[i]
		total += uint64(len(original.Data))
		original.Data = nil
		if !reflect.DeepEqual(got.Frame, original) || got.Frame.Data != nil {
			t.Fatal("original timing/stream/digest metadata changed or raw bytes exposed")
		}
		if original.Kind == Gap || original.Kind == Exit {
			continue
		}
		var offset uint32
		for _, span := range got.Spans {
			if span.Offset != offset || span.Bytes == 0 || (span.Text == "") == (span.Reason == "") {
				t.Fatal("incomplete, duplicated or ambiguous span accounting", span)
			}
			offset += span.Bytes
			if span.Reason != "" {
				omitted += uint64(span.Bytes)
			}
		}
		if int(offset) != len(frames[i].Data) {
			t.Fatal("unaccounted original octets")
		}
	}
	if view.OriginalBytes != total || view.OmittedBytes != omitted {
		t.Fatal("derived coverage counts changed")
	}
}

func TestRedactedViewPreservesDiagnosticAtomsLongTracesAndExactOriginals(t *testing.T) {
	frames := verifiedViewFrames(t,
		viewInput{Stdout, "0wdiagnostic.atom prefix Authorization: Be"},
		viewInput{Stderr, "stderr 0wother.atom\n"},
		viewInput{Stdout, "arer owner-token\r\ntrace " + strings.Repeat("x", 12288) + "\n"},
		viewInput{Stdout, "Cookie: session-fake\nSet-Cookie: second-fake\nboot.key = key-fake\nweb login code: ~fake-code\n"},
		viewInput{Stdout, "0wdiagnostic2.atom\r\x1b]52;clipboard\x07\x00\xff\u202e\n"},
	)
	before := make([][]byte, len(frames))
	for i := range frames {
		before[i] = bytes.Clone(frames[i].Data)
	}
	view, err := RenderRedactedFrames(frames, RedactionOptions{VerifiedLineStarts: []Kind{Stdout, Stderr}})
	if err != nil {
		t.Fatal(err)
	}
	checkViewAccounting(t, frames, view)
	text := viewText(view)
	for _, kept := range []string{"0wdiagnostic.atom", "0wother.atom", "0wdiagnostic2.atom", "Authorization:", "Cookie:", "boot.key =", strings.Repeat("x", 12288), `\x1b`, `\a`, `\x00`, `\xff`, `\u202e`} {
		if !strings.Contains(text, kept) {
			t.Fatal("diagnostic context or escaped original missing", kept[:min(len(kept), 50)])
		}
	}
	encoded, err := json.Marshal(view)
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{"Bearer owner-token", "owner-token", "session-fake", "second-fake", "key-fake", "~fake-code"} {
		if bytes.Contains(encoded, []byte(secret)) || bytes.Contains(encoded, []byte(base64.StdEncoding.EncodeToString([]byte(secret)))) {
			t.Fatal("derived view leaked raw or encoded credential", secret)
		}
	}
	for i, original := range before {
		if !bytes.Equal(frames[i].Data, original) {
			t.Fatal("redaction changed original bytes")
		}
	}
	for _, b := range []byte{0, 7, 13, 27, 255} {
		if strings.IndexByte(text, b) >= 0 {
			t.Fatal("original terminal/control/invalid octet escaped containment")
		}
	}
}

func TestCredentialAndSelectedLiteralCannotEscapeAcrossAnyFrameBoundary(t *testing.T) {
	data := "prefix Authorization: Bearer private-value\nordinary selected-secret suffix\n"
	for split := 1; split < len(data); split++ {
		frames := verifiedViewFrames(t, viewInput{Stdout, data[:split]}, viewInput{Stderr, "independent\n"}, viewInput{Stdout, data[split:]})
		view, err := RenderRedactedFrames(frames, RedactionOptions{VerifiedLineStarts: []Kind{Stdout, Stderr}, KnownSecrets: [][]byte{[]byte("selected-secret")}})
		if err != nil {
			t.Fatal(split, err)
		}
		checkViewAccounting(t, frames, view)
		var omitted string
		for i, frame := range view.Frames {
			for _, span := range frame.Spans {
				if span.Reason != "" {
					omitted += string(frames[i].Data[span.Offset : span.Offset+span.Bytes])
				}
			}
		}
		if !strings.Contains(omitted, "Bearer private-value") || !strings.Contains(omitted, "selected-secret") {
			t.Fatal("split credential/literal lost redaction", split)
		}
	}
}

func TestUnknownWindowEdgesAndGapsWithholdUnverifiableLineFragments(t *testing.T) {
	gap, _ := (GapEvent{Version: 1, Reason: "transport-interrupted"}).Bytes()
	frames := verifiedViewFrames(t,
		viewInput{Stdout, "secret-continuation\n0wdiagnostic.atom\npartial-secret"},
		viewInput{Gap, string(gap)},
		viewInput{Stdout, "unknown-after-gap\n0wstill-visible.atom\ntrailing-fragment"},
	)
	view, err := RenderRedactedFrames(frames, RedactionOptions{})
	if err != nil {
		t.Fatal(err)
	}
	checkViewAccounting(t, frames, view)
	text := viewText(view)
	if !strings.Contains(text, "0wdiagnostic.atom") || !strings.Contains(text, "0wstill-visible.atom") || strings.Contains(text, "secret") || strings.Contains(text, "unknown-after-gap") || strings.Contains(text, "trailing-fragment") || view.Frames[1].Gap == nil || view.Frames[1].Gap.Reason != "transport-interrupted" {
		t.Fatal("unknown window/gap coverage was hidden or leaked", view)
	}
	// Even an independently proven initial boundary is invalidated by a gap.
	view, err = RenderRedactedFrames(frames, RedactionOptions{VerifiedLineStarts: []Kind{Stdout}})
	if err != nil || strings.Contains(viewText(view), "unknown-after-gap") {
		t.Fatal("gap inherited earlier boundary evidence", err)
	}
}

func TestSelectedSecretsHandleOverlapsAndRepetitionWithinLinearBounds(t *testing.T) {
	for _, pair := range [][2]string{{"abababa", "ababa"}, {strings.Repeat("a", MaxFrameBytes-1), strings.Repeat("a", 4096)}} {
		frames := verifiedViewFrames(t, viewInput{PTYCombined, pair[0] + "\n"})
		view, err := RenderRedactedFrames(frames, RedactionOptions{VerifiedLineStarts: []Kind{PTYCombined}, KnownSecrets: [][]byte{[]byte(pair[1])}})
		if err != nil || view.OmittedBytes != uint64(len(pair[0])) || viewText(view) != `\n` {
			t.Fatal("overlapping/repeated secret left a visible suffix", err)
		}
		checkViewAccounting(t, frames, view)
	}
}

func TestRedactedViewRefusesUnboundedOrInvalidWindowsWithoutPartialOutput(t *testing.T) {
	frames := verifiedViewFrames(t, viewInput{Stdout, "safe\n"})
	for _, options := range []RedactionOptions{{KnownSecrets: [][]byte{nil}}, {KnownSecrets: [][]byte{[]byte("abc")}}, {KnownSecrets: [][]byte{bytes.Repeat([]byte("s"), 4097)}}, {VerifiedLineStarts: []Kind{Gap}}, {VerifiedLineStarts: []Kind{Stdout, Stdout}}} {
		view, err := RenderRedactedFrames(frames, options)
		if !errors.Is(err, ErrInvalid) || len(view.Frames) != 0 {
			t.Fatal("invalid options exposed a partial export", err)
		}
	}
	broken := append([]Frame(nil), frames...)
	broken[0].Sequence = 0
	if view, err := RenderRedactedFrames(broken, RedactionOptions{}); !errors.Is(err, ErrInvalid) || len(view.Frames) != 0 {
		t.Fatal("invalid original frame accepted", err)
	}
	var inputs []viewInput
	for range 17 {
		inputs = append(inputs, viewInput{Stdout, strings.Repeat("x", MaxFrameBytes)})
	}
	if view, err := RenderRedactedFrames(verifiedViewFrames(t, inputs...), RedactionOptions{}); !errors.Is(err, ErrInvalid) || len(view.Frames) != 0 {
		t.Fatal("oversized export accepted", err)
	}
	inputs = []viewInput{{Stdout, strings.Repeat("safesecret", 6000) + "\n"}}
	view, err := RenderRedactedFrames(verifiedViewFrames(t, inputs...), RedactionOptions{VerifiedLineStarts: []Kind{Stdout}, KnownSecrets: [][]byte{[]byte("secret")}})
	if !errors.Is(err, ErrLimit) || len(view.Frames) != 0 {
		t.Fatal("excessive transformation spans exposed a partial export", err)
	}
}
