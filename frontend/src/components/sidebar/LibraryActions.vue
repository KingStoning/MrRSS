<script setup lang="ts">
import { PhPlus, PhArrowClockwise, PhGear } from '@phosphor-icons/vue';
import { useI18n } from 'vue-i18n';
import { useAppStore } from '@/stores/app';
const { t } = useI18n();
const store = useAppStore();
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
      <PhPlus :size="19" />
    </button>
    <button
      :title="t('article.action.refresh')"
      :aria-label="t('article.action.refresh')"
      @click="store.refreshFeeds()"
    >
      <PhArrowClockwise :size="19" />
    </button>
    <button
      :title="t('setting.tab.settings')"
      :aria-label="t('setting.tab.settings')"
      @click="dispatch('show-settings')"
    >
      <PhGear :size="19" />
    </button>
  </div>
</template>
<style scoped>
.library-actions {
  display: flex;
  gap: 12px;
  padding: 10px 14px;
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
