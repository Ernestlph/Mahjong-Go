package main

import (
	"fmt"
	"sort"
)

// contains checks if a slice contains a specific comparable value.
func contains[T comparable](slice []T, val T) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
}

// CountTiles counts tiles in a slice based on Suit and Value (ignores IsRed, ID).
// Returns a map where key is "Suit-Value" and value is the count.
func CountTiles(tiles []Tile) map[string]int {
	counts := make(map[string]int)
	for _, t := range tiles {
		key := fmt.Sprintf("%s-%d", t.Suit, t.Value)
		counts[key]++
	}
	return counts
}

// GetUniqueTiles returns a slice containing one representative tile for each unique Suit+Value in the input.
func GetUniqueTiles(tiles []Tile) []Tile {
	uniqueMap := make(map[string]Tile)
	for _, t := range tiles {
		key := fmt.Sprintf("%s-%d", t.Suit, t.Value)
		if _, exists := uniqueMap[key]; !exists {
			uniqueMap[key] = t // Store the first encountered tile of this type
		}
	}
	uniqueSlice := make([]Tile, 0, len(uniqueMap))
	for _, tile := range uniqueMap {
		uniqueSlice = append(uniqueSlice, tile)
	}
	sort.Sort(BySuitValue(uniqueSlice)) // Sort for predictable order
	return uniqueSlice
}

// HasTileWithValue checks if a slice of tiles contains at least one tile matching the suit and value.
func HasTileWithValue(tiles []Tile, suit string, value int) bool {
	for _, t := range tiles {
		if t.Suit == suit && t.Value == value {
			return true
		}
	}
	return false
}

// FindTileWithValue finds the first tile in the slice matching the suit and value.
// Returns the tile and true if found, otherwise an empty Tile and false.
func FindTileWithValue(tiles []Tile, suit string, value int) (Tile, bool) {
	for _, t := range tiles {
		if t.Suit == suit && t.Value == value {
			return t, true
		}
	}
	return Tile{}, false // Return zero Tile struct explicitly
}

// RemoveTilesByIndices removes tiles from a slice at the given indices.
// It modifies a *copy* of the slice and returns the new slice.
// IMPORTANT: Assumes indices are valid and relative to the *original* slice.
// Sorts indices descending internally for safe removal.
func RemoveTilesByIndices(tiles []Tile, indices []int) []Tile {
	if len(indices) == 0 {
		return tiles // Nothing to remove, return original slice (or copy if mutation is a concern later)
	}

	// Sort indices descending to remove from end without messing up lower indices
	sort.Sort(sort.Reverse(sort.IntSlice(indices)))

	newTiles := append([]Tile{}, tiles...) // Make a copy to avoid modifying the original slice directly

	for _, index := range indices {
		if index < 0 || index >= len(newTiles) {
			fmt.Printf("Warning: Invalid index %d provided to RemoveTilesByIndices (len %d)\n", index, len(newTiles))
			// Depending on severity, could panic or just continue
			continue // Skip invalid index
		}
		// Perform removal
		newTiles = append(newTiles[:index], newTiles[index+1:]...)
	}
	return newTiles
}

// If is a basic ternary conditional helper for strings.
func If(condition bool, trueVal, falseVal string) string {
	if condition {
		return trueVal
	}
	return falseVal
}

// IfElseInt is a basic ternary conditional helper for ints.
func IfElseInt(condition bool, trueVal, falseVal int) int {
	if condition {
		return trueVal
	}
	return falseVal
}
