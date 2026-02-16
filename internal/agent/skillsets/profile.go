package skillsets

// ModelProfile defines capabilities and configuration for a specific model
type ModelProfile struct {
	Name             string             `yaml:"name"`
	Tier             int                `yaml:"tier"`      // 1=Advanced, 2=Competent, 3=Basic, 4=Minimal
	Skillsets        map[string]float64 `yaml:"skillsets"` // skillset -> proficiency (0-1)
	PromptStrategy   string             `yaml:"prompt_strategy"`
	ValidationLevel  ValidationLevel    `yaml:"validation_level"`
	MaxTokens        int                `yaml:"max_tokens"`
	SupportsNativeFC bool               `yaml:"supports_native_fc"` // Native function calling
}

// ValidationLevel determines how strictly to validate tool calls
type ValidationLevel int

const (
	ValidationNone ValidationLevel = iota
	ValidationLight
	ValidationMedium
	ValidationStrict
)

// Skillset names
const (
	SkillToolSelection      = "tool_selection"
	SkillParameterConstruct = "parameter_construction"
	SkillCommandConstruct   = "command_construction"
	SkillSearchQuery        = "search_query_formulation"
	SkillContentParsing     = "content_parsing"
	SkillErrorDiagnosis     = "error_diagnosis"
	SkillSequentialPlanning = "sequential_planning"
	SkillStateTracking      = "state_tracking"
	SkillDependencyResolve  = "dependency_resolution"
	SkillOutputInterpret    = "output_interpretation"
	SkillContextAwareness   = "context_awareness"
	SkillSelfCorrection     = "self_correction"
)

// AllSkillsets returns all defined skillset names
func AllSkillsets() []string {
	return []string{
		SkillToolSelection,
		SkillParameterConstruct,
		SkillCommandConstruct,
		SkillSearchQuery,
		SkillContentParsing,
		SkillErrorDiagnosis,
		SkillSequentialPlanning,
		SkillStateTracking,
		SkillDependencyResolve,
		SkillOutputInterpret,
		SkillContextAwareness,
		SkillSelfCorrection,
	}
}

// DefaultProfiles returns predefined model profiles
func DefaultProfiles() map[string]ModelProfile {
	return map[string]ModelProfile{
		// Tier 1: Advanced Models
		"gpt-4": {
			Name:             "gpt-4",
			Tier:             1,
			PromptStrategy:   "minimal",
			ValidationLevel:  ValidationLight,
			MaxTokens:        128000,
			SupportsNativeFC: true,
			Skillsets: map[string]float64{
				SkillToolSelection:      0.98,
				SkillParameterConstruct: 0.99,
				SkillCommandConstruct:   0.95,
				SkillSearchQuery:        0.97,
				SkillContentParsing:     0.96,
				SkillErrorDiagnosis:     0.94,
				SkillSequentialPlanning: 0.95,
				SkillStateTracking:      0.93,
				SkillDependencyResolve:  0.92,
				SkillOutputInterpret:    0.96,
				SkillContextAwareness:   0.94,
				SkillSelfCorrection:     0.90,
			},
		},
		"gpt-4o": {
			Name:             "gpt-4o",
			Tier:             1,
			PromptStrategy:   "minimal",
			ValidationLevel:  ValidationLight,
			MaxTokens:        128000,
			SupportsNativeFC: true,
			Skillsets: map[string]float64{
				SkillToolSelection:      0.99,
				SkillParameterConstruct: 0.99,
				SkillCommandConstruct:   0.96,
				SkillSearchQuery:        0.98,
				SkillContentParsing:     0.97,
				SkillErrorDiagnosis:     0.95,
				SkillSequentialPlanning: 0.96,
				SkillStateTracking:      0.94,
				SkillDependencyResolve:  0.93,
				SkillOutputInterpret:    0.97,
				SkillContextAwareness:   0.95,
				SkillSelfCorrection:     0.92,
			},
		},
		"claude-3.5-sonnet": {
			Name:             "claude-3.5-sonnet",
			Tier:             1,
			PromptStrategy:   "minimal",
			ValidationLevel:  ValidationLight,
			MaxTokens:        200000,
			SupportsNativeFC: true,
			Skillsets: map[string]float64{
				SkillToolSelection:      0.98,
				SkillParameterConstruct: 0.99,
				SkillCommandConstruct:   0.96,
				SkillSearchQuery:        0.97,
				SkillContentParsing:     0.98,
				SkillErrorDiagnosis:     0.96,
				SkillSequentialPlanning: 0.97,
				SkillStateTracking:      0.95,
				SkillDependencyResolve:  0.94,
				SkillOutputInterpret:    0.98,
				SkillContextAwareness:   0.96,
				SkillSelfCorrection:     0.93,
			},
		},

		// Tier 2: Competent Models
		"qwen2.5:14b": {
			Name:             "qwen2.5:14b",
			Tier:             2,
			PromptStrategy:   "react",
			ValidationLevel:  ValidationMedium,
			MaxTokens:        32768,
			SupportsNativeFC: false,
			Skillsets: map[string]float64{
				SkillToolSelection:      0.90,
				SkillParameterConstruct: 0.92,
				SkillCommandConstruct:   0.85,
				SkillSearchQuery:        0.88,
				SkillContentParsing:     0.87,
				SkillErrorDiagnosis:     0.75,
				SkillSequentialPlanning: 0.80,
				SkillStateTracking:      0.70,
				SkillDependencyResolve:  0.72,
				SkillOutputInterpret:    0.85,
				SkillContextAwareness:   0.82,
				SkillSelfCorrection:     0.68,
			},
		},
		"deepseek-r1:14b": {
			Name:             "deepseek-r1:14b",
			Tier:             2,
			PromptStrategy:   "react",
			ValidationLevel:  ValidationMedium,
			MaxTokens:        32768,
			SupportsNativeFC: false,
			Skillsets: map[string]float64{
				SkillToolSelection:      0.88,
				SkillParameterConstruct: 0.90,
				SkillCommandConstruct:   0.83,
				SkillSearchQuery:        0.86,
				SkillContentParsing:     0.85,
				SkillErrorDiagnosis:     0.78,
				SkillSequentialPlanning: 0.82,
				SkillStateTracking:      0.72,
				SkillDependencyResolve:  0.74,
				SkillOutputInterpret:    0.84,
				SkillContextAwareness:   0.80,
				SkillSelfCorrection:     0.70,
			},
		},
		"qwen2.5-coder:14b": {
			Name:             "qwen2.5-coder:14b",
			Tier:             2,
			PromptStrategy:   "react",
			ValidationLevel:  ValidationMedium,
			MaxTokens:        32768,
			SupportsNativeFC: false,
			Skillsets: map[string]float64{
				SkillToolSelection:      0.92,
				SkillParameterConstruct: 0.95,
				SkillCommandConstruct:   0.90,
				SkillSearchQuery:        0.88,
				SkillContentParsing:     0.89,
				SkillErrorDiagnosis:     0.80,
				SkillSequentialPlanning: 0.85,
				SkillStateTracking:      0.75,
				SkillDependencyResolve:  0.78,
				SkillOutputInterpret:    0.86,
				SkillContextAwareness:   0.84,
				SkillSelfCorrection:     0.75,
			},
		},
		"qwen2.5-coder:7b": {
			Name:             "qwen2.5-coder:7b",
			Tier:             1,
			PromptStrategy:   "minimal",
			ValidationLevel:  ValidationLight,
			MaxTokens:        32768,
			SupportsNativeFC: false,
			Skillsets: map[string]float64{
				SkillToolSelection:      0.95,
				SkillParameterConstruct: 0.95,
				SkillCommandConstruct:   0.90,
				SkillSearchQuery:        0.90,
				SkillContentParsing:     0.90,
				SkillErrorDiagnosis:     0.90,
				SkillSequentialPlanning: 0.90,
				SkillStateTracking:      0.90,
				SkillDependencyResolve:  0.90,
				SkillOutputInterpret:    0.90,
				SkillContextAwareness:   0.90,
				SkillSelfCorrection:     0.90,
			},
		},

		// Tier 3: Basic Models
		"qwen2.5:7b": {
			Name:             "qwen2.5:7b",
			Tier:             3,
			PromptStrategy:   "guided",
			ValidationLevel:  ValidationStrict,
			MaxTokens:        32768,
			SupportsNativeFC: false,
			Skillsets: map[string]float64{
				SkillToolSelection:      0.75,
				SkillParameterConstruct: 0.80,
				SkillCommandConstruct:   0.70,
				SkillSearchQuery:        0.72,
				SkillContentParsing:     0.68,
				SkillErrorDiagnosis:     0.55,
				SkillSequentialPlanning: 0.60,
				SkillStateTracking:      0.50,
				SkillDependencyResolve:  0.52,
				SkillOutputInterpret:    0.70,
				SkillContextAwareness:   0.65,
				SkillSelfCorrection:     0.45,
			},
		},

		// Tier 4: Minimal Models
		"qwen2.5:3b": {
			Name:             "qwen2.5:3b",
			Tier:             4,
			PromptStrategy:   "template",
			ValidationLevel:  ValidationStrict,
			MaxTokens:        32768,
			SupportsNativeFC: false,
			Skillsets: map[string]float64{
				SkillToolSelection:      0.60,
				SkillParameterConstruct: 0.65,
				SkillCommandConstruct:   0.50,
				SkillSearchQuery:        0.55,
				SkillContentParsing:     0.45,
				SkillErrorDiagnosis:     0.30,
				SkillSequentialPlanning: 0.35,
				SkillStateTracking:      0.25,
				SkillDependencyResolve:  0.28,
				SkillOutputInterpret:    0.50,
				SkillContextAwareness:   0.45,
				SkillSelfCorrection:     0.20,
			},
		},
	}
}

// DetectModelProfile returns a profile for the given model name
func DetectModelProfile(modelName string) ModelProfile {
	profiles := DefaultProfiles()

	// Exact match
	if profile, ok := profiles[modelName]; ok {
		return profile
	}

	// Fuzzy match (e.g., "gpt-4-turbo" -> "gpt-4")
	for key, profile := range profiles {
		if len(modelName) >= len(key) && modelName[:len(key)] == key {
			return profile
		}
	}

	// Default to Tier 3 (Basic) for unknown models
	return ModelProfile{
		Name:             modelName,
		Tier:             3,
		PromptStrategy:   "guided",
		ValidationLevel:  ValidationMedium,
		MaxTokens:        8192,
		SupportsNativeFC: false,
		Skillsets: map[string]float64{
			SkillToolSelection:      0.70,
			SkillParameterConstruct: 0.75,
			SkillCommandConstruct:   0.65,
			SkillSearchQuery:        0.68,
			SkillContentParsing:     0.60,
			SkillErrorDiagnosis:     0.50,
			SkillSequentialPlanning: 0.55,
			SkillStateTracking:      0.45,
			SkillDependencyResolve:  0.48,
			SkillOutputInterpret:    0.65,
			SkillContextAwareness:   0.60,
			SkillSelfCorrection:     0.40,
		},
	}
}

// GetWeakSkillsets returns skillsets with proficiency below threshold
func (p *ModelProfile) GetWeakSkillsets(threshold float64) []string {
	weak := []string{}
	for skill, proficiency := range p.Skillsets {
		if proficiency < threshold {
			weak = append(weak, skill)
		}
	}
	return weak
}

// NeedsTraining returns true if model needs training for given skillset
func (p *ModelProfile) NeedsTraining(skillset string) bool {
	proficiency, ok := p.Skillsets[skillset]
	if !ok {
		return true // Unknown skillset, assume needs training
	}
	return proficiency < 0.85 // Below 85% proficiency needs training
}
