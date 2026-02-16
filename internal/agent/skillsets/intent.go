package skillsets

import (
	"strings"
)

// Intent represents the user's intention
type Intent int

const (
	IntentGeneral Intent = iota
	IntentInstall
	IntentCodeReview
	IntentDeploy
	IntentDebug
	IntentSearch
	IntentFileOps
	IntentTest
	IntentOptimize
	IntentSecurity
)

// IntentKeywords maps intents to trigger keywords
var IntentKeywords = map[Intent][]string{
	IntentInstall: {
		"install", "setup", "download", "add package", "npm install",
		"pip install", "brew install", "apt install", "yarn add",
	},
	IntentCodeReview: {
		"review", "check code", "bugs", "issues", "problems",
		"code quality", "refactor", "improve code",
	},
	IntentDeploy: {
		"deploy", "release", "publish", "ship", "production",
		"staging", "build and deploy", "push to prod",
	},
	IntentDebug: {
		"error", "fix", "broken", "not working", "debug",
		"troubleshoot", "diagnose", "why doesn't", "failing",
	},
	IntentSearch: {
		"search", "find", "look up", "google", "documentation",
		"how to", "what is", "explain", "docs for",
	},
	IntentFileOps: {
		"read", "write", "edit", "create file", "delete file",
		"move", "copy", "rename", "show file", "list files",
	},
	IntentTest: {
		"test", "run tests", "unit test", "integration test",
		"check if works", "verify", "validate",
	},
	IntentOptimize: {
		"optimize", "performance", "slow", "faster", "speed up",
		"improve performance", "bottleneck", "profile",
	},
	IntentSecurity: {
		"security", "vulnerability", "exploit", "injection",
		"xss", "csrf", "auth", "permissions", "secure",
	},
}

// IntentSkillsets maps intents to relevant skillsets
var IntentSkillsets = map[Intent][]string{
	IntentGeneral: {
		SkillToolSelection,
		SkillParameterConstruct,
	},
	IntentInstall: {
		SkillToolSelection,
		SkillCommandConstruct,
		SkillErrorDiagnosis,
	},
	IntentCodeReview: {
		SkillContentParsing,
		SkillContextAwareness,
		// Custom skillsets would be added here
	},
	IntentDeploy: {
		SkillSequentialPlanning,
		SkillErrorDiagnosis,
		SkillSelfCorrection,
		SkillDependencyResolve,
	},
	IntentDebug: {
		SkillErrorDiagnosis,
		SkillSelfCorrection,
		SkillOutputInterpret,
	},
	IntentSearch: {
		SkillSearchQuery,
		SkillContentParsing,
	},
	IntentFileOps: {
		SkillToolSelection,
		SkillParameterConstruct,
		SkillContextAwareness,
	},
	IntentTest: {
		SkillSequentialPlanning,
		SkillErrorDiagnosis,
		SkillOutputInterpret,
	},
	IntentOptimize: {
		SkillContentParsing,
		SkillContextAwareness,
		SkillOutputInterpret,
	},
	IntentSecurity: {
		SkillContentParsing,
		SkillContextAwareness,
		SkillErrorDiagnosis,
	},
}

// DetectIntent analyzes user message and returns the most likely intent
func DetectIntent(userMsg string) Intent {
	lowerMsg := strings.ToLower(userMsg)

	// Score each intent based on keyword matches
	scores := make(map[Intent]int)

	for intent, keywords := range IntentKeywords {
		for _, keyword := range keywords {
			if strings.Contains(lowerMsg, keyword) {
				scores[intent]++
			}
		}
	}

	// Find highest scoring intent
	maxScore := 0
	detectedIntent := IntentGeneral

	for intent, score := range scores {
		if score > maxScore {
			maxScore = score
			detectedIntent = intent
		}
	}

	return detectedIntent
}

// GetSkillsetsForIntent returns relevant skillsets for an intent
func GetSkillsetsForIntent(intent Intent) []string {
	if skillsets, ok := IntentSkillsets[intent]; ok {
		return skillsets
	}
	return IntentSkillsets[IntentGeneral]
}

// BuildContextualPrompt builds a prompt with only relevant skillsets
func BuildContextualPrompt(intent Intent, profile ModelProfile) string {
	var b strings.Builder

	// Get relevant skillsets for this intent
	relevantSkills := GetSkillsetsForIntent(intent)

	var content strings.Builder
	for _, skill := range relevantSkills {
		// Check if model is weak in this skill
		if proficiency, ok := profile.Skillsets[skill]; ok {
			if proficiency < 0.85 {
				content.WriteString(GetSkillTraining(skill))
				content.WriteString("\n")
			}
		}
	}

	if content.Len() > 0 {
		b.WriteString("\n## 🎯 Context-Specific Guidance\n\n")
		b.WriteString(content.String())
	}

	return b.String()
}

// IntentName returns human-readable intent name
func IntentName(intent Intent) string {
	names := map[Intent]string{
		IntentGeneral:    "General",
		IntentInstall:    "Installation",
		IntentCodeReview: "Code Review",
		IntentDeploy:     "Deployment",
		IntentDebug:      "Debugging",
		IntentSearch:     "Search",
		IntentFileOps:    "File Operations",
		IntentTest:       "Testing",
		IntentOptimize:   "Optimization",
		IntentSecurity:   "Security",
	}
	if name, ok := names[intent]; ok {
		return name
	}
	return "Unknown"
}
