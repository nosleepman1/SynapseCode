# PROJECT STRUCTURE & DIRECTORY BLUEPRINT: SynapseCode

> Ce document fournit la décomposition exhaustive de l'arborescence du projet, le rôle précis de chaque fichier, ses entrées/sorties et les contrats d'interface pour les mainteneurs.

---

## 1. Arborescence Complète

```
synapse-code/
├── .github/
│   └── workflows/
│       ├── ci.yml                     # Tests unitaires, benchmarks, golangci-lint
│       └── release.yml                # Build binaire multi-OS via GoReleaser
├── cmd/
│   └── synapse/
│       ├── main.go                    # Point d'entrée principal (CLI Cobra + Mode MCP)
│       ├── root.go                    # Configuration racine, flags globaux
│       ├── mcp.go                     # Sous-commande: synapse mcp [--stdio|--port]
│       ├── map.go                     # Sous-commande: synapse map [--budget 2000]
│       ├── graph.go                   # Sous-commande: synapse graph --symbol <name>
│       └── context.go                 # Sous-commande: synapse context "<query>"
├── internal/
│   ├── ast/                           # Sous-système 1: Analyse Syntaxique Tree-sitter
│   │   ├── parser.go                  # Interface unifiée LanguageParser & structs
│   │   ├── registry.go                # Registre dynamique des parsers par extension
│   │   ├── golang/
│   │   │   ├── parser.go              # Implémentation Tree-sitter pour Go
│   │   │   └── queries.go             # Requêtes S-expressions Tree-sitter (.scm)
│   │   ├── typescript/
│   │   │   ├── parser.go              # Implémentation Tree-sitter TS / JS / TSX
│   │   │   └── queries.go
│   │   ├── python/
│   │   │   ├── parser.go              # Implémentation Tree-sitter Python
│   │   │   └── queries.go
│   │   └── rust/
│   │       ├── parser.go              # Implémentation Tree-sitter Rust
│   │       └── queries.go
│   ├── graph/                         # Sous-système 2: Modèle & Algorithmes de Graphe
│   │   ├── model.go                   # Structures Node, Edge, Symbol, Ref, Def
│   │   ├── store.go                   # Gestionnaire de graphe en mémoire
│   │   ├── pagerank.go                # Algorithme de PageRank personnalisé
│   │   ├── traversal.go               # k-hop BFS/DFS, extraction de sous-graphes
│   │   └── export.go                  # Sérialisation Graphviz (DOT), JSON, Mermaid
│   ├── indexer/                       # Sous-système 3: Scanner & Indexation Concurrente
│   │   ├── scanner.go                 # Walk du filesystem, respect du .gitignore
│   │   ├── worker_pool.go             # Pipeline parallèle Goroutines (Fan-out / Fan-in)
│   │   ├── cache.go                   # Invalidation incrémentale par hachage SHA-256
│   │   └── watcher.go                 # File watcher fsnotify pour mise à jour live
│   ├── search/                        # Sous-système 4: Indexation Lexicale BM25
│   │   ├── index.go                   # Moteur d'indexation lexical en mémoire (Bleve)
│   │   └── query.go                   # Parsing de requête utilisateur & scoring
│   ├── pruner/                        # Sous-système 5: Élagage & Formatage Contexte
│   │   ├── tokenizer.go               # Comptage strict de tokens (tiktoken-go / Claude)
│   │   ├── budget.go                  # Gestionnaire de budget de tokens (Knapsack)
│   │   ├── skeleton.go                # Générateur de squelettes (signatures seules)
│   │   └── formatter.go               # Rendu Markdown compact optimisé pour LLM
│   └── mcp/                           # Sous-système 6: Serveur Model Context Protocol
│       ├── server.go                  # Initialisation du serveur MCP (stdio / SSE)
│       ├── tools.go                   # Enregistrement des outils exposés à Claude
│       ├── handlers.go                # Logique métier des outils appelés par Claude
│       └── types.go                   # Schémas JSON-RPC 2.0 / MCP
├── pkg/
│   └── synapse/                       # API publique (si importé comme module Go externe)
│       └── client.go
├── testdata/                          # Projets de test multilingues pour benchmarks
│   ├── sample-go/
│   ├── sample-ts/
│   └── sample-py/
├── .gitignore
├── .golangci.yml                      # Règles strictes de linting
├── .goreleaser.yaml                   # Configuration de release automatisée
├── Makefile                           # Commandes dev: build, test, bench, lint, run
├── go.mod
├── go.sum
├── README.md                          # Documentation utilisateur & Quickstart
├── ARCHITECTURE.md                    # Spécification technique approfondie
├── PROMPTS.md                         # Prompts & instructions pour Claude Desktop / Agent
└── CONTRIBUTING.md                    # Guide pour les nouveaux contributeurs
```

---

## 2. Rôle Détaillé par Composant & Fichier

### 🔹 `cmd/synapse/` (Interface Utilisateur & CLI)
* **`main.go`** : Initialise la CLI via Cobra, configure les logs structurés (`slog`), gère les signaux d'arrêt gracieux (`SIGINT`, `SIGTERM`).
* **`mcp.go`** : Lance le serveur MCP en mode `stdio` (par défaut pour Claude Desktop) ou `--port 8080` (mode SSE pour intégrations web/docker).
* **`map.go`** : Commande CLI `synapse map --budget 2000` : génère un aperçu compressé du repo dans la console ou un fichier `.md`.
* **`context.go`** : Commande CLI `synapse context "refactor auth token"` : teste la génération de contexte élagué sans passer par Claude.

---

### 🔹 `internal/ast/` (Analyse Syntaxique & Tree-sitter)
* **`parser.go`** :
  - Déclare l'interface `LanguageParser` :
    ```go
    type LanguageParser interface {
        Language() string
        Extensions() []string
        Parse(ctx context.Context, path string, content []byte) (*FileAST, error)
    }
    ```
  - Définit `FileAST`, `Symbol` (nom, signature, docstring, position), `Import` et `CallSite`.
* **`registry.go`** : Map thread-safe qui associe une extension (`.go`, `.ts`, `.py`, `.rs`) à son parser respectif.
* **`golang/`, `typescript/`, `python/`, `rust/`** :
  - Chaque sous-dossier implémente `LanguageParser` en utilisant Tree-sitter et des requêtes compilées `.scm` (S-expressions) pour capturer les déclarations de fonctions, méthodes, classes, types et les sites d'appel.

---

### 🔹 `internal/graph/` (Modèle de Données & Algorithmes)
* **`model.go`** :
  - `NodeID` : Identifiant unique composé (ex: `filepath:symbol_name` ou `filepath`).
  - `Node` : Type (`File`, `Function`, `Struct`, `Interface`), métadonnées, lignes de code.
  - `Edge` : Type de relation (`Calls`, `Imports`, `Implements`, `Extends`, `Defines`).
* **`store.go`** :
  - Implémente le stockage en mémoire basé sur `dominikbraun/graph`.
  - Fournit les primitives `AddNode`, `AddEdge`, `GetCallers`, `GetCallees`, `GetDependencies`.
* **`pagerank.go`** :
  - Implémente le **Personalized PageRank** : calcule l'importance de chaque nœud dans le graphe en donnant un poids initial (*teleport vector*) aux symboles identifiés par la recherche sémantique.
* **`traversal.go`** :
  - Extraction de sous-graphe centré sur un nœud (*Ego-Graph*) jusqu'à $k$ sauts de distance.

---

### 🔹 `internal/indexer/` (Scanner Haute Performance)
* **`scanner.go`** :
  - Parcourt l'arborescence du disque en ignorant automatiquement les dossiers du `.gitignore` (ex: `node_modules/`, `vendor/`, `.git/`, `dist/`).
* **`worker_pool.go`** :
  - Divise la liste des fichiers entre $N$ goroutines (égal au nombre de cœurs CPU $GOMAXPROCS$).
  - Chaque worker lit le fichier, vérifie le cache, appelle le parser AST et émet les symboles vers le canal central de construction du graphe.
* **`cache.go`** :
  - Stocke un index léger `map[string]FileFingerprint` (SHA-256 du contenu + mtime).
  - Si un fichier n'a pas changé, le parsing AST est sauté, réduisant le temps de ré-indexation à **< 50ms**.
* **`watcher.go`** :
  - Écoute les événements du système de fichiers (`fsnotify`) pour mettre à jour le graphe en continu sans bloquer l'agent.

---

### 🔹 `internal/search/` (Indexation Lexicale & Découverte)
* **`index.go`** :
  - Maintient un index inversé en mémoire (BM25) indexant les noms de symboles, signatures, noms de fichiers et docstrings.
* **`query.go`** :
  - Reçoit une requête en langage naturel (ex: *"comment marche l'authentification JWT ?"*), extrait les termes clés et renvoie les 5 à 10 symboles de départ (*seed nodes*) pour amorcer le PageRank.

---

### 🔹 `internal/pruner/` (Élagage Intelligent & Budget de Tokens)
* **`tokenizer.go`** :
  - Encapsule `pkoukk/tiktoken-go` (compatible tokenizer Claude / BPE) pour mesurer la taille exacte en tokens de chaque fragment de code.
* **`budget.go`** :
  - Implémente un algorithme de sélection gloutonne / sac à dos (*Knapsack*) : remplit le budget de tokens alloué (ex: 4 000 tokens) avec les éléments dans l'ordre décroissant de leur score PageRank.
* **`skeleton.go`** :
  - Transforme un fichier complet en "squelette" en supprimant le corps des fonctions tout en conservant les signatures, annotations de types et docstrings.
* **`formatter.go`** :
  - Formate la sortie sous forme d'un document Markdown ultra-propre avec balises de fichiers et numéros de lignes pour une injection directe dans le prompt de Claude.

---

### 🔹 `internal/mcp/` (Serveur Model Context Protocol)
* **`server.go`** :
  - Démarre le serveur MCP en écoutant sur `stdin/stdout` (JSON-RPC 2.0).
* **`tools.go`** :
  - Déclare les 3 outils chirurgicaux exposés à Claude :
    1. `get_repo_map` (Vue d'ensemble architecturale).
    2. `get_context_for_task` (Recherche sémantique + élagage sous budget de tokens).
    3. `get_symbol_callers` (Graphe d'appels ascendant pour prévenir les régressions).
* **`handlers.go`** :
  - Exécute les requêtes de Claude et renvoie le contexte formaté.

---

## 3. Guide d'Extension pour les Futurs Mainteneurs

### Comment ajouter un nouveau langage en 3 étapes ?

1. Créer le dossier `internal/ast/<nouveau_langage>/`.
2. Implémenter l'interface `ast.LanguageParser` avec les requêtes Tree-sitter correspondantes.
3. Enregistrer le parser dans `internal/ast/registry.go` :
   ```go
   registry.Register(&nouveau_langage.Parser{})
   ```
4. Ajouter un projet de test dans `testdata/sample-<langage>/` et exécuter `make test`.
