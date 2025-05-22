package main

import (
	"bufio"
	"fmt"
	"strings"
)

// Game Phases
const (
	PhaseDealing      = "Dealing"
	PhasePlayerTurn   = "PlayerTurn"
	PhaseAwaitingCall = "AwaitingCall" // Maybe useful later for explicit call timing
	PhaseRoundEnd     = "RoundEnd"
	PhaseGameEnd      = "GameEnd"
)

// Tile represents a mahjong tile
type Tile struct {
	Suit  string // "Man", "Pin", "Sou", "Wind", "Dragon"
	Value int    // 1-9 for suits, 1-4 for Winds (E=1, S=2, W=3, N=4), 1-3 for Dragons (W=1, G=2, R=3)
	Name  string // User-friendly name, e.g., "Man 5", "East", "Red Dragon", "Red Pin 5"
	IsRed bool   // Is it a red five?
	ID    int    // Unique ID (0-135) for easy comparison/sorting if needed
}

// Meld represents an open or closed set of tiles (Chi, Pon, Kan)
type Meld struct {
	Type        string // "Chi", "Pon", "Ankan", "Daiminkan", "Shouminkan"
	Tiles       []Tile // Tiles in the meld, usually sorted
	CalledOn    Tile   // Which tile was called (for open melds) - For Shouminkan, it's the added tile.
	FromPlayer  int    // Index of the player the tile was called from (-1 for Ankan, Shouminkan uses original Pon source)
	IsConcealed bool   // True for Ankan
}

// Player represents a mahjong player
type Player struct {
	Name       string
	Hand       []Tile // Concealed part of the hand (sorted)
	Discards   []Tile // Tiles discarded by this player (in order)
	Melds      []Meld // Array of melded tile sets
	Score      int
	SeatWind   string // "East", "South", "West", "North"
	IsRiichi   bool   // Has declared Riichi
	RiichiTurn int    // Turn number Riichi was declared (-1 if not in Riichi)
	IsIppatsu  bool   // Eligible for Ippatsu (true between Riichi and next draw/call/own Kan)
	IsFuriten  bool   // Cannot Ron
	DeclaredDoubleRiichi bool // True if this player successfully declared Double Riichi
	// Add other state as needed (e.g., Menzenchin status - can be derived)
	HasMadeFirstDiscardThisRound bool // True if player has made their first discard in the current round
	HasDrawnFirstTileThisRound   bool // True if player has drawn their first tile in the current round
}

// RiichiOption stores details about a possible Riichi declaration
type RiichiOption struct {
	DiscardIndex int    // Index of the tile to discard in the 14-tile hand
	DiscardTile  Tile   // The tile to discard
	Waits        []Tile // List of tiles the hand will wait on after discard
}

// GameState represents the current game state
type GameState struct {
	Wall               []Tile // Remaining drawable tiles
	DeadWall           []Tile // 14 tiles: indicators + rinshan tiles
	Players            []*Player
	CurrentPlayerIndex int
	DiscardPile        []Tile        // All discarded tiles in order across all players (rarely needed directly)
	DoraIndicators     []Tile        // Revealed dora indicators (initial + Kan)
	UraDoraIndicators  []Tile        // Revealed only on Riichi win
	PrevalentWind      string        // "East" or "South" typically
	RoundNumber        int           // Dealer round (1-4 for East, 1-4 for South etc.)
	Honba              int           // Number of repeat rounds/counters
	RiichiSticks       int           // Number of 1000-point Riichi sticks on the table
	TurnNumber         int           // Overall turn number in the round (increments on discard)
	LastDiscard        *Tile         // Reference to the very last discarded tile by any player
	GamePhase          string        // e.g., PhaseDealing, PhasePlayerTurn, PhaseRoundEnd
	InputReader        *bufio.Reader // For reading user input

	// Flags for specific Yaku conditions
	IsChankanOpportunity bool // True if a player has just declared Shouminkan and the tile is available for Ron
	IsRinshanWin         bool // True if the current Tsumo check is for a Rinshan tile draw
	IsHouteiDiscard      bool // True if the current discard is the one immediately after the last wall tile was drawn
	AnyCallMadeThisRound bool // True if any player has made a Chi, Pon, Daiminkan, or Shouminkan this round
	IsFirstGoAround      bool // True until a player completes their first discard OR a call is made
}

// --- Sorting Tiles ---

// BySuitValue implements sort.Interface for []Tile based on suit then value
type BySuitValue []Tile

func (a BySuitValue) Len() int      { return len(a) }
func (a BySuitValue) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a BySuitValue) Less(i, j int) bool {
	suitOrder := map[string]int{"Man": 1, "Pin": 2, "Sou": 3, "Wind": 4, "Dragon": 5}
	s1 := a[i].Suit
	s2 := a[j].Suit
	v1 := a[i].Value
	v2 := a[j].Value

	order1, ok1 := suitOrder[s1]
	order2, ok2 := suitOrder[s2]

	if !ok1 || !ok2 { // Handle potential unexpected suits gracefully
		// Put unknown suits at the end? Or based on name?
		return fmt.Sprintf("%s%d", s1, v1) < fmt.Sprintf("%s%d", s2, v2)
	}

	if order1 != order2 {
		return order1 < order2
	}

	// Same suit
	if s1 == "Wind" || s1 == "Dragon" {
		// Use canonical order for honors, not necessarily Value
		nameOrder := map[string]int{
			"East": 1, "South": 2, "West": 3, "North": 4,
			"White": 5, "Green": 6, "Red": 7,
		}
		// Handle potential Red 5 name differences if sorting includes Name (shouldn't affect honors)
		nameI := strings.TrimPrefix(a[i].Name, "Red ")
		nameJ := strings.TrimPrefix(a[j].Name, "Red ")
		orderNameI, okI := nameOrder[nameI]
		orderNameJ, okJ := nameOrder[nameJ]
		if okI && okJ {
			return orderNameI < orderNameJ
		}
		// Fallback if names aren't standard
		return a[i].Name < a[j].Name
	}

	// For numbered suits, sort by value
	return v1 < v2
}
