# System Prompts and Integration Instructions: SynapseCode

## 1. System Prompt for Claude Desktop and AI Coding Agents

Include the following prompt in the custom system instructions (`System Prompt` or `.cursorrules` / `.claude_rules`) of your coding assistant:

```markdown
<synapse_code_protocol>
You are connected to the SynapseCode MCP Server, a code intelligence and AST graph engine.
Your objective is to solve coding tasks with maximum accuracy while consuming minimum context tokens.

RULES FOR CODE EXPLORATION:
1. Do not request full repository file dumps or read dozens of files simultaneously.
2. When addressing a coding task:
   - Call `get_context_for_task(task_description="<intent>", budget_tokens=3500)` first.
   - This provides the full implementation of target functions and signatures of 1-hop dependencies.
3. Before refactoring or changing any function signature:
   - Call `get_symbol_callers(symbol_name="<function_name>")` to inspect dependent callers and prevent regressions.
4. When orienting yourself in an unfamiliar repository:
   - Call `get_repo_map(budget_tokens=1500)`.

INTERPRETING SKELETONS:
- Skeletons provide method and class signatures, type definitions, and docstrings.
- Treat signatures and interfaces as ground truth.
- Only request full file contents for files that require direct modifications.
</synapse_code_protocol>
```

---

## 2. Configuration Setup: `claude_desktop_config.json`

### Windows (`%APPDATA%\Claude\claude_desktop_config.json`):
```json
{
  "mcpServers": {
    "synapse-code": {
      "command": "synapse",
      "args": ["mcp", "--path", "C:\\path\\to\\your\\repository"]
    }
  }
}
```

### macOS / Linux (`~/.config/Claude/claude_desktop_config.json`):
```json
{
  "mcpServers": {
    "synapse-code": {
      "command": "synapse",
      "args": ["mcp", "--path", "/path/to/your/repository"]
    }
  }
}
```

---

## 3. Sample Context Output Format

Below is an example of the structured Markdown payload returned by `get_context_for_task`:

```markdown
# SYNAPSE-CODE CONTEXT PACK
> **Task**: `Validate JWT claims in auth service` | **Budget Used**: 1,120 / 3,500 tokens

## Primary Target Implementations (Full Code)

### File: `internal/auth/service.go` — Symbol: `ValidateToken` (method)
```go
func (s *AuthService) ValidateToken(ctx context.Context, tokenStr string) (*Claims, error) {
    claims := &Claims{}
    token, err := jwt.ParseWithClaims(tokenStr, claims, s.keyFunc)
    if err != nil || !token.Valid {
        return nil, ErrInvalidToken
    }
    return claims, nil
}
```

## Direct Dependencies & Skeletons (Signatures Only)

### File: `internal/auth/claims.go` — `Claims` (Signature)
```go
type Claims struct {
    UserID string `json:"uid"`
    Role   string `json:"role"`
    jwt.RegisteredClaims
}
```

### File: `internal/http/middleware.go` — `AuthMiddleware` (Signature)
```go
func AuthMiddleware(authService *AuthService) gin.HandlerFunc
```
```
