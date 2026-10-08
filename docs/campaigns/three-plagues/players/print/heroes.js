// Hero data for the printable player cards: class bases from combat.json, ability text as the
// Players' Handbook words it (second person). Starting gear is left off on purpose: players
// write their gear in pencil. Shared by every card layout in this folder: sheet.html (the one
// in use, a Letter page per hero), tent.html and card.html.
// PDF (gitignored): "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome" --headless
//   --no-pdf-header-footer --virtual-time-budget=8000 --print-to-pdf=sheet.pdf file://$PWD/sheet.html
// Print at 100% scale.
window.HEROES = [
  {
    id: "brentanamo",
    name: "Brentanamo Bay",
    cls: "Barbarian",
    color: "#b91c1c",
    art: "../cards/web/brentanamo.jpg",
    fixed: [
      ["Body", "40"],
      ["Mind (Will)", "3"],
      ["Move", "2d6"],
      ["Hit dice", "1d20"],
      ["Crit", "17-20"],
      ["Defense dice", "1d6"],
    ],
    boxed: [
      ["Accuracy", "+3"],
      ["Damage", "3"],
      ["Avoidance", "2"],
      ["Mitigation", "1", "includes Tough as Nails"],
    ],
    gearCols: ["Dmg", "Acc", "Avoid", "Mitig"],
    reach: "You attack adjacent or diagonal squares. Only you use two-handed weapons and heavy armor.",
    abilities: [
      ["Unburdened Charge", "active · cooldown 3", "Charge 2-6 squares in a straight line (a diagonal step costs 2), stopping at the first enemy or wall. The enemy you hit takes your damage plus the squares charged. Traps on the way trigger but can't hurt you."],
      ["Unleash Fury", "active · cooldown 6", "Roar to the god of battle and attack the nearest enemy, rolling your hit dice twice and keeping the better; allies within 2 squares get +2 Accuracy on their next attack. For 3 rounds, or until you kill something: +3 damage on every hit and +2 Mitigation, but each turn you must move to and attack an enemy if you can."],
      ["Challenge", "free · cooldown 4", "Monsters around you must attack you on their next turn. Protect the Cleric."],
      ["Tough as Nails", "passive", "+1 Mitigation (already in your base). It can bring a hit to 0."],
      ["Cleave", "passive · once a turn", "When you kill a monster, you may attack another one around you at once."],
    ],
  },
  {
    id: "derrick",
    name: "Derrick Rosewood",
    cls: "Cleric",
    color: "#ca8a04",
    art: "../cards/web/derrick.jpg",
    fixed: [
      ["Body", "28"],
      ["Mind (Will)", "6"],
      ["Move", "2d6"],
      ["Hit dice", "2d8"],
      ["Crit", "20"],
      ["Defense dice", "1d6"],
    ],
    boxed: [
      ["Accuracy", "+4"],
      ["Damage", "2"],
      ["Avoidance", "3"],
      ["Mitigation", "0"],
      ["Mana", "16"],
      ["Mana regen", "2", "each round of a fight"],
    ],
    gearCols: ["Dmg", "Acc", "Avoid", "Mitig", "Mana", "Regen"],
    reach: "You attack adjacent squares only, not diagonal. Smite ignores your weapon: 1d20 + Mind to hit, 5 damage. Only you cast spells and can carry shields and tomes.",
    abilities: [
      ["Smite", "spell · 2 mana", "Your main attack, at range (line of sight): roll 1d20 + your Mind to hit, 5 damage, +2 against undead. Determination counts."],
      ["Heal Minor Wounds", "spell · 6 mana", "Heal a hero 12 Body, up to their maximum."],
      ["Turn Evil", "spell · 8 mana · cooldown 5", "The target skips its next attack and can't defend for a round: every hero's attack on it is a sure hit."],
      ["Divining", "spell · 3 mana", "Reveal traps, hidden doors and treasure within 6 squares, through walls and doors."],
      ["Prayer", "spell · no mana · cooldown 5", "Restore half your maximum mana (rounded up). It takes your action, and you can't defend until your next turn. The only way to get mana back outside a fight."],
    ],
  },
  {
    id: "mordecai",
    name: "Mordecai Muldoon",
    cls: "Ranger",
    color: "#15803d",
    art: "../cards/web/mordecai.jpg",
    fixed: [
      ["Body", "30"],
      ["Mind (Will)", "4"],
      ["Move", "2d6"],
      ["Hit dice", "2d10"],
      ["Crit", "18-20"],
      ["Defense dice", "1d6"],
    ],
    boxed: [
      ["Accuracy", "+4"],
      ["Damage", "2"],
      ["Avoidance", "4"],
      ["Mitigation", "0"],
    ],
    gearCols: ["Dmg", "Acc", "Avoid", "Mitig"],
    reach: "You shoot anything in line of sight, adjacent and diagonal squares included. Only you use bows and crossbows.",
    abilities: [
      ["Multi-Shot", "active · cooldown 5", "Shoot up to 3 different enemies in line of sight as one action."],
      ["Aimed Shot", "active · cooldown 5", "A sure hit. The target is marked: every hero gets +2 Accuracy against it until it dies or the fight ends. Your movement is halved this round and next."],
      ["Rain of Arrows", "active · cooldown 7", "Every enemy within 2 squares of a square you choose is attacked once, with half your Accuracy (rounded down). Roll 1d4 arrows once; each enemy hit takes your damage times the arrows."],
      ["Point Blank", "passive", "When you attack an adjacent monster: +1 Accuracy and +1 damage."],
    ],
  },
  {
    id: "papi",
    name: "Papi Ponzi",
    cls: "Rogue",
    color: "#4b5563",
    art: "../cards/web/papi.jpg",
    fixed: [
      ["Body", "28"],
      ["Mind (Will)", "5"],
      ["Move", "2d6"],
      ["Hit dice", "2d10"],
      ["Crit", "15-20"],
      ["Defense dice", "1d6"],
    ],
    boxed: [
      ["Accuracy", "+2"],
      ["Damage", "1"],
      ["Avoidance", "4"],
      ["Mitigation", "0"],
    ],
    gearCols: ["Dmg", "Acc", "Avoid", "Mitig"],
    reach: "You attack adjacent squares only, not diagonal. You always fight with two weapons; both add their damage. Only you can disarm traps.",
    abilities: [
      ["Nimble Fingers", "passive", "Disarm a trap on 1d8; you fail only on a 1."],
      ["Exploit Opening", "passive", "Crit on 13-20 (instead of 15-20) against a monster you're behind."],
      ["Fan of Cards", "active · cooldown 3", "Razor-edged cards at every enemy around you: one attack against each, rolled separately."],
      ["Venom Vial", "active · cooldown 4", "Crack a vial over your blades and strike: a normal attack, and on a hit the target loses 3 Body at the start of each of its next 3 turns (6 on a crit). A new vial restarts the count; it doesn't stack."],
      ["Vanish From Sight", "reaction · cooldown 5", "When attacked, vanish: that attack is canceled, and no enemy can target you until the monsters' next turn."],
      ["Riposte", "reaction · cooldown 2", "When a monster's attack on you misses, strike back at once for half your damage (rounded down), no roll."],
    ],
  },
];

// h builds an element: h("div.a.b", {style: "..."}, child, "text", ...).
window.h = function h(tag, attrs, ...kids) {
  const [name, ...classes] = tag.split(".");
  const el = document.createElement(name || "div");
  if (classes.length) el.className = classes.join(" ");
  if (attrs && typeof attrs === "object" && !Array.isArray(attrs) && !(attrs instanceof Node)) {
    for (const [k, v] of Object.entries(attrs)) {
      if (k === "style") el.style.cssText = v;
      else el.setAttribute(k, v);
    }
  } else if (attrs != null) {
    kids.unshift(attrs);
  }
  for (const kid of kids.flat()) {
    if (kid == null || kid === false) continue;
    el.append(kid instanceof Node ? kid : document.createTextNode(String(kid)));
  }
  return el;
};
