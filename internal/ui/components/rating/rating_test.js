const assert = require('node:assert/strict');
const fs = require('node:fs');
const vm = require('node:vm');
const path = require('node:path');

const listeners = {};
const attrs = new Map([
  ['role', 'slider'],
  ['data-tui-rating-initial-value', '3.00'],
  ['data-tui-rating-precision', '0.50'],
  ['data-tui-rating-readonly', 'false'],
  ['data-tui-rating-name', 'rating'],
]);
const input = { value: '3.00', dispatchEvent() {} };
const items = Array.from({ length: 5 }, (_, index) => ({
  getAttribute: () => String(index + 1),
  querySelector: () => ({ style: {} }),
}));
const rating = {
  getAttribute: (key) => attrs.has(key) ? attrs.get(key) : null,
  setAttribute: (key, value) => attrs.set(key, String(value)),
  querySelector: () => input,
  querySelectorAll: () => items,
  closest: () => rating,
  dispatchEvent() {},
};
const document = {
  readyState: 'loading', body: {},
  addEventListener: (name, handler) => { listeners[name] = handler; },
};
class Event { constructor(type, options) { this.type = type; this.detail = options?.detail; } }
class MutationObserver { observe() {} }
vm.runInNewContext(fs.readFileSync(path.join(__dirname, '../../../handlers/static/static/js/rating.js'), 'utf8'), {
  document, Event, CustomEvent: Event, MutationObserver,
});
function key(name) {
  listeners.keydown({ key: name, target: rating, preventDefault() {} });
}
key('ArrowRight');
assert.equal(input.value, '3.50');
key('Home');
assert.equal(input.value, '0.00');
key('ArrowRight');
assert.equal(input.value, '0.50');
assert.equal(attrs.get('aria-valuenow'), '0.5');
