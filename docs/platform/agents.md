# Custom Agents

Aseity (v1.1+) allows you to create specialized "personas" called **Agents**. Unlike the general-purpose assistant, these agents have specific roles, knowledge, and instructions.

## Creating an Agent
You create agents using **Natural Language**. You don't need to write code or config files manually.

**Command:**
Simply tell Aseity to create an agent.

> "Create a 'LinuxExpert' agent that specializes in bash scripting and system administration."

Aseity might ask for clarification if you are vague, but usually, it will generate:
- **Name**: LinuxExpert
- **Description**: Specialist in Linux systems.
- **System Prompt**: "You are a Linux Expert..."

### specifying Details
You can provide all details in one go:
> "Create a 'Reviewer' agent. Description: Code Reviewer. System Prompt: You are a strict code critic who focuses on security vulnerabilities."

## Knowledge Repositories (v1.1.1+)
You can attach **Knowledge Repositories** (local folders) to an agent. The agent will be explicitly instructed to search these folders for context before answering questions.

**Usage:**
> "Create a 'DocBot' agent with knowledge path '/Users/me/projects/legacy-docs'."

When you ask `DocBot` a question, it will first use `file_search` or `file_read` in `/Users/me/projects/legacy-docs` to find answers.

## Using an Agent
Once created, your agent is saved permanently to `~/.config/aseity/agents/`. You can spawn it anytime.

**Command:**
> "Ask LinuxExpert to check my disk usage."
> "Spawn Reviewer to audit this main.go file."

Aseity will:
1. Initialize the sub-agent with its custom persona.
2. Delegate the task.
3. Show you the agent's progress.
4. Return the final result to the main chat.

## Deleting an Agent
If you no longer need an agent, you can delete it using natural language.

**Command:**
> "Delete the LinuxExpert agent."

This permanently removes the configuration file.

## Managing Agents Manually
Advanced users can edit agent configurations directly.
Files are stored in YAML format at `~/.config/aseity/agents/`.

**Example `LinuxExpert.yaml`:**
```yaml
name: LinuxExpert
description: Specialist in Linux systems
system_prompt: |
  You are a Linux Expert. You excel at bash scripting...
knowledge_paths:
  - /usr/share/doc
  - /usr/share/doc
```

## Auto-Verification Loop (v1.1.2)
You can spawn agents that automatically verify their own work using the built-in Critic.

**Usage:**
- When spawning an agent (via tool or future commands), set `require_review: true`.
- The agent will:
  1. Draft a solution.
  2. Ask the Critic "Does this satisfy the goal?".
  3. If No, retry with feedback (Max 3 times).

## Default Personas
Aseity ships with built-in personas you can use immediately.

### Consultant
Role: Consultant & Architect
- Behavior: Helpful, plans before acting, asks clarifying questions.
- Usage: `aseity "Ask the Consultant to review this"`
