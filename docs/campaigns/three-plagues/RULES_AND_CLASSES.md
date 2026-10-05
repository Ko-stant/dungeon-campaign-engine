# The Three Plagues - House Rules and Hero Classes (draft)

**Last Updated**: 2026-10-05 05:20 EDT

Design notes for the campaign's custom rules. Nothing here is final. Values written as
**N** are still to be decided. Questions are collected under "Open questions" at the end.
The app records and reminds; it never enforces any of this (see `docs/IMPORTANT.md`).

## Design goals
- A much harder game than standard HeroQuest: more enemies, deadlier enemies, and death
  comes quickly for careless heroes.
- Heroes and monsters have much larger Body pools.
- Combat uses mixed dice (d4, d6, d8, d10, d12, d20) instead of combat dice: heroes roll to
  hit and to crit, and damage is a fixed number from their equipment (see "Combat").
- Every hero can finish off a badly hurt enemy, and abilities are worth using in every
  fight: holding back should never feel like the safe choice.
- Progress is steady: no run of unlucky rounds where nobody gets anywhere (the original
  rules could go five rounds without either side doing damage), but no fight is over in a
  single round either.
- Heroes get repeatable abilities limited by **cooldowns** (in turn cycles). The Cleric
  pays **mana** for most spells instead.

## Core rules
- **Turn sequence** (unchanged): Action -> Move -> End, or Move -> Action -> End, for heroes
  and monsters alike.
- **Turn cycle**: one full round (every hero and monster takes a turn). This is the
  tracker's "Round".
- **Cooldowns** count down at the start of each round. An ability with a 3-cycle cooldown
  used in round 4 has 2 cycles left in round 5, 1 in round 6, and is ready in round 7
  (ready round = round used + cooldown).
- **Cooldowns out of a fight** stop short of ready; see "Fights, cooldowns and mana".
- **Gold** belongs to the party, not to single heroes.
- **Potions**: drinking one is a free action, at any time, even during the monsters' turn
  (between their attacks). A healing potion restores 8 Body, a mana potion 6 mana.

## Combat
Agreed 2026-10-03, refined 2026-10-04. Numbers are still to be tuned (steps 2 and 3 under
"Combat roadmap"); `bun scripts/combat-odds.ts` prints exact odds for any set of values.

### Stats
- **Hero class**: Body, **hit dice** (a dice expression, different per class; a weapon may
  replace them, see "Equipment"), **Accuracy** (a flat bonus to the hit roll, any whole
  number), **crit range** on a d20, **base damage**, **base avoidance**, **defense dice**,
  **mitigation**, Mana (Cleric), Mind, Movement.
- **Monster**: Body, **Avoidance**, **hit dice**, **damage**, Mind, Movement. Monsters have
  no armor or mitigation: their toughness is Avoidance and Body.
- **Avoidance** (monsters) is always an even number: a sum of die sizes, from 4 (d4) to 20
  (d20, or d8+d12, ...), and may go past 20 later. Example: d4+d10 = 14.

### A hero attacks
1. Roll the class **hit dice** and the **d20 crit die** together; they are read separately.
2. **Hit** when hit dice + Accuracy **meets or beats** the monster's Avoidance.
3. **Crit** when the crit die lands in the class's crit range:
   - on a hit, damage is doubled (x2 for every class; no other multipliers);
   - on a miss, the crit adds a **near-miss bonus** (+2 to start, possibly per class later)
     to the hit total. If that reaches Avoidance, the attack hits for normal damage.
4. **Damage** is fixed: class base damage + weapon(s) + rare damage items. It applies in full.
5. **Critical miss**: every die (hit dice and crit die) shows 1. The attack misses and the
   hero loses their next turn.
6. **Special crit**: every die shows its maximum. Crit damage, plus a flourish the GM
   narrates (story, not a mechanical effect).

Both extremes need the crit d20 too, so they stay rare: 1 in 400 for a 1d20 hit die,
1 in 2,000 for 2d10, 1 in 4,320 for 3d6.

### A monster attacks
1. The monster rolls its **hit dice**, plus a separate **2d20 crit check**: a crit only on
   double 20s (1 in 400), doubling the damage. The monster crit may be dropped if it proves
   too strong.
2. The hero rolls their **defense dice** and adds their **base avoidance** (+ armor or staff
   avoidance): an opposed roll, as in original HeroQuest.
3. The monster hits only if its roll **beats** the hero's total; ties go to the hero.
4. Damage taken = monster damage (doubled on a crit) - the hero's **mitigation**. Mitigation
   can bring a hit to 0; the GM decides when heroes can get there and picks monsters to match
   (a few very weak monsters later can let the Barbarian feel how far they've come).

### Faltering
A monster that is badly hurt has Avoidance -4, so any hero can land the finishing blow.
Monsters only, never heroes. The threshold is to be set by simulation, somewhere between
1 Body left and 1/4 of the monster's maximum Body (fixed damage often skips a narrow window).

### Steady progress
- **Determination** (all heroes, passive; adopted 2026-10-04): each miss in a row adds +2
  Accuracy to that hero's next attack, up to +4. Any hit resets it. A token or die beside the
  hero card tracks it.
- **More hits per kill, at higher hit chances.** Average hero hit chance around 75% against
  ordinary monsters and 50-60% against elites (the Ranger higher), with monster Body raised
  so a fight still lasts several rounds. Scaling damage and Body up by the same factor
  changes nothing; what smooths the dice is needing more hits, each one more likely.
- With both, simulated against an Avoidance 14 elite with 20 Body, the slowest 10% of kills
  take 7-8 attacks instead of 8-17, and the Rogue no longer falls off against elites (see
  "Math cautions").
- Misses still happen, even to strong attackers, and that's part of the fun: the GM narrates
  them as blunders (the blade bites the door jamb, the club smashes the wall).

### Cannot defend
- A **hero** who cannot defend rolls no defense dice, but their base avoidance (and armor or
  staff avoidance) still counts.
- A **monster** that cannot defend is hit automatically: the hero rolls only the crit die, for
  double damage. Seasoned veterans don't miss a target that isn't moving.

### Abilities and the basic attack
- Abilities are usually better than a basic attack; the cooldown is the decision (when to
  spend it), not whether it's worth using.
- A hero with everything on cooldown must still matter: fixed damage, Determination and
  Faltering keep the basic attack worth taking.

### Class fantasy
Direction for the step 2 numbers.

| Class | Damage | Crit | Hit | Avoidance | Mitigation |
|---|---|---|---|---|---|
| Barbarian | high (hardest crits) | medium | medium | low | high (Barbarian-only heavy armor) |
| Rogue | low | very high | medium | high | low |
| Ranger | medium | medium | high | high | low |
| Cleric | low (Smite is their attack) | low | medium | low | low (the party protects them) |

### Equipment
- **Weapons** add damage, almost exclusively. A weapon may also **replace the class's hit
  dice**: a heavier weapon with more damage but bell-curve hit dice hits less often against
  high Avoidance. Simulations must show that a bigger weapon is not simply worse (or better).
- **Armor** mostly adds avoidance; rare pieces give mitigation. Barbarian armor leans toward
  mitigation, and some of it is Barbarian-only.
- Very rare **amulets, rings, bracers and gloves** may add damage.
- Loadouts: the Rogue always dual-wields (both weapons add damage; no main or off hand, and
  never one weapon alone), the Ranger uses a bow or crossbow, the Barbarian uses two-handed
  weapons, and the Cleric uses a one-handed weapon with a shield, a holy tome (off-hand), or a
  staff. Shields are Cleric-only; a staff adds some avoidance (it can block a swing).
- **Upgrade sizes** (an example, not fixed): a new sword or dagger for the Rogue adds 1-2
  damage over the one it replaces, and comes rarely; a new two-handed weapon for the
  Barbarian adds 2-4 in one go. Breakpoints from the simulations decide the real steps.
- No fixed list of gear slots. Items are described in general terms, so new finds can
  replace old ones or add something new (both are fun).
- Heroes keep old items when they get new ones. Inventory tracks what is equipped, used
  (removed), traded or deleted.

**Class base stats** (before gear; 2026-10-04, backfilled so the starting kit gives the step 2
totals):

| Class | Body | Hit dice + Accuracy | Crit | Damage | Avoidance | Defense dice | Mitigation | Mana |
|---|---|---|---|---|---|---|---|---|
| Barbarian | 40 | 1d20+3 | 17-20 | 3 | 2 | 1d6 | 1 (Tough as Nails) | - |
| Ranger | 30 | 2d10+4 | 18-20 | 2 | 4 | 1d6 | 0 | - |
| Rogue | 28 | 2d10+2 | 15-20 | 1 | 4 | 1d6 | 0 | - |
| Cleric | 28 | 2d8+4 | 20 | 2 | 3 | 1d6 | 0 | 16, regenerating 2 |

**Starting kit**:

| Class | Weapon | Chest | Class item |
|---|---|---|---|
| Barbarian | Greataxe (two-handed): +7 damage | Hide Cuirass (heavy, Barbarian-only): +1 mitigation | Iron-shod Boots: +1 avoidance |
| Rogue | Sword +3 damage and Dirk +2 damage | Leather Jerkin: +2 avoidance | Soft Boots: +1 avoidance |
| Ranger | Hunting Bow: +5 damage | Leather Armor: +2 avoidance | Archer's Gloves: +1 Accuracy |
| Cleric | Mace: +3 damage | Padded Robes: +1 avoidance | Holy Tome (off-hand): +1 mana regenerated each fight round |

With the kit: Barbarian 10 damage, avoidance 3, mitigation 2; Ranger 7 damage, Accuracy 5,
avoidance 6; Rogue 6 damage, avoidance 7; Cleric 5 damage, avoidance 4, 3 mana a round.
Later items must compete with or replace the holy tome's effect.

### Fights, cooldowns and mana
- A **fight** starts and ends when the GM says so.
- In a fight, cooldowns count down each round as before (ready round = round used +
  cooldown).
- **Out of a fight, cooldowns keep counting down but stop at 1 round left (short cooldowns,
  1-3 cycles) or 2 rounds left (long cooldowns, 4 or more); they only finish during a
  fight.** A cooldown already below its floor stays where it was (a long cooldown with 1
  round left at the end of a fight stays at 1). Stalling between fights can never fully reset
  an ability, a 10-cycle ability is about once per fight, and an ability left unused stays
  ready.
- **Mana** regenerates **N** at the start of each round of a fight, up to the maximum: enough
  to cast Smite every round. A heal costs about 2-3 rounds of regeneration, so Smiting now
  means fewer spells later. Out of a fight mana only comes back through Prayer, which has a
  cooldown like any ability, so healing up between fights leaves the Cleric strained for the
  next one.
- Keeping one harmless monster alive is not a fight: the GM ends the fight, or the monster
  flees or raises the alarm. There is no timer pushing the heroes along.

### Math cautions
- Bell-curve hit dice fall away against high Avoidance: against 15, 1d20 hits 30% of the time
  but 3d6 only 9%. Single big dice are the ones that can punch above their weight. The same
  effect lets weapons trade hit dice for damage (see "Equipment").
- Avoidance moves in steps of 2, and each step is a big swing for several hit dice. Ordinary
  monsters belong at Avoidance 6-12 and elites at 14. Bosses at 16 or more may be too high,
  depending on how fast hero Accuracy grows; watch it in the simulations. Accuracy (any
  whole number) is the hero-side fine tuning.
- Every class's hit dice + Accuracy (+ near-miss bonus) must reach the highest Avoidance in
  play, or that class can never hit it.
- A crit only doubles damage on a hit, so a very high crit chance pays less against high
  Avoidance. Base damage, not a bigger multiplier, keeps the Rogue level (simpler math at the
  table).
- Determination helps most where hit chances are lowest (bell curves against elites) and
  barely changes the Ranger, so it narrows the gap between classes on its own.

**Step 2 targets**: relative weights of average damage per attack when the heroes first enter
the dungeon, with the starting kit: Barbarian 6, Ranger 5, Rogue 4.5, Cleric 3. The
Barbarian is about twice as effective as the Cleric, the Ranger one step behind the
Barbarian and the Rogue half a step behind the Ranger. The weights may drift as gear is found
(or never found: desperation, bad luck or bad choices). Upgrades must keep
any one class from pulling ahead; well-placed early finds (accuracy or damage) in Quest 1's
easier rooms can lift a class that lags.

**Proposed step 2 attack numbers** (2026-10-04), with the starting kit; damage is class base
+ weapon(s), split when the starting items are written:

| Class | Hit dice | Crit range | Damage (crit) | Hit chance vs Avoidance 10 / 14 | Weight vs 10 / 14 |
|---|---|---|---|---|---|
| Barbarian | 1d20+3 | 17-20 | 10 (20) | 72% / 52% | 6.0 / 6.0 |
| Ranger | 2d10+5 | 18-20 | 7 (14) | 95% / 74% | 4.8 / 5.2 |
| Rogue | 2d10+2 | 15-20 | 6 (12), sword + dirk | 82% / 51% | 4.4 / 4.1 |
| Cleric | 2d8+4 | 20 | 5 (10) | 85% / 45% | 3.1 / 2.8 |

Weights are effective damage per attack over a long fight (Determination and Faltering at 1/4
included), scaled so the Barbarian is 6. Hit chances are for one attack without
Determination. `bun scripts/combat-odds.ts` prints these and attacks-to-kill tables.

Earlier candidates (simulated 2026-10-04; expected damage per attack against Avoidance 10 / 14):

| | Current sample | More accurate (with Determination proposal) |
|---|---|---|
| Barbarian | 1d20+1, crit 17-20, 8 damage: 5.9 / 4.0 | 1d20+3, crit 17-20, 8: 6.9 / 5.0 |
| Rogue | 2d10, crit 15-20, 5 damage: 4.4 / 2.1 | 2d10+2, crit 15-20, 5: 5.3 / 3.2 |
| Ranger | 2d10+4, crit 18-20, 5 damage: 5.2 / 3.8 | 2d10+5, crit 18-20, 5: 5.4 / 4.2 |
| Cleric | 2d8+2, crit 20, 4 damage: 2.9 / 1.0 | 2d8+4, crit 20, 4: 3.6 / 1.9 |

Attacks to kill 20 Body at Avoidance 14 (average, slowest 10%): current sample without
Determination: Barbarian 6.4 / 11, Rogue 10.1 / 17, Ranger 5.4 / 8, Cleric 19.6 / 30; more
accurate with Determination: 4.6 / 7, 5.5 / 8, 4.7 / 6, 8.5 / 11. Damage values get retuned
to the targets once the accuracy direction is agreed, and monster Body follows.

### Difficulty reference (original HeroQuest)
How hard the base monsters were, computed from the combat dice (skull 1/2; heroes block on
white shields 1/3, monsters on the black shield 1/6). Step 3 converts monsters to keep these
ratios: goblins die to any hero's hit.

| Monster | Atk/Def/Body | 2-dice hero: hit chance | Attacks to kill | vs a 2-defend-dice hero: hit chance / damage |
|---|---|---|---|---|
| Goblin | 2/1/1 | 67% | 1.5 | 44% / 0.56 |
| Orc | 3/2/1 | 59% | 1.7 | 62% / 0.96 |
| Skeleton | 2/2/1 | 59% | 1.7 | 44% / 0.56 |
| Zombie | 2/3/1 | 52% | 1.9 | 44% / 0.56 |
| Abomination | 3/3/2 | 52% | 3.3 | 62% / 0.96 |
| Mummy | 3/4/2 | 46% | 3.8 | 62% / 0.96 |
| Dread Warrior | 4/4/3 | 46% | 5.6 | 76% / 1.40 |
| Gargoyle | 4/5/3 | 40% | 6.4 | 76% / 1.40 |

A 3-dice hero (the original Barbarian) killed a goblin in 1.2 attacks and a gargoyle in 3.9;
a 1-die hero (the Wizard) needed 2.4 and 14.9. The original Barbarian (8 Body) lasted about
14 goblin attacks or 6 gargoyle attacks.

Monsters without original stats:
- **Goblin archer** and **goblin warlock**: as weak in defense as a goblin. The archer attacks
  from range for goblin damage; the warlock from range for a little more, and (proposed) its
  blast also hurts heroes beside the target, so holding a doorway isn't always safe.
- **Orc archer**: an orc's attack and defense, attacking from range.
- **Specter**: as hard as a Gargoyle.

### Simulator
`bun scripts/combat-sim.ts [runs] [--specter] [--melee=N] [--swap "Room 8=gargoyle,mummy"]`
plays the party through Quest 1 thousands of times, with abilities and attack-only (no
abilities or spells; potions and the pool still count). Numbers live in
`scripts/combat-config.ts` (shared with `combat-odds.ts`); encounters come from the board via
`bun scripts/quest-encounters.ts "Crumbling Halls" docs/campaigns/three-plagues/sim/crumbling-halls.json`
(rerun after moving monsters). `--swap` tries a room with other monsters without touching the
board.

What it models: every hero and monster attack under the rules above, Determination,
Faltering (1/4), the cooldown floor between fights, mana regeneration, potions (free, drunk
at a quarter of Body or less, even between monster attacks), the pool,
and the class abilities (see "Abilities (step 4)") with simple tactics: focus the weakest
monster, use an ability when its moment comes. What it doesn't: movement, positioning,
doors, traps. Instead:
- The heroes hold doorways: at most 2 melee monsters attack a round (ranged ones always can).
- Only 2 heroes attack in a fight's first round (the others are moving in after the door
  opens); after that every hero attacks (they rotate at the doorway).
- Rooms 19, 20, 4 and 5 come first; the rest come in a random order each run, since the
  party can go anywhere from there.

**Targets** (GM, 2026-10-04): about 75-80% of parties clear Quest 1 with nobody dead (a single
death can cascade quickly); a party that only attacks fails; a lone dread warrior against the
whole party is frightening but not extremely dangerous; a lone elite is a real threat to a
split or worn-down party; a wipe in Room 8 is acceptable.

**Current numbers: third calibration (2026-10-04, with the step 4 ability kit)**. Monsters:

| Monster | Body | Avoidance | Hit dice | Damage | Notes |
|---|---|---|---|---|---|
| Goblin | 5 | 6 | 1d12 | 4 | archer: the same at range |
| Goblin warlock | 5 | 6 | 1d12 | 5 | at range; blast: 2 damage to 2 heroes beside the target |
| Orc | 22 | 8 | 2d8 | 9 | archer: the same at range |
| Skeleton | 17 | 8 | 1d12 | 6 | undead |
| Zombie | 22 | 10 | 1d12+1 | 8 | undead |
| Abomination | 55 | 10 | 2d8+1 | 10 | |
| Mummy | 64 | 12 | 2d8+1 | 10 | undead |
| Dread warrior | 109 | 12 | 2d10+2 | 15 | |
| Gargoyle | 109 | 14 | 2d10+3 | 12 | strikes 2 squares in a straight line |
| Specter | 109 | 14 | 2d10+2 | 15 | undead |

Heroes as in the second calibration below, except the Cleric: 16 mana, Smite 5 damage (+2
against undead), heals of 12. Results (5,000 runs): 92% clear Quest 1,
**79% with nobody dead**, attack-only parties never do; with the Specter (the Stranger never
freed) 80% / 61%. Fresh-party fights: orc rooms about 2 rounds, Room 4 and the lone dread
warrior about 4 (11-15% of the party's Body), the lone gargoyle about 4 (30%), Room 8 about 5
(38%). Heroes use an ability on 52% (Barbarian), 36% (Ranger), 41% (Rogue: Venom Vial 34%) of
their turns; the Cleric heals on 23%, prays on 20% and turns evil on 18%.

**Second calibration (2026-10-04, provisional abilities)**, draft monsters (Body / Avoidance /
hit dice / damage):

| Monster | Body | Avoidance | Hit dice | Damage | Notes |
|---|---|---|---|---|---|
| Goblin | 5 | 6 | 1d12 | 4 | archer: the same at range |
| Goblin warlock | 5 | 6 | 1d12 | 5 | at range; proposed blast: 2 damage to 2 heroes beside the target |
| Orc | 15 | 8 | 2d8 | 9 | archer: the same at range |
| Zombie | 15 | 10 | 1d12+1 | 8 | |
| Abomination | 38 | 10 | 2d8+1 | 10 | |
| Mummy | 44 | 12 | 2d8+1 | 10 | |
| Dread warrior | 75 | 12 | 2d10+2 | 15 | |
| Gargoyle | 75 | 14 | 2d10+3 | 12 | proposed: strikes 2 squares in a straight line (a second hero defends separately) |
| Specter | 75 | 14 | 2d10+2 | 15 | |

Draft heroes: Barbarian 40 Body, base avoidance 3 + 1d6, mitigation 2 (with Tough as Nails);
Ranger 30, 6 + 1d6; Rogue 28, 7 + 1d6; Cleric 28, 4 + 1d6, 12 mana, 3 regenerated a round
(with the holy tome), Smite 2 mana (1d20+5, 6 damage), heal 6 mana for 12 Body. An orc hits
them 70% / 40% / 30% / 60% of the time.

Results (4,000 runs each, Specter left out; cleared / cleared with nobody dead; attack-only):

| Room 8 | With abilities | Attack-only |
|---|---|---|
| The old board: 2 dread warriors, gargoyle, 2 warlocks | 39% / 13% | 0% / 0% |
| Gargoyle, 2 warlocks, mummy | 95% / 75% | 2% / 0% |
| Gargoyle, 2 warlocks, abomination | 95% / 79% | 2% / 0% |
| Gargoyle, 2 warlocks, dread warrior | 70% / 34% | 0% / 0% |

These rows used the gargoyle without the line attack (2d10+2, 15 damage). With the line
attack the gargoyle needs less damage: at 15 it costs a fresh party 42% of its Body alone and
only 23% of parties clear Quest 1 with nobody dead. Damage 13 or more crosses a cliff (two
hits, plus a warlock blast, kill the 28-Body Rogue or Cleric): 12 gives 83%, 13 gives 54%. At
12 damage with 2d10+3 to hit:

| Room 8 (line gargoyle) | With abilities | Attack-only |
|---|---|---|
| Gargoyle, 2 warlocks | 93% / 78% | 1% / 0% |
| The old board (with 2 dread warriors) | 20% / 7% | 0% / 0% |

The GM moved Room 8 to a gargoyle and 2 warlocks on the board (2026-10-04; regenerated
encounters: 93% / 79%). With the line gargoyle Room 8 needs no mummy or dread warrior: a fresh party spends about 5
rounds and 44% of its Body there. The lone gargoyle (Room 11) becomes a real threat too (40% of
a fresh party's Body), while the lone dread warrior stays frightening but survivable (15%).
Watch the breakpoints: how many hits of a monster's damage kill each hero matters more than
the damage itself.

Without the line attack, with Room 8 as gargoyle, 2 warlocks and a mummy, the danger is spread: a fresh party beats the
lone dread warrior in about 4 rounds for 15% of its Body, and the lone gargoyle, Room 8 and the
mummy rooms are where worn-down parties die. Lightening Room 8 without raising the other
monsters made the quest trivial (100% with nobody dead, and attack-only cleared it 85-97% of
the time), so the danger now comes from attrition across the quest rather than one wall.
Fodder keeps the original feel (goblins die to any hit), while mid and elite monsters get
relatively more Body than the original ratios, so four heroes can't fell an elite in one round
with starter gear.

### Gear and upgrade tiers (2026-10-04)
- `bun scripts/combat-sim.ts --finds` picks the Quest 1 finds up as they're found;
  `--geared` starts with all of them. Items live in `scripts/combat-config.ts`
  (`STARTING_KIT`, `QUEST_1_FINDS`).
- Class balance: weights over a long fight against Avoidance 10 (Barbarian 6), starting kit
  6 / 4.9 / 4.2 / 3.0 (Barbarian / Ranger / Rogue / Cleric's basic attack); with every Quest 1
  find 6 / 4.8 / 4.2 / 2.6. The Barbarian (+2), Ranger (+1) and Rogue (+1) keep their
  proportions; the Cleric's basic attack slips, but Smite is their attack (a later item can
  add to Smite). A first try with Cardsharp's Gloves instead of the Wardens' Dirk let the
  Barbarian pull ahead (Rogue 4.2 -> 3.8).
- Quest 1 survival (nobody dead; prayer beads +2): 77% without finds, 84% picking them up as
  found; a party that skips some lands in between. **Since potions became a free action
  (2026-10-05)**, drunk between monster attacks, 10-Body potions lifted that to 84% / 88%;
  at 8 Body it is **72% without finds, 81% with finds** (89% geared, 70% with finds and the
  Specter; attack-only never clears it). There is a cliff between 8 and 9 Body (9: 81% /
  86%): from 9 up, a hero who drinks at a quarter of their Body survives one more elite hit.
- Upgrade tier: with every Quest 1 find the party clears Quest 1 at 93% with nobody dead;
  Quest 1's monsters would need about 30% more Body (x1.35: 75%) to challenge it the same
  way, and +1 damage on every monster drops it to 49-58%. The GM prefers not to lean on Body
  (the simulator can only capture so much of real play), so Quest 2 will use other levers.
- Fodder damage doesn't change survival: +1 or +2 damage for goblins, orcs, zombies and
  skeletons, or +1-2 for the abomination and mummy, all land within a point of the current
  numbers. Heroes act first and fodder dies to any hit, and what it does take off the party
  is healed between fights (mana regenerates each fight round), so it never reaches the
  elite fights where deaths happen. Fodder matters alongside an elite, or if healing between
  fights gets scarcer. The dread warrior is the one sensitive monster: at 16 damage (from 15)
  it kills the Barbarian in 3 hits instead of 4, and Quest 1 with finds drops to 80% (no
  finds 71%, geared 89%). Kept at 15 (GM, 2026-10-04).
- **Lone heroes against elites.** Elite Body is sized for all four heroes (about 4 rounds),
  so a hero alone almost never beats one: the Barbarian beats a full dread warrior (109 Body)
  under 2% of the time, striking first or second, where the original game's Barbarian beat a
  dread warrior (4 attack, 4 defend, 3 Body) 90% of the time striking first and 82% striking
  second. Splitting up against an elite is deadly, by design.
- **Trap G is the one scaled elite** (GM, 2026-10-04): its dread warrior has 27 Body per hero
  in the room (about one hero's share of 109), and it strikes first, since the heroes inside
  just spent their action searching or failing a disarm. Alone: the Barbarian survives 76%,
  the Rogue 44%, the Ranger 9%, the Cleric never; in pairs (54 Body) 80-99% win. Very likely
  a death when a hero opens those chests alone.

### Hallway monsters (2026-10-05)
Tried and removed: hallways stay mostly clear, apart from the few monsters guarding chests.
The GM added three hallway monsters to Quest 1: a skeleton at (10,14) between Room 4 and Room
20, a lone abomination at (18,1) and an orc at (13,7). Results (6,000 runs; cleared / nobody
dead):

| | No finds | Finds as found |
|---|---|---|
| Before | 90% / 78% | 94% / 85% |
| Each waits alone in its hallway | 90% / 80% | 94% / 86% |
| Each joins the nearest room fight (skeleton Room 4, abomination Room 11, orc Room 8) | 65% / 42% | 80% / 60% |
| Skeleton and orc join (Room 4, Room 8); the abomination waits | 86% / 70% | 92% / 81% |
| Only the abomination joins the gargoyle (Room 11) | 76% / 56% | 86% / 69% |
| Doubled (2 of each), each pair waits in its hallway | 89% / 78% | 93% / 86% |
| Doubled; skeletons join Room 4, orcs join Room 8, abominations wait | 82% / 67% | 89% / 79% |
| Doubled; all three pairs join their rooms | 51% / 29% | 70% / 49% |

- A lone hallway monster changes nothing: the heroes act first and gang up on it (skeleton
  about 1 round, orc 2, abomination 2.5; 0-4% of the party's Body). An ambush (the monster
  acts first) barely matters either (1-5%).
- Easy fights even help the party: mana regenerates and cooldowns finish only in fights, so a
  quick fight refills the Cleric for healing afterward (the "keep a weak monster alive" effect,
  here earned legitimately).
- Doubling them doesn't change that on its own: a fresh party beats 2 skeletons in about 2
  rounds (2% of its Body), 2 orcs in 2 (7%), 2 abominations in about 4 (14%), and heals up
  afterward. How many matters far less than whether they pile onto another fight.
- Hallway monsters bite when they join a room fight. The abomination joining the lone gargoyle
  turns Room 11 into half the party's Body (about 6 rounds); the skeleton and orc joining
  Room 4 and Room 8 land Quest 1 at 81% with finds.

### Combat roadmap
1. Attack and defense rules (this section) - done 2026-10-03.
2. Hero numbers: Body, hit dice, Accuracy, crit range, base damage, base avoidance, defense
   dice, mitigation, Mana and its regeneration, balanced around the starting kit; upgrade
   tiers simulated so no class pulls ahead.
3. Base monsters: Body, Avoidance, hit dice, damage for Quest 1's monsters (including goblin
   archers and warlocks, orc archers and the Specter), matched to the difficulty reference.
   The simulator (see "Simulator") checks the Quest 1 targets under "Campaign notes"; first
   calibration done 2026-10-04.
4. Abilities: done 2026-10-04. The merged kit is pruned, the Rogue has Venom Vial, and Quest 1
   is recalibrated to it (see "Abilities (step 4)" and "Simulator").
5. App support, in phases (GM, 2026-10-05: monster stats per campaign, one party purse with
   hero gold merged in, generic effects with countdowns):
   - 5a. Class combat stats - done 2026-10-05: hit dice, Accuracy, crit range, damage,
     defense dice, avoidance, mitigation and mana per fight round on the class form; sessions
     keep a frozen copy, shown on the hero cards.
   - 5b. Monster combat stats per campaign - done 2026-10-05: the campaign page's "Monster
     stats" section keeps a stat line per monster type (Body, Avoidance, hit dice, damage,
     ranged, reach, line, splash, undead); sessions freeze them into their monsters when set
     up or added (a quest's own Body still wins); the tracker shows them, with a Faltering
     badge at a quarter of Body.
   - 5c. Fights in the tracker - done 2026-10-05: "Start fight" / "End fight" by "Next
     round" (a "Fight" badge while one is on). In a fight each new round finishes cooldowns,
     regenerates mana (the class's mana per fight round) and counts effects down; out of a
     fight cooldowns stop at 1 round left (1-3) or 2 (4+) and nothing else changes. Effects
     are generic: any hero or monster can carry a named effect with an optional countdown and a
     note (`effect.add`/`effect.remove`); ending a fight ends those with a countdown.
   - 5d. Item stats, equipped items and hero totals; the party purse.
   - 5e. A script that fills the Three Plagues campaign with the agreed classes, monster stats
     and starting kits (run only when the GM asks).
   - 5f. Odds hints (advice only).

## Abilities (step 4)
Agreed 2026-10-04 from the GM's notes on the proposal (commit b39a157 has the full proposal
and notes). Overlapping abilities are merged rather than unlocked over time; once the
simulator runs the kit, abilities that add little, or that leave something ready every
turn (so cooldowns stop mattering), are pruned. Numbers are still to tune.

**Why** (simulator, provisional kit): without the Cleric's healing spell only 4% of parties
cleared Quest 1 with nobody dead (79% with it), and single martial abilities mattered little
(74-78% without any one). The kit aims to spread survival beyond the Cleric and to make each
ability strong in its moment.

### Common rules
- Using an active ability takes the hero's action unless it says "free". Reactions happen on
  the monsters' turn; passives are always on.
- An ability that attacks uses the normal attack (hit dice + Accuracy + Determination, crit
  die, near miss) unless it says **sure hit** (only the crit die is rolled). Any hit resets
  Determination.
- "Around" means the 8 squares touching the hero.
- Cooldown tiers, with fights lasting 2-5 rounds and the out-of-fight floor: **3** = once or
  twice a fight (ready from round 2 of the next fight); **5** = about once a fight (ready from
  round 3); **7 or more** = once a fight at most, not every fight.
- **Facing**: a monster faces the hero it last attacked (before its first attack, the way it
  last moved). A hero on the opposite side is **behind** it. The GM rules on close calls.

### Simulator results for the kit (2026-10-04)
`bun scripts/combat-sim.ts` runs the kit (`--without=`, `--with=`, `--mana=`, `--heal=` try
changes). With 20 mana and heals of 12 the party is too strong for the target: 93% clear
Quest 1 with nobody dead. With 12 mana and heals of 10 it is 78%, the baseline for this table.

| Change | Nobody dead | Change | Nobody dead |
|---|---|---|---|
| Baseline (12 mana, heal 10) | 78% | Without Prayer | 59% |
| Without Unleash Fury | 66% | Without Turn Evil | 72% |
| Without Aimed Shot | 73% | Without Vanish | 73% |
| Without Charge | 74% | Without Cleave | 75% |
| Without Exploit Opening | 76% | Without Multi-Shot | 76% |
| Without Challenge, Fan of Blades, Riposte or Rain of Arrows | 77% | Without Divine Blessing | 78% |
| Without Smite | 79% | Without Holy Blessing | 80% |
| With Second Wind | 83% | With Sanctuary / Circle of Mending | 77% / 76% |
| Every Barbarian, Rogue and Ranger ability off | 32% | | |

Reading it:
- Smite and Holy Blessing score at or above zero because they spend mana the Cleric later
  needs for heals (the simulated Cleric spends greedily; a player may save). Without Divine
  and Holy Blessing the party gets stronger (82% at 12 mana, 89% at 20 with heals of 10).
- The simulator can't value positioning, so Challenge (shielding the Cleric), Fan of Blades,
  Rain of Arrows, Exploit Opening and Riposte are likely worth more at the table than here;
  Pinning Shot (movement only) isn't modeled at all.
- Turns (12 mana, heal 10): the Barbarian uses an ability on about 56% of his turns (Charge
  whenever ready), the Ranger 41%, the Rogue 9% (plus Riposte and Vanish as reactions), and
  the Cleric casts on almost every turn, Prayer on about a quarter of them to refill mana.
- After pruning (below), with 16 mana: 87% with heals of 10, 95% with heals of 12, so the
  monsters were retuned (see "Current numbers: third calibration" under "Simulator").

### Pruned (2026-10-04)
- Barbarian **Second Wind**, Ranger **Pinning Shot**, Cleric **Holy Blessing**, **Divine
  Blessing**, **Sanctuary** and **Circle of Mending**. If the players struggle, the GM can hand
  out an item with one of these effects.
- Earlier: Shadowstep (too strong), Radiant Burst (dead weight when few undead are about);
  Poisoned Blades may come back as a special weapon.
- The Rogue gets one more active ability: **Venom Vial** (2026-10-04), from the player's
  character art (a vial from the pouch; Fan of Blades became Fan of Cards for the cards).

## Class exclusives
| Class     | Only this class can...                                             |
|-----------|--------------------------------------------------------------------|
| Barbarian | use two-handed weapons                                             |
| Cleric    | cast spells                                                        |
| Rogue     | disarm traps (no item gives this to another class)                 |
| Ranger    | use ranged weapons (bows, crossbows, ...)                          |

Planned with the combat rules (not yet in the app): only the Cleric uses shields, only the
Rogue dual-wields, and some heavy (mitigation) armor is Barbarian-only.

## Barbarian
| Ability | Type | Limit | Effect |
|---|---|---|---|
| Unburdened Charge | Active | Cooldown 3 | Charges 2-6 spaces (the player chooses) in one straight line; a diagonal step costs 2 spaces. The charge stops at the first enemy or wall. An enemy hit takes weapon damage + spaces charged. Traps on the path trigger, but the Barbarian takes no damage or penalty from them. |
| Unleash Fury | Active | Cooldown 6 | An echoing roar to the god of battle, then blind rage. The Barbarian attacks the nearest enemy, rolling the hit dice twice and keeping the better roll; allies within 2 squares get +2 Accuracy on their next attack. The rage lasts 3 rounds, or until the Barbarian kills an enemy (it ends at once): +3 damage on every hit and +2 mitigation, but each turn he must move to and attack an enemy if at all possible. (Echoing Roar is merged into it.) |
| Challenge | Free | Cooldown 4 | Monsters around the Barbarian must attack him on their next turn. |
| Tough as Nails | Passive | Always on | +1 mitigation (can bring a hit to 0). (Formerly "Shrug Off Pain".) |
| Cleave | Passive | Once a turn | When the Barbarian kills a monster, he may attack another monster around him at once. |

**Special item - Tempest-God Axe** (two-handed axe; Quest 2 reward): on a successful hit, air
collapses into the blade's wake and deals **N** extra damage to every enemy (not ally)
adjacent, orthogonally or diagonally, to the target. N is set during balancing.

## Cleric
Mana: maximum **16**, regenerating 2 each round of a fight, +1 with the holy tome (see
"Fights, cooldowns and mana"). Most spells have no cooldown.

| Spell | Cost | Limit | Effect |
|---|---|---|---|
| Smite | 2 mana | Repeatable | The Cleric's main attack, at range (line of sight): rolls to hit (1d20+5) and deals 5 damage, +2 against undead. |
| Heal Minor Wounds | 6 mana | Repeatable | Heals the target 10-12 Body (to tune; at least 10), up to their maximum. |
| Turn Evil | 8 mana | Cooldown 5 | The target enemy skips its next attack and cannot defend for 1 round: every hero's attack on it is a sure hit. |
| Divining | 3 mana | Repeatable | Reveals traps, hidden doors and treasure within 6 squares, through walls and doors. |
| Prayer | Free | Cooldown 5 | Restores half of maximum mana, rounded up. Takes the Cleric's action; they cannot defend until their next turn, then act normally. The only way to regain mana out of a fight. |

A later special item may add a little to Smite damage, so Smite stays worth casting in later
quests.

**Special item - Amulet of Duplicative Mind** (Quest 2 reward): the wearer may cast two
spells in one action phase, paying only the higher of the two costs (no movement between the
two casts).

## Rogue
| Ability | Type | Limit | Effect |
|---|---|---|---|
| Nimble Fingers | Passive | - | Disarms a trap on 1d8; fails only on a 1. |
| Exploit Opening | Passive | Always on | Crit range 13-20 (instead of 15-20) against a monster the Rogue is behind (see "Facing"). |
| Fan of Cards | Active | Cooldown 3 | Razor-edged cards hurled at every enemy around the Rogue: one attack against each, rolled separately. (Formerly Fan of Blades.) |
| Venom Vial | Active | Cooldown 4 | The Rogue cracks a vial from the pouch over their blades and strikes: a normal attack, and on a hit the target is also poisoned, losing 3 Body at the start of each of its next 3 turns (6 on a crit), no roll. A new vial restarts the count; it doesn't stack. A d4 beside the monster counts the turns down. |
| Vanish From Sight | Reaction | Cooldown 5 | When attacked, vanishes: the triggering attack is canceled, and no enemy can target the Rogue until the enemies' next turn cycle. |
| Riposte | Reaction | Cooldown 2 | When a monster's attack on the Rogue misses, the Rogue strikes back at once for half weapon damage (rounded down), no roll: they take advantage of the monster's misstep. |

**Special item - Quivering Boots** (Quest 1 reward): if the Rogue still has an action phase
this cycle, stepping on a trap does not trigger it. The Rogue immediately starts disarming it
instead; if the disarm succeeds, movement may continue. If it fails, the trap triggers and the
Rogue's movement ends there.

## Ranger
| Ability | Type | Limit | Effect |
|---|---|---|---|
| Multi-Shot | Active | Cooldown 5 | Shoots up to 3 different enemies in line of sight as one action. |
| Aimed Shot | Active | Cooldown 5 | A sure hit. The target is marked: every hero has +2 Accuracy against it until it dies or the fight ends. Movement is halved this round and next. (Hunter's Mark is merged into it.) |
| Rain of Arrows | Active | Cooldown 7 | Every enemy within 2 squares of a chosen square is attacked once, with half the Accuracy bonus (rounded down). Roll 1d4 arrows once; each enemy hit takes weapon damage x arrows (about 17 on average with the starting bow, 28 at most). |

**Special item - Aggamand's Quiver**: +1 Accuracy, permanently. A shot whose crit die shows
**14-17** is charged with elemental energy and deals 1.5x weapon damage, rounded up (not Rain
of Arrows). Nothing to count: the charge reads the crit die already rolled. With the Ranger's
crit range of 18-20, a shot is charged 20% of the time and a crit 15%.

## Campaign notes
**Items and treasure**
- Items are made to fit the balance: simulations set target damage and mitigation for the
  starting kit, the end of Quest 1, the end of Quest 2 and the Quest 3 vendor, then items
  (magical and rare ones too) are written to fill each tier. Special effects should be cool
  and unique without breaking the balance, and may be non-combat (like the Quivering Boots).
- Consumables (potions, scrolls) give meaningful choices without overshadowing combat and
  resource management. Healing and mana potions restore a flat amount. Players love saving the
  day with one, but they must not make resources trivial.
- The only vendor is in Quest 3. Quests 1 and 2 give gear and gold through exploration,
  combat and rewards. The vendor has a few fine pieces plus consumables, at high prices: the
  party can't afford everything (e.g. 2 of 4 very good items and some consumables).

**Quest 1 (Crumbling Halls)**, from the board notes (2026-10-04):
- Gold: 1,050-1,450 in all. Note G (two chests, 250 each; half goes missing if the Stranger
  warned the heroes first), K 350, P 400, and the Stranger's 200 (H, only if the heroes have
  already been through both trap rooms).
- Gear (approved 2026-10-05; each find replaces a starting item unless noted):

  | Note | Where | Item | Hero | Stats |
  |---|---|---|---|---|
  | W | Room 5, after the lone dread warrior (early) | Wardens' Longbow | Ranger | +6 damage (Hunting Bow +5) |
  | S | Room 7, beside Room 5 (early) | Wardens' Dirk | Rogue | +3 damage (Dirk +2) |
  | S | Room 7 | Wardens' Chain Shirt | Cleric | +2 avoidance (Padded Robes +1) |
  | O | Room 11, the lone gargoyle | Wardens' Greatsword | Barbarian | +9 damage (Greataxe +7) |
  | X | Room 17 | Quivering Boots | Rogue | +2 avoidance (Soft Boots +1), plus the trap effect |
  | T | the corridor by the east gate | Wardens' Scale Hauberk | Barbarian | +2 mitigation, heavy (Hide Cuirass +1) |
  | L | behind the hidden walls | Pilgrim's Prayer Beads | Cleric | +2 maximum mana, 18 in all (a trinket; left by Sister Wenna's pilgrims) |

  Early finds lift the weakest spots (the Ranger's and Rogue's damage, the Cleric's defense);
  the secret route earns the Cleric 2 more mana (note L on the board describes the beads).
  One gear chest may still swap with a gold chest (or the
  reverse) to lean less or more on upgrades. Cardsharp's Gloves (+1 Accuracy, Rogue) are kept
  for a later quest.
- Consumables: V, 3 healing potions (8 Body each) and 1 mana potion (6 mana); drinking is a
  free action. The pool (N) heals fully, once.
- Monsters: 5 orcs, 3 goblin archers, 3 mummies, 3 dread warriors (+1 from trap G), 2 orc
  archers, 2 goblin warlocks, 2 goblins, 2 gargoyles, 2 zombies, 1 abomination, and the
  Specter if the Stranger is never freed. Early rooms are easier; later ones mix easy and hard.
- Two major traps: the teleport trap (M) that searching can't find, which sends the first hero
  across the halls, splitting the party; and the two-chest room (G), which locks its door and
  spawns a dread warrior unless both chests are disarmed. The spawned dread warrior has 27
  Body per hero in the room and acts first; a hero alone in there will likely die (see "Gear
  and upgrade tiers").
- Targets: hard but not impossible. Avoiding those two traps, the party should clear it
  without too much trouble: about 75-80% survival. A party that never uses an ability and
  only attacks should fail. Goblins are mostly fodder; gargoyles and dread warriors are a
  force to be reckoned with and need abilities and teamwork.

**Party wipes**
- Quest 1: the heroes wake in town on the morning they planned to set out with Maren Ashby,
  unexplained, still carrying everything they had found. On the next attempt the chests that
  held that gear hold gold or consumables instead.
- Quest 2: the same, but back at the Crumbling Halls only one path is open (narrative only,
  no replay), with a stillness in the air they can't place; the road to Eastmarch is safe and
  they arrive the same day.
- Balance effect: a wiped party comes back stronger (it keeps its finds), a built-in catch-up.

## Open questions
1. **Faltering threshold**: between 1 Body and 1/4 of maximum Body (the simulator uses 1/4).
2. **Tempest-God Axe**: N extra damage, set during balancing.
3. **Later gear**: a Smite item for the Cleric; Quest 2 and 3 tiers (about +30% monster Body per
   tier of finds, to check per quest).

## Answered
- **Quest 1 finds** (2026-10-05): approved (see "Campaign notes"); Pilgrim's Prayer Beads +2
  mana (18 in all); trap G's dread warrior scales with the torches (27 Body per hero inside);
  the dread warrior stays at 15 damage.
- **Venom Vial** (2026-10-04): weapon damage plus poison (3 a turn for 3 turns); poison alone
  added nothing in fights that end in 3-4 rounds.
- **Quest 1 calibration** (2026-10-04): heals stay at 12; non-fodder monster Body x1.45 (third
  calibration).
- **Pruning** (2026-10-04): Second Wind, Pinning Shot, Holy Blessing, Divine Blessing,
  Sanctuary and Circle of Mending are cut; Cleric mana 16.
- **Abilities** (2026-10-04): merged, not unlocked over time (Echoing Roar into Unleash Fury,
  Hunter's Mark into Aimed Shot); Challenge, Cleave, Exploit Opening (when behind) and Riposte
  (half damage, no roll) join; Rain of Arrows uses 1d4 arrows; Turn Evil cooldown 5;
  Smite 5 damage, +2 against undead; Divining sees through walls and doors; near-miss bonus
  +2 for everyone.
- **Behind** (2026-10-04): a monster faces the hero it last attacked; behind is the opposite
  side, no ally needed.
- **Determination** (2026-10-04): adopted, +2 Accuracy per miss in a row, up to +4.
- **Step 2 targets** (2026-10-04): relative weights at dungeon entry, not exact values.
- **Orc archer** (2026-10-04): an orc's attack and defense, at range.
- **Cannot defend** (2026-10-04): a hero keeps base avoidance but rolls no defense dice; a
  monster is hit automatically (only the crit die is rolled).
- **Unleash Fury** (2026-10-04): the rage ends as soon as the Barbarian kills an enemy.
- **Unburdened Charge** (2026-10-04): the player chooses 2-6 spaces; the charge stops at the
  first enemy or wall; a diagonal step costs 2.
- **Echoing Roar** (2026-10-04): roll the hit dice twice and keep the better roll; no
  Accuracy bonus.
- **Aggamand's Quiver** (2026-10-04): reworked: a crit die of 14-17 charges the shot (1.5x,
  rounded up); no shot count to track.
- **Quivering Boots** (2026-10-04): a failed disarm triggers the trap and ends the Rogue's
  movement.
- **Tempest-God Axe** (2026-10-04): enemies only.
- **Vanish From Sight** (2026-10-04): the triggering attack is canceled.
- **Crit multiplier** (2026-10-04): x2 for every class.
- **Short and long cooldowns** (2026-10-04): 1-3 cycles is short; a cooldown already below
  its floor stays where it was.
- **Monster crit** (2026-10-04): doubles damage.
- **Faltering and Smite** (2026-10-04): both adopted. Smite rolls to hit, deals fixed damage
  and has no cooldown.
- **Special crit** (2026-10-04): a flourish the GM narrates, not a mechanical effect.
- **Dual wielding** (2026-10-04): both weapons add damage; the Rogue starts with a sword and a
  dirk and never fights with one weapon.
- **Prayer** (2026-10-04): restores half of maximum mana (rounded up), takes the action and
  leaves the Cleric unable to defend until their next turn.
- **Holy tome** (2026-10-04): an off-hand item, +1 mana regenerated each fight round.
- **Gold** (2026-10-04): shared by the party.
- **Tough as Nails** (2026-10-03): yes, mitigation can bring a hit to 0.
- **Accuracy** (2026-10-03): a flat bonus added to the class hit dice, rolled against a
  monster's Avoidance (see "Combat").
- **Cooldown counting** (2026-09-28): ready round = round used + cooldown (see Core rules).
- **Shrug Off Pain** (2026-09-28): always on; renamed **Tough as Nails**.
- **Multi-Shot and the Quiver** (2026-09-28): one draw = one shot; all three arrows are
  charged when the draw is the 3rd (superseded by the Quiver rework, 2026-10-04).
- **Rain of Arrows cooldown** (2026-09-28): long, 6-8 cycles; the half accuracy alone does
  not stop a lucky streak from clearing too many enemies too often.
