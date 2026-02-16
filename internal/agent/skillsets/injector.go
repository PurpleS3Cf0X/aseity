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

You MUST use this format for EVERY step. Never skip the <thought> section.

Format:
<thought>
Here you explain your reasoning.
- Analyze the user's request
- Decide which tool to use
- Verify parameters
</thought>
[TOOL:tool_name|{"param": "value"}]

**Example 1: Installation**
User: "install redis"

<thought>
The user wants to install redis. I am on macOS, so I should use brew.
I will use the bash tool to run the installation command.
</thought>
[TOOL:bash|{"command": "brew install redis"}]

**Example 2: Search**
User: "who is the ceo of google"

<thought>
I need to find the current CEO of Google. I will use web_search.
</thought>
[TOOL:web_search|{"query": "current CEO of Google"}]

**CRITICAL**: 
1. ALWAYS start with a <thought> block.
2. ALWAYS close with </thought>.
3. THEN provide the [TOOL:...] action on a new line.
4. Do NOT verify success inside the same turn. Wait for the tool result.`
}

// getGuidedFramework returns step-by-step guidance for Tier 3 models
func getGuidedFramework() string {
	return `## 🎯 Step-by-Step Execution Guide

CRITICAL: Do not say "Understood" or "I will do that".
JUST DO IT. Call the first tool immediately.

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
3. **READ the results**: Use web_fetch or read_page on the most relevant URL
4. Answer the user using the fetched content

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
- "write and run script/complex logic" → run_script

CRITICAL: Do NOT use 'open', 'xdg-open', 'start', 'curl', or 'wget' to read websites. I cannot see the browser and raw HTML is hard to parse.
ALWAYS use 'web_fetch' to read website content.

Example:
User: "Read google.com"
Correct: web_fetch({"url": "https://google.com"})
Wrong: bash({"command": "open https://google.com"}) ❌ (I can't see the window)

Example:
User: "Create a python script to calculate fibonacci and run it"
Correct: run_script (handles both steps)
Wrong: file_write then bash (slower)`

	case SkillParameterConstruct:
		return `### Parameter Construction
**Problem**: Wrong JSON format or missing fields
**Solution**: Match tool schema exactly

bash tool schema:
{"command": "string"}

run_script tool schema:
{"language": "python|bash|node|go", "content": "code"}

file_read tool schema:
{"path": "string"}

web_search tool schema:
{"query": "string"}

Example:
Tool: run_script
User: "Run python code"
Correct: {"language": "python", "content": "print('hi')"}
Wrong: {"script": "print('hi')"} ❌ (field is "content")`

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

	case SkillSearchQuery:
		return getSearchQueryTraining()

	case SkillContentParsing:
		return getContentParsingTraining()

	default:
		return fmt.Sprintf("### %s\nTraining content for this skillset is being developed.\n", skillset)
	}
}

// GetSkillTraining (continued) for new skills
func getSearchQueryTraining() string {
	return `### ⚡ HINT: How to Answer Search Questions
You must use TWO steps.

**EXAMPLE (Do exactly this loop):**
User: "Find the latest python version"
1. You: [TOOL:web_search|{"query": "latest python version"}]
2. (Tool returns snippets)
3. You: [TOOL:web_fetch|{"url": "https://www.python.org/downloads"}]
4. (Tool returns text: "Python 3.12 is available")
5. You: "The latest version is Python 3.12."

**NOW: Apply this loop to the USER'S request above.**`
}

func getContentParsingTraining() string {
	return `### Content Parsing
**Problem**: Can't find answer in text
**Solution**: Scan for keywords and extract specific data.

When you use web_fetch, you get a wall of text.
1. Look for headers related to your question
2. Look for tables or lists
3. If the answer isn't there, try another URL.`
}
