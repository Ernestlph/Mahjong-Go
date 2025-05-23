package main

import (
	"fmt"
	"sort"
	"strings"
)

// Helper function to remove the last discard from a specific player's list after a call.
// Assumes gs.LastDiscard holds the tile that was called.
func removeLastDiscardFromPlayer(gs *GameState, playerIndexWhoDiscarded int) {
	if playerIndexWhoDiscarded < 0 || playerIndexWhoDiscarded >= len(gs.Players) || gs.LastDiscard == nil {
		gs.AddToGameLog(fmt.Sprintf("Warning: removeLastDiscardFromPlayer with invalid index %d or nil LastDiscard.", playerIndexWhoDiscarded))
		// fmt.Println("Warning: Attempted to remove last discard with invalid index or nil LastDiscard.")
		return
	}
	player := gs.Players[playerIndexWhoDiscarded]
	if len(player.Discards) > 0 {
		// Check if the last element in the discarder's pile matches gs.LastDiscard
		lastIndex := len(player.Discards) - 1
		if player.Discards[lastIndex].ID == gs.LastDiscard.ID {
			player.Discards = player.Discards[:lastIndex]
			// gs.AddToGameLog(fmt.Sprintf("Debug: Removed %s from P%d's discards after call.", gs.LastDiscard.Name, playerIndexWhoDiscarded+1))
		} else {
			// Fallback/Warning: The last element didn't match. This could happen if multiple events occurred rapidly
			// or if gs.LastDiscard was somehow updated before this ran for the correct discard.
			// For now, log it. A more robust system might search the last few discards if performance isn't an issue.
			gs.AddToGameLog(fmt.Sprintf("Warning: Last discard mismatch when removing called tile %s for P%d. Discards: %v",
				gs.LastDiscard.Name, playerIndexWhoDiscarded+1, TilesToNames(player.Discards)))
			// fmt.Printf("Warning: Last discard mismatch when removing called tile %s for P%d. Discards: %v\n",
			// 	gs.LastDiscard.Name, playerIndexWhoDiscarded+1, TilesToNames(player.Discards))
		}
	} else {
		gs.AddToGameLog(fmt.Sprintf("Warning: Attempted to remove last discard from P%d, but their discard list is empty.", playerIndexWhoDiscarded+1))
		// fmt.Printf("Warning: Attempted to remove last discard from P%d, but their discard list is empty.\n", playerIndexWhoDiscarded+1)
	}
}

// DiscardTile handles the current player discarding a tile.
// It checks for calls (Ron, Kan, Pon, Chi) from other players and handles turn progression.
// Returns the discarded tile and if the game/round ended (e.g., Ron).
func DiscardTile(gs *GameState, player *Player, tileIndex int) (Tile, bool) {
	if tileIndex < 0 || tileIndex >= len(player.Hand) {
		gs.AddToGameLog(fmt.Sprintf("Error: Invalid tile index %d to discard for %s.", tileIndex, player.Name))
		// fmt.Println("Error: Invalid tile index to discard.")
		return Tile{}, false // Indicate error without ending game yet
	}

	// --- Riichi Discard Restrictions ---
	if player.IsRiichi && player.JustDrawnTile != nil && len(player.Hand) == HandSize+1 {
		// Player in Riichi must discard the tile they just drew, unless they Kan it (Kan handled before DiscardTile).
		drawnTileID := player.JustDrawnTile.ID
		chosenDiscardID := player.Hand[tileIndex].ID
		if chosenDiscardID != drawnTileID {
			gs.AddToGameLog(fmt.Sprintf("Warning: %s (Riichi) tried to discard %s, but must discard drawn %s. Forcing correct discard.",
				player.Name, player.Hand[tileIndex].Name, player.JustDrawnTile.Name))
			// fmt.Printf("Warning: %s (Riichi) tried to discard %s, but must discard drawn %s. Forcing correct discard.\n",
			// 	player.Name, player.Hand[tileIndex].Name, player.JustDrawnTile.Name)
			actualDrawnTileIndex := -1
			for i, t := range player.Hand {
				if t.ID == drawnTileID {
					actualDrawnTileIndex = i
					break
				}
			}
			if actualDrawnTileIndex != -1 {
				tileIndex = actualDrawnTileIndex // Correct the tileIndex
			} else {
				// This is a critical error state if the drawn tile is not found in hand.
				gs.AddToGameLog(fmt.Sprintf("CRITICAL ERROR: Could not find drawn tile %s in %s's hand for Riichi discard.",
					player.JustDrawnTile.Name, player.Name))
				// fmt.Printf("CRITICAL ERROR: Could not find drawn tile %s in %s's hand for Riichi discard.\n",
				// 	player.JustDrawnTile.Name, player.Name)
				// As a fallback, proceed with the user's chosen index, but log this severe issue.
			}
		}
	}
	discardedTile := player.Hand[tileIndex]

	// --- Perform Discard ---
	player.Hand = append(player.Hand[:tileIndex], player.Hand[tileIndex+1:]...)
	player.Discards = append(player.Discards, discardedTile)
	gs.LastDiscard = &discardedTile
	playerDiscarderIndex := gs.CurrentPlayerIndex // Store index of player who is discarding
	gs.TurnNumber++                               // Increment turn number *within the round*

	player.JustDrawnTile = nil // Clear the "just drawn" status after discard decision is locked in

	// Ssuufon Renda Check: Record first un-interrupted discard for each player by their initial seat order
	if gs.IsFirstGoAround && !gs.AnyCallMadeThisRound && !player.HasMadeFirstDiscardThisRound {
		initialSeatOrder := player.InitialTurnOrder // This should be 0 for East, 1 for South, etc. at game start.
		if initialSeatOrder >= 0 && initialSeatOrder < 4 {
			gs.FirstTurnDiscards[initialSeatOrder] = discardedTile
			gs.FirstTurnDiscardCount++
			if gs.FirstTurnDiscardCount == 4 { // All four players made their first un-interrupted discard
				if CheckSsuufonRenda(gs) {
					gs.AddToGameLog("Ssuufon Renda! Round ends in an abortive draw.")
					// fmt.Println("Ssuufon Renda! Round ends in an abortive draw.")
					gs.GamePhase = PhaseRoundEnd
					gs.RoundWinner = nil       // Mark as draw
					return discardedTile, true // Game/round ends
				}
			}
		}
	}
	player.HasMadeFirstDiscardThisRound = true

	UpdateFuritenStatus(player, gs) // Update based on *own* discard

	gs.AddToGameLog(fmt.Sprintf("%s (P%d) discards: %s (Turn %d)", player.Name, playerDiscarderIndex+1, discardedTile.Name, gs.TurnNumber))
	// fmt.Printf("%s discards: %s\n", player.Name, discardedTile.Name)

	// --- Check for Calls ---
	gs.SanchahouRonners = []*Player{} // Reset for this specific discard
	potentialRonCallers := []struct {
		*Player
		int
	}{} // Player and their index
	potentialKanCallers := []struct {
		*Player
		int
	}{} // Player and their index (for Daiminkan)
	potentialPonCallers := []struct {
		*Player
		int
	}{} // Player and their index
	var chiCaller *Player
	var chiCallerIndex int = -1

	// Iterate through other players in turn order starting from the player to the discarder's left
	for i := 1; i < len(gs.Players); i++ {
		otherPlayerIndex := (playerDiscarderIndex + i) % len(gs.Players)
		otherPlayer := gs.Players[otherPlayerIndex]

		// Check Ron
		if CanDeclareRon(otherPlayer, discardedTile, gs) {
			potentialRonCallers = append(potentialRonCallers, struct {
				*Player
				int
			}{otherPlayer, otherPlayerIndex})
		}
		// Players in Riichi cannot make open calls (Pon, Chi, Daiminkan)
		if otherPlayer.IsRiichi {
			continue
		}
		// Check Daiminkan (Open Kan)
		if CanDeclareDaiminkan(otherPlayer, discardedTile) {
			potentialKanCallers = append(potentialKanCallers, struct {
				*Player
				int
			}{otherPlayer, otherPlayerIndex})
		}
		// Check Pon
		if CanDeclarePon(otherPlayer, discardedTile) {
			potentialPonCallers = append(potentialPonCallers, struct {
				*Player
				int
			}{otherPlayer, otherPlayerIndex})
		}
		// Check Chi (Only player to the left of the discarder)
		if otherPlayerIndex == (playerDiscarderIndex+1)%len(gs.Players) {
			if CanDeclareChi(otherPlayer, discardedTile) {
				chiCaller = otherPlayer
				chiCallerIndex = otherPlayerIndex
			}
		}
	}

	// --- Handle Calls based on Priority: Ron > Kan/Pon > Chi ---
	callMade := false

	// 1. Ron
	if len(potentialRonCallers) > 0 {
		// Sanchahou Check (Three players Ron)
		if len(potentialRonCallers) >= 3 {
			gs.SanchahouRonners = make([]*Player, len(potentialRonCallers))
			for i, prc := range potentialRonCallers {
				gs.SanchahouRonners[i] = prc.Player
			}

			if CheckSanchahou(gs) { // CheckSanchahou confirms if all 3 will actually Ron
				// Simulate choice for Sanchahou decision
				actualRonners := 0
				for _, prc := range potentialRonCallers {
					isHumanRonner := gs.GetPlayerIndex(prc.Player) == 0
					ronConfirmChoice := true // AI accepts
					if isHumanRonner {
						fmt.Printf("--- Player %s (%s) Opportunity (Sanchahou context) ---\n", prc.Player.Name, "Ron")
						ronConfirmChoice = GetPlayerChoice(gs.InputReader, fmt.Sprintf("%s, declare RON on %s? (y/n): ", prc.Player.Name, discardedTile.Name))
					}
					if ronConfirmChoice {
						actualRonners++
					} else {
						gs.AddToGameLog(fmt.Sprintf("%s declined Ron in Sanchahou context.", prc.Player.Name))
					}
				}
				if actualRonners >= 3 {
					gs.AddToGameLog("Sanchahou! Round ends in an abortive draw as >=3 players confirmed Ron.")
					// fmt.Println("Sanchahou! Round ends in an abortive draw.")
					gs.GamePhase = PhaseRoundEnd
					gs.RoundWinner = nil
					return discardedTile, true
				}
				// If fewer than 3 confirmed, proceed with Atamahane below.
				// Update potentialRonCallers to only those who confirmed.
				confirmedRonCallers := []struct {
					*Player
					int
				}{}
				// Re-prompt for confirmation, this time with Atamahane in mind.
				// This part is complex. For now, if Sanchahou condition met but not all confirm,
				// it might fall through to Atamahane with fewer players.
			}
		}

		// Atamahane: Closest player in turn order wins. potentialRonCallers is already sorted by turn order.
		winnerInfo := potentialRonCallers[0]
		winner, winnerIndex := winnerInfo.Player, winnerInfo.int

		isHumanWinner := winnerIndex == 0 // Assuming player 0 is human
		gs.AddToGameLog(fmt.Sprintf("%s (P%d) has Ron opportunity on %s.", winner.Name, winnerIndex+1, discardedTile.Name))
		// fmt.Printf("--- Player %s (%s) Opportunity ---\n", winner.Name, "Ron")

		ronConfirm := true // Default AI to accept
		if isHumanWinner {
			ronConfirm = GetPlayerChoice(gs.InputReader, fmt.Sprintf("%s, declare RON on %s? (y/n): ", winner.Name, discardedTile.Name))
		} else {
			// fmt.Printf("(%s can Ron... AI Accepts)\n", winner.Name)
		}

		if ronConfirm {
			gs.AddToGameLog(fmt.Sprintf("!!! RON by %s (P%d) on %s from %s (P%d) !!!",
				winner.Name, winnerIndex+1, discardedTile.Name, player.Name, playerDiscarderIndex+1))
			// fmt.Printf("\n!!! RON by %s on %s !!!\n", winner.Name, discardedTile.Name)
			for _, p_ := range gs.Players {
				p_.IsIppatsu = false
			} // Ron breaks Ippatsu

			// For Ron, CurrentPlayerIndex should be the DISCARDER when HandleWin is called.
			// gs.CurrentPlayerIndex is currently playerDiscarderIndex.
			HandleWin(gs, winner, discardedTile, false) // Sets GamePhase to RoundEnd
			return discardedTile, true                  // Game/round ends
		} else { // Declined Ron
			gs.AddToGameLog(fmt.Sprintf("%s declined Ron on %s.", winner.Name, discardedTile.Name))
			// fmt.Printf("%s declined Ron.\n", winner.Name)
			winner.IsFuriten = true                  // Temporary Furiten for missing Ron
			winner.DeclinedRonOnTurn = gs.TurnNumber // Record turn of declined Ron
			winner.DeclinedRonTileID = discardedTile.ID
			if winner.IsRiichi { // If Riichi player misses Ron on a wait tile
				for _, wait := range winner.RiichiDeclaredWaits {
					if wait.Suit == discardedTile.Suit && wait.Value == discardedTile.Value {
						winner.IsPermanentRiichiFuriten = true
						gs.AddToGameLog(fmt.Sprintf("%s is now in Permanent Riichi Furiten for missing wait %s.", winner.Name, wait.Name))
						break
					}
				}
			}
		}
	}

	// 2. Kan / Pon (Kan has priority over Pon if from same player or closer player)
	// If multiple Kan/Pon callers, player closest to discarder in turn order gets priority.
	// If Kan and Pon are possible from different players, Kan takes precedence if its caller is same or closer.
	var callToProcess struct {
		*Player
		int
		string
	} // Player, Index, Type ("Kan" or "Pon")
	processCall := false

	if len(potentialKanCallers) > 0 {
		callToProcess = struct {
			*Player
			int
			string
		}{potentialKanCallers[0].Player, potentialKanCallers[0].int, "Kan"}
		processCall = true
	}
	if len(potentialPonCallers) > 0 {
		if !processCall || // No Kan call pending
			(potentialPonCallers[0].int == callToProcess.int && callToProcess.string != "Kan") || // Same player, Pon is somehow listed first (shouldn't happen if Kan checked first)
			(potentialPonCallers[0].int != callToProcess.int && isCloser(potentialPonCallers[0].int, callToProcess.int, playerDiscarderIndex, len(gs.Players))) { // Pon caller is closer
			callToProcess = struct {
				*Player
				int
				string
			}{potentialPonCallers[0].Player, potentialPonCallers[0].int, "Pon"}
			processCall = true
		}
	}

	if processCall && !callMade { // Ensure Ron wasn't made
		caller, callerIndex, callType := callToProcess.Player, callToProcess.int, callToProcess.string
		isHumanCaller := callerIndex == 0
		gs.AddToGameLog(fmt.Sprintf("%s (P%d) has %s opportunity on %s.", caller.Name, callerIndex+1, callType, discardedTile.Name))
		// fmt.Printf("--- Player %s (%s) Opportunity ---\n", caller.Name, callType)

		confirmCall := true // AI default
		if isHumanCaller {
			DisplayPlayerState(caller)
			confirmCall = GetPlayerChoice(gs.InputReader, fmt.Sprintf("%s, declare %s on %s? (y/n): ", caller.Name, strings.ToUpper(callType), discardedTile.Name))
		} else { /* AI logic for call decision */
		}

		if confirmCall {
			callMade = true
			for _, p_ := range gs.Players {
				p_.IsIppatsu = false
			} // Any call breaks Ippatsu

			removeLastDiscardFromPlayer(gs, playerDiscarderIndex) // Remove from original discarder's pile
			gs.CurrentPlayerIndex = callerIndex                   // Turn shifts to caller

			if callType == "Kan" { // Daiminkan
				HandleKanAction(gs, caller, discardedTile, "Daiminkan")
				// HandleKanAction calls PromptDiscard, game continues with Kan caller
			} else { // Pon
				HandlePonAction(gs, caller, discardedTile, playerDiscarderIndex) // Pass original discarder index
				PromptDiscard(gs, caller)                                        // Caller must discard next
			}
			return discardedTile, false // Game continues
		} else {
			gs.AddToGameLog(fmt.Sprintf("%s declined %s.", caller.Name, callType))
		}
	}

	// 3. Chi (only if no higher priority call made, and only by player to the left)
	if !callMade && chiCaller != nil {
		isHumanChiCaller := chiCallerIndex == 0
		gs.AddToGameLog(fmt.Sprintf("%s (P%d) has Chi opportunity on %s.", chiCaller.Name, chiCallerIndex+1, discardedTile.Name))
		// fmt.Printf("--- Player %s (%s) Opportunity ---\n", chiCaller.Name, "Chi")

		chiConfirmedAndHandled := false
		if isHumanChiCaller {
			DisplayPlayerState(chiCaller)
			choiceNum, sequence := GetChiChoice(gs, chiCaller, discardedTile)
			if choiceNum > 0 {
				callMade = true
				chiConfirmedAndHandled = true
				for _, p_ := range gs.Players {
					p_.IsIppatsu = false
				}
				removeLastDiscardFromPlayer(gs, playerDiscarderIndex)
				gs.CurrentPlayerIndex = chiCallerIndex
				HandleChiAction(gs, chiCaller, discardedTile, sequence, playerDiscarderIndex) // Pass original discarder index
				PromptDiscard(gs, chiCaller)
			} else {
				gs.AddToGameLog(fmt.Sprintf("%s declined Chi.", chiCaller.Name))
				// fmt.Printf("%s declined Chi.\n", chiCaller.Name)
			}
		} else { // AI Chi logic
			// Basic AI: Chi if it seems beneficial. For now, AI declines Chi.
			// gs.AddToGameLog(fmt.Sprintf("AI %s declines Chi on %s.", chiCaller.Name, discardedTile.Name))
			// fmt.Printf("(%s can declare Chi... AI declines for now)\n", chiCaller.Name)
		}
		if chiConfirmedAndHandled {
			return discardedTile, false // Game continues, Chi caller discards
		}
	}

	// --- No Calls Made or All Calls Declined ---
	if !callMade {
		if player.IsRiichi && player.IsIppatsu { // If Riichi player's discard wasn't Ronned/called
			player.IsIppatsu = false
			gs.AddToGameLog(fmt.Sprintf("Ippatsu broken for %s (no call on Riichi discard).", player.Name))
		}
		gs.NextPlayer() // Proceed to next player only if no call interrupted the flow
	}
	return discardedTile, false // Game continues
}

// isCloser determines if newCallerIdx is closer to discarderIdx than oldCallerIdx.
// Helper for call priority. Assumes clockwise turn order.
func isCloser(newCallerIdx, oldCallerIdx, discarderIdx, numPlayers int) bool {
	distNew := (newCallerIdx - discarderIdx + numPlayers) % numPlayers
	distOld := (oldCallerIdx - discarderIdx + numPlayers) % numPlayers
	return distNew < distOld
}

// HandlePonAction processes the Pon action, updates player state.
// discarderPlayerIndex is the index of the player who made the discard being Ponned.
func HandlePonAction(gs *GameState, player *Player, discardedTile Tile, discarderPlayerIndex int) {
	gs.AnyCallMadeThisRound = true
	gs.IsFirstGoAround = false
	gs.AddToGameLog(fmt.Sprintf("%s (P%d) PONS %s from P%d.",
		player.Name, gs.GetPlayerIndex(player)+1, discardedTile.Name, discarderPlayerIndex+1))
	// fmt.Printf("\n%s PONS %s from P%d!\n", player.Name, discardedTile.Name, discarderPlayerIndex+1)

	meldTiles := []Tile{discardedTile}
	indicesToRemove := []int{}
	foundCount := 0

	for i := len(player.Hand) - 1; i >= 0; i-- {
		tile := player.Hand[i]
		if tile.Suit == discardedTile.Suit && tile.Value == discardedTile.Value {
			meldTiles = append(meldTiles, tile)
			indicesToRemove = append(indicesToRemove, i)
			foundCount++
			if foundCount == 2 {
				break
			}
		}
	}

	if foundCount == 2 {
		player.Hand = RemoveTilesByIndices(player.Hand, indicesToRemove)
		sort.Sort(BySuitValue(meldTiles))
		newMeld := Meld{
			Type:        "Pon",
			Tiles:       meldTiles,
			CalledOn:    discardedTile,
			FromPlayer:  discarderPlayerIndex,
			IsConcealed: false,
		}
		player.Melds = append(player.Melds, newMeld)
		if discarderPlayerIndex >= 0 && discarderPlayerIndex < len(gs.Players) {
			gs.Players[discarderPlayerIndex].HasHadDiscardCalledThisRound = true
		}
		sort.Sort(BySuitValue(player.Hand))
	} else {
		gs.AddToGameLog(fmt.Sprintf("Error: %s Pon failed, couldn't find 2 tiles for %s.", player.Name, discardedTile.Name))
		// fmt.Println("Error: Could not find 2 tiles for Pon in player's hand.")
	}
}

// HandleChiAction processes the Chi action, updates player state.
// discarderPlayerIndex is the index of the player who made the discard being Chi'd (always player to caller's right).
func HandleChiAction(gs *GameState, player *Player, discardedTile Tile, sequence []Tile, discarderPlayerIndex int) {
	gs.AnyCallMadeThisRound = true
	gs.IsFirstGoAround = false
	gs.AddToGameLog(fmt.Sprintf("%s (P%d) CHIS %s from P%d (using %s, %s).",
		player.Name, gs.GetPlayerIndex(player)+1, discardedTile.Name, discarderPlayerIndex+1,
		sequence[0].Name, sequence[1].Name)) // Sequence already has 3 tiles.
	// fmt.Printf("\n%s CHIS %s from P%d (using ...)\n", player.Name, discardedTile.Name, discarderPlayerIndex+1)

	indicesToRemove := []int{}
	foundCount := 0
	tilesFromHandForMeld := []Tile{} // Tiles from hand that will form part of the meld

	for _, seqTile := range sequence {
		if seqTile.ID == discardedTile.ID {
			continue // Skip the called tile itself, we need to find the other two in hand
		}
		foundInHand := false
		for i := len(player.Hand) - 1; i >= 0; i-- {
			handTile := player.Hand[i]
			if handTile.ID == seqTile.ID { // Match specific tile ID
				isAlreadyMarked := false
				for _, idx := range indicesToRemove {
					if idx == i {
						isAlreadyMarked = true
						break
					}
				}
				if !isAlreadyMarked {
					indicesToRemove = append(indicesToRemove, i)
					tilesFromHandForMeld = append(tilesFromHandForMeld, handTile)
					foundCount++
					foundInHand = true
					break // Found this specific tile from sequence in hand
				}
			}
		}
		if !foundInHand {
			gs.AddToGameLog(fmt.Sprintf("Error: Chi failed for %s, could not find required tile %s in hand.", player.Name, seqTile.Name))
			// fmt.Printf("Error: Could not find required tile %s for Chi in hand.\n", seqTile.Name)
			return
		}
	}

	if foundCount == 2 {
		player.Hand = RemoveTilesByIndices(player.Hand, indicesToRemove)
		// sequence already contains discardedTile and the two from hand, and is sorted by GetChiChoice
		newMeld := Meld{
			Type:        "Chi",
			Tiles:       sequence, // sequence is the 3-tile meld
			CalledOn:    discardedTile,
			FromPlayer:  discarderPlayerIndex,
			IsConcealed: false,
		}
		player.Melds = append(player.Melds, newMeld)
		if discarderPlayerIndex >= 0 && discarderPlayerIndex < len(gs.Players) {
			gs.Players[discarderPlayerIndex].HasHadDiscardCalledThisRound = true
		}
		sort.Sort(BySuitValue(player.Hand))
	} else {
		gs.AddToGameLog(fmt.Sprintf("Error: Chi for %s found %d hand tiles, expected 2.", player.Name, foundCount))
		// fmt.Printf("Error: Found %d hand tiles for Chi, expected 2.\n", foundCount)
	}
}

// HandleKanAction processes Kan declarations (all types).
func HandleKanAction(gs *GameState, player *Player, targetTile Tile, kanType string) {
	gs.AnyCallMadeThisRound = true
	gs.IsFirstGoAround = false
	gs.TotalKansDeclaredThisRound++ // Increment global Kan counter for Suukaikan
	gs.AddToGameLog(fmt.Sprintf("%s (P%d) declares %s with %s. Total Kans this round: %d",
		player.Name, gs.GetPlayerIndex(player)+1, kanType, targetTile.Name, gs.TotalKansDeclaredThisRound))
	// fmt.Printf("\n%s declares %s with %s!\n", player.Name, kanType, targetTile.Name)

	// Suukaikan abortive draw check (pre-emptive based on number of kans)
	// Note: CheckSuukaikan returns true if conditions for *potential* abort are met.
	// The actual abort happens if Rinshan draw fails.
	if CheckSuukaikan(gs) {
		gs.AddToGameLog("Condition for Suukaikan (4+ Kans by multiple players) is met. Abort if Rinshan draw fails.")
	}

	meldTiles := []Tile{}
	indicesToRemove := []int{}
	newMeld := Meld{Type: kanType}
	success := false
	rinshanRequired := true
	originalDiscarderIndex := -1 // For Daiminkan Pao source

	// Break Ippatsu for all other players (or self if not Ankan maintaining Ippatsu)
	// Ankan by Riichi player usually doesn't break their *own* Ippatsu.
	// Shouminkan by Riichi player usually *does* break their Ippatsu if waits change (or always by some rules).
	// Daiminkan (open call) always breaks Ippatsu.
	if kanType == "Ankan" && player.IsRiichi && player.IsIppatsu {
		// Ankan by Riichi player: their Ippatsu potentially maintained.
		// Check if this Ankan changes waits. If so, Ippatsu might break by some rules, or Riichi itself invalid.
		// For simplicity, let's assume Ankan by Riichi player doesn't break their Ippatsu here.
		// Other players' Ippatsu would still be broken by the game interruption.
		for _, p := range gs.Players {
			if p != player && p.IsIppatsu {
				p.IsIppatsu = false
				gs.AddToGameLog(fmt.Sprintf("Ippatsu broken for %s due to %s's Ankan.", p.Name, player.Name))
			}
		}
	} else { // Daiminkan, Shouminkan, or Ankan by non-Riichi/non-Ippatsu player
		for _, p := range gs.Players {
			if p.IsIppatsu {
				p.IsIppatsu = false
				gs.AddToGameLog(fmt.Sprintf("Ippatsu broken for %s due to %s's %s.", p.Name, player.Name, kanType))
			}
		}
	}

	switch kanType {
	case "Ankan":
		if player.IsRiichi && checkWaitChangeForRiichiKan(player, gs, targetTile, "Ankan") {
			gs.AddToGameLog(fmt.Sprintf("%s Ankan on %s aborted: would change Riichi waits.", player.Name, targetTile.Name))
			// fmt.Println("Cannot declare Ankan: it would change your Riichi waits.")
			gs.TotalKansDeclaredThisRound-- // Revert count
			PromptDiscard(gs, player)       // Player must discard something else (original drawn tile if that was the case)
			return
		}
		count := 0
		for i := len(player.Hand) - 1; i >= 0; i-- {
			if player.Hand[i].Suit == targetTile.Suit && player.Hand[i].Value == targetTile.Value {
				meldTiles = append(meldTiles, player.Hand[i])
				indicesToRemove = append(indicesToRemove, i)
				count++
			}
		}
		if count == 4 {
			newMeld.Tiles = meldTiles
			newMeld.IsConcealed = true
			newMeld.FromPlayer = -1
			newMeld.CalledOn = Tile{}
			player.Melds = append(player.Melds, newMeld)
			player.Hand = RemoveTilesByIndices(player.Hand, indicesToRemove)
			success = true
		} else {
			gs.AddToGameLog(fmt.Sprintf("Error: Ankan %s failed for %s, count %d.", targetTile.Name, player.Name, count))
			// fmt.Printf("Error: Ankan declared but found %d matching tiles for %s in hand.\n", count, targetTile.Name)
		}

	case "Daiminkan":
		// Determine original discarder (player whose discard is being Kanned)
		// playerDiscarderIndex was passed to DiscardTile, and gs.CurrentPlayerIndex was that player.
		// Now, gs.CurrentPlayerIndex is the KANNER.
		// This needs to be robust. Assume for Daiminkan, gs.LastDiscard was set by player N,
		// and player (Kanner) is calling on it.
		// Let's assume the caller of HandleKanAction (DiscardTile) set originalDiscarderIndex correctly.
		// For Daiminkan, originalDiscarderIndex is the player whose turn it was when `targetTile` (gs.LastDiscard) was discarded.
		// This would be the player *before* the current player (Kanner) in the turn order if no other calls intervened.
		// This is complex. For now, assume `targetTile.FromPlayer` (if we add it) or passed index is correct.
		// Placeholder:
		originalDiscarderIndex = (gs.GetPlayerIndex(player) - 1 + len(gs.Players)) % len(gs.Players) // This is often wrong for Daiminkan.
		// Search for the player who actually discarded `targetTile` if gs.LastDiscard is reliable.
		// This is needed for Pao.
		// For now, this is a known weak point if not passed explicitly.

		meldTiles = append(meldTiles, targetTile)
		count := 0
		for i := len(player.Hand) - 1; i >= 0; i-- {
			if player.Hand[i].Suit == targetTile.Suit && player.Hand[i].Value == targetTile.Value {
				meldTiles = append(meldTiles, player.Hand[i])
				indicesToRemove = append(indicesToRemove, i)
				count++
				if count == 3 {
					break
				}
			}
		}
		if count == 3 {
			newMeld.Tiles = meldTiles
			newMeld.CalledOn = targetTile
			newMeld.FromPlayer = originalDiscarderIndex
			newMeld.IsConcealed = false
			player.Melds = append(player.Melds, newMeld)
			player.Hand = RemoveTilesByIndices(player.Hand, indicesToRemove)
			if originalDiscarderIndex >= 0 && originalDiscarderIndex < len(gs.Players) {
				gs.Players[originalDiscarderIndex].HasHadDiscardCalledThisRound = true
				// Pao check for Daisangen/Daisuushii if this Daiminkan enables it
				if isPaoConditionMetByKan(player, targetTile, "Daiminkan", gs.Players[originalDiscarderIndex]) {
					player.PaoSourcePlayerIndex = originalDiscarderIndex // Kanner is target, discarder is source
					gs.AddToGameLog(fmt.Sprintf("PAO Triggered: %s's discard of %s for Daiminkan by %s may lead to Pao.",
						gs.Players[originalDiscarderIndex].Name, targetTile.Name, player.Name))
				}
			}
			success = true
		} else {
			gs.AddToGameLog(fmt.Sprintf("Error: Daiminkan %s failed for %s, count %d.", targetTile.Name, player.Name, count))
			// fmt.Printf("Error: Daiminkan declared but found %d matching tiles for %s in hand.\n", count, targetTile.Name)
		}

	case "Shouminkan":
		if player.IsRiichi && checkWaitChangeForRiichiKan(player, gs, targetTile, "Shouminkan") {
			gs.AddToGameLog(fmt.Sprintf("%s Shouminkan on %s aborted: would change Riichi waits.", player.Name, targetTile.Name))
			// fmt.Println("Cannot declare Shouminkan: it would change your Riichi waits.")
			gs.TotalKansDeclaredThisRound--
			PromptDiscard(gs, player)
			return
		}

		foundPonIndex := -1
		var originalPon Meld
		for i, meld := range player.Melds {
			if meld.Type == "Pon" && meld.Tiles[0].Suit == targetTile.Suit && meld.Tiles[0].Value == targetTile.Value {
				foundPonIndex = i
				originalPon = meld
				break
			}
		}
		if foundPonIndex == -1 {
			gs.AddToGameLog(fmt.Sprintf("Error: Shouminkan %s failed for %s, no matching Pon found.", targetTile.Name, player.Name))
			// fmt.Printf("Error: Shouminkan declared for %s but no matching Pon found.\n", targetTile.Name)
			gs.TotalKansDeclaredThisRound--
			return
		}

		foundInHand := false
		idxToAdd := -1
		var tileToAdd Tile
		for i := len(player.Hand) - 1; i >= 0; i-- { // Search for the tile to add from hand
			if player.Hand[i].Suit == targetTile.Suit && player.Hand[i].Value == targetTile.Value {
				idxToAdd = i
				tileToAdd = player.Hand[i]
				meldTiles = append(player.Melds[foundPonIndex].Tiles, tileToAdd) // Prepare the 4 tiles for the Kan
				indicesToRemove = append(indicesToRemove, idxToAdd)
				foundInHand = true
				break
			}
		}
		if !foundInHand {
			gs.AddToGameLog(fmt.Sprintf("Error: Shouminkan %s failed for %s, tile not found in hand.", targetTile.Name, player.Name))
			// fmt.Printf("Error: Shouminkan declared for %s but tile not found in hand.\n", targetTile.Name)
			gs.TotalKansDeclaredThisRound--
			return
		}

		// --- Chankan Check (Robbing the Kan) ---
		gs.IsChankanOpportunity = true
		robbingPlayer := (*Player)(nil)
		for i, otherP := range gs.Players {
			if i == gs.GetPlayerIndex(player) {
				continue
			}
			if CanDeclareRon(otherP, tileToAdd, gs) { // tileToAdd is the one being added to the Pon
				robbingPlayer = otherP
				break
			}
		}
		gs.IsChankanOpportunity = false // Reset flag immediately after checks

		if robbingPlayer != nil {
			gs.AddToGameLog(fmt.Sprintf("%s has CHANKAN opportunity on %s's Shouminkan of %s.",
				robbingPlayer.Name, player.Name, tileToAdd.Name))
			// fmt.Printf("--- Player %s (%s) Opportunity ---\n", robbingPlayer.Name, "Chankan")
			chankanConfirm := gs.GetPlayerIndex(robbingPlayer) != 0 // AI default
			if gs.GetPlayerIndex(robbingPlayer) == 0 {              // Human player
				chankanConfirm = GetPlayerChoice(gs.InputReader, fmt.Sprintf("%s, declare CHANKAN (Ron) on %s? (y/n): ", robbingPlayer.Name, tileToAdd.Name))
			} else {
				// fmt.Printf("(%s can Chankan... AI Accepts)\n", robbingPlayer.Name)
			}

			if chankanConfirm {
				gs.AddToGameLog(fmt.Sprintf("!!! CHANKAN by %s on %s from %s adding to Shouminkan !!!",
					robbingPlayer.Name, tileToAdd.Name, player.Name))
				// fmt.Printf("\n!!! CHANKAN (Robbing the Kan) by %s on %s !!!\n", robbingPlayer.Name, tileToAdd.Name)
				gs.LastDiscard = &tileToAdd // The robbed tile is the winning tile

				// CurrentPlayerIndex for HandleWin should be the player *attempting* the Kan,
				// as they "exposed" the tile for Chankan.
				originalCurrentPlayerIndex := gs.CurrentPlayerIndex // Save current player (who is the Kanner)
				gs.CurrentPlayerIndex = gs.GetPlayerIndex(player)   // Set current player to the Kanner for HandleWin context

				HandleWin(gs, robbingPlayer, tileToAdd, false) // false for Ron

				// gs.CurrentPlayerIndex = originalCurrentPlayerIndex // Restore if needed, though game probably ends
				gs.TotalKansDeclaredThisRound-- // Kan was robbed, not completed successfully for Kanner
				return                          // Win takes precedence, stop Kan processing.
			} else {
				gs.AddToGameLog(fmt.Sprintf("%s declined Chankan.", robbingPlayer.Name))
				// fmt.Printf("%s declined Chankan.\n", robbingPlayer.Name)
				// TODO: Robbing player becomes Furiten for missing this Chankan? (Rules vary)
				// For now, no specific Furiten for declined Chankan.
			}
		}

		// --- Complete Shouminkan (if not robbed) ---
		player.Melds[foundPonIndex].Type = "Shouminkan"
		player.Melds[foundPonIndex].Tiles = meldTiles    // Update with 4 tiles
		player.Melds[foundPonIndex].CalledOn = tileToAdd // Tile that was added to make it a Kan
		// FromPlayer remains from the original Pon
		player.Hand = RemoveTilesByIndices(player.Hand, indicesToRemove) // Remove the single added tile

		// Pao check for Shouminkan: if originalPon.FromPlayer was another player
		// and this Shouminkan completes Daisangen/Daisuushii for 'player'.
		if originalPon.FromPlayer != -1 && originalPon.FromPlayer != gs.GetPlayerIndex(player) {
			if isPaoConditionMetByKan(player, targetTile, "Shouminkan", gs.Players[originalPon.FromPlayer]) {
				player.PaoSourcePlayerIndex = originalPon.FromPlayer
				gs.AddToGameLog(fmt.Sprintf("PAO Triggered: %s's original Pon, now Shouminkan by %s with %s, may lead to Pao.",
					gs.Players[originalPon.FromPlayer].Name, player.Name, targetTile.Name))
			}
		}
		success = true

	default:
		gs.AddToGameLog(fmt.Sprintf("Error: Unknown Kan type: %s for player %s", kanType, player.Name))
		// fmt.Println("Error: Unknown Kan type:", kanType)
		gs.TotalKansDeclaredThisRound--
		return
	}

	if !success {
		gs.AddToGameLog(fmt.Sprintf("Kan declaration %s for %s failed internally.", kanType, player.Name))
		// fmt.Println("Kan declaration failed.")
		gs.TotalKansDeclaredThisRound--
		return
	}

	sort.Sort(BySuitValue(player.Hand))
	// Meld tiles are already sorted if they came from hand; ensure CalledOn is handled for sort
	sort.Sort(BySuitValue(player.Melds[len(player.Melds)-1].Tiles))

	if rinshanRequired {
		rinshanTile, empty := gs.DrawRinshanTile() // This also reveals Kan Dora now
		if empty {
			gs.AddToGameLog(fmt.Sprintf("Could not draw Rinshan tile for %s after %s (no tiles left?).", player.Name, kanType))
			// fmt.Println("Could not draw Rinshan tile (no tiles left?).")
			// Check for Suukaikan abortive draw if 4 Kans were declared by different players and no Rinshan
			if CheckSuukaikan(gs) { // Checks if 4+ Kans by >= 2 players
				gs.AddToGameLog("Suukaikan! Rinshan tiles exhausted after 4th+ Kan by multiple players. Round ends in an abortive draw.")
				// fmt.Println("Suukaikan! Rinshan tiles exhausted. Round ends in an abortive draw.")
				gs.GamePhase = PhaseRoundEnd
				gs.RoundWinner = nil
				return
			}
			PromptDiscard(gs, player) // Player still needs to discard even if no Rinshan
			return
		}
		// gs.AddToGameLog(fmt.Sprintf("%s draws Rinshan tile: %s", player.Name, rinshanTile.Name)) // Logged in DrawRinshanTile
		// fmt.Printf("%s draws Rinshan tile: %s\n", player.Name, rinshanTile.Name)

		gs.IsRinshanWin = true // Set Rinshan flag before Tsumo check
		canTsumoOnRinshan := CanDeclareTsumo(player, gs)
		gs.IsRinshanWin = false // Reset after check

		if canTsumoOnRinshan {
			gs.AddToGameLog(fmt.Sprintf("!!! TSUMO (Rinshan Kaihou potentially) by %s on %s !!!", player.Name, rinshanTile.Name))
			// fmt.Printf("\n!!! TSUMO (Rinshan Kaihou potentially) by %s on %s !!!\n", player.Name, rinshanTile.Name)
			HandleWin(gs, player, rinshanTile, true) // true = Tsumo
			return                                   // End Kan handling, win takes precedence
		}
		// Player must discard again after Kan + Rinshan draw
		PromptDiscard(gs, player)
	} else {
		// Should not happen with current Kan types (Ankan, Daiminkan, Shouminkan all require Rinshan)
		PromptDiscard(gs, player)
	}
}

// HandleRiichiAction processes Riichi declaration.
func HandleRiichiAction(gs *GameState, player *Player, discardTileIndexInHand int) bool {
	// CanDeclareRiichi already verified basic conditions (score, concealed, wall tiles) in main.go
	// Now, verify *this specific* discard choice leads to Tenpai.
	if discardTileIndexInHand < 0 || discardTileIndexInHand >= len(player.Hand) {
		gs.AddToGameLog(fmt.Sprintf("Error in HandleRiichiAction: Invalid discard index %d for %s", discardTileIndexInHand, player.Name))
		return false
	}
	riichiDiscardCandidate := player.Hand[discardTileIndexInHand]
	tempHand13 := make([]Tile, 0, HandSize)
	for j, t := range player.Hand {
		if discardTileIndexInHand != j {
			tempHand13 = append(tempHand13, t)
		}
	}
	if !IsTenpai(tempHand13, player.Melds) {
		gs.AddToGameLog(fmt.Sprintf("Error: %s Riichi on %s failed internal Tenpai check.", player.Name, riichiDiscardCandidate.Name))
		// fmt.Printf("Error: Discarding %s for Riichi does not result in Tenpai. Choose another tile.\n", riichiDiscardCandidate.Name)
		return false // Invalid Riichi discard choice
	}
	player.RiichiDeclaredWaits = FindTenpaiWaits(tempHand13, player.Melds) // Store waits

	// --- Perform Riichi ---
	// gs.AddToGameLog(fmt.Sprintf("\n*** %s declares RIICHI! Discarding %s ***\n", player.Name, riichiDiscardCandidate.Name))
	// fmt.Printf("\n*** %s declares RIICHI! Discarding %s ***\n", player.Name, riichiDiscardCandidate.Name)

	player.Score -= RiichiBet
	gs.RiichiSticks++
	gs.AddToGameLog(fmt.Sprintf("%s (P%d) declares RIICHI! Discarding %s. Score: %d -> %d. Riichi Sticks: %d. Waits: %v",
		player.Name, gs.GetPlayerIndex(player)+1, riichiDiscardCandidate.Name, player.Score+RiichiBet, player.Score, gs.RiichiSticks, TilesToNames(player.RiichiDeclaredWaits)))
	// fmt.Printf("(Score: %d -> %d, Riichi Sticks: %d)\n", player.Score+RiichiBet, player.Score, gs.RiichiSticks)

	player.IsRiichi = true
	player.RiichiTurn = gs.TurnNumber // Turn when Riichi discard is made (before TurnNumber increments in DiscardTile)
	player.IsIppatsu = true           // Eligible for Ippatsu

	// Double Riichi Check: Player's first discard of the game, no prior calls.
	// GameState.TurnNumber is 0-indexed for turns *within the round*.
	// Player.InitialTurnOrder is 0-3 for their fixed seat at game start.
	// IsFirstGoAround is true if no calls and TurnNumber < numPlayers.
	isDealer := gs.Players[gs.DealerIndexThisRound] == player
	isEffectivelyFirstTurn := (isDealer && player.RiichiTurn == 0) || // Dealer's first turn
		(!isDealer && player.RiichiTurn == player.InitialTurnOrder && gs.IsFirstGoAround && !gs.AnyCallMadeThisRound) // Non-dealer, their first turn, no calls yet.
		// More robust: check if !player.HasMadeFirstDiscardThisRound *before this discard*.
		// This needs careful state. player.RiichiTurn and gs.IsFirstGoAround should cover it.

	if isEffectivelyFirstTurn && !gs.AnyCallMadeThisRound { // No calls before this Riichi
		player.DeclaredDoubleRiichi = true
		gs.AddToGameLog(fmt.Sprintf("%s also declares DOUBLE RIICHI!", player.Name))
		// fmt.Printf("%s also declares DOUBLE RIICHI!\n", player.Name)
	}
	gs.DeclaredRiichiPlayerIndices[gs.GetPlayerIndex(player)] = true // Mark player as having declared Riichi

	// Perform the discard associated with Riichi
	// DiscardTile handles removing the tile, updating LastDiscard, incrementing TurnNumber,
	// checking for calls (Ron), and potentially ending the game.
	_, gameShouldEnd := DiscardTile(gs, player, discardTileIndexInHand)

	if gameShouldEnd { // Ron was declared on the Riichi discard
		gs.AddToGameLog("Riichi declared, but Ron occurred on the discard!")
		// fmt.Println("Riichi declared, but Ron occurred on the discard!")
		return true // Indicate Riichi was successful but led immediately to game end
	}

	// If no Ron, Riichi is successfully established.
	// gs.AddToGameLog(fmt.Sprintf("Riichi successful for %s. Waiting for next turn.", player.Name))
	// fmt.Println("Riichi successful. Waiting for next turn.")
	return true // Riichi successful
}

// HandleWin processes a win by Tsumo or Ron. Calculates score and updates phase.
func HandleWin(gs *GameState, winner *Player, winningTile Tile, isTsumo bool) {
	eventPrefix := fmt.Sprintf("\n--- Round End: %s (P%d) Wins! ---", winner.Name, gs.GetPlayerIndex(winner)+1)
	gs.AddToGameLog(eventPrefix)
	// fmt.Printf(eventPrefix + "\n")

	discarder := (*Player)(nil)
	discarderIndex := -1 // Index of player who discarded for Ron
	if !isTsumo {
		// For Ron, gs.CurrentPlayerIndex is the DISCARDER when HandleWin is called
		// because DiscardTile calls HandleWin before NextPlayer or shifting turn for other calls.
		discarderIndex = gs.CurrentPlayerIndex
		if discarderIndex >= 0 && discarderIndex < len(gs.Players) {
			discarder = gs.Players[discarderIndex]
			if discarder == winner { // Should not happen if Ron logic is correct
				gs.AddToGameLog(fmt.Sprintf("Error: Winner %s is also identified as discarder %s in Ron.", winner.Name, discarder.Name))
			}
		} else {
			gs.AddToGameLog(fmt.Sprintf("Error: Invalid discarderIndex %d for Ron.", discarderIndex))
		}
		gs.AddToGameLog(fmt.Sprintf("Win Type: Ron on %s from %s (P%d).",
			winningTile.Name, If(discarder != nil, discarder.Name, "Unknown"), discarderIndex+1))
		// fmt.Printf("Win Type: Ron on %s (from %s)\n", winningTile.Name, If(discarder != nil, discarder.Name, "Error"))
	} else {
		gs.AddToGameLog(fmt.Sprintf("Win Type: Tsumo on %s by %s.", winningTile.Name, winner.Name))
		// fmt.Printf("Win Type: Tsumo on %s\n", winningTile.Name)
	}

	// gs.AddToGameLog("Winning Hand:")
	// Log full hand for debugging if needed, or rely on DisplayPlayerState
	// fmt.Println("Winning Hand:")
	// DisplayPlayerState(winner)

	if winner.IsRiichi {
		gs.RevealUraDoraIndicators() // Populates gs.UraDoraIndicators
		gs.AddToGameLog(fmt.Sprintf("Ura Dora Indicators Revealed: %v", TilesToNames(gs.UraDoraIndicators)))
		// fmt.Printf("Ura Dora Indicators Revealed: %v\n", TilesToNames(gs.UraDoraIndicators))
	}

	// gs.AddToGameLog("Calculating Yaku...")
	allWinningTiles := getAllTilesInHand(winner, winningTile, isTsumo)
	isMenzen := isMenzenchin(winner, isTsumo, winningTile)
	yakuListResults, han := IdentifyYaku(winner, winningTile, isTsumo, gs)

	if len(yakuListResults) == 0 {
		gs.AddToGameLog(fmt.Sprintf("!!! CRITICAL ERROR: No Yaku for %s's win on %s. Aborting round.", winner.Name, winningTile.Name))
		// fmt.Println("!!! CRITICAL ERROR: No Yaku found for a declared winning hand! !!!")
		gs.GamePhase = PhaseRoundEnd
		gs.RoundWinner = nil // Treat as draw/error
		return
	}
	yakuNames := []string{}
	for _, r := range yakuListResults {
		yakuNames = append(yakuNames, r.Name)
	}
	gs.AddToGameLog(fmt.Sprintf("%s Yaku: %v (%d Han)", winner.Name, yakuNames, han))
	// fmt.Printf("Yaku: %v (%d Han)\n", yakuNames, han)

	var decomposition []DecomposedGroup
	var fu int
	isYakumanWin := false
	for _, y := range yakuListResults {
		if y.Han >= 13 || strings.Contains(y.Name, "Yakuman") {
			isYakumanWin = true
			break
		}
	}
	isChiitoitsu := false
	for _, y := range yakuListResults {
		if y.Name == "Chiitoitsu" {
			isChiitoitsu = true
			break
		}
	}

	if isYakumanWin {
		fu = 0 // Fu usually not directly used for Yakuman point table lookups
		gs.AddToGameLog("Yakuman hand - Fu calculation for standard scoring table skipped.")
		// fmt.Println("Hand is Yakuman - Fu calculation skipped.")
	} else if isChiitoitsu {
		// CalculateFu will return 25 for Chiitoitsu
		fu = CalculateFu(winner, nil, winningTile, isTsumo, isMenzen, yakuListResults, gs)
		gs.AddToGameLog(fmt.Sprintf("Chiitoitsu hand - Calculated Fu: %d (should be 25)", fu))
		// fmt.Println("Hand is Chiitoitsu - Using fixed 25 Fu.")
	} else {
		var decompSuccess bool
		// gs.AddToGameLog("Decomposing standard hand...")
		decomposition, decompSuccess = DecomposeWinningHand(winner, allWinningTiles)
		if !decompSuccess {
			gs.AddToGameLog(fmt.Sprintf("!!! ERROR: Failed to decompose %s's standard winning hand! Using fallback Fu 30.", winner.Name))
			// fmt.Println("!!! ERROR: Failed to decompose standard winning hand! Cannot calculate Fu accurately. !!!")
			fu = 30 // Fallback Fu value
		} else {
			// gs.AddToGameLog("Calculating Fu based on decomposition...")
			fu = CalculateFu(winner, decomposition, winningTile, isTsumo, isMenzen, yakuListResults, gs)
		}
	}
	if !isYakumanWin { // Don't log Fu for Yakuman where it's mostly irrelevant for points
		gs.AddToGameLog(fmt.Sprintf("Calculated Fu: %d", fu))
		// fmt.Printf("Fu: %d\n", fu)
	}

	// payment := CalculatePointPayment(han, fu, winner.SeatWind == gs.PrevalentWind, isTsumo, gs.Honba, gs.RiichiSticks)
	// isWinnerDealer needs to check if winner's SEAT is East (or current dealer)
	isWinnerTheDealer := (gs.Players[gs.DealerIndexThisRound] == winner)
	payment := CalculatePointPayment(han, fu, isWinnerTheDealer, isTsumo, gs.Honba, gs.RiichiSticks)
	gs.AddToGameLog(fmt.Sprintf("Score Value: %s", payment.Description))
	// fmt.Printf("Score Value: %s\n", payment.Description)

	// --- Pao (Responsibility Payment) Logic ---
	if winner.PaoSourcePlayerIndex != -1 && isYakumanWin {
		// Pao applies if the Yakuman was directly enabled by a specific call from PaoSourcePlayerIndex
		// This check is simplified. A full Pao check would re-verify if the specific Yakuman
		// (e.g., Daisangen, Daisuushii) was indeed completed by the PaoSourcePlayer's discard/meld.
		paoSourcePlayer := gs.Players[winner.PaoSourcePlayerIndex]
		gs.AddToGameLog(fmt.Sprintf("PAO Condition: %s is responsible for %s's Yakuman!", paoSourcePlayer.Name, winner.Name))
		// fmt.Printf("PAO: %s is responsible for %s's Yakuman!\n", paoSourcePlayer.Name, winner.Name)

		// Pao player pays the full amount.
		var paoAmount int
		if isTsumo {
			// Refined Pao for Tsumo Yakuman: Pao source pays the full Ron value.
			paoAmount = payment.RonValue // This already includes Honba bonuses for a Ron.
			gs.AddToGameLog(fmt.Sprintf("PAO Tsumo Yakuman: %s (Pao source) pays full Ron value %d to %s.", paoSourcePlayer.Name, paoAmount, winner.Name))
			paoSourcePlayer.Score -= paoAmount
			winner.Score += paoAmount
			if paoSourcePlayer.Score < 0 {
				CheckAndHandleBust(gs, paoSourcePlayer, winner)
			}

			// Winner still gets Riichi sticks
			if gs.RiichiSticks > 0 {
				riichiBonus := gs.RiichiSticks * RiichiBet
				winner.Score += riichiBonus
				gs.AddToGameLog(fmt.Sprintf("%s also collects %d Riichi stick points.", winner.Name, riichiBonus))
				gs.RiichiSticks = 0
			}
			// Other players pay nothing for the Tsumo Yakuman itself.
			// Honba is included in the paoAmount (payment.RonValue).
		} else { // Ron Yakuman (Pao source is the discarder)
			paoAmount = payment.RonValue // Discarder (Pao source) pays full Ron value
			gs.AddToGameLog(fmt.Sprintf("PAO Ron Yakuman: %s (Pao source/discarder) pays %d to %s.", paoSourcePlayer.Name, paoAmount, winner.Name))
			paoSourcePlayer.Score -= paoAmount
			winner.Score += paoAmount
			if paoSourcePlayer.Score < 0 {
				CheckAndHandleBust(gs, paoSourcePlayer, winner)
			}
			// Winner still gets Riichi sticks
			if gs.RiichiSticks > 0 {
				riichiBonus := gs.RiichiSticks * RiichiBet
				winner.Score += riichiBonus
				gs.AddToGameLog(fmt.Sprintf("%s also collects %d Riichi stick points.", winner.Name, riichiBonus))
				gs.RiichiSticks = 0
			}
			// Honba is included in paoAmount.
		}
	} else { // Normal Point Transfer (No Pao, or Pao but not Yakuman)
		TransferPoints(gs, winner, discarder, isTsumo, payment)
	}

	scoreLog := "Scores: "
	for i, p := range gs.Players {
		scoreLog += fmt.Sprintf("P%d %s: %d; ", i+1, p.Name, p.Score)
	}
	gs.AddToGameLog(scoreLog)
	// fmt.Println("--- Scores After Transfer ---")
	// for i, p := range gs.Players { fmt.Printf("P%d %s: %d\n", i+1, p.Name, p.Score) }

	gs.GamePhase = PhaseRoundEnd
	gs.RoundWinner = winner
	// gs.AddToGameLog(fmt.Sprintf("\n--- Round Over. Winner: %s ---", winner.Name))
	// fmt.Println("\n--- Round Over ---")
}

// PromptDiscard forces the specified player (usually after a call or Kan) to discard.
func PromptDiscard(gs *GameState, player *Player) {
	gs.AddToGameLog(fmt.Sprintf("%s's turn (must discard after call/Kan).", player.Name))
	// fmt.Printf("\n--- %s's Turn (Must Discard after Call/Kan) ---\n", player.Name)
	// DisplayPlayerState(player) // Display is handled in main loop before choices for human

	// Check for Kan possibilities (Ankan/Shouminkan) *before* discarding
	canDeclareKan := false
	kanTarget := Tile{}
	kanType := ""
	if !player.IsRiichi { // Cannot declare Kan from hand if Riichi, unless it's Ankan not changing waits (handled in CanDeclareKanOnHand)
		uniqueHandTiles := GetUniqueTiles(player.Hand)
		for _, tileToCheck := range uniqueHandTiles {
			kType, kTarget := CanDeclareKanOnHand(player, tileToCheck, gs) // Pass gs for context
			if kType != "" {
				canDeclareKan = true
				kanType = kType
				kanTarget = kTarget
				break
			}
		}
	}

	if canDeclareKan { // This implies player is not Riichi or it's a valid Riichi Kan
		isHuman := gs.GetPlayerIndex(player) == 0
		kanConfirm := !isHuman // AI default: Kan if possible and safe
		if isHuman {
			DisplayPlayerState(player) // Show hand before Kan choice
			kanConfirm = GetPlayerChoice(gs.InputReader, fmt.Sprintf("Declare %s with %s before discarding? (y/n): ", kanType, kanTarget.Name))
		} else { // AI Kan logic (after call/Kan)
			if player.IsRiichi && checkWaitChangeForRiichiKan(player, gs, kanTarget, kanType) {
				kanConfirm = false // AI: Don't Kan if Riichi and it changes waits
				gs.AddToGameLog(fmt.Sprintf("AI %s skips %s with %s (unsafe for Riichi after call/Kan).", player.Name, kanType, kanTarget.Name))
			}
			// Otherwise, AI confirms
		}

		if kanConfirm {
			HandleKanAction(gs, player, kanTarget, kanType)
			return // Kan handler takes over, may lead to another PromptDiscard via Rinshan or win
		}
	}

	// --- Normal Discard after Call/Kan (if no further Kan declared) ---
	if len(player.Hand) == 0 {
		gs.AddToGameLog(fmt.Sprintf("Error: Player %s has no tiles to discard after call/Kan (PromptDiscard).", player.Name))
		// fmt.Printf("Error: Player %s has no tiles to discard after call/Kan?\n", player.Name)
		gs.GamePhase = PhaseRoundEnd
		gs.RoundWinner = nil // Error state, treat as draw
		return
	}

	var discardIndex int
	isHuman := gs.GetPlayerIndex(player) == 0
	if isHuman {
		DisplayPlayerState(player) // Show hand again if no Kan chosen, before discard
		// fmt.Println("Choose tile to discard:")
		discardIndex = GetPlayerDiscardChoice(gs.InputReader, player)
	} else { // AI discard logic after call/Kan
		// gs.AddToGameLog(fmt.Sprintf("AI %s thinking for discard after call/Kan...", player.Name))
		// Basic AI: discard the tile that was just drawn (player.JustDrawnTile),
		// or if that's null (e.g. after Pon/Chi), discard last tile.
		if player.JustDrawnTile != nil { // Should be set if this was after Rinshan
			foundDrawn := false
			for i, t := range player.Hand {
				if t.ID == player.JustDrawnTile.ID {
					discardIndex = i
					foundDrawn = true
					break
				}
			}
			if !foundDrawn { // Fallback
				if len(player.Hand) > 0 {
					discardIndex = len(player.Hand) - 1
				} else {
					discardIndex = -1
				}
			}
		} else { // After Pon/Chi, no JustDrawnTile from Rinshan. Discard last.
			if len(player.Hand) > 0 {
				discardIndex = len(player.Hand) - 1
			} else {
				discardIndex = -1
			}
		}

		// Safety check for AI
		if discardIndex < 0 || discardIndex >= len(player.Hand) {
			if len(player.Hand) > 0 {
				discardIndex = 0
			} else {
				discardIndex = -1
			} // Error state if no hand
		}
	}

	if discardIndex >= 0 && discardIndex < len(player.Hand) {
		DiscardTile(gs, player, discardIndex) // This will handle turn progression
	} else {
		gs.AddToGameLog(fmt.Sprintf("Error: Invalid discard index %d for %s in PromptDiscard (after call/Kan).", discardIndex, player.Name))
		// fmt.Printf("Error: Invalid discard index %d chosen in PromptDiscard.\n", discardIndex)
		if len(player.Hand) > 0 {
			// fmt.Println("Defaulting to discard index 0.")
			DiscardTile(gs, player, 0) // Fallback
		} else {
			gs.GamePhase = PhaseRoundEnd
			gs.RoundWinner = nil
		}
	}
}
