package main

import (
	"fmt"
	"sort"
	"strings"
)

// FormatHandForDisplay formats a player's hand for terminal output (sorted).
func FormatHandForDisplay(hand []Tile) string {
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
		sort.Sort(BySuitValue(meld.Tiles))
		meldStr := fmt.Sprintf("%s: [", meld.Type)
		tileNames := []string{}

		if meld.Type == "Ankan" {
			if len(meld.Tiles) == 4 {
				tileNames = append(tileNames, meld.Tiles[0].Name+"(?)", meld.Tiles[1].Name, meld.Tiles[2].Name, meld.Tiles[3].Name+"(?)")
			} else {
				tileNames = TilesToNames(meld.Tiles)
			}
		} else if meld.Type == "Shouminkan" || meld.Type == "Daiminkan" || meld.Type == "Pon" || meld.Type == "Chi" {
			tileNames = TilesToNames(meld.Tiles)
			calledIdx := -1
			for i, t := range meld.Tiles {
				if t.ID == meld.CalledOn.ID {
					calledIdx = i
				}
			}
			if calledIdx != -1 {
				tileNames[calledIdx] = tileNames[calledIdx] + "*"
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
	fmt.Printf("Round: %s %d (%d) | Honba: %d | Riichi Sticks: %d\n",
		gs.PrevalentWind, gs.CurrentWindRoundNumber, gs.DealerRoundCount, gs.Honba, gs.RiichiSticks)
	fmt.Printf("Wall Tiles: %d | Dead Wall Tiles: %d | Turn in Round: %d\n", len(gs.Wall), DeadWallSize, gs.TurnNumber)
	fmt.Printf("Dora Indicators: %v\n", TilesToNames(gs.DoraIndicators))
	if len(gs.UraDoraIndicators) > 0 {
		fmt.Printf("Ura Dora Indicators: %v\n", TilesToNames(gs.UraDoraIndicators))
	}
	fmt.Println("--- Players ---")
	for i, player := range gs.Players {
		marker := " "
		if i == gs.CurrentPlayerIndex {
			marker = ">"
		}
		riichiStatus := If(player.IsRiichi, "[Riichi]", "")
		if player.IsRiichi && player.DeclaredDoubleRiichi {
			riichiStatus = "[D.Riichi]"
		}
		furitenStatus := If(player.IsFuriten, "[F]", "")
		if player.IsPermanentRiichiFuriten {
			furitenStatus = "[Perm.F]"
		}

		tenpaiStatus := ""
		if gs.GamePhase == PhaseRoundEnd && gs.RoundWinner == nil { // Ryuukyoku
			tenpaiStatus = If(player.IsTenpai, "[Tenpai]", "[Noten]")
		}

		fmt.Printf("%s P%d %s (%s Wind): Score %d %s %s %s\n",
			marker, i+1, player.Name, player.SeatWind, player.Score,
			riichiStatus, furitenStatus, tenpaiStatus,
		)
		fmt.Printf("  Melds: %s\n", FormatMeldsForDisplay(player.Melds))
		if len(player.Discards) > 15 { // Truncate long discard list for display
			fmt.Printf("  Discards: %v ... (last 5: %v)\n", TilesToNames(player.Discards[:10]), TilesToNames(player.Discards[len(player.Discards)-5:]))
		} else {
			fmt.Printf("  Discards: %v\n", TilesToNames(player.Discards))
		}
		if player.PaoSourcePlayerIndex != -1 {
			fmt.Printf("  (Is Pao for P%d's Yakuman)\n", player.PaoSourcePlayerIndex+1)
		}
	}
	if gs.LastDiscard != nil {
		fmt.Printf("Last Discard: %s (by P%d)\n", gs.LastDiscard.Name, gs.CurrentPlayerIndex+1) // CurrentPlayerIndex is discarder before NextPlayer()
	}
	if len(gs.GameLog) > 0 {
		fmt.Printf("Last Log: %s\n", gs.GameLog[len(gs.GameLog)-1])
	}
	fmt.Println("=========================================")
}

// DisplayPlayerState shows details for a specific player.
func DisplayPlayerState(player *Player) {
	fmt.Printf("--- %s's State ---\n", player.Name)
	fmt.Printf("  Hand: %s (%d tiles)\n", FormatHandForDisplay(player.Hand), len(player.Hand))
	if player.JustDrawnTile != nil {
		fmt.Printf("  Just Drawn: %s\n", player.JustDrawnTile.Name)
	}
	fmt.Printf("  Melds: %s\n", FormatMeldsForDisplay(player.Melds))
	riichiStatus := If(player.IsRiichi, "[Riichi]", "")
	if player.IsRiichi && player.DeclaredDoubleRiichi {
		riichiStatus = "[D.Riichi]"
	}
	furitenStatus := If(player.IsFuriten, "[Furiten]", "")
	if player.IsPermanentRiichiFuriten {
		furitenStatus = "[Perm.Furiten]"
	}
	fmt.Printf("  Score: %d %s %s\n", player.Score, riichiStatus, furitenStatus)
	if len(player.RiichiDeclaredWaits) > 0 {
		fmt.Printf("  Riichi Waits: %v\n", TilesToNames(player.RiichiDeclaredWaits))
	}
}

// TilesToNames converts a slice of Tiles to a slice of their Names.
func TilesToNames(tiles []Tile) []string {
	names := make([]string, len(tiles))
	for i, t := range tiles {
		if t.Name == "" {
			names[i] = "??"
		} else {
			names[i] = t.Name
		}
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
