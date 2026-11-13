package ownerhost

import (
	"errors"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// Selected once at command startup, before any goroutines or runtime reads.
// This changes only this process. The ordinary installed profile is never edited.
var developmentRoot string

func ConfigureDevelopment(root string) error {
	if developmentRoot != "" {
		return errors.New("development root was already selected")
	}
	// A dedicated root-owned owner record authorizes exactly this side-by-side
	// installation. Do not infer authority from HOME, XDG or DOCKER_HOST.
	if !validDevelopmentRoot(root) {
		return errors.New("development root must be /srv/dev/urbit or /srv/dev/urbit-NAME")
	}
	p, err := loadProfile(filepath.Join(root, "owner.json"))
	if err != nil || p == nil {
		return errors.New("development owner profile is unavailable or unsafe")
	}
	if p.Piers != filepath.Join(root, "piers") {
		return errors.New("development piers must remain within their own root")
	}
	developmentRoot = root
	if _, err = Current(); err != nil {
		developmentRoot = ""
		return err
	}
	return nil
}

func validDevelopmentRoot(root string) bool {
	return root == "/srv/dev/urbit" || regexp.MustCompile(`^/srv/dev/urbit-[a-z0-9][a-z0-9-]{0,31}$`).MatchString(root)
}
func developmentRuntimeName(root string) string {
	return "omabit-dev" + strings.TrimPrefix(root, "/srv/dev/urbit")
}
func Development() bool { return developmentRoot != "" }
func selectedProfilePath() string {
	if Development() {
		return filepath.Join(developmentRoot, "owner.json")
	}
	return Path
}
func runtimeProfile(p *Profile) *Profile {
	out := *p
	if Development() {
		out.Home = filepath.Join(developmentRoot, "state")
	}
	return &out
}
func RuntimeDirectory() string {
	if !Development() {
		return ""
	}
	p, err := Current()
	if err != nil || p == nil {
		return "/invalid-omabit-development-runtime"
	}
	return filepath.Join("/run/user", strconv.FormatUint(uint64(p.UID), 10), developmentRuntimeName(developmentRoot))
}
