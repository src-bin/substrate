package main

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/src-bin/substrate/naming"
	"github.com/src-bin/substrate/ui"
)

func shellCompletion() {
	defer os.Exit(0)

	// The argument structure bash(1) uses with `complete -C` appears to
	// follow typical calling convention with argv[0], then give the most
	// recently typed argument, and then the previously typed argument as
	// some kind of confusing convenience. If the command needs the entire
	// typed command, it's available in the COMP_LINE environment variable.
	var word, previousWord string
	if len(os.Args) > 3 {
		word = os.Args[3]
	}
	if len(os.Args) > 4 {
		previousWord = os.Args[4]
	}
	compCWord, _ := strconv.Atoi(os.Getenv("COMP_CWORD"))
	compLine := strings.Split(os.Getenv("COMP_LINE"), " ") // not strictly correct but good enough to get non-space-containing subcommands
	//fmt.Fprintf(os.Stderr, "\nword: %q, previousWord: %q, compLine: %#v, compCWord: %#v\n\n", word, previousWord, compLine, compCWord)

	// zsh(1), however, doesn't do things quite like bash(1). So, if we find
	// ourselves with an empty word and a non-zero COMP_CWORD, try to use
	// that instead.
	if word == "" && compCWord != 0 && len(compLine) > compCWord {
		word = compLine[compCWord]
	}
	if previousWord == "" && compCWord != 0 && len(compLine) > compCWord {
		previousWord = compLine[compCWord-1]
	}
	//fmt.Fprintf(os.Stderr, "\nword: %q, previousWord: %q, compLine: %#v, compCWord: %#v\n\n", word, previousWord, compLine, compCWord)

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
		}
		options = append(options, "-auto-approve", "-format", "-no-apply", "-number", "-only-tags")
	case "assume-role":
		// TODO autocomplete for domains
		if previousWord == "-environment" {
			environments, err := naming.Environments()
			if err != nil {
				ui.Fatal(err)
			}
			shellCompletionMatches(environments, word)
			return
		}
		if previousWord == "-format" {
			shellCompletionMatches([]string{"env", "export", "json"}, word)
			return
		}
		if previousWord == "-quality" {
			qualities, err := naming.Qualities()
			if err != nil {
				ui.Fatal(err)
			}
			shellCompletionMatches(qualities, word)
			return
		}
		options = append(options, "-admin", "-console", "-domain", "-environment", "-format", "-management", "-number", "-quality", "-quiet", "-role", "-special")
	case "bootstrap-deploy-account":
		options = append(options, "-auto-approve", "-no-apply")
	case "bootstrap-management-account":
	case "bootstrap-network-account":
		options = append(options, "-auto-approve", "-ignore-service-quotas", "-no-apply")
	case "create-account":
		// TODO autocomplete for domains
		if previousWord == "-environment" {
			environments, err := naming.Environments()
			if err != nil {
				ui.Fatal(err)
			}
			shellCompletionMatches(environments, word)
			return
		}
		if previousWord == "-quality" {
			qualities, err := naming.Qualities()
			if err != nil {
				ui.Fatal(err)
			}
			shellCompletionMatches(qualities, word)
			return
		}
		options = append(options, "-auto-approve", "-create", "-domain", "-environment", "-ignore-service-quotas", "-no-apply", "-quality")
	case "create-admin-account":
		if previousWord == "-quality" {
			qualities, err := naming.Qualities()
			if err != nil {
				ui.Fatal(err)
			}
			shellCompletionMatches(qualities, word)
			return
		}
		options = append(options, "-auto-approve", "-create", "-ignore-service-quotas", "-no-apply", "-quality")
	case "create-terraform-module":
	case "credentials":
		if previousWord == "-format" {
			shellCompletionMatches([]string{"env", "export", "json"}, word)
			return
		}
		options = append(options, "-format", "-quiet")
	case "delete-static-access-keys":
	case "intranet-zip":
		if previousWord == "-format" {
			shellCompletionMatches([]string{"json", "text"}, word)
			return
		}
		options = append(options, "-base64sha256", "-format")
	case "root-modules":
		if previousWord == "-format" {
			shellCompletionMatches([]string{"json", "shell"}, word)
			return
		}
		options = append(options, "-format", "-quiet")
	case "terraform", "upgrade":
		options = append(options, "-no", "-yes")
	case "whoami":
		if previousWord == "-format" {
			shellCompletionMatches([]string{"env", "export", "json"}, word)
			return
		}
		options = append(options, "-format", "-quiet")
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
