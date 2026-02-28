package scraper

import (
	"testing"
)

func TestParseVTT_BasicParsing(t *testing.T) {
	vtt := `WEBVTT
Kind: captions
Language: en

00:00:01.000 --> 00:00:04.500
Hello world, this is the first line

00:00:05.000 --> 00:00:08.000
Second line of lyrics

00:00:09.000 --> 00:00:12.000
Third <b>line</b> with <c>tags</c>
`

	result := ParseVTT(vtt)

	if len(result) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(result))
	}

	// Check first line.
	if result[0].Start != 1.0 {
		t.Errorf("line 0 start: expected 1.0, got %f", result[0].Start)
	}
	if result[0].End != 4.5 {
		t.Errorf("line 0 end: expected 4.5, got %f", result[0].End)
	}
	if result[0].Text != "Hello world, this is the first line" {
		t.Errorf("line 0 text: got %q", result[0].Text)
	}

	// Check tags are stripped.
	if result[2].Text != "Third line with tags" {
		t.Errorf("line 2 text: expected tags stripped, got %q", result[2].Text)
	}
}

func TestParseVTT_DeduplicateConsecutive(t *testing.T) {
	vtt := `WEBVTT

00:00:01.000 --> 00:00:03.000
Same text

00:00:03.000 --> 00:00:05.000
Same text

00:00:05.000 --> 00:00:07.000
Different text
`

	result := ParseVTT(vtt)

	if len(result) != 2 {
		t.Fatalf("expected 2 lines (dedup), got %d", len(result))
	}

	if result[0].Text != "Same text" {
		t.Errorf("line 0: got %q", result[0].Text)
	}
	if result[1].Text != "Different text" {
		t.Errorf("line 1: got %q", result[1].Text)
	}
}

func TestParseVTT_SRTFormat(t *testing.T) {
	srt := `1
00:00:01,000 --> 00:00:04,000
SRT uses commas

2
00:00:05,000 --> 00:00:08,000
Instead of dots
`

	result := ParseVTT(srt)

	if len(result) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(result))
	}

	if result[0].Text != "SRT uses commas" {
		t.Errorf("line 0: got %q", result[0].Text)
	}
}

func TestParseVTT_HoursInTimestamp(t *testing.T) {
	vtt := `WEBVTT

01:02:03.456 --> 01:02:06.789
Long video timestamp
`

	result := ParseVTT(vtt)

	if len(result) != 1 {
		t.Fatalf("expected 1 line, got %d", len(result))
	}

	expected := 1*3600 + 2*60 + 3 + 0.456
	if result[0].Start != expected {
		t.Errorf("start: expected %f, got %f", expected, result[0].Start)
	}
}

func TestParseVTT_EmptyInput(t *testing.T) {
	result := ParseVTT("")
	if len(result) != 0 {
		t.Fatalf("expected 0 lines for empty input, got %d", len(result))
	}
}
