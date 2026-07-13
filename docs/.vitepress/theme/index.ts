import type { Theme } from "vitepress";
import DefaultTheme from "vitepress/theme";
import "./style.css";

export default {
  extends: DefaultTheme,
  enhanceApp() {
    // Scroll-reveal: fade sections in as they enter the viewport.
    if (typeof window !== "undefined") {
      const observer = new IntersectionObserver(
        (entries) => {
          for (const entry of entries) {
            if (entry.isIntersecting) {
              entry.target.classList.add("vt-revealed");
              observer.unobserve(entry.target);
            }
          }
        },
        { threshold: 0.08, rootMargin: "0px 0px -40px 0px" },
      );

      const observe = () => {
        document.querySelectorAll(".vt-section").forEach((el) => {
          if (!el.classList.contains("vt-revealed")) {
            observer.observe(el);
          }
        });
      };

      window.addEventListener("load", observe);
      if (document.readyState !== "loading") {
        observe();
      } else {
        document.addEventListener("DOMContentLoaded", observe);
      }

      // Re-observe on VitePress route changes.
      const origPushState = history.pushState;
      history.pushState = function (...args) {
        const ret = origPushState.apply(this, args);
        setTimeout(observe, 100);
        return ret;
      };
    }
  },
} satisfies Theme;