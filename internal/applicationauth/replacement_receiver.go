package applicationauth

import "context"

// ReplacementReceiver adds local preparation, one bound independent offer, and
// read-only delivery observation. The issuer still owns authorization, ancestry,
// credentials and recovery; capability is not permission to issue or redeliver.
type ReplacementReceiver interface {
	Reconciler
	SupportsReplacement() bool
	PrepareReplacement(context.Context, []byte) ([]byte, error)
	Replace(context.Context, []byte) ([]byte, error)
	ReplacementStatus(context.Context, []byte) ([]byte, error)
}

func (a *Artifact) SupportsReplacement() bool {
	var d Declaration
	if !strictDeclaration(a.files[DeclarationName], &d) || !validDeclaration(d) || d.Version != 2 || d.Entry != a.entry {
		return false
	}
	seen := map[string]bool{}
	for _, name := range d.Files {
		if seen[name] || a.files[name] == nil {
			return false
		}
		seen[name] = true
	}
	return true
}

func (a *Artifact) PrepareReplacement(ctx context.Context, data []byte) ([]byte, error) {
	return a.run(ctx, data, "--prepare-replacement")
}

func (a *Artifact) Replace(ctx context.Context, data []byte) ([]byte, error) {
	return a.run(ctx, data, "--replace")
}

func (a *Artifact) ReplacementStatus(ctx context.Context, data []byte) ([]byte, error) {
	return a.run(ctx, data, "--replacement-status")
}
