package api

import (
	"testing"

	"github.com/MOPDev/mop-backend-api/models"
)

func seg(id, stop, segment uint) models.Visit {
	s, n := segment, stop
	return models.Visit{Stopnr: &n, SegmentIndex: &s}
}

func TestAssertSegmentsValid(t *testing.T) {
	cases := []struct {
		name   string
		visits []models.Visit
		ok     bool
	}{
		{"empty", nil, true},
		{"single stop", []models.Visit{seg(1, 1, 0)}, true},
		{"spec example", []models.Visit{
			seg(1, 1, 0), seg(2, 2, 0), seg(4, 3, 0),
			seg(6, 4, 1), seg(8, 5, 1), seg(10, 6, 1),
			seg(3, 7, 2), seg(5, 8, 2), seg(7, 9, 2),
			seg(9, 10, 3),
		}, true},
		{"unsorted input is sorted first", []models.Visit{
			seg(9, 10, 3), seg(1, 1, 0), seg(6, 4, 1), seg(2, 2, 0),
			seg(3, 7, 2), seg(4, 3, 0), seg(8, 5, 1), seg(5, 8, 2),
			seg(10, 6, 1), seg(7, 9, 2),
		}, true},
		{"does not start at 0", []models.Visit{
			seg(1, 1, 1), seg(2, 2, 1),
		}, false},
		{"gap in sequence", []models.Visit{
			seg(1, 1, 0), seg(2, 2, 0), seg(3, 3, 2),
		}, false},
		{"segment not contiguous", []models.Visit{
			seg(1, 1, 0), seg(2, 2, 1), seg(3, 3, 0),
		}, false},
		{"segment reused later", []models.Visit{
			seg(1, 1, 0), seg(2, 2, 1), seg(3, 3, 2), seg(4, 4, 1),
		}, false},
		{"join in the middle of a segment", []models.Visit{
			// visit 5 joined the segment before it, cutting segment 2 in two
			seg(6, 4, 1), seg(8, 5, 1), seg(10, 6, 1),
			seg(3, 7, 2), seg(5, 8, 1), seg(7, 9, 2),
		}, false},
		{"nil segment_index counts as 0", []models.Visit{
			{Stopnr: uptr(1)}, seg(2, 2, 0),
		}, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := assertSegmentsValid(tc.visits)
			if tc.ok && err != nil {
				t.Fatalf("want valid, got %v", err)
			}
			if !tc.ok && err == nil {
				t.Fatal("want error, got nil")
			}
		})
	}
}

func uptr(v uint) *uint { return &v }
