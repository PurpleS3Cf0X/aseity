# Proposal: Recursive Hierarchical Task Planner

## Concept
The current `spawn_agent` tool allows for one layer of delegation. By formalizing a **Recursive Hierarchical Task Planner**, `Aseity` can handle significantly more complex software engineering tasks that require coordination across multiple files and layers of the stack (e.g., Database + API + Frontend).

## The Recursive Model
1.  **Decomposition**: When an agent determines a task is complex (e.g., "Implement User Auth"), it acts as a *Planner Node*.
2.  **Delegation (Recursion Step)**: The Planner assumes a parent role and spawns multiple child agents, each assigned a specific sub-component of the task.
3.  **Base Case**: If a child agent receives a task that fits within a single context window or file scope, it executes the task directly.
    *   *If the task is still too large, it effectively calls itself by spawning further sub-agents (up to a defined `MaxDepth`).*
4.  **Synthesis**: Child agents report success/failure and diffs back to the parent. The parent integrates the results and validates the overall solution.

## Practical Real-World Usage: "Automated Full-Stack Feature Implementation"

**Scenario**: User asks, *"Add a 'Dark Mode' feature to the web application that persists to the database."*

### Execution Flow:

1.  **Root Agent (Level 0)**:
    *   Analyzes the request.
    *   Identifies three main areas of work: Styles, Frontend Logic, and Backend Persistence.
    *   **Spawns**:
        *   `Agent_A`: "Define CSS variables and update generic styles for dark mode."
        *   `Agent_B`: "Create a ThemeToggle component and update the Layout."
        *   `Agent_C`: "Update the User model and API to store theme preference."

2.  **Agent_C (Level 1 - Backend)**:
    *   Realizes "Update User model and API" involves both DB schema and API handlers.
    *   **Spawns**:
        *   `Agent_C1` (Level 2): "Add `theme_preference` column to Users table and run migration."
        *   `Agent_C2` (Level 2): "Update `POST /user/settings` to accept and save theme preference."

3.  **Agent_C1 & C2 (Level 2 - Executors)**:
    *   `Agent_C1` writes the SQL/Go migration. Returns success.
    *   `Agent_C2` edits the handler code. Returns success.

4.  **Agent_C (Level 1)**:
    *   Receives success from C1 and C2.
    *   Verifies the backend builds.
    *   Returns "Backend persistence ready" to Root Agent.

5.  **Root Agent**:
    *   Receives completion from A, B, and C.
    *   Runs integration tests.
    *   Reports "Dark mode implementation complete" to the user.

## Implementation Requirements

To support this fully, we can iterate on the current `spawn_agent`:
1.  **Task Context Protocol**: Sub-agents need to inherit specific relevant context (file paths, read-only snapshots) to avoid re-reading the whole codebase.
2.  **Structured Output**: Sub-agents should return structured data (e.g., `{"status": "success", "modified_files": [...]}`) rather than just text.
3.  **Parallel Execution**: The current `spawn_agent` blocks the parent. Moving to non-blocking spawns (fire-and-forget with a callback or a `wait_all` tool) would speed up performance significantly.

## Benefits
-   **Context Window Management**: Each agent only loads the context relevant to its sub-task.
-   **Isolation**: A mistake in the frontend sub-agent doesn't hallucinate code into the backend.
-   **Speed**: Parallel execution of sub-tasks (with non-blocking update).
