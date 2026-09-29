package voice

import (
	"strings"
	"testing"
	"time"
)

func TestBuildArgs(t *testing.T) {
	client := NewClient(Config{
		Command:        "som-voice-wake",
		WakeWord:       "yui",
		WakeAliases:    "yui,ui,uy",
		CommandTimeout: 2500 * time.Millisecond,
	})

	args := strings.Join(client.buildArgs(), " ")
	want := "--json --wake-word yui --wake-aliases yui,ui,uy --command-timeout 2.5s"
	if args != want {
		t.Fatalf("args = %q, want %q", args, want)
	}
}
