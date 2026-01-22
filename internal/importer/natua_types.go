package importer

// NatuaData はnatua.devの楽曲データを表します
type NatuaData struct {
	Songs []NatuaSong `json:"songs"`
}

// NatuaSong はnatua.devの単一の楽曲データを表します
type NatuaSong struct {
	Meta     NatuaMeta  `json:"meta"`
	Basic    NatuaChart `json:"basic"`
	Advanced NatuaChart `json:"advanced"`
	Expert   NatuaChart `json:"expert"`
	Master   NatuaChart `json:"master"`
	Ultima   NatuaChart `json:"ultima"`
}

// NatuaMeta はnatua.devの楽曲のメタデータを表します
type NatuaMeta struct {
	OfficialID     string  `json:"official_id"`
	Name           string  `json:"name"`
	Reading        *string `json:"reading"` // nullable
	Artist         string  `json:"artist"`
	Genre          string  `json:"genre"`
	BPM            *int    `json:"bpm"` // nullable
	BPMNodata      bool    `json:"bpm_nodata"`
	Release        string  `json:"release"`
	ReleaseVersion string  `json:"release_version"`
	ImageURL       string  `json:"image_url"`
	FumenID        string  `json:"fumen_id"`
	IsWorldsend    bool    `json:"is_worldsend"`
	HasUltima      bool    `json:"has_ultima"`
	OnlyUltima     bool    `json:"only_ultima"`
	WeStar         *string `json:"we_star"`  // nullable
	WeKanji        *string `json:"we_kanji"` // nullable
}

// NatuaChart はnatua.devの単一の譜面データを表します
type NatuaChart struct {
	Const               *float64 `json:"const"` // nullable
	ConstNodata         bool     `json:"const_nodata"`
	Notes               *int     `json:"notes"` // nullable
	NotesNodata         bool     `json:"notes_nodata"`
	Notesdesigner       *string  `json:"notesdesigner"` // nullable
	NotesdesignerNodata bool     `json:"notesdesigner_nodata"`
}
