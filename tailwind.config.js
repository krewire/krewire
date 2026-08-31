/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ["pages/**/*.kiw", "components/**/*.kiw", "layouts/**/*.kiw", "content/**/*.md"],
  darkMode: "class",
  theme: {
    extend: {
      colors: {
        primary: {
          DEFAULT: "var(--color-primary)",
          light: "var(--color-primary-light)",
        },
        muted: "var(--color-muted)",
        border: "var(--color-border)",
        fg: "var(--color-fg)",
        bg: {
          DEFAULT: "var(--color-bg)",
          2: "var(--color-bg-2)",
        },
      },
      fontFamily: {
        sans: ["Inter", "ui-sans-serif", "system-ui", "-apple-system", "Segoe UI", "Roboto", "Helvetica", "Arial"],
        serif: ["ui-serif", "Georgia", "serif"],
        mono: ["ui-monospace", "SFMono-Regular", "Menlo", "Consolas", "monospace"],
      },
    },
  },
  plugins: [],
}
