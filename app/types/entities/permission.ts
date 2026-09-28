import type BaseEntity from '~/types/entities/base_entity'

export default interface Permission extends BaseEntity{
  name: string
  description: string
  feature_group: string
}
