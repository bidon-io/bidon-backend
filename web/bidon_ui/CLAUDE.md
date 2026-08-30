# CLAUDE.md — bidon_ui

This file provides context and guidelines for working with the Bidon admin web UI.

## Overview

**bidon_ui** is a Nuxt 3 SPA (Single Page Application) — the admin dashboard for the Bidon ad mediation platform. It provides CRUD management for:
- Apps, Demand Sources, Demand Source Accounts
- Line Items, App Demand Profiles
- Auction Configurations (v2)
- Segments, Users, API Keys
- AI Copilot (admin-only)

The app is built with `yarn build` and deployed as a standalone Nuxt/Node.js container (`bidon-ui`). A Nitro server-side middleware proxies `/api/**` and `/auth/**` to the Go `bidon-admin` backend — no reverse proxy (nginx/caddy) needed between the browser and the API.

## Tech Stack

| Category          | Library                         | Version         |
|-------------------|---------------------------------|-----------------|
| Framework         | Nuxt 3                          | 3.13.2          |
| UI                | Vue 3 + TypeScript              | 5.2.2           |
| Component Library | PrimeVue                        | 3.34.0          |
| Styling           | Tailwind CSS                    | 6.7.0           |
| State Management  | Pinia                           | 2.1.6           |
| Form Validation   | Vee-Validate + Yup              | 4.11.8 / 1.3.2  |
| HTTP              | ofetch (`$apiFetch`)            | (Nuxt built-in) |
| HTTP (legacy)     | Axios (`ApiService`)            | 1.5.1           |
| Utilities         | @vueuse/core, humps, inflection | —               |
| AI Copilot        | @langgraph-js/sdk               | 1.10.3          |
| Linting           | ESLint + Prettier               | 8.51.0 / 3.0.3  |

**Routing:** File-based (Nuxt pages directory)
**Mode:** SPA (`ssr: false`)
**Build output:** `.output/` — Nitro server entry at `.output/server/index.mjs`, static assets at `.output/public/`

## Project Structure

```
bidon_ui/
├── pages/                    # File-based routes (Nuxt auto-routing)
│   ├── index.vue             # Dashboard (home)
│   ├── login.vue             # Auth page
│   ├── line_items/           # List, new, [id]/index, [id]/edit
│   ├── apps/
│   ├── demand_sources/
│   ├── demand_source_accounts/
│   ├── app_demand_profiles/
│   ├── segments/
│   ├── users/
│   ├── api_keys/
│   ├── v2/auction_configurations/
│   ├── settings/security.vue
│   └── copilot/
├── components/
│   ├── base/                 # PageContainer, NavigationContainer
│   ├── layouts/              # Header, Sidebar, SidebarNavigation, Footer
│   ├── resources/            # ResourcesTable, ResourceCard, CRUD buttons
│   ├── form/                 # FormCard, FormField, dropdowns, JSON inputs
│   ├── line_items/
│   ├── app_demand_profiles/
│   └── auction_configurations/
├── composables/              # Vue 3 composables (useCreateResource, useNetworks, etc.)
├── layouts/
│   ├── default.vue           # Authenticated layout (sidebar + header)
│   └── auth.vue              # Unauthenticated layout (login)
├── middleware/
│   └── admin-only.ts         # Blocks non-admin users
├── services/
│   └── ApiService.js         # Axios instance (legacy)
├── utils/
│   ├── $apiFetch.ts          # ofetch wrapper (preferred)
│   ├── filterUtils.js        # Query param builders
│   ├── debounce.ts
│   └── jsonToFields.js
├── constants/
│   ├── API_URL.js            # Base URL = "/"
│   ├── ResourceTableFields.js
│   └── ResourceCardFields.js
├── plugins/
│   └── primevue.js           # PrimeVue component registration
├── types/index.ts
├── nuxt.config.ts
└── package.json
```

## Architecture Patterns

### 1. CRUD Page Pattern

Every resource follows the same four-page pattern:

| File                       | Route           | Purpose                        |
|----------------------------|-----------------|--------------------------------|
| `pages/foo/index.vue`      | `/foo`          | List with `LazyResourcesTable` |
| `pages/foo/new.vue`        | `/foo/new`      | Create form                    |
| `pages/foo/[id]/index.vue` | `/foo/:id`      | Detail with `ResourceCard`     |
| `pages/foo/[id]/edit.vue`  | `/foo/:id/edit` | Edit form                      |

### 2. API Calls

**Prefer `$apiFetch` (ofetch) over `ApiService` (Axios).** Axios is being phased out.

Both clients:
- Base URL: `/api`
- Header: `X-Bidon-App: web`
- Auto-convert request body to `snake_case`
- Auto-convert response body to `camelCase`
- Redirect to `/login` on 401

```typescript
// utils/$apiFetch.ts — preferred
const data = await $apiFetch("/line_items", { method: "GET" });

// services/ApiService.js — legacy
import ApiService from "~/services/ApiService";
const { data } = await ApiService.get("/line_items");
```

**Collection response shape:**

```typescript
{ items: T[], meta: { totalCount: number } }
```

### 3. Resource CRUD Composables

These composables wrap API calls with toast notifications and navigation:

| Composable          | Method | Purpose                     |
|---------------------|--------|-----------------------------|
| `useCreateResource` | POST   | Create + navigate to detail |
| `useUpdateResource` | PATCH  | Update + show success toast |
| `useDeleteResource` | DELETE | Confirm dialog + DELETE     |
| `useFormSubmit`     | any    | Generic form submission     |

```javascript
// Create example
const { submit } = useCreateResource({ path: "/line_items" });
await submit(formData); // navigates to /line_items/:id on success

// Delete example
const { destroy } = useDeleteResource({
  path: "/line_items",
  hook: () => refreshData(),
});
```

### 4. Form Pattern (Vee-Validate + Yup)

All forms use the same pattern:

```vue
<script setup>
import * as yup from "yup";
import { useForm } from "vee-validate";

const schema = yup.object({
  name: yup.string().required(),
  // ...
});

const { handleSubmit, values } = useForm({
  validationSchema: schema,
  initialValues,
});
const name = useFieldModel("name");

const onSubmit = handleSubmit(async (values) => {
  await submit(values);
});
</script>

<template>
  <FormCard title="Create">
    <FormField label="Name">
      <VeeFormFieldWrapper name="name">
        <InputText v-model="name" />
      </VeeFormFieldWrapper>
    </FormField>
    <FormSubmitButton @click="onSubmit" />
  </FormCard>
</template>
```

### 5. Table / List Pattern

Use `LazyResourcesTable` for all list pages — it handles server-side pagination, filtering, column definitions, and permission-aware action buttons.

Column definitions come from `constants/ResourceTableFields.js`.

```vue
<LazyResourcesTable path="/line_items" :fields="LINE_ITEM_FIELDS" />
```

Field types supported: `text`, `link`, `associated-link`, `copyable`.

Filter types: `input`, `select`, `select-filter`.

### 6. Detail / Show Pattern

Use `ResourceCard` for show pages. Field definitions from `constants/ResourceCardFields.js`.

```vue
<ResourceCard path="/line_items" :fields="LINE_ITEM_CARD_FIELDS" />
```

### 7. State Management (Pinia)

| Store             | File                             | Purpose                           |
|-------------------|----------------------------------|-----------------------------------|
| `useAuthStore`    | `composables/useAuthStore.ts`    | Current user, auth state          |
| `useResources`    | `composables/useResources.ts`    | Available resources + permissions |
| `useCopilotStore` | `composables/useCopilotStore.ts` | AI copilot thread state           |

**Permission checking:** Resources expose `_permissions` from the backend. The UI hides/shows buttons based on these.

### 8. Layouts & Routing

- Authenticated pages use `default.vue` layout (sidebar + header)
- Login uses `auth.vue` layout — set via `definePageMeta({ layout: 'auth' })`
- Admin-only pages use `definePageMeta({ middleware: 'admin-only' })`

## Data Transformation

All snake_case ↔ camelCase conversion is automatic:
- Request: `{ adType: "banner" }` → `{ ad_type: "banner" }`
- Response: `{ ad_type: "banner" }` → `{ adType: "banner" }`
- Fields starting with `_` are preserved (e.g. `_permissions`)

## Development Workflow

```bash
# Install dependencies
yarn install

# Start dev server (requires Go backend on :1323)
yarn dev
# → http://localhost:3000

# Lint
yarn lint

# Auto-fix lint
yarn lintfix
```

**Backend proxy:** `server/middleware/proxy.ts` forwards `/api/**` and `/auth/**` to `NUXT_API_PROXY_TARGET` (defaults to `http://localhost:1323` in dev).

## Build & Deployment

The UI runs as a standalone Nuxt/Node.js container (`bidon-ui`). A Nitro server-side middleware (`server/middleware/proxy.ts`) forwards `/api/**` and `/auth/**` requests to the Go `bidon-admin` backend, providing same-origin API access without an external reverse proxy.

```bash
# Build for production (outputs Nitro server + static assets to .output/)
yarn build

# Run the production server
node .output/server/index.mjs
# → http://localhost:3000
```

**Required environment variable:**
- `NUXT_API_PROXY_TARGET` — URL of the Go `bidon-admin` backend (e.g. `http://bidon-admin:1323`). A warning is logged at startup if unset; falls back to `http://localhost:1323`.

**Docker target:** `bidon-ui` in the root `Dockerfile` (uses `yarn build`, runs Node.js).

**Note:** The root `Dockerfile` also contains a `bidon-admin-builder` stage that still embeds static files (via `yarn generate`) into the Go binary for deployments that do not use the separate `bidon-ui` container.

## Testing

**There are currently no automated tests.** No testing framework is installed. Rely on manual testing during development.

## Key Conventions

- **No component name prefix** — components are auto-imported without namespace
- **TypeScript is partially adopted** — new files should use `.ts`/`<script setup lang="ts">`, but `.js` composables exist
- **Use `$apiFetch`** for new API calls, not `ApiService`
- **Prefer `<script setup>`** syntax for all new components
- **No SSR** — this is a pure SPA, do not add server-side data fetching that depends on SSR
- **PrimeVue 3.x** — the project uses v3, not v4; component APIs differ between versions
