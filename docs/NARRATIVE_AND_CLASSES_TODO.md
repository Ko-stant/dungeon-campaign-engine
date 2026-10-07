# Campaign Narrative, Custom Classes and Ability Tracking - TODO

**Last Updated**: 2026-10-06 19:12 EDT
**Branch**: `main` (the `dce-table-only` branch was merged on 2026-09-28; work happens on `main`)

Goal: run the "Three Plagues" campaign from the app. Story bible (world, cast, secrets):
`docs/campaigns/three-plagues/NARRATIVE.md`; read-aloud passages with ids, speakers and
voice notes: `docs/campaigns/three-plagues/script/` (one file per part). House rules and classes:
`docs/campaigns/three-plagues/RULES_AND_CLASSES.md`.

## Resume here (2026-10-03)

**Where things stand**
- App features for the campaign are built and committed: dice, custom classes, inventory,
  abilities and cooldowns, the read-aloud script with a reader and audio clips, and map
  editor aids (note list, placement previews, two-square and exit doors, room painting).
- The four classes exist in the dev database with placeholder stats (tuning pass later).
- **Combat system** (2026-10-03): the campaign replaces combat dice with hit dice + a d20
  crit die, fixed damage from equipment, monster Avoidance, opposed hero defense rolls,
  mitigation, and cooldowns that only finish during a fight. Step 1 (the rules) is written
  up under "Combat" in `docs/campaigns/three-plagues/RULES_AND_CLASSES.md`, with a five-step
  combat roadmap. `bun scripts/combat-odds.ts` prints exact odds (`internal/web/src/combat/odds.ts`).
  2026-10-04: the GM's answers are folded in (Determination adopted, Faltering, Smite,
  "cannot defend", starting kit, Quest 1 treasure and targets under "Campaign notes"), and
  the step 2 attack numbers are proposed (hit dice, Accuracy, crit, damage per class).
- The script is being revised part by part with the GM, toward a D&D-style campaign that
  does not tell the heroes about Soul Gems, plagues or bosses up front. Done: the prologue
  (P0-01 to P0-04), Quest 1 (with notes Q1-N1 to Q1-N3) and Quest 2 (with notes Q2-N1 to
  Q2-N3). The heroes learn the rest from notes left by failed parties.
- The campaign in the app has the revised 34-passage script (loaded by the GM on
  2026-10-06). Reload it after script edits only when the GM asks:
  `make load-script CAMPAIGN="Three Plagues"`.
- Maps: Quest 1's board ("Adventurers' Herald", quest "Crumbling Halls") is built; Quest
  2's board ("Eastmarch", 48x30) is started; Quest 3's is not. What each map must contain
  is in `docs/campaigns/three-plagues/MAPS.md`.

**Next**
- **Table testing round (2026-10-07)** on a test copy of the campaign (deleted after): the
  tracker got a loot quick-add on the campaign page, note letters in Read aloud, a fixed
  mode bar and hideable sidebars, sections that start closed, green class names on the TV,
  fight reminders (the fight button pulses), no log line for repeat reveals, monster
  movement on the selected monster (Goblin Archer 8, Goblin Warlock 6, Orc Archer 6,
  Specter 8; applied by the GM), red "needs N mana", and the Ranger's Point Blank passive
  (applied; handbook republished).
- **Voiced audio (2026-10-06): the prologue and Quest 1 are done** (10 clips, about 15
  minutes), made by the GM in ElevenLabs Studio with Eleven v4 and placed in
  `audio/<campaign id>/` (gitignored; the GM keeps copies elsewhere). `make narration-text
  ONLY=...` writes paste-ready text; the GM-approved copies with v4 delivery tags are in
  `docs/campaigns/three-plagues/narration/` (see its README for what worked). Along the way
  P0-04 gained a Chronicler lead-in (the night at the Tollen Arms, meeting Maren), Q1-01 an
  ending line, and Q1-02, Q1-N1 and Q1-N3 were brought in line with the GM's quest notes
  (the chalk safe path is dropped). Quests 2-3 audio waits until after game day.
- **Table Polls, 3 of 4 votes (2026-10-05; Brentanamo's player still to vote):** narration
  a mix of audio and the Chronicler (3-0); monster Body exact 2-1; critical misses keep
  (3-0); monster crits keep (3-0); the Chronicler tracks cooldowns and Determination 2-1
  (one "both"); special crits described by the player 2-1; the TV log always open (3-0),
  so the player screen now opens with the log showing.
- **Done 2026-10-05, applied by the GM:** `make fill-campaign CAMPAIGN="Three Plagues"
  APPLY=1` updated the Cleric's Smite text (1d20 + Mind) and added the loot list (the seven
  Quest 1 finds and the two potions) to the campaign. Smite on Mind was
  simulated as no meaningful change (72% / 81% with nobody dead, without / with finds).
- **Quest 1 dry run (2026-10-05)** on a throwaway copy (Crumbling Halls, the real party):
  reveal, fight, Determination, odds, abilities, mana, effects, traps, loot, potions, gold,
  fight end and quest completion all work. Findings, as the GM decided them:
  1. Done: a damage/heal box beside Body (and Mana) for heroes and monsters: 9 or -9 takes
     damage, +8 heals up to the maximum, =12 sets; one entry, one event.
  2. Kept: a monster at 0 Body stays until Kill is pressed (no accidental deaths from a
     mistyped number).
  3. Open: the Cleric's odds use the mace; Smite gets its own line once the GM decides
     whether Smite rolls 1d20 + Mind.
  4. Done: "gained 3 Healing Potions"; the Stranger has a body-only stat line (1 Body, no
     combat stats) in combat.json, applied to the Three Plagues campaign.
  Not pursued: removing a damaged monster leaves its damage lines (the GM kills instead).
  Waiting until after the Quest 1 session: the Tempest-God Axe, a Smite item, and any
  number changes (the GM takes notes on game day). Mental attacks wait for Quests 2-3.
- **Decided (2026-10-05): elite attacks stay as they are.** The GM Bestiary
  (https://claude.ai/artifact/CoGhfTV4sUUb5uecWw36xA, GM only; `bun scripts/gm-bestiary.ts`
  writes `docs/campaigns/three-plagues/gm/bestiary.html`) shows defense (avoidance + 1d6)
  barely matters against elites: dread warriors/specters (2d10+2) hit 67-91% and the
  gargoyle (2d10+3) 74-94%; orcs hit 30-70%, goblins 14-46%. The GM keeps it: the party
  must focus and choose its actions wisely, and the numbers change only if real fights go
  badly. Levers if needed: bigger defense dice for agile classes (Rogue/Ranger 1d10),
  swingier lower-average elite hit dice with more damage (e.g. 1d20+3), or higher starting
  avoidance with more elite Body; re-run the simulator against the ~80% Quest 1 target.
- **Player pages** (published, private until the GM shares them): Players' Handbook
  (https://claude.ai/artifact/C1FiMQLNVA3pw5Jmvu7Ma2, source `players/handbook.html`;
  hero sheets now split class base / starting gear / total, card art from `cards/web/`),
  Table Polls (https://claude.ai/artifact/LatPNDvxX5j7p3edeQgHNL, `players/polls.html`,
  votes in the page db). The GM may edit the HTML for wording and ask to republish (same
  file path keeps the URL; read their edits first).
- **Player screen (TV)**, built 2026-10-04/05 (phases 1-5 committed and pushed): a live,
  read-only player view in a second tab on the game-room TV ("Open player screen" in the
  tracker header). Decisions, what each phase did and what's next (player docs and polls,
  which need the GM) are in `docs/PLAYER_SCREEN_PLAN.md`.
0. Combat: the simulator is built (`bun scripts/combat-sim.ts`, numbers in
   `scripts/combat-config.ts`, "Simulator" in the rules doc) and calibrated to the GM's
   targets (75-80% clear Quest 1 with nobody dead; attack-only fails). Step 4 (abilities) is
   done: the merged, pruned kit with Venom Vial is in the simulator and Quest 1 is
   recalibrated (92% clear, 79% with nobody dead). Gear: class base stats, the starting kit
   and 7 proposed Quest 1 finds are in `scripts/combat-config.ts` and the rules doc (finds
   lift Quest 1 to 81% with free 8-Body potions; fodder damage doesn't move survival). The dread warrior stays at 15;
   only trap G's dread warrior scales (27 Body per hero in the room; note G on the board
   explains it with four torches). The finds are approved (2026-10-05). Next: step 5 (app
   support: class, monster and item stats; fights in the tracker). Notes G and L on the
   board describe the trap G torches, the prayer beads, the other finds (W, S, O, X, T) and
   the potions (V). Step 5 is under way in phases 5a-5f (rules doc, "Combat roadmap");
   5a (class combat stats) and 5b (per-campaign monster stats) are done (committed).

   5c (fights in the tracker: fight start/end, mana regeneration, the out-of-fight cooldown
   floor, generic effects) is done, checked in the browser and committed. 5d (item kind,
   stats and equipped flag; hero totals; one party purse, hero gold merged in by migration
   00007) is done and committed. 5e (`make fill-campaign CAMPAIGN="Three Plagues"`, a dry
   run unless `APPLY=1`; numbers in `docs/campaigns/three-plagues/combat.json`, which the
   simulator reads too) is built, and the GM ran it on the Three Plagues campaign
   (2026-10-04): the four classes, 12 monster stat lines, and the party (Mordecai Muldoon,
   Ranger; Derrick Rosewood, Cleric; Brentanamo Bay, Barbarian; Papi Ponzi, Rogue) with
   their starting kits equipped (committed 7715e32). 5f (odds hints in the monster panel) is
   done and checked in the browser, which completes step 5. Next for combat: play Quest 1
   with the new rules and tune from the table (Smite odds, effects such as Raging and the
   Quiver could join the hints later).
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

How: `internal/dice` (Go: `Parse`, `Expr.String/Min/Max/Roll`, text marshaling) and
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
- [x] Campaign page "Inventory" section: gold (`+25`, `-10` or `=40`), add / edit / remove
      items. Warns that a running quest's inventory replaces these edits when completed.
- [x] Tracker: gold box, item +/-, remove, give one to another hero, add item. Commands
      `item.add` (merges by name), `item.update`, `item.remove`, `item.give`; gold via
      `hero.update`.
- [x] Item library (done 2026-10-05): each campaign's loot list (kind, stats, healing,
      notes), filled from combat.json and handed out from the tracker's loot picker;
      potions are used with one click. Counters ("every 3rd shot") are still open (section 4).

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
- [x] Effects (done 2026-10-05, step 5c): `effect.add` / `effect.remove` on a hero or
      monster with an optional countdown in fight rounds; `round.advance` lists effects that
      ended, a fight's end clears timed ones, and the TV shows them on the cards.
- [ ] Item counters (Aggamand's Quiver "every 3rd shot").

How: `internal/tracker/abilities.go` (+ `abilities_test.go`); TS `abilityRows`,
`abilityLimit` in `src/tracker/abilities.ts`. Sessions started before this have no
abilities or mana on their heroes; start a new session to get them.

## 5. Stronger monsters
- [x] Monster combat stats (done 2026-10-04, step 5b): per-campaign stat lines (Body,
      Avoidance, hit dice, damage, traits, abilities text; Body-only lines for monsters
      that don't fight), filled from combat.json.
- [ ] Monster abilities (bosses) with the same ability model and cooldowns (Quests 2-3;
      today a stat line's abilities text is shown on the TV card).

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
