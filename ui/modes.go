package ui

import (
	"github.com/spf13/pflag"
)

type InteractivityLevel int

// InteractivityLevel constants are ordered such that ordering comparisons
// make as much sense as equality comparisons.
const (
	NonInteractive InteractivityLevel = iota
	MinimallyInteractive
	FullyInteractive
)

func Interactivity() InteractivityLevel {
	if fullyInteractive == nil || minimallyInteractive == nil || nonInteractive == nil { // true if InteractivityFlags was never called
		return MinimallyInteractive // default
	}

	if !*fullyInteractive && !*minimallyInteractive && !*nonInteractive {
		return MinimallyInteractive // default
	}
	if !*fullyInteractive && !*minimallyInteractive && *nonInteractive {
		return NonInteractive
	}
	if !*fullyInteractive && *minimallyInteractive && !*nonInteractive {
		return MinimallyInteractive
	}
	if *fullyInteractive && !*minimallyInteractive && !*nonInteractive {
		return FullyInteractive
	}
	Fatal("can't mix --non-interactive, --minimally-interactive, and --fully-interactive")
	return 0
}

func InteractivityFlagSet() *pflag.FlagSet {
	set := pflag.NewFlagSet("[interactivity flags]", pflag.ExitOnError)
	set.BoolVar(fullyInteractive, "fully-interactive", false, "fully interactive mode - all prompts and file editors")
	set.BoolVar(minimallyInteractive, "minimally-interactive", false, "minimally interactive mode - only prompts with no cached responses (default)")
	set.BoolVar(nonInteractive, "non-interactive", false, "non-interactive mode - no prompts or file editors")
	return set
}

func Quiet() {
	op(opQuiet, "")
}

var fullyInteractive, minimallyInteractive, nonInteractive = new(bool), new(bool), new(bool)
