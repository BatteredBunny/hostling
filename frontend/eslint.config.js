import solid from "eslint-plugin-solid/configs/recommended";
import oxlint from "eslint-plugin-oxlint";
import tseslint from "typescript-eslint";

export default [
  {
    ignores: ["dist/**"],
  },
  tseslint.configs.base,
  {
    ...solid,
    files: ["**/*.{ts,tsx}"],
  },
  ...oxlint.configs["flat/recommended"],
];
