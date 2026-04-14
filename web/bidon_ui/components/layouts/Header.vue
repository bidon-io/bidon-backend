<template>
  <header class="z-5 flex-shrink-0 app-header">
    <div class="px-6 lg:px-8 py-2 flex items-center justify-between gap-4">

      <!-- Breadcrumb -->
      <nav class="flex items-center gap-1.5 min-w-0 text-sm">
        <NuxtLink to="/" class="shrink-0 font-medium hover:opacity-70 transition-opacity" style="color: var(--bidon-accent);">
          Home
        </NuxtLink>
        <template v-for="(crumb, i) in breadcrumbs" :key="crumb.path">
          <span style="color: var(--bidon-muted);">/</span>
          <NuxtLink
            v-if="i < breadcrumbs.length - 1"
            :to="crumb.path"
            class="hover:opacity-70 transition-opacity truncate"
            style="color: var(--bidon-accent);"
          >{{ crumb.label }}</NuxtLink>
          <span v-else class="font-semibold truncate" style="color: var(--bidon-text-primary);">{{ crumb.label }}</span>
        </template>
      </nav>

      <!-- User info -->
      <div v-if="currentUser" class="flex items-center gap-2 shrink-0">
        <!-- Dark mode toggle -->
        <button
          class="header-icon-btn"
          :title="isDark ? 'Switch to light mode' : 'Switch to dark mode'"
          @click="toggleDark()"
        >
          <i :class="isDark ? 'pi pi-sun' : 'pi pi-moon'" class="text-xs" />
        </button>
        <span v-if="currentUser.isAdmin" class="header-admin-badge">
          Admin
        </span>
        <span class="text-sm" style="color: var(--bidon-muted);">{{ currentUser.email }}</span>
        <button class="logout-btn" @click="logout">
          <i class="pi pi-sign-out text-xs" />
          Log out
        </button>
      </div>

    </div>
  </header>
</template>

<script setup lang="ts">
const { isDark, toggleDark } = useColorMode();
import { titleize, pluralize } from "inflection";

const { currentUser, logout } = useAuthStore();
const route = useRoute();

const ROUTE_LABELS: Record<string, string> = {
  apps: "Apps",
  app_demand_profiles: "App Demand Profiles",
  line_items: "Line Items",
  demand_sources: "Demand Sources",
  demand_source_accounts: "Demand Source Accounts",
  segments: "Segments",
  users: "Users",
  api_keys: "API Credentials",
  auction_configurations: "Auction Configurations",
  copilot: "AI Copilot",
  settings: "Settings",
  security: "Passwords",
  v2: null, // skip v2 prefix in breadcrumb
};

const breadcrumbs = computed(() => {
  const parts = route.path.split("/").filter(Boolean);
  const crumbs: { label: string; path: string }[] = [];
  let pathSoFar = "";

  for (const part of parts) {
    pathSoFar += `/${part}`;

    // Skip "v2" as a standalone segment
    if (part === "v2") continue;

    const isId = /^\d+$/.test(part);
    let label: string;

    if (isId) {
      label = `#${part}`;
    } else if (part === "new") {
      label = "New";
    } else if (part === "edit") {
      label = "Edit";
    } else if (part in ROUTE_LABELS) {
      label = ROUTE_LABELS[part] ?? part;
    } else {
      label = pluralize(titleize(part.replace(/_/g, " ")));
    }

    crumbs.push({ label, path: pathSoFar });
  }

  return crumbs;
});
</script>
