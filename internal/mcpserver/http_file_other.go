//go:build !linux && !darwin

package mcpserver

import (
	"errors"
	"os"
)

func readHTTPPrivateFile(string) ([]byte, os.FileInfo, error) {
	return nil, nil, errors.New("private HTTP credential files require Linux or macOS")
}
func writeHTTPPrivateFile(string, []byte) error {
	return errors.New("private HTTP credential files require Linux or macOS")
}
