```markdown
# Mahjong-Go

## Overview

This project is a command-line implementation of the Japanese Riichi Mahjong game, written in Go.

## Features Implemented

This project is a Go-based implementation of Riichi Mahjong. The following features have been implemented:

### Phase 1: Core Game Mechanics & Flow
*   **Game Start & Seating:** Randomized initial dealer, seat wind assignment.
*   **Dealing Mechanism:** Traditional wall breaking and dealing pattern.
*   **Turn Progression:** Correct turn cycling, turn counting.
*   **Wall Management:** Live wall, dead wall, Dora (initial, Kan-Dora, Ura-Dora), Rinshan draws.
*   **Exhaustive Draw (Ryuukyoku):**
    *   Triggered when live wall is empty.
    *   Tenpai/Notenpai player checks at Ryuukyoku.
    *   Noten Bappu point exchange (3000 points total distributed).
    *   Dealer retention (Renchan) if dealer is Tenpai. Honba increments regardless if dealer passes due to Noten.
*   **Abortive Draws:**
    *   Kyuushuu Kyuuhai (Nine Unique Terminals/Honors) implemented.
    *   Ssuufon Renda (Four players discard same wind on first non-interrupted turn).
    *   Suu Riichi (Four players declare Riichi, round aborts if 4th Riichi discard is not Ronned).
    *   Sanchahou (Three players Ron on the same discard).
    *   Suukaikan (Four Kans by two or more different players leading to no more Rinshan tiles).
*   **Honba & Riichi Sticks:**
    *   Honba increments on dealer retention (dealer win or dealer Tenpai draw) and on some abortive draws (ruleset dependent, current basic increment).
    *   Riichi sticks added on declaration, collected by winner, carried over.
*   **Dealer Retention (Renchan):** Implemented for dealer win or dealer Tenpai at Ryuukyoku.
*   **Game End Conditions:**
    *   Hanchan end (configurable `MaxWindRounds`, e.g., East & South by default).
    *   Player busting (score < 0). Basic game end, no advanced Tobu/Dobon scoring.
    *   **Agari Yame / Tenpai Yame:** The dealer has the option to end the game if all the following conditions are met: it's the final programmed turn of the game (e.g., South 4), the dealer is the winner of the hand (Agari Yame) or Tenpai in a drawn round (Tenpai Yame), and the dealer is the top-scoring player. If chosen, the game ends immediately. If declined at the absolute end of programmed rounds, the game still ends as per normal round limits.

### Phase 2: Yaku Implementation & Validation
*   **Hand Decomposition:** Core logic (`DecomposeWinningHand`) for standard hands.
*   **Yakuman Implemented:**
    *   Kokushi Musou (Thirteen Orphans) - including Juusanmenmachi (Double Yakuman).
    *   Suuankou (Four Concealed Pungs) - including Tanki (Double Yakuman).
    *   Daisangen (Big Three Dragons).
    *   Shousuushii (Little Four Winds).
    *   Daisuushii (Big Four Winds) - Double Yakuman.
    *   Tsuuiisou (All Honors) - standard and Chiitoitsu forms.
    *   Chinroutou (All Terminals) - standard and Chiitoitsu forms.
    *   Ryuuiisou (All Green) - standard and Chiitoitsu forms.
    *   Chuuren Poutou (Nine Gates) - including Junsei (Double Yakuman).
    *   Suukantsu (Four Kans).
    *   Tenhou (Blessing of Heaven), Chihou (Blessing of Earth), Renhou (Hand From Man) - valued as Yakuman.
*   **Regular Yaku Implemented:**
    *   **1 Han:** Riichi, Ippatsu, Menzen Tsumo, Pinfu, Tanyao (Kuitan allowed), Yakuhai (Seat/Prevalent/Dragons), Haitei Raoyue / Houtei Raoyui, Rinshan Kaihou, Chankan. Double Riichi (bonus 1 Han to Riichi).
    *   **2 Han:** Sanshoku Doukou, Chiitoitsu, Toitoihou, Sanankou, Shousangen, Honroutou, Sankantsu.
    *   **3+ Han:** Sanshoku Doujun (2 open, 3 closed), Ittsuu (1 open, 2 closed), Ryanpeikou (3 closed), Junchan Taiyao (2 open, 3 closed), Honitsu (2 open, 3 closed), Chinitsu (5 open, 6 closed).
*   **Yaku Precedence:** Yakuman > Regular Yaku. Chinitsu > Honitsu. Ryanpeikou > Iipeikou.
*   **Dora Handling:** Dora, Aka-Dora, Kan-Dora, Ura-Dora. Dora do not enable a win alone.
*   **Unit Tests:** Comprehensive suite in `yaku_test.go`.

### Phase 3: Scoring System (Fu & Points)
*   **Detailed Fu Calculation (`fu_calculation.go`):**
    *   Base Fu, Win Method Fu, Wait Pattern Fu, Pair Fu, Group Fu.
    *   Special Fu Cases: Chiitoitsu (25), Pinfu Tsumo (20), Pinfu Ron (30).
    *   Rounding up to nearest 10 Fu. Minimum 30 Fu (non-Pinfu/Chiitoi).
*   **Point Calculation (`rules.go`):**
    *   Full Mahjong score table: Mangan, Haneman, Baiman, Sanbaiman, Yakuman, Kazoe Yakuman.
    *   Correct payment calculations for Ron and Tsumo (Dealer vs. Non-dealer).
    *   Honba bonus applied.

### Phase 4: Special Rules & Conditions
*   **Advanced Furiten Logic (`rules.go`):**
    *   Temporary Furiten (own discards).
    *   Temporary Furiten (missed Ron, lasts until player's next discard).
    *   Permanent Riichi Furiten (missing Ron on a Riichi wait).
*   **Pao (Responsibility/Liability for Yakuman):**
    *   Pao is implemented for Daisangen and Daisuushii. If the Yakuman is achieved by Ron, the liable player (discarder or enabler of the final meld) pays the full Ron value. If achieved by Tsumo, the liable player now pays the winner an amount equivalent to the Ron value of the Yakuman; other players do not contribute to the Yakuman point payment in this Tsumo Pao scenario.
*   **Ryanhan Shibari (Two-Han Minimum):**
    *   Implemented if Honba >= 5 (configurable). Dora do not count towards minimum.

### Phase 5: Calls and Interruptions (`actions.go`, `checks.go`)
*   **Multiple Callers:**
    *   Atamahane (head bump) for multiple Ron.
    *   Priority: Kan > Pon > Chi. Closest player for same-priority.
*   **Ippatsu Interruption:**
    *   Ankan by Riichi player does not break Ippatsu. Other calls do.

### Phase 6: Riichi Mechanics (`actions.go`, `checks.go`)
*   **Riichi Discard Restrictions:**
    *   Implemented: Riichi player must discard drawn tile if no Kan (using `player.JustDrawnTile`).
*   **Actions During Riichi:**
    *   Wait change checks for Ankan/Shouminkan during Riichi are implemented using `checkWaitChangeForRiichiKan` (which utilizes `compareTileSlicesUnordered`). Ankan/Shouminkan are allowed only if the player's waits do not change.

## Future Enhancements / To-Do (Selected)

*   **AI Player Enhancement.**
*   **Robust User Interface and Input Validation.**
*   **Comprehensive Integration Testing.**
```