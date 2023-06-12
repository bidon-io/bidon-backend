export let API_URL;

if (import.meta.env.VITE_APP_ENV === "production") {
  API_URL = "https://bidon-go.appodeal.com/";
} else {
  API_URL = "http://localhost:1323";
}
