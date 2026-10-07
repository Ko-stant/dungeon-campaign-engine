# Online play: getting players into a game

**Last Updated**: 2026-10-06

How friends get into an online game at `https://dce.kostant.dev`, what the GM clicks, and what
the players see, scenario by scenario. (How it is built: `docs/ONLINE_AND_RULES_PLAN.md`,
Phases 4-6.)

## The two gates, and who plays which hero

1. **The site (once per person).** Anyone can sign in with Discord, but they see only
   "Waiting for the GM" until you let them in from **Members**.
2. **The game (each session).** A session is listed in the players' **Play** lobby only while
   it is **open to players**. Its row on the campaign page says which: **Open to players** or
   **Closed to players**.

**Heroes belong to players for the whole campaign.** Once a hero is someone's, it is theirs in
every later session and quest of that campaign, until you change it. On the campaign page
each hero has:

- **Player** (the dropdown): who plays the hero online right now, or **Nobody (free)**.
- **usually ...** (under it): the hero's usual player, the name you gave at the table. A
  handover never changes it.

Players can take only **free** heroes. Handing over a hero someone else plays is your call.

## Scenario 1: a friend's first time

1. They sign in at `https://dce.kostant.dev` with Discord and see "Waiting for the GM".
2. You open **Members** (it shows "Members (1)" while someone waits) and click **Let in**.
3. They reload the page. They land on **Campaigns**, which points them to **Play**. They
   don't need their own campaign.

## Scenario 2: game night

**You:**

1. On the campaign page, start the session for the quest, as at the table.
2. On its row, click **Start online play**. That turns the rules on and opens the game to
   players in one step. The row then shows **Open to players** and **Online**.
3. Run the game from the tracker (**Resume**). The rules console at the top right shows whose
   turn it is and who has acted, and the header shows who is connected.

**Each player:**

1. **Play** (in the nav) lists the open games. They click **Join** on yours.
2. The join page shows:
   - **Your heroes**: heroes already theirs. They click **Play**.
   - **Free heroes**: unclaimed heroes, with who usually plays each. **Play this hero** makes it
     theirs and brings it into the game.
   - **Make a new hero**: a name and a class. The hero joins the campaign as theirs.
3. **Play** opens their seat: their hero sheet, the board, their buttons and the log.

Joining after the game has started is fine. If the start squares are taken, the new hero
waits off the board until you place them from the tracker.

If the lobby is empty, check the session's row: it must say **Open to players** (click **Let
players join** if not). The lobby doesn't update by itself, so the player reloads it.

## Scenario 3: the same group, the next session or quest

Nothing to set up for the players. Start the next session, then **Start online play**. Each
player joins and finds their hero under **Your heroes**, along with everything the hero
carried over.

## Scenario 4: someone covers for a friend who can't come

Say Pat can't make it, and Sam will play Pat's Ilsa as well as their own Grom.

1. Before or during the game, on the campaign page, set Ilsa's **Player** to **Sam**. The
   page saves as soon as you pick.
2. Sam's seat gains an **Ilsa** tab at once, even if it is already open. Sam plays both: the
   seat follows whichever hero's turn it is, and the tabs switch sheets.
3. Ilsa still reads "usually Pat".
4. Afterward, set Ilsa's **Player** back to **Pat** (the list offers members, so Pat must have
   been let in), or to **Nobody (free)**, and Pat picks Ilsa again from **Free heroes** next
   time.

## Scenario 5: one player, several heroes

A player can hold any number of heroes: they pick more on the join page, or you hand them
over (Scenario 4). Their seat has a tab for each.

## Scenario 6: a new campaign

1. Create the campaign and add the heroes as usual. The **Player** box when adding a hero is
   the usual player's name.
2. Start the first session and **Start online play**.
3. Players pick their heroes from **Free heroes** (they can see who each is meant for), or
   make new ones. Or you set each hero's **Player** yourself before the game.

## Scenario 7: taking a hero back, or someone leaving the group

- **A hero:** set its **Player** to **Nobody (free)**. It drops off that player's seat at once
  and becomes free to pick.
- **A person:** on **Members**, **Remove** them. They can't get into anything again until you
  **Let in** them again. Their heroes stay theirs on the campaign page until you change them.

## Closing a game

- **Close** on the session's row takes it out of the lobby. Players already in keep playing
  from their seats; nobody new can join.
- **Complete** the session in the tracker as at the table when the quest is over.

## When something looks wrong

| What you see | Why, and what to do |
|---|---|
| A player sees "Waiting for the GM" | They aren't let in yet: **Members**, **Let in**. |
| The lobby is empty | The session isn't open: on its row, **Let players join** (or **Start online play**). Then the player reloads **Play**. |
| A hero is missing from **Free heroes** | Someone already plays it (shown under "taken"). Hand it over on the campaign page. |
| A player has no **Play** button for the game | They don't have a hero in it yet: they pick or make one on the join page first. |
| A player has no turn buttons | Online play isn't started, or it isn't their hero's turn yet: the top line says which. |
| The page says it is reconnecting | The site was updated or the connection dropped. It reconnects by itself; reload if it doesn't. |
