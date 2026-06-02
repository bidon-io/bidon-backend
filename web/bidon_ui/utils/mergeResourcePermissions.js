/**
 * Merge API payload into local state without dropping `_permissions`.
 *
 * PATCH/POST responses return bare records (no `_permissions`); list/GET responses include them.
 *
 * - **Update:** `prev` is the existing row — we keep `prev._permissions` when the response omits them.
 * - **Create:** `prev` is undefined — there is nothing to preserve, so the default applies.
 *   Default `{ update: true, delete: true }` is intentional: the user just created the resource and
 *   is effectively its owner; the backend still enforces authorization on actual mutations.
 */
export function mergeResourcePermissions(prev, data) {
  if (!data) return data;
  return {
    ...data,
    _permissions: data._permissions ??
      prev?._permissions ?? { update: true, delete: true },
  };
}
