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

		DisplayGameState(gameState) // Show state at start of turn

		// --- Draw Phase ---
		fmt.Printf("\n--- %s's Turn (%s Wind) ---\n", currentPlayer.Name, currentPlayer.SeatWind)
		drawnTile, wallEmpty := gameState.DrawTile()
		if wallEmpty {
			fmt.Println("\nWall is empty! Round ends in a draw (Ryuukyoku).")
			// *** Handle Ryuukyoku scoring/dealer retention ***
			gameState.GamePhase = PhaseRoundEnd // Simple end for now
			break
		}
		fmt.Printf("%s draws: %s\n", currentPlayer.Name, drawnTile.Name) // Show drawn tile name
		// Show hand only *after* draw for human player
		if isHumanPlayer {
			// DisplayPlayerState shows hand AFTER draw, before action/discard
			// DisplayPlayerState(currentPlayer)
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
			// DiscardTile handles calls, Furiten update, and turn advancement
			_, gameShouldEnd := DiscardTile(gameState, currentPlayer, discardIndex)
			if gameShouldEnd {
				// Ron occurred on the normal discard
				break // End the main loop
			}
		} else if !riichiDeclaredSuccessfully && discardIndex == -1 {
			fmt.Println("Error: No valid discard index determined.")
			// Handle error state? Maybe end round as draw?
			gameState.GamePhase = PhaseRoundEnd
			break
		}

		// Check for end conditions again (e.g., wall empty after calls/discard)
		if len(gameState.Wall) == 0 && gameState.GamePhase != PhaseRoundEnd && gameState.GamePhase != PhaseGameEnd {
			fmt.Println("\nWall is empty! Round ends in a draw (Ryuukyoku).")
			// *** Handle Ryuukyoku scoring/dealer retention ***
			gameState.GamePhase = PhaseRoundEnd // Simple end for now
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
