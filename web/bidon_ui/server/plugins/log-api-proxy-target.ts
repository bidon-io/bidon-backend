export default defineNitroPlugin(() => {
  if (process.env.NUXT_API_PROXY_TARGET) return;

  const { apiProxyTarget } = useRuntimeConfig();
  console.warn(
    `[bidon-ui] NUXT_API_PROXY_TARGET is not set; defaulting API proxy to ${apiProxyTarget}`,
  );
});
