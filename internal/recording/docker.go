package recording

import (
	"context"
	"encoding/binary"
	"errors"
	"io"
)

var ErrTransportFrame = errors.New("invalid or interrupted non-TTY recording transport; observe runtime and record a gap")
var ErrRecordingWrite = errors.New("original recording write failed; preserve the partial segment and report missing coverage")

type CopyStats struct {
	TransportFrames      uint64
	RecordedChunks       uint64
	OriginalBytes        uint64
	KnownUnrecordedBytes uint64
	AtFrameBoundary      bool
}

// CopyDockerStream decodes only Docker's verified non-TTY multiplex format.
// Caller must bind/verify the container, transport and TTY configuration before
// calling. It creates no connection and never starts/stops/attaches a runtime.
// Payloads are streamed in bounded chunks without line splitting, timestamps,
// terminal processing or redaction. The serialized append callback supplies its
// own receive-clock observation and durable storage. EOF is transport completion,
// never proof of a clean runtime exit. Cancellation must also close the upstream
// transport if its Reader blocks; a context alone cannot interrupt arbitrary I/O.
func CopyDockerStream(ctx context.Context, reader io.Reader, appendFrame func(Kind, []byte) error) (CopyStats, error) {
	var stats CopyStats
	if reader == nil || appendFrame == nil {
		return stats, ErrInvalid
	}
	buffer := make([]byte, MaxFrameBytes)
	for {
		stats.AtFrameBoundary = true
		if err := ctx.Err(); err != nil {
			return stats, err
		}
		var header [8]byte
		n, err := io.ReadFull(reader, header[:])
		if err == io.EOF && n == 0 {
			return stats, nil
		}
		stats.AtFrameBoundary = false
		if err != nil {
			return stats, errors.Join(ErrTransportFrame, incomplete(err))
		}
		if (header[0] != 1 && header[0] != 2) || header[1] != 0 || header[2] != 0 || header[3] != 0 {
			return stats, ErrTransportFrame
		}
		kind := Stdout
		if header[0] == 2 {
			kind = Stderr
		}
		remaining := uint64(binary.BigEndian.Uint32(header[4:]))
		for remaining > 0 {
			if err := ctx.Err(); err != nil {
				return stats, err
			}
			size := min(remaining, uint64(len(buffer)))
			n, readErr := io.ReadFull(reader, buffer[:size])
			if n > 0 {
				if writeErr := appendFrame(kind, buffer[:n]); writeErr != nil {
					stats.KnownUnrecordedBytes += uint64(n)
					return stats, errors.Join(ErrRecordingWrite, writeErr)
				}
				stats.RecordedChunks++
				stats.OriginalBytes += uint64(n)
				remaining -= uint64(n)
			}
			if readErr != nil {
				return stats, errors.Join(ErrTransportFrame, incomplete(readErr))
			}
		}
		stats.TransportFrames++
	}
}
