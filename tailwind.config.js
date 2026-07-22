module.exports = {
  content: [
    "./web/templates/**/*.html",
    "./cmd/**/*.go",
    "./internal/**/*.go"
  ],
  theme: {
    extend: {}
  },
  plugins: [
    require("@tailwindcss/typography"),
    require("daisyui")
  ],
  daisyui: {
    themes: ["light"]
  },
  safelist: [
    "drawer",
    "drawer-toggle",
    "drawer-content",
    "drawer-side",
    "menu",
    "btn",
    "btn-primary",
    "btn-outline",
    "alert",
    "table",
    "card"
  ]
};
