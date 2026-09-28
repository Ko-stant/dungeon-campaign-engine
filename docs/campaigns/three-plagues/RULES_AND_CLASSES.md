# The Three Plagues - House Rules and Hero Classes (draft)

**Last Updated**: 2026-09-28 17:53 EDT

Design notes for the campaign's custom rules. Nothing here is final. Values written as
**N** are still to be decided. Questions are collected under "Open questions" at the end.
The app records and reminds; it never enforces any of this (see `docs/IMPORTANT.md`).

## Design goals
- A much harder game than standard HeroQuest: more enemies, deadlier enemies, and death
  comes quickly for careless heroes.
- Heroes and monsters have much larger Body pools.
- Damage and defense use mixed dice (d4, d6, d8, d10, d12, d20) instead of combat dice.
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
- **Base stats** (planned changes, values TBD): Body (larger pools), Mind, Movement,
  attack dice, defense dice, **Accuracy** (to-hit; used by the Ranger's abilities and the
  Quiver), and **Mana** (Cleric only, for now).
- **Weapon damage** is a dice expression per weapon (e.g. `1d8`, `2d6+1`).

## Class exclusives
| Class     | Only this class can...                                             |
|-----------|--------------------------------------------------------------------|
| Barbarian | use two-handed weapons                                             |
| Cleric    | cast spells                                                        |
| Rogue     | disarm traps (no item gives this to another class)                 |
| Ranger    | use ranged weapons (bows, crossbows, ...)                          |

## Barbarian
| Ability | Type | Limit | Effect |
|---|---|---|---|
| Echoing Roar | Active | Once every 5 cycles | Calls to the god of battle: higher chance to hit on the next attack. May attack immediately this cycle. |
| Unburdened Charge | Active | Once every 3 cycles | Charges 2-6 spaces in one straight line. An enemy hit takes weapon damage + spaces charged. Traps on the path trigger, but the Barbarian takes no damage or penalty from them. |
| Unleash Fury | Active | Cooldown TBD | Blind rage: immediately attacks the nearest enemy. Lasts 3 cycles. While raging, the Barbarian must move to and attack an enemy each turn if at all possible. Killing an enemy sates the rage. |
| Tough as Nails | Passive | Always on | Takes 1 less damage from every attack. (Formerly "Shrug Off Pain".) |

**Special item - Tempest-God Axe** (two-handed axe): on a successful hit, air collapses
into the blade's wake and deals **N** extra damage to every enemy adjacent (orthogonally or
diagonally) to the target.

## Cleric
Spells cost mana (**N** each; starting mana **N**) and most have no cooldown.

| Spell | Cost | Limit | Effect |
|---|---|---|---|
| Heal Minor Wounds | N mana | Repeatable | Heals the target **N** Body, up to their maximum. |
| Holy Blessing | N mana | Repeatable | Target adds **N** to attack and defense rolls for 1 cycle. |
| Turn Evil | N mana | Repeatable | Target enemy skips its next attack and cannot defend for 1 cycle. |
| Divine Blessing | N mana | Once every 10 cycles | Ends every cooldown of one other hero (not the Cleric). |
| Divining | N mana | Repeatable | Reveals traps, hidden doors and treasure within **N** squares, through walls. |
| Prayer | Free | Repeatable | Restores **N** mana. Costs the next action phase (movement still allowed); cannot defend until the next cycle. |

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
1. **Tough as Nails**: can it reduce a hit to 0 damage?
2. **Unleash Fury**: does killing an enemy end the rage early, or only count as the attack
   for that turn? What is its cooldown?
3. **Unburdened Charge**: does the player choose 2-6 spaces, or roll for it? Does the charge
   stop at the first enemy?
4. **Echoing Roar**: how much more likely to hit (+N to the roll, roll twice, ...)?
5. **Aggamand's Quiver**: does the shot count carry over between quests? How is 1.5x
   rounded?
6. **Multi-Shot / Aimed Shot**: cooldowns still TBD. Rain of Arrows: 6, 7 or 8?
7. **Quivering Boots**: what happens when the disarm fails - the trap triggers, and the
   Rogue stops moving?
8. **Tempest-God Axe**: does the extra damage hit adjacent allies too, or only enemies?
9. **Vanish From Sight**: does it cancel the attack that triggered it?
10. **Accuracy**: what does it roll against (a die, a monster's defense)?

## Answered
- **Cooldown counting** (2026-09-28): ready round = round used + cooldown (see Core rules).
- **Shrug Off Pain** (2026-09-28): always on; renamed **Tough as Nails**.
- **Multi-Shot and the Quiver** (2026-09-28): one draw = one shot; all three arrows are
  charged when the draw is the 3rd.
- **Rain of Arrows cooldown** (2026-09-28): long, 6-8 cycles; the half accuracy alone does
  not stop a lucky streak from clearing too many enemies too often.
