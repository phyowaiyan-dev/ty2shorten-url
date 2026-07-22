# Analytics

Ty2Shorten URL includes first-party analytics for public page views and redirect traffic. Tracking starts automatically for eligible public website traffic.

## What Is Collected

- First-party analytics session ID stored in `ty2_analytics_session`.
- Public page views for eligible HTML pages.
- Redirect events for `/android`, `/apple`, `/get`, and `/r/:slug`.
- Safe route data such as path, redirect type, redirect status, and destination host/path.
- Conservative User-Agent inference for device type, operating system, browser, and bot status.
- Referrer host and path without external query strings.
- Approximate unique visitor hashes when enabled.
- HMAC IP hash or truncated network information depending on the configured IP mode.

## What Is Not Collected

The analytics system does not use third-party analytics scripts, advertising pixels, or invasive fingerprinting. It does not collect canvas, audio, WebGL, font, hardware serial, MAC address, IMEI, contact-list, clipboard, geolocation, or installed-application data.

Raw User-Agent storage and raw IP display are intentionally unavailable in the standard dashboard.

## Admin Controls

Admins can configure analytics under **Admin -> Settings -> Analytics**:

- Enable or disable analytics.
- Enable or disable page-view and redirect tracking.
- Track or exclude bot traffic.
- Use `hashed`, `network_only`, or `none` IP handling.
- Set first-party analytics cookie lifetime.
- Set data retention period.
- Configure internal CIDR exclusions.
- Enable or disable CSV export.

The public Privacy page does not ask visitors to allow or reject analytics. Analytics are controlled by the site administrator.

## Dashboard And Export

The admin analytics dashboard is available at `/admin/analytics`. It shows page views, redirect events, sessions, estimated visitors, bot sessions, app redirect totals, device mix, browser/OS breakdowns, referrers, destination platforms, and a detailed recent redirect table.

The recent redirect table shows when redirects happened, whether they were Android, Apple, desktop, or custom short-link events, the selected platform, IP network, shortened IP hash, inferred device, operating system, browser, and referrer.

CSV export is available only to authenticated administrators and excludes raw IP and session cookies. CSV cells are protected against spreadsheet formula injection.

Session detail pages show a shortened opaque session ID, first/last seen, session duration, page-view count, redirect count, inferred device information, referrer summary, bot status, and a page/redirect timeline. Full session IDs are not shown.

Retention cleanup can be run from the admin analytics dashboard. It deletes page views, redirect events, and sessions older than the configured retention period.

## Limitations

Device, browser, operating-system, bot, and unique-visitor values are estimates. User-Agent and Client Hint data can be incomplete or misleading. This feature is product analytics, not a legal compliance platform. Final legal requirements depend on deployment region and organization policy.
