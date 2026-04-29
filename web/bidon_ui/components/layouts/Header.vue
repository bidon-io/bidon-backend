<template>
  <header
    class="sticky top-0 z-[45] flex-shrink-0 app-header lg:static lg:z-auto"
  >
    <div class="flex items-center gap-2 px-4 py-2 sm:gap-3 sm:px-6 lg:px-8">
      <!-- Mobile nav -->
      <button
        type="button"
        class="header-icon-btn -ml-1 shrink-0 lg:hidden"
        :aria-expanded="sidebarOpen"
        aria-controls="sidebar-navigation"
        :aria-label="sidebarOpen ? 'Close menu' : 'Open menu'"
        @click="toggleSidebar()"
      >
        <i
          :class="sidebarOpen ? 'pi pi-times' : 'pi pi-bars'"
          class="text-sm"
        />
      </button>

      <!-- Breadcrumb -->
      <nav
        aria-label="Breadcrumb"
        class="flex min-w-0 flex-1 flex-wrap items-center gap-x-1.5 gap-y-1 text-xs sm:text-sm"
      >
        <NuxtLink
          to="/"
          class="shrink-0 font-medium hover:opacity-70 transition-opacity"
          style="color: var(--bidon-accent)"
        >
          Home
        </NuxtLink>
        <template v-for="(crumb, i) in breadcrumbs" :key="crumb.path">
          <span style="color: var(--bidon-muted)">/</span>
          <NuxtLink
            v-if="i < breadcrumbs.length - 1"
            :to="crumb.path"
            class="truncate hover:opacity-70 transition-opacity"
            style="color: var(--bidon-accent)"
            >{{ crumb.label }}</NuxtLink
          >
          <span
            v-else
            class="truncate font-semibold max-w-[40vw] sm:max-w-none"
            style="color: var(--bidon-text-primary)"
            >{{ crumb.label }}</span
          >
        </template>
      </nav>

      <!-- User info -->
      <div
        v-if="currentUser"
        class="-mr-1 flex shrink-0 flex-wrap items-center justify-end gap-1.5 sm:gap-2"
      >
        <button
          class="header-icon-btn"
          :title="isDark ? 'Switch to light mode' : 'Switch to dark mode'"
          @click="toggleDark()"
        >
          <i :class="isDark ? 'pi pi-sun' : 'pi pi-moon'" class="text-xs" />
        </button>
        <span
          v-if="currentUser.isAdmin"
          class="header-admin-badge hidden sm:inline-flex"
        >
          Admin
        </span>
        <span
          class="hidden max-w-[10rem] truncate text-xs md:inline lg:max-w-[15rem]"
          style="color: var(--bidon-muted)"
          :title="currentUser.email"
          >{{ currentUser.email }}</span
        >
        <button
          type="button"
          class="logout-btn"
          aria-label="Log out"
          @click="logout"
        >
          <i class="pi pi-sign-out text-xs" />
          <span class="hidden sm:inline">Log out</span>
        </button>
      </div>
    </div>
  </header>
</template>

<script setup lang="ts">
import { titleize, pluralize } from "inflection";

const { isDark, toggleDark } = useColorMode();

const { sidebarOpen, toggleSidebar } = useLayoutNav();

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
