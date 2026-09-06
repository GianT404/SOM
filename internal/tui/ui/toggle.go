package ui

import (
	"strings"

	"charm.land/lipgloss/v2"
)

type Switch struct {
	label string
	desc  string
	value bool
}

func NewSwitch(label, desc string, value bool) Switch {
	return Switch{label: label, desc: desc, value: value}
}

func (s Switch) Label() string { return s.label }
func (s Switch) Desc() string  { return s.desc }
func (s Switch) Value() bool   { return s.value }

func (s *Switch) Set(v bool) {
	s.value = v
}

// ToggleLeft/Right đảo giá trị (wrap-around 2 trạng thái).
func (s *Switch) ToggleLeft() {
	s.value = !s.value
}

func (s *Switch) ToggleRight() {
	s.value = !s.value
}

func (s Switch) valueText() string {
	if s.value {
		return "ON"
	}
	return "OFF"
}

func segment(st lipgloss.Style, text string) string {
	return st.Render(text)
}

// fillToWidth căn giữa text trong width bằng spaces cùng style
func fillToWidth(st lipgloss.Style, text string, width int) string {
	tw := len(text)
	if tw >= width {
		return segment(st, text)
	}
	pad := width - tw
	l := pad / 2
	r := pad - l
	return segment(st, strings.Repeat(" ", l)) + segment(st, text) + segment(st, strings.Repeat(" ", r))
}

func (s Switch) arrowText(side int, focused bool) string {
	ch := "<-"
	if side > 0 {
		ch = "->"
	}

	if focused {
		return segment(SelectedItemStyle, ch)
	}
	return DimItemStyle.Render(ch)
}

func (s Switch) buttonRow(width int, focused bool) string {
	if width < 5 {
		width = 5
	}

	var st lipgloss.Style
	if focused {
		st = SelectedItemStyle
	} else {
		st = lipgloss.NewStyle()
	}

	// '<' và '>' cố định ở 2 rìa
	midW := width - 4
	state := s.valueText()
	pad := midW - len(state)
	if pad < 0 {
		pad = 0
	}
	l := pad / 2
	r := pad - l

	return s.arrowText(-1, focused) +
		segment(st, strings.Repeat(" ", l)) +
		segment(st, state) +
		segment(st, strings.Repeat(" ", r)) +
		s.arrowText(1, focused)
}

func (s Switch) View(width int, focused bool) string {
	if width < 3 {
		width = 3
	}

	var title string
	if focused {
		title = fillToWidth(SelectedItemStyle, s.label, width)
	} else {
		title = fillToWidth(lipgloss.NewStyle(), s.label, width)
	}

	return strings.Join([]string{title, s.buttonRow(width, focused)}, "\n")
}
