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
	gameState := NewGameState(playerNames)

	// Deal initial hands
	gameState.DealInitialHands()

	// Game loop
	for gameState.GamePhase != PhaseRoundEnd && gameState.GamePhase != PhaseGameEnd {
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
			break
		}
		// Also, a general check if wall somehow emptied and game didn't end (e.g. after calls)
		if wallNowEmpty && len(gameState.Wall) == 0 && gameState.GamePhase != PhaseRoundEnd && gameState.GamePhase != PhaseGameEnd {
			// This condition implies the wall was empty AFTER the draw, and no win occurred.
			// Redundant if isHaiteiDraw handles it, but a safety.
			fmt.Println("\nWall is empty after player's turn! Round ends in a draw (Ryuukyoku).")
			gameState.GamePhase = PhaseRoundEnd
			break
		}


		// Small delay for non-human players
		if !isHumanPlayer && gameState.GamePhase != PhaseRoundEnd && gameState.GamePhase != PhaseGameEnd {
			time.Sleep(500 * time.Millisecond)
		}

	} // End game loop

	fmt.Println("\n--- Final Game State ---")
	DisplayGameState(gameState)
	// TODO: Display final scores, winner, etc.
	fmt.Println("Game Over.")
}
