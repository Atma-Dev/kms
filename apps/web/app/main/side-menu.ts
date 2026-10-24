import { type Icon } from '~/base/ui/lucide'
import { Permission } from '~/enums/Permission'

export interface Menu {
  icon?: Icon
  title?: string
  route_name?: string
  params?: any
  badge?: number
  sub_menu?: Menu[]
  /** Hidden from the menu unless the user holds this permission. */
  permission?: Permission
}

const mainMenu: (string | Menu)[] = [
  'nav.groups.dashboard',
  {
    icon: 'House',
    route_name: 'home',
    title: 'nav.items.home',
  },
  'nav.groups.knowledge',
  {
    icon: 'Search',
    route_name: 'search',
    title: 'nav.items.search',
    // Searching is reading documents — the API gates /search on the same
    // permission, so the menu entry hides for anyone it would 403.
    permission: Permission.DOCUMENTS_SHOW,
  },
  {
    icon: 'BookOpen',
    route_name: 'spaces',
    title: 'nav.items.spaces',
    permission: Permission.MASTERS_SPACES_SHOW,
  },
  {
    icon: 'Tags',
    route_name: 'tags',
    title: 'nav.items.tags',
    permission: Permission.MASTERS_TAGS_SHOW,
  },
  'nav.groups.admin',
  {
    icon: 'UserRoundCog',
    route_name: 'account',
    title: 'nav.items.account',
    // /account redirects to /account/users.
    permission: Permission.MANAGEMENT_USERS_SHOW,
  },
  {
    icon: 'KeySquare',
    route_name: 'm2m-clients',
    title: 'nav.items.m2mClients',
    permission: Permission.MANAGEMENT_M2M_CLIENTS_SHOW,
  },
  {
    icon: 'Cog',
    title: 'nav.items.system',
    sub_menu: [
      {
        icon: 'History',
        route_name: 'activity-logs',
        title: 'nav.items.activityLogs',
        permission: Permission.MANAGEMENT_ACTIVITY_LOGS_SHOW,
      },
      {
        icon: 'FileDiff',
        route_name: 'audits',
        title: 'nav.items.audits',
        permission: Permission.MANAGEMENT_AUDITS_SHOW,
      },
      {
        icon: 'LayoutGrid',
        route_name: 'jobs-dashboard',
        title: 'nav.items.jobsDashboard',
        permission: Permission.MANAGEMENT_JOB_LOGS_SHOW,
      },
      {
        icon: 'ScrollText',
        route_name: 'job-logs',
        title: 'nav.items.jobLogs',
        permission: Permission.MANAGEMENT_JOB_LOGS_SHOW,
      },
      {
        icon: 'ListOrdered',
        route_name: 'queue',
        title: 'nav.items.queue',
        permission: Permission.MANAGEMENT_QUEUE_SHOW,
      },
      {
        icon: 'CalendarClock',
        route_name: 'schedules',
        title: 'nav.items.schedules',
        permission: Permission.MANAGEMENT_SCHEDULES_SHOW,
      },
    ],
  },
]

export default mainMenu
