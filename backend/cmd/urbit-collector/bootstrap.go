package main

import (
	"context"
	"errors"
	"os"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/bootpill"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/client"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func bootstrapChoice(kind, source string) (string, error) {
	if kind == "" {
		kind = "bare"
		if source != "" {
			kind = "recommended"
		}
	}
	if kind != "bare" && kind != "recommended" || kind == "bare" && source != "" || kind == "recommended" && source == "" {
		return "", errors.New("choose --bootstrap bare without --pill, or --bootstrap recommended with --pill URL_OR_FILE")
	}
	if source != "" {
		if err := bootpill.ValidateSource(source); err != nil {
			return "", err
		}
	}
	return kind, nil
}

func stageSelectedPill(ctx context.Context, c *client.Client, source string) (*protocol.BootPill, error) {
	return stageSelectedPillExpected(ctx, c, source, nil)
}

func expectedPill(source, sha string, size int64) (*protocol.BootPill, error) {
	if sha == "" && size == 0 {
		return nil, nil
	}
	p := &protocol.BootPill{SHA256: sha, Bytes: size}
	if source == "" || !protocol.ValidBootPill(p) {
		return nil, errors.New("--pill-sha256 and --pill-size require --pill and a valid expected SHA-256/byte count")
	}
	if bootpill.IsRecommended(source) && *p != bootpill.RecommendedDescriptor() {
		return nil, errors.New("expected descriptor conflicts with the Recommended Apps release")
	}
	return p, nil
}

func stageSelectedPillExpected(ctx context.Context, c *client.Client, source string, expected *protocol.BootPill) (*protocol.BootPill, error) {
	if source == "" {
		return nil, nil
	}
	dir, err := os.MkdirTemp("", "omabit-boot-pill-")
	if err != nil {
		return nil, errors.New("cannot stage selected pill")
	}
	defer os.RemoveAll(dir)
	path, meta, err := bootpill.ResolveExpected(ctx, source, dir, expected)
	if err != nil {
		return nil, err
	}
	if client.IsCreationReplayOnly(ctx) {
		// Reproduce the historical fingerprint, but do not mutate the runner's
		// artifact cache before the replay-only creation lookup can refuse it.
		return &meta, nil
	}
	f, err := bootpill.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	if err = c.StageBootPill(ctx, f, meta); err != nil {
		return nil, err
	}
	return &meta, nil
}
