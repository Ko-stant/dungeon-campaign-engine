# The Three Plagues - Read-Aloud Script

**Last Updated**: 2026-09-29 16:39 EDT

Everything the players hear, in play order, split into one file per part so each part can
be edited on its own. Each passage is meant to become one audio clip. World, cast and
secrets are in [NARRATIVE.md](../NARRATIVE.md).

## Files

The numbered files, in name order, make up the script. This README is not part of it.

| File | Section |
|---|---|
| [01-prologue.md](01-prologue.md) | Prologue |
| [02-quest-1-the-crumbling-halls.md](02-quest-1-the-crumbling-halls.md) | Quest 1 - The Crumbling Halls |
| [03-quest-2-the-bloated-fields.md](03-quest-2-the-bloated-fields.md) | Quest 2 - The Bloated Fields |
| [04-quest-3-the-wardens-rise.md](04-quest-3-the-wardens-rise.md) | Quest 3 - The Wardens' Rise |
| [05-soul-gem-moments.md](05-soul-gem-moments.md) | Soul Gem moments (optional) |
| [06-the-end.md](06-the-end.md) | The End |

To add a part, add a numbered file that starts its script with a `## Section` heading.
Anything above that heading (title, Last Updated line, notes) is left out when the files
are joined.

## Loading it into the app

```bash
make load-script CAMPAIGN="Three Plagues"
```

This joins the files, checks the result (an error names the file's section and line of
the joined script) and saves it as the campaign's script. A running quest picks it up with
**Reload** in the tracker's Read aloud panel. `make print-script` writes the joined script
to the terminal instead, for pasting into the campaign page's Script box.

## How the script is written for audio

- **One passage, one clip.** Name each file after its ID (`P0-01.mp3`, `Q2-03.mp3`) and
  upload it on the campaign page (Audio clips); the tracker's reader plays it. Extra clips
  for one passage add a letter: `Q3-09a.mp3` to `Q3-09e.mp3`. Formats: mp3, m4a, ogg,
  opus, wav, webm, flac.
- **The quoted text is exactly what is spoken.** Numbers are written as words and there
  are no stage directions inside the text. Delivery notes are in each passage's **Voice**
  line.
- **Passages with more than one speaker** are split into lines, one speaker each. Record
  or generate each line separately and join them, or use a tool that supports several
  voices.
- **Pronunciation** of every name is in the cast table in [NARRATIVE.md](../NARRATIVE.md).
- **Quest goals** are table text. Read them out or show them; they are not part of a clip.
- **Sound** notes are optional background suggestions.
- **Keep the format** the tracker reads: `### ID - Title` headings, `- **Label:**` notes,
  `**Speaker**` lines and `>` quotes. A `\` at the end of a quoted line keeps the line
  break (used in the verse).
- **The Soul Gems do nothing in play** except open the portal. The G passages are flavor
  for quiet moments between fights.

## Passage list

| ID | Passage | Speaker(s) |
|---|---|---|
| P0-01 | Haldmere Cross | Narrator, Sergeant Hale |
| P0-02 | The road to Voss Keep | Narrator |
| P0-03 | The lord's request | Lord Voss, Narrator |
| P0-04 | At the gate | Maren Ashby |
| Q1-01 | The Crumbling Halls | Narrator |
| Q1-02 | The Warden's verse | Narrator |
| Q1-N1 | The patrol's last watch (note) | Narrator, Tomas Reed |
| Q1-N2 | A warning in red (note) | Narrator, Bram Greyford |
| Q1-N3 | The pilgrims' mark (note) | Narrator, Sister Wenna |
| Q1-03 | The eastern gate | Narrator |
| Q2-01 | The Eastmarch | Narrator |
| Q2-02 | The boy in the cellar (optional) | Narrator, Tam |
| Q2-N1 | The Greyford camp (note) | Narrator, Lark |
| Q2-N2 | The pilgrims' road (note) | Narrator, Sister Wenna |
| Q2-N3 | A note for Tam (note) | Narrator, Hesta Fenwick |
| Q2-03 | The tithe barn | Narrator |
| Q2-04 | The green stone | Narrator |
| Q2-05 | Smoke on the hills | Narrator |
| Q3-01 | The Wardens' Rise | Narrator |
| Q3-02 | The Hollow Merchant (secret) | Narrator, Ilsabet |
| Q3-03 | The granary | Narrator, Gorrak |
| Q3-04 | The second Soul Gem | Narrator |
| Q3-05 | The great hall | Narrator, Varnok |
| Q3-06 | The third Soul Gem | Varnok, Narrator |
| Q3-07 | The portal | Narrator |
| Q3-08 | The gems are set | Narrator, Dread Wraith |
| Q3-09 | The Wraith's taunts (optional) | Dread Wraith |
| G-01 | The gem that wails (optional) | Narrator |
| G-02 | Two gems together (optional) | Narrator |
| G-03 | The pull of the Rise (optional) | Narrator |
| E-01 | The Chambers restored | Narrator |
| E-02 | The lord's thanks | Lord Voss |
| E-03 | Epilogue (optional) | Narrator |
| X-01 | If the heroes fall | Narrator |
