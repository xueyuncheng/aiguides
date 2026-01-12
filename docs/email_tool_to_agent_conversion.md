# Email Query: Tool to Agent Conversion

## Summary

Successfully converted the email query functionality from a simple `functiontool` to a full-fledged `agent` wrapped inside an `agenttool`. This change enhances the email query capabilities by adding LLM reasoning to understand and process natural language email queries.

## Problem Statement

用户提出了一个问题：目前查询邮件功能是做成了一个 tool，但是 AI 建议说做成 agent 好一点。经过分析，我们认为将邮件查询转换为 agent 确实有诸多好处。

## Analysis: Tool vs Agent

### When to Use Tools (functiontool)
- Simple, direct operations with clear input/output
- No need for LLM reasoning
- Examples: Simple API calls, data transformations

### When to Use Agents (llmagent wrapped in agenttool)
- Complex operations requiring reasoning
- Natural language understanding needed
- Multi-step processes
- Needs to interpret user intent
- Examples: Search (already implemented), Email Query (now implemented)

## Why Agent is Better for Email Query

### 1. **Natural Language Understanding**
With an agent, users can ask:
- "查询我的重要邮件" (Show me my important emails)
- "最近一周的未读邮件" (Unread emails from the last week)  
- "来自老板的邮件" (Emails from my boss)

The LLM in the agent can interpret these natural language queries and translate them into appropriate `query_emails` tool calls.

### 2. **Context-Aware Filtering**
The agent can understand concepts like:
- "Important" emails (might look at subject keywords, sender)
- "Recent" emails (understand time references)
- "Work-related" emails (filter by domain/sender patterns)

### 3. **Better Error Handling**
The agent can:
- Provide more helpful error messages
- Guide users through configuration issues
- Suggest alternative queries when results are empty

### 4. **Intelligent Results Presentation**
The agent can:
- Summarize email contents
- Group related emails
- Highlight key information
- Format responses in a user-friendly way

### 5. **Consistency with Existing Patterns**
Follows the same pattern as `GoogleSearchAgent`:
- Both wrap external tools/services
- Both benefit from LLM reasoning
- Both need natural language understanding

## Implementation Details

### Architecture

```
SearchAgent (Main Agent)
  ├── GoogleSearchTool (agenttool wrapping llmagent)
  │     └── GoogleSearch (geminitool)
  ├── ImageGenTool (functiontool)
  └── EmailAgent (agenttool wrapping llmagent)
        └── EmailQueryTool (functiontool)
```

### Files Created/Modified

#### New File: `internal/pkg/tools/email_agent.go`
- Creates `NewEmailAgent()` function
- Wraps `NewEmailQueryTool()` inside an `llmagent`
- Provides detailed instructions for the LLM
- Returns an `agenttool` that can be used by parent agents

#### Modified: `internal/app/aiguide/assistant/search.go`
- Changed from calling `NewEmailQueryTool()` to `NewEmailAgent()`
- Updated variable names to reflect the change
- No change to external API - still works the same for users

#### Modified: `internal/pkg/tools/email_test.go`
- Added tests for `NewEmailAgent()`
- Kept existing tool tests
- All tests pass successfully

### Agent Instructions

The email agent includes comprehensive instructions:

```go
Instruction: `你是一个专业的邮件查询助手。你的职责是帮助用户查询和管理他们的邮件。

当用户请求查询邮件时，你需要：
1. 理解用户的查询意图（例如："最近的邮件"、"未读邮件"、"重要邮件"等）
2. 使用 query_emails 工具查询邮件
3. 将查询结果以清晰、有条理的方式呈现给用户

注意事项：
- 默认查询 INBOX 文件夹中的最新 10 封邮件
- 如果用户只想看未读邮件，设置 unseen=true
- 如果用户想看更多邮件，可以增加 limit 参数（最多 50）
- 如果用户提到特定的邮箱文件夹（如 Sent、Drafts），使用相应的 mailbox 参数
- 对于查询结果，重点关注邮件的主题、发件人、日期和是否已读
- 如果邮件内容很长，可以适当总结

请直接使用邮件查询工具来完成用户的请求，不需要过多的解释，专注于提供准确、有用的邮件信息。`
```

## Benefits

### For Users
1. **More intuitive** - Can use natural language
2. **Better results** - LLM helps filter and present relevant information
3. **Easier to use** - Don't need to know exact tool parameters

### For Developers
1. **Consistent architecture** - Follows established patterns
2. **Extensible** - Easy to add more email-related capabilities
3. **Maintainable** - Clear separation of concerns

### For the AI System
1. **Better reasoning** - Can make intelligent decisions about email queries
2. **Context awareness** - Understands user intent beyond literal keywords
3. **Composable** - Can combine with other agents/tools

## Testing

All tests pass successfully:

```bash
$ go test ./internal/pkg/tools/... -v
=== RUN   TestNewEmailQueryTool
--- PASS: TestNewEmailQueryTool (0.00s)
=== RUN   TestNewEmailAgent
--- PASS: TestNewEmailAgent (0.00s)
=== RUN   TestNewEmailAgentWithModel
--- PASS: TestNewEmailAgentWithModel (0.00s)
...
PASS
ok  	aiguide/internal/pkg/tools	0.009s

$ go test ./internal/app/aiguide/assistant/... -v
=== RUN   TestNewSearchAgent
--- PASS: TestNewSearchAgent (0.00s)
=== RUN   TestNewSearchAgentWithModel
--- PASS: TestNewSearchAgentWithModel (0.00s)
...
PASS
ok  	aiguide/internal/app/aiguide/assistant	0.009s
```

## Backward Compatibility

- **No breaking changes** - The underlying `NewEmailQueryTool()` function remains unchanged
- **External API unchanged** - Users interact with the system the same way
- **Internal improvement** - Better processing under the hood

## Future Enhancements

With the agent architecture in place, we can easily add:

1. **Email summarization** - Automatically summarize long email threads
2. **Smart filtering** - "Show me urgent emails from this week"
3. **Email categorization** - Automatically group emails by topic
4. **Multi-mailbox search** - Search across multiple email accounts
5. **Email translation** - Translate foreign language emails
6. **Priority detection** - Identify high-priority emails automatically

## Conclusion

Converting the email query tool to an agent was the right decision. It:
- ✅ Enhances natural language understanding
- ✅ Follows established project patterns (GoogleSearchAgent)
- ✅ Provides better user experience
- ✅ Maintains backward compatibility
- ✅ Enables future enhancements
- ✅ All tests pass

The change is minimal, focused, and brings significant benefits to the email query functionality.
