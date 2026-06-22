package main

import (
	"flag"
	"fmt"
	"os"

	"gojvm-backend/src/cmd/compile/internal/gojvm/checks"
)

func main() {
	mode := flag.String("buildmode", "exe", "build mode to validate")
	root := flag.String("root", ".", "path to scan for unsupported JVM targets")
	flag.Parse()

	issues, err := checks.ValidateSources([]string{*root})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if err := checks.ValidateBuildMode(*mode); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	for _, issue := range issues {
		if issue.Line > 0 {
			fmt.Fprintf(os.Stderr, "%s:%d:%d: %s (%s)\n", issue.File, issue.Line, issue.Column, issue.Message, issue.Kind)
		} else {
			fmt.Fprintf(os.Stderr, "%s: %s (%s)\n", issue.File, issue.Message, issue.Kind)
		}
	}

	if len(issues) != 0 {
		os.Exit(1)
	}
}
