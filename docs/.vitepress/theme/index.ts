import type { Theme } from "vitepress";
import DefaultTheme from "vitepress/theme";
import { h } from "vue";

import "@fontsource/ibm-plex-sans/400.css";
import "@fontsource/ibm-plex-sans/500.css";
import "@fontsource/ibm-plex-sans/600.css";
import "@fontsource/ibm-plex-sans/700.css";
import "@fontsource/ibm-plex-mono/400.css";
import "@fontsource/ibm-plex-mono/500.css";
import "@fontsource/ibm-plex-mono/600.css";

import "./style.css";
import HomePage from "./components/HomePage.vue";
import NotFound from "./components/NotFound.vue";

export default {
  extends: DefaultTheme,
  Layout: () =>
    h(DefaultTheme.Layout, null, {
      "not-found": () => h(NotFound),
    }),
  enhanceApp({ app }) {
    app.component("HomePage", HomePage);
  },
} satisfies Theme;
