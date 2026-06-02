/** PATCH/POST responses return bare records without `_permissions`; keep prior permissions for UI controls. */
export function mergeResourcePermissions(prev, data) {
  if (!data) return data;
  return {
    ...data,
    _permissions: data._permissions ??
      prev?._permissions ?? { update: true, delete: true },
  };
}
