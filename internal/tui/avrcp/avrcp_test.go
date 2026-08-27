package avrcp

import (
	"fmt"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func dbusSend(args ...string) (string, error) {
	cmd := append([]string{"dbus-send", "--session", "--print-reply"}, args...)
	out, err := exec.Command(cmd[0], cmd[1:]...).CombinedOutput()
	return string(out), err
}

func TestFullSetup(t *testing.T) {
	s := New()
	if s == nil {
		t.Fatal("avrcp.New() returned nil")
	}
	defer s.Close()

	time.Sleep(300 * time.Millisecond)

	// 1. Name owned?
	out, err := dbusSend("--dest=org.freedesktop.DBus", "/org/freedesktop/DBus",
		"org.freedesktop.DBus.NameHasOwner", "string:org.mpris.MediaPlayer2.SOM")
	if err != nil || !strings.Contains(out, "boolean true") {
		t.Fatalf("name NOT owned:\n%s", out)
	}
	fmt.Println("PASS: D-Bus name owned")

	// 2. Get properties WITHOUT metadata update
	out, err = dbusSend("--dest=org.mpris.MediaPlayer2.SOM", dbusPath,
		"org.freedesktop.DBus.Properties.GetAll", "string:org.mpris.MediaPlayer2.Player")
	if err != nil {
		t.Fatalf("GetAll failed (no update): %v\n%s", err, out)
	}
	fmt.Println("PASS: GetAll works without update")
	if strings.Contains(out, "Stopped") {
		fmt.Println("PASS: PlaybackStatus is Stopped")
	}

	// 3. Now update playback status only (no metadata)
	s.UpdatePlaybackStatus("Playing")
	time.Sleep(100 * time.Millisecond)

	out, err = dbusSend("--dest=org.mpris.MediaPlayer2.SOM", dbusPath,
		"org.freedesktop.DBus.Properties.GetAll", "string:org.mpris.MediaPlayer2.Player")
	if err != nil {
		t.Fatalf("GetAll failed after status update: %v\n%s", err, out)
	}
	if strings.Contains(out, "Playing") {
		fmt.Println("PASS: PlaybackStatus updated to Playing")
	} else {
		t.Fatalf("Status not updated:\n%s", out)
	}

	// 4. Now update metadata (with ObjectPath)
	s.UpdateMetadata("test-1", "Test Song", "Test Artist", "", "", 210_000_000)
	time.Sleep(100 * time.Millisecond)

	out, err = dbusSend("--dest=org.mpris.MediaPlayer2.SOM", dbusPath,
		"org.freedesktop.DBus.Properties.GetAll", "string:org.mpris.MediaPlayer2.Player")
	if err != nil {
		t.Fatalf("GetAll failed after metadata update: %v\n%s", err, out)
	}
	if strings.Contains(out, "Test Song") {
		fmt.Println("PASS: Metadata visible")
	} else {
		t.Fatalf("Metadata NOT found:\n%s", out)
	}

	// 5. Introspect
	out, err = dbusSend("--dest=org.mpris.MediaPlayer2.SOM", dbusPath,
		"org.freedesktop.DBus.Introspectable.Introspect")
	if err != nil {
		t.Fatalf("Introspect failed: %v\n%s", err, out)
	}
	if strings.Contains(out, "org.mpris.MediaPlayer2.Player") {
		fmt.Println("PASS: Player in introspection")
	}

	// 6. PlayPause
	exec.Command("dbus-send", "--session", "--type=method_call",
		"--dest=org.mpris.MediaPlayer2.SOM", dbusPath,
		"org.mpris.MediaPlayer2.Player.PlayPause").Run()
	time.Sleep(100 * time.Millisecond)

	select {
	case cmd := <-s.CmdCh:
		fmt.Printf("PASS: Got command: %s\n", cmd)
	default:
		t.Log("WARN: No command on CmdCh")
	}
}
