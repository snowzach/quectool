import { defineConfig } from "vitest/config";
import react from "@vitejs/plugin-react";

export default defineConfig({
  plugins: [react()],
  test: {
    // Use a custom environment that wraps jsdom but restores Node's
    // AbortController/AbortSignal after jsdom replaces them, so that
    // MSW's fetch interceptor (which relies on Node's undici AbortSignal)
    // works correctly in component tests.
    environment: "./src/test/environment-jsdom-node-abort.ts",
    globals: true,
    setupFiles: ["./src/test/setup.ts"],
    css: false,
  },
});
