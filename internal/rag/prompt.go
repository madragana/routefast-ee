package rag

import (
	"fmt"
	"strings"

	"github.com/madragana/routefast-ee/internal/storage"
)

const askSystemPrompt = `You are the RouteFast Enterprise copilot. RouteFast EE is the central control plane for a fleet of autonomous network-defense nodes. Each node senses anomalies, decides on a mitigation, corroborates it with peers via LIP-4D quorum, and actuates over FRR/BGP. Every decision is reported to this server's append-only audit log.

Answer the operator's question using ONLY the provided context, which is drawn from this customer's audit records. Rules:
- Ground every claim in the context. If the context does not contain the answer, say so plainly. Do not invent decisions, nodes, timestamps, or events.
- Cite the audit records you used by their [n] tag.
- You are advisory and read-only. You never issue credentials, change policy, or take any action, and you never advise bypassing quorum or corroboration.
- Be concise and operational.`

// buildContext renders retrieved audit matches into a prompt-ready block with
// stable [n] tags, and returns the distinct sources for citation.
func buildContext(matches []storage.AuditMatch) (string, []string) {
	var sb strings.Builder
	var sources []string
	for i, m := range matches {
		src := fmt.Sprintf("audit %s · %s · %s", shortID(m.AuditID), m.LoggedAt.Format("2006-01-02 15:04:05Z"), m.EventType)
		sources = append(sources, src)
		fmt.Fprintf(&sb, "[%d] (%s, similarity %.2f)\n%s\n\n", i+1, src, m.Similarity, strings.TrimSpace(m.Content))
	}
	return strings.TrimSpace(sb.String()), sources
}

func shortID(id string) string {
	if len(id) >= 8 {
		return id[:8]
	}
	return id
}

func askUserPrompt(question, context string) string {
	return fmt.Sprintf("Context (this customer's audit records):\n%s\n\nQuestion: %s", context, question)
}
