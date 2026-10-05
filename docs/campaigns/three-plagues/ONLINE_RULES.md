# The Three Plagues - Online rules

**Last Updated**: 2026-10-05 15:53 EDT

These are the rules the online rules engine enforces for heroes played from a seat or by a
bot (`docs/ONLINE_AND_RULES_PLAN.md`). The combat math, classes and abilities are in
`RULES_AND_CLASSES.md`. This file covers what the table leaves to the GM's eye: movement,
doors, sight, reveals, traps, searching, reactions and monster behavior.

At the table nothing changes: the GM runs the tracker in table mode, which records only. Even
in rules mode the GM can override anything.

## Decisions (GM, 2026-10-05)

### Movement
- **D1. Rolled each turn.**
  - A hero rolls their class's movement dice (2d6 by default) when they start moving; the
    engine rolls.
  - Aimed Shot halves the roll (this round and next).
  - Monsters move a fixed number of squares.
- **D2. No diagonal steps.** Unburdened Charge keeps its own rule (a diagonal step costs 2
  spaces).
- **D3. Passing pieces.**
  - Heroes may move through other heroes but never through monsters; monsters likewise pass
    through monsters but not heroes.
  - A move must end on an empty square.
  - A monster that covers several squares blocks all of them.
- **D4. No split move.** A turn is Move then Action, or Action then Move. Moving ends the
  movement step.

### Doors
- **D5. Opening is free, and doors stay open.**
  - A hero beside a door opens it for free during their move.
  - An opened door stays open.
  - Monsters move through open doors but don't open closed ones.

### Sight
The line runs from the center of one square to the center of the other. A line that only
touches a wall corner is still clear. There are two kinds of sight:
- **D6. Attack sight** (the Ranger's basic attack, Smite, Multi-Shot and other ranged
  abilities): blocked by walls, closed doors, blocked squares, tall furniture, and any hero
  or monster in between.
- **D7. View sight** (what the heroes see; this drives reveals):
  - It is true line of sight, square by square, everywhere: into a room through an open door,
    and inside rooms too. From outside a wide room a hero may not see its near corners.
  - Tall furniture blocks the view; heroes and monsters don't.
  - A square comes into view after any move or door opening that gives some hero a clear view
    of it.
  - A piece (monster, furniture, door) is revealed when any of its squares is in view.
  - Hidden traps and unfound secret doors stay hidden until searched for, triggered or found.

### Traps
- **D8. Stepping on a hidden trap triggers it, and the move ends there.** The Rogue's
  Quivering Boots keep their own rule. Until each trap kind's effect is entered as data, the
  GM resolves the effect.

### Searching
- **D9. Treasure, traps and secret doors are three separate search actions**, each taking
  the hero's action.
  - **Treasure:**
    - The hero must stand next to the thing searched (a chest, a cupboard, a weapons rack).
      This is the campaign's big change from the original game, which searched whole rooms:
      it makes traps more likely and more punishing.
    - Each piece (a chest, a cupboard, a fireplace) can be searched for treasure **once per
      party**: after any hero has searched it, nobody can search it again in that quest.
      The original game allowed one search per hero per room.
    - A quest can switch treasure searching off.
  - **Traps** and **secret doors:** each search reveals the hidden ones of its kind in the
    hero's view. These searches have no limit, because searching again from the same spot
    finds nothing new.
  - **No searching during combat.** No kind of search is allowed while any monster is
    revealed on the board, whether or not it is in line of sight. A revealed monster means
    combat is active (GM, 2026-10-05).
  - Some traps can't be found by searching (Quest 1's teleport trap); they are flagged
    `unsearchable`.
  - Rewards are written as data (gold or a loot-list item). Until then the GM resolves them.

### Reactions
- **D10. A prompt with a timer.**
  - When a reaction could be used (Riposte, Vanish From Sight, a potion between attacks), the
    monster's action pauses and that player gets a prompt, about 15 seconds.
  - When the time runs out, the reaction isn't used.
  - Bots answer at once.

### Monsters
- **D11. The GM, or the AI GM, chooses the order.** Each monster acts once per round.
  - A monster may act before it moves, and there are no free attacks against a piece that
    moves away.
  - So the AI should find the **rotation**: a monster attacks a hero next to it, then steps
    away so another monster can take its place and hit the same hero. The AI plans the whole
    monster phase, not one monster at a time.
- **D12. Tactical behavior.**
  - Monsters are dormant until spotted. Once spotted, they focus the weakest or most exposed
    hero, surround, rotate attackers, and archers back away to keep their range.
  - Each monster type gets its own profile when the AI is built.
  - **Consequence for balance:** the simulator so far assumes a random target and at most two
    melee attackers a round (heroes holding a doorway). Against this AI, online fights are
    likely harder than at the table. Balance is re-checked with the headless simulator once
    it exists. The table game is unaffected, because the GM runs the monsters there.

## Answered
- **Searching with monsters revealed** (2026-10-05): not allowed, for treasure, traps and
  secret doors alike, as long as any monster is revealed on the board (see D9).
- **Treasure searches** (2026-10-05): once per piece of furniture per party, not once per
  hero per room (see D9).
