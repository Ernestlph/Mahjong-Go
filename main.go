package main

import (
	"fmt"
	"math/rand"
	"os"
	"sort"

	// "strings" // Not directly used in this version of main.go
	"time"
)

// Constants are now in types.go

func main() {
	rand.Seed(time.Now().UnixNano()) // Seed random number generator once
	fmt.Println("Starting Riichi Mahjong Game")
	gsLog := []string{} // Game-level log, if needed beyond gs.GameLog

	playerNames := []string{"Player 1 (You)", "Player 2 (AI)", "Player 3 (AI)", "Player 4 (AI)"}
	gameState := NewGameState(playerNames) // NewGameState sets PhaseDealing, PrevalentWind, etc.

	// Main Game Loop - continues as long as the game is not over
	for gameState.GamePhase != PhaseGameEnd {

		// --- New Round Setup ---
		if gameState.GamePhase == PhaseDealing {
			// This block is for the very start of a new round.
			// setupNewRoundDeck (called by NewGameState or end of previous round) prepares Wall, DeadWall, Dora.
			// DealInitialHands deals tiles and sets GamePhase to PhasePlayerTurn.
			gameState.DealInitialHands()
			gameState.AddToGameLog(fmt.Sprintf("--- Round %s %d (%d of Wind) Starting ---", gameState.PrevalentWind, gameState.RoundNumber, gameState.DealerRoundCount))
			// fmt.Printf("\n--- Round %s %d (%d of Wind, Dealer Round %d) Starting ---\n",
			// gameState.PrevalentWind, gameState.CurrentWindRoundNumber, gameState.RoundNumber, gameState.DealerRoundCount)
		}

		// --- Player Turn Loop (Inner loop for a single round's turns) ---
		// This loop runs as long as it's a player's turn and the round/game hasn't ended.
		for gameState.GamePhase == PhasePlayerTurn {
			currentPlayer := gameState.Players[gameState.CurrentPlayerIndex]
			isHumanPlayer := gameState.CurrentPlayerIndex == 0

			// Reset turn-specific flags for the current player's action sequence
			gameState.IsChankanOpportunity = false
			gameState.IsRinshanWin = false
			// IsHouteiDiscard is set specifically when the Haitei discard happens
			// It should be false at the start of a "normal" turn.
			// If a turn *becomes* the Houtei discard turn, DiscardTile will set it.
			// To be safe, ensure it's false unless explicitly set by Haitei discard logic.
			if !gameState.IsHouteiDiscard { // Only reset if not already set this turn by Haitei logic
				gameState.IsHouteiDiscard = false
			}
			gameState.SanchahouRonners = []*Player{} // Clear for current discard check

			if isHumanPlayer || true { // Always display for debugging for now
				DisplayGameState(gameState)
			}

			// --- Kyuushuu Kyuuhai Check (before draw) ---
			// Conditions: Player's first turn of the round, no calls made by anyone yet in the round.
			if !currentPlayer.HasDrawnFirstTileThisRound && !gameState.AnyCallMadeThisRound &&
				gameState.TurnNumber < len(gameState.Players) { // Ensures it's within the first cycle of turns

				if CheckKyuushuuKyuuhai(currentPlayer.Hand, currentPlayer.Melds) { // Pass melds to ensure no open melds
					kyuushuuDeclared := false
					if isHumanPlayer {
						fmt.Println("Your hand qualifies for Kyuushuu Kyuuhai (9+ unique terminal/honor tiles).")
						if GetPlayerChoice(gameState.InputReader, "Declare Kyuushuu Kyuuhai for an abortive draw? (y/n): ") {
							kyuushuuDeclared = true
						}
					} else { // AI always declares
						// gameState.AddToGameLog(fmt.Sprintf("%s's hand qualifies for Kyuushuu Kyuuhai.", currentPlayer.Name))
						kyuushuuDeclared = true
					}

					if kyuushuuDeclared {
						gameState.AddToGameLog(fmt.Sprintf("%s declares Kyuushuu Kyuuhai! Round ends in an abortive draw.", currentPlayer.Name))
						// fmt.Printf("%s declares Kyuushuu Kyuuhai! Round ends in an abortive draw.\n", currentPlayer.Name)
						gameState.GamePhase = PhaseRoundEnd
						gameState.RoundWinner = nil // Mark as draw
						// Honba usually increments for abortive draws. This is handled in Round End processing.
						break // Exit player turn loop, proceed to Round End processing
					}
				}
			}
			if gameState.GamePhase != PhasePlayerTurn {
				break
			} // If Kyuushuu ended round, skip rest of turn logic

			// --- Draw Phase ---
			if isHumanPlayer {
				fmt.Printf("\n--- %s's Turn (%s Wind) ---\n", currentPlayer.Name, currentPlayer.SeatWind)
			} else {
				// gameState.AddToGameLog(fmt.Sprintf("--- %s's Turn (%s Wind) ---", currentPlayer.Name, currentPlayer.SeatWind))
			}

			isHaiteiDraw := len(gameState.Wall) == 1 // If 1 tile left, this draw makes the wall empty (Haitei)
			drawnTile, wallNowEmptyAfterDraw := gameState.DrawTile()
			currentPlayer.HasDrawnFirstTileThisRound = true // Player has now drawn their first tile

			if wallNowEmptyAfterDraw && !isHaiteiDraw { // Wall emptied unexpectedly
				gameState.AddToGameLog("Wall empty unexpectedly after draw! Round ends in Ryuukyoku.")
				// fmt.Println("\nWall is empty! Round ends in a draw (Ryuukyoku).")
				gameState.GamePhase = PhaseRoundEnd
				gameState.RoundWinner = nil
				break // Exit player turn loop
			}
			gameState.AddToGameLog(fmt.Sprintf("%s draws: %s", currentPlayer.Name, drawnTile.Name))
			// fmt.Printf("%s draws: %s\n", currentPlayer.Name, drawnTile.Name)
			if isHaiteiDraw {
				gameState.AddToGameLog("This is the Haitei tile (last from wall).")
				// fmt.Println("This is the last tile from the wall (Haitei).")
			}

			// --- Action Phase (Tsumo, Kan on Draw) ---
			actionTakenThisSegment := false

			if CanDeclareTsumo(currentPlayer, gameState) {
				tsumoConfirm := !isHumanPlayer // AI default: Tsumo if possible
				if isHumanPlayer {
					DisplayPlayerState(currentPlayer) // Show hand before Tsumo choice
					tsumoConfirm = GetPlayerChoice(gameState.InputReader, "Declare TSUMO? (y/n): ")
				}
				if tsumoConfirm {
					// gameState.AddToGameLog(fmt.Sprintf("%s declares TSUMO!", currentPlayer.Name))
					// fmt.Printf("%s declares TSUMO!\n", currentPlayer.Name)
					HandleWin(gameState, currentPlayer, drawnTile, true) // Sets GamePhase to RoundEnd
					actionTakenThisSegment = true
				}
			}
			if gameState.GamePhase != PhasePlayerTurn {
				break
			} // If Tsumo ended round

			// Check for Kan on Draw (only if Tsumo was not declared or declined)
			if !actionTakenThisSegment {
				possibleKanType, kanTargetTile := CanDeclareKanOnDraw(currentPlayer, drawnTile, gameState)
				if possibleKanType != "" {
					kanConfirm := !isHumanPlayer // AI decision for Kan
					if isHumanPlayer {
						DisplayPlayerState(currentPlayer) // Show hand before Kan choice
						kanConfirm = GetPlayerChoice(gameState.InputReader, fmt.Sprintf("Declare %s with %s? (y/n): ", possibleKanType, kanTargetTile.Name))
					} else { // AI Kan logic
						// AI: Kan if not Riichi, or if Riichi and waits don't change
						if !currentPlayer.IsRiichi || (currentPlayer.IsRiichi && !checkWaitChangeForRiichiKan(currentPlayer, gameState, kanTargetTile, possibleKanType)) {
							// AI confirms safe Kan
						} else {
							kanConfirm = false // AI skips unsafe Kan
							gameState.AddToGameLog(fmt.Sprintf("AI %s skips %s with %s (unsafe for Riichi).", currentPlayer.Name, possibleKanType, kanTargetTile.Name))
						}
					}

					if kanConfirm {
						// gameState.AddToGameLog(fmt.Sprintf("%s declares %s with %s.", currentPlayer.Name, possibleKanType, kanTargetTile.Name))
						// fmt.Printf("%s declares %s!\n", currentPlayer.Name, possibleKanType)
						HandleKanAction(gameState, currentPlayer, kanTargetTile, possibleKanType)
						actionTakenThisSegment = true // Kan action handles next step (Rinshan, then PromptDiscard or win)
					}
				}
			}
			if gameState.GamePhase != PhasePlayerTurn {
				break
			} // If Kan led to win and ended round

			// --- Discard Phase ---
			// If actionTakenThisSegment is true (due to Kan on draw that didn't end game),
			// HandleKanAction would have called PromptDiscard, which calls DiscardTile.
			// So, this block is for when no Tsumo/Kan happened on draw, or if a Kan happened but didn't result in a win or further required actions *before* a normal discard.
			// The `PromptDiscard` called by `HandleKanAction` is the key here.
			// If `actionTakenThisSegment` is true, it means the turn flow is managed by `HandleKanAction` (which calls `PromptDiscard`).
			// If `actionTakenThisSegment` is false, we proceed to the normal discard logic here.
			if !actionTakenThisSegment {
				canRiichi, riichiOptions := CanDeclareRiichi(currentPlayer, gameState)
				discardIndex := -1
				riichiDeclaredSuccessfully := false

				if isHumanPlayer {
					DisplayPlayerState(currentPlayer) // Show hand before any discard choice
					if canRiichi {
						chosenOptionIndex, choiceMade := GetPlayerRiichiChoice(gameState.InputReader, riichiOptions)
						if choiceMade {
							selectedOption := riichiOptions[chosenOptionIndex]
							discardIndex = selectedOption.DiscardIndex // This is index in the 14-tile hand
							if HandleRiichiAction(gameState, currentPlayer, discardIndex) {
								riichiDeclaredSuccessfully = true // Riichi and discard happened
							} else { // Riichi validation failed (e.g., chosen discard wrong)
								gameState.AddToGameLog("Riichi declaration failed internal validation. Proceeding with normal discard.")
								// fmt.Println("Riichi declaration failed validation. Proceeding with normal discard.")
								discardIndex = GetPlayerDiscardChoice(gameState.InputReader, currentPlayer)
							}
						} else { // Player cancelled Riichi choice
							gameState.AddToGameLog(fmt.Sprintf("%s cancelled Riichi. Proceeding with normal discard.", currentPlayer.Name))
							// fmt.Println("Proceeding with normal discard.")
							discardIndex = GetPlayerDiscardChoice(gameState.InputReader, currentPlayer)
						}
					} else { // Cannot Riichi, just get normal discard
						discardIndex = GetPlayerDiscardChoice(gameState.InputReader, currentPlayer)
					}
				} else { // AI Logic for discard
					// gameState.AddToGameLog(fmt.Sprintf("AI %s thinking for discard...", currentPlayer.Name))
					if currentPlayer.IsRiichi {
						foundDrawn := false
						if currentPlayer.JustDrawnTile != nil { // Must discard drawn tile if Riichi
							for i, t := range currentPlayer.Hand {
								if t.ID == currentPlayer.JustDrawnTile.ID {
									discardIndex = i
									foundDrawn = true
									break
								}
							}
						}
						if !foundDrawn {
							gameState.AddToGameLog(fmt.Sprintf("Error: AI %s (Riichi) couldn't find JustDrawnTile for discard. Discarding last.", currentPlayer.Name))
							if len(currentPlayer.Hand) > 0 {
								discardIndex = len(currentPlayer.Hand) - 1
							} else {
								discardIndex = -1
							}
						}
					} else { // AI not in Riichi
						if canRiichi {
							gameState.AddToGameLog(fmt.Sprintf("AI %s can Riichi, chooses first option.", currentPlayer.Name))
							// fmt.Printf("(%s can Riichi, AI chooses to Riichi!)\n", currentPlayer.Name)
							chosenOption := riichiOptions[0]
							discardIndex = chosenOption.DiscardIndex
							if HandleRiichiAction(gameState, currentPlayer, discardIndex) {
								riichiDeclaredSuccessfully = true
							} else {
								gameState.AddToGameLog(fmt.Sprintf("Error: AI %s Riichi failed validation. Discarding last.", currentPlayer.Name))
								// fmt.Println("Error: AI Riichi failed validation?")
								if len(currentPlayer.Hand) > 0 {
									discardIndex = len(currentPlayer.Hand) - 1
								} else {
									discardIndex = -1
								}
							}
						} else { // AI cannot Riichi, basic discard: JustDrawnTile
							foundDrawn := false
							if currentPlayer.JustDrawnTile != nil {
								for i, t := range currentPlayer.Hand {
									if t.ID == currentPlayer.JustDrawnTile.ID {
										discardIndex = i
										foundDrawn = true
										break
									}
								}
							}
							if !foundDrawn {
								if len(currentPlayer.Hand) > 0 {
									discardIndex = len(currentPlayer.Hand) - 1
								} else {
									discardIndex = -1
								}
							}
						}
					}
					if !riichiDeclaredSuccessfully && (discardIndex < 0 || discardIndex >= len(currentPlayer.Hand)) {
						gameState.AddToGameLog(fmt.Sprintf("Error: AI %s calculated invalid discard index %d. Defaulting to 0.", currentPlayer.Name, discardIndex))
						// fmt.Printf("Error: AI calculated invalid discard index %d (Hand Size %d). Defaulting to 0.\n", discardIndex, len(currentPlayer.Hand))
						if len(currentPlayer.Hand) > 0 {
							discardIndex = 0
						} else {
							discardIndex = -1
						}
					}
				} // End AI Logic

				// Perform the discard *only if* it wasn't handled by Riichi declaration and index is valid
				if !riichiDeclaredSuccessfully && discardIndex != -1 {
					if isHaiteiDraw { // isHaiteiDraw means the drawnTile was the last one from wall
						gameState.IsHouteiDiscard = true // This discard is Houtei
						gameState.AddToGameLog("This discard is Houtei Raoyui opportunity.")
						// fmt.Println("This discard is Houtei (last discard of the game).")
					}
					_, gameShouldEnd := DiscardTile(gameState, currentPlayer, discardIndex) // DiscardTile handles calls, Furiten, turn advancement
					if gameShouldEnd {                                                      // Ron occurred on the discard
						break // Exit player turn loop
					}
				} else if !riichiDeclaredSuccessfully && discardIndex == -1 {
					gameState.AddToGameLog(fmt.Sprintf("Error: No valid discard index determined for %s after choices.", currentPlayer.Name))
					// fmt.Println("Error: No valid discard index determined.")
					gameState.GamePhase = PhaseRoundEnd
					gameState.RoundWinner = nil // Treat as error/draw
					break
				}
			} // End of discard phase logic block (if !actionTakenThisSegment)
			if gameState.GamePhase != PhasePlayerTurn {
				break
			} // If Riichi action or DiscardTile ended round

			// --- Post-Discard Checks (Abortive Draws, Ryuukyoku) ---
			// Check for Suu Riichi completion (abort if 4th Riichi discard not Ronned)
			if CheckSuuRiichi(gameState) {
				// Abort only if the discard *just made* by the 4th Riichi player was not Ronned.
				// DiscardTile would have set GamePhase to RoundEnd if Ron occurred.
				if gameState.GamePhase == PhasePlayerTurn {
					gameState.AddToGameLog("Suu Riichi! Round aborts as 4th Riichi player's discard was not Ronned.")
					// fmt.Println("Suu Riichi! Round aborts as 4th Riichi player's discard was not Ronned.")
					gameState.GamePhase = PhaseRoundEnd
					gameState.RoundWinner = nil
					break
				}
			}

			// Ryuukyoku due to Haitei/Houtei with no win on that last action
			if isHaiteiDraw && gameState.GamePhase == PhasePlayerTurn {
				gameState.AddToGameLog("Haitei/Houtei passed with no win. Round ends in Ryuukyoku.")
				// fmt.Println("\nLast tile drawn and discarded with no win. Round ends in a draw (Ryuukyoku).")
				gameState.GamePhase = PhaseRoundEnd
				gameState.RoundWinner = nil
				// Nagashi Mangan check now happens in Round End Processing.
				break
			}
			// General wall empty check (safety, should be covered by Haitei)
			if wallNowEmptyAfterDraw && len(gameState.Wall) == 0 && gameState.GamePhase == PhasePlayerTurn {
				gameState.AddToGameLog("Wall empty after player's turn actions. Round ends in Ryuukyoku.")
				// fmt.Println("\nWall is empty after player's turn! Round ends in a draw (Ryuukyoku).")
				gameState.GamePhase = PhaseRoundEnd
				gameState.RoundWinner = nil
				break
			}

			// Small delay for AI players if game continues
			if !isHumanPlayer && gameState.GamePhase == PhasePlayerTurn {
				time.Sleep(100 * time.Millisecond) // Shortened for faster simulation
			}
		} // End of Player Turn Loop

		// --- Round End Processing ---
		if gameState.GamePhase == PhaseRoundEnd {
			gameState.AddToGameLog(fmt.Sprintf("--- Round %s %d (%d for Dealer, %d Wind Round) Ended ---",
				gameState.PrevalentWind, gameState.RoundNumber, gameState.DealerRoundCount, gameState.CurrentWindRoundNumber))
			// fmt.Println("\n--- Round End Processing ---")

			nagashiWinner := (*Player)(nil)
			if gameState.RoundWinner == nil { // Exhaustive draw or abortive draw
				for _, p := range gameState.Players {
					if isNagashi, nagashiName, _ := checkNagashiMangan(p, gameState); isNagashi {
						gameState.AddToGameLog(fmt.Sprintf("!!! %s achieves %s !!!", p.Name, nagashiName))
						// fmt.Printf("!!! %s achieves %s! (Scoring to be refined) !!!\n", p.Name, nagashiName)
						nagashiWinner = p
						// Simulate Mangan Tsumo for Nagashi winner.
						isWinnerDealer := (gameState.Players[gameState.DealerIndexThisRound] == p)
						// Create a dummy winning tile for payment calculation, Yaku calc will be overridden.
						dummyAgari := Tile{Suit: "Special", Value: 1, Name: "Nagashi"}
						payment := CalculatePointPayment(5, 30, isWinnerDealer, true, gameState.Honba, gameState.RiichiSticks) // Mangan

						// Pao logic for Nagashi is not standard. Direct transfer:
						nagashiTotalPayment := 0
						if isWinnerDealer {
							nagashiTotalPayment = payment.TsumoNonDealerPay * (len(gameState.Players) - 1)
						} else {
							nagashiTotalPayment = payment.TsumoDealerPay + payment.TsumoNonDealerPay*(len(gameState.Players)-2)
						}
						// Transfer from others to Nagashi winner
						for _, otherP := range gameState.Players {
							if otherP == p {
								continue
							}
							var amountToPay int
							if isWinnerDealer {
								amountToPay = payment.TsumoNonDealerPay
							} else {
								if gameState.Players[gameState.DealerIndexThisRound] == otherP {
									amountToPay = payment.TsumoDealerPay
								} else {
									amountToPay = payment.TsumoNonDealerPay
								}
							}
							otherP.Score -= amountToPay
							gameState.AddToGameLog(fmt.Sprintf("%s pays %d to %s for Nagashi Mangan.", otherP.Name, amountToPay, p.Name))
						}
						p.Score += nagashiTotalPayment
						p.Score += gameState.RiichiSticks * RiichiBet // Nagashi winner gets Riichi sticks
						gameState.RiichiSticks = 0

						gameState.RoundWinner = p // Nagashi Mangan is a form of win.
						break                     // Only one Nagashi Mangan.
					}
				}
			}

			// Tenpai/Notenpai for Ryuukyoku (if no winner from Nagashi etc.)
			if gameState.RoundWinner == nil { // Still a draw after Nagashi check (or no Nagashi)
				for _, p := range gameState.Players {
					p.IsTenpai = IsTenpai(p.Hand, p.Melds)
					gameState.AddToGameLog(fmt.Sprintf("%s is %s at Ryuukyoku.", p.Name, If(p.IsTenpai, "Tenpai", "Noten")))
				}
				HandleNotenBappu(gameState) // Handles point transfers for Noten Bappu
				if isHumanPlayer || true {
					DisplayGameState(gameState)
				} // Show Tenpai statuses and score changes
			}

			// --- Game End Conditions Check ---
			gameShouldActuallyEnd := false
			for _, p := range gameState.Players {
				if p.Score < 0 {
					gameState.AddToGameLog(fmt.Sprintf("Player %s has busted (score: %d)! Game Over.", p.Name, p.Score))
					// fmt.Printf("Player %s has a negative score (%d). Game Over!\n", p.Name, p.Score)
					gameShouldActuallyEnd = true
					break
				}
			}
			// Hanchan End Logic
			if !gameShouldActuallyEnd {
				// Check if the game should end based on rounds completed (e.g., South 4)
				// gameState.RoundNumber is the round counter within the current PrevalentWind (1-4)
				// gameState.CurrentWindRoundNumber is 1 for East, 2 for South, etc.
				isLastProgrammedWind := gameState.CurrentWindRoundNumber >= gameState.MaxWindRounds
				isLastDealerTurnOfFinalWind := gameState.RoundNumber >= 4 // Current dealer completed their 4th turn as dealer for this wind

				if isLastProgrammedWind && isLastDealerTurnOfFinalWind {
					dealerPlayer := gameState.Players[gameState.DealerIndexThisRound] // Dealer of the round that just ended
					dealerWins := gameState.RoundWinner == dealerPlayer
					dealerTenpaiAtDraw := (gameState.RoundWinner == nil && dealerPlayer.IsTenpai)

					// Standard: Game ends after last programmed round unless dealer wins/tenpai AND is not top.
					// Simplified: Game ends after South 4 (or equivalent final wind round).
					// Agari-yame/Tenpai-yame (dealer can choose to end if top) is advanced.
					if !dealerWins && !dealerTenpaiAtDraw {
						// Dealer did not win or was not tenpai (if draw), so no renchan on the final turn. Game ends.
						gameState.AddToGameLog(fmt.Sprintf("Final programmed round (%s %d) completed. Dealer did not win/Tenpai. Game Over.", gameState.PrevalentWind, gameState.RoundNumber))
						gameShouldActuallyEnd = true
					} else {
						// Dealer won or was tenpai. Normally, they could choose to continue (if not top) or end.
						// For simplicity, if it's the absolute last round (e.g., South 4), it ends.
						// If rules allowed for West round etc., then Renchan would continue.
						gameState.AddToGameLog(fmt.Sprintf("Final programmed round (%s %d) dealer Renchan. Game ends (standard).", gameState.PrevalentWind, gameState.RoundNumber))
						gameShouldActuallyEnd = true
					}
				}
			}

			if gameShouldActuallyEnd {
				gameState.GamePhase = PhaseGameEnd
			} else { // Prepare for Next Round
				gameState.AddToGameLog("Preparing for next round setup...")
				currentRoundDealerPlayer := gameState.Players[gameState.DealerIndexThisRound] // Dealer of the round that just ended
				isDealerWin := gameState.RoundWinner == currentRoundDealerPlayer
				isDealerTenpaiAtDraw := (gameState.RoundWinner == nil && currentRoundDealerPlayer.IsTenpai)

				// Renchan Logic for Honba & Dealer Position
				if isDealerWin || isDealerTenpaiAtDraw { // Dealer Renchan
					gameState.Honba++
					// DealerIndexThisRound remains the same.
					gameState.DealerRoundCount++ // This dealer's consecutive rounds as dealer.
					gameState.AddToGameLog(fmt.Sprintf("Dealer %s retained (Renchan). Honba to %d. Dealer's %d round as dealer.",
						currentRoundDealerPlayer.Name, gameState.Honba, gameState.DealerRoundCount))
				} else { // Dealer changes
					// If dealer is Noten at Ryuukyoku, Honba still increments even if dealership passes.
					if gameState.RoundWinner == nil && !currentRoundDealerPlayer.IsTenpai {
						gameState.Honba++
						gameState.AddToGameLog(fmt.Sprintf("Dealer %s Noten at Ryuukyoku. Honba to %d. Dealership passes.",
							currentRoundDealerPlayer.Name, gameState.Honba))
					} else { // Non-dealer win
						gameState.Honba = 0 // Reset Honba
						gameState.AddToGameLog("Non-dealer win. Honba reset.")
					}

					gameState.DealerIndexThisRound = (gameState.DealerIndexThisRound + 1) % len(gameState.Players)
					gameState.DealerRoundCount = 1 // New dealer starts their 1st round count.

					// Advance Round Number (within the current Prevalent Wind)
					gameState.RoundNumber++ // This is the round number for the current Prevalent Wind (e.g., East 1, East 2 ...)
					if gameState.RoundNumber > 4 {
						gameState.RoundNumber = 1          // Reset to 1 for the new Prevalent Wind
						gameState.CurrentWindRoundNumber++ // This tracks which wind it is (1=E, 2=S)
						switch gameState.PrevalentWind {
						case "East":
							gameState.PrevalentWind = "South"
						case "South":
							gameState.PrevalentWind = "West" // If MaxWindRounds allows
						case "West":
							gameState.PrevalentWind = "North" // If MaxWindRounds allows
						case "North": // Game usually ends or loops based on complex rules
							if gameState.MaxWindRounds > 4 {
								gameState.PrevalentWind = "East"
							} else { /* Game should have ended */
							}
						}
						gameState.AddToGameLog(fmt.Sprintf("Prevalent Wind advances to %s.", gameState.PrevalentWind))
					}
					gameState.AddToGameLog(fmt.Sprintf("Dealer changes to %s. Wind Round: %s. Dealer turn in wind: %d. Their dealer streak: %d.",
						gameState.Players[gameState.DealerIndexThisRound].Name, gameState.PrevalentWind, gameState.RoundNumber, gameState.DealerRoundCount))
				}

				// Update Seat Winds based on new DealerIndexThisRound for the *next* round
				winds := []string{"East", "South", "West", "North"}
				for i := 0; i < len(gameState.Players); i++ {
					seatWindIndex := (i - gameState.DealerIndexThisRound + len(gameState.Players)) % len(gameState.Players)
					gameState.Players[i].SeatWind = winds[seatWindIndex]
				}
				// gameState.AddToGameLog("Seat winds updated for new round.")

				// Reset round-specific player flags
				for _, p := range gameState.Players {
					p.Hand = []Tile{}
					p.Discards = []Tile{}
					p.Melds = []Meld{}
					p.IsRiichi = false
					p.RiichiTurn = -1
					p.IsIppatsu = false
					p.DeclaredDoubleRiichi = false
					p.HasMadeFirstDiscardThisRound = false
					p.HasDrawnFirstTileThisRound = false
					p.JustDrawnTile = nil
					p.IsFuriten = false
					p.IsPermanentRiichiFuriten = false
					p.DeclinedRonOnTurn = -1
					p.DeclinedRonTileID = -1
					p.RiichiDeclaredWaits = []Tile{}
					p.IsTenpai = false
					p.PaoSourcePlayerIndex = -1
					p.PaoTargetFor = nil
					p.HasHadDiscardCalledThisRound = false
				}
				// Reset round-specific game state flags
				gameState.LastDiscard = nil
				// DoraIndicators and UraDoraIndicators are cleared and re-revealed by setupNewRoundDeck
				gameState.AnyCallMadeThisRound = false
				gameState.IsFirstGoAround = true
				gameState.TurnNumber = 0 // Reset turn counter for the new round (discards in round)
				gameState.DeclaredRiichiPlayerIndices = make(map[int]bool)
				gameState.TotalKansDeclaredThisRound = 0
				gameState.FirstTurnDiscardCount = 0
				gameState.FirstTurnDiscards = [4]Tile{} // Reset for Ssuufon Renda
				gs.SanchahouRonners = []*Player{}
				gameState.RoundWinner = nil

				gameState.setupNewRoundDeck()      // Sets up Wall, DeadWall, initial Dora
				gameState.GamePhase = PhaseDealing // Ready for next round's deal
			}
		} // End Round End Processing
	} // End Main Game Loop

	// --- Final Game Outcome ---
	if gameState.GamePhase == PhaseGameEnd {
		fmt.Println("\n\n--- FINAL GAME RESULTS ---")
		gsLog = append(gsLog, "--- FINAL GAME RESULTS ---") // Game-level log
		DisplayGameState(gameState)                         // Show final state with scores

		// Sort players by score for final display
		finalScores := make([]*Player, len(gameState.Players))
		copy(finalScores, gameState.Players)
		sort.Slice(finalScores, func(i, j int) bool {
			return finalScores[i].Score > finalScores[j].Score
		})
		fmt.Println("Final Scores:")
		gsLog = append(gsLog, "Final Scores:")
		for i, p := range finalScores {
			scoreStr := fmt.Sprintf("%d. %s: %d points", i+1, p.Name, p.Score)
			fmt.Println(scoreStr)
			gsLog = append(gsLog, scoreStr)
		}
		fmt.Println("Game Over.")
		gsLog = append(gsLog, "Game Over.")

		// Optionally print full game log from gameState
		fmt.Println("\n--- Full Game Log from GameState ---")
		for _, entry := range gameState.GameLog {
			fmt.Println(entry)
		}
		// Exit or offer new game
		os.Exit(0)
	}
}
