package entity_test

import (
	"testing"

	"github.com/chunisupport/chunisupport-song-batch/internal/domain/entity"
)

func TestNewCourse(t *testing.T) {
	course, err := entity.NewCourse(" 50000 ", " First Step ", " 1 ")
	if err != nil {
		t.Fatalf("NewCourse() error = %v", err)
	}

	if course.DisplayID.IsEmpty() {
		t.Error("DisplayID is empty")
	}
	if course.OfficialIdx != "50000" {
		t.Errorf("OfficialIdx = %q, want %q", course.OfficialIdx, "50000")
	}
	if course.Name != "First Step" {
		t.Errorf("Name = %q, want %q", course.Name, "First Step")
	}
	if course.ClassName != "1" {
		t.Errorf("ClassName = %q, want %q", course.ClassName, "1")
	}
}
