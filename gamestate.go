package main

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"sort"
	"time" // Used by rand.Seed, but seed is called in main.go
)

// NewGameState initializes a new game state for the given player names.
func NewGameState(playerNames []string) *GameState {
	if len(playerNames) != 4 {
		panic("Must initialize game with exactly 4 players")
	}
	// rand.Seed(time.Now().UnixNano()) // Seed is called once in main.go

	players := make([]*Player, len(playerNames))
	initialDealerIndex := rand.Intn(len(playerNames)) // Randomly select the initial dealer for the game

	winds := []string{"East", "South", "West", "North"}
	for i, name := range playerNames {
		// Assign seat wind based on the initial dealer
		// The dealer is East, then South, West, North in order of player index (0-3)
		seatWindIndex := (i - initialDealerIndex + len(playerNames)) % len(playerNames)
		players[i] = &Player{
			Name:                         name,
			Hand:                         []Tile{},
			Discards:                     []Tile{},
			Melds:                        []Meld{},
			Score:                        InitialScore, // from types.go
			SeatWind:                     winds[seatWindIndex],
			IsRiichi:                     false,
			RiichiTurn:                   -1,
			IsIppatsu:                    false,
			IsFuriten:                    false,
			IsPermanentRiichiFuriten:     false,
			DeclinedRonOnTurn:            -1,
			DeclinedRonTileID:            -1,
			RiichiDeclaredWaits:          []Tile{},
			DeclaredDoubleRiichi:         false,
			HasMadeFirstDiscardThisRound: false,
			HasDrawnFirstTileThisRound:   false,
			HasHadDiscardCalledThisRound: false,
			JustDrawnTile:                nil,
			IsTenpai:                     false,
			PaoTargetFor:                 nil,
			PaoSourcePlayerIndex:         -1,
			InitialTurnOrder:             i, // Store initial fixed order (0-3) for rules like Ssuufon Renda
		}
	}

	gs := &GameState{
		Players:              players,
		CurrentPlayerIndex:   initialDealerIndex, // Current player to act; dealer starts the first round.
		DealerIndexThisRound: initialDealerIndex, // Tracks who is dealer for this specific round.
		// DiscardPile is not strictly necessary if each player tracks their own discards for furiten.
		// DoraIndicators and UraDoraIndicators initialized by setupNewRoundDeck.
		PrevalentWind:               "East", // Game starts with East wind
		RoundNumber:                 1,      // Round number within the current Prevalent Wind (e.g., East 1, East 2)
		DealerRoundCount:            1,      // How many consecutive rounds the current dealer has been dealer
		Honba:                       0,
		RiichiSticks:                0,
		TurnNumber:                  0, // Overall turn number in the round (increments on each discard)
		GamePhase:                   PhaseDealing,
		InputReader:                 bufio.NewReader(os.Stdin),
		LastDiscard:                 nil,
		AnyCallMadeThisRound:        false,
		IsFirstGoAround:             true,
		RoundWinner:                 nil,
		FirstTurnDiscards:           [4]Tile{}, // For Ssuufon Renda, indexed by player.InitialTurnOrder
		FirstTurnDiscardCount:       0,
		DeclaredRiichiPlayerIndices: make(map[int]bool),
		TotalKansDeclaredThisRound:  0,
		MaxWindRounds:               2, // Default to East and South rounds (Tonpuusen + Nanbausen)
		CurrentWindRoundNumber:      1, // 1 for East, 2 for South, etc.
		SanchahouRonners:            []*Player{},
		GameLog:                     []string{fmt.Sprintf("Game Started. Initial Dealer: P%d %s", initialDealerIndex+1, players[initialDealerIndex].Name)},
	}

	// Initialize player-specific flags not covered by the Player struct zero values if needed
	for _, p := range gs.Players {
		// p.JustDrawnTile = nil // Already nil by default
		// p.RiichiDeclaredWaits = []Tile{} // Already empty slice by default
	}

	gs.setupNewRoundDeck() // Sets up Wall, DeadWall, and reveals initial Dora
	return gs
}

// setupNewRoundDeck prepares the deck, wall, dead wall, and initial Dora for a new round.
func (gs *GameState) setupNewRoundDeck() {
	deck := GenerateDeck()
	gs.Wall = deck[:TotalTiles-DeadWallSize]
	gs.DeadWall = deck[TotalTiles-DeadWallSize:] // Last 14 tiles
	gs.DoraIndicators = []Tile{}                 // Clear previous Dora
	gs.UraDoraIndicators = []Tile{}              // Clear previous Ura Dora
	gs.RevealInitialDoraIndicator()
	if len(gs.DoraIndicators) > 0 {
		gs.AddToGameLog(fmt.Sprintf("New round deck setup. Initial Dora Indicator: %s", gs.DoraIndicators[0].Name))
	} else {
		gs.AddToGameLog("New round deck setup. Error revealing initial Dora.") // Should not happen
	}
}

// RevealInitialDoraIndicator reveals the first dora indicator from the dead wall.
// Standard position: 3rd tile from the right end of the dead wall (when looking from player's perspective).
func (gs *GameState) RevealInitialDoraIndicator() {
	if len(gs.DeadWall) < 3 { // Need at least 3 tiles for standard Dora reveal position
		gs.AddToGameLog("Error: Dead wall too small for initial Dora revelation.")
		// fmt.Println("Error: Dead wall size incorrect for revealing initial Dora.")
		// Fallback: Add a dummy Dora if wall is critically small (should signal a major issue).
		if len(gs.DoraIndicators) == 0 { // Only if no Dora somehow got added
			gs.DoraIndicators = append(gs.DoraIndicators, Tile{Suit: "Man", Value: 1, Name: "Man 1"}) // Dummy
		}
		return
	}
	// Dead wall indices 0..13. Right end is 13. 3rd from right end is index 11.
	indicatorIndex := DeadWallSize - 3
	indicator := gs.DeadWall[indicatorIndex]
	gs.DoraIndicators = append(gs.DoraIndicators, indicator)
	// gs.AddToGameLog(fmt.Sprintf("Revealed Initial Dora Indicator: %s (From Dead Wall pos %d)", indicator.Name, indicatorIndex))
	// fmt.Printf("Revealed Dora Indicator: %s (From Dead Wall pos %d)\n", indicator.Name, indicatorIndex)
}

// DealInitialHands deals 13 tiles to each player from the wall.
func (gs *GameState) DealInitialHands() {
	numPlayers := len(gs.Players)
	dealerToStartDealing := gs.DealerIndexThisRound // The actual dealer for the current round starts the deal

	if len(gs.Wall) < HandSize*numPlayers {
		panic(fmt.Sprintf("Not enough tiles in wall (%d) to deal initial hands for %d players", len(gs.Wall), numPlayers))
	}

	// Phase 1: Deal 4 tiles to each player, 3 times (total 12 tiles each)
	for i := 0; i < 3; i++ { // 3 passes
		for j := 0; j < numPlayers; j++ { // Each player in turn order from dealer
			playerIndex := (dealerToStartDealing + j) % numPlayers
			for k := 0; k < 4; k++ { // 4 tiles
				if len(gs.Wall) == 0 {
					panic("Wall empty during initial deal (phase 1)")
				}
				tile := gs.Wall[0]
				gs.Wall = gs.Wall[1:]
				gs.Players[playerIndex].Hand = append(gs.Players[playerIndex].Hand, tile)
			}
		}
	}

	// Phase 2: Deal 1 tile to each player (total 13 tiles each)
	for j := 0; j < numPlayers; j++ { // Each player in turn order from dealer
		playerIndex := (dealerToStartDealing + j) % numPlayers
		if len(gs.Wall) == 0 {
			panic("Wall empty during initial deal (phase 2)")
		}
		tile := gs.Wall[0]
		gs.Wall = gs.Wall[1:]
		gs.Players[playerIndex].Hand = append(gs.Players[playerIndex].Hand, tile)
	}

	// Sort initial hands
	for _, player := range gs.Players {
		sort.Sort(BySuitValue(player.Hand))
	}

	gs.GamePhase = PhasePlayerTurn                  // Transition to first player's turn
	gs.CurrentPlayerIndex = gs.DealerIndexThisRound // Dealer makes the first move of the round
	gs.AddToGameLog("Initial hands dealt. Dealer's turn.")
}

// DrawTile draws a tile from the wall for the current player.
// Returns the drawn tile and a boolean indicating if the wall is now empty.
func (gs *GameState) DrawTile() (Tile, bool) {
	if len(gs.Wall) == 0 {
		gs.AddToGameLog(fmt.Sprintf("Attempted to draw from empty wall by %s.", gs.Players[gs.CurrentPlayerIndex].Name))
		// fmt.Println("Wall is empty!")
		return Tile{}, true // Wall is empty
	}
	tile := gs.Wall[0]
	gs.Wall = gs.Wall[1:] // Consume tile from wall

	player := gs.Players[gs.CurrentPlayerIndex]
	player.Hand = append(player.Hand, tile)
	player.JustDrawnTile = &tile // Track the drawn tile for Riichi discard rules, Tsumo Yaku, etc.
	sort.Sort(BySuitValue(player.Hand))

	// Drawing any tile breaks Ippatsu eligibility for ALL players currently eligible.
	for _, p := range gs.Players {
		if p.IsIppatsu {
			p.IsIppatsu = false
			gs.AddToGameLog(fmt.Sprintf("Ippatsu broken for %s due to draw by %s.", p.Name, player.Name))
		}
	}
	// gs.AddToGameLog(fmt.Sprintf("%s drew %s. Wall: %d", player.Name, tile.Name, len(gs.Wall)))
	return tile, false // Tile drawn, wall not necessarily empty now
}

// DrawRinshanTile draws a replacement tile from the dead wall after a Kan.
// Returns the drawn tile and a boolean indicating if no Rinshan tile was available.
func (gs *GameState) DrawRinshanTile() (Tile, bool) {
	// Rinshan tiles are taken from the "left" end (indices 0, 1, 2, 3) of the dead wall.
	// gs.TotalKansDeclaredThisRound tracks how many Kans have happened this round.
	// This determines which Rinshan tile to take and which Kan Dora to reveal.

	if gs.TotalKansDeclaredThisRound > RinshanTiles { // Using > because TotalKansDeclaredThisRound is incremented *before* calling this
		gs.AddToGameLog("Error: No more Rinshan tiles left in Dead Wall! (TotalKansDeclared > RinshanTiles)")
		// fmt.Println("Error: No more Rinshan tiles left in Dead Wall!")
		// This might lead to an abortive draw (Suukaikan) if conditions met.
		return Tile{}, true // Indicate error/empty
	}
	// The Rinshan tile index is TotalKansDeclaredThisRound - 1, because TotalKansDeclaredThisRound
	// was incremented by HandleKanAction *before* calling DrawRinshanTile.
	// So, for 1st Kan, TotalKans=1, index=0. For 4th Kan, TotalKans=4, index=3.
	rinshanTileIndex := gs.TotalKansDeclaredThisRound - 1
	if rinshanTileIndex < 0 || rinshanTileIndex >= RinshanTiles {
		gs.AddToGameLog(fmt.Sprintf("Error: Invalid Rinshan tile index %d (TotalKans: %d)", rinshanTileIndex, gs.TotalKansDeclaredThisRound))
		return Tile{}, true
	}

	rinshanTile := gs.DeadWall[rinshanTileIndex]
	// The tile is "taken", but the DeadWall structure itself isn't modified for simplicity of Dora indexing.
	// We just note that this tile is now in the player's hand.

	player := gs.Players[gs.CurrentPlayerIndex]
	player.Hand = append(player.Hand, rinshanTile)
	player.JustDrawnTile = &rinshanTile // Track this as the most recent draw
	sort.Sort(BySuitValue(player.Hand))

	// Drawing Rinshan also breaks Ippatsu for all players (if any were eligible).
	for _, p := range gs.Players {
		if p.IsIppatsu {
			p.IsIppatsu = false
			gs.AddToGameLog(fmt.Sprintf("Ippatsu broken for %s due to Rinshan draw by %s.", p.Name, player.Name))
		}
	}

	// Reveal Kan Dora Indicator *after* drawing Rinshan tile.
	gs.RevealKanDoraIndicator() // This function uses the number of *already revealed* Doras.

	gs.AddToGameLog(fmt.Sprintf("%s drew Rinshan tile: %s (from Dead Wall pos %d)", player.Name, rinshanTile.Name, rinshanTileIndex))
	return rinshanTile, false
}

// RevealKanDoraIndicator reveals a new dora indicator after a Kan.
func (gs *GameState) RevealKanDoraIndicator() {
	// Number of Dora indicators already revealed (initial + previous Kan Doras)
	numDorasCurrentlyRevealed := len(gs.DoraIndicators)

	if numDorasCurrentlyRevealed >= MaxRevealedDora {
		gs.AddToGameLog("Maximum number of Dora indicators (5) already revealed.")
		// fmt.Println("Maximum number of Dora indicators already revealed.")
		return
	}

	// Kan Dora indicators are revealed from right-to-left, next to the previous ones.
	// Initial Dora: DeadWall[DW_Size - 3] (index 11)
	// 1st Kan Dora: DeadWall[DW_Size - 5] (index 9) -> when numDorasCurrentlyRevealed is 1 (only initial revealed)
	// 2nd Kan Dora: DeadWall[DW_Size - 7] (index 7) -> when numDorasCurrentlyRevealed is 2
	// Nth Kan Dora: DeadWall[DW_Size - 3 - (N * 2)]
	// Here, N is numDorasCurrentlyRevealed (since it's 1 for 1st Kan Dora, 2 for 2nd etc.)
	kanDoraIndicatorIndexInDeadWall := DeadWallSize - 3 - (numDorasCurrentlyRevealed * 2)

	// Ensure we don't reveal from the Rinshan tile area (indices 0-3) or go out of bounds.
	if kanDoraIndicatorIndexInDeadWall < RinshanTiles {
		gs.AddToGameLog(fmt.Sprintf("Error: Calculated Kan Dora index %d overlaps with Rinshan area (0-3).", kanDoraIndicatorIndexInDeadWall))
		// fmt.Printf("Error calculating Kan Dora index %d, potential overlap with Rinshan area.\n", kanDoraIndicatorIndexInDeadWall)
		return
	}
	if kanDoraIndicatorIndexInDeadWall < 0 { // Should be caught by above, but safety.
		gs.AddToGameLog(fmt.Sprintf("Error: Calculated Kan Dora index %d is negative.", kanDoraIndicatorIndexInDeadWall))
		// fmt.Println("Error: Calculated Kan Dora index is negative.")
		return
	}

	indicator := gs.DeadWall[kanDoraIndicatorIndexInDeadWall]
	gs.DoraIndicators = append(gs.DoraIndicators, indicator) // Add to the list of all revealed indicators
	gs.AddToGameLog(fmt.Sprintf("Revealed Kan Dora Indicator: %s (From Dead Wall pos %d)", indicator.Name, kanDoraIndicatorIndexInDeadWall))
	// fmt.Printf("Revealed Kan Dora Indicator: %s (From Dead Wall pos %d)\n", indicator.Name, kanDoraIndicatorIndexInDeadWall)
}

// RevealUraDoraIndicators reveals the Ura Dora indicators corresponding to revealed Dora/KanDora.
// Called only on a Riichi win.
func (gs *GameState) RevealUraDoraIndicators() {
	gs.UraDoraIndicators = []Tile{}           // Clear previous Ura Dora, if any
	numDoraRevealed := len(gs.DoraIndicators) // How many Dora/KanDora were flipped in total

	if numDoraRevealed == 0 {
		return
	} // No Dora revealed, so no Ura Dora
	if len(gs.DeadWall) != DeadWallSize {
		gs.AddToGameLog("Error: Dead wall size incorrect for revealing Ura Dora.")
		// fmt.Println("Error: Dead wall size incorrect for revealing Ura Dora.")
		return
	}
	gs.AddToGameLog("Revealing Ura Dora Indicators...")
	// fmt.Println("Revealing Ura Dora...")

	for i := 0; i < numDoraRevealed; i++ {
		// The i-th revealed Dora indicator (0-indexed) corresponds to:
		// Dora Indicator index in DeadWall: DeadWallSize - 3 - (i * 2)
		// Ura Dora Indicator index in DeadWall: DeadWallSize - 3 - (i * 2) + 1
		doraIndicatorPhysicalIndex := DeadWallSize - 3 - (i * 2)
		uraIndicatorPhysicalIndex := doraIndicatorPhysicalIndex + 1

		// Check validity of Ura Dora index (must be within DeadWall and not in Rinshan area)
		if uraIndicatorPhysicalIndex >= RinshanTiles && uraIndicatorPhysicalIndex < DeadWallSize {
			uraIndicator := gs.DeadWall[uraIndicatorPhysicalIndex]
			gs.UraDoraIndicators = append(gs.UraDoraIndicators, uraIndicator)
			gs.AddToGameLog(fmt.Sprintf(" - Ura Dora Indicator: %s (From Dead Wall pos %d)", uraIndicator.Name, uraIndicatorPhysicalIndex))
			// fmt.Printf(" - Ura Dora Indicator: %s (From Dead Wall pos %d)\n", uraIndicator.Name, uraIndicatorPhysicalIndex)
		} else {
			gs.AddToGameLog(fmt.Sprintf("Warning: Invalid or out-of-bounds Ura Dora index %d (for Dora at DW pos %d).", uraIndicatorPhysicalIndex, doraIndicatorPhysicalIndex))
			// fmt.Printf("Warning: Invalid or out-of-bounds Ura Dora index %d calculated for Dora at %d.\n", uraIndicatorPhysicalIndex, doraIndicatorPhysicalIndex)
		}
	}
	sort.Sort(BySuitValue(gs.UraDoraIndicators)) // Sort for consistent display if needed
}

// NextPlayer moves the turn to the next player in sequence.
func (gs *GameState) NextPlayer() {
	prevPlayerIndex := gs.CurrentPlayerIndex
	if prevPlayerIndex >= 0 && prevPlayerIndex < len(gs.Players) { // Safety check
		gs.Players[prevPlayerIndex].JustDrawnTile = nil // Clear just drawn tile when turn passes
	}
	gs.CurrentPlayerIndex = (gs.CurrentPlayerIndex + 1) % len(gs.Players)
	// gs.AddToGameLog(fmt.Sprintf("Turn passes from P%d (%s) to P%d (%s).",
	// 	prevPlayerIndex+1, gs.Players[prevPlayerIndex].Name,
	// 	gs.CurrentPlayerIndex+1, gs.Players[gs.CurrentPlayerIndex].Name))
}

// GetPlayerIndex returns the index (0-3) of a given player object.
func (gs *GameState) GetPlayerIndex(player *Player) int {
	for i, p := range gs.Players {
		if p == player {
			return i
		}
	}
	gs.AddToGameLog(fmt.Sprintf("Critical Error: Player object %s not found in gs.Players.", player.Name))
	return -1 // Should not happen if player objects are managed correctly
}

// AddToGameLog adds a message to the game log, with a limit on log size.
func (gs *GameState) AddToGameLog(message string) {
	fmt.Println("LOG: " + message) // Print to console for immediate visibility during CLI play
	gs.GameLog = append(gs.GameLog, time.Now().Format("15:04:05")+" | "+message)
	if len(gs.GameLog) > 200 { // Keep log from growing indefinitely
		gs.GameLog = gs.GameLog[len(gs.GameLog)-100:] // Keep last 100 entries
	}
}
