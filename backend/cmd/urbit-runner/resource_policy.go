package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/ownerhost"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/resourcepolicy"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/rootless"
	"io"
	"os/user"
)

const resourcePolicyUsage = `Usage: urbit-runner resource-policy show
       urbit-runner resource-policy set --phase new|existing [--minimum-ram-mib N]
         [--minimum-disk-mib N] [--recommended-ram-mib N] [--recommended-disk-mib N]

Run as the installed pier owner. Values are available host capacity in MiB,
not container limits. Minimum values gate start; recommendations are advisory.
New applies until an initialized pier has been recorded, including boot retries.
Existing applies to initialized ships, imports and startup recovery. Maintenance
keeps its separate resource requirements. Changes apply to subsequent admission
checks without restarting the runner. Stop remains available with a bad policy.
`

func runResourcePolicy(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || len(args) == 1 && (args[0] == "--help" || args[0] == "-h") {
		fmt.Fprint(stdout, resourcePolicyUsage)
		return 0
	}
	if len(args) == 2 && (args[1] == "--help" || args[1] == "-h") {
		return resourcePolicyAtHome("", args, stdout, stderr)
	}
	p, err := ownerhost.Current()
	home := ""
	if err == nil && p != nil {
		home = p.Home
	} else if err == nil {
		u, e := user.Current()
		if e != nil || u.Username != "urbit-runner" || u.HomeDir != rootless.HomeDir {
			err = errors.New("run resource-policy as the installed pier owner")
		} else {
			home = u.HomeDir
		}
	}
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return resourcePolicyAtHome(home, args, stdout, stderr)
}
func resourcePolicyAtHome(home string, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, resourcePolicyUsage)
		return 2
	}
	f := flag.NewFlagSet("resource-policy", flag.ContinueOnError)
	f.SetOutput(stderr)
	f.Usage = func() { fmt.Fprint(stderr, resourcePolicyUsage) }
	phase := f.String("phase", "", "new or existing")
	minRAM := f.Uint64("minimum-ram-mib", 0, "minimum available RAM in MiB")
	minDisk := f.Uint64("minimum-disk-mib", 0, "minimum free disk in MiB")
	recRAM := f.Uint64("recommended-ram-mib", 0, "recommended available RAM in MiB")
	recDisk := f.Uint64("recommended-disk-mib", 0, "recommended free disk in MiB")
	if err := f.Parse(args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if f.NArg() != 0 {
		f.Usage()
		return 2
	}
	var p resourcepolicy.Policy
	var err error
	switch args[0] {
	case "show":
		if f.NFlag() != 0 {
			f.Usage()
			return 2
		}
		p, err = resourcepolicy.Load(home)
	case "set":
		seen := map[string]bool{}
		f.Visit(func(v *flag.Flag) { seen[v.Name] = true })
		if (*phase != "new" && *phase != "existing") || len(seen) < 2 {
			f.Usage()
			return 2
		}
		p, err = resourcepolicy.Update(home, func(p *resourcepolicy.Policy) error {
			t := &p.New
			if *phase == "existing" {
				t = &p.Existing
			}
			if seen["minimum-ram-mib"] {
				t.Minimum.RAMMiB = *minRAM
			}
			if seen["minimum-disk-mib"] {
				t.Minimum.DiskMiB = *minDisk
			}
			if seen["recommended-ram-mib"] {
				t.Recommended.RAMMiB = *recRAM
			}
			if seen["recommended-disk-mib"] {
				t.Recommended.DiskMiB = *recDisk
			}
			return nil
		})
	default:
		f.Usage()
		return 2
	}
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	e := json.NewEncoder(stdout)
	e.SetIndent("", "  ")
	if err = e.Encode(p); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}
