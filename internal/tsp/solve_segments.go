package tsp

import "math"

type pairKey struct{ entry, exit int }

type segEntry struct {
	cost  float64
	order []int // global indices for this segment (includes entry and exit)
}

// SolveSegmentedWaypoints solves a segmented open TSP over all waypoints.
// All waypoints are in a single list; segmentSizes[i] tells how many
// waypoints belong to segment i. Visits within a segment can be freely
// reordered but segment order is fixed. The first segment's entry and the
// last segment's exit are free.
func SolveSegmentedWaypoints(waypoints []Waypoint, segmentSizes []int, costing, mode string) ([]Waypoint, error) {
	matrix, err := BuildMatrix(waypoints, costing, mode)
	if err != nil {
		return nil, err
	}

	// Build global index ranges per segment
	segIdx := make([][]int, len(segmentSizes))
	offset := 0
	for i, sz := range segmentSizes {
		idx := make([]int, sz)
		for j := 0; j < sz; j++ {
			idx[j] = offset + j
		}
		segIdx[i] = idx
		offset += sz
	}

	order, _ := SolveSegmentedTSP(matrix, segIdx)

	ordered := make([]Waypoint, len(waypoints))
	for i, idx := range order {
		ordered[i] = waypoints[idx]
	}
	return ordered, nil
}

// SolveSegmentedTSP solves a segmented open TSP.
//
// All visits within a segment can be freely reordered, but the segment
// order is fixed. The first segment's entry point is free and the last
// segment's exit point is free (open start, open end).
//
// For small segments (<= 10) every possible (entry, exit) pair is
// evaluated exactly via Held-Karp. For larger segments the free-start/
// free-end solution (nearest-neighbour + 2-opt) is used in both
// directions as two candidate (entry, exit) pairs.
func SolveSegmentedTSP(matrix [][]float64, segIdx [][]int) ([]int, float64) {
	nSegs := len(segIdx)

	type segSolMap map[pairKey]segEntry
	sols := make([]segSolMap, nSegs)
	for i := 0; i < nSegs; i++ {
		sols[i] = computeSegmentSolutions(matrix, segIdx[i])
	}

	// DP across segments
	type dpState struct {
		cost                         float64
		prevSegExitLocal, entryLocal int
		firstSegEntryLocal           int
	}
	dp := make([]map[int]dpState, nSegs)
	for i := range dp {
		dp[i] = make(map[int]dpState)
	}

	// Segment 0: free entry, any exit
	for key, sol := range sols[0] {
		exitLocal := key.exit
		existing, seen := dp[0][exitLocal]
		if !seen || sol.cost < existing.cost {
			dp[0][exitLocal] = dpState{
				cost:               sol.cost,
				firstSegEntryLocal: key.entry,
			}
		}
	}

	// Subsequent segments: connect prev exit -> this entry + internal cost
	for i := 1; i < nSegs; i++ {
		prevGlobalToGlobal := segIdx[i-1]
		thisGlobal := segIdx[i]

		for key, sol := range sols[i] {
			entryLocal := key.entry
			exitLocal := key.exit
			entryGlobal := thisGlobal[entryLocal]

			bestCost := math.Inf(1)
			bestPrevExitLocal := -1

			for prevExitLocal, prevState := range dp[i-1] {
				prevExitGlobal := prevGlobalToGlobal[prevExitLocal]
				connCost := matrix[prevExitGlobal][entryGlobal]
				total := prevState.cost + connCost + sol.cost
				if total < bestCost {
					bestCost = total
					bestPrevExitLocal = prevExitLocal
				}
			}

			if bestPrevExitLocal >= 0 {
				existing, seen := dp[i][exitLocal]
				if !seen || bestCost < existing.cost {
					dp[i][exitLocal] = dpState{
						cost:             bestCost,
						prevSegExitLocal: bestPrevExitLocal,
						entryLocal:       entryLocal,
					}
				}
			}
		}
	}

	// Find best final exit (free end for last segment)
	lastSegIdx := nSegs - 1
	bestExitLocal := -1
	bestTotal := math.Inf(1)
	for exitLocal, state := range dp[lastSegIdx] {
		if state.cost < bestTotal {
			bestTotal = state.cost
			bestExitLocal = exitLocal
		}
	}

	if bestExitLocal < 0 {
		order := make([]int, 0)
		for _, seg := range segIdx {
			order = append(order, seg...)
		}
		return order, math.Inf(1)
	}

	// Backtrack
	type segChoice struct{ entryLocal, exitLocal int }
	choices := make([]segChoice, nSegs)
	curExitLocal := bestExitLocal
	for i := nSegs - 1; i >= 0; i-- {
		state := dp[i][curExitLocal]
		if i == 0 {
			choices[i] = segChoice{entryLocal: state.firstSegEntryLocal, exitLocal: curExitLocal}
		} else {
			choices[i] = segChoice{entryLocal: state.entryLocal, exitLocal: curExitLocal}
			curExitLocal = state.prevSegExitLocal
		}
	}

	// Build final order
	order := make([]int, 0, len(matrix))
	for i := 0; i < nSegs; i++ {
		key := pairKey{entry: choices[i].entryLocal, exit: choices[i].exitLocal}
		sol := sols[i][key]
		order = append(order, sol.order...)
	}
	return order, bestTotal
}

// computeSegmentSolutions returns the best path for every (entry, exit) pair
// within the segment. For segments <= 10, all pairs are solved exactly via
// Held-Karp. For larger segments, the free-start/free-end solution is used
// (nearest-neighbour + 2-opt) in both directions.
func computeSegmentSolutions(matrix [][]float64, globalIdxs []int) map[pairKey]segEntry {
	s := len(globalIdxs)
	sols := make(map[pairKey]segEntry)

	if s == 1 {
		sols[pairKey{entry: 0, exit: 0}] = segEntry{
			cost:  0,
			order: []int{globalIdxs[0]},
		}
		return sols
	}

	sub := make([][]float64, s)
	for i := 0; i < s; i++ {
		sub[i] = make([]float64, s)
		for j := 0; j < s; j++ {
			sub[i][j] = matrix[globalIdxs[i]][globalIdxs[j]]
		}
	}

	if s <= 20 {
		for entry := 0; entry < s; entry++ {
			for exit := 0; exit < s; exit++ {
				if entry == exit {
					continue
				}
				path, cost, _ := heldKarp(sub, entry, exit)
				globalOrder := make([]int, s)
				for k, idx := range path {
					globalOrder[k] = globalIdxs[idx]
				}
				sols[pairKey{entry, exit}] = segEntry{
					cost:  cost,
					order: globalOrder,
				}
			}
		}
	} else {
		order := solveOrder(sub, false, false)
		cost := pathCost(sub, order)

		globalOrder := make([]int, s)
		for k, idx := range order {
			globalOrder[k] = globalIdxs[idx]
		}
		entry := order[0]
		exit := order[s-1]

		sols[pairKey{entry, exit}] = segEntry{cost: cost, order: globalOrder}

		// Also offer reverse direction
		revOrder := make([]int, s)
		for k := 0; k < s; k++ {
			revOrder[k] = globalOrder[s-1-k]
		}
		revCost := pathCost(sub, revOrder)
		sols[pairKey{exit, entry}] = segEntry{cost: revCost, order: revOrder}
	}
	return sols
}

// pathCost calculates the total cost of following the given order in the matrix.
func pathCost(matrix [][]float64, order []int) float64 {
	cost := 0.0
	for i := 1; i < len(order); i++ {
		cost += matrix[order[i-1]][order[i]]
	}
	return cost
}
