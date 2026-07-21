import { createRouter, createWebHashHistory, type RouteRecordRaw } from 'vue-router'
import { useTabStore } from '@/store/tab'

const routes: RouteRecordRaw[] = [
  {
    path: '/login',
    name: 'Login',
    component: () => import('@/views/Login.vue'),
    meta: { title: '登录', noAuth: true },
  },
  {
    path: '/',
    component: () => import('@/components/Layout.vue'),
    redirect: '/dashboard',
    children: [
      {
        path: 'dashboard',
        name: 'Dashboard',
        component: () => import('@/views/Dashboard.vue'),
        meta: { title: '控制台' },
      },
      {
        path: 'system/admin',
        name: 'SystemAdmin',
        component: () => import('@/views/system/AdminList.vue'),
        meta: { title: '用户管理' },
      },
      {
        path: 'system/role',
        name: 'SystemRole',
        component: () => import('@/views/system/RoleList.vue'),
        meta: { title: '角色管理' },
      },
      {
        path: 'system/menu',
        name: 'SystemMenu',
        component: () => import('@/views/system/MenuList.vue'),
        meta: { title: '菜单管理' },
      },
      {
        path: 'system/config',
        name: 'SystemConfig',
        component: () => import('@/views/system/ConfigList.vue'),
        meta: { title: '系统配置' },
      },
      {
        path: 'system/delay-config',
        name: 'SystemDelayConfig',
        component: () => import('@/views/system/DelayConfigList.vue'),
        meta: { title: 'OB 延迟配置' },
      },
      {
        path: 'system/country',
        name: 'SystemCountry',
        component: () => import('@/views/system/CountryList.vue'),
        meta: { title: '国家管理' },
      },
      {
        path: 'system/operlog',
        name: 'SystemOperLog',
        component: () => import('@/views/system/OperationLogList.vue'),
        meta: { title: '操作日志' },
      },
      {
        path: 'subscription/vip',
        name: 'VIPSubscriptionList',
        component: () => import('@/views/subscription/VIPSubscriptionList.vue'),
        meta: { title: 'VIP 订阅' },
      },
      {
        path: 'package/list',
        name: 'PackageList',
        component: () => import('@/views/package/PackageList.vue'),
        meta: { title: '安装包管理' },
      },
      {
        path: 'channel/list',
        name: 'ChannelList',
        component: () => import('@/views/channel/ChannelList.vue'),
        meta: { title: '渠道管理' },
      },
      {
        path: 'template/positions',
        name: 'DisplayPositionList',
        component: () => import('@/views/template/DisplayPositionList.vue'),
        meta: { title: '展示位置' },
      },
      {
        path: 'template/types',
        name: 'TemplateTypeList',
        component: () => import('@/views/template/TemplateTypeList.vue'),
        meta: { title: '模板分类' },
      },
      {
        path: 'template/list',
        name: 'VideoTemplateList',
        component: () => import('@/views/template/TemplateList.vue'),
        meta: { title: '视频模板' },
      },
      {
        path: 'template/banners',
        name: 'BannerList',
        component: () => import('@/views/template/BannerList.vue'),
        meta: { title: 'Banner 管理' },
      },
      {
        path: 'template/display-configs',
        name: 'TemplateDisplayConfigList',
        component: () => import('@/views/template/TemplateDisplayConfigList.vue'),
        meta: { title: '模板展示配置' },
      },
      {
        path: 'user/list',
        name: 'UserList',
        component: () => import('@/views/user/UserList.vue'),
        meta: { title: '客户端用户' },
      },
      {
        path: 'attribution/list',
        name: 'AttributionList',
        component: () => import('@/views/attribution/AttributionList.vue'),
        meta: { title: '用户归因' },
      },
      {
        path: 'subscription/points',
        name: 'PointsPackageList',
        component: () => import('@/views/subscription/PointsPackageList.vue'),
        meta: { title: '积分套餐' },
      },
      {
        path: 'subscription/points-ledger',
        name: 'UserPointsLedgerList',
        component: () => import('@/views/subscription/UserPointsLedgerList.vue'),
        meta: { title: '积分明细' },
      },
    ],
  },
]

const router = createRouter({
  history: createWebHashHistory(),
  routes,
})

router.beforeEach((to, _from, next) => {
  document.title = `${to.meta.title || ''} - AI Video Admin`
  const token = localStorage.getItem('token')
  if (!to.meta.noAuth && !token) {
    next('/login')
    return
  }

  if (to.name && to.meta.title && !to.meta.noAuth) {
    const tabStore = useTabStore()
    tabStore.addTab({
      path: to.path,
      name: to.name as string,
      title: to.meta.title as string,
      affix: to.path === '/dashboard',
    })
  }

  next()
})

export default router
