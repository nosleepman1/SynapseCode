# Guide de Contribution (CONTRIBUTING.md)

Merci de vous intéresser à la contribution de **SynapseCode** ! Ce projet a été conçu pour être facilement extensible, propre et accueillant pour les mainteneurs open-source.

---

## Prérequis de Développement

* **Go** $\ge$ 1.22
* **Make**
* **golangci-lint** $\ge$ v1.58
* **Git**

Pour cloner et préparer l'environnement local :
```bash
git clone https://github.com/your-username/synapse-code.git
cd synapse-code
go mod download
```

---

##  Commandes Essentielles (`Makefile`)

Nous utilisons un `Makefile` standard pour toutes les tâches d'ingénierie :

```bash
# Compiler le binaire localement
make build

# Exécuter tous les tests unitaires avec détection de race conditions
make test

# Lancer la suite de benchmarks de performance (parsing & indexation)
make bench

# Vérifier la conformité du code avec golangci-lint
make lint

# Lancer le serveur MCP en local pour tester avec un projet
make run-mcp PATH=/chemin/vers/un/projet
```

---

## Comment Ajouter le Support d'un Nouveau Langage

L'architecture est 100% découplée. Pour ajouter un nouveau langage (ex: **Java** ou **C#**) :

### Étape 1 : Créer le package dans `internal/ast/<langage>/`
Créez `internal/ast/java/parser.go` :

```go
package java

import (
    "context"
    "synapse-code/internal/ast"
    sitter "github.com/tree-sitter/go-tree-sitter"
    // Import du binding tree-sitter java
)

type Parser struct {
    language *sitter.Language
}

func New() *Parser {
    return &Parser{/* ... */}
}

func (p *Parser) Language() string {
    return "java"
}

func (p *Parser) Extensions() []string {
    return []string{".java"}
}

func (p *Parser) Parse(ctx context.Context, path string, content []byte) (*ast.FileAST, error) {
    // 1. Parser le contenu avec Tree-sitter
    // 2. Extraire les classes, méthodes, interfaces et imports
    // 3. Retourner la structure *ast.FileAST
    return nil, nil
}
```

### Étape 2 : Enregistrer le parser dans `internal/ast/registry.go`
```go
func init() {
    DefaultRegistry.Register(java.New())
}
```

### Étape 3 : Ajouter un cas de test dans `testdata/sample-java/`
Ajoutez un fichier de test Java représentatif et écrivez un test unitaire validant l'extraction correcte des symboles et des appels de méthodes.

---

## Règles & Standards de Code

1. **Zéro allocation inutile dans les boucles chaudes** : Les fonctions d'indexation et de parsing traitent des millions de lignes de code. Privilégiez les buffers réutilisables (`sync.Pool`) et évitez les allocations d'objets superflues.
2. **Gestion d'erreur explicite avec `%w`** : Toujours envelopper les erreurs pour préserver la stack (`fmt.Errorf("ast parse failed: %w", err)`).
3. **Concurrence sûre** : Toute structure partagée entre Goroutines doit être protégée par un `sync.RWMutex` ou passer par des canaux (`channels`).
4. **Couverture de tests** : Toute nouvelle fonctionnalité ou correction de bug doit comporter un test unitaire (`_test.go`).

---

## Checklist avant de Soumettre une Pull Request (PR)

- [ ] `make test` passe sans aucune erreur ni race condition (`-race`).
- [ ] `make lint` ne remonte aucun avertissement.
- [ ] `make bench` ne montre aucune régression de vitesse sur l'indexation.
- [ ] Les nouveaux fichiers publics sont commentés conformément aux conventions `godoc`.
- [ ] Votre branche est rebasée sur la dernière version de `main`.
