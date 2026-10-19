package hostcompat

import (
	"bytes"
	"encoding/hex"
	"errors"
	"strings"
)

type nativeSignatureExit struct{ code int }

func (e nativeSignatureExit) Error() string {
	return "native signature verification reported unavailable or invalid signatures"
}
func (e nativeSignatureExit) ExitCode() int { return e.code }

func signatureExitAllowed(err error) bool {
	if err == nil {
		return true
	}
	var exit interface{ ExitCode() int }
	return errors.As(err, &exit) && (exit.ExitCode() == 1 || exit.ExitCode() == 2)
}

func strongValidSignature(fields []string) bool {
	if len(fields) < 11 || (len(fields[2]) != 40 && len(fields[2]) != 64) {
		return false
	}
	if _, err := hex.DecodeString(fields[2]); err != nil {
		return false
	}
	return fields[9] == "8" || fields[9] == "9" || fields[9] == "10" || fields[9] == "11"
}

func keyID(s string) bool {
	if len(s) != 16 && len(s) != 40 && len(s) != 64 {
		return false
	}
	_, err := hex.DecodeString(s)
	return err == nil
}

// Each NEWSIG record is evaluated independently. gpgv only receives this
// source's selected keyrings: a VALIDSIG is therefore from an authorized key.
// An additional unknown signer is neutral only with matching ERRSIG(code 9)
// and NO_PUBKEY records. Available bad/revoked/expired/weak selected signatures,
// malformed status, other native failures and invalid-only input are rejected.
func aptSignatureStatus(status []byte) bool {
	if len(status) > 64<<10 {
		return false
	}
	type signature struct {
		started                 bool
		valid                   bool
		good, errorKey, missing string
	}
	current := signature{}
	records, accepted := 0, 0
	finish := func() bool {
		if !current.started {
			return true
		}
		records++
		if records > 16 {
			return false
		}
		if current.valid {
			if current.errorKey != "" || current.missing != "" {
				return false
			}
			accepted++
			return true
		}
		return current.good == "" && current.errorKey != "" && current.errorKey == current.missing
	}
	for _, line := range strings.Split(string(status), "\n") {
		f := strings.Fields(line)
		if len(f) < 2 || f[0] != "[GNUPG:]" {
			continue
		}
		switch f[1] {
		case "NEWSIG":
			if !finish() {
				return false
			}
			current = signature{started: true}
		case "GOODSIG":
			if len(f) < 3 || !keyID(f[2]) || current.good != "" {
				return false
			}
			current.started = true
			current.good = strings.ToUpper(f[2])
		case "VALIDSIG":
			if current.valid || !strongValidSignature(f) {
				return false
			}
			if current.good != "" && !strings.HasSuffix(strings.ToUpper(f[2]), current.good) {
				return false
			}
			current.started = true
			current.valid = true
		case "ERRSIG":
			if len(f) < 8 || !keyID(f[2]) || f[7] != "9" || current.errorKey != "" {
				return false
			}
			current.started = true
			current.errorKey = strings.ToUpper(f[2])
		case "NO_PUBKEY":
			if len(f) != 3 || !keyID(f[2]) || current.missing != "" {
				return false
			}
			current.started = true
			current.missing = strings.ToUpper(f[2])
		case "BADSIG", "EXPSIG", "EXPKEYSIG", "REVKEYSIG", "KEYREVOKED", "KEYEXPIRED", "SIGEXPIRED", "NODATA", "FAILURE", "ERROR", "BADARMOR":
			return false
		}
	}
	return finish() && accepted > 0
}

// Decode only the standard InRelease clear-sign envelope after native signature
// verification. Extract from the captured, identity-bound signed bytes rather
// than a second possibly partial/erroring gpgv plaintext command. No signature
// algorithm or trust decision is implemented here.
func releaseCleartext(raw []byte) ([]byte, error) {
	if len(raw) > 4<<20 || bytes.IndexByte(raw, 0) >= 0 {
		return nil, errEvidence
	}
	text := strings.ReplaceAll(string(raw), "\r\n", "\n")
	if strings.ContainsRune(text, '\r') {
		return nil, errEvidence
	}
	lines := strings.Split(text, "\n")
	if len(lines) < 6 || lines[0] != "-----BEGIN PGP SIGNED MESSAGE-----" {
		return nil, errEvidence
	}
	i := 1
	headers := 0
	for i < len(lines) && lines[i] != "" {
		if !strings.HasPrefix(lines[i], "Hash: ") || len(lines[i]) > 256 {
			return nil, errEvidence
		}
		headers++
		i++
	}
	if headers == 0 || headers > 8 || i >= len(lines) {
		return nil, errEvidence
	}
	i++
	var plain strings.Builder
	signature := -1
	for ; i < len(lines); i++ {
		line := lines[i]
		if line == "-----BEGIN PGP SIGNATURE-----" {
			signature = i
			break
		}
		if strings.HasPrefix(line, "- ") {
			line = line[2:]
		} else if strings.HasPrefix(line, "-") {
			return nil, errEvidence
		}
		plain.WriteString(line)
		plain.WriteByte('\n')
	}
	if signature < 0 || plain.Len() == 0 {
		return nil, errEvidence
	}
	end := -1
	for i := signature + 1; i < len(lines); i++ {
		if lines[i] == "-----END PGP SIGNATURE-----" {
			end = i
			break
		}
	}
	if end < 0 {
		return nil, errEvidence
	}
	for _, line := range lines[end+1:] {
		if strings.TrimSpace(line) != "" {
			return nil, errEvidence
		}
	}
	return []byte(plain.String()), nil
}
