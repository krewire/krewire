/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ["pages/**/*.kiw", "components/**/*.kiw", "layouts/**/*.kiw", "content/**/*.md"],
  theme: {
    extend: {
      colors: {
        primary: "var(--color-primary)",
      },
    },
  },
  plugins: [],
}
