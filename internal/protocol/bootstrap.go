package protocol

// Bootstrap is immutable first-boot intent. Bare delegates bootstrap selection to
// Vere; recommended binds an already uploaded pill by its content, not a live URL.
type Bootstrap struct {
	Kind string    `json:"kind"`
	Pill *BootPill `json:"pill,omitempty"`
}

func ValidBootstrap(b *Bootstrap) bool {
	return b != nil && (b.Kind == "bare" && b.Pill == nil || b.Kind == "recommended" && ValidBootPill(b.Pill))
}
