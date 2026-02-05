# Design Documentation - Local Reference Only

**⚠️ IMPORTANT**: This file is `.gitignore`d and will NOT be pushed to GitHub.

## Purpose

This directory contains internal design documentation, logic flows, and architecture diagrams that are for local development reference only.

## Files

- `LOGIC_FLOWS.md` - Comprehensive logic flow diagrams (19KB, 15+ Mermaid diagrams)
- Other design docs as needed

## Why Not in GitHub?

Per user request, design documentation and logic flows should remain local to:
- Protect proprietary system architecture
- Keep internal implementation details private
- Maintain competitive advantage

## Gitignore Patterns

The following patterns are excluded from Git:

```
docs/LOGIC_FLOWS.md
docs/*_FLOWS.md
docs/design/
docs/architecture/
**/design_*.md
**/logic_*.md
**/architecture_*.md
```

## Usage

These documents are for:
- Developer reference
- Debugging complex flows
- Understanding system architecture
- Onboarding new team members (locally)

**Do NOT commit or push these files to any remote repository.**
