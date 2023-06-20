import { useToast } from "primevue/usetoast";
import { useToastStore } from "@/store/toast";

export function useShowToasts() {
  const toastService = useToast();
  const { showToasts } = useToastStore();

  return () => showToasts(toastService);
}
