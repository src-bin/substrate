package shellcompletion

import (
	"context"
	"flag"
	"fmt"
	"sort"
	"strings"

	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/ui"
	"github.com/src-bin/substrate/version"
)

// <https://kubernetes.io/docs/tasks/tools/included/optional-kubectl-configs-bash-linux/>
// echo 'source <(kubectl completion bash)' >>~/.bashrc
// <https://github.com/devstructure/blueprint/blob/master/etc/bash_completion.d/blueprint>
// <https://www.gnu.org/software/bash/manual/html_node/Programmable-Completion.html>
// <https://tldp.org/LDP/abs/html/tabexpansion.html>

//go:generate go run ../../../tools/dispatch-map/main.go -package shellcompletion ..

func Main(context.Context, *awscfg.Main) {
	shell := flag.String("shell", "bash", `shell to target for autocompletion (default and only supported option is "bash")`)
	flag.Parse()
	version.Flag()
	if *shell != "bash" {
		ui.Fatal(`-shell="bash" is the only supported option`)
	}
	if flag.Arg(0) != "substrate" {
		ui.Fatalf("this is autocomplete for substrate, not %s", flag.Arg(0))
	}

	// The argument structure bash(1) uses with `complete -C` appears to
	// follow typical calling convention with argv[0], then give the most
	// recently typed argument, and then the previously typed argument as
	// some kind of confusing convenience. If the command needs the entire
	// typed command, it's available in the COMP_LINE environment variable.
	word := flag.Arg(1)
	//log.Printf("word: %q", word)
	previousWord := flag.Arg(2)
	//log.Printf("previousWord: %q", previousWord)

	if previousWord == "substrate" {
		if _, ok := dispatchMap[word]; ok {
			fmt.Println(word)
			return
		}
		var subcommands []string
		for subcommand, _ := range dispatchMap {
			if strings.HasPrefix(subcommand, word) {
				subcommands = append(subcommands, subcommand)
				//log.Printf("prefix match: %q for %q", word, subcommand)
			}
		}
		sort.Strings(subcommands)
		for _, subcommand := range subcommands {
			fmt.Println(subcommand)
		}
		return
	}

	/*
		log.Printf("%#v", os.Args)
		log.Printf("%#v", flag.Args())
		for _, envVar := range os.Environ() {
			if strings.HasPrefix(envVar, "COMP") {
				log.Printf("%v", envVar)
			}
		}
	*/

	/*
	   	fmt.Print(`_substrate_bash_completion() {
	   	case "$COMP_CWORD" in
	   		0)
	   			return 0;;
	   		1)
	   			if [ "${COMP_WORDS[0]}" != "substrate" ]
	   			then return 1
	   			fi
	   			COMPREPLY=( $(compgen -W "accounts assume-role TODO et cetera" -- "${COMP_WORDS[COMP_CWORD]}") )
	   			return 0;;
	   		*)
	   			if [ "${COMP_WORDS[0]}" != "substrate" ]
	   			then return 1
	   			fi
	   			case "${COMP_WORDS[1]}" in
	   				"accounts")
	   					COMPREPLY=( $(compgen -W "foo bar baz" -- "${COMP_WORDS[COMP_CWORD]}") )
	   					return 0;;
	   				"assume-role")
	   					return 0;;
	   				*)
	   					return 1;;
	   			esac;;
	   	esac
	   	return 0
	   }
	   complete -F _substrate_bash_completion substrate
	   `)
	*/
}
