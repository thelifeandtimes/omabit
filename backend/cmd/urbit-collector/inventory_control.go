package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/collection"
	"golang.org/x/sys/unix"
)

// The native desktop keeps stdin open for coalesced schedule updates. This pipe
// carries no selection, credentials or actions. EOF releases the watch owner.
func readInventoryControls(input io.Reader, cadence *collection.WatchCadence) error {
	scanner := bufio.NewScanner(input)
	scanner.Buffer(make([]byte, 1024), 1024)
	for scanner.Scan() {
		var command struct {
			Interval string `json:"interval"`
			Refresh  bool   `json:"refresh"`
		}
		decoder := json.NewDecoder(bytes.NewReader(scanner.Bytes()))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&command); err != nil {
			return errors.New("invalid inventory control; use interval and optional refresh")
		}
		if decoder.Decode(new(any)) != io.EOF {
			return errors.New("inventory control requires one JSON object per line")
		}
		duration, err := time.ParseDuration(command.Interval)
		if err != nil {
			return errors.New("inventory control requires a duration interval")
		}
		if err = cadence.Update(duration, command.Refresh); err != nil {
			return err
		}
	}
	if scanner.Err() != nil {
		return errors.New("inventory control input failed or exceeded 1024 bytes")
	}
	return nil
}

func controlledInventory(ctx context.Context, entries []collection.Connection, interval time.Duration, input *os.File, emit func(collection.Observation) error) error {
	cadence, err := collection.NewWatchCadence(interval)
	if err != nil {
		return err
	}
	// Inherited stdio may be blocking and outside Go's poller. As with the SSH
	// bridge, use a pollable duplicate so cancellation joins a blocked reader.
	fd, err := unix.FcntlInt(input.Fd(), unix.F_DUPFD_CLOEXEC, 0)
	if err != nil {
		return err
	}
	if err = unix.SetNonblock(fd, true); err != nil {
		unix.Close(fd)
		return err
	}
	reader := os.NewFile(uintptr(fd), "inventory-control")
	defer reader.Close()
	parent := ctx
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	done := make(chan error, 1)
	go func() { err := readInventoryControls(reader, cadence); done <- err; cancel() }()
	err = collection.WatchWithCadence(ctx, entries, cadence, emit)
	reader.Close()
	controlErr := <-done
	if parent.Err() != nil {
		return parent.Err()
	}
	if err != nil && !errors.Is(err, context.Canceled) {
		return err
	}
	return controlErr // clean EOF is a successful watch-owner shutdown
}
