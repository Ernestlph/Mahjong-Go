// yaku.go
package main

import (
	"fmt"
	"sort"
)

// YakuResult holds the name and Han value of an identified Yaku.
type YakuResult struct {
	Name string
	Han  int
}

// IdentifyYaku analyzes the winning hand and conditions to determine all applicable Yaku and total Han.
// This is the main entry point for Yaku calculation.
func IdentifyYaku(player *Player, agariHai Tile, isTsumo bool, gs *GameState) ([]YakuResult, int) {
	results := []YakuResult{}
	totalHan := 0

	// --- Pre-computation ---
	isMenzen := isMenzenchin(player, isTsumo, agariHai)
	allTiles := getAllTilesInHand(player, agariHai, isTsumo) // Get all 14 tiles

	// Group decomposition (Needed for many Yaku, complex to implement robustly)
	// TODO: Implement robust hand decomposition into 4 groups + 1 pair
	// For now, some checks might rely on simplified counting or checks.go structures.

	// --- 1. Yakuman Checks ---
	// If any Yakuman is present, usually other Yaku are ignored (except maybe Daisangen + Tsuuiisou etc., or multiple Yakuman).
	// Keeping it simple: First Yakuman found wins.
	yakumanFound := false

	if ok, name, han := checkKokushiMusou(allTiles, agariHai); ok {
		results = append(results, YakuResult{name, han})
		totalHan += han
		yakumanFound = true
	}
	// TODO: checkSuuankou (Four Concealed Pungs) - Needs hand decomposition. Check if Tsumo/Tanki wait matters.
	// TODO: checkDaisangen (Big Three Dragons) - Needs check for Pungs/Kans of all 3 dragons.
	// TODO: checkShousuushii (Little Four Winds) - Needs 3 Pungs/Kans of winds + pair of 4th wind.
	// TODO: checkDaisuushii (Big Four Winds) - Needs 4 Pungs/Kans of winds.
	// TODO: checkTsuuiisou (All Honors) - Check if all tiles are honors.
	// TODO: checkChinroutou (All Terminals) - Check if all tiles are 1s or 9s.
	// TODO: checkRyuuiisou (All Green) - Check for specific green tiles (Sou 2,3,4,6,8 + Green Dragon).
	// TODO: checkChuurenPoutou (Nine Gates) - Specific concealed pure suit pattern. Check for Junsei (9-wait) variant.
	// TODO: checkSuukantsu (Four Kans) - Check player.Melds for 4 Kans.

	if yakumanFound {
		fmt.Printf("Yakuman Found: %s (%d Han)\n", results[0].Name, results[0].Han)
		// Add Dora calculation even for Yakuman? Rules vary. Assume NO for now.
		// return results, totalHan // Usually return just the Yakuman
		// Let's allow Dora counting for simplicity now, but comment it
	}

	// --- 2. Regular Yaku Checks ---
	// Check common Yaku if no Yakuman was found (or if rules allow stacking, which we aren't fully handling yet).

	// --- 1 Han Yaku ---
	if ok, han := checkRiichi(player, gs); ok {
		results = append(results, YakuResult{"Riichi", han}) // Base 1 Han
		totalHan += han
		if okI, hanI := checkIppatsu(player, gs); okI {
			results = append(results, YakuResult{"Ippatsu", hanI})
			totalHan += hanI
		}
		// TODO: checkDoubleRiichi (Needs check on turn 1)
	}

	if ok, han := checkMenzenTsumo(isTsumo, isMenzen); ok {
		// Pinfu Tsumo gives 20fu base, Pinfu Ron gives 30fu base.
		// Standard Menzen Tsumo adds 1 Han, but doesn't get the Pinfu yaku itself unless conditions met.
		// If Pinfu is *also* present, it's counted separately.
		results = append(results, YakuResult{"Menzen Tsumo", han})
		totalHan += han
	}

	// Pinfu - Requires Menzen, 4 sequences, non-yakuhai pair, ryanmen wait.
	// Call to the already defined checkPinfu.
	if isMenzen { 
		if ok, han := checkPinfu(player, agariHai, isMenzen, allTiles, gs); ok {
			results = append(results, YakuResult{"Pinfu", han})
			totalHan += han
		}
	}

	// Tanyao (All Simples) - Check if Kuitan (open Tanyao) is allowed. Assuming YES for now.
	if ok, han := checkTanyao(allTiles); ok {
		results = append(results, YakuResult{"Tanyao", han})
		totalHan += han
	}

	// Yakuhai (Seat/Prevalent Wind, Dragons)
	if yakuYakuhai, hanYakuhai := checkYakuhai(player, gs, allTiles); len(yakuYakuhai) > 0 {
		results = append(results, yakuYakuhai...)
		totalHan += hanYakuhai
	}

	// Haitei/Houtei (Last Draw/Discard)
	if ok, name, han := checkHaiteiHoutei(gs, isTsumo); ok {
		results = append(results, YakuResult{name, han})
		totalHan += han
	}

	// Rinshan Kaihou (After Kan Draw)
	if ok, han := checkRinshanKaihou(gs, isTsumo); ok {
		results = append(results, YakuResult{"Rinshan Kaihou", han})
		totalHan += han
	}

	// Chankan (Robbing a Kan)
	if !isTsumo { // Chankan is always a Ron
		if ok, han := checkChankan(gs); ok {
			results = append(results, YakuResult{"Chankan", han})
			totalHan += han
		}
	}

	// --- 2 Han Yaku ---
	chiitoitsuFound := false
	if ok, han := checkChiitoitsu(player, allTiles, isMenzen); ok {
		results = append(results, YakuResult{"Chiitoitsu", han})
		totalHan += han
		chiitoitsuFound = true // Chiitoitsu doesn't combine well with sequence Yaku.
	}

	// Toitoi (All Pungs) - Cannot be Chiitoitsu
	if !chiitoitsuFound {
		// Call to the already defined checkToitoi.
		if ok, han := checkToitoi(player, allTiles); ok { 
			results = append(results, YakuResult{"Toitoi", han})
			totalHan += han
		}
	}

	// Sanankou (Three Concealed Pungs) - Cannot be Chiitoitsu
	if !chiitoitsuFound {
		// Call to the already defined checkSanankou. Signature: checkSanankou(player *Player, agariHai Tile, isTsumo bool, allTiles []Tile)
		if ok, han := checkSanankou(player, agariHai, isTsumo, allTiles); ok { 
			results = append(results, YakuResult{"Sanankou", han}) 
			totalHan += han
		}
	}

	// Sankantsu (Three Kans)
	// TODO: Check player.Melds for 3 Kans.
	// if ok, han := checkSankantsu(player); ok {
	//     results = append(results, YakuResult{"Sankantsu", 2})
	//     totalHan += han
	// }

	// Shousangen (Little Three Dragons) - Cannot be Chiitoitsu
	if !chiitoitsuFound {
		if ok, han := checkShousangen(player, allTiles); ok {
			results = append(results, YakuResult{"Shousangen", 2})
			totalHan += han
		}
	}

	// Honroutou (All Terminals and Honors) - Cannot be Chiitoitsu
	// Often combined with Toitoi or Chiitoitsu. Check logic carefully.
	// If Chiitoitsu is found AND it's Honroutou, value is just 2 Han? Rules can vary. Assume separate check for now.
	if !chiitoitsuFound { // Check if not Chiitoitsu? Or check even if? Let's check separately.
		if ok, han := checkHonroutou(allTiles); ok {
			results = append(results, YakuResult{"Honroutou", 2})
			totalHan += han
		}
	} else {
		// Check if Chiitoitsu is ALSO Honroutou (terminals/honors only)
		if ok, _ := checkHonroutou(allTiles); ok {
			// Chiitoitsu + Honroutou is still just 2 han for Chiitoitsu usually.
			// We already added Chiitoitsu han. Add Honroutou to name list?
			// Maybe add a combined name? For now, do nothing extra here.
		}
	}

	// --- 3+ Han Yaku ---

	// Iipeikou (One Pure Double Sequence) - 1 Han. Cannot be Chiitoitsu. Requires Menzen.
	if !chiitoitsuFound && isMenzen {
		// Call to the already defined checkIipeikou.
		if ok, han := checkIipeikou(player, isMenzen, allTiles); ok { 
			results = append(results, YakuResult{"Iipeikou", han})
			totalHan += han
		}
	}

	// Ryanpeikou (Two Iipeikou) - 3 Han. Overrides Iipeikou. Requires Menzen. Cannot be Chiitoitsu.
	if !chiitoitsuFound && isMenzen {
		if okRyan, hanRyan := checkRyanpeikou(player, isMenzen, allTiles); okRyan {
			// Remove Iipeikou if it was previously added
			newResults := []YakuResult{}
			foundIipeikou := false
			for _, r := range results {
				if r.Name == "Iipeikou" {
					foundIipeikou = true
					// Do not add Iipeikou to newResults
				} else {
					newResults = append(newResults, r)
				}
			}
			if foundIipeikou {
				// Adjust totalHan if it was being summed incrementally.
				// For simplicity, finalHan is recalculated at the end, so direct subtraction here isn't critical
				// but good practice if other logic depended on intermediate totalHan.
				// totalHan -= 1 // Assuming Iipeikou is 1 Han
			}
			results = newResults
			results = append(results, YakuResult{"Ryanpeikou", hanRyan}) // hanRyan is 3
			// totalHan will be recalculated at the end.
		}
	}

	honitsuFound := false
	if ok, han := checkHonitsu(allTiles, isMenzen); ok {
		results = append(results, YakuResult{"Honitsu", han})
		totalHan += han
		honitsuFound = true
	}

	// Junchan (Outside Hand with Terminals) - Cannot be Chiitoitsu. 3 Han Menzen, 2 Han Open.
	if !chiitoitsuFound {
		// Signature: checkJunchan(player *Player, isMenzen bool, allTiles []Tile)
		if ok, han := checkJunchan(player, isMenzen, allTiles); ok {
			results = append(results, YakuResult{"Junchan", han})
			// totalHan will be updated by final loop.
		}
	}

	// --- 6+ Han Yaku ---
	// Chinitsu (Pure Hand) - Overrides Honitsu
	if ok, han := checkChinitsu(allTiles, isMenzen); ok {
		// If Chinitsu found, remove Honitsu if it was added
		if honitsuFound {
			results = removeYakuByName(results, "Honitsu")
			totalHan -= IfElseInt(isMenzen, 3, 2) // Remove Honitsu Han
		}
		results = append(results, YakuResult{"Chinitsu", han}) // 6 concealed, 5 open
		totalHan += han
	}

	// --- 3. Dora Calculation ---
	// Dora are added *after* all other Yaku are calculated. They don't enable a win on their own.
	doraCount := 0
	// a) Regular Dora
	doraCount += countDora(allTiles, gs.DoraIndicators)
	// b) Red Dora (Aka Dora)
	doraCount += countRedDora(allTiles)
	// c) Ura Dora (Only if Riichi)
	if player.IsRiichi && len(gs.UraDoraIndicators) > 0 {
		doraCount += countDora(allTiles, gs.UraDoraIndicators)
	}

	if doraCount > 0 {
		results = append(results, YakuResult{fmt.Sprintf("Dora %d", doraCount), doraCount})
		totalHan += doraCount
	}

	// --- Final Check ---
	if totalHan == 0 && !yakumanFound {
		// Should not happen if CanDeclareWin checked Yaku, but as a fallback
		fmt.Println("Warning: IdentifyYaku resulted in 0 Han and no Yakuman.")
		return []YakuResult{}, 0
	}

	// Recalculate totalHan just to be safe after potential removals (like for Ryanpeikou)
	finalHan := 0
	for _, r := range results {
		finalHan += r.Han
	}

	return results, finalHan
}

// --- Helper Functions ---

// checkDaisangen (Big Three Dragons) - Yakuman (13 Han)
func checkDaisangen(player *Player, allTiles []Tile) (bool, string, int) {
	decomposition, success := DecomposeWinningHand(player, allTiles)
	if !success || decomposition == nil || len(decomposition) != 5 {
		return false, "", 0
	}

	dragonPungKan := map[int]bool{1: false, 2: false, 3: false} // White, Green, Red

	for _, group := range decomposition {
		if group.Type == TypeTriplet || group.Type == TypeQuad {
			if len(group.Tiles) > 0 && group.Tiles[0].Suit == "Dragon" {
				dragonPungKan[group.Tiles[0].Value] = true
			}
		}
	}

	if dragonPungKan[1] && dragonPungKan[2] && dragonPungKan[3] {
		return true, "Daisangen", 13
	}
	return false, "", 0
}

// checkSuuankou (Four Concealed Pungs/Kans) - Yakuman (13 Han)
// Note: Suuankou Tanki (double Yakuman on pair wait) is not differentiated here.
func checkSuuankou(player *Player, agariHai Tile, isTsumo bool, isMenzen bool, allTiles []Tile) (bool, string, int) {
	if !isMenzen {
		return false, "", 0
	}

	decomposition, success := DecomposeWinningHand(player, allTiles)
	if !success || decomposition == nil || len(decomposition) != 5 {
		return false, "", 0
	}

	concealedPungKanCount := 0
	for _, group := range decomposition {
		isEffectiveConcealedMeld := false
		if group.Type == TypeQuad && group.IsConcealed { // Ankan
			isEffectiveConcealedMeld = true
		} else if group.Type == TypeTriplet && group.IsConcealed {
			// If Ron, and this group was completed by agariHai, it does NOT count as concealed for Suuankou.
			if !isTsumo && groupContainsTileID(group, agariHai.ID) {
				// This Pung was completed by Ron, so it's not "concealed" in the Suuankou sense.
			} else {
				isEffectiveConcealedMeld = true
			}
		}
		if isEffectiveConcealedMeld {
			concealedPungKanCount++
		}
	}

	if concealedPungKanCount == 4 {
		// Additional check for Suuankou Tanki (pair wait) - if agariHai completes the pair.
		// For single Yakuman, this distinction isn't needed, but for double, it is.
		// For now, just 13 Han.
		// if isTsumo { /* always valid */ } else { /* if Ron, agariHai must complete the pair */ }
		// The logic above for Ron already handles the pung completion. If it's a Tanki wait,
		// agariHai would be part of the pair, and all 4 pungs would be fully concealed before the win.
		return true, "Suuankou", 13
	}
	return false, "", 0
}

// checkTsuuiisou (All Honors) - Yakuman (13 Han)
func checkTsuuiisou(player *Player, allTiles []Tile) (bool, string, int) {
	if len(allTiles) != 14 { return false, "", 0} // Ensure correct number of tiles for checks
	for _, tile := range allTiles {
		if !isHonor(tile) {
			return false, "", 0 // Found a non-honor tile
		}
	}

	// All tiles are honors. Now check for valid hand structure.
	// Try standard decomposition first.
	decomposition, success := DecomposeWinningHand(player, allTiles)
	if success && decomposition != nil {
		// DecomposeWinningHand validates 4 groups + 1 pair.
		return true, "Tsuuiisou", 13
	}
	
	// If standard decomposition fails, check for Chiitoitsu structure.
	if IsChiitoitsu(allTiles) { // Assumes IsChiitoitsu checks for 7 distinct pairs
		return true, "Tsuuiisou", 13
	}

	return false, "", 0 // All honors but not a valid hand structure
}

// checkChinroutou (All Terminals) - Yakuman (13 Han)
func checkChinroutou(player *Player, allTiles []Tile) (bool, string, int) {
	if len(allTiles) != 14 { return false, "", 0} // Ensure correct number of tiles
	for _, tile := range allTiles {
		if !isTerminal(tile) { // Must be only terminals
			return false, "", 0
		}
		// isTerminal implies not an honor, so no separate honor check needed if isTerminal is strict.
	}

	// All tiles are terminals. Check for valid hand structure.
	decomposition, success := DecomposeWinningHand(player, allTiles)
	if success && decomposition != nil {
		return true, "Chinroutou", 13
	}
	
	if IsChiitoitsu(allTiles) { // 7 pairs of terminals
		return true, "Chinroutou", 13
	}
	
	return false, "", 0 // All terminals but not a valid hand structure
}

// groupContainsTileID checks if a tile ID exists in a list of DecomposedGroup tiles.
// Local helper for Yaku checks, especially Sanankou and Pinfu.
func groupContainsTileID(group DecomposedGroup, tileID int) bool {
	for _, t := range group.Tiles {
		if t.ID == tileID {
			return true
		}
	}
	return false
}

// sequencesAreEqual checks if two sequences (represented by their tiles) are identical.
// Assumes tiles within each sequence are sorted.
func sequencesAreEqual(seq1Tiles []Tile, seq2Tiles []Tile) bool {
	if len(seq1Tiles) != 3 || len(seq2Tiles) != 3 {
		return false // Sequences must have 3 tiles
	}
	for i := 0; i < 3; i++ {
		// Compare by Suit and Value, ignoring ID and IsRed for structural equality
		if seq1Tiles[i].Suit != seq2Tiles[i].Suit || seq1Tiles[i].Value != seq2Tiles[i].Value {
			return false
		}
	}
	return true
}

// isMenzenchin checks if the hand is considered concealed.
// Ankan does NOT break concealment. Daiminkan/Shouminkan/Pon/Chi do.
// Ron on the completing tile maintains concealment if no prior open melds.
func isMenzenchin(player *Player, isTsumo bool, agariHai Tile) bool {
	for _, meld := range player.Melds {
		// Ankan is concealed. Daiminkan/Shouminkan are open. Pon/Chi are open.
		if !meld.IsConcealed {
			// Check if the open meld is a Shouminkan being robbed (Chankan)
			// If the win is Chankan on this meld, the hand *was* menzen before the Kan attempt.
			// This specific case needs careful handling based on Chankan detection logic.
			// For simplicity now, any open meld means not Menzen, unless it's the Ron tile itself completing the hand.

			// Generally, if any non-Ankan meld exists, it's not Menzenchin.
			return false
		}
	}
	// If no open melds, the hand is Menzenchin, even if won by Ron.
	return true
}

// getAllTilesInHand collects all 14 tiles involved in the win.
// For Ron, it adds the agariHai to the player's 13 tiles.
// For Tsumo, it uses the player's 14 tiles directly.
func getAllTilesInHand(player *Player, agariHai Tile, isTsumo bool) []Tile {
	allTiles := []Tile{}

	// Add tiles from melds
	for _, meld := range player.Melds {
		allTiles = append(allTiles, meld.Tiles...)
	}

	// Add tiles from concealed hand part
	allTiles = append(allTiles, player.Hand...)

	// Ensure the winning tile is included if not already there (e.g., Ron)
	if !isTsumo {
		found := false
		for _, t := range allTiles {
			if t.ID == agariHai.ID {
				found = true
				break
			}
		}
		if !found {
			// This happens if player.Hand didn't include agariHai yet. Add it.
			// This assumes player.Hand has 13 tiles + melds before Ron.
			allTiles = append(allTiles, agariHai)
		}
	}

	// Sort for consistency
	sort.Sort(BySuitValue(allTiles))

	if len(allTiles) != 14 {
		fmt.Printf("Warning: getAllTilesInHand resulted in %d tiles. Hand: %v, Melds: %v, Agari: %s, Tsumo: %v\n",
			len(allTiles), TilesToNames(player.Hand), FormatMeldsForDisplay(player.Melds), agariHai.Name, isTsumo)
		// Pad or truncate? For now, return what we have, but it indicates an issue.
	}
	// Ensure exactly 14 tiles if possible, crucial for checks like Chiitoitsu/Kokushi
	if len(allTiles) > 14 {
		return allTiles[:14] // Should not happen
	}
	// Padding with empty tiles might be worse. Best to fix the source.

	return allTiles
}

// isTerminal checks if a tile is a 1 or 9.
func isTerminal(tile Tile) bool {
	return (tile.Suit == "Man" || tile.Suit == "Pin" || tile.Suit == "Sou") && (tile.Value == 1 || tile.Value == 9)
}

// isHonor checks if a tile is a Wind or Dragon.
func isHonor(tile Tile) bool {
	return tile.Suit == "Wind" || tile.Suit == "Dragon"
}

// isSimple checks if a tile is a number tile from 2 to 8.
func isSimple(tile Tile) bool {
	return (tile.Suit == "Man" || tile.Suit == "Pin" || tile.Suit == "Sou") && (tile.Value >= 2 && tile.Value <= 8)
}

// isYakuhai checks if a tile is a relevant Honor tile for the player.
func isYakuhai(tile Tile, player *Player, gs *GameState) bool {
	if tile.Suit == "Dragon" {
		return true // White, Green, Red dragons always Yakuhai
	}
	if tile.Suit == "Wind" {
		// Check prevalent wind
		if (gs.PrevalentWind == "East" && tile.Value == 1) ||
			(gs.PrevalentWind == "South" && tile.Value == 2) ||
			(gs.PrevalentWind == "West" && tile.Value == 3) || // For West/North rounds if implemented
			(gs.PrevalentWind == "North" && tile.Value == 4) { // For West/North rounds if implemented
			return true
		}
		// Check player's seat wind
		if (player.SeatWind == "East" && tile.Value == 1) ||
			(player.SeatWind == "South" && tile.Value == 2) ||
			(player.SeatWind == "West" && tile.Value == 3) ||
			(player.SeatWind == "North" && tile.Value == 4) {
			return true
		}
	}
	return false
}

// getDoraTile returns the tile indicated by a Dora indicator.
func getDoraTile(indicator Tile) Tile {
	dora := indicator  // Copy indicator
	dora.IsRed = false // Dora value is never red itself
	dora.ID = -1       // Indicate it's a derived tile type

	suit := indicator.Suit
	value := indicator.Value

	switch suit {
	case "Man", "Pin", "Sou":
		if value == 9 {
			dora.Value = 1
		} else {
			dora.Value = value + 1
		}
		dora.Name = fmt.Sprintf("%s %d", suit, dora.Value) // Update name
	case "Wind": // E->S->W->N->E
		if value == 4 { // North
			dora.Value = 1 // Becomes East
			dora.Name = "East"
		} else {
			dora.Value = value + 1
			winds := []string{"", "East", "South", "West", "North"}
			dora.Name = winds[dora.Value] // Update name
		}
	case "Dragon": // W->G->R->W
		if value == 3 { // Red Dragon
			dora.Value = 1 // Becomes White Dragon
			dora.Name = "White"
		} else {
			dora.Value = value + 1
			dragons := []string{"", "White", "Green", "Red"}
			dora.Name = dragons[dora.Value] // Update name
		}
	}
	return dora
}

// countDora counts how many Dora (regular or Ura) tiles are in the hand.
func countDora(handTiles []Tile, indicators []Tile) int {
	count := 0
	for _, indicator := range indicators {
		doraValueTile := getDoraTile(indicator)
		for _, handTile := range handTiles {
			// Compare Suit and Value only for Dora matching
			if handTile.Suit == doraValueTile.Suit && handTile.Value == doraValueTile.Value {
				count++
			}
		}
	}
	return count
}

// countRedDora counts how many Red Fives are in the hand.
func countRedDora(handTiles []Tile) int {
	count := 0
	for _, tile := range handTiles {
		if tile.IsRed {
			count++
		}
	}
	return count
}

// Helper to remove a Yaku by name from the results slice
func removeYakuByName(results []YakuResult, nameToRemove string) []YakuResult {
	newResults := []YakuResult{}
	for _, r := range results {
		if r.Name != nameToRemove {
			newResults = append(newResults, r)
		}
	}
	return newResults
}

// Helper to recalculate total Han from a results slice
func calculateTotalHan(results []YakuResult) int {
	han := 0
	for _, r := range results {
		han += r.Han
	}
	return han
}

// --- Individual Yaku Check Functions ---

// checkTenhou (Blessing of Heaven) - Yakuman (13 Han)
// Player must be dealer, win by Tsumo on their very first draw, with no intervening calls.
func checkTenhou(player *Player, gs *GameState, isTsumo bool) (bool, string, int) {
	// Conceptual checks (actual flag implementation is separate):
	// 1. Player is dealer (East seat in the first round of the game is a common way to define initial dealer)
	//    A more robust check would be `gs.Players[gs.DealerPlayerIndex] == player`.
	//    For now, using SeatWind == "East" AND it's very early in the game.
	// 2. Win by Tsumo.
	// 3. Player's first draw of the game (gs.TurnNumber roughly indicates this, or a specific !player.HasDrawnThisGameYet flag).
	// 4. No calls (Pon, Chi, open Kan) have occurred before this Tsumo.
	
	// Simplified conditions based on prompt:
	// player.SeatWind == "East" (initial dealer)
	// isTsumo
	// gs.TurnNumber <= 1 (very early in the game, implies first draw for East)
	// !gs.AnyCallMadeThisRound (conceptual flag for no interruptions)

	// A more precise check for Tenhou would be:
	// Player is dealer (e.g. gs.Players[gs.DealerIndex] == player)
	// isTsumo
	// It is the dealer's very first draw and discard cycle (e.g. gs.TurnNumber == 0 for dealer's draw phase)
	// No melds (Ankan by dealer before their first discard might be an exception in some rules, but generally not for Tenhou)
	
	// Using the conceptual flags from the prompt:
	// Let's assume a more direct way to check if it's the dealer's first action.
	// For Tenhou, gs.TurnNumber is typically 0 when the dealer draws their first tile (14th tile).
	// The win happens *on* this draw.
	if player.SeatWind == "East" && isTsumo && gs.TurnNumber == 0 && !gs.AnyCallMadeThisRound {
		// Further refinement could be ensuring player.Discards is empty, player.Melds is empty (or only Ankan if allowed by specific ruleset).
		// For simplicity, the above conditions are a strong proxy.
		return true, "Tenhou", 13
	}
	return false, "", 0
}

// checkChihou (Blessing of Earth) - Yakuman (13 Han)
// Non-dealer wins by Tsumo on their very first draw, before their first discard, with no intervening calls.
func checkChihou(player *Player, gs *GameState, isTsumo bool) (bool, string, int) {
	// Conceptual checks:
	// 1. Player is NOT dealer.
	// 2. Win by Tsumo.
	// 3. Player's first draw of the round, before their first discard (e.g., !player.HasMadeFirstDiscardThisRound).
	// 4. No calls by anyone that interrupted the "first go-around" before this player's turn.
	
	// Using the conceptual flags from the prompt:
	// player.SeatWind != "East" (not initial dealer - this is a simplification)
	// isTsumo
	// gs.IsFirstGoAround (conceptual: no player has completed their first turn, or no calls made)
	// !player.HasMadeFirstDiscard (conceptual)
	// !gs.AnyCallMadeThisRound (conceptual)

	// A more precise check for Chihou:
	// Player is NOT the dealer.
	// isTsumo
	// It is the player's very first draw of the round.
	// No calls (Pon, Chi, open Kan) have occurred by anyone in the round prior to this Tsumo.
	// (Ankan by another player before this player's turn might be allowed in some rules).
	if player.SeatWind != "East" && isTsumo && gs.IsFirstGoAround && !player.HasMadeFirstDiscard && !gs.AnyCallMadeThisRound {
		// Similar to Tenhou, ensure player.Discards is empty for this round, player.Melds is empty (or only Ankan if allowed).
		return true, "Chihou", 13
	}
	return false, "", 0
}

// checkRenhou (Man by Human) - Yakuman (13 Han) or Mangan (depends on ruleset)
// Player wins by Ron on a discard made during the first un-interrupted go-around of turns,
// before their own first discard.
func checkRenhou(player *Player, gs *GameState, isTsumo bool) (bool, string, int) {
	// Conceptual checks:
	// 1. Win by Ron (!isTsumo).
	// 2. Player has not yet made their first discard in this round (!player.HasMadeFirstDiscardThisRound).
	// 3. The discarder made their discard during the "first go-around" with no prior interrupting calls.
	
	// Using conceptual flags from the prompt:
	// !isTsumo
	// gs.IsFirstGoAround (conceptual)
	// !player.HasMadeFirstDiscard (conceptual, for the winning player)
	// !gs.AnyCallMadeThisRound (conceptual, for no interruptions before the Ron)
	if !isTsumo && gs.IsFirstGoAround && !player.HasMadeFirstDiscard && !gs.AnyCallMadeThisRound {
		// Renhou's value can vary (Mangan to Yakuman). We'll use 13 for consistency with prompt.
		return true, "Renhou", 13 
	}
	return false, "", 0
}


// checkShousuushii (Little Four Winds) - Yakuman (13 Han)
func checkShousuushii(player *Player, allTiles []Tile) (bool, string, int) {
	decomposition, success := DecomposeWinningHand(player, allTiles)
	if !success || decomposition == nil || len(decomposition) != 5 {
		return false, "", 0
	}

	windPungKanCount := 0
	windPairFound := false
	foundWindValuesForPungKan := make(map[int]bool)
	var foundWindValueForPair int = 0

	for _, group := range decomposition {
		if group.Type == TypeTriplet || group.Type == TypeQuad {
			if len(group.Tiles) > 0 && group.Tiles[0].Suit == "Wind" {
				windValue := group.Tiles[0].Value
				if !foundWindValuesForPungKan[windValue] { // Count distinct wind pungs/kans
					foundWindValuesForPungKan[windValue] = true
					windPungKanCount++
				} else { // Same wind type pung/kan twice - not Shousuushii
					// This check might be too strict if player has e.g. two pungs of East wind.
					// The core is 3 types of wind pungs and 1 type of wind pair.
					// Let's refine: just count distinct pungs.
				}
			}
		} else if group.Type == TypePair {
			if len(group.Tiles) > 0 && group.Tiles[0].Suit == "Wind" {
				if windPairFound { return false, "", 0 } // Multiple wind pairs or multiple pairs in general
				foundWindValueForPair = group.Tiles[0].Value
				windPairFound = true
			}
		}
	}
    // Recount distinct wind pungs/kans
    distinctWindPungKanCount := len(foundWindValuesForPungKan)

	if distinctWindPungKanCount == 3 && windPairFound {
		// Ensure the pair's wind is the 4th distinct wind type
		if _, isPairWindAlsoPung := foundWindValuesForPungKan[foundWindValueForPair]; !isPairWindAlsoPung {
			return true, "Shousuushii", 13
		}
	}
	return false, "", 0
}

// checkDaisuushii (Big Four Winds) - Yakuman (13 Han)
func checkDaisuushii(player *Player, allTiles []Tile) (bool, string, int) {
	decomposition, success := DecomposeWinningHand(player, allTiles)
	if !success || decomposition == nil || len(decomposition) != 5 {
		return false, "", 0
	}

	windPungKanCount := 0
	foundWindValues := make(map[int]bool)

	for _, group := range decomposition {
		if group.Type == TypeTriplet || group.Type == TypeQuad {
			if len(group.Tiles) > 0 && group.Tiles[0].Suit == "Wind" {
				// No need to check for duplicates here, DecomposeWinningHand won't give two pungs of same tile.
				// We just need to ensure all four distinct wind values are present as pungs/kans.
				foundWindValues[group.Tiles[0].Value] = true
				windPungKanCount++ // Counts total wind pungs/kans
			}
		}
	}

	if windPungKanCount == 4 && len(foundWindValues) == 4 {
		return true, "Daisuushii", 13 // Some rules give Double Yakuman
	}
	return false, "", 0
}

// checkRyuuiisou (All Green) - Yakuman (13 Han)
func checkRyuuiisou(player *Player, allTiles []Tile) (bool, string, int) {
	if len(allTiles) != 14 { return false, "", 0}
	
	greenTiles := map[string]map[int]bool{
		"Sou":    {2: true, 3: true, 4: true, 6: true, 8: true},
		"Dragon": {2: true}, // Green Dragon (Value 2)
	}

	for _, tile := range allTiles {
		suitMap, ok := greenTiles[tile.Suit]
		if !ok { return false, "", 0 } // Not Sou or Dragon
		if !suitMap[tile.Value] { return false, "", 0 } // Not a green number/dragon
	}

	// All tiles are green. Now check for valid hand structure.
	decomposition, success := DecomposeWinningHand(player, allTiles)
	if success && decomposition != nil {
		return true, "Ryuuiisou", 13
	}
	if IsChiitoitsu(allTiles) {
		return true, "Ryuuiisou", 13
	}
	return false, "", 0
}

// checkChuurenPoutou (Nine Gates) - Yakuman (13 Han)
func checkChuurenPoutou(isMenzen bool, allTiles []Tile) (bool, string, int) {
	if !isMenzen {
		return false, "", 0
	}
	if len(allTiles) != 14 { return false, "", 0 }

	firstTileSuit := allTiles[0].Suit
	if firstTileSuit == "Wind" || firstTileSuit == "Dragon" {
		return false, "", 0 // Must be Man, Pin, or Sou
	}

	counts := make(map[int]int)
	for _, tile := range allTiles {
		if tile.Suit != firstTileSuit || isHonor(tile) { // Must all be same suit, no honors
			return false, "", 0
		}
		counts[tile.Value]++
	}

	// Standard Chuuren: 1,1,1, 2,3,4,5,6,7,8, 9,9,9 (13 tiles) + 1 extra tile of the same suit
	requiredCounts := map[int]int{1:3, 2:1, 3:1, 4:1, 5:1, 6:1, 7:1, 8:1, 9:3}
	extraTileFound := false
	
	for val := 1; val <= 9; val++ {
		if counts[val] < requiredCounts[val] {
			return false, "", 0 // Missing a required tile
		}
		if counts[val] > requiredCounts[val] {
			if counts[val] == requiredCounts[val]+1 && !extraTileFound {
				extraTileFound = true // This is the 14th tile
			} else {
				return false, "", 0 // Too many of one tile, or multiple extra tiles
			}
		}
	}
	
	if extraTileFound || (len(allTiles) == 13 && verifyChuurenBase(counts)) { 
		// The verifyChuurenBase is more for a 13-tile check, here allTiles is 14.
		// The loop already checks the 14-tile structure.
		// The condition `extraTileFound` is sufficient if all `requiredCounts` are met.
		
		// Final check: sum of counts must be 14
		sumCounts := 0
		for _, c := range counts { sumCounts += c }
		if sumCounts != 14 { return false, "", 0}

		return true, "Chuuren Poutou", 13
		// Junsei Chuuren (9-sided wait) is a double yakuman, not differentiated here.
	}

	return false, "", 0
}

// verifyChuurenBase (helper for a 13-tile "pure" nine gates pattern before the 14th tile)
// Not strictly needed if checkChuurenPoutou always receives 14 tiles.
func verifyChuurenBase(counts map[int]int) bool {
    required := map[int]int{1:3, 2:1, 3:1, 4:1, 5:1, 6:1, 7:1, 8:1, 9:3}
    for val, reqCount := range required {
        if counts[val] != reqCount {
            return false
        }
    }
    return true
}


// checkSuukantsu (Four Kans) - Yakuman (13 Han)
func checkSuukantsu(player *Player) (bool, string, int) {
	kanCount := 0
	for _, meld := range player.Melds {
		if meld.Type == "Ankan" || meld.Type == "Daiminkan" || meld.Type == "Shouminkan" {
			kanCount++
		}
	}
	if kanCount == 4 {
		// Hand must also have a pair to be complete.
		// DecomposeWinningHand would be needed to confirm the pair if we passed allTiles.
		// However, Suukantsu is typically identified by the 4 Kans alone, assuming the pair exists.
		// The game rules often make 4 kans by one player a special case (e.g. abortive draw if not won on 4th kan's rinshan).
		// For Yaku check, 4 Kans is sufficient.
		return true, "Suukantsu", 13
	}
	return false, "", 0
}


// checkDaisangen (Big Three Dragons) - Yakuman (13 Han)
func checkDaisangen(player *Player, allTiles []Tile) (bool, string, int) {
	decomposition, success := DecomposeWinningHand(player, allTiles)
	if !success || decomposition == nil {
		return false, "", 0
	}

	dragonPungKan := map[int]bool{1: false, 2: false, 3: false} // White, Green, Red

	for _, group := range decomposition {
		if group.Type == TypeTriplet || group.Type == TypeQuad {
			if len(group.Tiles) > 0 && group.Tiles[0].Suit == "Dragon" {
				dragonPungKan[group.Tiles[0].Value] = true
			}
		}
	}

	if dragonPungKan[1] && dragonPungKan[2] && dragonPungKan[3] {
		return true, "Daisangen", 13
	}
	return false, "", 0
}

// checkSuuankou (Four Concealed Pungs/Kans) - Yakuman (13 Han)
// Note: Suuankou Tanki (double Yakuman on pair wait) is not differentiated here.
func checkSuuankou(player *Player, agariHai Tile, isTsumo bool, isMenzen bool, allTiles []Tile) (bool, string, int) {
	if !isMenzen {
		return false, "", 0
	}

	decomposition, success := DecomposeWinningHand(player, allTiles)
	if !success || decomposition == nil {
		// Suuankou might not always decompose cleanly if it's a Chiitoitsu-like wait for the last pung.
		// However, standard Suuankou is 4 pungs + 1 pair.
		// If DecomposeWinningHand fails, it's unlikely to be standard Suuankou.
		return false, "", 0
	}

	concealedPungKanCount := 0
	for _, group := range decomposition {
		if group.Type == TypeQuad && group.IsConcealed { // Ankan
			concealedPungKanCount++
		} else if group.Type == TypeTriplet && group.IsConcealed {
			// If Ron, and this group was completed by agariHai, it does NOT count as concealed for Suuankou.
			if !isTsumo && groupContainsTileID(group, agariHai.ID) {
				// This Pung was completed by Ron, so it's not "concealed" in the Suuankou sense.
			} else {
				concealedPungKanCount++
			}
		}
	}

	if concealedPungKanCount == 4 {
		return true, "Suuankou", 13
	}
	return false, "", 0
}

// checkTsuuiisou (All Honors) - Yakuman (13 Han)
func checkTsuuiisou(player *Player, allTiles []Tile) (bool, string, int) {
	if len(allTiles) == 0 { return false, "", 0} // Should not happen with a winning hand
	for _, tile := range allTiles {
		if !isHonor(tile) {
			return false, "", 0 // Found a non-honor tile
		}
	}

	// All tiles are honors. Now check for valid hand structure (standard or Chiitoitsu).
	decomposition, success := DecomposeWinningHand(player, allTiles)
	if success && decomposition != nil {
		// Check if it's a standard 4 melds + 1 pair structure
		// DecomposeWinningHand already validates the 4 groups + 1 pair structure.
		return true, "Tsuuiisou", 13
	}
	
	// If not standard, check for Chiitoitsu (7 pairs of honors)
	// Re-check allTiles for Chiitoitsu structure, as DecomposeWinningHand might fail for it.
	if IsChiitoitsu(allTiles) {
		return true, "Tsuuiisou", 13
	}

	return false, "", 0 // All honors but not a valid hand structure
}

// checkChinroutou (All Terminals) - Yakuman (13 Han)
func checkChinroutou(player *Player, allTiles []Tile) (bool, string, int) {
	if len(allTiles) == 0 { return false, "", 0}
	for _, tile := range allTiles {
		if !isTerminal(tile) { // Must be only terminals
			return false, "", 0
		}
		// Implicitly, if it's a terminal, it's not an honor.
		// No need for an explicit isHonor check if isTerminal is strict.
	}

	// All tiles are terminals. Check for valid hand structure.
	decomposition, success := DecomposeWinningHand(player, allTiles)
	if success && decomposition != nil {
		return true, "Chinroutou", 13
	}
	
	if IsChiitoitsu(allTiles) { // 7 pairs of terminals
		return true, "Chinroutou", 13
	}
	
	return false, "", 0 // All terminals but not a valid hand structure
}


// groupContainsTileID checks if a tile ID exists in a list of DecomposedGroup tiles.
// Local helper for Yaku checks.
func groupContainsTileID(group DecomposedGroup, tileID int) bool {
	for _, t := range group.Tiles {
		if t.ID == tileID {
			return true
		}
	}
	return false
}

// sequencesAreEqual checks if two sequences (represented by their tiles) are identical.
// Assumes tiles within each sequence are sorted.
func sequencesAreEqual(seq1Tiles []Tile, seq2Tiles []Tile) bool {
	if len(seq1Tiles) != 3 || len(seq2Tiles) != 3 {
		return false // Sequences must have 3 tiles
	}
	for i := 0; i < 3; i++ {
		// Compare by Suit and Value.
		if seq1Tiles[i].Suit != seq2Tiles[i].Suit || seq1Tiles[i].Value != seq2Tiles[i].Value {
			return false
		}
	}
	return true
}

// checkPinfu (Peaceful Hand) - 1 Han
func checkPinfu(player *Player, agariHai Tile, isMenzen bool, allTiles []Tile, gs *GameState) (bool, int) {
	if !isMenzen {
		return false, 0
	}

	decomposition, success := DecomposeWinningHand(player, allTiles)
	if !success || decomposition == nil || len(decomposition) != 5 {
		return false, 0
	}

	sequenceCount := 0
	var pairGroup DecomposedGroup
	foundPairInDecomp := false

	for _, group := range decomposition {
		if group.Type == TypeSequence {
			sequenceCount++
		} else if group.Type == TypePair {
			if foundPairInDecomp { return false, 0 }
			pairGroup = group
			foundPairInDecomp = true
		} else {
			return false, 0 // Must be only sequences and one pair
		}
	}
	if sequenceCount != 4 || !foundPairInDecomp {
		return false, 0
	}

	if len(pairGroup.Tiles) == 0 { return false, 0 }
	pairTile := pairGroup.Tiles[0]
	if isYakuhai(pairTile, player, gs) {
		return false, 0
	}

	ryanmenWaitFound := false
	for _, group := range decomposition {
		if group.Type == TypeSequence && groupContainsTileID(group, agariHai.ID) {
			t1, t2, t3 := group.Tiles[0], group.Tiles[1], group.Tiles[2]
			
			// Use Suit and Value for comparing agariHai with sequence tiles
			isAgariHaiT1 := (agariHai.Suit == t1.Suit && agariHai.Value == t1.Value)
			isAgariHaiT2 := (agariHai.Suit == t2.Suit && agariHai.Value == t2.Value)
			isAgariHaiT3 := (agariHai.Suit == t3.Suit && agariHai.Value == t3.Value)

			if isAgariHaiT2 { // Kanchan wait (e.g., 4-s-6 waiting on 5-s)
				ryanmenWaitFound = false; break
			}
			// Penchan waits: 1-2 waiting on 3 (agariHai is t3, value 3) or 7-8 waiting on 9 (agariHai is t3, value 9)
			// OR 1-2-3 waiting on 1 (agariHai is t1, value 1) or 7-8-9 waiting on 7 (agariHai is t1, value 7)
			if (isAgariHaiT3 && t3.Value == 3 && t2.Value == 2 && t1.Value == 1) || // 1-2 completed by 3
			   (isAgariHaiT1 && t1.Value == 7 && t2.Value == 8 && t3.Value == 9) {  // 8-9 completed by 7
				ryanmenWaitFound = false; break
			}
			// More general Penchan: if agariHai is t1 and t1.Value == 1
			if isAgariHaiT1 && t1.Value == 1 {
				ryanmenWaitFound = false; break;
			}
			// if agariHai is t3 and t3.Value == 9
			if isAgariHaiT3 && t3.Value == 9 {
				ryanmenWaitFound = false; break;
			}

			if isAgariHaiT1 || isAgariHaiT3 { // If not Kanchan or specific Penchans, it's Ryanmen
				ryanmenWaitFound = true; break
			}
		}
	}

	if !ryanmenWaitFound {
		return false, 0
	}
	return true, 1
}

// checkToitoi (All Pungs/Kans) - 2 Han
func checkToitoi(player *Player, allTiles []Tile) (bool, int) {
	decomposition, success := DecomposeWinningHand(player, allTiles)
	if !success || decomposition == nil || len(decomposition) != 5 {
		return false, 0
	}

	pungKanCount := 0
	pairCount := 0
	for _, group := range decomposition {
		if group.Type == TypeTriplet || group.Type == TypeQuad {
			pungKanCount++
		} else if group.Type == TypePair {
			pairCount++
		} else { // Found a sequence
			return false, 0
		}
	}

	if pungKanCount == 4 && pairCount == 1 {
		return true, 2
	}
	return false, 0
}

// checkIipeikou (One Pure Double Sequence) - 1 Han
func checkIipeikou(player *Player, isMenzen bool, allTiles []Tile) (bool, int) {
	if !isMenzen {
		return false, 0
	}
	decomposition, success := DecomposeWinningHand(player, allTiles)
	if !success || decomposition == nil || len(decomposition) != 5 {
		return false, 0
	}

	sequences := []DecomposedGroup{}
	for _, group := range decomposition {
		if group.Type == TypeSequence {
			sequences = append(sequences, group)
		}
	}

	if len(sequences) < 2 {
		return false, 0
	}

	identicalPairCount := 0
	for i := 0; i < len(sequences); i++ {
		for j := i + 1; j < len(sequences); j++ {
			if sequencesAreEqual(sequences[i].Tiles, sequences[j].Tiles) {
				identicalPairCount++
			}
		}
	}

	if identicalPairCount == 1 { // Exactly one pair of identical sequences
		return true, 1
	}
	// If identicalPairCount > 1, it could be Ryanpeikou (not handled here) or just multiple identical sequences if > 2.
	return false, 0
}

// checkSanankou (Three Concealed Pungs/Kans) - 2 Han
func checkSanankou(player *Player, agariHai Tile, isTsumo bool, allTiles []Tile) (bool, int) {
	// isMenzen is not directly a Sanankou requirement, but concealed nature of pungs is.
	decomposition, success := DecomposeWinningHand(player, allTiles)
	if !success || decomposition == nil || len(decomposition) != 5 {
		return false, 0
	}

	sanankouCounter := 0
	for _, group := range decomposition {
		isEffectiveConcealedMeld := false
		if group.Type == TypeTriplet && group.IsConcealed {
			if !isTsumo && groupContainsTileID(group, agariHai.ID) {
				// This triplet was completed by Ron, so it's not counted as concealed for Sanankou.
				isEffectiveConcealedMeld = false
			} else {
				isEffectiveConcealedMeld = true
			}
		} else if group.Type == TypeQuad && group.IsConcealed { // Ankan
			isEffectiveConcealedMeld = true // Ankans always count as concealed sets.
		}

		if isEffectiveConcealedMeld {
			sanankouCounter++
		}
	}

	if sanankouCounter == 3 {
		return true, 2
	}
	return false, 0
}


// --- Yakuman ---

func checkKokushiMusou(allTiles []Tile, agariHai Tile) (bool, string, int) {
	// Use the existing IsKokushiMusou check, assuming it checks the 13 unique + 1 pair structure.
	if IsKokushiMusou(allTiles) {
		// Check for Junsei (13-sided wait) - Player must be Menzenchin and win on any of the 13 required tiles *if* they already held the other 12 unique ones.
		// This requires knowing the hand state *before* the win. Complex.
		// Simple version: just return single Yakuman.
		return true, "Kokushi Musou", 13 // 13 Han = Yakuman
	}
	return false, "", 0
}

// --- 1 Han ---

func checkRiichi(player *Player, gs *GameState) (bool, int) {
	// We don't check eligibility here, just if the state is set.
	if player.IsRiichi {
		// Base Riichi is 1 Han. Ippatsu/Double Riichi are handled separately if applicable.
		return true, 1
	}
	return false, 0
}

func checkIppatsu(player *Player, gs *GameState) (bool, int) {
	// Check if Riichi was declared on the *immediately preceding* turn cycle.
	// And no calls (or closed Kan by self?) occurred between Riichi and win.
	// The IsIppatsu flag should be set correctly by the game logic.
	if player.IsRiichi && player.IsIppatsu {
		return true, 1
	}
	return false, 0
}

func checkMenzenTsumo(isTsumo bool, isMenzen bool) (bool, int) {
	// Player draws the winning tile themselves with a concealed hand.
	if isTsumo && isMenzen {
		return true, 1
	}
	return false, 0
}

// The checkPinfu placeholder is already removed and replaced by the full implementation
// in the previous step (where all new Yaku functions were defined).
// This SEARCH block is for the *old* placeholder which should no longer exist if prior step was complete.
// Assuming the prior step correctly placed the new checkPinfu function definition.

// checkRinshanKaihou (Win on a replacement tile after a Kan) - 1 Han
func checkRinshanKaihou(gs *GameState, isTsumo bool) (bool, int) {
	// Assumes gs.IsRinshanWin is true if the win was on a tile drawn after a Kan.
	// Must be a Tsumo win.
	if isTsumo && gs.IsRinshanWin {
		return true, 1
	}
	return false, 0
}

// checkChankan (Robbing a Kan) - 1 Han
func checkChankan(gs *GameState) (bool, int) {
	// Assumes gs.IsChankanOpportunity is true if another player just declared Shouminkan
	// and this player can Ron that tile.
	if gs.IsChankanOpportunity {
		return true, 1
	}
	return false, 0
}

func checkTanyao(allTiles []Tile) (bool, int) {
	// All tiles must be Simples (2-8 in Man, Pin, Sou). No Terminals or Honors.
	for _, tile := range allTiles {
		if !isSimple(tile) {
			return false, 0 // Found a terminal or honor
		}
	}
	// TODO: Check game rules setting for Kuitan (Open Tanyao allowed?). Assuming yes.
	return true, 1
}

func checkYakuhai(player *Player, gs *GameState, allTiles []Tile) ([]YakuResult, int) {
	results := []YakuResult{}
	totalHan := 0

	// Needs hand decomposition or counting of Pungs/Kans.
	// Let's use a simpler counting method for now.

	counts := CountTiles(allTiles) // Counts based on Suit-Value string

	checkHonorSet := func(suit string, value int, name string) {
		key := fmt.Sprintf("%s-%d", suit, value)
		if count, ok := counts[key]; ok && count >= 3 { // Found at least a Pung
			// Check if this honor tile is Yakuhai for the player
			// Create a representative tile to check
			repTile := Tile{Suit: suit, Value: value}
			if isYakuhai(repTile, player, gs) {
				results = append(results, YakuResult{fmt.Sprintf("Yakuhai (%s)", name), 1})
				totalHan += 1
			}
		}
	}

	// Dragons
	checkHonorSet("Dragon", 1, "White Dragon")
	checkHonorSet("Dragon", 2, "Green Dragon")
	checkHonorSet("Dragon", 3, "Red Dragon")
	// Winds
	checkHonorSet("Wind", 1, "East")
	checkHonorSet("Wind", 2, "South")
	checkHonorSet("Wind", 3, "West")
	checkHonorSet("Wind", 4, "North")

	return results, totalHan
}

func checkHaiteiHoutei(gs *GameState, isTsumo bool) (bool, string, int) {
	// Haitei Raoyue (Last Tile Tsumo)
	// Assumes gs.Wall being empty is checked *after* the player draws for Tsumo.
	if isTsumo && len(gs.Wall) == 0 {
		return true, "Haitei Raoyue", 1
	}
	// Houtei Raoyui (Win on Last Discard)
	// Assumes gs.IsHouteiDiscard is true if the current Ron is on the very last discard of the game.
	if !isTsumo && gs.IsHouteiDiscard {
		return true, "Houtei Raoyui", 1
	}
	return false, "", 0
}

// --- 2 Han ---

func checkChiitoitsu(player *Player, allTiles []Tile, isMenzen bool) (bool, int) {
	if !isMenzen {
		return false, 0
	}
	// Use the check from checks.go which should handle 7 distinct pairs.
	if IsChiitoitsu(allTiles) {
		// Note: Fu calculation for Chiitoitsu is fixed at 25.
		return true, 2
	}
	return false, 0
}

// The checkToitoi placeholder is already removed and replaced by the full implementation.

func checkShousangen(player *Player, allTiles []Tile) (bool, int) {
	// Two Pungs/Kans of Dragons + Pair of the third Dragon.
	// TODO: Needs decomposition or counting.

	counts := CountTiles(allTiles)
	dragonPungKanCount := 0
	dragonPairCount := 0

	checkDragon := func(value int) {
		key := fmt.Sprintf("Dragon-%d", value)
		if count, ok := counts[key]; ok {
			if count >= 3 {
				dragonPungKanCount++
			} else if count == 2 {
				dragonPairCount++
			}
		}
	}
	checkDragon(1) // White
	checkDragon(2) // Green
	checkDragon(3) // Red

	if dragonPungKanCount == 2 && dragonPairCount == 1 {
		// Also implicitly gets 2 Han from the two Yakuhai dragon pungs.
		// Shousangen itself is worth 2 Han. Total = 2 + 1 + 1 = 4 Han usually.
		// Return only the Shousangen value here, Yakuhai check adds the rest.
		return true, 2
	}
	return false, 0
}

func checkHonroutou(allTiles []Tile) (bool, int) {
	// All tiles must be Terminals (1, 9) or Honors (Winds, Dragons). No Simples (2-8).
	for _, tile := range allTiles {
		if isSimple(tile) {
			return false, 0 // Found a simple tile
		}
	}
	// All tiles are terminals or honors.
	// Often combined with Toitoi (-> 4 Han) or Chiitoitsu (still 2 Han?).
	return true, 2
}

// --- 3+ Han ---

// The checkIipeikou placeholder is already removed and replaced by the full implementation.

// checkRyanpeikou (Two Pure Double Sequences) - 3 Han
func checkRyanpeikou(player *Player, isMenzen bool, allTiles []Tile) (bool, int) {
	if !isMenzen {
		return false, 0
	}
	decomposition, success := DecomposeWinningHand(player, allTiles)
	if !success || decomposition == nil || len(decomposition) != 5 {
		return false, 0
	}

	sequences := []DecomposedGroup{}
	pairCount := 0
	for _, group := range decomposition {
		if group.Type == TypeSequence {
			sequences = append(sequences, group)
		} else if group.Type == TypePair {
			pairCount++
		} else { // Contains Pung/Kan, not valid for Ryanpeikou
			return false, 0
		}
	}

	if len(sequences) != 4 || pairCount != 1 { // Must be 4 sequences and 1 pair
		return false, 0
	}

	// Check for two pairs of identical sequences
	// Example: S1, S2, S3, S4. We need S1=S2 and S3=S4, and S1 != S3.
	// Or S1=S3 and S2=S4, and S1 != S2 etc.
	// A simpler approach: count occurrences of each unique sequence type.
	// We need two types of sequences, each appearing twice.
	
	if len(sequences) != 4 { return false, 0 } // Should be redundant due to above check

    // Sort sequences by their tile composition to make comparison easier for pairing up.
    // This isn't strictly necessary if using a map key, but good for deterministic logic.
    sort.Slice(sequences, func(i, j int) bool {
        for k := 0; k < 3; k++ {
            if sequences[i].Tiles[k].Suit != sequences[j].Tiles[k].Suit {
                return sequences[i].Tiles[k].Suit < sequences[j].Tiles[k].Suit
            }
            if sequences[i].Tiles[k].Value != sequences[j].Tiles[k].Value {
                return sequences[i].Tiles[k].Value < sequences[j].Tiles[k].Value
            }
        }
        return false
    })

    // Check for AABB pattern
    // (S0 == S1 && S2 == S3 && S0 != S2)
    cond1 := sequencesAreEqual(sequences[0].Tiles, sequences[1].Tiles) &&
             sequencesAreEqual(sequences[2].Tiles, sequences[3].Tiles) &&
             !sequencesAreEqual(sequences[0].Tiles, sequences[2].Tiles) // Ensure the two pairs are different

    // Check for ABAB pattern (after sort, this would become AABB if S0=S2, S1=S3)
    // This case is covered by the sort and then AABB check if S0,S1,S2,S3 are truly A,B,A,B
    // After sort, it becomes A,A,B,B.

    // Check for ABBA pattern (after sort, this would become AABB if S0=S3, S1=S2)
    // This case is also covered by sort and AABB.

	if cond1 {
		return true, 3
	}
	
	return false, 0
}


// checkJunchan (Junchan Taiyao or "Terminals in All Sets") - 3 Han (Menzen), 2 Han (Open)
func checkJunchan(player *Player, isMenzen bool, allTiles []Tile) (bool, int) {
	decomposition, success := DecomposeWinningHand(player, allTiles)
	if !success || decomposition == nil || len(decomposition) != 5 {
		return false, 0
	}

	for _, group := range decomposition {
		hasTerminalInGroup := false
		for _, tile := range group.Tiles {
			if isHonor(tile) {
				return false, 0 // No honor tiles allowed for Junchan
			}
			if isTerminal(tile) {
				hasTerminalInGroup = true
			}
		}
		if !hasTerminalInGroup {
			return false, 0 // This group does not contain a terminal
		}
	}

	// All groups (4 melds + 1 pair) contain at least one terminal, and no honor tiles were found.
	han := 2
	if isMenzen {
		han = 3
	}
	return true, han
}


func checkHonitsu(allTiles []Tile, isMenzen bool) (bool, int) {
	// Hand uses only tiles from ONE suit (Man, Pin, or Sou) plus any Honor tiles.
	targetSuit := ""
	hasHonors := false

	for _, tile := range allTiles {
		if isHonor(tile) {
			hasHonors = true
		} else if isSimple(tile) || isTerminal(tile) {
			if targetSuit == "" {
				targetSuit = tile.Suit // First suit found
			} else if tile.Suit != targetSuit {
				return false, 0 // Found a second suit
			}
		} else {
			return false, 0 // Should not happen (tile isn't honor, simple, or terminal?)
		}
	}

	// If we only found honors or tiles from one suit (or only honors)
	if targetSuit != "" || hasHonors {
		// Need at least one number tile to distinguish from Tsuuiisou (all honors yakuman)
		hasNumbers := false
		for _, tile := range allTiles {
			if !isHonor(tile) {
				hasNumbers = true
				break
			}
		}
		if !hasNumbers { // Only honors -> Tsuuiisou (should be caught by Yakuman check)
			return false, 0
		}

		// It's Honitsu
		if isMenzen {
			return true, 3
		}
		return true, 2

	}

	return false, 0 // Only found invalid tiles? Or empty hand?
}

// --- 6+ Han ---

func checkChinitsu(allTiles []Tile, isMenzen bool) (bool, int) {
	// Hand uses only tiles from ONE suit. No Honor tiles allowed.
	targetSuit := ""

	for _, tile := range allTiles {
		if isHonor(tile) {
			return false, 0 // Found an honor tile
		} else if isSimple(tile) || isTerminal(tile) {
			if targetSuit == "" {
				targetSuit = tile.Suit // First suit found
			} else if tile.Suit != targetSuit {
				return false, 0 // Found a second suit
			}
		} else {
			return false, 0 // Invalid tile type
		}
	}

	// If we reached here, all tiles are from the targetSuit (and targetSuit must have been set)
	if targetSuit != "" {
		if isMenzen {
			return true, 6
		}
		return true, 5
	}

	return false, 0 // Empty hand or only invalid tiles
}
