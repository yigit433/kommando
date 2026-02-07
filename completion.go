package kommando

import (
	"fmt"
	"io"
	"strings"
)

// Shell represents a shell type for completion script generation.
type Shell string

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
		return fmt.Errorf("unsupported shell: %s", shell)
	}
}

func (a *App) generateBash(w io.Writer) error {
	fmt.Fprintf(w, `_%s_completions() {
    local cur prev commands
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"

`, a.name)

	// Build command list.
	var cmdNames []string
	for _, cmd := range a.commands {
		cmdNames = append(cmdNames, cmd.Name)
		cmdNames = append(cmdNames, cmd.Aliases...)
	}

	fmt.Fprintf(w, "    commands=%q\n\n", strings.Join(cmdNames, " "))

	// If completing first argument, suggest commands.
	fmt.Fprintf(w, `    if [[ ${COMP_CWORD} -eq 1 ]]; then
        COMPREPLY=( $(compgen -W "${commands}" -- "${cur}") )
        return 0
    fi

`)

	// Per-command completions.
	fmt.Fprintf(w, "    case \"${COMP_WORDS[1]}\" in\n")
	for _, cmd := range a.commands {
		names := []string{cmd.Name}
		names = append(names, cmd.Aliases...)
		fmt.Fprintf(w, "    %s)\n", strings.Join(names, "|"))
		a.writeBashCommandCase(w, cmd, 2)
		fmt.Fprintf(w, "        ;;\n")
	}
	fmt.Fprintf(w, "    esac\n}\n\n")
	fmt.Fprintf(w, "complete -F _%s_completions %s\n", a.name, a.name)
	return nil
}

func (a *App) writeBashCommandCase(w io.Writer, cmd *Command, depth int) {
	// Collect flags (command + global).
	var flagOpts []string
	for _, f := range cmd.Flags {
		flagOpts = append(flagOpts, "--"+f.Name)
		if f.Short != 0 {
			flagOpts = append(flagOpts, fmt.Sprintf("-%c", f.Short))
		}
	}
	for _, gf := range a.globalFlags {
		flagOpts = append(flagOpts, "--"+gf.Name)
		if gf.Short != 0 {
			flagOpts = append(flagOpts, fmt.Sprintf("-%c", gf.Short))
		}
	}

	// Collect subcommand names.
	var subNames []string
	for _, sub := range cmd.SubCommands {
		subNames = append(subNames, sub.Name)
		subNames = append(subNames, sub.Aliases...)
	}

	allOpts := append(subNames, flagOpts...)

	if len(cmd.SubCommands) > 0 {
		// If at the right depth, suggest subcommands and flags.
		fmt.Fprintf(w, "        if [[ ${COMP_CWORD} -eq %d ]]; then\n", depth)
		fmt.Fprintf(w, "            COMPREPLY=( $(compgen -W %q -- \"${cur}\") )\n", strings.Join(allOpts, " "))
		fmt.Fprintf(w, "            return 0\n")
		fmt.Fprintf(w, "        fi\n")

		// Handle deeper subcommand matching.
		fmt.Fprintf(w, "        case \"${COMP_WORDS[%d]}\" in\n", depth)
		for _, sub := range cmd.SubCommands {
			names := []string{sub.Name}
			names = append(names, sub.Aliases...)
			fmt.Fprintf(w, "        %s)\n", strings.Join(names, "|"))
			a.writeBashSubFlags(w, sub)
			fmt.Fprintf(w, "            ;;\n")
		}
		fmt.Fprintf(w, "        esac\n")
	} else {
		fmt.Fprintf(w, "        COMPREPLY=( $(compgen -W %q -- \"${cur}\") )\n", strings.Join(flagOpts, " "))
		fmt.Fprintf(w, "        return 0\n")
	}
}

func (a *App) writeBashSubFlags(w io.Writer, cmd *Command) {
	var flagOpts []string
	for _, f := range cmd.Flags {
		flagOpts = append(flagOpts, "--"+f.Name)
		if f.Short != 0 {
			flagOpts = append(flagOpts, fmt.Sprintf("-%c", f.Short))
		}
	}
	for _, gf := range a.globalFlags {
		flagOpts = append(flagOpts, "--"+gf.Name)
		if gf.Short != 0 {
			flagOpts = append(flagOpts, fmt.Sprintf("-%c", gf.Short))
		}
	}
	if len(flagOpts) > 0 {
		fmt.Fprintf(w, "            COMPREPLY=( $(compgen -W %q -- \"${cur}\") )\n", strings.Join(flagOpts, " "))
		fmt.Fprintf(w, "            return 0\n")
	}
}

func (a *App) generateZsh(w io.Writer) error {
	fmt.Fprintf(w, "#compdef %s\n\n", a.name)
	fmt.Fprintf(w, "_%s() {\n", a.name)
	fmt.Fprintf(w, "    local -a commands\n")
	fmt.Fprintf(w, "    commands=(\n")
	for _, cmd := range a.commands {
		desc := strings.ReplaceAll(cmd.Description, "'", "'\\''")
		fmt.Fprintf(w, "        '%s:%s'\n", cmd.Name, desc)
		for _, alias := range cmd.Aliases {
			fmt.Fprintf(w, "        '%s:%s'\n", alias, desc)
		}
	}
	fmt.Fprintf(w, "    )\n\n")

	fmt.Fprintf(w, `    if (( CURRENT == 2 )); then
        _describe 'command' commands
        return
    fi

`)

	fmt.Fprintf(w, "    case ${words[2]} in\n")
	for _, cmd := range a.commands {
		names := []string{cmd.Name}
		names = append(names, cmd.Aliases...)
		fmt.Fprintf(w, "    %s)\n", strings.Join(names, "|"))
		a.writeZshCommandCase(w, cmd)
		fmt.Fprintf(w, "        ;;\n")
	}
	fmt.Fprintf(w, "    esac\n")
	fmt.Fprintf(w, "}\n\n")
	fmt.Fprintf(w, "_%s\n", a.name)
	return nil
}

func (a *App) writeZshCommandCase(w io.Writer, cmd *Command) {
	if len(cmd.SubCommands) > 0 {
		fmt.Fprintf(w, "        local -a subcmds\n")
		fmt.Fprintf(w, "        subcmds=(\n")
		for _, sub := range cmd.SubCommands {
			desc := strings.ReplaceAll(sub.Description, "'", "'\\''")
			fmt.Fprintf(w, "            '%s:%s'\n", sub.Name, desc)
		}
		fmt.Fprintf(w, "        )\n")
		fmt.Fprintf(w, "        _describe 'subcommand' subcmds\n")
	}

	flags := a.collectAllFlags(cmd)
	if len(flags) > 0 {
		fmt.Fprintf(w, "        _arguments \\\n")
		for i, f := range flags {
			desc := strings.ReplaceAll(f.Description, "'", "'\\''")
			trail := " \\"
			if i == len(flags)-1 {
				trail = ""
			}
			fmt.Fprintf(w, "            '--%-s[%s]'%s\n", f.Name, desc, trail)
		}
	}
}

func (a *App) generateFish(w io.Writer) error {
	// Disable file completions by default.
	fmt.Fprintf(w, "complete -c %s -f\n\n", a.name)

	// Top-level commands.
	for _, cmd := range a.commands {
		fmt.Fprintf(w, "complete -c %s -n '__fish_use_subcommand' -a %s -d %q\n",
			a.name, cmd.Name, cmd.Description)
		for _, alias := range cmd.Aliases {
			fmt.Fprintf(w, "complete -c %s -n '__fish_use_subcommand' -a %s -d %q\n",
				a.name, alias, cmd.Description)
		}
	}
	fmt.Fprintln(w)

	// Per-command flags and subcommands.
	for _, cmd := range a.commands {
		a.writeFishCommand(w, cmd, cmd.Name)
	}

	return nil
}

func (a *App) writeFishCommand(w io.Writer, cmd *Command, parentCondition string) {
	condition := fmt.Sprintf("__fish_seen_subcommand_from %s", parentCondition)

	// Subcommands.
	for _, sub := range cmd.SubCommands {
		fmt.Fprintf(w, "complete -c %s -n '%s' -a %s -d %q\n",
			a.name, condition, sub.Name, sub.Description)
		a.writeFishCommand(w, sub, sub.Name)
	}

	// Command-specific flags.
	for _, f := range cmd.Flags {
		short := ""
		if f.Short != 0 {
			short = fmt.Sprintf(" -s %c", f.Short)
		}
		fmt.Fprintf(w, "complete -c %s -n '%s' -l %s%s -d %q\n",
			a.name, condition, f.Name, short, f.Description)
	}

	// Global flags.
	for _, f := range a.globalFlags {
		short := ""
		if f.Short != 0 {
			short = fmt.Sprintf(" -s %c", f.Short)
		}
		fmt.Fprintf(w, "complete -c %s -n '%s' -l %s%s -d %q\n",
			a.name, condition, f.Name, short, f.Description)
	}
}

func (a *App) generatePowerShell(w io.Writer) error {
	fmt.Fprintf(w, `Register-ArgumentCompleter -CommandName %s -ScriptBlock {
    param($commandName, $wordToComplete, $cursorPosition)

    $commands = @{
`, a.name)

	for _, cmd := range a.commands {
		flags := a.collectAllFlags(cmd)
		var flagStrs []string
		for _, f := range flags {
			flagStrs = append(flagStrs, fmt.Sprintf("'--%s'", f.Name))
			if f.Short != 0 {
				flagStrs = append(flagStrs, fmt.Sprintf("'-%c'", f.Short))
			}
		}
		var subStrs []string
		for _, sub := range cmd.SubCommands {
			subStrs = append(subStrs, fmt.Sprintf("'%s'", sub.Name))
		}
		allStrs := append(subStrs, flagStrs...)
		fmt.Fprintf(w, "        '%s' = @(%s)\n", cmd.Name, strings.Join(allStrs, ", "))
	}

	fmt.Fprintf(w, `    }

    $cmdList = @(%s)

    $tokens = $wordToComplete -split '\s+'
    if ($tokens.Count -le 1) {
        $cmdList | Where-Object { $_ -like "$wordToComplete*" } | ForEach-Object {
            [System.Management.Automation.CompletionResult]::new($_, $_, 'ParameterValue', $_)
        }
    } else {
        $cmd = $tokens[0]
        if ($commands.ContainsKey($cmd)) {
            $commands[$cmd] | Where-Object { $_ -like "$($tokens[-1])*" } | ForEach-Object {
                [System.Management.Automation.CompletionResult]::new($_, $_, 'ParameterValue', $_)
            }
        }
    }
}
`, a.quotedCommandList())

	return nil
}

func (a *App) quotedCommandList() string {
	var names []string
	for _, cmd := range a.commands {
		names = append(names, fmt.Sprintf("'%s'", cmd.Name))
		for _, alias := range cmd.Aliases {
			names = append(names, fmt.Sprintf("'%s'", alias))
		}
	}
	return strings.Join(names, ", ")
}

// collectAllFlags returns command flags combined with global flags.
func (a *App) collectAllFlags(cmd *Command) []Flag {
	var flags []Flag
	flags = append(flags, cmd.Flags...)
	for _, gf := range a.globalFlags {
		if findFlag(cmd, gf.Name) == nil {
			flags = append(flags, gf)
		}
	}
	return flags
}
