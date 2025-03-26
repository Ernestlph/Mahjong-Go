package main

import (
	"fmt"
	"math"
)

// UpdateFuritenStatus checks and updates the player's Furiten state.
// Called typically after the player discards.
func UpdateFuritenStatus(player *Player, gs *GameState) {
	// --- Permanent Furiten (Riichi) ---
	// If in Riichi and previously missed a Ron on a tile that is one of their waits,
	// they become permanently Furiten for that hand. This state needs to be tracked.
	// Requires storing waits at Riichi declaration and checking discards against them.
	// Complex: Skip permanent Riichi Furiten for now.

	// --- Temporary Furiten ---
	// Occurs if player is Tenpai and *any* of their winning tiles (waits)
	// are present in their *own discard pile*.
	// Also occurs for one turn if player *could have* Ron'd a discard but chose not to.

	player.IsFuriten = false // Reset before checking

	// Check 1: Tenpai waits in own discards?
	// Need to check Tenpai based on hand *before* the latest discard? No, check current state.
	// If currently Tenpai, check waits against all previous discards.
	if IsTenpai(player.Hand, player.Melds) {
		waits := FindTenpaiWaits(player.Hand, player.Melds)
		if len(waits) > 0 {
			// fmt.Printf("Debug: %s is Tenpai, waits: %v\n", player.Name, TilesToNames(waits)) // Debug
		}

		for _, waitTile := range waits {
			for _, discarded := range player.Discards {
				// Check Suit and Value match
				if discarded.Suit == waitTile.Suit && discarded.Value == waitTile.Value {
					// fmt.Printf("Debug: Player %s is temporarily Furiten (wait %s found in own discards)\n", player.Name, waitTile.Name)
					player.IsFuriten = true
					return // Found a reason for Furiten, no need to check further
				}
			}
		}
	}

	// Check 2: Missed Ron (Temporary Furiten for one turn cycle)
	// This requires tracking if a player declined Ron on the *previous* discard.
	// Needs additional state in GameState or Player, e.g., `DeclinedRonOnTurn int`.
	// If gs.TurnNumber == player.DeclinedRonOnTurn + 1 (approx), set Furiten.
	// Skip this complex part for now.

	// If no conditions met, player is not Furiten.
	if !player.IsFuriten {
		// fmt.Printf("Debug: %s is not Furiten.\n", player.Name) // Debug
	}
}

// --- Scoring (Placeholders) ---

// Payment represents the points transferred in a win.
type Payment struct {
	Description    string // e.g., "Mangan", "1000/2000 points"
	RonValue       int    // Points paid by discarder on Ron
	TsumoDealer    int    // Points paid by non-dealers on dealer Tsumo
	TsumoNonDealer int    // Points paid by dealer/others on non-dealer Tsumo
}

// rules.go / CalculatePointPayment (example modification)
func CalculatePointPayment(han, fu int, isDealer, isTsumo bool, honba, riichiSticks int) Payment {
	// *** IMPORTANT: This needs full score table implementation ***
	// Including limits: Mangan, Haneman, Baiman, Sanbaiman, Yakuman

	isYakuman := false // TODO: Determine if han corresponds to a Yakuman (e.g., han >= 13) or from Yakuman Yaku

	if isYakuman {
		// Yakuman Scoring (fixed base points)
		yakumanMultiplier := 1                            // For single Yakuman. TODO: Handle multiple Yakuman?
		basePoints := 8000.0 * float64(yakumanMultiplier) // Make basePoints float64 explicitly
		// Convert IfElseInt results to float64 for multiplication
		ronValue := basePoints * float64(IfElseInt(isDealer, 6, 4))
		tsumoDealerPays := basePoints * float64(IfElseInt(isDealer, 0, 2))    // Dealer pays on non-dealer Yakuman Tsumo
		tsumoNonDealerPays := basePoints * float64(IfElseInt(isDealer, 2, 1)) // Non-dealers pay

		// Convert results back to int after calculation (usually rounding up/ceiling)
		// Apply ceiling and convert to int, rounding to nearest 100
		ronValueInt := int(math.Ceil(ronValue/100.0)) * 100
		tsumoDealerPaysInt := int(math.Ceil(tsumoDealerPays/100.0)) * 100
		tsumoNonDealerPaysInt := int(math.Ceil(tsumoNonDealerPays/100.0)) * 100

		// Add Honba bonus (300 per honba for Ron, 100 per player for Tsumo)
		ronValueInt += honba * 300
		tsumoDealerPaysInt += honba * 100
		tsumoNonDealerPaysInt += honba * 100

		desc := fmt.Sprintf("Yakuman") // TODO: Add specific Yakuman names?

		if isTsumo {
			if isDealer {
				desc += fmt.Sprintf(" (%d All)", tsumoNonDealerPaysInt)
			} else {
				desc += fmt.Sprintf(" (%d/%d)", tsumoNonDealerPaysInt, tsumoDealerPaysInt)
			}
		} else {
			desc += fmt.Sprintf(" (%d Ron)", ronValueInt)
		}

		return Payment{Description: desc, RonValue: ronValueInt, TsumoDealer: tsumoDealerPaysInt, TsumoNonDealer: tsumoNonDealerPaysInt}

	} else {
		// Standard Scoring using Han & Fu
		if fu < 20 {
			fu = 20
		} // Ensure minimum Fu (already done in CalculateFu?)
		if fu == 25 { // Chiitoitsu
			// Force minimum Han for Chiitoitsu? Usually 2.
			if han < 2 {
				han = 2
			}
		}

		basePoints := float64(fu) * math.Pow(2, float64(han+2)) // basePoints is float64
		limitName := ""

		// Apply Score Limits (Mangan etc.)
		limitBasePoints := 0.0 // Use float for limit base points too
		if han >= 13 {
			limitBasePoints = 8000
			limitName = "Yakuman (Counted)"
		} else // Counted Yakuman
		if han >= 11 {
			limitBasePoints = 6000
			limitName = "Sanbaiman"
		} else if han >= 8 {
			limitBasePoints = 4000
			limitName = "Baiman"
		} else if han >= 6 {
			limitBasePoints = 3000
			limitName = "Haneman"
		} else if han >= 5 || (han == 4 && fu >= 40) || (han == 3 && fu >= 70) {
			limitBasePoints = 2000
			limitName = "Mangan"
		}

		if limitBasePoints > 0 {
			basePoints = limitBasePoints // Apply the limit base points
		}

		// Calculate payments using float64 basePoints
		ronValueF := basePoints * float64(IfElseInt(isDealer, 6, 4))
		tsumoDealerPaysF := basePoints * float64(IfElseInt(isDealer, 0, 2))    // Dealer pays on non-dealer Tsumo
		tsumoNonDealerPaysF := basePoints * float64(IfElseInt(isDealer, 2, 1)) // Non-dealers pay

		// Round UP to nearest 100 and convert to int
		ronValue := int(math.Ceil(ronValueF/100.0)) * 100
		tsumoDealerPays := int(math.Ceil(tsumoDealerPaysF/100.0)) * 100
		tsumoNonDealerPays := int(math.Ceil(tsumoNonDealerPaysF/100.0)) * 100

		// Add Honba bonus (300 per honba for Ron, 100 per player for Tsumo)
		ronValue += honba * 300
		tsumoDealerPays += honba * 100
		tsumoNonDealerPays += honba * 100

		desc := ""
		if limitName != "" {
			desc = limitName
		} else {
			desc = fmt.Sprintf("%d Han, %d Fu", han, fu)
		}

		if isTsumo {
			if isDealer {
				desc += fmt.Sprintf(" (%d All)", tsumoNonDealerPays)
			} else {
				desc += fmt.Sprintf(" (%d/%d)", tsumoNonDealerPays, tsumoDealerPays)
			}
		} else {
			desc += fmt.Sprintf(" (%d Ron)", ronValue)
		}

		return Payment{Description: desc, RonValue: ronValue, TsumoDealer: tsumoDealerPays, TsumoNonDealer: tsumoNonDealerPays}
	}
}

// TransferPoints updates player scores based on the payment structure.
func TransferPoints(gs *GameState, winner, discarder *Player, isTsumo bool, payment Payment) {
	winnerIndex := gs.GetPlayerIndex(winner)
	isDealerWin := winner.SeatWind == "East" // TODO: Use gs.PrevalentWind relation? No, player's seat wind matters.

	// 1. Collect Riichi Sticks
	if gs.RiichiSticks > 0 {
		fmt.Printf("%s collects %d Riichi stick points.\n", winner.Name, gs.RiichiSticks*1000)
		winner.Score += gs.RiichiSticks * 1000
		gs.RiichiSticks = 0
	}

	// 2. Transfer Win Points
	if isTsumo {
		totalPayment := 0
		for i, p := range gs.Players {
			if i == winnerIndex {
				continue
			}
			var amount int
			if isDealerWin {
				// Dealer Tsumo: All non-dealers pay equally
				amount = payment.TsumoNonDealer // Use the 'non-dealer pays' value
			} else {
				// Non-dealer Tsumo: Dealer pays more
				if p.SeatWind == "East" { // Is this player the dealer?
					amount = payment.TsumoDealer
				} else {
					amount = payment.TsumoNonDealer
				}
			}
			fmt.Printf("%s pays %d to %s.\n", p.Name, amount, winner.Name)
			p.Score -= amount
			totalPayment += amount
		}
		winner.Score += totalPayment
	} else { // Ron
		if discarder != nil {
			amount := payment.RonValue
			fmt.Printf("%s pays %d to %s.\n", discarder.Name, amount, winner.Name)
			discarder.Score -= amount
			winner.Score += amount
		} else {
			fmt.Println("Error: Discarder was nil during Ron point transfer.")
		}
	}
}
