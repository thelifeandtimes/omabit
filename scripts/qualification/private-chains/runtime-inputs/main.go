// This qualification entry point only inspects explicitly selected inert inputs.
// It is not a public runner lifecycle command or an installation operation.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/privatechain"
)

func main() {
	directory := flag.String("directory", "", "Absolute path to the sealed owner-only input bundle")
	manifest := flag.String("manifest-sha256", "", "Exact manifest hash obtained during separate input review")
	foundation := flag.String("foundation-digest", "", "Foundation digest from the separately reviewed private-chain plan")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "Read-only private-chain runtime input inspection. No execution, chain access, installation or startup authority.")
		flag.PrintDefaults()
	}
	flag.Parse()
	if flag.NArg() != 0 || *directory == "" || *manifest == "" || *foundation == "" {
		flag.Usage()
		os.Exit(2)
	}
	result, err := privatechain.InspectRuntimeInputs(context.Background(), *directory, *manifest, *foundation)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Private-chain runtime input inspection refused.")
		os.Exit(1)
	}
	if json.NewEncoder(os.Stdout).Encode(result) != nil {
		os.Exit(1)
	}
}
