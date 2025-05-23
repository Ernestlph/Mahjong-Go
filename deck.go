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
			// Man 5, Pin 5, Sou 5 each have one red version.
			// Total of 4 of each number tile. One of the 5s is red.
			isFive := (value == 5)
			for i := 0; i < 4; i++ {
				// Make the first '5' tile of Man, Pin, Sou red
				isRed := isFive && i == 0
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
	for i, dragonName := range dragons {
		dragonValue := i + 1
		for j := 0; j < 4; j++ {
			deck = append(deck, NewTile("Dragon", dragonValue, dragonName, false, idCounter))
			idCounter++
		}
	}

	if len(deck) != TotalTiles {
		panic(fmt.Sprintf("Internal error: Generated deck size is %d, expected %d", len(deck), TotalTiles))
	}

	rand.Shuffle(len(deck), func(i, j int) {
		deck[i], deck[j] = deck[j], deck[i]
	})

	return deck
}

// GetAllPossibleTiles returns a sorted list of all 34 unique tile types (ignoring duplicates/reds).
func GetAllPossibleTiles() []Tile {
	uniqueTiles := []Tile{}
	suits := []string{"Man", "Pin", "Sou"}
	winds := []string{"East", "South", "West", "North"}
	dragons := []string{"White", "Green", "Red"}
	idCounter := -1

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
	for i, dragonName := range dragons {
		uniqueTiles = append(uniqueTiles, NewTile("Dragon", i+1, dragonName, false, idCounter))
		idCounter--
	}

	sort.Sort(BySuitValue(uniqueTiles))
	return uniqueTiles
}
