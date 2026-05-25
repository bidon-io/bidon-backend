import { proxyRequest } from "h3";

export default defineEventHandler(async (event) => {
  const path = event.path;
  if (!path.startsWith("/api/") && !path.startsWith("/auth/")) return;

  const { apiProxyTarget } = useRuntimeConfig();
  return proxyRequest(event, `${apiProxyTarget}${path}`);
});
