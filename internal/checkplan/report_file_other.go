//go:build !linux && !darwin

package checkplan

func readReportFile(string) ([]byte, error) { return nil, ErrReport }
