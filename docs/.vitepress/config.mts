import { defineConfig } from "vitepress";

export default defineConfig({
  title: "vidtrace",
  description: "Bug-video evidence for humans and coding agents.",
  cleanUrls: true,
  lastUpdated: true,
  themeConfig: {
    logo: "/logo.svg",
    nav: [
      { text: "Guide", link: "/USAGE" },
      { text: "Studio", link: "/STUDIO" },
      { text: "CLI", link: "/CLI_CONTRACT" },
      { text: "GitHub", link: "https://github.com/abdul-hamid-achik/vidtrace" }
    ],
    sidebar: [
      {
        text: "Start",
        items: [
          { text: "Overview", link: "/" },
          { text: "Install", link: "/INSTALL" },
          { text: "Usage", link: "/USAGE" },
          { text: "Analysis", link: "/ANALYSIS" },
          { text: "Studio", link: "/STUDIO" }
        ]
      },
      {
        text: "Reference",
        items: [
          { text: "CLI Contract", link: "/CLI_CONTRACT" },
          { text: "Artifact Schema", link: "/ARTIFACT_SCHEMA" },
          { text: "Testing", link: "/TESTING" },
          { text: "Release", link: "/RELEASE" },
          { text: "Documentation Site", link: "/SITE" }
        ]
      },
      {
        text: "Architecture",
        items: [
          { text: "Architecture", link: "/ARCHITECTURE" },
          { text: "Roadmap", link: "/ROADMAP" },
          { text: "ADRs", link: "/adr/" }
        ]
      }
    ],
    search: {
      provider: "local"
    },
    socialLinks: [
      { icon: "github", link: "https://github.com/abdul-hamid-achik/vidtrace" }
    ],
    editLink: {
      pattern: "https://github.com/abdul-hamid-achik/vidtrace/edit/main/docs/:path",
      text: "Edit this page on GitHub"
    }
  },
  vite: {
    build: {
      emptyOutDir: true
    }
  }
});
