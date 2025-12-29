# Content Effects Implementation Plan

## Overview

This document provides a phased approach to updating all equipment and artifact content files with proper effect structures based on the Hybrid Approach defined in SPECIAL_EFFECTS_SYSTEM.md.

---

## Phase 1: Schema Update

### 1.1 Update CONTENT_JSON_SCHEMAS.md

Add the following field definitions to the Equipment and Artifact schemas:

#### New Top-Level Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `body_bonus` | int | No | Permanent Body Points increase (default: 0) |
| `mind_bonus` | int | No | Permanent Mind Points increase (default: 0) |
| `movement_bonus` | int | No | Movement modifier, can be negative (default: 0) |
| `uses` | int | No | Number of uses before item is consumed (omit if unlimited) |
| `throwable` | bool | No | Can this weapon be thrown? (default: false) |
| `effect` | object | No | Structured effect definition for special mechanics |
| `restrictions` | string[] | No | Equipment restrictions: ["no_shield", "1_movement_die"] |

#### Effect Object Structure

```json
{
  "effect": {
    "type": "effect_type_identifier",
    // Additional fields vary by effect type
  }
}
```

### 1.2 Define Effect Types

Document these effect type structures in the schema:

#### Simple Stat Bonuses
```json
{
  "effect": {
    "type": "stat_bonus",
    "stat": "body_points" | "mind_points",
    "value": 1
  }
}
```

#### Protection
```json
{
  "effect": {
    "type": "protection",
    "protection_type": "dread_spell",
    "spell_types": ["fire"],
    "charges": 2,
    "consumed_on_depletion": true
  }
}
```

#### Revival
```json
{
  "effect": {
    "type": "revival",
    "target": "dead_hero",
    "restore_body": "full",
    "restore_mind": "full"
  }
}
```

#### Multi-Choice
```json
{
  "effect": {
    "type": "multi_choice",
    "choose_one": true,
    "options": [
      {
        "id": "option_1",
        "name": "Option Name",
        "effect_type": "bonus_movement",
        "value": 5,
        "description": "Description of what this option does"
      }
    ]
  }
}
```

#### Conditional Bonus
```json
{
  "effect": {
    "type": "conditional_bonus",
    "trigger": "target_type",
    "target_types": ["skeleton", "zombie", "mummy"],
    "bonus_type": "attack_dice",
    "bonus_value": 1
  }
}
```

#### Special Ability
```json
{
  "effect": {
    "type": "special_ability",
    "id": "ability_id",
    "name": "Ability Name",
    "uses_per_quest": 1,
    "trigger": "on_attack" | "on_defend" | "active_use",
    "ability_effect": "ignore_defense" | "reroll_dice" | "skip_turn",
    "ability_details": {
      // Varies by ability type
    }
  }
}
```

#### Attack Mode Selection
```json
{
  "effect": {
    "type": "dual_mode_attack",
    "modes": [
      {
        "mode": "melee",
        "attack_dice": 1,
        "range": 1
      },
      {
        "mode": "thrown",
        "attack_dice": 1,
        "range": 26,
        "consumable": true
      }
    ]
  }
}
```

#### Spell Modifier
```json
{
  "effect": {
    "type": "spell_modifier",
    "modifier_type": "extra_cast" | "multi_cast",
    "details": {
      // Varies by modifier type
    }
  }
}
```

---

## Phase 2: Simple Updates (No Complex Effects)

Update items that only need basic field additions or cleanup.

### 2.1 Items with Stat Bonuses Only

**Ring of Fortitude** (internal/artifacts/ring_of_fortitude.json:9)
- Already has: `"body_bonus": 1`
- Action: None needed, already correct

**Talisman of Lore** (internal/artifacts/talisman_of_lore.json:7)
- Already has: `"mind_bonus": 1`
- Action: None needed, already correct

**Borins Armor** (internal/artifacts/borins_armor.json:13)
- Already has: `"movement_bonus": 0`
- Action: None needed, already correct

### 2.2 Items with Standard Equipment Stats

**Bracers** (internal/equipment/bracers.json)
- Action: None needed, standard defense item

**Chain Mail** (internal/equipment/chain_mail.json)
- Action: None needed, standard defense item

**Helmet** (internal/equipment/helmet.json)
- Action: None needed, standard defense item

**Shield** (internal/equipment/shield.json)
- Action: None needed, standard defense item

**Broadsword** (internal/equipment/broadsword.json)
- Action: None needed, standard weapon

**Shortsword** (internal/equipment/shortsword.json)
- Action: None needed, standard weapon

**Longsword** (internal/equipment/longsword.json)
- Action: None needed, standard weapon

**Crossbow** (internal/equipment/crossbow.json)
- Action: None needed, standard ranged weapon

**Staff** (internal/equipment/staff.json)
- Action: None needed, standard weapon with restriction

**Wizards Staff** (internal/artifacts/wizards_staff.json)
- Action: None needed, standard weapon

### 2.3 Items Needing Restriction Field

**Battle Axe** (internal/equipment/battle_axe.json:25)
- Already has: `"restrictions": ["no_shield"]`
- Action: None needed

**Platemail** (internal/equipment/platemail.json:14)
- Already has: `"restrictions": ["1_movement_die"]`
- Action: None needed

---

## Phase 3: Items with Special Effects

### 3.1 Protection Items

**Fire Ring** (internal/artifacts/fire_ring.json)
- Current: `"effect": "dread_fire_protection"`, `"uses": 2`
- Update to:
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

### 3.2 Revival Items

**Elixir of Life** (internal/artifacts/elixir_of_life.json)
- Current: `"effect": "revive"`, `"uses": 1`
- Update to:
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

### 3.3 Consumable Combat Items

**Holy Water** (internal/equipment/holy_water.json)
- Current: `"uses": 1`
- Update to:
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

**Magical Throwing Dagger** (internal/artifacts/magical_throwing_dagger.json)
- Current: `"uses": 1`, `"damage": 1`
- Update to:
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

## Phase 4: Multi-Choice and Dual-Mode Items

### 4.1 Multi-Choice Potions

**Potion of Dexterity** (internal/equipment/potion_of_dexterity.json)
- Current: `"movement_bonus": 5`, `"effects": ["movement_bonus_5", "pit_jump"]`
- Update to:
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

**Potion of Restoration** (internal/equipment/potion_of_restoration.json)
- Current: `"effects": ["body_mind_restore_1"]`
- Update to:
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

**Potion of Speed** (internal/equipment/potion_of_speed.json)
- Current: `"effects": ["2x_movement_dice"]`
- Update to:
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

**Potion of Battle** (internal/equipment/potion_of_battle.json)
- Current: `"attack_reroll": 1`
- Update to:
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

### 4.2 Throwable Weapons

**Dagger** (internal/equipment/dagger.json)
- Current: `"throwable": true`
- Update to:
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

**Handaxe** (internal/equipment/handaxe.json)
- Current: `"throwable": true`
- Update to: Same structure as Dagger but with `"attack_dice": 2`

---

## Phase 5: Conditional and Triggered Effects

### 5.1 Conditional Attack Bonuses

**Spirit Blade** (internal/artifacts/spirit_blade.json)
- Current: `"attack_dice": 3`, `"attack_bonus": 1`, `"bonus_trigger": "undead"`
- Update to:
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

**Orcs Bane** (internal/artifacts/orcs_bane.json)
- Current: No effect field
- Update to:
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

### 5.2 Once-Per-Quest Abilities

**Phantom Blade** (internal/artifacts/phantom_blade.json)
- Current: No effect field
- Update to:
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

**Fortunes Longsword** (internal/artifacts/fortunes_longsword.json:24)
- Current: No effect field
- Update to:
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

---

## Phase 6: Complex Utility Items

### 6.1 Placement and Board Effects

**Caltrops** (internal/equipment/caltrops.json)
- Current: Descriptive only
- Update to:
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

**Smoke Bomb** (internal/equipment/smoke_bomb.json)
- Current: Descriptive only
- Update to:
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

**Tool Kit** (internal/equipment/tool_kit.json)
- Current: Descriptive only
- Update to:
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

### 6.2 Enemy Interaction

**Rod of Telekinesis** (internal/artifacts/rod_of_telekinesis.json:9)
- Current: Descriptive only
- Update to:
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
    }
  }
}
```

---

## Phase 7: Spell System Modifiers

### 7.1 Spell Enhancement Items

**Spell Ring** (internal/artifacts/spell_ring.json)
- Current: Descriptive only
- Update to:
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

**Wand of Magic** (internal/artifacts/wand_of_magic.json:9)
- Current: Descriptive only
- Update to:
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

### 7.2 Party-Wide Effects

**Ring of Return** (internal/artifacts/ring_of_return.json)
- Current: Descriptive only
- Update to:
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

## Phase 8: Validation and Testing

### 8.1 Create Validation Script

Create a Go or Node.js script to validate all JSON files against the updated schema:
- Check required fields
- Validate effect structures
- Ensure no orphaned fields
- Verify consistency

### 8.2 Create Test Content Loader

Build a content loading utility that:
- Loads all equipment and artifacts
- Parses effect structures
- Reports any parsing errors
- Generates summary statistics

---

## Implementation Checklist

### Phase 1: Schema
- [ ] Update CONTENT_JSON_SCHEMAS.md with new fields
- [ ] Document all effect type structures
- [ ] Add examples for each effect type

### Phase 2: Simple Updates (10 items)
- [ ] Verify stat bonus items (ring_of_fortitude, talisman_of_lore, borins_armor)
- [ ] Verify standard equipment items (7 items)

### Phase 3: Special Effects (4 items)
- [ ] Update fire_ring.json
- [ ] Update elixir_of_life.json
- [ ] Update holy_water.json
- [ ] Update magical_throwing_dagger.json

### Phase 4: Multi-Choice (6 items)
- [ ] Update potion_of_dexterity.json
- [ ] Update potion_of_restoration.json
- [ ] Update potion_of_speed.json
- [ ] Update potion_of_battle.json
- [ ] Update dagger.json
- [ ] Update handaxe.json

### Phase 5: Conditional Effects (4 items)
- [ ] Update spirit_blade.json
- [ ] Update orcs_bane.json
- [ ] Update phantom_blade.json
- [ ] Update fortunes_longsword.json

### Phase 6: Complex Utility (4 items)
- [ ] Update caltrops.json
- [ ] Update smoke_bomb.json
- [ ] Update tool_kit.json
- [ ] Update rod_of_telekinesis.json

### Phase 7: Spell Modifiers (3 items)
- [ ] Update spell_ring.json
- [ ] Update wand_of_magic.json
- [ ] Update ring_of_return.json

### Phase 8: Validation
- [ ] Create JSON validation script
- [ ] Run validation on all files
- [ ] Fix any validation errors
- [ ] Create content loader test utility

---

## File-by-File Status

| File | Phase | Status | Notes |
|------|-------|--------|-------|
| ring_of_fortitude.json | 2 | ✓ Complete | Has body_bonus |
| talisman_of_lore.json | 2 | ✓ Complete | Has mind_bonus |
| borins_armor.json | 2 | ✓ Complete | Has movement_bonus |
| bracers.json | 2 | ✓ Complete | Standard item |
| chain_mail.json | 2 | ✓ Complete | Standard item |
| helmet.json | 2 | ✓ Complete | Standard item |
| shield.json | 2 | ✓ Complete | Standard item |
| broadsword.json | 2 | ✓ Complete | Standard item |
| shortsword.json | 2 | ✓ Complete | Standard item |
| longsword.json | 2 | ✓ Complete | Standard item |
| crossbow.json | 2 | ✓ Complete | Standard item |
| staff.json | 2 | ✓ Complete | Standard item |
| battle_axe.json | 2 | ✓ Complete | Has restrictions |
| platemail.json | 2 | ✓ Complete | Has restrictions |
| wizards_staff.json | 2 | ✓ Complete | Standard weapon |
| wizards_cloak.json | 2 | ✓ Complete | Fixed defend_dice |
| fire_ring.json | 3 | Pending | Needs protection effect |
| elixir_of_life.json | 3 | Pending | Needs revival effect |
| holy_water.json | 3 | Pending | Needs instant_kill effect |
| magical_throwing_dagger.json | 3 | Pending | Needs guaranteed_damage effect |
| potion_of_dexterity.json | 4 | Pending | Needs multi_choice effect |
| potion_of_restoration.json | 4 | Pending | Needs restore_points effect |
| potion_of_speed.json | 4 | Pending | Needs movement_multiplier effect |
| potion_of_battle.json | 4 | Pending | Needs reroll_dice effect |
| dagger.json | 4 | Pending | Needs dual_mode_attack effect |
| handaxe.json | 4 | Pending | Needs dual_mode_attack effect |
| spirit_blade.json | 5 | Pending | Needs conditional_bonus effect |
| orcs_bane.json | 5 | Pending | Needs conditional_bonus effect |
| phantom_blade.json | 5 | Pending | Needs special_ability effect |
| fortunes_longsword.json | 5 | Pending | Needs special_ability effect |
| caltrops.json | 6 | Pending | Needs placement effect |
| smoke_bomb.json | 6 | Pending | Needs area_effect |
| tool_kit.json | 6 | Pending | Needs utility effect |
| rod_of_telekinesis.json | 6 | Pending | Needs special_ability with resistance |
| spell_ring.json | 7 | Pending | Needs spell_modifier effect |
| wand_of_magic.json | 7 | Pending | Needs spell_modifier effect |
| ring_of_return.json | 7 | Pending | Needs party_effect |

**Total: 37 items**
- Complete: 16 (43%)
- Pending: 21 (57%)

---

## Next Steps

1. **Decide on approach:**
   - Update schema document first (Phase 1)
   - Then update content files in phases (Phases 3-7)
   - Or update both in parallel

2. **Determine if you want to:**
   - Update files manually one by one
   - Have me generate all updates at once
   - Do phases sequentially with review between each

3. **Consider tooling:**
   - Should we build validation scripts as we go?
   - Or complete all content updates first?

---

*Document Version: 1.0*
*Last Updated: 2025-10-13*
