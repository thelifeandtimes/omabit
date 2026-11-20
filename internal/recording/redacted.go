package recording

import (
	"bytes"
	"regexp"
	"strconv"
)

const (
	MaxViewBytes = 1 << 20
	MaxViewSpans = 8192
)

// RedactionOptions applies only to a derived view, never to stored originals.
// KnownSecrets are ephemeral literal octets, not part of metadata or the result.
// VerifiedLineStarts must come from independent retained-prefix evidence. Without
// it, omit each stream's leading partial line. A gap invalidates that evidence.
// Trailing partial lines are always withheld: an independently exported window
// must not reveal a piece of a credential split across windows.
type RedactionOptions struct {
	KnownSecrets       [][]byte
	VerifiedLineStarts []Kind
}

// ViewSpan accounts for a contiguous original byte range within its frame.
// Text is an escaped literal: no original terminal controls, bidi controls or
// invalid UTF-8 are executable through this representation. Render it as plain
// text, never HTML/Markdown/terminal input. Omitted spans carry only their reason
// and original length, preserving evidence of a transformation.
type ViewSpan struct {
	Offset uint32 `json:"offset"`
	Bytes  uint32 `json:"bytes"`
	Text   string `json:"escaped_text,omitempty"`
	Reason string `json:"omission_reason,omitempty"`
}

type RedactedFrame struct {
	Frame Frame      `json:"frame"`
	Spans []ViewSpan `json:"spans,omitempty"`
	Gap   *GapEvent  `json:"gap,omitempty"`
	Exit  *ExitEvent `json:"exit,omitempty"`
}

type RedactedView struct {
	Version        int             `json:"version"`
	Policy         string          `json:"policy"`
	Representation string          `json:"representation"`
	OriginalBytes  uint64          `json:"original_bytes"`
	OmittedBytes   uint64          `json:"omitted_bytes"`
	Frames         []RedactedFrame `json:"frames"`
}

// Match credential field labels, not arbitrary diagnostic atoms or whole lines.
// These finite rules are deliberately reviewable; they cannot identify every
// secret/private application value. Owners must review exports before sharing.
var credentialField = regexp.MustCompile(`(?i)(?:^|[ \t])(?:authorization|proxy-authorization|cookie|set-cookie|boot[.]key|(?:web[ \t]+)?login[ \t]+code)[ \t]*[:=][ \t]*`)

type viewPart struct{ frame, start, size int }
type viewStream struct {
	data  []byte
	parts []viewPart
	start bool
}

// RenderRedactedFrames consumes a bounded, already verified original window.
// This function does not authorize access, verify the digest chain, certify full
// session coverage or publish an export. The caller must select retained frames
// and enforce target/operator/explicit export-choice checks first. Stream order
// and timing stay associated with their original frame even when a field/secret
// spans frames or stdout/stderr are interleaved. Gap/exit payloads remain typed.
func RenderRedactedFrames(frames []Frame, options RedactionOptions) (RedactedView, error) {
	var out RedactedView
	if len(frames) == 0 || len(frames) > 256 || validateRedactionOptions(options) != nil {
		return out, ErrInvalid
	}
	return renderRedactedFrames(frames, options)
}

func validateRedactionOptions(options RedactionOptions) error {
	if len(options.KnownSecrets) > 32 || len(options.VerifiedLineStarts) > 3 {
		return ErrInvalid
	}
	secretBytes := 0
	for _, secret := range options.KnownSecrets {
		if len(secret) < 4 || len(secret) > 4096 {
			return ErrInvalid
		}
		secretBytes += len(secret)
	}
	if secretBytes > 64<<10 {
		return ErrInvalid
	}
	return nil
}

func renderRedactedFrames(frames []Frame, options RedactionOptions) (RedactedView, error) {
	var out RedactedView
	streams := map[Kind]*viewStream{Stdout: {}, Stderr: {}, PTYCombined: {}}
	for _, kind := range options.VerifiedLineStarts {
		stream := streams[kind]
		if stream == nil || stream.start {
			return out, ErrInvalid
		}
		stream.start = true
	}
	out = RedactedView{Version: 1, Policy: "credential-fields-and-selected-literals-v1", Representation: "escaped-literal-plain-text", Frames: make([]RedactedFrame, len(frames))}
	spanCount := 0
	flush := func(stream *viewStream) error {
		if len(stream.data) == 0 {
			stream.start = false
			return nil
		}
		marks := redactWindow(stream.data, stream.start, options.KnownSecrets)
		for _, part := range stream.parts {
			for i := 0; i < part.size; {
				j := i + 1
				reason := marks[part.start+i]
				for j < part.size && marks[part.start+j] == reason {
					j++
				}
				span := ViewSpan{Offset: uint32(i), Bytes: uint32(j - i)}
				spanCount++
				if spanCount > MaxViewSpans {
					return ErrLimit
				}
				if reason != 0 {
					span.Reason = omissionReason(reason)
					out.OmittedBytes += uint64(j - i)
				} else {
					quoted := strconv.QuoteToASCII(string(stream.data[part.start+i : part.start+j]))
					span.Text = quoted[1 : len(quoted)-1]
				}
				out.Frames[part.frame].Spans = append(out.Frames[part.frame].Spans, span)
				i = j
			}
		}
		stream.data, stream.parts, stream.start = nil, nil, false
		return nil
	}
	for i, frame := range frames {
		if frame.Sequence == 0 || i > 0 && (frames[i-1].Sequence == ^uint64(0) || frame.Sequence != frames[i-1].Sequence+1) || frame.UnixNano <= 0 || frame.Digest == [32]byte{} || frame.Kind < Stdout || frame.Kind > Exit || len(frame.Data) == 0 || len(frame.Data) > MaxFrameBytes || validateLifecycle(frame.Kind, frame.Data) != nil {
			return RedactedView{}, ErrInvalid
		}
		out.OriginalBytes += uint64(len(frame.Data))
		if out.OriginalBytes > MaxViewBytes {
			return RedactedView{}, ErrInvalid
		}
		meta := frame
		meta.Data = nil
		out.Frames[i].Frame = meta
		if stream := streams[frame.Kind]; stream != nil {
			stream.parts = append(stream.parts, viewPart{frame: i, start: len(stream.data), size: len(frame.Data)})
			stream.data = append(stream.data, frame.Data...)
			continue
		}
		for _, stream := range streams {
			if err := flush(stream); err != nil {
				return RedactedView{}, err
			}
		}
		if frame.Kind == Gap {
			gap, _ := ReadGap(frame.Data)
			out.Frames[i].Gap = &gap
		} else {
			exit, _ := ReadExit(frame.Data)
			out.Frames[i].Exit = &exit
		}
	}
	for _, stream := range streams {
		if err := flush(stream); err != nil {
			return RedactedView{}, err
		}
	}
	return out, nil
}

func omissionReason(reason byte) string {
	switch reason {
	case 1:
		return "unknown-leading-line-context"
	case 2:
		return "incomplete-trailing-line"
	case 3:
		return "credential-field-value"
	case 4:
		return "owner-selected-secret"
	}
	return "invalid-redaction-reason"
}

func redactWindow(data []byte, startsLine bool, secrets [][]byte) []byte {
	marks := make([]byte, len(data))
	mark := func(start, end int, reason byte) {
		for i := start; i < end; i++ {
			if marks[i] == 0 {
				marks[i] = reason
			}
		}
	}
	if !startsLine {
		end := bytes.IndexByte(data, '\n')
		if end < 0 {
			end = len(data)
		}
		mark(0, end, 1)
	}
	if data[len(data)-1] != '\n' {
		mark(bytes.LastIndexByte(data, '\n')+1, len(data), 2)
	}
	for start := 0; start < len(data); {
		end := bytes.IndexAny(data[start:], "\r\n")
		if end < 0 {
			end = len(data)
		} else {
			end += start
		}
		if match := credentialField.FindIndex(data[start:end]); match != nil {
			mark(start+match[1], end, 3)
		}
		start = end + 1
	}
	for _, secret := range secrets {
		// Linear matching and merged intervals bound repeated/overlapping secrets
		// such as thousands of identical bytes. Never rescan every matching span.
		prefix := make([]int, len(secret))
		for i, matched := 1, 0; i < len(secret); i++ {
			for matched > 0 && secret[i] != secret[matched] {
				matched = prefix[matched-1]
			}
			if secret[i] == secret[matched] {
				matched++
			}
			prefix[i] = matched
		}
		begin, end, matched := -1, 0, 0
		for i, b := range data {
			for matched > 0 && b != secret[matched] {
				matched = prefix[matched-1]
			}
			if b == secret[matched] {
				matched++
			}
			if matched == len(secret) {
				at := i + 1 - len(secret)
				if begin < 0 {
					begin = at
				} else if at > end {
					mark(begin, end, 4)
					begin = at
				}
				end = i + 1
				matched = prefix[matched-1]
			}
		}
		if begin >= 0 {
			mark(begin, end, 4)
		}
	}
	return marks
}
