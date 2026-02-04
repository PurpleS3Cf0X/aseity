# Automation & Headless Mode

Aseity isn't just a chat app; it's a CLI tool you can script.

## Headless Mode
You can run Aseity without the UI using the `--headless` flag.

**Syntax:**
```bash
aseity --headless "Your Prompt Here"
```

**Implicit Mode:**
If you pass a prompt argument without `--headless`, Aseity assumes you want a "one-shot" interaction but still shows the TUI for progress.
To suppress the TUI entirely (for pipes), use `--headless`.

## Scripting Examples

### 1. Auto-approve (`-y`)
The `-y` flag allows Aseity to run tools **without asking you**. Be careful!

```bash
# Scan a target and save the report entirely automatically
aseity --headless -y "Run nmap on localhost and save open ports to ports.txt"
```

### 2. Piping Input
You can pipe text *into* Aseity.

```bash
# Analyze a log file
cat error.log | aseity --headless "Explain what went wrong in these logs"
```

### 3. Piping Output
You can pipe Aseity's answer *out* to other tools.

```bash
# Generate a git commit message
git diff | aseity --headless "Generate a conventional commit message for these changes" > commit_msg.txt
git commit -F commit_msg.txt
```

### 4. Red Teaming Workflow
Chain commands together in a script:

```bash
#!/bin/bash

# Create a specialized agent
aseity --headless -y "Create a RedTeam agent..."

# Run a scan
aseity --headless -y "Ask RedTeam to scan target.com..."

# Cleanup
aseity --headless -y "Delete RedTeam agent"
```
