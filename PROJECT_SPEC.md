# PROJECT SPECIFICATION: SynapseCode (v1.0)
> **Ultra-Fast AST & Code-Graph Indexer & MCP Server for LLMs**  
> *Réduire de 70% à 90% la consommation de tokens des modèles comme Claude sans perdre le contexte architectural.*

---

## 1. Executive Summary (Résumé Exécutif)

**SynapseCode** est un moteur d'indexation de code haute performance écrit en **Go**, conçu pour résoudre le problème fondamental de la surcharge de contexte (*context bloat*) et des coûts astronomiques d'API lors de l'utilisation d'assistants de code IA (Claude, GPT-4, etc.).

Au lieu d'injecter des milliers de lignes de code brut (100k+ tokens) dans le prompt du LLM, SynapseCode :
1. **Parse l'AST** (*Abstract Syntax Tree*) via Tree-sitter pour extraire le squelette exact du projet (signatures, interfaces, types, docstrings).
2. **Construit un Graphe de Dépendances en mémoire** (appels de fonctions, imports, héritages).
3. **Calcule un score de pertinence dynamique** (PageRank personnalisé + traversée $k$-hop) basé sur la tâche de l'utilisateur.
4. **Élague le contexte sous un budget strict de tokens** (ex: 3 000 tokens) et l'expose nativement à Claude via le **Model Context Protocol (MCP)**.

---

## 2. Le Problème (The Pain Points)

| Problème Actuel | Conséquence Réelle | Solution SynapseCode |
| :--- | :--- | :--- |
| **Lecture brute de fichiers** | Envoyer 50 fichiers consomme 80k à 200k tokens par prompt. | Seules les signatures et les 2-3 fonctions critiques sont envoyées (3k à 8k tokens). |
| **Facture API exponentielle** | 10 requêtes par jour sur un gros projet = 15\$ à 50\$/jour. | **Réduction de 75% à 90% de la facture d'API.** |
| **Latence élevée (TTFT)** | Temps de premier token de 8 à 15 secondes sur les gros contextes. | Temps de réponse divisé par 4 (moins de 2s). |
| **"Lost in the Middle"** | Le LLM noie son attention dans les détails d'implémentation inutiles. | Précision accrue : le LLM voit l'architecture globale sans bruit. |

---

## 3. Objectifs Quantitatifs & Métriques Clés

* **Réduction de tokens** : $\ge 75\%$ d'économie de tokens par interaction.
* **Vitesse d'indexation initiale** : $\le 1.5\text{ seconde}$ pour 10 000 fichiers de code.
* **Mise à jour incrémentale** : $\le 50\text{ ms}$ lors de la modification d'un fichier.
* **Empreinte mémoire (RAM)** : $\le 80\text{ MB}$ pour un projet de 1 million de lignes de code.
* **Zéro dépendance externe** : Binaire unique autonome distribué via `go install`, `brew` ou binaire précompilé.

---

## 4. Périmètre & Fonctionnalités (Scope)

### ✅ Dans le périmètre (In-Scope)
1. **Multi-langages via Tree-sitter** :
   - Support v1.0 : Go, TypeScript/JavaScript, Python, Rust.
   - Support v1.1 : Java, C/C++, PHP, C#.
2. **Graphe de Dépendances & Appels** :
   - Nœuds : Fichiers, Classes, Interfaces, Fonctions, Méthodes, Types.
   - Arêtes : `imports`, `calls`, `implements`, `defines`, `references`.
3. **Moteur d'Élagage Intelligent (Pruner)** :
   - Algorithme de PageRank personnalisé indexé sur les mots-clés de la tâche.
   - Respect d'un budget strict de tokens via `tiktoken-go`.
4. **Serveur MCP (Model Context Protocol)** :
   - Transport `stdio` pour Claude Desktop, Cursor, Antigravity et Claude Code CLI.
   - Transport `SSE` (Server-Sent Events) pour usage distant ou conteneurisé.
5. **Mode CLI & Scripting** :
   - Commandes `repograph map`, `repograph graph`, `repograph context`.

### ❌ Hors périmètre (Non-Goals)
- Ce n'est **pas** un compilateur complet (pas de vérification de type statique lourde de type `rustc` ou `tsc`).
- Ce n'est **pas** un éditeur de code ou un IDE graphique.
- Ce n'est **pas** un service SaaS cloud payant : 100% open-source, local-first et respectueux de la vie privée.

---

## 5. Matrice des Langages & Support AST

```
+---------------+------------------------+----------------------+--------------------+
| Langage       | Parser Tree-sitter     | Extraction Symboles  | Résolution Appels  |
+---------------+------------------------+----------------------+--------------------+
| Go            | tree-sitter-go         | Structs, Funcs, Intf | Direct + Package   |
| TypeScript/JS | tree-sitter-typescript | Classes, Funcs, Type | Import paths       |
| Python        | tree-sitter-python     | Classes, Defs, Type  | Import / Calls     |
| Rust          | tree-sitter-rust       | Structs, Traits, Impl| Module tree        |
+---------------+------------------------+----------------------+--------------------+
```
