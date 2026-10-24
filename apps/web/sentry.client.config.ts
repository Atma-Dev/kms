import * as Sentry from '@sentry/nuxt'

const config = useRuntimeConfig()

Sentry.init({
  dsn: config.public.sentryDsn,
  enabled: !!config.public.sentryDsn,
})
