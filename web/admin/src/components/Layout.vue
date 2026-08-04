<template>
  <el-container class="layout-container">
    <el-aside :width="isCollapse ? '72px' : '232px'" class="layout-aside">
      <div class="logo">
        <span class="logo-mark"><el-icon><VideoCameraFilled /></el-icon></span>
        <span v-show="!isCollapse" class="logo-copy">
          <strong>AI VIDEO</strong>
          <small>ADMIN CENTER</small>
        </span>
      </div>
      <el-menu
        :default-active="route.path"
        :collapse="isCollapse"
        router
        class="aside-menu"
      >
        <el-menu-item index="/dashboard">
          <el-icon><Odometer /></el-icon>
          <template #title>控制台</template>
        </el-menu-item>
        <MenuItem v-for="menu in userStore.menuTree" :key="menu.id" :item="menu" />
      </el-menu>
    </el-aside>
    <el-container class="layout-workspace">
      <el-header class="layout-header">
        <div class="header-left">
          <button class="collapse-btn" type="button" aria-label="切换侧边栏" @click="isCollapse = !isCollapse">
            <el-icon :size="18"><Fold v-if="!isCollapse" /><Expand v-else /></el-icon>
          </button>
          <span class="header-divider" />
          <div class="page-context">
            <span>管理后台</span>
            <strong>{{ pageTitle }}</strong>
          </div>
        </div>
        <div class="header-right">
          <span class="system-state"><i />系统运行中</span>
          <el-dropdown @command="handleCommand">
            <span class="user-info">
              <span class="user-avatar">{{ userInitial }}</span>
              <span class="user-copy">
                <strong>{{ displayName }}</strong>
                <small>管理员</small>
              </span>
              <el-icon class="dropdown-arrow"><ArrowDown /></el-icon>
            </span>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="logout"><el-icon><SwitchButton /></el-icon>退出登录</el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </el-header>
      <TabBar ref="tabBarRef" @refresh="handleRefresh" />
      <el-main class="layout-main">
        <router-view v-slot="{ Component }">
          <keep-alive :include="tabStore.cachedNames()">
            <component :is="Component" v-if="showPage" :key="route.path" />
          </keep-alive>
        </router-view>
      </el-main>
    </el-container>
  </el-container>
</template>

<script setup lang="ts">
import { computed, ref, nextTick, onBeforeUnmount, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useUserStore } from '@/store/user'
import { useTabStore } from '@/store/tab'
import TabBar from './TabBar.vue'
import MenuItem from './MenuItem.vue'

const route = useRoute()
const router = useRouter()
const userStore = useUserStore()
const tabStore = useTabStore()
const isCollapse = ref(false)
const showPage = ref(true)
const tabBarRef = ref()
const pageTitle = computed(() => String(route.meta.title || '管理后台'))
const displayName = computed(() => userStore.userInfo?.nickname || userStore.userInfo?.username || '管理员')
const userInitial = computed(() => displayName.value.trim().charAt(0).toUpperCase() || 'A')

function handleViewportChange() {
  if (window.innerWidth < 960) isCollapse.value = true
}

async function handleRefresh() {
  showPage.value = false
  await nextTick()
  showPage.value = true
}

onMounted(async () => {
  handleViewportChange()
  window.addEventListener('resize', handleViewportChange)
  if (userStore.token) {
    try {
      if (!userStore.userInfo) {
        await userStore.fetchProfile()
      }
      if (userStore.menuTree.length === 0) {
        await userStore.fetchMenus()
      }
      if (userStore.permissions.length === 0) {
        await userStore.fetchPermissions()
      }
    } catch {
      // 401 已由 request 拦截器统一清登录态并跳转；
      // 其他错误（网络抖动 / 5xx）不强制登出，避免把用户误踢下线
    }
  }
})

onBeforeUnmount(() => window.removeEventListener('resize', handleViewportChange))

async function handleCommand(command: string) {
  if (command === 'logout') {
    await userStore.logout()
    tabStore.closeAll()
    router.push('/login')
  }
}
</script>

<style scoped>
.layout-container {
  height: 100vh;
  background: #f3f6fa;
}
.layout-aside {
  position: relative;
  z-index: 20;
  height: 100vh;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  background:
    radial-gradient(circle at 18% 5%, rgba(50, 142, 255, .19), transparent 28%),
    linear-gradient(180deg, #101d35 0%, #0b1629 56%, #091321 100%);
  box-shadow: 8px 0 28px rgba(22, 41, 70, .1);
  transition: width .25s ease;
}
.layout-aside::after {
  position: absolute;
  right: 0;
  bottom: 0;
  width: 1px;
  height: 76%;
  background: linear-gradient(transparent, rgba(120, 174, 235, .18));
  content: '';
  pointer-events: none;
}
.layout-workspace {
  min-width: 0;
  overflow: hidden;
}
.logo {
  height: 68px;
  flex: 0 0 68px;
  display: flex;
  align-items: center;
  gap: 11px;
  padding: 0 16px;
  box-sizing: border-box;
  color: #fff;
  border-bottom: 1px solid rgba(255, 255, 255, .075);
}
.logo-mark {
  display: grid;
  flex: 0 0 40px;
  width: 40px;
  height: 40px;
  place-items: center;
  border: 1px solid rgba(132, 206, 255, .28);
  border-radius: 12px;
  background: linear-gradient(145deg, #2d84f1, #51b8eb);
  box-shadow: 0 8px 22px rgba(34, 128, 229, .28);
  font-size: 20px;
}
.logo-copy {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 3px;
  white-space: nowrap;
}
.logo-copy strong { font-size: 15px; font-weight: 750; letter-spacing: 1.3px; }
.logo-copy small { color: #8298b8; font-size: 8px; font-weight: 650; letter-spacing: 2.1px; }
.aside-menu {
  flex: 1 1 auto;
  min-height: 0;
  padding: 13px 9px 22px;
  overflow-x: hidden;
  overflow-y: auto;
  border-right: none;
  background: transparent;
}
.aside-menu:deep(.el-menu) {
  border: 0;
  background: transparent;
}
.aside-menu:deep(.el-menu-item),
.aside-menu:deep(.el-sub-menu__title) {
  height: 44px;
  margin: 3px 0;
  border-radius: 10px;
  color: #9aabc2;
  font-size: 13px;
  transition: color .18s ease, background .18s ease, transform .18s ease;
}
.aside-menu:deep(.el-menu-item:hover),
.aside-menu:deep(.el-sub-menu__title:hover) {
  background: rgba(255, 255, 255, .055);
  color: #e8f2ff;
}
.aside-menu:deep(.el-menu-item.is-active) {
  background: linear-gradient(100deg, rgba(50, 132, 240, .28), rgba(54, 172, 235, .12));
  box-shadow: inset 3px 0 #52b6f0;
  color: #fff;
  font-weight: 600;
}
.aside-menu:deep(.el-menu-item.is-active .el-icon) { color: #7dcdf8; }
.aside-menu:deep(.el-menu-item .el-icon),
.aside-menu:deep(.el-sub-menu__title .el-icon) {
  font-size: 17px;
}
.aside-menu:deep(.el-sub-menu .el-menu-item) {
  min-width: 0;
  padding-left: 50px !important;
  background: transparent;
}
.aside-menu.el-menu--collapse:deep(.el-menu-item),
.aside-menu.el-menu--collapse:deep(.el-sub-menu__title) {
  justify-content: center;
  padding: 0 !important;
}
.aside-menu::-webkit-scrollbar {
  width: 4px;
}
.aside-menu::-webkit-scrollbar-thumb {
  background: rgba(148, 175, 207, .2);
  border-radius: 999px;
}
.aside-menu::-webkit-scrollbar-track {
  background: transparent;
}
.layout-header {
  height: 64px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 22px 0 18px;
  border-bottom: 1px solid #e8edf4;
  background: rgba(255, 255, 255, .96);
  box-shadow: 0 3px 14px rgba(25, 48, 79, .025);
}
.header-left {
  display: flex;
  align-items: center;
  min-width: 0;
}
.collapse-btn {
  display: grid;
  width: 36px;
  height: 36px;
  padding: 0;
  place-items: center;
  border: 1px solid #e7ebf1;
  border-radius: 10px;
  background: #f8fafc;
  color: #5d6b7f;
  cursor: pointer;
  transition: border-color .18s ease, color .18s ease, background .18s ease;
}
.collapse-btn:hover {
  border-color: #cfe1f8;
  background: #f0f6ff;
  color: #3277df;
}
.header-divider {
  width: 1px;
  height: 24px;
  margin: 0 15px;
  background: #e6ebf1;
}
.page-context {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 3px;
}
.page-context span { color: #a0a9b7; font-size: 9px; font-weight: 600; letter-spacing: 1px; }
.page-context strong { overflow: hidden; color: #253044; font-size: 15px; font-weight: 700; text-overflow: ellipsis; white-space: nowrap; }
.header-right {
  display: flex;
  align-items: center;
  gap: 18px;
}
.system-state {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  color: #748196;
  font-size: 11px;
}
.system-state i {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: #2cc597;
  box-shadow: 0 0 0 4px rgba(44, 197, 151, .11);
}
.user-info {
  display: flex;
  align-items: center;
  gap: 9px;
  padding: 5px 7px 5px 5px;
  border: 1px solid transparent;
  border-radius: 11px;
  color: #344054;
  cursor: pointer;
  outline: none;
  transition: border-color .18s ease, background .18s ease;
}
.user-info:hover { border-color: #e5ebf2; background: #f8fafc; }
.user-avatar {
  display: grid;
  width: 32px;
  height: 32px;
  place-items: center;
  border-radius: 9px;
  background: linear-gradient(145deg, #e8f2ff, #d8eaff);
  color: #3374ce;
  font-size: 12px;
  font-weight: 750;
}
.user-copy { display: flex; min-width: 76px; flex-direction: column; gap: 2px; }
.user-copy strong { max-width: 120px; overflow: hidden; font-size: 12px; font-weight: 650; text-overflow: ellipsis; white-space: nowrap; }
.user-copy small { color: #9aa5b4; font-size: 9px; }
.dropdown-arrow { color: #9ca7b5; font-size: 11px; }
.layout-main {
  padding: 20px;
  overflow-y: auto;
  background: #f3f6fa;
}

@media (max-width: 760px) {
  .layout-header { padding: 0 12px; }
  .header-divider, .system-state, .user-copy { display: none; }
  .page-context { margin-left: 10px; }
  .header-right { gap: 8px; }
  .layout-main { padding: 14px; }
}
</style>
