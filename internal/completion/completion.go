package completion

import (
	"fmt"
	"strings"
)

// SupportedShells lists the shells supported for completion script generation.
var SupportedShells = []string{"bash", "zsh", "fish"}

// Generate returns a shell completion script for bash, zsh, or fish.
func Generate(shell string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(shell)) {
	case "bash":
		return Bash(), nil
	case "zsh":
		return Zsh(), nil
	case "fish":
		return Fish(), nil
	default:
		return "", fmt.Errorf("unsupported shell %q, expected 'bash', 'zsh', or 'fish'", shell)
	}
}

// Bash returns a bash completion script for gowiz.
func Bash() string {
	return `# bash completion for gowiz                              -*- shell-script -*-

_gowiz_completions() {
    local cur prev opts verbs presets categories shells
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"

    opts="--ip --cmd --config --json --mock --version -v --help --check --daemon --once --interval --server -s --port --api-key --webhook"
    verbs="on off toggle power dim bright warm cool daylight hex color temp rgb speed scene ocean sunset party cozy forest fireplace romance relax focus nightlight preset room group fade sunrise circadian rhythm daemon schedule service systemd launchd serve server flash pulse strobe rainbow effect cat category categories info diag status config recent export import undo help scan discover quit completion"
    presets="evening movie night focus work relax party list save delete"
    groups="list set create delete rm"
    effects="flash pulse strobe rainbow"
    categories="nature cozy white party mood dynamic"
    shells="bash zsh fish"
    services="install uninstall status systemd launchd"

    if [[ ${cur} == -* ]] ; then
        COMPREPLY=( $(compgen -W "${opts}" -- ${cur}) )
        return 0
    fi

    if [[ ${prev} == "service" ]] ; then
        COMPREPLY=( $(compgen -W "${services}" -- ${cur}) )
        return 0
    fi

    if [[ ${prev} == "preset" ]] ; then
        COMPREPLY=( $(compgen -W "${presets}" -- ${cur}) )
        return 0
    fi

    if [[ ${prev} == "group" ]] ; then
        COMPREPLY=( $(compgen -W "${groups}" -- ${cur}) )
        return 0
    fi

    if [[ ${prev} == "completion" ]] ; then
        COMPREPLY=( $(compgen -W "${shells}" -- ${cur}) )
        return 0
    fi

    if [[ ${prev} == "cat" || ${prev} == "category" ]] ; then
        COMPREPLY=( $(compgen -W "${categories}" -- ${cur}) )
        return 0
    fi

    COMPREPLY=( $(compgen -W "${verbs}" -- ${cur}) )
    return 0
}

complete -F _gowiz_completions gowiz
`
}

// Zsh returns a zsh completion script for gowiz.
func Zsh() string {
	return `#compdef gowiz

_gowiz() {
    local -a commands
    commands=(
        'on:Turn smart light on'
        'off:Turn smart light off'
        'toggle:Toggle power state'
        'warm:Set warm white color temperature (2700K)'
        'cool:Set cool white color temperature (4200K)'
        'daylight:Set daylight white color temperature (6500K)'
        'dim:Set brightness level (10-100%)'
        'scene:Set dynamic scene'
        'preset:Manage or apply lighting presets'
        'room:Assign device to room'
        'group:Manage device groups or execute batch command'
        'fade:Start linear dimming fade transition'
        'sunrise:Start sunrise lighting simulation'
        'sunset:Start sunset lighting simulation'
        'circadian:Apply 24-hour circadian lighting rhythm'
        'daemon:Run background circadian schedule sync daemon'
        'service:Manage systemd or launchd background daemon service'
        'serve:Start HTTP REST API server'
        'flash:Flash smart lights for notification'
        'pulse:Pulse smart light dimming level'
        'strobe:Strobe light alert effect'
        'rainbow:Sweep RGB spectrum hues'
        'effect:Trigger dynamic lighting effect'
        'cat:Filter scenes by category'
        'info:Display device telemetry diagnostics'
        'config:Display configuration status'
        'export:Export configuration JSON'
        'import:Import configuration JSON'
        'scan:Rescan local network for WiZ devices'
        'discover:Discover WiZ smart lights on local network broadcast'
        'completion:Generate shell autocompletion script'
    )

    local -a flags
    flags=(
        '--ip[Target IP address]:IP:_hosts'
        '--cmd[Command string to execute]:command:'
        '--config[Path to configuration JSON]:file:_files'
        '--json[Output results in JSON format]'
        '--mock[Run in mock hardware mode]'
        '--daemon[Run in background circadian sync daemon mode]'
        '--once[Run a single circadian sync pass]'
        '--interval[Sync interval duration in daemon mode]:duration:'
        '--server[Run in HTTP REST API server mode]'
        '-s[Run in HTTP REST API server mode]'
        '--port[Port for HTTP REST API server]:port:'
        '--api-key[API key for securing HTTP REST API server]:key:'
        '--version[Print version information]'
        '-v[Print version information]'
        '--help[Show help message]'
    )

    _arguments -s \
        $flags \
        '1: :->command' \
        '*: :->args'

    case $state in
        command)
            _describe -t commands 'gowiz command' commands
            ;;
        args)
            case $words[2] in
                completion)
                    _values 'shell' bash zsh fish
                    ;;
                preset)
                    _values 'preset action' list save delete evening movie night focus work relax party
                    ;;
                group)
                    _values 'group action' list set create delete rm
                    ;;
                service)
                    _values 'service action' install uninstall status systemd launchd
                    ;;
                cat|category)
                    _values 'category' nature cozy white party mood dynamic
                    ;;
            esac
            ;;
    esac
}

_gowiz "$@"
`
}

// Fish returns a fish completion script for gowiz.
func Fish() string {
	return `# fish completion for gowiz

complete -c gowiz -s v -l version -d "Print version information"
complete -c gowiz -l help -d "Show help message"
complete -c gowiz -l ip -d "Target IP address" -r
complete -c gowiz -l cmd -d "Command string to execute" -r
complete -c gowiz -l config -d "Path to configuration JSON" -r
complete -c gowiz -l json -d "Output results in JSON format"
complete -c gowiz -l mock -d "Run in mock hardware mode"
complete -c gowiz -l daemon -d "Run in background circadian sync daemon mode"
complete -c gowiz -l once -d "Run a single circadian sync pass"
complete -c gowiz -l interval -d "Sync interval duration" -r
complete -c gowiz -l server -s s -d "Run in HTTP REST API server mode"
complete -c gowiz -l port -d "Port for HTTP REST API server" -r
complete -c gowiz -l api-key -d "API key for securing HTTP REST API server" -r

# Commands
complete -c gowiz -n "__fish_use_subcommand" -a on -d "Turn smart light on"
complete -c gowiz -n "__fish_use_subcommand" -a off -d "Turn smart light off"
complete -c gowiz -n "__fish_use_subcommand" -a toggle -d "Toggle power state"
complete -c gowiz -n "__fish_use_subcommand" -a warm -d "Set warm white color temperature"
complete -c gowiz -n "__fish_use_subcommand" -a cool -d "Set cool white color temperature"
complete -c gowiz -n "__fish_use_subcommand" -a daylight -d "Set daylight white color temperature"
complete -c gowiz -n "__fish_use_subcommand" -a dim -d "Set brightness level"
complete -c gowiz -n "__fish_use_subcommand" -a preset -d "Manage or apply lighting presets"
complete -c gowiz -n "__fish_use_subcommand" -a room -d "Assign device to room"
complete -c gowiz -n "__fish_use_subcommand" -a group -d "Manage device groups or execute batch command"
complete -c gowiz -n "__fish_use_subcommand" -a fade -d "Start linear dimming fade transition"
complete -c gowiz -n "__fish_use_subcommand" -a circadian -d "Apply 24-hour circadian lighting rhythm"
complete -c gowiz -n "__fish_use_subcommand" -a daemon -d "Run background circadian schedule sync daemon"
complete -c gowiz -n "__fish_use_subcommand" -a service -d "Manage systemd or launchd background daemon service"
complete -c gowiz -n "__fish_use_subcommand" -a serve -d "Start HTTP REST API server"
complete -c gowiz -n "__fish_use_subcommand" -a cat -d "Filter scenes by category"
complete -c gowiz -n "__fish_use_subcommand" -a info -d "Display device telemetry diagnostics"
complete -c gowiz -n "__fish_use_subcommand" -a config -d "Display current configuration"
complete -c gowiz -n "__fish_use_subcommand" -a export -d "Export configuration JSON"
complete -c gowiz -n "__fish_use_subcommand" -a import -d "Import configuration JSON"
complete -c gowiz -n "__fish_use_subcommand" -a completion -d "Generate shell autocompletion script"

# Subcommands
complete -c gowiz -n "__fish_seen_subcommand_from completion" -a "bash zsh fish"
complete -c gowiz -n "__fish_seen_subcommand_from preset" -a "list save delete evening movie night focus work relax party"
complete -c gowiz -n "__fish_seen_subcommand_from group" -a "list set create delete rm"
complete -c gowiz -n "__fish_seen_subcommand_from service" -a "install uninstall status systemd launchd"
`
}
