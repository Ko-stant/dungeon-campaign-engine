# The Three Plagues - Maps

**Last Updated**: 2026-10-03 16:01 EDT

What each quest's board has to contain for the story to work, so map building and the
script stay in step. The story is in [NARRATIVE.md](NARRATIVE.md); the read-aloud text,
with a board checklist at the top of each quest file, is in [script/](script/README.md).
When a map changes the story (a place moves, is added or cut), update the matching script
file and its checklist in the same change, and the other way round.

The GM builds the boards; Claude does not edit them unless asked. Placing furniture and
monsters comes later; this file is about layout.

## How the app represents things (useful while building)

- **Rooms are named places.** Name each room after the place ("Tithe barn", "Fenwick
  house") so the tracker's hover label and the room reveal read naturally.
- **Corridors are paths**: halls indoors; roads, lanes and field tracks outdoors. Solid
  rock is anything impassable (hillside, deep water, thick hedgerow). Draw walls where two
  open areas must not connect.
- **Doors**: normal, secret, gate, and **exit door** (for the board's edge; no warning
  there). Doors and gates can be **two squares wide** for the physical models. Gates can
  start **locked**.
- **Exit squares** mark where the party leaves the map; **start squares** where they
  arrive. Moving between maps mid-session is the tracker's Travel.
- **Quest notes** mark each find; start the note text with the passage id ("Q2-N1 ...").
- Map-creator shortcuts: N new room; "New room for each shape" under the Room brush.

## Quest 1 - The Crumbling Halls

- **Board**: "Adventurers' Herald", 30x24 (the GM's). Quest: "Crumbling Halls",
  chapter 1 of the Three Plagues campaign.
- **Status**: board built; notes being placed. The board checklist is at the top of
  `script/02-quest-1-the-crumbling-halls.md`.
- **Must contain**: the western doors (start, P0-04 / Q1-01); a Warden burial room with a
  tomb (Q1-02, the scratched verse); a collapsed passage with Tomas Reed (Q1-N1); a room
  near the eastern gate for Bram's warning (Q1-N2); a statue with the pilgrims' chalk mark
  (Q1-N3); the eastern gate as the exit (Q1-03). Optional: a chalk-marked safe route past
  a trap; something in the walls that shies from light.
- **Leads to**: Quest 2's start, by the Halls' eastern gate.

## Quest 2 - The Bloated Fields

- **Board**: "Eastmarch", 48x30 (started by the GM). Not yet a chapter of the campaign.
- **Setting**: open farmland under a yellow sky, criss-crossed by gray threads that all run
  to the tithe barn at the center. Outdoors, so rooms are buildings and fields, corridors
  are lanes and tracks.
- **Must contain** (see the checklist in `script/03-quest-2-the-bloated-fields.md`):
  - **Start** by the Crumbling Halls' eastern gate, on the side the party enters from.
  - **Millbrook**, a small village on the edge of the blight, recently overrun: several
    houses, including **the Fenwick house** with a cellar (Q2-N3, and Tam in Q2-02 if used).
  - **A farmhouse** where the Greyford Company camped, six years ago (Q2-N1).
  - **The tithe barn** at the heart of the farmland: the Blightweaver's lair, a large room
    with cocoons in the web (Q2-03, Q2-04). Harl, Brother Ansel and the Millbrook folk hang
    there.
  - **A wayside shrine or roadside stone north of the barn**, on the pilgrims' route around
    the fields (Q2-N2).
  - **Exit** on the road toward the Rise, on the far side from the start (Q2-05).
- **Suggested**: the threads' pull toward the barn read as lanes converging on the center;
  the blight worse near the barn than at the edges (Millbrook only just reached).

## Quest 3 - The Wardens' Rise

- **Board**: not started; 48x30.
- **Setting**: an old Warden fortress on the highest of the broken hills, overrun by ogres.
  The Elemental Chambers are a sealed section of this same map.
- **Must contain** (the Quest 3 script is still to be revised; its checklist will follow):
  - **Start** at the foot of the Rise, on the road from the Fields.
  - **Ilsabet Crane's hidden fire** near the start, behind a collapsed wall where no ogre
    looks (Q3-02, the secret vendor).
  - **Ogre camps** in the cracked courtyards.
  - **The old granaries**: Gorrak Hollowgut's hoard of stolen grain and carcasses (Q3-03,
    Q3-04).
  - **The great hall**: Varnok Ironjaw on the Wardens' stone seat, with Skarr and Brugg
    (Q3-05, Q3-06).
  - **The summit**, open to the sky, with **the portal**: a locked gate (two squares wide
    if the model is) carved with the four signs, three sockets and a scratched-out fifth
    sigil (Q3-07, Q3-08). It opens only when the three stones are set.
  - **The Elemental Chambers** behind the portal, sealed off until then: four vaults,
    **Stone, Tide, Gale and Hearth**. The Dread Wraith rises in the vault of Stone (Q3-08,
    Q3-09, E-01).
  - Notes from Sister Wenna's pilgrims on the Rise (still to be written).
- **Open questions for the GM**: the order the party meets Gorrak and Varnok (the script
  assumes Gorrak first); whether the Chambers' four vaults connect to each other or only to
  a central space; where the pilgrims' last notes are found.

## Campaign chapters

Only Quest 1 is a chapter of the Three Plagues campaign so far. Add the Eastmarch quest
(and later the Rise) as chapters 2 and 3 on the campaign page, so the Read aloud panel
opens the right script section and Travel lists the next map.
