# The Three Plagues - Narration (tagged for voicing)

**Last Updated**: 2026-10-06 19:12 EDT

The text the GM pasted into ElevenLabs Studio (Eleven v4) to voice each passage, one
`<passage id>.txt` per clip, with delivery tags in square brackets. Strip the tags and the
words match the script in [../script/](../script/) (Q1-01 keeps a "But," comma as a
breath). The plain, untagged text and each passage's speaker turns come from
`make narration-text ONLY=P0,Q1` (written to `narration/` at the repo root, gitignored).
Finished clips go in `audio/<campaign id>/` named after the passage id.

Done: P0-01 to P0-04 and the whole of Quest 1 (Q1-01, Q1-02, Q1-N1 to Q1-N3, Q1-03).

## What worked with Eleven v4

- **One opening tag per paragraph, repeated for the same speaker** (the Chronicler's
  `[hushed, unhurried]` in Quest 1, `[warmly, unhurried]` in the prologue). Each paragraph
  starts from the same pace.
- **Mid-paragraph, only pauses and volume:** `[pause]`, `[quietly]`, `[low]`, `[softly]`.
  Attitude words (`[matter-of-fact]`, `[apologetic]`, `[brisk]`) changed pace and pitch
  too much.
- **Keep a character's attitude steady.** Maren went from cold to caring when every
  paragraph had its own mood; one tag (`[dry, practical, a little weary]`) fixed it.
- **"Warmly" brightens.** It stood out in somber scenes; `[gently, unhurried]` or
  `[unhurried, a little somber]` worked there.
- **"Exhausted" and whispers come out too quiet** next to the Chronicler; `[weary, speaking
  clearly]` and `[quieter]` kept Tomas audible.
- **Sound effects** (P0-01 to Q1-01 only) were generated in Studio (Create + > SFX) and
  placed on the chapter's timeline, so the exported clip has them mixed in. Prompts that
  worked: one continuous sound per prompt, ending "no music, no voices"; a quiet, featureless
  room tone looped at 30 seconds under each chapter, matched to the space (open square,
  small study, stone gatehouse).
