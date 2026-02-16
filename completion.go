package kommando

import (
	"fmt"
	"io"
	"strings"
)

// Shell represents a shell type for completion script generation.
type Shell string

// String returns the string representation of the Shell.
func (s Shell) String() string {
	return string(s)
}

const (
	Bash       Shell = "bash"
	Zsh        Shell = "zsh"
	Fish       Shell = "fish"
	PowerShell Shell = "powershell"
)

// GenerateCompletion writes a shell completion script for the application.
func (a *App) GenerateCompletion(w io.Writer, shell Shell) error {
	a.ensureHelp()

	switch shell {
	case Bash:
		return a.generateBash(w)
	case Zsh:
		return a.generateZsh(w)
	case Fish:
		return a.generateFish(w)
	case PowerShell:
		return a.generatePowerShell(w)
	default:
		return fmt.Errorf("%w: %s", ErrUnsupportedShell, shell)
	}
}

// ── helpers ──────────────────────────────────────────────────────────────

// completionFlagList returns command flags merged with global flags,
// deduplicating by name. Command flags take precedence.
func (a *App) completionFlagList(cmdFlags []Flag) []Flag {
	seen := make(map[string]bool, len(cmdFlags))
	flags := make([]Flag, 0, len(cmdFlags)+len(a.globalFlags))
	flags = append(flags, cmdFlags...)
	for _, f := range cmdFlags {
		seen[f.Name] = true
	}
	for _, gf := range a.globalFlags {
		if !seen[gf.Name] {
			flags = append(flags, gf)
		}
	}
	return flags
}

// completionFlagNames returns flag option strings (--name, -x) for the
// given command flags merged with global flags.
func (a *App) completionFlagNames(cmdFlags []Flag) []string {
	var opts []string
	for _, f := range a.completionFlagList(cmdFlags) {
		opts = append(opts, "--"+f.Name)
		if f.Short != 0 {
			opts = append(opts, fmt.Sprintf("-%c", f.Short))
		}
	}
	return opts
}

// ── Bash ─────────────────────────────────────────────────────────────────
//
// Strategy: generate a single function that resolves the active command
// path by walking COMP_WORDS, then completes based on that path.
//
//   COMP_WORDS: [myapp, server, start, --po]
//   Resolver:   ROOT -> ROOT/server -> ROOT/server/start
//   Complete:   flags for ROOT/server/start matching "--po"

func (a *App) generateBash(w io.Writer) error {
	fmt.Fprintf(w, `_%s_completions() {
    local cur="${COMP_WORDS[COMP_CWORD]}"
    COMPREPLY=()

    # Resolve the deepest subcommand path from COMP_WORDS.
    local path="ROOT"
    local i=1
    while [[ $i -lt $COMP_CWORD ]]; do
        case "${COMP_WORDS[$i]}" in
            -*) ;;
            *)
                case "${path}/${COMP_WORDS[$i]}" in
`, a.name)

	a.bashResolverEntries(w, a.commands, "ROOT")

	fmt.Fprintf(w, `                esac
                ;;
        esac
        ((i++))
    done

    # Complete based on the resolved path.
    local opts=""
    case "$path" in
`)

	a.bashCompletionEntry(w, "ROOT", a.commands, nil)
	a.bashCompletionTree(w, a.commands, "ROOT")

	fmt.Fprintf(w, `    esac
    COMPREPLY=( $(compgen -W "$opts" -- "$cur") )
}

complete -F _%s_completions %s
`, a.name, a.name)
	return nil
}

// bashResolverEntries writes case patterns that map COMP_WORDS to canonical paths.
// Aliases are mapped to the canonical command name path.
func (a *App) bashResolverEntries(w io.Writer, cmds []*Command, prefix string) {
	for _, cmd := range cmds {
		canonical := prefix + "/" + cmd.Name
		patterns := []string{canonical}
		for _, alias := range cmd.Aliases {
			patterns = append(patterns, prefix+"/"+alias)
		}
		fmt.Fprintf(w, "                    %s) path=%q ;;\n",
			strings.Join(patterns, "|"), canonical)
		if len(cmd.SubCommands) > 0 {
			a.bashResolverEntries(w, cmd.SubCommands, canonical)
		}
	}
}

// bashCompletionEntry writes a single case entry mapping a path to its completions.
func (a *App) bashCompletionEntry(w io.Writer, path string, subs []*Command, cmdFlags []Flag) {
	var opts []string
	for _, sub := range subs {
		opts = append(opts, sub.Name)
		opts = append(opts, sub.Aliases...)
	}
	opts = append(opts, a.completionFlagNames(cmdFlags)...)
	if len(opts) > 0 {
		fmt.Fprintf(w, "        %s) opts=%q ;;\n", path, strings.Join(opts, " "))
	}
}

// bashCompletionTree recursively writes completion entries for all commands.
func (a *App) bashCompletionTree(w io.Writer, cmds []*Command, prefix string) {
	for _, cmd := range cmds {
		path := prefix + "/" + cmd.Name
		a.bashCompletionEntry(w, path, cmd.SubCommands, cmd.Flags)
		if len(cmd.SubCommands) > 0 {
			a.bashCompletionTree(w, cmd.SubCommands, path)
		}
	}
}

// ── Zsh ──────────────────────────────────────────────────────────────────
//
// Strategy: generate one zsh function per command node. Each function uses
// _arguments -C to route into subcommand functions, enabling unlimited depth.
//
//   _myapp            -> routes to _myapp__server, _myapp__deploy, ...
//   _myapp__server    -> routes to _myapp__server__start, ...
//   _myapp__server__start -> completes flags only (leaf)

func (a *App) generateZsh(w io.Writer) error {
	fmt.Fprintf(w, "#compdef %s\n\n", a.name)
	a.zshCommandFunc(w, a.name, a.commands, nil)
	fmt.Fprintf(w, "_%s\n", a.name)
	return nil
}

// zshCommandFunc generates a zsh completion function for a command node
// and recursively generates functions for all subcommands.
func (a *App) zshCommandFunc(w io.Writer, funcName string, subs []*Command, cmdFlags []Flag) {
	fmt.Fprintf(w, "_%s() {\n", funcName)

	flags := a.completionFlagList(cmdFlags)

	if len(subs) > 0 {
		fmt.Fprintf(w, "    local line state\n\n")
		fmt.Fprintf(w, "    _arguments -C \\\n")
		for _, f := range flags {
			desc := strings.ReplaceAll(f.Description, "'", "'\\''")
			fmt.Fprintf(w, "        '--%s[%s]' \\\n", f.Name, desc)
		}
		fmt.Fprintf(w, "        '1:command:->cmds' \\\n")
		fmt.Fprintf(w, "        '*::arg:->args'\n\n")

		fmt.Fprintf(w, "    case $state in\n")
		fmt.Fprintf(w, "    cmds)\n")
		fmt.Fprintf(w, "        local -a commands\n")
		fmt.Fprintf(w, "        commands=(\n")
		for _, sub := range subs {
			desc := strings.ReplaceAll(sub.Description, "'", "'\\''")
			fmt.Fprintf(w, "            '%s:%s'\n", sub.Name, desc)
			for _, alias := range sub.Aliases {
				fmt.Fprintf(w, "            '%s:%s'\n", alias, desc)
			}
		}
		fmt.Fprintf(w, "        )\n")
		fmt.Fprintf(w, "        _describe 'command' commands\n")
		fmt.Fprintf(w, "        ;;\n")

		fmt.Fprintf(w, "    args)\n")
		fmt.Fprintf(w, "        case ${line[1]} in\n")
		for _, sub := range subs {
			names := []string{sub.Name}
			names = append(names, sub.Aliases...)
			childFunc := funcName + "__" + sub.Name
			fmt.Fprintf(w, "        %s) _%s ;;\n", strings.Join(names, "|"), childFunc)
		}
		fmt.Fprintf(w, "        esac\n")
		fmt.Fprintf(w, "        ;;\n")
		fmt.Fprintf(w, "    esac\n")
	} else if len(flags) > 0 {
		fmt.Fprintf(w, "    _arguments \\\n")
		for i, f := range flags {
			desc := strings.ReplaceAll(f.Description, "'", "'\\''")
			trail := " \\"
			if i == len(flags)-1 {
				trail = ""
			}
			fmt.Fprintf(w, "        '--%s[%s]'%s\n", f.Name, desc, trail)
		}
	}

	fmt.Fprintf(w, "}\n\n")

	// Recurse: generate a function for each subcommand.
	for _, sub := range subs {
		childFunc := funcName + "__" + sub.Name
		a.zshCommandFunc(w, childFunc, sub.SubCommands, sub.Flags)
	}
}

// ── Fish ─────────────────────────────────────────────────────────────────
//
// Strategy: use chained conditions that include aliases.
//
//   server (alias s):  condition = "__fish_seen_subcommand_from server s"
//   server > start:    condition += "; and __fish_seen_subcommand_from start"

func (a *App) generateFish(w io.Writer) error {
	fmt.Fprintf(w, "complete -c %s -f\n\n", a.name)

	for _, cmd := range a.commands {
		fmt.Fprintf(w, "complete -c %s -n '__fish_use_subcommand' -a %s -d %q\n",
			a.name, cmd.Name, cmd.Description)
		for _, alias := range cmd.Aliases {
			fmt.Fprintf(w, "complete -c %s -n '__fish_use_subcommand' -a %s -d %q\n",
				a.name, alias, cmd.Description)
		}
	}
	fmt.Fprintln(w)

	for _, cmd := range a.commands {
		names := cmd.Name
		for _, alias := range cmd.Aliases {
			names += " " + alias
		}
		a.writeFishCommand(w, cmd, "__fish_seen_subcommand_from "+names)
	}

	return nil
}

func (a *App) writeFishCommand(w io.Writer, cmd *Command, condition string) {
	// Subcommands.
	for _, sub := range cmd.SubCommands {
		fmt.Fprintf(w, "complete -c %s -n '%s' -a %s -d %q\n",
			a.name, condition, sub.Name, sub.Description)
		for _, alias := range sub.Aliases {
			fmt.Fprintf(w, "complete -c %s -n '%s' -a %s -d %q\n",
				a.name, condition, alias, sub.Description)
		}

		subNames := sub.Name
		for _, alias := range sub.Aliases {
			subNames += " " + alias
		}
		a.writeFishCommand(w, sub, condition+"; and __fish_seen_subcommand_from "+subNames)
	}

	// Flags (command + global).
	for _, f := range a.completionFlagList(cmd.Flags) {
		short := ""
		if f.Short != 0 {
			short = fmt.Sprintf(" -s %c", f.Short)
		}
		fmt.Fprintf(w, "complete -c %s -n '%s' -l %s%s -d %q\n",
			a.name, condition, f.Name, short, f.Description)
	}
}

// ── PowerShell ───────────────────────────────────────────────────────────
//
// Strategy: generate a flat path-based lookup table (like Bash) and a
// resolver table for aliases. Walk $commandAst tokens to find the deepest
// matching path.

func (a *App) generatePowerShell(w io.Writer) error {
	fmt.Fprintf(w, `Register-ArgumentCompleter -Native -CommandName %s -ScriptBlock {
    param($wordToComplete, $commandAst, $cursorPosition)

`, a.name)

	// Write the completions lookup table.
	fmt.Fprintf(w, "    $completions = @{\n")
	a.poshCompletionEntry(w, "ROOT", a.commands, nil)
	a.poshCompletionTree(w, a.commands, "ROOT")
	fmt.Fprintf(w, "    }\n\n")

	// Write the alias resolver table.
	fmt.Fprintf(w, "    $resolve = @{\n")
	a.poshResolverEntries(w, a.commands, "ROOT")
	fmt.Fprintf(w, "    }\n")

	fmt.Fprintf(w, `
    # Resolve the deepest subcommand path.
    $line = $commandAst.ToString()
    $tokens = $line -split '\s+'
    $path = 'ROOT'
    for ($i = 1; $i -lt ($tokens.Count - 1); $i++) {
        $t = $tokens[$i]
        if ($t -notlike '-*') {
            $try = "$path/$t"
            if ($resolve.ContainsKey($try)) { $try = $resolve[$try] }
            if ($completions.ContainsKey($try)) { $path = $try }
        }
    }

    if ($completions.ContainsKey($path)) {
        $completions[$path] | Where-Object { $_ -like "$wordToComplete*" } | ForEach-Object {
            [System.Management.Automation.CompletionResult]::new($_, $_, 'ParameterValue', $_)
        }
    }
}
`)
	return nil
}

// poshCompletionEntry writes a single path -> options entry.
func (a *App) poshCompletionEntry(w io.Writer, path string, subs []*Command, cmdFlags []Flag) {
	var items []string
	for _, sub := range subs {
		items = append(items, fmt.Sprintf("'%s'", sub.Name))
		for _, alias := range sub.Aliases {
			items = append(items, fmt.Sprintf("'%s'", alias))
		}
	}
	for _, f := range a.completionFlagNames(cmdFlags) {
		items = append(items, fmt.Sprintf("'%s'", f))
	}
	if len(items) > 0 {
		fmt.Fprintf(w, "        '%s' = @(%s)\n", path, strings.Join(items, ", "))
	}
}

// poshCompletionTree recursively writes completion entries for all commands.
func (a *App) poshCompletionTree(w io.Writer, cmds []*Command, prefix string) {
	for _, cmd := range cmds {
		path := prefix + "/" + cmd.Name
		a.poshCompletionEntry(w, path, cmd.SubCommands, cmd.Flags)
		if len(cmd.SubCommands) > 0 {
			a.poshCompletionTree(w, cmd.SubCommands, path)
		}
	}
}

// poshResolverEntries writes alias -> canonical path mappings.
func (a *App) poshResolverEntries(w io.Writer, cmds []*Command, prefix string) {
	for _, cmd := range cmds {
		canonical := prefix + "/" + cmd.Name
		for _, alias := range cmd.Aliases {
			fmt.Fprintf(w, "        '%s/%s' = '%s'\n", prefix, alias, canonical)
		}
		if len(cmd.SubCommands) > 0 {
			a.poshResolverEntries(w, cmd.SubCommands, canonical)
		}
	}
}
