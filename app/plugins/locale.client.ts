export default defineNuxtPlugin((nuxtApp) => {
  const i18n = nuxtApp.$i18n as {
    locale: { value: string }
    locales: { value: { code: string }[] }
    setLocale: (code: string) => Promise<void>
  }
  const saved = localStorage.getItem('locale')
  const supported = i18n.locales.value.map((l) => l.code)

  if (saved && supported.includes(saved)) {
    i18n.setLocale(saved)
  }
})
