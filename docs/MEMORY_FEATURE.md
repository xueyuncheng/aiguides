# Memory Feature

## Overview

The Memory feature allows the AI assistant to remember user characteristics, preferences, and context across different chat sessions. This enables more personalized and continuous conversations.

## Database Schema

### UserMemory Table

| Field | Type | Description |
|-------|------|-------------|
| ID | int | Primary key |
| UserID | int | Foreign key to User table |
| MemoryType | string | Type of memory: `fact`, `preference`, or `context` |
| Content | text | The actual memory content |
| Importance | int | Priority level (1-10), used for sorting |
| Metadata | text | Additional metadata in JSON format |
| CreatedAt | timestamp | When the memory was created |
| UpdatedAt | timestamp | When the memory was last updated |

## Memory Types

1. **fact** - Objective facts about the user
   - Examples: "User is a software engineer", "User works at Google", "User knows Go and Python"

2. **preference** - User's subjective preferences and habits
   - Examples: "User prefers concise code", "User likes dark themes", "User prefers functional programming"

3. **context** - Short-term context and ongoing projects
   - Examples: "User is building an e-commerce website", "User is learning machine learning"

## Agent Tool: manage_memory

The AI agent has access to a `manage_memory` tool with the following operations:

### Save Memory
```json
{
  "action": "save",
  "user_id": 1,
  "memory_type": "fact",
  "content": "User is a software engineer",
  "importance": 8
}
```

### Retrieve Memories
```json
{
  "action": "retrieve",
  "user_id": 1,
  "memory_type": "fact"  // Optional: filter by type
}
```

### Update Memory
```json
{
  "action": "update",
  "user_id": 1,
  "memory_id": 123,
  "content": "User is a senior software engineer",
  "importance": 9
}
```

### Delete Memory
```json
{
  "action": "delete",
  "user_id": 1,
  "memory_id": 123
}
```

## How It Works

1. **Automatic Memory Retrieval**: When a new chat session starts, the system checks if there are any existing memories for the user.

2. **Agent Decision**: The AI agent decides when to save, update, or retrieve memories based on the conversation context.

3. **Memory Management**: The agent uses the `manage_memory` tool to:
   - Save important information shared by the user
   - Retrieve memories to provide personalized responses
   - Update memories when user's situation changes
   - Delete outdated or incorrect memories

4. **Cross-Session Persistence**: All memories are stored in the database and persist across sessions.

## Agent Instructions

The agent has been instructed to:
- Automatically identify and save important user information
- Respect user privacy (only save what users explicitly share)
- Use memories appropriately to enhance conversation quality
- Keep memories accurate and up-to-date
- Be transparent about using memory information

## Usage Examples

### Example 1: Saving a Fact
**User**: "I'm a software engineer specializing in Go."
**Agent**: [Internally calls manage_memory with action="save", content="User is a software engineer specializing in Go", memory_type="fact"]
**Agent**: "Got it! I'll remember that you're a Go specialist."

### Example 2: Retrieving Memories
**User**: "Can you suggest some learning resources?"
**Agent**: [Internally calls manage_memory with action="retrieve"]
**Agent**: "Since you're a Go specialist, I recommend these advanced Go resources..."

### Example 3: Using Context
**User**: "What should I learn next?"
**Agent**: [Retrieves: memory_type="fact" shows "User knows Go", memory_type="context" shows "User is building a microservices platform"]
**Agent**: "Given that you're building a microservices platform with Go, I suggest learning about service mesh technologies like Istio or Linkerd..."

## Future Enhancements

Potential improvements for future versions:

1. **Memory Summarization**: Automatically merge similar memories
2. **Memory Decay**: Reduce importance of old, unused memories
3. **Memory Search**: Full-text search across all memories
4. **Memory Categories**: More granular categorization
5. **User Interface**: Web UI for managing memories manually
6. **Memory Export**: Allow users to export their memory data
7. **Privacy Controls**: Let users control what can be remembered

## Technical Details

### Implementation Files

- `internal/app/aiguide/table/table.go` - Database schema
- `internal/pkg/tools/memory.go` - Memory tool implementation
- `internal/pkg/tools/memory_test.go` - Unit tests
- `internal/app/aiguide/assistant/agent.go` - Agent integration
- `internal/app/aiguide/assistant/assistant_agent_prompt.md` - Agent instructions

### Testing

Run memory tests:
```bash
go test ./internal/pkg/tools -run Memory -v
```

Run all tests:
```bash
go test ./... -v
```

## Privacy Considerations

- Memories are user-specific and isolated
- Only information explicitly shared by users is saved
- Users should be able to view and delete their memories (future feature)
- Memory data should be encrypted at rest (future enhancement)
- Comply with data protection regulations (GDPR, CCPA, etc.)
