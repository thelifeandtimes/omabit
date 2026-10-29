//go:build linux

package privatechain

import "context"

// VerifyGatewayInitializer binds the fixed initializer path to a caller's
// qualified source hash. It supplements Verify, never replaces full generation
// validation, and does not execute or return program bytes. The hash must be an
// implementation pin, not an untrusted request's claim about its own program.
func (v *RuntimeInputVerifier) VerifyGatewayInitializer(ctx context.Context, expectedSHA string) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	if !hash(expectedSHA, false) {
		return errRuntimeInputs
	}
	if err := v.currentRoot(); err != nil {
		return err
	}
	const name = "gateway/docker_gateway.cjs"
	info, err := v.root.Lstat(name)
	if err != nil || info.Size() <= 0 || info.Size() > 1<<20 {
		return errRuntimeInputs
	}
	file := RuntimeInputFile{Path: name, Bytes: info.Size(), SHA256: expectedSHA}
	if _, err = readInputFile(ctx, v.root, name, 1<<20, &file); err != nil {
		return err
	}
	if err = v.currentRoot(); err != nil {
		return err
	}
	return ctx.Err()
}
