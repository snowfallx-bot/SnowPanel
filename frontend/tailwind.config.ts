import type { Config } from "tailwindcss";

const config: Config = {
  content: ["./index.html", "./src/**/*.{ts,tsx}"],
  theme: {
    extend: {
      colors: {
        panel: {
          50: "#f2f8f7",
          100: "#dceeea",
          200: "#b7ddd6",
          300: "#89c4ba",
          400: "#58a89d",
          500: "#2f8b81",
          600: "#1d6f67",
          700: "#1a5752",
          800: "#194743",
          900: "#183d39"
        }
      }
    }
  },
  plugins: []
};

export default config;
