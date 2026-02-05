# Aseity - Logic Flow Reference

**Purpose**: Visual pipeline documentation for all major features to ensure clarity and prevent future issues.

**Last Updated**: 2026-02-05

---

## Table of Contents

1. [Agent Lifecycle](#1-agent-lifecycle)
2. [Conversation Management](#2-conversation-management)
3. [Tool Execution Pipeline](#3-tool-execution-pipeline)
4. [Skillsets System](#4-skillsets-system)
5. [Web Crawl Feature](#5-web-crawl-feature)
6. [Error Handling & Recovery](#6-error-handling--recovery)
7. [TUI Event Loop](#7-tui-event-loop)
8. [Session Management](#8-session-management)

---

## 1. Agent Lifecycle

### 1.1 Agent Creation Flow

```mermaid
flowchart TD
    Start([User Starts Aseity]) --> CheckConv{Existing<br/>Conversation?}
    
    CheckConv -->|No| NewAgent[agent.New]
    CheckConv -->|Yes| RestoreAgent[agent.NewWithConversation]
    
    NewAgent --> LoadConfig[Load User Config<br/>~/.aseity/skillsets.yaml]
    RestoreAgent --> LoadConfig
    
    LoadConfig --> DetectProfile[Detect Model Profile<br/>skillsets.DetectModelProfile]
    DetectProfile --> MergeProfiles[Merge with User Config<br/>skillsets.MergeProfiles]
    MergeProfiles --> InjectSkillsets[Inject Base Skillsets<br/>into System Prompt]
    InjectSkillsets --> CreateAgent[Create Agent Instance]
    
    CreateAgent --> SetFields{Set Agent Fields}
    SetFields --> ProviderField[prov: Provider]
    SetFields --> ToolsField[tools: Registry]
    SetFields --> ConvField[conv: Conversation]
    SetFields --> ProfileField[profile: ModelProfile]
    SetFields --> ConfigField[userConfig: UserConfig]
    SetFields --> ChannelsField[ConfirmCh, InputCh, RequestCh]
    
    ProviderField --> Ready([Agent Ready])
    ToolsField --> Ready
    ConvField --> Ready
    ProfileField --> Ready
    ConfigField --> Ready
    ChannelsField --> Ready
    
    style NewAgent fill:#90EE90
    style RestoreAgent fill:#90EE90
    style Ready fill:#4CAF50,color:#fff
```

### 1.2 Agent State Preservation (Critical!)

```mermaid
flowchart LR
    subgraph "Initial State"
        A1[Agent Created<br/>✅ profile<br/>✅ userConfig<br/>✅ conversation]
    end
    
    subgraph "Ctrl+C Pressed"
        B1[Cancel Context]
        B2[Save Conversation]
        B3[NewWithConversation]
    end
    
    subgraph "Recreated State"
        C1[New Agent<br/>✅ profile RELOADED<br/>✅ userConfig RELOADED<br/>✅ conversation PRESERVED]
    end
    
    A1 --> B1
    B1 --> B2
    B2 --> B3
    B3 --> C1
    
    style A1 fill:#4CAF50,color:#fff
    style C1 fill:#4CAF50,color:#fff
    style B3 fill:#FFA500,color:#fff
```

**Critical Code Path**:
```
File: internal/agent/agent.go:115-149
Function: NewWithConversation()
Must: Reload userConfig AND profile (fixed in commit d94aef5)
```

---

## 2. Conversation Management

### 2.1 Message Flow

```mermaid
flowchart TD
    UserInput([User Types Message]) --> AddUser[conv.AddUser]
    AddUser --> CheckTokens{Tokens > 80%<br/>of Max?}
    
    CheckTokens -->|No| SendToAgent[agent.Send]
    CheckTokens -->|Yes| Compact[conv.Compact]
    Compact --> SendToAgent
    
    SendToAgent --> RunLoop[agent.runLoop]
    RunLoop --> GetMessages[conv.Messages]
    GetMessages --> AddReminder[Add Turn Reminder<br/>System Message]
    AddReminder --> SendToLLM[prov.Chat]
    
    SendToLLM --> StreamResponse{Response Type}
    StreamResponse -->|Text| AddAssistant[conv.AddAssistant]
    StreamResponse -->|Tool Call| ExecuteTool[Execute Tool]
    
    ExecuteTool --> AddToolResult[conv.AddToolResult]
    AddToolResult --> CheckTokens2{Tokens > 80%?}
    CheckTokens2 -->|Yes| Compact2[conv.Compact]
    CheckTokens2 -->|No| NextTurn[Next Turn]
    Compact2 --> NextTurn
    
    AddAssistant --> Done([Response Complete])
    NextTurn --> RunLoop
    
    style AddUser fill:#90EE90
    style AddAssistant fill:#90EE90
    style AddToolResult fill:#90EE90
    style Compact fill:#FFA500
```

### 2.2 Context Window Management

```mermaid
flowchart LR
    subgraph "Conversation State"
        A[messages: Message array]
        B[totalTokens: int]
        C[maxTokens: 100000]
    end
    
    subgraph "Compaction Trigger"
        D{totalTokens ><br/>maxTokens * 0.8?}
    end
    
    subgraph "Compaction Process"
        E[Keep System Message]
        F[Keep Last 6 Messages]
        G[Summarize Older Messages]
        H[Rebuild Array]
    end
    
    A --> D
    B --> D
    D -->|Yes| E
    E --> F
    F --> G
    G --> H
    H --> A
    D -->|No| Continue([Continue])
    
    style D fill:#FFA500
    style H fill:#4CAF50,color:#fff
```

**Key Files**:
- `internal/agent/conversation.go:104-165`
- Compaction keeps last 6 messages (3 exchanges)
- Summarizes older messages to save tokens

---

## 3. Tool Execution Pipeline

### 3.1 Tool Call Flow

```mermaid
flowchart TD
    LLMResponse([LLM Returns Tool Calls]) --> ParseCalls{Native or<br/>Fallback?}
    
    ParseCalls -->|Native| NativeCalls[provider.ToolCall array]
    ParseCalls -->|Fallback| RegexParse[Regex: TOOL:name args]
    
    RegexParse --> FallbackCalls[Create ToolCall array]
    FallbackCalls --> CheckParallel
    NativeCalls --> CheckParallel{Tool Safe for<br/>Parallel?}
    
    CheckParallel -->|Yes| ParallelExec[Execute in Parallel<br/>WaitGroup]
    CheckParallel -->|No| SerialExec[Execute Serially]
    
    ParallelExec --> ToolRegistry[tools.Execute]
    SerialExec --> ToolRegistry
    
    ToolRegistry --> ToolImpl{Tool Type}
    ToolImpl -->|bash| BashTool[Run Command]
    ToolImpl -->|web_crawl| CrawlTool[Web Crawl]
    ToolImpl -->|file_read| FileTool[File Operations]
    
    BashTool --> NeedsConfirm{Needs<br/>Confirmation?}
    CrawlTool --> NeedsConfirm
    FileTool --> NeedsConfirm
    
    NeedsConfirm -->|Yes| AskUser[Send ConfirmRequest Event]
    NeedsConfirm -->|No| Execute[Execute Tool]
    
    AskUser --> WaitConfirm[Wait on ConfirmCh]
    WaitConfirm --> UserChoice{User Approves?}
    UserChoice -->|Yes| Execute
    UserChoice -->|No| CancelTool[Return Error]
    
    Execute --> Result[tools.Result]
    CancelTool --> Result
    Result --> AddToConv[conv.AddToolResult]
    AddToConv --> NextTurn([Next Agent Turn])
    
    style ParallelExec fill:#4CAF50,color:#fff
    style Execute fill:#90EE90
    style AskUser fill:#FFA500
```

### 3.2 Parallel Execution Safety

```mermaid
flowchart LR
    subgraph "Safe Tools (Parallel)"
        A1[web_crawl]
        A2[web_search]
        A3[web_fetch]
        A4[file_read]
        A5[file_search]
    end
    
    subgraph "Unsafe Tools (Serial)"
        B1[bash]
        B2[file_write]
        B3[spawn_agent]
    end
    
    subgraph "Execution"
        C1[WaitGroup.Add]
        C2[goroutine]
        C3[Sequential]
    end
    
    A1 --> C1
    A2 --> C1
    A3 --> C1
    A4 --> C1
    A5 --> C1
    
    C1 --> C2
    
    B1 --> C3
    B2 --> C3
    B3 --> C3
    
    style C2 fill:#4CAF50,color:#fff
    style C3 fill:#FFA500
```

**Key File**: `internal/agent/parallel.go:4-13`

---

## 4. Skillsets System

### 4.1 Dynamic Skillset Injection

```mermaid
flowchart TD
    UserMessage([User Sends Message]) --> CheckEnabled{Dynamic<br/>Skillsets<br/>Enabled?}
    
    CheckEnabled -->|No| AddMessage[conv.AddUser]
    CheckEnabled -->|Yes| DetectIntent[skillsets.DetectIntent]
    
    DetectIntent --> IntentType{Intent Type}
    IntentType -->|Testing| TestSkills[Load Testing Skillset]
    IntentType -->|Coding| CodeSkills[Load Coding Skillset]
    IntentType -->|Research| ResearchSkills[Load Research Skillset]
    IntentType -->|Security| SecSkills[Load Security Skillset]
    IntentType -->|General| NoExtra[No Extra Skillset]
    
    TestSkills --> BuildPrompt[BuildContextualPrompt]
    CodeSkills --> BuildPrompt
    ResearchSkills --> BuildPrompt
    SecSkills --> BuildPrompt
    NoExtra --> AddMessage
    
    BuildPrompt --> SendEvent[Send EventThinking<br/>Detected intent: X]
    SendEvent --> AddMessage
    AddMessage --> RunLoop[agent.runLoop]
    
    RunLoop --> InjectTemp[Inject Skillset<br/>as Temp System Message]
    InjectTemp --> SendToLLM[prov.Chat]
    
    style DetectIntent fill:#4CAF50,color:#fff
    style InjectTemp fill:#FFA500
```

### 4.2 Intent Detection Logic

```mermaid
flowchart LR
    subgraph "Keywords Analysis"
        A[User Message] --> B[Convert to Lowercase]
        B --> C[Check Keywords]
    end
    
    subgraph "Intent Mapping"
        D{Keywords Match}
        D -->|test, verify, check| E1[Testing]
        D -->|code, implement, function| E2[Coding]
        D -->|research, find, search| E3[Research]
        D -->|security, vulnerability| E4[Security]
        D -->|deploy, docker, setup| E5[DevOps]
        D -->|debug, error, fix| E6[Debugging]
        D -->|explain, what, how| E7[Explanation]
        D -->|create, new, generate| E8[Creation]
        D -->|analyze, review| E9[Analysis]
        D -->|none| E10[General]
    end
    
    C --> D
    E1 --> Result([Intent])
    E2 --> Result
    E3 --> Result
    E4 --> Result
    E5 --> Result
    E6 --> Result
    E7 --> Result
    E8 --> Result
    E9 --> Result
    E10 --> Result
    
    style E1 fill:#90EE90
    style E2 fill:#90EE90
    style E3 fill:#90EE90
```

**Key File**: `internal/agent/skillsets/intent.go:14-103`

---

## 5. Web Crawl Feature

### 5.1 Web Crawl Decision Tree

```mermaid
flowchart TD
    Start([web_crawl Tool Called]) --> ParseArgs[Parse JSON Args<br/>url, wait_for, screenshot]
    ParseArgs --> CheckCrawl4AI{Crawl4AI<br/>Available?}
    
    CheckCrawl4AI -->|Yes| HealthCheck{Health Check<br/>Pass?}
    CheckCrawl4AI -->|No| TryChromedp[Try Chromedp]
    
    HealthCheck -->|Yes| UseCrawl4AI[Use Crawl4AI Service]
    HealthCheck -->|No| TryChromedp
    
    UseCrawl4AI --> Retry{Success?}
    Retry -->|No| RetryCount{Retry < 3?}
    RetryCount -->|Yes| Wait[Exponential Backoff]
    Wait --> UseCrawl4AI
    RetryCount -->|No| RecordFailure[Record Failure<br/>Circuit Breaker]
    
    Retry -->|Yes| RecordSuccess[Record Success]
    RecordSuccess --> ReturnMarkdown[Return Clean Markdown]
    
    TryChromedp --> ChromeAvail{Chrome<br/>Installed?}
    ChromeAvail -->|Yes| UseChromedp[Use Chromedp<br/>Headless Browser]
    ChromeAvail -->|No| BasicHTTP[Fallback to HTTP GET]
    
    UseChromedp --> ChromeSuccess{Success?}
    ChromeSuccess -->|Yes| ReturnText[Return Extracted Text]
    ChromeSuccess -->|No| BasicHTTP
    
    RecordFailure --> BasicHTTP
    BasicHTTP --> ConvertHTML[htmlToText Conversion]
    ConvertHTML --> ReturnBasic[Return with Warning]
    
    ReturnMarkdown --> Truncate[Truncate to 5000 chars]
    ReturnText --> Truncate
    ReturnBasic --> Truncate
    Truncate --> Done([Return Result])
    
    style UseCrawl4AI fill:#4CAF50,color:#fff
    style UseChromedp fill:#90EE90
    style BasicHTTP fill:#FFA500
```

### 5.2 Crawl4AI Health Monitoring

```mermaid
flowchart LR
    subgraph "Health Check Loop"
        A[Every 30 seconds] --> B[HTTP GET /health]
        B --> C{Status 200?}
        C -->|Yes| D[Set isHealthy = true]
        C -->|No| E[Set isHealthy = false]
        D --> A
        E --> A
    end
    
    subgraph "Circuit Breaker"
        F[consecutiveFailures] --> G{Count >= 5?}
        G -->|Yes| H[Disable Crawl4AI<br/>for 5 minutes]
        G -->|No| I[Continue Using]
        H --> J[Auto-recovery Timer]
        J --> K[Re-enable]
        K --> F
    end
    
    style D fill:#4CAF50,color:#fff
    style H fill:#FF6B6B,color:#fff
```

**Key Files**:
- `internal/tools/crawl.go:1-155`
- `docker-compose.yml:36-126` (Crawl4AI service)

---

## 6. Error Handling & Recovery

### 6.1 Error Propagation

```mermaid
flowchart TD
    Error([Error Occurs]) --> ErrorType{Error Type}
    
    ErrorType -->|Tool Execution| ToolError[tools.Result.Error]
    ErrorType -->|Provider| ProviderError[provider.Error]
    ErrorType -->|Context| ContextError[context.Canceled]
    
    ToolError --> FormatError[Format Error Message<br/>Add Suggestions]
    ProviderError --> FormatError
    ContextError --> UserCancel[User Cancelled]
    
    FormatError --> AddToConv[conv.AddToolResult<br/>with error]
    UserCancel --> ShowMessage[Show Cancellation Message]
    
    AddToConv --> SendEvent[EventToolResult<br/>with Error field]
    ShowMessage --> RecreateAgent{Preserve<br/>Agent State?}
    
    RecreateAgent -->|Yes| NewWithConv[NewWithConversation<br/>✅ Preserves config]
    RecreateAgent -->|No| Continue[Continue with<br/>existing agent]
    
    SendEvent --> NextTurn([Agent Next Turn])
    NewWithConv --> Ready([Agent Ready])
    Continue --> Ready
    
    style FormatError fill:#FFA500
    style NewWithConv fill:#4CAF50,color:#fff
```

### 6.2 Graceful Degradation

```mermaid
flowchart LR
    subgraph "Primary"
        A1[Crawl4AI Service]
    end
    
    subgraph "Fallback 1"
        B1[Chromedp Browser]
    end
    
    subgraph "Fallback 2"
        C1[HTTP GET]
    end
    
    subgraph "Always Works"
        D1[Basic HTML Fetch]
    end
    
    A1 -->|Fails| B1
    B1 -->|Fails| C1
    C1 -->|Fails| D1
    D1 --> E([Return Result<br/>with Warning])
    
    style A1 fill:#4CAF50,color:#fff
    style D1 fill:#90EE90
```

---

## 7. TUI Event Loop

### 7.1 Main Event Flow

```mermaid
flowchart TD
    Start([TUI Starts]) --> Init[Model.Init]
    Init --> MainLoop{tea.Update}
    
    MainLoop --> EventType{Event Type}
    
    EventType -->|KeyMsg| HandleKey[Handle Keyboard]
    EventType -->|agentEventMsg| HandleAgent[Handle Agent Event]
    EventType -->|WindowSizeMsg| HandleResize[Handle Resize]
    EventType -->|MouseMsg| HandleMouse[Handle Mouse]
    
    HandleKey --> KeyType{Key Type}
    KeyType -->|Enter| SendMessage[Send to Agent]
    KeyType -->|Ctrl+C| CancelOp[Cancel Operation]
    KeyType -->|Esc| Quit[Save & Quit]
    KeyType -->|PgUp/PgDn| Scroll[Scroll Viewport]
    
    SendMessage --> CreateEvent[Create Event Channel]
    CreateEvent --> GoRoutine[go agent.Send]
    GoRoutine --> WaitEvent[waitForEvent]
    
    CancelOp --> CheckBusy{Agent<br/>Busy?}
    CheckBusy -->|Yes| CancelCtx[Cancel Context]
    CheckBusy -->|No| QuitApp[Quit Application]
    
    CancelCtx --> PreserveConv[Get Conversation]
    PreserveConv --> RecreateAgent[NewWithConversation<br/>✅ With config]
    RecreateAgent --> ShowCancel[Show Cancelled Message]
    
    HandleAgent --> AgentEventType{Event Type}
    AgentEventType -->|EventDelta| AppendText[Append to Message]
    AgentEventType -->|EventToolCall| ShowTool[Show Tool Execution]
    AgentEventType -->|EventToolResult| ShowResult[Show Tool Result]
    AgentEventType -->|EventDone| Complete[Mark Complete]
    
    AppendText --> RebuildView[Rebuild Viewport]
    ShowTool --> RebuildView
    ShowResult --> RebuildView
    Complete --> RebuildView
    ShowCancel --> RebuildView
    Scroll --> RebuildView
    QuitApp --> End([Exit])
    Quit --> End
    
    WaitEvent --> MainLoop
    RebuildView --> MainLoop
    
    style SendMessage fill:#4CAF50,color:#fff
    style RecreateAgent fill:#FFA500
    style RebuildView fill:#90EE90
```

### 7.2 Agent-TUI Communication

```mermaid
sequenceDiagram
    participant User
    participant TUI
    participant Agent
    participant Provider
    participant Tool
    
    User->>TUI: Types message & Enter
    TUI->>TUI: Create event channel (buffered 64)
    TUI->>Agent: go agent.Send(ctx, msg, eventCh)
    
    Agent->>Agent: conv.AddUser(msg)
    Agent->>Agent: Detect intent (if enabled)
    Agent->>TUI: EventThinking (intent detected)
    
    Agent->>Provider: prov.Chat(msgs, tools)
    Provider-->>Agent: Stream chunks
    
    loop For each chunk
        Agent->>TUI: EventDelta (text)
        TUI->>TUI: Append to message
        TUI->>TUI: Rebuild view
    end
    
    Provider-->>Agent: ToolCalls
    Agent->>TUI: EventToolCall
    TUI->>TUI: Show tool execution
    
    Agent->>Tool: Execute(args)
    Tool-->>Agent: Result
    Agent->>Agent: conv.AddToolResult
    Agent->>TUI: EventToolResult
    
    Agent->>TUI: EventDone
    TUI->>User: Display complete response
```

---

## 8. Session Management

### 8.1 Save & Restore Flow

```mermaid
flowchart TD
    subgraph "Save Session"
        A1[User Presses Esc] --> A2[agent.Conversation.Save]
        A2 --> A3[Marshal to JSON]
        A3 --> A4[Write to ~/.config/aseity/sessions/]
        A4 --> A5[Return session path]
    end
    
    subgraph "Restore Session"
        B1[User Loads Session] --> B2[LoadConversation path]
        B2 --> B3[Read JSON file]
        B3 --> B4[Unmarshal messages]
        B4 --> B5[Create Conversation]
        B5 --> B6[NewWithConversation<br/>✅ Reload config]
        B6 --> B7[Rehydrate TUI messages]
    end
    
    A5 --> Done1([Session Saved])
    B7 --> Done2([Session Restored])
    
    style A2 fill:#4CAF50,color:#fff
    style B6 fill:#4CAF50,color:#fff
```

### 8.2 Data Persistence

```mermaid
flowchart LR
    subgraph "In-Memory"
        A[Conversation.messages]
        B[Agent.profile]
        C[Agent.userConfig]
    end
    
    subgraph "On Disk"
        D[~/.config/aseity/sessions/ID.json]
        E[~/.aseity/skillsets.yaml]
    end
    
    subgraph "On Exit"
        F[Save Conversation]
    end
    
    subgraph "On Start"
        G[Load User Config]
        H[Restore Session if provided]
    end
    
    A -->|Esc key| F
    F --> D
    
    E --> G
    G --> B
    G --> C
    
    D -->|--session flag| H
    H --> A
    
    style F fill:#4CAF50,color:#fff
    style G fill:#90EE90
```

---

## Critical Code Paths Reference

### Must-Preserve State on Agent Recreation

**Location**: `internal/agent/agent.go:115-149`

```go
func NewWithConversation(prov, registry, conv) *Agent {
    // ✅ CRITICAL: Must reload these
    userConfig := skillsets.LoadUserConfig()
    profile := skillsets.DetectModelProfile(modelName)
    
    return &Agent{
        conv:       conv,        // ✅ Conversation
        profile:    profile,     // ✅ Skillsets
        userConfig: userConfig,  // ✅ User settings
        // ... other fields
    }
}
```

**Why**: Agent recreation happens on:
- Ctrl+C cancellation (`internal/tui/app.go:378-380`)
- Session restore (`internal/tui/app.go:218-220`)

**Impact if missing**: Agent loses skillsets, intent detection fails, context appears lost

---

### Conversation Compaction Threshold

**Location**: `internal/agent/conversation.go:106-110`

```go
func (c *Conversation) compactIfNeeded() {
    if c.totalTokens < c.maxTokens*80/100 {  // 80% threshold
        return
    }
    c.Compact()  // Keep last 6 messages, summarize rest
}
```

**Why**: Prevents context window overflow

**Tuning**: Adjust `80/100` ratio or message count in `Compact()` if needed

---

### Parallel Tool Execution Safety

**Location**: `internal/agent/parallel.go:4-13`

```go
func IsSafeToParallelize(name string) bool {
    switch name {
    case "web_crawl", "web_search", "web_fetch", "file_read", "file_search":
        return true  // Read-only operations
    default:
        return false  // Write operations must be serial
    }
}
```

**Why**: Prevents race conditions and data corruption

**Adding new tools**: Mark as parallel only if truly read-only

---

## Maintenance Checklist

When adding new features, ensure:

- [ ] Logic flow diagram created/updated
- [ ] State preservation verified (if agent recreation involved)
- [ ] Error handling paths documented
- [ ] Fallback mechanisms identified
- [ ] Critical code paths annotated
- [ ] Test scenarios documented

---

**Document Version**: 1.0  
**Last Updated**: 2026-02-05  
**Maintained By**: Development Team
