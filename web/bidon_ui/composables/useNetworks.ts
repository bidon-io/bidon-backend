import { defineStore } from "pinia";
import { useAsyncState } from "@vueuse/core";

export interface NetworkCatalogItem {
  key: string;
  label: string;
  accountType: string;
  supportsBidding: boolean;
  supportsWaterfall: boolean;
}

const LOAD_ERROR = "Could not load the network catalog. Please retry.";

/** Fetches and caches the admin network catalog from GET /api/networks. */
export const useNetworks = defineStore("networks", () => {
  // Pass a factory so execute() can retry after a failed fetch.
  const { state, isReady, isLoading, error, execute } = useAsyncState(
    () => $apiFetch<NetworkCatalogItem[]>("/networks"),
    [] as NetworkCatalogItem[],
  );

  // Share one in-flight request so concurrent callers never double-fetch.
  let pending: Promise<unknown> | null = null;

  async function ensureLoaded() {
    if (!isReady.value) {
      // On failure VueUse leaves isReady=false, so the next call retries.
      pending ??= execute().finally(() => {
        pending = null;
      });
      await pending;
    }

    if (!isReady.value) {
      throw error.value instanceof Error
        ? error.value
        : new Error(String(error.value ?? "Failed to load networks"));
    }

    return state.value;
  }

  /**
   * Loads the catalog; returns a user-facing error message, or null on success.
   * Never throws, so it is safe to call without awaiting.
   */
  async function tryLoad(): Promise<string | null> {
    try {
      await ensureLoaded();
      return null;
    } catch (e) {
      console.error("Failed to load network catalog", e);
      return LOAD_ERROR;
    }
  }

  const byKey = computed(() =>
    Object.fromEntries(state.value.map((n) => [n.key, n])),
  );

  const demandSourceOptions = computed(() =>
    state.value.map((n) => ({
      label: n.label,
      value: n.accountType,
    })),
  );

  function labelFor(key: string) {
    return byKey.value[key]?.label ?? key;
  }

  function accountTypeFor(key: string) {
    return byKey.value[key]?.accountType ?? null;
  }

  function keyForAccountType(accountType: string) {
    return (
      state.value.find((n) => n.accountType === accountType)?.key ?? accountType
    );
  }

  function auctionNetworks(isBidding: boolean) {
    return state.value
      .filter((n) => (isBidding ? n.supportsBidding : n.supportsWaterfall))
      .map((n) => ({
        key: n.key,
        label: n.label,
        accountType: n.accountType,
        isBidding,
      }));
  }

  return {
    state,
    isReady,
    isLoading,
    error,
    execute,
    ensureLoaded,
    tryLoad,
    byKey,
    demandSourceOptions,
    labelFor,
    accountTypeFor,
    keyForAccountType,
    auctionNetworks,
  };
});
