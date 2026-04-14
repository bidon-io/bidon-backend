/**
 * BidOn Brand Tokens — 2026
 * Generated from BidOn Brand Guidelines PDF
 *
 * Usage:
 *   import { colors, typography, spacing, shadows, components } from './bidon-brand-tokens'
 *
 * For Tailwind: paste the `tailwindTheme` export into tailwind.config.js → theme.extend
 * For CSS-in-JS (Emotion, styled-components): use the `colors` and `typography` objects directly
 * For global CSS variables: call injectCSSVariables() once at app root
 */

// ─── Color Palette ────────────────────────────────────────────────────────────

export const colors = {
  // Primary brand greens
  mountainMeadow: '#10AF6C',   // Primary CTA, active states, brand hero color
  moss:           '#0E9C62',   // Hover states, secondary green, logo accent

  // Dark backgrounds / text
  ebonyClay:      '#23283B',   // Primary dark background, nav, cards on dark
  midnight:       '#0A0630',   // Deepest dark — hero backgrounds, overlays

  // Neutrals
  manatee:        '#8C8C9C',   // Muted text, placeholders, disabled states
  geyser:         '#E1E7E5',   // Borders, dividers, input backgrounds, subtle fills
  pureWhite:      '#FFFFFF',   // Page background, light card surfaces

  // Semantic aliases (reference palette above)
  brand: {
    primary:      '#10AF6C',   // mountainMeadow
    primaryHover: '#0E9C62',   // moss
    dark:         '#23283B',   // ebonyClay
    darker:       '#0A0630',   // midnight
    muted:        '#8C8C9C',   // manatee
    subtle:       '#E1E7E5',   // geyser
    white:        '#FFFFFF',   // pureWhite
  },

  // Text hierarchy (light mode)
  text: {
    primary:      '#23283B',   // ebonyClay — headings, body
    secondary:    '#8C8C9C',   // manatee — supporting text, captions
    inverse:      '#FFFFFF',   // on dark backgrounds
    onGreen:      '#FFFFFF',   // text on mountain meadow / moss buttons
    link:         '#10AF6C',   // mountainMeadow
    linkHover:    '#0E9C62',   // moss
  },

  // Backgrounds
  bg: {
    page:         '#FFFFFF',   // pureWhite
    surface:      '#E1E7E5',   // geyser — cards, panels
    dark:         '#23283B',   // ebonyClay — dark sections
    darker:       '#0A0630',   // midnight — hero, footer
  },

  // Borders
  border: {
    default:      '#E1E7E5',   // geyser
    strong:       '#8C8C9C',   // manatee
    focus:        '#10AF6C',   // mountainMeadow — input focus ring
  },
};


// ─── Typography ───────────────────────────────────────────────────────────────

export const typography = {
  fontFamily: {
    // Primary: Garet Bold — H1, display headings
    // Secondary: Garet Regular/Book — H2, body text
    // Fallback stack if Garet not loaded
    primary:   "'Garet', 'Inter', 'Helvetica Neue', Arial, sans-serif",
    secondary: "'Garet', 'Inter', 'Helvetica Neue', Arial, sans-serif",
    mono:      "'JetBrains Mono', 'Fira Code', 'Courier New', monospace",
  },

  fontWeight: {
    regular: 400,   // Garet Regular/Book — body, H2
    bold:    700,   // Garet Bold — H1, display, CTAs
  },

  // Type scale
  fontSize: {
    xs:   '0.75rem',    // 12px — labels, captions
    sm:   '0.875rem',   // 14px — secondary body, meta
    base: '1rem',       // 16px — body text
    lg:   '1.125rem',   // 18px — large body, H3
    xl:   '1.25rem',    // 20px — H2
    '2xl':'1.5rem',     // 24px — H2 large
    '3xl':'1.875rem',   // 30px — H1 medium
    '4xl':'2.25rem',    // 36px — H1
    '5xl':'3rem',       // 48px — Display
    '6xl':'3.75rem',    // 60px — Hero display
  },

  lineHeight: {
    tight:  1.2,
    snug:   1.375,
    normal: 1.5,
    relaxed:1.625,
  },

  letterSpacing: {
    tight:  '-0.02em',
    normal: '0',
    wide:   '0.05em',
    wider:  '0.1em',
  },
};


// ─── Spacing ──────────────────────────────────────────────────────────────────

export const spacing = {
  0:    '0',
  1:    '0.25rem',   // 4px
  2:    '0.5rem',    // 8px
  3:    '0.75rem',   // 12px
  4:    '1rem',      // 16px
  5:    '1.25rem',   // 20px
  6:    '1.5rem',    // 24px
  8:    '2rem',      // 32px
  10:   '2.5rem',    // 40px
  12:   '3rem',      // 48px
  16:   '4rem',      // 64px
  20:   '5rem',      // 80px
  24:   '6rem',      // 96px
};


// ─── Border Radius ────────────────────────────────────────────────────────────

export const borderRadius = {
  none:   '0',
  sm:     '4px',
  md:     '8px',
  lg:     '12px',
  xl:     '16px',
  '2xl':  '24px',
  full:   '9999px',   // pills, badges, avatar circles
};


// ─── Shadows ──────────────────────────────────────────────────────────────────

export const shadows = {
  none: 'none',
  sm:   '0 1px 3px rgba(10, 6, 48, 0.08)',
  md:   '0 4px 12px rgba(10, 6, 48, 0.10)',
  lg:   '0 8px 24px rgba(10, 6, 48, 0.12)',
  focus:'0 0 0 3px rgba(16, 175, 108, 0.35)',   // mountainMeadow focus ring
};


// ─── Component Tokens ─────────────────────────────────────────────────────────

export const components = {

  // Primary button — green CTA
  buttonPrimary: {
    background:       colors.mountainMeadow,
    backgroundHover:  colors.moss,
    color:            colors.pureWhite,
    borderRadius:     borderRadius.md,
    fontWeight:       typography.fontWeight.bold,
    fontSize:         typography.fontSize.base,
    paddingX:         '1.5rem',
    paddingY:         '0.75rem',
    border:           'none',
    transition:       'background 0.15s ease',
  },

  // Secondary button — outlined
  buttonSecondary: {
    background:       'transparent',
    backgroundHover:  colors.geyser,
    color:            colors.ebonyClay,
    border:           `1.5px solid ${colors.ebonyClay}`,
    borderRadius:     borderRadius.md,
    fontWeight:       typography.fontWeight.bold,
    fontSize:         typography.fontSize.base,
    paddingX:         '1.5rem',
    paddingY:         '0.75rem',
    transition:       'background 0.15s ease',
  },

  // Ghost button — for dark backgrounds
  buttonGhost: {
    background:       'transparent',
    backgroundHover:  'rgba(255,255,255,0.1)',
    color:            colors.pureWhite,
    border:           `1.5px solid ${colors.pureWhite}`,
    borderRadius:     borderRadius.md,
    fontWeight:       typography.fontWeight.bold,
    fontSize:         typography.fontSize.base,
    paddingX:         '1.5rem',
    paddingY:         '0.75rem',
  },

  // Input field
  input: {
    background:       colors.pureWhite,
    border:           `1px solid ${colors.geyser}`,
    borderFocus:      `1px solid ${colors.mountainMeadow}`,
    borderRadius:     borderRadius.md,
    color:            colors.ebonyClay,
    placeholderColor: colors.manatee,
    fontSize:         typography.fontSize.base,
    paddingX:         '0.875rem',
    paddingY:         '0.625rem',
    boxShadowFocus:   shadows.focus,
  },

  // Card — light surface
  card: {
    background:       colors.pureWhite,
    border:           `1px solid ${colors.geyser}`,
    borderRadius:     borderRadius.lg,
    padding:          '1.5rem',
    shadow:           shadows.sm,
  },

  // Card — dark variant
  cardDark: {
    background:       colors.ebonyClay,
    border:           'none',
    borderRadius:     borderRadius.lg,
    padding:          '1.5rem',
    color:            colors.pureWhite,
  },

  // Badge / tag
  badge: {
    green: {
      background:   'rgba(16, 175, 108, 0.12)',
      color:        colors.moss,
      borderRadius: borderRadius.full,
      fontSize:     typography.fontSize.xs,
      fontWeight:   typography.fontWeight.bold,
      paddingX:     '0.625rem',
      paddingY:     '0.25rem',
    },
    dark: {
      background:   colors.ebonyClay,
      color:        colors.pureWhite,
      borderRadius: borderRadius.full,
      fontSize:     typography.fontSize.xs,
      fontWeight:   typography.fontWeight.bold,
      paddingX:     '0.625rem',
      paddingY:     '0.25rem',
    },
    muted: {
      background:   colors.geyser,
      color:        colors.manatee,
      borderRadius: borderRadius.full,
      fontSize:     typography.fontSize.xs,
      fontWeight:   typography.fontWeight.bold,
      paddingX:     '0.625rem',
      paddingY:     '0.25rem',
    },
  },

  // Navigation
  nav: {
    background:       colors.ebonyClay,
    color:            colors.pureWhite,
    activeColor:      colors.mountainMeadow,
    hoverColor:       'rgba(255,255,255,0.8)',
    height:           '64px',
  },

  // Section backgrounds
  section: {
    light:    colors.pureWhite,
    subtle:   colors.geyser,
    dark:     colors.ebonyClay,
    darkest:  colors.midnight,
    brand:    colors.mountainMeadow,
  },
};


// ─── Tailwind Theme Extension ──────────────────────────────────────────────────
// Paste into tailwind.config.js → theme: { extend: { ...tailwindTheme } }

export const tailwindTheme = {
  colors: {
    bidon: {
      'mountain-meadow': '#10AF6C',
      'moss':            '#0E9C62',
      'ebony-clay':      '#23283B',
      'midnight':        '#0A0630',
      'manatee':         '#8C8C9C',
      'geyser':          '#E1E7E5',
      'white':           '#FFFFFF',
    },
  },
  fontFamily: {
    sans:  ["'Garet'", "'Inter'", "'Helvetica Neue'", "Arial", "sans-serif"],
    mono:  ["'JetBrains Mono'", "'Fira Code'", "'Courier New'", "monospace"],
  },
};


// ─── CSS Custom Properties Injector ───────────────────────────────────────────
// Call once at app root: injectCSSVariables()
// Then use var(--bidon-primary) etc. in any CSS file

export function injectCSSVariables() {
  const root = document.documentElement;
  const vars = {
    '--bidon-primary':        colors.mountainMeadow,
    '--bidon-primary-hover':  colors.moss,
    '--bidon-dark':           colors.ebonyClay,
    '--bidon-darker':         colors.midnight,
    '--bidon-muted':          colors.manatee,
    '--bidon-subtle':         colors.geyser,
    '--bidon-white':          colors.pureWhite,

    '--bidon-text-primary':   colors.text.primary,
    '--bidon-text-secondary': colors.text.secondary,
    '--bidon-text-inverse':   colors.text.inverse,
    '--bidon-text-link':      colors.text.link,

    '--bidon-bg-page':        colors.bg.page,
    '--bidon-bg-surface':     colors.bg.surface,
    '--bidon-bg-dark':        colors.bg.dark,

    '--bidon-border-default': colors.border.default,
    '--bidon-border-focus':   colors.border.focus,

    '--bidon-radius-sm':      borderRadius.sm,
    '--bidon-radius-md':      borderRadius.md,
    '--bidon-radius-lg':      borderRadius.lg,
    '--bidon-radius-full':    borderRadius.full,

    '--bidon-shadow-sm':      shadows.sm,
    '--bidon-shadow-md':      shadows.md,
    '--bidon-shadow-focus':   shadows.focus,

    '--bidon-font-primary':   typography.fontFamily.primary,
    '--bidon-font-mono':      typography.fontFamily.mono,
  };
  Object.entries(vars).forEach(([k, v]) => root.style.setProperty(k, v));
}


// ─── Garet Font Loader ────────────────────────────────────────────────────────
// Call if Garet is self-hosted. Adjust paths to your font directory.

export function loadGaretFont(basePath = '/fonts') {
  const style = document.createElement('style');
  style.textContent = `
    @font-face {
      font-family: 'Garet';
      src: url('${basePath}/Garet-Book.woff2') format('woff2'),
           url('${basePath}/Garet-Book.woff') format('woff');
      font-weight: 400;
      font-style: normal;
      font-display: swap;
    }
    @font-face {
      font-family: 'Garet';
      src: url('${basePath}/Garet-Heavy.woff2') format('woff2'),
           url('${basePath}/Garet-Heavy.woff') format('woff');
      font-weight: 700;
      font-style: normal;
      font-display: swap;
    }
  `;
  document.head.appendChild(style);
}
