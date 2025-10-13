/**
 * Actions Panel UI Controller
 * Manages the tabbed interface for Movement/Items/Actions
 */

export class ActionsPanelController {
  constructor() {
    this.currentTab = 'movement';
    this.tabs = {
      movement: {
        button: document.getElementById('tabMovement'),
        content: document.getElementById('movementTabContent')
      },
      items: {
        button: document.getElementById('tabItems'),
        content: document.getElementById('itemsTabContent')
      },
      actions: {
        button: document.getElementById('tabActions'),
        content: document.getElementById('actionsTabContent')
      }
    };

    this.init();
  }

  init() {
    // Set up tab button click handlers
    Object.entries(this.tabs).forEach(([tabName, tab]) => {
      if (tab.button) {
        tab.button.addEventListener('click', () => this.switchTab(tabName));
      }
    });

    // Set up keyboard shortcuts
    document.addEventListener('keydown', (e) => {
      // Don't trigger if user is typing in an input
      if (e.target.tagName === 'INPUT' || e.target.tagName === 'TEXTAREA') {
        return;
      }

      switch (e.key.toLowerCase()) {
        case 'm':
          this.switchTab('movement');
          break;
        case 'i':
          this.switchTab('items');
          break;
        case 'a':
          this.switchTab('actions');
          break;
      }
    });

    // Show movement tab by default
    this.switchTab('movement');
  }

  switchTab(tabName) {
    if (!this.tabs[tabName]) {
      console.warn(`Tab "${tabName}" does not exist`);
      return;
    }

    this.currentTab = tabName;

    // Update button styles
    Object.entries(this.tabs).forEach(([name, tab]) => {
      if (tab.button) {
        if (name === tabName) {
          // Active tab styling
          tab.button.classList.add('bg-blue-600/20', 'border-blue-600/40', 'text-blue-300');
          tab.button.classList.remove('bg-surface', 'border-border/60');
        } else {
          // Inactive tab styling
          tab.button.classList.remove('bg-blue-600/20', 'border-blue-600/40', 'text-blue-300');
          tab.button.classList.add('bg-surface', 'border-border/60');
        }
      }
    });

    // Show/hide content
    Object.entries(this.tabs).forEach(([name, tab]) => {
      if (tab.content) {
        if (name === tabName) {
          tab.content.classList.remove('hidden');
        } else {
          tab.content.classList.add('hidden');
        }
      }
    });
  }

  getCurrentTab() {
    return this.currentTab;
  }

  // Enable/disable specific actions based on game state
  updateAvailableActions(availableActions) {
    // TODO: Enable/disable action buttons based on what's available
    // e.g., disable "Search Treasure" if not on a searchable tile
  }
}
