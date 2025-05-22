package main

import (
	"fmt"
	"sort"
	"strings"
)

// ==========================================================
// Hand Completion & Structure Checks
// ==========================================================

// IsCompleteHand checks if a hand forms a valid winning shape (Standard, Chiitoi, Kokushi).
// `handTilesForCheck` should be the concealed tiles (including the 14th winning tile).
// `melds` are the player's existing melds.
func IsCompleteHand(handTilesForCheck []Tile, melds []Meld) bool {
	numMelds := len(melds)
	groupsNeeded := 4 - numMelds
	pairsNeeded := 1 // Standard hand always needs 1 pair

	if groupsNeeded < 0 { // More than 4 melds (shouldn't happen with valid Kan logic)
		// fmt.Printf("Debug Warning: IsCompleteHand called with >4 melds (%d).\n", numMelds)
		return false
	}
	expectedHandTiles := groupsNeeded*3 + pairsNeeded*2
	if len(handTilesForCheck) != expectedHandTiles {
		// This check is vital. If the tile count is off, it cannot form the required structure.
		// fmt.Printf("Debug Warning: IsCompleteHand inconsistent tile count. Hand: %d, Expected: %d (for %d groups, %d pair from %d melds).\n",
		// 	len(handTilesForCheck), expectedHandTiles, groupsNeeded, pairsNeeded, numMelds)
		return false
	}

	// Create a mutable copy for recursive checks, ensure it's sorted.
	handCopy := make([]Tile, len(handTilesForCheck))
	copy(handCopy, handTilesForCheck)
	sort.Sort(BySuitValue(handCopy))

	// Check Special Hands (Kokushi, Chiitoitsu)
	// These require a fully concealed hand (only Ankans allowed as "melds" which are part of hand)
	// and exactly 14 tiles in the `handTilesForCheck` if no melds exist.
	isEffectivelyConcealed := true
	if numMelds > 0 { // If there are melds, check if all are Ankan
		for _, m := range melds {
			if m.Type != "Ankan" { // Ankan is considered part of a concealed hand for these purposes
				isEffectivelyConcealed = false
				break
			}
		}
	}

	if isEffectivelyConcealed && numMelds == 0 && len(handTilesForCheck) == 14 { // Must be 14 tiles in hand if no melds
		if IsKokushiMusou(handCopy) {
			return true
		}
		if IsChiitoitsu(handCopy) {
			return true
		}
	}

	// Check Standard Hand (4 Groups + 1 Pair)
	// `groupsNeeded` and `pairsNeeded` were calculated based on existing `melds`.
	// `handCopy` contains the tiles that need to form these remaining groups/pair.
	return CheckStandardHandRecursive(handCopy, groupsNeeded, pairsNeeded)
}

// CheckStandardHandRecursive attempts to find `groupsNeeded` groups (Pung/Chi)
// and `pairsNeeded` pairs from the `currentHand` tiles. Assumes `currentHand` is sorted.
func CheckStandardHandRecursive(currentHand []Tile, groupsNeeded int, pairsNeeded int) bool {
	// Base Case: Success
	if len(currentHand) == 0 && groupsNeeded == 0 && pairsNeeded == 0 {
		return true
	}
	// Base Case: Failure (impossible to form remaining with leftover tiles, or negative counts)
	if groupsNeeded < 0 || pairsNeeded < 0 || len(currentHand) < (groupsNeeded*3+pairsNeeded*2) {
		return false
	}
	// Base Case: No tiles left but still need groups/pairs
	if len(currentHand) == 0 && (groupsNeeded > 0 || pairsNeeded > 0) {
		return false
	}

	// 1. Try Removing a Pair (if needed)
	if pairsNeeded > 0 && len(currentHand) >= 2 {
		// Check if first two tiles form a pair (Suit/Value match)
		if currentHand[0].Suit == currentHand[1].Suit && currentHand[0].Value == currentHand[1].Value {
			if CheckStandardHandRecursive(currentHand[2:], groupsNeeded, pairsNeeded-1) {
				return true
			}
		}
	}

	// 2. Try Removing a Pung (Triplet) (if needed)
	if groupsNeeded > 0 && len(currentHand) >= 3 {
		if currentHand[0].Suit == currentHand[1].Suit && currentHand[0].Value == currentHand[1].Value &&
			currentHand[0].Suit == currentHand[2].Suit && currentHand[0].Value == currentHand[2].Value {
			if CheckStandardHandRecursive(currentHand[3:], groupsNeeded-1, pairsNeeded) {
				return true
			}
		}
	}

	// 3. Try Removing a Chi (Sequence) (if needed)
	if groupsNeeded > 0 && len(currentHand) >= 3 && IsSimple(currentHand[0]) || IsTerminal(currentHand[0]) && currentHand[0].Value <= 7 {
		// Sequences only for Man, Pin, Sou and starting tile must allow for a sequence (e.g., not 8 or 9 for some)
		if currentHand[0].Suit != "Wind" && currentHand[0].Suit != "Dragon" {
			v1, s1 := currentHand[0].Value, currentHand[0].Suit
			idx2, idx3 := -1, -1

			for k := 1; k < len(currentHand); k++ { // Find v1+1
				if currentHand[k].Suit == s1 && currentHand[k].Value == v1+1 {
					idx2 = k; break
				}
			}
			if idx2 != -1 {
				for k := idx2 + 1; k < len(currentHand); k++ { // Find v1+2
					if currentHand[k].Suit == s1 && currentHand[k].Value == v1+2 {
						idx3 = k; break
					}
				}
			}

			if idx3 != -1 { // Found sequence 0, idx2, idx3
				remainingHand := []Tile{}
				indicesUsed := map[int]bool{0: true, idx2: true, idx3: true}
				for k := 0; k < len(currentHand); k++ {
					if !indicesUsed[k] {
						remainingHand = append(remainingHand, currentHand[k])
					}
				}
				if CheckStandardHandRecursive(remainingHand, groupsNeeded-1, pairsNeeded) {
					return true
				}
			}
		}
	}
	return false // No path found from current state with greedy choices
}

// IsKokushiMusou checks for the 13 Orphans hand (14 tiles version - pair wait).
func IsKokushiMusou(hand []Tile) bool {
	if len(hand) != 14 { return false }
	terminalsAndHonors := map[string]int{
		"Man 1": 0, "Man 9": 0, "Pin 1": 0, "Pin 9": 0, "Sou 1": 0, "Sou 9": 0,
		"East": 0, "South": 0, "West": 0, "North": 0,
		"White": 0, "Green": 0, "Red": 0,
	}
	requiredTypes := len(terminalsAndHonors)
	foundTypes, hasPair := 0, false
	tileCountsByName := make(map[string]int)
	for _, tile := range hand {
		baseName := strings.TrimPrefix(tile.Name, "Red ") // Red fives don't affect Kokushi
		tileCountsByName[baseName]++
	}
	for name, count := range tileCountsByName {
		_, isRequired := terminalsAndHonors[name]
		if isRequired {
			if count > 2 { return false } // Max 2 of any required type (for the pair)
			if count >= 1 {
				if terminalsAndHonors[name] == 0 { foundTypes++ } // Count unique type found
				terminalsAndHonors[name] = count
			}
			if count == 2 {
				if hasPair { return false } // Only one pair allowed
				hasPair = true
			}
		} else { return false } // Contains a tile not part of Kokushi set
	}
	return foundTypes == requiredTypes && hasPair
}

// IsChiitoitsu checks for the Seven Pairs hand (14 tiles).
func IsChiitoitsu(hand []Tile) bool {
	if len(hand) != 14 { return false }
	tileCountsByID := make(map[int]int) // Use specific tile ID for Chiitoitsu (e.g. Red 5m is different from normal 5m)
	for _, t := range hand { tileCountsByID[t.ID]++ }
	pairCountByID := 0
	for _, count := range tileCountsByID {
		if count == 2 { pairCountByID++ } else if count == 4 { pairCountByID += 2 } // 4 identical tiles = 2 pairs
		else if count != 0 { return false } // Any other count (1, 3) invalidates
	}
	return pairCountByID == 7
}

// IsTenpai checks if a 13-tile hand state (currentHand + melds) is one tile away from being complete.
func IsTenpai(currentHand []Tile, melds []Meld) bool {
	numKans := 0
	for _, m := range melds { if strings.Contains(m.Type, "Kan") { numKans++ } }
	
	// Expected number of tiles in currentHand (concealed part) for a 13-tile state
	// A 13-tile hand means total 13 tiles *before* drawing the 14th.
	// So, HandSize (13) - (tiles_in_melds_not_kans*3) - (tiles_in_kans*4) + kans.
	// Simpler: expectedHandSize = HandSize (13) - (number of tiles in melds that are not the pair).
	// If player has 1 meld (3 tiles), hand should have 10. Total 13.
	// If player has 1 Kan (4 tiles), hand should have 9. Total 13.
	// This means player.Hand should have 13 - (tiles_in_melds_effectively).
	// For Tenpai check, currentHand + melds should effectively be 13 tiles.
	// If a Kan exists, currentHand will be smaller.
	// The number of tiles in currentHand should be HandSize - numKans.

	// This check might be too restrictive if IsTenpai is called in intermediate states.
	// The core logic relies on adding a test tile to form 14 and checking IsCompleteHand.
	// expectedConcealedTilesForTenpaiCheck := HandSize - numKans
	// if len(currentHand) != expectedConcealedTilesForTenpaiCheck {
	//  fmt.Printf("Debug IsTenpai: currentHand len %d, expected %d (HandSize %d - Kans %d)\n",
	// 	len(currentHand), expectedConcealedTilesForTenpaiCheck, HandSize, numKans)
	// // return false // Can be too strict if called at odd times.
	// }


	possibleTiles := GetAllPossibleTiles() // Unique 34 types
	for _, testTile := range possibleTiles {
		tempConcealedHandWithTestTile := append([]Tile{}, currentHand...)
		tempConcealedHandWithTestTile = append(tempConcealedHandWithTestTile, testTile)
		// sort.Sort(BySuitValue(tempConcealedHandWithTestTile)) // IsCompleteHand will sort its copy

		// IsCompleteHand expects the concealed part (which now includes the test tile, making it 14-equivalent)
		// and the existing melds.
		if IsCompleteHand(tempConcealedHandWithTestTile, melds) {
			return true // Found a tile that completes the hand
		}
	}
	return false // No tile completes the hand
}

// FindTenpaiWaits returns a list of *unique tile types* that would complete the hand.
// Expects a 13-tile hand state (currentHand + melds).
func FindTenpaiWaits(currentHand []Tile, melds []Meld) []Tile {
	waits := []Tile{}
	possibleTiles := GetAllPossibleTiles()     // Unique 34 types
	seenWaits := make(map[string]bool)         // Track waits by Suit-Value

	for _, testTile := range possibleTiles {
		tempConcealedHandWithTestTile := append([]Tile{}, currentHand...)
		tempConcealedHandWithTestTile = append(tempConcealedHandWithTestTile, testTile)
		// sort.Sort(BySuitValue(tempConcealedHandWithTestTile)) // IsCompleteHand sorts

		if IsCompleteHand(tempConcealedHandWithTestTile, melds) {
			// Use non-red version for wait key to group red/non-red waits
			waitKeyTile := testTile 
			if waitKeyTile.IsRed { // Create a non-red equivalent for the key
				waitKeyTile.IsRed = false
				waitKeyTile.Name = strings.TrimPrefix(waitKeyTile.Name, "Red ")
			}
			waitKey := fmt.Sprintf("%s-%d", waitKeyTile.Suit, waitKeyTile.Value)
			if !seenWaits[waitKey] {
				waits = append(waits, testTile) // Add the actual tile (can be red or not)
				seenWaits[waitKey] = true
			}
		}
	}
	sort.Sort(BySuitValue(waits)) // Sort the waits for display/consistency
	return waits
}

// FindPossibleChiSequences identifies the sets of *two hand tiles* needed to form Chi with the discard.
func FindPossibleChiSequences(player *Player, discardedTile Tile) [][]Tile {
	var sequences [][]Tile
	if IsHonor(discardedTile) { return sequences } // Cannot Chi honor tiles

	val, suit, hand := discardedTile.Value, discardedTile.Suit, player.Hand
	findIndices := func(targetValue int) []int {
		indices := []int{}
		for i, tile := range hand {
			if tile.Suit == suit && tile.Value == targetValue { indices = append(indices, i) }
		}
		return indices
	}
	valM2Indices, valM1Indices := findIndices(val-2), findIndices(val-1)
	valP1Indices, valP2Indices := findIndices(val+1), findIndices(val+2)
	foundSequencesMap := make(map[string][]Tile) // Use map to store unique pairs from hand

	// Pattern 1: Hand has (Value-2, Value-1) for sequence [Val-2, Val-1, discard]
	if val >= 3 && len(valM2Indices) > 0 && len(valM1Indices) > 0 {
		for _, idxM2 := range valM2Indices {
			for _, idxM1 := range valM1Indices {
				if idxM1 == idxM2 { continue } // Cannot use the same tile instance
				tile1, tile2 := hand[idxM2], hand[idxM1]
				seqKey := GenerateSequenceKey(tile1, tile2) // Key based on sorted IDs of hand tiles
				foundSequencesMap[seqKey] = []Tile{tile1, tile2}
			}
		}
	}
	// Pattern 2: Hand has (Value-1, Value+1) for sequence [Val-1, discard, Val+1]
	if val >= 2 && val <= 8 && len(valM1Indices) > 0 && len(valP1Indices) > 0 {
		for _, idxM1 := range valM1Indices {
			for _, idxP1 := range valP1Indices {
				if idxM1 == idxP1 { continue }
				tile1, tile2 := hand[idxM1], hand[idxP1]
				seqKey := GenerateSequenceKey(tile1, tile2)
				foundSequencesMap[seqKey] = []Tile{tile1, tile2}
			}
		}
	}
	// Pattern 3: Hand has (Value+1, Value+2) for sequence [discard, Val+1, Val+2]
	if val <= 7 && len(valP1Indices) > 0 && len(valP2Indices) > 0 {
		for _, idxP1 := range valP1Indices {
			for _, idxP2 := range valP2Indices {
				if idxP1 == idxP2 { continue }
				tile1, tile2 := hand[idxP1], hand[idxP2]
				seqKey := GenerateSequenceKey(tile1, tile2)
				foundSequencesMap[seqKey] = []Tile{tile1, tile2}
			}
		}
	}
	for _, seq := range foundSequencesMap { sequences = append(sequences, seq) }
	sort.Slice(sequences, func(i, j int) bool { // Sort outer slice for consistent UI
		if sequences[i][0].ID != sequences[j][0].ID { return sequences[i][0].ID < sequences[j][0].ID }
		return sequences[i][1].ID < sequences[j][1].ID
	})
	return sequences
}

// GenerateSequenceKey creates a unique key for a pair of tiles based on sorted IDs.
func GenerateSequenceKey(t1, t2 Tile) string {
	if t1.ID < t2.ID { return fmt.Sprintf("%d-%d", t1.ID, t2.ID) }
	return fmt.Sprintf("%d-%d", t2.ID, t1.ID)
}

// ==========================================================
// Action Possibility Checks
// ==========================================================

// CanDeclareRon checks if a player can win by Ron on the discarded tile.
func CanDeclareRon(player *Player, discardedTile Tile, gs *GameState) bool {
	// Furiten Checks
	if player.IsPermanentRiichiFuriten { return false }
	if player.IsFuriten {
		// If general Furiten (due to own discards or recently missed Ron), cannot Ron.
		// The logic in UpdateFuritenStatus and DiscardTile (for missed Ron) handles setting IsFuriten.
		return false
	}

	// Hand Completion Check
	// Player.Hand (13 tiles) + discardedTile (1 tile)
	tempConcealedHand := append([]Tile{}, player.Hand...)
	tempConcealedHand = append(tempConcealedHand, discardedTile)
	// sort.Sort(BySuitValue(tempConcealedHand)) // IsCompleteHand sorts its copy

	if !IsCompleteHand(tempConcealedHand, player.Melds) {
		return false // Hand shape is not complete
	}

	// Yaku Check
	// Ryanhan Shibari (Two-Han Minimum if Honba >= 5)
	if gs.Honba >= RyanhanShibariHonbaThreshold {
		yakuResults, _ := IdentifyYaku(player, discardedTile, false, gs) // isTsumo = false
		hanWithoutDora := 0
		for _, yr := range yakuResults {
			if !strings.HasPrefix(yr.Name, "Dora") { hanWithoutDora += yr.Han }
		}
		if hanWithoutDora < 2 {
			// gs.AddToGameLog(fmt.Sprintf("Debug: %s cannot Ron due to Ryanhan Shibari (needs 2 Han excluding Dora, has %d).", player.Name, hanWithoutDora))
			return false
		}
	} else { // Standard Yaku check (at least 1 Han from any source)
		_, han := IdentifyYaku(player, discardedTile, false, gs) // isTsumo = false
		if han == 0 {
			// gs.AddToGameLog(fmt.Sprintf("Debug: Ron possible shape for %s, but no Yaku found.", player.Name))
			return false // No Yaku, cannot Ron
		}
	}
	return true // Hand is complete AND has Yaku (and passes Ryanhan Shibari if applicable)
}

// CanDeclareTsumo checks if the player can win by Tsumo after drawing.
// Assumes player.Hand includes the drawn tile (player.JustDrawnTile).
func CanDeclareTsumo(player *Player, gs *GameState) bool {
	// Check total tiles (player.Hand should be 14-equivalent based on melds)
	// IsCompleteHand will validate the structure based on hand tiles passed and melds.
	// player.Hand here should be the concealed part *including* the JustDrawnTile.
	if player.JustDrawnTile == nil {
		gs.AddToGameLog(fmt.Sprintf("Error in CanDeclareTsumo: %s's JustDrawnTile is nil.", player.Name))
		return false // Cannot determine Tsumo without knowing the drawn tile
	}
	// Furiten does not prevent Tsumo, even permanent Riichi Furiten.

	// Hand Completion Check (player.Hand already includes the drawn tile)
	if !IsCompleteHand(player.Hand, player.Melds) {
		return false
	}

	// Yaku Check (using player.JustDrawnTile as the agariHai for Tsumo)
	// Ryanhan Shibari
	if gs.Honba >= RyanhanShibariHonbaThreshold {
		yakuResults, _ := IdentifyYaku(player, *player.JustDrawnTile, true, gs) // isTsumo = true
		hanWithoutDora := 0
		for _, yr := range yakuResults {
			if !strings.HasPrefix(yr.Name, "Dora") { hanWithoutDora += yr.Han }
		}
		if hanWithoutDora < 2 { return false }
	} else { // Standard Yaku check
		_, han := IdentifyYaku(player, *player.JustDrawnTile, true, gs) // isTsumo = true
		if han == 0 {
			// gs.AddToGameLog(fmt.Sprintf("Debug: Tsumo possible shape for %s, but no Yaku found.", player.Name))
			return false
		}
	}
	return true
}

// CanDeclarePon checks if a player can call Pon on a discarded tile.
func CanDeclarePon(player *Player, discardedTile Tile) bool {
	if player.IsRiichi { return false } // Cannot make open calls if in Riichi
	count := 0
	for _, tile := range player.Hand {
		if tile.Suit == discardedTile.Suit && tile.Value == discardedTile.Value {
			count++
		}
	}
	return count >= 2
}

// CanDeclareChi checks if a player can call Chi on a discarded tile.
func CanDeclareChi(player *Player, discardedTile Tile) bool {
	if player.IsRiichi { return false } // Cannot make open calls if in Riichi
	if IsHonor(discardedTile) { return false } // Cannot Chi honor tiles

	val, suit, hand := discardedTile.Value, discardedTile.Suit, player.Hand
	// Pattern 1: Need (Value-2, Value-1)
	if val >= 3 && HasTileWithValue(hand, suit, val-2) && HasTileWithValue(hand, suit, val-1) { return true }
	// Pattern 2: Need (Value-1, Value+1)
	if val >= 2 && val <= 8 && HasTileWithValue(hand, suit, val-1) && HasTileWithValue(hand, suit, val+1) { return true }
	// Pattern 3: Need (Value+1, Value+2)
	if val <= 7 && HasTileWithValue(hand, suit, val+1) && HasTileWithValue(hand, suit, val+2) { return true }
	return false
}

// CanDeclareDaiminkan checks specifically for calling Kan on another player's discard.
func CanDeclareDaiminkan(player *Player, discardedTile Tile) bool {
	if player.IsRiichi { return false } // Riichi players cannot make open calls (Daiminkan)

	// Check Ssuukantsu (4 Kans by ONE player is Yakuman, Suukaikan by multiple is abortive)
	// gs.TotalKansDeclaredThisRound is for Suukaikan. Player's own Kan count matters for 5th Kan.
	numPlayerKans := 0
	for _, m := range player.Melds { if strings.Contains(m.Type, "Kan") { numPlayerKans++ } }
	if numPlayerKans >= 4 { return false } // Cannot declare a 5th Kan

	countInHand := 0
	for _, t := range player.Hand {
		if t.Suit == discardedTile.Suit && t.Value == discardedTile.Value { countInHand++ }
	}
	return countInHand == 3
}

// compareTileSlicesUnordered checks if two slices of Tiles contain the same set of tile types.
// Used for checking if Riichi waits change.
func compareTileSlicesUnordered(s1, s2 []Tile) bool {
	if len(s1) != len(s2) { return false }
	counts1 := make(map[string]int)
	counts2 := make(map[string]int)
	for _, t := range s1 { counts1[fmt.Sprintf("%s-%d-%t", t.Suit, t.Value, t.IsRed)]++ } // Include IsRed for exact match
	for _, t := range s2 { counts2[fmt.Sprintf("%s-%d-%t", t.Suit, t.Value, t.IsRed)]++ }
	if len(counts1) != len(counts2) { return false }
	for key, count1 := range counts1 {
		if counts2[key] != count1 { return false }
	}
	return true
}


// checkWaitChangeForRiichiKan checks if a Kan declaration would change a Riichi player's waits.
// This is a complex check. A placeholder implementation.
func checkWaitChangeForRiichiKan(player *Player, gs *GameState, kanTile Tile, kanType string) bool {
	if !player.IsRiichi || len(player.RiichiDeclaredWaits) == 0 {
		return false // Not applicable or no stored waits to compare against
	}

	// 1. Create a deep copy of the player's current hand and melds to simulate the Kan.
	tempPlayerHand := make([]Tile, len(player.Hand)); copy(tempPlayerHand, player.Hand)
	tempPlayerMelds := make([]Meld, len(player.Melds)); copy(tempPlayerMelds, player.Melds)
	
	// 2. Simulate performing the Kan on the temporary structures.
	switch kanType {
	case "Ankan":
		// Find 4 'kanTile' in tempPlayerHand and remove them. Add Ankan to tempPlayerMelds.
		// This logic needs to be robust.
		indicesToKan := []int{}
		for i, t := range tempPlayerHand {
			if t.Suit == kanTile.Suit && t.Value == kanTile.Value {
				indicesToKan = append(indicesToKan, i)
			}
		}
		if len(indicesToKan) < 4 { return true } // Should not happen if CanDeclareKan was true
		tempPlayerHand = RemoveTilesByIndices(tempPlayerHand, indicesToKan[:4]) // Remove first 4 found
		newAnkanMeld := Meld{Type: "Ankan", Tiles: []Tile{kanTile, kanTile, kanTile, kanTile}, IsConcealed: true}
		tempPlayerMelds = append(tempPlayerMelds, newAnkanMeld)

	case "Shouminkan":
		// Find 'kanTile' in tempPlayerHand and remove it. Find matching Pon in tempPlayerMelds and upgrade it.
		idxToRemove := -1
		for i, t := range tempPlayerHand {
			if t.Suit == kanTile.Suit && t.Value == kanTile.Value {
				idxToRemove = i; break
			}
		}
		if idxToRemove == -1 { return true } // Tile not in hand for Shouminkan
		tempPlayerHand = RemoveTilesByIndices(tempPlayerHand, []int{idxToRemove})

		ponFoundAndUpgraded := false
		for i, m := range tempPlayerMelds {
			if m.Type == "Pon" && m.Tiles[0].Suit == kanTile.Suit && m.Tiles[0].Value == kanTile.Value {
				tempPlayerMelds[i].Type = "Shouminkan"
				tempPlayerMelds[i].Tiles = append(tempPlayerMelds[i].Tiles, kanTile) // Add the 4th tile
				sort.Sort(BySuitValue(tempPlayerMelds[i].Tiles))
				ponFoundAndUpgraded = true; break
			}
		}
		if !ponFoundAndUpgraded { return true } // No matching Pon to upgrade
	default:
		return true // Unknown Kan type for this check, assume waits change for safety
	}

	// 3. Find new waits with the temporary hand state.
	// Note: After Kan, hand size reduces by 1 (Ankan) or stays same (Shouminkan, but one tile moved from hand to meld).
	// FindTenpaiWaits expects the concealed part of a 13-tile equivalent hand.
	// If Ankan, tempPlayerHand is now 13 - 1 = 12 tiles (relative to before draw).
	// Rinshan draw will bring it back to 13 for next discard.
	// For wait check *after* Kan declaration but *before* Rinshan draw, the hand is smaller.
	// This is complex. The most common rule is: Ankan is fine if the 4 tiles were already in hand.
	// Shouminkan is fine if it adds to an existing Pon and doesn't "create" new waits.

	// Simplified placeholder: Assume for now Ankan with drawn tile doesn't change waits if other 3 were a triplet.
	// Assume Shouminkan adding drawn tile to existing Pon doesn't change waits.
	// A full check is much more involved.
	// return false // Placeholder: For now, assume common safe Kans don't change waits.
	// This needs to be accurate:
	newWaits := FindTenpaiWaits(tempPlayerHand, tempPlayerMelds)
	
	// 4. Compare newWaits with player.RiichiDeclaredWaits.
	return !compareTileSlicesUnordered(player.RiichiDeclaredWaits, newWaits)
}


// CanDeclareKanOnDraw checks if the player can declare Ankan or Shouminkan using the drawn tile.
// Assumes player.Hand includes the drawn tile (player.JustDrawnTile).
func CanDeclareKanOnDraw(player *Player, drawnTile Tile, gs *GameState) (string, Tile) {
	// Check for Suukaikan / Max Kans by player
	numPlayerKans := 0; for _, m := range player.Melds { if strings.Contains(m.Type, "Kan") { numPlayerKans++ } }
	if numPlayerKans >= 4 { return "", Tile{} } // Player cannot make a 5th Kan
	if gs.TotalKansDeclaredThisRound >= 4 && !CheckSuukantsu(player) { // Suukaikan condition for abort
		// This means 4 kans by multiple players. If a 5th kan is attempted by anyone, and rinshan fails, it's abort.
		// This check is more about "is the game in a state where another Kan could trigger abort".
	}

	// Check for Ankan (4 identical tiles in hand including the draw)
	countInHandForAnkan := 0
	for _, t := range player.Hand { // player.Hand already includes drawnTile
		if t.Suit == drawnTile.Suit && t.Value == drawnTile.Value { countInHandForAnkan++ }
	}
	if countInHandForAnkan == 4 {
		if player.IsRiichi && checkWaitChangeForRiichiKan(player, gs, drawnTile, "Ankan") {
			// gs.AddToGameLog(fmt.Sprintf("%s Ankan with drawn %s would change Riichi waits. Denied.", player.Name, drawnTile.Name))
			return "", Tile{}
		}
		return "Ankan", drawnTile
	}

	// Check for Shouminkan (add drawn tile to existing Pon)
	for _, meld := range player.Melds {
		if meld.Type == "Pon" {
			ponTile := meld.Tiles[0]
			if drawnTile.Suit == ponTile.Suit && drawnTile.Value == ponTile.Value {
				if player.IsRiichi && checkWaitChangeForRiichiKan(player, gs, drawnTile, "Shouminkan") {
					// gs.AddToGameLog(fmt.Sprintf("%s Shouminkan with drawn %s would change Riichi waits. Denied.", player.Name, drawnTile.Name))
					return "", Tile{}
				}
				return "Shouminkan", drawnTile
			}
		}
	}
	return "", Tile{}
}

// CanDeclareKanOnHand checks for Ankan or Shouminkan using tiles currently in hand/melds (not necessarily just drawn).
// `checkTile` is one representative tile from the potential Kan group.
func CanDeclareKanOnHand(player *Player, checkTile Tile, gs *GameState) (string, Tile) {
	numPlayerKans := 0; for _, m := range player.Melds { if strings.Contains(m.Type, "Kan") { numPlayerKans++ } }
	if numPlayerKans >= 4 { return "", Tile{} }
	if gs.TotalKansDeclaredThisRound >= 4 && !CheckSuukantsu(player) { /* Potential Suukaikan */ }

	// Check for Ankan (4 identical tiles currently in hand)
	countInHand := 0
	for _, t := range player.Hand {
		if t.Suit == checkTile.Suit && t.Value == checkTile.Value { countInHand++ }
	}
	if countInHand == 4 {
		if player.IsRiichi && checkWaitChangeForRiichiKan(player, gs, checkTile, "Ankan") { return "", Tile{} }
		return "Ankan", checkTile
	}

	// Check for Shouminkan (1 tile in hand + existing Pon)
	hasTileInHandForShouminkan := false
	for _, t := range player.Hand {
		if t.Suit == checkTile.Suit && t.Value == checkTile.Value { hasTileInHandForShouminkan = true; break }
	}
	if hasTileInHandForShouminkan {
		for _, meld := range player.Melds {
			if meld.Type == "Pon" {
				ponTile := meld.Tiles[0]
				if checkTile.Suit == ponTile.Suit && checkTile.Value == ponTile.Value {
					if player.IsRiichi && checkWaitChangeForRiichiKan(player, gs, checkTile, "Shouminkan") { return "", Tile{} }
					return "Shouminkan", checkTile // checkTile is the tile from hand to add
				}
			}
		}
	}
	return "", Tile{}
}

// FindRiichiOptions iterates through a 14-tile hand and finds all discards that result in Tenpai.
func FindRiichiOptions(hand14 []Tile, melds []Meld) []RiichiOption {
	options := []RiichiOption{}
	if len(hand14) != HandSize+1 { return options } // Must be 14 tiles (13 + draw)

	isConcealedHand := true // Check if hand is truly concealed for Riichi
	for _, m := range melds { if !m.IsConcealed { isConcealedHand = false; break } } // Ankans are concealed
	if !isConcealedHand { return options }

	for i := 0; i < len(hand14); i++ {
		discardCandidate := hand14[i]
		tempHand13 := make([]Tile, 0, HandSize)
		for j, t := range hand14 { if i != j { tempHand13 = append(tempHand13, t) } }
		// sort.Sort(BySuitValue(tempHand13)) // IsTenpai will sort

		if IsTenpai(tempHand13, melds) {
			waits := FindTenpaiWaits(tempHand13, melds)
			if len(waits) > 0 {
				options = append(options, RiichiOption{
					DiscardIndex: i, DiscardTile: discardCandidate, Waits: waits,
				})
			}
		}
	}
	return options
}

// CanDeclareRiichi checks if the player can declare Riichi.
func CanDeclareRiichi(player *Player, gs *GameState) (bool, []RiichiOption) {
	options := []RiichiOption{}
	if player.IsRiichi { return false, options } // Already in Riichi

	isConcealed := true // Hand must be concealed (only Ankans allowed as "melds")
	for _, m := range player.Melds { if !m.IsConcealed { isConcealed = false; break } }
	if !isConcealed { return false, options }

	if player.Score < RiichiBet { return false, options } // Not enough points
	if len(gs.Wall) < 4 { return false, options } // Not enough wall tiles for Riichi (Ippatsu, Ura)
	if len(player.Hand) != HandSize+1 { // Must be holding 13 + drawn tile
		// gs.AddToGameLog(fmt.Sprintf("Debug CanDeclareRiichi: %s hand size %d, expected %d", player.Name, len(player.Hand), HandSize+1))
		return false, options
	}

	options = FindRiichiOptions(player.Hand, player.Melds)
	return len(options) > 0, options // Can Riichi if any valid Tenpai discards found
}

// CheckKyuushuuKyuuhai (Nine Different Terminals/Honors on First Uninterrupted Draw).
func CheckKyuushuuKyuuhai(hand []Tile, melds []Meld) bool {
	// Condition: Player's first turn, no calls made by anyone yet, no melds by player.
	// `main.go` checks these conditions (`!player.HasDrawnFirstTileThisRound && !gs.AnyCallMadeThisRound`).
	if len(melds) > 0 { return false } // Must have no melds (implicit from no calls)
	if len(hand) != 13 { return false } // Checked with the initial 13 tiles before first draw

	uniqueTerminalsAndHonors := make(map[string]bool)
	count := 0
	for _, tile := range hand {
		if IsTerminalOrHonor(tile) {
			// Use tile Name (or Suit+Value string) as key for uniqueness of *type*
			key := fmt.Sprintf("%s-%d", tile.Suit, tile.Value) // Ensures 1m is different from 1p
			if !uniqueTerminalsAndHonors[key] {
				uniqueTerminalsAndHonors[key] = true
				count++
			}
		}
	}
	return count >= 9
}

// --- Abortive Draw Condition Checks ---

// CheckSsuufonRenda (Four Players Discard Same Wind on First Uninterrupted Turn).
// gs.FirstTurnDiscards must be populated correctly.
func CheckSsuufonRenda(gs *GameState) bool {
	if gs.FirstTurnDiscardCount < 4 { return false } // Not all 4 players made their first discard yet
	
	firstDiscard := gs.FirstTurnDiscards[0] // Assumes gs.FirstTurnDiscards indexed by initial seat order
	if firstDiscard.Suit != "Wind" { return false }

	for i := 1; i < 4; i++ {
		if gs.FirstTurnDiscards[i].Suit != "Wind" || gs.FirstTurnDiscards[i].Value != firstDiscard.Value {
			return false // Not all were the same wind tile
		}
	}
	gs.AddToGameLog("Ssuufon Renda condition met (4 same first wind discards).")
	return true
}

// CheckSuuRiichi (Four Players Declare Riichi).
// gs.DeclaredRiichiPlayerIndices map must be up to date.
func CheckSuuRiichi(gs *GameState) bool {
	count := 0
	for _, declared := range gs.DeclaredRiichiPlayerIndices {
		if declared { count++ }
	}
	if count == 4 {
		gs.AddToGameLog("Suu Riichi condition met (4 players declared Riichi).")
		return true
	}
	return false
}

// CheckSanchahou (Three Players Ron on the Same Discard).
// gs.SanchahouRonners list must be populated by DiscardTile.
func CheckSanchahou(gs *GameState) bool {
	if len(gs.SanchahouRonners) >= 3 {
		gs.AddToGameLog("Sanchahou condition met (3+ Ron declarations on same discard).")
		return true
	}
	return false
}

// CheckSuukaikan (Four Kans by Different Players resulting in no more Rinshan tiles).
// This function checks if the *conditions* for Suukaikan are met (4+ Kans by >=2 players).
// The actual abortive draw happens if DrawRinshanTile then fails.
func CheckSuukaikan(gs *GameState) bool {
	if gs.TotalKansDeclaredThisRound < 4 { return false }

	kansByPlayer := make(map[int]int) // Player index -> count of their Kans
	playersMakingKans := 0
	for playerIdx, p := range gs.Players {
		playerKanCount := 0
		for _, m := range p.Melds {
			if strings.Contains(m.Type, "Kan") { playerKanCount++ }
		}
		if playerKanCount > 0 {
			kansByPlayer[playerIdx] = playerKanCount
			playersMakingKans++
		}
		if playerKanCount == 4 { // One player has 4 Kans
			return false // This is SuuKANTSU (Yakuman), not Suukaikan abortive draw.
		}
	}

	if gs.TotalKansDeclaredThisRound >= 4 && playersMakingKans >= 2 {
		// gs.AddToGameLog("Suukaikan condition (4+ Kans by >=2 players) is met. Abort if next Rinshan fails.")
		return true
	}
	return false
}

// CheckSuukantsu (Four Kans YAKUMAN by a single player).
// This is a Yaku check, distinct from Suukaikan abortive draw.
func CheckSuukantsu(player *Player) bool { // Note: This is the YAKU check
	kanCount := 0
	for _, meld := range player.Melds {
		if strings.Contains(meld.Type, "Kan") { kanCount++ }
	}
	return kanCount == 4
}