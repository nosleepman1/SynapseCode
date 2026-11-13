package context

import (
	"fmt"
	"strings"

	"github.com/nosleepman1/synapse-code/pkg/model"
)

// FormatContextPack formats selected context items into a clean, LLM-optimized Markdown string.
func FormatContextPack(pack *model.ContextPack) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# SYNAPSE-CODE CONTEXT PACK\n"))
	sb.WriteString(fmt.Sprintf("> **Task**: `%s` | **Budget Used**: %d / %d tokens\n\n", pack.TaskDescription, pack.UsedTokens, pack.BudgetTokens))

	var fullItems []model.ContextItem
	var skelItems []model.ContextItem

	for _, item := range pack.Items {
		if item.Mode == model.ContextFullCode {
			fullItems = append(fullItems, item)
		} else {
			skelItems = append(skelItems, item)
		}
	}

	// 1. Full Implementations (Primary Targets)
	if len(fullItems) > 0 {
		sb.WriteString("## 🎯 Primary Target Implementations (Full Code)\n\n")
		for _, item := range fullItems {
			sb.WriteString(fmt.Sprintf("### File: `%s` — Symbol: `%s` (%s)\n", item.FilePath, item.SymbolName, item.Kind))
			sb.WriteString(fmt.Sprintf("```%s\n%s\n```\n\n", getLangFence(item.FilePath), strings.TrimSpace(item.Content)))
		}
	}

	// 2. Direct Skeletons & Dependencies (1-Hop Neighbors)
	if len(skelItems) > 0 {
		sb.WriteString("## 🔗 Direct Dependencies & Skeletons (Signatures Only)\n\n")
		for _, item := range skelItems {
			sb.WriteString(fmt.Sprintf("### File: `%s` — `%s` (Signature)\n", item.FilePath, item.SymbolName))
			sb.WriteString(fmt.Sprintf("```%s\n%s\n```\n\n", getLangFence(item.FilePath), strings.TrimSpace(item.Content)))
		}
	}

	if pack.RepoSummary != "" {
		sb.WriteString("## 🗺️ Architectural Map\n\n")
		sb.WriteString(pack.RepoSummary)
		sb.WriteString("\n")
	}

	return sb.String()
}

func getLangFence(filePath string) string {
	lower := strings.ToLower(filePath)
	if strings.HasSuffix(lower, ".go") {
		return "go"
	}
	if strings.HasSuffix(lower, ".ts") || strings.HasSuffix(lower, ".tsx") {
		return "typescript"
	}
	if strings.HasSuffix(lower, ".js") || strings.HasSuffix(lower, ".jsx") {
		return "javascript"
	}
	if strings.HasSuffix(lower, ".py") {
		return "python"
	}
	if strings.HasSuffix(lower, ".rs") {
		return "rust"
	}
	return ""
}
