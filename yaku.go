// yaku.go
package main

import (
	"fmt"
	"sort"
	"strings" // For string operations like HasPrefix
)

// YakuResult holds the name and Han value of an identified Yaku.
type YakuResult struct {
	Name string
	Han  int
}

// IdentifyYaku analyzes the winning hand and conditions to determine all applicable Yaku and total Han.
func IdentifyYaku(player *Player, agariHai Tile, isTsumo bool, gs *GameState) ([]YakuResult, int) {
	var results []YakuResult // Use var for initial empty slice

	isMenzen := isMenzenchin(player, isTsumo, agariHai)
	allTiles := getAllTilesInHand(player, agariHai, isTsumo)

	if len(allTiles) != 14 {
		gs.AddToGameLog(fmt.Sprintf("Error in IdentifyYaku: Hand for %s has %d tiles, expected 14. Cannot evaluate Yaku.", player.Name, len(allTiles)))
		return []YakuResult{}, 0
	}

	// --- 1. Yakuman Checks ---
	var yakumanResults []YakuResult

	// Luck-based Yakuman first
	if ok, name, han := checkTenhou(player, gs, isTsumo); ok {
		addUniqueYakuman(&yakumanResults, YakuResult{name, han})
	}
	// Check next only if previous luck yakuman not found (they are mutually exclusive)
	if len(yakumanResults) == 0 {
		if ok, name, han := checkChihou(player, gs, isTsumo); ok {
			addUniqueYakuman(&yakumanResults, YakuResult{name, han})
		}
	}
	if len(yakumanResults) == 0 && !isTsumo { // Renhou is Ron only
		if ok, name, han := checkRenhou(player, gs, isTsumo); ok {
			addUniqueYakuman(&yakumanResults, YakuResult{name, han})
		}
	}

	// If a luck-based Yakuman was found, we typically stop and don't check structural ones,
	// unless specific rules allow stacking (e.g. Tenhou + Daisangen).
	// For now, if a luck yakuman is found, it is *the* yakuman.
	if len(yakumanResults) > 0 {
		finalLuckYakumanHan := 0
		for _, r := range yakumanResults {finalLuckYakumanHan += r.Han}
		gs.AddToGameLog(fmt.Sprintf("Luck-Based Yakuman Identified: %v. Total Han: %d", yakumanResults, finalLuckYakumanHan))
		return yakumanResults, finalLuckYakumanHan
	}


	// Structural Yakuman - if no luck yakuman found
	if ok, name, han := checkKokushiMusou(allTiles, agariHai); ok {
		addUniqueYakuman(&yakumanResults, YakuResult{name, han})
	}
	if ok, name, han := checkSuuankou(player, agariHai, isTsumo, isMenzen, allTiles); ok {
		addUniqueYakuman(&yakumanResults, YakuResult{name, han})
	}
	if ok, name, han := checkDaisangen(player, allTiles); ok {
		addUniqueYakuman(&yakumanResults, YakuResult{name, han})
	}

	daisuushiiFound := false // Flag to manage Daisuushii/Shousuushii precedence
	for _, yr := range yakumanResults { if yr.Name == "Daisuushii" { daisuushiiFound = true; break } }

	if !daisuushiiFound {
		if ok, name, han := checkDaisuushii(player, allTiles); ok {
			// If Daisuushii is found, remove Shousuushii if it was somehow added (shouldn't happen if checks are ordered)
			var tempResults []YakuResult
			for _, yr := range yakumanResults {
				if yr.Name != "Shousuushii" { tempResults = append(tempResults, yr) }
			}
			yakumanResults = tempResults
			addUniqueYakuman(&yakumanResults, YakuResult{name, han})
			daisuushiiFound = true
		}
	}
	if !daisuushiiFound { // Only check Shousuushii if Daisuushii was NOT found
		if ok, name, han := checkShousuushii(player, allTiles); ok {
			addUniqueYakuman(&yakumanResults, YakuResult{name, han})
		}
	}

	if ok, name, han := checkTsuuiisou(player, allTiles); ok {
		addUniqueYakuman(&yakumanResults, YakuResult{name, han})
	}
	if ok, name, han := checkChinroutou(player, allTiles); ok {
		addUniqueYakuman(&yakumanResults, YakuResult{name, han})
	}
	if ok, name, han := checkRyuuiisou(player, allTiles); ok {
		addUniqueYakuman(&yakumanResults, YakuResult{name, han})
	}
	if ok, name, han := checkChuurenPoutou(isMenzen, allTiles, agariHai); ok {
		addUniqueYakuman(&yakumanResults, YakuResult{name, han})
	}
	if ok, name, han := checkSuukantsu(player); ok {
		addUniqueYakuman(&yakumanResults, YakuResult{name, han})
	}

	if len(yakumanResults) > 0 {
		finalYakumanHan := 0
		// Sum Han for multiple distinct Yakuman (e.g. Daisangen + Tsuuiisou if allowed by ruleset)
		for _, r := range yakumanResults { finalYakumanHan += r.Han }
		gs.AddToGameLog(fmt.Sprintf("Structural Yakuman(s) Identified: %v. Total Han: %d", yakumanResults, finalYakumanHan))
		return yakumanResults, finalYakumanHan
	}

	// --- 2. Regular Yaku Checks (Only if NO Yakuman was found) ---
	var regularResults []YakuResult

	// --- 1 Han Yaku ---
	if ok, han := checkRiichi(player, gs); ok {
		regularResults = append(regularResults, YakuResult{"Riichi", han})
		if okDR, nameDR, hanDR := checkDoubleRiichi(player, gs); okDR {
			regularResults = append(regularResults, YakuResult{nameDR, hanDR})
		}
		if okIp, hanIp := checkIppatsu(player, gs); okIp {
			regularResults = append(regularResults, YakuResult{"Ippatsu", hanIp})
		}
	}

	if ok, han := checkMenzenTsumo(isTsumo, isMenzen); ok {
		regularResults = append(regularResults, YakuResult{"Menzen Tsumo", han})
	}

	pinfuAwarded := false
	if isMenzen {
		if ok, han := checkPinfu(player, agariHai, isMenzen, allTiles, gs); ok {
			regularResults = append(regularResults, YakuResult{"Pinfu", han})
			pinfuAwarded = true
		}
	}

	if ok, han := checkTanyao(allTiles); ok {
		regularResults = append(regularResults, YakuResult{"Tanyao", han})
	}

	if yakuhaiRes, _ := checkYakuhai(player, gs, allTiles); len(yakuhaiRes) > 0 {
		regularResults = append(regularResults, yakuhaiRes...)
	}

	if ok, name, han := checkHaiteiHoutei(gs, isTsumo); ok {
		regularResults = append(regularResults, YakuResult{name, han})
	}

	if ok, han := checkRinshanKaihou(gs, isTsumo); ok {
		regularResults = append(regularResults, YakuResult{"Rinshan Kaihou", han})
	}

	if !isTsumo { if ok, han := checkChankan(gs); ok { regularResults = append(regularResults, YakuResult{"Chankan", han}) } }

	// --- 2 Han Yaku ---
	chiitoitsuFound := false
	if !pinfuAwarded { // Pinfu and Chiitoitsu are mutually exclusive
		if ok, han := checkChiitoitsu(player, allTiles, isMenzen); ok {
			regularResults = append(regularResults, YakuResult{"Chiitoitsu", han})
			chiitoitsuFound = true
		}
	}

	if !chiitoitsuFound {
		if ok, han := checkToitoi(player, allTiles); ok {
			regularResults = append(regularResults, YakuResult{"Toitoi", han})
		}
		if ok, han := checkSanankou(player, agariHai, isTsumo, allTiles); ok {
			regularResults = append(regularResults, YakuResult{"Sanankou", han})
		}
		if ok, name, han := checkSanshokuDoukou(player, allTiles); ok {
			regularResults = append(regularResults, YakuResult{name, han})
		}
		if ok, han := checkShousangen(player, allTiles); ok {
			regularResults = append(regularResults, YakuResult{"Shousangen", han})
		}
	}

	if ok, name, han := checkSankantsu(player); ok {
		regularResults = append(regularResults, YakuResult{name, han})
	}
	
	if okHonro, hanHonro := checkHonroutou(allTiles); okHonro {
		if !chiitoitsuFound {
			regularResults = append(regularResults, YakuResult{"Honroutou", hanHonro})
		} else {
			// gs.AddToGameLog("Note: Hand is Chiitoitsu and also Honroutou structure; Honroutou not added as separate Yaku.")
		}
	}

	// --- 3+ Han Yaku ---
	if !chiitoitsuFound {
		if ok, name, han := checkSanshokuDoujun(player, isMenzen, allTiles); ok {
			regularResults = append(regularResults, YakuResult{name, han})
		}
		if ok, name, han := checkIttsuu(player, isMenzen, allTiles); ok {
			regularResults = append(regularResults, YakuResult{name, han})
		}

		ryanpeikouFound := false
		if isMenzen {
			if okRyan, hanRyan := checkRyanpeikou(player, isMenzen, allTiles); okRyan {
				regularResults = append(regularResults, YakuResult{"Ryanpeikou", hanRyan})
				ryanpeikouFound = true
			}
			if !ryanpeikouFound {
				if okIipe, hanIipe := checkIipeikou(player, isMenzen, allTiles); okIipe {
					regularResults = append(regularResults, YakuResult{"Iipeikou", hanIipe})
				}
			}
		}
		
		// Junchan Taiyou vs Honroutou: Mutually exclusive by definition
		// (Junchan requires simples, Honroutou forbids simples)
		// No explicit check needed if individual Yaku are correct.
		if ok, han := checkJunchan(player, isMenzen, allTiles); ok {
			regularResults = append(regularResults, YakuResult{"Junchan Taiyou", han})
		}

		chinitsuFound := false
		if okC, hanC := checkChinitsu(allTiles, isMenzen); okC {
			regularResults = append(regularResults, YakuResult{"Chinitsu", hanC})
			chinitsuFound = true
		}
		if !chinitsuFound {
			if okH, hanH := checkHonitsu(allTiles, isMenzen); okH {
				regularResults = append(regularResults, YakuResult{"Honitsu", hanH})
			}
		}
	}
	
	// Nagashi Mangan check: Typically scores at Ryuukyoku, not as a winning Yaku.
	// If rules allow it as a win:
	// if ok, name, han := checkNagashiMangan(player, gs); ok {
	// 	regularResults = []YakuResult{YakuResult{name, han}} // Nagashi often overrides other regular yaku
	// }

	// --- 3. Dora Calculation ---
	currentRegularHan := 0
	for _, r := range regularResults { currentRegularHan += r.Han }

	if currentRegularHan > 0 {
		doraCount := 0
		doraCount += countDora(allTiles, gs.DoraIndicators)
		doraCount += countRedDora(allTiles)
		if player.IsRiichi && len(gs.UraDoraIndicators) > 0 {
			doraCount += countDora(allTiles, gs.UraDoraIndicators)
		}
		if doraCount > 0 {
			regularResults = append(regularResults, YakuResult{fmt.Sprintf("Dora %d", doraCount), doraCount})
		}
	}

	// --- Final Han Summation & Result Consolidation ---
	finalHanSum := 0
	finalResultsToReturn := []YakuResult{}
	seenNames := make(map[string]int)

	for _, r := range regularResults {
		isUniqueOrAllowedMultiple := true
		if seenNames[r.Name] > 0 { // If name already seen
			if !(strings.HasPrefix(r.Name, "Yakuhai")) { // Non-Yakuhai Yaku should be unique by name
				isUniqueOrAllowedMultiple = false
			} else { // For Yakuhai, ensure it's a different specific Yakuhai
				alreadyAddedExactYakuhai := false
				for _, fr := range finalResultsToReturn {
					if fr.Name == r.Name { alreadyAddedExactYakuhai = true; break }
				}
				if alreadyAddedExactYakuhai { isUniqueOrAllowedMultiple = false }
			}
		}

		if isUniqueOrAllowedMultiple {
			finalResultsToReturn = append(finalResultsToReturn, r)
			finalHanSum += r.Han
			seenNames[r.Name]++
		}
	}
	
	if finalHanSum == 0 && len(finalResultsToReturn) > 0 && strings.HasPrefix(finalResultsToReturn[0].Name, "Dora") {
		gs.AddToGameLog(fmt.Sprintf("IdentifyYaku for %s resulted in only Dora (%v), no other actual Yaku. Invalid win.", player.Name, finalResultsToReturn))
		return []YakuResult{}, 0
	}
	if finalHanSum == 0 && len(finalResultsToReturn) == 0 {
		// This means no Yaku were found at all. CanDeclareRon/Tsumo should prevent this.
		// gs.AddToGameLog(fmt.Sprintf("IdentifyYaku for %s resulted in 0 Han and no Yaku.", player.Name))
	}

	return finalResultsToReturn, finalHanSum
}

// addUniqueYakuman adds a yakuman to the list only if its specific name isn't already present,
// or if the new one is a higher-valued version of an existing one (e.g. double yakuman).
func addUniqueYakuman(yakumanList *[]YakuResult, newYakuman YakuResult) {
	for i, existingYakuman := range *yakumanList {
		if existingYakuman.Name == newYakuman.Name {
			if newYakuman.Han > existingYakuman.Han { // New one is stronger version
				(*yakumanList)[i] = newYakuman // Replace
			}
			return // Either replaced or already have same/stronger
		}
		// Handle cases like "Kokushi Musou" vs "Kokushi Musou Juusanmenmachi"
		if (strings.HasPrefix(newYakuman.Name, existingYakuman.Name) && newYakuman.Han > existingYakuman.Han) ||
		   (strings.HasPrefix(existingYakuman.Name, newYakuman.Name) && existingYakuman.Han > newYakuman.Han) {
			// Complex case: one is a prefix of the other (e.g. base vs double version)
			// Keep the one with higher Han. If new is higher, replace. If existing is higher, do nothing.
			if newYakuman.Han > existingYakuman.Han {
				(*yakumanList)[i] = newYakuman
			}
			return
		}
	}
	*yakumanList = append(*yakumanList, newYakuman) // Add if truly new
}

// --- Helper Functions ---
func isMenzenchin(player *Player, isTsumo bool, agariHai Tile) bool {
	for _, meld := range player.Melds { if !meld.IsConcealed { return false } }
	return true
}

func getAllTilesInHand(player *Player, agariHai Tile, isTsumo bool) []Tile {
	allWinningTiles := []Tile{}
	allWinningTiles = append(allWinningTiles, player.Hand...)
	for _, meld := range player.Melds {
		allWinningTiles = append(allWinningTiles, meld.Tiles...)
	}
	if !isTsumo { // If Ron, ensure agariHai is added
		// Check if agariHai (by ID) is already effectively in the list from hand/melds
		// This is a safeguard, normally player.Hand for Ron is the 13-tile state
		isAgariAlreadyPresent := false
		for _, t := range allWinningTiles {
			if t.ID == agariHai.ID {
				isAgariAlreadyPresent = true
				break
			}
		}
		if !isAgariAlreadyPresent {
			allWinningTiles = append(allWinningTiles, agariHai)
		}
	}
	sort.Sort(BySuitValue(allWinningTiles))
	if len(allWinningTiles) != 14 && len(allWinningTiles) != 0 {
		// fmt.Printf("CRITICAL WARNING in getAllTilesInHand for %s: Resulted in %d tiles, expected 14. Hand: %v, Melds: %v, Agari: %s, Tsumo: %v\n",
		//  player.Name, len(allWinningTiles), TilesToNames(player.Hand), FormatMeldsForDisplay(player.Melds), agariHai.Name, isTsumo)
	}
	return allWinningTiles
}

func getDoraTile(indicator Tile) Tile {
	dora := indicator; dora.IsRed = false; dora.ID = -1
	suit, value := indicator.Suit, indicator.Value
	switch suit {
	case "Man", "Pin", "Sou":
		if value == 9 { dora.Value = 1 } else { dora.Value = value + 1 }
		dora.Name = fmt.Sprintf("%s %d", suit, dora.Value)
	case "Wind":
		dora.Value = (value % 4) + 1
		winds := []string{"", "East", "South", "West", "North"}
		dora.Name = winds[dora.Value]
	case "Dragon":
		dora.Value = (value % 3) + 1
		dragons := []string{"", "White", "Green", "Red"}
		dora.Name = dragons[dora.Value]
	}
	return dora
}

func countDora(handTiles []Tile, indicators []Tile) int {
	count := 0; if len(indicators) == 0 { return 0 }
	for _, indicator := range indicators {
		doraValueTile := getDoraTile(indicator)
		for _, handTile := range handTiles {
			if handTile.Suit == doraValueTile.Suit && handTile.Value == doraValueTile.Value {
				count++
			}
		}
	}
	return count
}

func countRedDora(handTiles []Tile) int {
	count := 0; for _, tile := range handTiles { if tile.IsRed { count++ } }; return count
}

func groupContainsTileID(group DecomposedGroup, tileID int) bool {
	for _, t := range group.Tiles { if t.ID == tileID { return true } }
	return false
}

func sequencesAreEqual(seq1Tiles, seq2Tiles []Tile) bool {
	if len(seq1Tiles) != 3 || len(seq2Tiles) != 3 { return false }
	for i := 0; i < 3; i++ {
		if seq1Tiles[i].Suit != seq2Tiles[i].Suit || seq1Tiles[i].Value != seq2Tiles[i].Value { return false }
	}
	return true
}

func WindValueFromName(windName string) int {
	switch windName {
	case "East": return 1; case "South": return 2; case "West": return 3; case "North": return 4
	}
	return 0
}

// --- Individual Yaku Check Functions ---

// == YAKUMAN ==
func checkKokushiMusou(allTiles []Tile, agariHai Tile) (bool, string, int) {
	if !IsKokushiMusou(allTiles) { return false, "", 0 }
	agariHaiCount := 0
	for _, tile := range allTiles {
		if tile.Suit == agariHai.Suit && tile.Value == agariHai.Value { agariHaiCount++ }
	}
	if agariHaiCount == 2 {
		tempHand13 := make([]Tile, 0, 13); removedOneAgariHai := false
		for _, t := range allTiles {
			if !removedOneAgariHai && t.Suit == agariHai.Suit && t.Value == agariHai.Value {
				removedOneAgariHai = true
			} else { tempHand13 = append(tempHand13, t) }
		}
		if len(tempHand13) == 13 {
			uniqueTileTypes := make(map[string]bool); allAreKokushiTypes := true
			for _, t := range tempHand13 {
				if !IsTerminalOrHonor(t) { allAreKokushiTypes = false; break }
				uniqueTileTypes[fmt.Sprintf("%s-%d", t.Suit, t.Value)] = true
			}
			if allAreKokushiTypes && len(uniqueTileTypes) == 13 {
				return true, "Kokushi Musou Juusanmenmachi", 26
			}
		}
	}
	return true, "Kokushi Musou", 13
}

func checkSuuankou(player *Player, agariHai Tile, isTsumo bool, isMenzen bool, allTiles []Tile) (bool, string, int) {
	if !isMenzen { return false, "", 0 }
	decomposition, success := DecomposeWinningHand(player, allTiles)
	if !success || decomposition == nil { return false, "", 0 }
	concealedPungKanCount := 0; var pairGroup DecomposedGroup; foundPairInDecomp := false
	for _, group := range decomposition {
		isEffectiveConcealedMeld := false
		if group.Type == TypeTriplet && group.IsConcealed {
			if !isTsumo && groupContainsTileID(group, agariHai.ID) {} else { isEffectiveConcealedMeld = true }
		} else if group.Type == TypeQuad && group.IsConcealed { isEffectiveConcealedMeld = true }
		else if group.Type == TypePair { if foundPairInDecomp { return false, "", 0 }; pairGroup = group; foundPairInDecomp = true }
		if isEffectiveConcealedMeld { concealedPungKanCount++ }
	}
	if concealedPungKanCount == 4 && foundPairInDecomp {
		if len(pairGroup.Tiles) == 2 && pairGroup.Tiles[0].Suit == agariHai.Suit && pairGroup.Tiles[0].Value == agariHai.Value {
			return true, "Suuankou Tanki", 26
		}
		return true, "Suuankou", 13
	}
	return false, "", 0
}

func checkDaisangen(player *Player, allTiles []Tile) (bool, string, int) {
	decomposition, success := DecomposeWinningHand(player, allTiles)
	if !success || decomposition == nil { return false, "", 0 }
	dragonsFound := map[int]bool{1:false, 2:false, 3:false}
	for _, group := range decomposition {
		if group.Type == TypeTriplet || group.Type == TypeQuad {
			if len(group.Tiles) > 0 && group.Tiles[0].Suit == "Dragon" {
				dragonsFound[group.Tiles[0].Value] = true
			}
		}
	}
	if dragonsFound[1] && dragonsFound[2] && dragonsFound[3] { return true, "Daisangen", 13 }
	return false, "", 0
}

func checkShousuushii(player *Player, allTiles []Tile) (bool, string, int) {
	decomposition, success := DecomposeWinningHand(player, allTiles)
	if !success || decomposition == nil { return false, "", 0 }
	windPungKanCount := 0; windPairValue := 0; foundWindPungTypes := make(map[int]bool); pairFound := false
	for _, group := range decomposition {
		if len(group.Tiles) == 0 { continue }
		tile := group.Tiles[0]
		if group.Type == TypePair {
			pairFound = true
			if tile.Suit == "Wind" { windPairValue = tile.Value }
		} else if (group.Type == TypeTriplet || group.Type == TypeQuad) && tile.Suit == "Wind" {
			windPungKanCount++; foundWindPungTypes[tile.Value] = true
		}
	}
	if pairFound && windPungKanCount == 3 && windPairValue != 0 && !foundWindPungTypes[windPairValue] && len(foundWindPungTypes) == 3 {
		return true, "Shousuushii", 13
	}
	return false, "", 0
}

func checkDaisuushii(player *Player, allTiles []Tile) (bool, string, int) {
	decomposition, success := DecomposeWinningHand(player, allTiles)
	if !success || decomposition == nil { return false, "", 0 }
	windPungKanCount := 0; windsFound := make(map[int]bool); pairFound := false
	for _, group := range decomposition {
		if len(group.Tiles) == 0 { continue }
		tile := group.Tiles[0]
		if (group.Type == TypeTriplet || group.Type == TypeQuad) && tile.Suit == "Wind" {
			windPungKanCount++; windsFound[tile.Value] = true
		} else if group.Type == TypePair {
			pairFound = true
		}
	}
	if windPungKanCount == 4 && len(windsFound) == 4 && pairFound { return true, "Daisuushii", 26 }
	return false, "", 0
}

func checkTsuuiisou(player *Player, allTiles []Tile) (bool, string, int) {
	for _, tile := range allTiles { if !IsHonor(tile) { return false, "", 0 } }
	if IsChiitoitsu(allTiles) { return true, "Tsuuiisou", 13 }
	decomp, success := DecomposeWinningHand(player, allTiles)
	if success && decomp != nil { return true, "Tsuuiisou", 13 }
	return false, "", 0
}

func checkChinroutou(player *Player, allTiles []Tile) (bool, string, int) {
	for _, tile := range allTiles { if !IsTerminal(tile) { return false, "", 0 } }
	if IsChiitoitsu(allTiles) { return true, "Chinroutou", 13 }
	decomp, success := DecomposeWinningHand(player, allTiles)
	if success && decomp != nil { return true, "Chinroutou", 13 }
	return false, "", 0
}

func checkRyuuiisou(player *Player, allTiles []Tile) (bool, string, int) {
	greenTilesDef := map[string]map[int]bool{
		"Sou":    {2: true, 3: true, 4: true, 6: true, 8: true},
		"Dragon": {2: true},
	}
	for _, tile := range allTiles {
		suitMap, ok := greenTilesDef[tile.Suit]; if !ok || !suitMap[tile.Value] { return false, "", 0 }
	}
	if IsChiitoitsu(allTiles) { return true, "Ryuuiisou", 13 }
	decomp, success := DecomposeWinningHand(player, allTiles)
	if success && decomp != nil { return true, "Ryuuiisou", 13 }
	return false, "", 0
}

func checkChuurenPoutou(isMenzen bool, allTiles []Tile, agariHai Tile) (bool, string, int) {
	if !isMenzen || len(allTiles) != 14 { return false, "", 0 }
	handSuit := ""; if len(allTiles) > 0 { handSuit = allTiles[0].Suit } else { return false, "", 0 }
	if handSuit == "Wind" || handSuit == "Dragon" { return false, "", 0 }
	tileCounts := make(map[int]int); for _, t := range allTiles { if t.Suit != handSuit || IsHonor(t) { return false,"",0 }; tileCounts[t.Value]++ }
	basePtn := map[int]int{1:3,2:1,3:1,4:1,5:1,6:1,7:1,8:1,9:3}
	isChuuren, extraVal := false, -1
	for i:=1; i<=9; i++ {
		tempCounts := make(map[int]int); for k,v := range tileCounts { tempCounts[k]=v }
		if tempCounts[i]>0 { tempCounts[i]--; match := true
			for bpVal, bpCount := range basePtn { if tempCounts[bpVal] != bpCount { match=false; break } }
			if match { isChuuren=true; extraVal=i; break }
		}
	}
	if !isChuuren { return false, "", 0 }
	if agariHai.Suit == handSuit && agariHai.Value == extraVal { return true, "Junsei Chuuren Poutou", 26 }
	return true, "Chuuren Poutou", 13
}

func checkSuukantsu(player *Player) (bool, string, int) { // Yakuman
	if CheckSuukantsu(player) { return true, "Suukantsu", 13 } // CheckSuukantsu from checks.go
	return false, "", 0
}

// == 1 HAN YAKU ==
func checkDoubleRiichi(player *Player, gs *GameState) (bool, string, int) {
	if player.IsRiichi && player.DeclaredDoubleRiichi { return true, "Double Riichi Bonus", 1 }
	return false, "", 0
}

func checkMenzenTsumo(isTsumo bool, isMenzen bool) (bool, int) {
	if isTsumo && isMenzen { return true, 1 }
	return false, 0
}

func checkPinfu(player *Player, agariHai Tile, isMenzen bool, allTiles []Tile, gs *GameState) (bool, int) {
	if !isMenzen { return false, 0 }
	decomp, success := DecomposeWinningHand(player, allTiles)
	if !success || decomp == nil { return false, 0 }
	seqCount := 0; var pairGrp DecomposedGroup; pairFound := false
	for _, grp := range decomp {
		if grp.Type == TypeSequence { seqCount++ }
		else if grp.Type == TypePair { if pairFound {return false,0}; pairGrp = grp; pairFound=true }
		else { return false, 0 }
	}
	if seqCount != 4 || !pairFound || len(pairGrp.Tiles) == 0 { return false, 0 }
	if isYakuhai(pairGrp.Tiles[0], player, gs) { return false, 0 }
	ryanmenWait := false
	for _, grp := range decomp {
		if grp.Type == TypeSequence && groupContainsTileID(grp, agariHai.ID) {
			t1,t2,t3 := grp.Tiles[0],grp.Tiles[1],grp.Tiles[2]
			isAgariS1 := (agariHai.ID == t1.ID); isAgariS3 := (agariHai.ID == t3.ID)
			if (isAgariS1 && !(t1.Value==7 && t2.Value==8 && t3.Value==9)) ||
			   (isAgariS3 && !(t1.Value==1 && t2.Value==2 && t3.Value==3)) {
				ryanmenWait=true; break
			}
			if agariHai.ID == t2.ID { ryanmenWait=false; break } // Kanchan
		}
	}
	if !ryanmenWait { return false, 0 }
	return true, 1
}

func checkTanyao(allTiles []Tile) (bool, int) {
	for _, tile := range allTiles { if !IsSimple(tile) { return false, 0 } }
	return true, 1
}

func checkYakuhai(player *Player, gs *GameState, allTiles []Tile) ([]YakuResult, int) {
	results := []YakuResult{}; totalHan := 0
	decomp, success := DecomposeWinningHand(player, allTiles)
	if !success || decomp == nil { return nil, 0 }
	seatWindVal := WindValueFromName(player.SeatWind); prevWindVal := WindValueFromName(gs.PrevalentWind)
	for _, group := range decomp {
		if group.Type == TypeTriplet || group.Type == TypeQuad {
			if len(group.Tiles) == 0 { continue }
			tile := group.Tiles[0]
			if tile.Suit == "Dragon" {
				results = append(results, YakuResult{fmt.Sprintf("Yakuhai (%s)", tile.Name), 1}); totalHan++
			} else if tile.Suit == "Wind" {
				isSeat := (tile.Value == seatWindVal); isPrev := (tile.Value == prevWindVal)
				if isSeat && isPrev {
					results = append(results, YakuResult{fmt.Sprintf("Yakuhai (Seat & Prevalent %s)", tile.Name), 2}); totalHan += 2
				} else if isSeat {
					results = append(results, YakuResult{fmt.Sprintf("Yakuhai (Seat Wind %s)", tile.Name), 1}); totalHan++
				} else if isPrev {
					results = append(results, YakuResult{fmt.Sprintf("Yakuhai (Prevalent Wind %s)", tile.Name), 1}); totalHan++
				}
			}
		}
	}
	return results, totalHan
}

func checkHaiteiHoutei(gs *GameState, isTsumo bool) (bool, string, int) {
	if isTsumo && len(gs.Wall) == 0 { return true, "Haitei Raoyue", 1 }
	if !isTsumo && gs.IsHouteiDiscard { return true, "Houtei Raoyui", 1 }
	return false, "", 0
}

func checkRinshanKaihou(gs *GameState, isTsumo bool) (bool, int) {
	if isTsumo && gs.IsRinshanWin { return true, 1 }
	return false, 0
}

func checkChankan(gs *GameState) (bool, int) {
	if gs.IsChankanOpportunity { return true, 1 }
	return false, 0
}

// == 2 HAN YAKU ==
func checkChiitoitsu(player *Player, allTiles []Tile, isMenzen bool) (bool, int) {
	if !isMenzen { return false, 0 }
	if IsChiitoitsu(allTiles) { return true, 2 } // IsChiitoitsu from checks.go
	return false, 0
}

func checkToitoi(player *Player, allTiles []Tile) (bool, int) {
	decomp, success := DecomposeWinningHand(player, allTiles)
	if !success || decomp == nil { return false, 0 }
	pungKanCount, pairCount := 0,0
	for _, group := range decomp {
		if group.Type == TypeTriplet || group.Type == TypeQuad { pungKanCount++ }
		else if group.Type == TypePair { pairCount++ }
		else { return false, 0 }
	}
	if pungKanCount == 4 && pairCount == 1 { return true, 2 }
	return false, 0
}

func checkSanankou(player *Player, agariHai Tile, isTsumo bool, allTiles []Tile) (bool, int) {
	decomp, success := DecomposeWinningHand(player, allTiles)
	if !success || decomp == nil { return false, 0 }
	concealedPungCount := 0
	for _, group := range decomp {
		if group.Type == TypeTriplet && group.IsConcealed {
			if !isTsumo && groupContainsTileID(group, agariHai.ID) {} else { concealedPungCount++ }
		} else if group.Type == TypeQuad && group.IsConcealed { concealedPungCount++ }
	}
	if concealedPungCount == 3 { return true, 2 }
	return false, 0
}

func checkSanshokuDoukou(player *Player, allTiles []Tile) (bool, string, int) {
	decomp, success := DecomposeWinningHand(player, allTiles)
	if !success || decomp == nil { return false, "", 0 }
	pungsByValue := make(map[int]map[string]bool)
	for _, group := range decomp {
		if group.Type == TypeTriplet || group.Type == TypeQuad {
			if len(group.Tiles)==0 || IsHonor(group.Tiles[0]) {continue}
			val,suit := group.Tiles[0].Value, group.Tiles[0].Suit
			if pungsByValue[val] == nil { pungsByValue[val] = make(map[string]bool) }
			pungsByValue[val][suit] = true
		}
	}
	for _, suits := range pungsByValue {
		if len(suits) == 3 && suits["Man"] && suits["Pin"] && suits["Sou"] {
			return true, "Sanshoku Doukou", 2
		}
	}
	return false, "", 0
}

func checkSankantsu(player *Player) (bool, string, int) {
	kanCount := 0; for _, meld := range player.Melds { if strings.Contains(meld.Type, "Kan") { kanCount++ } }
	if kanCount == 3 { return true, "Sankantsu", 2 }
	return false, "", 0
}

func checkShousangen(player *Player, allTiles []Tile) (bool, int) {
	decomp, success := DecomposeWinningHand(player, allTiles)
	if !success || decomp == nil { return false, 0 }
	dragonPungKans := make(map[int]bool); dragonPairVal := 0; numDragonPungs, numDragonPairs := 0,0
	for _, group := range decomp {
		if len(group.Tiles) == 0 { continue }
		tile := group.Tiles[0]
		if tile.Suit == "Dragon" {
			if group.Type == TypeTriplet || group.Type == TypeQuad { dragonPungKans[tile.Value] = true; numDragonPungs++ }
			if group.Type == TypePair { dragonPairVal = tile.Value; numDragonPairs++ }
		}
	}
	if numDragonPungs == 2 && numDragonPairs == 1 && dragonPairVal != 0 && !dragonPungKans[dragonPairVal] && len(dragonPungKans) == 2 {
		return true, 2
	}
	return false, 0
}

func checkHonroutou(allTiles []Tile) (bool, int) {
	for _, tile := range allTiles { if IsSimple(tile) { return false, 0 } }
	return true, 2
}

// == 3+ HAN YAKU ==
func checkSanshokuDoujun(player *Player, isMenzen bool, allTiles []Tile) (bool, string, int) {
	decomp, success := DecomposeWinningHand(player, allTiles)
	if !success || decomp == nil { return false, "", 0 }
	sequences := []DecomposedGroup{}; for _, grp := range decomp { if grp.Type == TypeSequence { sequences = append(sequences, grp) } }
	if len(sequences) < 3 { return false, "", 0 }
	for i := 0; i < len(sequences); i++ {
		for j := i + 1; j < len(sequences); j++ {
			for k := j + 1; k < len(sequences); k++ {
				s1,s2,s3 := sequences[i],sequences[j],sequences[k]
				if s1.Tiles[0].Value == s2.Tiles[0].Value && s1.Tiles[0].Value == s3.Tiles[0].Value {
					suits := map[string]bool{s1.Tiles[0].Suit:true, s2.Tiles[0].Suit:true, s3.Tiles[0].Suit:true}
					if len(suits)==3 && suits["Man"] && suits["Pin"] && suits["Sou"] {
						han := 1; if isMenzen { han=2 }; return true, "Sanshoku Doujun", han
					}
				}
			}
		}
	}
	return false, "", 0
}

func checkIttsuu(player *Player, isMenzen bool, allTiles []Tile) (bool, string, int) {
	decomp, success := DecomposeWinningHand(player, allTiles)
	if !success || decomp == nil { return false, "", 0 }
	seqsBySuit := make(map[string]map[int]bool)
	for _, grp := range decomp {
		if grp.Type == TypeSequence {
			suit := grp.Tiles[0].Suit; if IsHonor(grp.Tiles[0]) { continue }
			if seqsBySuit[suit] == nil { seqsBySuit[suit] = make(map[int]bool) }
			seqsBySuit[suit][grp.Tiles[0].Value] = true
		}
	}
	for _, starts := range seqsBySuit {
		if starts[1] && starts[4] && starts[7] {
			han := 1; if isMenzen { han=2 }; return true, "Ittsuu", han
		}
	}
	return false, "", 0
}

func checkRyanpeikou(player *Player, isMenzen bool, allTiles []Tile) (bool, int) {
	if !isMenzen { return false, 0 }
	decomp, success := DecomposeWinningHand(player, allTiles)
	if !success || decomp == nil { return false, 0 }
	sequences := []DecomposedGroup{}; pairCount := 0
	for _, grp := range decomp {
		if grp.Type == TypeSequence { sequences = append(sequences, grp) }
		else if grp.Type == TypePair { pairCount++ }
		else { return false, 0 }
	}
	if len(sequences) != 4 || pairCount != 1 { return false, 0 }
	sort.Slice(sequences, func(i, j int) bool { /* complex sort needed or map approach */
		keyI := fmt.Sprintf("%s-%d%d%d", sequences[i].Tiles[0].Suit, sequences[i].Tiles[0].Value, sequences[i].Tiles[1].Value, sequences[i].Tiles[2].Value)
		keyJ := fmt.Sprintf("%s-%d%d%d", sequences[j].Tiles[0].Suit, sequences[j].Tiles[0].Value, sequences[j].Tiles[1].Value, sequences[j].Tiles[2].Value)
		return keyI < keyJ
	})
	if sequencesAreEqual(sequences[0].Tiles, sequences[1].Tiles) &&
	   sequencesAreEqual(sequences[2].Tiles, sequences[3].Tiles) &&
	   !sequencesAreEqual(sequences[0].Tiles, sequences[2].Tiles) {
		return true, 3
	}
	return false, 0
}

func checkIipeikou(player *Player, isMenzen bool, allTiles []Tile) (bool, int) {
	if !isMenzen { return false, 0 }
	decomp, success := DecomposeWinningHand(player, allTiles)
	if !success || decomp == nil { return false, 0 }
	sequences := []DecomposedGroup{}; for _, grp := range decomp { if grp.Type == TypeSequence { sequences = append(sequences, grp) } }
	if len(sequences) < 2 { return false, 0 }
	identicalPairCount := 0
	for i := 0; i < len(sequences); i++ {
		for j := i + 1; j < len(sequences); j++ {
			if sequencesAreEqual(sequences[i].Tiles, sequences[j].Tiles) { identicalPairCount++ }
		}
	}
	if identicalPairCount == 1 { return true, 1 }
	return false, 0
}

func checkJunchan(player *Player, isMenzen bool, allTiles []Tile) (bool, int) { // Junchan Taiyou
	decomp, success := DecomposeWinningHand(player, allTiles)
	if !success || decomp == nil { return false, 0 }
	for _, group := range decomp {
		hasTerminal := false
		for _, tile := range group.Tiles {
			if IsHonor(tile) { return false, 0 }
			if IsTerminal(tile) { hasTerminal = true }
		}
		if !hasTerminal { return false, 0 }
	}
	han := 2; if isMenzen { han = 3 }; return true, han
}

func checkHonitsu(allTiles []Tile, isMenzen bool) (bool, int) {
	targetSuit := ""; hasHonors := false; hasNumbers := false
	for _, tile := range allTiles {
		if IsHonor(tile) { hasHonors = true }
		else {
			hasNumbers = true
			if targetSuit == "" { targetSuit = tile.Suit }
			else if tile.Suit != targetSuit { return false, 0 }
		}
	}
	if targetSuit != "" && hasHonors && hasNumbers {
		han := 2; if isMenzen { han = 3 }; return true, han
	}
	return false, 0
}

func checkChinitsu(allTiles []Tile, isMenzen bool) (bool, int) {
	targetSuit := ""; if len(allTiles) > 0 { targetSuit = allTiles[0].Suit } else { return false, 0}
	if IsHonor(allTiles[0]) { return false, 0 }
	for _, tile := range allTiles { if IsHonor(tile) || tile.Suit != targetSuit { return false, 0 } }
	if targetSuit != "" {
		han := 5; if isMenzen { han = 6 }; return true, han
	}
	return false, 0
}

// checkNagashiMangan is defined in checks.go as it's an abortive draw / special condition,
// but if treated as a winning Yaku, its check would be:
/*
func checkNagashiMangan(player *Player, gs *GameState) (bool, string, int) {
	if player.HasHadDiscardCalledThisRound { return false, "", 0 }
	if len(player.Discards) == 0 { return false, "", 0 }
	for _, tile := range player.Discards {
		if !IsTerminalOrHonor(tile) { return false, "", 0 }
	}
	// This Yaku is typically only awarded at Ryuukyoku (exhaustive draw)
	if gs.RoundWinner == nil && len(gs.Wall) == 0 { // Check if it's an exhaustive draw
		return true, "Nagashi Mangan", 5 // Mangan-level Yaku
	}
	return false, "", 0
}
*/