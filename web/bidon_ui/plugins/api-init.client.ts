import api from "~/services/ApiService";

export default defineNuxtPlugin(() => {
  const {
    public: { apiBase },
  } = useRuntimeConfig();
  if (apiBase) {
    api.defaults.baseURL = `${(apiBase as string).replace(/\/$/, "")}/api`;
  }
});
