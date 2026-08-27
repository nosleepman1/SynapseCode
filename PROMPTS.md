# MASTER PROMPTS & INSTRUCTIONS LLM: SynapseCode

> Ce fichier contient les **System Prompts**, les **Instructions MCP** et les **Templates de Prompts** à injecter dans Claude Desktop, Claude Code CLI, Cursor ou Antigravity pour maximiser la réduction de tokens et l'efficacité du modèle.

---

## 1. System Prompt pour Claude (Configuration Claude Desktop / Code Agent)

Copiez-collez ce prompt dans les instructions personnalisées (*System Prompt* ou `.cursorrules` / `.claude_rules`) de votre agent de code :

```markdown
<synapse_code_protocol>
You are connected to the SynapseCode MCP Server, an ultra-fast Code Graph & AST indexing engine.
Your objective is to solve coding tasks with MAXIMUM precision while consuming the MINIMUM number of tokens.

## RULES FOR CODE INVESTIGATION:
1. NEVER request full raw directory listings or read 20 files at once.
2. When starting a new task or exploring an unfamiliar area:
   - Call `get_context_for_task(task_description="<your intent>", budget_tokens=3000)` first.
   - This provides the exact target implementation AND the signatures of its 1-hop dependencies.
3. Before refactoring or changing a function signature:
   - Always call `get_symbol_callers(symbol_name="<func_name>")` to see every location that depends on it, preventing regressions without loading those full files.
4. If you only need to understand the project architecture:
   - Call `get_repo_map(budget_tokens=1500)`.

## INTERPRETING CODE SKELETONS:
- Skeletons provide `// ... implementation hidden ...` or `/* ... */` for body content.
- Treat signatures, docstrings, and type definitions as absolute truth.
- Only request full file content for files you explicitly need to modify or debug deeply.
</synapse_code_protocol>
```

---

## 2. Configuration `claude_desktop_config.json`

Pour activer SynapseCode dans Claude Desktop sur Windows, macOS ou Linux :

### Windows (`%APPDATA%\Claude\claude_desktop_config.json`) :
```json
{
  "mcpServers": {
    "synapse-code": {
      "command": "C:\\Users\\abash\\go\\bin\\synapse.exe",
      "args": ["mcp", "--path", "C:\\Users\\abash\\Documents\\MonProjet"],
      "env": {
        "SYNAPSE_LOG_LEVEL": "info"
      }
    }
  }
}
```

### macOS / Linux (`~/.config/Claude/claude_desktop_config.json`) :
```json
{
  "mcpServers": {
    "synapse-code": {
      "command": "synapse",
      "args": ["mcp", "--path", "/home/user/projects/mon-projet"],
      "env": {
        "SYNAPSE_LOG_LEVEL": "info"
      }
    }
  }
}
```

---

## 3. Template de Sortie Généré par l'Outil `get_context_for_task`

Voici le format exact que SynapseCode renvoie à Claude lorsque l'outil `get_context_for_task` est déclenché. Ce format a été optimisé pour le raisonnement des LLMs :

```markdown
# SYNAPSE-CODE RELEVANT CONTEXT (Budget: 3,210 / 3,500 tokens)

## 🎯 Target Implementations (Full Source)

### File: `internal/auth/jwt_service.go` [Lines 42-88]
```go
func (s *JWTService) ValidateToken(ctx context.Context, tokenString string) (*Claims, error) {
    token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
        if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
            return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
        }
        return s.secretKey, nil
    })
    if err != nil {
        return nil, fmt.Errorf("token validation failed: %w", err)
    }
    if claims, ok := token.Claims.(*Claims); ok && token.Valid {
        return claims, nil
    }
    return nil, ErrInvalidClaims
}
```

---

## 🔗 Direct Dependencies & Skeletons (1-Hop Callers & Callees)

### File: `internal/auth/claims.go` (Signatures & Types only)
```go
package auth

type Claims struct {
    UserID    string   `json:"uid"`
    Role      string   `json:"role"`
    ExpiresAt int64    `json:"exp"`
    jwt.RegisteredClaims
}

func (c *Claims) HasRole(role string) bool { /* ... */ }
```

### File: `internal/http/middleware/auth_middleware.go` (Caller Signature)
```go
package middleware

// Calls: JWTService.ValidateToken at line 54
func Authenticate(jwtService *auth.JWTService) gin.HandlerFunc { /* ... */ }
```

---

## 🗺️ Project Architecture Summary
- `internal/auth`: Handles token signing, verification, and role-based ACL.
- `internal/http/middleware`: Injects authenticated user claims into HTTP contexts.
- `internal/db`: User repository and token blacklist store.
```

---

## 4. Exemple de Scénario Réel & Comparatif de Tokens

### 📝 Scénario : L'utilisateur demande de corriger un bug d'expiration de token JWT

#### Approche Naive (Lecture brute de 6 fichiers complets) :
1. `internal/auth/jwt_service.go` (450 lignes $\to$ 3 200 tokens)
2. `internal/auth/claims.go` (120 lignes $\to$ 950 tokens)
3. `internal/http/middleware/auth_middleware.go` (280 lignes $\to$ 2 100 tokens)
4. `internal/db/user_repository.go` (600 lignes $\to$ 4 800 tokens)
5. `internal/config/config.go` (200 lignes $\to$ 1 600 tokens)
6. `cmd/api/main.go` (350 lignes $\to$ 2 800 tokens)
* **Total consommé : ~15 450 tokens**
* **Coût par prompt : ~0.046\$**
* **Temps de premier token : ~7.2 secondes**

#### Approche SynapseCode (AST + Call Graph + PageRank Pruning) :
1. `ValidateToken` code complet $\to$ 380 tokens
2. `Claims` struct et signatures $\to$ 120 tokens
3. `Authenticate` middleware signature $\to$ 65 tokens
4. Résumé structurel $\to$ 150 tokens
* **Total consommé : ~715 tokens**
* **Coût par prompt : ~0.0021\$**
* **Temps de premier token : ~1.1 seconde**

🔥 **Économie réelle constatée : 95.3% de tokens en moins pour un résultat identique !**
