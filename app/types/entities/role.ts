import type BaseEntity from '~/types/entities/base_entity'
import type Permission from '~/types/entities/permission'

export default interface Role extends BaseEntity{
  name: string
  is_mutable: boolean
  permissions: Permission[]
}
