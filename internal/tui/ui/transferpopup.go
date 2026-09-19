package ui

import (
	"fmt"
	"strings"

)

func (a *App) renderTransferPopup() string {
	const width = 76
	if a.transferSession == nil {
		return renderBox(50, "SOM Sync", "\n  No active sync session.\n", colorAccent)
	}

	info := a.transferSession.Info()
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(StatusOKStyle.Render("  Direct device sync is running"))
	b.WriteString("\n\n")
	b.WriteString(DimItemStyle.Render("  Open this URL on the mobile app:"))
	b.WriteString("\n\n")

	if len(info.URLs) == 0 {
		b.WriteString(StatusErrStyle.Render("  No private IPv4 address found."))
		b.WriteString("\n")
	} else {
		for i, u := range info.URLs {
			b.WriteString(fmt.Sprintf("  %d. %s\n", i+1, u))
		}
	}

	b.WriteString("\n")
	b.WriteString(DimItemStyle.Render("  URL is also copied to clipboard."))
	b.WriteString("\n")
	b.WriteString(DimItemStyle.Render("  c: copy  x: stop server  esc: close"))
	b.WriteString("\n")

	return renderBox(width, "SOM Sync", b.String(), colorAccent)
}

