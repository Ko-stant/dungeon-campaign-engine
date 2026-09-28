# Campaign Narrative, Custom Classes and Ability Tracking - TODO

**Last Updated**: 2026-09-28 18:37 EDT
**Branch**: `dce-table-only`

Goal: run the "Three Plagues" campaign from the app. Story text:
`docs/campaigns/three-plagues/NARRATIVE.md`. House rules and classes:
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

## 3. Special items
- [ ] Custom item library: name, text, class it suits (advice only), optional counter
      ("every 3rd shot") and stat reminders (+1 Accuracy).
- [ ] Assign items to campaign heroes; carried between quests.

## 4. Tracker: abilities, mana, effects
- [ ] Hero state: mana / max mana; per-ability `readyRound`; item counters.
- [ ] Commands (one event each, readable summaries):
  - `ability.use` - spends mana, starts the cooldown (ready round = current round +
    cooldown; "Ranger used Multi-Shot, ready in round 7"). The panel shows cycles left
    (ready round - current round).
  - `ability.reset` - ends one or all cooldowns of a hero (Divine Blessing, GM fix).
  - `mana.set` / `mana.adjust`; `counter.adjust`.
  - `effect.add` / `effect.remove` on a hero or monster, with an optional end round
    (Holy Blessing, Turn Evil, Vanished, Raging, Aimed Shot slowed, cannot defend).
- [ ] `round.advance` lists effects that just ended and abilities that became ready.
- [ ] Hero panel: ability buttons with ready / cooldown / mana state; passives as reminders.
- [ ] Monster panel and board: effect badges.

## 5. Stronger monsters
- [ ] Custom monsters: dice expressions for attack and defense, accuracy, larger Body.
- [ ] Monster abilities (bosses) with the same ability model and cooldowns.

## 6. Campaign and quest story
- [ ] Campaign: intro, closing, GM secret notes.
- [ ] Quest: read-aloud intro, goals, GM notes, named passages; "Read aloud" tracker panel.

## Open questions
- Is the Elemental Chambers part of the third map, or a fourth map?
- Rules questions: see the end of `RULES_AND_CLASSES.md`.
