import { defineStore } from "pinia";
import { until, useAsyncState } from "@vueuse/core";

export interface NetworkCatalogItem {
  key: string;
  label: string;
  accountType: string;
  supportsBidding: boolean;
  supportsWaterfall: boolean;
}

/** Fetches and caches the admin network catalog from GET /api/networks. */
export const useNetworks = defineStore("networks", () => {
  const { state, isReady, isLoading, error, execute } = useAsyncState(
    $apiFetch<NetworkCatalogItem[]>("/networks"),
    [] as NetworkCatalogItem[],
  );

  async function ensureLoaded() {
    if (!isReady.value) {
      await until(isReady).toBe(true);
    }
    return state.value;
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
      state.value.find((n) => n.accountType === accountType)?.key ??
      accountType
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
    byKey,
    demandSourceOptions,
    labelFor,
    accountTypeFor,
    keyForAccountType,
    auctionNetworks,
  };
});
