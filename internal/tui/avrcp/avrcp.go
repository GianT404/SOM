package avrcp

import (
	"fmt"
	"log"
	"sync"

	tea "charm.land/bubbletea/v2"
	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/introspect"
)

// Server implements an MPRIS2 media player over D-Bus.
type Server struct {
	conn   *dbus.Conn
	mu     sync.Mutex
	closed bool
	CmdCh  chan string
	props  map[string]map[string]dbus.Variant
}

func New() *Server {
	conn, err := dbus.SessionBus()
	if err != nil {
		log.Printf("[avrcp] D-Bus session bus unavailable: %v", err)
		return nil
	}

	reply, err := conn.RequestName(dbusDest, dbus.NameFlagDoNotQueue)
	if err != nil {
		log.Printf("[avrcp] failed to request D-Bus name: %v", err)
		conn.Close()
		return nil
	}
	if reply != dbus.RequestNameReplyPrimaryOwner {
		log.Printf("[avrcp] D-Bus name %s already taken", dbusDest)
		conn.Close()
		return nil
	}

	s := &Server{
		conn:  conn,
		CmdCh: make(chan string, 1),
		props: map[string]map[string]dbus.Variant{
			dbusMediaPlayer2: {
				"CanQuit":             dbus.MakeVariant(true),
				"CanRaise":            dbus.MakeVariant(false),
				"HasTrackList":        dbus.MakeVariant(false),
				"Identity":            dbus.MakeVariant("SOM"),
				"DesktopEntry":        dbus.MakeVariant(""),
				"SupportedUriSchemes": dbus.MakeVariant([]string{}),
				"SupportedMimeTypes":  dbus.MakeVariant([]string{}),
			},
			dbusPlayerIface: {
				"PlaybackStatus": dbus.MakeVariant("Stopped"),
				"LoopStatus":     dbus.MakeVariant("None"),
				"Rate":           dbus.MakeVariant(1.0),
				"Shuffle":        dbus.MakeVariant(false),
				"Metadata":       dbus.MakeVariant(map[string]dbus.Variant{}),
				"Volume":         dbus.MakeVariant(0.0),
				"Position":       dbus.MakeVariant(int64(0)),
				"MinimumRate":    dbus.MakeVariant(1.0),
				"MaximumRate":    dbus.MakeVariant(1.0),
				"CanGoNext":      dbus.MakeVariant(true),
				"CanGoPrevious":  dbus.MakeVariant(true),
				"CanPlay":        dbus.MakeVariant(true),
				"CanPause":       dbus.MakeVariant(true),
				"CanSeek":        dbus.MakeVariant(true),
				"CanControl":     dbus.MakeVariant(true),
			},
		},
	}

	if err := s.setup(); err != nil {
		log.Printf("[avrcp] setup failed: %v", err)
		conn.Close()
		return nil
	}

	log.Printf("[avrcp] MPRIS2 registered as %s", dbusDest)
	return s
}

func (s *Server) setup() error {
	// Export player methods.
	if err := s.conn.Export(s, dbus.ObjectPath(dbusPath), dbusPlayerIface); err != nil {
		return fmt.Errorf("export player: %w", err)
	}

	// Export Properties interface (we handle GetAll/Get/Set ourselves).
	if err := s.conn.Export(s, dbus.ObjectPath(dbusPath), dbusProperties); err != nil {
		return fmt.Errorf("export properties: %w", err)
	}

	// Export Introspectable.
	node := &introspect.Node{
		Interfaces: []introspect.Interface{
			introspect.IntrospectData,
			{
				Name:       dbusMediaPlayer2,
				Properties: s.buildIntrospectionProps(dbusMediaPlayer2),
			},
			{
				Name:       dbusPlayerIface,
				Methods:    introspect.Methods(s),
				Properties: s.buildIntrospectionProps(dbusPlayerIface),
			},
		},
	}
	if err := s.conn.Export(
		introspect.NewIntrospectable(node),
		dbus.ObjectPath(dbusPath),
		dbusIntrospect,
	); err != nil {
		return fmt.Errorf("export introspect: %w", err)
	}

	return nil
}

func (s *Server) buildIntrospectionProps(iface string) []introspect.Property {
	s.mu.Lock()
	defer s.mu.Unlock()

	props := s.props[iface]
	result := make([]introspect.Property, 0, len(props))
	for name, v := range props {
		result = append(result, introspect.Property{
			Name:   name,
			Type:   v.Signature().String(),
			Access: "read",
		})
	}
	return result
}

func (s *Server) GetAll(iface string) (map[string]dbus.Variant, *dbus.Error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	props, ok := s.props[iface]
	if !ok {
		return nil, &dbus.Error{
			Name: "org.freedesktop.DBus.Error.UnknownInterface",
			Body: []interface{}{fmt.Sprintf("Unknown interface: %s", iface)},
		}
	}

	result := make(map[string]dbus.Variant, len(props))
	for k, v := range props {
		result[k] = v
	}
	return result, nil
}

func (s *Server) Get(iface, prop string) (dbus.Variant, *dbus.Error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	props, ok := s.props[iface]
	if !ok {
		return dbus.Variant{}, &dbus.Error{
			Name: "org.freedesktop.DBus.Error.UnknownInterface",
			Body: []interface{}{fmt.Sprintf("Unknown interface: %s", iface)},
		}
	}
	v, ok := props[prop]
	if !ok {
		return dbus.Variant{}, &dbus.Error{
			Name: "org.freedesktop.DBus.Error.UnknownProperty",
			Body: []interface{}{fmt.Sprintf("Unknown property: %s", prop)},
		}
	}
	return v, nil
}

func (s *Server) Set(iface, prop string, value dbus.Variant) *dbus.Error {
	return nil
}

// ── Introspectable ───────────────────────────────────────────────────

func (s *Server) Introspect() (string, *dbus.Error) {
	s.mu.Lock()
	node := &introspect.Node{
		Interfaces: []introspect.Interface{
			introspect.IntrospectData,
			{
				Name:       dbusMediaPlayer2,
				Properties: s.buildIntrospectionProps(dbusMediaPlayer2),
			},
			{
				Name:       dbusPlayerIface,
				Methods:    introspect.Methods(s),
				Properties: s.buildIntrospectionProps(dbusPlayerIface),
			},
		},
	}
	s.mu.Unlock()

	xml := introspect.NewIntrospectable(node)
	return string(xml), nil
}

// ── Player methods (called from D-Bus goroutine) ─────────────────────

func (s *Server) sendCmd(cmd string) {
	select {
	case s.CmdCh <- cmd:
	default:
	}
}

func (s *Server) Next() *dbus.Error                                  { s.sendCmd("next"); return nil }
func (s *Server) Previous() *dbus.Error                              { s.sendCmd("previous"); return nil }
func (s *Server) Pause() *dbus.Error                                 { s.sendCmd("pause"); return nil }
func (s *Server) PlayPause() *dbus.Error                             { s.sendCmd("playpause"); return nil }
func (s *Server) Stop() *dbus.Error                                  { s.sendCmd("stop"); return nil }
func (s *Server) Play() *dbus.Error                                  { s.sendCmd("play"); return nil }
func (s *Server) SetPosition(_ dbus.ObjectPath, _ int64) *dbus.Error { return nil }

func (s *Server) Seek(offset int64) *dbus.Error {
	s.sendCmd(fmt.Sprintf("seek:%d", offset))
	return nil
}

// ── Property updates (called from Bubble Tea goroutine) ──────────────

func (s *Server) SetProperty(iface, name string, v dbus.Variant) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.props[iface] == nil {
		s.props[iface] = make(map[string]dbus.Variant)
	}
	s.props[iface][name] = v

	if s.conn != nil {
		s.conn.Emit(
			dbus.ObjectPath(dbusPath),
			dbusProperties+".PropertiesChanged",
			iface,
			map[string]dbus.Variant{name: v},
			[]string{},
		)
	}
}

func (s *Server) UpdatePlaybackStatus(status string) {
	s.SetProperty(dbusPlayerIface, "PlaybackStatus", dbus.MakeVariant(status))
}

func (s *Server) UpdatePosition(positionUs int64) {
	s.SetProperty(dbusPlayerIface, "Position", dbus.MakeVariant(positionUs))
}

func (s *Server) UpdateMetadata(trackID, title, artist, album, artURL string, durationUs int64) {
	metadata := map[string]dbus.Variant{
		"mpris:trackid": dbus.MakeVariant("/org/mpris/MediaPlayer2/track/" + trackID),
	}
	if title != "" {
		metadata["xesam:title"] = dbus.MakeVariant(title)
	}
	if artist != "" {
		metadata["xesam:artist"] = dbus.MakeVariant([]string{artist})
	}
	if album != "" {
		metadata["xesam:album"] = dbus.MakeVariant(album)
	}
	if artURL != "" {
		metadata["mpris:artUrl"] = dbus.MakeVariant(artURL)
	}
	if durationUs > 0 {
		metadata["mpris:length"] = dbus.MakeVariant(durationUs)
	}

	s.SetProperty(dbusPlayerIface, "Metadata", dbus.MakeVariant(metadata))
}

// ── Bubble Tea integration ───────────────────────────────────────────

type AVRCPCmdMsg struct{ Cmd string }

func (s *Server) WatchCommands() tea.Cmd {
	return func() tea.Msg {
		cmd, ok := <-s.CmdCh
		if !ok {
			return nil
		}
		return AVRCPCmdMsg{Cmd: cmd}
	}
}

// Close releases the D-Bus name and closes the connection.
func (s *Server) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return
	}
	s.closed = true

	if s.conn != nil {
		s.conn.ReleaseName(dbusDest)
		s.conn.Close()
		s.conn = nil
	}
	close(s.CmdCh)
	log.Printf("[avrcp] closed")
}
