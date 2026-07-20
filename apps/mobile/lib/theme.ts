/**
 * TypeScript mirror of the CSS variables defined in apps/mobile/global.css.
 *
 * - `THEME` is the raw token object for inline styles, animations, and
 *   anywhere a Tailwind class can't reach.
 * - `NAV_THEME` is the React Navigation theme — passed into <ThemeProvider />
 *   in app/_layout.tsx so headers, modals, and the back button match.
 *
 * If you change a variable in global.css, update the matching key here.
 * See apps/mobile/docs/rnr-migration.md §5 for the sync rule.
 */
import { DarkTheme, DefaultTheme, type Theme } from "@react-navigation/native";

export const THEME = {
  light: {
    background: "hsl(0 0% 100%)",
    foreground: "hsl(240 28% 14%)",
    card: "hsl(240 12% 97%)",
    cardForeground: "hsl(240 28% 14%)",
    popover: "hsl(0 0% 100%)",
    popoverForeground: "hsl(240 28% 14%)",
    primary: "hsl(221 83% 53%)",
    primaryForeground: "hsl(0 0% 100%)",
    secondary: "hsl(240 5% 93%)",
    secondaryForeground: "hsl(240 28% 14%)",
    muted: "hsl(240 5% 93%)",
    mutedForeground: "hsl(0 0% 33%)",
    accent: "hsl(221 83% 95%)",
    accentForeground: "hsl(221 83% 53%)",
    destructive: "hsl(0 72% 51%)",
    destructiveForeground: "hsl(0 0% 100%)",
    border: "hsl(240 6% 90%)",
    input: "hsl(240 6% 90%)",
    ring: "hsl(221 83% 53%)",
    radius: "0.625rem",
    chart1: "hsl(221 83% 53%)",
    chart2: "hsl(217 91% 60%)",
    chart3: "hsl(213 94% 68%)",
    chart4: "hsl(190 80% 45%)",
    chart5: "hsl(158 64% 52%)",

    // Multica custom
    brand: "hsl(221 83% 53%)",
    brandForeground: "hsl(0 0% 100%)",
    success: "hsl(158 94% 30%)",
    warning: "hsl(32 95% 44%)",
    info: "hsl(221 83% 53%)",
    priority: "hsl(25 95% 53%)",
    codeSurface: "hsl(240 6% 95%)",
    // Surface elevation tiers — see global.css for the full scale.
    surface1: "hsl(240 12% 97%)",
    surface2: "hsl(240 5% 93%)",
  },
  dark: {
    background: "hsl(240 10% 4%)",
    foreground: "hsl(0 0% 98%)",
    card: "hsl(240 6% 6%)",
    cardForeground: "hsl(0 0% 98%)",
    popover: "hsl(240 6% 6%)",
    popoverForeground: "hsl(0 0% 98%)",
    primary: "hsl(213 94% 68%)",
    primaryForeground: "hsl(240 10% 4%)",
    secondary: "hsl(240 7% 11%)",
    secondaryForeground: "hsl(0 0% 98%)",
    muted: "hsl(240 7% 11%)",
    mutedForeground: "hsl(240 5% 65%)",
    accent: "hsl(222 47% 16%)",
    accentForeground: "hsl(213 94% 68%)",
    destructive: "hsl(0 75% 60%)",
    destructiveForeground: "hsl(0 0% 98%)",
    border: "hsl(240 6% 16%)",
    input: "hsl(240 6% 18%)",
    ring: "hsl(213 94% 68%)",
    radius: "0.625rem",
    chart1: "hsl(213 94% 68%)",
    chart2: "hsl(217 91% 60%)",
    chart3: "hsl(190 85% 55%)",
    chart4: "hsl(158 64% 52%)",
    chart5: "hsl(158 55% 62%)",

    // Multica custom
    brand: "hsl(221 83% 53%)",
    brandForeground: "hsl(0 0% 98%)",
    success: "hsl(158 64% 52%)",
    warning: "hsl(37 91% 55%)",
    info: "hsl(213 94% 68%)",
    priority: "hsl(25 95% 53%)",
    // code-surface = LunarWing dark code bg (#111113). Keep in sync with .dark:root.
    codeSurface: "hsl(240 6% 7%)",
    // Dark elevation tiers — lightness INCREASES with elevation. See global.css.
    surface1: "hsl(240 6% 6%)",
    surface2: "hsl(240 7% 11%)",
  },
};

export const NAV_THEME: Record<"light" | "dark", Theme> = {
  light: {
    ...DefaultTheme,
    colors: {
      background: THEME.light.background,
      border: THEME.light.border,
      card: THEME.light.card,
      notification: THEME.light.destructive,
      primary: THEME.light.primary,
      text: THEME.light.foreground,
    },
  },
  dark: {
    ...DarkTheme,
    colors: {
      background: THEME.dark.background,
      border: THEME.dark.border,
      card: THEME.dark.card,
      notification: THEME.dark.destructive,
      primary: THEME.dark.primary,
      text: THEME.dark.foreground,
    },
  },
};
