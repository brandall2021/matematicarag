package agent

import (
	"fmt"
	"strings"
)

type Citation struct {
	Index   int    `json:"index"`
	Title   string `json:"title"`
	Page    int    `json:"page,omitempty"`
	Section string `json:"section,omitempty"`
	Content string `json:"content,omitempty"`
}

type CitationManager struct {
	citations []Citation
	nextIdx   int
}

func NewCitationManager() *CitationManager {
	return &CitationManager{
		citations: make([]Citation, 0),
		nextIdx:   1,
	}
}

func (cm *CitationManager) AddFromToolResults(results []map[string]any) {
	for _, r := range results {
		sourcesRaw, ok := r["sources"]
		if !ok {
			continue
		}
		sources, ok := sourcesRaw.([]any)
		if !ok {
			if typed, ok := sourcesRaw.([]map[string]any); ok {
				sources = make([]any, len(typed))
				for i, s := range typed {
					sources[i] = s
				}
			} else {
				continue
			}
		}
		for _, srcRaw := range sources {
			src, ok := srcRaw.(map[string]any)
			if !ok {
				continue
			}
			title, _ := src["document_title"].(string)
			if title == "" {
				continue
			}
			page, _ := src["page"].(float64)
			section, _ := src["section"].(string)

			cm.citations = append(cm.citations, Citation{
				Index:   cm.nextIdx,
				Title:   title,
				Page:    int(page),
				Section: section,
			})
			cm.nextIdx++
		}
	}
}

func (cm *CitationManager) FormatCitations() string {
	if len(cm.citations) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("\n\n📚 **Fuentes académicas**\n")
	for _, c := range cm.citations {
		sb.WriteString(fmt.Sprintf("\n[%d] **%s**", c.Index, c.Title))
		if c.Page > 0 {
			sb.WriteString(fmt.Sprintf(" — Página %d", c.Page))
		}
		if c.Section != "" {
			sb.WriteString(fmt.Sprintf(" — %s", c.Section))
		}
	}
	sb.WriteString("\n")
	return sb.String()
}

func (cm *CitationManager) GetCitationsJSON() []map[string]any {
	result := make([]map[string]any, 0, len(cm.citations))
	for _, c := range cm.citations {
		entry := map[string]any{
			"index": c.Index,
			"title": c.Title,
		}
		if c.Page > 0 {
			entry["page"] = c.Page
		}
		if c.Section != "" {
			entry["section"] = c.Section
		}
		result = append(result, entry)
	}
	return result
}

func (cm *CitationManager) HasCitations() bool {
	return len(cm.citations) > 0
}
