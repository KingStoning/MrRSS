<script setup lang="ts">
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';
import { PhTray, PhCircle, PhStar, PhBookmarkSimple } from '@phosphor-icons/vue';
import { useAppStore } from '@/stores/app';
import { useSettings } from '@/composables/core/useSettings';
import { useArticleFilter } from '@/composables/article/useArticleFilter';
const store = useAppStore();
const { t } = useI18n();
const { settings } = useSettings();
const { clearAllFilters } = useArticleFilter();
const items = computed(() => [
  { id: 'all', icon: PhTray, label: t('sidebar.activity.allArticles') },
  { id: 'unread', icon: PhCircle, label: t('sidebar.feedList.unread') },
  { id: 'favorites', icon: PhStar, label: t('sidebar.activity.favorites') },
  { id: 'readLater', icon: PhBookmarkSimple, label: t('sidebar.activity.readLater') },
]);
function select(id: string) {
  clearAllFilters();
  store.setFilter(id as 'all' | 'unread' | 'favorites' | 'readLater' | 'imageGallery');
}
</script>

<template>
  <nav class="library-navigation" :aria-label="t('sidebar.library')">
    <div class="library-label">{{ t('sidebar.library') }}</div>
    <button
      v-for="item in items"
      :key="item.id"
      class="library-item"
      :class="{
        selected: store.currentFilter === item.id && !store.currentFeedId && !store.currentCategory,
      }"
      @click="select(item.id)"
    >
      <component :is="item.icon" :size="16" />
      <span>{{ item.label }}</span>
      <span
        v-if="item.id === 'unread' && settings.show_unread_counts && store.unreadCounts?.total"
        class="library-count"
        >{{ store.unreadCounts.total }}</span
      >
    </button>
  </nav>
</template>

<style scoped>
.library-navigation {
  padding: 12px 0 10px;
  font-family: var(--ui-font-family);
  flex-shrink: 0;
}
.library-label {
  color: var(--text-secondary);
  font-size: 11px;
  font-weight: 600;
  padding: 0 10px 6px;
}
.library-item {
  display: flex;
  align-items: center;
  gap: 9px;
  width: 100%;
  height: 30px;
  padding: 0 10px;
  border-radius: 7px;
  text-align: left;
  font-size: 13px;
}
.library-item svg {
  color: var(--text-secondary);
  flex-shrink: 0;
}
.library-item:hover {
  background: var(--hover-bg);
}
.library-item.selected {
  background: var(--selected-bg);
}
.library-count {
  margin-left: auto;
  color: var(--text-secondary);
  font-size: 11px;
}
</style>
