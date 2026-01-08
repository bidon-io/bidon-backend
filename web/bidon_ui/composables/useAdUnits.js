import axios from "@/services/ApiService";

export function useAdUnits(appId, adType, isBidding) {
  return useAsyncData(
    `line-items-${appId.value}-${adType.value}-${isBidding}`,
    async () => {
      if (!appId.value || !adType.value) return [];

      const url = `/line_items?app_id=${appId.value}&ad_type=${adType.value}&is_bidding=${isBidding}`;
      const result = (await axios.get(url)).data;

      return result.map((adUnit) => ({
        id: adUnit.id,
        label: adUnit.humanName,
        networkKey: adUnit.accountType.split("::")[1].toLowerCase(),
        uid: adUnit.publicUid,
        pricefloor: parseFloat(adUnit.bidFloor),
        account: `${adUnit.accountType.split("::")[1].toLowerCase()} (${
          adUnit.accountId
        })`,
        isBidding: adUnit.isBidding,
        extra: adUnit.extra || {},
      }));
    },
    {
      watch: [appId, adType],
      default: () => [],
    },
  );
}
