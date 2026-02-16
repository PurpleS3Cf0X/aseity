package memory

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ProjectContext holds the parsed information from ASEITY.md or CLAUDE.md
type ProjectContext struct {
	Title        string
	Commands     []string
	Style        []string
	Architecture []string
	Notes        []string
	RawContent   string
}

// LoadProjectContext looks for ASEITY.md or CLAUDE.md in the root
// and parses it into a structured context.
func LoadProjectContext(root string) (*ProjectContext, error) {
	// Try ASEITY.md first, then CLAUDE.md
	files := []string{"ASEITY.md", "CLAUDE.md"}
	var content []byte
	var err error
	var foundPath string

	for _, f := range files {
		path := filepath.Join(root, f)
		content, err = os.ReadFile(path)
		if err == nil {
			foundPath = path
			break
		}
	}

	if foundPath == "" {
		return nil, fmt.Errorf("no project context file found (checked ASEITY.md, CLAUDE.md)")
	}

	ctx := parseMarkdown(string(content))
	return ctx, nil
}

// parseMarkdown is a simple parser for the specific CLAUDE.md format
// It looks for H2 headers like "## Commands", "## Code Style", etc.
func parseMarkdown(raw string) *ProjectContext {
	ctx := &ProjectContext{
		RawContent: raw,
	}

	scanner := bufio.NewScanner(strings.NewReader(raw))
	var currentSection string

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "# ") {
			ctx.Title = strings.TrimPrefix(line, "# ")
			continue
		}

		if strings.HasPrefix(line, "## ") {
			header := strings.ToLower(strings.TrimPrefix(line, "## "))
			if strings.Contains(header, "command") {
				currentSection = "commands"
			} else if strings.Contains(header, "style") || strings.Contains(header, "convention") {
				currentSection = "style"
			} else if strings.Contains(header, "architecture") || strings.Contains(header, "structure") {
				currentSection = "architecture"
			} else {
				currentSection = "notes"
			}
			continue
		}

		// content lines
		switch currentSection {
		case "commands":
			ctx.Commands = append(ctx.Commands, line)
		case "style":
			ctx.Style = append(ctx.Style, line)
		case "architecture":
			ctx.Architecture = append(ctx.Architecture, line)
		case "notes":
			ctx.Notes = append(ctx.Notes, line)
		}
	}

	return ctx
}

// String returns a formatted prompt string for the LLM
func (p *ProjectContext) ToPrompt() string {
	var b strings.Builder

	b.WriteString("## 📂 Project Context\n\n")

	if p.Title != "" {
		b.WriteString(fmt.Sprintf("**Project**: %s\n\n", p.Title))
	}

	if len(p.Style) > 0 {
		b.WriteString("### 🎨 Code Style & Conventions\n")
		for _, s := range p.Style {
			b.WriteString(s + "\n")
		}
		b.WriteString("\n")
	}

	if len(p.Commands) > 0 {
		b.WriteString("### 🛠 Common Commands\n")
		for _, s := range p.Commands {
			b.WriteString(s + "\n")
		}
		b.WriteString("\n")
	}

	if len(p.Architecture) > 0 {
		b.WriteString("### 🏗 Architecture\n")
		for _, s := range p.Architecture {
			b.WriteString(s + "\n")
		}
		b.WriteString("\n")
	}

	return b.String()
}

// LoadTodoList looks for TODO.md or tasks.md in the root
// and returns its content formatted for the prompt.
func LoadTodoList(root string) (string, error) {
	files := []string{"TODO.md", "tasks.md", "todo.md", "TASKS.md"}
	var content []byte
	var err error

	for _, f := range files {
		path := filepath.Join(root, f)
		content, err = os.ReadFile(path)
		if err == nil {
			return fmt.Sprintf("## 📝 Project Context: Open Tasks (Reference Only)\n\nThe following are open tasks in %s. Use them for context, but PRIORITIZE the User's latest request below.\n\n%s", f, string(content)), nil
		}
	}

	return "", fmt.Errorf("no todo list found")
}
