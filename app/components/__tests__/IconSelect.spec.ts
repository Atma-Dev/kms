import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import { defineComponent } from 'vue'
import IconSelect from '../IconSelect.vue'

// The whole point of the component is the option markup: a lucide Vue
// component turned into a DOM node tom-select can put in its dropdown.
const TomSelectStub = defineComponent({
  props: ['modelValue', 'options', 'placeholder', 'render'],
  template: '<div />',
})

const renderOption = () => {
  const wrapper = mount(IconSelect, { global: { components: { TomSelect: TomSelectStub } } })
  const stub = wrapper.findComponent(TomSelectStub)
  return {
    options: stub.props('options') as { id: string; name: string }[],
    render: (stub.props('render') as { option: (d: { value: string; text: string }) => HTMLElement }).option,
  }
}

describe('IconSelect', () => {
  it('offers the lucide icon names', () => {
    const { options } = renderOption()
    expect(options.length).toBeGreaterThan(1000)
    expect(options.some((o) => o.id === 'Book')).toBe(true)
  })

  it('draws an svg preview next to the name', () => {
    const { render } = renderOption()
    const el = render({ value: 'Book', text: 'Book' })
    expect(el.querySelector('svg')).not.toBeNull()
    expect(el.textContent).toContain('Book')
  })

  it('survives the empty placeholder option', () => {
    const { render } = renderOption()
    const el = render({ value: '', text: 'Select an icon…' })
    expect(el.querySelector('svg')).toBeNull()
    expect(el.textContent).toContain('Select an icon…')
  })
})
