// https://nuxt.com/docs/api/configuration/nuxt-config
const apiProxyTarget =
  process.env.NUXT_API_PROXY_TARGET || "http://localhost:1323";

const apiBase = process.env.NUXT_PUBLIC_API_BASE || "";

export default defineNuxtConfig({
  alias: {
    assets: "/<rootDir>/assets",
  },

  ssr: false,

  runtimeConfig: {
    public: {
      // Set NUXT_PUBLIC_API_BASE at build time to point at the API origin when
      // the frontend is served from a separate domain (e.g. DO Spaces/CDN).
      // Leave unset for local dev — the Nuxt dev-server proxy handles routing.
      apiBase: "",
      copilotBase: apiBase
        ? `${apiBase.replace(/\/$/, "")}/api/copilot`
        : "/api/copilot",
    },
  },

  app: {
    head: {
      meta: [
        {
          name: "viewport",
          content: "width=device-width, initial-scale=1, viewport-fit=cover",
        },
      ],
      link: [
        {
          rel: "preconnect",
          href: "https://fonts.googleapis.com",
        },
        {
          rel: "preconnect",
          href: "https://fonts.gstatic.com",
          crossorigin: "",
        },
        {
          rel: "stylesheet",
          href: "https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&display=swap",
        },
        {
          rel: "icon",
          type: "image/png",
          sizes: "64x64",
          href: "/favicon-64.png",
        },
        {
          rel: "icon",
          type: "image/png",
          sizes: "32x32",
          href: "/favicon-32.png",
        },
        {
          rel: "icon",
          type: "image/png",
          sizes: "16x16",
          href: "/favicon-16.png",
        },
      ],
    },
  },

  css: [
    "primevue/resources/themes/lara-light-teal/theme.css",
    "primevue/resources/primevue.css",
    "primeicons/primeicons.css",
  ],

  tailwindcss: {
    cssPath: "~/assets/css/components.css",
    configPath: "~/tailwind.config.js",
  },

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
