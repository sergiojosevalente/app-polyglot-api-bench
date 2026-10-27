// Test runner bawaan Node (node:test) — tanpa Jest/Mocha. Jalankan: npm test
import assert from 'node:assert/strict';
import { describe, it } from 'node:test';
import { fib, summarize } from '../src/compute.ts';
import { ItemStore } from '../src/store.ts';

describe('compute', () => {
  for (const [n, want] of [[0, 0], [1, 1], [2, 1], [10, 55], [25, 75025]] as const) {
    it(`fib(${n}) = ${want}`, () => assert.equal(fib(n), want));
  }

  it('summarize', () => {
    assert.deepEqual(summarize([1, 2, 3, 4.5, -2]), { count: 5, sum: 8.5, min: -2, max: 4.5, mean: 1.7 });
  });

  it('summarize empty', () => assert.equal(summarize([]), undefined));
});

describe('store', () => {
  it('seed and get', () => {
    const s = new ItemStore();
    s.seed(10);
    assert.deepEqual(s.get(7), { id: 7, name: 'item-7', price: 10.5 });
    assert.equal(s.get(99), undefined);
  });
});
