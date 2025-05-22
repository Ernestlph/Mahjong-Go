package main

import (
	"bufio"
	"fmt"
)

// Game Phases
const (
	PhaseDealing      = "Dealing"
	PhasePlayerTurn   = "PlayerTurn"
	PhaseAwaitingCall = "AwaitingCall" // May not be explicitly used if calls handled within PlayerTurn
	PhaseRoundEnd     = "RoundEnd"
	PhaseGameEnd      = "GameEnd"
)

// Constants for Riichi Mahjong Rules
const (
	HandSize        = 13  // Tiles in a complete hand before the 14th winning tile
	DeadWallSize    = 14  // Total tiles in the dead wall
	RinshanTiles    = 4   // Number of replacement tiles for Kans in the dead wall
	MaxRevealedDora = 5   // Max number of Dora indicators (1 initial + 4 Kan) that can be revealed
	TotalTiles      = 136 // 4 * (9*3 suits + 7 honors) = 4 * (27 + 7) = 4 * 34 = 136

	InitialScore = 25000 // Standard starting score
	RiichiBet    = 1000  // Points bet for Riichi

	// Noten Bappu Constants (Standard Values for 4 players)
	NotenBappuTotal       = 3000 // Total points exchanged
	NotenBappuPayment1T3N = 1000 // Each of 3 Noten pays 1000 to 1 Tenpai
	NotenBappuPayment2T2N = 1500 // Each of 2 Noten pays 1500 (split among 2 Tenpai)
	NotenBappuPayment3T1N = 3000 // The 1 Noten pays 3000 (split among 3 Tenpai)
	// Derived gains:
	// 1 Tenpai gains 3000.
	// 2 Tenpai gain 1500 each.
	// 3 Tenpai gain 1000 each.

	RyanhanShibariHonbaThreshold = 5 // Honba count at which 2-han minimum (excluding Dora) applies
)

// Tile represents a mahjong tile
type Tile struct {
	Suit  string // "Man", "Pin", "Sou", "Wind", "Dragon"
	Value int    // 1-9 for suits; Winds: E=1,S=2,W=3,N=4; Dragons: W=1(Haku),G=2(Hatsu),R=3(Chun)
	Name  string // User-friendly name, e.g., "Man 5", "East", "Red Dragon", "Red Pin 5"
	IsRed bool   // True if this tile is a red five
	ID    int    // Unique ID (0-135) for exact tile instance comparison
}

// Meld represents an open or closed set of tiles (Chi, Pon, Kan)
type Meld struct {
	Type        string // "Chi", "Pon", "Ankan", "Daiminkan", "Shouminkan"
	Tiles       []Tile // Tiles in the meld, should be sorted for consistency
	CalledOn    Tile   // Which tile was called (for open melds like Pon, Chi, Daiminkan). For Shouminkan, it's the tile added to the Pon.
	FromPlayer  int    // Index of the player the tile was called from (-1 for Ankan). For Shouminkan, this refers to the player the original Pon was called from.
	IsConcealed bool   // True for Ankan (concealed Kan)
}

// Player represents a mahjong player
type Player struct {
	Name                         string
	Hand                         []Tile // Concealed part of the hand (should be kept sorted)
	Discards                     []Tile // Tiles discarded by this player (in order of discard)
	Melds                        []Meld // Array of melded tile sets
	Score                        int
	SeatWind                     string  // Player's current seat wind ("East", "South", "West", "North")
	IsRiichi                     bool    // True if player has declared Riichi
	RiichiTurn                   int     // Turn number (within the round) Riichi was declared (-1 if not in Riichi)
	IsIppatsu                    bool    // True if eligible for Ippatsu (win within one turn cycle of Riichi, no interruptions)
	IsFuriten                    bool    // General Furiten status (due to own discards matching waits, or recently missed Ron)
	IsPermanentRiichiFuriten     bool    // True if in Riichi and missed a Ron on a declared wait tile
	DeclinedRonOnTurn            int     // Turn number (within round) when player last declined a Ron option (-1 if none)
	DeclinedRonTileID            int     // ID of the tile on which Ron was declined (-1 if none)
	RiichiDeclaredWaits          []Tile  // Slice of tile *types* the player is waiting on if Riichi declared
	DeclaredDoubleRiichi         bool    // True if this player successfully declared Double Riichi this round
	HasMadeFirstDiscardThisRound bool    // True if player has made their first discard in the current round (for Renhou/Chihou)
	HasDrawnFirstTileThisRound   bool    // True if player has drawn their first tile in the current round (for Tenhou/Chihou/Kyuushuu)
	HasHadDiscardCalledThisRound bool    // True if any of this player's discards in the current round were called for an open meld (for Nagashi Mangan)
	JustDrawnTile                *Tile   // Pointer to the tile most recently drawn by this player (nil otherwise)
	IsTenpai                     bool    // Status at Ryuukyoku (exhaustive draw)
	PaoTargetFor                 *Player // If this player's call caused another player (`PaoTargetFor`) to win a Yakuman (Pao liability)
	PaoSourcePlayerIndex         int     // Index of player who is Pao for this player's Yakuman (-1 if none this player is the target)
	InitialTurnOrder             int     // Player's fixed turn order index at the start of the game (0-3), used for Ssuufon Renda.
}

// RiichiOption stores details about a possible Riichi declaration
type RiichiOption struct {
	DiscardIndex int    // Index of the tile to discard in the player's 14-tile hand
	DiscardTile  Tile   // The actual tile to discard
	Waits        []Tile // List of tile *types* the hand will wait on after this discard
}

// GameState represents the current game state
type GameState struct {
	Wall                 []Tile        // Remaining drawable tiles in the live wall
	DeadWall             []Tile        // 14 tiles: Dora/Ura/Kan Dora indicators + Rinshan replacement tiles
	Players              []*Player     // Slice of all players in the game
	CurrentPlayerIndex   int           // Index of the player whose turn it is currently
	DealerIndexThisRound int           // Index of the player who is the dealer for the current round
	DiscardPile          []Tile        // All discarded tiles in order across all players (rarely used directly now, player.Discards is primary)
	DoraIndicators       []Tile        // Revealed Dora indicators (initial + Kan Doras)
	UraDoraIndicators    []Tile        // Revealed Ura Dora indicators (only on Riichi win)
	PrevalentWind        string        // Current prevalent wind ("East", "South", "West", "North")
	RoundNumber          int           // Round number within the current Prevalent Wind (e.g., East 1, East 2, ..., South 1)
	DealerRoundCount     int           // How many consecutive rounds the current dealer has held dealership (for Renchan display)
	Honba                int           // Number of repeat rounds/counters on the table (adds to win value)
	RiichiSticks         int           // Number of 1000-point Riichi sticks on the table
	TurnNumber           int           // Overall turn number *within the current round* (increments on each discard)
	LastDiscard          *Tile         // Pointer to the very last tile discarded by any player
	GamePhase            string        // Current phase of the game (e.g., PhaseDealing, PhasePlayerTurn)
	InputReader          *bufio.Reader // For reading user input from console

	// Flags for specific Yaku conditions and game state tracking
	IsChankanOpportunity        bool         // True if a Shouminkan is declared and available for Chankan Ron
	IsRinshanWin                bool         // True if the current Tsumo check is for a Rinshan tile draw
	IsHouteiDiscard             bool         // True if the current discard is the one immediately after the last wall tile was drawn (Haitei)
	AnyCallMadeThisRound        bool         // True if any player has made a Chi, Pon, Daiminkan, or Shouminkan this round
	IsFirstGoAround             bool         // True until a player completes their first discard OR a call is made this round
	RoundWinner                 *Player      // Tracks the winner of the round, nil if draw/abort
	FirstTurnDiscards           [4]Tile      // Stores the first un-interrupted discard of each player (by InitialTurnOrder index) for Ssuufon Renda
	FirstTurnDiscardCount       int          // Count of players who have made their first un-interrupted discard
	DeclaredRiichiPlayerIndices map[int]bool // Tracks which player indices have declared Riichi this round (for Suu Riichi)
	TotalKansDeclaredThisRound  int          // Total number of Kans (any type) declared in the current round (for Suukaikan)
	MaxWindRounds               int          // Number of prevalent winds to play (e.g., 2 for East-South game, 1 for East-only)
	CurrentWindRoundNumber      int          // Tracks which wind round it is (1 for East, 2 for South, etc.)
	SanchahouRonners            []*Player    // Stores players who declared Ron on the same discard (for Sanchahou check)
	GameLog                     []string     // Log of major game events
}

// --- Sorting Tiles ---

// BySuitValue implements sort.Interface for []Tile based on suit then value, then ID for stability.
type BySuitValue []Tile

func (a BySuitValue) Len() int      { return len(a) }
func (a BySuitValue) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a BySuitValue) Less(i, j int) bool {
	suitOrder := map[string]int{"Man": 1, "Pin": 2, "Sou": 3, "Wind": 4, "Dragon": 5}
	s1, v1, id1 := a[i].Suit, a[i].Value, a[i].ID
	s2, v2, id2 := a[j].Suit, a[j].Value, a[j].ID

	order1, ok1 := suitOrder[s1]
	order2, ok2 := suitOrder[s2]

	if !ok1 || !ok2 { // Should not happen with valid tiles
		return fmt.Sprintf("%s%d%d", s1, v1, id1) < fmt.Sprintf("%s%d%d", s2, v2, id2)
	}

	if order1 != order2 {
		return order1 < order2
	} // Different suits

	// Same suit
	if v1 != v2 {
		return v1 < v2
	} // Different values

	return id1 < id2 // Same suit and value, sort by ID for stability (e.g. for red fives if names are same)
}

// --- Tile Property Helpers ---

// IsTerminal checks if a tile is a terminal tile (1 or 9 of a numbered suit).
func IsTerminal(tile Tile) bool {
	return (tile.Suit == "Man" || tile.Suit == "Pin" || tile.Suit == "Sou") && (tile.Value == 1 || tile.Value == 9)
}

// IsHonor checks if a tile is an honor tile (Wind or Dragon).
func IsHonor(tile Tile) bool {
	return tile.Suit == "Wind" || tile.Suit == "Dragon"
}

// IsTerminalOrHonor checks if a tile is a terminal or an honor tile.
func IsTerminalOrHonor(tile Tile) bool {
	return IsTerminal(tile) || IsHonor(tile)
}

// IsSimple checks if a tile is a "simple" tile (numbered suit, 2 through 8).
func IsSimple(tile Tile) bool {
	return (tile.Suit == "Man" || tile.Suit == "Pin" || tile.Suit == "Sou") && (tile.Value >= 2 && tile.Value <= 8)
}
