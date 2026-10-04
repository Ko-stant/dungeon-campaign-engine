# The Three Plagues - House Rules and Hero Classes (draft)

**Last Updated**: 2026-10-03 17:34 EDT

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

## Combat
Agreed 2026-10-03. Numbers are still to be tuned (steps 2 and 3 under "Combat roadmap");
`bun scripts/combat-odds.ts` prints exact odds for any set of values.

### Stats
- **Hero class**: Body, **hit dice** (a dice expression, different per class), **Accuracy**
  (a flat bonus to the hit roll, any whole number), **crit range** on a d20, **base damage**,
  **base avoidance**, **defense dice**, **mitigation**, Mana (Cleric), Mind, Movement.
- **Monster**: Body, **Avoidance**, **hit dice**, **damage**, Mind, Movement. Monsters have
  no armor or mitigation: their toughness is Avoidance and Body.
- **Avoidance** (monsters) is always an even number: a sum of die sizes, from 4 (d4) to 20
  (d20, or d8+d12, ...), and may go past 20 later. Example: d4+d10 = 14.

### A hero attacks
1. Roll the class **hit dice** and the **d20 crit die** together; they are read separately.
2. **Hit** when hit dice + Accuracy **meets or beats** the monster's Avoidance.
3. **Crit** when the crit die lands in the class's crit range:
   - on a hit, damage is multiplied (x2 to start; tuned per class in step 2);
   - on a miss, the crit adds a **near-miss bonus** (+2 to start) to the hit total. If that
     reaches Avoidance, the attack hits for normal damage.
4. **Damage** is fixed: class base damage + weapon(s) + rare damage items. It applies in full.
5. **Critical miss**: every die (hit dice and crit die) shows 1. The attack misses and the
   hero loses their next turn.
6. **Special crit**: every die shows its maximum. Crit damage plus a flourish (stun, extra
   damage, ...; per class or the GM's call).

Both extremes need the crit d20 too, so they stay rare: 1 in 400 for a 1d20 hit die,
1 in 2,000 for 2d10, 1 in 4,320 for 3d6.

### A monster attacks
1. The monster rolls its **hit dice**, plus a separate **2d20 crit check**: a crit only on
   double 20s (1 in 400). The monster crit may be dropped if it proves too strong.
2. The hero rolls their **defense dice** and adds their **base avoidance** (+ armor or staff
   avoidance): an opposed roll, as in original HeroQuest.
3. The monster hits only if its roll **beats** the hero's total; ties go to the hero.
4. Damage taken = monster damage (doubled on a crit) - the hero's **mitigation**. Mitigation
   can bring a hit to 0; the GM decides when heroes can get there and picks monsters to match.

### Class fantasy
Direction for the step 2 numbers.

| Class | Damage | Crit | Hit | Avoidance | Mitigation |
|---|---|---|---|---|---|
| Barbarian | high | medium | medium | low | high (Barbarian-only heavy armor) |
| Rogue | low | very high | medium | high | low |
| Ranger | medium | medium | high | high | low |
| Cleric | low | low | medium | low | low (the party protects them) |

### Equipment
- **Weapons** add damage, almost exclusively.
- **Armor** mostly adds avoidance; rare pieces give mitigation. Barbarian armor leans toward
  mitigation, and some of it is Barbarian-only.
- Very rare **amulets, rings, bracers and gloves** may add damage.
- Loadouts: the Rogue dual-wields, the Ranger uses a bow or crossbow, the Barbarian uses
  two-handed weapons, and the Cleric uses a one-handed weapon and a shield, or a staff.
  Shields are Cleric-only; a staff adds some avoidance (it can block a swing).

### Fights, cooldowns and mana
- A **fight** starts and ends when the GM says so.
- In a fight, cooldowns count down each round as before (ready round = round used +
  cooldown).
- **Out of a fight, cooldowns keep counting down but stop at 1 round left (short cooldowns)
  or 2 rounds left (long cooldowns); they only finish during a fight.** Stalling between
  fights can never fully reset an ability, a 10-cycle ability is about once per fight, and an
  ability left unused stays ready. Proposed: a cooldown of 3 or less is short.
- **Mana** regenerates **N** at the start of each round of a fight, up to the maximum. Out of
  a fight it only comes back through Prayer, which has a cooldown like any ability, so
  healing up between fights leaves the Cleric strained for the next one.
- Keeping one harmless monster alive is not a fight: the GM ends the fight, or the monster
  flees or raises the alarm. There is no timer pushing the heroes along.

### Proposed (not yet agreed)
- **Faltering**: a monster at 1/4 of its maximum Body or less has Avoidance -4, so any hero
  can land the finishing blow.
- **Smite** (Cleric spell, cheap): the Cleric's real attack. With low damage and a low crit
  chance, an ordinary Cleric attack repeats the Wizard problem (never worth the action).
- Abilities should be situational, not strictly better than a basic attack; choosing between
  them is where the thinking comes from (step 4).

### Math cautions
- Bell-curve hit dice fall away against high Avoidance: against 15, 1d20 hits 30% of the time
  but 3d6 only 9%. Single big dice are the ones that can punch above their weight.
- Avoidance moves in steps of 2, and each step is a big swing for several hit dice. Ordinary
  monsters belong at Avoidance 6-12, elites at 14 and bosses at 16 or more. Accuracy (any
  whole number) is the hero-side fine tuning.
- Every class's hit dice + Accuracy (+ near-miss bonus) must reach the highest Avoidance in
  play, or that class can never hit it.
- A crit only multiplies damage on a hit, so a very high crit chance pays less against high
  Avoidance. With the sample numbers below the Rogue falls behind (expect a x3 Rogue crit or
  more base damage).

Sample starting point for step 2 (not decided): Barbarian 1d20+1, crit 18-20, 8 damage;
Rogue 2d10, crit 13-20, 4 damage; Ranger 2d10+4, crit 18-20, 5 damage; Cleric 2d8+2, crit
20, 3 damage. Against Avoidance 10 they hit 62% / 70% / 91% / 68% of the time for 5.6 / 3.8
/ 5.2 / 2.1 expected damage per attack.

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

### Combat roadmap
1. Attack and defense rules (this section) - done 2026-10-03.
2. Hero numbers: Body, hit dice, Accuracy, crit range and multiplier, base damage, base
   avoidance, defense dice, mitigation, Mana and its regeneration; starting gear per loadout.
3. Base monsters: Body, Avoidance, hit dice, damage for Quest 1's monsters and the Specter,
   matched to the difficulty reference.
4. Abilities: restate them in these terms (Echoing Roar, Aimed Shot, Holy Blessing, Turn
   Evil, ...), set cooldowns with the out-of-fight rule, give Prayer a cooldown, add Smite.
5. App support: the new class, monster and item stats; fight start/end in the tracker with
   the cooldown floor and mana regeneration; odds hints (advice only).

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
| Echoing Roar | Active | Once every 5 cycles | Calls to the god of battle: higher chance to hit on the next attack. May attack immediately this cycle. |
| Unburdened Charge | Active | Once every 3 cycles | Charges 2-6 spaces in one straight line. An enemy hit takes weapon damage + spaces charged. Traps on the path trigger, but the Barbarian takes no damage or penalty from them. |
| Unleash Fury | Active | Cooldown TBD | Blind rage: immediately attacks the nearest enemy. Lasts 3 cycles. While raging, the Barbarian must move to and attack an enemy each turn if at all possible. Killing an enemy sates the rage. |
| Tough as Nails | Passive | Always on | Takes 1 less damage from every attack (adds 1 mitigation; can bring a hit to 0). (Formerly "Shrug Off Pain".) |

**Special item - Tempest-God Axe** (two-handed axe): on a successful hit, air collapses
into the blade's wake and deals **N** extra damage to every enemy adjacent (orthogonally or
diagonally) to the target.

## Cleric
Spells cost mana (**N** each; starting mana **N**) and most have no cooldown. Mana
regenerates **N** each round of a fight (see "Fights, cooldowns and mana").

| Spell | Cost | Limit | Effect |
|---|---|---|---|
| Heal Minor Wounds | N mana | Repeatable | Heals the target **N** Body, up to their maximum. |
| Holy Blessing | N mana | Repeatable | Target adds **N** to attack and defense rolls for 1 cycle. |
| Turn Evil | N mana | Repeatable | Target enemy skips its next attack and cannot defend for 1 cycle. |
| Divine Blessing | N mana | Once every 10 cycles | Ends every cooldown of one other hero (not the Cleric). |
| Divining | N mana | Repeatable | Reveals traps, hidden doors and treasure within **N** squares, through walls. |
| Prayer | Free | Cooldown N (counts down out of a fight only to 1-2 rounds left) | Restores **N** mana. Costs the next action phase (movement still allowed); cannot defend until the next cycle. The only way to regain mana out of a fight. |

**Special item - Amulet of Duplicative Mind**: the wearer may cast two spells in one action
phase (no movement between the two casts).

## Rogue
| Ability | Type | Limit | Effect |
|---|---|---|---|
| Nimble Fingers | Passive | - | Disarms a trap on 1d8; fails only on a 1. |
| Vanish From Sight | Reaction | Once every 5 cycles | When attacked, vanishes: no enemy can target the Rogue until the enemies' next turn cycle. |
| Fan of Blades | Active | Once every 3 cycles | Weapon damage to every enemy around the Rogue. |

**Special item - Quivering Boots**: if the Rogue still has an action phase this cycle,
stepping on a trap does not trigger it. The Rogue immediately starts disarming it instead;
if the disarm succeeds, movement may continue.

## Ranger
| Ability | Type | Limit | Effect |
|---|---|---|---|
| Multi-Shot | Active | Cooldown TBD | Shoots up to 3 enemies in range as one action. |
| Aimed Shot | Active | Cooldown TBD | Movement is halved this cycle and next; the shot cannot miss. |
| Rain of Arrows | Active | Once every 6-8 cycles (TBD) | Looses 1d4+1 arrows at every enemy within 2 squares of a chosen square. Roll to hit each enemy once at half accuracy. Each enemy hit takes weapon damage x arrows (rolling 4 = 5 arrows = 5x weapon damage). |

**Special item - Aggamand's Quiver**: +1 Accuracy, permanently. Every 3rd shot is charged
with elemental energy and deals 1.5x weapon damage (not Rain of Arrows). The charge comes from drawing from the
quiver: a Multi-Shot is one draw and counts as one shot, and if that draw is the 3rd, all
three arrows are charged.

## Open questions
1. **Unleash Fury**: does killing an enemy end the rage early, or only count as the attack
   for that turn? What is its cooldown?
2. **Unburdened Charge**: does the player choose 2-6 spaces, or roll for it? Does the charge
   stop at the first enemy?
3. **Echoing Roar**: how much more likely to hit (+N Accuracy, roll the hit dice twice, ...)?
4. **Aggamand's Quiver**: does the shot count carry over between quests? How is 1.5x
   rounded?
5. **Multi-Shot / Aimed Shot**: cooldowns still TBD. Rain of Arrows: 6, 7 or 8?
6. **Quivering Boots**: what happens when the disarm fails - the trap triggers, and the
   Rogue stops moving?
7. **Tempest-God Axe**: does the extra damage hit adjacent allies too, or only enemies?
8. **Vanish From Sight**: does it cancel the attack that triggered it?
9. **Near-miss bonus**: +2 for everyone, or per class?
10. **Crit multipliers**: x2 for everyone, or per class (x3 for the Rogue)? How are
    fractions rounded?
11. **Short and long cooldowns**: is 3 or less short (stops at 1 round left out of a fight)?
12. **Mana**: how much regenerates each round of a fight; starting and maximum mana.
13. **Monster crit**: double damage, or something else? Keep it at all?
14. **Faltering** and **Smite** (see "Proposed"): adopt them?
15. **Special crit**: the flourish for each class.
16. **Dual wielding**: do both of the Rogue's weapons add to damage?

## Answered
- **Tough as Nails** (2026-10-03): yes, mitigation can bring a hit to 0.
- **Accuracy** (2026-10-03): a flat bonus added to the class hit dice, rolled against a
  monster's Avoidance (see "Combat").
- **Cooldown counting** (2026-09-28): ready round = round used + cooldown (see Core rules).
- **Shrug Off Pain** (2026-09-28): always on; renamed **Tough as Nails**.
- **Multi-Shot and the Quiver** (2026-09-28): one draw = one shot; all three arrows are
  charged when the draw is the 3rd.
- **Rain of Arrows cooldown** (2026-09-28): long, 6-8 cycles; the half accuracy alone does
  not stop a lucky streak from clearing too many enemies too often.
