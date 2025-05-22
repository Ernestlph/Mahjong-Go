package main

import (
	"fmt"
	"math"
	// For Yaku name checks in Ryanhan Shibari
)

// UpdateFuritenStatus checks and updates the player's Furiten state.
// Called typically after the player discards, or when a Ron is declined.
func UpdateFuritenStatus(player *Player, gs *GameState) {
	// Cache current Furiten state to see if it changes
	// oldFuritenState := player.IsFuriten

	// 1. Clear temporary Furiten from a previously declined Ron IF the player has since discarded.
	//    The turn number when Ron was declined is stored in player.DeclinedRonOnTurn.
	//    gs.TurnNumber is the current turn *after* a discard has just been made.
	//    So, if player.DeclinedRonOnTurn is from a *previous* discard cycle, it should clear.
	//    A simple check: if player is the current player and has just discarded.
	if player.DeclinedRonOnTurn != -1 && gs.Players[gs.CurrentPlayerIndex] == player && gs.LastDiscard != nil {
		// If the current player (who just discarded) had a DeclinedRonOnTurn set,
		// that specific temporary Furiten condition is now lifted.
		// We still need to check for other Furiten conditions.
		player.DeclinedRonOnTurn = -1
		player.DeclinedRonTileID = -1
		// gs.AddToGameLog(fmt.Sprintf("Temporary Furiten (missed Ron) cleared for %s after their discard.", player.Name))
	}

	// Reset Furiten status before re-evaluating all conditions.
	player.IsFuriten = false

	// Condition 1: Permanent Riichi Furiten
	// If player is in Riichi and has missed a Ron on any of their declared Riichi waits previously.
	if player.IsRiichi && player.IsPermanentRiichiFuriten {
		player.IsFuriten = true
		// gs.AddToGameLog(fmt.Sprintf("%s is under Permanent Riichi Furiten.", player.Name))
		return // This type of Furiten overrides others.
	}

	// Condition 2: Standard Furiten (Tenpai waits are in own discard pile)
	if IsTenpai(player.Hand, player.Melds) {
		waits := FindTenpaiWaits(player.Hand, player.Melds)
		if len(waits) > 0 {
			// gs.AddToGameLog(fmt.Sprintf("Debug: %s is Tenpai, waits: %v", player.Name, TilesToNames(waits)))
		}
		for _, waitTile := range waits {
			for _, discarded := range player.Discards {
				// Compare type (Suit and Value), ignoring IsRed for Furiten purposes
				if discarded.Suit == waitTile.Suit && discarded.Value == waitTile.Value {
					player.IsFuriten = true
					// gs.AddToGameLog(fmt.Sprintf("%s is Furiten (wait %s found in own discards: %v).", player.Name, waitTile.Name, TilesToNames(player.Discards)))
					return // Found a reason for Furiten
				}
			}
		}
	}

	// Condition 3: Temporary Furiten from a recently declined Ron (if not yet cleared by own discard)
	// This is set directly in DiscardTile when Ron is declined.
	// If player.DeclinedRonOnTurn is still set to the current or immediately preceding opponent's turn,
	// it means the player is still under this temporary Furiten.
	// The logic at the start of this function handles clearing it after the player's *own* subsequent discard.
	if player.DeclinedRonOnTurn != -1 {
		// This check confirms if the declined Ron event is still "active" (i.e., player hasn't discarded since)
		// This means player.IsFuriten would have been set to true when Ron was declined.
		// The purpose of checking it here again is mostly for completeness or if an external event modified IsFuriten.
		player.IsFuriten = true
		// gs.AddToGameLog(fmt.Sprintf("%s is still Furiten (recently declined Ron on tile ID %d on turn %d).", player.Name, player.DeclinedRonTileID, player.DeclinedRonOnTurn))
		return
	}

	// if player.IsFuriten != oldFuritenState { // Log change if any
	// 	gs.AddToGameLog(fmt.Sprintf("%s Furiten status changed to %v.", player.Name, player.IsFuriten))
	// }
}

// Payment represents the points transferred in a win.
type Payment struct {
	Description       string
	RonValue          int // Total points paid by discarder on Ron
	TsumoDealerPay    int // Points paid BY Dealer ON Non-Dealer Tsumo
	TsumoNonDealerPay int // Points paid BY EACH Non-Dealer (for Dealer Tsumo, or to Non-Dealer Tsumo)
}

// CalculatePointPayment calculates point values based on Han, Fu, win conditions, and game state.
func CalculatePointPayment(han, fu int, isWinnerDealer, isTsumo bool, honba, riichiSticks int) Payment {
	// Yakuman has fixed base points, Fu is generally not used for point table lookup.
	// If han indicates Yakuman (e.g., >= 13, or specific Yakuman Yaku identified)
	isYakumanScoreLevel := han >= 13 // Simplified: Kazoe Yakuman and above
	// More robust would be if Yaku identification specifically marked a hand as "Yakuman Type"

	basePoints := 0.0
	limitName := ""

	if isYakumanScoreLevel {
		yakumanMultiplier := han / 13 // 1 for 13-25 Han, 2 for 26-38 Han (Double Yakuman), etc.
		if yakumanMultiplier == 0 {
			yakumanMultiplier = 1
		} // Should not happen if han >= 13
		basePoints = 8000.0 * float64(yakumanMultiplier)
		limitName = fmt.Sprintf("%dx Yakuman", yakumanMultiplier)
	} else {
		// Standard Scoring: Base Points = Fu * 2^(Han + 2)
		// Ensure minimum Fu, except for Chiitoitsu (25 Fu) and Pinfu (20/30 Fu) which are handled by CalculateFu.
		if fu < 20 && fu != 0 {
			fu = 20
		} // Should be rare if CalculateFu is robust
		if fu == 25 && han < 2 { /* Chiitoitsu should have at least 2 han from Yaku struct */
		}

		basePoints = float64(fu) * math.Pow(2, float64(han+2))

		// Apply Score Limits (Mangan, Haneman, Baiman, Sanbaiman)
		// These limits apply if the calculated basePoints EXCEED them, OR if Han count dictates them.
		cappedBasePoints := 0.0
		if han >= 11 {
			cappedBasePoints = 6000.0
			limitName = "Sanbaiman" // 11-12 Han
		} else if han >= 8 {
			cappedBasePoints = 4000.0
			limitName = "Baiman" // 8-10 Han
		} else if han >= 6 {
			cappedBasePoints = 3000.0
			limitName = "Haneman" // 6-7 Han
		} else if han == 5 || (han == 4 && fu >= 40) || (han == 3 && fu >= 70) {
			cappedBasePoints = 2000.0
			limitName = "Mangan"
		}

		if cappedBasePoints > 0 { // A limit applies based on Han/Fu combination
			if basePoints > cappedBasePoints {
				basePoints = cappedBasePoints
			}
			// If basePoints is less but Han dictates a limit (e.g. 5 Han but low Fu), it's still Mangan.
			if limitName != "" && basePoints < cappedBasePoints && (han == 5 || han == 6 || han == 7 || han == 8 || han == 9 || han == 10 || han == 11 || han == 12) {
				basePoints = cappedBasePoints // Mangan by Han count
			}
		} else if basePoints > 2000.0 { // Calculated points exceed Mangan, but doesn't hit higher limits by Han/Fu
			basePoints = 2000.0
			limitName = "Mangan (Capped by points)"
		}
		if limitName == "" { // No specific limit name hit yet
			limitName = fmt.Sprintf("%d Han, %d Fu", han, fu)
		}
	}

	// Calculate actual payment amounts, rounding up to nearest 100
	var ronValue, tsumoDealerPayValue, tsumoNonDealerPayValue int

	if isTsumo {
		if isWinnerDealer { // Dealer Tsumo: each non-dealer pays basePoints * 2
			tsumoNonDealerPayValue = int(math.Ceil((basePoints*2)/100.0)) * 100
			tsumoDealerPayValue = 0 // Dealer doesn't pay self
			limitName += fmt.Sprintf(" (%d All from non-dealers)", tsumoNonDealerPayValue)
		} else { // Non-Dealer Tsumo: dealer pays basePoints * 2, other two non-dealers pay basePoints * 1
			tsumoDealerPayValue = int(math.Ceil((basePoints*2)/100.0)) * 100
			tsumoNonDealerPayValue = int(math.Ceil(basePoints/100.0)) * 100
			limitName += fmt.Sprintf(" (Dealer pays %d, Others pay %d)", tsumoDealerPayValue, tsumoNonDealerPayValue)
		}
		// Add Honba for Tsumo (100 per Honba stick, per player paying)
		tsumoDealerPayValue += honba * 100    // Dealer's share of Honba (if they pay)
		tsumoNonDealerPayValue += honba * 100 // Each non-dealer's share of Honba
	} else { // Ron
		if isWinnerDealer { // Dealer Ron: discarder pays basePoints * 6
			ronValue = int(math.Ceil((basePoints*6)/100.0)) * 100
		} else { // Non-Dealer Ron: discarder pays basePoints * 4
			ronValue = int(math.Ceil((basePoints*4)/100.0)) * 100
		}
		// Add Honba for Ron (300 per Honba stick, paid by discarder)
		ronValue += honba * 300
		limitName += fmt.Sprintf(" (%d from discarder)", ronValue)
	}

	return Payment{
		Description:       limitName,
		RonValue:          ronValue,
		TsumoDealerPay:    tsumoDealerPayValue,
		TsumoNonDealerPay: tsumoNonDealerPayValue,
	}
}

// TransferPoints updates player scores based on the payment structure.
// discarder is only relevant for Ron.
func TransferPoints(gs *GameState, winner, discarder *Player, isTsumo bool, payment Payment) {
	winnerIndex := gs.GetPlayerIndex(winner)
	isWinnerDealer := (gs.Players[gs.DealerIndexThisRound] == winner)

	// 1. Collect Riichi Sticks (winner gets all Riichi sticks on the table)
	if gs.RiichiSticks > 0 {
		riichiPoints := gs.RiichiSticks * RiichiBet
		gs.AddToGameLog(fmt.Sprintf("%s collects %d Riichi stick points.", winner.Name, riichiPoints))
		winner.Score += riichiPoints
		gs.RiichiSticks = 0
	}

	// 2. Transfer Win Points (Main payment from Yaku/Fu/Han)
	if isTsumo {
		totalPaymentReceivedByWinner := 0
		for i, p := range gs.Players {
			if i == winnerIndex {
				continue
			} // Winner doesn't pay self

			var amountToPay int
			if isWinnerDealer { // Winner is Dealer, all non-dealers pay TsumoNonDealerPay
				amountToPay = payment.TsumoNonDealerPay
			} else { // Winner is Non-Dealer
				if gs.Players[gs.DealerIndexThisRound] == p { // This payer 'p' is the dealer
					amountToPay = payment.TsumoDealerPay
				} else { // This payer 'p' is another non-dealer
					amountToPay = payment.TsumoNonDealerPay
				}
			}
			gs.AddToGameLog(fmt.Sprintf("%s (P%d) pays %d to %s (P%d) for Tsumo.",
				p.Name, i+1, amountToPay, winner.Name, winnerIndex+1))
			p.Score -= amountToPay
			totalPaymentReceivedByWinner += amountToPay
			if p.Score < 0 {
				CheckAndHandleBust(gs, p, winner)
			}
		}
		winner.Score += totalPaymentReceivedByWinner
	} else { // Ron
		if discarder != nil {
			amountToPay := payment.RonValue
			gs.AddToGameLog(fmt.Sprintf("%s (P%d, discarder) pays %d to %s (P%d) for Ron.",
				discarder.Name, gs.GetPlayerIndex(discarder)+1, amountToPay, winner.Name, winnerIndex+1))
			discarder.Score -= amountToPay
			winner.Score += amountToPay
			if discarder.Score < 0 {
				CheckAndHandleBust(gs, discarder, winner)
			}
		} else {
			gs.AddToGameLog(fmt.Sprintf("Error: Discarder was nil during Ron point transfer to %s.", winner.Name))
		}
	}
}

// CheckAndHandleBust handles player busting (score < 0).
func CheckAndHandleBust(gs *GameState, bustedPlayer *Player, winner *Player) {
	gs.AddToGameLog(fmt.Sprintf("!!! Player %s (P%d) has busted (score: %d) !!!",
		bustedPlayer.Name, gs.GetPlayerIndex(bustedPlayer)+1, bustedPlayer.Score))
	// Basic bust handling: Game ends immediately.
	// TODO: Implement Tobu/Dobon scoring (winner might get remaining points or a bonus from busting player).
	// For now, just flag the game to end. The main loop will catch this.
	gs.GamePhase = PhaseGameEnd
}

// HandleNotenBappu processes point transfers for Ryuukyoku (exhaustive draw with no winner).
func HandleNotenBappu(gs *GameState) {
	tenpaiPlayers := []*Player{}
	notenPlayers := []*Player{}
	for _, p := range gs.Players {
		// p.IsTenpai should have been set in main.go before calling this
		if p.IsTenpai {
			tenpaiPlayers = append(tenpaiPlayers, p)
		} else {
			notenPlayers = append(notenPlayers, p)
		}
	}

	numTenpai := len(tenpaiPlayers)
	numNoten := len(notenPlayers)

	gs.AddToGameLog(fmt.Sprintf("Ryuukyoku: Tenpai players: %d, Noten players: %d", numTenpai, numNoten))

	if numTenpai == 0 || numTenpai == 4 { // All noten or all tenpai
		gs.AddToGameLog("No Noten Bappu payment (all players are Tenpai or all are Noten).")
		return // No payment in these cases
	}

	// Points are exchanged based on NotenBappuTotal (usually 3000 points).
	// Noten players pay, Tenpai players receive.
	switch numTenpai {
	case 1: // 1 Tenpai, 3 Noten
		for _, notenP := range notenPlayers { // Each of 3 Noten players pays 1000
			notenP.Score -= NotenBappuPayment1T3N
			tenpaiPlayers[0].Score += NotenBappuPayment1T3N
			gs.AddToGameLog(fmt.Sprintf("%s (Noten) pays %d to %s (Tenpai). Scores: %s=%d, %s=%d",
				notenP.Name, NotenBappuPayment1T3N, tenpaiPlayers[0].Name,
				notenP.Name, notenP.Score, tenpaiPlayers[0].Name, tenpaiPlayers[0].Score))
		}
	case 2: // 2 Tenpai, 2 Noten
		for _, notenP := range notenPlayers { // Each of 2 Noten players pays 1500
			notenP.Score -= NotenBappuPayment2T2N
			gs.AddToGameLog(fmt.Sprintf("%s (Noten) pays %d (to be split by Tenpai players). Score: %s=%d",
				notenP.Name, NotenBappuPayment2T2N, notenP.Name, notenP.Score))
		}
		for _, tenpaiP := range tenpaiPlayers { // Each of 2 Tenpai players receives 1500
			tenpaiP.Score += NotenBappuPayment2T2N
			gs.AddToGameLog(fmt.Sprintf("%s (Tenpai) receives %d. Score: %s=%d",
				tenpaiP.Name, NotenBappuPayment2T2N, tenpaiP.Name, tenpaiP.Score))
		}
	case 3: // 3 Tenpai, 1 Noten
		for _, tenpaiP := range tenpaiPlayers { // The 1 Noten player pays 1000 to each of 3 Tenpai players
			tenpaiP.Score += NotenBappuPayment3T1N / 3 // Each Tenpai gets 1000
			notenPlayers[0].Score -= NotenBappuPayment3T1N / 3
			gs.AddToGameLog(fmt.Sprintf("%s (Noten) pays %d to %s (Tenpai). Scores: %s=%d, %s=%d",
				notenPlayers[0].Name, NotenBappuPayment3T1N/3, tenpaiP.Name,
				notenPlayers[0].Name, notenPlayers[0].Score, tenpaiP.Name, tenpaiP.Score))
		}
	}
}
