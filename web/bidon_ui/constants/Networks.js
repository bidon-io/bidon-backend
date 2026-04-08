/**
 * Canonical network registry — single source of truth for key, label,
 * accountType, and bidding/waterfall membership.
 */
export const NETWORK_DEFS = [
  { key: "admob", label: "AdMob", accountType: "DemandSourceAccount::Admob" },
  {
    key: "applovin",
    label: "AppLovin",
    accountType: "DemandSourceAccount::Applovin",
  },
  {
    key: "bidmachine",
    label: "BidMachine",
    accountType: "DemandSourceAccount::BidMachine",
  },
  {
    key: "bigoads",
    label: "Bigo Ads",
    accountType: "DemandSourceAccount::BigoAds",
  },
  {
    key: "chartboost",
    label: "Chartboost",
    accountType: "DemandSourceAccount::Chartboost",
  },
  {
    key: "dtexchange",
    label: "DT Exchange",
    accountType: "DemandSourceAccount::DtExchange",
  },
  {
    key: "gam",
    label: "Google Ad Manager",
    accountType: "DemandSourceAccount::GAM",
  },
  {
    key: "mintegral",
    label: "Mintegral",
    accountType: "DemandSourceAccount::Mintegral",
  },
  {
    key: "unityads",
    label: "Unity Ads",
    accountType: "DemandSourceAccount::UnityAds",
  },
  {
    key: "ironsource",
    label: "IronSource",
    accountType: "DemandSourceAccount::IronSource",
  },
  { key: "vkads", label: "VK Ads", accountType: "DemandSourceAccount::VKAds" },
  {
    key: "vungle",
    label: "Vungle",
    accountType: "DemandSourceAccount::Vungle",
  },
  {
    key: "yandex",
    label: "Yandex",
    accountType: "DemandSourceAccount::Yandex",
  },
  {
    key: "amazon",
    label: "Amazon",
    accountType: "DemandSourceAccount::Amazon",
  },
  {
    key: "inmobi",
    label: "InMobi",
    accountType: "DemandSourceAccount::Inmobi",
  },
  { key: "meta", label: "Meta", accountType: "DemandSourceAccount::Meta" },
  {
    key: "mobilefuse",
    label: "MobileFuse",
    accountType: "DemandSourceAccount::MobileFuse",
  },
  {
    key: "moloco",
    label: "Moloco",
    accountType: "DemandSourceAccount::Moloco",
  },
  {
    key: "startio",
    label: "Start.io",
    accountType: "DemandSourceAccount::StartIO",
  },
  {
    key: "taurusx",
    label: "TaurusX",
    accountType: "DemandSourceAccount::TaurusX",
  },
  {
    key: "zmaticoo",
    label: "Zmaticoo",
    accountType: "DemandSourceAccount::Zmaticoo",
  },
];

/** Keys that participate in waterfall (CPM) auctions */
export const WATERFALL_NETWORK_KEYS = [
  "admob",
  "applovin",
  "bidmachine",
  "bigoads",
  "chartboost",
  "dtexchange",
  "gam",
  "mintegral",
  "unityads",
  "ironsource",
  "vkads",
  "vungle",
  "yandex",
];

/** Keys that participate in bidding (RTB) auctions */
export const BIDDING_NETWORK_KEYS = [
  "amazon",
  "bidmachine",
  "bigoads",
  "inmobi",
  "meta",
  "mintegral",
  "mobilefuse",
  "moloco",
  "startio",
  "taurusx",
  "vungle",
  "vkads",
  "yandex",
  "zmaticoo",
];

const defByKey = Object.fromEntries(NETWORK_DEFS.map((n) => [n.key, n]));

/** Map from network key → human-readable label */
export const NETWORK_LABEL_BY_KEY = Object.fromEntries(
  NETWORK_DEFS.map((n) => [n.key, n.label]),
);

/** Map from network key → backend account type string */
export const NETWORK_ACCOUNT_TYPE_BY_KEY = Object.fromEntries(
  NETWORK_DEFS.map((n) => [n.key, n.accountType]),
);

/**
 * Full auction network list for AuctionConfigurationsForm.
 * Each entry: { key, label, accountType, isBidding }
 */
export const AUCTION_NETWORKS = [
  ...WATERFALL_NETWORK_KEYS.map((key) => ({
    ...defByKey[key],
    isBidding: false,
  })),
  ...BIDDING_NETWORK_KEYS.map((key) => ({ ...defByKey[key], isBidding: true })),
];
