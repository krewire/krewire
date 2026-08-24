/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ["pages/**/*.kiw", "components/**/*.kiw", "layouts/**/*.kiw", "content/**/*.md"],
  theme: {
    extend: {
      colors: {
        primary: "var(--color-primary)",
        muted: "var(--muted)",
        border: "var(--border)",
        fg: "var(--fg)",
        bg: {
          DEFAULT: "var(--bg)",
          2: "var(--bg-2)",
        },
      },
    },
  },
  plugins: [],
}
