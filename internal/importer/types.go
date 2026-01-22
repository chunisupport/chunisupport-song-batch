package importer

// MusicData は単一の楽曲データを表します
type MusicData struct {
	ID      string `json:"id"`
	Catname string `json:"catname"`
	Newflag string `json:"newflag"`
	Title   string `json:"title"`
	Reading string `json:"reading"`
	Artist  string `json:"artist"`
	LevBas  string `json:"lev_bas"`
	LevAdv  string `json:"lev_adv"`
	LevExp  string `json:"lev_exp"`
	LevMas  string `json:"lev_mas"`
	LevUlt  string `json:"lev_ult"`
	WeKanji string `json:"we_kanji"`
	WeStar  string `json:"we_star"`
	Image   string `json:"image"`
}

// MusicCollection は楽曲データのコレクションです
type MusicCollection []MusicData
