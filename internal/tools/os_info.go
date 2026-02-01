package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// --- system_info ---

type SystemInfoTool struct{}

func (s *SystemInfoTool) Name() string        { return "system_info" }
func (s *SystemInfoTool) NeedsConfirmation() bool { return false }
func (s *SystemInfoTool) Description() string {
	return "Get system information: OS, architecture, hostname, CPU count, memory, and Go runtime version."
}

func (s *SystemInfoTool) Parameters() any {
	return map[string]any{
		"type":       "object",
		"properties": map[string]any{},
	}
}

func (s *SystemInfoTool) Execute(_ context.Context, _ string) (Result, error) {
	hostname, _ := os.Hostname()
	cwd, _ := os.Getwd()

	var sb strings.Builder
	fmt.Fprintf(&sb, "OS:        %s\n", runtime.GOOS)
	fmt.Fprintf(&sb, "Arch:      %s\n", runtime.GOARCH)
	fmt.Fprintf(&sb, "Hostname:  %s\n", hostname)
	fmt.Fprintf(&sb, "CPUs:      %d\n", runtime.NumCPU())
	fmt.Fprintf(&sb, "GoVersion: %s\n", runtime.Version())
	fmt.Fprintf(&sb, "CWD:       %s\n", cwd)
	fmt.Fprintf(&sb, "User:      %s\n", os.Getenv("USER"))
	fmt.Fprintf(&sb, "Shell:     %s\n", os.Getenv("SHELL"))
	fmt.Fprintf(&sb, "Home:      %s\n", os.Getenv("HOME"))

	// Try to get memory info (platform-specific)
	if runtime.GOOS == "darwin" {
		if out, err := exec.Command("sysctl", "-n", "hw.memsize").Output(); err == nil {
			fmt.Fprintf(&sb, "Memory:    %s bytes\n", strings.TrimSpace(string(out)))
		}
	} else if runtime.GOOS == "linux" {
		if out, err := exec.Command("free", "-h").Output(); err == nil {
			lines := strings.Split(string(out), "\n")
			if len(lines) > 1 {
				fmt.Fprintf(&sb, "Memory:    %s\n", strings.TrimSpace(lines[1]))
			}
		}
	}

	// Disk usage for cwd
	if out, err := exec.Command("df", "-h", cwd).Output(); err == nil {
		lines := strings.Split(string(out), "\n")
		if len(lines) > 1 {
			fmt.Fprintf(&sb, "Disk:      %s\n", strings.TrimSpace(lines[1]))
		}
	}

	return Result{Output: sb.String()}, nil
}

// --- process_list ---

type ProcessListTool struct{}

type processListArgs struct {
	Filter string `json:"filter,omitempty"`
}

func (p *ProcessListTool) Name() string        { return "process_list" }
func (p *ProcessListTool) NeedsConfirmation() bool { return false }
func (p *ProcessListTool) Description() string {
	return "List running processes, optionally filtered by name."
}

func (p *ProcessListTool) Parameters() any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"filter": map[string]any{"type": "string", "description": "Filter processes by name (case-insensitive substring match)"},
		},
	}
}

func (p *ProcessListTool) Execute(ctx context.Context, rawArgs string) (Result, error) {
	var args processListArgs
	if rawArgs != "" && rawArgs != "{}" {
		json.Unmarshal([]byte(rawArgs), &args)
	}

	var cmd *exec.Cmd
	if args.Filter != "" {
		// ps aux | grep -i filter (but without the grep process itself)
		cmd = exec.CommandContext(ctx, "bash", "-c",
			fmt.Sprintf("ps aux | head -1; ps aux | grep -i '%s' | grep -v grep", args.Filter))
	} else {
		cmd = exec.CommandContext(ctx, "ps", "aux", "--sort=-%mem")
		if runtime.GOOS == "darwin" {
			cmd = exec.CommandContext(ctx, "ps", "aux", "-r")
		}
	}

	out, err := cmd.Output()
	if err != nil {
		// grep returns exit 1 when no matches
		if len(out) == 0 {
			return Result{Output: "No matching processes found."}, nil
		}
	}

	output := string(out)
	lines := strings.Split(output, "\n")
	if len(lines) > 50 {
		output = strings.Join(lines[:50], "\n") + fmt.Sprintf("\n... (%d more processes)", len(lines)-50)
	}

	return Result{Output: output}, nil
}

// --- network_info ---

type NetworkInfoTool struct{}

func (n *NetworkInfoTool) Name() string        { return "network_info" }
func (n *NetworkInfoTool) NeedsConfirmation() bool { return false }
func (n *NetworkInfoTool) Description() string {
	return "Get network information: interfaces, IP addresses, and listening ports."
}

func (n *NetworkInfoTool) Parameters() any {
	return map[string]any{
		"type":       "object",
		"properties": map[string]any{},
	}
}

func (n *NetworkInfoTool) Execute(ctx context.Context, _ string) (Result, error) {
	var sb strings.Builder

	// Network interfaces
	ifaces, err := net.Interfaces()
	if err == nil {
		sb.WriteString("=== Network Interfaces ===\n")
		for _, iface := range ifaces {
			if iface.Flags&net.FlagUp == 0 {
				continue
			}
			addrs, _ := iface.Addrs()
			addrStrs := make([]string, 0, len(addrs))
			for _, addr := range addrs {
				addrStrs = append(addrStrs, addr.String())
			}
			if len(addrStrs) > 0 {
				fmt.Fprintf(&sb, "  %s: %s\n", iface.Name, strings.Join(addrStrs, ", "))
			}
		}
	}

	// Listening ports
	sb.WriteString("\n=== Listening Ports ===\n")
	var cmd *exec.Cmd
	if runtime.GOOS == "darwin" {
		cmd = exec.CommandContext(ctx, "lsof", "-iTCP", "-sTCP:LISTEN", "-P", "-n")
	} else {
		cmd = exec.CommandContext(ctx, "ss", "-tlnp")
	}
	if out, err := cmd.Output(); err == nil {
		lines := strings.Split(string(out), "\n")
		if len(lines) > 30 {
			lines = lines[:30]
			lines = append(lines, "... (truncated)")
		}
		sb.WriteString(strings.Join(lines, "\n"))
	} else {
		sb.WriteString("  (could not list ports)\n")
	}

	return Result{Output: sb.String()}, nil
}

// --- clipboard ---

type ClipboardTool struct{}

type clipboardArgs struct {
	Action  string `json:"action"`
	Content string `json:"content,omitempty"`
}

func (c *ClipboardTool) Name() string        { return "clipboard" }
func (c *ClipboardTool) NeedsConfirmation() bool { return true }
func (c *ClipboardTool) Description() string {
	return "Read or write the system clipboard. Action must be 'read' or 'write'."
}

func (c *ClipboardTool) Parameters() any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"action":  map[string]any{"type": "string", "enum": []string{"read", "write"}, "description": "read or write"},
			"content": map[string]any{"type": "string", "description": "Content to write (only for write action)"},
		},
		"required": []string{"action"},
	}
}

func (c *ClipboardTool) Execute(ctx context.Context, rawArgs string) (Result, error) {
	var args clipboardArgs
	if err := json.Unmarshal([]byte(rawArgs), &args); err != nil {
		return Result{Error: "invalid arguments: " + err.Error()}, nil
	}

	switch args.Action {
	case "read":
		var cmd *exec.Cmd
		if runtime.GOOS == "darwin" {
			cmd = exec.CommandContext(ctx, "pbpaste")
		} else {
			cmd = exec.CommandContext(ctx, "xclip", "-selection", "clipboard", "-o")
		}
		out, err := cmd.Output()
		if err != nil {
			return Result{Error: "failed to read clipboard: " + err.Error()}, nil
		}
		return Result{Output: string(out)}, nil

	case "write":
		var cmd *exec.Cmd
		if runtime.GOOS == "darwin" {
			cmd = exec.CommandContext(ctx, "pbcopy")
		} else {
			cmd = exec.CommandContext(ctx, "xclip", "-selection", "clipboard")
		}
		cmd.Stdin = strings.NewReader(args.Content)
		if err := cmd.Run(); err != nil {
			return Result{Error: "failed to write clipboard: " + err.Error()}, nil
		}
		return Result{Output: "Content copied to clipboard."}, nil

	default:
		return Result{Error: "action must be 'read' or 'write'"}, nil
	}
}
