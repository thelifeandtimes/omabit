package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/epochs"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func runStorage(ctx context.Context, command string, args []string, dir, socket, runner, expected, format string, out io.Writer, fail func(error, int) int) int {
	f := flag.NewFlagSet(command, flag.ContinueOnError)
	f.SetOutput(io.Discard)
	var req protocol.ImportPierRequest
	var confirm bool
	var port uint
	var storageRoot string
	if command == "preview-pier" {
		f.StringVar(&storageRoot, "storage-root", "", "configured destination location; enables automatic identity and layout review")
	}
	if command != "preview-pier" && len(args) > 0 && args[0] != "--help" {
		req.ID, args = args[0], args[1:]
	}
	if command == "preview-pier" || command == "import-pier" {
		f.StringVar(&req.Source, "source", "", "absolute existing pier path on the selected runner host")
	}
	if command != "preview-pier" {
		f.StringVar(&req.RequestID, "request-id", "", "actor-scoped request ID; no automatic retries")
	}
	if command == "storage-root-add" || command == "import-pier" {
		f.StringVar(&req.Path, "path", "", "host storage root (add) or convention-relative destination (import)")
	}
	if command == "delete-pier" {
		f.Uint64Var(&req.ExpectedRevision, "revision", 0, "selected configuration revision")
		f.BoolVar(&confirm, "confirm-delete", false, "permanently delete pier data and remove its registration")
	}
	if command == "import-pier" {
		f.BoolVar(&req.Identify, "identify", false, "recheck the automatically identified ship and fake/live network")
		f.StringVar(&req.Ship, "ship", "", "expected Urbit identity")
		f.StringVar(&req.Fingerprint, "fingerprint", "", "fingerprint returned by preview-pier")
		f.UintVar(&port, "http-port", 0, "host HTTP port")
		f.BoolVar(&req.ConfirmStopped, "confirm-stopped", false, "source is stopped; authorize move into managed storage")
	}
	if err := f.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Fprintf(out, "Usage: urbit-collector --runner NAME --expected-runner-id ID %s [NAME] [flags]\n", command)
			f.SetOutput(out)
			f.PrintDefaults()
			return 0
		}
		return fail(err, 2)
	}
	if f.NArg() != 0 || socket != "" || !validID(runner) || !validID(expected) {
		return fail(errors.New("storage commands require a pinned enrolled runner"), 2)
	}
	if command != "preview-pier" && (!validID(req.ID) || !validID(req.RequestID)) {
		return fail(errors.New("valid name and request ID required"), 2)
	}
	if (command == "preview-pier" || command == "import-pier") && (!filepath.IsAbs(req.Source) || filepath.Clean(req.Source) != req.Source) {
		return fail(errors.New("use the absolute source directory on the runner host"), 2)
	}
	if command == "storage-root-add" && (!filepath.IsAbs(req.Path) || filepath.Clean(req.Path) != req.Path) {
		return fail(errors.New("storage root must be an absolute host path"), 2)
	}
	if command == "import-pier" {
		if !strings.HasPrefix(req.Ship, "~") {
			req.Ship = "~" + req.Ship
		}
		if !protocol.ValidFakeShip(req.Ship) || !pierstore.ValidPath(req.Path) || !epochs.ValidDigest(req.Fingerprint) || !req.ConfirmStopped || port < 1024 || port > 65535 {
			return fail(errors.New("require reviewed source fingerprint, expected ship, destination, port and stopped/move confirmation"), 2)
		}
		req.HTTPPort = uint16(port)
	}
	if command == "delete-pier" && (!confirm || req.ExpectedRevision == 0) {
		return fail(errors.New("deletion requires --revision and --confirm-delete"), 2)
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
	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	h, err := c.Hello(ctx)
	if err != nil {
		return fail(err, 1)
	}
	feature := "import-pier"
	if storageRoot != "" || req.Identify {
		feature = "guided-pier-import"
	}
	if command == "storage-root-add" {
		feature = "storage-locations"
	}
	if command == "delete-pier" {
		feature = command
	}
	if h.AccessRole != "operator" || !slices.Contains(h.Features, feature) {
		return fail(errors.New("update this runner and use operator access for storage changes"), 1)
	}
	var result any
	switch command {
	case "storage-root-add":
		result, err = c.AddStorageRoot(ctx, protocol.StorageRootRequest{ID: req.ID, Path: req.Path, RequestID: req.RequestID})
	case "preview-pier":
		if storageRoot != "" {
			result, err = c.PreviewGuidedPierImport(ctx, expected, req.Source, storageRoot)
		} else {
			result, err = c.PreviewPierImport(ctx, req.Source)
		}
	case "import-pier":
		result, err = c.ImportPier(ctx, req)
	case "delete-pier":
		result, err = c.Operate(ctx, req.ID, protocol.OperationRequest{Kind: command, RequestID: req.RequestID, ExpectedRevision: req.ExpectedRevision, ConfirmDelete: true})
	}
	if err != nil {
		return fail(err, 1)
	}
	if err = newOutput(out, format, false).Encode(result); err != nil {
		return fail(err, 1)
	}
	return 0
}
