package importer

// AdditionalSongsData は追加楽曲データのコレクションです
type AdditionalSongsData struct {
	Songs    []AdditionalSong    `json:"songs"`
	Charts   []AdditionalChart   `json:"charts"`
	WECharts []AdditionalWEChart `json:"we_charts"`
}

// AdditionalSong は追加楽曲シート(additional_songs)の1行を表します
// シートカラム: id, title, artist, genre, release, bpm, basnt, bas, basuk, advnt, adv, advuk,
//
//	expnt, exp, expuk, masnt, mas, masuk, ultnt, ult, ultuk, img
type AdditionalSong struct {
	ID      string  `json:"id"`      // 楽曲ID (official_idx)
	Title   string  `json:"title"`   // タイトル
	Artist  string  `json:"artist"`  // アーティスト
	Genre   string  `json:"genre"`   // ジャンル
	Release string  `json:"release"` // リリース日 (YYYY/MM/DD)
	BPM     *int    `json:"bpm"`     // BPM (nilの場合は未設定)
	BasNt   *int    `json:"basnt"`   // BASICノーツ数
	Bas     float64 `json:"bas"`     // BASIC定数
	BasUK   bool    `json:"basuk"`   // BASIC定数不明フラグ
	AdvNt   *int    `json:"advnt"`   // ADVANCEDノーツ数
	Adv     float64 `json:"adv"`     // ADVANCED定数
	AdvUK   bool    `json:"advuk"`   // ADVANCED定数不明フラグ
	ExpNt   *int    `json:"expnt"`   // EXPERTノーツ数
	Exp     float64 `json:"exp"`     // EXPERT定数
	ExpUK   bool    `json:"expuk"`   // EXPERT定数不明フラグ
	MasNt   *int    `json:"masnt"`   // MASTERノーツ数
	Mas     float64 `json:"mas"`     // MASTER定数
	MasUK   bool    `json:"masuk"`   // MASTER定数不明フラグ
	UltNt   *int    `json:"ultnt"`   // ULTIMAノーツ数
	Ult     float64 `json:"ult"`     // ULTIMA定数 (0の場合は譜面なし)
	UltUK   bool    `json:"ultuk"`   // ULTIMA定数不明フラグ
	Img     string  `json:"img"`     // 画像ハッシュ
}

// AdditionalChart は追加譜面シート(additional_charts)の1行を表します
// シートカラム: id, diff, const, csuk, notes
// 既存楽曲に追加されるULTIMA譜面専用
type AdditionalChart struct {
	ID    string  `json:"id"`    // 楽曲ID (official_idx への外部キー)
	Diff  string  `json:"diff"`  // 難易度 (基本的にULTIMAのみ)
	Const float64 `json:"const"` // 定数
	CsUK  bool    `json:"csuk"`  // 定数不明フラグ
	Notes *int    `json:"notes"` // ノーツ数 (nilの場合は未設定)
}

// AdditionalWEChart は追加WORLD'S END譜面シート(additional_songs_charts_we)の1行を表します
// シートカラム: id, title, artist, genre, release, we_kanji, we_star, notes, img
// WORLD'S END楽曲は1曲1譜面のため、楽曲情報と譜面情報が一体化しています
type AdditionalWEChart struct {
	ID      string `json:"id"`       // 楽曲ID (unique識別子)
	Title   string `json:"title"`    // タイトル
	Artist  string `json:"artist"`   // アーティスト
	Genre   string `json:"genre"`    // ジャンル
	Release string `json:"release"`  // リリース日 (YYYY/MM/DD)
	WEKanji string `json:"we_kanji"` // カテゴリ漢字（光、蔵、改、狂、etc.）
	WEStar  *int   `json:"we_star"`  // 星の数（1～5）
	Notes   *int   `json:"notes"`    // ノーツ数 (nilの場合は未設定)
	Img     string `json:"img"`      // 画像ハッシュ
}
