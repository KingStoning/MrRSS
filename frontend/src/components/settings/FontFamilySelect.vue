<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import BaseSelect from '@/components/common/BaseSelect.vue';
import { getSystemFonts, resolveFontFamily } from '@/utils/fontDetector';
const props = defineProps<{ modelValue: string }>();
const emit = defineEmits<{ 'update:modelValue': [value: string | number] }>();
const { t } = useI18n();
const fonts = ref<string[]>([]);
const loading = ref(false);
const customFont = ref('');
const presets = computed(() => [
  { value: 'system', label: t('setting.typography.fontSystemDefault') },
  { value: 'serif', label: t('setting.typography.fontSerifDefault') },
  { value: 'sans-serif', label: t('setting.typography.fontSansSerifDefault') },
  { value: 'monospace', label: t('setting.typography.fontMonospaceDefault') },
  { value: 'hyperlegible', label: t('setting.typography.fontHyperlegible') },
]);
const fontOptions = computed(() => [
  { label: t('setting.typography.fontSystem'), options: presets.value },
  {
    label: t('setting.typography.installedFonts'),
    options: [
      ...new Set([
        ...fonts.value,
        ...(!presets.value.some((p) => p.value === props.modelValue) ? [props.modelValue] : []),
      ]),
    ]
      .filter(Boolean)
      .sort((a, b) => a.localeCompare(b))
      .map((font) => ({
        value: font,
        label: font,
        style: { fontFamily: resolveFontFamily(font) },
      })),
  },
]);
async function loadFonts(requestAccess = false) {
  loading.value = true;
  try {
    const browser = window as Window & { queryLocalFonts?: () => Promise<{ family: string }[]> };
    if (requestAccess && browser.queryLocalFonts) {
      fonts.value = (await browser.queryLocalFonts()).map((font) => font.family);
      return;
    }
    const res = await fetch('/api/settings/fonts');
    if (res.ok) {
      const families: unknown = await res.json();
      if (Array.isArray(families))
        fonts.value = families.filter((f): f is string => typeof f === 'string' && !!f.trim());
    } else if (requestAccess) throw new Error('Font access unavailable');
  } catch {
    if (requestAccess) window.showToast?.(t('setting.typography.fontAccessFailed'), 'error');
  } finally {
    loading.value = false;
  }
}
function applyCustom() {
  const family = customFont.value.trim();
  if (family) {
    fonts.value.push(family);
    emit('update:modelValue', family);
    customFont.value = '';
  }
}
onMounted(() => {
  fonts.value = getSystemFonts().all;
  void loadFonts();
});
</script>
<template>
  <div class="font-picker">
    <BaseSelect
      :model-value="modelValue"
      :options="fontOptions"
      searchable
      width="w-full"
      max-height="max-h-60"
      @update:model-value="emit('update:modelValue', $event)"
    >
      <template #option="{ option }"
        ><span :style="option.style">{{ option.label }}</span></template
      >
    </BaseSelect>
    <div class="font-custom">
      <input
        v-model="customFont"
        :aria-label="t('setting.typography.customFont')"
        :placeholder="t('setting.typography.customFont')"
        @keydown.enter.prevent="applyCustom"
      />
      <button type="button" :disabled="!customFont.trim()" @click="applyCustom">
        {{ t('common.form.apply') }}
      </button>
    </div>
    <button type="button" class="font-load" :disabled="loading" @click="loadFonts(true)">
      {{ t('setting.typography.loadFonts') }}
    </button>
  </div>
</template>
<style scoped>
.font-picker {
  width: min(280px, 100%);
  display: grid;
  gap: 8px;
}
.font-custom {
  display: flex;
  gap: 6px;
}
.font-custom input {
  min-width: 0;
  width: 100%;
  padding: 7px 9px;
  border: 1px solid var(--border-color);
  border-radius: 6px;
  background: var(--bg-primary);
  font-size: 12px;
}
.font-custom button,
.font-load {
  font-size: 12px;
  color: var(--accent-color);
  white-space: nowrap;
}
.font-load {
  text-align: left;
}
button:disabled {
  opacity: 0.5;
}
</style>
