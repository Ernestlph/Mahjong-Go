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
		return false
	}
	expectedHandTiles := groupsNeeded*3 + pairsNeeded*2
	if len(handTilesForCheck) != expectedHandTiles {
		return false
	}

	handCopy := make([]Tile, len(handTilesForCheck))
	copy(handCopy, handTilesForCheck)
	sort.Sort(BySuitValue(handCopy))

	isEffectivelyConcealed := true
	if numMelds > 0 {
		for _, m := range melds {
			if m.Type != "Ankan" {
				isEffectivelyConcealed = false
				break
			}
		}
	}

	if isEffectivelyConcealed && numMelds == 0 && len(handTilesForCheck) == 14 {
		if IsKokushiMusou(handCopy) {
			return true
		}
		if IsChiitoitsu(handCopy) {
			return true
		}
	}
	return CheckStandardHandRecursive(handCopy, groupsNeeded, pairsNeeded)
}

// CheckStandardHandRecursive attempts to find `groupsNeeded` groups (Pung/Chi)
// and `pairsNeeded` pairs from the `currentHand` tiles. Assumes `currentHand` is sorted.
func CheckStandardHandRecursive(currentHand []Tile, groupsNeeded int, pairsNeeded int) bool {
	if len(currentHand) == 0 && groupsNeeded == 0 && pairsNeeded == 0 {
		return true
	}
	if groupsNeeded < 0 || pairsNeeded < 0 || len(currentHand) < (groupsNeeded*3+pairsNeeded*2) {
		return false
	}
	if len(currentHand) == 0 && (groupsNeeded > 0 || pairsNeeded > 0) {
		return false
	}

	if pairsNeeded > 0 && len(currentHand) >= 2 {
		if currentHand[0].Suit == currentHand[1].Suit && currentHand[0].Value == currentHand[1].Value {
			if CheckStandardHandRecursive(currentHand[2:], groupsNeeded, pairsNeeded-1) {
				return true
			}
		}
	}

	if groupsNeeded > 0 && len(currentHand) >= 3 {
		if currentHand[0].Suit == currentHand[1].Suit && currentHand[0].Value == currentHand[1].Value &&
			currentHand[0].Suit == currentHand[2].Suit && currentHand[0].Value == currentHand[2].Value {
			if CheckStandardHandRecursive(currentHand[3:], groupsNeeded-1, pairsNeeded) {
				return true
			}
		}
	}

	if groupsNeeded > 0 && len(currentHand) >= 3 &&
		(IsSimple(currentHand[0]) || (IsTerminal(currentHand[0]) && currentHand[0].Value <= 7)) { // Corrected parentheses for logic
		if currentHand[0].Suit != "Wind" && currentHand[0].Suit != "Dragon" {
			v1, s1 := currentHand[0].Value, currentHand[0].Suit
			idx2, idx3 := -1, -1

			for k := 1; k < len(currentHand); k++ {
				if currentHand[k].Suit == s1 && currentHand[k].Value == v1+1 {
					idx2 = k
					break
				}
			}
			if idx2 != -1 {
				for k := idx2 + 1; k < len(currentHand); k++ {
					if currentHand[k].Suit == s1 && currentHand[k].Value == v1+2 {
						idx3 = k
						break
					}
				}
			}

			if idx3 != -1 {
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
	return false
}

// IsKokushiMusou checks for the 13 Orphans hand (14 tiles version - pair wait).
func IsKokushiMusou(hand []Tile) bool {
	if len(hand) != 14 {
		return false
	}
	terminalsAndHonors := map[string]int{
		"Man 1": 0, "Man 9": 0, "Pin 1": 0, "Pin 9": 0, "Sou 1": 0, "Sou 9": 0,
		"East": 0, "South": 0, "West": 0, "North": 0,
		"White": 0, "Green": 0, "Red": 0,
	}
	requiredTypes := len(terminalsAndHonors)
	foundTypes, hasPair := 0, false
	tileCountsByName := make(map[string]int)
	for _, tile := range hand {
		baseName := strings.TrimPrefix(tile.Name, "Red ")
		tileCountsByName[baseName]++
	}
	for name, count := range tileCountsByName {
		_, isRequired := terminalsAndHonors[name]
		if isRequired {
			if count > 2 {
				return false
			}
			if count >= 1 {
				if terminalsAndHonors[name] == 0 {
					foundTypes++
				}
				terminalsAndHonors[name] = count
			}
			if count == 2 {
				if hasPair {
					return false
				}
				hasPair = true
			}
		} else {
			return false
		}
	}
	return foundTypes == requiredTypes && hasPair
}

// IsChiitoitsu checks for the Seven Pairs hand (14 tiles).
func IsChiitoitsu(hand []Tile) bool {
	if len(hand) != 14 {
		return false
	}
	tileCountsByID := make(map[int]int)
	for _, t := range hand {
		tileCountsByID[t.ID]++
	}
	pairCountByID := 0
	for _, count := range tileCountsByID {
		if count == 2 {
			pairCountByID++
		} else if count == 4 {
			pairCountByID += 2
		} else if count != 0 {
			return false
		}
	}
	return pairCountByID == 7
}

// IsTenpai checks if a 13-tile hand state (currentHand + melds) is one tile away from being complete.
func IsTenpai(currentHand []Tile, melds []Meld) bool {
	numKans := 0
	for _, m := range melds {
		if strings.Contains(m.Type, "Kan") {
			numKans++
		}
	}

	possibleTiles := GetAllPossibleTiles()
	for _, testTile := range possibleTiles {
		tempConcealedHandWithTestTile := append([]Tile{}, currentHand...)
		tempConcealedHandWithTestTile = append(tempConcealedHandWithTestTile, testTile)

		if IsCompleteHand(tempConcealedHandWithTestTile, melds) {
			return true
		}
	}
	return false
}

// FindTenpaiWaits returns a list of *unique tile types* that would complete the hand.
// Expects a 13-tile hand state (currentHand + melds).
func FindTenpaiWaits(currentHand []Tile, melds []Meld) []Tile {
	waits := []Tile{}
	possibleTiles := GetAllPossibleTiles()
	seenWaits := make(map[string]bool)

	for _, testTile := range possibleTiles {
		tempConcealedHandWithTestTile := append([]Tile{}, currentHand...)
		tempConcealedHandWithTestTile = append(tempConcealedHandWithTestTile, testTile)

		if IsCompleteHand(tempConcealedHandWithTestTile, melds) {
			waitKeyTile := testTile
			if waitKeyTile.IsRed {
				waitKeyTile.IsRed = false
				waitKeyTile.Name = strings.TrimPrefix(waitKeyTile.Name, "Red ")
			}
			waitKey := fmt.Sprintf("%s-%d", waitKeyTile.Suit, waitKeyTile.Value)
			if !seenWaits[waitKey] {
				waits = append(waits, testTile)
				seenWaits[waitKey] = true
			}
		}
	}
	sort.Sort(BySuitValue(waits))
	return waits
}

// FindPossibleChiSequences identifies the sets of *two hand tiles* needed to form Chi with the discard.
func FindPossibleChiSequences(player *Player, discardedTile Tile) [][]Tile {
	var sequences [][]Tile
	if IsHonor(discardedTile) {
		return sequences
	}

	val, suit, hand := discardedTile.Value, discardedTile.Suit, player.Hand
	findIndices := func(targetValue int) []int {
		indices := []int{}
		for i, tile := range hand {
			if tile.Suit == suit && tile.Value == targetValue {
				indices = append(indices, i)
			}
		}
		return indices
	}
	valM2Indices, valM1Indices := findIndices(val-2), findIndices(val-1)
	valP1Indices, valP2Indices := findIndices(val+1), findIndices(val+2)
	foundSequencesMap := make(map[string][]Tile)

	if val >= 3 && len(valM2Indices) > 0 && len(valM1Indices) > 0 {
		for _, idxM2 := range valM2Indices {
			for _, idxM1 := range valM1Indices {
				if idxM1 == idxM2 {
					continue
				}
				tile1, tile2 := hand[idxM2], hand[idxM1]
				seqKey := GenerateSequenceKey(tile1, tile2)
				foundSequencesMap[seqKey] = []Tile{tile1, tile2}
			}
		}
	}
	if val >= 2 && val <= 8 && len(valM1Indices) > 0 && len(valP1Indices) > 0 {
		for _, idxM1 := range valM1Indices {
			for _, idxP1 := range valP1Indices {
				if idxM1 == idxP1 {
					continue
				}
				tile1, tile2 := hand[idxM1], hand[idxP1]
				seqKey := GenerateSequenceKey(tile1, tile2)
				foundSequencesMap[seqKey] = []Tile{tile1, tile2}
			}
		}
	}
	if val <= 7 && len(valP1Indices) > 0 && len(valP2Indices) > 0 {
		for _, idxP1 := range valP1Indices {
			for _, idxP2 := range valP2Indices {
				if idxP1 == idxP2 {
					continue
				}
				tile1, tile2 := hand[idxP1], hand[idxP2]
				seqKey := GenerateSequenceKey(tile1, tile2)
				foundSequencesMap[seqKey] = []Tile{tile1, tile2}
			}
		}
	}
	for _, seq := range foundSequencesMap {
		sequences = append(sequences, seq)
	}
	sort.Slice(sequences, func(i, j int) bool {
		if sequences[i][0].ID != sequences[j][0].ID {
			return sequences[i][0].ID < sequences[j][0].ID
		}
		return sequences[i][1].ID < sequences[j][1].ID
	})
	return sequences
}

// GenerateSequenceKey creates a unique key for a pair of tiles based on sorted IDs.
func GenerateSequenceKey(t1, t2 Tile) string {
	if t1.ID < t2.ID {
		return fmt.Sprintf("%d-%d", t1.ID, t2.ID)
	}
	return fmt.Sprintf("%d-%d", t2.ID, t1.ID)
}

// ==========================================================
// Action Possibility Checks
// ==========================================================

// CanDeclareRon checks if a player can win by Ron on the discarded tile.
func CanDeclareRon(player *Player, discardedTile Tile, gs *GameState) bool {
	if player.IsPermanentRiichiFuriten {
		return false
	}
	if player.IsFuriten {
		return false
	}

	tempConcealedHand := append([]Tile{}, player.Hand...)
	tempConcealedHand = append(tempConcealedHand, discardedTile)

	if !IsCompleteHand(tempConcealedHand, player.Melds) {
		return false
	}

	if gs.Honba >= RyanhanShibariHonbaThreshold {
		yakuResults, _ := IdentifyYaku(player, discardedTile, false, gs)
		hanWithoutDora := 0
		for _, yr := range yakuResults {
			if !strings.HasPrefix(yr.Name, "Dora") {
				hanWithoutDora += yr.Han
			}
		}
		if hanWithoutDora < 2 {
			return false
		}
	} else {
		_, han := IdentifyYaku(player, discardedTile, false, gs)
		if han == 0 {
			return false
		}
	}
	return true
}

// CanDeclareTsumo checks if the player can win by Tsumo after drawing.
// Assumes player.Hand includes the drawn tile (player.JustDrawnTile).
func CanDeclareTsumo(player *Player, gs *GameState) bool {
	if player.JustDrawnTile == nil {
		gs.AddToGameLog(fmt.Sprintf("Error in CanDeclareTsumo: %s's JustDrawnTile is nil.", player.Name))
		return false
	}

	if !IsCompleteHand(player.Hand, player.Melds) {
		return false
	}

	if gs.Honba >= RyanhanShibariHonbaThreshold {
		yakuResults, _ := IdentifyYaku(player, *player.JustDrawnTile, true, gs)
		hanWithoutDora := 0
		for _, yr := range yakuResults {
			if !strings.HasPrefix(yr.Name, "Dora") {
				hanWithoutDora += yr.Han
			}
		}
		if hanWithoutDora < 2 {
			return false
		}
	} else {
		_, han := IdentifyYaku(player, *player.JustDrawnTile, true, gs)
		if han == 0 {
			return false
		}
	}
	return true
}

// CanDeclarePon checks if a player can call Pon on a discarded tile.
func CanDeclarePon(player *Player, discardedTile Tile) bool {
	if player.IsRiichi {
		return false
	}
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
	if player.IsRiichi {
		return false
	}
	if IsHonor(discardedTile) {
		return false
	}

	val, suit, hand := discardedTile.Value, discardedTile.Suit, player.Hand
	if val >= 3 && HasTileWithValue(hand, suit, val-2) && HasTileWithValue(hand, suit, val-1) {
		return true
	}
	if val >= 2 && val <= 8 && HasTileWithValue(hand, suit, val-1) && HasTileWithValue(hand, suit, val+1) {
		return true
	}
	if val <= 7 && HasTileWithValue(hand, suit, val+1) && HasTileWithValue(hand, suit, val+2) {
		return true
	}
	return false
}

// CanDeclareDaiminkan checks specifically for calling Kan on another player's discard.
func CanDeclareDaiminkan(player *Player, discardedTile Tile) bool {
	if player.IsRiichi {
		return false
	}

	numPlayerKans := 0
	for _, m := range player.Melds {
		if strings.Contains(m.Type, "Kan") {
			numPlayerKans++
		}
	}
	if numPlayerKans >= 4 {
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

// compareTileSlicesUnordered checks if two slices of Tiles contain the same set of tile types.
func compareTileSlicesUnordered(s1, s2 []Tile) bool {
	if len(s1) != len(s2) {
		return false
	}
	counts1 := make(map[string]int)
	counts2 := make(map[string]int)
	for _, t := range s1 {
		counts1[fmt.Sprintf("%s-%d-%t", t.Suit, t.Value, t.IsRed)]++
	}
	for _, t := range s2 {
		counts2[fmt.Sprintf("%s-%d-%t", t.Suit, t.Value, t.IsRed)]++
	}
	if len(counts1) != len(counts2) {
		return false
	}
	for key, count1 := range counts1 {
		if counts2[key] != count1 {
			return false
		}
	}
	return true
}

// checkWaitChangeForRiichiKan checks if a Kan declaration would change a Riichi player's waits.
func checkWaitChangeForRiichiKan(player *Player, gs *GameState, kanTile Tile, kanType string) bool {
	if !player.IsRiichi || len(player.RiichiDeclaredWaits) == 0 {
		return false
	}

	tempPlayerHand := make([]Tile, len(player.Hand))
	copy(tempPlayerHand, player.Hand)
	tempPlayerMelds := make([]Meld, len(player.Melds))
	copy(tempPlayerMelds, player.Melds)

	switch kanType {
	case "Ankan":
		indicesToKan := []int{}
		for i, t := range tempPlayerHand {
			if t.Suit == kanTile.Suit && t.Value == kanTile.Value {
				indicesToKan = append(indicesToKan, i)
			}
		}
		if len(indicesToKan) < 4 {
			return true
		}
		tempPlayerHand = RemoveTilesByIndices(tempPlayerHand, indicesToKan[:4])
		newAnkanMeld := Meld{Type: "Ankan", Tiles: []Tile{kanTile, kanTile, kanTile, kanTile}, IsConcealed: true}
		tempPlayerMelds = append(tempPlayerMelds, newAnkanMeld)

	case "Shouminkan":
		idxToRemove := -1
		for i, t := range tempPlayerHand {
			if t.Suit == kanTile.Suit && t.Value == kanTile.Value {
				idxToRemove = i
				break
			}
		}
		if idxToRemove == -1 {
			return true
		}
		tempPlayerHand = RemoveTilesByIndices(tempPlayerHand, []int{idxToRemove})

		ponFoundAndUpgraded := false
		for i, m := range tempPlayerMelds {
			if m.Type == "Pon" && m.Tiles[0].Suit == kanTile.Suit && m.Tiles[0].Value == kanTile.Value {
				tempPlayerMelds[i].Type = "Shouminkan"
				tempPlayerMelds[i].Tiles = append(tempPlayerMelds[i].Tiles, kanTile)
				sort.Sort(BySuitValue(tempPlayerMelds[i].Tiles))
				ponFoundAndUpgraded = true
				break
			}
		}
		if !ponFoundAndUpgraded {
			return true
		}
	default:
		return true
	}

	newWaits := FindTenpaiWaits(tempPlayerHand, tempPlayerMelds)
	return !compareTileSlicesUnordered(player.RiichiDeclaredWaits, newWaits)
}

// CanDeclareKanOnDraw checks if the player can declare Ankan or Shouminkan using the drawn tile.
func CanDeclareKanOnDraw(player *Player, drawnTile Tile, gs *GameState) (string, Tile) {
	numPlayerKans := 0
	for _, m := range player.Melds {
		if strings.Contains(m.Type, "Kan") {
			numPlayerKans++
		}
	}
	if numPlayerKans >= 4 {
		return "", Tile{}
	}
	if gs.TotalKansDeclaredThisRound >= 4 && !CheckSuukantsu(player) {
	}

	countInHandForAnkan := 0
	for _, t := range player.Hand {
		if t.Suit == drawnTile.Suit && t.Value == drawnTile.Value {
			countInHandForAnkan++
		}
	}
	if countInHandForAnkan == 4 {
		if player.IsRiichi && checkWaitChangeForRiichiKan(player, gs, drawnTile, "Ankan") {
			return "", Tile{}
		}
		return "Ankan", drawnTile
	}

	for _, meld := range player.Melds {
		if meld.Type == "Pon" {
			ponTile := meld.Tiles[0]
			if drawnTile.Suit == ponTile.Suit && drawnTile.Value == ponTile.Value {
				if player.IsRiichi && checkWaitChangeForRiichiKan(player, gs, drawnTile, "Shouminkan") {
					return "", Tile{}
				}
				return "Shouminkan", drawnTile
			}
		}
	}
	return "", Tile{}
}

// CanDeclareKanOnHand checks for Ankan or Shouminkan using tiles currently in hand/melds (not necessarily just drawn).
func CanDeclareKanOnHand(player *Player, checkTile Tile, gs *GameState) (string, Tile) {
	numPlayerKans := 0
	for _, m := range player.Melds {
		if strings.Contains(m.Type, "Kan") {
			numPlayerKans++
		}
	}
	if numPlayerKans >= 4 {
		return "", Tile{}
	}
	if gs.TotalKansDeclaredThisRound >= 4 && !CheckSuukantsu(player) {
	}

	countInHand := 0
	for _, t := range player.Hand {
		if t.Suit == checkTile.Suit && t.Value == checkTile.Value {
			countInHand++
		}
	}
	if countInHand == 4 {
		if player.IsRiichi && checkWaitChangeForRiichiKan(player, gs, checkTile, "Ankan") {
			return "", Tile{}
		}
		return "Ankan", checkTile
	}

	hasTileInHandForShouminkan := false
	for _, t := range player.Hand {
		if t.Suit == checkTile.Suit && t.Value == checkTile.Value {
			hasTileInHandForShouminkan = true
			break
		}
	}
	if hasTileInHandForShouminkan {
		for _, meld := range player.Melds {
			if meld.Type == "Pon" {
				ponTile := meld.Tiles[0]
				if checkTile.Suit == ponTile.Suit && checkTile.Value == ponTile.Value {
					if player.IsRiichi && checkWaitChangeForRiichiKan(player, gs, checkTile, "Shouminkan") {
						return "", Tile{}
					}
					return "Shouminkan", checkTile
				}
			}
		}
	}
	return "", Tile{}
}

// FindRiichiOptions iterates through a 14-tile hand and finds all discards that result in Tenpai.
func FindRiichiOptions(hand14 []Tile, melds []Meld) []RiichiOption {
	options := []RiichiOption{}
	if len(hand14) != HandSize+1 {
		return options
	}

	isConcealedHand := true
	for _, m := range melds {
		if !m.IsConcealed {
			isConcealedHand = false
			break
		}
	}
	if !isConcealedHand {
		return options
	}

	for i := 0; i < len(hand14); i++ {
		discardCandidate := hand14[i]
		tempHand13 := make([]Tile, 0, HandSize)
		for j, t := range hand14 {
			if i != j {
				tempHand13 = append(tempHand13, t)
			}
		}

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
	if player.IsRiichi {
		return false, options
	}

	isConcealed := true
	for _, m := range player.Melds {
		if !m.IsConcealed {
			isConcealed = false
			break
		}
	}
	if !isConcealed {
		return false, options
	}

	if player.Score < RiichiBet {
		return false, options
	}
	if len(gs.Wall) < 4 {
		return false, options
	}
	if len(player.Hand) != HandSize+1 {
		return false, options
	}

	options = FindRiichiOptions(player.Hand, player.Melds)
	return len(options) > 0, options
}

// CheckKyuushuuKyuuhai (Nine Different Terminals/Honors on First Uninterrupted Draw).
func CheckKyuushuuKyuuhai(hand []Tile, melds []Meld) bool {
	if len(melds) > 0 {
		return false
	}
	if len(hand) != 13 {
		return false
	}

	uniqueTerminalsAndHonors := make(map[string]bool)
	count := 0
	for _, tile := range hand {
		if IsTerminalOrHonor(tile) {
			key := fmt.Sprintf("%s-%d", tile.Suit, tile.Value)
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
func CheckSsuufonRenda(gs *GameState) bool {
	if gs.FirstTurnDiscardCount < 4 {
		return false
	}

	firstDiscard := gs.FirstTurnDiscards[0]
	if firstDiscard.Suit != "Wind" {
		return false
	}

	for i := 1; i < 4; i++ {
		if gs.FirstTurnDiscards[i].Suit != "Wind" || gs.FirstTurnDiscards[i].Value != firstDiscard.Value {
			return false
		}
	}
	gs.AddToGameLog("Ssuufon Renda condition met (4 same first wind discards).")
	return true
}

// CheckSuuRiichi (Four Players Declare Riichi).
func CheckSuuRiichi(gs *GameState) bool {
	count := 0
	for _, declared := range gs.DeclaredRiichiPlayerIndices {
		if declared {
			count++
		}
	}
	if count == 4 {
		gs.AddToGameLog("Suu Riichi condition met (4 players declared Riichi).")
		return true
	}
	return false
}

// CheckSanchahou (Three Players Ron on the Same Discard).
func CheckSanchahou(gs *GameState) bool {
	if len(gs.SanchahouRonners) >= 3 {
		gs.AddToGameLog("Sanchahou condition met (3+ Ron declarations on same discard).")
		return true
	}
	return false
}

// CheckSuukaikan (Four Kans by Different Players resulting in no more Rinshan tiles).
func CheckSuukaikan(gs *GameState) bool {
	if gs.TotalKansDeclaredThisRound < 4 {
		return false
	}

	kansByPlayer := make(map[int]int)
	playersMakingKans := 0
	for playerIdx, p := range gs.Players {
		playerKanCount := 0
		for _, m := range p.Melds {
			if strings.Contains(m.Type, "Kan") {
				playerKanCount++
			}
		}
		if playerKanCount > 0 {
			kansByPlayer[playerIdx] = playerKanCount
			playersMakingKans++
		}
		if playerKanCount == 4 {
			return false
		}
	}

	if gs.TotalKansDeclaredThisRound >= 4 && playersMakingKans >= 2 {
		return true
	}
	return false
}

// CheckSuukantsu (Four Kans YAKUMAN by a single player).
func CheckSuukantsu(player *Player) bool {
	kanCount := 0
	for _, meld := range player.Melds {
		if strings.Contains(meld.Type, "Kan") {
			kanCount++
		}
	}
	return kanCount == 4
}
