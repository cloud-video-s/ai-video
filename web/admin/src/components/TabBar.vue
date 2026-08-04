<template>
  <div class="tab-bar">
    <div class="tab-bar-scroll">
      <div
        v-for="tab in tabStore.tabs"
        :key="tab.path"
        class="tab-item"
        :class="{ active: tabStore.activeTab === tab.path }"
        @click="switchTab(tab.path)"
        @contextmenu.prevent="openCtxMenu($event, tab)"
      >
        <span class="tab-title">{{ tab.title }}</span>
        <el-icon
          v-if="!tab.affix"
          class="tab-close"
          @click.stop="closeTab(tab.path)"
        >
          <Close />
        </el-icon>
      </div>
    </div>

    <!-- 右键菜单 -->
    <teleport to="body">
      <div
        v-if="ctxVisible"
        class="tab-ctx-menu"
        :style="{ left: ctxX + 'px', top: ctxY + 'px' }"
      >
        <div class="ctx-item" @click="refreshTab">刷新当前</div>
        <div
          class="ctx-item"
          :class="{ disabled: ctxTab?.affix }"
          @click="closeCurrent"
        >
          关闭当前
        </div>
        <div class="ctx-item" @click="closeOthers">关闭其他</div>
        <div class="ctx-item" @click="closeAll">关闭所有</div>
      </div>
    </teleport>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { useTabStore, type TabItem } from '@/store/tab'

const router = useRouter()
const tabStore = useTabStore()

const emit = defineEmits<{ refresh: [] }>()

function switchTab(path: string) {
  tabStore.activeTab = path
  router.push(path)
}

function closeTab(path: string) {
  const redirect = tabStore.removeTab(path)
  if (redirect) router.push(redirect)
}

// --- 右键菜单 ---
const ctxVisible = ref(false)
const ctxX = ref(0)
const ctxY = ref(0)
const ctxTab = ref<TabItem | null>(null)

function openCtxMenu(e: MouseEvent, tab: TabItem) {
  ctxTab.value = tab
  ctxX.value = e.clientX
  ctxY.value = e.clientY
  ctxVisible.value = true
}

function hideCtxMenu() {
  ctxVisible.value = false
}

function refreshTab() {
  hideCtxMenu()
  emit('refresh')
}

function closeCurrent() {
  hideCtxMenu()
  if (ctxTab.value && !ctxTab.value.affix) {
    closeTab(ctxTab.value.path)
  }
}

function closeOthers() {
  hideCtxMenu()
  if (ctxTab.value) {
    tabStore.closeOthers(ctxTab.value.path)
    router.push(ctxTab.value.path)
  }
}

function closeAll() {
  hideCtxMenu()
  const path = tabStore.closeAll()
  router.push(path)
}

onMounted(() => document.addEventListener('click', hideCtxMenu))
onUnmounted(() => document.removeEventListener('click', hideCtxMenu))
</script>

<style scoped>
.tab-bar {
  display: flex;
  align-items: center;
  min-height: 45px;
  padding: 6px 16px;
  border-bottom: 1px solid #e6ebf1;
  background: #fbfcfe;
  user-select: none;
}
.tab-bar-scroll {
  display: flex;
  gap: 6px;
  min-width: 0;
  overflow-x: auto;
  flex: 1;
}
.tab-bar-scroll::-webkit-scrollbar {
  height: 0;
}
.tab-item {
  position: relative;
  display: inline-flex;
  align-items: center;
  gap: 7px;
  min-height: 31px;
  padding: 0 11px;
  border: 1px solid transparent;
  border-radius: 8px;
  background: transparent;
  color: #778397;
  cursor: pointer;
  font-size: 12px;
  white-space: nowrap;
  transition: color .18s ease, border-color .18s ease, background .18s ease, box-shadow .18s ease;
}
.tab-item:hover {
  background: #f1f5fa;
  color: #3b6fae;
}
.tab-item.active {
  border-color: #dfe8f4;
  background: #fff;
  box-shadow: 0 4px 12px rgba(31, 57, 91, .07);
  color: #2e70d5;
  font-weight: 650;
}
.tab-item.active::before {
  width: 5px;
  height: 5px;
  border-radius: 50%;
  background: #3d8df2;
  box-shadow: 0 0 0 3px rgba(61, 141, 242, .1);
  content: '';
}
.tab-close {
  display: grid;
  width: 16px;
  height: 16px;
  place-items: center;
  border-radius: 50%;
  color: #a2acb9;
  font-size: 10px;
  transition: color .18s ease, background .18s ease;
}
.tab-close:hover {
  background: #e76b71;
  color: #fff;
}

/* 右键菜单 */
.tab-ctx-menu {
  position: fixed;
  z-index: 9999;
  min-width: 136px;
  padding: 6px;
  border: 1px solid #e6ebf2;
  border-radius: 10px;
  background: #fff;
  box-shadow: 0 14px 35px rgba(22, 45, 76, .16);
}
.ctx-item {
  padding: 8px 11px;
  border-radius: 7px;
  color: #536075;
  cursor: pointer;
  font-size: 12px;
}
.ctx-item:hover {
  background: #eef5ff;
  color: #2f73df;
}
.ctx-item.disabled {
  color: #c0c4cc;
  cursor: not-allowed;
}
.ctx-item.disabled:hover {
  background: transparent;
  color: #c0c4cc;
}

@media (max-width: 760px) {
  .tab-bar { padding-right: 10px; padding-left: 10px; }
}
</style>
