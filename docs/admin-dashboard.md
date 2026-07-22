# Admin Dashboard

The admin UI is server-rendered with Go templates and compiled Tailwind/daisyUI CSS.

Implemented admin areas:

- Dashboard
- Short Links
- General Settings
- Branding
- SEO
- Footer
- Audit Logs
- Account

The shared admin navigation includes desktop sidebar and mobile drawer markers, visible focus styles through daisyUI/Tailwind, responsive tables, flash messages, and accessible form labels.

Known limits:

- The mobile drawer is intentionally minimal and does not yet include custom JavaScript to auto-close after navigation.
- Browser-level accessibility and responsive screenshot checks are still pending.
