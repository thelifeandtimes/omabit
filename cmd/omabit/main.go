// omabit owns desktop/application policy; urbit-collector remains generic.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"sort"
	"sync"
	"syscall"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/collection"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/omabittheme"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	code := run(ctx, os.Args[1:])
	cancel()
	os.Exit(code)
}
func run(ctx context.Context, args []string) int {
	format := "json"
	fail := func(err error) int {
		if format == "text" {
			fmt.Fprintln(os.Stderr, err)
		} else {
			_ = json.NewEncoder(os.Stderr).Encode(map[string]any{"error": map[string]string{"code": "omabit_error", "message": err.Error()}})
		}
		return 1
	}
	flags := flag.NewFlagSet("omabit", flag.ContinueOnError)
	config := flags.String("config-dir", "", "collector config directory")
	stateDir := flags.String("state-dir", "", "Omabit preference directory")
	flags.StringVar(&format, "format", "json", "json or text")
	flags.Usage = func() {
		fmt.Fprintln(os.Stdout, "Usage: omabit [--format text] [--config-dir DIR] [--state-dir DIR] COMMAND\nCommands: theme-preview, theme-status, theme-sync, theme-auto NAME --expected-ship-id ID --enabled=true|false, theme-publish NAME --expected-ship-id ID, theme-migrate [--apply --expected-digest DIGEST] [--legacy-dir DIR]")
	}
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	args = flags.Args()
	if len(args) == 0 || (format != "json" && format != "text") {
		flags.Usage()
		return 2
	}
	command := args[0]
	args = args[1:]
	output := func(value any) int {
		if format == "text" && readable(os.Stdout, value) {
			return 0
		}
		data, err := json.MarshalIndent(value, "", "  ")
		if err != nil {
			return fail(err)
		}
		fmt.Println(string(data))
		return 0
	}
	if command == "theme-preview" {
		if len(args) != 0 {
			return 2
		}
		p, err := omabittheme.Resolve(ctx)
		if err != nil {
			return fail(err)
		}
		return output(p)
	}
	if *config == "" {
		dir, err := collection.DefaultDir()
		if err != nil {
			return fail(err)
		}
		*config = dir
	}
	r, err := collection.Open(*config)
	if err != nil {
		return fail(err)
	}
	if *stateDir == "" {
		dir, err := omabittheme.DefaultDir()
		if err != nil {
			return fail(err)
		}
		*stateDir = dir
	}
	store, err := omabittheme.OpenStore(*stateDir)
	if err != nil {
		return fail(err)
	}
	if command == "theme-migrate" {
		sub := flag.NewFlagSet(command, flag.ContinueOnError)
		legacy := sub.String("legacy-dir", "", "legacy state directory")
		apply := sub.Bool("apply", false, "explicitly pause the old publisher and transfer its sessions/preferences")
		expected := sub.String("expected-digest", "", "required source digest from preview")
		if err := sub.Parse(args); err != nil || sub.NArg() != 0 {
			return 2
		}
		if *legacy == "" {
			dir, err := omabittheme.LegacyDir()
			if err != nil {
				return fail(err)
			}
			*legacy = dir
		}
		if !*apply {
			plan, err := omabittheme.PlanMigration(ctx, r, *legacy)
			if err != nil {
				return fail(err)
			}
			return output(plan)
		}
		if err := omabittheme.ApplyMigration(ctx, r, store, *legacy, *expected, omabittheme.DesktopMigration{}); err != nil {
			return fail(err)
		}
		return output(map[string]any{"status": "migrated", "legacy_publisher": "disabled", "credentials": "transferred without logout"})
	}
	if command == "theme-status" {
		if len(args) != 0 {
			return 2
		}
		state, err := store.Snapshot(ctx)
		if err != nil {
			return fail(err)
		}
		return output(state)
	}
	if command == "theme-sync" {
		if len(args) != 0 {
			return 2
		}
		palette, err := omabittheme.Resolve(ctx)
		if err != nil {
			return fail(err)
		}
		ships, err := r.Ships(ctx)
		if err != nil {
			return fail(err)
		}
		var wg sync.WaitGroup
		var mu sync.Mutex
		failed := []string{}
		for _, ship := range ships {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if err := omabittheme.Sync(ctx, r, store, ship, palette, false); err != nil {
					mu.Lock()
					failed = append(failed, ship.Name)
					mu.Unlock()
				}
			}()
		}
		wg.Wait()
		sort.Strings(failed)
		output(map[string]any{"complete": len(failed) == 0, "failed": failed})
		if len(failed) > 0 {
			return 1
		}
		return 0
	}
	if command != "theme-auto" && command != "theme-publish" {
		return 2
	}
	if len(args) == 0 {
		return 2
	}
	name := args[0]
	sub := flag.NewFlagSet(command, flag.ContinueOnError)
	id := sub.String("expected-ship-id", "", "required enrollment identity")
	enabled := sub.Bool("enabled", false, "explicit automatic publishing preference")
	if err := sub.Parse(args[1:]); err != nil {
		return 2
	}
	if sub.NArg() != 0 || *id == "" {
		return 2
	}
	ship, err := r.Ship(ctx, name)
	if err != nil {
		return fail(err)
	}
	if ship.ID != *id {
		return fail(collection.ErrShipConflict)
	}
	if command == "theme-auto" {
		set := false
		sub.Visit(func(f *flag.Flag) { set = set || f.Name == "enabled" })
		if !set {
			return 2
		}
		if err := store.Auto(ctx, ship, *enabled); err != nil {
			return fail(err)
		}
	} else {
		state, err := store.Snapshot(ctx)
		if err != nil {
			return fail(err)
		}
		if _, ok := state.Ships[ship.ID]; !ok {
			if err := store.Auto(ctx, ship, false); err != nil {
				return fail(err)
			}
		}
		palette, err := omabittheme.Resolve(ctx)
		if err != nil {
			return fail(err)
		}
		if err := omabittheme.Sync(ctx, r, store, ship, palette, true); err != nil {
			return fail(err)
		}
	}
	state, err := store.Snapshot(ctx)
	if err != nil {
		return fail(err)
	}
	return output(state.Ships[ship.ID])
}
