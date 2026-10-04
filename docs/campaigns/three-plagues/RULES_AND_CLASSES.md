# The Three Plagues - House Rules and Hero Classes (draft)

**Last Updated**: 2026-10-04 18:20 EDT

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

**Starting kit** (stats backfilled during step 2; class base stats go down to match):

| Class | Weapon | Chest | Class item |
|---|---|---|---|
| Barbarian | two-handed weapon | armor | boots |
| Rogue | sword and dirk | armor | boots |
| Ranger | bow | armor | gloves |
| Cleric | one-handed weapon | armor | holy tome (off-hand): +1 mana regenerated each fight round |

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
Faltering (1/4), the cooldown floor between fights, mana regeneration, Prayer, potions, the
pool, and simple tactics (focus the weakest monster; provisional abilities until step 4).
What it doesn't: movement, positioning, doors, traps, Turn Evil, Holy Blessing, Divine
Blessing, Unleash Fury. Instead:
- The heroes hold doorways: at most 2 melee monsters attack a round (ranged ones always can).
- Only 2 heroes attack in a fight's first round (the others are moving in after the door
  opens); after that every hero attacks (they rotate at the doorway).
- Rooms 19, 20, 4 and 5 come first; the rest come in a random order each run, since the
  party can go anywhere from there.

**Targets** (GM, 2026-10-04): about 75-80% of parties clear Quest 1 with nobody dead (a single
death can cascade quickly); a party that only attacks fails; a lone dread warrior against the
whole party is frightening but not extremely dangerous; a lone elite is a real threat to a
split or worn-down party; a wipe in Room 8 is acceptable.

**Second calibration (2026-10-04)**, draft monsters (Body / Avoidance / hit dice / damage):

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

### Combat roadmap
1. Attack and defense rules (this section) - done 2026-10-03.
2. Hero numbers: Body, hit dice, Accuracy, crit range, base damage, base avoidance, defense
   dice, mitigation, Mana and its regeneration, balanced around the starting kit; upgrade
   tiers simulated so no class pulls ahead.
3. Base monsters: Body, Avoidance, hit dice, damage for Quest 1's monsters (including goblin
   archers and warlocks, orc archers and the Specter), matched to the difficulty reference.
   The simulator (see "Simulator") checks the Quest 1 targets under "Campaign notes"; first
   calibration done 2026-10-04.
4. Abilities: restate them in these terms (Echoing Roar, Aimed Shot, Holy Blessing, Turn
   Evil, ...), set cooldowns with the out-of-fight rule, give Prayer a cooldown, add Smite.
   Also brainstorm alternative abilities for every class, in case the first ideas aren't the
   best.
5. App support: the new class, monster and item stats; items marked equipped; party gold;
   fight start/end in the tracker with the cooldown floor and mana regeneration; odds hints
   (advice only). Once balanced, the app's items and monsters are generated from the agreed
   numbers rather than entered by hand.

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
| Echoing Roar | Active | Once every 5 cycles | Calls to the god of battle: on the next attack, roll the hit dice twice and keep the better roll (no Accuracy bonus). May attack immediately this cycle. |
| Unburdened Charge | Active | Once every 3 cycles | Charges 2-6 spaces (the player chooses) in one straight line; a diagonal step costs 2 spaces. The charge stops at the first enemy or wall. An enemy hit takes weapon damage + spaces charged. Traps on the path trigger, but the Barbarian takes no damage or penalty from them. |
| Unleash Fury | Active | Cooldown TBD | Blind rage: immediately attacks the nearest enemy. Lasts 3 cycles, or until the Barbarian kills an enemy (the rage ends at once). While raging, the Barbarian must move to and attack an enemy each turn if at all possible. Combat benefit TBD (mitigation, damage, hit chance, or a mix). |
| Tough as Nails | Passive | Always on | Takes 1 less damage from every attack (adds 1 mitigation; can bring a hit to 0). (Formerly "Shrug Off Pain".) |

**Special item - Tempest-God Axe** (two-handed axe; Quest 2 reward): on a successful hit, air
collapses into the blade's wake and deals **N** extra damage to every enemy (not ally)
adjacent, orthogonally or diagonally, to the target. N is set during balancing.

## Cleric
Spells cost mana (**N** each; starting mana **N**) and most have no cooldown. Mana
regenerates **N** each round of a fight, +1 with the holy tome (see "Fights, cooldowns and
mana").

| Spell | Cost | Limit | Effect |
|---|---|---|---|
| Smite | N mana (about one round's regeneration) | Repeatable | The Cleric's main attack, also against high-Avoidance monsters: rolls to hit and deals fixed damage. |
| Heal Minor Wounds | N mana (about 2-3 rounds' regeneration) | Repeatable | Heals the target **N** Body, up to their maximum. |
| Holy Blessing | N mana | Repeatable | Target adds **N** to attack and defense rolls for 1 cycle. |
| Turn Evil | N mana | Repeatable | Target enemy skips its next attack and cannot defend for 1 cycle: every hero attack on it hits, rolling only the crit die. |
| Divine Blessing | N mana | Once every 10 cycles | Ends every cooldown of one other hero (not the Cleric). |
| Divining | N mana | Repeatable | Reveals traps, hidden doors and treasure within **N** squares, through walls. |
| Prayer | Free | Cooldown N (4-5 cycles) | Restores half of maximum mana, rounded up. Takes the Cleric's action; they cannot defend until their next turn, then act normally. The only way to regain mana out of a fight. |

A later special item may add a little to Smite damage, so Smite stays worth casting in later
quests.

**Special item - Amulet of Duplicative Mind** (Quest 2 reward): the wearer may cast two
spells in one action phase, paying only the higher of the two costs (no movement between the
two casts).

## Rogue
| Ability | Type | Limit | Effect |
|---|---|---|---|
| Nimble Fingers | Passive | - | Disarms a trap on 1d8; fails only on a 1. |
| Vanish From Sight | Reaction | Once every 5 cycles | When attacked, vanishes: the triggering attack is canceled, and no enemy can target the Rogue until the enemies' next turn cycle. |
| Fan of Blades | Active | Once every 3 cycles | Weapon damage to every enemy around the Rogue. |

**Special item - Quivering Boots** (Quest 1 reward): if the Rogue still has an action phase
this cycle, stepping on a trap does not trigger it. The Rogue immediately starts disarming it
instead; if the disarm succeeds, movement may continue. If it fails, the trap triggers and the
Rogue's movement ends there.

## Ranger
| Ability | Type | Limit | Effect |
|---|---|---|---|
| Multi-Shot | Active | Cooldown N (4-6, same as Aimed Shot) | Shoots up to 3 enemies in range as one action. |
| Aimed Shot | Active | Cooldown N (4-6, same as Multi-Shot) | Movement is halved this cycle and next; the shot cannot miss. |
| Rain of Arrows | Active | Once every 6-8 cycles (TBD) | Looses 1d4+1 arrows at every enemy within 2 squares of a chosen square. Roll to hit each enemy once at half accuracy. Each enemy hit takes weapon damage x arrows (rolling 4 = 5 arrows = 5x weapon damage). |

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
- Gear: 7 pieces still to choose: L, O, T (weapon or armor), S (two chests, weapon or armor
  each), W (weapon), X (armor). The Quivering Boots take one of these. One gear chest may
  swap with a gold chest (or the reverse) to lean less or more on upgrades.
- Consumables: V, 3 healing potions and 1 mana potion. The pool (N) heals fully, once.
- Monsters: 5 orcs, 3 goblin archers, 3 mummies, 3 dread warriors (+1 from trap G), 2 orc
  archers, 2 goblin warlocks, 2 goblins, 2 gargoyles, 2 zombies, 1 abomination, and the
  Specter if the Stranger is never freed. Early rooms are easier; later ones mix easy and hard.
- Two major traps: the teleport trap (M) that searching can't find, which sends the first hero
  across the halls, splitting the party; and the two-chest room (G), which locks its door and
  spawns a dread warrior unless both chests are disarmed. A hero alone in there may die.
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
1. **Unleash Fury**: what is its combat benefit (mitigation, damage, hit chance, a mix), and
   its cooldown?
2. **Multi-Shot / Aimed Shot**: the shared cooldown (4-6). Rain of Arrows: 6, 7 or 8? What
   does "half accuracy" mean with hit dice (half the Accuracy bonus, or half the total)?
3. **Near-miss bonus**: +2 for everyone, or per class?
4. **Mana**: regeneration per fight round, Smite and heal costs, starting and maximum mana;
   Prayer's cooldown (4 or 5).
5. **Faltering threshold**: between 1 Body and 1/4 of maximum Body; set by simulation.
6. **Turn Evil**: with every attack on the target hitting, its mana cost must reflect a
   whole party's sure hits (step 4).
7. **Tempest-God Axe**: N extra damage, set during balancing.

## Answered
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
