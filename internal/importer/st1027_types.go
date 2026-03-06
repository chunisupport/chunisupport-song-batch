package importer

// St1027Data はst1027の楽曲データ全体を表します
type St1027Data struct {
	Songs []St1027Song `json:"songs"`
}

// St1027Song はst1027の単一の楽曲データを表します
type St1027Song struct {
	Meta     St1027Meta  `json:"meta"`
	Basic    St1027Chart `json:"BAS"`
	Advanced St1027Chart `json:"ADV"`
	Expert   St1027Chart `json:"EXP"`
	Master   St1027Chart `json:"MAS"`
	Ultima   St1027Chart `json:"ULT"`
}

// St1027Meta はst1027の楽曲のメタデータを表します（ノーツ数補完に必要な最小限のフィールド）
type St1027Meta struct {
	OfficialID string `json:"official_id"`
}

// St1027Chart はst1027の単一の譜面データを表します（ノーツ数のみ参照）
type St1027Chart struct {
	NotesAll *int `json:"notes_all"`
}
