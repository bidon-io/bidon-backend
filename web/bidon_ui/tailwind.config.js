/** @type {import('tailwindcss').Config} */
module.exports = {
  theme: {
    extend: {
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
        sans: ["'Inter'", "'Helvetica Neue'", 'Arial', 'sans-serif'],
        mono: ["'Inter'", "'Helvetica Neue'", 'Arial', 'sans-serif'],
      },
      boxShadow: {
        'bidon-sm':    '0 1px 3px rgba(10, 6, 48, 0.08)',
        'bidon-md':    '0 4px 12px rgba(10, 6, 48, 0.10)',
        'bidon-lg':    '0 8px 24px rgba(10, 6, 48, 0.12)',
        'bidon-focus': '0 0 0 3px rgba(16, 175, 108, 0.35)',
      },
    },
  },
};
