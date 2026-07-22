# Redirect Engine

## `/android`

Loads the configured Android URL and returns HTTP 307 Temporary Redirect. If missing, it renders a clean fallback page.

## `/apple`

Loads the configured Apple URL and returns HTTP 307 Temporary Redirect. If missing, it renders a clean fallback page.

## `/get`

Uses simple User-Agent matching:

- Android -> `/android`
- iPhone, iPad, iPod -> `/apple`
- Desktop, bots, unknown -> `/`

## `/r/:slug`

Normalizes and validates the slug. Active links return HTTP 307 and increment click count. Missing, inactive, invalid, or reserved slugs render an HTML 404.

## Limitations

Device detection is intentionally simple and may not classify every device or bot correctly. Redirect analytics are limited to click counts.
