export default defineNuxtRouteMiddleware((to) => {
    const token = useCookie('access_token').value || (import.meta.client ? localStorage.getItem('access_token') : null)

    const isPublic = to.meta.public === true
    if (!isPublic && !token) {
        return navigateTo("/auth")
    }

    if (token && to.path === "/auth") {
        return navigateTo("/home")
    }
})