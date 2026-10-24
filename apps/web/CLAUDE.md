# web-kms

Nuxt 4 SPA (`ssr: false`) — the frontend for the KMS (knowledge management system). The API lives at `../api-kms`.

Started as a copy of `lms-web` with the LMS pages removed. What is left is the
base — auth, account/RBAC and the operational pages — and the KMS pages go on
top of it.

---

## The KMS pages

Modul 2 (Content Management), FR-1..FR-9, Modul 3 (Search & Discovery) and
Modul 4 (Viewer & Protection).

| Route | What |
|---|---|
| `/spaces` | Space cards, search, archived toggle, create/edit/archive modals |
| `/spaces/[id]` | The category tree beside the space's document list |
| `/documents/[id]` | The document: viewer/editor, lifecycle, tags, attachments, versions, diff, access, related |
| `/tags` | The tag master — rename and prune the vocabulary |
| `/search` | Modul 3: full-text results with snippets, filters and sorting |
| `/shared/[token]` | Modul 4: one document, opened by share link, no account |

Modul 3 also lives in the ⌘K dialog (`base/quick-search-dialog`) — see below.

### Search: the highlight is the API's, and ⌘K is where the box already was

`snippet` arrives as HTML with the matched words already wrapped in `<mark>`
and every other character escaped, so `/search` renders it with `v-html`.
Highlighting here instead would need the client to know how the query was
tokenised, which only the index does.

The global search box is the **existing ⌘K dialog**, extended rather than
replaced: it already had the input, the shortcut, the recents list and the
navigation. It now also shows document titles (`/search/suggest`, debounced
250ms) while typing, this user's last queries (`/search/history`) when the box
is empty, and a "see all results" row that hands off to `/search` — which is
always offered, including when nothing matched, because the dropdown only
completes *titles* while `/search` reads the content as well.

`/search` is built on `useServerList`, so every filter, the sort and the page
live in the URL and a result list is a shareable link. The relevance option is
hidden when the box is empty: the API sorts by newest then whatever the client
asks, and an option that is quietly ignored is worse than one that is not
there.

The category filter's lookup is handed the space filter's id **by reference**
and reloaded when it changes — offering another space's categories would be a
filter that can never match.

### The API decides which buttons exist

The document detail response carries `available_actions` — the lifecycle moves
legal from this status, already filtered by what the caller holds — and
`can_edit`, which answers the author-vs-anyone rule the client cannot (it does
not know who wrote the row).

`app/pages/documents/[id].vue` renders exactly those. `utils/document.ts` says
only what each button *looks* like; re-deriving *which* ones to show is how the
two copies of the rules drift apart.

### Every write replaces state from the response

`DocumentService.updateById`, `transition` and `rollback` all set `document`
from what comes back rather than patching the local copy. Status,
`available_actions` and the version pointer move together on most of these, and
reassembling them here would be a second implementation of the lifecycle.
`CategoryService.reorder` does the same with the whole tree.

### Reordering is up/down buttons, not drag

No drag-and-drop library is installed, and buttons work on touch and with a
keyboard for free. The API takes the whole ordered sibling list either way, so
swapping in drag later changes nothing server-side.

`CategoryTree.vue` is recursive (an SFC may refer to itself by filename) and
stops offering "add subcategory" at the third level, matching the API's cap.

### The editor is TipTap, and content is its JSON

`RichTextEditor.vue` wraps `@tiptap/vue-3` + StarterKit and is the *same*
component in read-only mode, so a document never looks different depending on
who is looking at it. `v-model` carries the ProseMirror node tree, stored
as-is — never HTML, so it diffs per node and can be re-rendered by any editor.

The incoming-value watcher compares against `editor.getJSON()` before calling
`setContent`: the parent replaces content wholesale after a save or rollback,
and writing it straight back would move the caret on every keystroke.

Node styles are global CSS namespaced by `.kms-prose` — scoped styles cannot
reach TipTap's generated nodes.

### Tags are edited as a whole set

`DocumentTags.vue` emits the *new list*, never "add this" or "remove that" —
the API replaces the set, so the client never has to describe a change. A name
already on the document is dropped before the request rather than sent: the API
would fold it anyway, and a pointless request is a pointless toast.

The answer can differ from what was sent, because a name resolves to an
existing tag with whatever casing that one has. The chips are repainted from
the response.

### File URLs are resolved, not used as-is

The API stores a bare `/api/attachments/{id}/download`. `useAttachmentUrl`
prepends the API origin and appends the short-lived WS-scoped token, because a
browser-issued request — `<img src>`, `<a download>` — carries no
Authorization header.

The token is **never written into a document's content**. A token in stored
content would leak with every copy of the document and expire minutes later,
leaving a permanently broken image. `RichTextEditor` keeps the bare path on the
node and rewrites the rendered `<img>` src after each render instead.

One token is fetched per page, not per file: a document with twelve
attachments would otherwise ask for twelve tokens to draw one list.

### Inline images arrive by drag, paste or button

All three go through one `insertImage`, which uploads first and inserts the
stored path second — so the node in the document is never a `blob:` or
`data:` URL that would be saved as content.

The editor only offers images when it has a `spaceId`: images are filed against
a space, and without one there is nowhere to put them.

### The access modal says which way the rule runs

Adding the first person **restricts** the document rather than widening it,
which is the opposite of what "grant access" suggests. The modal states which
state it is in — open or restricted — above the form, because the wrong
reading here locks a team out of its own document.

The document header shows only a lock badge. Who is on the list is gated on
`documents.access.manage` separately, so a reader who is allowed in never sees
who else is.

### Diffing offers `current` alongside the version numbers

Snapshots are taken *before* an overwrite, so the live content is not itself a
version row. Without `current`, "what changed since v3" — the comparison a
reviewer actually wants — would be unanswerable.

### Nullable columns are held as `''` in forms

An input can only bind a string, so `description`, `icon` and `category_id` are
`''` in form state and sent as `null`. A cleared field then clears the column
instead of storing an empty string. `v-model` cannot bind a type assertion
(`v-model="x as string"` is not a valid assignment target), which is what
forces the separate form type in `ModalFormSpace`.


## Viewer & protection (Modul 4)

Four components beside the document page, one service, one public route:

| File | What |
|---|---|
| `DocumentProtection.vue` | The tiled watermark overlay plus the select/right-click blocking (FR-1, FR-3) |
| `DocumentPages.vue` | The strict viewer: server-rendered page images (FR-3) |
| `ModalProtectionSetting.vue` | The per-document level, including "inherit" (FR-6) |
| `ModalShareLink.vue` | Expiring links, shown once (FR-5) |
| `ModalAccessLog.vue` | The trail for one document (FR-4) |
| `ProtectionService.ts` | All of the above, plus the PDF export and the print report |

### Say what the overlay is, not what it looks like

The watermark in `DocumentProtection.vue` is a CSS overlay in the reader's own
browser. Anyone who opens dev tools deletes it in seconds, and so does anyone
who disables the `select-none`. It raises the cost of a careless copy; it is
not evidence, and the copy on these screens never implies it is.

What cannot be defeated client-side is what the server draws: the exported PDF
and, at the strict level, the page images. `ModalProtectionSetting` states the
difference in the dialog itself, because that is the moment somebody decides
which one they are relying on.

The overlay text is built here from the logged-in account rather than fetched.
It is a deterrent either way, so asking the API for a string it would only echo
back would buy nothing — and the watermark that matters is drawn server-side
from the token, where the client cannot reach it.

The tile is **one SVG data URL repeated by the browser**, not a few hundred
absolutely positioned spans: one element, and nothing to reflow. A small
per-reader offset and angle come from hashing the watermark text, so two
people's overlays do not line up pixel for pixel (FR-1) and the same reader
sees a steady pattern rather than one that jumps on every render.
`pointer-events-none` and `aria-hidden` are not decoration — a watermark that
swallows clicks is one somebody will ask to have removed, and a screen reader
repeating an email address every few words is unusable.

### A strict document has no text to render

The detail response drops `content` and sets `content_hidden` for a caller who
may only read a strict document, so the page renders `DocumentPages` instead.
Without the flag an empty content tree would be indistinguishable from an empty
document. An editor still gets the editor: they can rewrite the text, so
withholding it protects nothing.

`DocumentPages` takes an optional `pages` prop because the shared-link page
already has them in its response — one render, not two.

### Pages arrive as data URLs, which is why there is no token dance

A browser-issued `<img src>` sends no Authorization header. That is the whole
reason `useAttachmentUrl` has to append a short-lived WS token; embedding the
JPEG bytes in the JSON response sidesteps it entirely, and the pages were
rendered in one pass anyway. Same reason the PDF is fetched through `$api` as a
blob rather than pointed at with an `<a href>`.

### The share token is shown once

`ModalShareLink` holds the link between creating it and closing the dialog, and
throws it away after. The API stores only its hash, so there is nothing to look
up later — the dialog says so rather than implying the link can be recovered.
Revoking is by the link's id, for the same reason.

### Print is reported, everything else is observed

`documents/[id].vue` listens for `beforeprint` and posts `action: print`. It is
the one access the server cannot see for itself, and the row is a hint rather
than evidence: a reader who blocks the request simply does not appear. The call
swallows its own failure — nobody printing a document should get an error
banner because a log row did not save.

### The space's level is saved through its own endpoint

`ModalFormSpace` holds `protection_level` in form state but sends it with a
second request, and only when it changed. `PUT /spaces/{id}` is a full replace,
so folding one enum into it would carry this client's copy of the name and
description along with it. The field only appears on edit — a new space takes
the server's default, and there is nothing to set it on until it exists.

## Auth

- Token stored in `localStorage` under key `access_token` (NOT `useCookie` — unreliable in async Nuxt 4 contexts).
- OAuth SSO flow: frontend redirects to Go backend (`$config.public.apiBase/api/auth/{provider}/redirect`) using `navigateTo(..., { external: true })`. PKCE is handled entirely by the backend.
- Callback page at `/auth/callback` reads `?access_token=` or `?error=` from query params.
- API plugin (`app/plugins/api.ts`) reads token from `localStorage` and sets `Authorization: Bearer <token>`.

---

## API Plugin Pattern

`$api` in `app/plugins/api.ts` sets headers on every request. Caller-supplied headers win because `...options.headers` comes last:

```ts
options.headers = {
  'Content-Type': 'application/json',
  Accept: 'application/json',
  Authorization: token ? `Bearer ${token}` : '',
  ...options.headers,
}
```

---

## Page Conventions

Follow `app/pages/account/roles.vue` as the reference pattern for list pages:

```ts
definePageMeta({
  title: 'Page Title',
  pageSubTitle: 'Short description',
})
```

- Use `TH` with `icon` prop for all table headers: `<TH icon="Hash">No.</TH>`
- Use `TD` for table cells — both wrap the base `TableCell` component.
- Use `SmallInfo` for secondary/helper text inside cells.
- Action buttons: `ButtonEdit`, `ButtonDelete` components.
- Modal pattern: `useTemplateRef` + `handleModal(true/false, data?)`.

---

## Form Rules

Three rules, no exceptions worth the inconsistency:

**1. Every select is `TomSelect`, never a bare `<select>`.** It carries the app's
input styling, gives every dropdown type-to-search for free, and renders its menu
into `<body>` so a long list is not clipped by the dialog it sits in. Options are
`{ id, name }`:

```vue
<TomSelect v-model="form.status" :options="statusOptions" :placeholder="$t('courses.statusPlaceholder')" />
```

Static option lists (a status, an enum) are built in the script alongside the
dynamic ones, so both reach the component the same way. `TomSelect` emits a
string, so an enum-typed field is held as `string` in the form object and cast
once when the payload is built — not typed as the union and lied about.

**Size and space it from a wrapper `<div>`, never a class on the component.**
TomSelect replaces the `<select>` it renders with its own wrapper, so `class`
lands on a hidden element and nothing visible changes:

```vue
<div class="mt-2">
  <TomSelect v-model="form.teacher_id" :options="teachers.options.value" :placeholder="..." />
</div>
```

**2. Every required field's label ends in `<RequiredMark />`.** It is the red
asterisk, and it goes in the `<Label>`, not in the placeholder:

```vue
<Label for="course-title">{{ $t('courses.courseTitle') }} <RequiredMark /></Label>
```

The rule is about what the *server* requires, not what the form happens to
enforce — a field the API rejects when blank gets the mark even if the input has
no `required` attribute.

**3. Every field has a placeholder**, from i18n, never hardcoded. A label says
what the field is; a placeholder shows what a valid answer looks like
("2026/2027", "X IPA 1", "https://www.youtube.com/watch?v=...").

The one exception is `<Input type="date">`: browsers ignore `placeholder` on it
and render their own format hint, so passing one adds an attribute that never
appears. Use the label alone there.

### Field layout

```vue
<div>
  <Label for="x">{{ $t('...') }} <RequiredMark /></Label>
  <Input id="x" v-model="form.x" class="mt-2" :placeholder="$t('...')" />
  <FormFeedback feedbackKey="x" />
</div>
```

`FormFeedback` renders the API's per-field validation errors, so it belongs under
every field the server validates — including ones the browser already guards,
since the server checks things the browser cannot (uniqueness, cross-field rules).

## Server-Side Pagination

Use `ServerSidePagination` component (`app/components/ServerSidePagination.vue`) for paginated API data.

```vue
<ServerSidePagination :meta="meta" :loading="loading" is-with-box>
  <Table variant="boxed">
    ...
  </Table>
</ServerSidePagination>
```

Drive it with `useServerList` (`app/composables/useServerList.ts`) — never hand-roll the page/filter wiring:

```ts
const { users, meta, loading, getAll } = useUserService()

const { filters, hasActiveFilters, reset, refresh } = useServerList({
  filters: { search: '', is_active: '' },   // keys = query-param names, values = "empty" defaults
  debounce: ['search'],                     // free-text inputs only
  fetch: (page, perPage, applied) => getAll(page, perPage, applied),
})
```

- Bind inputs with `v-model="filters.search"`, the Reset button with `@click="reset"` and `:disabled="!hasActiveFilters"`.
- Call `refresh()` after a mutation (modal submit, cancel, assign) — never re-call the service directly.
- `applied` values are `''` when unset; every service `buildQuery` drops falsy filters, so pass them straight through.
- Something outside the filters changing the result set (e.g. a tab that picks the endpoint) goes in `deps: [activeTab]`, and its handler calls `reset()` so the new list starts clean — triggers that land in the same action are coalesced into one request, so a dep change plus a URL rewrite still costs a single fetch.

The rule it enforces: **the URL is the single source of truth**. Inputs only write to the URL; one watcher turns the URL into a request. Never sync the URL *and* call the service side by side — that double-fetches and breaks back/forward.

The component handles page/per-page navigation internally via `navigateTo`. Per-page options default to `[10, 20, 50, 100]` — the same set `ClientSidePagination` uses, so both look identical.

Do **not** wrap `ServerSidePagination` in a `<Box>` *and* pass `is-with-box` — that nests two boxes. Either wrap in `<Box>` (matches `roles.vue` / `permissions.vue`) or pass `is-with-box`, not both.

Use `<Table class="-mt-2" variant="boxed">` inside it so server-paginated tables match the client-paginated ones (`ClientSidePagination` applies those classes to its own `Table` internally).

---

## Client-Side Pagination

Use `ClientSidePagination` component for small datasets (< ~100 rows, e.g. roles). Pair with `ClientSideFilter` for search.

`/account/users` is one of these despite having two filters: `GET /api/management/users` returns every user in one call (it is one row per employee), so `filters` is a plain `reactive` and the list is a `computed` over it. Backend-side that is also what makes the query cacheable — see **Read Cache** in the API's CLAUDE.md. Move it back to `useServerList` if the user count ever outgrows a single response.

---

## Badge / Status Variants

Use the `Badge` component from `~/base/ui/badge` with `look="outline"`. Map status strings to variants:

```ts
import { Badge } from '~/base/ui/badge'
import type { BadgeVariants } from '~/base/ui/styles/badge.styles'

type BadgeVariant = NonNullable<BadgeVariants['variant']>

const STATUS_VARIANT: Record<string, BadgeVariant> = {
  open:      'primary',
  pending:   'pending',
  completed: 'success',
  canceled:  'warning',
  rejected:  'danger',
}

const statusVariant = (status: string): BadgeVariant =>
  STATUS_VARIANT[status.toLowerCase()] ?? 'secondary'
```

```vue
<Badge :variant="statusVariant(row.status)" look="outline">
  {{ row.status }}
</Badge>
```

Always call `.toLowerCase()` on the status — the backend returns uppercase values.

---

## Zag-js / Ark UI: Controlled Props

Zag-js machines (`PaginationRoot`, etc.) only read props at initialization. To force re-sync with external state (e.g. URL query params), use `:key`:

```vue
<PaginationRoot
  :key="`page-${currentPage}-size-${currentPerPage}`"
  :page="currentPage"
  ...
>
```

---

## Table Styles — `bg-*` in Boxed Variant

`Table variant="boxed"` applies `bg-background` to every `TableCell` via `tableCellVariants`. This overrides any `bg-*` class added to `<TD>`. To color child elements, use inline styles or a child element (e.g. a `<span>` or `Badge`) rather than coloring the `<TD>` itself.

---

## Types

- `ServerPaginationMeta` — in `app/types/response.ts`. Used by all server-paginated services.
- Backend returns `f_` prefixed fields for formatted currency and datetime. Use these directly — do not reformat on the frontend.
- Backend user relations are `User | null`. Access `full_name` directly; it is optional (`full_name?`) since it may not be present in embedded relations.
---

## Shell Pages (`/account`)

`app/pages/account.vue` is a shell: it only declares a `menus` array and renders
`<SectionNav :menus="menus" />` (`app/components/SectionNav.vue`), which draws the left
vertical nav plus `<NuxtPage />`. Each menu is a **real child route**
(`app/pages/account/users.vue`), not a local tab, so every list is deep-linkable and
mounts/unmounts on its own. `account/index.vue` is only
`definePageMeta({ redirect: '/account/users' })`.

Adding a resource = one entry in the `menus` array + one page file. The side-menu entry
stays `route_name: 'account'` — `SideMenu` matches the path prefix, so it highlights for
every child. Account pages keep their API paths under `api/management/*`; only the
frontend routes moved.
