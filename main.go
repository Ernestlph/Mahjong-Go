package main

import (
	"fmt"
	"math/rand"
	"time"
)

// Constants
const (
	HandSize        = 13
	DeadWallSize    = 14  // Total tiles in the dead wall
	RinshanTiles    = 4   // Number of replacement tiles for Kans in the dead wall
	MaxRevealedDora = 5   // Max number of Dora indicators (1 initial + 4 Kan) that can be revealed
	TotalTiles      = 136 // 4 * (9*3 + 7)
)

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
		fmt.Printf("%s draws: %s\n", currentPlayer.Name, drawnTile.Name)
		if isHumanPlayer {
			DisplayPlayerState(currentPlayer) // Show human player their hand + draw
		}

		// --- Action Phase (Tsumo, Kan) ---
		canTsumo := CanDeclareTsumo(currentPlayer, gameState) // Pass gameState if needed by Yaku checks later
		// Check for Kan involving the *drawn* tile (Ankan or Shouminkan)
		possibleKanType, kanTargetTile := CanDeclareKanOnDraw(currentPlayer, drawnTile) // Check specific Kan types on draw

		actionTaken := false
		if canTsumo {
			if isHumanPlayer {
				if GetPlayerChoice(gameState.InputReader, "Declare TSUMO? (y/n): ") {
					HandleWin(gameState, currentPlayer, drawnTile, true) // Pass gs
					actionTaken = true
					// HandleWin should set GamePhase to RoundEnd/GameEnd
				}
			} else {
				// Basic AI: Always Tsumo if possible
				fmt.Printf("%s declares TSUMO!\n", currentPlayer.Name)
				HandleWin(gameState, currentPlayer, drawnTile, true) // Pass gs
				actionTaken = true
			}
		}

		// Check for Kan only if Tsumo wasn't declared
		if !actionTaken && possibleKanType != "" {
			if isHumanPlayer {
				if GetPlayerChoice(gameState.InputReader, fmt.Sprintf("Declare %s with %s? (y/n): ", possibleKanType, kanTargetTile.Name)) {
					HandleKanAction(gameState, currentPlayer, kanTargetTile, possibleKanType) // Pass gs
					actionTaken = true                                                        // Kan action handles the next step (Rinshan draw + discard prompt)
				}
			} else {
				// Basic AI: Always Kan if possible? (Maybe add some logic later)
				// fmt.Printf("%s declares %s!\n", currentPlayer.Name, possibleKanType)
				// HandleKanAction(gameState, currentPlayer, kanTargetTile, possibleKanType) // Pass gs
				// actionTaken = true
				fmt.Printf("(%s could declare %s, skipping for AI)\n", currentPlayer.Name, possibleKanType) // AI skips Kan for simplicity now
			}
		}

		if actionTaken {
			// If Tsumo or Kan occurred, the turn structure changes or ends.
			// Kan handler calls PromptDiscard. Tsumo ends the round.
			if gameState.GamePhase == PhaseRoundEnd || gameState.GamePhase == PhaseGameEnd {
				break
			} // End loop if win occurred
			continue // Kan handler will manage the next discard prompt
		}

		// --- Discard Phase ---
		canRiichi := CanDeclareRiichi(currentPlayer, gameState) // Pass gs
		discardIndex := -1

		if isHumanPlayer {
			DisplayPlayerState(currentPlayer) // Ensure hand is visible before discard choice

			// Riichi Option
			if canRiichi {
				if GetPlayerChoice(gameState.InputReader, "Declare RIICHI? (y/n): ") {
					// Player must choose which tile to discard for Riichi
					fmt.Println("Choose discard for Riichi:")
					discardIndex = GetPlayerDiscardChoice(gameState.InputReader, currentPlayer)
					if HandleRiichiAction(gameState, currentPlayer, discardIndex) { // Pass gs
						// Riichi action handled discard and checks, continue to next player turn potentially
						if gameState.GamePhase == PhaseRoundEnd || gameState.GamePhase == PhaseGameEnd {
							break
						} // Ron on Riichi discard
						continue // Skip normal discard, Riichi handler did it
					} else {
						// Riichi failed (e.g., invalid choice), fall back to normal discard
						fmt.Println("Riichi declaration failed or canceled.")
						discardIndex = GetPlayerDiscardChoice(gameState.InputReader, currentPlayer)
					}

				} else {
					// Player chose not to Riichi
					discardIndex = GetPlayerDiscardChoice(gameState.InputReader, currentPlayer)
				}
			} else {
				// Cannot Riichi, just get normal discard
				discardIndex = GetPlayerDiscardChoice(gameState.InputReader, currentPlayer)
			}
		} else {
			// Basic AI Discard Logic
			fmt.Printf("(%s thinking...)\n", currentPlayer.Name)
			if currentPlayer.IsRiichi {
				// Must discard drawn unless Kan (Kan handled above)
				// Find the index of the drawn tile (which should be the last one after sorting)
				if len(currentPlayer.Hand) > HandSize { // Should have 14 tiles before discard
					discardIndex = len(currentPlayer.Hand) - 1
				} else {
					fmt.Println("Error: AI in Riichi but hand size isn't 14?")
					discardIndex = 0 // Fallback
				}
			} else {
				// Very basic: discard the drawn tile
				if len(currentPlayer.Hand) > HandSize {
					discardIndex = len(currentPlayer.Hand) - 1 // Index of drawn tile after sort
				} else {
					fmt.Println("Error: AI not in Riichi but hand size isn't 14?")
					discardIndex = 0 // Fallback
				}
				// Improvement: Discard loose honors/terminals first? Needs state analysis.
			}
			if discardIndex < 0 || discardIndex >= len(currentPlayer.Hand) { // Safety check
				fmt.Printf("Error: AI calculated invalid discard index %d. Defaulting to 0.\n", discardIndex)
				discardIndex = 0
			}
		}

		// Perform the discard (if not handled by Riichi/Kan/Tsumo)
		if discardIndex != -1 {
			_, gameShouldEnd := DiscardTile(gameState, currentPlayer, discardIndex) // Pass gs
			if gameShouldEnd {
				// Ron occurred
				break // End the main loop
			}
			// If DiscardTile resulted in a call (Pon/Chi/Kan), CurrentPlayerIndex was updated
			// If no call, CurrentPlayerIndex was advanced by NextPlayer() inside DiscardTile's path
		}

		// Check for end conditions again (e.g., wall empty after calls)
		if len(gameState.Wall) == 0 && gameState.GamePhase != PhaseRoundEnd {
			fmt.Println("\nWall is empty after calls! Round ends in a draw (Ryuukyoku).")
			// *** Handle Ryuukyoku scoring/dealer retention ***
			gameState.GamePhase = PhaseRoundEnd // Simple end for now
			break
		}

		// Small delay for non-human players to simulate thinking/make output readable
		if !isHumanPlayer && gameState.GamePhase != PhaseRoundEnd {
			time.Sleep(500 * time.Millisecond)
		}

	} // End game loop

	fmt.Println("\n--- Final Game State ---")
	DisplayGameState(gameState)
	fmt.Println("Game Over.")
}
