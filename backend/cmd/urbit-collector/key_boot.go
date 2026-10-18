package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
	"syscall"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func readBootKey(path string) (string, error) {
	bad := errors.New("boot key must be an owned private regular file containing a bounded 0w key")
	f, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return "", bad
	}
	defer f.Close()
	i, err := f.Stat()
	if err != nil {
		return "", bad
	}
	st, ok := i.Sys().(*syscall.Stat_t)
	if !ok || !i.Mode().IsRegular() || i.Mode().Perm()&0077 != 0 || st.Uid != uint32(os.Geteuid()) || st.Nlink != 1 || i.Size() > protocol.MaxBootKeyBytes {
		return "", bad
	}
	b, err := io.ReadAll(io.LimitReader(f, protocol.MaxBootKeyBytes+1))
	key := strings.TrimSpace(string(b))
	if err != nil || !protocol.ValidBootKey(key) {
		return "", bad
	}
	return key, nil
}

func runKeyBoot(ctx context.Context, args []string, dir, socket, runner, expected, format string, out io.Writer, fail func(error, int) int) int {
	var req protocol.CreateShipRequest
	var file, pillSource, bootstrap string
	var port, ames, loom uint
	var native, runtimeSettings bool
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		req.ID, args = args[0], args[1:]
	}
	f := flag.NewFlagSet("create-ship", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	f.StringVar(&req.Ship, "ship", "", "expected owned Urbit identity")
	f.StringVar(&req.PierPath, "pier-path", "", "convention-relative livenet destination")
	f.StringVar(&req.RequestID, "request-id", "", "actor-scoped request identity")
	f.StringVar(&req.Label, "label", "", "instance label")
	f.StringVar(&file, "key-file", "", "private local boot key file; contents never enter command arguments")
	f.UintVar(&port, "http-port", 0, "host loopback HTTP port")
	f.UintVar(&ames, "ames-port", 0, "host Ames UDP port; zero/omitted chooses automatically")
	f.UintVar(&loom, "loom", protocol.DefaultLoom, "loom byte exponent 30..34: 30=1 GiB, 31=2 GiB, 34=16 GiB address space; not resident memory or the separate 4 GiB container cap")
	f.BoolVar(&native, "native-control", false, "enable local-owner native control (default off)\n\n"+protocol.NativeControlWarning)
	f.StringVar(&pillSource, "pill", "", "optional boot pill URL or local file; blank uses Vere canonical default")
	f.StringVar(&bootstrap, "bootstrap", "", "bare or recommended; recommended requires --pill")
	f.BoolVar(&req.ConfirmNetwork, "confirm-network", false, "boot a real networked identity on explicit Start")
	if err := f.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Fprintln(out, "Usage: urbit-collector --runner NAME --expected-runner-id ID create-ship --ship ~SHIP --key-file FILE --request-id ID --confirm-network [--http-port PORT --ames-port PORT --bootstrap bare|recommended --pill URL_OR_FILE]\nInstance name and tier path are derived; omitted ports are allocated atomically. Bare uses Vere's canonical default; recommended requires a selected pill.")
			f.SetOutput(out)
			f.PrintDefaults()
			return 0
		}
		return fail(errors.New("invalid key-boot arguments"), 2)
	}
	f.Visit(func(flag *flag.Flag) {
		if flag.Name == "loom" || flag.Name == "native-control" {
			runtimeSettings = true
		}
	})
	if loom < protocol.MinLoom || loom > protocol.MaxLoom {
		return fail(errors.New("--loom must be in 30..34 (1–16 GiB)"), 2)
	}
	req.Ship = strings.TrimSpace(req.Ship)
	if !strings.HasPrefix(req.Ship, "~") {
		req.Ship = "~" + req.Ship
	}
	if message := protocol.ShipNameProblem(req.Ship); message != "" {
		return fail(errors.New(message), 2)
	}
	class := protocol.ShipClass(req.Ship)
	derived := strings.TrimPrefix(req.Ship, "~")
	if req.ID == "" {
		req.ID = derived
	}
	if req.PierPath == "" {
		req.PierPath = "livenet/" + map[string]string{"galaxy": "galaxies", "star": "stars", "planet": "planets", "moon": "moons"}[class] + "/" + derived + "/" + derived
	}
	if f.NArg() != 0 {
		return fail(errors.New("unexpected trailing key-boot arguments"), 2)
	}
	if socket != "" || !validID(runner) || !validID(expected) {
		return fail(errors.New("select an enrolled runner and its expected runner ID"), 2)
	}
	if class == "comet" {
		return fail(errors.New("use New comet to generate a comet identity; key boot accepts galaxies, stars, planets and moons"), 2)
	}
	if !validID(req.ID) || req.ID != derived {
		return fail(errors.New("instance name must be the Urbit ID without ~"), 2)
	}
	if !validID(req.RequestID) {
		return fail(errors.New("key boot requires a valid request ID"), 2)
	}
	if pierstore.Kind(req.PierPath) != class {
		return fail(fmt.Errorf("pier path must match the %s tier", class), 2)
	}
	if port != 0 && port < 1024 || port > 65535 || ames != 0 && ames < 1024 || ames > 65535 {
		return fail(errors.New("HTTP/Ames ports must be automatic (zero/omitted) or 1024–65535"), 2)
	}
	if !req.ConfirmNetwork {
		return fail(errors.New("key boot requires --confirm-network"), 2)
	}
	if _, err := bootstrapChoice(bootstrap, pillSource); err != nil {
		return fail(err, 2)
	}
	r, err := openRegistry(dir)
	if err != nil {
		return fail(err, 1)
	}
	entry, err := r.Get(ctx, runner)
	if err != nil {
		return fail(err, 1)
	}
	if entry.RunnerID != expected {
		return fail(errors.New("runner selection changed"), 1)
	}
	c := entry.Client(true)
	defer c.Close()
	h, err := c.Hello(ctx)
	if err != nil {
		return fail(err, 1)
	}
	if h.AccessRole != "operator" || !slices.Contains(h.Features, "key-boot-options") {
		return fail(errors.New("runner requires operator key-boot capability"), 1)
	}
	if runtimeSettings && !slices.Contains(h.Features, protocol.RuntimeSettingsFeature) {
		return fail(errors.New("update this runner for runtime-settings capability before changing loom/native control"), 1)
	}
	if native && !slices.Contains(h.Features, protocol.NativeControlAllShipsFeature) {
		return fail(errors.New("update this runner for native-control-all-ships capability before enabling native control"), 1)
	}
	req.Key, err = readBootKey(file)
	if err != nil {
		return fail(err, 2)
	}
	req.Pill, err = stageSelectedPill(ctx, c, pillSource)
	if err != nil {
		return fail(err, 1)
	}
	req.Config = protocol.Configuration{HTTPPort: uint16(port), AmesPort: uint16(ames), Loom: uint8(loom), NativeControl: native}
	op, err := c.CreateShip(ctx, req)
	req.Key = ""
	if err != nil {
		return fail(err, 1)
	}
	if err := newOutput(out, format, false).Encode(op); err != nil {
		return fail(errors.New("could not write registration result"), 1)
	}
	return 0
}
