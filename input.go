package main

import (
	"bufio"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// GetPlayerDiscardChoice prompts the current player to choose a tile to discard by index.
func GetPlayerDiscardChoice(reader *bufio.Reader, player *Player) int {
	if len(player.Hand) == 0 {
		fmt.Println("Error: Player has no tiles to discard!")
		return -1 // Indicate error
	}
	fmt.Println("\nYour hand:")
	for i, tile := range player.Hand {
		fmt.Printf("[%d] %s  ", i+1, tile.Name)
	}
	fmt.Printf("\nChoose a tile to discard (1-%d): ", len(player.Hand))

	input, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println("Error reading input:", err)
		return GetPlayerDiscardChoice(reader, player) // Retry
	}
	input = strings.TrimSpace(input)

	tileIndex, err := strconv.Atoi(input)
	if err != nil || tileIndex < 1 || tileIndex > len(player.Hand) {
		fmt.Println("Invalid input. Please enter a number corresponding to a tile.")
		return GetPlayerDiscardChoice(reader, player) // Retry
	}
	return tileIndex - 1 // Return 0-based index
}

// GetPlayerChoice gets a simple y/n confirmation from the player.
func GetPlayerChoice(reader *bufio.Reader, prompt string) bool {
	fmt.Print(prompt)
	input, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println("Error reading input, assuming 'no':", err)
		return false
	}
	return strings.ToLower(strings.TrimSpace(input)) == "y"
}

// GetPlayerRiichiChoice presents the player with valid Riichi discard options
// and prompts them to choose one or cancel.
// Returns the index of the chosen option in the slice (0-based), and true if a choice was made.
// Returns -1 and false if the player cancels.
func GetPlayerRiichiChoice(reader *bufio.Reader, options []RiichiOption) (int, bool) {
	if len(options) == 0 {
		fmt.Println("Error: No Riichi options available.") // Should not happen if called correctly
		return -1, false
	}

	fmt.Println("\n--- Declare Riichi ---")
	fmt.Println("Choose discard to declare Riichi:")
	for i, opt := range options {
		// Sort waits for display consistency
		sort.Sort(BySuitValue(opt.Waits))
		fmt.Printf("[%d] Discard %s -> Waits: %v\n",
			i+1, // 1-based index for user
			opt.DiscardTile.Name,
			TilesToNames(opt.Waits), // Use helper for clean names
		)
	}
	fmt.Printf("[0] Cancel Riichi\n")
	fmt.Print("Enter choice: ")

	input, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println("Error reading input, canceling Riichi.", err)
		return -1, false
	}
	choice, err := strconv.Atoi(strings.TrimSpace(input))
	if err != nil || choice < 0 || choice > len(options) {
		fmt.Println("Invalid choice, canceling Riichi.")
		return -1, false
	}

	if choice == 0 {
		fmt.Println("Riichi cancelled.")
		return -1, false // User cancelled
	}

	// Return 0-based index of the chosen option
	return choice - 1, true
}

// GetChiChoice prompts player to choose which Chi sequence (if multiple options).
// Returns choice number (1-based) and the 3 tiles for the chosen sequence, or 0, nil if cancelled/invalid.
func GetChiChoice(gs *GameState, player *Player, discardedTile Tile) (int, []Tile) {
	// Find the pairs of hand tiles that enable Chi
	possibleHandTilePairs := FindPossibleChiSequences(player, discardedTile)

	if len(possibleHandTilePairs) == 0 {
		fmt.Println("Error: GetChiChoice called but no Chi sequences found.") // Should not happen
		return 0, nil                                                         // No valid Chi
	}

	fmt.Printf("\n%s, choose Chi sequence for %s:\n", player.Name, discardedTile.Name)
	fullSequences := [][]Tile{} // Store the complete 3-tile sequences

	for i, handTiles := range possibleHandTilePairs {
		sequence := append([]Tile{}, handTiles...)
		sequence = append(sequence, discardedTile)
		sort.Sort(BySuitValue(sequence)) // Sort the full sequence for display
		fullSequences = append(fullSequences, sequence)
		// Display the two tiles from hand
		fmt.Printf("[%d] %s + %s (using %s)\n", i+1, handTiles[0].Name, handTiles[1].Name, discardedTile.Name)
	}
	fmt.Printf("[0] Cancel\n")
	fmt.Print("Enter choice: ")

	input, err := gs.InputReader.ReadString('\n')
	if err != nil {
		fmt.Println("Error reading input, canceling Chi.")
		return 0, nil
	}
	choice, err := strconv.Atoi(strings.TrimSpace(input))
	if err != nil || choice < 0 || choice > len(possibleHandTilePairs) {
		fmt.Println("Invalid choice, canceling Chi.")
		return 0, nil
	}

	if choice == 0 {
		return 0, nil // User cancelled
	}

	// Return the chosen sequence (already includes discardedTile and is sorted)
	chosenSequence := fullSequences[choice-1]
	return choice, chosenSequence
}
