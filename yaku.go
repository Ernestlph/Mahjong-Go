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
	if ok, han := checkPinfu(player, agariHai, isMenzen, allTiles); ok {
		results = append(results, YakuResult{"Pinfu", han})
		totalHan += han
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
	// TODO: Needs flag passed from HandleKanAction if win was on Rinshan tile.
	// if isRinshanWin {
	// 	results = append(results, YakuResult{"Rinshan Kaihou", 1})
	// 	totalHan += 1
	// }

	// Chankan (Robbing a Kan)
	// TODO: Needs flag passed from HandleKanAction if win was Chankan.
	// if isChankanWin {
	// 	results = append(results, YakuResult{"Chankan", 1})
	// 	totalHan += 1
	// }

	// --- 2 Han Yaku ---
	chiitoitsuFound := false
	if ok, han := checkChiitoitsu(player, allTiles, isMenzen); ok {
		results = append(results, YakuResult{"Chiitoitsu", han})
		totalHan += han
		chiitoitsuFound = true // Chiitoitsu doesn't combine well with sequence Yaku.
	}

	// Toitoi (All Pungs) - Cannot be Chiitoitsu
	if !chiitoitsuFound {
		if ok, han := checkToitoi(player, allTiles); ok {
			results = append(results, YakuResult{"Toitoi", han})
			totalHan += han
		}
	}

	// Sanankou (Three Concealed Pungs) - Cannot be Chiitoitsu
	// TODO: Needs robust decomposition or specific checks. Needs care with Ron vs Tsumo completion.
	// if !chiitoitsuFound {
	//     if ok, han := checkSanankou(player, agariHai, isTsumo, isMenzen, allTiles); ok {
	//         results = append(results, YakuResult{"Sanankou", 2})
	//         totalHan += han
	//     }
	// }

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

	if !chiitoitsuFound { // Iipeikou needs sequences
		if ok, han := checkIipeikou(player, isMenzen, allTiles); ok {
			results = append(results, YakuResult{"Iipeikou", han})
			totalHan += han

		}
	}

	// Ryanpeikou (Two Iipeikou) - Overrides Iipeikou and Chiitoitsu
	// TODO: Implement Ryanpeikou check (needs decomposition)
	// if ok, han := checkRyanpeikou(player, isMenzen, allTiles); ok {
	//     // Remove Iipeikou and Chiitoitsu if found
	//     results = removeYakuByName(results, "Iipeikou")
	//     results = removeYakuByName(results, "Chiitoitsu")
	//     totalHan = calculateTotalHan(results) // Recalculate Han
	//     results = append(results, YakuResult{"Ryanpeikou", 3})
	//     totalHan += 3
	//     iipeikouFound = false // No longer counts as single iipeikou
	//     chiitoitsuFound = false
	// }

	honitsuFound := false
	if ok, han := checkHonitsu(allTiles, isMenzen); ok {
		results = append(results, YakuResult{"Honitsu", han})
		totalHan += han
		honitsuFound = true
	}

	// Junchan (Outside Hand with Terminals) - Cannot be Chiitoitsu
	// TODO: Implement Junchan (needs decomposition, check all groups+pair have terminal)
	// if !chiitoitsuFound {
	//     if ok, han := checkJunchan(player, isMenzen, allTiles); ok {
	//         results = append(results, YakuResult{"Junchan", han}) // 3 concealed, 2 open
	//         totalHan += han
	//     }
	// }

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

func checkPinfu(player *Player, agariHai Tile, isMenzen bool, allTiles []Tile) (bool, int) {
	if !isMenzen {
		return false, 0
	}
	// Requires:
	// 1. Menzenchin (checked)
	// 2. 4 Sequences (Chi) + 1 Pair
	// 3. Pair is NOT Yakuhai (Dragons, Seat Wind, Prevalent Wind)
	// 4. Wait must be Ryanmen (two-sided sequence wait, e.g., waiting on 3 or 6 for a 4-5 shape)

	// TODO: This requires robust hand decomposition and wait analysis. Very complex.
	// Placeholder: Return false for now.
	// --- Rough steps ---
	// Decompose hand into 4 groups + 1 pair. Check all groups are sequences.
	// Find the pair. Check if isYakuhai(pairTile, player, gs).
	// Determine the wait type based on hand *before* agariHai. Check if it was Ryanmen.

	return false, 0 // Placeholder
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
	if isTsumo && len(gs.Wall) == 0 {
		return true, "Haitei Raoyue", 1
	}
	// Houtei Raoyui (Win on Last Discard)
	// Need to know if the discard causing the Ron was the very last discard of the game *after* the last tile was drawn.
	// TODO: Requires tracking if the wall was exhausted *before* the final discard.
	// if !isTsumo && isLastDiscardOfTheRound {
	//     return true, "Houtei Raoyui", 1
	// }
	return false, "", 0 // Placeholder for Houtei
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

func checkToitoi(player *Player, allTiles []Tile) (bool, int) {
	// Hand must be 4 Pungs/Kans + 1 Pair.
	// Cannot coexist with Chiitoitsu. Cannot contain sequences.

	// TODO: Requires hand decomposition.
	// --- Rough steps ---
	// Decompose hand. Check if all 4 groups are Pungs or Kans.
	// If yes, return true, 2 Han.

	// Simple check (less reliable): Count number tiles. If < 3*4 = 12, cannot be 4 sequences.
	// Count pairs: Should have exactly one pair type appearing twice, others 3 or 4 times.
	counts := CountTiles(allTiles)
	numTripletsOrQuads := 0
	numPairs := 0

	for _, count := range counts {
		if count == 2 {
			numPairs++
		} else if count == 3 || count == 4 {
			numTripletsOrQuads++
		} else {
			return false, 0 // Count isn't 2, 3, or 4 - invalid for Toitoi
		}

	}

	// If hand is 4 Pungs/Kans and 1 Pair
	if numTripletsOrQuads == 4 && numPairs == 1 {
		// Extra check: If it contains sequences, it's not Toitoi (this check is weak)
		// A proper decomposition is needed. Assume for now simple count is enough.
		return true, 2
	}

	return false, 0 // Placeholder
}

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

func checkIipeikou(player *Player, isMenzen bool, allTiles []Tile) (bool, int) {
	if !isMenzen {
		return false, 0
	}
	// Two identical sequences. E.g., 2p 3p 4p + 2p 3p 4p.
	// Cannot coexist with Chiitoitsu.

	// TODO: Requires hand decomposition.
	// --- Rough steps ---
	// Decompose hand. Check if exactly two of the groups are identical sequences.

	return false, 0 // Placeholder
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
