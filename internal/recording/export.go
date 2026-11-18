package recording

// OriginalFrameExport deliberately includes the unmodified original octets.
// Ordinary Frame/Page JSON omits them. This is a bounded encoding choice, not
// authorization, redaction, verification of the digest chain or a complete
// session export. A future API must select an actual verified retained frame,
// check its target and explicit raw-export grant before calling this function.
type OriginalFrameExport struct {
	Frame         Frame  `json:"frame"`
	OriginalBytes []byte `json:"original_bytes_base64"`
}

func ExportOriginalFrame(frame Frame) (OriginalFrameExport, error) {
	if frame.Sequence == 0 || frame.UnixNano <= 0 || frame.Digest == [32]byte{} || frame.Kind < Stdout || frame.Kind > Exit || len(frame.Data) == 0 || len(frame.Data) > MaxFrameBytes || validateLifecycle(frame.Kind, frame.Data) != nil {
		return OriginalFrameExport{}, ErrInvalid
	}
	data := append([]byte{}, frame.Data...)
	frame.Data = nil
	return OriginalFrameExport{Frame: frame, OriginalBytes: data}, nil
}
