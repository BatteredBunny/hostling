import solid from "eslint-plugin-solid/configs/recommended";
import oxlint from "eslint-plugin-oxlint";
import tseslint from "typescript-eslint";
import astro from "eslint-plugin-astro";

export default [
  {
    ignores: ["dist/**", "src/generated/**"],
  },
  ...astro.configs.recommended,
  {
    ...tseslint.configs.base,
    files: ["**/*.{ts,tsx}"],
  },
  {
    files: ["**/*.astro"],
    languageOptions: {
      parserOptions: { parser: tseslint.parser },
    },
  },
  {
    ...solid,
    files: ["**/*.{ts,tsx}"],
  },
  ...oxlint.configs["flat/recommended"],
];
