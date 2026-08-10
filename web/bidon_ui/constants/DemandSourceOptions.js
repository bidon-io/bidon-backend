import { NETWORK_DEFS } from "./Networks.js";

/** Dropdown options derived from NETWORK_DEFS (label + STI account type). */
export const DEMAND_SOURCE_OPTIONS = NETWORK_DEFS.map((network) => ({
  label: network.label,
  value: network.accountType,
}));
