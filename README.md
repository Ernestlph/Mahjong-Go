# Mahjong-Go

## Overview

This project is a command-line implementation of the Japanese Riichi Mahjong game, written in Go.

## Features Implemented

This project is a Go-based implementation of Riichi Mahjong. The following features have been implemented:

### Phase 1: Core Game Mechanics & Flow
*   **Game Start & Seating:** Randomized initial dealer selection and appropriate seat wind assignment.
*   **Dealing Mechanism:** Traditional wall breaking and dealing pattern implemented (13 tiles per player).
*   **Turn Progression:** Correct turn cycling (`NextPlayer`) and turn counting (`TurnNumber`).
*   **Wall Management:** Accurate handling of live wall, dead wall, Dora indicators (initial, Kan-Dora, Ura-Dora), and Rinshan (replacement tile) draws.
*   **Exhaustive Draw (Ryuukyoku):** Basic trigger implemented for when the live wall is empty and no win occurs.
*   **Abortive Draws:** Kyuushuu Kyuuhai (Nine Unique Terminals/Honors) is implemented, allowing a player to declare an abortive draw on their first non-interrupted turn.
*   **Honba & Riichi Sticks:**
    *   Honba counter increments on dealer retention (dealer win or draw - simplified rules currently).
    *   Riichi sticks are correctly added upon declaration, collected by the winner, and carried over on draws.
*   **Dealer Retention (Renchan):** Basic logic implemented for the dealer to retain their turn upon winning or (currently) any draw.

### Phase 2: Yaku Implementation & Validation
*   **Hand Decomposition:** Core logic (`DecomposeWinningHand`) for breaking down standard winning hands (4 melds + 1 pair) is in place and reviewed.
*   **Yakuman Implemented:**
    *   Kokushi Musou (Thirteen Orphans) - including 13-sided wait (Double Yakuman).
    *   Suuankou (Four Concealed Pungs) - including Tanki (pair wait) (Double Yakuman).
    *   Daisangen (Big Three Dragons).
    *   Shousuushii (Little Four Winds).
    *   Daisuushii (Big Four Winds) - valued as Double Yakuman (26 Han).
    *   Tsuuiisou (All Honors) - standard and Chiitoitsu forms.
    *   Chinroutou (All Terminals) - standard and Chiitoitsu forms.
    *   Ryuuiisou (All Green) - standard and Chiitoitsu forms.
    *   Chuuren Poutou (Nine Gates) - including Junsei (9-sided wait) (Double Yakuman).
    *   Suukantsu (Four Kans).
    *   Tenhou (Blessing of Heaven), Chihou (Blessing of Earth), Renhou (Hand From Man).
*   **Regular Yaku Implemented:**
    *   **1 Han:** Riichi, Ippatsu, Menzen Tsumo, Pinfu, Tanyao (Kuitan allowed), Yakuhai (Seat/Prevalent/Dragons), Haitei Raoyue / Houtei Raoyui, Rinshan Kaihou, Chankan. Double Riichi is supported via a player flag.
    *   **2 Han:** Sanshoku Doukou (Triple Pungs), Chiitoitsu (Seven Pairs), Toitoihou (All Pungs), Sanankou (Three Concealed Pungs), Shousangen (Little Three Dragons), Honroutou (All Terminals & Honors), Sankantsu (Three Kans).
    *   **3+ Han:** Sanshoku Doujun (Mixed Triple Sequence), Ittsuu (Pure Straight), Ryanpeikou (Two Pure Double Sequences), Junchan Taiyao (Terminals in All Sets), Honitsu (Half Flush), Chinitsu (Full Flush).
*   **Yaku Precedence:** Logic in place for Yakuman > Regular Yaku, Chinitsu > Honitsu, Ryanpeikou > Iipeikou.
*   **Dora Handling:** Correct calculation for Dora, Aka-Dora (Red Fives assumed if `IsRed` flag is used on tiles), Kan-Dora, and Ura-Dora. Dora do not contribute to winning a hand if no other Yaku are present.
*   **Unit Tests:** A comprehensive suite of unit tests (`yaku_test.go`) has been created for Yaku functions, covering various valid and invalid scenarios.

## Future Enhancements / To-Do

The following features and areas are planned for future development to create a more complete Riichi Mahjong game:

### Phase 1: Core Game Mechanics & Flow (Remaining)
*   **Exhaustive Draw (Ryuukyoku) Details:**
    *   Implement Tenpai/Notenpai player checks at Ryuukyoku.
    *   Handle point exchange based on Tenpai status.
    *   Refine dealer retention (Renchan) logic: dealer must be Tenpai for Renchan on Ryuukyoku; if dealer is Noten, dealership passes and Honba still increments.
*   **Additional Abortive Draws:**
    *   Implement Ssuufon Renda (Four players discard same wind on first non-interrupted turn).
    *   Implement Suu Riichi (Four players declare Riichi).
    *   Implement Sanchahou (Three players Ron on the same discard).
    *   Implement Suukaikan (Four Kans by two or more different players - requires ruleset clarification for exact trigger and if it's always abortive).
*   **Game End Conditions:**
    *   Implement Hanchan end (e.g., after South 4, or West 4 if game extends due to dealer wins/Tenpai).
    *   Implement player busting (score < 0) with options to continue or end game.
    *   Implement scoring penalties for busting (Tobu/Dobon).

### Phase 3: Scoring System (Fu & Points)
*   **Detailed Fu Calculation (`fu_calculation.go`):**
    *   Thoroughly verify and implement all Fu calculation components:
        *   Base Fu (20 Fu).
        *   Win Method Fu: Menzen Ron (+10 Fu), Tsumo (+2 Fu, excluding Pinfu Tsumo).
        *   Wait Pattern Fu (+2 Fu for Kanchan, Penchan, Tanki; 0 Fu for Ryanmen).
        *   Pair Fu (+2/+4 Fu for Dragon, Seat Wind, Prevalent Wind, Double Wind pairs).
        *   Group Fu for Triplets/Quads (Open/Concealed, Simple/Terminal/Honor).
    *   Implement Special Fu Cases:
        *   Chiitoitsu (fixed 25 Fu).
        *   Pinfu Tsumo (20 Fu total).
        *   Pinfu Ron (30 Fu total).
        *   Consider rules for "Open Pinfu-like hands" rounding up to 30 Fu.
    *   Ensure correct rounding up to nearest 10 Fu.
    *   Verify minimum Fu rules (e.g., 30 Fu for non-Pinfu/Chiitoitsu).
*   **Point Calculation (`rules.go`):**
    *   Implement the full Mahjong score table:
        *   Mangan (5 Han, or 3 Han 70+ Fu, 4 Han 40+ Fu).
        *   Haneman (6-7 Han).
        *   Baiman (8-10 Han).
        *   Sanbaiman (11-12 Han).
        *   Yakuman (13+ Han or specific Yakuman Yaku).
        *   Kazoe Yakuman (Counted Yakuman - already implicitly handled by point limits).
    *   Verify correct payment calculations for Ron and Tsumo (Dealer vs. Non-dealer, payments from whom to whom).
    *   Ensure Honba bonus (+300 per Honba for Ron, +100 per Honba per player for Tsumo) is correctly applied.

### Phase 4: Special Rules & Conditions
*   **Advanced Furiten Logic (`rules.go`):**
    *   Implement Temporary Furiten due to missed Ron (lasts until player's next discard).
    *   Implement Permanent Riichi Furiten (missing Ron on a Riichi wait makes player Furiten for all waits for the rest of the hand, Tsumo only).
    *   Refine Furiten clearing logic.
*   **Pao (Responsibility/Liability for Yakuman):**
    *   Implement rules for Pao (e.g., for Daisangen, Daisuushii), where a player whose call enables another's Yakuman pays all/part of the score.
*   **Ryanhan Shibari (Two-Han Minimum):**
    *   Implement (if desired) a 2-Han minimum requirement to win after a certain number of Honba (e.g., 5). Dora do not count towards this minimum.
*   **Multiple/Combined Yakuman Scoring:**
    *   Implement rules for scoring combined Yakuman (e.g., Daisangen + Tsuuiisou as a Double Yakuman, or specific combinations having unique values) if the "first Yakuman wins" rule is to be expanded.

### Phase 5: Calls and Interruptions (`actions.go`, `checks.go`)
*   **Multiple Callers:**
    *   Implement Atamahane (head bump) for multiple Ron declarations (player closest to discarder wins).
    *   Handle priority for multiple Pon/Kan calls (closest player or Kan > Pon).
*   **Call Mechanics:**
    *   Thoroughly test `HandlePonAction`, `HandleChiAction`, `HandleKanAction`.
    *   Ensure `removeLastDiscardFromPlayer` is robust.
*   **Ippatsu Interruption Nuances:**
    *   Refine Ippatsu breaking: Ankan by the Riichi player should generally not break their Ippatsu. Other calls (including Shouminkan by Riichi player if it changes waits, or Daiminkan) should.

### Phase 6: Riichi Mechanics (`actions.go`, `checks.go`, `main.go`)
*   **Riichi Discard Restrictions:**
    *   Implement logic to force a Riichi player to discard the tile they just drew if they declare Riichi after drawing (unless they declare Kan with the drawn tile). This requires robust tracking of the "just drawn tile".
*   **Actions During Riichi:**
    *   Implement wait change checks for Ankan/Shouminkan declared during Riichi (rules vary, common is allowed if waits don't change, or Ankan always allowed).

### Phase 7: Broader Testing and Refinement
*   **Integration Tests:** Simulate mini-games or specific scenarios to test interactions between modules.
*   **Full Game Testing:** Play through many full games to catch bugs in game flow, scoring, and complex rule interactions.
*   **Code Review & Refactoring:** Conduct peer reviews, address warnings, and optimize code.
*   **AI Player Enhancement:** Improve AI decision-making (currently very basic).