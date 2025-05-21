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

	// 2. Attempt to find a pair and then decompose the remaining hand tiles into melds.
	// pairsNeeded is effectively always 1 for a standard hand decomposition.

	// If no further groups need to be found in hand (i.e., 4 melds are open)
	if groupsNeeded == 0 {
		if len(handTilesForDecomp) == 2 && tilesAreEqual(handTilesForDecomp[0], handTilesForDecomp[1]) {
			pairGroup := DecomposedGroup{
				Type:        TypePair,
				Tiles:       handTilesForDecomp,
				IsConcealed: true, // Pair from hand is concealed
			}
			// All components are the already melded groups + this pair
			allComponents := append(meldedGroups, pairGroup)
			// Final validation for this specific case
			if len(allComponents) != 5 {
				// This implies an issue with meldedGroups count if groupsNeeded was 0
				// fmt.Printf("Debug: (groupsNeeded=0) Expected 5 components, got %d\n", len(allComponents))
				return nil, false
			}
			numFoundGroups := 0
			numFoundPairs := 0
			for _, comp := range allComponents {
				if comp.Type == TypePair {
					numFoundPairs++
				} else {
					numFoundGroups++
				}
			}
			if numFoundGroups == 4 && numFoundPairs == 1 {
				return allComponents, true
			}
			// fmt.Printf("Debug: (groupsNeeded=0) Expected 4 groups & 1 pair, got %d & %d\n", numFoundGroups, numFoundPairs)
			return nil, false
		}
		// Not a valid pair left in hand, or wrong number of tiles.
		// fmt.Printf("Debug: groupsNeeded is 0, but handTilesForDecomp is not a pair. Size: %d\n", len(handTilesForDecomp))
		return nil, false
	}

	// If we need to find groups and must have a pair from hand.
	// At least 2 tiles for pair + 3 for each group needed.
	if len(handTilesForDecomp) < (2 + groupsNeeded*3) {
		// fmt.Printf("Debug: Not enough hand tiles for a pair and %d groups. Have %d, need %d.\n", groupsNeeded, len(handTilesForDecomp), 2+groupsNeeded*3)
		return nil, false
	}

	// Iterate through all unique tiles in handTilesForDecomp to select a pair.
	// Then, try to decompose the rest into 'groupsNeeded' melds.
	tileCounts := make(map[string]int) // Counts of each tile type
	uniqueTileExemplars := []Tile{}    // One exemplar of each unique tile type
	for _, t := range handTilesForDecomp {
		// Key for tile type uniqueness is Suit and Value.
		key := fmt.Sprintf("%s-%d", t.Suit, t.Value)
		if tileCounts[key] == 0 {
			uniqueTileExemplars = append(uniqueTileExemplars, t)
		}
		tileCounts[key]++
	}
	// uniqueTileExemplars are already effectively sorted by appearance in sorted handTilesForDecomp.

	for _, pairExemplar := range uniqueTileExemplars {
		key := fmt.Sprintf("%s-%d", pairExemplar.Suit, pairExemplar.Value)
		if tileCounts[key] >= 2 { // Found a potential pair
			pairGroup := DecomposedGroup{
				Type: TypePair,
				// Create the pair with two distinct tile instances if possible, or use exemplar twice.
				// For logic, exemplar twice is fine as long as counts are right.
				// Actual tile instances might matter for IDs if used later, but for structure, this is okay.
				Tiles:       []Tile{pairExemplar, pairExemplar},
				IsConcealed: true,
			}

			// Create remaining hand after removing this pair
			remainingForMelds := make([]Tile, 0, len(handTilesForDecomp)-2)
			removedCount := 0
			for _, t := range handTilesForDecomp { // Iterate original sorted hand to build remaining
				if tilesAreEqual(t, pairExemplar) && removedCount < 2 {
					removedCount++
					continue
				}
				remainingForMelds = append(remainingForMelds, t)
			}
			// 'remainingForMelds' is already sorted because 'handTilesForDecomp' was sorted
			// and we iterated through it, preserving relative order of non-pair tiles.

			// Now, find 'groupsNeeded' melds from 'remainingForMelds'
			foundMelds, success := findMeldsRecursive(remainingForMelds, groupsNeeded)
			if success {
				allComponents := append(meldedGroups, pairGroup)
				allComponents = append(allComponents, foundMelds...)

				// Final validation: Should have exactly 5 components (4 groups + 1 pair)
				if len(allComponents) != 5 {
					// This should ideally not happen if logic is correct
					// fmt.Printf("Debug: Final validation failed. Expected 5 components, got %d\n", len(allComponents))
					continue // Try next pair, this path was somehow invalid
				}
				numFoundGroups := 0
				numFoundPairs := 0
				for _, comp := range allComponents {
					if comp.Type == TypePair {
						numFoundPairs++
					} else {
						numFoundGroups++
					}
				}
				if numFoundGroups == 4 && numFoundPairs == 1 {
					return allComponents, true // Success!
				}
				// This also should ideally not happen with correct group/pair counts
				// fmt.Printf("Debug: Final validation failed. Expected 4 groups & 1 pair, got %d & %d\n", numFoundGroups, numFoundPairs)
				// Continue to try next pair if this specific combination didn't meet final counts.
			}
			// If findMeldsRecursive failed, backtrack: the loop will try the next potential pair.
		}
	}

	// If no pair combination led to a successful decomposition of the remaining tiles.
	// fmt.Println("Debug: DecomposeWinningHand could not find a valid pair + melds combination.")
	return nil, false // Moved the original allComponents and related checks into the loop success path
}

// tilesAreEqual checks if two tiles are of the same suit and value.
// This is a helper function, place it appropriately (e.g., near findMeldsRecursive or as a package utility).
func tilesAreEqual(t1, t2 Tile) bool {
	return t1.Suit == t2.Suit && t1.Value == t2.Value
}

// findMeldsRecursive attempts to find 'groupsNeeded' melds (Pungs or Chis) from the 'currentHand'.
// 'currentHand' must be sorted.
// This function tries to form a meld using the first tile (currentHand[0]),
// then recursively calls itself for the remaining tiles and groups.
func findMeldsRecursive(currentHand []Tile, groupsNeeded int) ([]DecomposedGroup, bool) {
	// Base Case: Success - all groups found
	if groupsNeeded == 0 {
		if len(currentHand) == 0 {
			return []DecomposedGroup{}, true // Successfully decomposed all groups, no tiles left
		}
		// fmt.Printf("Debug: findMeldsRecursive groupsNeeded is 0, but %d tiles remain.\n", len(currentHand))
		return nil, false // All groups found, but tiles are inexplicably left over
	}

	// Base Case: Failure - not enough tiles to form remaining groups, or no tiles left when groups are still needed.
	if len(currentHand) < groupsNeeded*3 || len(currentHand) == 0 {
		// fmt.Printf("Debug: findMeldsRecursive not enough tiles or no tiles. Have %d, need %d groups (min %d tiles).\n", len(currentHand), groupsNeeded, groupsNeeded*3)
		return nil, false
	}

	// Option 1: Try to form a Pung (triplet) with the first tile
	// Check if the first three tiles are identical
	if len(currentHand) >= 3 && tilesAreEqual(currentHand[0], currentHand[1]) && tilesAreEqual(currentHand[0], currentHand[2]) {
		pungGroup := DecomposedGroup{
			Type:        TypeTriplet,
			Tiles:       []Tile{currentHand[0], currentHand[1], currentHand[2]},
			IsConcealed: true, // Melds found in hand are concealed
		}
		// Recursively find remaining groups from the rest of the hand (after the Pung)
		remainingMelds, success := findMeldsRecursive(currentHand[3:], groupsNeeded-1)
		if success {
			return append([]DecomposedGroup{pungGroup}, remainingMelds...), true
		}
		// If forming a Pung with currentHand[0] and then decomposing the rest failed,
		// we backtrack and next try to form a Chi with currentHand[0] (if applicable).
		// Execution will flow to the Chi attempt below.
	}

	// Option 2: Try to form a Chi (sequence) with the first tile
	// Sequences cannot be made with Wind or Dragon tiles.
	if currentHand[0].Suit != "Wind" && currentHand[0].Suit != "Dragon" {
		tile1 := currentHand[0]
		idx2 := -1 // Index of the second tile in the sequence (value + 1)
		idx3 := -1 // Index of the third tile in the sequence (value + 2)

		// Find the first occurrence of tile1.Value + 1 in the same suit
		for i := 1; i < len(currentHand); i++ { // Start from 1 as currentHand[0] is tile1
			if tilesAreEqual(Tile{Suit: tile1.Suit, Value: tile1.Value + 1}, currentHand[i]) {
				idx2 = i
				break
			}
		}

		// If the second tile was found, find the first occurrence of tile1.Value + 2
		if idx2 != -1 {
			// Start search for the third tile *after* the found second tile (idx2)
			for i := idx2 + 1; i < len(currentHand); i++ {
				if tilesAreEqual(Tile{Suit: tile1.Suit, Value: tile1.Value + 2}, currentHand[i]) {
					idx3 = i
					break
				}
			}
		}

		// If all three tiles for a sequence were found (idx2 and idx3 are valid)
		if idx3 != -1 { // implies idx2 is also valid
			chiGroup := DecomposedGroup{
				Type:        TypeSequence,
				Tiles:       []Tile{currentHand[0], currentHand[idx2], currentHand[idx3]},
				IsConcealed: true, // Melds found in hand are concealed
			}

			// Create the next hand by carefully removing the used tiles (currentHand[0], currentHand[idx2], currentHand[idx3])
			nextHand := make([]Tile, 0, len(currentHand)-3)
			// This map helps identify which indices to skip when building nextHand
			indicesUsed := map[int]bool{0: true, idx2: true, idx3: true}
			for i := 0; i < len(currentHand); i++ {
				if !indicesUsed[i] {
					nextHand = append(nextHand, currentHand[i])
				}
			}
			// The 'nextHand' will remain sorted relative to itself because 'currentHand' was sorted,
			// and we are iterating through 'currentHand' in order, appending kept tiles.

			// Recursively find remaining groups from the 'nextHand'
			remainingMelds, success := findMeldsRecursive(nextHand, groupsNeeded-1)
			if success {
				return append([]DecomposedGroup{chiGroup}, remainingMelds...), true
			}
			// If forming a Chi with currentHand[0] and then decomposing the rest failed, we backtrack.
			// Since Pung was tried before (if applicable), and now Chi also failed with currentHand[0],
			// this means currentHand[0] cannot start any meld that leads to a valid full decomposition
			// down this particular recursive path.
		}
	}

	// If we reach here, it means neither a Pung nor a Chi starting with currentHand[0]
	// (that could be successfully formed and lead to a full decomposition of the rest) was found.
	// This specific path of recursion fails.
	// fmt.Printf("Debug: findMeldsRecursive could not form Pung or Chi with %s to satisfy %d groups from %v\n", currentHand[0].Name, groupsNeeded, currentHand)
	return nil, false
}

// decomposeRecursive attempts to find groups/pairs in the sorted hand tiles.
// Returns the list of DecomposedGroup found and success boolean.
// The old decomposeRecursive and groupContainsTileID functions are now removed.
// The main DecomposeWinningHand function above has been updated to use findMeldsRecursive.
// The findMeldsRecursive and tilesAreEqual functions are already part of the file from the previous partial update.
