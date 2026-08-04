package domain

type Track struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Artist    string `json:"artist"`
	Duration  int    `json:"duration"`
	Thumbnail string `json:"thumbnail"`
}

type LyricLine struct {
	Time float64 `json:"time"`
	End  float64 `json:"end"`
	Text string  `json:"text"`
}

type LyricsTrack struct {
	Language   string      `json:"language"`
	Synced     []LyricLine `json:"synced,omitempty"`
	Plain      string      `json:"plain,omitempty"`
	TrackName  string      `json:"trackName,omitempty"`
	ArtistName string      `json:"artistName,omitempty"`
}

type LyricsResp struct {
	Synced []LyricLine `json:"synced"`
	Plain  string      `json:"plain"`
	Lrclib *struct {
		Synced []LyricLine `json:"synced"`
	} `json:"lrclib,omitempty"`
	Artist    string        `json:"artist,omitempty"`
	Title     string        `json:"title,omitempty"`
	VideoID   string        `json:"video_id,omitempty"`
	Language  string        `json:"language,omitempty"`
	AllTracks []LyricsTrack `json:"all_tracks,omitempty"`
}

func (l *LyricsResp) Normalize() {
	if len(l.Synced) == 0 && l.Lrclib != nil && len(l.Lrclib.Synced) > 0 {
		l.Synced = l.Lrclib.Synced
	}
}

func (l *LyricsResp) SelectLanguage(i int) {
	if i < 0 || i >= len(l.AllTracks) {
		return
	}
	t := l.AllTracks[i]
	l.Synced = t.Synced
	l.Plain = t.Plain
	l.Language = t.Language
	if t.TrackName != "" {
		l.Title = t.TrackName
	}
	if t.ArtistName != "" {
		l.Artist = t.ArtistName
	}
}

func (l *LyricsResp) LanguageIndex() int {
	for i, t := range l.AllTracks {
		if t.Language == l.Language {
			return i
		}
	}
	return 0
}

type ServerLyricTrack struct {
	Language string `json:"language"`
	Lines    []struct {
		Start float64 `json:"start"`
		End   float64 `json:"end"`
		Text  string  `json:"text"`
	} `json:"lines"`
	TrackName  string `json:"trackName"`
	ArtistName string `json:"artistName"`
}
