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
	return kind, nil
}

func stageSelectedPill(ctx context.Context, c *client.Client, source string) (*protocol.BootPill, error) {
	if source == "" {
		return nil, nil
	}
	dir, err := os.MkdirTemp("", "omabit-boot-pill-")
	if err != nil {
		return nil, errors.New("cannot stage selected pill")
	}
	defer os.RemoveAll(dir)
	path, meta, err := bootpill.Resolve(ctx, source, dir)
	if err != nil {
		return nil, err
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
