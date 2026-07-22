# API and Route Reference

| Method | Path | Auth | Description | Response |
| --- | --- | --- | --- | --- |
| GET | `/` | Public after setup | Landing page | HTML |
| GET | `/health` | Public | Health and DB check | JSON |
| GET | `/robots.txt` | Public after database setup | Dynamic robots rules | Text |
| GET | `/sitemap.xml` | Public after database setup | Dynamic sitemap | XML or 404 |
| GET | `/media/:name` | Public after setup | Uploaded branding media | Image or 404 |
| GET | `/privacy` | Public after setup | Placeholder legal page | HTML |
| GET | `/terms` | Public after setup | Placeholder legal page | HTML |
| GET | `/setup/database` | Public before database setup | Database bootstrap form | HTML |
| POST | `/setup/database/test` | CSRF before database setup | Test selected database config | HTML |
| POST | `/setup/database` | CSRF before database setup | Save tested database config | HTML |
| GET | `/setup` | Public before setup | First-run setup form | HTML or redirect |
| POST | `/setup` | CSRF | Complete setup | Redirect or validation HTML |
| GET | `/admin/login` | Public after setup | Login form | HTML |
| POST | `/admin/login` | CSRF | Login | Redirect or validation HTML |
| POST | `/admin/logout` | Admin + CSRF | Logout | Redirect |
| GET | `/admin` | Admin | Dashboard | HTML |
| GET | `/admin/settings` | Admin | Settings form | HTML |
| POST | `/admin/settings` | Admin + CSRF | Update settings | Redirect or validation HTML |
| GET | `/admin/settings/branding` | Admin | Branding and media form | HTML |
| POST | `/admin/settings/branding` | Admin + CSRF | Update branding and upload media | Redirect or validation HTML |
| GET | `/admin/settings/seo` | Admin | SEO settings form | HTML |
| POST | `/admin/settings/seo` | Admin + CSRF | Update SEO settings | Redirect or validation HTML |
| GET | `/admin/settings/footer` | Admin | Footer settings form | HTML |
| POST | `/admin/settings/footer` | Admin + CSRF | Update footer settings | Redirect or validation HTML |
| GET | `/admin/links` | Admin | List links | HTML |
| GET | `/admin/links/new` | Admin | New link form | HTML |
| POST | `/admin/links` | Admin + CSRF | Create link | Redirect or validation HTML |
| GET | `/admin/links/:id/edit` | Admin | Edit link form | HTML |
| POST | `/admin/links/:id` | Admin + CSRF | Update link | Redirect or validation HTML |
| POST | `/admin/links/:id/delete` | Admin + CSRF | Delete link | Redirect |
| GET | `/admin/account` | Admin | Account form | HTML |
| POST | `/admin/account/password` | Admin + CSRF | Change password | Redirect or validation HTML |
| GET | `/admin/audit-logs` | Admin | Read-only audit log list with filters | HTML |
| GET | `/admin/audit-logs/:id` | Admin | Read-only audit log detail | HTML or 404 |
| GET | `/android` | Public after setup | Android app redirect | 307 or fallback HTML |
| GET | `/apple` | Public after setup | Apple app redirect | 307 or fallback HTML |
| GET | `/get` | Public after setup | Device detection redirect | 307 |
| GET | `/r/:slug` | Public after setup | Short-link redirect | 307 or 404 HTML |

Before database setup, only `/setup/database`, `/setup/database/test`, `/health`, and `/static/*` are available.

After database setup but before application setup, only `/setup`, `/health`, and `/static/*` are available.
