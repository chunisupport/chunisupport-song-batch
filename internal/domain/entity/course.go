package entity

import (
	"strings"

	vo "github.com/chunisupport/chunisupport-song-batch/internal/domain/valueobject"
)

// Course はコースマスタのエンティティです。
type Course struct {
	DisplayID   vo.DisplayID
	OfficialIdx string
	Name        string
	ClassName   string
}

// NewCourse はコースマスタのエンティティを生成します。
func NewCourse(officialIdx, name, className string) (Course, error) {
	displayID, err := vo.NewDisplayID()
	if err != nil {
		return Course{}, err
	}

	return Course{
		DisplayID:   displayID,
		OfficialIdx: strings.TrimSpace(officialIdx),
		Name:        strings.TrimSpace(name),
		ClassName:   strings.TrimSpace(className),
	}, nil
}
