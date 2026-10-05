# The Three Plagues - Online rules

**Last Updated**: 2026-10-05 16:42 EDT

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

### Attacks
- **Basic attack reach** follows the class's reach (`RULES_AND_CLASSES.md`, "Who a basic attack
  reaches"), set on the class form:
  - **Adjacent:** orthogonally beside, with no wall or closed door between.
  - **Diagonal:** adjacent or diagonal. A diagonal only counts around a corner that leaves a
    gap.
  - **Line of sight:** attack sight (D6).
- **Ranged attacks have no range limit** (GM, 2026-10-05). The Ranger's bow, Smite, Multi-Shot
  and ranged monsters reach anything in attack sight, however far away, as the original
  crossbow did.
- **The engine rolls.** The hero's hit dice plus Accuracy plus Determination are rolled
  against the monster's Avoidance (minus 4 while it falters), with the d20 crit die, a
  near-miss bonus of +2, and doubled damage on a crit. A miss adds +2 Determination (up to
  +4), and a hit resets it. A critical miss costs the hero's next turn.
- **An attack is the turn's action.** After moving, acting ends the movement step (D4).
- **Monsters** attack once and move once per round, in the monsters' phase, in either order.
  They pass through other monsters but not heroes, don't set off traps, and are revealed
  when they step into the heroes' view. A hero brought to 0 Body falls.
  - **Melee monsters never attack diagonally** (GM, 2026-10-05), with or without reach. They
    attack an orthogonally adjacent square.
  - **Reach** adds the square two away in a straight line, past whoever stands between, but not
    through a wall, a closed door or tall furniture.
  - Monsters marked `ranged` attack anything in attack sight.

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

### Furniture, searching and disarming (GM, 2026-10-05)
- **All furniture blocks movement:** nobody stands on a chest or a table. Tall pieces block
  sight too.
- **Searching for treasure:**
  - The hero must stand orthogonally beside the piece, with no wall between.
  - Searching a trapped piece that hasn't been disarmed sets the trap off. A disarmed trap
    stays quiet.
  - The GM's event line names any quest note on the piece, so the GM can read out the reward.
- **Searching for traps** never reveals the GM's own trigger markers.
- **Searching for secret doors** also opens blocked squares that hide a secret door.
- **Disarming:**
  - The player announces it.
  - A trap on the floor is disarmed by stepping onto its square from beside it. The step is
    part of the action and costs no movement.
  - A trap on furniture (a chest) is disarmed from beside the piece.
  - It needs the class's "disarm" exclusive (the Rogue), and the trap must be known.
  - The roll is Nimble Fingers: 1d8, failing only on a 1. A failure sets the trap off, under
    the hero for a floor trap.

### Locked doors and keys
- A locked door may name its key (an item). A hero carrying an item of that name (any case)
  unlocks and opens it for free while moving. Without the key only the GM can open it.

### Quest goals and objectives
- A quest has a **goal** the players see ("Slay the Witch Lord and escape"). It never says
  where anything is, so a goal behind a hidden door stays hidden.
- It may list **objectives**, and all of them must be met:
  - **kill** the named monsters, or all of them;
  - **collect** an item by name: any hero carries it, whether found in a chest or dropped by
    the main enemy (the GM hands it out until rewards are data);
  - **escape:** every hero still standing has left the board.
- A hero on an exit square leaves with `turn.exit`, which ends their turn.
- **A quest without objectives is completed once every monster is dead.**
- **Lost** when no hero is left standing, or when everyone has left before the quest was won.
- The GM can end a quest either way at any time.
- The map editor can't set goals, objectives or door keys yet. The editor keeps them through
  edits.

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
- **Melee monsters and diagonals** (2026-10-05): never, reach included; reach is a 2-square
  straight-line attack.
- **Ranged range** (2026-10-05): no limit, only line of sight, for heroes and monsters alike.
- **Searching with monsters revealed** (2026-10-05): not allowed, for treasure, traps and
  secret doors alike, as long as any monster is revealed on the board (see D9).
- **Treasure searches** (2026-10-05): once per piece of furniture per party, not once per
  hero per room (see D9).
