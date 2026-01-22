package importer

import "bytes"

// removeBOM はByte Order Mark (BOM)をデータから除去します。
// UTF-8 BOM (EF BB BF) を検出して削除します。
func removeBOM(data []byte) []byte {
	// UTF-8 BOM: 0xEF, 0xBB, 0xBF
	if len(data) >= 3 && data[0] == 0xEF && data[1] == 0xBB && data[2] == 0xBF {
		return data[3:]
	}
	// UTF-16 BE BOM: 0xFE, 0xFF
	if len(data) >= 2 && data[0] == 0xFE && data[1] == 0xFF {
		return data[2:]
	}
	// UTF-16 LE BOM: 0xFF, 0xFE
	if len(data) >= 2 && data[0] == 0xFF && data[1] == 0xFE {
		return data[2:]
	}
	return data
}

// trimBOM はバイトスライスの先頭からBOMを除去します（bytes.Trimを使用）
func trimBOM(data []byte) []byte {
	return bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF})
}
