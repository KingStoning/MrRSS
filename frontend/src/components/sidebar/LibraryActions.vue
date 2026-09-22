<script setup lang="ts">
import { PhPlus, PhArrowClockwise, PhGear, PhImages } from '@phosphor-icons/vue';
import { useI18n } from 'vue-i18n';
import { useSettings } from '@/composables/core/useSettings';
import { useAppStore } from '@/stores/app';
const { t } = useI18n();
const store = useAppStore();
const { settings } = useSettings();
function dispatch(name: string) {
  window.dispatchEvent(new CustomEvent(name));
}
</script>
<template>
  <div class="library-actions">
    <button
      :title="t('sidebar.activity.addFeed')"
      :aria-label="t('sidebar.activity.addFeed')"
      @click="dispatch('show-add-feed')"
    >
      <PhPlus :size="16" />
    </button>
    <button
      :title="t('article.action.refresh')"
      :aria-label="t('article.action.refresh')"
      @click="store.refreshFeeds()"
    >
      <PhArrowClockwise :size="16" />
    </button>
    <button
      v-if="settings.image_gallery_enabled"
      :title="t('sidebar.activity.imageGallery')"
      :aria-label="t('sidebar.activity.imageGallery')"
      @click="store.setFilter('imageGallery')"
    >
      <PhImages :size="16" />
    </button>
    <button
      :title="t('setting.tab.settings')"
      :aria-label="t('setting.tab.settings')"
      @click="dispatch('show-settings')"
    >
      <PhGear :size="16" />
    </button>
  </div>
</template>
<style scoped>
.library-actions {
  display: flex;
  gap: 12px;
  padding: 7px 12px;
  flex-shrink: 0;
  border-top: 1px solid var(--border-color);
  color: var(--text-secondary);
}
.library-actions button {
  padding: 6px;
  border-radius: 5px;
}
.library-actions button:hover {
  background: var(--hover-bg);
}
.library-actions button:last-child {
  margin-left: auto;
}
</style>
