// https://nuxt.com/docs/api/configuration/nuxt-config
const apiProxyTarget =
  process.env.NUXT_API_PROXY_TARGET || "http://localhost:1323";

export default defineNuxtConfig({
  alias: {
    assets: "/<rootDir>/assets",
  },

  ssr: false,

  runtimeConfig: {
    public: {
      copilotBase: "/api/copilot",
    },
  },

  css: [
    "primevue/resources/themes/lara-light-blue/theme.css",
    "primevue/resources/primevue.css",
    "primeicons/primeicons.css",
  ],

  components: [
    {
      path: "~/components",
      pathPrefix: false,
    },
  ],

  modules: ["@nuxtjs/tailwindcss", "@pinia/nuxt", "@vee-validate/nuxt"],

  build: {
    transpile: ["primevue"],
  },

  routeRules: {
    "/auth/**": { proxy: `${apiProxyTarget}/auth/**` },
    "/api/**": { proxy: `${apiProxyTarget}/api/**` },
  },

  compatibilityDate: "2024-10-31",
});
