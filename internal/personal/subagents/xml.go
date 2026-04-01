package subagents

import (
	"fmt"
	"strings"
)

// ToPromptXML genera el fragmento XML para insertar en el prompt del agente.
func ToPromptXML(items []*Subagent) string {
	if len(items) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("<available_subagents>\n")
	for _, s := range items {
		sb.WriteString("  <subagent>\n")
		fmt.Fprintf(&sb, "    <name>%s</name>\n", escape(s.Name))
		fmt.Fprintf(&sb, "    <description>%s</description>\n", escape(s.Description))
		fmt.Fprintf(&sb, "    <model>%s</model>\n", escape(s.Model))
		fmt.Fprintf(&sb, "    <auto_delegate>%t</auto_delegate>\n", s.AutoDelegate)
		fmt.Fprintf(&sb, "    <visibility>%s</visibility>\n", escape(string(s.Visibility)))
		if len(s.Tools) > 0 {
			sb.WriteString("    <tools>\n")
			for _, tool := range s.Tools {
				fmt.Fprintf(&sb, "      <tool>%s</tool>\n", escape(strings.TrimSpace(tool)))
			}
			sb.WriteString("    </tools>\n")
		}
		fmt.Fprintf(&sb, "    <location>%s</location>\n", escape(s.FilePath))
		sb.WriteString("  </subagent>\n")
	}
	sb.WriteString("</available_subagents>")
	return sb.String()
}

func escape(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", "\"", "&quot;", "'", "&apos;")
	return r.Replace(s)
}
