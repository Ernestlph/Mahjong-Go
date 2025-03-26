package main

import (
	"fmt"
	"sort"
)

// GroupType represents the type of a group in a decomposed hand.
type GroupType int

const (
	TypeSequence GroupType = iota // Chi
	TypeTriplet                   // Pung / Ankou
	TypeQuad                      // Kan (Ankan, Daiminkan, Shouminkan)
	TypePair
)

// DecomposedGroup represents one component (group or pair) of a hand.
type DecomposedGroup struct {
	Type        GroupType
	Tiles       []Tile
	IsConcealed bool // Relevant for Triplets/Quads. True if Ankou/Ankan.
}

// DecomposeWinningHand attempts to break down a 14-tile completed hand into 4 groups and 1 pair.
// It considers existing melds to determine concealment and types.
// Returns the list of 5 DecomposedGroup structs and a boolean indicating success.
// Returns nil, false if the hand isn't a standard 4-group, 1-pair shape (e.g., Kokushi, Chiitoitsu, or invalid).
func DecomposeWinningHand(player *Player, allWinningTiles []Tile) ([]DecomposedGroup, bool) {
	if len(allWinningTiles) != 14 {
		// fmt.Printf("Debug: DecomposeWinningHand called with %d tiles, expected 14.\n", len(allWinningTiles))
		// Handle special hands explicitly before decomposition if needed (Chiitoitsu, Kokushi)
		if IsChiitoitsu(allWinningTiles) || IsKokushiMusou(allWinningTiles) {
			return nil, false // These don't use standard decomposition
		}
		// Otherwise, likely an error state if not 14 tiles for a standard hand.
		// Allow proceeding cautiously, maybe it's an intermediate check.
	}

	// 1. Separate melded groups from potential hand tiles
	meldedGroups := []DecomposedGroup{}
	handTilesForDecomp := []Tile{}
	meldTileIDs := make(map[int]bool) // Track IDs used in melds

	for _, meld := range player.Melds {
		group := DecomposedGroup{Tiles: meld.Tiles, IsConcealed: meld.IsConcealed}
		switch meld.Type {
		case "Chi":
			group.Type = TypeSequence
		case "Pon":
			group.Type = TypeTriplet // Concealment already set from meld
		case "Ankan", "Daiminkan", "Shouminkan":
			group.Type = TypeQuad // Concealment already set from meld
		}
		meldedGroups = append(meldedGroups, group)
		for _, t := range meld.Tiles {
			meldTileIDs[t.ID] = true
		}
	}

	// Add remaining tiles from the full 14-tile set that weren't in melds
	for _, tile := range allWinningTiles {
		if !meldTileIDs[tile.ID] {
			handTilesForDecomp = append(handTilesForDecomp, tile)
		}
	}
	sort.Sort(BySuitValue(handTilesForDecomp)) // MUST sort remaining tiles

	groupsNeeded := 4 - len(meldedGroups)
	pairsNeeded := 1

	// Expected number of tiles left to decompose in hand
	expectedHandTiles := groupsNeeded*3 + pairsNeeded*2
	if len(handTilesForDecomp) != expectedHandTiles {
		fmt.Printf("Warning: Mismatch in tiles for hand decomposition. Hand has %d, expected %d for %d groups, %d pair.\n",
			len(handTilesForDecomp), expectedHandTiles, groupsNeeded, pairsNeeded)
		// This often indicates an earlier error or unsupported hand type.
		return nil, false // Cannot decompose if tile counts don't match
	}

	// 2. Recursively decompose the remaining hand tiles
	handComponents, success := decomposeRecursive(handTilesForDecomp, groupsNeeded, pairsNeeded)

	if !success {
		fmt.Println("Debug: Recursive decomposition failed.")
		return nil, false
	}

	// 3. Combine melded groups and decomposed hand components
	allComponents := append(meldedGroups, handComponents...)

	// Final validation: Should have exactly 5 components (4 groups + 1 pair)
	if len(allComponents) != 5 {
		fmt.Printf("Warning: Decomposition resulted in %d components, expected 5.\n", len(allComponents))
		return nil, false
	}
	numGroups := 0
	numPairs := 0
	for _, comp := range allComponents {
		if comp.Type == TypePair {
			numPairs++
		} else {
			numGroups++
		}
	}
	if numGroups != 4 || numPairs != 1 {
		fmt.Printf("Warning: Decomposition resulted in %d groups and %d pairs, expected 4 and 1.\n", numGroups, numPairs)
		return nil, false
	}

	return allComponents, true
}

// decomposeRecursive attempts to find groups/pairs in the sorted hand tiles.
// Returns the list of DecomposedGroup found and success boolean.
// IMPORTANT: This is a greedy implementation. It might fail on complex valid hands
// where the initial greedy choice prevents finding the correct overall decomposition.
func decomposeRecursive(currentHand []Tile, groupsNeeded int, pairsNeeded int) ([]DecomposedGroup, bool) {
	components := []DecomposedGroup{}

	// Base Case: Success
	if len(currentHand) == 0 && groupsNeeded == 0 && pairsNeeded == 0 {
		return components, true
	}
	// Base Case: Failure / Impossible state
	if groupsNeeded < 0 || pairsNeeded < 0 || len(currentHand) < (groupsNeeded*3+pairsNeeded*2) || (len(currentHand) == 0 && (groupsNeeded > 0 || pairsNeeded > 0)) {
		return nil, false
	}

	// --- Try Removing Components (Greedy Order: Pair > Pung > Chi) ---

	// 1. Try Removing a Pair
	if pairsNeeded > 0 && len(currentHand) >= 2 {
		if currentHand[0].Suit == currentHand[1].Suit && currentHand[0].Value == currentHand[1].Value {
			pairGroup := DecomposedGroup{
				Type:        TypePair,
				Tiles:       []Tile{currentHand[0], currentHand[1]},
				IsConcealed: true, // Pairs are always from hand/concealed part
			}
			remainingComponents, success := decomposeRecursive(currentHand[2:], groupsNeeded, pairsNeeded-1)
			if success {
				return append([]DecomposedGroup{pairGroup}, remainingComponents...), true
			}
			// Backtrack: Removing this pair didn't work, continue trying other options below.
		}
	}

	// 2. Try Removing a Pung (Triplet)
	if groupsNeeded > 0 && len(currentHand) >= 3 {
		if currentHand[0].Suit == currentHand[1].Suit && currentHand[0].Value == currentHand[1].Value &&
			currentHand[0].Suit == currentHand[2].Suit && currentHand[0].Value == currentHand[2].Value {
			pungGroup := DecomposedGroup{
				Type:        TypeTriplet,
				Tiles:       []Tile{currentHand[0], currentHand[1], currentHand[2]},
				IsConcealed: true, // If found here, it's an Ankou (concealed triplet)
			}
			remainingComponents, success := decomposeRecursive(currentHand[3:], groupsNeeded-1, pairsNeeded)
			if success {
				return append([]DecomposedGroup{pungGroup}, remainingComponents...), true
			}
			// Backtrack
		}
	}

	// 3. Try Removing a Chi (Sequence)
	if groupsNeeded > 0 && len(currentHand) >= 3 && currentHand[0].Suit != "Wind" && currentHand[0].Suit != "Dragon" {
		v1 := currentHand[0].Value
		s1 := currentHand[0].Suit
		idx2 := -1
		idx3 := -1
		// Find first occurrence of v+1
		for k := 1; k < len(currentHand); k++ {
			if currentHand[k].Suit == s1 && currentHand[k].Value == v1+1 {
				idx2 = k
				break
			}
		}
		// Find first occurrence of v+2 *after* v+1
		if idx2 != -1 {
			for k := idx2 + 1; k < len(currentHand); k++ {
				if currentHand[k].Suit == s1 && currentHand[k].Value == v1+2 {
					idx3 = k
					break
				}
			}
		}

		if idx3 != -1 { // Found a Chi: currentHand[0], currentHand[idx2], currentHand[idx3]
			chiGroup := DecomposedGroup{
				Type:        TypeSequence,
				Tiles:       []Tile{currentHand[0], currentHand[idx2], currentHand[idx3]},
				IsConcealed: true, // If found here, it's from the concealed hand part
			}
			// Create remaining hand *carefully* excluding used indices
			remainingHand := []Tile{}
			indicesUsed := map[int]bool{0: true, idx2: true, idx3: true}
			for k := 0; k < len(currentHand); k++ {
				if !indicesUsed[k] {
					remainingHand = append(remainingHand, currentHand[k])
				}
			}

			remainingComponents, success := decomposeRecursive(remainingHand, groupsNeeded-1, pairsNeeded)
			if success {
				return append([]DecomposedGroup{chiGroup}, remainingComponents...), true
			}
			// Backtrack
		}
	}

	// If none of the removals starting with the first tile worked, this path fails (greedy limitation).
	// A more robust decomposer would need to try removing elements not starting at index 0.
	return nil, false
}

// Helper to check if a tile ID exists in a list of DecomposedGroup tiles
func groupContainsTileID(group DecomposedGroup, tileID int) bool {
	for _, t := range group.Tiles {
		if t.ID == tileID {
			return true
		}
	}
	return false
}
