import { defineConfig } from "vitepress";

export default defineConfig({
  title: "vidtrace",
  description: "Bug-video evidence for humans and coding agents.",
  cleanUrls: true,
  lastUpdated: true,

  head: [
    ['link', { rel: 'icon', type: 'image/svg+xml', href: '/favicon.svg' }],
    ['link', { rel: 'icon', type: 'image/x-icon', href: '/favicon.ico' }],
    ['link', { rel: 'apple-touch-icon', sizes: '180x180', href: '/apple-touch-icon.png' }],
    ['meta', { name: 'description', content: 'vidtrace documentation site.' }],
  ],

  sitemap: { hostname: 'https://vidtrace.dev' },
  themeConfig: {
    logo: "/logo.svg",
    nav: [
      { text: "Guide", link: "/usage" },
      { text: "Studio", link: "/studio" },
      { text: "CLI", link: "/cli-contract" },
      { text: "GitHub", link: "https://github.com/abdul-hamid-achik/vidtrace" }
    ],
    sidebar: [
      {
        text: "Start",
        items: [
          { text: "Overview", link: "/" },
          { text: "Install", link: "/install" },
          { text: "Usage", link: "/usage" },
          { text: "Analysis", link: "/analysis" },
          { text: "Studio", link: "/studio" }
        ]
      },
      {
        text: "Reference",
        items: [
          { text: "CLI Contract", link: "/cli-contract" },
          { text: "Artifact Schema", link: "/artifact-schema" },
          { text: "Testing", link: "/testing" },
          { text: "Release", link: "/release" },
          { text: "Documentation Site", link: "/site" }
        ]
      },
      {
        text: "Architecture",
        items: [
          { text: "Architecture", link: "/architecture" },
          { text: "Roadmap", link: "/roadmap" }
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
