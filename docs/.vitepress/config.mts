import { defineConfig } from "vitepress";

const siteUrl = "https://vidtrace.dev";
const title = "vidtrace. Bug video evidence, timestamped.";
const description =
  "Turn screen recordings into structured evidence bundles with frames, OCR, transcripts, and a timestamped timeline. Local-first Go CLI for humans and coding agents.";
const ogImage = `${siteUrl}/og-image.png`;

export default defineConfig({
  title: "vidtrace",
  description: description,
  cleanUrls: true,
  lastUpdated: true,

  head: [
    // Favicons
    ['link', { rel: 'icon', type: 'image/svg+xml', href: '/favicon.svg' }],
    ['link', { rel: 'icon', type: 'image/png', sizes: '32x32', href: '/favicon-32.png' }],
    ['link', { rel: 'apple-touch-icon', sizes: '180x180', href: '/apple-touch-icon.png' }],

    // Primary meta
    ['meta', { name: 'keywords', content: 'bug video, screen recording, evidence, OCR, transcription, whisper, ffmpeg, tesseract, developer tools, CLI, Go, timestamped evidence, coding agents, MCP, VecLite, semantic search, QA, bug reproduction' }],
    ['meta', { name: 'author', content: 'abdul-hamid-achik' }],
    ['meta', { name: 'theme-color', content: '#12161C' }],
    ['meta', { name: 'robots', content: 'index, follow' }],

    // Canonical
    ['link', { rel: 'canonical', href: siteUrl }],

    // Open Graph
    ['meta', { property: 'og:type', content: 'website' }],
    ['meta', { property: 'og:site_name', content: 'vidtrace' }],
    ['meta', { property: 'og:title', content: title }],
    ['meta', { property: 'og:description', content: description }],
    ['meta', { property: 'og:image', content: ogImage }],
    ['meta', { property: 'og:image:width', content: '1200' }],
    ['meta', { property: 'og:image:height', content: '630' }],
    ['meta', { property: 'og:image:alt', content: 'vidtrace. Bug video evidence, timestamped.' }],
    ['meta', { property: 'og:url', content: siteUrl }],
    ['meta', { property: 'og:locale', content: 'en_US' }],

    // Twitter Card
    ['meta', { name: 'twitter:card', content: 'summary_large_image' }],
    ['meta', { name: 'twitter:title', content: title }],
    ['meta', { name: 'twitter:description', content: description }],
    ['meta', { name: 'twitter:image', content: ogImage }],
    ['meta', { name: 'twitter:image:alt', content: 'vidtrace. Bug video evidence, timestamped.' }],
    ['meta', { name: 'twitter:creator', content: '@abdulachik' }],

    // JSON-LD structured data: SoftwareApplication
    ['script', { type: 'application/ld+json' }, JSON.stringify({
      "@context": "https://schema.org",
      "@type": "SoftwareApplication",
      "name": "vidtrace",
      "applicationCategory": "DeveloperApplication",
      "operatingSystem": "macOS, Linux",
      "description": description,
      "url": siteUrl,
      "downloadUrl": "https://github.com/abdul-hamid-achik/vidtrace/releases",
      "codeRepository": "https://github.com/abdul-hamid-achik/vidtrace",
      "license": "https://github.com/abdul-hamid-achik/vidtrace/blob/main/LICENSE",
      "programmingLanguage": "Go",
      "softwareVersion": "0.19.0",
      "offers": {
        "@type": "Offer",
        "price": "0",
        "priceCurrency": "USD"
      },
      "featureList": [
        "Timestamped frame extraction with OCR and Whisper transcription",
        "Agent-ready JSON output with stable contracts",
        "Terminal Studio for human evidence review",
        "BM25 keyword, semantic, and hybrid evidence search",
        "Ticket-vs-video comparison with confidence scoring",
        "Clip cutting, GIF creation, and video stitching",
        "MCP server for coding agent integration",
        "fcheap vault stashing and vecgrep codebase search"
      ]
    })],

    // JSON-LD structured data: FAQPage
    ['script', { type: 'application/ld+json' }, JSON.stringify({
      "@context": "https://schema.org",
      "@type": "FAQPage",
      "mainEntity": [
        {
          "@type": "Question",
          "name": "What does vidtrace actually do?",
          "acceptedAnswer": {
            "@type": "Answer",
            "text": "It takes a screen recording of a bug and produces a structured evidence bundle: extracted frames (PNG), OCR text per frame, a Whisper transcript in 5 formats, ffprobe metadata, and a timeline.json that maps every frame to overlapping transcript segments. Everything is timestamped and citable."
          }
        },
        {
          "@type": "Question",
          "name": "What runtime tools do I need?",
          "acceptedAnswer": {
            "@type": "Answer",
            "text": "ffmpeg, ffprobe, tesseract, and whisper. Run vidtrace doctor after install to verify. Optional tools: Ollama for semantic search, fcheap for bundle stashing, vecgrep for codebase search, codemap for structural code graph queries."
          }
        },
        {
          "@type": "Question",
          "name": "Can coding agents use vidtrace?",
          "acceptedAnswer": {
            "@type": "Answer",
            "text": "Yes. Query and extract commands emit stable JSON with --json. Agents read output_dir from stdout, then inspect timeline.json, metadata.json, OCR text, and selected frames. The MCP server (vidtrace mcp) exposes read-only tools over stdio including validate, search, compare, analyze, investigate, timeline, and frame."
          }
        },
        {
          "@type": "Question",
          "name": "Does it upload my videos anywhere?",
          "acceptedAnswer": {
            "@type": "Answer",
            "text": "No. vidtrace is local-first. All extraction runs on your machine with ffmpeg, tesseract, and whisper. No cloud uploads. Optional Ollama embeddings also run locally."
          }
        },
        {
          "@type": "Question",
          "name": "How do I install it?",
          "acceptedAnswer": {
            "@type": "Answer",
            "text": "The fastest path is Homebrew: brew tap abdul-hamid-achik/tap and brew install --cask abdul-hamid-achik/tap/vidtrace. Linux .deb and .rpm packages are also published. Or build from source with task build."
          }
        },
        {
          "@type": "Question",
          "name": "What is timeline.json?",
          "acceptedAnswer": {
            "@type": "Answer",
            "text": "The main agent-facing artifact. It maps every extracted frame to its OCR text and any overlapping transcript segments, with second-accurate timestamps. It is the structured, citable evidence that replaces vague reproduction notes."
          }
        },
        {
          "@type": "Question",
          "name": "Can I search across multiple bug videos?",
          "acceptedAnswer": {
            "@type": "Answer",
            "text": "Yes. vidtrace index accepts multiple bundle paths (shell globs) and indexes them into one VecLite database. Search with keyword, semantic, or hybrid mode, filtered by bundle, source video, evidence source, and time window."
          }
        }
      ]
    })],
  ],

  sitemap: { hostname: siteUrl },

  themeConfig: {
    logo: { src: "/favicon.svg", dark: "/favicon.svg" },
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
          { text: "Studio", link: "/studio" },
          { text: "Clip", link: "/clip" }
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
    footer: {
      message: "Local-first. MIT licensed."
    },
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