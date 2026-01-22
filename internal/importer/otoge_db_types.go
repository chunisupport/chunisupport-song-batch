package importer

// OtogeDbSong はotoge-dbデータソースの単一の楽曲エントリを表します
type OtogeDbSong struct {
	ID        string `json:"id"`         // 楽曲ID (文字列)
	Title     string `json:"title"`      // 楽曲タイトル
	DateAdded string `json:"date_added"` // 追加日 (YYYYMMDD形式)
	Version   string `json:"version"`    // バージョン情報
	Catname   string `json:"catname"`    // カテゴリ名
	Artist    string `json:"artist"`     // アーティスト
	Image     string `json:"image"`      // 画像ファイル名
}

// OtogeDbData はotoge-dbデータソースの楽曲データのコレクションです
type OtogeDbData []OtogeDbSong
