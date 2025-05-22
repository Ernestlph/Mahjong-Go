package main

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"sort"
	"time"
)

// NewGameState initializes a new game state for the given player names.
func NewGameState(playerNames []string) *GameState {
	if len(playerNames) != 4 {
		panic("Must initialize game with exactly 4 players")
	}
	rand.Seed(time.Now().UnixNano()) // Seed the random number generator

	deck := GenerateDeck()

	// The dead wall is the *last* 14 tiles of the shuffled deck.
	// The drawable wall is the first 136 - 14 = 122 tiles.
	wall := deck[:TotalTiles-DeadWallSize]
	deadWall := deck[TotalTiles-DeadWallSize:] // Last 14 tiles

	players := make([]*Player, len(playerNames))
	initialDealerIndex := rand.Intn(len(playerNames)) // Randomly select the initial dealer

	winds := []string{"East", "South", "West", "North"}
	for i, name := range playerNames {
		// Assign seat wind based on the initial dealer
		// The dealer is East, then South, West, North in order of player index
		seatWindIndex := (i - initialDealerIndex + len(playerNames)) % len(playerNames)
		players[i] = &Player{
			Name:       name,
			Hand:       []Tile{},
			Discards:   []Tile{},
			Melds:      []Meld{},
			Score:      25000, // Starting score
			SeatWind:   winds[seatWindIndex],
			IsRiichi:   false,
			RiichiTurn: -1,
			IsIppatsu:  false,
			IsFuriten:  false,
		}
	}

	gs := &GameState{
		Wall:                 wall,
		DeadWall:             deadWall,
		Players:              players,
		CurrentPlayerIndex:   initialDealerIndex, // Current player to act; dealer starts.
		DealerIndexThisRound: initialDealerIndex, // Tracks who is dealer for this specific round.
		DiscardPile:          []Tile{},
		DoraIndicators:       []Tile{}, // Will hold revealed initial + Kan Dora
		UraDoraIndicators:    []Tile{}, // Initially hidden, populated on Riichi win reveal
		PrevalentWind:        "East",   // Assuming East round 1
		RoundNumber:          1,
		Honba:                0,
		RiichiSticks:         0,
		TurnNumber:           0,
		GamePhase:            PhaseDealing,
		InputReader:          bufio.NewReader(os.Stdin),
		LastDiscard:          nil, // Explicitly nil at start
		// Initialize new flags for Tenhou/Chihou/Renhou
		AnyCallMadeThisRound: false,
		IsFirstGoAround:      true, // Starts as true at the beginning of a round
		RoundWinner:          nil,  // Initialize RoundWinner
		// IsChankanOpportunity, IsRinshanWin, IsHouteiDiscard are already initialized to false by Go's default bool.
	}

	// Initialize new player flags
	for _, p := range gs.Players {
		p.HasMadeFirstDiscardThisRound = false
		p.HasDrawnFirstTileThisRound = false
	}

	gs.RevealInitialDoraIndicator()
	return gs
}

// RevealInitialDoraIndicator reveals the first dora indicator from the dead wall.
// Standard position: 3rd tile from the right end of the dead wall.
func (gs *GameState) RevealInitialDoraIndicator() {
	if len(gs.DeadWall) != DeadWallSize {
		fmt.Println("Error: Dead wall size incorrect for revealing initial Dora.")
		return
	}
	// Index calculation: DeadWall is 0..13. Right end is 13. 3rd from right is 11.
	indicatorIndex := DeadWallSize - 3
	indicator := gs.DeadWall[indicatorIndex]
	gs.DoraIndicators = append(gs.DoraIndicators, indicator)
	fmt.Printf("Revealed Dora Indicator: %s (From Dead Wall pos %d)\n", indicator.Name, indicatorIndex)
	// The corresponding Ura Dora is the tile below it (index + 1). Revealed only on Riichi win.
}

// DealInitialHands deals 13 tiles to each player from the wall.
func (gs *GameState) DealInitialHands() {
	numPlayers := len(gs.Players)
	dealerIndex := gs.CurrentPlayerIndex

	// Check if there are enough tiles for everyone (13 tiles * 4 players = 52 tiles)
	if len(gs.Wall) < HandSize*numPlayers {
		panic("Not enough tiles in wall to deal initial hands")
	}

	// Phase 1: Deal 4 tiles to each player, 3 times
	for i := 0; i < 3; i++ { // 3 passes
		for j := 0; j < numPlayers; j++ { // Each player
			playerIndex := (dealerIndex + j) % numPlayers
			for k := 0; k < 4; k++ { // 4 tiles
				if len(gs.Wall) == 0 {
					panic("Not enough tiles in wall during dealing (phase 1)")
				}
				tile := gs.Wall[0]
				gs.Wall = gs.Wall[1:]
				gs.Players[playerIndex].Hand = append(gs.Players[playerIndex].Hand, tile)
			}
		}
	}

	// Phase 2: Deal 1 tile to each player
	for j := 0; j < numPlayers; j++ { // Each player
		playerIndex := (dealerIndex + j) % numPlayers
		if len(gs.Wall) == 0 {
			panic("Not enough tiles in wall during dealing (phase 2)")
		}
		tile := gs.Wall[0]
		gs.Wall = gs.Wall[1:]
		gs.Players[playerIndex].Hand = append(gs.Players[playerIndex].Hand, tile)
	}

	// Sort initial hands
	for _, player := range gs.Players {
		sort.Sort(BySuitValue(player.Hand))
	}
	gs.GamePhase = PhasePlayerTurn // Transition to first player's turn
}

// DrawTile draws a tile from the wall for the current player.
// Returns the drawn tile and a boolean indicating if the wall is now empty.
func (gs *GameState) DrawTile() (Tile, bool) {
	if len(gs.Wall) == 0 {
		fmt.Println("Wall is empty!")
		// Trigger draw condition (Ryuukyoku) handled in main loop
		return Tile{}, true
	}
	tile := gs.Wall[0]
	gs.Wall = gs.Wall[1:] // Consume tile from wall

	player := gs.Players[gs.CurrentPlayerIndex]
	player.Hand = append(player.Hand, tile)
	sort.Sort(BySuitValue(player.Hand)) // Keep hand sorted

	// Drawing any tile breaks Ippatsu eligibility
	for _, p := range gs.Players {
		p.IsIppatsu = false
	}

	return tile, false
}

// DrawRinshanTile draws a replacement tile from the dead wall after a Kan.
// Returns the drawn tile and a boolean indicating if no Rinshan tile was available.
func (gs *GameState) DrawRinshanTile() (Tile, bool) {
	// Rinshan tiles are taken from the "left" end (index 0, 1, 2, 3) of the dead wall.
	// Check if any Rinshan tiles are left *before* consuming Dora indicators.
	// The number of revealed Kan Doras tells us how many Rinshan tiles were taken.
	numKanDeclared := 0
	for _, p := range gs.Players {
		for _, m := range p.Melds {
			if m.Type == "Ankan" || m.Type == "Daiminkan" || m.Type == "Shouminkan" {
				numKanDeclared++
			}
		}
	}

	if numKanDeclared >= RinshanTiles {
		fmt.Println("Error: No more Rinshan tiles left in Dead Wall!")
		// This might lead to an abortive draw if 4 kans were declared.
		return Tile{}, true // Indicate error/empty
	}

	// The next Rinshan tile is at index `numKanDeclared`.
	rinshanTile := gs.DeadWall[numKanDeclared]
	// We don't remove it from the slice, just note it's conceptually consumed.
	// The dead wall structure remains fixed, we just track usage.

	player := gs.Players[gs.CurrentPlayerIndex]
	player.Hand = append(player.Hand, rinshanTile)
	sort.Sort(BySuitValue(player.Hand))

	// Drawing Rinshan breaks Ippatsu
	player.IsIppatsu = false

	// Reveal Kan Dora Indicator *after* drawing Rinshan
	gs.RevealKanDoraIndicator() // This function increments the count used above implicitly

	return rinshanTile, false
}

// RevealKanDoraIndicator reveals a new dora indicator after a Kan.
func (gs *GameState) RevealKanDoraIndicator() {
	currentRevealedCount := len(gs.DoraIndicators)
	if currentRevealedCount >= MaxRevealedDora {
		fmt.Println("Maximum number of Dora indicators already revealed.")
		return
	}

	// Kan dora indicators are revealed from right-to-left, next to the previous ones.
	// Initial Dora is at DeadWallSize - 3 (index 11).
	// First Kan Dora is at DeadWallSize - 5 (index 9).
	// Second Kan Dora is at DeadWallSize - 7 (index 7), etc.
	indicatorIndex := DeadWallSize - 3 - (currentRevealedCount * 2)

	if indicatorIndex < RinshanTiles { // Ensure we don't reveal from the Rinshan area (indices 0-3)
		fmt.Printf("Error calculating Kan Dora index %d, potential overlap with Rinshan area.\n", indicatorIndex)
		return
	}
	if indicatorIndex < 0 {
		fmt.Println("Error: Calculated Kan Dora index is negative.")
		return
	}

	indicator := gs.DeadWall[indicatorIndex]
	gs.DoraIndicators = append(gs.DoraIndicators, indicator) // Add to the list of revealed indicators
	fmt.Printf("Revealed Kan Dora Indicator: %s (From Dead Wall pos %d)\n", indicator.Name, indicatorIndex)
	// Corresponding Ura Kan Dora is at index + 1
}

// RevealUraDoraIndicators reveals the Ura Dora indicators corresponding to revealed Dora/KanDora.
// Called only on a Riichi win.
func (gs *GameState) RevealUraDoraIndicators() {
	gs.UraDoraIndicators = []Tile{}           // Clear previous just in case
	numDoraRevealed := len(gs.DoraIndicators) // How many Dora/KanDora were flipped

	if numDoraRevealed == 0 {
		return // No Dora revealed, no Ura Dora
	}
	if len(gs.DeadWall) != DeadWallSize {
		fmt.Println("Error: Dead wall size incorrect for revealing Ura Dora.")
		return
	}

	fmt.Println("Revealing Ura Dora...")
	for i := 0; i < numDoraRevealed; i++ {
		// Index of the i-th revealed Dora/Kan Dora indicator (0=initial, 1=1st Kan, etc.)
		doraIndicatorIndex := DeadWallSize - 3 - (i * 2)
		uraIndicatorIndex := doraIndicatorIndex + 1 // Tile directly below it

		// Check validity of index
		if uraIndicatorIndex >= 0 && uraIndicatorIndex < DeadWallSize && uraIndicatorIndex >= RinshanTiles {
			uraIndicator := gs.DeadWall[uraIndicatorIndex]
			gs.UraDoraIndicators = append(gs.UraDoraIndicators, uraIndicator)
			fmt.Printf(" - Ura Dora Indicator: %s (From Dead Wall pos %d)\n", uraIndicator.Name, uraIndicatorIndex)
		} else {
			fmt.Printf("Warning: Invalid or out-of-bounds Ura Dora index %d calculated for Dora at %d.\n", uraIndicatorIndex, doraIndicatorIndex)
		}
	}
	sort.Sort(BySuitValue(gs.UraDoraIndicators)) // Sort for consistent display if needed
}

// NextPlayer moves the turn to the next player in sequence.
func (gs *GameState) NextPlayer() {
	gs.CurrentPlayerIndex = (gs.CurrentPlayerIndex + 1) % len(gs.Players)
}

// GetPlayerIndex returns the index (0-3) of a given player object.
func (gs *GameState) GetPlayerIndex(player *Player) int {
	for i, p := range gs.Players {
		if p == player {
			return i
		}
	}
	return -1 // Should not happen
}
