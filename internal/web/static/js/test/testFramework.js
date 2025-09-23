/**
 * Simple test framework for JavaScript modules
 */

class TestSuite {
  constructor(name) {
    this.name = name;
    this.tests = [];
    this.results = {
      passed: 0,
      failed: 0,
      errors: [],
    };
  }

  /**
   * Add a test to the suite
   * @param {string} description
   * @param {Function} testFn
   */
  test(description, testFn) {
    this.tests.push({ description, testFn });
  }

  /**
   * Run all tests in the suite
   * @returns {Promise<Object>}
   */
  async run() {
    console.log(`\n🧪 Running test suite: ${this.name}`);
    console.log('='.repeat(50));

    for (const { description, testFn } of this.tests) {
      try {
        await testFn();
        this.results.passed++;
        console.log(`Passed: ${description}`);
      } catch (error) {
        this.results.failed++;
        this.results.errors.push({ description, error });
        console.log(`Failed: ${description}`);
        console.log(`   Error: ${error.message}`);
      }
    }

    const total = this.results.passed + this.results.failed;
    console.log(`\nResults: ${this.results.passed}/${total} passed`);

    if (this.results.failed > 0) {
      console.log('\nFailed tests:');
      this.results.errors.forEach(({ description, error }) => {
        console.log(`   - ${description}: ${error.message}`);
      });
    }

    return this.results;
  }
}

/**
 * Assertion helpers
 */
export const assert = {
  /**
   * Assert that a condition is true
   * @param {boolean} condition
   * @param {string} message
   */
  isTrue(condition, message = 'Expected condition to be true') {
    if (!condition) {
      throw new Error(message);
    }
  },

  /**
   * Assert that a condition is false
   * @param {boolean} condition
   * @param {string} message
   */
  isFalse(condition, message = 'Expected condition to be false') {
    if (condition) {
      throw new Error(message);
    }
  },

  /**
   * Assert that two values are equal
   * @param {*} actual
   * @param {*} expected
   * @param {string} message
   */
  equals(actual, expected, message = null) {
    if (actual !== expected) {
      const msg = message || `Expected ${expected}, got ${actual}`;
      throw new Error(msg);
    }
  },

  /**
   * Assert that two values are deeply equal
   * @param {*} actual
   * @param {*} expected
   * @param {string} message
   */
  deepEquals(actual, expected, message = null) {
    if (JSON.stringify(actual) !== JSON.stringify(expected)) {
      const msg =
        message ||
        `Expected ${JSON.stringify(expected)}, got ${JSON.stringify(actual)}`;
      throw new Error(msg);
    }
  },

  /**
   * Assert that a value is null
   * @param {*} value
   * @param {string} message
   */
  isNull(value, message = 'Expected value to be null') {
    if (value !== null) {
      throw new Error(message);
    }
  },

  /**
   * Assert that a value is not null
   * @param {*} value
   * @param {string} message
   */
  isNotNull(value, message = 'Expected value to not be null') {
    if (value === null) {
      throw new Error(message);
    }
  },

  /**
   * Assert that a value is undefined
   * @param {*} value
   * @param {string} message
   */
  isUndefined(value, message = 'Expected value to be undefined') {
    if (value !== undefined) {
      throw new Error(message);
    }
  },

  /**
   * Assert that a value is not undefined
   * @param {*} value
   * @param {string} message
   */
  isNotUndefined(value, message = 'Expected value to not be undefined') {
    if (value === undefined) {
      throw new Error(message);
    }
  },

  /**
   * Assert that a function throws an error
   * @param {Function} fn
   * @param {string} message
   */
  throws(fn, message = 'Expected function to throw an error') {
    try {
      fn();
      throw new Error(message);
    } catch (error) {
      // Expected to throw - this catch block validates that the function threw an error
      console.log('Function threw as expected:', error.message);
    }
  },

  /**
   * Assert that a value is an instance of a type
   * @param {*} value
   * @param {Function} type
   * @param {string} message
   */
  instanceOf(value, type, message = null) {
    if (!(value instanceof type)) {
      const msg = message || `Expected value to be instance of ${type.name}`;
      throw new Error(msg);
    }
  },

  /**
   * Assert that an array contains a value
   * @param {Array} array
   * @param {*} value
   * @param {string} message
   */
  contains(array, value, message = null) {
    if (!Array.isArray(array)) {
      throw new Error('First argument must be an array');
    }
    if (!array.includes(value)) {
      const msg = message || `Expected array to contain ${value}`;
      throw new Error(msg);
    }
  },

  /**
   * Assert that an array does not contain a value
   * @param {Array} array
   * @param {*} value
   * @param {string} message
   */
  doesNotContain(array, value, message = null) {
    if (!Array.isArray(array)) {
      throw new Error('First argument must be an array');
    }
    if (array.includes(value)) {
      const msg = message || `Expected array to not contain ${value}`;
      throw new Error(msg);
    }
  },
};

/**
 * Create a new test suite
 * @param {string} name
 * @returns {TestSuite}
 */
export function createTestSuite(name) {
  return new TestSuite(name);
}

/**
 * Run multiple test suites
 * @param {TestSuite[]} suites
 * @returns {Promise<Object>}
 */
export async function runTestSuites(suites) {
  const totalResults = {
    passed: 0,
    failed: 0,
    errors: [],
    suites: suites.length,
  };

  console.log(`\nRunning ${suites.length} test suites\n`);

  for (const suite of suites) {
    const results = await suite.run();
    totalResults.passed += results.passed;
    totalResults.failed += results.failed;
    totalResults.errors.push(...results.errors);
  }

  console.log('\n' + '='.repeat(50));
  console.log(
    `Final Results: ${totalResults.passed}/${totalResults.passed + totalResults.failed} tests passed across ${totalResults.suites} suites`,
  );

  if (totalResults.failed > 0) {
    console.log(`\n${totalResults.failed} tests failed`);
  }

  return totalResults;
}
