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

func TestComputeArrivalTimes(t *testing.T) {
	cases := []struct {
		name     string
		start    string
		service  uint
		end      string
		anchor   string
		legTimes []float64
		want     []string
		overrun  bool
	}{
		{
			name:  "forward from start",
			start: "13:00", service: 15, end: "20:00", anchor: "start",
			legTimes: []float64{600}, // 10 min travel
			want:     []string{"13:00", "13:25"},
		},
		{
			name:  "forward keeps start, overrun when route too long",
			start: "13:00", service: 60, end: "14:00", anchor: "start",
			legTimes: []float64{3600}, // 1h travel
			want:     []string{"13:00", "15:00"},
			overrun:  true,
		},
		{
			name:  "backward from end",
			start: "13:00", service: 15, end: "20:00", anchor: "end",
			legTimes: []float64{600},
			// last lands on 20:00, prev = 20:00 - 15min service - 10min travel
			want: []string{"19:35", "20:00"},
		},
		{
			name:  "backward overrun when start would be before start_time",
			start: "19:00", service: 60, end: "20:00", anchor: "end",
			legTimes: []float64{3600}, // 1h travel -> begin at 18:00
			want:     []string{"18:00", "20:00"},
			overrun:  true,
		},
		{
			name:  "backward without end time falls back to forward",
			start: "13:00", service: 15, end: "", anchor: "end",
			legTimes: []float64{600},
			want:     []string{"13:00", "13:25"},
		},
		{
			name:  "no travel between stops",
			start: "13:00", service: 15, end: "", anchor: "start",
			legTimes: []float64{0},
			want:     []string{"13:00", "13:15"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, overrun := computeArrivalTimes(tc.start, tc.service, tc.end, tc.anchor, tc.legTimes)
			if overrun != tc.overrun {
				t.Fatalf("overrun: want %v got %v", tc.overrun, overrun)
			}
			if len(got) != len(tc.want) {
				t.Fatalf("len: want %v got %v", tc.want, got)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("times: want %v got %v", tc.want, got)
				}
			}
		})
	}
}

func TestGroupSegments(t *testing.T) {
	visits := []models.Visit{
		seg(1, 1, 0), seg(2, 2, 0), seg(4, 3, 0),
		seg(6, 4, 1), seg(8, 5, 1), seg(10, 6, 1),
		seg(3, 7, 2), seg(5, 8, 2), seg(7, 9, 2),
		seg(9, 10, 3), // own segment, size 1, locked final stop
	}

	segs := groupSegments(visits, true)

	if len(segs) != 4 {
		t.Fatalf("want 4 segments, got %d", len(segs))
	}
	wantSizes := []int{3, 3, 3, 1}
	for i, s := range segs {
		if len(s.visits) != wantSizes[i] {
			t.Fatalf("segment %d: want %d visits, got %d", i, wantSizes[i], len(s.visits))
		}
		if s.isLast != (i == len(segs)-1) {
			t.Fatalf("segment %d: isLast=%v", i, s.isLast)
		}
	}
}

func TestGroupSegmentsNilIndexCountsAsZero(t *testing.T) {
	// nil segment_index == 0, so stop 1 stays in the same segment as stop 2
	visits := []models.Visit{
		{Stopnr: uptr(1)}, seg(2, 2, 0),
	}
	segs := groupSegments(visits, true)
	if len(segs) != 1 {
		t.Fatalf("want 1 segment, got %d", len(segs))
	}
	if len(segs[0].visits) != 2 {
		t.Fatalf("want 2 visits in segment, got %d", len(segs[0].visits))
	}
}

func TestGroupSegmentsUnsortedInput(t *testing.T) {
	visits := []models.Visit{
		seg(9, 10, 3), seg(1, 1, 0), seg(6, 4, 1),
		seg(8, 5, 1), seg(10, 6, 1), seg(3, 7, 2),
		seg(2, 2, 0), seg(4, 3, 0), seg(5, 8, 2), seg(7, 9, 2),
	}
	segs := groupSegments(visits, true)
	// same grouping as the design doc example: 3,3,3,1
	wantSizes := []int{3, 3, 3, 1}
	if len(segs) != len(wantSizes) {
		t.Fatalf("want %d segments, got %d", len(wantSizes), len(segs))
	}
	for i, s := range segs {
		if len(s.visits) != wantSizes[i] {
			t.Fatalf("segment %d: want %d visits, got %d", i, wantSizes[i], len(s.visits))
		}
	}
}
