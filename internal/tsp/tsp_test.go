package tsp

import (
	"math"
	"testing"
)

// ponytail: hits real Valhalla endpoint, skip in CI with -short
func TestMatrixJyllandSjaellandCrossing(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network-dependent test")
	}

	points := []Location{
		{55.697838, 12.453019, 100}, // J1
		{55.663697, 12.400465, 100}, // J2
		{55.654232, 12.277284, 100}, // J3
		{55.745842, 12.312660, 100}, // S1
		{55.941055, 12.341422, 100}, // S2
		{56.081821, 12.389189, 100}, // S3
	}

	matrix, err := getMatrixFromValhalla(points, "auto", "distance")
	if err != nil {
		t.Fatalf("getMatrixFromValhalla failed: %v", err)
	}

	if len(matrix) != len(points) {
		t.Fatalf("expected %d rows, got %d", len(points), len(matrix))
	}
	for i, row := range matrix {
		if len(row) != len(points) {
			t.Fatalf("row %d: expected %d cols, got %d", i, len(points), len(row))
		}
	}

	// Jylland-internal hops must be shorter than any Storebælt crossing.
	maxInternal := max(matrix[0][1], matrix[1][2])
	for i := 0; i < 3; i++ {
		for j := 3; j < 6; j++ {
			if matrix[i][j] <= maxInternal {
				t.Errorf("crossing %d->%d (%.0f) not longer than Jylland internal (%.0f)",
					i, j, matrix[i][j], maxInternal)
			}
		}
	}
}

func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

// symmetric distance matrix for a 5-city square-ish layout,
// small enough for brute force to check optimality.
func testMatrix() [][]float64 {
	return [][]float64{
		{-1, 10, 15, 20, 25},
		{10, -1, 35, 25, 30},
		{15, 35, -1, 30, 5},
		{20, 25, 30, -1, 15},
		{25, 30, 5, 15, -1},
	}
}

func TestHeldKarpMatchesBruteForce_FreeEnd(t *testing.T) {
	m := testMatrix()
	startIdx := 0

	hkOrder, hkCost, hkEnd := heldKarp(m, startIdx, -1)
	bfOrder, bfCost := bruteForceTSP(m, startIdx)

	t.Logf("heldKarp:    order=%v cost=%.2f end=%d", hkOrder, hkCost, hkEnd)
	t.Logf("bruteForce:  order=%v cost=%.2f", bfOrder, bfCost)

	if math.Abs(hkCost-bfCost) > 1e-9 {
		t.Fatalf("heldKarp cost %.2f != bruteForce cost %.2f", hkCost, bfCost)
	}
}

func TestHeldKarpTrivialTriangle(t *testing.T) {
	// 3 cities, straight line: 0-1-2, edges 1 and 1, diagonal 2.
	m := [][]float64{
		{0, 1, 2},
		{1, 0, 1},
		{2, 1, 0},
	}
	order, cost, end := heldKarp(m, 0, -1)
	t.Logf("order=%v cost=%.2f end=%d", order, cost, end)
	want := 2.0 // 0->1->2
	if math.Abs(cost-want) > 1e-9 {
		t.Fatalf("got cost %.2f want %.2f", cost, want)
	}
}

func TestHeldKarpRespectsFixedEnd(t *testing.T) {
	m := testMatrix()
	order, _, end := heldKarp(m, 0, 4)

	if end != 4 {
		t.Fatalf("expected fixed end 4, got %d", end)
	}
	if order[len(order)-1] != 4 {
		t.Fatalf("expected order to end at 4, got order %v", order)
	}
	if order[0] != 0 {
		t.Fatalf("expected order to start at 0, got order %v", order)
	}
}

func TestHeldKarpVisitsEveryCityOnce(t *testing.T) {
	m := testMatrix()
	order, _, _ := heldKarp(m, 0, -1)

	seen := make(map[int]bool)
	for _, idx := range order {
		if seen[idx] {
			t.Fatalf("city %d visited twice in order %v", idx, order)
		}
		seen[idx] = true
	}
	if len(seen) != len(m) {
		t.Fatalf("expected %d cities visited, got %d", len(m), len(seen))
	}
}

func TestTwoOptNeverMakesTourWorse(t *testing.T) {
	m := testMatrix()
	nnOrder := nearestNeighbor(m, 0, -1)
	before := calculateTourDistance(nnOrder, m)

	_, after := twoOpt(nnOrder, m, 0, -1, false)

	if after > before+1e-9 {
		t.Fatalf("2-opt made tour worse: before=%.2f after=%.2f", before, after)
	}
}

func TestNearestNeighborRespectsFixedEnd(t *testing.T) {
	m := testMatrix()
	order := nearestNeighbor(m, 0, 4)

	if order[0] != 0 || order[len(order)-1] != 4 {
		t.Fatalf("expected order to start at 0 and end at 4, got %v", order)
	}
}

func TestCalculateTourDistance(t *testing.T) {
	m := testMatrix()
	order := []int{0, 1, 2, 3, 4}
	got := calculateTourDistance(order, m)
	want := m[0][1] + m[1][2] + m[2][3] + m[3][4]

	if got != want {
		t.Fatalf("got %.2f want %.2f", got, want)
	}
}

func TestReverseMatrixAndOrder(t *testing.T) {
	m := testMatrix()
	rm := reverseMatrix(m)

	for i := range m {
		for j := range m[i] {
			if rm[i][j] != m[j][i] {
				t.Fatalf("reverseMatrix mismatch at [%d][%d]", i, j)
			}
		}
	}

	order := []int{0, 1, 2, 3}
	got := reverseOrder(order)
	want := []int{3, 2, 1, 0}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("reverseOrder: got %v want %v", got, want)
		}
	}
}

func TestCollapseConsecutiveAndExpand(t *testing.T) {
	wp := func(lat, lon float64) Waypoint { return Waypoint{Lat: lat, Lon: lon} }

	// two stops on top of each other, followed by a third further away
	waypoints := []Waypoint{wp(55.0, 12.0), wp(55.0, 12.0), wp(56.0, 13.0)}

	unique, reps := collapseConsecutive(waypoints)
	if len(unique) != 2 {
		t.Fatalf("want 2 unique waypoints, got %d", len(unique))
	}
	wantReps := []int{0, 0, 1}
	for i := range wantReps {
		if reps[i] != wantReps[i] {
			t.Fatalf("reps[%d]: want %d got %d", i, wantReps[i], reps[i])
		}
	}

	// collapsed route [A, B] gives one leg with real geometry + time
	geometry, legTimes := expandLegs(reps, []string{"g_AB"}, []float64{300})
	if len(geometry) != 2 || len(legTimes) != 2 {
		t.Fatalf("want 2 expanded legs, got %d/%d", len(geometry), len(legTimes))
	}
	// leg 0: A->A (co-located) -> empty/0, leg 1: A->B -> real
	if geometry[0] != "" || legTimes[0] != 0 {
		t.Fatalf("leg 0 must be zero, got %q / %.0f", geometry[0], legTimes[0])
	}
	if geometry[1] != "g_AB" || legTimes[1] != 300 {
		t.Fatalf("leg 1 must be real, got %q / %.0f", geometry[1], legTimes[1])
	}
}

func TestCollapseConsecutiveAllDuplicates(t *testing.T) {
	wp := func(lat, lon float64) Waypoint { return Waypoint{Lat: lat, Lon: lon} }
	waypoints := []Waypoint{wp(55.0, 12.0), wp(55.0, 12.0), wp(55.0, 12.0)}

	unique, reps := collapseConsecutive(waypoints)
	if len(unique) != 1 {
		t.Fatalf("want 1 unique waypoint, got %d", len(unique))
	}
	geometry, legTimes := expandLegs(reps, nil, nil)
	for i := range geometry {
		if geometry[i] != "" || legTimes[i] != 0 {
			t.Fatalf("leg %d must be zero, got %q / %.0f", i, geometry[i], legTimes[i])
		}
	}
}

func TestCoLocatedMatrixNotUnreachable(t *testing.T) {
	// replicate the matrix-cell decision: co-located pair is 0, not 9999999
	a := Location{Lat: 55.0, Lon: 12.0}
	b := Location{Lat: 55.0, Lon: 12.0}
	if !coLocated(a, b) {
		t.Fatal("coLocated should be true for identical locations")
	}
	c := Location{Lat: 56.0, Lon: 13.0}
	if coLocated(a, c) {
		t.Fatal("coLocated should be false for distant locations")
	}
}

func TestHasUnreachablePair(t *testing.T) {
	ok := [][]float64{{0, 5}, {5, 0}}
	if hasUnreachablePair(ok) {
		t.Fatal("expected reachable matrix to pass")
	}
	bad := [][]float64{{0, -0.0}, {5, 0}}
	if !hasUnreachablePair(bad) {
		t.Fatal("expected unreachable cell to be detected")
	}
}
