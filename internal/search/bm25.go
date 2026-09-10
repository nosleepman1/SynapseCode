package search

import (
	"math"
	"sort"
	"strings"

	"github.com/nosleepman1/synapse-code/pkg/model"
)

// BM25Doc represents a tokenized document node for BM25 indexing.
type BM25Doc struct {
	NodeID      model.NodeID
	Node        *model.Node
	SymbolName  string
	Length      float64
	TermWeights map[string]float64 // term -> weighted term frequency
}

// BM25Index implements the Okapi BM25 scoring algorithm with field weighting.
type BM25Index struct {
	k1           float64
	b            float64
	docCount     int
	avgDocLength float64
	docs         []*BM25Doc
	idf          map[string]float64
}

// NewBM25Index constructs a BM25 index over a collection of graph nodes.
func NewBM25Index(nodes []*model.Node) *BM25Index {
	var docs []*BM25Doc
	docFreqs := make(map[string]int)
	totalLength := 0.0

	// Field Weights
	const (
		weightName      = 3.5
		weightPath      = 1.5
		weightSignature = 1.8
		weightDoc       = 1.0
	)

	for _, node := range nodes {
		termWeights := make(map[string]float64)
		docLen := 0.0
		symbolName := ""

		addTerms := func(text string, weight float64) {
			tokens := TokenizeQuery(text)
			for _, t := range tokens {
				termWeights[t] += weight
				docLen += weight
			}
		}

		// 1. Path tokens
		addTerms(node.FilePath, weightPath)

		// 2. Symbol tokens
		if node.Symbol != nil {
			symbolName = node.Symbol.Name
			addTerms(node.Symbol.Name, weightName)
			addTerms(node.Symbol.QualifiedName, weightName)
			addTerms(node.Symbol.Signature, weightSignature)
			if node.Symbol.Documentation != "" {
				addTerms(node.Symbol.Documentation, weightDoc)
			}
		}

		if docLen == 0 {
			docLen = 1.0
		}

		totalLength += docLen

		// Track document frequencies (unique terms per doc)
		for t := range termWeights {
			docFreqs[t]++
		}

		docs = append(docs, &BM25Doc{
			NodeID:      node.ID,
			Node:        node,
			SymbolName:  strings.ToLower(symbolName),
			Length:      docLen,
			TermWeights: termWeights,
		})
	}

	docCount := len(docs)
	avgDocLen := 1.0
	if docCount > 0 {
		avgDocLen = totalLength / float64(docCount)
	}

	// Compute Okapi BM25 IDF
	idf := make(map[string]float64, len(docFreqs))
	for term, df := range docFreqs {
		// Standard Okapi IDF with +1 smoothing
		idfVal := math.Log((float64(docCount)-float64(df)+0.5)/(float64(df)+0.5) + 1.0)
		if idfVal < 0.1 {
			idfVal = 0.1 // Minimum floor for ubiquitous terms
		}
		idf[term] = idfVal
	}

	return &BM25Index{
		k1:           1.2,
		b:            0.75,
		docCount:     docCount,
		avgDocLength: avgDocLen,
		docs:         docs,
		idf:          idf,
	}
}

// Search calculates BM25 scores for all matching documents in the index.
func (idx *BM25Index) Search(query string) []MatchResult {
	queryTokens := TokenizeQuery(query)
	if len(queryTokens) == 0 || idx.docCount == 0 {
		return nil
	}

	var results []MatchResult

	for _, doc := range idx.docs {
		score := 0.0
		var matchedTerms []string

		for _, q := range queryTokens {
			tf, exists := doc.TermWeights[q]
			if !exists || tf <= 0 {
				continue
			}

			termIDF := idx.idf[q]

			// BM25 term saturation formula: (TF * (k1 + 1)) / (TF + k1 * (1 - b + b * (docLen / avgdl)))
			lenNorm := 1.0 - idx.b + idx.b*(doc.Length/idx.avgDocLength)
			numerator := tf * (idx.k1 + 1.0)
			denominator := tf + idx.k1*lenNorm

			termScore := termIDF * (numerator / denominator)

			// Exact Name Match Bonus
			if doc.SymbolName != "" && doc.SymbolName == q {
				termScore *= 2.5
			} else if doc.SymbolName != "" && strings.Contains(doc.SymbolName, q) {
				termScore *= 1.3
			}

			score += termScore
			matchedTerms = append(matchedTerms, q)
		}

		if score > 0 {
			results = append(results, MatchResult{
				NodeID: doc.NodeID,
				Score:  score,
				Reason: strings.Join(matchedTerms, ","),
			})
		}
	}

	sort.SliceStable(results, func(i, j int) bool {
		if results[i].Score == results[j].Score {
			return results[i].NodeID < results[j].NodeID
		}
		return results[i].Score > results[j].Score
	})

	return results
}
