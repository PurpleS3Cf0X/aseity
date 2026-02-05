package tui

import (
	"fmt"
	"strings"

	"github.com/jeanpaul/aseity/internal/agent"
	"github.com/jeanpaul/aseity/internal/agent/skillsets"
)

// formatProfileInfo formats the current model profile for display
func formatProfileInfo(ag *agent.Agent) string {
	if ag == nil {
		return "No active agent"
	}

	profile := ag.GetProfile()

	var b strings.Builder

	b.WriteString("📊 Model Profile\n")
	b.WriteString(strings.Repeat("─", 50) + "\n\n")

	b.WriteString(fmt.Sprintf("Model: %s\n", profile.Name))
	b.WriteString(fmt.Sprintf("Tier: %d (%s)\n", profile.Tier, getTierName(profile.Tier)))
	b.WriteString(fmt.Sprintf("Strategy: %s\n", profile.PromptStrategy))
	b.WriteString(fmt.Sprintf("Validation: %s\n", getValidationName(profile.ValidationLevel)))
	b.WriteString(fmt.Sprintf("Max Tokens: %d\n", profile.MaxTokens))
	b.WriteString(fmt.Sprintf("Native Function Calling: %v\n\n", profile.SupportsNativeFC))

	b.WriteString("Skillset Proficiencies:\n")
	for _, skill := range skillsets.AllSkillsets() {
		if prof, ok := profile.Skillsets[skill]; ok {
			bar := makeBar(prof, 20)
			status := "✓"
			if prof < 0.70 {
				status = "⚠"
			}
			b.WriteString(fmt.Sprintf("  %s %-25s %.2f %s\n", status, skill+":", prof, bar))
		}
	}

	return b.String()
}

// formatSkillsetsInfo formats skillsets information for display
func formatSkillsetsInfo(ag *agent.Agent) string {
	if ag == nil {
		return "No active agent"
	}

	profile := ag.GetProfile()
	config := ag.GetUserConfig()

	var b strings.Builder

	b.WriteString("🎓 Skillsets Configuration\n")
	b.WriteString(strings.Repeat("─", 50) + "\n\n")

	// Dynamic skillsets status
	if config != nil && config.Settings.EnableDynamicSkillsets {
		b.WriteString("✓ Dynamic Skillsets: ENABLED\n")
		b.WriteString("  Context-aware skillset selection active\n\n")
	} else {
		b.WriteString("✗ Dynamic Skillsets: DISABLED\n")
		b.WriteString("  All skillsets loaded on every request\n\n")
	}

	// Weak skillsets (need training)
	weakSkills := profile.GetWeakSkillsets(0.70)
	if len(weakSkills) > 0 {
		b.WriteString("⚠ Weak Skillsets (receiving extra training):\n")
		for _, skill := range weakSkills {
			prof := profile.Skillsets[skill]
			b.WriteString(fmt.Sprintf("  • %s (%.2f)\n", skill, prof))
		}
		b.WriteString("\n")
	}

	// Custom skillsets
	if config != nil {
		customSkills := skillsets.GetEnabledCustomSkillsets(config)
		if len(customSkills) > 0 {
			b.WriteString("✨ Custom Skillsets:\n")
			for _, skill := range customSkills {
				b.WriteString(fmt.Sprintf("  • %s: %s\n", skill.Name, skill.Description))
			}
			b.WriteString("\n")
		}
	}

	b.WriteString("💡 Tip: Edit ~/.aseity/skillsets.yaml to customize\n")

	return b.String()
}

// Helper functions
func getTierName(tier int) string {
	names := map[int]string{
		1: "Advanced",
		2: "Competent",
		3: "Basic",
		4: "Minimal",
	}
	if name, ok := names[tier]; ok {
		return name
	}
	return "Unknown"
}

func getValidationName(level skillsets.ValidationLevel) string {
	names := map[skillsets.ValidationLevel]string{
		skillsets.ValidationNone:   "None",
		skillsets.ValidationLight:  "Light",
		skillsets.ValidationMedium: "Medium",
		skillsets.ValidationStrict: "Strict",
	}
	if name, ok := names[level]; ok {
		return name
	}
	return "Unknown"
}

func makeBar(value float64, width int) string {
	filled := int(value * float64(width))
	if filled > width {
		filled = width
	}
	return strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
}
