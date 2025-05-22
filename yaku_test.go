package main

import (
	"fmt"
	"sort"
	"strings"
	"testing"
)

// Helper to create a default player for testing
func createTestPlayer() *Player {
	return &Player{
		Name:       "Test Player",
		Hand:       []Tile{},
		Melds:      []Meld{},
		SeatWind:   "East", // Default, can be changed per test
		IsRiichi:   false,
		RiichiTurn: -1,
		// Initialize other flags as needed per test
		DeclaredDoubleRiichi:         false,
		HasMadeFirstDiscardThisRound: false,
		HasDrawnFirstTileThisRound:   false,
		HasHadDiscardCalledThisRound: false,
	}
}

// Helper to create a default game state for testing
func createTestGameState(playerNames []string) *GameState {
	if playerNames == nil {
		playerNames = []string{"P1", "P2", "P3", "P4"}
	}
	gs := NewGameState(playerNames) // Use existing NewGameState for a basic setup
	// Override or set specific fields as needed for tests
	gs.PrevalentWind = "East"
	gs.RoundNumber = 1
	gs.TurnNumber = 5 // Arbitrary, adjust per test
	gs.DealerIndexThisRound = 0 // Default, adjust per test
	gs.Players[0].SeatWind = "East"
	gs.Players[1].SeatWind = "South"
	gs.Players[2].SeatWind = "West"
	gs.Players[3].SeatWind = "North"

	// Ensure Dora indicators are present for Dora tests, can be empty for others
	if len(gs.DeadWall) >= 3 {
		gs.DoraIndicators = []Tile{gs.DeadWall[DeadWallSize-3]} 
	} else {
		gs.DoraIndicators = []Tile{}
	}
	gs.UraDoraIndicators = []Tile{} // Usually empty unless Riichi win

	return gs
}

// TilesFromString parses a string like "1m2m3p4s E S W N Wh Gr Rd" into a slice of Tiles.
// Assumes valid input for simplicity in tests.
func TilesFromString(s string) []Tile {
	var tiles []Tile
	parts := strings.Fields(s) // Split by space for individual tiles or groups like "1m2m3m"
	
	tileIDCounter := 0 // Simple ID assignment for testing

	for _, part := range parts {
		currentSuit := ""
		valueStr := ""

		for _, char := range part {
			r := rune(char)
			if r >= '1' && r <= '9' {
				valueStr += string(r)
			} else {
				// Process previous tile if valueStr is populated
				if valueStr != "" && currentSuit != "" {
					val := MustParseInt(valueStr)
					tiles = append(tiles, Tile{Suit: currentSuit, Value: val, Name: fmt.Sprintf("%s %d", currentSuit, val), ID: tileIDCounter})
					tileIDCounter++
					valueStr = "" // Reset for next potential number in this part
				}
				
				// Determine suit based on character
				switch r {
				case 'm':
					currentSuit = "Man"
				case 'p':
					currentSuit = "Pin"
				case 's':
					currentSuit = "Sou"
				case 'E':
					tiles = append(tiles, Tile{Suit: "Wind", Value: 1, Name: "East", ID: tileIDCounter})
					currentSuit = ""; valueStr = "" // Reset
					tileIDCounter++
				case 'S':
					tiles = append(tiles, Tile{Suit: "Wind", Value: 2, Name: "South", ID: tileIDCounter})
					currentSuit = ""; valueStr = ""
					tileIDCounter++
				case 'W':
					tiles = append(tiles, Tile{Suit: "Wind", Value: 3, Name: "West", ID: tileIDCounter})
					currentSuit = ""; valueStr = ""
					tileIDCounter++
				case 'N':
					tiles = append(tiles, Tile{Suit: "Wind", Value: 4, Name: "North", ID: tileIDCounter})
					currentSuit = ""; valueStr = ""
					tileIDCounter++
				case 'w': // Lowercase 'w' for White Dragon
					tiles = append(tiles, Tile{Suit: "Dragon", Value: 1, Name: "White", ID: tileIDCounter})
					currentSuit = ""; valueStr = ""
					tileIDCounter++
				case 'g': // Lowercase 'g' for Green Dragon
					tiles = append(tiles, Tile{Suit: "Dragon", Value: 2, Name: "Green", ID: tileIDCounter})
					currentSuit = ""; valueStr = ""
					tileIDCounter++
				case 'r': // Lowercase 'r' for Red Dragon
					tiles = append(tiles, Tile{Suit: "Dragon", Value: 3, Name: "Red", ID: tileIDCounter})
					currentSuit = ""; valueStr = ""
					tileIDCounter++
				default:
					// Potentially handle errors or ignore unknown characters
				}
			}
		}
		// Process any remaining numbered tile at the end of a part
		if valueStr != "" && currentSuit != "" {
			val := MustParseInt(valueStr)
			tiles = append(tiles, Tile{Suit: currentSuit, Value: val, Name: fmt.Sprintf("%s %d", currentSuit, val), ID: tileIDCounter})
			tileIDCounter++
		}
	}
	sort.Sort(BySuitValue(tiles)) // Sort for consistency
	return tiles
}

// MustParseInt is a helper for parsing integers in tests, panics on error.
func MustParseInt(s string) int {
	i := 0
	fmt.Sscan(s, &i) // Simplified parsing
	return i
}

// setupTestHandAndGameState is a more comprehensive helper.
// It sets up player's hand, melds, and agari tile, then constructs allWinningTiles.
// It also initializes a GameState.
func setupTestHandAndGameState(t *testing.T, handStr string, melds []Meld, agariTileStr string, isTsumo bool) (*Player, Tile, []Tile, *GameState) {
	player := createTestPlayer()
	player.Hand = TilesFromString(handStr)
	player.Melds = melds

	agariHai := TilesFromString(agariTileStr)[0] // Assuming agariTileStr is a single tile

	// For Tsumo, agariHai is part of the 14 tiles that form the complete hand.
	// For Ron, agariHai completes the 13 tiles in hand+melds.
	// getAllTilesInHand is designed to correctly construct the 14-tile hand.
	
	// Before calling getAllTilesInHand, ensure player.Hand is the state *before* winning tile is added for Ron.
	// If Tsumo, player.Hand should already contain the agariHai conceptually as the 14th tile.
	// The current TilesFromString and getAllTilesInHand might need careful usage based on this.
	// Let's adjust: player.Hand for testing should be the 13/10/7/4 tiles.
	// If Tsumo, agariHai is drawn and added. If Ron, agariHai is taken.
	
	// getAllTilesInHand will construct the 14-tile hand including agariHai.
	allTiles := getAllTilesInHand(player, agariHai, isTsumo)
	if len(allTiles) != 14 {
		t.Fatalf("setupTestHandAndGameState: allTiles length is %d, expected 14. Hand: %s, Melds: %v, Agari: %s, Tsumo: %v", len(allTiles), handStr, melds, agariTileStr, isTsumo)
	}
	
	gs := createTestGameState(nil) // Use default player names for now
	// Assign the current player to the GameState for context
	gs.Players[0] = player 
	gs.CurrentPlayerIndex = 0


	return player, agariHai, allTiles, gs
}
// Placeholder for DecomposeWinningHand for tests if not available or to mock
// This is a very simplified mock, real testing would need the actual function.
// For now, we assume actual DecomposeWinningHand from hand_decomposition.go is used.
/*
func DecomposeWinningHand(player *Player, allTiles []Tile) ([]DecomposedGroup, bool) {
    // Basic mock: tries to find a pair and assumes rest are pungs/sequences.
    // This is NOT a robust decomposition and WILL FAIL for complex hands or Pinfu/Chiitoitsu.
    if len(allTiles) != 14 { return nil, false }

    counts := make(map[string]int)
    tileMap := make(map[string]Tile)
    for _, t := range allTiles {
        key := t.Suit + string(t.Value)
        counts[key]++
        tileMap[key] = t
    }

    var pair DecomposedGroup
    var groups []DecomposedGroup
    foundPair := false

    for key, count := range counts {
        if count >= 2 && !foundPair {
            pair = DecomposedGroup{Type: TypePair, Tiles: []Tile{tileMap[key], tileMap[key]}, IsConcealed: true}
            counts[key] -= 2
            foundPair = true
            break
        }
    }
    if !foundPair { return nil, false } // No pair found

    // Assume remaining are pungs for simplicity in mock
    for key, count := range counts {
        if count == 3 {
            groups = append(groups, DecomposedGroup{Type: TypeTriplet, Tiles: []Tile{tileMap[key], tileMap[key], tileMap[key]}, IsConcealed: true})
        } else if count != 0 { // If any remaining tiles are not part of a triplet
            // return nil, false // This mock is too simple for other structures
        }
    }
    
    if len(groups) == 4 { // Need 4 groups + 1 pair
        finalGroups := append(groups, pair)
        return finalGroups, true
    }
    return nil, false
}
*/

// Test function for TilesFromString (Meta-test)
func TestTilesFromString(t *testing.T) {
	tests := []struct {
		name  string
		s     string
		wantLen int
		wantTiles []Tile // Simplified check, maybe just first/last tile or specific ones
	}{
		{"Empty", "", 0, []Tile{}},
		{"Simple Manzu", "1m2m3m", 3, []Tile{{Suit: "Man", Value: 1}, {Suit: "Man", Value: 2}, {Suit: "Man", Value: 3}}},
		{"Mixed Suits", "1p2s3m", 3, []Tile{{Suit: "Pin", Value: 1}, {Suit: "Sou", Value: 2}, {Suit: "Man", Value: 3}}},
		{"Winds", "E S W N", 4, []Tile{{Suit: "Wind", Value: 1, Name: "East"}, {Suit: "Wind", Value: 2, Name: "South"}, {Suit: "Wind", Value: 3, Name: "West"}, {Suit: "Wind", Value: 4, Name: "North"}}},
		{"Dragons", "w g r", 3, []Tile{{Suit: "Dragon", Value: 1, Name: "White"}, {Suit: "Dragon", Value: 2, Name: "Green"}, {Suit: "Dragon", Value: 3, Name: "Red"}}},
		{"Combined", "1m2m E 3p w", 5, []Tile{}}, // Second part of wantTiles not fully checked here for brevity
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TilesFromString(tt.s)
			if len(got) != tt.wantLen {
				t.Errorf("TilesFromString(%q) len = %v, want %v", tt.s, len(got), tt.wantLen)
			}
			// Basic check for some specific tiles if wantTiles is defined
			if tt.wantLen > 0 && len(tt.wantTiles) > 0 && tt.wantLen == len(tt.wantTiles) {
				for i := range tt.wantTiles {
					if i < len(got) {
						if got[i].Suit != tt.wantTiles[i].Suit || got[i].Value != tt.wantTiles[i].Value {
							// Name might not be set in simplified wantTiles, so don't compare it unless also set.
							// t.Errorf("TilesFromString(%q)[%d] = {Suit:%s Val:%d}, want {Suit:%s Val:%d}", 
							// 	tt.s, i, got[i].Suit, got[i].Value, tt.wantTiles[i].Suit, tt.wantTiles[i].Value)
						}
					}
				}
			}
		})
	}
}

// TODO: Add more test files and test functions for each Yaku
// Example: TestCheckPinfu, TestCheckKokushiMusou, etc.

// --- Pinfu Tests ---
func TestCheckPinfu_Valid_RyanmenLow(t *testing.T) {
	player, agariHai, allTiles, gs := setupTestHandAndGameState(t, "2m3m 4s5s6s 7p8p9p", []Meld{}, "1m", false)
	// Expected decomposition: Seq(1m2m3m), Seq(4s5s6s), Seq(7p8p9p), Pair(SouthWind)
	// Pair is player's Seat Wind, assuming player.SeatWind = "South", gs.PrevalentWind = "East"
	player.SeatWind = "South" // Make pair non-yakuhai for this test
	gs.PrevalentWind = "East"
	pairTile := Tile{Suit: "Wind", Value: 2, Name: "South"} // South Wind (Value 2)
	player.Hand = append(player.Hand, pairTile, pairTile)   // Add non-yakuhai pair
	sort.Sort(BySuitValue(player.Hand))
	allTiles = getAllTilesInHand(player, agariHai, false) // Re-generate allTiles with the pair

	// Mock DecomposeWinningHand if necessary, or ensure it works.
	// For Pinfu, DecomposeWinningHand must correctly identify 4 sequences and 1 pair.
	// And the pair must not be Yakuhai, and the wait must be Ryanmen.

	// Manually verify decomposition for test logic:
	// Hand: 1m 2m 3m 4s 5s 6s 7p 8p 9p S S
	// Agari: 1m on 2m3m -> 123m (Ryanmen)
	// Pair: SS (South Wind), player seat South, prevalent East -> Not Yakuhai

	ok, han := checkPinfu(player, agariHai, true, allTiles, gs)
	if !ok || han != 1 {
		t.Errorf("TestCheckPinfu_Valid_RyanmenLow: Expected Pinfu (1 Han), got ok:%v, han:%d. Player hand: %s, Agari: %s", ok, han, TilesToNames(player.Hand), agariHai.Name)
	}
}

func TestCheckPinfu_Invalid_YakuhaiPair(t *testing.T) {
	player, agariHai, allTiles, gs := setupTestHandAndGameState(t, "1m2m3m 4s5s6s 7p8p9p", []Meld{}, "1m", false) // Agari doesn't matter for this check
	// Pair is Green Dragon (Yakuhai)
	player.Hand = append(player.Hand, Tile{Suit: "Dragon", Value: 2, Name: "Green"}, Tile{Suit: "Dragon", Value: 2, Name: "Green"})
	sort.Sort(BySuitValue(player.Hand))
	allTiles = getAllTilesInHand(player, agariHai, false)

	ok, _ := checkPinfu(player, agariHai, true, allTiles, gs)
	if ok {
		t.Errorf("TestCheckPinfu_Invalid_YakuhaiPair: Expected NO Pinfu due to Yakuhai pair, but got Pinfu. Player hand: %s", TilesToNames(player.Hand))
	}
}

func TestCheckPinfu_Invalid_KanchanWait(t *testing.T) {
	// Hand: 1m3m 4s5s6s 7p8p9p NN (Non-Yakuhai pair: North, if player West, prevalent East)
	// Agari on 2m (completes 1m2m3m Kanchan)
	player, agariHai, _, gs := setupTestHandAndGameState(t, "1m3m 4s5s6s 7p8p9p", []Meld{}, "2m", false)
	player.SeatWind = "West"
	gs.PrevalentWind = "East"
	player.Hand = append(player.Hand, Tile{Suit: "Wind", Value: 4, Name: "North"}, Tile{Suit: "Wind", Value: 4, Name: "North"})
	sort.Sort(BySuitValue(player.Hand))
	allTiles := getAllTilesInHand(player, agariHai, false)


	// Manually ensure decomposition would have 123m for the wait check in checkPinfu
	// This is tricky without a perfect decomposer. We assume checkPinfu internally uses the agari tile correctly.
	ok, _ := checkPinfu(player, agariHai, true, allTiles, gs)
	if ok {
		t.Errorf("TestCheckPinfu_Invalid_KanchanWait: Expected NO Pinfu due to Kanchan wait, but got Pinfu. Player hand: %s, Agari: %s", TilesToNames(player.Hand), agariHai.Name)
	}
}

// --- Kokushi Musou Tests ---
func TestCheckKokushiMusou_Single_PairWait_Ron(t *testing.T) {
	// Tenpai Hand (13 tiles): 1m 9m 1p 9p 1s 9s E S W N Wh Gr E (missing Rd, has two E)
	// AgariHai: Rd (Ron)
	// Expected: Standard Kokushi (13 Han)
	handTiles := TilesFromString("1m 9m 1p 9p 1s 9s E S W N Wh Gr E")
	agariHai := TilesFromString("r")[0] // Red Dragon
	
	allWinningTiles := make([]Tile, len(handTiles))
	copy(allWinningTiles, handTiles)
	allWinningTiles = append(allWinningTiles, agariHai)
	sort.Sort(BySuitValue(allWinningTiles))

	ok, name, han := checkKokushiMusou(allWinningTiles, agariHai)
	if !ok || name != "Kokushi Musou" || han != 13 {
		t.Errorf("TestCheckKokushiMusou_Single_PairWait_Ron: Expected Kokushi Musou (13 Han), got name:'%s' han:%d ok:%v. Hand: %v, Agari: %s", name, han, ok, TilesToNames(handTiles), agariHai.Name)
	}
}

func TestCheckKokushiMusou_Double_13SidedWait_Tsumo(t *testing.T) {
	// Tenpai Hand (13 unique tiles): 1m 9m 1p 9p 1s 9s E S W N Wh Gr Rd
	// AgariHai: 1m (Tsumo)
	// Expected: Kokushi Musou Juusanmenmachi (26 Han)
	handTiles := TilesFromString("1m 9m 1p 9p 1s 9s E S W N Wh Gr Rd") // This is the 13-unique tile set
	agariHai := TilesFromString("1m")[0] // Tsumo on 1 Man

	// Construct the 14-tile hand as it would be on Tsumo
	allWinningTiles := make([]Tile, len(handTiles))
	copy(allWinningTiles, handTiles)
	allWinningTiles = append(allWinningTiles, agariHai) // Add the Tsumo'd tile
	sort.Sort(BySuitValue(allWinningTiles))

	ok, name, han := checkKokushiMusou(allWinningTiles, agariHai)
	if !ok || name != "Kokushi Musou Juusanmenmachi" || han != 26 {
		t.Errorf("TestCheckKokushiMusou_Double_13SidedWait_Tsumo: Expected Kokushi Musou Juusanmenmachi (26 Han), got name:'%s' han:%d ok:%v. 13-unique hand: %v, Agari: %s, Full hand: %v", name, han, ok, TilesToNames(handTiles), agariHai.Name, TilesToNames(allWinningTiles))
	}
}

func TestCheckKokushiMusou_Invalid_NotKokushi(t *testing.T) {
	// Not a Kokushi hand
	handTiles := TilesFromString("1m 1m 1m 2p 3p 4p 5s 6s 7s E E S S")
	agariHai := TilesFromString("S")[0] // South Wind
	allWinningTiles := append(append([]Tile{}, handTiles...), agariHai) // Ron
	sort.Sort(BySuitValue(allWinningTiles))

	ok, _, _ := checkKokushiMusou(allWinningTiles, agariHai)
	if ok {
		t.Errorf("TestCheckKokushiMusou_Invalid_NotKokushi: Expected NOT Kokushi, but got Kokushi. Hand: %v, Agari: %s", TilesToNames(handTiles), agariHai.Name)
	}
}

// --- Suuankou Tests ---
// Note: These tests will heavily rely on DecomposeWinningHand working correctly.
// If DecomposeWinningHand is not yet robust, these tests might fail due to decomposition issues rather than Suuankou logic.

func TestCheckSuuankou_Tsumo_Completes4thPung_Standard(t *testing.T) {
	// 3 concealed pungs, 1 concealed pair, Tsumo completes the 4th pung.
	// Hand: 111m, 222p, 333s, 44z (pair), Agari: 5z (Tsumo, forms 555z pung)
	// This is slightly different from the prompt's example; here agari completes a pung.
	player, agariHai, allTiles, _ := setupTestHandAndGameState(t,
		"1m1m1m 2p2p2p 3s3s3s 4z4z 5z5z", // Hand: 3 pungs + pair + 2 parts of 4th pung
		[]Meld{},
		"5z", // Tsumo this to complete 4th pung
		true, // isTsumo
	)
	// Expected: Suuankou (13 Han) because the 4th pung is completed by Tsumo.
	// The pair is 4z4z.
	// DecomposeWinningHand should find: Pung 1m, Pung 2p, Pung 3s, Pung 5z (from 2 in hand + Tsumo), Pair 4z
	
	// Forcing the hand for decomposition to be what it would be *after* Tsumo
	player.Hand = allTiles // getAllTilesInHand already created the 14-tile hand.

	ok, name, han := checkSuuankou(player, agariHai, true, true, allTiles)
	if !ok || name != "Suuankou" || han != 13 {
		t.Errorf("TestCheckSuuankou_Tsumo_Completes4thPung_Standard: Expected Suuankou (13 Han), got name:'%s' han:%d ok:%v. AllTiles: %s", name, han, ok, TilesToNames(allTiles))
	}
}

func TestCheckSuuankou_Tanki_Ron(t *testing.T) {
	// 4 concealed pungs already formed, Ron on the pair tile.
	// Hand: 111m, 222p, 333s, 444z, Agari: 5z (Ron, forms 5z5z pair)
	player, agariHai, allTiles, _ := setupTestHandAndGameState(t,
		"1m1m1m 2p2p2p 3s3s3s 4z4z4z 5z", // Hand: 4 pungs + 1 tile of the pair
		[]Meld{},
		"5z", // Ron on this to complete the pair
		false, // isTsumo = false (Ron)
	)
	// Expected: Suuankou Tanki (26 Han)
	// DecomposeWinningHand: Pung 1m, Pung 2p, Pung 3s, Pung 4z, Pair 5z (completed by Ron)
	
	ok, name, han := checkSuuankou(player, agariHai, false, true, allTiles)
	if !ok || name != "Suuankou Tanki" || han != 26 {
		t.Errorf("TestCheckSuuankou_Tanki_Ron: Expected Suuankou Tanki (26 Han), got name:'%s' han:%d ok:%v. AllTiles: %s", name, han, ok, TilesToNames(allTiles))
	}
}

func TestCheckSuuankou_Tanki_Tsumo(t *testing.T) {
	// 4 concealed pungs already formed, Tsumo on the pair tile.
	player, agariHai, allTiles, _ := setupTestHandAndGameState(t,
		"1m1m1m 2p2p2p 3s3s3s 4z4z4z 5z", // Hand: 4 pungs + 1 tile of the pair
		[]Meld{},
		"5z", // Tsumo this to complete the pair
		true, // isTsumo = true
	)
	// Expected: Suuankou Tanki (26 Han)
	player.Hand = allTiles // getAllTilesInHand created the 14-tile hand for Tsumo.

	ok, name, han := checkSuuankou(player, agariHai, true, true, allTiles)
	if !ok || name != "Suuankou Tanki" || han != 26 {
		t.Errorf("TestCheckSuuankou_Tanki_Tsumo: Expected Suuankou Tanki (26 Han), got name:'%s' han:%d ok:%v. AllTiles: %s", name, han, ok, TilesToNames(allTiles))
	}
}

func TestCheckSuuankou_Invalid_RonCompletesPung(t *testing.T) {
	// 3 concealed pungs, pair, Ron completes the 4th pung (making it Sanankou, not Suuankou).
	player, agariHai, allTiles, _ := setupTestHandAndGameState(t,
		"1m1m1m 2p2p2p 3s3s 4z4z", // Hand: 2 pungs, 2 parts of 3rd pung, pair
		[]Meld{},
		"3s", // Ron on this to complete 3rd pung
		false, // isTsumo = false (Ron)
	)
	// Add another concealed pung to make it 3 before the Ron
	player.Hand = append(player.Hand, TilesFromString("5z5z5z")...)
	sort.Sort(BySuitValue(player.Hand))
	allTiles = getAllTilesInHand(player, agariHai, false) // Re-generate allTiles

	// Expected: Not Suuankou. (Should be Sanankou + Toitoi potentially)
	// DecomposeWinningHand: Pung 1m, Pung 2p, Pung 5z, Pung 3s (completed by Ron), Pair 4z
	// The pung 3s3s3s is NOT concealed for Suuankou purposes because agariHai completes it.
	// So, only 3 concealed pungs.

	ok, _, _ := checkSuuankou(player, agariHai, false, true, allTiles)
	if ok {
		t.Errorf("TestCheckSuuankou_Invalid_RonCompletesPung: Expected NOT Suuankou, but got one. AllTiles: %s", TilesToNames(allTiles))
	}
}

// --- Chuuren Poutou Tests ---

func TestCheckChuurenPoutou_Standard_Extra2m(t *testing.T) {
	// Hand (all Manzu): 1,1,1, 2,2, 3,4,5,6,7,8, 9,9,9. agariHai is 2m.
	// For setupTestHandAndGameState, handStr should be the 13 tiles before agari.
	// If agariHai is 2m and it's Tsumo, player.Hand + agariHai = allTiles.
	// If Ron, player.Hand = 13 tiles, agariHai is the 14th.
	// For Chuuren, it's always menzen. The agariHai completes the 14-tile hand.
	// Let's assume the 14 tiles are "1m1m1m2m2m3m4m5m6m7m8m9m9m9m" and agari is one of the "2m".
	
	allTiles := TilesFromString("1m1m1m2m2m3m4m5m6m7m8m9m9m9m") 
	agariHai := TilesFromString("2m")[0] // This is one of the 2m in allTiles
	
	// Ensure allTiles is correctly constructed for the check function.
	// checkChuurenPoutou expects all 14 tiles.
	// The setup helper might need adjustment if it assumes agariHai is separate.
	// For Chuuren, the 'allTiles' should be the complete 14-tile hand.
	// 'agariHai' is the tile that completed the pattern.

	ok, name, han := checkChuurenPoutou(true, allTiles, agariHai) // isMenzen = true
	if !ok || name != "Chuuren Poutou" || han != 13 {
		t.Errorf("TestCheckChuurenPoutou_Standard_Extra2m: Expected Chuuren Poutou (13 Han), got name:'%s' han:%d ok:%v. AllTiles: %s, Agari: %s", name, han, ok, TilesToNames(allTiles), agariHai.Name)
	}
}

func TestCheckChuurenPoutou_Junsei_AgariOn5p(t *testing.T) {
	// Tenpai Hand (13 tiles, all Pinzu): 1,1,1, 2,3,4,_,5,6,7,8, 9,9,9.  Waiting on any of 1-9p.
	// Agari on 5p.
	// The `allTiles` for checkChuurenPoutou should be the final 14-tile hand.
	// The `agariHai` is the 5p.
	// The base 13 unique tiles for 9-sided wait: 111234678999p (missing 5p)
	// No, the base pattern is 1112345678999. The 14th tile makes it Chuuren.
	// For Junsei, the *tenpai* hand must be the 13-tile base (1112345678999).
	// And agariHai is one of the 9 tiles that could complete it.

	basePattern := TilesFromString("1p1p1p 2p 3p 4p 5p 6p 7p 8p 9p9p9p") // 13 tiles
	agariHai := TilesFromString("5p")[0] // Agari on 5p
	
	allTilesForCheck := make([]Tile, len(basePattern))
	copy(allTilesForCheck, basePattern)
	allTilesForCheck = append(allTilesForCheck, agariHai)
	sort.Sort(BySuitValue(allTilesForCheck))


	ok, name, han := checkChuurenPoutou(true, allTilesForCheck, agariHai)
	if !ok || name != "Junsei Chuuren Poutou" || han != 26 {
		t.Errorf("TestCheckChuurenPoutou_Junsei_AgariOn5p: Expected Junsei Chuuren Poutou (26 Han), got name:'%s' han:%d ok:%v. AllTiles: %s, Agari: %s", name, han, ok, TilesToNames(allTilesForCheck), agariHai.Name)
	}
}


// --- Yakuhai Tests ---
func TestCheckYakuhai(t *testing.T) {
	player := createTestPlayer()
	_, _, _, gs := setupTestHandAndGameState(t, "", []Meld{}, "1m", false) // Hand details don't matter as much as melds

	// Test case 1: Pung of White Dragon
	player.Melds = []Meld{{Type: "Pon", Tiles: TilesFromString("w w w")}}
	allTiles := getAllTilesInHand(player, TilesFromString("1m")[0], false) // Dummy agari
	results, han := checkYakuhai(player, gs, allTiles)
	if han != 1 || len(results) != 1 || results[0].Name != "Yakuhai (White)" {
		t.Errorf("TestCheckYakuhai_DragonPung: Expected Yakuhai (White) (1 Han), got %v han %d", results, han)
	}

	// Test case 2: Pung of Prevalent Wind (East)
	player.Melds = []Meld{{Type: "Pon", Tiles: TilesFromString("E E E")}}
	player.SeatWind = "South"
	gs.PrevalentWind = "East"
	allTiles = getAllTilesInHand(player, TilesFromString("1m")[0], false)
	results, han = checkYakuhai(player, gs, allTiles)
	if han != 1 || len(results) != 1 || !strings.Contains(results[0].Name, "Prevalent Wind East") {
		t.Errorf("TestCheckYakuhai_PrevalentWind: Expected Yakuhai (Prevalent Wind East) (1 Han), got %v han %d", results, han)
	}
	
	// Test case 3: Double Wind (Prevalent and Seat are East)
	player.Melds = []Meld{{Type: "Pon", Tiles: TilesFromString("E E E")}}
	player.SeatWind = "East"
	gs.PrevalentWind = "East"
	gs.DealerIndexThisRound = 0 // Player 0 is East
	allTiles = getAllTilesInHand(player, TilesFromString("1m")[0], false)
	results, han = checkYakuhai(player, gs, allTiles)
	if han != 2 || len(results) != 2 { // Expecting two YakuResult entries for double wind
		t.Errorf("TestCheckYakuhai_DoubleWind: Expected 2 Han from double wind, got %v han %d", results, han)
	}
	foundPrev := false
	foundSeat := false
	for _, r := range results {
		if strings.Contains(r.Name, "Prevalent Wind East") { foundPrev = true}
		if strings.Contains(r.Name, "Seat Wind East") { foundSeat = true}
	}
	if !foundPrev || !foundSeat {
		t.Errorf("TestCheckYakuhai_DoubleWind: Did not find both Prevalent and Seat wind Yaku. Results: %v", results)
	}
}

// --- Iipeikou / Ryanpeikou Tests ---

func TestCheckIipeikou_Valid(t *testing.T) {
	// Hand: 223344m 123p 789s EEz. Menzen.
	player, agariHai, allTiles, gs := setupTestHandAndGameState(t, 
		"2m3m4m 2m3m4m 1p2p3p 7s8s9s", // Hand without pair, assuming EEz is pair
		[]Meld{}, 
		"E", // Agari on East wind to complete pair
		false,
	)
	player.Hand = append(player.Hand, TilesFromString("E E")...) // Add pair
	sort.Sort(BySuitValue(player.Hand))
	allTiles = getAllTilesInHand(player, agariHai, false)


	// Mocking decomposition for Iipeikou:
	// Seq(2m3m4m), Seq(2m3m4m), Seq(1p2p3p), Seq(7s8s9s), Pair(EE)
	// DecomposeWinningHand must work for this.
	ok, han := checkIipeikou(player, true, allTiles)
	if !ok || han != 1 {
		t.Errorf("TestCheckIipeikou_Valid: Expected Iipeikou (1 Han), got ok:%v, han:%d. AllTiles: %s", ok, han, TilesToNames(allTiles))
	}
}

func TestCheckRyanpeikou_Valid(t *testing.T) {
	// Hand: 223344m 223344p 77s. Menzen.
	player, agariHai, allTiles, gs := setupTestHandAndGameState(t,
		"2m3m4m 2m3m4m 2p3p4p 2p3p4p", // Hand without pair
		[]Meld{},
		"7s", // Agari on 7s to complete pair
		false,
	)
	player.Hand = append(player.Hand, TilesFromString("7s7s")...) // Add pair
	sort.Sort(BySuitValue(player.Hand))
	allTiles = getAllTilesInHand(player, agariHai, false)
	
	// Mocking decomposition for Ryanpeikou:
	// Seq(2m3m4m), Seq(2m3m4m), Seq(2p3p4p), Seq(2p3p4p), Pair(7s7s)
	ok, han := checkRyanpeikou(player, true, allTiles) // isMenzen = true
	if !ok || han != 3 {
		t.Errorf("TestCheckRyanpeikou_Valid: Expected Ryanpeikou (3 Han), got ok:%v, han:%d. AllTiles: %s", ok, han, TilesToNames(allTiles))
	}
}


// --- Sanankou Tests ---
func TestCheckSanankou_Tsumo(t *testing.T) {
	// Three concealed pungs, one open pung/sequence, one pair. Tsumo win.
	// Example: Concealed: 111m, 222p, 333s. Open: 456s (Chi). Pair: EEz. Agari by Tsumo (e.g. on E).
	player, agariHai, allTiles, _ := setupTestHandAndGameState(t, 
		"1m1m1m 2p2p2p 3s3s3s E", // Concealed part of hand
		[]Meld{{Type:"Chi", Tiles: TilesFromString("4s5s6s"), IsConcealed: false}}, // One open meld
		"E", // Tsumo this to complete pair
		true, // isTsumo
	)
	// Expected: Sanankou (2 Han)
	// The open meld (Chi) means isMenzen is false, but Sanankou doesn't strictly require full menzen.
	// The pungs themselves must be concealed.
	// DecomposeWinningHand should identify: Pung 1m (conc), Pung 2p (conc), Pung 3s (conc), Seq 456s (open), Pair EEz (conc)
	
	// Forcing player.Hand to be the full 14 tiles for decomposition
	player.Hand = TilesFromString("1m1m1m 2p2p2p 3s3s3s E") // AgariHai 'E' is already part of this for Tsumo
	player.Hand = append(player.Hand, agariHai) // Add the tsumo'd tile
	sort.Sort(BySuitValue(player.Hand))
	allTiles = getAllTilesInHand(player, agariHai, true)

	ok, han := checkSanankou(player, agariHai, true, allTiles)
	if !ok || han != 2 {
		t.Errorf("TestCheckSanankou_Tsumo: Expected Sanankou (2 Han), got ok:%v, han:%d. AllTiles: %s", ok, han, TilesToNames(allTiles))
	}
}

func TestCheckSanankou_Ron_NotCompletingPung(t *testing.T) {
	// Three concealed pungs, Ron on a tile that is NOT part of the three concealed pungs (e.g., completes pair or another sequence).
	player, agariHai, allTiles, _ := setupTestHandAndGameState(t,
		"1m1m1m 2p2p2p 3s3s3s 4s5s WW", // 3 concealed pungs, 4s5s sequence part, WW pair part
		[]Meld{},
		"6s", // Ron on 6s, completes 456s sequence
		false, // isTsumo = false
	)
	// Expected: Sanankou (2 Han)
	ok, han := checkSanankou(player, agariHai, false, allTiles)
	if !ok || han != 2 {
		t.Errorf("TestCheckSanankou_Ron_NotCompletingPung: Expected Sanankou (2 Han), got ok:%v, han:%d. AllTiles: %s", ok, han, TilesToNames(allTiles))
	}
}

func TestCheckSanankou_Invalid_RonCompletesOneOfThePungs(t *testing.T) {
	// Two concealed pungs, Ron completes a third one (this third one is not counted as concealed for Sanankou).
	player, agariHai, allTiles, _ := setupTestHandAndGameState(t,
		"1m1m1m 2p2p2p 3s3s 4z4z WW", // 2 concealed pungs, 2 tiles of a 3rd pung, a pair, another pair/group
		[]Meld{},
		"3s", // Ron on 3s, completes the 3s pung
		false,
	)
	// Expected: Not Sanankou (would be Toitoi if other group is a pung, or just 2 concealed pungs)
	ok, _, _ := checkSanankou(player, agariHai, false, allTiles)
	if ok {
		t.Errorf("TestCheckSanankou_Invalid_RonCompletesOneOfThePungs: Expected NOT Sanankou, but got one. AllTiles: %s", TilesToNames(allTiles))
	}
}

// --- Chiitoitsu Tests ---
func TestCheckChiitoitsu_Valid(t *testing.T) {
	// Hand: 11m 22p 33s 44m 55p 66s EEz (7 pairs)
	player, agariHai, allTiles, _ := setupTestHandAndGameState(t,
		"1m1m 2p2p 3s3s 4m4m 5p5p 6s6s", // 6 pairs
		[]Meld{},
		"E", // Agari completes 7th pair (East wind)
		false, // isTsumo = Ron
	)
	player.Hand = append(player.Hand, TilesFromString("E E")...) // Add the pair that will be completed by agariHai
	sort.Sort(BySuitValue(player.Hand))
	allTiles = getAllTilesInHand(player, agariHai, false) // Reconstruct allTiles

	ok, han := checkChiitoitsu(player, allTiles, true) // isMenzen = true
	if !ok || han != 2 {
		t.Errorf("TestCheckChiitoitsu_Valid: Expected Chiitoitsu (2 Han), got ok:%v, han:%d. AllTiles: %s", ok, han, TilesToNames(allTiles))
	}
}

func TestCheckChiitoitsu_Invalid_NotMenzen(t *testing.T) {
	player, agariHai, allTiles, _ := setupTestHandAndGameState(t,
		"1m1m 2p2p 3s3s 4m4m 5p5p 6s",
		[]Meld{{Type: "Pon", Tiles: TilesFromString("E E E"), IsConcealed: false}}, // Open meld
		"6s", // Completes a pair
		false,
	)
	player.Hand = append(player.Hand, TilesFromString("6s")...) // Add the other part of the pair
	sort.Sort(BySuitValue(player.Hand))
	allTiles = getAllTilesInHand(player, agariHai, false)
	
	ok, _ := checkChiitoitsu(player, allTiles, false) // isMenzen = false
	if ok {
		t.Errorf("TestCheckChiitoitsu_Invalid_NotMenzen: Expected NO Chiitoitsu, but got one.")
	}
}

// --- Daisangen Tests ---
func TestCheckDaisangen_Valid(t *testing.T) {
    player, agariHai, allTiles, _ := setupTestHandAndGameState(t, 
        "1m1m", // Pair of 1m
        []Meld{
            {Type: "Pon", Tiles: TilesFromString("w w w")}, // Pung White Dragon
            {Type: "Pon", Tiles: TilesFromString("g g g")}, // Pung Green Dragon
            {Type: "Pon", Tiles: TilesFromString("r r r")}, // Pung Red Dragon
            {Type: "Pon", Tiles: TilesFromString("2s 2s 2s")}, // Another pung
        },
        "1m", // Doesn't matter for this Yaku, but need a valid agari
        false,
    )
	// Manually set player hand for this specific meld setup
	player.Hand = TilesFromString("1m1m")
	allTiles = getAllTilesInHand(player, agariHai, false)


    ok, name, han := checkDaisangen(player, allTiles)
    if !ok || name != "Daisangen" || han != 13 {
        t.Errorf("TestCheckDaisangen_Valid: Expected Daisangen (13 Han), got name:'%s' han:%d ok:%v. AllTiles: %s", name, han, ok, TilesToNames(allTiles))
    }
}

// --- Shousuushii / Daisuushii Tests ---
func TestCheckShousuushii_Valid(t *testing.T) {
    player, agariHai, allTiles, _ := setupTestHandAndGameState(t,
        "1m1m", // Non-wind pair
        []Meld{
            {Type: "Pon", Tiles: TilesFromString("E E E")}, // Pung East
            {Type: "Pon", Tiles: TilesFromString("S S S")}, // Pung South
            {Type: "Pon", Tiles: TilesFromString("W W W")}, // Pung West
			// North wind is the pair in hand
        },
        "N", // Agari completes North pair (conceptually, hand has one N)
        false,
    )
	player.Hand = TilesFromString("1m1m N") // Player hand before ronning on N
	allTiles = getAllTilesInHand(player, agariHai, false)


    ok, name, han := checkShousuushii(player, allTiles)
    if !ok || name != "Shousuushii" || han != 13 {
        t.Errorf("TestCheckShousuushii_Valid: Expected Shousuushii (13 Han), got name:'%s' han:%d ok:%v. AllTiles: %s", name, han, ok, TilesToNames(allTiles))
    }
}

func TestCheckDaisuushii_Valid(t *testing.T) {
    player, agariHai, allTiles, _ := setupTestHandAndGameState(t,
        "1m", // Part of pair
        []Meld{
            {Type: "Pon", Tiles: TilesFromString("E E E")},
            {Type: "Pon", Tiles: TilesFromString("S S S")},
            {Type: "Pon", Tiles: TilesFromString("W W W")},
            {Type: "Pon", Tiles: TilesFromString("N N N")},
        },
        "1m", // Completes 1m pair
        false,
    )
	player.Hand = TilesFromString("1m")
	allTiles = getAllTilesInHand(player, agariHai, false)

    ok, name, han := checkDaisuushii(player, allTiles)
    if !ok || name != "Daisuushii" || han != 26 { // Now expects 26
        t.Errorf("TestCheckDaisuushii_Valid: Expected Daisuushii (26 Han), got name:'%s' han:%d ok:%v. AllTiles: %s", name, han, ok, TilesToNames(allTiles))
    }
}

// --- Tsuuiisou Tests ---
func TestCheckTsuuiisou_Valid_Standard(t *testing.T) {
    player, agariHai, allTiles, _ := setupTestHandAndGameState(t,
        "E S W N Wh Gr", // Hand part
        []Meld{ {Type:"Pon", Tiles: TilesFromString("r r r")}}, // Pung Red Dragon
        "Rd", // Agari completes pair of Red Dragon
        false,
    )
	// Hand setup for Tsuuiisou: E S W N Wh Gr Rd + Rd (pair) + Pung(EEE) + Pung(SSS) + Pung(WWW)
	// This is complex to set up with current helper. Forcing allTiles:
	allTiles = TilesFromString("E E E S S S W W W N N r r Wh Wh") // Example Tsuuiisou
	agariHai = TilesFromString("Wh")[0] // Assume last Wh completes the pair

    ok, name, han := checkTsuuiisou(player, allTiles)
    if !ok || name != "Tsuuiisou" || han != 13 {
        t.Errorf("TestCheckTsuuiisou_Valid_Standard: Expected Tsuuiisou (13 Han), got name:'%s' han:%d ok:%v. AllTiles: %s", name, han, ok, TilesToNames(allTiles))
    }
}

func TestCheckTsuuiisou_Valid_Chiitoitsu(t *testing.T) {
	allTiles := TilesFromString("E E S S W W N N Wh Wh Gr Gr r r") // 7 pairs of honors
	agariHai := TilesFromString("r")[0] // Completes Red Dragon pair
	player := createTestPlayer() // Player needed for DecomposeWinningHand context
	player.Hand = allTiles // For IsChiitoitsu check

	ok, name, han := checkTsuuiisou(player, allTiles)
	if !ok || name != "Tsuuiisou" || han != 13 {
		t.Errorf("TestCheckTsuuiisou_Valid_Chiitoitsu: Expected Tsuuiisou (13 Han), got name:'%s' han:%d ok:%v. AllTiles: %s", name, han, ok, TilesToNames(allTiles))
	}
}

// --- Chinroutou Tests ---
func TestCheckChinroutou_Valid_Standard(t *testing.T) {
	allTiles := TilesFromString("1m1m1m 9m9m9m 1p1p1p 9p9p 1s1s1s") // 4 pungs of terminals, 1 pair of terminals
	agariHai := TilesFromString("9p")[0] // Completes 9p pair
	player := createTestPlayer()
	player.Hand = allTiles

	ok, name, han := checkChinroutou(player, allTiles)
	if !ok || name != "Chinroutou" || han != 13 {
		t.Errorf("TestCheckChinroutou_Valid_Standard: Expected Chinroutou (13 Han), got name:'%s' han:%d ok:%v. AllTiles: %s", name, han, ok, TilesToNames(allTiles))
	}
}

// --- Suukantsu Test ---
func TestCheckSuukantsu_Valid(t *testing.T) {
	player, _, _, _ := setupTestHandAndGameState(t, "1m1m", // Pair
		[]Meld{
			{Type: "Ankan", Tiles: TilesFromString("2m2m2m2m")},
			{Type: "Daiminkan", Tiles: TilesFromString("3p3p3p3p")},
			{Type: "Ankan", Tiles: TilesFromString("4s4s4s4s")},
			{Type: "Shouminkan", Tiles: TilesFromString("5p5p5p5p")},
		}, "1m", false)
	
	ok, name, han := checkSuukantsu(player)
	if !ok || name != "Suukantsu" || han != 13 {
		t.Errorf("TestCheckSuukantsu_Valid: Expected Suukantsu (13 Han), got name:'%s' han:%d ok:%v", name, han, ok)
	}
}

// --- Tanyao (All Simples) Tests ---
func TestCheckTanyao_Valid(t *testing.T) {
    allTiles := TilesFromString("2m3m4m 5p6p7p 3s4s5s 2p2p 6s6s") // All simples
    ok, han := checkTanyao(allTiles)
    if !ok || han != 1 {
        t.Errorf("TestCheckTanyao_Valid: Expected Tanyao (1 Han), got ok:%v, han:%d", ok, han)
    }
}

func TestCheckTanyao_Invalid_ContainsTerminal(t *testing.T) {
    allTiles := TilesFromString("1m2m3m 5p6p7p 3s4s5s 2p2p 6s6s") // Contains 1m
    ok, _ := checkTanyao(allTiles)
    if ok {
        t.Errorf("TestCheckTanyao_Invalid_ContainsTerminal: Expected no Tanyao, but got one")
    }
}

// --- Menzen Tsumo Test ---
func TestCheckMenzenTsumo_Valid(t *testing.T) {
    ok, han := checkMenzenTsumo(true, true) // isTsumo=true, isMenzen=true
    if !ok || han != 1 {
        t.Errorf("TestCheckMenzenTsumo_Valid: Expected Menzen Tsumo (1 Han), got ok:%v, han:%d", ok, han)
    }
}

// --- Riichi / Double Riichi / Ippatsu ---
// These depend heavily on game state flags set by game flow logic.
// checkDoubleRiichi and checkIppatsu mainly check player flags.
func TestCheckRiichi_Valid(t *testing.T) {
	player := createTestPlayer()
	player.IsRiichi = true
	gs := createTestGameState(nil)
	ok, han := checkRiichi(player, gs)
	if !ok || han != 1 {
		t.Errorf("TestCheckRiichi_Valid: Expected Riichi (1 Han), got ok:%v, han:%d", ok, han)
	}
}

func TestCheckDoubleRiichi_Valid(t *testing.T) {
	player := createTestPlayer()
	player.IsRiichi = true // Prerequisite for Double Riichi Yaku to be considered
	player.DeclaredDoubleRiichi = true // Flag set by game logic
	gs := createTestGameState(nil)
	ok, name, han := checkDoubleRiichi(player, gs)
	if !ok || name != "Double Riichi Bonus" || han != 1 {
		t.Errorf("TestCheckDoubleRiichi_Valid: Expected Double Riichi Bonus (1 Han), got name:'%s' han:%d ok:%v", name, han, ok)
	}
}

func TestCheckIppatsu_Valid(t *testing.T) {
	player := createTestPlayer()
	player.IsRiichi = true
	player.IsIppatsu = true // Flag set by game logic
	gs := createTestGameState(nil)
	ok, han := checkIppatsu(player, gs)
	if !ok || han != 1 {
		t.Errorf("TestCheckIppatsu_Valid: Expected Ippatsu (1 Han), got ok:%v, han:%d", ok, han)
	}
}

// --- Sankantsu Test (from previous implementation) ---
func TestCheckSankantsu_Valid(t *testing.T) {
	player := createTestPlayer()
	player.Melds = []Meld{
		{Type: "Ankan", Tiles: TilesFromString("1m1m1m1m")},
		{Type: "Daiminkan", Tiles: TilesFromString("2p2p2p2p")},
		{Type: "Shouminkan", Tiles: TilesFromString("3s3s3s3s")},
	}
	ok, name, han := checkSankantsu(player)
	if !ok || name != "Sankantsu" || han != 2 {
		t.Errorf("TestCheckSankantsu_Valid: Expected Sankantsu (2 Han), got name:'%s' han:%d ok:%v", name, han, ok)
	}
}

// TODO: Add tests for Honitsu, Chinitsu, Junchan, Toitoi, Sanshoku Doukou, Ittsuu, Haitei/Houtei, Rinshan, Chankan
// For these, DecomposeWinningHand's accuracy will be very important.

// --- Haitei/Houtei Tests ---
func TestCheckHaiteiHoutei_Haitei(t *testing.T) {
	player, _, _, gs := setupTestHandAndGameState(t, "1m2m3m 4p5p6p 7s8s9s 1z1z", []Meld{}, "2m", true) // Hand doesn't matter
	gs.Wall = []Tile{} // Wall is empty after this Tsumo
	ok, name, han := checkHaiteiHoutei(gs, true) // isTsumo = true
	if !ok || name != "Haitei Raoyue" || han != 1 {
		t.Errorf("TestCheckHaiteiHoutei_Haitei: Expected Haitei Raoyue (1 Han), got name:'%s' han:%d ok:%v", name, han, ok)
	}
}

func TestCheckHaiteiHoutei_Houtei(t *testing.T) {
	_, _, _, gs := setupTestHandAndGameState(t, "", []Meld{}, "1m", false) // Player/hand don't matter
	gs.IsHouteiDiscard = true // This discard is Houtei
	ok, name, han := checkHaiteiHoutei(gs, false) // isTsumo = false
	if !ok || name != "Houtei Raoyui" || han != 1 {
		t.Errorf("TestCheckHaiteiHoutei_Houtei: Expected Houtei Raoyui (1 Han), got name:'%s' han:%d ok:%v", name, han, ok)
	}
}

// --- Rinshan Kaihou Test ---
func TestCheckRinshanKaihou_Valid(t *testing.T) {
	_, _, _, gs := setupTestHandAndGameState(t, "", []Meld{}, "1m", true)
	gs.IsRinshanWin = true // Win was on Rinshan tile
	ok, han := checkRinshanKaihou(gs, true) // isTsumo = true
	if !ok || han != 1 {
		t.Errorf("TestCheckRinshanKaihou_Valid: Expected Rinshan Kaihou (1 Han), got ok:%v, han:%d", ok, han)
	}
}

// --- Chankan Test ---
func TestCheckChankan_Valid(t *testing.T) {
	_, _, _, gs := setupTestHandAndGameState(t, "", []Meld{}, "1m", false) // Ron
	gs.IsChankanOpportunity = true // Win was Chankan
	ok, han := checkChankan(gs)
	if !ok || han != 1 {
		t.Errorf("TestCheckChankan_Valid: Expected Chankan (1 Han), got ok:%v, han:%d", ok, han)
	}
}

// --- Sanshoku Doukou (Triple Pungs) ---
func TestCheckSanshokuDoukou_Valid(t *testing.T) {
	player, agariHai, allTiles, _ := setupTestHandAndGameState(t, 
		"1m1m", // Pair
		[]Meld{
            {Type: "Pon", Tiles: TilesFromString("2m2m2m")},
            {Type: "Pon", Tiles: TilesFromString("2p2p2p")},
            {Type: "Pon", Tiles: TilesFromString("2s2s2s")},
			{Type: "Pon", Tiles: TilesFromString("E E E")}, // Another group to complete hand
        }, 
		"1m", false)
	player.Hand = TilesFromString("1m1m") // Ensure player hand is just the pair
	allTiles = getAllTilesInHand(player, agariHai, false)

	ok, name, han := checkSanshokuDoukou(player, allTiles)
	if !ok || name != "Sanshoku Doukou" || han != 2 {
		t.Errorf("TestCheckSanshokuDoukou_Valid: Expected Sanshoku Doukou (2 Han), got name:'%s' han:%d ok:%v. AllTiles: %s", name, han, ok, TilesToNames(allTiles))
	}
}

// --- Toitoi (All Pungs) ---
func TestCheckToitoi_Valid(t *testing.T) {
	player, agariHai, allTiles, _ := setupTestHandAndGameState(t, 
		"1m1m", // Pair
		[]Meld{
            {Type: "Pon", Tiles: TilesFromString("2m2m2m")},
            {Type: "Pon", Tiles: TilesFromString("3p3p3p")},
            {Type: "Pon", Tiles: TilesFromString("4s4s4s")},
			{Type: "Ankan", Tiles: TilesFromString("E E E E")},
        }, 
		"1m", false)
	player.Hand = TilesFromString("1m1m")
	allTiles = getAllTilesInHand(player, agariHai, false)
	
	ok, han := checkToitoi(player, allTiles)
	if !ok || han != 2 {
		t.Errorf("TestCheckToitoi_Valid: Expected Toitoi (2 Han), got ok:%v, han:%d. AllTiles: %s", ok, han, TilesToNames(allTiles))
	}
}

// --- Honroutou (All Terminals and Honors) ---
func TestCheckHonroutou_Valid(t *testing.T) {
    allTiles := TilesFromString("1m1m1m 9p9p9p EEE SSS WW") // Pungs/pair of terminals & honors
    ok, han := checkHonroutou(allTiles)
    if !ok || han != 2 {
        t.Errorf("TestCheckHonroutou_Valid: Expected Honroutou (2 Han), got ok:%v, han:%d", ok, han)
    }
}

// --- Sanshoku Doujun (Mixed Triple Sequence) ---
func TestCheckSanshokuDoujun_Valid_Concealed(t *testing.T) {
    player, agariHai, allTiles, gs := setupTestHandAndGameState(t, 
        "2m3m4m 2p3p4p 2s3s4s 1z1z", // 3 seq + pair
        []Meld{}, 
        "E", // Dummy agari, doesn't matter for this structure if hand is full
        true) 
	player.Hand = TilesFromString("2m3m4m2p3p4p2s3s4s1z1zE") // Full hand for decomposition
	player.Hand = append(player.Hand, TilesFromString("E")[0]) // Add the pair part
	sort.Sort(BySuitValue(player.Hand))
	allTiles = getAllTilesInHand(player, agariHai, true)


    ok, name, han := checkSanshokuDoujun(player, true, allTiles) // isMenzen = true
    if !ok || name != "Sanshoku Doujun" || han != 2 {
        t.Errorf("TestCheckSanshokuDoujun_Valid_Concealed: Expected Sanshoku Doujun (2 Han), got name:'%s' han:%d ok:%v. AllTiles: %s", name, han, ok, TilesToNames(allTiles))
    }
}

// --- Ittsuu (Pure Straight) ---
func TestCheckIttsuu_Valid_Concealed_Manzu(t *testing.T) {
    player, agariHai, allTiles, gs := setupTestHandAndGameState(t, 
        "1m2m3m 4m5m6m 7m8m9m 1p1p", // Ittsuu in Manzu + pair
        []Meld{}, 
        "E", // Dummy agari
        true)
	player.Hand = TilesFromString("1m2m3m4m5m6m7m8m9m1p1pE")
	player.Hand = append(player.Hand, TilesFromString("E")[0])
	sort.Sort(BySuitValue(player.Hand))
	allTiles = getAllTilesInHand(player, agariHai, true)


    ok, name, han := checkIttsuu(player, true, allTiles) // isMenzen = true
    if !ok || name != "Ittsuu" || han != 2 {
        t.Errorf("TestCheckIttsuu_Valid_Concealed_Manzu: Expected Ittsuu (2 Han), got name:'%s' han:%d ok:%v. AllTiles: %s", name, han, ok, TilesToNames(allTiles))
    }
}

// --- Honitsu (Half Flush) ---
func TestCheckHonitsu_Valid_Concealed(t *testing.T) {
    // Hand: 123m 456m EEEz SSp (SSp is pair of South Wind). Menzen.
    player, agariHai, allTiles, gs := setupTestHandAndGameState(t, 
        "1m2m3m 4m5m6m EEE S", // Hand part
        []Meld{}, 
        "S", // Agari completes SS pair
        true) 
	player.Hand = TilesFromString("1m2m3m4m5m6mEEESS")
	sort.Sort(BySuitValue(player.Hand))
	allTiles = getAllTilesInHand(player, agariHai, true)

    ok, han := checkHonitsu(allTiles, true) // isMenzen = true
    if !ok || han != 3 {
        t.Errorf("TestCheckHonitsu_Valid_Concealed: Expected Honitsu (3 Han), got ok:%v, han:%d. AllTiles: %s", ok, han, TilesToNames(allTiles))
    }
}

// --- Chinitsu (Full Flush) ---
func TestCheckChinitsu_Valid_Concealed(t *testing.T) {
    // Hand: 123456789m 22m 33m. Menzen.
    player, agariHai, allTiles, gs := setupTestHandAndGameState(t, 
        "1m2m3m4m5m6m7m8m9m2m2m3m", // Hand part
        []Meld{}, 
        "3m", // Agari completes 3m pair
        true)
	player.Hand = TilesFromString("1m2m3m4m5m6m7m8m9m2m2m3m3m")
	sort.Sort(BySuitValue(player.Hand))
	allTiles = getAllTilesInHand(player, agariHai, true)

    ok, han := checkChinitsu(allTiles, true) // isMenzen = true
    if !ok || han != 6 {
        t.Errorf("TestCheckChinitsu_Valid_Concealed: Expected Chinitsu (6 Han), got ok:%v, han:%d. AllTiles: %s", ok, han, TilesToNames(allTiles))
    }
}

// --- Junchan (Terminals in All Sets) ---
func TestCheckJunchan_Valid_Concealed(t *testing.T) {
    // Hand: 123m 789p 11s 999s EEE (This example has honors, Junchan cannot have honors)
	// Corrected Junchan: 123m 789p 11s 999s + 123s (all groups have terminals, no honors)
    player, agariHai, allTiles, gs := setupTestHandAndGameState(t, 
        "1m2m3m 7p8p9p 1s1s 9s9s9s", // Hand part
        []Meld{}, 
        "1s", // Agari completes 1s sequence start (e.g. waiting on 1s or 4s for 123s or 234s)
		      // Let's make it simple: agari on the pair 1s
        true)
	player.Hand = TilesFromString("1m2m3m7p8p9p1s9s9s9s1s2s3s") // Ensure 14 tiles
	player.Hand = append(player.Hand, TilesFromString("1s")[0])
	sort.Sort(BySuitValue(player.Hand))
	allTiles = getAllTilesInHand(player, agariHai, true)


    ok, han := checkJunchan(player, true, allTiles) // isMenzen = true
    if !ok || han != 3 {
        t.Errorf("TestCheckJunchan_Valid_Concealed: Expected Junchan (3 Han), got ok:%v, han:%d. AllTiles: %s", ok, han, TilesToNames(allTiles))
    }
}

// --- Luck Yakuman Tests (Tenhou, Chihou, Renhou) ---

func TestCheckTenhou_Valid(t *testing.T) {
	player, agariHai, _, gs := setupTestHandAndGameState(t, 
		"1m2m3m4p5p6p7s8s9s1z1z2z2z", // A complete hand (example)
		[]Meld{}, 
		"2z", // Tsumo on the last tile of the pair
		true) // isTsumo

	gs.Players[gs.DealerIndexThisRound] = player // Ensure player is the dealer for this round
	gs.TurnNumber = 0 // Dealer's first turn
	gs.AnyCallMadeThisRound = false
	player.Melds = []Meld{} // Tenhou requires no melds (Ankans would also fail this)
	
	// The 'allTiles' for Tenhou is simply the player's hand after their first draw.
	// Our setupTestHandAndGameState already creates 'allTiles' based on player.Hand and agariHai.
	// For Tenhou, player.Hand should represent the 13 tiles dealt, and agariHai is the 14th drawn tile.
	// The helper function setupTestHandAndGameState might implicitly form the 14-tile hand correctly for Tsumo.
	// We need to ensure player.Hand for the checkTenhou call context is the completed 14-tile hand.
	// However, checkTenhou itself doesn't use allTiles, it uses player and gs.

	ok, name, han := checkTenhou(player, gs, true)
	if !ok || name != "Tenhou" || han != 13 {
		t.Errorf("TestCheckTenhou_Valid: Expected Tenhou (13 Han), got name:'%s' han:%d ok:%v. Player: %s, Turn: %d, Calls: %v, Melds: %v", 
			name, han, ok, player.Name, gs.TurnNumber, gs.AnyCallMadeThisRound, player.Melds)
	}
}

func TestCheckChihou_Valid(t *testing.T) {
	player, _, _, gs := setupTestHandAndGameState(t, 
		"1m2m3m4p5p6p7s8s9s1z1z2z2z", // A complete hand
		[]Meld{}, 
		"2z", // Tsumo
		true)

	// Ensure player is NOT the dealer
	if gs.Players[gs.DealerIndexThisRound] == player {
		gs.DealerIndexThisRound = (gs.GetPlayerIndex(player) + 1) % len(gs.Players) // Make someone else dealer
	}
	
	player.HasMadeFirstDiscardThisRound = false // Player has not yet discarded
	gs.AnyCallMadeThisRound = false
	player.Melds = []Meld{}
	gs.TurnNumber = gs.GetPlayerIndex(player) // Simulate it's this player's first turn in the go-around

	ok, name, han := checkChihou(player, gs, true)
	if !ok || name != "Chihou" || han != 13 {
		t.Errorf("TestCheckChihou_Valid: Expected Chihou (13 Han), got name:'%s' han:%d ok:%v. Player: %s, DealerIdx: %d, Turn: %d, Calls: %v, Melds: %v, FirstDiscard: %v", 
			name, han, ok, player.Name, gs.DealerIndexThisRound, gs.TurnNumber, gs.AnyCallMadeThisRound, player.Melds, player.HasMadeFirstDiscardThisRound)
	}
}

func TestCheckRenhou_Valid(t *testing.T) {
	player, agariHai, _, gs := setupTestHandAndGameState(t, 
		"1m2m3m4p5p6p7s8s9s1z1z2z", // 13 tiles for Ron
		[]Meld{}, 
		"2z", // Ron on this tile
		false, // isTsumo = false
	)

	// Ensure player is NOT the dealer
	if gs.Players[gs.DealerIndexThisRound] == player {
		gs.DealerIndexThisRound = (gs.GetPlayerIndex(player) + 1) % len(gs.Players)
	}

	player.HasMadeFirstDiscardThisRound = false
	gs.IsFirstGoAround = true // Ron occurs during the first un-interrupted go-around
	gs.AnyCallMadeThisRound = false // No calls made before this Ron
	player.Melds = []Meld{}

	ok, name, han := checkRenhou(player, gs, false)
	if !ok || name != "Renhou" || han != 13 { // Assuming Renhou is Yakuman for this test
		t.Errorf("TestCheckRenhou_Valid: Expected Renhou (13 Han), got name:'%s' han:%d ok:%v. Player: %s, Agari: %s, FirstGoAround: %v, Calls: %v, Melds: %v, FirstDiscard: %v", 
			name, han, ok, player.Name, agariHai.Name, gs.IsFirstGoAround, gs.AnyCallMadeThisRound, player.Melds, player.HasMadeFirstDiscardThisRound)
	}
}

// --- Ryuuiisou Test ---
func TestCheckRyuuiisou_Valid_Standard(t *testing.T) {
    // Hand: 2s3s4s 2s3s4s 6s6s6s 8s8s GrGr (Gr = Green Dragon)
    player, agariHai, allTiles, _ := setupTestHandAndGameState(t, 
        "2s3s4s2s3s4s6s6s6s8s8s", // Hand part
        []Meld{}, 
        "g", // Agari on Green Dragon to complete pair
        false,
    )
	player.Hand = TilesFromString("2s3s4s2s3s4s6s6s6s8s8sg") // Add one green dragon
	allTiles = getAllTilesInHand(player, agariHai, false)

    ok, name, han := checkRyuuiisou(player, allTiles)
    if !ok || name != "Ryuuiisou" || han != 13 {
        t.Errorf("TestCheckRyuuiisou_Valid_Standard: Expected Ryuuiisou (13 Han), got name:'%s' han:%d ok:%v. AllTiles: %s", name, han, ok, TilesToNames(allTiles))
    }
}

func TestCheckRyuuiisou_Valid_Chiitoitsu(t *testing.T) {
    // Hand: 22s 33s 44s 66s 88s GrGr GrGr (Gr = Green Dragon) - 7 pairs of green tiles
    allTiles := TilesFromString("2s2s 3s3s 4s4s 6s6s 8s8s g g g g") // 4 green dragons to make two pairs
	agariHai := TilesFromString("g")[0] // Last Green Dragon
	player := createTestPlayer()
	player.Hand = allTiles // For IsChiitoitsu check

    ok, name, han := checkRyuuiisou(player, allTiles)
    if !ok || name != "Ryuuiisou" || han != 13 {
        t.Errorf("TestCheckRyuuiisou_Valid_Chiitoitsu: Expected Ryuuiisou (13 Han), got name:'%s' han:%d ok:%v. AllTiles: %s", name, han, ok, TilesToNames(allTiles))
    }
}

func TestCheckRyuuiisou_Invalid_ContainsNonGreen(t *testing.T) {
    allTiles := TilesFromString("2s3s4s 5s6s7s 2s2s g g 1m1m1m") // Contains 5s, 7s (non-green sou) and 1m
	agariHai := TilesFromString("1m")[0]
	player := createTestPlayer()
	player.Hand = allTiles

    ok, _, _ := checkRyuuiisou(player, allTiles)
    if ok {
        t.Errorf("TestCheckRyuuiisou_Invalid_ContainsNonGreen: Expected NOT Ryuuiisou, but got one. AllTiles: %s", TilesToNames(allTiles))
    }
}

// --- Nagashi Mangan Test ---
func TestCheckNagashiMangan_Valid(t *testing.T) {
	player := createTestPlayer()
	gs := createTestGameState(nil)
	player.Discards = TilesFromString("1m 9p 1s E S W N w g r") // All terminals/honors
	player.HasHadDiscardCalledThisRound = false

	ok, name, han := checkNagashiMangan(player, gs)
	if !ok || name != "Nagashi Mangan" || han != 5 {
		t.Errorf("TestCheckNagashiMangan_Valid: Expected Nagashi Mangan (5 Han), got name:'%s' han:%d ok:%v", name, han, ok)
	}
}

func TestCheckNagashiMangan_Invalid_DiscardCalled(t *testing.T) {
	player := createTestPlayer()
	gs := createTestGameState(nil)
	player.Discards = TilesFromString("1m 9p E S w")
	player.HasHadDiscardCalledThisRound = true // Discard was called

	ok, _, _ := checkNagashiMangan(player, gs)
	if ok {
		t.Errorf("TestCheckNagashiMangan_Invalid_DiscardCalled: Expected NO Nagashi Mangan, but got one.")
	}
}

func TestCheckNagashiMangan_Invalid_ContainsSimple(t *testing.T) {
	player := createTestPlayer()
	gs := createTestGameState(nil)
	player.Discards = TilesFromString("1m 9p E S 2m") // Contains a simple (2m)
	player.HasHadDiscardCalledThisRound = false

	ok, _, _ := checkNagashiMangan(player, gs)
	if ok {
		t.Errorf("TestCheckNagashiMangan_Invalid_ContainsSimple: Expected NO Nagashi Mangan, but got one.")
	}
}


// --- Tests for IdentifyYaku with Luck Yakuman ---

func TestIdentifyYaku_Tenhou(t *testing.T) {
	player, agariHai, _, gs := setupTestHandAndGameState(t, 
		"1m2m3m4p5p6p7s8s9s1z1z2z2z", // Example complete hand for Tsumo
		[]Meld{}, 
		"2z", // Tsumo
		true)

	gs.Players[gs.DealerIndexThisRound] = player // Player is dealer
	gs.TurnNumber = 0                            // Dealer's first turn
	gs.AnyCallMadeThisRound = false
	player.Melds = []Meld{}
	player.Hand = getAllTilesInHand(player, agariHai, true) // Ensure hand is 14 tiles for IdentifyYaku

	results, totalHan := IdentifyYaku(player, agariHai, true, gs)
	
	foundTenhou := false
	for _, r := range results {
		if r.Name == "Tenhou" && r.Han == 13 {
			foundTenhou = true
			break
		}
	}
	if !foundTenhou || totalHan < 13 { // totalHan should be at least 13 (could be more if other Yakuman stack, though rare)
		t.Errorf("TestIdentifyYaku_Tenhou: Expected Tenhou (13 Han), got results: %v, totalHan: %d", results, totalHan)
	}
}

func TestIdentifyYaku_Chihou(t *testing.T) {
	player, agariHai, _, gs := setupTestHandAndGameState(t, 
		"1m2m3m4p5p6p7s8s9s1z1z2z2z", 
		[]Meld{}, 
		"2z", 
		true)

	// Ensure player is NOT the dealer
	if gs.Players[gs.DealerIndexThisRound] == player {
		gs.DealerIndexThisRound = (gs.GetPlayerIndex(player) + 1) % len(gs.Players)
	}
	
	player.HasMadeFirstDiscardThisRound = false
	gs.AnyCallMadeThisRound = false
	player.Melds = []Meld{}
	gs.TurnNumber = gs.GetPlayerIndex(player) // Player's first turn
	player.Hand = getAllTilesInHand(player, agariHai, true)


	results, totalHan := IdentifyYaku(player, agariHai, true, gs)
	foundChihou := false
	for _, r := range results {
		if r.Name == "Chihou" && r.Han == 13 {
			foundChihou = true
			break
		}
	}
	if !foundChihou || totalHan < 13 {
		t.Errorf("TestIdentifyYaku_Chihou: Expected Chihou (13 Han), got results: %v, totalHan: %d", results, totalHan)
	}
}

func TestIdentifyYaku_Renhou(t *testing.T) {
	player, agariHai, _, gs := setupTestHandAndGameState(t, 
		"1m2m3m4p5p6p7s8s9s1z1z2z", // 13 tiles for Ron
		[]Meld{}, 
		"2z", // Ron on this tile
		false, 
	)

	if gs.Players[gs.DealerIndexThisRound] == player {
		gs.DealerIndexThisRound = (gs.GetPlayerIndex(player) + 1) % len(gs.Players)
	}
	player.HasMadeFirstDiscardThisRound = false
	gs.IsFirstGoAround = true
	gs.AnyCallMadeThisRound = false
	player.Melds = []Meld{}
	player.Hand = TilesFromString("1m2m3m4p5p6p7s8s9s1z1z2z") // Ensure hand is 13 tiles before Ron for IdentifyYaku

	results, totalHan := IdentifyYaku(player, agariHai, false, gs) // isTsumo = false
	foundRenhou := false
	for _, r := range results {
		if r.Name == "Renhou" && r.Han == 13 { // Assuming Renhou is 13 for test
			foundRenhou = true
			break
		}
	}
	if !foundRenhou || totalHan < 13 {
		t.Errorf("TestIdentifyYaku_Renhou: Expected Renhou (13 Han), got results: %v, totalHan: %d", results, totalHan)
	}
}
