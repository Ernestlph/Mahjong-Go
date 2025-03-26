package main

import (
	"fmt"
	"math/rand"
	"sort"
)

// NewTile creates a new tile instance
func NewTile(suit string, value int, name string, isRed bool, id int) Tile {
	return Tile{Suit: suit, Value: value, Name: name, IsRed: isRed, ID: id}
}

// GenerateDeck creates a standard mahjong deck of 136 tiles and shuffles it.
func GenerateDeck() []Tile {
	var deck []Tile
	suits := []string{"Man", "Pin", "Sou"}
	winds := []string{"East", "South", "West", "North"} // Value 1, 2, 3, 4
	dragons := []string{"White", "Green", "Red"}        // Value 1, 2, 3 (Haku, Hatsu, Chun)
	idCounter := 0

	// Numbered tiles (1-9 for Man, Pin, Sou)
	for _, suit := range suits {
		for value := 1; value <= 9; value++ {
			hasRed := suit != "Sou" && value == 5 // Man and Pin have red 5s
			for i := 0; i < 4; i++ {
				isRed := hasRed && i == 0 // Make the first '5' tile red
				tileName := fmt.Sprintf("%s %d", suit, value)
				if isRed {
					tileName = "Red " + tileName
				}
				deck = append(deck, NewTile(suit, value, tileName, isRed, idCounter))
				idCounter++
			}
		}
	}

	// Wind tiles
	for i, wind := range winds {
		windValue := i + 1
		for j := 0; j < 4; j++ {
			deck = append(deck, NewTile("Wind", windValue, wind, false, idCounter))
			idCounter++
		}
	}

	// Dragon tiles
	for i, dragon := range dragons {
		dragonValue := i + 1
		for j := 0; j < 4; j++ {
			deck = append(deck, NewTile("Dragon", dragonValue, dragon, false, idCounter))
			idCounter++
		}
	}

	if len(deck) != TotalTiles {
		panic(fmt.Sprintf("Internal error: Generated deck size is %d, expected %d", len(deck), TotalTiles))
	}

	// Shuffle the deck using the seeded random source
	rand.Shuffle(len(deck), func(i, j int) {
		deck[i], deck[j] = deck[j], deck[i]
	})

	return deck
}

// GetAllPossibleTiles returns a sorted list of all 34 unique tile types (ignoring duplicates/reds).
// Useful for Tenpai checks.
func GetAllPossibleTiles() []Tile {
	uniqueTiles := []Tile{}
	// Use a predictable way to generate one of each type
	suits := []string{"Man", "Pin", "Sou"}
	winds := []string{"East", "South", "West", "North"}
	dragons := []string{"White", "Green", "Red"}

	idCounter := -1 // Use negative IDs to distinguish from real deck IDs if needed

	for _, suit := range suits {
		for value := 1; value <= 9; value++ {
			tileName := fmt.Sprintf("%s %d", suit, value)
			uniqueTiles = append(uniqueTiles, NewTile(suit, value, tileName, false, idCounter))
			idCounter--
		}
	}
	for i, wind := range winds {
		uniqueTiles = append(uniqueTiles, NewTile("Wind", i+1, wind, false, idCounter))
		idCounter--
	}
	for i, dragon := range dragons {
		uniqueTiles = append(uniqueTiles, NewTile("Dragon", i+1, dragon, false, idCounter))
		idCounter--
	}

	sort.Sort(BySuitValue(uniqueTiles))
	return uniqueTiles
}
