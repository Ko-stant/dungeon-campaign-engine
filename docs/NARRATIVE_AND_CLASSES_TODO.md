# Campaign Narrative, Custom Classes and Ability Tracking - TODO

**Last Updated**: 2026-09-28 19:33 EDT
**Branch**: `dce-table-only`

Goal: run the "Three Plagues" campaign from the app. Story bible (world, cast, secrets):
`docs/campaigns/three-plagues/NARRATIVE.md`; read-aloud passages with ids, speakers and
voice notes: `docs/campaigns/three-plagues/SCRIPT.md`. House rules and classes:
`docs/campaigns/three-plagues/RULES_AND_CLASSES.md`.

The app records and reminds; it never blocks the GM. Cooldowns, mana, class exclusives and
effects are tracked and shown, and the GM can override any of them.

In the app a "turn cycle" is a round (`State.Round`, advanced by `round.advance`).

## 1. Dice expressions - done 2026-09-28
- [x] Parse and validate expressions with d4, d6, d8, d10, d12, d20: `1d8`, `2d6+1`, `d20`,
      `1d8+1d4`, `1d8-1d4`, plain numbers. Limits: 20 dice per term, 50 per expression,
      modifier -99..99.
- [x] Canonical formatting ("2 D6 + 1" -> "2d6+1"; constants summed, modifier last),
      min/max, and a roller that takes the die function (for tests and a later UI).
- [ ] On-screen roller in the tracker (physical dice stay the default).

How: `internal/dice` (Go: `Parse`, `Expr.String/Min/Max/Roll`, text marshalling) and
`internal/web/src/dice/dice.ts` (TS: `parseDice`, `formatDice`, `rollDice`) share the same
rules and error wording; both were written test-first (`dice_test.go`, `dice.test.ts`).

## 2. Custom hero classes - done 2026-09-28
- [x] Migration `00004_custom_hero_class.sql`: `custom_hero_class (id, name, doc jsonb, ...)`.
- [x] Doc: color, description; body, mind, accuracy, mana (0 = none); attack dice, defend
      dice, movement (dice expressions, stored canonical); class exclusives (tags:
      two-handed, ranged, spells, disarm).
- [x] Abilities: `{id, name, kind (active|passive|reaction|spell), manaCost, cooldown,
      text}`. Mana and cooldown can both be set. Ids (`ability-N`) are never reused
      within a class (`nextAbility` in the doc), so sessions can refer to them later.
- [x] Store CRUD + `CampaignsUsingClass` (store tests); `/classes` list, `/classes/new`,
      `/classes/{id}` edit, delete. Nav link "Classes".
- [x] Merged into catalog `heroes` as `custom-<uuid>` via `Server.catalogFor(ctx)`; the
      campaign page, hero validation and new sessions use it (Body and Mind come from the
      class).
- [x] A class that a campaign hero still has is not deleted (409 with the campaign names).

How: `internal/store/hero_classes.go`, `internal/app/classes.go`,
`internal/web/views/classes.templ`; `content.HeroDef` gained optional custom fields and
`content.Ability`. Tests: `internal/store/hero_classes_test.go`,
`internal/app/classes_test.go` (form validation, ids across edits, campaign + session use,
delete protection).

## 3. Inventory and special items
- [x] Hero inventory (done 2026-09-28): gold plus items (name, quantity, notes). No size
      limit, nothing equipped; the GM decides what items do. Carried between quests.
- [x] Campaign page "Inventory" section: gold (`+25`, `-10` or `40`), add / edit / remove
      items. Warns that a running quest's inventory replaces these edits when completed.
- [x] Tracker: gold box, item +/-, remove, give one to another hero, add item. Commands
      `item.add` (merges by name), `item.update`, `item.remove`, `item.give`; gold via
      `hero.update`.
- [ ] Special item library (optional): reusable item definitions with text and counters
      ("every 3rd shot"). Starting gear and the secret vendor stay in the GM's hands.

How: pure `tracker.AddItem/RemoveItem/UpdateItem/NormalizeItems/GoldChange`
(`internal/tracker/inventory.go`, shared by the tracker commands and the campaign forms in
`internal/app/inventory.go`), view `internal/web/views/inventory.templ`, TS
`goldChange` in `src/tracker/abilities.ts`, hero card sections in `src/ui/heroSections.ts`.
Tests: `internal/tracker/inventory_test.go`, `internal/app/inventory_test.go`,
`src/tracker/abilities.test.ts`.

## 4. Tracker: abilities, mana, effects
- [x] Hero state (done 2026-09-28): mana / max mana and abilities copied from the class at
      session start; `cooldowns` maps ability id -> ready round (ready round = round used +
      cooldown; the card shows rounds left).
- [x] `ability.use`: spends mana (stops at 0), starts the cooldown; using it early or short
      of mana is recorded with a note, never refused. Spells say "cast".
- [x] `ability.reset`: one ability or all (Divine Blessing, GM fix).
- [x] Mana via `hero.update` (`mana`, `maxMana`).
- [x] `round.advance` / `round.set` name abilities that became ready.
- [x] Hero card: Mana control, collapsible Abilities (ready / rounds left, Use / Cast,
      Ready, Make all ready; passives as reminders) and Inventory sections.
- [x] Live updates no longer wipe what the GM is typing (`preserveFocus` in `ui/dom.ts`).
- [ ] Effects: `effect.add` / `effect.remove` on a hero or monster, with an optional end
      round (Holy Blessing, Turn Evil, Vanished, Raging, Aimed Shot slowed, cannot defend);
      `round.advance` lists effects that ended; monster panel and board badges.
- [ ] Item counters (Aggamand's Quiver "every 3rd shot").

How: `internal/tracker/abilities.go` (+ `abilities_test.go`); TS `abilityRows`,
`abilityLimit` in `src/tracker/abilities.ts`. Sessions started before this have no
abilities or mana on their heroes; start a new session to get them.

## 5. Stronger monsters
- [ ] Custom monsters: dice expressions for attack and defense, accuracy, larger Body.
- [ ] Monster abilities (bosses) with the same ability model and cooldowns.

## 6. Campaign and quest story
- [ ] Campaign: intro, closing, GM secret notes.
- [ ] Quest: read-aloud intro, goals, GM notes, named passages; "Read aloud" tracker panel.
- [x] Written script (2026-09-28): 23 passages with ids (`P0-01`, `Q2-03`, ...) in
      SCRIPT.md, meant to become one audio clip each.
- [ ] Optional: attach an audio clip to each passage and play it from the tracker's
      "Read aloud" panel.

## Open questions
- Is the Elemental Chambers part of the third map, or a fourth map?
- Rules questions: see the end of `RULES_AND_CLASSES.md`.
