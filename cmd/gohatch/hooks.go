// SPDX-License-Identifier: EUPL-1.2
// Copyright (c) 2025 Oliver Andrich

package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"charm.land/huh/v2"
	gohatchcfg "github.com/oliverandrich/gohatch/internal/config"
)

// hookTimeout is the maximum duration for a single hook command.
const hookTimeout = 5 * time.Minute

// substituteHookVars replaces __VarName__ placeholders in a hook command.
func substituteHookVars(command string, vars map[string]string) string {
	for key, value := range vars {
		command = strings.ReplaceAll(command, "__"+key+"__", value)
	}
	return command
}

// confirmHooks asks the user to confirm hook execution.
// Defined as a variable for test overriding.
var confirmHooks = func(hooks []gohatchcfg.Hook, vars map[string]string) (bool, error) {
	fmt.Println("\nPost-generation hooks:")
	for i, h := range hooks {
		resolved := substituteHookVars(h.Command, vars)
		fmt.Printf("  %d. %s: %s\n", i+1, h.Name, resolved)
	}
	fmt.Println()

	var confirmed bool
	form := huh.NewConfirm().Title("Run these hooks?").Value(&confirmed)
	if err := form.Run(); err != nil {
		return false, err
	}
	return confirmed, nil
}

// runHooks executes post-generation hooks sequentially.
func runHooks(ctx context.Context, hooks []gohatchcfg.Hook, dir string, vars map[string]string) error {
	if len(hooks) == 0 || opts.noHooks {
		return nil
	}

	if !isInteractive() {
		fmt.Println("Warning: skipping hooks (non-interactive terminal)")
		return nil
	}

	confirmed, err := confirmHooks(hooks, vars)
	if err != nil {
		return fmt.Errorf("confirming hooks: %w", err)
	}
	if !confirmed {
		fmt.Println("Skipping hooks")
		return nil
	}

	for _, h := range hooks {
		resolved := substituteHookVars(h.Command, vars)
		fmt.Printf("Running hook: %s\n", h.Name)
		hookCtx, cancel := context.WithTimeout(ctx, hookTimeout)
		cmd := exec.CommandContext(hookCtx, "sh", "-c", resolved) //nolint:gosec // hooks are user-confirmed
		cmd.Dir = dir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		cancel()
		if err != nil {
			return fmt.Errorf("hook %q failed: %w", h.Name, err)
		}
	}

	return nil
}
