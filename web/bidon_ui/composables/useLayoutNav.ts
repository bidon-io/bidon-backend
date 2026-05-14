import { watch, onMounted, onUnmounted, readonly } from "vue";
import { useBreakpoints } from "@vueuse/core";

const SIDEBAR_KEY = "layout-sidebar-open";

export function useLayoutNav() {
  const sidebarOpen = useState<boolean>(SIDEBAR_KEY, () => false);
  const isLarge = useBreakpoints({ lg: 1024 }).greaterOrEqual("lg");

  return {
    sidebarOpen: readonly(sidebarOpen),
    closeSidebar: () => {
      sidebarOpen.value = false;
    },
    toggleSidebar: () => {
      sidebarOpen.value = !sidebarOpen.value;
    },
    isLarge,
  };
}

/** Wire up sidebar side-effects — call once from the default layout only. */
export function useLayoutNavSetup() {
  const sidebarOpen = useState<boolean>(SIDEBAR_KEY, () => false);
  const { closeSidebar, isLarge } = useLayoutNav();
  const route = useRoute();

  watch(() => route.path, closeSidebar);

  watch(isLarge, (large) => {
    if (large) closeSidebar();
  });

  watch([sidebarOpen, isLarge], ([open, large]) => {
    if (!import.meta.client) return;
    document.body.style.overflow = open && !large ? "hidden" : "";
  });

  function onEscape(e: KeyboardEvent) {
    if (e.key === "Escape") closeSidebar();
  }

  onMounted(() => window.addEventListener("keydown", onEscape));
  onUnmounted(() => {
    window.removeEventListener("keydown", onEscape);
    if (import.meta.client) document.body.style.overflow = "";
  });
}
