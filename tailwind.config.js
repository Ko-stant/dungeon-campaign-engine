/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./internal/web/views/**/*.templ",
    "./internal/web/static/**/*.js",
    "./cmd/**/*.go",
    "./internal/**/*.go"
  ],
  theme: {
    extend: {
      colors: {
        surface: "rgb(var(--color-surface) / <alpha-value>)",
        "surface-2": "rgb(var(--color-surface-2) / <alpha-value>)",
        content: "rgb(var(--color-content) / <alpha-value>)",
        border: "rgb(var(--color-border) / <alpha-value>)",
        brand: "rgb(var(--color-brand) / <alpha-value>)",
        accent: "rgb(var(--color-accent) / <alpha-value>)",
        positive: "rgb(var(--color-positive) / <alpha-value>)",
        warning: "rgb(var(--color-warning) / <alpha-value>)",
        danger: "rgb(var(--color-danger) / <alpha-value>)"
      }
    }
  },
  plugins: []
}
