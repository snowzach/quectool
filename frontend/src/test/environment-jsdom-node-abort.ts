/**
 * Custom vitest environment: jsdom + AbortController/AbortSignal patch.
 *
 * Problem: vitest's jsdom environment calls populateGlobal(global, dom.window)
 * which installs a getter on globalThis.AbortController/AbortSignal that
 * returns jsdom's implementations. Node's built-in fetch (undici) checks
 *   signal instanceof AbortSignal
 * using its own internal AbortSignal class (captured at Node startup). Since
 * jsdom's AbortSignal is a pure-JS class, the instanceof check fails and
 * msw/node's fetch interceptor throws.
 *
 * Fix: after jsdom's populateGlobal runs, override AbortController/AbortSignal
 * back to Node's originals by *assigning* to globalThis — populateGlobal
 * installs a setter that writes to an overrideObject map, so subsequent reads
 * return our restored value instead of jsdom's.
 *
 * We capture Node's originals BEFORE this environment's setup() is called
 * (i.e. before jsdom has touched the globals), which is when this module
 * is first evaluated.
 */

import { builtinEnvironments } from "vitest/environments";

// Capture Node's original classes at module-evaluation time, which happens
// before vitest calls setup() and before jsdom's populateGlobal() replaces them.
const NodeAbortController = globalThis.AbortController;
const NodeAbortSignal = globalThis.AbortSignal;

const jsdomEnv = builtinEnvironments.jsdom;

export default {
  name: "jsdom-node-abort",
  transformMode: "web" as const,

  async setupVM(options: Record<string, unknown>) {
    return jsdomEnv.setupVM?.(options);
  },

  async setup(global: typeof globalThis, options: Record<string, unknown>) {
    // Let jsdom do its thing (this replaces AbortController/AbortSignal with jsdom's)
    const result = await jsdomEnv.setup(global, options);

    // Now restore Node's originals by assigning to globalThis.
    // populateGlobal installs setters that store the value in overrideObject,
    // so subsequent gets will return our value.
    (global as Record<string, unknown>).AbortController = NodeAbortController;
    (global as Record<string, unknown>).AbortSignal = NodeAbortSignal;

    return result;
  },
};
