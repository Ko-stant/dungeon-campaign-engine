/**
 * Test runner for all JavaScript modules
 */

import { runTestSuites } from './testFramework.js';
import geometryTests from './geometryTests.js';

/**
 * Run all tests
 */
async function runAllTests() {
  console.log('Starting Dungeon Campaign Engine JavaScript Tests');
  console.log('Environment: Browser ES6 Modules');
  console.log('Date:', new Date().toISOString());

  const suites = [
    geometryTests,
    // Add more test suites here as they're created
  ];

  try {
    const results = await runTestSuites(suites);

    if (results.failed === 0) {
      console.log('\nAll tests passed!');
      return true;
    } else {
      console.log(`\n${results.failed} tests failed`);
      return false;
    }
  } catch (error) {
    console.error('\nTest runner failed:', error);
    return false;
  }
}

// Export for manual use
export { runAllTests };

// Auto-run if this script is loaded directly
const isMainModule = import.meta.url === `file://${process.argv[1]}`;
if (isMainModule || (typeof window !== 'undefined' && import.meta.url === window.location.href)) {
  runAllTests().then(success => {
    console.log(
      success
        ? '\nTest run completed successfully'
        : '\nTest run completed with failures',
    );
    if (typeof process !== 'undefined') {
      process.exit(success ? 0 : 1);
    }
  });
}
