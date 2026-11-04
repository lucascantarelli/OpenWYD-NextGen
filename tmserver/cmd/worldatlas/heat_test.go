package main

import "testing"

// TestSpawnHeat: buckets count per 64² cell, the peak drives the colour ramp and
// generators outside the world are dropped instead of skewing the scale.
func TestSpawnHeat(t *testing.T) {
	spawns := []spawn{
		{X: 0, Y: 0},
		{X: 10, Y: 10}, // same cell as above
		{X: 4000, Y: 4000},
		{X: 5000, Y: 0}, // outside the world
		{X: -1, Y: 100}, // outside the world
	}

	grid := spawnHeat(spawns, 64)
	if grid.Cols != 64 || grid.Rows != 64 || len(grid.Data) != 64*64 {
		t.Fatalf("grid = %dx%d with %d cells, want 64x64 with 4096", grid.Cols, grid.Rows, len(grid.Data))
	}
	sum := 0
	for _, value := range grid.Data {
		sum += value
	}
	if sum != 3 {
		t.Errorf("sum = %d, want 3 (out-of-world generators dropped)", sum)
	}
	if grid.Max != 2 {
		t.Errorf("Max = %d, want 2", grid.Max)
	}
	if grid.Data[0] != 2 {
		t.Errorf("cell (0,0) = %d, want 2", grid.Data[0])
	}
	if got := grid.Data[62*64+62]; got != 1 {
		t.Errorf("cell (62,62) = %d, want 1", got)
	}
}
