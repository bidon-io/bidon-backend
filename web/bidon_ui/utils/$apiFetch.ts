import { camelizeKeys, decamelizeKeys } from "humps";
import { $fetch } from "ofetch";

export const $apiFetch = $fetch.create({
  headers: {
    "X-Bidon-App": "web",
  },
  onRequest(context) {
    if (
      typeof context.request === "string" &&
      !context.request.startsWith("http")
    ) {
      context.request = `/api${context.request.startsWith("/") ? "" : "/"}${context.request}`;
    }

    if (
      context.options.body &&
      typeof context.options.body === "object" &&
      !(context.options.body instanceof FormData)
    ) {
      context.options.body = decamelizeKeys(
        context.options.body as Record<string, unknown>,
      );
    }

    if (
      context.options.params &&
      Object.prototype.toString.call(context.options.params) ===
        "[object Object]"
    ) {
      context.options.params = decamelizeKeys(context.options.params);
    }
  },
  onResponse({ response }) {
    if (
      response._data &&
      response.headers.get("Content-Type")?.includes("application/json")
    ) {
      response._data = camelizeKeys(response._data, (key, convert, options) =>
        key[0] === "_" ? key : convert(key, options),
      );
    }
  },
  async onResponseError({ response }) {
    if (response.status === 401) {
      useAuthStore().setUnauthorized();
      await navigateTo("/login");
    }
  },
});
