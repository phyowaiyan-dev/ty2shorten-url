# Authentication

## Login

`GET /admin/login` renders the login form. `POST /admin/login` verifies the normalized email and password hash. Errors use generic invalid-credential messages.

## Password Hashing

Passwords are hashed with bcrypt. Password hashes are never rendered or logged.

## Sessions

The auth session is a signed cookie containing:

- Admin ID
- Expiration timestamp
- Nonce

Cookies are `HttpOnly`, `SameSite=Lax`, and `Secure` in production.

## Logout

`POST /admin/logout` clears the session cookie and redirects to `/admin/login`.

## Password Changes

`POST /admin/account/password` requires the current password, a new password of at least 10 characters, and confirmation. The session is rotated after success.

## Admin Protection

Admin routes require an authenticated session. Browser requests redirect to `/admin/login`.

## Login Throttling

A small in-memory limiter throttles repeated failed login attempts by normalized email and client IP. Multi-instance deployments should replace it with persistent throttling.
