import { defineStore } from "pinia";
import { camelizeKeys } from "humps";
import axios from "axios";
import { useToast } from "primevue/usetoast";
import { API_URL } from "@/constants/index.js";

export const useAuthStore = defineStore("authStore", () => {
  const api = axios.create({ baseURL: API_URL });
  const toast = useToast();
  const router = useRouter();

  const localStorageUser = localStorage.getItem("user");
  const user = ref(
    localStorageUser ? camelizeKeys(JSON.parse(localStorageUser)) : null,
  );
  const accessToken = ref(localStorage.getItem("accessToken") || null);

  async function login(email, password) {
    api
      .post("/auth/login", { email, password })
      .then((response) => {
        router.push("/");
      })
      .catch((error) => {
        toast.add({
          severity: "error",
          summary: "Error",
          detail:
            error?.response?.data?.error?.message || "Something went wrong",
        });
      });
  }

  async function logout() {
    api
        .post("/auth/logout")
        .then((response) => {
            router.push("/login");
        });
  }

  return {
    user,
    accessToken,
    login,
    logout,
  };
});
