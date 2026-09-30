# Campaign Narrative, Custom Classes and Ability Tracking - TODO

**Last Updated**: 2026-09-29 21:30 EDT
**Branch**: `main` (the `dce-table-only` branch was merged on 2026-09-28; work happens on `main`)

Goal: run the "Three Plagues" campaign from the app. Story bible (world, cast, secrets):
`docs/campaigns/three-plagues/NARRATIVE.md`; read-aloud passages with ids, speakers and
voice notes: `docs/campaigns/three-plagues/script/` (one file per part). House rules and classes:
`docs/campaigns/three-plagues/RULES_AND_CLASSES.md`.

## Resume here (2026-09-29)

**Where things stand**
- App features for the campaign are built and committed: dice, custom classes, inventory,
  abilities and cooldowns, the read-aloud script with a reader and audio clips, and map
  editor aids (note list, placement previews, two-square and exit doors, room painting).
- The four classes exist in the dev database with placeholder stats (tuning pass later).
- The script is being revised part by part with the GM, towards a D&D-style campaign that
  does not tell the heroes about Soul Gems, plagues or bosses up front. Done: the prologue
  (P0-01 to P0-04), Quest 1 (with notes Q1-N1 to Q1-N3) and Quest 2 (with notes Q2-N1 to
  Q2-N3). The heroes learn the rest from notes left by failed parties.
- The campaign in the app still has the **old 26-passage script**. Load the revised one
  only when the GM asks: `make load-script CAMPAIGN="Three Plagues"`.
- Maps: Quest 1's board ("Adventurers' Herald", quest "Crumbling Halls") is built; Quest
  2's board ("Eastmarch", 48x30) is started; Quest 3's is not. What each map must contain
  is in `docs/campaigns/three-plagues/MAPS.md`.

**Next**
1. Maps for Quests 2 and 3, checked against `MAPS.md` and the script checklists (the GM
   builds; keep the story and the maps in step).
2. Script: Quest 3 (`04-quest-3-the-wardens-rise.md`): drop the up-front Ogre Lord names,
   gems and sockets from Q3-01, stop counting plagues in Q3-04 and Q3-06, add the
   pilgrims' notes from the Rise and a board checklist.
3. Script: Soul Gem moments (`05-soul-gem-moments.md`) say "Soul Gem" before the heroes
   know the name; the ending (`06-the-end.md`) has Voss say "three monsters".
4. Then load the script into the campaign, and add Quests 2 and 3 as campaign chapters.

**Working with the GM**
- The GM reviews each part and gives line edits; apply them as written.
- Keep `NARRATIVE.md` (story bible: cast, places, clues, failed parties) in step with
  every script change, including clue numbers referenced in the script.
- Edit one script part file at a time; `make print-script` checks the whole script parses.

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
- [x] Read-aloud script and tracker panel (done 2026-09-28). The script is one Markdown
      text per campaign (the script folder's format), pasted into the campaign page's Script box
      and parsed on save (errors name the line; a bad script is not saved). The tracker's
      right panel lists it by section, opening the section for the current quest ("Quest N"
      = chapter N, else a title naming the quest). Clicking a passage opens a large-type
      reader with notes, speakers, goals, Previous/Next (and arrow keys); "Mark as read"
      sends `passage.read`, which logs "Read aloud: Q2-03 The tithe barn" and ticks it off
      (kept across maps in `readPassages`).
      How: `internal/script` (parser + tests, including the real script), migration
      `00005_campaign_script.sql`, `internal/store/campaign_script.go`,
      `internal/app/script.go`, `views/script.templ`, `internal/tracker/passages.go`,
      TS `src/tracker/script.ts` and `src/ui/readAloud.ts`.
- [x] Campaign intro, closing and quest text live in the script; no separate fields.
- [x] Written script (2026-09-28): passages with ids (`P0-01`, `Q2-N1`, ...), each meant to
      become one audio clip. 34 passages as of 2026-09-29 (note passages are `Qn-Nm`).
- [x] Script split into part files (done 2026-09-29): `script/01-prologue.md` ...
      `06-the-end.md` plus a README (format guide, passage list). `script.Assemble` joins
      the numbered files (dropping each file's header above its `##` heading) and
      `script.Locate` turns a parse error's line into "file line N".
      `make load-script CAMPAIGN="Three Plagues"` checks and saves it to the campaign
      (`cmd/load-script`); `make print-script` prints the joined text for pasting.
- [x] Audio clips (done 2026-09-28): files named after passage ids (`Q2-03.mp3`; `Q3-09a`
      is extra clip "a" of Q3-09) in `AUDIO_DIR/<campaign id>/` (default `./audio`,
      gitignored). Uploaded on the campaign page (all-or-nothing name check) or copied in by
      hand; listed with their passages and playable there. The reader loads the passage's
      clip into one persistent player (playback survives live updates), shows lettered
      clip buttons, optional Autoplay (remembered per browser) and Space to play/pause.
      How: `internal/audio` (library + tests), `internal/app/audio.go` (+ tests), TS
      `passageClips` in `src/tracker/script.ts`, player in `pages/tracker.ts` and
      `ui/readAloud.ts`.

## Open questions
- Settled: the Elemental Chambers are a sealed section of map 3 behind a locked gate.
- Rules questions: see the end of `RULES_AND_CLASSES.md`.
