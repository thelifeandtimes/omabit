package recording

import (
	"errors"
	"time"
)

var (
	ErrStorage = errors.New("recording storage ownership or layout is invalid")
	ErrLimit   = errors.New("recording session byte limit reached; original bytes retained")
	ErrReserve = errors.New("recording persistence reserve would be consumed; original bytes retained")
	ErrCursor  = errors.New("recording cursor no longer matches verified retained coverage")
)

// Limits bound encoded session bytes and metadata3-or-later segment size. Host/ship
// quotas, incident pins and retention are not implemented here. Free-space
// admission is not an exclusive reservation against unrelated filesystem users.
type Limits struct {
	MaxSessionBytes  int64
	ReserveFreeBytes uint64
	// Zero selects16MiB (bounded by MaxSessionBytes); metadata3 and later rotate.
	MaxSegmentBytes int64
}

type Cursor struct {
	Version        int    `json:"version"`
	SessionID      string `json:"session_id"`
	MetadataSHA256 string `json:"metadata_sha256"`
	Sequence       uint64 `json:"sequence"`
	Digest         string `json:"digest"`
}
type Page struct {
	Metadata      Metadata  `json:"metadata"`
	Frames        []Frame   `json:"frames"`
	Next          Cursor    `json:"next"`
	Tail          string    `json:"tail"`
	WriterPresent bool      `json:"writer_present"`
	ReadAt        time.Time `json:"read_at"`
	Basis         string    `json:"basis"`
}
