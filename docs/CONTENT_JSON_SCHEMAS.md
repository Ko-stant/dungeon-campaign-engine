# Content JSON Schemas

## Overview

This document defines the JSON schemas for all content types in the HeroQuest engine. Each schema represents a card or game element that can be loaded from individual JSON files.

## Content Organization

```
/content/
  ├── equipment/          # Weapons, armor, potions
  ├── artifacts/          # Magical artifacts and jewelry
  ├── treasures/          # Treasure deck cards (gold, hazards, wandering monsters)
  ├── spells/             # Spell cards for Wizard/Elf
  ├── dread_spells/       # GM dread spell cards
  └── campaigns/
      └── base/
          ├── campaign.json
          ├── equipment_deck.json
          ├── artifact_deck.json
          ├── treasure_deck.json
          ├── spell_deck.json
          └── quests/
```

---

## Equipment Schema

**Location**: `/content/equipment/*.json`

**Types**: weapon, armor, potion, shield

### Weapon Example

```json
{
  "id": "longsword",
  "name": "Longsword",
  "type": "weapon",
  "subtype": "melee",
  "attack_dice": 3,
  "defense_dice": 0,
  "attack_bonus": 0,
  "defense_bonus": 0,
  "attack_diagonal": true,
  "attack_adjacent": true,
  "ranged": false,
  "range": 1,
  "cost": 350,
  "usable_by": ["berserker", "barbarian", "knight", "dwarf", "elf", "explorer", "druid"],
  "description": "This long blade gives you the attack strength of 3 combat dice. Because of its length, the longsword enables you to attack diagonally. May not be used by the wizard.",
  "card_image": "/assets/cards/equipment/longsword.png"
}
```

### Ranged Weapon Example

```json
{
  "id": "crossbow",
  "name": "Crossbow",
  "type": "weapon",
  "subtype": "ranged",
  "attack_dice": 3,
  "defense_dice": 0,
  "attack_bonus": 0,
  "defense_bonus": 0,
  "attack_diagonal": true,
  "attack_adjacent": false,
  "ranged": true,
  "range": 6,
  "cost": 250,
  "usable_by": ["all"],
  "description": "A mechanical bow with superior accuracy. Grants 3 attack dice at range.",
  "card_image": "/assets/cards/equipment/crossbow.png"
}
```

### Armor Example

```json
{
  "id": "chainmail",
  "name": "Chainmail",
  "type": "armor",
  "subtype": "body",
  "slot": "body",
  "attack_dice": 0,
  "defense_dice": 1,
  "attack_bonus": 0,
  "defense_bonus": 1,
  "cost": 200,
  "usable_by": ["all"],
  "description": "Chain mail adds 1 die to your defense.",
  "card_image": "/assets/cards/equipment/chainmail.png"
}
```

### Helmet Example

```json
{
  "id": "helmet",
  "name": "Helmet",
  "type": "armor",
  "subtype": "helmet",
  "slot": "helmet",
  "attack_dice": 0,
  "defense_dice": 1,
  "attack_bonus": 0,
  "defense_bonus": 0,
  "cost": 100,
  "usable_by": ["all"],
  "description": "A sturdy helmet adds 1 die to your defense.",
  "card_image": "/assets/cards/equipment/helmet.png"
}
```

### Shield Example

```json
{
  "id": "shield",
  "name": "Shield",
  "type": "armor",
  "subtype": "shield",
  "slot": "shield",
  "attack_dice": 0,
  "defense_dice": 1,
  "attack_bonus": 0,
  "defense_bonus": 0,
  "cost": 50,
  "usable_by": ["all"],
  "description": "A wooden shield adds 1 die to your defense.",
  "card_image": "/assets/cards/equipment/shield.png"
}
```

### Potion Example (Equipment)

```json
{
  "id": "potion_dexterity",
  "name": "Potion of Dexterity",
  "type": "potion",
  "subtype": "movement",
  "effect": "bonus_movement",
  "effect_value": 5,
  "cost": 100,
  "usage_rules": ["single_use_per_turn"],
  "description": "This sparkling liquid adds 5 movement squares to your next dice roll or guarantees one successful pit jump. If you purchase more than one of these potions, you may only use one per turn.",
  "card_image": "/assets/cards/equipment/potion_dexterity.png"
}
```

### Field Definitions

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | string | Yes | Unique identifier (e.g., "longsword") |
| `name` | string | Yes | Display name |
| `category` | string | Yes | Parent category: "equipment" or "artifact" |
| `type` | string | Yes | Card type: "weapon", "armor", "potion", "item", "held" |
| `subtype` | string | No | Subtype: "sword", "axe", "bow", "body", "helmet", "shield", "staff", "dagger", "bracer", "cloak", "utility", "throwable" |
| `attack_dice` | int | No | Number of attack dice this item provides (default: 0) |
| `defense_dice` | int | No | Number of defense dice this item provides (default: 0) |
| `attack_bonus` | int | No | Bonus to attack (default: 0) |
| `defense_bonus` | int | No | Bonus to defense (default: 0) |
| `body_bonus` | int | No | Permanent Body Points increase (default: 0) |
| `mind_bonus` | int | No | Permanent Mind Points increase (default: 0) |
| `movement_bonus` | int | No | Movement modifier, can be negative (default: 0) |
| `attack_diagonal` | bool | No | Can attack diagonally (default: false) |
| `attack_adjacent` | bool | No | Can attack adjacent (default: true) |
| `ranged` | bool | No | Is ranged weapon (default: false) |
| `throwable` | bool | No | Can this weapon be thrown? (default: false) |
| `range` | int | No | Attack range in squares (default: 1) |
| `uses` | int | No | Number of uses before item is consumed (omit if unlimited) |
| `uses_per_quest` | int | No | Number of uses per quest for rechargeable abilities |
| `cost` | int | Yes | Gold cost to purchase |
| `usable_by` | string[] | Yes | Hero classes that can use: ["all"], ["barbarian", "dwarf"], etc. |
| `restrictions` | string[] | No | Equipment restrictions: ["no_shield", "1_movement_die"] |
| `effect` | object | No | Structured effect definition (see Effect Types section) |
| `usage_restrictions` | object | No | Usage limitation rules (see Usage Restrictions) |
| `description` | string | Yes | Card description text |
| `card_image` | string | Yes | Path to card image asset |

**Note**: The `slot` field has been deprecated in favor of deriving equipment slots from the `subtype` field.

---

## Treasure Schema

**Location**: `/content/treasures/*.json`

**Types**: gold, potion, monster, hazard

### Gold Example

```json
{
  "id": "gold_35",
  "name": "Gold Coins",
  "category": "treasure",
  "type": "gold",
  "value": 35,
  "description": "Tucked into the toe of an old boot you find a small gem worth 35 gold coins. Record the money on your character sheet. Do not return this card to the deck.",
  "return_to_deck": false,
  "card_image": "/assets/cards/treasures/gold_35.jpg"
}
```

### Potion Example (Treasure - Healing)

```json
{
  "id": "potion_of_healing",
  "name": "Potion of Healing",
  "category": "treasure",
  "type": "potion",
  "subtype": "healing",
  "uses": 1,
  "effect": {
    "type": "restore_points",
    "restore_body": "1d6"
  },
  "usage_restrictions": {
    "timing": "any_time"
  },
  "description": "In a bundle of rags, you find a small bluish liquid. You can drink this potion at any time, restoring the number of Body Points equal to the roll of 1d6. You cannot, however, exceed your starting number of Body Points. This may only be used once. Do not return this card to the deck.",
  "return_to_deck": false,
  "card_image": "/assets/cards/treasures/potion_of_healing.jpg"
}
```

### Potion Example (Treasure - Combat)

```json
{
  "id": "potion_of_strength",
  "name": "Potion of Strength",
  "category": "treasure",
  "type": "potion",
  "subtype": "combat",
  "uses": 1,
  "effect": {
    "type": "bonus_dice",
    "dice_type": "attack",
    "bonus_value": 2,
    "duration": "next_attack"
  },
  "usage_restrictions": {
    "timing": "any_time"
  },
  "description": "You find a small purple flask. You can drink this strange smelling liquid at any time, enabling you to roll 2 extra combat dice the next time you attack. Must be used before rolling attack dice. Do not return this card to the deck.",
  "return_to_deck": false,
  "card_image": "/assets/cards/treasures/potion_of_strength.jpg"
}
```

### Wandering Monster Example

```json
{
  "id": "wandering_monster",
  "name": "Wandering Monster",
  "category": "treasure",
  "type": "monster",
  "description": "A wandering monster appears! The monster spawns adjacent to you and immediately attacks. The specific monster type is defined by the quest. Return this card to the deck after use.",
  "return_to_deck": true,
  "card_image": "/assets/cards/treasures/wandering_monster.jpg"
}
```

### Hazard Example

```json
{
  "id": "hazard_shallow_hole",
  "name": "Shallow Hole",
  "category": "treasure",
  "type": "hazard",
  "subtype": "pit",
  "damage": 1,
  "end_turn": true,
  "description": "Suddenly, the stone beneath your feet gives way. You fall into a shallow hole, losing 1 Body Point and ending your turn. You may climb out and move normally on your next turn. Return this card to the bottom of the deck.",
  "return_to_deck": true,
  "card_image": "/assets/cards/treasures/hazard_shallow_hole.jpg"
}
```

### Field Definitions

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | string | Yes | Unique identifier |
| `name` | string | Yes | Display name |
| `category` | string | Yes | Always "treasure" |
| `type` | string | Yes | Card type: "gold", "potion", "monster", "hazard" |
| `subtype` | string | No | Subtype: "healing", "combat", "utility", "arrow", "pit" |
| `value` | int | No | Gold value (for type: "gold") |
| `uses` | int | No | Number of uses (for potions, typically 1) |
| `effect` | object | No | Structured effect definition (for potions - see Effect Types section) |
| `usage_restrictions` | object | No | Usage limitation rules (see Usage Restrictions) |
| `damage` | int | No | Damage dealt (for hazards) |
| `end_turn` | bool | No | Does this end the hero's turn? (for hazards) |
| `description` | string | Yes | Card description text |
| `return_to_deck` | bool | Yes | Return to deck after draw? |
| `card_image` | string | Yes | Path to card image asset |

**Note**: Treasure potions now use structured `effect` objects consistent with equipment/artifact schemas. The legacy `effect_value` and `usage_rules` fields have been replaced with `effect` and `usage_restrictions`.

---

## Artifact Schema

**Location**: `/content/artifacts/*.json`

**Types**: jewelry, artifact

### Ring Example

```json
{
  "id": "fire_ring",
  "name": "Ring of Fire",
  "type": "jewelry",
  "subtype": "ring",
  "slot": "ring",
  "effect": "fire_immunity",
  "cost": 500,
  "usable_by": ["all"],
  "description": "This magical ring protects the wearer from all fire-based attacks and spells.",
  "card_image": "/assets/cards/artifacts/fire_ring.png"
}
```

### Artifact Example

```json
{
  "id": "spirit_blade",
  "name": "Spirit Blade",
  "type": "artifact",
  "subtype": "weapon",
  "slot": "weapon",
  "attack_dice": 4,
  "defense_dice": 0,
  "attack_bonus": 0,
  "defense_bonus": 0,
  "attack_diagonal": true,
  "attack_adjacent": true,
  "ranged": false,
  "range": 1,
  "effect": "ignores_armor",
  "cost": 0,
  "usable_by": ["all"],
  "quest_reward": true,
  "description": "This ethereal blade passes through armor, ignoring all defense bonuses. Grants 4 attack dice.",
  "card_image": "/assets/cards/artifacts/spirit_blade.png"
}
```

### Field Definitions

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | string | Yes | Unique identifier |
| `name` | string | Yes | Display name |
| `category` | string | Yes | Always "artifact" |
| `type` | string | Yes | Card type: "jewelry", "weapon", "armor", "potion", "held" |
| `subtype` | string | No | Subtype: "ring", "amulet", "sword", "staff", "utility", "cloak" |
| `attack_dice` | int | No | Attack dice provided |
| `defense_dice` | int | No | Defense dice provided |
| `attack_bonus` | int | No | Attack bonus |
| `defense_bonus` | int | No | Defense bonus |
| `body_bonus` | int | No | Permanent Body Points increase (default: 0) |
| `mind_bonus` | int | No | Permanent Mind Points increase (default: 0) |
| `movement_bonus` | int | No | Movement modifier, can be negative (default: 0) |
| `attack_diagonal` | bool | No | Can attack diagonally |
| `attack_adjacent` | bool | No | Can attack adjacent |
| `ranged` | bool | No | Is ranged |
| `range` | int | No | Attack range |
| `uses` | int | No | Number of uses before item is consumed (omit if unlimited) |
| `uses_per_quest` | int | No | Number of uses per quest for rechargeable abilities |
| `cost` | int | No | Gold cost (0 if quest reward) |
| `usable_by` | string[] | Yes | Hero classes that can use |
| `quest_reward` | bool | No | Is this a quest reward item? |
| `effect` | object | No | Structured effect definition (see Effect Types section) |
| `description` | string | Yes | Card description text |
| `card_image` | string | Yes | Path to card image asset |

**Note**: The `slot` field has been deprecated in favor of deriving equipment slots from the `subtype` field.

---

## Spell Schema

**Location**: `/content/spells/*.json`

**Types**: spell

### Healing Spell Example

```json
{
  "id": "heal_body",
  "name": "Heal Body",
  "type": "spell",
  "category": "healing",
  "target": "self_or_ally",
  "range": 26,
  "effect": "restore_body_points",
  "effect_value": "1d6",
  "usable_by": ["wizard", "elf"],
  "description": "Restores Body Points equal to 1d6 to target hero within 26 squares. Cannot exceed starting Body Points.",
  "card_image": "/assets/cards/spells/heal_body.png"
}
```

### Attack Spell Example

```json
{
  "id": "ball_of_flame",
  "name": "Ball of Flame",
  "type": "spell",
  "category": "attack",
  "target": "enemy",
  "range": 6,
  "effect": "damage",
  "effect_value": "3d6",
  "attack_type": "fire",
  "usable_by": ["wizard", "elf"],
  "description": "Hurl a ball of flame at a monster within 6 squares. Roll 3 combat dice for attack. Fire damage ignores some defenses.",
  "card_image": "/assets/cards/spells/ball_of_flame.png"
}
```

### Utility Spell Example

```json
{
  "id": "pass_through_rock",
  "name": "Pass Through Rock",
  "type": "spell",
  "category": "utility",
  "target": "self",
  "range": 0,
  "effect": "ignore_walls",
  "duration": "1_turn",
  "usable_by": ["wizard", "elf"],
  "description": "You may pass through walls and furniture for the duration of this turn. You cannot end your movement inside a wall or obstacle.",
  "card_image": "/assets/cards/spells/pass_through_rock.png"
}
```

### Field Definitions

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | string | Yes | Unique identifier |
| `name` | string | Yes | Display name |
| `type` | string | Yes | Always "spell" |
| `category` | string | Yes | Spell category: "healing", "attack", "utility", "buff" |
| `target` | string | Yes | Target type: "self", "self_or_ally", "ally", "enemy", "area" |
| `range` | int | Yes | Range in squares |
| `effect` | string | Yes | Effect type: "restore_body_points", "damage", "ignore_walls", "bonus_dice" |
| `effect_value` | string/int | No | Effect magnitude: "1d6", "3d6", 5, etc. |
| `attack_type` | string | No | Damage type: "fire", "ice", "lightning", etc. |
| `duration` | string | No | Duration: "instant", "1_turn", "quest" |
| `usable_by` | string[] | Yes | Hero classes that can cast: ["wizard", "elf"] |
| `description` | string | Yes | Spell description |
| `card_image` | string | Yes | Path to card image asset |

---

## Dread Spell Schema

**Location**: `/content/dread_spells/*.json`

**Types**: dread_spell

### Dread Spell Example

```json
{
  "id": "rust",
  "name": "Rust",
  "type": "dread_spell",
  "category": "debuff",
  "target": "hero",
  "effect": "destroy_metal_items",
  "description": "All metal weapons and armor carried by the targeted hero are destroyed by rust. The hero may not use any metal equipment for the remainder of the quest.",
  "card_image": "/assets/cards/dread_spells/rust.png"
}
```

### Field Definitions

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | string | Yes | Unique identifier |
| `name` | string | Yes | Display name |
| `type` | string | Yes | Always "dread_spell" |
| `category` | string | Yes | Category: "debuff", "damage", "summon" |
| `target` | string | Yes | Target type: "hero", "all_heroes", "area" |
| `effect` | string | Yes | Effect type: "destroy_metal_items", "damage", etc. |
| `effect_value` | string/int | No | Effect magnitude if applicable |
| `description` | string | Yes | Spell description |
| `card_image` | string | Yes | Path to card image asset |

---

## Effect Types System

This section documents the structured effect system for equipment and artifacts with special mechanics beyond basic stat bonuses.

### Overview

The `effect` field uses a **Hybrid Approach**: base stats (attack_dice, defense_dice, etc.) remain top-level fields, while special mechanics are defined in a typed `effect` object. This provides both simplicity and flexibility.

### Effect Type Summary

| Effect Type | Used For | Example Items |
|-------------|----------|---------------|
| `protection` | Block specific damage/spell types | Fire Ring |
| `revival` | Bring dead hero back to life | Elixir of Life |
| `instant_kill` | Kill target without combat | Holy Water |
| `guaranteed_damage` | Deal damage without defense roll | Magical Throwing Dagger |
| `multi_choice` | Player chooses between options | Potion of Dexterity |
| `restore_points` | Restore Body/Mind points | Potion of Restoration, Potion of Healing (Treasure) |
| `movement_multiplier` | Multiply movement dice | Potion of Speed |
| `reroll_dice` | Reroll attack/defense dice | Potion of Battle |
| `dual_mode_attack` | Choose melee or thrown | Dagger, Handaxe |
| `conditional_bonus` | Bonus based on condition | Spirit Blade, Orcs Bane |
| `special_ability` | Once-per-quest triggered ability | Phantom Blade, Fortunes Longsword |
| `placement` | Place tile/hazard on board | Caltrops |
| `area_effect` | Temporary area effect | Smoke Bomb |
| `utility` | Skill check enhancement | Tool Kit |
| `spell_modifier` | Modify spell casting | Spell Ring, Wand of Magic |
| `party_effect` | Affect multiple heroes | Ring of Return |
| `bonus_dice` | Temporary dice bonus | Potion of Strength, Potion of Defense (Treasure) |
| `extra_action` | Grant additional actions | Heroic Brew (Treasure) |
| `movement_immunity` | Immunity to movement hazards | Air Walk (Treasure) |

---

### Protection Effect

**Type**: `"protection"`

Protects from specific damage or spell types for a limited number of uses.

**Fields**:
- `protection_type` (string, required): Type of protection ("dread_spell", "elemental", "physical")
- `spell_types` (string[], optional): Specific spell types blocked (e.g., ["fire"])
- `charges` (int, required): Number of uses
- `consumed_on_depletion` (bool, required): Does item disappear when charges run out?

**Example** (Fire Ring):
```json
{
  "uses": 2,
  "effect": {
    "type": "protection",
    "protection_type": "dread_spell",
    "spell_types": ["fire"],
    "charges": 2,
    "consumed_on_depletion": true
  }
}
```

---

### Revival Effect

**Type**: `"revival"`

Brings a dead hero back to life.

**Fields**:
- `target` (string, required): Must be "dead_hero"
- `restore_body` (string, required): "full" or numeric value
- `restore_mind` (string, required): "full" or numeric value

**Example** (Elixir of Life):
```json
{
  "uses": 1,
  "effect": {
    "type": "revival",
    "target": "dead_hero",
    "restore_body": "full",
    "restore_mind": "full"
  }
}
```

---

### Instant Kill Effect

**Type**: `"instant_kill"`

Instantly kills target creatures without combat roll.

**Fields**:
- `target_types` (string[], required): Monster types affected (e.g., ["skeleton", "zombie", "mummy"])
- `replaces_attack` (bool, required): Does this use your attack action?

**Example** (Holy Water):
```json
{
  "uses": 1,
  "effect": {
    "type": "instant_kill",
    "target_types": ["skeleton", "zombie", "mummy"],
    "replaces_attack": true
  }
}
```

---

### Guaranteed Damage Effect

**Type**: `"guaranteed_damage"`

Deals damage that cannot be defended against.

**Fields**:
- `damage` (int, required): Amount of damage dealt
- `no_defense` (bool, required): Cannot be defended
- `range` (string, required): "line_of_sight" or numeric
- `consumable` (bool, required): Is item lost after use?

**Example** (Magical Throwing Dagger):
```json
{
  "uses": 1,
  "effect": {
    "type": "guaranteed_damage",
    "damage": 1,
    "no_defense": true,
    "range": "line_of_sight",
    "consumable": true
  }
}
```

---

### Multi-Choice Effect

**Type**: `"multi_choice"`

Player chooses one option from multiple effects.

**Fields**:
- `choose_one` (bool, required): Must choose exactly one option
- `options` (array, required): Array of option objects

**Option Object Fields**:
- `id` (string, required): Unique identifier for this option
- `name` (string, required): Display name
- `effect_type` (string, required): Type of effect ("bonus_movement", "auto_pit_success", etc.)
- `value` (int, optional): Numeric value if applicable
- `description` (string, required): What this option does

**Example** (Potion of Dexterity):
```json
{
  "uses": 1,
  "usage_restrictions": {
    "max_per_turn": 1
  },
  "effect": {
    "type": "multi_choice",
    "choose_one": true,
    "options": [
      {
        "id": "movement_boost",
        "name": "Movement Boost",
        "effect_type": "bonus_movement",
        "value": 5,
        "description": "Add 5 movement squares to your next dice roll"
      },
      {
        "id": "pit_jump",
        "name": "Guaranteed Pit Jump",
        "effect_type": "auto_pit_success",
        "description": "Automatically succeed on one pit jump"
      }
    ]
  }
}
```

---

### Restore Points Effect

**Type**: `"restore_points"`

Restores Body and/or Mind points.

**Fields**:
- `restore_body` (int, optional): Body Points restored
- `restore_mind` (int, optional): Mind Points restored

**Example** (Potion of Restoration):
```json
{
  "uses": 1,
  "effect": {
    "type": "restore_points",
    "restore_body": 1,
    "restore_mind": 1
  }
}
```

---

### Movement Multiplier Effect

**Type**: `"movement_multiplier"`

Multiplies movement dice rolled.

**Fields**:
- `multiplier` (int, required): Multiply movement dice by this amount
- `duration` (string, required): "next_movement", "1_turn", "quest"

**Example** (Potion of Speed):
```json
{
  "uses": 1,
  "effect": {
    "type": "movement_multiplier",
    "multiplier": 2,
    "duration": "next_movement"
  }
}
```

---

### Reroll Dice Effect

**Type**: `"reroll_dice"`

Allows rerolling attack or defense dice.

**Fields**:
- `dice_type` (string, required): "attack" or "defense"
- `reroll_count` (string/int, required): "all" or specific number
- `timing` (string, required): "after_roll", "before_roll"

**Example** (Potion of Battle):
```json
{
  "uses": 1,
  "effect": {
    "type": "reroll_dice",
    "dice_type": "attack",
    "reroll_count": "all",
    "timing": "after_roll"
  }
}
```

---

### Dual Mode Attack Effect

**Type**: `"dual_mode_attack"`

Weapon can be used in melee or thrown (consuming it).

**Fields**:
- `modes` (array, required): Array of mode objects

**Mode Object Fields**:
- `mode` (string, required): "melee" or "thrown"
- `attack_dice` (int, required): Dice for this mode
- `range` (int, required): Attack range
- `attack_adjacent` (bool, optional): For melee mode
- `attack_diagonal` (bool, optional): For melee mode
- `ranged` (bool, optional): For thrown mode
- `consumable` (bool, optional): Is weapon lost? (thrown mode)
- `description` (string, optional): Additional notes

**Example** (Dagger):
```json
{
  "effect": {
    "type": "dual_mode_attack",
    "modes": [
      {
        "mode": "melee",
        "attack_dice": 1,
        "range": 1,
        "attack_adjacent": true,
        "attack_diagonal": false
      },
      {
        "mode": "thrown",
        "attack_dice": 1,
        "range": 26,
        "ranged": true,
        "consumable": true,
        "description": "Lost once thrown"
      }
    ]
  }
}
```

---

### Conditional Bonus Effect

**Type**: `"conditional_bonus"`

Provides bonus when specific conditions are met.

**Fields**:
- `trigger` (string, required): Condition type ("target_type", "situation")
- `target_types` (string[], optional): Monster types triggering bonus (e.g., ["skeleton", "zombie", "mummy"])
- `bonus_type` (string, required): Type of bonus ("attack_dice", "defense_dice", "extra_attack")
- `bonus_value` (int, optional): Amount of bonus (if numeric)
- `description` (string, required): Human-readable explanation

**Example** (Spirit Blade - bonus vs undead):
```json
{
  "attack_dice": 3,
  "effect": {
    "type": "conditional_bonus",
    "trigger": "target_type",
    "target_types": ["skeleton", "zombie", "mummy"],
    "bonus_type": "attack_dice",
    "bonus_value": 1,
    "description": "Roll 4 Attack dice when attacking undead"
  }
}
```

**Example** (Orcs Bane - double attack vs orcs):
```json
{
  "effect": {
    "type": "conditional_bonus",
    "trigger": "target_type",
    "target_types": ["orc"],
    "bonus_type": "extra_attack",
    "description": "May attack twice when attacking an orc"
  }
}
```

---

### Special Ability Effect

**Type**: `"special_ability"`

Once-per-quest or limited-use special power.

**Fields**:
- `id` (string, required): Unique ability identifier
- `name` (string, required): Ability name
- `uses_per_quest` (int, optional): Uses per quest (omit for unlimited)
- `trigger` (string, required): When it activates ("on_attack", "on_defend", "active_use")
- `ability_effect` (string, required): What it does ("ignore_defense", "reroll_dice", "skip_turn")
- `target` (string, optional): Who/what is affected ("self", "monster", "hero")
- `ability_details` (object, optional): Additional details specific to ability
- `resistance_check` (object, optional): If target can resist (see Resistance Check format)
- `description` (string, required): What the ability does

**Example** (Phantom Blade):
```json
{
  "effect": {
    "type": "special_ability",
    "id": "phantom_strike",
    "name": "Phantom Strike",
    "uses_per_quest": 1,
    "trigger": "on_attack",
    "ability_effect": "ignore_defense",
    "description": "Target may not defend as weapon passes through armor"
  }
}
```

**Example** (Fortunes Longsword):
```json
{
  "effect": {
    "type": "special_ability",
    "id": "fortune_reroll",
    "name": "Fortune's Favor",
    "uses_per_quest": 1,
    "trigger": "on_attack",
    "ability_effect": "reroll_dice",
    "ability_details": {
      "dice_type": "attack",
      "reroll_count": 1
    },
    "description": "Reroll 1 Attack die"
  }
}
```

**Example** (Rod of Telekinesis - with resistance):
```json
{
  "uses_per_quest": 1,
  "effect": {
    "type": "special_ability",
    "id": "force_trap",
    "name": "Magical Force Trap",
    "trigger": "active_use",
    "ability_effect": "skip_turn",
    "target": "monster",
    "resistance_check": {
      "stat": "mind_points",
      "dice_per_point": 1,
      "die_type": "red",
      "success_on": [6],
      "description": "Monster rolls 1 red die per Mind Point; 6 resists"
    },
    "description": "Trapped monster misses next turn unless it resists"
  }
}
```

---

### Placement Effect

**Type**: `"placement"`

Places a tile or object on the board with an effect.

**Fields**:
- `placement_type` (string, required): "hazard_tile", "marker"
- `timing` (string, required): When it can be placed ("during_movement", "any_time", "on_turn")
- `action_cost` (string, required): "free", "action", "movement"
- `tile_effect` (object, required): Effect when triggered

**Tile Effect Fields**:
- `trigger` (string, required): "creature_enters", "creature_ends_turn"
- `check` (object, optional): Roll check object
- `removal` (string, required): When tile is removed

**Check Object Fields**:
- `dice` (int, required): Number of dice
- `die_type` (string, required): "combat", "red", "white"
- `success_on` (array, required): What results count as success
- `on_success` (string, required): What happens on success
- `on_failure` (string, required): What happens on failure

**Example** (Caltrops):
```json
{
  "uses": 1,
  "effect": {
    "type": "placement",
    "placement_type": "hazard_tile",
    "timing": "during_movement",
    "action_cost": "free",
    "tile_effect": {
      "trigger": "creature_enters",
      "check": {
        "dice": 1,
        "die_type": "combat",
        "success_on": ["white_shield"],
        "on_success": "continue_movement",
        "on_failure": "end_movement"
      },
      "removal": "creature_ends_turn_on_tile"
    }
  }
}
```

---

### Area Effect

**Type**: `"area_effect"`

Creates a temporary area-based effect.

**Fields**:
- `area` (string, required): "adjacent_monster", "radius", "line"
- `timing` (string, required): When it can be used
- `duration` (string, required): How long it lasts
- `effect_type` (string, required): What it does ("pass_through", "block_sight", "damage")
- `description` (string, required): Effect description

**Example** (Smoke Bomb):
```json
{
  "uses": 1,
  "effect": {
    "type": "area_effect",
    "area": "adjacent_monster",
    "timing": "during_movement",
    "duration": "until_monster_next_turn",
    "effect_type": "pass_through",
    "description": "Heroes may move through affected monster's space"
  }
}
```

---

### Utility Effect

**Type**: `"utility"`

Provides utility function like trap disarming or lock picking.

**Fields**:
- `utility_type` (string, required): "trap_disarm", "lock_pick", "detect"
- `trigger` (string, required): When it can be used ("trap_discovered", "on_door", "any_time")
- `check` (object, optional): Success check (see Check Object format)
- `reusable` (bool, required): Can be used multiple times?

**Example** (Tool Kit):
```json
{
  "effect": {
    "type": "utility",
    "utility_type": "trap_disarm",
    "trigger": "trap_discovered",
    "check": {
      "dice": 1,
      "die_type": "combat",
      "failure_on": ["skull"],
      "success_chance": "5/6"
    },
    "reusable": true
  }
}
```

---

### Spell Modifier Effect

**Type**: `"spell_modifier"`

Modifies spell casting abilities.

**Fields**:
- `modifier_type` (string, required): "duplicate_spell", "multi_cast", "extra_range", "extra_power"
- `spells_per_turn` (int, optional): For multi_cast
- `restriction` (string, optional): "different_spells", "same_spell"
- `uses_per_quest` (int, optional): For duplicate_spell
- `requires_declaration` (bool, optional): Must declare at quest start?
- `declaration_timing` (string, optional): "quest_start", "before_use"
- `description` (string, required): How it modifies casting

**Example** (Spell Ring - duplicate spell):
```json
{
  "effect": {
    "type": "spell_modifier",
    "modifier_type": "duplicate_spell",
    "uses_per_quest": 1,
    "requires_declaration": true,
    "declaration_timing": "quest_start",
    "description": "Cast chosen spell twice per quest; declare at quest start"
  }
}
```

**Example** (Wand of Magic - multi cast):
```json
{
  "effect": {
    "type": "spell_modifier",
    "modifier_type": "multi_cast",
    "spells_per_turn": 2,
    "restriction": "different_spells",
    "description": "Cast two different spells per turn"
  }
}
```

---

### Party Effect

**Type**: `"party_effect"`

Affects multiple heroes at once.

**Fields**:
- `effect_type` (string, required): "teleport", "buff", "heal"
- `target` (string, required): "visible_heroes", "all_heroes", "adjacent_heroes"
- `destination` (string, optional): For teleport ("quest_start", "coordinates")
- `range` (string, optional): "line_of_sight", "unlimited"
- `description` (string, required): What happens

**Example** (Ring of Return):
```json
{
  "uses": 1,
  "effect": {
    "type": "party_effect",
    "effect_type": "teleport",
    "target": "visible_heroes",
    "destination": "quest_start",
    "range": "line_of_sight",
    "description": "Returns all visible heroes to quest starting point"
  }
}
```

---

### Bonus Dice Effect

**Type**: `"bonus_dice"`

Temporarily adds dice to attack or defense for the next roll.

**Fields**:
- `dice_type` (string, required): "attack" or "defense"
- `bonus_value` (int, required): Number of extra dice
- `duration` (string, required): "next_attack", "next_defense", "1_turn"

**Example** (Potion of Strength - Treasure):
```json
{
  "category": "treasure",
  "type": "potion",
  "uses": 1,
  "effect": {
    "type": "bonus_dice",
    "dice_type": "attack",
    "bonus_value": 2,
    "duration": "next_attack"
  },
  "usage_restrictions": {
    "timing": "any_time"
  }
}
```

**Example** (Potion of Defense - Treasure):
```json
{
  "category": "treasure",
  "type": "potion",
  "uses": 1,
  "effect": {
    "type": "bonus_dice",
    "dice_type": "defense",
    "bonus_value": 2,
    "duration": "next_defense"
  },
  "usage_restrictions": {
    "timing": "any_time"
  }
}
```

---

### Extra Action Effect

**Type**: `"extra_action"`

Grants additional actions during a turn.

**Fields**:
- `action_type` (string, required): "attack", "movement", "search", "any"
- `count` (int, required): Number of extra actions
- `duration` (string, required): "this_turn", "next_turn", "quest"

**Example** (Heroic Brew - Treasure):
```json
{
  "category": "treasure",
  "type": "potion",
  "uses": 1,
  "effect": {
    "type": "extra_action",
    "action_type": "attack",
    "count": 1,
    "duration": "this_turn"
  },
  "usage_restrictions": {
    "timing": "before_attack"
  }
}
```

---

### Movement Immunity Effect

**Type**: `"movement_immunity"`

Provides immunity to hazards during movement.

**Fields**:
- `immunity_types` (string[], required): Types of hazards ignored (e.g., ["trap", "pit"])
- `duration` (string, required): "1_turn", "quest", "permanent"

**Example** (Air Walk - Treasure):
```json
{
  "category": "treasure",
  "type": "potion",
  "uses": 1,
  "effect": {
    "type": "movement_immunity",
    "immunity_types": ["trap", "pit"],
    "duration": "1_turn"
  },
  "usage_restrictions": {
    "timing": "before_movement"
  }
}
```

---

### Resistance Check Format

Used within special abilities to allow targets to resist effects.

**Fields**:
- `stat` (string, required): Stat used for check ("mind_points", "body_points")
- `dice_per_point` (int, required): How many dice per stat point
- `die_type` (string, required): "red", "white", "combat"
- `success_on` (array, required): Which roll results count as success (e.g., [6])
- `description` (string, required): Explanation of the check

**Example**:
```json
{
  "resistance_check": {
    "stat": "mind_points",
    "dice_per_point": 1,
    "die_type": "red",
    "success_on": [6],
    "description": "Monster rolls 1 red die per Mind Point; 6 resists"
  }
}
```

---

### Usage Restrictions Format

Defines limits on when/how items can be used.

**Fields**:
- `max_per_turn` (int, optional): Maximum uses per turn
- `timing` (string, optional): "before_movement", "during_combat", "any_time"
- `requires_action` (bool, optional): Does it consume an action?

**Example** (Potion of Dexterity restriction):
```json
{
  "usage_restrictions": {
    "max_per_turn": 1
  }
}
```

---

## Campaign Deck Schemas

### Equipment Deck

**Location**: `/content/campaigns/{campaign}/equipment_deck.json`

```json
{
  "campaign": "base",
  "deck_type": "equipment",
  "items": [
    { "id": "longsword", "path": "/content/equipment/longsword.json" },
    { "id": "crossbow", "path": "/content/equipment/crossbow.json" },
    { "id": "battle_axe", "path": "/content/equipment/battle_axe.json" },
    { "id": "shield", "path": "/content/equipment/shield.json" },
    { "id": "chainmail", "path": "/content/equipment/chainmail.json" },
    { "id": "helmet", "path": "/content/equipment/helmet.json" },
    { "id": "potion_dexterity", "path": "/content/equipment/potion_dexterity.json" }
  ]
}
```

### Artifact Deck

**Location**: `/content/campaigns/{campaign}/artifact_deck.json`

```json
{
  "campaign": "base",
  "deck_type": "artifact",
  "items": [
    { "id": "fire_ring", "path": "/content/artifacts/fire_ring.json" },
    { "id": "spirit_blade", "path": "/content/artifacts/spirit_blade.json" }
  ]
}
```

### Treasure Deck

**Location**: `/content/campaigns/{campaign}/treasure_deck.json`

```json
{
  "campaign": "base",
  "deck_type": "treasure",
  "shuffle_on_load": true,
  "cards": [
    { "id": "gold_15", "path": "/content/treasures/gold_15.json", "count": 1 },
    { "id": "gold_25", "path": "/content/treasures/gold_25.json", "count": 1 },
    { "id": "gold_35", "path": "/content/treasures/gold_35.json", "count": 1 },
    { "id": "gold_50", "path": "/content/treasures/gold_50.json", "count": 1 },
    { "id": "gold_300", "path": "/content/treasures/gold_300.json", "count": 1 },
    { "id": "potion_healing", "path": "/content/treasures/potion_healing.json", "count": 1 },
    { "id": "wandering_monster", "path": "/content/treasures/wandering_monster.json", "count": 6 },
    { "id": "hazard_arrow", "path": "/content/treasures/hazard_arrow.json", "count": 3 },
    { "id": "hazard_pit", "path": "/content/treasures/hazard_pit.json", "count": 3 }
  ]
}
```

**Notes**:
- `count` field allows multiple copies of the same card in the deck
- `shuffle_on_load` determines if deck is randomized when loaded

### Spell Deck

**Location**: `/content/campaigns/{campaign}/spell_deck.json`

```json
{
  "campaign": "base",
  "deck_type": "spell",
  "spells": [
    { "id": "heal_body", "path": "/content/spells/heal_body.json" },
    { "id": "heal_mind", "path": "/content/spells/heal_mind.json" },
    { "id": "ball_of_flame", "path": "/content/spells/ball_of_flame.json" },
    { "id": "swift_wind", "path": "/content/spells/swift_wind.json" },
    { "id": "pass_through_rock", "path": "/content/spells/pass_through_rock.json" },
    { "id": "courage", "path": "/content/spells/courage.json" }
  ]
}
```

### Campaign Metadata

**Location**: `/content/campaigns/{campaign}/campaign.json`

```json
{
  "id": "base",
  "name": "HeroQuest Base Game",
  "description": "The original HeroQuest campaign with 14 quests",
  "version": "1.0.0",
  "decks": {
    "equipment": "equipment_deck.json",
    "artifacts": "artifact_deck.json",
    "treasures": "treasure_deck.json",
    "spells": "spell_deck.json",
    "dread_spells": "dread_spell_deck.json"
  },
  "content_paths": {
    "monsters": "monsters.json",
    "heroes": "heroes.json",
    "quests_dir": "quests/"
  },
  "quests": [
    { "id": "quest-01", "path": "quests/quest-01.json", "order": 1 },
    { "id": "quest-02", "path": "quests/quest-02.json", "order": 2 },
    { "id": "quest-03", "path": "quests/quest-03.json", "order": 3 },
    { "id": "quest-04", "path": "quests/quest-04.json", "order": 4 },
    { "id": "quest-05", "path": "quests/quest-05.json", "order": 5 },
    { "id": "quest-06", "path": "quests/quest-06.json", "order": 6 },
    { "id": "quest-07", "path": "quests/quest-07.json", "order": 7 },
    { "id": "quest-08", "path": "quests/quest-08.json", "order": 8 },
    { "id": "quest-09", "path": "quests/quest-09.json", "order": 9 },
    { "id": "quest-10", "path": "quests/quest-10.json", "order": 10 },
    { "id": "quest-11", "path": "quests/quest-11.json", "order": 11 },
    { "id": "quest-12", "path": "quests/quest-12.json", "order": 12 },
    { "id": "quest-13", "path": "quests/quest-13.json", "order": 13 },
    { "id": "quest-14", "path": "quests/quest-14.json", "order": 14 }
  ]
}
```

---

## Quest Schema Extensions

### Quest Notes Section

Add to existing quest JSON structure:

```json
{
  "quest_notes": {
    "A": {
      "note_id": "A",
      "location": {
        "room": 18,
        "furniture_id": "furniture-4",
        "x": 11,
        "y": 15
      },
      "description": "The weapons on this weapons rack are not cursed and not broken. There are 2 weapons here.",
      "note_type": "treasure",
      "treasure": {
        "type": "fixed",
        "items": [
          { "id": "crossbow", "path": "/content/equipment/crossbow.json" },
          { "id": "battle_axe", "path": "/content/equipment/battle_axe.json" }
        ]
      },
      "consumed_for_party": true
    },
    "B": {
      "note_id": "B",
      "location": {
        "room": 19,
        "furniture_id": "furniture-6",
        "x": 17,
        "y": 16
      },
      "description": "The first hero who searches for treasure finds 84 gold coins in this treasure chest.",
      "note_type": "treasure",
      "treasure": {
        "type": "fixed",
        "gold": 84
      },
      "consumed_for_party": true
    },
    "C": {
      "note_id": "C",
      "location": {
        "room": 3,
        "furniture_id": "furniture-12",
        "x": 10,
        "y": 5
      },
      "description": "This treasure chest is empty.",
      "note_type": "treasure",
      "treasure": {
        "type": "empty"
      },
      "consumed_for_party": true
    },
    "D": {
      "note_id": "D",
      "location": {
        "room": 2,
        "monster_id": "monster-12",
        "x": 6,
        "y": 2
      },
      "description": "The guardian of Fellmarg's tomb was once a mighty warrior. It rolls 4 Attack dice instead of 3.",
      "note_type": "monster_modifier",
      "monster_modifier": {
        "monster_id": "monster-12",
        "attack_dice_bonus": 1,
        "defense_dice_bonus": 0
      }
    }
  }
}
```

### Quest Notes Field Definitions

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `note_id` | string | Yes | Note identifier (A, B, C, etc.) |
| `location.room` | int | Yes | Room number |
| `location.furniture_id` | string | No | Associated furniture ID |
| `location.monster_id` | string | No | Associated monster ID |
| `location.x` | int | Yes | Tile X coordinate |
| `location.y` | int | Yes | Tile Y coordinate |
| `description` | string | Yes | GM narration text |
| `note_type` | string | Yes | Note type: "treasure", "monster_modifier", "trap", "rule" |
| `treasure.type` | string | No | Treasure type: "fixed", "empty" |
| `treasure.items` | array | No | Fixed item rewards |
| `treasure.gold` | int | No | Fixed gold amount |
| `monster_modifier` | object | No | Monster stat modifications |
| `consumed_for_party` | bool | Yes | Is this note consumed for all heroes? |

---

## Type Constants

### Common Enums

**Hero Classes**:
- `barbarian`
- `dwarf`
- `elf`
- `wizard`
- `berserker`
- `knight`
- `explorer`
- `druid`
- `monk`
- `all` (special: usable by all classes)

**Item Types**:
- `weapon`
- `armor`
- `potion`
- `jewelry`
- `artifact`
- `spell`
- `gold`
- `monster`
- `hazard`

**Equipment Slots**:
- `weapon`
- `body` (armor)
- `helmet`
- `shield`
- `gloves`
- `boots`
- `ring`

**Effect Types**:
- `restore_body_points`
- `restore_mind_points`
- `bonus_movement`
- `bonus_attack_dice`
- `bonus_defense_dice`
- `fire_immunity`
- `ignores_armor`
- `ignore_walls`
- `damage`

**Equipment Restrictions**:
- `no_shield` (cannot use with shield)
- `1_movement_die` (can only roll 1 movement die)
- `no_diagonal_attack` (cannot attack diagonally)

**Usage Timing**:
- `any_time` (can be used any time)
- `before_movement` (must use before moving)
- `during_movement` (used while moving)
- `during_combat` (only during combat)
- `on_turn` (during your turn)

---

## Validation Rules

### All Cards
- `id` must be unique across all cards of the same type
- `id` should use lowercase with underscores (e.g., `gold_35`, `potion_healing`)
- `card_image` path must exist in `/assets/cards/`

### Equipment
- If `type` is "weapon", must specify `attack_dice` or `attack_bonus`
- If `type` is "armor", must specify `defense_dice` or `defense_bonus`
- `ranged` weapons must have `range > 1`
- `usable_by` must contain valid hero class names or "all"

### Treasures
- If `type` is "gold", must specify `value`
- If `type` is "potion", must specify `effect` and `effect_value`
- If `type` is "hazard", must specify `damage` or `special_mechanic`
- `return_to_deck` must be explicitly true or false

### Campaign Decks
- All referenced `path` values must point to existing files
- `count` must be >= 1
- Treasure deck should include at least 1 wandering monster and 1 hazard

---

## Migration Notes

When migrating existing content:

1. **Move files**: Relocate from `/content/base/*` to `/content/*`
2. **Update paths**: Ensure all `path` references in deck files are correct
3. **Add missing fields**: Add `card_image` paths and any new required fields
4. **Validate**: Run validation script to ensure all schemas are correct
5. **Update quest files**: Add `quest_notes` sections to all quest JSON files

---

## Change Log

**2025-10-13**: Added comprehensive Effect Types System
- Added `category`, `body_bonus`, `mind_bonus`, `movement_bonus`, `uses`, `uses_per_quest`, `throwable`, and `restrictions` fields
- Deprecated `slot` field in favor of deriving slots from `subtype`
- Added 19 effect types with detailed documentation:
  - protection, revival, instant_kill, guaranteed_damage
  - multi_choice, restore_points, movement_multiplier, reroll_dice
  - dual_mode_attack, conditional_bonus, special_ability
  - placement, area_effect, utility, spell_modifier, party_effect
  - bonus_dice, extra_action, movement_immunity (for treasure cards)
- Added Resistance Check and Usage Restrictions formats
- Updated Equipment and Artifact field definitions
- Added treasure-specific effect types for potions and utility items

---

*This schema document should be updated as new card types and fields are added.*
