package voice

import (
	"fmt"
	"os"
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

func TestClientSurfacesProcessError(t *testing.T) {
	client := NewClient(Config{
		Command:        os.Args[0],
		WakeWord:       "yui",
		CommandTimeout: time.Second,
	})
	t.Setenv("SOM_VOICE_TEST_HELPER", "error")

	events, err := client.Start()
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer func() { _ = client.Close() }()

	select {
	case event, ok := <-events:
		if !ok {
			t.Fatal("voice event channel closed before error event")
		}
		if event.Event != "error" || !strings.Contains(event.State, "synthetic voice failure") {
			t.Fatalf("event = %#v", event)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for voice error event")
	}
}

func TestClientReadsJSONEvent(t *testing.T) {
	client := NewClient(Config{
		Command:        os.Args[0],
		WakeWord:       "yui",
		WakeAliases:    "yui,ui,uy",
		CommandTimeout: time.Second,
	})
	t.Setenv("SOM_VOICE_TEST_HELPER", "1")

	events, err := client.Start()
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer func() { _ = client.Close() }()

	select {
	case event, ok := <-events:
		if !ok {
			t.Fatal("voice event channel closed before event")
		}
		if event.Event != "command" || event.WakeAlias != "yui" || event.Command != "phát nhạc" {
			t.Fatalf("event = %#v", event)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for voice event")
	}
}

func TestMain(m *testing.M) {
	switch os.Getenv("SOM_VOICE_TEST_HELPER") {
	case "1":
		fmt.Println(`{"event":"command","timestamp":"2026-09-29T00:00:00Z","wake_alias":"yui","command":"phát nhạc","state":"idle"}`)
		os.Exit(0)
	case "error":
		fmt.Fprintln(os.Stderr, "synthetic voice failure")
		os.Exit(1)
	default:
		os.Exit(m.Run())
	}
}
