package importer

// MainframeChartData はmainframeデータソースの単一の譜面エントリを表します
type MainframeChartData struct {
	Title string  `json:"title"`
	Diff  string  `json:"diff"`
	Genre string  `json:"genre"`
	Const float64 `json:"const"`
}

// MainframeData はmainframeデータソースの譜面データのコレクションです
type MainframeData []MainframeChartData
