package main

import (
	"fmt"
	"io"
	"os"

	governance "github.com/oshethai/oshe-platform/modules/contract-migration-governance"
)

// run executes the compatibility validation CLI and returns the process exit code.
func run(args []string, stdout, stderr io.Writer) int {
	if len(args) < 2 {
		fmt.Fprintf(stderr, "Usage: compatibility-check <path-to-json-fixture>\n")
		return 1
	}

	filePath := args[1]
	data, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Fprintf(stderr, "Error reading file %q: %v\n", filePath, err)
		return 1
	}

	reg, errs := governance.ValidateRegistryJSON(data)
	if len(errs) > 0 {
		fmt.Fprintf(stderr, "Compatibility registry validation FAILED with %d error(s):\n", len(errs))
		for _, e := range errs {
			fmt.Fprintf(stderr, "  - %s\n", e)
		}
		return 1
	}

	fmt.Fprintf(stdout, "Compatibility registry validation PASSED (%d contract version(s) registered)\n", reg.Count())
	return 0
}

func main() {
	os.Exit(run(os.Args, os.Stdout, os.Stderr))
}
