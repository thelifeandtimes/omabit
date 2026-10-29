//go:build !linux

package privatechain

import "context"

func InspectRuntimeInputs(ctx context.Context, directory, expectedSHA, expectedFoundation string) (RuntimeInputInspection, error) {
	if err := ctx.Err(); err != nil {
		return RuntimeInputInspection{}, err
	}
	return RuntimeInputInspection{}, errRuntimeInputs
}
