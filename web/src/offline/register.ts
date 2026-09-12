import { registerSW } from "virtual:pwa-register";

export function registerDriverSW() {
  if (!location.pathname.startsWith("/driver")) return;
  registerSW({ immediate: true });
}
