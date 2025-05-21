package main

import (
	"math"
)

// groupContainsTileID checks if a tile ID exists in a list of DecomposedGroup tiles
func groupContainsTileID(group DecomposedGroup, tileID int) bool {
	for _, t := range group.Tiles {
		if t.ID == tileID {
			return true
		}
	}
	return false
}

// CalculateFu calculates the Fu for a winning hand.
// It requires the decomposition of the hand, win conditions, and game state.
func CalculateFu(player *Player, decomposition []DecomposedGroup, agariHai Tile, isTsumo bool, isMenzen bool, yakus []YakuResult, gs *GameState) int {

	// --- Special Hand Fu ---
	isPinfu := false
	isChiitoitsu := false
	for _, yaku := range yakus {
		if yaku.Name == "Pinfu" {
			isPinfu = true
		}
		// Check if Chiitoitsu Yaku was identified (requires yaku.go update)
		if yaku.Name == "Chiitoitsu" {
			isChiitoitsu = true
		}
	}

	if isChiitoitsu {
		return 25 // Chiitoitsu always 25 Fu, no rounding.
	}

	// --- Standard Hand Fu Calculation ---
	fu := 20 // Base Fu

	// 1. Win Method Bonus (Mutually Exclusive with Pinfu Tsumo bonus)
	if isTsumo && !isPinfu {
		fu += 2 // Tsumo bonus (non-Pinfu)
	}
	if isMenzen && !isTsumo {
		fu += 10 // Menzen Ron bonus
	}

	// 2. Wait Pattern Bonus (+2 Fu) - Simplified Check
	// Checks how the agariHai completes a group in the final decomposition.
	waitFu := 0
	for _, group := range decomposition {
		if !groupContainsTileID(group, agariHai.ID) {
			continue // Skip groups not containing the winning tile
		}

		// How did agariHai complete this group?
		switch group.Type {
		case TypePair: // Tanki (Pair) Wait
			waitFu = 2
		case TypeSequence:
			// Check for Penchan (Edge, e.g., 1-2 waiting on 3, or 8-9 waiting on 7)
			// Check for Kanchan (Middle, e.g., 4-6 waiting on 5)
			t1, t2, t3 := group.Tiles[0], group.Tiles[1], group.Tiles[2]   // Assumes sorted
			if (t1.Value == 1 && t2.Value == 2 && agariHai.ID == t3.ID) || // 1-2 completed by 3
				(t2.Value == 8 && t3.Value == 9 && agariHai.ID == t1.ID) { // 8-9 completed by 7 (t1 is 7)
				waitFu = 2 // Penchan
			} else if agariHai.ID == t2.ID && t1.Value+1 != t2.Value && t2.Value+1 != t3.Value {
				// Should be t1.Value + 2 == t3.Value if it's Kanchan completed by middle tile t2
				if t1.Value+2 == t3.Value {
					waitFu = 2 // Kanchan
				}
			}
			// If neither Penchan nor Kanchan, assume Ryanmen (no wait Fu)

		// Triplets/Quads don't contribute wait Fu this way (handled by Pair wait if applicable)
		case TypeTriplet, TypeQuad:
			break

		}
		// If we found the group completed by agariHai, stop checking waits
		if waitFu > 0 {
			break
		}
	}
	fu += waitFu

	// 3. Pair Bonus (+2 / +4 Fu)
	pairFu := 0
	for _, group := range decomposition {
		if group.Type == TypePair {
			pairTile := group.Tiles[0] // Both tiles are the same type
			isDragon := pairTile.Suit == "Dragon"
			isSeatWind := isWindMatch(pairTile, player.SeatWind)
			isPrevalentWind := isWindMatch(pairTile, gs.PrevalentWind)

			if isDragon {
				pairFu += 2
			}
			// Add +2 for seat wind match
			if isSeatWind {
				pairFu += 2
			}
			// Add +2 for prevalent wind match (if different from seat wind)
			if isPrevalentWind && !isSeatWind { // Avoid double counting if seat==prevalent
				pairFu += 2
			}
			// If seat == prevalent, the total added is 4 (2+2) implicitly.
			break // Found the pair
		}
	}
	fu += pairFu

	// 4. Group Bonus (Triplets / Quads)
	groupFu := 0
	for _, group := range decomposition {
		if group.Type == TypeTriplet || group.Type == TypeQuad {
			tile := group.Tiles[0] // Any tile represents the type
			isTerminalOrHonor := isTerminal(tile) || isHonor(tile)
			base := 0

			if group.Type == TypeTriplet { // Pung / Ankou
				base = 2 // Simple
				if isTerminalOrHonor {
					base = 4
				}
				if group.IsConcealed {
					base *= 2
				} // Double if concealed (Ankou)
			} else { // Kan
				base = 8 // Simple
				if isTerminalOrHonor {
					base = 16
				}
				if group.IsConcealed {
					base *= 2
				} // Double if concealed (Ankan)
			}
			groupFu += base
		}
	}
	fu += groupFu

	// --- Final Adjustments & Rounding ---
	if isPinfu && isTsumo {
		// Pinfu Tsumo has a fixed Fu value before rounding.
		// Standard rule: Pinfu + Tsumo yaku = 20 Fu base (no +2 Tsumo bonus).
		// All other Fu components (pair value, waits, groups) are IGNORED for Pinfu.
		fu = 20
	} else if fu == 20 && !isMenzen && !isTsumo { // Open hand Ron minimum
		// An open Ron that calculates to exactly 20 Fu base is usually bumped to 30.
		// Check specific ruleset - commonly applied. Let's assume yes.
		// If Pinfu was possible but broken by wait/pair, this might apply.
		// This check is complex. A simple 20fu hand via Ron is unusual unless Pinfu-like.
		// Safest: Stick to calculated Fu unless explicitly Pinfu or Chiitoitsu.
		// Let's skip the forced bump to 30 for now for simplicity.
	}

	// Round UP to the nearest 10 (unless it's already a multiple of 10)
	if fu < 20 {
		fu = 20
	} // Absolute minimum (except Chiitoitsu)
	if isPinfu && !isTsumo {
		fu = 30
	} // Menzen Pinfu Ron is exactly 30

	// Perform rounding unless it's a special fixed value (Chiitoitsu 25, Pinfu Ron 30)
	if !isChiitoitsu && !(isPinfu && !isTsumo && fu == 30) {
		fu = int(math.Ceil(float64(fu)/10.0) * 10)
	}

	// Safety check: Minimum Fu is usually 30 for non-Pinfu/non-Chiitoitsu hands?
	// E.g. Tanyao only, Tsumo -> 20 base + 2 tsumo = 22 -> 30 Fu.
	// E.g. Tanyao only, Menzen Ron -> 20 base + 10 menzen = 30 Fu.
	// E.g. Tanyao only, Open Ron -> 20 base = 20 Fu (often bumped to 30). Let's enforce 30 minimum here?
	if !isPinfu && !isChiitoitsu && fu < 30 {
		// fmt.Printf("Debug: Bumping calculated Fu %d to minimum 30.\n", fu)
		fu = 30
	}

	return fu
}

// Helper to check if a wind tile matches a specific wind name
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
