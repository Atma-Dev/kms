export default defineNuxtRouteMiddleware((to, from) => {
  if (to.path !== from.path) {
    const flashStore = useFlashStore()
    flashStore.clearMessages()
  }
})
