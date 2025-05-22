// main.go
package main

import (
	"fmt"
	"math/rand"
	"time"
)

// Constants  <<<<<<<<<<<<<<<<<<<<< HERE THEY ARE
const (
	HandSize        = 13
	DeadWallSize    = 14  // Total tiles in the dead wall
	RinshanTiles    = 4   // Number of replacement tiles for Kans in the dead wall
	MaxRevealedDora = 5   // Max number of Dora indicators (1 initial + 4 Kan) that can be revealed
	TotalTiles      = 136 // 4 * (9*3 + 7)
)

// <<<<<<<<<<<<<<<<<<<<< END OF DEFINITIONS

// ... (Constants remain the same) ...

func main() {
	rand.Seed(time.Now().UnixNano()) // Seed random number generator once
	fmt.Println("Starting Riichi Mahjong Game (Simplified Core Rules)")

	// Initialize game state
	playerNames := []string{"Player 1 (You)", "Player 2", "Player 3", "Player 4"} // Assume Player 1 is human
	gameState := NewGameState(playerNames) // NewGameState sets PhaseDealing initially

	// Main Game Loop - continues as long as the game is not over
	for gameState.GamePhase != PhaseGameEnd {

		// If starting a new round or after round processing, ensure hands are dealt.
		// NewGameState sets PhaseDealing. DealInitialHands sets PhasePlayerTurn.
		// After round processing, if not PhaseGameEnd, we need to ensure we enter turn processing.
		if gameState.GamePhase == PhaseDealing || gameState.GamePhase == PhaseRoundEnd { // PhaseRoundEnd means previous round finished, setup next
			if gameState.GamePhase == PhaseRoundEnd { // If previous round just ended, game not over yet
				// Reset relevant flags or ensure DealInitialHands does it.
				// The round end processing block already handles resetting player/game state
				// and preparing for a new deal by setting up Wall, DeadWall.
				// It also calls RevealInitialDoraIndicator and DealInitialHands.
				// So, if we reached here from RoundEnd processing and it's not GameEnd,
				// DealInitialHands would have been called already by the RoundEnd block.
				// Let's simplify: DealInitialHands should be called if we are in a state
				// that precedes active player turns.
				// The round end block already calls DealInitialHands.
				// So, if we are here and it's PhaseRoundEnd, it means the round processing block
				// decided to continue the game, and hands are ready.
				// DealInitialHands sets GamePhase to PhasePlayerTurn.
				// If it's PhaseDealing (initial game start), call DealInitialHands.
				if gameState.GamePhase == PhaseDealing { // Initial game start
					fmt.Println("Dealing initial hands for the first round...")
					gameState.DealInitialHands()
				}
				// If it was PhaseRoundEnd, the round end processing logic should have called DealInitialHands
				// and set the phase to PhasePlayerTurn if the game is to continue.
				// If it's still PhaseRoundEnd here, something is wrong, or game should end.
				if gameState.GamePhase == PhaseRoundEnd {
					fmt.Println("Error: Game stuck in PhaseRoundEnd after processing. Forcing Game End.");
					gameState.GamePhase = PhaseGameEnd;
					break; // Exit main game loop
				}

			} else if gameState.GamePhase == PhaseDealing { // Only for the very first deal
				gameState.DealInitialHands()
			}
		}


		// Inner loop for a single round's turns
		// This loop runs as long as it's a player's turn and the round/game hasn't ended.
		// Note: The original loop condition was effectively this inner loop.
		// The player turn logic (draw, discard, calls, win checks) will set
		// gameState.GamePhase to PhaseRoundEnd if the round concludes.
		// The outer loop (`for gameState.GamePhase != PhaseGameEnd`) will then catch this,
		// execute the round end processing, and then either terminate or start a new round.

		// The following is the player turn logic, which should only run if GamePhase is PlayerTurn
		if gameState.GamePhase == PhasePlayerTurn {
			currentPlayer := gameState.Players[gameState.CurrentPlayerIndex]
			isHumanPlayer := gameState.CurrentPlayerIndex == 0 // Assuming player 0 is human

		// Reset flags at the start of each player's turn
		gameState.IsChankanOpportunity = false
		gameState.IsRinshanWin = false 
		// IsHouteiDiscard is set specifically when the last tile is drawn and then discarded.
		// It should be reset if the round continues beyond that specific discard without a win.
		// For safety, reset it here. If it's a Houtei discard turn, it will be set to true later.
		gameState.IsHouteiDiscard = false


		DisplayGameState(gameState) // Show state at start of turn

		// --- Kyuushuu Kyuuhai Check (before draw) ---
		// Conditions: Player's first turn, no calls made, within first set of turns.
		// Player.Hand will have 13 tiles at this point.
		if !currentPlayer.HasDrawnFirstTileThisRound && !gameState.AnyCallMadeThisRound && gameState.TurnNumber < len(gameState.Players) {
			if CheckKyuushuuKyuuhai(currentPlayer.Hand) {
				kyuushuuDeclared := false
				if isHumanPlayer {
					fmt.Println("Your hand qualifies for Kyuushuu Kyuuhai (9+ unique terminal/honor tiles).")
					if GetPlayerChoice(gameState.InputReader, "Declare Kyuushuu Kyuuhai for an abortive draw? (y/n): ") {
						kyuushuuDeclared = true
					}
				} else {
					// AI always declares Kyuushuu Kyuuhai if conditions are met
					fmt.Printf("%s's hand qualifies for Kyuushuu Kyuuhai.\n", currentPlayer.Name)
					kyuushuuDeclared = true
				}

				if kyuushuuDeclared {
					fmt.Printf("%s declares Kyuushuu Kyuuhai! Round ends in an abortive draw.\n", currentPlayer.Name)
					gameState.GamePhase = PhaseRoundEnd
					// TODO: Handle Honba increment for abortive draws if applicable by ruleset
					break // End the current round
				}
			}
		}

		// --- Draw Phase ---
		fmt.Printf("\n--- %s's Turn (%s Wind) ---\n", currentPlayer.Name, currentPlayer.SeatWind)

		// Check for Haitei condition (last tile from wall)
		isHaiteiDraw := len(gameState.Wall) == 1 // If 1 tile left, this draw will be Haitei

		drawnTile, wallNowEmpty := gameState.DrawTile() // wallNowEmpty is true if wall was emptied by this draw
		currentPlayer.HasDrawnFirstTileThisRound = true // Player has drawn their first tile

		if wallNowEmpty && !isHaiteiDraw { // Wall became empty unexpectedly (e.g. error in logic)
			fmt.Println("\nWall is empty! Round ends in a draw (Ryuukyoku).")
			gameState.GamePhase = PhaseRoundEnd 
			break
		}
		fmt.Printf("%s draws: %s\n", currentPlayer.Name, drawnTile.Name) 
		if isHaiteiDraw {
			fmt.Println("This is the last tile from the wall (Haitei).")
		}


		// Show hand only *after* draw for human player
		if isHumanPlayer {
			// DisplayPlayerState(currentPlayer) // Called later before discard choice
		}

		// --- Action Phase (Tsumo, Kan on Draw) ---
		canTsumo := CanDeclareTsumo(currentPlayer, gameState)
		possibleKanType, kanTargetTile := CanDeclareKanOnDraw(currentPlayer, drawnTile)

		actionTaken := false
		if canTsumo {
			if isHumanPlayer {
				// Display hand *before* Tsumo choice
				DisplayPlayerState(currentPlayer)
				if GetPlayerChoice(gameState.InputReader, "Declare TSUMO? (y/n): ") {
					HandleWin(gameState, currentPlayer, drawnTile, true)
					actionTaken = true
				}
			} else {
				fmt.Printf("%s declares TSUMO!\n", currentPlayer.Name)
				HandleWin(gameState, currentPlayer, drawnTile, true)
				actionTaken = true
			}
		}

		// Check for Kan only if Tsumo wasn't declared
		if !actionTaken && possibleKanType != "" {
			if isHumanPlayer {
				// Display hand *before* Kan choice
				DisplayPlayerState(currentPlayer)
				if GetPlayerChoice(gameState.InputReader, fmt.Sprintf("Declare %s with %s? (y/n): ", possibleKanType, kanTargetTile.Name)) {
					HandleKanAction(gameState, currentPlayer, kanTargetTile, possibleKanType)
					actionTaken = true // Kan action handles the next step
				}
			} else {
				// Basic AI: Always Kan if possible? (Maybe add some logic later)
				// fmt.Printf("(%s could declare %s, skipping for AI)\n", currentPlayer.Name, possibleKanType) // AI skips Kan for simplicity now
				// AI Decides to Kan (example)
				fmt.Printf("%s declares %s!\n", currentPlayer.Name, possibleKanType)
				HandleKanAction(gameState, currentPlayer, kanTargetTile, possibleKanType) // Pass gs
				actionTaken = true
			}
		}

		if actionTaken {
			// If Tsumo or Kan occurred, the turn structure changes or ends.
			if gameState.GamePhase == PhaseRoundEnd || gameState.GamePhase == PhaseGameEnd {
				break
			}
			continue // Kan handler will manage the next discard prompt or turn ended with Tsumo
		}

		// --- Discard Phase ---
		// Check Riichi possibilities FIRST
		canRiichi, riichiOptions := CanDeclareRiichi(currentPlayer, gameState) // Gets bool and options
		discardIndex := -1                                                     // Initialize discard index

		riichiDeclaredSuccessfully := false // Flag to track if Riichi was handled

		if isHumanPlayer {
			// Display hand state before any discard choice (Riichi or normal)
			DisplayPlayerState(currentPlayer)

			if canRiichi {
				// Present Riichi options
				chosenOptionIndex, choiceMade := GetPlayerRiichiChoice(gameState.InputReader, riichiOptions)

				if choiceMade {
					// Player chose a Riichi option
					selectedOption := riichiOptions[chosenOptionIndex]
					discardIndex = selectedOption.DiscardIndex // Get the index to discard

					// Attempt Riichi declaration
					if HandleRiichiAction(gameState, currentPlayer, discardIndex) {
						riichiDeclaredSuccessfully = true // Riichi was declared and discard happened
						// HandleRiichiAction calls DiscardTile, which handles next steps
						// Check if game ended due to Ron on Riichi discard
						if gameState.GamePhase == PhaseRoundEnd || gameState.GamePhase == PhaseGameEnd {
							break
						}
						// continue // Turn logic handled by DiscardTile called within HandleRiichiAction
					} else {
						// Riichi failed validation within HandleRiichiAction (e.g., chosen discard was wrong - safety check)
						fmt.Println("Riichi declaration failed validation. Proceeding with normal discard.")
						// Fall back to normal discard choice
						discardIndex = GetPlayerDiscardChoice(gameState.InputReader, currentPlayer)
					}
				} else {
					// Player cancelled Riichi choice
					fmt.Println("Proceeding with normal discard.")
					discardIndex = GetPlayerDiscardChoice(gameState.InputReader, currentPlayer)
				}
			} else {
				// Cannot Riichi, just get normal discard
				discardIndex = GetPlayerDiscardChoice(gameState.InputReader, currentPlayer)
			}
		} else { // AI Logic
			fmt.Printf("(%s thinking...)\n", currentPlayer.Name)
			if currentPlayer.IsRiichi {
				// AI is already in Riichi - must discard drawn tile unless Kan
				// Kan was handled earlier. Find index of drawn tile.
				// Assuming drawn tile is last after sort (may need better tracking)
				if len(currentPlayer.Hand) == HandSize+1 {
					discardIndex = len(currentPlayer.Hand) - 1
				} else {
					fmt.Println("Error: AI in Riichi but hand size isn't 14?")
					discardIndex = 0 // Fallback
				}
			} else { // AI not in Riichi
				if canRiichi { // AI *could* declare Riichi
					// Basic AI: Always Riichi if possible? Choose first option?
					fmt.Printf("(%s can Riichi, AI chooses to Riichi!)\n", currentPlayer.Name)
					chosenOption := riichiOptions[0] // AI picks first option
					discardIndex = chosenOption.DiscardIndex
					if HandleRiichiAction(gameState, currentPlayer, discardIndex) {
						riichiDeclaredSuccessfully = true
						if gameState.GamePhase == PhaseRoundEnd || gameState.GamePhase == PhaseGameEnd {
							break
						}
						// continue // Handled by DiscardTile within HandleRiichiAction
					} else {
						fmt.Println("Error: AI Riichi failed validation?")
						discardIndex = len(currentPlayer.Hand) - 1 // Fallback: discard drawn
					}
				} else {
					// AI cannot Riichi, basic discard: drawn tile
					if len(currentPlayer.Hand) == HandSize+1 {
						discardIndex = len(currentPlayer.Hand) - 1 // Index of drawn tile after sort?
					} else {
						fmt.Println("Error: AI not in Riichi but hand size isn't 14?")
						discardIndex = 0 // Fallback
					}
				}
			}
			// Safety check for AI discard index
			if !riichiDeclaredSuccessfully && (discardIndex < 0 || discardIndex >= len(currentPlayer.Hand)) {
				fmt.Printf("Error: AI calculated invalid discard index %d (Hand Size %d). Defaulting to 0.\n", discardIndex, len(currentPlayer.Hand))
				if len(currentPlayer.Hand) > 0 {
					discardIndex = 0
				} else {
					discardIndex = -1 /* No tiles? Error */
				}
			}
		} // End AI Logic

		// Perform the discard *only if* it wasn't handled by Riichi declaration
		if !riichiDeclaredSuccessfully && discardIndex != -1 {
			// If this discard is after the Haitei tile was drawn, it's a Houtei discard.
			if isHaiteiDraw { // isHaiteiDraw means the drawnTile was the last one.
				gameState.IsHouteiDiscard = true
				fmt.Println("This discard is Houtei (last discard of the game).")
			}

			// DiscardTile handles calls, Furiten update, and turn advancement
			_, gameShouldEnd := DiscardTile(gameState, currentPlayer, discardIndex)
			currentPlayer.HasMadeFirstDiscardThisRound = true

			// Update IsFirstGoAround after the first discard of each player in the initial set of turns
			// A simple proxy: if TurnNumber (which increments in DiscardTile) reaches number of players -1,
			// it means all players have had one turn (0, 1, 2, 3 for 4 players).
			// This should happen *after* the current player's discard is fully processed by DiscardTile.
			// TurnNumber is 0-indexed for the round's turns.
			// If TurnNumber is 3 (4th discard of the round), first go-around is done.
			if gameState.IsFirstGoAround && gameState.TurnNumber >= (len(gameState.Players)-1) {
				// This logic might be too simple if calls interrupt the first go-around.
				// AnyCallMadeThisRound will also set IsFirstGoAround to false more reliably.
				// However, for a natural first go-around without calls:
				// gameState.IsFirstGoAround = false; // This is one way to mark end of natural first round.
				// Let's rely on AnyCallMadeThisRound and a more explicit check for Renhou/Chihou context.
				// The Yaku checks use IsFirstGoAround, which is primarily falsified by calls.
				// The HasMadeFirstDiscardThisRound flag on player is more direct for Renhou/Chihou.
			}
			
			if gameShouldEnd {
				// Ron occurred on the normal discard (could be Houtei)
				break // End the main loop
			}
		} else if !riichiDeclaredSuccessfully && discardIndex == -1 {
			fmt.Println("Error: No valid discard index determined.")
			gameState.GamePhase = PhaseRoundEnd
			break
		}

		// Check for end conditions again
		// If it was a Haitei draw and no win occurred on Tsumo or subsequent Houtei discard, it's Ryuukyoku.
		if isHaiteiDraw && gameState.GamePhase != PhaseRoundEnd && gameState.GamePhase != PhaseGameEnd {
			// No one won on Haitei Tsumo or Houtei Ron
			fmt.Println("\nLast tile drawn and discarded with no win. Round ends in a draw (Ryuukyoku).")
			gameState.GamePhase = PhaseRoundEnd
			gameState.RoundWinner = nil // Ensure draw state for Honba/Renchan logic
			// Nagashi Mangan Check for Haitei Ryuukyoku
			for _, p := range gameState.Players {
				if isNagashi, nagashiName, _ := checkNagashiMangan(p, gameState); isNagashi {
					fmt.Printf("!!! %s achieves %s! (Further scoring TBD) !!!\n", p.Name, nagashiName)
					// TODO: Handle Nagashi Mangan payment logic here or in a dedicated scoring phase.
					// Nagashi Mangan usually means the player gets Mangan points from others.
					// This might override normal Tenpai/Notenpai payments for this player.
				}
			}
			break
		}
		// Also, a general check if wall somehow emptied and game didn't end (e.g. after calls)
		if wallNowEmpty && len(gameState.Wall) == 0 && gameState.GamePhase != PhaseRoundEnd && gameState.GamePhase != PhaseGameEnd {
			// This condition implies the wall was empty AFTER the draw, and no win occurred.
			// Redundant if isHaiteiDraw handles it, but a safety.
			fmt.Println("\nWall is empty after player's turn! Round ends in a draw (Ryuukyoku).")
			gameState.GamePhase = PhaseRoundEnd
			gameState.RoundWinner = nil // Ensure draw state for Honba/Renchan logic
			// Nagashi Mangan Check for general Ryuukyoku
			for _, p := range gameState.Players {
				if isNagashi, nagashiName, _ := checkNagashiMangan(p, gameState); isNagashi {
					fmt.Printf("!!! %s achieves %s! (Further scoring TBD) !!!\n", p.Name, nagashiName)
				}
			}
			break
		}


		// Small delay for non-human players
		if !isHumanPlayer && gameState.GamePhase != PhaseRoundEnd && gameState.GamePhase != PhaseGameEnd {
			time.Sleep(500 * time.Millisecond)
		}

	} // End of inner loop for player turns (terminates when PhaseRoundEnd or PhaseGameEnd)

	// --- Round End Processing ---
	if gameState.GamePhase == PhaseRoundEnd {
		fmt.Println("\n--- Round End Processing ---")

		// Check for game end conditions (e.g. player score < 0, or specific round limits like South 4)
		gameShouldActuallyEnd := false
		for _, p := range gameState.Players {
			if p.Score < 0 {
				fmt.Printf("Player %s has a negative score (%d). Game Over!\n", p.Name, p.Score)
				gameShouldActuallyEnd = true
				break
			}
		}
		// TODO: Add more game end conditions (e.g., end of South 4, if not dealer and not top, etc.)
		// Example: if gameState.PrevalentWind == "South" && gameState.RoundNumber > 4 { gameShouldActuallyEnd = true }

		if gameShouldActuallyEnd {
			gameState.GamePhase = PhaseGameEnd
		} else {
			fmt.Println("Preparing for next round...")

			dealerAtRoundStart := gameState.CurrentPlayerIndex // Who was dealer for the round that just ended
			isDealerWin := false
			isDraw := gameState.RoundWinner == nil

			if !isDraw && gameState.Players[dealerAtRoundStart] == gameState.RoundWinner {
				isDealerWin = true
			}

			// Renchan: Dealer wins or any draw (including Kyuushuu, Ryuukyoku)
			if isDealerWin || isDraw {
				fmt.Println("Dealer is retained (Renchan) or round was a draw.")
				gameState.Honba++
				// CurrentPlayerIndex for next round remains dealerAtRoundStart
				gameState.CurrentPlayerIndex = dealerAtRoundStart
			} else { // Non-dealer win: Dealer changes, Honba resets
				fmt.Println("Non-dealer win. Dealer changes.")
				gameState.Honba = 0 
				gameState.CurrentPlayerIndex = (dealerAtRoundStart + 1) % len(gameState.Players)
				gameState.RoundNumber++ // Increment round number 
				
				// Update SeatWinds for all players based on new dealer
				newDealerForNextRound := gameState.CurrentPlayerIndex
				winds := []string{"East", "South", "West", "North"}
				fmt.Printf("New dealer for the next round will be %s.\n", gameState.Players[newDealerForNextRound].Name)
				for i := 0; i < len(gameState.Players); i++ {
					seatWindIndex := (i - newDealerForNextRound + len(gameState.Players)) % len(gameState.Players)
					gameState.Players[i].SeatWind = winds[seatWindIndex]
				}
				fmt.Printf("All players' seat winds updated. New dealer %s is %s Wind.\n", 
					gameState.Players[newDealerForNextRound].Name, gameState.Players[newDealerForNextRound].SeatWind)
			}

			fmt.Printf("Honba counters for next round: %d\n", gameState.Honba)
			fmt.Printf("Riichi sticks on table (carried over): %d\n", gameState.RiichiSticks)

			// Reset round-specific player flags and states
			for _, player := range gameState.Players {
				player.Hand = []Tile{} // Clear hand (will be replaced by new deal)
				player.IsRiichi = false
				player.RiichiTurn = -1
				player.IsIppatsu = false
				player.DeclaredDoubleRiichi = false
				player.HasMadeFirstDiscardThisRound = false
				player.HasDrawnFirstTileThisRound = false
				player.Discards = []Tile{} 
				player.Melds = []Meld{}    // Melds are cleared between rounds
			}

			// Reset game state flags for the new round
			gameState.LastDiscard = nil
			gameState.DoraIndicators = []Tile{}
			gameState.UraDoraIndicators = []Tile{}
			gameState.AnyCallMadeThisRound = false
			gameState.IsFirstGoAround = true
			gameState.TurnNumber = 0
			gameState.DiscardPile = []Tile{} 
			gameState.RoundWinner = nil      // Reset round winner tracker for the new round

			// The main game loop will handle starting the new round:
			// It will call DealInitialHands() which shuffles, deals, reveals Dora, and sets GamePhase to PlayerTurn.
			// For this to work, GamePhase must not be PhaseGameEnd.
			// If we are continuing, DealInitialHands will set it up.
			// No need to explicitly set PhaseDealing here if the outer loop handles it.
			fmt.Println("\n--- Setup for New Round Complete ---")
		}
	} // End of Round End Processing

			// ... (The entire player turn logic from the original main.go from line 36 down to the original end of the player turn loop)
			// This includes: Reset flags, DisplayGameState, Kyuushuu Check, Draw Phase, Action Phase, Discard Phase, End Checks
			// Ensure all `break` statements within this player turn logic correctly terminate this phase of play
			// and lead to the Round End Processing block.
			// If any action sets gameState.GamePhase = PhaseRoundEnd, this inner block of logic will complete,
			// and the outer loop (`for gameState.GamePhase != PhaseGameEnd`) will then execute the
			// Round End Processing block.

			// Reset flags at the start of each player's turn
			gameState.IsChankanOpportunity = false
			gameState.IsRinshanWin = false
			gameState.IsHouteiDiscard = false

			DisplayGameState(gameState) // Show state at start of turn

			// --- Kyuushuu Kyuuhai Check (before draw) ---
			if !currentPlayer.HasDrawnFirstTileThisRound && !gameState.AnyCallMadeThisRound && gameState.TurnNumber < len(gameState.Players) {
				if CheckKyuushuuKyuuhai(currentPlayer.Hand) {
					kyuushuuDeclared := false
					if isHumanPlayer {
						fmt.Println("Your hand qualifies for Kyuushuu Kyuuhai (9+ unique terminal/honor tiles).")
						if GetPlayerChoice(gameState.InputReader, "Declare Kyuushuu Kyuuhai for an abortive draw? (y/n): ") {
							kyuushuuDeclared = true
						}
					} else {
						fmt.Printf("%s's hand qualifies for Kyuushuu Kyuuhai.\n", currentPlayer.Name)
						kyuushuuDeclared = true // AI always declares
					}

					if kyuushuuDeclared {
						fmt.Printf("%s declares Kyuushuu Kyuuhai! Round ends in an abortive draw.\n", currentPlayer.Name)
						gameState.GamePhase = PhaseRoundEnd
						gameState.RoundWinner = nil // Ensure it's a draw
						// Honba increment for abortive draw handled in round end processing block
					}
				}
			}
			if gameState.GamePhase == PhaseRoundEnd { continue } // Skip to next iteration of outer loop if round ended

			// --- Draw Phase ---
			fmt.Printf("\n--- %s's Turn (%s Wind) ---\n", currentPlayer.Name, currentPlayer.SeatWind)
			isHaiteiDraw := len(gameState.Wall) == 1
			drawnTile, wallNowEmpty := gameState.DrawTile()
			currentPlayer.HasDrawnFirstTileThisRound = true

			if wallNowEmpty && !isHaiteiDraw {
				fmt.Println("\nWall is empty! Round ends in a draw (Ryuukyoku).")
				gameState.GamePhase = PhaseRoundEnd
				gameState.RoundWinner = nil // Ensure it's a draw
			}
			if gameState.GamePhase == PhaseRoundEnd { continue }

			fmt.Printf("%s draws: %s\n", currentPlayer.Name, drawnTile.Name)
			if isHaiteiDraw {
				fmt.Println("This is the last tile from the wall (Haitei).")
			}

			// --- Action Phase (Tsumo, Kan on Draw) ---
			canTsumo := CanDeclareTsumo(currentPlayer, gameState)
			possibleKanType, kanTargetTile := CanDeclareKanOnDraw(currentPlayer, drawnTile)
			actionTaken := false

			if canTsumo {
				if isHumanPlayer {
					DisplayPlayerState(currentPlayer)
					if GetPlayerChoice(gameState.InputReader, "Declare TSUMO? (y/n): ") {
						HandleWin(gameState, currentPlayer, drawnTile, true) // Sets GamePhase to RoundEnd
						actionTaken = true
					}
				} else {
					fmt.Printf("%s declares TSUMO!\n", currentPlayer.Name)
					HandleWin(gameState, currentPlayer, drawnTile, true) // Sets GamePhase to RoundEnd
					actionTaken = true
				}
			}

			if !actionTaken && possibleKanType != "" {
				if isHumanPlayer {
					DisplayPlayerState(currentPlayer)
					if GetPlayerChoice(gameState.InputReader, fmt.Sprintf("Declare %s with %s? (y/n): ", possibleKanType, kanTargetTile.Name)) {
						HandleKanAction(gameState, currentPlayer, kanTargetTile, possibleKanType)
						actionTaken = true
					}
				} else {
					fmt.Printf("%s declares %s!\n", currentPlayer.Name, possibleKanType)
					HandleKanAction(gameState, currentPlayer, kanTargetTile, possibleKanType)
					actionTaken = true
				}
			}

			if actionTaken { // If Tsumo or Kan occurred
				if gameState.GamePhase == PhaseRoundEnd || gameState.GamePhase == PhaseGameEnd {
					continue // To outer loop for round/game end processing
				}
				// If Kan, turn might continue for same player (discard after Kan)
				// The HandleKanAction and PromptDiscard logic handles this by recalling PromptDiscard
				// or if win, sets PhaseRoundEnd.
				// If it was just a Kan and no win, the turn continues for this player, so `continue` the inner player turn logic.
				// This needs to be handled carefully. If HandleKanAction leads to a discard prompt,
				// that prompt will then call DiscardTile, which can end the round.
				// The `continue` here means "skip the rest of *this specific turn's code block* and re-evaluate player turn".
				// This is generally correct if HandleKanAction changes player or requires new input.
				// For simplicity, let's assume HandleKanAction itself correctly manages the flow,
				// including setting PhaseRoundEnd if a win occurs during Kan (Rinshan).
				// If just a Kan, the same player discards, so the player turn logic should effectively repeat for discard.
				// This implies PromptDiscard will be called by HandleKanAction.
				// So, `continue` to the next iteration of the *player turn processing part of the code* is not what we want.
				// We want the current player to continue their turn (discard).
				// The current structure where PromptDiscard is called by HandleKanAction handles this.
				// So if actionTaken is true, and game hasn't ended, the turn proceeds.
				// No special `continue` for the player turn loop is needed here if Kan path manages its own discards.
			}
			if gameState.GamePhase == PhaseRoundEnd { continue }


			// --- Discard Phase --- (Only if no win/Kan ended turn above)
			if !actionTaken {
				canRiichi, riichiOptions := CanDeclareRiichi(currentPlayer, gameState)
				discardIndex := -1
				riichiDeclaredSuccessfully := false

				if isHumanPlayer {
					DisplayPlayerState(currentPlayer)
					if canRiichi {
						chosenOptionIndex, choiceMade := GetPlayerRiichiChoice(gameState.InputReader, riichiOptions)
						if choiceMade {
							selectedOption := riichiOptions[chosenOptionIndex]
							discardIndex = selectedOption.DiscardIndex
							if HandleRiichiAction(gameState, currentPlayer, discardIndex) { // Calls DiscardTile
								riichiDeclaredSuccessfully = true
							} else { // Riichi failed validation
								discardIndex = GetPlayerDiscardChoice(gameState.InputReader, currentPlayer)
							}
						} else { // Player cancelled Riichi
							discardIndex = GetPlayerDiscardChoice(gameState.InputReader, currentPlayer)
						}
					} else { // Cannot Riichi
						discardIndex = GetPlayerDiscardChoice(gameState.InputReader, currentPlayer)
					}
				} else { // AI Logic
					fmt.Printf("(%s thinking...)\n", currentPlayer.Name)
					if currentPlayer.IsRiichi { // AI already in Riichi
						if len(currentPlayer.Hand) == HandSize+1 { discardIndex = len(currentPlayer.Hand) - 1 } else { discardIndex = 0 } // Should be drawn tile
					} else { // AI not in Riichi
						if canRiichi {
							chosenOption := riichiOptions[0] // AI picks first Riichi option
							discardIndex = chosenOption.DiscardIndex
							if HandleRiichiAction(gameState, currentPlayer, discardIndex) {
								riichiDeclaredSuccessfully = true
							} else { // AI Riichi failed validation?
								discardIndex = len(currentPlayer.Hand) - 1 // Fallback: discard drawn
							}
						} else { // AI cannot Riichi
							if len(currentPlayer.Hand) == HandSize+1 { discardIndex = len(currentPlayer.Hand) - 1 } else { discardIndex = 0 } // Discard drawn
						}
					}
					// Safety for AI discard index
					if !riichiDeclaredSuccessfully && (discardIndex < 0 || discardIndex >= len(currentPlayer.Hand)) {
						if len(currentPlayer.Hand) > 0 { discardIndex = 0 } else { discardIndex = -1 }
					}
				}

				if !riichiDeclaredSuccessfully && discardIndex != -1 {
					if isHaiteiDraw { // Mark if this is a Houtei discard
						gameState.IsHouteiDiscard = true
						fmt.Println("This discard is Houtei (last discard of the game).")
					}
					_, _ = DiscardTile(gameState, currentPlayer, discardIndex) // DiscardTile can set PhaseRoundEnd on Ron
					currentPlayer.HasMadeFirstDiscardThisRound = true
				} else if !riichiDeclaredSuccessfully && discardIndex == -1 {
					fmt.Println("Error: No valid discard index determined.")
					gameState.GamePhase = PhaseRoundEnd; gameState.RoundWinner = nil // Error, treat as draw
				}
			} // End of discard phase logic block
			if gameState.GamePhase == PhaseRoundEnd { continue }


			// Check for end conditions again (Haitei/Wall Empty Ryuukyoku)
			if isHaiteiDraw && gameState.GamePhase != PhaseRoundEnd && gameState.GamePhase != PhaseGameEnd {
				fmt.Println("\nLast tile drawn and discarded with no win. Round ends in a draw (Ryuukyoku).")
				gameState.GamePhase = PhaseRoundEnd; gameState.RoundWinner = nil
			}
			if wallNowEmpty && len(gameState.Wall) == 0 && gameState.GamePhase != PhaseRoundEnd && gameState.GamePhase != PhaseGameEnd {
				fmt.Println("\nWall is empty after player's turn! Round ends in a draw (Ryuukyoku).")
				gameState.GamePhase = PhaseRoundEnd; gameState.RoundWinner = nil
			}
			if gameState.GamePhase == PhaseRoundEnd { continue }


			// Small delay for non-human players
			if !isHumanPlayer && gameState.GamePhase != PhaseRoundEnd && gameState.GamePhase != PhaseGameEnd {
				time.Sleep(100 * time.Millisecond) // Shorter delay
			}

			// If no call/win ended the turn, DiscardTile would have called gs.NextPlayer()
			// The turn processing for the current player is over. Loop will check phase and continue.
		} // End of `if gameState.GamePhase == PhasePlayerTurn`

		// --- Round End Processing (moved from the previous diff, executed after each turn loop or if round ends prematurely) ---
		if gameState.GamePhase == PhaseRoundEnd {
			fmt.Println("\n--- Round End Processing ---")

			gameShouldActuallyEnd := false
			for _, p := range gameState.Players {
				if p.Score < 0 {
					fmt.Printf("Player %s has a negative score (%d). Game Over!\n", p.Name, p.Score)
					gameShouldActuallyEnd = true; break
				}
			}
			// TODO: Add more game end conditions (e.g., end of South 4)

			if gameShouldActuallyEnd {
				gameState.GamePhase = PhaseGameEnd
			} else {
				fmt.Println("Preparing for next round...")
				dealerAtRoundStart := gameState.CurrentPlayerIndex 
				isDealerWin := false
				isDraw := gameState.RoundWinner == nil
				if !isDraw && gameState.Players[dealerAtRoundStart] == gameState.RoundWinner { isDealerWin = true }

				if isDealerWin || isDraw { // Renchan
					fmt.Println("Dealer is retained (Renchan) or round was a draw.")
					gameState.Honba++
					gameState.CurrentPlayerIndex = dealerAtRoundStart
				} else { // Dealer changes
					fmt.Println("Non-dealer win. Dealer changes.")
					gameState.Honba = 0
					gameState.CurrentPlayerIndex = (dealerAtRoundStart + 1) % len(gameState.Players)
					gameState.RoundNumber++
					newDealerForNextRound := gameState.CurrentPlayerIndex
					winds := []string{"East", "South", "West", "North"}
					fmt.Printf("New dealer for the next round will be %s.\n", gameState.Players[newDealerForNextRound].Name)
					for i := 0; i < len(gameState.Players); i++ {
						seatWindIndex := (i - newDealerForNextRound + len(gameState.Players)) % len(gameState.Players)
						gameState.Players[i].SeatWind = winds[seatWindIndex]
					}
					fmt.Printf("All players' seat winds updated. New dealer %s is %s Wind.\n", gameState.Players[newDealerForNextRound].Name, gameState.Players[newDealerForNextRound].SeatWind)
				}
				fmt.Printf("Honba counters for next round: %d\n", gameState.Honba)
				fmt.Printf("Riichi sticks on table (carried over): %d\n", gameState.RiichiSticks)

				for _, player := range gameState.Players {
					player.Hand = []Tile{}; player.IsRiichi = false; player.RiichiTurn = -1; player.IsIppatsu = false
					player.DeclaredDoubleRiichi = false; player.HasMadeFirstDiscardThisRound = false
					player.HasDrawnFirstTileThisRound = false; player.Discards = []Tile{}; player.Melds = []Meld{}
				}
				gameState.LastDiscard = nil; gameState.DoraIndicators = []Tile{}; gameState.UraDoraIndicators = []Tile{}
				gameState.AnyCallMadeThisRound = false; gameState.IsFirstGoAround = true; gameState.TurnNumber = 0
				gameState.DiscardPile = []Tile{}; gameState.RoundWinner = nil

				fmt.Println("Dealing new hands for the next round...")
				newDeck := GenerateDeck()
				gameState.Wall = newDeck[:TotalTiles-DeadWallSize]
				gameState.DeadWall = newDeck[TotalTiles-DeadWallSize:]
				gameState.RevealInitialDoraIndicator()
				gameState.DealInitialHands() // This sets GamePhase to PhasePlayerTurn

				fmt.Println("\n--- New Round Starting ---")
			}
		} // End of Round End Processing block
	} // End Main Game Loop (`for gameState.GamePhase != PhaseGameEnd`)

	// --- Final Game Outcome Display (if game truly ended) ---
	if gameState.GamePhase == PhaseGameEnd {
		fmt.Println("\n--- Final Game State ---")
		DisplayGameState(gameState)
		// TODO: Display final scores, overall winner, etc.
		fmt.Println("Game Over.")
	}
} // End main function
