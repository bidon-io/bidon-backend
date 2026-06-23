import { proxyRequest, createError } from "h3";

export default defineEventHandler(async (event) => {
  const path = event.path;
  if (!path.startsWith("/api/") && !path.startsWith("/auth/")) return;

  const { apiProxyTarget } = useRuntimeConfig();
  try {
    return await proxyRequest(event, `${apiProxyTarget}${path}`);
  } catch (err) {
    throw createError({
      statusCode: 502,
      statusMessage: "Bad Gateway",
      message: `API backend unreachable: ${(err as Error).message}`,
    });
  }
});
