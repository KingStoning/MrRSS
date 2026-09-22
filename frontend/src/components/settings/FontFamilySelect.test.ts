import { mount, flushPromises } from '@vue/test-utils';
import { describe, it, expect, vi, afterEach } from 'vitest';
import FontFamilySelect from './FontFamilySelect.vue';

vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }));
vi.mock('@/utils/fontDetector', () => ({
  getSystemFonts: () => ({ all: ['Arial'] }),
  resolveFontFamily: (font: string) => `"${font}", sans-serif`,
}));
const options = {
  global: {
    stubs: {
      BaseSelect: { props: ['options', 'modelValue'], name: 'BaseSelect', template: '<div />' },
    },
  },
};
afterEach(() => vi.unstubAllGlobals());

describe('installed font selection', () => {
  it('lists native custom families and preserves a saved unavailable font', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        ok: true,
        json: async () => ['My Custom Font', 'My Custom Font', null],
      })
    );
    const wrapper = mount(FontFamilySelect, { ...options, props: { modelValue: 'Saved Font' } });
    await flushPromises();
    const groups = wrapper.findComponent({ name: 'BaseSelect' }).props('options');
    expect(groups[1].options.map((o: { value: string }) => o.value)).toEqual([
      'My Custom Font',
      'Saved Font',
    ]);
    await wrapper.get('input').setValue('User Installed Font');
    await wrapper.get('input').trigger('keydown.enter');
    expect(wrapper.emitted('update:modelValue')?.[0]).toEqual(['User Installed Font']);
    wrapper.unmount();
  });
  it('retains detected fonts when native enumeration is unavailable', async () => {
    vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new Error('offline')));
    const wrapper = mount(FontFamilySelect, { ...options, props: { modelValue: 'serif' } });
    await flushPromises();
    expect(wrapper.findComponent({ name: 'BaseSelect' }).props('options')[1].options[0].value).toBe(
      'Arial'
    );
    wrapper.unmount();
  });
});
