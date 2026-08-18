package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type helpBind [2]string

type helpSection struct {
	title string
	binds []helpBind
}

var helpSections = []helpSection{
	{
		title: "Global & Navigation",
		binds: []helpBind{
			{"?", "Toggle Help"},
			{"tab", "Switch Tab"},
			{"1-5", "Jump to Tab"},
			{"/", "Search"},
			{":", "Commands"},
			{"\\", "Visualizer"},
			{"alt + q", "Quit"},
		},
	},
	{
		title: "Player",
		binds: []helpBind{
			{"enter", "Play Selected"},
			{"space", "Play / Pause"},
			{"] / [", "Next / Prev"},
			{"r", "Toggle Random"},
			{"←/→", "Seek +/- 5s"},
			{"+/−", "Volume Up / Down"},
		},
	},
	{
		title: "Playlists & Actions",
		binds: []helpBind{
			{"/", "Create Playlist"},
			{"delete", "Remove Playlist"},
			{"d", "Download"},
			{"l", "Lyrics Language"},
		},
	},
	{
		title: "Flags",
		binds: []helpBind{
			{"--version", "Show version"},
			{"--upgrade", "True to it's name."},
			{"--uninstall", "Remove binary from /usr/local/bin"},
			{"--check-update", "Check for updates without installing"},
		},
	},
}

func renderHelpSection(sec helpSection, keyWidth int, maxDesc int) string {
	headerStyle := lipgloss.NewStyle().Foreground(colorAccent).Bold(true)
	keyStyle := lipgloss.NewStyle().Foreground(colorAccent).Bold(true)
	descStyle := lipgloss.NewStyle().Foreground(colorWhite)
	dividerStyle := lipgloss.NewStyle().Foreground(colorBorder)

	var b strings.Builder
	b.WriteString(headerStyle.Render(sec.title) + "\n")
	b.WriteString(dividerStyle.Render(strings.Repeat("─", keyWidth+16)) + "\n")

	for i, bind := range sec.binds {
		desc := bind[1]
		if maxDesc > 0 {
			desc = truncateRunes(desc, maxDesc)
		}
		key := fmt.Sprintf("%*s", keyWidth, bind[0])
		line := keyStyle.Render(key) + descStyle.Render("  "+desc)
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString(line)
	}
	return b.String()
}

func truncateRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	if n < 1 {
		return ""
	}
	return string(r[:n-1]) + "…"
}

func maxKeyWidth(secs []helpSection) int {
	w := 0
	for _, sec := range secs {
		for _, b := range sec.binds {
			if l := len([]rune(b[0])); l > w {
				w = l
			}
		}
	}
	return w
}

// truncateBodyHeight cắt bớt số dòng hiển thị khi terminal quá thấp để
// chứa hết nội dung, kèm dòng nhắc cuối thay vì tràn ra ngoài khung box.
func truncateBodyHeight(body string, maxLines int) (string, int) {
	if maxLines < 1 {
		maxLines = 1
	}
	lines := strings.Split(body, "\n")
	if len(lines) <= maxLines {
		return body, len(lines)
	}
	keep := maxLines - 1
	if keep < 1 {
		keep = 1
	}
	lines = lines[:keep]
	lines = append(lines, DimItemStyle.Render("… resize terminal to see more"))
	return strings.Join(lines, "\n"), len(lines)
}

const helpGapW = 4 // khoảng cách ngang giữa 2 section trên cùng 1 hàng

// flowHelpLayout xếp từng section như flexbox "flex-wrap": trái→phải theo
// đúng thứ tự khai báo trong helpSections, hết chỗ ngang (availW) thì tự
// xuống hàng mới. Không có khái niệm "cột cố định" hay ghép 2 section
// chung 1 cột — nên thêm section/flag mới sau này tự động chạy đúng, khỏi
// phải sửa lại helpMaxColumns hay logic chia cột nào cả.
func flowHelpLayout(availW int, maxDesc int) (body string, w int, h int) {
	type block struct {
		text string
		w, h int
	}
	blocks := make([]block, len(helpSections))
	for i, sec := range helpSections {
		keyW := maxKeyWidth([]helpSection{sec})
		text := renderHelpSection(sec, keyW, maxDesc)
		blocks[i] = block{text, lipgloss.Width(text), strings.Count(text, "\n") + 1}
	}

	var rows [][]block
	var row []block
	rowW := 0
	for _, b := range blocks {
		add := b.w
		if len(row) > 0 {
			add += helpGapW
		}
		if len(row) > 0 && rowW+add > availW {
			rows = append(rows, row)
			row = nil
			rowW = 0
			add = b.w
		}
		row = append(row, b)
		rowW += add
	}
	if len(row) > 0 {
		rows = append(rows, row)
	}

	gap := strings.Repeat(" ", helpGapW)
	rowStrs := make([]string, 0, len(rows))
	for _, r := range rows {
		parts := make([]string, 0, len(r)*2-1)
		rw := 0
		for i, b := range r {
			if i > 0 {
				parts = append(parts, gap)
				rw += helpGapW
			}
			parts = append(parts, b.text)
			rw += b.w
		}
		rowStrs = append(rowStrs, lipgloss.JoinHorizontal(lipgloss.Top, parts...))
		if rw > w {
			w = rw
		}

		rh := 0
		for _, b := range r {
			if b.h > rh {
				rh = b.h
			}
		}
		h += rh
	}
	h += len(rows) - 1 // dòng trống giữa các hàng

	body = strings.Join(rowStrs, "\n\n")
	return body, w, h
}

func (a *App) renderHelpPopup() string {
	availW := a.width - 2
	if availW < 10 {
		availW = 10
	}
	availH := a.height - 2
	if availH < 6 {
		availH = 6
	}

	body, bodyW, bodyH := flowHelpLayout(availW-6, 0)

	if bodyW+6 > availW {
		keyW := maxKeyWidth(helpSections)
		maxDesc := availW - 6 - keyW - 2
		if maxDesc < 3 {
			maxDesc = 3
		}
		body, bodyW, bodyH = flowHelpLayout(availW-6, maxDesc)
	}

	// Terminal quá thấp: cắt bớt số dòng hiển thị.
	if bodyH+6 > availH {
		body, bodyH = truncateBodyHeight(body, availH-6)
		bodyW = lipgloss.Width(body)
	}

	footer := lipgloss.NewStyle().
		Width(bodyW).
		Align(lipgloss.Center).
		Render(DimItemStyle.Render("Press ? or Esc to close"))

	content := "\n" + body + "\n\n" + footer

	return renderBox(bodyW+6, "Keyboard Shortcuts", content, lipgloss.Color("#e8593c"))
}
