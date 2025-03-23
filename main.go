package main

import (
	"fmt"
	"math/rand"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Tile represents a mahjong tile
type Tile struct {
	Suit  string
	Value int
	Name  string
}

// Helper function to check if a slice contains a value
func contains[T comparable](slice []T, val T) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
}

// Function to create a new tile
func NewTile(suit string, value int, name string) Tile {
	return Tile{Suit: suit, Value: value, Name: name}
}

// Function to generate a standard mahjong deck
func GenerateDeck() []Tile {
	var deck []Tile
	suits := []string{"Man", "Pin", "Sou", "Honors"}
	honorTiles := []string{"East", "South", "West", "North", "Green", "Red", "White"}
	redDoraValues := []int{5} // Values that can be red dora

	// Create numbered tiles (1-9 for Man, Pin, Sou)
	for _, suit := range suits[:3] { // Man, Pin, Sou
		for value := 1; value <= 9; value++ {
			for i := 0; i < 4; i++ { // 4 of each tile
				tileName := fmt.Sprintf("%s %d", suit, value)
				if contains(redDoraValues, value) && i == 0 { // Make one of each 5-tile a red dora
					tileName = "Red " + tileName
				}
				deck = append(deck, NewTile(suit, value, tileName))
			}
		}
	}

	// Create honor tiles (winds and dragons)
	for _, honor := range honorTiles {
		for i := 0; i < 4; i++ { // 4 of each tile
			deck = append(deck, NewTile("Honors", 0, honor)) // Value 0 for honor tiles
		}
	}

	// Add dora indicators and kan dora indicators (initially empty, will be revealed during game)
	for i := 0; i < 4; i++ { // 4 dora indicators and kan dora indicators
		deck = append(deck, NewTile("Dora", 0, "Dora Indicator"))
		deck = append(deck, NewTile("Kan Dora", 0, "Kan Dora Indicator"))
	}

	// Shuffle the deck
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(deck), func(i, j int) {
		deck[i], deck[j] = deck[j], deck[i]
	})

	return deck
}

// Helper function to check if a slice contains a value
func containsTile(tiles []Tile, target Tile) bool {
	for _, t := range tiles {
		if t.Suit == target.Suit && t.Value == target.Value {
			return true
		}
	}
	return false
}

func main() {
	fmt.Println("Starting Riichi Mahjong Game in Go!")

	// Initialize game state
	playerNames := []string{"Player1", "Player2", "Player3", "Player4"}
	gameState := NewGameState(playerNames)

	// Deal initial hands
	gameState.DealInitialHands()

	// Game loop
	for {
		currentPlayer := gameState.Players[gameState.currentPlayerIndex]
		fmt.Printf("\n--- %s's Turn ---\n", currentPlayer.Name)

		// Display game state
		DisplayGameState(gameState)

		// Display current player's hand
		formattedHand := FormatHandForDisplay(currentPlayer.Hand)
		fmt.Println("Your Hand:", formattedHand)

		// Player draws a tile
		drawnTile := gameState.DrawTile()
		fmt.Println("Drawn Tile:", drawnTile.Name)
		formattedNewHand := FormatHandForDisplay(currentPlayer.Hand) // Format hand again after drawing
		fmt.Println("New Hand:", formattedNewHand)

		// Player discards a tile
		if len(currentPlayer.Hand) > 0 {
			discardIndex := GetPlayerDiscardChoice(currentPlayer)
			discardedTile := gameState.DiscardTile(discardIndex)
			fmt.Println("Discarded Tile:", discardedTile.Name)
		}

		gameState.NextPlayer()        // Move to the next player's turn
		if len(gameState.Wall) == 0 { // Basic game end condition (wall runs out)
			fmt.Println("\nGame Over! Wall has run out.")
			break
		}
	}

	fmt.Println("\n--- Final Game State ---")
	DisplayGameState(gameState)
}

// Player represents a mahjong player
type Player struct {
	Name     string
	Hand     []Tile
	Discards []Tile
	Melds    [][]Tile // Array of melded tile sets
}

// GameState represents the current game state
type GameState struct {
	Deck               []Tile
	DeadWall           []Tile // Tiles for rinchans, dora indicators, etc.
	Wall               []Tile // Main wall (remaining tiles after dead wall)
	Players            []*Player
	currentPlayerIndex int
	DiscardPile        []Tile
	DoraIndicators     []Tile
	KanDoraIndicators  []Tile
}

// GetPungChoice prompts a player to choose whether to pung or not
func GetPungChoice(player *Player, discardedTile Tile) bool {
	fmt.Printf("\nPlayer %s, you can Pung %s. Do you want to Pung? (y/n): ", player.Name, discardedTile.Name)
	var input string
	_, err := fmt.Scanln(&input)
	if err != nil {
		fmt.Println("Invalid input. Assuming no Pung.")
		return false
	}
	return strings.ToLower(input) == "y"
}

// NewGameState initializes a new game state
func NewGameState(playerNames []string) *GameState {
	deck := GenerateDeck()
	deadWallSize := 14 // Standard dead wall size in Riichi
	deadWall := deck[:deadWallSize]
	wall := deck[deadWallSize:]

	players := make([]*Player, len(playerNames))
	for i, name := range playerNames {
		players[i] = &Player{Name: name, Hand: []Tile{}, Discards: []Tile{}, Melds: [][]Tile{}}
	}
	gs := &GameState{
		Deck:               deck, // Keep full deck for debugging/reference
		DeadWall:           deadWall,
		Wall:               wall,
		Players:            players,
		currentPlayerIndex: 0,
		DiscardPile:        []Tile{},
		DoraIndicators:     []Tile{},
		KanDoraIndicators:  []Tile{},
	}
	gs.RevealDoraIndicator() // Reveal initial dora indicator
	return gs
}

// RevealDoraIndicator reveals a dora indicator from the dead wall
func (gs *GameState) RevealDoraIndicator() {
	if len(gs.DeadWall) < 4 {
		fmt.Println("Dead wall is too small to reveal dora indicator!")
		return // Handle error appropriately later
	}
	indicator := gs.DeadWall[len(gs.DeadWall)-1-len(gs.DoraIndicators)] // Get the next dora indicator from dead wall
	gs.DoraIndicators = append(gs.DoraIndicators, indicator)
	fmt.Println("Revealed Dora Indicator:", indicator.Name) // Debugging
}

// DealInitialHands deals 13 tiles to each player
func (gs *GameState) DealInitialHands() {
	tilesPerHand := 13
	for _, player := range gs.Players {
		player.Hand = gs.Wall[:tilesPerHand]
		gs.Wall = gs.Wall[tilesPerHand:]
	}
}

// DrawTile draws a tile from the wall for the current player
func (gs *GameState) DrawTile() Tile {
	if len(gs.Wall) == 0 {
		fmt.Println("Wall is empty!") // Handle wall empty scenario later
		return Tile{}                 // Return empty tile for now
	}
	tile := gs.Wall[0]
	gs.Wall = gs.Wall[1:]
	gs.Players[gs.currentPlayerIndex].Hand = append(gs.Players[gs.currentPlayerIndex].Hand, tile)
	return tile
}

// DrawRinshanTile draws a replacement tile from the dead wall (for kans)
func (gs *GameState) DrawRinshanTile() Tile {
	if len(gs.DeadWall) < 4 {
		fmt.Println("Dead wall is too small to draw rinshan tile!")
		return Tile{} // Handle error appropriately later
	}
	rinshanTile := gs.DeadWall[0]
	gs.DeadWall = gs.DeadWall[1:]
	gs.Players[gs.currentPlayerIndex].Hand = append(gs.Players[gs.currentPlayerIndex].Hand, rinshanTile)
	return rinshanTile
}

// DiscardTile allows the current player to discard a tile
func (gs *GameState) DiscardTile(tileIndex int) Tile {
	if tileIndex < 0 || tileIndex >= len(gs.Players[gs.currentPlayerIndex].Hand) {
		fmt.Println("Invalid tile index to discard.")
		return Tile{} // Return empty tile for now
	}
	discardedTile := gs.Players[gs.currentPlayerIndex].Hand[tileIndex]
	gs.Players[gs.currentPlayerIndex].Discards = append(gs.Players[gs.currentPlayerIndex].Discards, discardedTile)
	gs.Players[gs.currentPlayerIndex].Hand = append(gs.Players[gs.currentPlayerIndex].Hand[:tileIndex], gs.Players[gs.currentPlayerIndex].Hand[tileIndex+1:]...)
	gs.DiscardPile = append(gs.DiscardPile, discardedTile)

	// Check for Pung/Ron after discard
	pungPlayers := []*Player{}
	for i, player := range gs.Players {
		if i != gs.currentPlayerIndex { // Check for other players
			if gs.CheckPung(player, discardedTile) {
				pungPlayers = append(pungPlayers, player)
			}
			if gs.CheckRon(player, discardedTile) {
				fmt.Printf("\nPlayer %s can Ron %s. (Ron implementation pending)\n", player.Name, discardedTile.Name)
			}
		}
	}

	// Handle Punging
	if len(pungPlayers) > 0 {
		for _, player := range pungPlayers {
			if GetPungChoice(player, discardedTile) {
				fmt.Printf("\nPlayer %s chose to Pung %s!\n", player.Name, discardedTile.Name)
				gs.HandlePungAction(player, discardedTile)
				return discardedTile // IMPORTANT: Exit DiscardTile after Pung is handled
			} else { // Check for Chow if Pung is not called
				chowType := gs.GetChowChoice(gs.Players[gs.currentPlayerIndex], discardedTile)
				if chowType > 0 {
					fmt.Printf("\nPlayer %s chose to Chow %s!\n", gs.Players[gs.currentPlayerIndex].Name, discardedTile.Name)
					gs.HandleChowAction(gs.Players[gs.currentPlayerIndex], discardedTile, chowType)
					return discardedTile // IMPORTANT: Exit DiscardTile after Chow is handled
				}
			}
		}
	}

	return discardedTile
}

// HandleChowAction processes the Chow action
func (gs *GameState) HandleChowAction(player *Player, discardedTile Tile, chowType int) {
	var tilesToRemove []Tile
	switch chowType {
	case 1: // discardedTile, discardedTile+1, discardedTile+2
		tilesToRemove = []Tile{
			NewTile(discardedTile.Suit, discardedTile.Value+1, ""),
			NewTile(discardedTile.Suit, discardedTile.Value+2, ""),
		}
	case 2: // discardedTile-1, discardedTile, discardedTile+1
		tilesToRemove = []Tile{
			NewTile(discardedTile.Suit, discardedTile.Value-1, ""),
			NewTile(discardedTile.Suit, discardedTile.Value+1, ""),
		}
	case 3: // discardedTile-2, discardedTile-1, discardedTile
		tilesToRemove = []Tile{
			NewTile(discardedTile.Suit, discardedTile.Value-2, ""),
			NewTile(discardedTile.Suit, discardedTile.Value-1, ""),
		}
	default:
		fmt.Println("Invalid Chow type")
		return
	}

	meld := []Tile{discardedTile}
	handTilesToRemoveIndices := []int{}
	removedCount := 0

	// Find and remove tiles from hand
	for _, toRemoveTile := range tilesToRemove {
		for i := 0; i < len(player.Hand); i++ {
			if removedCount < 2 && player.Hand[i].Suit == toRemoveTile.Suit && player.Hand[i].Value == toRemoveTile.Value {
				meld = append(meld, player.Hand[i])
				handTilesToRemoveIndices = append(handTilesToRemoveIndices, i)
				removedCount++
				break // Break inner loop after finding and marking a tile
			}
		}
	}

	if removedCount == 2 { // Corrected line: was 'if pungCount == 2 {' but should be 'if removedCount == 2 {'
		// Remove chow tiles from hand
		var updatedHand []Tile
		sort.Ints(handTilesToRemoveIndices) // Sort indices to remove in descending order
		offset := 0
		for i := 0; i < len(player.Hand); i++ {
			isRemoved := false
			for _, removeIndex := range handTilesToRemoveIndices {
				if removeIndex == i-offset {
					isRemoved = true
					break
				}
			}
			if !isRemoved {
				updatedHand = append(updatedHand, player.Hand[i])
			} else {
				offset++
			}
		}
		player.Hand = updatedHand

		// Add chow meld

		player.Melds = append(player.Melds, meld)
		fmt.Printf("\nPlayer %s chowed %s. Melds: %v, Hand: %s\n", player.Name, discardedTile.Name, player.Melds, FormatHandForDisplay(player.Hand))

		// Current player becomes the chowing player
		gs.currentPlayerIndex = gs.GetPlayerIndex(player)
	} else {
		fmt.Println("Error: Could not find 2 tiles for chow in player's hand.") // Should not happen based on CheckChow
	}
}

// GetPlayerIndex returns the index of a player in the game state
func (gs *GameState) GetPlayerIndex(player *Player) int {
	for i, p := range gs.Players {
		if p == player {
			return i
		}
	}
	return -1 // Should not happen if player is valid
}

// NextPlayer moves to the next player's turn
func (gs *GameState) NextPlayer() {
	gs.currentPlayerIndex = (gs.currentPlayerIndex + 1) % len(gs.Players)
}

// FormatHandForDisplay formats a player's hand for terminal output
func FormatHandForDisplay(hand []Tile) string {
	sort.Slice(hand, func(i, j int) bool {
		suitOrder := []string{"Man", "Pin", "Sou", "Honors", "Dora", "Kan Dora"}
		s1 := hand[i].Suit
		s2 := hand[j].Suit
		v1 := hand[i].Value
		v2 := hand[j].Value

		suitIndex1 := -1
		suitIndex2 := -1

		for index, suit := range suitOrder {
			if suit == s1 {
				suitIndex1 = index
			}
			if suit == s2 {
				suitIndex2 = index
			}
		}

		if suitIndex1 != suitIndex2 {
			return suitIndex1 < suitIndex2
		}
		return v1 < v2
	})

	var tileNames []string
	for _, tile := range hand {
		tileNames = append(tileNames, tile.Name)
	}
	return strings.Join(tileNames, ", ")
}

// DisplayGameState outputs the current game state to the terminal
func DisplayGameState(gs *GameState) {
	fmt.Println("\n--- Game State ---")
	for i, player := range gs.Players {
		formattedHand := FormatHandForDisplay(player.Hand)
		fmt.Printf("%s's Hand: %s\n", player.Name, formattedHand)
		fmt.Printf("%s's Discards: %v\n", player.Name, gs.Players[i].Discards)
		fmt.Printf("%s's Melds: %v\n", player.Name, gs.Players[i].Melds)
	}
	fmt.Printf("Discard Pile: %v\n", gs.DiscardPile)
	fmt.Printf("Dora Indicators: %v\n", gs.DoraIndicators)
	fmt.Printf("Wall Size: %d, Dead Wall Size: %d\n", len(gs.Wall), len(gs.DeadWall))
}

// GetPlayerDiscardChoice prompts the current player to choose a tile to discard
func GetPlayerDiscardChoice(player *Player) int {
	fmt.Println("\nYour hand:")
	for i, tile := range player.Hand {
		fmt.Printf("[%d] %s  ", i+1, tile.Name)
	}
	fmt.Println("\nChoose a tile to discard (enter the number):")

	var input string
	_, err := fmt.Scanln(&input)
	if err != nil {
		fmt.Println("Invalid input. Please enter a number.")
		return GetPlayerDiscardChoice(player)
	}

	tileIndex, err := strconv.Atoi(input)
	if err != nil || tileIndex < 1 || tileIndex > len(player.Hand) {
		fmt.Println("Invalid tile number. Please choose a number from the list.")
		return GetPlayerDiscardChoice(player)
	}
	return tileIndex - 1
}

// CheckPung checks if a player can call pung on a discarded tile
func (gs *GameState) CheckPung(player *Player, discardedTile Tile) bool {
	pungCount := 0
	for _, tile := range player.Hand {
		if tile.Suit == discardedTile.Suit && tile.Value == discardedTile.Value {
			pungCount++
		}
	}
	return pungCount >= 2
}

// CheckChow checks if a player can call chow on a discarded tile
func (gs *GameState) CheckChow(player *Player, discardedTile Tile) bool {
	if discardedTile.Suit == "Honors" {
		return false
	}
	playerIndex := gs.GetPlayerIndex(player)
	requiredDiscarderIndex := (playerIndex - 1 + len(gs.Players)) % len(gs.Players)
	if gs.currentPlayerIndex != requiredDiscarderIndex {
		return false
	}

	chowCount := 0

	// Check for sequences (example checks)
	possibleValues := [][]int{
		{discardedTile.Value, discardedTile.Value + 1, discardedTile.Value + 2},
		{discardedTile.Value - 1, discardedTile.Value, discardedTile.Value + 1},
		{discardedTile.Value - 2, discardedTile.Value - 1, discardedTile.Value},
	}

	for _, values := range possibleValues {
		hasValues := true
		for _, v := range values {
			found := false
			for _, tile := range player.Hand {
				if tile.Suit == discardedTile.Suit && tile.Value == v {
					found = true
					break
				}
			}
			if !found {
				hasValues = false
				break
			}
		}
		if hasValues {
			chowCount++
		}
	}

	return chowCount > 0
}

// GetChowChoice prompts player to choose chow type (1-3 for sequence options)
func (gs *GameState) GetChowChoice(player *Player, discardedTile Tile) int {
	fmt.Printf("\nPlayer %s can Chow %s. Choose sequence type:\n", player.Name, discardedTile.Name)

	// Find possible chow combinations
	possibleSequences := gs.FindPossibleChowSequences(player, discardedTile)

	// List options
	for i, seq := range possibleSequences {
		fmt.Printf("[%d] %s\n", i+1, FormatSequence(seq))
	}
	fmt.Print("Enter choice (0 to cancel): ")

	var input string
	fmt.Scanln(&input)
	choice, err := strconv.Atoi(input)
	if err != nil || choice < 0 || choice > len(possibleSequences) {
		fmt.Println("Invalid choice. Canceling chow.")
		return 0
	}
	return choice
}

// Helper to format sequence for display
func FormatSequence(tiles []Tile) string {
	var names []string
	for _, t := range tiles {
		names = append(names, t.Name)
	}
	return strings.Join(names, "-")
}

// FindPossibleChowSequences identifies valid chow sequences
func (gs *GameState) FindPossibleChowSequences(player *Player, discardedTile Tile) [][]Tile {
	var sequences [][]Tile
	if discardedTile.Suit == "Honors" {
		return sequences
	}

	// Check possible sequences requiring the discarded tile
	requiredValues := [][]int{
		{discardedTile.Value - 2, discardedTile.Value - 1}, // X, X+1, discard
		{discardedTile.Value - 1, discardedTile.Value + 1}, // X, discard, X+2
		{discardedTile.Value + 1, discardedTile.Value + 2}, // discard, X+1, X+2
	}

	for _, vals := range requiredValues {
		if isValidSequence(vals, discardedTile.Suit, player.Hand) {
			seq := []Tile{
				findTile(player.Hand, discardedTile.Suit, vals[0]),
				findTile(player.Hand, discardedTile.Suit, vals[1]),
				discardedTile,
			}
			sequences = append(sequences, seq)
		}
	}
	return sequences
}

// Helper functions
func isValidSequence(values []int, suit string, hand []Tile) bool {
	if values[0] < 1 || values[1] > 9 {
		return false
	}

	count := 0
	for _, v := range values {
		if hasTile(hand, suit, v) {
			count++
		}
	}
	return count == 2
}

func hasTile(hand []Tile, suit string, value int) bool {
	for _, t := range hand {
		if t.Suit == suit && t.Value == value {
			return true
		}
	}
	return false
}

func findTile(hand []Tile, suit string, value int) Tile {
	for _, t := range hand {
		if t.Suit == suit && t.Value == value {
			return t
		}
	}
	return Tile{}
}

// HandlePungAction handles the Pung action when a player declares Pung
func (gs *GameState) HandlePungAction(player *Player, discardedTile Tile) {
	pungTiles := []Tile{}
	pungCount := 0

	// Find 2 tiles in player's hand matching discardedTile
	indicesToRemove := []int{}
	for i := 0; i < len(player.Hand); i++ {
		if player.Hand[i].Suit == discardedTile.Suit && player.Hand[i].Value == discardedTile.Value && player.Hand[i].Name == discardedTile.Name {
			pungTiles = append(pungTiles, player.Hand[i])
			indicesToRemove = append(indicesToRemove, i)
			pungCount++
			if pungCount == 2 {
				break // Found 2 tiles for pung
			}
		}
	}

	if pungCount == 2 {
		// Remove pung tiles from hand
		var updatedHand []Tile
		removedCount := 0
		for i := 0; i < len(player.Hand); i++ {
			removeIndex := -1
			for _, indexToRemove := range indicesToRemove {
				if indexToRemove == i {
					removeIndex = indexToRemove
					break
				}
			}
			if removeIndex == -1 && removedCount < 2 {
				updatedHand = append(updatedHand, player.Hand[i])
			} else {
				removedCount++
			}
		}
		player.Hand = updatedHand

		// Add pung meld
		meld := append([]Tile{discardedTile}, pungTiles...)
		player.Melds = append(player.Melds, meld)

		fmt.Printf("\nPlayer %s punged %s. Melds: %v, Hand: %s\n", player.Name, discardedTile.Name, player.Melds, FormatHandForDisplay(player.Hand))

		// Player who pung'd becomes current player (for next turn's discard)
		gs.currentPlayerIndex = gs.GetPlayerIndex(player)
	} else {
		fmt.Println("Error: Could not find 2 tiles for pung in player's hand.") // Should not happen based on CheckPung
	}
}

// CheckRon checks if a player can Ron on a discarded tile (basic check for now)
func (gs *GameState) CheckRon(player *Player, discardedTile Tile) bool {
	// Basic placeholder - actual Ron logic is complex and needs hand analysis
	return false // Placeholder: Ron not implemented yet
}
