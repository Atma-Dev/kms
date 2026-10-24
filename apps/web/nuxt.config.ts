import { fileURLToPath } from 'node:url'
import tailwindcss from '@tailwindcss/vite'

// https://nuxt.com/docs/api/configuration/nuxt-config
export default defineNuxtConfig({
  ssr: false,
  devServer: { port: 3019 },
  compatibilityDate: '2025-07-15',
  devtools: { enabled: true },
  runtimeConfig: {
    public: {
      apiBase: process.env['NUXT_PUBLIC_API_BASE'],
      sentryDsn: process.env['NUXT_PUBLIC_SENTRY_DSN'] ?? '',
    },
  },
  components: [
    { path: '~/components', pathPrefix: false },
    {
      path: '~/base/ui',
      pathPrefix: false,
      extensions: ['vue'],
      ignore: [
        '**/styles/**',
        '**/utils/**',
        '**/usage/**',
        '**/progress-circular/**',
      ],
    },
  ],
  modules: ['@pinia/nuxt', '@nuxtjs/i18n', '@sentry/nuxt/module'],
  css: ['@assets/css/index.css'],
  vite: {
    plugins: [tailwindcss()],
  },
  i18n: {
    langDir: 'locales',
    strategy: 'no_prefix',
    defaultLocale: 'en',
    detectBrowserLanguage: false,
    locales: [
      {
        code: 'en',
        name: 'English',
        files: [
          'en/common.json',
          'en/nav.json',
          'en/account.json',
          'en/activity-logs.json',
          'en/audits.json',
          'en/job-logs.json',
          'en/queue.json',
          'en/schedules.json',
          'en/jobs-dashboard.json',
          'en/notifications.json',
          'en/spaces.json',
          'en/categories.json',
          'en/documents.json',
          'en/tags.json',
          'en/search.json',
          'en/auth.json',
          'en/home.json',
        ],
      },
      {
        code: 'id',
        name: 'Bahasa Indonesia',
        files: [
          'id/common.json',
          'id/nav.json',
          'id/account.json',
          'id/activity-logs.json',
          'id/audits.json',
          'id/job-logs.json',
          'id/queue.json',
          'id/schedules.json',
          'id/jobs-dashboard.json',
          'id/notifications.json',
          'id/spaces.json',
          'id/categories.json',
          'id/documents.json',
          'id/tags.json',
          'id/search.json',
          'id/auth.json',
          'id/home.json',
        ],
      },
    ],
  },
  alias: {
    '@': fileURLToPath(new URL('./app', import.meta.url)),
    '~': fileURLToPath(new URL('./app', import.meta.url)),
    '@assets': fileURLToPath(new URL('./app/assets', import.meta.url)),
    '@midoneui': fileURLToPath(new URL('./app/base/ui', import.meta.url)),
  },
})
