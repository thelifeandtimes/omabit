//go:build !linux || !amd64

package managed

import "errors"

func configureRecordingServer(*Server) error {
	return errors.New("recording pilot requires linux/amd64")
}
