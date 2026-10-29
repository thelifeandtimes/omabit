package payload

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Only regular files/directories in a fresh destination are admitted. No links,
// devices, archive ownership, or paths escaping the private guest input tree.
func Unpack(reader io.Reader, target string) error {
	if err := os.Mkdir(target, 0755); err != nil {
		return err
	}
	tr := tar.NewReader(reader)
	var bytes int64
	count := 0
	seen := map[string]bool{}
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		name := strings.TrimSuffix(h.Name, "/")
		clean := filepath.Clean(name)
		if name == "" || name != clean || filepath.IsAbs(name) || clean == ".." || strings.HasPrefix(clean, "../") || strings.Contains(name, "\\") || seen[name] {
			return fmt.Errorf("payload path refused")
		}
		top := strings.Split(name, "/")[0]
		if top != "src" && top != "node" && top != "tools" && top != "source" && top != "compiled" && top != "deployment" && top != "vere" {
			return fmt.Errorf("payload root refused")
		}
		seen[name] = true
		count++
		bytes += h.Size
		if count > 40000 || h.Size < 0 || bytes > 1024*1024*1024 {
			return fmt.Errorf("payload bounds exceeded")
		}
		p := filepath.Join(target, name)
		switch h.Typeflag {
		case tar.TypeDir:
			if h.Size != 0 {
				return fmt.Errorf("directory body refused")
			}
			if err = os.MkdirAll(p, 0755); err != nil {
				return err
			}
		case tar.TypeReg, tar.TypeRegA:
			if err = os.MkdirAll(filepath.Dir(p), 0755); err != nil {
				return err
			}
			mode := os.FileMode(0644)
			if h.Mode&0111 != 0 {
				mode = 0755
			}
			f, e := os.OpenFile(p, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
			if e != nil {
				return e
			}
			_, e = io.CopyN(f, tr, h.Size)
			closeErr := f.Close()
			if e != nil {
				return e
			}
			if closeErr != nil {
				return closeErr
			}
		default:
			return fmt.Errorf("payload special file refused")
		}
	}
	if count < 10 {
		return fmt.Errorf("payload incomplete")
	}
	return nil
}

// Config binds every replacement to the reviewed pristine and candidate bytes.
type Config struct {
	PayloadSHA        string
	PreviousGuestSHA  string
	PreviousRunnerSHA string
	RunnerSHA         string
}

func verifyRunner(previous, candidate []byte, previousSHA, candidateSHA string) error {
	oldHash := sha256.Sum256(previous)
	if hex.EncodeToString(oldHash[:]) != previousSHA {
		return fmt.Errorf("pristine runner differs")
	}
	newHash := sha256.Sum256(candidate)
	if hex.EncodeToString(newHash[:]) != candidateSHA {
		return fmt.Errorf("candidate runner differs")
	}
	return nil
}

// Install runs only as the already-admitted VM init. The paths are fixed inside
// that fresh guest overlay; no host path or runtime state is accepted as an input.
func Install(config Config) error {
	if os.Getpid() != 1 || os.Geteuid() != 0 {
		return fmt.Errorf("guest PID1 required")
	}
	file, err := os.Open("/payload.tar.gz")
	if err != nil {
		return err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err = io.Copy(hash, file); err != nil {
		return err
	}
	if hex.EncodeToString(hash.Sum(nil)) != config.PayloadSHA {
		return fmt.Errorf("payload hash differs")
	}
	if _, err = file.Seek(0, 0); err != nil {
		return err
	}
	gz, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gz.Close()
	target := "/newroot/opt/omabit-lab/private-chain"
	if err = Unpack(gz, target); err != nil {
		return err
	}
	if err = gz.Close(); err != nil {
		return err
	}
	original, err := os.ReadFile("/newroot/opt/omabit-lab/guest.py")
	if err != nil {
		return err
	}
	sum := sha256.Sum256(original)
	if hex.EncodeToString(sum[:]) != config.PreviousGuestSHA {
		return fmt.Errorf("pristine guest differs")
	}
	oldRunner, err := os.ReadFile("/newroot/opt/omabit-lab/urbit-runner")
	if err != nil {
		return err
	}
	candidate, err := os.ReadFile(target + "/src/urbit-runner")
	if err != nil {
		return err
	}
	if err = verifyRunner(oldRunner, candidate, config.PreviousRunnerSHA, config.RunnerSHA); err != nil {
		return err
	}
	if err = os.WriteFile("/newroot/opt/omabit-lab/urbit-runner", candidate, 0755); err != nil {
		return err
	}
	guest, err := os.ReadFile(target + "/src/guest.py")
	if err != nil {
		return err
	}
	if err = os.WriteFile("/newroot/opt/omabit-lab/guest.py", guest, 0644); err != nil {
		return err
	}
	if err = os.WriteFile("/newroot/opt/omabit-lab/case.json", []byte("{\"exercise_private_chain\":true}\n"), 0644); err != nil {
		return err
	}
	if err = os.Remove("/payload.tar.gz"); err != nil {
		return err
	}
	fmt.Println("OMABIT_GUEST_INIT verified private-chain input payload installed")
	return nil
}
