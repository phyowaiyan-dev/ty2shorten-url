# Short Links

## Slug Format

```text
^[a-z0-9][a-z0-9-_]{1,49}$
```

Slugs are normalized to lowercase.

## Reserved Slugs

`admin`, `setup`, `login`, `logout`, `health`, `android`, `apple`, `get`, `static`, `assets`, and `r` are reserved.

## Destination URLs

Only `http` and `https` URLs are allowed. Production prefers HTTPS except explicitly supported local/private development URLs.

## Active Status

Inactive links do not redirect and render a 404.

## Click Counts

Click count increments after a link is resolved. Counter update failures are logged but do not crash the request.

## Deletion

Links can be deleted from the admin UI. For frequently used links, disabling is safer than deletion.

## Limitations

No pagination, bulk actions, expiration, import/export, or soft delete exists yet.
