# HeroQuest Engine Development Roadmap

## Current Foundation Assessment

### Already Implemented
- Hero action system framework with all 6 core actions (attack, cast spell, search treasure/traps/secret, disarm trap)
- Turn management with hero/GameMaster alternation
- Player/character system with 4 hero classes (Barbarian, Dwarf, Elf, Wizard) --- EDIT: add additional hero classes as well
- Dice system with debug overrides
- Movement validation and processing
- Monster system with visibility tracking
- WebSocket communication and broadcasting
- Real-time game state synchronization
- Furniture system with rendering and collision
- Door opening mechanics with visibility updates

### Partially Implemented
-  ~~Basic attack action (needs proper HeroQuest dice mechanics)~~ - **COMPLETED**
- Simple treasure search (needs quest-specific loot tables)
-  ~~Monster positioning and rendering (needs death mechanics)~~ - **COMPLETED**
-  ~~Movement system (needs dice-based movement)~~ - **COMPLETED**

## Implementation Phases

### **PHASE 1: Core Mechanics Enhancement**  *Highest Priority*

#### 1.1 Combat System with Dice Rolling **COMPLETED**
**Goal**: Implement authentic HeroQuest combat mechanics
- [x] Replace simple damage calculation with skull/shield dice mechanics
- [x] Add hero character stats (Body Points, Mind Points, Attack Dice, Defense Dice per class)
- [x] Implement monster defense dice rolling when attacked
- [x] Create proper damage application and monster death handling
- [x] Add visual representation of combat dice results (skulls, white shields, black shields)
- [x] Implement attack resolution: skulls hit, shields defend, net damage calculation
- [x] Add debug override for combat dice for testing

#### 1.2 Movement Dice System **COMPLETED**
**Goal**: Replace hardcoded movement with dice-based mechanics
- [x] Implement movement dice rolling (2d6 movement dice per turn, each die 1-6 squares)
- [x] Add character-specific movement dice count (2 dice per hero class)
- [x] Update movement validation to require dice rolls first
- [x] Create visual feedback for movement dice rolls with server-side processing
- [x] Add InstantAction system for rolling movement dice
- [x] Integrate WebSocket handling for movement dice requests

#### 1.3 Monster Death and Damage System
**Goal**: Handle monster lifecycle and rewards
- [ ] Implement monster body point tracking and reduction
- [ ] Add monster death mechanics and removal from game state
- [ ] Create treasure drops from defeated monsters -- EDIT: per quest configuration, this should be flexible and customizable inside each quest's json file
- [ ] Update visibility system to handle dead monsters
- [ ] Add monster death animations/feedback

### **PHASE 2: Quest System and Customization**

#### 2.1 Quest-Specific Notes System
**Goal**: Flexible quest configuration and special effects
- [ ] Design quest notes/effects framework in quest definition files
- [ ] Implement treasure chest contents configuration per quest
- [ ] Add weapon rack and furniture interaction effects
- [ ] Add other special quest treasures per notes for other furniture or rooms
- [ ] Support conditional effects (room cleared, specific triggers)
- [ ] Create quest event system for special encounters

#### 2.2 Monster Stat Overrides
**Goal**: Per-quest monster customization
- [ ] Allow quest-specific monster stat modifications (extra attack dice, body points)
- [ ] Implement special monster abilities per quest, e.g., dread spells for quest specific monsters
- [ ] Add conditional monster spawning based on quest events, e.g., spawning via "wandering monster" treasure cards
- [ ] Create monster variant system for quest diversity

### **PHASE 3: Character System Expansion**

#### 3.1 Character Sheets and Stats
**Goal**: Detailed character progression and management
- [ ] Create comprehensive character sheets for each hero class
- [ ] Implement class-specific abilities and bonuses, each hero class has special info asset cards that need to be implemented
- [ ] Add equipment slots and inventory management
- [ ] Implement stat modifications from equipment, attack dice never falls below 1, but weapon upgrades and armor upgrades are additive based on a 0 value for weapons and additive to the base defense of armor, unless other armor is already equipped for a slot. For example chainmail adds +1 defense, plate armor adds +2 defense, but you cannot wear both at the same time, and equipping platemail removes the +1 defense from chainmail.
- [x] There is no level progression system in HeroQuest, so no need to implement that.

#### 3.2 Hero Selection and Death
**Goal**: Campaign setup and consequence management
- [ ] Implement hero selection interface at campaign start
- [ ] Add hero death mechanics:
  - Unconsciousness vs permanent death
  - Equipment loss consequences
  - Revival mechanics (other heroes can help)
- [ ] Design resurrection/recovery system
- [ ] Handle campaign continuation with dead heroes, e.g., replacing with new heroes or for a TPK (total party kill) scenario, replaying the quest with all new heroes, restarting the campaign, or loading a previous save.

### **PHASE 4: Spell and Item Systems**

#### 4.1 Spell System
**Goal**: Complete magic system for Wizard and Elf
- [ ] Implement spell inventory and spell cards
- [x] Mind points are not consumed during spell casting in HeroQuest, so no need to implement that.
- [ ] Create spell effect duration tracking
- [ ] Design spell targeting (self, ally, enemy, area)
- [ ] Implement specific HeroQuest spells:
  - Heal Body/Mind
  - Ball of Flame
  - Swift Wind (movement)
  - Pass Through Rock (ignore walls/furniture)
  - Courage (bonus dice)

#### 4.2 Item and Equipment System
**Goal**: Complete treasure and equipment mechanics
- [ ] Create comprehensive item database
- [ ] Implement equipment effects on character stats
- [ ] Add potion types and usage mechanics
- [ ] Design item trading between adjacent players
- [x] There are no carrying limits in HeroQuest, so no need to implement that.
- [ ] Implement artifact and magical item effects

### **PHASE 5: Advanced Search Mechanics**

#### 5.1 Enhanced Search Systems
**Goal**: Room-specific and realistic search mechanics
- [ ] Implement room-specific treasure generation tables
- [ ] Add search limitations (one search per room type per hero)
- [ ] Create treasure variety based on room type and quest
- [ ] Design search interactions with furniture
  - Searching for anything in a room searches the whole of that room, no need to be next to a specific piece of furniture.
- [ ] Add "already searched" tracking per room/hero

#### 5.2 Trap System
**Goal**: Complete trap mechanics
- [ ] Implement trap types:
  - Pit traps
  - Poison needle traps (future implementation as it only pertain to expansion quests)
  - Spear traps
  - Falling rock traps
- [ ] Add trap placement in quest definitions
- [ ] Create trap detection and disarming mechanics
  - Traps in rooms are detected when searching the room, no need to be next to a specific piece of furniture.
  - Traps in corridors are detected when in line of sight with the corridor tile containing the trap.
- [ ] Implement tool kit requirements (except for Dwarf/Explorer who can disarm without tools)
- [ ] Design trap consequences and damage types

### **PHASE 6: UI and UX Improvements**

#### 6.1 Game Master vs Player Views
**Goal**: Role-appropriate interfaces
- [ ] Create GM-specific UI with full game state visibility
- [ ] Implement player-limited view with fog of war
- [ ] Add GM controls for:
  - Manual monster placement, e.g. when spawning via "wandering monster" treasure cards
  - Dice roll overrides
  - Quest event triggers
  - Rule modifications
- [ ] Design different action palettes per role
- [ ] Add GM tools for campaign management

#### 6.2 Visual Quest Editor
**Goal**: Replace manual JSON editing
- [ ] Create drag-and-drop room/corridor designer
- [ ] Add visual monster placement interface
- [ ] Implement treasure and note configuration UI
- [ ] Design door and wall placement tools
- [ ] Add quest testing and validation tools
- [ ] Create quest export/import functionality

### **PHASE 7: Multiplayer Enhancement**

#### 7.1 Session Management
**Goal**: Proper multiplayer campaign support
- [ ] Implement campaign creation by GameMaster
- [ ] Add player joining mechanisms with room codes/links
- [ ] Create lobby system for pre-game hero selection
- [ ] Add player authentication and session persistence
- [ ] Implement campaign save/load functionality

#### 7.2 Advanced Multiplayer Features
**Goal**: Enhanced multiplayer experience
- [ ] Support multiple concurrent campaigns
- [ ] Add spectator mode for observers
- [ ] Create player reconnection handling
- [ ] Design campaign progression tracking
- [ ] Add text chat integration
- [ ] Implement player avatar customization (optional feature for the future)

## Custom Rules and GM Overrides

### Monster Placement Flexibility
- **Delayed Monster Reveal**: GM can choose when to reveal monsters for dramatic effect
- **Dynamic Placement**: Place monsters when heroes enter rooms vs pre-placed
- **Narrative Control**: Override strict line-of-sight for storytelling

### Custom Dice Rolling Rules (Optional Toggles)
- **Double Dice Effects**:
  - Double 1s: Roll one fewer attack die next attack
  - Double 2-5: Reroll one attack die
  - Double 6s: Reroll 2 attack dice
- **Alternative Movement**: Option for fixed movement vs dice-based, e.g., always allow 8 tile movement when no monsters are present on gameboard

### GM Authority System
- **Rule Override Controls**: Real-time rule modifications during gameplay
- **Bargaining System**: Allow alternative outcomes based on player negotiation
- **Flexible Line of Sight**: Override center-to-center calculations when appropriate
- **Custom Rewards**: Dynamic treasure and experience modifications

## Technical Implementation Notes

### Architecture Considerations
- All new systems should integrate with existing WebSocket protocol
- Maintain thread-safe game state management
- Use existing dice system with proper extensions
- Follow established pattern of system separation (MonsterSystem, FurnitureSystem, etc.)

### Database Requirements
- Quest definitions with embedded customizations
- Player/character persistence across sessions
- Campaign state saving and loading
- Item/spell/treasure databases

### Testing Strategy
- Unit tests for each new system component
- Integration tests for combat mechanics
- End-to-end tests for complete game scenarios
- Performance tests for multiplayer scenarios

## Next Steps

**Recommended Starting Point**: Phase 1.1 - Combat System with Dice Rolling

**Rationale**:
1. **High Impact**: Combat is core to HeroQuest gameplay
2. **Foundation**: Many other systems depend on proper combat mechanics
3. **Existing Code**: Can build on current attack action framework
4. **Immediately Testable**: Easy to verify with existing monster system

**First Implementation Tasks**:
1. Define hero class stats (Body/Mind points, Attack/Defense dice)
2. Implement HeroQuest combat dice mechanics (skulls/shields)
3. Add monster defense dice rolling
4. Create damage application and monster death handling

---

*This roadmap should be treated as a living document, updated as requirements evolve and implementation progresses.*