/** @type {import('tailwindcss').Config} */
module.exports = {
  content: [
    "pages/**/*.kiw",
    "components/**/*.kiw",
    "layouts/**/*.kiw",
    "content/**/*.md",
  ],
  darkMode: "class",
  theme: {
    screens: {
      sm: "640px",
      md: "768px",
      lg: "1024px",
      xl: "1280px",
    },
    extend: {
      colors: {
        primary: {
          DEFAULT: "#39D353",
          content: "#0B1F3B",
        },
        secondary: {
          DEFAULT: "#00D1C1",
          content: "#0B1F3B",
        },
        accent: {
          DEFAULT: "#FF3B2E",
          content: "#ffffff",
        },
        fg: {
          DEFAULT: "#0B1F3B",
          dark: "#FAF8F4",
        },
        bg: {
          DEFAULT: "#FAF8F4",
          2: "#f2ede4",
          dark: "#0B1F3B",
          "dark-2": "#112646",
        },
        muted: {
          DEFAULT: "#57677d",
          dark: "#9ab0ce",
        },
        border: {
          DEFAULT: "#0B1F3B",
          dark: "#FAF8F4",
        },
      },
      fontFamily: {
        sans: ["Inter", "ui-sans-serif", "system-ui", "-apple-system", "Segoe UI", "Roboto", "Helvetica", "Arial"],
        mono: ["ui-monospace", "SFMono-Regular", "Menlo", "Consolas", "monospace"],
      },
      fontSize: {
        "2xs": ["10px", { lineHeight: "1.4" }],
        xs: ["11px", { lineHeight: "1.5" }],
        sm: ["13px", { lineHeight: "1.6" }],
        base: ["14.5px", { lineHeight: "1.65" }],
        lg: ["17px", { lineHeight: "1.5" }],
        xl: ["20px", { lineHeight: "1.4" }],
        "2xl": ["24px", { lineHeight: "1.3" }],
        "3xl": ["30px", { lineHeight: "1.2" }],
        "4xl": ["36px", { lineHeight: "1.15" }],
        "5xl": ["44px", { lineHeight: "1.08" }],
        "6xl": ["56px", { lineHeight: "1.05" }],
      },
      maxWidth: {
        container: "80rem",
        prose: "760px",
      },
      boxShadow: {
        pop:    "4px 4px 0px #0B1F3B",
        "pop-lg": "6px 6px 0px #0B1F3B",
        "pop-sm": "2px 2px 0px #0B1F3B",
        "pop-dark":    "4px 4px 0px #39D353",
        "pop-dark-lg": "6px 6px 0px #39D353",
        "pop-dark-sm": "2px 2px 0px #39D353",
      },
      borderWidth: {
        2.5: "2.5px",
      },
      borderRadius: {
        DEFAULT: "10px",
        md: "8px",
        lg: "12px",
        xl: "14px",
        "2xl": "18px",
      },
      spacing: {
        18: "4.5rem",
      },
    },
  },
  plugins: [],
}
