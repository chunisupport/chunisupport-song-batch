package importer

// OtogeDbSong はotoge-dbデータソースの単一の楽曲エントリを表します
type OtogeDbSong struct {
	ID            string `json:"id"`              // 楽曲ID (文字列)
	Title         string `json:"title"`           // 楽曲タイトル
	DateAdded     string `json:"date_added"`      // 追加日 (YYYYMMDD形式)
	Version       string `json:"version"`         // バージョン情報
	Catname       string `json:"catname"`         // カテゴリ名
	Artist        string `json:"artist"`          // アーティスト
	Image         string `json:"image"`           // 画像ファイル名
	BPM           string `json:"bpm"`             // BPM
	WeKanji       string `json:"we_kanji"`        // WORLD'S ENDカテゴリ漢字
	WeStar        string `json:"we_star"`         // WORLD'S END星数
	LevWENotes    string `json:"lev_we_notes"`    // WORLD'S END ノーツ数
	LevWEDesigner string `json:"lev_we_designer"` // WORLD'S END 譜面製作者
}

// OtogeDbData はotoge-dbデータソースの楽曲データのコレクションです
type OtogeDbData []OtogeDbSong
