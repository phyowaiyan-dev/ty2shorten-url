# SEO

SEO settings are typed columns on `AppSetting`, edited at `/admin/settings/seo`.

Implemented metadata:

- `<title>`
- description
- keywords
- robots directive
- canonical URL
- Open Graph title, description, URL, site name, image, locale, and type
- Twitter/X card, title, description, image, and site handle
- Google and Bing verification tags
- JSON-LD structured data for the public landing page

Admin, setup, login, and placeholder/error pages use `noindex, nofollow` metadata.

Canonical and image URLs are built from the configured canonical/public base URL, not from arbitrary request Host headers. Relative media paths such as `/media/logo.png` are converted to absolute URLs for social metadata.

Known limits:

- Only the public landing page is included today because there are no other intentional public content pages.
- The app uses GORM `AutoMigrate`; add versioned migrations before production schema churn.
