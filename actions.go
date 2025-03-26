package main

import (
	"fmt"
	"sort"
	"strings"
	// Added for checking Kan types
)

// Helper function to remove the last discard from a specific player's list after a call.
// Assumes gs.LastDiscard holds the tile that was called.
func removeLastDiscardFromPlayer(gs *GameState, playerIndex int) {
	if playerIndex < 0 || playerIndex >= len(gs.Players) || gs.LastDiscard == nil {
		fmt.Println("Warning: Attempted to remove last discard with invalid index or nil LastDiscard.")
		return // Invalid index or no last discard to remove
	}
	player := gs.Players[playerIndex]
	if len(player.Discards) > 0 {
		// Assume the discard just added is the last element and matches gs.LastDiscard
		lastIndex := len(player.Discards) - 1
		if player.Discards[lastIndex].ID == gs.LastDiscard.ID {
			// Remove the last element efficiently
			player.Discards = player.Discards[:lastIndex]
			// fmt.Printf("Debug: Removed %s from P%d's discards after call.\n", gs.LastDiscard.Name, playerIndex+1) // Optional Debug
		} else {
			// Fallback/Warning: The last element didn't match the called tile.
			fmt.Printf("Warning: Last discard mismatch when removing called tile %s for P%d. Discards: %v\n",
				gs.LastDiscard.Name, playerIndex+1, TilesToNames(player.Discards))
			// Slower search could be added here if needed.
		}
	} else {
		fmt.Printf("Warning: Attempted to remove last discard from P%d, but their discard list is empty.\n", playerIndex+1)
	}
}

// DiscardTile handles the current player discarding a tile.
// It checks for calls (Ron, Kan, Pon, Chi) from other players and handles turn progression.
// Returns the discarded tile and if the game/round ended (e.g., Ron).
func DiscardTile(gs *GameState, player *Player, tileIndex int) (Tile, bool) {
	if tileIndex < 0 || tileIndex >= len(player.Hand) {
		fmt.Println("Error: Invalid tile index to discard.")
		return Tile{}, false // Indicate error without ending game yet
	}
	discardedTile := player.Hand[tileIndex]

	// --- Riichi Discard Restrictions ---
	if player.IsRiichi && len(player.Hand) == HandSize+1 {
		// Find the actual drawn tile (might not be the last one if Kan checks happened?)
		// A more robust method would be to store the drawn tile temporarily.
		// Simple assumption: find the tile *not* present before the draw. This is hard.
		// Easiest (but potentially wrong if sorting changes things): assume drawn is last.
		// Let's refine: In Riichi, you MUST discard the drawn tile unless you ANKAN/SHOUMINKAN the drawn tile.
		// We need to know WHICH tile was drawn. Assume it's stored somewhere or passed in.
		// FOR NOW: Assume the check `CanDeclareKanOnDraw` handled the Kan case. If we reach here in Riichi,
		// the discard *must* be the drawn tile. Let's find it by comparing counts? No, that's complex.
		// SAFEST SIMPLE RULE: If Riichi, force discard of the tile at index HandSize (the 14th tile).

		// Correct approach: Need to know the drawn tile. Let's assume it was added last before sorting.
		// This needs careful handling in the main loop. Re-evaluate this logic.
		// If we assume the hand *is already sorted* after draw, the drawn tile *could be anywhere*.

		// Let's just force the provided index for now, but add a warning.
		// A better implementation passes the drawn tile's ID.
		fmt.Println("--- RIICHI DISCARD ---")
		// We assume the main loop ensured the chosen discardIndex corresponds to the drawn tile
		// if no Kan was declared. If the user chose wrong, this might error later.
	}

	// --- Perform Discard ---
	player.Hand = append(player.Hand[:tileIndex], player.Hand[tileIndex+1:]...)
	player.Discards = append(player.Discards, discardedTile) // Add to discarder's list FIRST
	gs.LastDiscard = &discardedTile                          // Update game state's tracked last discard
	gs.TurnNumber++                                          // Increment turn number *after* discard is made

	UpdateFuritenStatus(player, gs) // Update based on *own* discard

	fmt.Printf("%s discards: %s\n", player.Name, discardedTile.Name)

	// --- Check for Calls ---
	ronPlayers := []*Player{}
	kanPlayers := []*Player{}
	ponPlayers := []*Player{}
	chiPlayer := (*Player)(nil)
	potentialCalls := make(map[string][]*Player)
	playerIdx := gs.CurrentPlayerIndex // The index of the player who just discarded

	for i, otherPlayer := range gs.Players {
		if i == playerIdx {
			continue // Cannot call on own discard
		}
		// Check Ron
		if CanDeclareRon(otherPlayer, discardedTile, gs) {
			potentialCalls["Ron"] = append(potentialCalls["Ron"], otherPlayer)
			ronPlayers = append(ronPlayers, otherPlayer)
		}
		// *** ADD THIS CHECK: Players in Riichi cannot make open calls ***
		if otherPlayer.IsRiichi {
			continue // Skip Pon, Kan, Chi checks for players in Riichi
		}

		// Now check other calls only if player is NOT in Riichi
		// Check Daiminkan (Open Kan)
		if CanDeclareDaiminkan(otherPlayer, discardedTile) {
			potentialCalls["Kan"] = append(potentialCalls["Kan"], otherPlayer)
			kanPlayers = append(kanPlayers, otherPlayer)
		}
		// Check Pon
		if CanDeclarePon(otherPlayer, discardedTile) {
			potentialCalls["Pon"] = append(potentialCalls["Pon"], otherPlayer)
			ponPlayers = append(ponPlayers, otherPlayer)
		}
		// Check Chi (Only player to the left)
		if i == (playerIdx+1)%len(gs.Players) {
			if CanDeclareChi(otherPlayer, discardedTile) {
				chiPlayer = otherPlayer
				// Add Chi possibility marker? Not strictly needed as only one player can Chi.
			}
		}
	}

	// --- Handle Calls based on Priority ---
	callMade := false

	// 1. Ron
	if len(ronPlayers) > 0 {
		// Simplification: First Ron wins. Multiple Ron is complex (head bump etc.).
		winner := ronPlayers[0]
		isHumanWinner := gs.GetPlayerIndex(winner) == 0
		fmt.Printf("--- Player %s (%s) Opportunity ---\n", winner.Name, "Ron")

		ronConfirm := true // Default AI to accept
		if isHumanWinner {
			ronConfirm = GetPlayerChoice(gs.InputReader, fmt.Sprintf("%s, declare RON on %s? (y/n): ", winner.Name, discardedTile.Name))
		} else {
			fmt.Printf("(%s can Ron... AI Accepts)\n", winner.Name)
		}

		if ronConfirm {
			fmt.Printf("\n!!! RON by %s on %s !!!\n", winner.Name, discardedTile.Name)
			for _, p := range gs.Players {
				p.IsIppatsu = false
			} // Ron breaks Ippatsu
			HandleWin(gs, winner, discardedTile, false)
			return discardedTile, true // Game/round ends
		} else { // Declined Ron
			fmt.Printf("%s declined Ron.\n", winner.Name)
			winner.IsFuriten = true // Temporary Furiten for missing Ron
			// TODO: Need precise Furiten timing (until own next discard)
			potentialCalls["Ron"] = nil // Remove Ron possibility
			ronPlayers = nil
		}
	}

	// 2. Kan / Pon (Highest priority among open melds)
	// Check Daiminkan
	if !callMade && len(potentialCalls["Kan"]) > 0 {
		// Simplification: Only handle one Kan caller for now
		caller := potentialCalls["Kan"][0]
		isHumanCaller := gs.GetPlayerIndex(caller) == 0
		fmt.Printf("--- Player %s (%s) Opportunity ---\n", caller.Name, "Kan")

		kanConfirm := false // Default AI to decline
		if isHumanCaller {
			DisplayPlayerState(caller) // Show hand before decision
			kanConfirm = GetPlayerChoice(gs.InputReader, fmt.Sprintf("%s, declare KAN on %s? (y/n): ", caller.Name, discardedTile.Name))
		} else {
			fmt.Printf("(%s can declare Kan... AI declines for now)\n", caller.Name)
		}

		if kanConfirm {
			callMade = true
			discarderIndex := playerIdx // Save who discarded
			for _, p := range gs.Players {
				p.IsIppatsu = false
			} // Call breaks Ippatsu
			callerIndex := gs.GetPlayerIndex(caller)
			removeLastDiscardFromPlayer(gs, discarderIndex) // Remove tile from discarder's list *before* turn potentially shifts
			gs.CurrentPlayerIndex = callerIndex             // Turn shifts to caller *immediately* for Kan
			HandleKanAction(gs, caller, discardedTile, "Daiminkan")
			// HandleKanAction -> Rinshan -> PromptDiscard sequence follows, turn stays with caller
			return discardedTile, false // Game continues, Kan caller takes over
		} else {
			if isHumanCaller {
				fmt.Printf("%s declined Kan.\n", caller.Name)
			}
			potentialCalls["Kan"] = nil
		}
	}

	// Check Pon (only if Kan not called or declined)
	if !callMade && len(potentialCalls["Pon"]) > 0 {
		// Simplification: Only handle one Pon caller for now
		caller := potentialCalls["Pon"][0]
		isHumanCaller := gs.GetPlayerIndex(caller) == 0
		fmt.Printf("--- Player %s (%s) Opportunity ---\n", caller.Name, "Pon")

		ponConfirm := false // Default AI to decline
		if isHumanCaller {
			DisplayPlayerState(caller) // Show hand before decision
			ponConfirm = GetPlayerChoice(gs.InputReader, fmt.Sprintf("%s, declare PON on %s? (y/n): ", caller.Name, discardedTile.Name))
		} else {
			fmt.Printf("(%s can declare Pon... AI declines for now)\n", caller.Name)
		}

		if ponConfirm {
			callMade = true
			discarderIndex := playerIdx // Save who discarded
			for _, p := range gs.Players {
				p.IsIppatsu = false
			} // Call breaks Ippatsu
			callerIndex := gs.GetPlayerIndex(caller)
			removeLastDiscardFromPlayer(gs, discarderIndex) // Remove tile from discarder's list
			gs.CurrentPlayerIndex = callerIndex             // Turn shifts to caller
			HandlePonAction(gs, caller, discardedTile)      // Update state
			PromptDiscard(gs, caller)                       // Caller must discard next
			return discardedTile, false                     // Game continues, Pon caller discards
		} else {
			if isHumanCaller {
				fmt.Printf("%s declined Pon.\n", caller.Name)
			}
			potentialCalls["Pon"] = nil
		}
	}

	// 3. Chi (only if no higher priority call made, and only by player to the left)
	if !callMade && chiPlayer != nil {
		caller := chiPlayer
		isHumanCaller := gs.GetPlayerIndex(caller) == 0
		fmt.Printf("--- Player %s (%s) Opportunity ---\n", caller.Name, "Chi")

		chiConfirmedAndHandled := false
		if isHumanCaller {
			DisplayPlayerState(caller) // Show hand before decision
			choiceNum, sequence := GetChiChoice(gs, caller, discardedTile)
			if choiceNum > 0 {
				callMade = true
				chiConfirmedAndHandled = true
				discarderIndex := playerIdx // Save who discarded
				for _, p := range gs.Players {
					p.IsIppatsu = false
				} // Call breaks Ippatsu
				callerIndex := gs.GetPlayerIndex(caller)
				removeLastDiscardFromPlayer(gs, discarderIndex)      // Remove tile from discarder's list
				gs.CurrentPlayerIndex = callerIndex                  // Turn shifts to caller
				HandleChiAction(gs, caller, discardedTile, sequence) // Update state
				PromptDiscard(gs, caller)                            // Caller must discard next
				// Game continues, Chi caller discards
			} else {
				fmt.Printf("%s declined Chi.\n", caller.Name)
			}
		} else {
			fmt.Printf("(%s can declare Chi... AI declines for now)\n", caller.Name)
		}
		if chiConfirmedAndHandled {
			return discardedTile, false // Game continues, Chi caller discards
		}
		// If Chi declined or AI, fall through
	}

	// --- No Calls Made or All Calls Declined ---
	if !callMade {
		// If the discarding player was in Riichi, their Ippatsu chance is now gone.
		if player.IsRiichi {
			player.IsIppatsu = false
		}
		gs.NextPlayer() // Proceed to next player only if no call interrupted the flow
	}

	return discardedTile, false // Game continues, next player's turn (or caller's discard)
}

// HandlePonAction processes the Pon action, updates player state.
func HandlePonAction(gs *GameState, player *Player, discardedTile Tile) {
	meldTiles := []Tile{discardedTile} // Start with the called tile
	indicesToRemove := []int{}
	foundCount := 0

	// Find 2 matching tiles in hand (use ID for robustness against red fives)
	for i := len(player.Hand) - 1; i >= 0; i-- { // Iterate backwards for safe removal
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
		// Remove tiles from hand (using the collected indices)
		player.Hand = RemoveTilesByIndices(player.Hand, indicesToRemove)

		// Determine source player index (player who just discarded)
		// This is tricky because CurrentPlayerIndex might have changed.
		// We need the index of the player *before* the caller.
		callerIndex := gs.GetPlayerIndex(player)
		discarderIndex := (callerIndex - 1 + len(gs.Players)) % len(gs.Players)
		// OR rely on gs.LastDiscard being set correctly and find the player whose last discard matches?
		// Let's assume the calculation based on current caller index is correct for now.

		// Add Pon meld
		sort.Sort(BySuitValue(meldTiles)) // Sort the meld tiles for consistency
		newMeld := Meld{
			Type:        "Pon",
			Tiles:       meldTiles,
			CalledOn:    discardedTile,
			FromPlayer:  discarderIndex, // Index of player who discarded
			IsConcealed: false,
		}
		player.Melds = append(player.Melds, newMeld)
		sort.Sort(BySuitValue(player.Hand)) // Keep hand sorted

		fmt.Printf("\n%s PONS %s from P%d!\n", player.Name, discardedTile.Name, discarderIndex+1)
		// DisplayPlayerState(player) // Displayed in PromptDiscard

		// Turn progression handled in DiscardTile: Caller is set, PromptDiscard is called.

	} else {
		fmt.Println("Error: Could not find 2 tiles for Pon in player's hand. (Should not happen if CheckPon was correct)")
	}
}

// HandleChiAction processes the Chi action, updates player state.
// `sequence` includes the 3 tiles: 2 from hand + discardedTile.
func HandleChiAction(gs *GameState, player *Player, discardedTile Tile, sequence []Tile) {
	indicesToRemove := []int{}
	foundCount := 0
	tilesFromHand := []Tile{} // Keep track of the actual tiles removed from hand

	// Find the two tiles *other than* discardedTile in the player's hand
	for _, seqTile := range sequence {
		if seqTile.ID == discardedTile.ID {
			continue
		} // Skip the discarded tile itself

		found := false
		// Iterate backwards for safe removal indices
		for i := len(player.Hand) - 1; i >= 0; i-- {
			handTile := player.Hand[i]
			// Use ID for exact match
			if handTile.ID == seqTile.ID {
				// Check if this index is already marked for removal
				alreadyMarked := false
				for _, idx := range indicesToRemove {
					if idx == i {
						alreadyMarked = true
						break
					}
				}
				if !alreadyMarked {
					indicesToRemove = append(indicesToRemove, i)
					tilesFromHand = append(tilesFromHand, handTile) // Store the tile being removed
					foundCount++
					found = true
					break // Found one instance of this required tile
				}
			}
		}
		if !found {
			fmt.Printf("Error: Could not find required tile %s for Chi in hand.\n", seqTile.Name)
			return // Error state
		}
	}

	if foundCount == 2 {
		// Remove the two hand tiles
		player.Hand = RemoveTilesByIndices(player.Hand, indicesToRemove)

		// Determine source player index (player who just discarded)
		// Chi can only be called from player to the left.
		callerIndex := gs.GetPlayerIndex(player)
		discarderIndex := (callerIndex - 1 + len(gs.Players)) % len(gs.Players)

		// Add Chi meld
		sort.Sort(BySuitValue(sequence)) // Sort the full sequence
		newMeld := Meld{
			Type:        "Chi",
			Tiles:       sequence,       // Contains all 3 tiles
			CalledOn:    discardedTile,  // The specific tile called
			FromPlayer:  discarderIndex, // Player who discarded
			IsConcealed: false,
		}
		player.Melds = append(player.Melds, newMeld)
		sort.Sort(BySuitValue(player.Hand)) // Keep hand sorted

		fmt.Printf("\n%s CHIS %s from P%d (using %s, %s)!\n",
			player.Name, discardedTile.Name, discarderIndex+1, tilesFromHand[0].Name, tilesFromHand[1].Name) // Show the two from hand
		// DisplayPlayerState(player) // Displayed in PromptDiscard

		// Turn progression handled in DiscardTile

	} else {
		fmt.Printf("Error: Found %d hand tiles for Chi, expected 2. Indices: %v (Should not happen)\n", foundCount, indicesToRemove)
	}
}

// HandleKanAction processes Kan declarations (all types).
// Handles tile removal, meld creation, Rinshan draw, and triggers next discard.
func HandleKanAction(gs *GameState, player *Player, targetTile Tile, kanType string) {
	meldTiles := []Tile{}
	indicesToRemove := []int{} // Indices in the player's hand to remove
	newMeld := Meld{Type: kanType}
	success := false
	rinshanRequired := true // Most Kans require a Rinshan draw

	switch kanType {
	case "Ankan": // Closed Kan (4 identical tiles from hand)
		count := 0
		for i := len(player.Hand) - 1; i >= 0; i-- {
			// Use Suit/Value match for grouping, but collect specific tiles
			if player.Hand[i].Suit == targetTile.Suit && player.Hand[i].Value == targetTile.Value {
				meldTiles = append(meldTiles, player.Hand[i])
				indicesToRemove = append(indicesToRemove, i)
				count++
			}
		}
		if count == 4 {
			// TODO: Check Riichi wait change rules if player is in Riichi
			newMeld.Tiles = meldTiles
			newMeld.IsConcealed = true
			newMeld.FromPlayer = -1   // No tile called from others
			newMeld.CalledOn = Tile{} // No specific called tile
			player.Melds = append(player.Melds, newMeld)
			player.Hand = RemoveTilesByIndices(player.Hand, indicesToRemove)
			success = true
		} else {
			fmt.Printf("Error: Ankan declared but found %d matching tiles for %s in hand.\n", count, targetTile.Name)
		}

	case "Daiminkan": // Open Kan (3 from hand + 1 from discard)
		meldTiles = append(meldTiles, targetTile) // The discarded tile is part of the meld
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
			// Determine discarder index - tricky after turn shift.
			// Assume it was the player *before* the current player index.
			callerIndex := gs.GetPlayerIndex(player)
			discarderIndex := (callerIndex - 1 + len(gs.Players)) % len(gs.Players) // Player who discarded

			newMeld.Tiles = meldTiles
			newMeld.CalledOn = targetTile
			newMeld.FromPlayer = discarderIndex
			newMeld.IsConcealed = false
			player.Melds = append(player.Melds, newMeld)
			player.Hand = RemoveTilesByIndices(player.Hand, indicesToRemove)
			success = true
		} else {
			fmt.Printf("Error: Daiminkan declared but found %d matching tiles for %s in hand.\n", count, targetTile.Name)
		}

	case "Shouminkan": // Added Kan (1 from hand/draw + existing Pon)
		foundPonIndex := -1
		var originalPon Meld
		for i, meld := range player.Melds {
			if meld.Type == "Pon" && meld.Tiles[0].Suit == targetTile.Suit && meld.Tiles[0].Value == targetTile.Value {
				foundPonIndex = i
				originalPon = meld // Store the original Pon details
				break
			}
		}
		if foundPonIndex == -1 {
			fmt.Printf("Error: Shouminkan declared for %s but no matching Pon found.\n", targetTile.Name)
			break // Exit switch
		}

		// Find the matching tile in hand (the one being added)
		foundInHand := false
		idxToAdd := -1
		var tileToAdd Tile
		for i := len(player.Hand) - 1; i >= 0; i-- {
			if player.Hand[i].Suit == targetTile.Suit && player.Hand[i].Value == targetTile.Value {
				idxToAdd = i
				tileToAdd = player.Hand[idxToAdd] // Use idxToAdd here
				meldTiles = append(player.Melds[foundPonIndex].Tiles, tileToAdd)
				indicesToRemove = append(indicesToRemove, idxToAdd) // And here
				foundInHand = true
				break
			}
		}
		if !foundInHand {
			fmt.Printf("Error: Shouminkan declared for %s but tile not found in hand.\n", targetTile.Name)
			break // Exit switch
		}

		// --- Chankan Check (Robbing the Kan) ---
		canBeRobbed := false
		robbingPlayer := (*Player)(nil)
		for i, otherPlayer := range gs.Players {
			if i == gs.GetPlayerIndex(player) {
				continue
			}

			// Check if another player can Ron on the tile *being added* (tileToAdd)
			// Need to consider the state *as if* the Kan hasn't fully happened yet.
			// Pass tileToAdd as the winning tile. The 'player' state still has the Pon.
			if CanDeclareRon(otherPlayer, tileToAdd, gs) {
				// Check for Yaku. Chankan itself is a Yaku (1 Han).
				// IdentifyYaku called by CanDeclareRon should find Chankan if applicable.
				// TODO: Ensure IdentifyYaku has logic for Chankan situation.
				canBeRobbed = true
				robbingPlayer = otherPlayer // Assume first one robs for now
				break
			}
		}

		if canBeRobbed {
			isHumanRobber := gs.GetPlayerIndex(robbingPlayer) == 0
			fmt.Printf("--- Player %s (%s) Opportunity ---\n", robbingPlayer.Name, "Chankan")
			chankanConfirm := true // AI Default
			if isHumanRobber {
				chankanConfirm = GetPlayerChoice(gs.InputReader, fmt.Sprintf("%s, declare CHANKAN (Ron) on %s? (y/n): ", robbingPlayer.Name, tileToAdd.Name))
			} else {
				fmt.Printf("(%s can Chankan... AI Accepts)\n", robbingPlayer.Name)
			}

			if chankanConfirm {
				fmt.Printf("\n!!! CHANKAN (Robbing the Kan) by %s on %s !!!\n", robbingPlayer.Name, tileToAdd.Name)
				// The 'discarder' for Chankan is the player attempting the Shouminkan.
				gs.LastDiscard = &tileToAdd                       // Treat the robbed tile as the discard for scoring/state? Rules vary. Set it for HandleWin.
				gs.CurrentPlayerIndex = gs.GetPlayerIndex(player) // Set index to Kan declarer for discarder payment logic
				HandleWin(gs, robbingPlayer, tileToAdd, false)    // Ron win for robbing player
				// Game potentially ends, HandleWin sets phase.
				return // Stop Kan processing, win takes precedence.
			} else {
				fmt.Printf("%s declined Chankan.\n", robbingPlayer.Name)
				// Robbing player becomes Furiten? Check rules.
			}
		}

		// --- Complete Shouminkan (if not robbed) ---
		// TODO: Check Riichi wait change rules if player is in Riichi
		// Update the existing Pon meld in place
		player.Melds[foundPonIndex].Type = "Shouminkan"
		player.Melds[foundPonIndex].Tiles = meldTiles    // Update with 4 tiles
		player.Melds[foundPonIndex].CalledOn = tileToAdd // Tile that was added
		// Use the FromPlayer from the original Pon meld!
		player.Melds[foundPonIndex].FromPlayer = originalPon.FromPlayer
		player.Melds[foundPonIndex].IsConcealed = false // Remains open

		// Remove the single added tile from hand
		player.Hand = RemoveTilesByIndices(player.Hand, indicesToRemove)
		success = true

	default:
		fmt.Println("Error: Unknown Kan type:", kanType)
		return
	}

	if !success {
		fmt.Println("Kan declaration failed.")
		return // Do not proceed with Rinshan draw etc.
	}

	sort.Sort(BySuitValue(player.Hand)) // Re-sort hand
	sort.Sort(BySuitValue(meldTiles))   // Sort the meld itself for display consistency

	fmt.Printf("\n%s declares %s with %s!\n", player.Name, kanType, targetTile.Name)
	// DisplayPlayerState(player) // Displayed again before discard

	// --- Kan Consequences ---
	// Any Kan breaks Ippatsu for anyone who was eligible
	for _, p := range gs.Players {
		p.IsIppatsu = false
	}

	// Reveal Kan Dora Indicator *before* drawing Rinshan? Or after?
	// Standard: Draw Rinshan, *then* flip Kan Dora indicator.
	// gs.RevealKanDoraIndicator() // Moved inside DrawRinshanTile

	if rinshanRequired {
		// 1. Draw Rinshan tile
		rinshanTile, empty := gs.DrawRinshanTile() // This also reveals Kan Dora now
		if empty {
			fmt.Println("Could not draw Rinshan tile (no tiles left?).")
			// Handle abortive draw? (e.g., Ssu Kaikan - 4 Kans)
			// Check if 4 Kans were declared?
			// TODO: Implement Ssu Kaikan check (4 Kans by *different* players is usually abortive draw)
			// For now, just end the turn? This state is unusual.
			PromptDiscard(gs, player) // Force discard even without Rinshan? Or abort? Let's prompt discard.
			return
		}
		fmt.Printf("%s draws Rinshan tile: %s\n", player.Name, rinshanTile.Name)
		// DisplayPlayerState(player) // Displayed again before discard

		// 2. Check for Tsumo on Rinshan (Rinshan Kaihou Yaku)
		// CanDeclareTsumo now checks for Yaku, Rinshan Kaihou should be added in yaku.go
		if CanDeclareTsumo(player, gs) {
			// IdentifyYaku needs to know it's a Rinshan win context if needed
			// TODO: Pass Rinshan context to IdentifyYaku if it affects other Yaku checks
			fmt.Printf("\n!!! TSUMO (Rinshan Kaihou potentially) by %s !!!\n", player.Name)
			HandleWin(gs, player, rinshanTile, true) // true = Tsumo
			// HandleWin should set GamePhase to RoundEnd/GameEnd
			return // End Kan handling
		}

		// 3. Player must discard again after Kan + Rinshan draw
		// Turn remains with the Kan caller. Prompt for discard.
		PromptDiscard(gs, player)

	} else {
		// Should not happen with current Kan types, but if a Kan existed that didn't need Rinshan:
		PromptDiscard(gs, player)
	}
}

// HandleRiichiAction function...
// Use both return values from CanDeclareRiichi
func HandleRiichiAction(gs *GameState, player *Player, discardIndex int) bool {
	canRiichi, _ := CanDeclareRiichi(player, gs) // Get both values, ignore options here if not needed
	if !canRiichi {                              // Use the boolean result
		fmt.Println("Cannot declare Riichi (check conditions failed again - internal check).")
		return false // Failed
	}

	// Verify the chosen discard actually results in Tenpai (CanDeclareRiichi just checks if *any* discard works)
	riichiDiscard := player.Hand[discardIndex]
	tempHand13 := make([]Tile, 0, HandSize)
	for j, t := range player.Hand {
		if discardIndex != j {
			tempHand13 = append(tempHand13, t)
		}
	}
	if !IsTenpai(tempHand13, player.Melds) {
		fmt.Printf("Error: Discarding %s for Riichi does not result in Tenpai. Choose another tile.\n", riichiDiscard.Name)
		return false // Invalid Riichi discard choice
	}

	// --- Perform Riichi ---
	fmt.Printf("\n*** %s declares RIICHI! Discarding %s ***\n", player.Name, riichiDiscard.Name)

	// Deduct 1000 points, add stick to table
	player.Score -= 1000
	gs.RiichiSticks++
	fmt.Printf("(Score: %d -> %d, Riichi Sticks: %d)\n", player.Score+1000, player.Score, gs.RiichiSticks)

	// Set Riichi state
	player.IsRiichi = true
	// Record turn number *before* discard increments it in DiscardTile
	// Store the turn *index* when the Riichi discard is *made*.
	player.RiichiTurn = gs.TurnNumber
	player.IsIppatsu = true // Eligible for Ippatsu until next draw/call/own Kan

	// Perform the discard associated with Riichi
	// DiscardTile handles removing the tile, updating LastDiscard, incrementing TurnNumber,
	// checking for calls (Ron), and potentially ending the game.
	_, gameShouldEnd := DiscardTile(gs, player, discardIndex)

	// If Ron was declared on the Riichi discard, the game ends
	if gameShouldEnd {
		fmt.Println("Riichi declared, but Ron occurred on the discard!")
		return true // Indicate Riichi was successful but led immediately to game end
	}

	// If no Ron, Riichi is successfully established.
	// Turn proceeds normally after DiscardTile handles call checks/next player advancement.
	fmt.Println("Riichi successful. Waiting for next turn.")
	return true // Riichi successful
}

// HandleWin processes a win by Tsumo or Ron. Calculates score and updates phase.
func HandleWin(gs *GameState, winner *Player, winningTile Tile, isTsumo bool) {
	fmt.Printf("\n--- Round End: %s Wins! ---\n", winner.Name)

	// Determine source of winning tile
	discarder := (*Player)(nil)
	discarderIndex := -1
	if !isTsumo {
		// Ron - find who discarded the winningTile (gs.LastDiscard should be it)
		// The player index *before* the winner made the call is the discarder.
		// If Ron on Riichi discard, CurrentPlayerIndex was winner, discarder was previous.
		// If Ron via call, CurrentPlayerIndex was discarder when call happened.
		// Use gs.LastDiscard owner if available? No, use player index.

		// When HandleWin is called for Ron, gs.CurrentPlayerIndex points to the DISCARDER
		// because the turn hadn't advanced yet (Ron has priority).
		discarderIndex = gs.CurrentPlayerIndex
		if discarderIndex >= 0 && discarderIndex < len(gs.Players) {
			discarder = gs.Players[discarderIndex]
		}

		// Sanity check: Does LastDiscard match winningTile?
		if gs.LastDiscard == nil || gs.LastDiscard.ID != winningTile.ID {
			fmt.Printf("Warning: Winning tile %s for Ron doesn't match gs.LastDiscard %v.\n", winningTile.Name, gs.LastDiscard)
			// Try to find discarder based on last discard in player lists? Risky.
			// Assume calculated discarderIndex is correct contextually.
		}

		fmt.Printf("Win Type: Ron on %s", winningTile.Name)
		if discarder != nil {
			fmt.Printf(" (from %s)\n", discarder.Name)
		} else {
			fmt.Println(" (Error: Could not identify discarder!)")
		}
	} else {
		fmt.Printf("Win Type: Tsumo on %s\n", winningTile.Name)
	}

	// 1. Reveal Hand
	fmt.Println("Winning Hand:")
	DisplayPlayerState(winner) // Show full hand + melds

	// 2. Reveal Ura Dora (if Riichi)
	if winner.IsRiichi {
		gs.RevealUraDoraIndicators() // Populate gs.UraDoraIndicators
		// Display is handled later by DisplayGameState if needed, or here:
		fmt.Printf("Ura Dora Indicators Revealed: %v\n", TilesToNames(gs.UraDoraIndicators))
	}

	// 3. Calculate Yaku and Score
	fmt.Println("Calculating Yaku...")
	allWinningTiles := getAllTilesInHand(winner, winningTile, isTsumo) // Get all 14 tiles
	isMenzen := isMenzenchin(winner, isTsumo, winningTile)             // Determine concealment status
	// Pass context: winner, winning tile, Tsumo/Ron, game state
	yakuListResults, han := IdentifyYaku(winner, winningTile, isTsumo, gs)

	if len(yakuListResults) == 0 {
		// This case *should* be prevented by CanDeclareRon/CanDeclareTsumo checking for Yaku.
		// If it happens, it indicates a bug in Yaku logic or checks.
		fmt.Println("!!! CRITICAL ERROR: No Yaku found for a declared winning hand! !!!")
		fmt.Println("This indicates a bug. Ending round abnormally.")
		gs.GamePhase = PhaseRoundEnd // End round, maybe treat as draw?
		// TODO: Decide how to handle this error state. Score reset? Abortive draw?
		return
	}

	yakuNames := []string{}
	for _, r := range yakuListResults {
		yakuNames = append(yakuNames, r.Name)
	}

	// 4. Decompose Hand (for Fu calculation, except Chiitoitsu/Kokushi)
	var decomposition []DecomposedGroup
	var fu int
	isKokushi := false    // Check if Kokushi was the Yaku
	isChiitoitsu := false // Check if Chiitoitsu was the Yaku
	for _, yaku := range yakuListResults {
		if yaku.Name == "Kokushi Musou" { // Or check Yakuman status?
			isKokushi = true
			break
		}
		if yaku.Name == "Chiitoitsu" {
			isChiitoitsu = true
			// break // Don't break, might have Dora etc.
		}
	}

	if isKokushi {
		fmt.Println("Hand is Kokushi Musou - Fu calculation skipped.")
		fu = 0 // Fu is irrelevant for Yakuman scoring calculation (base points handled differently)
	} else if isChiitoitsu {
		fmt.Println("Hand is Chiitoitsu - Using fixed 25 Fu.")
		fu = CalculateFu(winner, nil, winningTile, isTsumo, isMenzen, yakuListResults, gs) // Let CalculateFu handle the fixed 25
	} else {
		// Standard hand - attempt decomposition
		var decompSuccess bool
		fmt.Println("Decomposing standard hand...")
		decomposition, decompSuccess = DecomposeWinningHand(winner, allWinningTiles)
		if !decompSuccess {
			fmt.Println("!!! ERROR: Failed to decompose standard winning hand! Cannot calculate Fu accurately. !!!")
			// Fallback: Use minimum Fu? Or base on Yaku? This indicates an error.
			fmt.Println("Using fallback Fu value: 30")
			fu = 30 // Fallback Fu value
		} else {
			fmt.Println("Calculating Fu based on decomposition...")
			fu = CalculateFu(winner, decomposition, winningTile, isTsumo, isMenzen, yakuListResults, gs)
		}
	}

	// 5. Calculate Payment
	// Yakuman scoring uses fixed base points, not Han/Fu directly in the standard table.
	// CalculatePointPayment needs adjustment for Yakuman.
	payment := CalculatePointPayment(han, fu, winner.SeatWind == "East", isTsumo, gs.Honba, gs.RiichiSticks) // Pass calculated Han & Fu

	fmt.Printf("Yaku: %v (%d Han)\n", yakuNames, han) // Display calculated Han
	if !isKokushi {                                   // Don't display Fu for Yakuman where it's irrelevant
		fmt.Printf("Fu: %d\n", fu)
	}
	fmt.Printf("Score Value: %s\n", payment.Description)

	// 6. Point Transfer
	fmt.Println("--- Point Transfer ---")
	TransferPoints(gs, winner, discarder, isTsumo, payment)
	fmt.Println("--- Scores After Transfer ---")
	for i, p := range gs.Players {
		fmt.Printf("P%d %s: %d\n", i+1, p.Name, p.Score)
	}

	// 7. End Round (or Game)
	gs.GamePhase = PhaseRoundEnd
	// TODO: Add logic here or in main loop to:
	// - Check for game end conditions (player scores < 0, final round reached).
	// - Determine if dealer is retained (Renchan) or rotated (Nagare).
	// - Update Honba counter.
	// - Reset Riichi sticks (done in TransferPoints).
	// - Prepare for next round dealing.
	fmt.Println("\n--- Round Over ---")
	// Consider setting a flag gs.RoundWinnerIndex or similar if needed for next round setup.
}

// PromptDiscard forces the specified player (usually after a call or Kan) to discard.
func PromptDiscard(gs *GameState, player *Player) {

	fmt.Printf("\n--- %s's Turn (Must Discard after Call/Kan) ---\n", player.Name)
	DisplayPlayerState(player) // Show current hand

	// Check for Kan possibilities *before* discarding
	canDeclareKan := false
	kanTarget := Tile{}
	kanType := ""

	uniqueHandTiles := GetUniqueTiles(player.Hand)
	for _, tileToCheck := range uniqueHandTiles {
		kType, kTarget := CanDeclareKanOnHand(player, tileToCheck)
		if kType != "" {
			canDeclareKan = true
			kanType = kType
			kanTarget = kTarget
			break
		}
	}

	if canDeclareKan {
		isHuman := gs.GetPlayerIndex(player) == 0
		kanConfirm := false
		if isHuman {
			kanConfirm = GetPlayerChoice(gs.InputReader, fmt.Sprintf("Declare %s with %s before discarding? (y/n): ", kanType, kanTarget.Name))
		} else {
			fmt.Printf("(%s could declare %s, AI skipping for now)\n", player.Name, kanType)
			kanConfirm = false
		}

		if kanConfirm {
			HandleKanAction(gs, player, kanTarget, kanType)
			return // Kan handler takes over
		}
	}

	// --- Normal Discard after Call/Kan (if no further Kan declared) ---
	// Player needs to discard ONE tile if they have any left.
	// The goal is to reach the correct number of tiles for the meld configuration.
	// e.g., 1 meld -> 10 tiles, 2 melds -> 7 tiles, 3 melds -> 4 tiles, 4 melds -> 1 tile (pair)
	// Reference HandSize correctly for calculating target size based on melds
	numMeldTiles := 0
	numKans := 0 // Kans 'consume' an extra tile conceptually
	for _, m := range player.Melds {
		numMeldTiles += len(m.Tiles)
		if strings.Contains(m.Type, "Kan") {
			numKans++
		}
	}
	// Expected tiles in hand after discard = 13 tiles base - 3 tiles per meld (excluding pair)
	// Or simpler: after discard, total tiles (hand + melds) should be 13
	// Let's just check if the hand is empty instead of complex size calculation
	if len(player.Hand) == 0 { // Check if HandSize is correct
		fmt.Printf("Error: Player %s has no tiles to discard after call/Kan?\n", player.Name)
		// This state indicates an error elsewhere. Abort?
		gs.GamePhase = PhaseRoundEnd // Treat as error/end round?
		return
	}

	// Proceed to get discard choice
	var discardIndex int
	isHuman := gs.GetPlayerIndex(player) == 0
	if isHuman {
		fmt.Println("Choose tile to discard:")
		discardIndex = GetPlayerDiscardChoice(gs.InputReader, player)
	} else {
		// AI discard logic after call/Kan
		fmt.Printf("(%s thinking after call/Kan...)\n", player.Name)
		// Discard the last tile (often the drawn Rinshan tile if Kan occurred).
		discardIndex = len(player.Hand) - 1
	}

	// Validate index before discarding
	if discardIndex >= 0 && discardIndex < len(player.Hand) {
		DiscardTile(gs, player, discardIndex)
	} else {
		fmt.Printf("Error: Invalid discard index %d chosen in PromptDiscard.\n", discardIndex)
		if len(player.Hand) > 0 {
			fmt.Println("Defaulting to discard index 0.")
			DiscardTile(gs, player, 0) // Fallback
		} else {
			// Already checked for empty hand above, but safety first.
			fmt.Println("Cannot discard, hand is empty.")
			gs.GamePhase = PhaseRoundEnd
		}
	}
}
