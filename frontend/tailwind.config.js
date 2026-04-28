/** @type {import('tailwindcss').Config} */
export default {
  content: ["./index.html", "./src/**/*.{ts,tsx}"],
  theme: {
    extend: {
      colors: {
        accent: { DEFAULT: "#2563eb", hover: "#1d4ed8" },
      },
    },
  },
  plugins: [],
};
