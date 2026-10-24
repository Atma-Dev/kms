# kms

Monorepo: `apps/api` (Go) and `apps/web` (Nuxt). Each app has its own
CLAUDE.md; this file covers running them together.

## Starting the local servers

Ports and CORS are set per machine in the `.env` files, so read them before
starting anything instead of assuming the defaults (8019/3019):

- API port: `APP_PORT` in `apps/api/.env`
- Allowed CORS origin: `FRONTEND_URL` in `apps/api/.env` (the API accepts only
  this one origin)
- API base the web calls: `NUXT_PUBLIC_API_BASE` in `apps/web/.env`

Steps:

1. **Check for port clashes.** Run `ss -ltnp` for the API port and the web port
   (the port in `FRONTEND_URL`). If another project already holds one, don't
   start on top of it or let Nuxt silently pick another port. Tell the user and
   agree on free ports.
2. **Keep the three values consistent.** The web port must equal the port in
   `FRONTEND_URL`, and `NUXT_PUBLIC_API_BASE` must point to `APP_PORT`. If you
   change one, change the others too.
3. **Start with the web port set explicitly.** `nuxt.config.ts` sets 3019, but
   pass the port from `FRONTEND_URL` anyway, e.g. `PORT=3019 make dev` (the API
   reads `APP_PORT`, not `PORT`). Run it in the background.
4. **Verify that both servers are up and CORS works.**
   ```sh
   curl -s -o /dev/null -w '%{http_code}\n' localhost:<web-port>
   curl -s -o /dev/null -D - -X OPTIONS localhost:<api-port>/api \
     -H 'Origin: http://localhost:<web-port>' \
     -H 'Access-Control-Request-Method: GET' | grep -i access-control-allow-origin
   ```
   The preflight must echo `Access-Control-Allow-Origin: http://localhost:<web-port>`.
   A 404 on the API root is normal.
5. Report the web and API URLs to the user.

## Deployment

`.gitlab-ci.yml` at the root is the only CI file (GitLab ignores the ones in
subfolders). The server holds one checkout and runs compose from each app
folder; the compose files pin `name: api-kms` / `name: web-kms` so container
and volume names match the pre-monorepo ones. Don't remove those lines.
