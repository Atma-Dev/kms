/** Matches `enums.SUPER_ADMIN` ("Super-Admin") on the backend. */
const SUPER_ADMIN_ROLE = 'super-admin'

export const isSuperAdminRole = (name: string): boolean =>
  name.toLowerCase() === SUPER_ADMIN_ROLE
