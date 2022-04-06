package main

import (
	"fmt"
	"os"
	"sort"
	"strings"
)

func shellCompletion() {
	defer os.Exit(0)

	// The argument structure bash(1) uses with `complete -C` appears to
	// follow typical calling convention with argv[0], then give the most
	// recently typed argument, and then the previously typed argument as
	// some kind of confusing convenience. If the command needs the entire
	// typed command, it's available in the COMP_LINE environment variable.
	word := os.Args[3]
	previousWord := os.Args[4]
	compLine := strings.Split(os.Getenv("COMP_LINE"), " ") // not strictly correct but good enough to get non-space-containing subcommands
	fmt.Fprintf(os.Stderr, "\nword: %q, previousWord: %q, compLine: %#v\n\n", word, previousWord, compLine)

	// This should never happen since `complete -C "substrate
	// --shell-completion" "substrate"` would never even invoke this program
	// with anything less.
	if len(compLine) < 2 {
		return
	}

	// Complete subcommands and a few global options before getting into the
	// details of each subcommand's options.
	if len(compLine) == 2 && compLine[0] == "substrate" {
		candidates := []string{"--help", "--shell-completion", "--version"}
		for subcommand, _ := range dispatchMap {
			candidates = append(candidates, subcommand)
		}
		shellCompletionMatches(candidates, word)
		return
	}

	// Decide which subcommand we're completing and cover all its options.
	// This could absolutely be computed from usage messages eventually but
	// that will require a refactor of every command to expose its FlagSet.
	var subcommand string
	if compLine[0] == "substrate" {
		subcommand = compLine[1]
	} else {
		subcommand = strings.TrimPrefix(compLine[0], "substrate-")
	}
	options := []string{
		"-fully-interactive",
		"-minimally-interactive",
		"-non-interactive",
		"-version",
	}
	switch subcommand {
	case "accounts":
		if previousWord == "-format" {
			shellCompletionMatches([]string{"json", "shell"}, word)
			return
		} else {
			options = append(options, "-format")
		}
		//options = append(options, "-format json", "-format shell")
	case "assume-role":
		if previousWord == "-format" {
			shellCompletionMatches([]string{"env", "export", "json"}, word)
			return
		} else {
			options = append(options, "-admin", "-console", "-format", "-management", "-quiet")
		}
		//options = append(options, "-format env", "-format export", "-format json")
	case "bootstrap-deploy-account":
	case "bootstrap-management-account":
	case "bootstrap-network-account":
	case "create-account":
	case "create-admin-account":
	case "create-terraform-module":
	case "credentials":
		if previousWord == "-format" {
			shellCompletionMatches([]string{"env", "export", "json"}, word)
			return
		} else {
			options = append(options, "-format")
		}
		//options = append(options, "-format env", "-format export", "-format json")
	case "delete-static-access-keys":
	case "intranet-zip":
	case "root-modules":
		if previousWord == "-format" {
			shellCompletionMatches([]string{"json", "shell"}, word)
			return
		} else {
			options = append(options, "-format")
		}
		//options = append(options, "-format json", "-format shell")
	case "whoami":
		if previousWord == "-format" {
			shellCompletionMatches([]string{"env", "export", "json"}, word)
			return
		} else {
			options = append(options, "-format")
		}
		//options = append(options, "-format env", "-format export", "-format json")
	}
	shellCompletionMatches(options, word)

}

func shellCompletionMatches(candidates []string, word string) {
	sort.Strings(candidates)
	for _, candidate := range candidates {
		if strings.HasPrefix(candidate, word) {
			fmt.Println(candidate)
		}
	}
}
