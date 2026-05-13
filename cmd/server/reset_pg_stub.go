//go:build !dev

package main

import (
	"fmt"
	"os"
)

func runResetPGCommand(_ []string) {
	fmt.Fprintln(os.Stderr, "reset-pg is only available in dev builds; run with -tags dev")
	os.Exit(1)
}
