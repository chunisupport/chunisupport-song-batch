package entity

import (
	"strings"

	vo "chunisupport-song-batch/internal/domain/valueobject"
	"chunisupport-song-batch/internal/importer"
)

// NewSongFromOfficial は公式データからSongエンティティを生成します
func NewSongFromOfficial(official *importer.OfficialSong, genreID int) (*Song, error) {
	officialIdx, err := vo.NewOfficialIdx(official.ID)
	if err != nil {
		return nil, err
	}

	displayID, err := vo.NewDisplayID()
	if err != nil {
		return nil, err
	}

	// WORLD'S END判定はエンティティ生成時に行う
	isWorldsEnd := strings.TrimSpace(official.WeKanji) != "" ||
		strings.TrimSpace(official.WeStar) != ""

	song := NewSong(
		displayID,
		strings.TrimSpace(official.Title),
		strings.TrimSpace(official.Artist),
		genreID,
		officialIdx,
		isWorldsEnd,
	)

	// ジャケット画像を設定
	jacket := vo.NewJacketImage(official.Image)
	song.SetJacket(jacket)

	return song, nil
}

// NewSongFromAdditional は追加楽曲データからSongエンティティを生成します
func NewSongFromAdditional(additional *importer.AdditionalSong, genreID int) (*Song, error) {
	officialIdx, err := vo.NewOfficialIdx(additional.ID)
	if err != nil {
		return nil, err
	}

	displayID, err := vo.NewDisplayID()
	if err != nil {
		return nil, err
	}

	song := NewSong(
		displayID,
		strings.TrimSpace(additional.Title),
		strings.TrimSpace(additional.Artist),
		genreID,
		officialIdx,
		false, // 追加楽曲はWorld's Endではない
	)

	// ジャケット画像を設定
	jacket := vo.NewJacketImage(additional.Img)
	song.SetJacket(jacket)

	// BPMを設定
	if additional.BPM != nil && *additional.BPM > 0 {
		song.SetBPM(*additional.BPM)
	}

	// リリース日を設定
	if additional.Release != "" {
		releasedAt, err := vo.ParseReleaseDate(additional.Release)
		if err == nil {
			song.SetReleasedAt(releasedAt)
		}
	}

	return song, nil
}

// DetermineIsWorldsEnd は公式データからWORLD'S END楽曲かどうかを判定します
func DetermineIsWorldsEnd(official *importer.OfficialSong) bool {
	return strings.TrimSpace(official.WeKanji) != "" ||
		strings.TrimSpace(official.WeStar) != ""
}
