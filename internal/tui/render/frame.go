package render

import "strings"

// BaseFrame ghép các vùng chính theo thứ tự cố định.
func BaseFrame(logo, separator, content, status, progress, help string, hideLogo, hideHint bool) string {
	var b strings.Builder
	if !hideLogo {
		b.WriteString(logo)
		b.WriteString("\n")
	}
	b.WriteString(separator)
	b.WriteString("\n")
	b.WriteString(content)
	b.WriteString("\n")
	b.WriteString(status)
	b.WriteString("\n")
	b.WriteString(progress)
	b.WriteString("\n")
	if !hideHint {
		b.WriteString(help)
	}
	return b.String()
}
