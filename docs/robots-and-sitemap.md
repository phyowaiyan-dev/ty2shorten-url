# Robots and Sitemap

The application serves crawler resources dynamically:

```text
GET /robots.txt
GET /sitemap.xml
```

When indexing is enabled, `robots.txt` allows `/`, disallows `/admin/` and `/setup/`, and includes the sitemap URL when sitemap generation is enabled.

When indexing is disabled, `robots.txt` returns:

```text
User-agent: *
Disallow: /
```

`sitemap.xml` is generated with Go's XML encoder and currently includes only the public home page. Admin, setup, login, logout, health, and redirect-only routes are excluded.

Both endpoints use cache headers and configured canonical/public base URLs.
