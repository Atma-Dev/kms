# api-kms

Go REST API for the KMS (knowledge management system). Owns its accounts (email + password) and issues its own JWT tokens. Google login is available as an alternate way to prove the same identity.

Frontend lives at `../web-kms` (Nuxt 4 SPA).

Started as a copy of `lms-api` with the LMS domain removed. What is left is the
base — auth/RBAC/users plus notifications, activity log, audit trail, job logs,
queue, scheduler, m2m clients, lookups — and the KMS domain goes on top of it.

## Stack

- **Framework**: Gin
- **ORM**: GORM (MySQL)
- **JWT**: `gofortify` (iqbalatma/gofortify)
- **Migrations**: `gartisan migrate` locally, plain `golang-migrate` CLI in production/CI (both wrap `golang-migrate`)
- **OAuth**: `golang.org/x/oauth2` for redirect/PKCE, manual HTTP POST for token exchange

## Architecture

```
api-kms (this repo)  →  Authentication (password / Google) + resource server, issues its own JWT
```

## Auth

Two ways in, one tail. Both end at `issueTokens` (`app/auth/handler.go`), which
mints the access/refresh pair and — for a web client — writes both cookies.

### Password (`POST /api/auth/login`)

The normal path. `{email, password}` in, access token out; bcrypt via
`utils.CheckHash`, hash column is `users.password`.

A wrong email and a wrong password return the **same** 401 with the same
message, and the comparison runs against a dummy hash when the row is missing,
so a rejection costs the same ~60ms either way. Both halves matter: a distinct
message or a fast "no" would each let anyone enumerate registered addresses.
The `membership_status` check runs only *after* the password proved the caller
owns the account — an inactive account is a message they are entitled to.

### Google (`GET /api/auth/google/redirect` → `/callback`)

An alternate way to prove the same identity, not a second authorization path.
Confidential client (client secret, no PKCE — the standard server-side "Sign in
with Google" shape). Its endpoints (`golang.org/x/oauth2/endpoints.Google`, plus
the fixed `userinfo` URL) are hardcoded rather than read from env: they are
public URLs that never vary by environment.

Unlike password login it goes through `resolveMember`, which **never creates a
user** — an unknown email or an inactive `membership_status` redirects to
`FRONTEND_URL/auth/callback?error=no_account` / `?error=membership_inactive`.
Distinguishing the two is safe here: the caller already proved a real identity
at Google before reaching it.

### Tokens

6. Issues Go JWT via `gofortify.Encode` (access token + refresh token)
7. Sets `access_token_verifier` and `refresh_token` as httpOnly cookies
8. Web gets the access token in the response body (or the callback URL for
   Google); a non-web client gets the refresh token in the body too, since it
   cannot read the cookie.

### Changing a password

`PATCH /api/profiles/password` takes `current_password` + `password` +
`password_confirmation`. No permission gates it — it only ever acts on the
caller's own row — but the current password is re-checked rather than trusted
from the session: a token proves who they are, not that whoever is at the
keyboard knows the password being replaced. Only the fact of the change is
audited, never the values.

## The KMS domain

Modul 2 (Content Management), FR-1..FR-9, plus Modul 3 (Search & Discovery)
and Modul 4 (Viewer & Protection) on top of it. Six packages for the content
itself:

```
spaces            one per team/product/domain      archived, never deleted
   │
categories        a tree, max 3 levels, ordered    ON DELETE CASCADE
   │
documents         title, slug, content (JSON), status, soft-deleted
   │
document_versions snapshots taken BEFORE an edit overwrites the content
   │
document_access   grants that RESTRICT a document to named subjects
   │
attachments       files on disk, kind = attachment | inline
   │
tags ──── document_tags   global vocabulary, created freely
```

| Package | Owns |
|---|---|
| `app/space` | The container. `DELETE /spaces/:id` archives — see below |
| `app/category` | The tree, its depth cap and its sibling ordering |
| `app/document` | The document, its lifecycle, its version snapshots, and its access grants |
| `app/tag` | The open vocabulary and the document_tags join |
| `app/attachment` | Files on disk: attachments and inline images |
| `app/search` | Modul 3: full-text search, autocomplete and search history |
| `app/protection` | Modul 4: the access trail, the share links, the protection level |
| `app/render` | Modul 4: one layout engine, a PDF canvas and a raster canvas |

Versions live inside `app/document` rather than a package of their own: a
version is only reachable through the document that owns it, and every write to
one is part of a write to the other — splitting them would put the two halves
of a single transaction in two packages.

**A space is archived, not deleted.** It owns documents, so dropping the row
would orphan or cascade-destroy every one of them. `DELETE /spaces/:id` sets
`is_archived`, and the list hides archived spaces unless `?include_archived=1`.

**The category depth cap is enforced in Go, not in the schema.** MySQL cannot
express "the parent's parent has no parent". `category.Service.checkParent`
walks to the root on every create and move, counting — which doubles as the
cycle check, since a move into your own subtree meets yourself on the way up.

**Reordering takes the whole ordered sibling list**, and renumbers it 0..n-1 in
one transaction. A partial list is refused with 422: renumbering half a level
leaves two categories sharing a position. Same reasoning as the curriculum
reorder this codebase used to carry.

**Slugs are derived from the name, never accepted from the client.** Two fields
naming the same URL is how they end up disagreeing. `utils.UniqueSlug` probes
`-2`, `-3` … and the unique index is what actually decides — the probe is
advisory, and the repository translates the index violation into a 409.
Document and category slugs are unique *per space*, not globally.

### The lifecycle is a table, not a switch

`app/document/transition.go` holds every legal move as data:

```
draft ──submit-review──> in_review ──approve──> published
  │                          │                      │
  │<────────reject───────────┘                      │
  │<───────────────────unpublish────────────────────┘
  └──publish──> published        (skips review; documents.review)
archived <──archive── any        restore ──> draft
```

Keeping it as data is what makes "which actions can I take right now"
answerable for the UI (`Available`) without restating the rules a second time.
The detail response carries `available_actions`, already filtered by what the
caller holds, so the client renders exactly the buttons that will work.

`approve` and `publish` both land on `published` and are deliberately separate
actions: approve finishes someone else's submission, publish skips review
entirely. The PRD grants both to Reviewer and up.

An illegal move is **422 naming both statuses** ("cannot publish a document
that is archived"), never a bare 403 — a caller who cannot tell "you lack the
right" from "wrong moment" cannot tell which one to fix.

`reject` requires a reason. A rejection with no note leaves the author nothing
to act on, which is the whole point of sending it back.

### Ownership is checked on the row, not on the route

The PRD's Contributor/Editor split became two global permissions:
`documents.update` (what I authored) and `documents.update.all` (anyone's).
Holding `.all` implies the narrower one.

A route gate cannot see whose document it is, so the check lives in
`user.User.CanEditDocument` and is called from the service. `middleware`'s own
`hasPermission` delegates to `user.User.HasPermission` so the route gates and
these row-level checks answer from one implementation.

> The PRD specifies per-space roles (Editor in one space, Viewer in another).
> This build deliberately uses the existing **global** RBAC instead, by
> decision — `space_members` is the change to make if that is revisited.

### The role ladder is seeded, not assumed

`database/seeders/role_permission.go` builds the PRD's five content roles by
extension, so the ladder is visible in one place instead of five hand-copied
lists that drift apart on the first edit:

| Role | Adds |
|---|---|
| Viewer | read documents, spaces, categories, and the lookups that fill the pickers |
| Contributor | `documents.store` + `documents.update` (what they authored) |
| Editor | `documents.update.all` instead — holding it already implies the narrow one |
| Reviewer | `documents.review` (approve / reject / publish / unpublish) |
| Space-Admin | archive, destroy, access.manage, Modul 4's three (`protection.manage`, `share`, `access-log.show`), and the `masters.*` writes |

The PRD's "System Admin" is `Super-Admin`, which short-circuits every check and
needs no row of its own.

Grants are filled **only for a role holding nothing**, so re-running the seeder
never overwrites what someone configured in the role editor. A grant naming a
permission that does not exist fails the seeder loudly rather than quietly
handing the role a smaller set than intended.

**A route that means "change this document" gates on `RequireAnyPermission`,
not on `documents.update` alone.** Editor and above hold only
`documents.update.all`, so the narrow gate locked out exactly the roles that
can do more — and the seeder's own tests could not see it, because they check
which permissions a role gets and never which gate stands in front of the
route. Whether the caller may edit *this* document is still settled per row in
the service, which is the only place that knows who wrote it.

### Only published documents are versioned

`Document.HasBeenPublished()` is `published_at != nil`, and `published_at` is
**never cleared on unpublish** so it doubles as "has this ever been published".
A draft nobody has seen has no reader to protect, so its edits are not
snapshotted (FR-5).

A snapshot holds the content the edit is about to **replace**, so a document
published once and never edited has zero versions — correctly, there is no
earlier state. `current_version_id` points at the most recent snapshot and
carries no foreign key: `documents` and `document_versions` reference each
other, and no insert order satisfies a FK both ways.

`Repository.SaveWithSnapshot` writes both in one transaction (NFR). The unique
key on `(document_id, version_number)` is what turns a concurrent second edit
into a rollback and a "reload and try again" rather than two versions sharing a
number.

### Diffing is over the visible text

`Flatten` walks the editor's node tree and returns one line per block. Diffing
the raw JSON would report every invisible attribute move as a change; diffing
rendered HTML would report every markup change as a text change. A reviewer is
comparing what a reader sees.

`Diff` is a plain LCS over those lines — no dependency, since the input is a
handful of paragraphs. It is O(n·m) in the line count; switch to Myers if
documents ever run to thousands of lines.

`compare?from=&to=` accepts `current` on either side as well as a version
number. With snapshots taken *before* an overwrite, the live content is not
itself a version row, so "what changed since v3" would otherwise be
unanswerable — and it is the comparison a reviewer actually wants.

### View counts are throttled in the read cache

`GET /documents/:id` increments at most once per user per document per hour
(NFR), keyed under its own `view:` prefix rather than `utils.ReadCachePrefix`
— clearing the read cache is a routine admin action, and sharing the prefix
would let anyone reset every throttle and inflate the counts on demand. Only
published documents are counted: an author reloading their own editor is not
readership. A failed increment never fails the read.

### Tags are a global open vocabulary

Anyone who may edit a document may tag it, and a name nobody has used becomes a
tag — that is what "bukan dari daftar tertutup" means (FR-6). `masters.tags.*`
governs the vocabulary itself: renaming it, and deleting a tag out of every
document at once. Neither gates tagging.

Identity is the **slug**, so "Deployment", "deployment" and "  deployment  "
are one tag rather than three. `FirstOrCreateBySlug` resolves a name to an
existing row, and the unique index settles the race when two requests both
miss the SELECT — the loser re-reads instead of inserting a near-duplicate.

`POST /documents/:id/tags` takes the **whole set by name**, and lives on
`document.Handler` rather than `tag.Handler`: retagging is an edit of the
document, and only `app/document` knows who authored the row the edit rule
turns on.

### Files never leave the storage root, and the uploaded name never reaches it

The uploaded filename is kept in a database column, where it is data. On disk
a file is `{uuid}{ext}` — a name can contain `..`, a separator, a NUL or 3,000
characters of unicode, and every one of those is a way to write outside the
store.

The stored **type comes from the bytes**, not the browser's `Content-Type`:
that header is a claim by the uploader, and a `.pdf` holding HTML would
otherwise be served back as a PDF and rendered as a page.

Layout is per space, not per document — a space is the unit anyone acts on
(granting, backing up, deleting), and a folder per document would be thousands
of directories holding one file each.

On the two failure orders that matter: an upload writes the **file first and
the row second**, removing the file if the row fails (a file with no row is
invisible litter); a delete removes the **row first and the file second** (a
row with no file 404s, which is recoverable).

### Download is a handler, not a static mount

`GET /attachments/:id/download` asks `app/document` whether this caller may see
the owning document before it streams anything. A static folder would hand the
whole store to anyone who guessed a path, and FR-9's restricted documents would
be restricted in name only.

It is the one route using `middleware.TokenOrHeaderAuthMiddleware`, which takes
either the Bearer header or `?token=`. An `<img src>` for an inline image and
an `<a download>` are issued by the browser, which attaches no Authorization
header. The query token is the short-lived WS-scoped one, so a URL that leaks
(a referer, a pasted link, shell history) stops working in minutes rather than
granting full API access — and the token is appended by the client at render
time, never stored in a document's content.

### Abandoned inline uploads are reclaimed, not orphaned

FR-8 uploads an image the moment it is dropped into the editor, filed against
the **space** rather than the document — the editor needs a URL before there is
a document to attach it to. Every draft abandoned after an image was dropped
therefore leaves a file nothing points at, and nothing else removes it: a
document delete cascades to its own attachments, but these belong to no
document to begin with.

`attachment.PruneOrphanUploadsJob` sweeps them hourly. Two details are
deliberate:

- **File first, then the row** — the opposite order to `Service.DeleteById`.
  There the row is what makes the file reachable, so losing the row first is
  recoverable; here nothing points at the file already, so dropping the row
  first would lose the only record of which file to delete.
- **A file that cannot be removed leaves its row alone** and is counted as a
  failure rather than returned as one. One unreadable path must not stop the
  sweep, and keeping the row means the next hour retries instead of stranding
  the file forever.

`DELETE /attachments/:id` also accepts an orphan, from whoever uploaded it.
It used to refuse outright — there is no document to check edit rights against
— which left every abandoned upload unremovable through the API by anyone.

### Access grants restrict, they do not widen

> FR-9's wording ("di luar aturan default role") reads both ways. The example
> beside it does not: "dokumen sensitif hanya untuk beberapa user tertentu
> meski mereka Viewer di space itu". So a document with **no** rows in
> `document_access` follows the ordinary permissions, and a document **with**
> rows is reachable only by the subjects listed. Adding the first grant narrows
> a document; it never widens one.

`AccessLevel` is ordered (`manage` > `edit` > `view`) so "may they edit" does
not have to list every level that counts as yes. A subject matched both
directly and through a role gets the **stronger** of the two — the grants are
permissions, and the narrower one was not meant as a cap.

Holders of `documents.access.manage` are never locked out. Without that escape
hatch a mistaken grant would need a database client to undo.

**The list and the detail must answer the same question.** `GetById` lets an
access manager through, so `GetAllPaginated` skips the hidden-id filter for
them too — a list that omitted a document the same caller can open by URL is
two different answers to "what is in this space", and the hiding one is what
makes a bad grant impossible to find and fix.

Filtering happens **in the query**, not after paginating: a page of 10 that
renders 7 is a bug report waiting to happen.

A restricted document the caller is not named on answers **404, not 403** —
telling the two apart confirms the document exists to exactly the people kept
out of it. The same stance `app/learning` took in the LMS this started from.

## Search & discovery (Modul 3)

`app/search` (FR-1..FR-3, FR-5, FR-6) plus `GET /documents/:id/related`
(FR-4), which lives in `app/document` because everything it needs — the tags,
the space, the access rule — is already there.

### The PRD says Postgres; this is MySQL

There is no `tsvector`, no `setweight`, no `ts_rank` and no `ts_headline`. The
equivalents:

| PRD (Postgres) | Here |
|---|---|
| `search_vector tsvector`, generated | `documents.search_body`, a `MEDIUMTEXT` filled in Go |
| GIN index | two InnoDB `FULLTEXT` indexes: `title`, `search_body` |
| `setweight` A vs C, `ts_rank` | `MATCH(title) * 3 + MATCH(search_body)`, ordered on as a select alias |
| `ts_headline` | `search.Snippet` — the window and the `<mark>` are cut in Go |
| a trigger to keep the index in sync | `Document.setContent`, in the write that was happening anyway |

**`search_body` exists because MySQL cannot index a JSON column as text.** It
is the flattened plain text of `content`, written by `setContent` beside the
excerpt — the same call, so the NFR's "ter-update dalam transaksi yang sama"
costs no trigger and no second write. Every path that changes content goes
through that one method precisely so the two derived fields cannot be
half-updated.

**The weighting is arithmetic, not a property of the index.** MySQL scores one
FULLTEXT index at a time, so a title match is worth three body matches by
multiplication at query time (`search.TitleWeight`).

**The ranking travels through a select alias.** Not decoration: gorm drops an
`ORDER BY` carrying bind variables *without reporting it*, and
`utils.Paginate`'s `COUNT` replaces the whole select clause and its variables.
Computing the score in the select and ordering on `score` is the one shape
where the sort applies and the count query stays valid. The same reason
`RelatedTo` orders on `shared_tags` / `same_category`, and its `HAVING` names
those aliases — MySQL only resolves a qualified column in a `HAVING` when it
happens to be in the select list.

**A query of only short words falls back to a `LIKE`.** Anything below
`innodb_ft_min_token_size` is not in the index at all, so "CI" would truthfully
answer "no results" for a word plainly in the document. `booleanQuery`
returning empty is the signal.

The limit is **read from the running server** in `NewRepository`, once, and
`DefaultMinTokenSize` (InnoDB's own 3) is only the fallback for a server that
will not answer. An install that tuned it down would otherwise keep taking the
LIKE fallback for words the index now holds; one that tuned it up would search
the index for words that are not in it. Refusing to boot over an unreadable
optimisation hint would be worse than either.

**Boolean-mode operators are stripped and every term gets a `*`.** `+`, `-`,
`*`, `"`, `(` and `~` are syntax there, so "C++ (draft)" would be a parse error
or a search for something else. Terms are OR-ed rather than each required with
`+`: a four-word question would otherwise find nothing, and the relevance score
already floats the document matching all four to the top.

### Search hides exactly what the space list hides

FR-3 is two rules. Only published documents are searchable, **except the
caller's own drafts** — deliberately not widened to reviewers, whose queue is a
list, not a search result. And restricted documents (FR-9) the caller is not
named on never appear: `RestrictedIDsExcluding` grew an optional space scope so
one query answers both "this space" and "everywhere", rather than search
restating a rule that already exists.

Searching gates on **`documents.show`** — it is reading documents. A second
permission naming the same capability is one more row to seed and one more to
forget. The history routes carry no permission at all: they only ever touch the
caller's own rows, exactly like `PATCH /profiles/password`.

### Snippets are HTML built here, and escaped here

`search.Snippet` escapes every character of the document and adds the `<mark>`
itself, in that order — escaping afterwards would escape the marks, and
escaping only the unhighlighted parts would ship a document's `<script>` to the
browser intact. It works on runes, because only a rune-wise `ToLower` leaves
the lowercased copy the same length as the original, and a match extends to the
end of its word since the index matched a prefix.

### Search history is one row per query, not an append log

`search_histories` carries `UNIQUE (user_id, query)` and the write is an
upsert, so searching the same thing twice moves one row instead of filling the
list with itself — which is also what gives `DELETE /search/history/{id}`
something stable to point at. `updated_at` is "last searched"; `created_at`
stays "first searched".

`search.PruneHistoriesJob` trims it nightly to `SEARCH_HISTORY_KEEP_PER_USER`
(50) rows per account. **Per user, not by age**: the ceiling is one account
with thousands of distinct phrases, which an age cutoff does nothing about —
and a cutoff long enough to keep a useful history is long enough to keep all of
them. One `DELETE ... JOIN (ROW_NUMBER() OVER (PARTITION BY user_id))`, not a
sweep per account: an unattended job that costs a round trip per user is one
that quietly stops finishing.

### document_views is the detail behind view_count, not a replacement for it

One row per counted read, written beside the `view_count` increment and under
the same once-per-hour throttle, so the number and the rows cannot disagree.
"Terpopuler" still sorts on `documents.view_count`: a `COUNT` over every read
ever recorded, to order one page, is the query that gets slower every day.

### The index backend is replaceable at `search.Engine`

`Search` and `Suggest`, taking a `search.Query` — a value with no HTTP and no
gorm in it. Swapping MySQL for Meilisearch is a second implementation of those
two methods and one argument in `NewContainer`.

**`Query` is why the interface is worth anything.** An interface that still
took a `*gin.Context` would be a seam only something already reading a Gin
request could sit behind, which is not a seam. `search.ParseQuery` is the one
place in the package that touches the request, and it reads `?page` and
`?per_page` through `utils.PageParams` so it cannot disagree with the pager
about what a valid page is.

**The search history is deliberately outside the interface.** Those are this
application's own rows and would stay exactly where they are if the index
moved to another process tomorrow. `NewService` therefore takes the engine and
the history store as separate arguments — the same `*Repository` twice today,
which is what the split looks like while MySQL is the only engine.

`utils.Paginate` gained `PaginateAt` for this: the request reading and the
paging were one function, and an engine that holds a page number rather than a
context needs the second half without the first. `Paginate` is now
`PageParams` + `PaginateAt`, so there is still one implementation of paging.

### The `tag` filter carries an id

The PRD writes `&tag={tag}`. It is the tag's **id**, like `space_id`,
`category_id` and `author_id` beside it — a slug here would be the one
identifier in the row that is not one, and the picker filling the filter is the
same lookup the rest of the app uses, which answers in ids.


## Viewer & protection (Modul 4)

Two packages plus a handful of routes on `app/document`. What each mechanism
actually promises matters more here than anywhere else in this codebase,
because a protection feature is the easiest kind to oversell:

| Mechanism | Rendered where | Can a reader defeat it |
|---|---|---|
| PDF watermark (FR-2) | server | no — it is in the file they were handed |
| Strict page images (FR-3) | server | no — the watermark is the same pixels as the words |
| Viewer overlay (FR-1) | the reader's browser | yes, in seconds, with dev tools |
| Disabled select / right-click (FR-3) | the reader's browser | yes, same |
| Anything at all | — | a photograph of the screen defeats all of it |

The overlay and the select-blocking are deterrents against a careless copy.
The PDF and the page images are evidence. The comments say so, the UI says so,
and nothing here should start claiming otherwise.

### app/protection knows nothing about documents

Resolving a level takes two stored values, recording an access takes a document
id, and opening a share link returns one. None of that needs to load a document
or decide who may read it — so `app/protection` imports nothing of ours, and
`app/document` is what calls into it. The dependency runs one way and the two
packages never have to import each other.

Everything that starts with "may this person see this document" therefore lives
in `app/document/protect.go`: the FR-9 grants, the author rule and the
lifecycle are all already there.

### The protection level is two columns, not the PRD's table

The PRD models FR-6 as a `DocumentProtectionSetting` table with a nullable
`document_id` standing for "the space default". It is `spaces.protection_level`
(NOT NULL) and `documents.protection_level` (nullable) here:

- MySQL treats NULLs as distinct in a unique index, so
  `UNIQUE (space_id, document_id)` would not stop a second, conflicting space
  default from being inserted. A column cannot disagree with itself.
- `GetById` joins the space for its level, so resolving one costs nothing
  beyond the read that was already happening. A table would add a query to the
  hot path of every document open.

`documents.protection_level` being NULL is what the PRD's absent row meant:
inherit. `created_by` is not carried — who changed a setting is what the audit
trail already records.

**Strict withholds the text, it does not just hide it.** `ContentVisibleTo`
drops `content` from the detail response for a caller who may only read a
strict document, and the client loads page images instead. Shipping the JSON
and then asking the browser not to let anyone select it would be a lock beside
an open window. Anyone who may *edit* still gets the text: they can rewrite the
document, so withholding it protects nothing and breaks the editor.

### One layout engine, two canvases

`app/render` lays a document out once and draws it through whichever backend
the level asks for — `pdf.go` for a real text PDF, `image.go` (gg + the Go
fonts, both compiled into the binary) for JPEG pages. `layout()` is the only
place a page break is decided, which is what makes the PDF a reader exports
match the pages they were looking at. Two renderers would agree on the day they
were written and never again.

The fonts are `golang.org/x/image/font/gofont` bytes rather than files on disk:
a PDF that depends on a font file renders differently — or not at all —
depending on what the container image happens to ship.

`document.Blocks` is what turns the editor's node tree into `render.Block`s. It
sits beside `Flatten` (diff.go) rather than replacing it: a diff deliberately
discards the block type, and a reviewer's tool that started reporting "this
paragraph became a heading" as a text change would be a regression.

A leading heading that repeats the document's title is dropped — that shape
comes straight out of the editor, and printing both puts the same words twice
at the top of every export.

### Two tables, because a view count and a trail are not the same thing

`document_views` (Modul 3) is written under a once-per-hour throttle so it and
`documents.view_count` cannot disagree. `document_access_logs` (FR-4) is
throttled by nothing: an investigation asks "who opened this, how often, from
where", and a table that drops 59 minutes of every hour cannot answer it.

Their foreign keys differ for the same reason. `document_views.user_id`
cascades — the aggregate that survives a deleted reader is already in
`view_count`. `document_access_logs.user_id` is ON DELETE SET NULL, because the
row *is* the evidence and removing an account must not be a way to erase the
trail behind it.

**Append-only means there is no route that writes one any other way.** MySQL is
not asked to enforce it: a trigger that refuses UPDATE still yields to whoever
could drop the trigger. The repository has a `RecordAccess` and no update or
delete beside it.

**`print` is the one action a client reports**, because a browser print never
reaches the API at all. `POST /documents/:id/access-log` accepts nothing else —
an endpoint that took any action would let a client write rows describing
things that never happened, and a trail anyone can forge into is worse than one
with a known gap. Everything else is recorded where it happens: `view` in
`Show`, `export_pdf` in the export, `download` in the attachment handler (real
attachments only — an inline image is fetched once per `<img>` per page load
and would bury the trail in its own noise).

Recording is best effort and logged on failure. The trail lives in the same
database as the document that was just read, so a write that fails here almost
certainly means the read failed too — and nobody should see an error page
because a log row did not save.

### A share link stores its hash, which is what moved the revoke route

FR-5's token travels in a URL: browser history, a referer header, wherever the
recipient pastes it. `view_sessions.token_hash` is its SHA-256, so a copy of
the table is not a set of working links, and the plaintext exists exactly once
— in the response that created it.

SHA-256 rather than `utils.MakeHash`'s bcrypt: a password is low entropy and
needs the work factor, a 256-bit random token has nothing to guess. Bcrypt here
would only make every link open pay ~60ms and rule out the unique index.

That is why revoking is `DELETE /documents/:id/view-sessions/:sessionId` rather
than the PRD's `DELETE /view-session/{token}` — nothing can find a row from a
token nobody kept, and the id is what the person revoking is actually looking
at.

`GET /view-session/:token` is the only route in the application reachable with
no account, and deliberately not a session on the API: it opens one document
and nothing else. The FR-9 grants are not re-checked there — they restrict a
document to named *accounts*, and the recipient has none; the check that
mattered ran when whoever created the link proved they could read it. An
expired link answers 403 ("this expired") while a token that never existed
answers 404, so nothing confirms a guess.

`prune-view-sessions` sweeps expired rows nightly. Housekeeping, not a control:
`OpenSession` checks the clock, not the presence of the row.

### There is no GET for either protection setting

The PRD lists `GET /spaces/{spaceId}/protection-setting`. The space detail
already carries `protection_level`, and the document detail carries both the
resolved level and the override — a second endpoint returning one field of a
resource the client is already holding is one more thing to keep in step.

## Ids validate as `uuid`, not `uuid4`

Seeded accounts carry UUIDv5 (a stable id derived from the seed code — see
`database/seeders/user.go`), and ids imported from another system may be v1.
Demanding v4 rejects real rows for the version nibble alone.

## Routes

```
POST /api/auth/login                — email + password
GET  /api/auth/google/redirect      — trigger Google login (browser redirect)
GET  /api/auth/google/callback      — Google OAuth callback (backend only)
GET  /api/auth/dev-login?email=...  — local dev only (APP_ENV=local), no password
POST /api/auth/refresh              — [refresh token] new access token
POST /api/auth/logout               — [protected] blacklists the current pair
POST /api/auth/ws-token             — [protected] issues a short-lived token scoped to WS routes only
GET  /api/profiles/me               — returns authenticated user [protected]
PATCH /api/profiles/password        — change own password [protected]

GET/POST                /api/spaces                 — DELETE /:id archives
GET/PUT/DELETE          /api/spaces/:spaceId
GET/POST                /api/spaces/:spaceId/categories
PATCH                   /api/spaces/:spaceId/categories/order   — whole ordered sibling list
PUT/DELETE              /api/categories/:id
GET/POST                /api/spaces/:spaceId/documents
GET/PUT/DELETE          /api/documents/:id          — DELETE is a soft delete
POST                    /api/documents/:id/{submit-review|approve|reject|publish|unpublish|archive|restore}
GET                     /api/documents/:id/versions
GET                     /api/documents/:id/versions/compare?from=&to=   — a number, or `current`
GET                     /api/documents/:id/versions/:versionNumber
POST                    /api/documents/:id/versions/:versionNumber/rollback
POST                    /api/documents/:id/tags     — whole set, by name
GET/POST                /api/documents/:id/attachments
GET/POST                /api/documents/:id/access
DELETE                  /api/documents/:id/access/:accessId
GET                     /api/documents/:id/related  — tag overlap, then same category

GET                     /api/documents/:id/export-pdf         — watermarked, per caller
GET                     /api/documents/:id/pages              — strict viewer: page images as data URLs
PUT                     /api/documents/:id/protection-setting — absent level clears the override
GET                     /api/documents/:id/access-log         — the trail [documents.access-log.show]
POST                    /api/documents/:id/access-log         — client-reported print, nothing else
GET/POST                /api/documents/:id/view-sessions      — share links [documents.share]
DELETE                  /api/documents/:id/view-sessions/:sessionId
PUT                     /api/spaces/:spaceId/protection-setting
GET                     /api/management/users/:id/access-log  — the same trail, by reader
GET                     /api/view-session/:token              — no account needed; the one such route
GET                     /api/search?q=&space_id=&category_id=&tag=&author_id=&from=&to=&sort=
GET                     /api/search/suggest?q=      — title prefix match
GET                     /api/search/history         — own rows only, no permission
DELETE                  /api/search/history/:id
GET/POST                /api/tags
DELETE                  /api/tags/:id
POST                    /api/uploads/inline-image   — multipart: file + space_id
GET                     /api/attachments/:id/download  — Bearer header OR ?token=
DELETE                  /api/attachments/:id
```

## JWT (gofortify)

- Requires `JWT_SECRET_KEY` and `JWT_SIGNING_METHOD=HS256` in `.env`
- `access_token_verifier` cookie is required alongside the Bearer token for validation
- `AuthMiddleware` reads `Authorization: Bearer <token>` header + `access_token_verifier` cookie
- User is stored in gin context as `*models.User` via `c.Set("user", &user)`

## Error Handling

`errors/exception.go` builds `*utils.HTTPError`, whose `Code` decides the HTTP
status via `enums.ResponseCode.HTTPStatus()`. `middleware/ErrorHandler` renders it
and logs it.

### Never put `err.Error()` in the message

`Message` is sent to the client. The underlying error goes in the cause instead,
which is unexported and therefore never serialized:

```go
return nil, errors.InternalServerError("failed to add new course").WithCause(err)
```

`Error()` appends `, cause: ...` so the log gets the full story while the response
keeps the safe message. `Unwrap()` keeps `errors.Is` working through the wrap.

### Who translates errors

**Repositories translate every error they can name**, rather than leaving it to
the caller. One repository method has many callers, so translating at the call
site means many chances to forget — and a raw error that reaches `ErrorHandler`
gets only a generic `500`, losing the message that would have explained it.

The exception is an error from a database primitive that has no meaningful name:
gorm's `Transaction` returns a raw error when `Begin` or `Commit` fails, which is
always infrastructure (pool exhausted, connection dropped before `COMMIT`, server
restarting) and never something the client can act on. Return it as is — a
generic `500` is the correct answer and `logError` still records the detail.

A query that can miss must separate the two cases — otherwise a database outage
is reported as `404`:

```go
if err := ...First(&role).Error; err != nil {
    if stderrors.Is(err, gorm.ErrRecordNotFound) {
        return nil, errors.EmptyDataException("role not found").WithCause(err)
    }
    return nil, errors.InternalServerError("failed to get role by id").WithCause(err)
}
```

This applies to `First` / `Take` / `Last`. `Find` returns no error on an empty
result, so those may wrap everything as `InternalServerError`.

Callers pass an `*utils.HTTPError` straight up with `return nil, err`. Re-wrapping
it discards both the cause and the status the repository already chose.

`utils` cannot import `api-kms/errors` (that package imports `utils`), so
`utils/paginate.go` builds its errors with local helpers `queryFailed` and
`invalidQueryParam`.

### APP_DEBUG

With `APP_DEBUG=true` every error response gains a `debug` field holding the raw
error — the cause for an `HTTPError`, the error itself for anything else:

```json
{"code":"ERR_INTERNAL_SERVER_ERROR","message":"failed to create course","status_code":500,
 "debug":"Error 1054 (42S22): Unknown column 'foo' in 'field list'"}
```

`omitempty` means responses are byte identical to the flag not existing when it is
off. Parsing fails closed: missing, empty, or unparseable all mean off, so a
broken `.env` in production cannot switch it on.

It is deliberately a flag of its own rather than an `APP_ENV` allowlist, so a
staging box that is reachable from the internet can run with detail off.

`middleware/withDebug` is the only place allowed to consult `config.DebugEnabled`.
Keeping the single gate at the rendering boundary is what stops the decision from
being duplicated somewhere it will be forgotten.

### ErrorHandler branch order

`errors.As(originalErr, &httpResponse)` must stay the **first** branch. `Unwrap`
makes `errors.Is` see through `WithCause`, so a later `gorm.ErrRecordNotFound`
check would hijack an `EmptyDataException("course not found")` and replace its
message with the generic "Data not found". The `io.EOF` and `ErrRecordNotFound`
branches are fallbacks for raw errors that never got wrapped.

The final fallback must not put `originalErr.Error()` in the response — that sent
driver text like `Unknown column 'canceled_by_id'` to the client. `logError`
already recorded the real error.

### Context in repositories

Repository and service methods take `context.Context`; the handler converts with
`c.Request.Context()`. Do not pass `*gin.Context` into `db.WithContext`: the
engine leaves `ContextWithFallback` at its default `false`, so `gin.Context.Done()`
returns nil and the query never cancels when the client goes away.

`GetAllPaginated` is the one exception — it reads `page` / `per_page` / filters
straight off the request, so it needs the `*gin.Context`. `utils.Paginate` calls
`WithContext(c.Request.Context())` internally, so its queries still cancel.

### Import alias

The local package `api-kms/errors` shadows stdlib `errors`. Where both are
needed, stdlib is aliased to `stderrors` and the local package keeps its own name.

### Middleware

`AuthMiddleware` and `RefreshMiddleware` report failures with `abortWithError`
(`c.Error` + `c.Abort`), not `c.AbortWithStatusJSON`. `ErrorHandler` is registered
ahead of them in `main.go`, so writing the response directly would skip `logError`
and the rejection would never appear in `go.log`.

**Anything about the token answers `401`** — missing, malformed, expired, wrong
type, or a subject whose user row is gone. Only `ERR_AUTHENTICATION` constructors
belong on those paths (`InvalidCredential`, `TokenExpiredException`,
`InvalidTokenTypeException`). `UnauthorizedException` is `ERR_ACTION_UNAUTHORIZED`
→ `403` and means the caller is authenticated but lacks permission; do not use it
for token failures. The frontend keys its "log in again" redirect off this split.

A failed subject lookup must abort. Letting it fall through leaves a zero valued
`user.User` in the context and the request runs as nobody.

## Read Cache

A handful of repository reads are served from an in-process map
(`utils.Remember` / `utils.Forget`), not Redis. The app runs as a single
container (`.gitlab-ci.yml` is a plain `docker compose up -d`, one `web`
service, the same assumption the in-process notification and activitylog hubs
already make), so a Redis round trip plus a decode would cost more than the
small SELECT it replaces.

### Cached in the repository, not per endpoint

`/api/masters/regions` and `/api/lookups/regions` run the *same query* and
differ only in what they map the rows to. Same for areas, cash-and-banks,
roles and permissions. So the cache sits on the repository method: one query,
one key, and each caller maps the shared slice to its own shape. Caching per
endpoint would run every query twice and hold the same rows under two keys.

Callers must treat what comes back as **read only** — the slice is shared.

| Key | Repository method | TTL |
|---|---|---|
| `read:regions` | `RegionRepository.GetAll` | 1h |
| `read:areas` | `AreaRepository.GetAll` (unfiltered only) | 1h |
| `read:cash-and-banks` | `CashAndBankRepository.GetAll` (unfiltered only) | 1h |
| `read:roles` | `RoleRepository.GetAll` | 5m |
| `read:permissions` | `PermissionRepository.GetAll` | 5m |
| `read:users` | `user.Repository.GetAll` | 5m |
| `read:users:lookup` | `user.Repository.GetLookup` | 5m |
| `read:users:by-role:<role>` | `user.Repository.GetByRole` (unfiltered only) | 5m |

Every key carries `utils.ReadCachePrefix` (`read:`), so one `Forget` clears the
lot and no writer has to name the key it invalidates.

TTLs are a backstop, not the invalidation mechanism — they only bound a row
changed outside this process. Which one a key gets depends on who owns the
table: `utils.OwnTableCacheTTL` for tables this app created,
`utils.SharedTableCacheTTL` for tables another app may also write.

`GET /api/management/users` returns the whole list rather than a page of it —
one row per employee, filtered in the browser like the roles and master lists.
That is what made it cacheable: a per-page, per-search query never was.

### Only unfiltered reads are cached

`AreaRepository.GetAll`, `CashAndBankRepository.GetAll` and
`user.Repository.GetByRole` take filters that arrive off a query string. Keying
the cache by them would let any caller grow the map without limit, so a
filtered call goes straight to the database. This costs nothing in practice:
the master pages hold the whole list and filter in the browser.

### Invalidation runs after the write, never in a hook

`rbac.Service` and `user.Service` call `utils.Forget(utils.ReadCachePrefix)`
once the repository has returned.

This was a GORM `AfterSave`/`AfterDelete` hook on the models first, and that is
wrong twice over:

- Model hooks run **inside** the write's transaction. A read arriving between
  the hook and the `COMMIT` re-caches the pre-write rows and holds them for a
  whole TTL.
- `RoleRepository.UpdateById` saves the role and *then* replaces its
  permissions, so a hook on the save fires before half the change exists.

The cost of doing it explicitly is that a new write path has to remember the
call. A write that can change any cached read needs one
`utils.Forget(utils.ReadCachePrefix)` after it succeeds.

`permissions` is the exception with no invalidation at all: the repository is
read only and the rows come from `database/seeders`, which runs as its own
binary. The TTL is the whole story there.

### Clearing it by hand

```bash
docker compose kill -s HUP web          # server shell
DELETE /api/management/cache            # browser, management.cache.destroy
DELETE /api/management/cache?key=read:users:lookup
```

`GET /api/management/cache` (`management.cache.show`) lists the live keys with
their item count and remaining TTL; `/account/cache` in `../web-kms` renders it.
`watchCacheClearSignal` in `main.go` is the SIGHUP half — it cannot be a binary
under `cmd/`, since the cache is a map in the serving process and a second
process would only clear its own empty copy.

## Scheduler

`app/scheduler` has no cron dependency, just a `time.Ticker` per
job (`scheduler.Runner`, started from `main.go`, one goroutine per job so each
keeps its own interval). Nothing here needs cron-expression scheduling: a
job's `Interval()` is just a fixed `time.Duration`.

| Job | What it does |
|---|---|
| `prune-orphan-uploads` | Deletes inline images that never reached a document, older than `UPLOAD_ORPHAN_RETENTION_HOURS` (default 24). Hourly, not daily: the window is measured in hours because it only has to outlive a writing session. See below. |
| `prune-view-sessions` | Deletes expired share links (Modul 4 FR-5). Housekeeping rather than a control — an expired row grants nothing, since `OpenSession` checks the clock. Daily, in the quiet hours. |
| `prune-job-logs` | Deletes `job_logs` rows older than `SCHEDULE_JOB_LOG_RETENTION_DAYS` (default 3). Runs once a day, not every minute — retention is measured in days, so minute-level polling would only add this job's own bookkeeping rows faster. Without it `job_logs` grows forever: two minute-jobs alone write ~2,880 rows/day. |

Retention is a plain env var (`SCHEDULE_JOB_LOG_RETENTION_DAYS`) — this app
has no runtime-editable configuration store yet (`management.configurations.*`
is seeded but unbuilt), so it is sized the same as `JWT_ACCESS_TOKEN_TTL`.

### Run history (`app/joblog`)

Every run (success or failure, scheduled or panicked) is written to
`job_logs` and pushed live, the same shape as `app/activitylog`'s audit
trail: `Repository` + `Hub` (WebSocket fan-out) + `Broadcaster` (persist then
push). Not the same table as `activity_logs` — that one is HTTP-request
shaped (method/path/status_code), which a background run has none of.

`type` (`schedule` | `queue`) is a discriminator, not two tables: a future
queue system is "a unit of background work ran", the same shape as a
scheduled run, so it logs to the same table once one exists — nothing in
`app/joblog` writes queue rows today.

```
GET /api/management/job-logs              — paginated run history [management.job-logs.show]
GET /api/management/job-logs/definitions  — static list of registered jobs + interval
GET /api/management/job-logs/ws           — live feed, same auth-via-?token= as /management/audits/ws
```

A panic inside `Job.Run` is recovered in `Runner.runOnce` and logged as a
failed run rather than killing that job's ticker goroutine — without it, one
bad run would silently end that job forever with nothing in `go.log` to
explain why it stopped ticking.

## Resources Pattern

All API responses go through resource structs in `app/resources/`:
- `UserResource` — `id`, `full_name`, `first_name`, `last_name`, `email`

### Formatting helpers (`app/resources/format.go`)
- `formatDateTime(*time.Time) *string` — format `02/01/2006 15:04:05`, nil-safe
- `formatDateTimeVal(time.Time) string` — same for non-pointer
- `formatDate(*time.Time) *string` — format `2006-01-02`, nil-safe
- `formatRupiah(float64) string` — formats as `Rp 1.500.000`

All formatted fields are prefixed with `f_` (e.g. `f_amount`, `f_created_at`).

## Getting a database populated

```bash
go run ./cmd/seed    # reference data: permissions, roles, Super-Admin grants, QA users
```

In the production image this is `./seed`.

`database/seeders` holds the permission list, the roles and their grants — real
reference data, safe to re-run. `User` is the one exception: fixture QA accounts
with no `APP_ENV` guard. Drop it from the registry in `database/seeders/seeder.go`
before running `seed` against a production database.

## Database Migrations

Migrations live in `database/migrations/`. Local dev uses `gartisan` (a thin
wrapper over `golang-migrate`, also used for scaffolding new migration files):

```bash
gartisan migrate                      # or migrate:up
gartisan migrate:status               # current version and dirty state
gartisan migrate:up force <version>   # mark a version applied without running it
gartisan migrate:down [n]
gartisan migrate:drop                 # drop every table, asks first
gartisan make:migration <name>
gartisan db:seed                      # runs go run ./cmd/seed
```

Production/CI has no `gartisan` dependency — the Docker image installs the plain
`golang-migrate` CLI instead and runs it directly against
`GARTISAN_DATABASE_URL`. Both read and write the exact same
`file://database/migrations` source and the same `schema_migrations` row, so
either tool sees the state the other left:

```bash
migrate -path database/migrations -database "$GARTISAN_DATABASE_URL" up
```

### There is one account flag, not two

`users.membership_status` decides whether an account may be used at all, and
login is what checks it. Nothing else gates access.

`users.is_active` used to sit beside it — ng-sales' operator availability flag,
which decided who was in the round robin that got assigned work. This app has
no such queue and never read the column, so it was dropped
(`20260918000009_drop_is_active_from_users_table`) along with the scheduled job
that switched it off for idle accounts.

If "active student vs alumni" is ever needed, it belongs on the per-term
classroom membership, not on a global flag: a student is enrolled *this term*,
not enrolled in the abstract.

### RBAC is not Spatie's schema

`20260610000001` started as a copy of Spatie's tables; it is now built for
what this app actually uses. Three things went away, and none of them should
come back without a reason:

- **`model_type`.** Spatie's pivots are polymorphic because a role can attach
  to any model. Here a role only ever belongs to a user, so the column held one
  constant on every row. `model_has_roles` is now a plain `user_roles`
  (`user_id`, `role_id`) — which can carry a real foreign key to `users`, as
  the polymorphic version could not. GORM's defaults line up with that shape,
  so `User.Roles` needs only `gorm:"many2many:user_roles"`, and the
  `ModelHasRole` join model plus its `SetupJoinTables` call in `main.go` are
  gone.
- **`model_has_permissions`.** Permissions reach a user only through their role
  (`user_roles` → `role_has_permissions` → `permissions`). Nothing ever granted
  one directly, so no code read the table.
- **`guard_name`.** This app has no guards. The column existed to satisfy
  Laravel's `NOT NULL` and its `(name, guard_name)` unique key; the unique key
  is on `name` alone now, and the `BeforeCreate` hooks that defaulted it are
  gone.

## Environment Variables

```env
APP_ENV=local                     # set to non-local in production to disable dev-login
APP_DEBUG=false                   # true adds `debug` (the raw error) to error responses; never true in production
APP_PORT=8000

DB_USER=
DB_PASSWORD=
DB_HOST=localhost
DB_PORT=3306
DB_NAME=

OAUTH_GOOGLE_CLIENT_ID=           # from Google Cloud Console OAuth client (confidential, web application type)
OAUTH_GOOGLE_CLIENT_SECRET=
OAUTH_GOOGLE_REDIRECT_URL=        # must match an authorized redirect URI on that client

FRONTEND_URL=                     # e.g. http://localhost:3000

JWT_SECRET_KEY=
JWT_SIGNING_METHOD=HS256
JWT_ACCESS_TOKEN_TTL=30
JWT_REFRESH_TOKEN_TTL=10080
JWT_REDIS_HOST=localhost
JWT_REDIS_PORT=6379
JWT_REDIS_PASSWORD=
JWT_REDIS_DB=0
JWT_BLACKLIST_INCIDENT_TIME_KEY=

SCHEDULE_JOB_LOG_RETENTION_DAYS=3               # prune-job-logs job; defaults to 3 if unset/invalid

STORAGE_PATH=storage             # root every attachments.file_path is relative to; mount as a volume
UPLOAD_MAX_SIZE_MB=10            # attachments (FR-7)
UPLOAD_IMAGE_MAX_SIZE_MB=5       # inline images (FR-8) — decoded by every reader on every view
UPLOAD_ORPHAN_RETENTION_HOURS=24 # prune-orphan-uploads; hours, not days — it only outlives a writing session

VIEW_SESSION_MAX_TTL_HOURS=720   # Modul 4 FR-5: ceiling on a share link's expiry. 720h = 30 days

```

## Dev Login (local only)

```
GET http://localhost:8000/api/auth/dev-login?email=user@example.com
```

Returns `access_token` directly. Only works when `APP_ENV=local`. Returns 404 otherwise.
Use this in Postman to get a token without going through the browser OAuth flow.
