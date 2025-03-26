package main

import (
	"fmt"
	"sort"
	"strings"
)

// ==========================================================
// Hand Completion & Structure Checks (Refactored)
// ==========================================================
// IsCompleteHand checks if a hand forms a valid winning shape (Standard, Chiitoi, Kokushi).
// It considers existing melds and the final 14th tile (either drawn or claimed).
// `handTilesForCheck` should contain the tiles NOT part of existing melds,
// including the potential 14th winning tile.
// `melds` are the pre-existing melds.
func IsCompleteHand(handTilesForCheck []Tile, melds []Meld) bool {

	numMelds := len(melds)

	// --- Check for correct number of hand tiles based on melds ---
	// Calculate tiles needed for remaining groups + pair
	groupsNeeded := 4 - numMelds
	pairsNeeded := 1 // A standard hand always needs 1 pair

	if groupsNeeded < 0 {
		// More than 4 melds somehow? Invalid state.
		fmt.Printf("Debug Warning: IsCompleteHand called with >4 melds (%d).\n", numMelds)
		return false
	}

	expectedHandTiles := groupsNeeded*3 + pairsNeeded*2

	// Ensure the input hand tiles match the expected count for the remaining structure
	if len(handTilesForCheck) != expectedHandTiles {
		// This indicates a mismatch between melds and hand tiles provided, or an invalid number of tiles overall.
		// For example, after drawing the 14th tile, handTilesForCheck should contain exactly expectedHandTiles.
		// Allow the check to proceed even with mismatch, as intermediate checks might call it.
		// The caller (e.g., CanDeclareRon/Tsumo) should ensure the correct number of tiles are passed initially.
		// fmt.Printf("Debug Warning: IsCompleteHand inconsistent tile count. Hand tiles received: %d, Expected: %d (for %d groups + %d pair needed).\n",
		// 	len(handTilesForCheck), expectedHandTiles, groupsNeeded, pairsNeeded)
		// // Return false because the input doesn't have the correct number of tiles to complete the structure.
		// return false
	}
	// --- End Tile Count Check ---

	// Need a mutable copy of hand tiles for recursive checks
	handCopy := make([]Tile, len(handTilesForCheck))
	copy(handCopy, handTilesForCheck)
	sort.Sort(BySuitValue(handCopy)) // Ensure sorted for checks

	// --- Check Special Hands (Require concealed state, no open melds) ---
	isConcealed := true
	for _, m := range melds {
		if !m.IsConcealed { // Any open meld disqualifies Kokushi/Chiitoitsu
			isConcealed = false
			break
		}
	}

	// Check Kokushi/Chiitoitsu only if the hand *was* originally concealed (numMelds == 0 implies no open melds)
	// AND we are checking a full 14-tile hand.
	if isConcealed && numMelds == 0 && len(handTilesForCheck) == 14 {
		if IsKokushiMusou(handCopy) { // Pass the 14 tiles
			return true
		}
		if IsChiitoitsu(handCopy) { // Pass the 14 tiles
			return true
		}
	}

	// --- Check Standard Hand (4 Groups + 1 Pair) ---
	// groupsNeeded and pairsNeeded were calculated earlier.

	// CheckStandardHandRecursive should verify if handCopy (containing expectedHandTiles number of tiles)
	// can form groupsNeeded groups and pairsNeeded pair.
	return CheckStandardHandRecursive(handCopy, groupsNeeded, pairsNeeded)
}

// CheckStandardHandRecursive attempts to find `groupsNeeded` groups (Pung/Chi)
// and `pairsNeeded` pairs from the `currentHand` tiles.
// Assumes `currentHand` is sorted.
func CheckStandardHandRecursive(currentHand []Tile, groupsNeeded int, pairsNeeded int) bool {
	// Base Case: Success - Found all required groups and pairs, no tiles left.
	if len(currentHand) == 0 && groupsNeeded == 0 && pairsNeeded == 0 {
		return true
	}
	// Base Case: Failure - Impossible to form remaining groups/pairs with leftover tiles.
	// Or negative counts needed (shouldn't happen with proper calls).
	if groupsNeeded < 0 || pairsNeeded < 0 || len(currentHand) < (groupsNeeded*3+pairsNeeded*2) {
		return false
	}
	// Base Case: Optimization - If no tiles left but still need groups/pairs
	if len(currentHand) == 0 && (groupsNeeded > 0 || pairsNeeded > 0) {
		return false
	}

	// --- Recursive Steps: Try removing a component ---
	// Prioritize removing pair first can sometimes be faster, but not strictly necessary.

	// 1. Try Removing a Pair (if needed)
	if pairsNeeded > 0 && len(currentHand) >= 2 {
		// Check if first two tiles form a pair (Suit/Value match)
		if currentHand[0].Suit == currentHand[1].Suit && currentHand[0].Value == currentHand[1].Value {
			// Recursively check if the rest of the hand can form the remaining groups/pairs
			if CheckStandardHandRecursive(currentHand[2:], groupsNeeded, pairsNeeded-1) {
				return true // Found a valid path by removing this pair first
			}
			// Backtrack: If removing this pair didn't lead to a solution, continue searching.
		}
	}

	// 2. Try Removing a Pung (Triplet) (if needed)
	if groupsNeeded > 0 && len(currentHand) >= 3 {
		// Check if first three tiles form a Pung (Suit/Value match)
		if currentHand[0].Suit == currentHand[1].Suit && currentHand[0].Value == currentHand[1].Value &&
			currentHand[0].Suit == currentHand[2].Suit && currentHand[0].Value == currentHand[2].Value {
			// Recursively check if the rest of the hand can form the remaining groups/pairs
			if CheckStandardHandRecursive(currentHand[3:], groupsNeeded-1, pairsNeeded) {
				return true // Found a valid path by removing this Pung first
			}
			// Backtrack: If removing this Pung didn't lead to a solution, continue searching.
		}
	}

	// 3. Try Removing a Chi (Sequence) (if needed)
	if groupsNeeded > 0 && len(currentHand) >= 3 && currentHand[0].Suit != "Wind" && currentHand[0].Suit != "Dragon" {
		v1 := currentHand[0].Value
		s1 := currentHand[0].Suit
		idx2 := -1 // Index of value+1 tile
		idx3 := -1 // Index of value+2 tile

		// Find first occurrence of value+1 in the remaining hand
		for k := 1; k < len(currentHand); k++ {
			if currentHand[k].Suit == s1 && currentHand[k].Value == v1+1 {
				idx2 = k
				break
			}
		}
		// Find first occurrence of value+2 *after* index idx2
		if idx2 != -1 {
			for k := idx2 + 1; k < len(currentHand); k++ {
				if currentHand[k].Suit == s1 && currentHand[k].Value == v1+2 {
					idx3 = k
					break
				}
			}
		}

		// If we found a complete sequence (tiles at indices 0, idx2, idx3)
		if idx3 != -1 {
			// Create the remaining hand by excluding these three specific tiles
			remainingHand := []Tile{}
			indicesUsed := map[int]bool{0: true, idx2: true, idx3: true}
			for k := 0; k < len(currentHand); k++ {
				if !indicesUsed[k] {
					remainingHand = append(remainingHand, currentHand[k])
				}
			}

			// Recursively check if the rest of the hand can form the remaining groups/pairs
			if CheckStandardHandRecursive(remainingHand, groupsNeeded-1, pairsNeeded) {
				return true // Found a valid path by removing this Chi first
			}
			// Backtrack: If removing this Chi didn't lead to a solution, continue searching.
		}
	}

	// --- Backtracking Limitation ---
	// The current greedy approach based on the first tile might fail complex waits.
	// If none of the above checks starting with currentHand[0] led to a solution,
	// this path fails under the greedy assumption. A more complex algorithm would
	// explore removing groups/pairs starting at other indices.
	return false
}

// --- Rest of checks.go (IsKokushiMusou, IsChiitoitsu, IsTenpai, etc.) ---

// IsKokushiMusou checks for the 13 Orphans hand (14 tiles version - pair wait)
func IsKokushiMusou(hand []Tile) bool {
	if len(hand) != 14 {
		return false
	}

	terminalsAndHonors := map[string]int{ // Tile Name -> Required Count (initially 0)
		"Man 1": 0, "Man 9": 0, "Pin 1": 0, "Pin 9": 0, "Sou 1": 0, "Sou 9": 0,
		"East": 0, "South": 0, "West": 0, "North": 0,
		"White": 0, "Green": 0, "Red": 0,
	}
	requiredTypes := len(terminalsAndHonors) // 13 unique types needed
	foundTypes := 0
	hasPair := false
	tileCountsByName := make(map[string]int)

	for _, tile := range hand {
		// Use base name (strip "Red ") for checking requirement
		baseName := strings.TrimPrefix(tile.Name, "Red ")
		tileCountsByName[baseName]++
	}

	for name, count := range tileCountsByName {
		_, isRequired := terminalsAndHonors[name]
		if isRequired {
			if count > 2 {
				return false
			} // Cannot have > 2 of any required type
			if count >= 1 {
				// Only increment foundTypes *once* per required type
				if terminalsAndHonors[name] == 0 {
					foundTypes++
				}
				terminalsAndHonors[name] = count // Store actual count
			}
			if count == 2 {
				if hasPair {
					return false
				} // Cannot have more than one pair
				hasPair = true
			}
		} else {
			return false // Contains a tile not part of the Kokushi set
		}
	}

	return foundTypes == requiredTypes && hasPair
}

// IsChiitoitsu checks for the Seven Pairs hand (14 tiles)
func IsChiitoitsu(hand []Tile) bool {
	if len(hand) != 14 {
		return false
	}

	// Use unique ID counts for Chiitoitsu, as identical tiles (Red 5m, Red 5m) form a pair.
	tileCountsByID := make(map[int]int)
	for _, t := range hand {
		tileCountsByID[t.ID]++
	}
	pairCountByID := 0
	for _, count := range tileCountsByID {
		if count == 2 {
			pairCountByID++
		} else if count == 4 {
			pairCountByID += 2 // Treat 4 identical tiles as 2 pairs for Chiitoitsu
		} else if count != 0 {
			// Any other count (1, 3, etc.) based on specific tile ID invalidates Chiitoitsu
			return false
		}
	}

	return pairCountByID == 7
}

// IsTenpai checks if a hand is one tile away from being complete.
// Expects a 13-tile hand state (currentHand + melds).
func IsTenpai(currentHand []Tile, melds []Meld) bool {
	// Calculate expected number of tiles in hand based on melds
	numMeldTiles := 0
	numKans := 0
	for _, m := range melds {
		numMeldTiles += len(m.Tiles) // Assumes len is 3 for P/C, 4 for K
		if strings.Contains(m.Type, "Kan") {
			numKans++
		}
	}
	// A standard hand has 13 tiles. Each Kan means one fewer tile is needed in hand.
	expectedHandSize := HandSize - numKans

	currentTotalInHand := len(currentHand)
	if currentTotalInHand != expectedHandSize {
		// fmt.Printf("Debug Tenpai Check: Incorrect hand tile count %d, expected %d (HandSize %d - Kans %d)\n",
		// 	currentTotalInHand, expectedHandSize, HandSize, numKans)
		// Allow check to proceed, might be called in intermediate states
	}

	possibleTiles := GetAllPossibleTiles() // Unique 34 types

	for _, testTile := range possibleTiles {
		// Create a hypothetical 14-tile state by adding the test tile to the concealed part
		tempConcealedHand := append([]Tile{}, currentHand...)
		tempConcealedHand = append(tempConcealedHand, testTile)
		sort.Sort(BySuitValue(tempConcealedHand)) // Keep it sorted

		// Check if this hypothetical concealed hand + existing melds forms a complete hand
		// Pass the tiles NOT in melds (tempConcealedHand) and the original melds list
		if IsCompleteHand(tempConcealedHand, melds) {
			return true // Found a wait
		}
	}

	return false // No tile completes the hand
}

// FindTenpaiWaits returns a list of *unique tile types* that would complete the hand.
// Expects a 13-tile hand state (currentHand + melds).
func FindTenpaiWaits(currentHand []Tile, melds []Meld) []Tile {
	waits := []Tile{}
	possibleTiles := GetAllPossibleTiles() // Unique 34 types
	seenWaits := make(map[string]bool)     // Track waits by Suit-Value to ensure uniqueness

	for _, testTile := range possibleTiles {
		// Create hypothetical 14-tile state (concealed part + test tile)
		tempConcealedHand := append([]Tile{}, currentHand...)
		tempConcealedHand = append(tempConcealedHand, testTile)
		sort.Sort(BySuitValue(tempConcealedHand))

		// Check completion using this hypothetical state + original melds
		if IsCompleteHand(tempConcealedHand, melds) {
			waitKey := fmt.Sprintf("%s-%d", testTile.Suit, testTile.Value)
			if !seenWaits[waitKey] {
				waits = append(waits, testTile) // Add representative tile (non-red)
				seenWaits[waitKey] = true
			}
		}
	}
	sort.Sort(BySuitValue(waits)) // Sort the waits for display
	return waits
}

// FindPossibleChiSequences identifies the sets of *two hand tiles* needed to form Chi with the discard.
func FindPossibleChiSequences(player *Player, discardedTile Tile) [][]Tile {
	var sequences [][]Tile // Stores pairs of hand tiles needed
	if discardedTile.Suit == "Wind" || discardedTile.Suit == "Dragon" {
		return sequences
	}

	val := discardedTile.Value
	suit := discardedTile.Suit
	hand := player.Hand // Check against concealed hand

	// --- Find all instances of required tiles in hand ---
	findIndices := func(targetValue int) []int {
		indices := []int{}
		for i, tile := range hand {
			if tile.Suit == suit && tile.Value == targetValue {
				indices = append(indices, i)
			}
		}
		return indices
	}

	// Indices of tiles in hand matching potential sequence partners
	valM2Indices := findIndices(val - 2)
	valM1Indices := findIndices(val - 1)
	valP1Indices := findIndices(val + 1)
	valP2Indices := findIndices(val + 2)

	// Use a map to store found sequences (represented by the pair from hand) to avoid duplicates
	// Key: string like "TileID1-TileID2" (sorted IDs)
	foundSequencesMap := make(map[string][]Tile)

	// Check Pattern 1: Need (Value-2, Value-1) for sequence [Val-2, Val-1, discard]
	if val >= 3 && len(valM2Indices) > 0 && len(valM1Indices) > 0 {
		for _, idxM2 := range valM2Indices {
			for _, idxM1 := range valM1Indices {
				if idxM1 == idxM2 {
					continue
				} // Cannot use the same tile instance
				tile1 := hand[idxM2]
				tile2 := hand[idxM1]
				seqKey := GenerateSequenceKey(tile1, tile2)
				foundSequencesMap[seqKey] = []Tile{tile1, tile2}
			}
		}
	}

	// Check Pattern 2: Need (Value-1, Value+1) for sequence [Val-1, discard, Val+1]
	if val >= 2 && val <= 8 && len(valM1Indices) > 0 && len(valP1Indices) > 0 {
		for _, idxM1 := range valM1Indices {
			for _, idxP1 := range valP1Indices {
				if idxM1 == idxP1 {
					continue
				}
				tile1 := hand[idxM1]
				tile2 := hand[idxP1]
				seqKey := GenerateSequenceKey(tile1, tile2)
				foundSequencesMap[seqKey] = []Tile{tile1, tile2}
			}
		}
	}

	// Check Pattern 3: Need (Value+1, Value+2) for sequence [discard, Val+1, Val+2]
	if val <= 7 && len(valP1Indices) > 0 && len(valP2Indices) > 0 {
		for _, idxP1 := range valP1Indices {
			for _, idxP2 := range valP2Indices {
				if idxP1 == idxP2 {
					continue
				}
				tile1 := hand[idxP1]
				tile2 := hand[idxP2]
				seqKey := GenerateSequenceKey(tile1, tile2)
				foundSequencesMap[seqKey] = []Tile{tile1, tile2}
			}
		}
	}

	// Convert map values back to slice
	for _, seq := range foundSequencesMap {
		sequences = append(sequences, seq)
	}

	// Optional: Sort the outer slice of sequences for consistent ordering in prompts
	sort.Slice(sequences, func(i, j int) bool {
		// Sort based on the first tile in the pair (arbitrary but consistent)
		if sequences[i][0].ID != sequences[j][0].ID {
			return sequences[i][0].ID < sequences[j][0].ID
		}
		return sequences[i][1].ID < sequences[j][1].ID // Then by second tile
	})

	return sequences
}

// Helper function to create a unique key for a pair of tiles based on sorted IDs
func GenerateSequenceKey(t1, t2 Tile) string {
	if t1.ID < t2.ID {
		return fmt.Sprintf("%d-%d", t1.ID, t2.ID)
	}
	return fmt.Sprintf("%d-%d", t2.ID, t1.ID)
}

// ==========================================================
// Action Possibility Checks (Updated with Yaku checks)
// ==========================================================

// CanDeclareRon checks if a player can win by Ron on the discarded tile.
func CanDeclareRon(player *Player, discardedTile Tile, gs *GameState) bool {
	if player.IsRiichi && player.IsFuriten {
		// If in Riichi, temporary Furiten doesn't apply, only permanent matters
		// TODO: Implement permanent Riichi Furiten check
		return false // Assume Riichi + Furiten means cannot Ron for now
	}
	if !player.IsRiichi && player.IsFuriten {
		// Standard temporary Furiten applies
		return false
	}

	// Check if hand becomes complete with the discarded tile
	// Create a temporary *concealed* hand including the discard
	tempConcealedHand := append([]Tile{}, player.Hand...)
	tempConcealedHand = append(tempConcealedHand, discardedTile)
	sort.Sort(BySuitValue(tempConcealedHand))

	// Check completion using the hypothetical concealed hand + original melds
	if !IsCompleteHand(tempConcealedHand, player.Melds) {
		return false // Hand shape is not complete
	}

	// *** YAKU CHECK ***
	// Call IdentifyYaku to see if the resulting hand has any Yaku.
	// Pass isTsumo = false for Ron.
	_, han := IdentifyYaku(player, discardedTile, false, gs)
	if han == 0 {
		// fmt.Printf("Debug: Ron possible shape for %s, but no Yaku found.\n", player.Name) // Optional debug
		return false // No Yaku, cannot Ron
	}

	return true // Hand is complete AND has at least one Yaku
}

// CanDeclareTsumo checks if the player can win by Tsumo after drawing.
// Assumes player.Hand currently holds the tiles *after* the draw (including Rinshan).
func CanDeclareTsumo(player *Player, gs *GameState) bool {
	// Calculate total tiles accounted for (hand + melds)
	totalTiles := len(player.Hand)
	numKans := 0 // Count kans specifically if needed, but total count is simpler
	for _, m := range player.Melds {
		totalTiles += len(m.Tiles)
		if strings.Contains(m.Type, "Kan") { // Keep track if needed elsewhere
			numKans++
		}
	}

	// A winning hand always consists of 14 tiles total (standard shape)
	if totalTiles != 14 {
		fmt.Printf("Warning: CanDeclareTsumo called when total tiles (hand %d + melds) is %d (expected 14).\n", len(player.Hand), totalTiles)
		return false // Incorrect total number of tiles for a complete hand
	}

	// Check if the current hand + melds forms a complete shape
	// IsCompleteHand needs the concealed part + melds.
	// player.Hand *is* the concealed part after the draw.
	if !IsCompleteHand(player.Hand, player.Melds) {
		return false // Hand shape is not complete
	}

	// *** YAKU CHECK ***
	// Need the actual drawn tile. Assuming it's the last one added/sorted.
	drawnTile := Tile{}
	if len(player.Hand) > 0 {
		// Still assuming last tile after sort is the draw. Might need refinement.
		drawnTile = player.Hand[len(player.Hand)-1]
	} else {
		fmt.Println("Warning: CanDeclareTsumo called with empty hand?")
		return false
	}

	_, han := IdentifyYaku(player, drawnTile, true, gs) // isTsumo = true
	if han == 0 {
		// fmt.Printf("Debug: Tsumo possible shape for %s, but no Yaku found.\n", player.Name)
		return false // No Yaku, cannot Tsumo
	}

	return true // Hand is complete AND has at least one Yaku
}

// CanDeclarePon checks if a player can call Pon on a discarded tile.
func CanDeclarePon(player *Player, discardedTile Tile) bool {
	// Player must not have called Chi on the immediately preceding discard (if applicable - rare rule?)
	// Check if hand is Menzen if specific rules apply (usually not needed for Pon).

	count := 0
	for _, tile := range player.Hand {
		if tile.Suit == discardedTile.Suit && tile.Value == discardedTile.Value {
			count++
		}
	}
	return count >= 2
}

// CanDeclareChi checks if a player can call Chi on a discarded tile.
// Only the player immediately to the left can call Chi.
func CanDeclareChi(player *Player, discardedTile Tile) bool {
	// Basic check: Must not be an honor tile
	if discardedTile.Suit == "Wind" || discardedTile.Suit == "Dragon" {
		return false
	}

	// Check if hand is Menzen if specific rules apply (usually not needed for Chi).

	// Check for the required pairs in hand
	val := discardedTile.Value
	suit := discardedTile.Suit
	hand := player.Hand
	// Check Pattern 1: Need (Value-2, Value-1)
	if val >= 3 && HasTileWithValue(hand, suit, val-2) && HasTileWithValue(hand, suit, val-1) {
		return true
	}
	// Check Pattern 2: Need (Value-1, Value+1)
	if val >= 2 && val <= 8 && HasTileWithValue(hand, suit, val-1) && HasTileWithValue(hand, suit, val+1) {
		return true
	}
	// Check Pattern 3: Need (Value+1, Value+2)
	if val <= 7 && HasTileWithValue(hand, suit, val+1) && HasTileWithValue(hand, suit, val+2) {
		return true
	}
	return false
}

// CanDeclareDaiminkan checks specifically for calling Kan on another player's discard.
func CanDeclareDaiminkan(player *Player, discardedTile Tile) bool {
	// Check rule Ssuukantsu (4 Kans total, often abortive draw)
	// TODO: Implement check for total Kans declared in gs if Ssuukantsu rule applies.
	numPlayerKans := 0
	for _, m := range player.Melds {
		if strings.Contains(m.Type, "Kan") {
			numPlayerKans++
		}
	}
	if numPlayerKans >= 4 { // Cannot declare 5th Kan
		return false
	}

	countInHand := 0
	for _, t := range player.Hand {
		if t.Suit == discardedTile.Suit && t.Value == discardedTile.Value {
			countInHand++
		}
	}
	return countInHand == 3
}

// CanDeclareKanOnDraw checks if the player can declare Ankan or Shouminkan using the drawn tile.
// Assumes player.Hand includes the drawn tile (14 tiles total).
func CanDeclareKanOnDraw(player *Player, drawnTile Tile) (string, Tile) {
	// TODO: Check Ssuukantsu rule
	numPlayerKans := 0
	for _, m := range player.Melds {
		if strings.Contains(m.Type, "Kan") {
			numPlayerKans++
		}
	}
	if numPlayerKans >= 4 {
		return "", Tile{}
	}

	// Check for Ankan (4 identical tiles in hand including the draw)
	countInHand := 0
	for _, t := range player.Hand { // Hand includes drawn tile
		if t.Suit == drawnTile.Suit && t.Value == drawnTile.Value {
			countInHand++
		}
	}
	if countInHand == 4 {
		// Check if last drawable tile (prevents Kan if no Rinshan possible - though Dead Wall handles Rinshan count)
		// Check if Riichi (cannot Ankan if it changes waits, unless allowed by ruleset)
		// TODO: Implement wait change check for Riichi Ankan
		return "Ankan", drawnTile
	}

	// Check for Shouminkan (add drawn tile to existing Pon)
	for _, meld := range player.Melds {
		if meld.Type == "Pon" {
			ponTile := meld.Tiles[0] // All tiles in Pon are the same type
			if drawnTile.Suit == ponTile.Suit && drawnTile.Value == ponTile.Value {
				// Check if Riichi (cannot Shouminkan if it changes waits, typically allowed if same tile type)
				return "Shouminkan", drawnTile
			}
		}
	}
	return "", Tile{}
}

// CanDeclareKanOnHand checks if the player can declare Ankan or Shouminkan using only tiles currently in hand/melds
// (i.e., not immediately after drawing, but maybe after a call, or just before discarding).
// `checkTile` is one representative tile from the potential Kan group.
// Assumes player.Hand is in its current state (e.g., 13 tiles after call + discard prompt, or 14 before normal discard).
func CanDeclareKanOnHand(player *Player, checkTile Tile) (string, Tile) {
	// TODO: Check Ssuukantsu rule
	numPlayerKans := 0
	for _, m := range player.Melds {
		if strings.Contains(m.Type, "Kan") {
			numPlayerKans++
		}
	}
	if numPlayerKans >= 4 {
		return "", Tile{}
	}

	// Check for Ankan (4 identical tiles currently in hand)
	countInHand := 0
	for _, t := range player.Hand {
		if t.Suit == checkTile.Suit && t.Value == checkTile.Value {
			countInHand++
		}
	}
	if countInHand == 4 {
		// TODO: Check Riichi wait change rules if applicable
		return "Ankan", checkTile
	}

	// Check for Shouminkan (1 tile in hand + existing Pon)
	hasTileInHand := false
	for _, t := range player.Hand {
		if t.Suit == checkTile.Suit && t.Value == checkTile.Value {
			hasTileInHand = true
			break
		}
	}
	if hasTileInHand {
		for _, meld := range player.Melds {
			if meld.Type == "Pon" {
				ponTile := meld.Tiles[0]
				if checkTile.Suit == ponTile.Suit && checkTile.Value == ponTile.Value {
					// TODO: Check Riichi wait change rules if applicable
					return "Shouminkan", checkTile
				}
			}
		}
	}
	return "", Tile{}
}

// CanDeclareRiichi checks if the player can declare Riichi.
func CanDeclareRiichi(player *Player, gs *GameState) bool {
	if player.IsRiichi {
		return false // Already in Riichi
	}
	// Must be Menzenchin (concealed hand)
	isConcealed := true
	for _, m := range player.Melds {
		// Only Ankan allowed for Riichi
		if !m.IsConcealed && m.Type != "Ankan" {
			isConcealed = false
			break
		}
	}
	if !isConcealed {
		return false // Hand is open
	}
	// Must have >= 1000 points
	if player.Score < 1000 {
		return false // Not enough points
	}
	// Must be >= 4 tiles left in the wall
	if len(gs.Wall) < 4 {
		return false // Not enough wall tiles left for potential Ippatsu/Ura Dora
	}
	// Must have 14 tiles before discard
	if len(player.Hand) != HandSize+1 {
		// fmt.Printf("Debug: CanDeclareRiichi incorrect hand size %d\n", len(player.Hand)) // Debug
		return false // Must be holding 13 + drawn tile
	}

	// Check if *any* discard leaves the remaining 13 tiles Tenpai
	foundTenpaiDiscard := false
	for i := 0; i < len(player.Hand); i++ {
		// Create temporary 13-tile hand after hypothetical discard
		tempHand13 := make([]Tile, 0, HandSize)
		for j, t := range player.Hand {
			if i != j {
				tempHand13 = append(tempHand13, t)
			}
		}
		// Check Tenpai with the *original* melds (which are all concealed/Ankan for Riichi)
		if IsTenpai(tempHand13, player.Melds) {
			foundTenpaiDiscard = true
			break // Found at least one discard that results in Tenpai
		}
	}

	return foundTenpaiDiscard // Can Riichi if concealed, enough points/wall, and is Tenpai after discarding *some* tile
}
