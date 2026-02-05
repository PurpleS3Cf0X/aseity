package skillsets

import (
	"testing"

	"github.com/jeanpaul/aseity/internal/provider"
)

func TestValidateJSON(t *testing.T) {
	tests := []struct {
		name    string
		args    string
		wantErr bool
	}{
		{"valid JSON", `{"command": "ls"}`, false},
		{"invalid JSON", `{command: ls}`, true},
		{"empty JSON", `{}`, false},
		{"malformed JSON", `{"command":`, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateJSON(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateJSON() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateParameters(t *testing.T) {
	tests := []struct {
		name    string
		call    provider.ToolCall
		wantErr bool
	}{
		{
			name:    "bash with command",
			call:    provider.ToolCall{Name: "bash", Args: `{"command": "ls -la"}`},
			wantErr: false,
		},
		{
			name:    "bash without command",
			call:    provider.ToolCall{Name: "bash", Args: `{}`},
			wantErr: true,
		},
		{
			name:    "bash with empty command",
			call:    provider.ToolCall{Name: "bash", Args: `{"command": ""}`},
			wantErr: true,
		},
		{
			name:    "file_read with path",
			call:    provider.ToolCall{Name: "file_read", Args: `{"path": "/etc/hosts"}`},
			wantErr: false,
		},
		{
			name:    "file_read without path",
			call:    provider.ToolCall{Name: "file_read", Args: `{}`},
			wantErr: true,
		},
		{
			name:    "web_search with query",
			call:    provider.ToolCall{Name: "web_search", Args: `{"query": "golang tutorial"}`},
			wantErr: false,
		},
		{
			name:    "web_search without query",
			call:    provider.ToolCall{Name: "web_search", Args: `{}`},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateParameters(tt.call)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateParameters() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateContext(t *testing.T) {
	tests := []struct {
		name    string
		call    provider.ToolCall
		ctx     ValidationContext
		wantErr bool
	}{
		{
			name:    "apt-get on macOS",
			call:    provider.ToolCall{Name: "bash", Args: `{"command": "apt-get install redis"}`},
			ctx:     ValidationContext{OS: "darwin"},
			wantErr: true,
		},
		{
			name:    "brew on macOS",
			call:    provider.ToolCall{Name: "bash", Args: `{"command": "brew install redis"}`},
			ctx:     ValidationContext{OS: "darwin"},
			wantErr: false,
		},
		{
			name:    "brew on Windows",
			call:    provider.ToolCall{Name: "bash", Args: `{"command": "brew install redis"}`},
			ctx:     ValidationContext{OS: "windows"},
			wantErr: true,
		},
		{
			name:    "dangerous command",
			call:    provider.ToolCall{Name: "bash", Args: `{"command": "rm -rf /"}`},
			ctx:     ValidationContext{OS: "linux"},
			wantErr: true,
		},
		{
			name:    "safe command",
			call:    provider.ToolCall{Name: "bash", Args: `{"command": "ls -la"}`},
			ctx:     ValidationContext{OS: "linux"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateContext(tt.call, tt.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateContext() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateToolCall(t *testing.T) {
	ctx := ValidationContext{OS: "darwin"}

	tests := []struct {
		name    string
		call    provider.ToolCall
		level   ValidationLevel
		wantErr bool
	}{
		{
			name:    "None level - invalid JSON passes",
			call:    provider.ToolCall{Name: "bash", Args: `{invalid}`},
			level:   ValidationNone,
			wantErr: false,
		},
		{
			name:    "Light level - invalid JSON fails",
			call:    provider.ToolCall{Name: "bash", Args: `{invalid}`},
			level:   ValidationLight,
			wantErr: true,
		},
		{
			name:    "Medium level - missing params fails",
			call:    provider.ToolCall{Name: "bash", Args: `{}`},
			level:   ValidationMedium,
			wantErr: true,
		},
		{
			name:    "Strict level - wrong OS command fails",
			call:    provider.ToolCall{Name: "bash", Args: `{"command": "apt-get install redis"}`},
			level:   ValidationStrict,
			wantErr: true,
		},
		{
			name:    "Strict level - correct command passes",
			call:    provider.ToolCall{Name: "bash", Args: `{"command": "brew install redis"}`},
			level:   ValidationStrict,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateToolCall(tt.call, tt.level, ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateToolCall() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSuggestCorrection(t *testing.T) {
	ctx := ValidationContext{OS: "darwin"}

	tests := []struct {
		name     string
		call     provider.ToolCall
		errorMsg string
		want     string
	}{
		{
			name:     "apt-get on macOS",
			call:     provider.ToolCall{Name: "bash"},
			errorMsg: "apt-get is not available",
			want:     "Use 'brew' instead of 'apt-get' on macOS",
		},
		{
			name:     "command not found",
			call:     provider.ToolCall{Name: "bash"},
			errorMsg: "redis-server: command not found",
			want:     "The command may not be installed. Try installing it first.",
		},
		{
			name:     "permission denied",
			call:     provider.ToolCall{Name: "bash"},
			errorMsg: "permission denied",
			want:     "You may need elevated permissions. Consider using sudo (with caution).",
		},
		{
			name:     "file not found",
			call:     provider.ToolCall{Name: "file_read"},
			errorMsg: "no such file or directory",
			want:     "The file or directory doesn't exist. Check the path and try again.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SuggestCorrection(tt.call, tt.errorMsg, ctx)
			if got != tt.want {
				t.Errorf("SuggestCorrection() = %v, want %v", got, tt.want)
			}
		})
	}
}
