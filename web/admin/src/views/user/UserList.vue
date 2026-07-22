<template>
  <div class="user-center">
    <el-card shadow="never">
      <template #header>
        <div class="header">
          <div>
            <div class="title">用户管理中心</div>
            <div class="subtitle">集中查询用户账户、会员、身份、归因、积分及设备信息</div>
          </div>
        </div>
      </template>

      <el-form class="search-form" inline @submit.prevent="handleSearch">
        <el-form-item label="用户 ID / 邮箱">
          <el-input v-model="searchValue" clearable placeholder="输入用户 ID、登录邮箱或第三方邮箱" @keyup.enter="handleSearch" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :loading="searching" @click="handleSearch">查询</el-button>
        </el-form-item>
      </el-form>

      <el-empty v-if="!detail && !searching" description="请输入用户 ID 或邮箱查询" />

      <div v-if="detail" v-loading="loadingDetail">
        <el-descriptions :column="2" border class="summary">
          <el-descriptions-item label="用户 ID">{{ user.id }}</el-descriptions-item>
          <el-descriptions-item label="昵称">{{ user.username || '-' }}</el-descriptions-item>
          <el-descriptions-item label="是否会员">
            <el-tag :type="detail.is_member ? 'success' : 'info'">{{ detail.is_member ? '是' : '否' }}</el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="VIP 等级">{{ user.vip_level || 0 }}</el-descriptions-item>
          <el-descriptions-item label="VIP 开始时间">{{ formatDate(user.vip_started_at) }}</el-descriptions-item>
          <el-descriptions-item label="VIP 结束时间">{{ formatDate(user.vip_expires_at) }}</el-descriptions-item>
          <el-descriptions-item label="手机号">{{ user.phone || '-' }}</el-descriptions-item>
          <el-descriptions-item label="用户类型">{{ user.user_type === 2 ? '付费用户' : '免费用户' }}</el-descriptions-item>
          <el-descriptions-item label="是否冻结">
            <el-tag :type="user.is_frozen ? 'danger' : 'success'">{{ user.is_frozen ? '是' : '否' }}</el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="是否黑名单">
            <el-tag :type="user.is_blacklisted ? 'danger' : 'success'">{{ user.is_blacklisted ? '是' : '否' }}</el-tag>
          </el-descriptions-item>
        </el-descriptions>

        <div v-if="canManage" class="actions">
          <el-button type="primary" plain @click="openVIPDialog">添加 VIP</el-button>
          <el-button :type="user.is_frozen ? 'success' : 'warning'" plain @click="toggleFrozen">
            {{ user.is_frozen ? '解除冻结' : '冻结用户' }}
          </el-button>
          <el-button :type="user.is_blacklisted ? 'success' : 'danger'" plain @click="toggleBlacklisted">
            {{ user.is_blacklisted ? '移出黑名单' : '拉黑用户' }}
          </el-button>
          <el-button plain @click="bindPhone">绑定手机号</el-button>
          <el-button plain @click="transferVIP">转移会员</el-button>
          <el-button type="danger" plain @click="terminateVIP">终止会员</el-button>
          <el-button plain @click="extendVIP">延长会员</el-button>
          <el-button type="warning" plain @click="clearDevice">清除设备信息</el-button>
        </div>

        <el-tabs v-model="activeTab" class="detail-tabs">
          <el-tab-pane label="账户与设备" name="account">
            <el-descriptions :column="2" border>
              <el-descriptions-item label="登录账号">{{ user.login_account || user.email || '-' }}</el-descriptions-item>
              <el-descriptions-item label="登录方式">{{ loginTypeLabel(user.login_type) }}</el-descriptions-item>
              <el-descriptions-item label="设备编号">{{ user.device_code || '-' }}</el-descriptions-item>
              <el-descriptions-item label="IMEI">{{ user.imei || '-' }}</el-descriptions-item>
              <el-descriptions-item label="设备型号">{{ user.phone_model || '-' }}</el-descriptions-item>
              <el-descriptions-item label="设备国家">{{ user.device_country || '-' }}</el-descriptions-item>
              <el-descriptions-item label="最近登录 IP">{{ user.last_login_ip || '-' }}</el-descriptions-item>
              <el-descriptions-item label="最近登录时间">{{ formatDate(user.last_login_at) }}</el-descriptions-item>
              <el-descriptions-item label="积分余额">{{ formatNumber(user.points_balance) }}</el-descriptions-item>
              <el-descriptions-item label="创建时间">{{ formatDate(user.created_at) }}</el-descriptions-item>
            </el-descriptions>
          </el-tab-pane>

          <el-tab-pane :label="`第三方身份 (${detail.identities.length})`" name="identities">
            <el-table :data="detail.identities" border empty-text="暂无第三方身份">
              <el-table-column prop="provider" label="平台" width="120" />
              <el-table-column prop="email" label="邮箱" min-width="220" />
              <el-table-column prop="display_name" label="显示名称" min-width="150" />
              <el-table-column label="最后登录" min-width="180"><template #default="{ row }">{{ formatDate(row.last_login_at) }}</template></el-table-column>
            </el-table>
          </el-tab-pane>

          <el-tab-pane label="用户归因" name="attribution">
            <el-descriptions v-if="detail.attribution" :column="2" border>
              <el-descriptions-item label="渠道">{{ detail.attribution.channel_code || '-' }}</el-descriptions-item>
              <el-descriptions-item label="归因时间">{{ formatDate(detail.attribution.attributed_at) }}</el-descriptions-item>
              <el-descriptions-item label="OAID">{{ detail.attribution.oaid || '-' }}</el-descriptions-item>
              <el-descriptions-item label="Android ID">{{ detail.attribution.android_id || '-' }}</el-descriptions-item>
              <el-descriptions-item label="归因 IMEI">{{ detail.attribution.imei || '-' }}</el-descriptions-item>
              <el-descriptions-item label="归因 IP">{{ detail.attribution.ip || '-' }}</el-descriptions-item>
              <el-descriptions-item label="备注" :span="2">{{ detail.attribution.remark || '-' }}</el-descriptions-item>
            </el-descriptions>
            <el-empty v-else description="暂无归因记录" />
          </el-tab-pane>

          <el-tab-pane :label="`积分明细 (${detail.points_ledgers.length})`" name="points">
            <div class="points-summary">
              <el-tag type="success">累计收入 {{ formatNumber(detail.points_summary.income_total) }}</el-tag>
              <el-tag type="danger">累计支出 {{ formatNumber(detail.points_summary.expense_total) }}</el-tag>
            </div>
            <el-table :data="detail.points_ledgers" border empty-text="暂无积分明细">
              <el-table-column prop="source_type" label="来源" width="130" />
              <el-table-column label="变动" width="110">
                <template #default="{ row }"><span :class="row.points_change >= 0 ? 'income' : 'expense'">{{ row.points_change > 0 ? '+' : '' }}{{ row.points_change }}</span></template>
              </el-table-column>
              <el-table-column prop="balance_after" label="变动后余额" width="130" />
              <el-table-column prop="description" label="说明" min-width="220" show-overflow-tooltip />
              <el-table-column label="发生时间" min-width="180"><template #default="{ row }">{{ formatDate(row.occurred_at) }}</template></el-table-column>
            </el-table>
          </el-tab-pane>
        </el-tabs>
      </div>
    </el-card>

    <el-dialog v-model="vipDialogVisible" title="添加 VIP" width="480px">
      <el-form label-width="100px">
        <el-form-item label="VIP 等级" required><el-input-number v-model="vipForm.level" :min="1" :max="999" /></el-form-item>
        <el-form-item label="开始时间"><el-date-picker v-model="vipForm.started_at" type="datetime" value-format="YYYY-MM-DDTHH:mm:ssZ" style="width:100%" /></el-form-item>
        <el-form-item label="结束时间" required><el-date-picker v-model="vipForm.expires_at" type="datetime" value-format="YYYY-MM-DDTHH:mm:ssZ" style="width:100%" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="vipDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="operating" @click="submitVIP">确认</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  bindUserPhone, clearUserDevice, extendUserVIP, getUserCenter, grantUserVIP,
  lookupAppUser, setUserBlacklisted, setUserFrozen, terminateUserVIP, transferUserVIP,
  type UserCenterDetail,
} from '@/api/appUser'
import { useUserStore } from '@/store/user'

const userStore = useUserStore()
const canManage = computed(() => userStore.hasPermission('system:app-user:manage'))
const searchValue = ref('')
const searching = ref(false)
const loadingDetail = ref(false)
const operating = ref(false)
const detail = ref<UserCenterDetail | null>(null)
const user = computed(() => detail.value!.user)
const activeTab = ref('account')
const vipDialogVisible = ref(false)
const vipForm = reactive({ level: 1, started_at: '', expires_at: '' })

async function handleSearch() {
  const query = searchValue.value.trim()
  if (!query) { ElMessage.warning('请输入用户 ID 或邮箱'); return }
  searching.value = true
  try {
    const result: any = await lookupAppUser(query)
    await loadDetail(result.data.id)
  } finally { searching.value = false }
}

async function loadDetail(id: number) {
  loadingDetail.value = true
  try {
    const result: any = await getUserCenter(id)
    detail.value = result.data
  } finally { loadingDetail.value = false }
}

async function runOperation(message: string, operation: () => Promise<unknown>) {
  operating.value = true
  try {
    await operation()
    ElMessage.success(message)
    await loadDetail(user.value.id)
  } finally { operating.value = false }
}

function openVIPDialog() {
  vipForm.level = user.value.vip_level || 1
  vipForm.started_at = new Date().toISOString()
  const expires = new Date(); expires.setDate(expires.getDate() + 30)
  vipForm.expires_at = expires.toISOString()
  vipDialogVisible.value = true
}

async function submitVIP() {
  if (!vipForm.expires_at) { ElMessage.warning('请选择 VIP 结束时间'); return }
  await runOperation('VIP 已添加', () => grantUserVIP(user.value.id, {
    level: vipForm.level, started_at: vipForm.started_at || null, expires_at: vipForm.expires_at,
  }))
  vipDialogVisible.value = false
}

async function toggleFrozen() {
  const enabled = !user.value.is_frozen
  await ElMessageBox.confirm(`确认${enabled ? '冻结' : '解除冻结'}该用户？`, '用户状态确认', { type: 'warning' })
  await runOperation(enabled ? '用户已冻结' : '用户已解除冻结', () => setUserFrozen(user.value.id, enabled))
}

async function toggleBlacklisted() {
  const enabled = !user.value.is_blacklisted
  await ElMessageBox.confirm(`确认${enabled ? '将用户加入' : '将用户移出'}黑名单？`, '黑名单确认', { type: 'warning' })
  await runOperation(enabled ? '用户已加入黑名单' : '用户已移出黑名单', () => setUserBlacklisted(user.value.id, enabled))
}

async function bindPhone() {
  const result = await ElMessageBox.prompt('请输入要绑定的手机号', '绑定手机号', { inputValue: user.value.phone || '', inputPattern: /^\+?[0-9 -]{5,32}$/, inputErrorMessage: '手机号格式不正确' })
  await runOperation('手机号已绑定', () => bindUserPhone(user.value.id, result.value.trim()))
}

async function extendVIP() {
  const result = await ElMessageBox.prompt('请输入延长天数', '延长会员', { inputValue: '30', inputPattern: /^(?:[1-9]\d{0,2}|[1-2]\d{3}|3[0-5]\d{2}|36[0-4]\d|3650)$/, inputErrorMessage: '请输入 1 到 3650 天' })
  await runOperation('会员期限已延长', () => extendUserVIP(user.value.id, Number(result.value)))
}

async function transferVIP() {
  const result = await ElMessageBox.prompt('请输入目标用户 ID', '转移会员', { inputPattern: /^[1-9]\d*$/, inputErrorMessage: '请输入正确的用户 ID' })
  await ElMessageBox.confirm(`会员权益将转移到用户 ${result.value}，原用户会员会终止。是否继续？`, '转移确认', { type: 'warning' })
  await runOperation('会员已转移', () => transferUserVIP(user.value.id, Number(result.value)))
}

async function terminateVIP() {
  await ElMessageBox.confirm('确认立即终止该用户会员？', '终止会员', { type: 'warning' })
  await runOperation('会员已终止', () => terminateUserVIP(user.value.id))
}

async function clearDevice() {
  await ElMessageBox.confirm('将清除 IMEI、设备型号、国家、最近登录 IP 和归因设备标识，并使当前会话失效。是否继续？', '清除设备信息', { type: 'warning' })
  await runOperation('设备信息已清除', () => clearUserDevice(user.value.id))
}

function loginTypeLabel(value: number) { return value === 2 ? 'Google' : value === 3 ? 'Apple' : '游客' }
function formatNumber(value: number) { return new Intl.NumberFormat('zh-CN').format(value || 0) }
function formatDate(value?: string | null) {
  if (!value) return '-'
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString('zh-CN', { hour12: false })
}
</script>

<style scoped>
.user-center { min-width: 0; }
.header { display: flex; align-items: center; justify-content: space-between; }
.title { color: #303133; font-size: 18px; font-weight: 600; }
.subtitle { margin-top: 5px; color: #909399; font-size: 12px; }
.search-form { display: flex; align-items: center; margin-bottom: 18px; }
.search-form :deep(.el-form-item:first-child) { flex: 1; max-width: 720px; }
.search-form :deep(.el-form-item:first-child .el-form-item__content), .search-form :deep(.el-input) { width: 100%; }
.summary { max-width: 1080px; }
.actions { display: flex; flex-wrap: wrap; gap: 10px; margin: 18px 0; }
.actions .el-button { margin-left: 0; }
.detail-tabs { margin-top: 22px; }
.points-summary { display: flex; gap: 10px; margin-bottom: 12px; }
.income { color: #67c23a; font-weight: 600; }
.expense { color: #f56c6c; font-weight: 600; }
@media (max-width: 700px) {
  .search-form { align-items: stretch; flex-direction: column; }
  .search-form :deep(.el-form-item) { width: 100%; margin-right: 0; }
  .summary :deep(.el-descriptions__body) { overflow-x: auto; }
}
</style>
