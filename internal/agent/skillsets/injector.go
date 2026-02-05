package skillsets

import (
	"fmt"
	"strings"
)

// InjectSkillsets enhances a base prompt with skillset training based on model profile
func InjectSkillsets(basePrompt string, profile ModelProfile) string {
	var enhanced strings.Builder
	enhanced.WriteString(basePrompt)
	enhanced.WriteString("\n\n")

	// Add strategy-specific enhancements
	switch profile.PromptStrategy {
	case "minimal":
		// Tier 1: Trust the model, minimal guidance
		enhanced.WriteString(getMinimalGuidance())
	case "react":
		// Tier 2: ReAct framework
		enhanced.WriteString(getReActFramework())
	case "guided":
		// Tier 3: Step-by-step guidance
		enhanced.WriteString(getGuidedFramework())
	case "template":
		// Tier 4: Explicit templates
		enhanced.WriteString(getTemplateFramework())
	}

	// Add training for weak skillsets (< 0.70 proficiency)
	weakSkills := profile.GetWeakSkillsets(0.70)
	if len(weakSkills) > 0 {
		enhanced.WriteString("\n\n## 🎓 Skillset Training\n\n")
		enhanced.WriteString("You need extra guidance in these areas:\n\n")

		for _, skill := range weakSkills {
			enhanced.WriteString(GetSkillTraining(skill))
			enhanced.WriteString("\n")
		}
	}

	return enhanced.String()
}

// getMinimalGuidance returns minimal guidance for Tier 1 models
func getMinimalGuidance() string {
	return `## 🎯 Execution Mode

You are highly capable. Use your best judgment for tool selection and execution.
Focus on efficiency and accuracy.`
}

// getReActFramework returns ReAct framework for Tier 2 models
func getReActFramework() string {
	return `## 🎯 ReAct Framework (Reasoning + Acting)

Use this pattern for EVERY request:

**Step 1: Thought**
<thought>
- What does the user want?
- What tool should I use?
- What are the exact parameters?
</thought>

**Step 2: Action**
[TOOL:tool_name|{"param": "value"}]

**Step 3: Observation**
<thought>
- Did it work?
- What's next?
</thought>

**Example**:
User: "install redis"

<thought>
User wants redis installed. I'll use bash with brew.
</thought>

[TOOL:bash|{"command": "brew install redis"}]

<thought>
Command executed. Check if it succeeded.
</thought>`
}

// getGuidedFramework returns step-by-step guidance for Tier 3 models
func getGuidedFramework() string {
	return `## 🎯 Step-by-Step Execution Guide

Follow these steps for EVERY task:

### Step 1: Understand
- Read the user's request carefully
- Identify the action needed (install, check, run, etc.)

### Step 2: Plan
- What tool do I need? (bash, file_read, web_search, etc.)
- What are the exact parameters?
- Do I need to check anything first?

### Step 3: Execute
- Call the tool with correct parameters
- Use [TOOL:name|{json}] format

### Step 4: Verify
- Did it work?
- Do I need to do anything else?

### Common Patterns:

**Install something**:
1. Identify package manager (npm, pip, brew, apt)
2. Use bash tool
3. Command: "package_manager install name"

**Check if something exists**:
1. Use bash tool
2. Command: "which name" or "command -v name"

**Search for information**:
1. Use web_search tool
2. Query: simplified search terms

**Read a file**:
1. Use file_read tool
2. Path: absolute path to file`
}

// getTemplateFramework returns explicit templates for Tier 4 models
func getTemplateFramework() string {
	return `## 🎯 Execution Templates

Use these EXACT templates for common tasks:

### Template 1: Install Package
User says: "install X"
You respond: [TOOL:bash|{"command": "PACKAGE_MANAGER install X"}]

Where PACKAGE_MANAGER is:
- npm (for JavaScript packages)
- pip (for Python packages)
- brew (for macOS tools)
- apt (for Linux tools)

### Template 2: Check if Installed
User says: "is X installed?" or "check X"
You respond: [TOOL:bash|{"command": "which X"}]

### Template 3: Run Command
User says: "run X" or "execute X"
You respond: [TOOL:bash|{"command": "X"}]

### Template 4: Search Web
User says: "search for X" or "find X online"
You respond: [TOOL:web_search|{"query": "X"}]

### Template 5: Read File
User says: "read FILE" or "show me FILE"
You respond: [TOOL:file_read|{"path": "FILE"}]

### Template 6: List Files
User says: "list files" or "show files"
You respond: [TOOL:bash|{"command": "ls -la"}]

**CRITICAL**: Do NOT explain how to do something. Just use the template!`
}

// GetSkillTraining returns training content for a specific skillset
func GetSkillTraining(skillset string) string {
	switch skillset {
	case SkillToolSelection:
		return `### Tool Selection
**Problem**: Choosing wrong tool for task
**Solution**: Match user intent to tool purpose

Tool Guide:
- "install/download/setup" → bash
- "search/find/look up" → web_search
- "read/show/display file" → file_read
- "write/edit/modify file" → file_write
- "check/verify/test" → bash

Example:
User: "install numpy"
Correct: bash (installation requires shell)
Wrong: web_search (don't search HOW to install, just install)`

	case SkillParameterConstruct:
		return `### Parameter Construction
**Problem**: Wrong JSON format or missing fields
**Solution**: Match tool schema exactly

bash tool schema:
{"command": "string"}

file_read tool schema:
{"path": "string"}

web_search tool schema:
{"query": "string"}

Example:
Tool: bash
User: "list files"
Correct: {"command": "ls -la"}
Wrong: {"cmd": "ls"} ❌ (field name is "command" not "cmd")`

	case SkillCommandConstruct:
		return `### Command Construction
**Problem**: Wrong shell commands
**Solution**: Use correct syntax for OS

Common Commands:
- List files: ls -la
- Check if installed: which PROGRAM
- Install (macOS): brew install NAME
- Install (Linux): apt install NAME
- Install (npm): npm install NAME
- Install (pip): pip install NAME

Example:
OS: macOS
Task: "install redis"
Correct: brew install redis
Wrong: apt install redis ❌ (apt is for Linux)`

	case SkillErrorDiagnosis:
		return `### Error Diagnosis
**Problem**: Not understanding what went wrong
**Solution**: Read error messages carefully

Common Errors:
- "command not found" → Program not installed
- "permission denied" → Need sudo or file permissions
- "no such file" → Path doesn't exist
- "connection refused" → Service not running

Example:
Error: "npm: command not found"
Diagnosis: npm not installed
Solution: Install Node.js first
NOT: Try npm again ❌`

	case SkillSequentialPlanning:
		return `### Sequential Planning
**Problem**: Wrong order of steps
**Solution**: Think about dependencies

Correct Order:
1. Check prerequisites
2. Install dependencies
3. Run main action
4. Verify result

Example:
Task: "Deploy app"
Correct Order:
1. Run tests
2. Build
3. Deploy
4. Verify

Wrong Order:
1. Deploy
2. Test ❌ (test BEFORE deploy)`

	case SkillSelfCorrection:
		return `### Self-Correction
**Problem**: Repeating same mistake
**Solution**: Try different approach after failure

Pattern:
1. Try action
2. If fails, analyze error
3. Try DIFFERENT approach
4. Don't repeat same command

Example:
Attempt 1: pip install numpy → Error: pip not found
Analysis: pip not installed
Attempt 2: brew install python (includes pip)
Attempt 3: pip install numpy → Success ✓
NOT: pip install numpy again ❌`

	default:
		return fmt.Sprintf("### %s\nTraining content for this skillset is being developed.\n", skillset)
	}
}
