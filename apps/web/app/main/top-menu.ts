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

const mainMenu: Array<Menu> = [
  {
    icon: 'House',
    route_name: 'home',
    title: 'Home',
  },
  {
    icon: 'BookOpen',
    route_name: 'spaces',
    title: 'Spaces',
    permission: Permission.MASTERS_SPACES_SHOW,
  },
  {
    icon: 'Tags',
    route_name: 'tags',
    title: 'Tags',
    permission: Permission.MASTERS_TAGS_SHOW,
  },
  {
    icon: 'UserRoundCog',
    route_name: 'account',
    title: 'Account',
    // /account redirects to /account/users.
    permission: Permission.MANAGEMENT_USERS_SHOW,
  },
  {
    icon: 'KeySquare',
    route_name: 'm2m-clients',
    title: 'M2M Clients',
    permission: Permission.MANAGEMENT_M2M_CLIENTS_SHOW,
  },
  {
    icon: 'Cog',
    title: 'System',
    sub_menu: [
      {
        icon: 'History',
        route_name: 'activity-logs',
        title: 'Activity Logs',
        permission: Permission.MANAGEMENT_ACTIVITY_LOGS_SHOW,
      },
      {
        icon: 'FileDiff',
        route_name: 'audits',
        title: 'Audits',
        permission: Permission.MANAGEMENT_AUDITS_SHOW,
      },
      {
        icon: 'LayoutGrid',
        route_name: 'jobs-dashboard',
        title: 'Jobs Dashboard',
        permission: Permission.MANAGEMENT_JOB_LOGS_SHOW,
      },
      {
        icon: 'ScrollText',
        route_name: 'job-logs',
        title: 'Job Logs',
        permission: Permission.MANAGEMENT_JOB_LOGS_SHOW,
      },
      {
        icon: 'ListOrdered',
        route_name: 'queue',
        title: 'Queue',
        permission: Permission.MANAGEMENT_QUEUE_SHOW,
      },
      {
        icon: 'CalendarClock',
        route_name: 'schedules',
        title: 'Schedules',
        permission: Permission.MANAGEMENT_SCHEDULES_SHOW,
      },
    ],
  },
]

export default mainMenu
