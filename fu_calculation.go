package main

import (
	"fmt"
	"math"
	// "strings" // Not used directly in this file after updates
)

// CalculateFu calculates the Fu for a winning hand.
// It requires the decomposition of the hand, win conditions, and game state.
func CalculateFu(player *Player, decomposition []DecomposedGroup, agariHai Tile, isTsumo bool, isMenzen bool, yakus []YakuResult, gs *GameState) int {
	isPinfu := false
	isChiitoitsu := false
	isYakuman := false // Check if any Yakuman is present (fu calc might be skipped or different)

	for _, yaku := range yakus {
		if yaku.Name == "Pinfu" {
			isPinfu = true
		}
		if yaku.Name == "Chiitoitsu" {
			isChiitoitsu = true
		}
		if yaku.Han >= 13 || yaku.Name == "Kokushi Musou" || yaku.Name == "Kokushi Musou Juusanmenmachi" { // Add other Yakuman names if they don't always give >=13 Han
			isYakuman = true
		}
	}

	if isYakuman {
		gs.AddToGameLog("Fu Calc: Yakuman hand, Fu calculation typically skipped for points. Returning 0 for Fu value.")
		return 0 // Or a conventional value if your scoring system uses it for Yakuman, e.g. some might treat as Mangan base
	}

	if isChiitoitsu {
		gs.AddToGameLog("Fu Calc: Chiitoitsu, Fu fixed at 25.")
		return 25 // Chiitoitsu always 25 Fu, no rounding needed beyond this fixed value.
	}

	// --- Standard Hand Fu Calculation ---
	currentFu := 20 // Base Fu (Foutei)
	gs.AddToGameLog(fmt.Sprintf("Fu Calc: Base Fu = %d", currentFu))

	// 1. Win Method Bonus
	// Pinfu Tsumo is a special case: it's 20 Fu total, no +2 Tsumo bonus. This is handled later.
	if isTsumo && !isPinfu {
		currentFu += 2 // Tsumo bonus (non-Pinfu)
		gs.AddToGameLog(fmt.Sprintf("Fu Calc: +2 (Tsumo, not Pinfu). Total: %d", currentFu))
	}
	if isMenzen && !isTsumo { // Menzen Ron bonus
		currentFu += 10
		gs.AddToGameLog(fmt.Sprintf("Fu Calc: +10 (Menzen Ron). Total: %d", currentFu))
	}

	// If Pinfu, all other Fu components (waits, pair values, group values) are ignored.
	// Pinfu Ron is 30 Fu total. Pinfu Tsumo is 20 Fu total.
	if isPinfu {
		if isTsumo {
			gs.AddToGameLog(fmt.Sprintf("Fu Calc: Pinfu Tsumo. Fu fixed at 20."))
			return 20
		} else { // Pinfu Ron
			gs.AddToGameLog(fmt.Sprintf("Fu Calc: Pinfu Ron. Fu fixed at 30."))
			return 30
		}
	}

	// 2. Wait Pattern Bonus (+2 Fu)
	// Only applies if the hand is NOT Pinfu.
	// Requires decomposition to identify the group completed by agariHai.
	if decomposition != nil {
		waitFuAdded := 0
		for _, group := range decomposition {
			if !groupContainsTileID(group, agariHai.ID) {
				continue // Winning tile not in this group
			}

			// How did agariHai complete this group?
			switch group.Type {
			case TypePair: // Tanki (Pair) Wait
				waitFuAdded = 2
				gs.AddToGameLog(fmt.Sprintf("Fu Calc: +2 (Tanki Wait on Pair %s).", agariHai.Name))
			case TypeSequence:
				// Tiles t1, t2, t3 are the sorted tiles of the sequence in the decomposition.
				// agariHai is the tile that completed this sequence.
				t1, t2, t3 := group.Tiles[0], group.Tiles[1], group.Tiles[2]

				// Penchan (Edge wait): 1-2 waiting on 3, or 7-8 waiting on 9.
				// If agariHai is the '3' of a 1-2-3 sequence:
				if agariHai.ID == t3.ID && t1.Value == 1 && t2.Value == 2 && t3.Value == 3 {
					waitFuAdded = 2
					gs.AddToGameLog(fmt.Sprintf("Fu Calc: +2 (Penchan Wait %s-%s on %s).", t1.Name, t2.Name, agariHai.Name))
				}
				// If agariHai is the '7' of a 7-8-9 sequence:
				if agariHai.ID == t1.ID && t1.Value == 7 && t2.Value == 8 && t3.Value == 9 {
					waitFuAdded = 2
					gs.AddToGameLog(fmt.Sprintf("Fu Calc: +2 (Penchan Wait %s-%s on %s).", t2.Name, t3.Name, agariHai.Name))
				}

				// Kanchan (Middle wait): e.g., 4-6 waiting on 5.
				// AgariHai is the middle tile (t2) of the formed sequence.
				if agariHai.ID == t2.ID && t1.Value+1 == t2.Value && t2.Value+1 == t3.Value {
					waitFuAdded = 2
					gs.AddToGameLog(fmt.Sprintf("Fu Calc: +2 (Kanchan Wait %s_%s on %s).", t1.Name, t3.Name, agariHai.Name))
				}
				// Ryanmen (Open wait) and Shanpon (two-pair wait where one becomes Pung) get no *wait* Fu here.
				// Shanpon Pung Fu is handled by Group Bonus.
			case TypeTriplet, TypeQuad:
				// If agariHai completed a Pung (Shanpon wait), it's handled by Tanki on the other pair,
				// or by the Pung's value itself. No specific "wait fu" for completing a Pung.
				break
			}
			if waitFuAdded > 0 {
				break
			} // Found the wait type related to agariHai
		}
		currentFu += waitFuAdded
		if waitFuAdded > 0 {
			gs.AddToGameLog(fmt.Sprintf("Fu Calc: Wait Fu added. Total: %d", currentFu))
		}
	}

	// 3. Pair Bonus (+2 / +4 Fu)
	// Only applies if not Pinfu.
	if decomposition != nil {
		pairFuAdded := 0
		for _, group := range decomposition {
			if group.Type == TypePair {
				pairTile := group.Tiles[0]
				tempPairFu := 0
				reason := ""
				if pairTile.Suit == "Dragon" {
					tempPairFu += 2
					reason += fmt.Sprintf("Dragon Pair (%s); ", pairTile.Name)
				}
				if isWindMatch(pairTile, player.SeatWind) {
					tempPairFu += 2
					reason += fmt.Sprintf("Seat Wind (%s); ", player.SeatWind)
				}
				// Prevalent wind bonus stacks unless it's the same as seat wind (already counted)
				if isWindMatch(pairTile, gs.PrevalentWind) && !(player.SeatWind == gs.PrevalentWind && isWindMatch(pairTile, player.SeatWind)) {
					tempPairFu += 2
					reason += fmt.Sprintf("Prevalent Wind (%s); ", gs.PrevalentWind)
				}
				if tempPairFu > 0 {
					gs.AddToGameLog(fmt.Sprintf("Fu Calc: +%d (Pair Bonus: %s).", tempPairFu, reason))
					pairFuAdded = tempPairFu
				}
				break // Found the pair
			}
		}
		currentFu += pairFuAdded
		if pairFuAdded > 0 {
			gs.AddToGameLog(fmt.Sprintf("Fu Calc: Pair Fu added. Total: %d", currentFu))
		}
	}

	// 4. Group Bonus (Triplets / Quads)
	// Only applies if not Pinfu.
	if decomposition != nil {
		groupFuAdded := 0
		for _, group := range decomposition {
			if group.Type == TypeTriplet || group.Type == TypeQuad {
				tile := group.Tiles[0] // Representative tile
				isTermOrHonor := IsTerminal(tile) || IsHonor(tile)
				base := 0
				meldTypeStr := ""

				if group.Type == TypeTriplet { // Pung/Ankou
					if group.IsConcealed { // Ankou (Concealed Pung)
						base = IfElseInt(isTermOrHonor, 8, 4)
						meldTypeStr = "Ankou"
					} else { // Pon (Open Pung)
						base = IfElseInt(isTermOrHonor, 4, 2)
						meldTypeStr = "Pon"
					}
				} else { // Kan
					if group.IsConcealed { // Ankan (Concealed Kan)
						base = IfElseInt(isTermOrHonor, 32, 16)
						meldTypeStr = "Ankan"
					} else { // Daiminkan/Shouminkan (Open Kan)
						base = IfElseInt(isTermOrHonor, 16, 8)
						meldTypeStr = "Open Kan" // Generic for Daimin/Shoumin
					}
				}
				gs.AddToGameLog(fmt.Sprintf("Fu Calc: +%d (%s of %s %s).",
					base, meldTypeStr, tile.Name, If(isTermOrHonor, "(Term/Honor)", "(Simple)")))
				groupFuAdded += base
			}
		}
		currentFu += groupFuAdded
		if groupFuAdded > 0 {
			gs.AddToGameLog(fmt.Sprintf("Fu Calc: Group Fu added. Total: %d", currentFu))
		}
	}

	// --- Final Adjustments & Rounding (Only if not Pinfu/Chiitoitsu, as they have fixed Fu) ---
	// Pinfu was handled earlier. Chiitoitsu returns 25 directly.

	// Kuisagari for open hands (some rules): If an open hand calculates to 20 Fu, it's bumped to 30.
	// This is often for hands that would be Pinfu if closed.
	// For simplicity, we'll rely on a general minimum Fu rule after rounding if not Pinfu/Chiitoitsu.
	// If currentFu == 20 && !isMenzen && !isTsumo { currentFu = 30; gs.AddToGameLog("Fu Calc: Open Ron 20fu bumped to 30.")}

	// Round UP to the nearest 10
	originalFuBeforeRounding := currentFu
	if currentFu%10 != 0 {
		currentFu = int(math.Ceil(float64(currentFu)/10.0) * 10)
		gs.AddToGameLog(fmt.Sprintf("Fu Calc: Rounded %d -> %d.", originalFuBeforeRounding, currentFu))
	}

	// Minimum Fu (usually 30 for non-Pinfu/non-Chiitoitsu hands after all calculations and rounding)
	if currentFu < 30 {
		gs.AddToGameLog(fmt.Sprintf("Fu Calc: Adjusted to minimum 30 from %d (non-Pinfu/Chiitoitsu).", currentFu))
		currentFu = 30
	}

	gs.AddToGameLog(fmt.Sprintf("Fu Calc: Final Fu = %d", currentFu))
	return currentFu
}

// isWindMatch helper: checks if a wind tile matches a specific wind name (e.g., player's seat wind string)
func isWindMatch(tile Tile, windName string) bool {
	if tile.Suit != "Wind" {
		return false
	}
	switch windName {
	case "East":
		return tile.Value == 1
	case "South":
		return tile.Value == 2
	case "West":
		return tile.Value == 3
	case "North":
		return tile.Value == 4
	default:
		return false
	}
}

// groupContainsTileID checks if a specific tile (by ID) is part of a decomposed group.
// Used for determining if agariHai completed a specific group.
func groupContainsTileID(group DecomposedGroup, tileID int) bool {
	for _, t := range group.Tiles {
		if t.ID == tileID {
			return true
		}
	}
	return false
}
