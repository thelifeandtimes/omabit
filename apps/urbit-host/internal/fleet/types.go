// Package fleet owns one execution host. No client-side cache is authoritative.
package fleet

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const Version = "0.1.0-pilot"
const Schema = 1
const LabelPrefix = "io.omarchy.urbit"

type Paths struct{ Config, Data, State, Runtime string }

func DefaultPaths() (Paths, error) {
	home, e := os.UserHomeDir()
	if e != nil {
		return Paths{}, e
	}
	x := func(env, fallback string) string {
		if v := os.Getenv(env); v != "" && filepath.IsAbs(v) {
			return v
		}
		return fallback
	}
	p := Paths{x("XDG_CONFIG_HOME", filepath.Join(home, ".config")), x("XDG_DATA_HOME", filepath.Join(home, ".local/share")), x("XDG_STATE_HOME", filepath.Join(home, ".local/state")), x("XDG_RUNTIME_DIR", fmt.Sprintf("/run/user/%d", os.Getuid()))}
	p.Config = filepath.Join(p.Config, "omarchy-urbit")
	p.Data = filepath.Join(p.Data, "omarchy-urbit")
	p.State = filepath.Join(p.State, "omarchy-urbit")
	p.Runtime = filepath.Join(p.Runtime, "omarchy-urbit")
	return p, nil
}
func (p Paths) Ensure() error {
	for _, d := range []string{p.Config, p.Data, p.State, p.Runtime} {
		if e := os.MkdirAll(d, 0700); e != nil {
			return e
		}
		if e := os.Chmod(d, 0700); e != nil {
			return e
		}
	}
	return nil
}
func ID() string {
	var b [16]byte
	if _, e := rand.Read(b[:]); e != nil {
		panic(e)
	}
	return hex.EncodeToString(b[:])
}
func Now() string { return time.Now().UTC().Format(time.RFC3339Nano) }

var safeName = regexp.MustCompile(`^[a-z][a-z0-9-]{0,47}$`)
var safeID = regexp.MustCompile(`^[a-f0-9]{32}$`)
var imageName = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._/@:+-]{0,511}$`)

func ValidateName(v string) error {
	if !safeName.MatchString(v) {
		return fmt.Errorf("name must start with a lowercase letter, then use lowercase letters, digits or hyphens (maximum 48 characters)")
	}
	return nil
}
func ValidateImage(v string) error {
	if !imageName.MatchString(v) {
		return fmt.Errorf("invalid container image reference")
	}
	return nil
}

// These canonical suffixes encode the 256 galaxy identities. Initially only fake
// galaxies are supported; fake stars/planets need an extended route directory.
const suffixes = "zodnecbudwessevpersutletfulpensytdurwepserwylsunrypsyxdyrnuphebpeglupdepdysputlughecryttyvsydnexlunmeplutseppesdelsulpedtemledtulmetwenbynhexfebpyldulhetmevruttylwydtepbesdexsefwycburderneppurrysrebdennutsubpetrulsynregtydsupsemwynrecmegnetsecmulnymtevwebsummutnyxrextebfushepbenmuswyxsymselrucdecwexsyrwetdylmynmesdetbetbeltuxtugmyrpelsyptermebsetdutdegtexsurfeltudnuxruxrenwytnubmedlytdusnebrumtynseglyxpunresredfunrevrefmectedrusbexlebduxrynnumpyxrygryxfeptyrtustyclegnemfermertenlusnussyltecmexpubrymtucfyllepdebbermughuttunbylsudpemdevlurdefbusbeprunmelpexdytbyttyplevmylwedducfurfexnulluclennerlexrupnedlecrydlydfenwelnydhusrelrudneshesfetdesretdunlernyrsebhulrylludremlysfynwerrycsugnysnyllyndyndemluxfedsedbecmunlyrtesmudnytbyrsenwegfyrmurtelreptegpecnelnevfes"

func Galaxy(v string) (int, error) {
	v = strings.TrimPrefix(v, "~")
	if len(v) == 3 {
		for i := 0; i < len(suffixes); i += 3 {
			if suffixes[i:i+3] == v {
				return i / 3, nil
			}
		}
	}
	return 0, fmt.Errorf("pilot fake ships must be canonical galaxies, e.g. zod, nec, or bus")
}

type HostConfig struct {
	Schema        int     `json:"schema"`
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	DataRoot      string  `json:"data_root"`
	Channel       string  `json:"channel"`
	ImageOverride string  `json:"image_override,omitempty"`
	VersionURL    string  `json:"version_url"`
	AllowReal     bool    `json:"allow_real_ships"`
	CPUWarning    float64 `json:"fleet_cpu_warning_percent"`
	MemoryWarning uint64  `json:"fleet_memory_warning_bytes"`
}

func InitHost(p Paths, name, root string) (HostConfig, error) {
	if e := ValidateName(name); e != nil {
		return HostConfig{}, e
	}
	if root == "" {
		root = p.Data
	}
	if !filepath.IsAbs(root) || strings.ContainsAny(root, ",\n\r") {
		return HostConfig{}, fmt.Errorf("data root must be absolute and contain no commas or newlines")
	}
	if _, e := os.Lstat(filepath.Join(p.Config, "host.json")); e == nil {
		return HostConfig{}, fmt.Errorf("host already initialized; refusing overwrite")
	} else if !os.IsNotExist(e) {
		return HostConfig{}, e
	}
	if e := p.Ensure(); e != nil {
		return HostConfig{}, e
	}
	c := HostConfig{Schema: Schema, ID: ID(), Name: name, DataRoot: filepath.Clean(root), Channel: "latest", VersionURL: "https://version.groundseg.app"}
	return c, AtomicJSON(filepath.Join(p.Config, "host.json"), c)
}
func LoadHost(p Paths) (HostConfig, error) {
	var c HostConfig
	e := ReadJSON(filepath.Join(p.Config, "host.json"), &c)
	if e != nil {
		return c, e
	}
	if c.Schema != Schema || !safeID.MatchString(c.ID) {
		return c, fmt.Errorf("unsupported or invalid host configuration")
	}
	if !filepath.IsAbs(c.DataRoot) || strings.ContainsAny(c.DataRoot, ",\n\r") {
		return c, fmt.Errorf("invalid data root")
	}
	return c, nil
}

type Instance struct {
	ID            string `json:"id"`
	Label         string `json:"label"`
	Identity      string `json:"identity,omitempty"`
	Kind          string `json:"kind"` // fake, comet, keyed
	Group         string `json:"group,omitempty"`
	Channel       string `json:"channel"`
	Image         string `json:"image,omitempty"`
	ImageOverride string `json:"image_override,omitempty"`
	Pill          string `json:"pill,omitempty"`
	Loom          int    `json:"loom"`
	Container     string `json:"container"`
	HTTPPort      int    `json:"http_port"` // within its namespace
	HostPort      int    `json:"host_port"` // real ships only; fake group has a base range
	AmesPort      int    `json:"ames_port"`
	Desired       string `json:"desired"`
	Lifecycle     string `json:"lifecycle"`
	Created       string `json:"created_at"`
	Archived      string `json:"archived_at,omitempty"`
	Busy          string `json:"operation_id,omitempty"`
}

func (i Instance) Real() bool { return i.Kind != "fake" }
func (i Instance) RelativeDir() string {
	if i.Archived != "" {
		return filepath.Join("archive", i.ID)
	}
	if i.Kind == "fake" {
		return filepath.Join("piers", "dev", i.Group, i.ID)
	}
	return filepath.Join("piers", "real", i.ID)
}

type Group struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Network   string `json:"network"`
	Namespace string `json:"namespace_container"`
	HTTPBase  int    `json:"http_base"`
	Image     string `json:"image,omitempty"`
	Created   string `json:"created_at"`
}
type Operation struct {
	ID          string `json:"id"`
	RequestID   string `json:"request_id"`
	Fingerprint string `json:"-"`
	Action      string `json:"action"`
	Instance    string `json:"instance_id,omitempty"`
	Role        string `json:"role"`
	State       string `json:"state"`
	Message     string `json:"message,omitempty"`
	Started     string `json:"started_at"`
	Finished    string `json:"finished_at,omitempty"`
}

// Fingerprints are persisted separately from public operation records.
type DB struct {
	Schema       int                  `json:"schema"`
	Instances    map[string]Instance  `json:"instances"`
	Groups       map[string]Group     `json:"groups"`
	Operations   map[string]Operation `json:"operations"`
	Requests     map[string]string    `json:"requests"`
	Fingerprints map[string]string    `json:"fingerprints"`
	Notices      map[string]Notice    `json:"notices"`
}

func NewDB() DB {
	return DB{Schema: Schema, Instances: map[string]Instance{}, Groups: map[string]Group{}, Operations: map[string]Operation{}, Requests: map[string]string{}, Fingerprints: map[string]string{}, Notices: map[string]Notice{}}
}

type Notice struct {
	ID        string `json:"id"`
	Instance  string `json:"instance_id"`
	Message   string `json:"message"`
	Image     string `json:"available_image"`
	Created   string `json:"created_at"`
	Dismissed bool   `json:"dismissed"`
}
type Observation struct {
	Instance   Instance `json:"instance"`
	Container  string   `json:"container_status"`
	Runtime    string   `json:"runtime_status"`
	Endpoint   string   `json:"endpoint_health"`
	GUIEnabled bool     `json:"gui_enabled"`
	GUIReason  string   `json:"gui_reason,omitempty"`
	HostPort   int      `json:"host_port"`
	Observed   string   `json:"observed_at"`
	Error      string   `json:"error,omitempty"`
}
type Request struct {
	Schema   int    `json:"schema"`
	ID       string `json:"request_id,omitempty"`
	Method   string `json:"method"`
	Target   string `json:"target,omitempty"`
	Label    string `json:"label,omitempty"`
	Kind     string `json:"kind,omitempty"`
	Identity string `json:"identity,omitempty"`
	Group    string `json:"group,omitempty"`
	Channel  string `json:"channel,omitempty"`
	Image    string `json:"image,omitempty"`
	Pill     string `json:"pill,omitempty"`
	Loom     int    `json:"loom,omitempty"`
	Key      string `json:"key,omitempty"`
	Hoon     string `json:"hoon,omitempty"`
	Confirm  string `json:"confirm,omitempty"`
}
type RPCError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *RPCError) Error() string  { return e.Code + ": " + e.Message }
func E(code, msg string) *RPCError { return &RPCError{code, msg} }

type Response struct {
	Schema int             `json:"schema"`
	OK     bool            `json:"ok"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  *RPCError       `json:"error,omitempty"`
}

func Success(v any) Response {
	b, e := json.Marshal(v)
	if e != nil {
		return Failure(e)
	}
	return Response{Schema: Schema, OK: true, Result: b}
}
func Failure(e error) Response {
	if e == nil {
		e = fmt.Errorf("unspecified operation failure")
	}
	r, ok := e.(*RPCError)
	if !ok {
		r = E("operation_failed", e.Error())
	}
	return Response{Schema: Schema, OK: false, Error: r}
}
