package main

import (
	"fmt"
	"sort"
	"strings"
)

// FormatHandForDisplay formats a player's hand for terminal output (sorted).
func FormatHandForDisplay(hand []Tile) string {
	// Ensure sorted - create copy to avoid modifying original slice order if not intended
	handCopy := make([]Tile, len(hand))
	copy(handCopy, hand)
	sort.Sort(BySuitValue(handCopy))
	return strings.Join(TilesToNames(handCopy), ", ")
}

// FormatMeldsForDisplay formats melds for display, showing concealment.
func FormatMeldsForDisplay(melds []Meld) string {
	if len(melds) == 0 {
		return "None"
	}
	var displayMelds []string
	for _, meld := range melds {
		// Sort tiles within the meld for consistent display
		sort.Sort(BySuitValue(meld.Tiles))
		meldStr := fmt.Sprintf("%s: [", meld.Type)
		tileNames := []string{}

		// Show concealment for Ankan (typically ends face down)
		if meld.Type == "Ankan" {
			// Show ends face down, middle two face up
			if len(meld.Tiles) == 4 {
				tileNames = append(tileNames, meld.Tiles[0].Name+"(?)", meld.Tiles[1].Name, meld.Tiles[2].Name, meld.Tiles[3].Name+"(?)")
			} else {
				tileNames = TilesToNames(meld.Tiles)
			} // Fallback
		} else if meld.Type == "Shouminkan" || meld.Type == "Daiminkan" || meld.Type == "Pon" || meld.Type == "Chi" {
			// Indicate which tile was called and from whom
			tileNames = TilesToNames(meld.Tiles)
			calledIdx := -1
			for i, t := range meld.Tiles {
				if t.ID == meld.CalledOn.ID {
					calledIdx = i
				}
			}
			if calledIdx != -1 {
				tileNames[calledIdx] = tileNames[calledIdx] + "*" // Mark called tile
			}
		} else {
			tileNames = TilesToNames(meld.Tiles)
		}

		meldStr += strings.Join(tileNames, ", ") + "]"
		if !meld.IsConcealed && meld.FromPlayer != -1 {
			meldStr += fmt.Sprintf(" (P%d)", meld.FromPlayer+1)
		}
		displayMelds = append(displayMelds, meldStr)
	}
	return strings.Join(displayMelds, " | ")
}

// DisplayGameState outputs the current game state to the terminal.
func DisplayGameState(gs *GameState) {
	fmt.Println("\n=========================================")
	fmt.Printf("Round: %s %d | Honba: %d | Riichi Sticks: %d\n", gs.PrevalentWind, gs.RoundNumber, gs.Honba, gs.RiichiSticks)
	fmt.Printf("Wall Tiles: %d | Dead Wall Tiles: %d | Turn: %d\n", len(gs.Wall), DeadWallSize, gs.TurnNumber)
	fmt.Printf("Dora Indicators: %v\n", TilesToNames(gs.DoraIndicators))
	// Kan Dora are included in DoraIndicators now
	// if len(gs.KanDoraIndicators) > 0 { // Keep separate track if needed?
	// 	fmt.Printf("Kan Dora Indicators: %v\n", TilesToNames(gs.KanDoraIndicators))
	// }
	if len(gs.UraDoraIndicators) > 0 { // Only show if revealed
		fmt.Printf("Ura Dora Indicators: %v\n", TilesToNames(gs.UraDoraIndicators))
	}
	fmt.Println("--- Players ---")
	for i, player := range gs.Players {
		marker := " "
		if i == gs.CurrentPlayerIndex {
			marker = ">"
		} // Indicate current player
		fmt.Printf("%s P%d %s (%s Wind): Score %d %s %s\n",
			marker, i+1, player.Name, player.SeatWind, player.Score,
			If(player.IsRiichi, "[Riichi]", ""),
			If(player.IsFuriten, "[Furiten]", ""),
		)
		// Don't show hand unless it's the current player or debugging
		// if i == gs.CurrentPlayerIndex || true { // Show all hands for debugging
		//    fmt.Printf("  Hand: %s\n", FormatHandForDisplay(player.Hand))
		// }
		fmt.Printf("  Melds: %s\n", FormatMeldsForDisplay(player.Melds))
		fmt.Printf("  Discards: %v\n", TilesToNames(player.Discards))
	}
	// fmt.Printf("Full Discard Pile: %v\n", TilesToNames(gs.DiscardPile)) // Can be long
	if gs.LastDiscard != nil {
		fmt.Printf("Last Discard: %s\n", gs.LastDiscard.Name)
	}
	fmt.Println("=========================================")
}

// DisplayPlayerState shows details for a specific player (Hand, Melds, Score, Status).
func DisplayPlayerState(player *Player) {
	fmt.Printf("--- %s's State ---\n", player.Name)
	fmt.Printf("  Hand: %s\n", FormatHandForDisplay(player.Hand)) // Always show hand here
	fmt.Printf("  Melds: %s\n", FormatMeldsForDisplay(player.Melds))
	fmt.Printf("  Score: %d %s %s\n", player.Score, If(player.IsRiichi, "[Riichi]", ""), If(player.IsFuriten, "[Furiten]", ""))
}

// TilesToNames converts a slice of Tiles to a slice of their Names.
func TilesToNames(tiles []Tile) []string {
	names := make([]string, len(tiles))
	for i, t := range tiles {
		if t.Name == "" {
			names[i] = "??"
		} else {
			names[i] = t.Name
		} // Handle empty tile case
	}
	return names
}

// PlayerNames extracts names from a slice of players.
func PlayerNames(players []*Player) []string {
	names := make([]string, len(players))
	for i, p := range players {
		names[i] = p.Name
	}
	return names
}
