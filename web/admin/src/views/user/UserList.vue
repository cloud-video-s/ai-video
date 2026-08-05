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

      <el-form class="list-filters" inline @submit.prevent="handleListSearch">
        <el-form-item label="关键词">
          <el-input v-model="filters.keyword" clearable placeholder="ID、昵称、账号、邮箱、手机或设备" @keyup.enter="handleListSearch" />
        </el-form-item>
        <el-form-item label="国家">
          <el-input v-model="filters.device_country" clearable placeholder="例如 CN" />
        </el-form-item>
        <el-form-item label="渠道">
          <el-input v-model="filters.channel_id" clearable placeholder="渠道编码" />
        </el-form-item>
        <el-form-item label="用户类型">
          <el-select v-model="filters.user_type" clearable placeholder="全部" style="width: 120px">
            <el-option label="免费用户" :value="1" />
            <el-option label="付费用户" :value="2" />
          </el-select>
        </el-form-item>
        <el-form-item label="登录方式">
          <el-select v-model="filters.login_type" clearable placeholder="全部" style="width: 120px">
            <el-option label="游客" :value="1" />
            <el-option label="Google" :value="2" />
            <el-option label="Apple" :value="3" />
          </el-select>
        </el-form-item>
        <el-form-item label="状态">
          <el-select v-model="filters.status" clearable placeholder="全部" style="width: 110px">
            <el-option label="正常" :value="1" />
            <el-option label="禁用" :value="0" />
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="handleListSearch">查询</el-button>
          <el-button @click="resetListFilters">重置</el-button>
        </el-form-item>
      </el-form>

      <el-table v-loading="listLoading" :data="listData" row-key="id" stripe>
        <el-table-column prop="id" label="用户 ID" width="95" />
        <el-table-column label="用户" min-width="210">
          <template #default="{ row }">
            <div class="primary-text">{{ row.username || '-' }}</div>
            <div class="secondary-text">{{ row.login_account || row.email || row.phone || '-' }}</div>
          </template>
        </el-table-column>
        <el-table-column label="应用/版本" min-width="170">
          <template #default="{ row }">
            <div class="primary-text">{{ row.app_name || '-' }}</div>
            <div class="secondary-text">{{ row.app_version || '-' }} · {{ row.package_code || '-' }}</div>
          </template>
        </el-table-column>
        <el-table-column label="国家/渠道" min-width="150">
          <template #default="{ row }">
            <div>{{ row.client_country || row.server_country || '-' }}</div>
            <div class="secondary-text">{{ row.channel_id || '-' }}</div>
          </template>
        </el-table-column>
        <el-table-column label="类型" width="120">
          <template #default="{ row }">
            <el-tag :type="row.user_type === 2 ? 'warning' : 'info'" size="small">{{ row.user_type === 2 ? '付费' : '免费' }}</el-tag>
            <div class="secondary-text">{{ loginTypeLabel(row.login_type) }}</div>
          </template>
        </el-table-column>
        <el-table-column label="会员/积分" min-width="145">
          <template #default="{ row }">
            <div>VIP {{ row.vip_level || 0 }} · {{ subscriptionLabel(row.subscription_status) }}</div>
            <div class="secondary-text">积分 {{ formatNumber(row.points_balance) }}</div>
          </template>
        </el-table-column>
        <el-table-column label="账号状态" width="150">
          <template #default="{ row }">
            <el-tag :type="row.status === 1 ? 'success' : 'danger'" size="small">{{ row.status === 1 ? '正常' : '禁用' }}</el-tag>
            <el-tag v-if="flagActive(row.is_frozen)" type="warning" size="small" class="status-tag">冻结</el-tag>
            <el-tag v-if="flagActive(row.is_blacklisted)" type="danger" size="small" class="status-tag">黑名单</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="最近活跃" min-width="170">
          <template #default="{ row }">{{ formatDate(row.last_opened_at || row.last_login_at) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="90" fixed="right">
          <template #default="{ row }"><el-button link type="primary" @click="openDetail(row)">详情</el-button></template>
        </el-table-column>
      </el-table>

      <div class="pagination-wrap">
        <el-pagination
          v-model:current-page="page"
          v-model:page-size="pageSize"
          :total="total"
          :page-sizes="[10, 20, 50, 100]"
          layout="total, sizes, prev, pager, next"
          @size-change="handlePageSizeChange"
          @current-change="fetchList"
        />
      </div>
    </el-card>

    <el-drawer v-model="detailVisible" title="用户详情与管理" size="min(960px, 90vw)" destroy-on-close>
      <el-skeleton v-if="loadingDetail && !detail" :rows="10" animated />
      <div v-if="detail" v-loading="loadingDetail">
        <el-descriptions :column="2" border class="summary">
          <el-descriptions-item label="用户 ID">{{ user.id }}</el-descriptions-item>
          <el-descriptions-item label="昵称">{{ user.username || '-' }}</el-descriptions-item>
          <el-descriptions-item label="用户类型">{{ user.user_type == 2 ? '付费用户' : '免费用户' }}</el-descriptions-item>
          <el-descriptions-item label="是否订阅">
            <el-tag :type="user.subscription_status == 2 ? 'success' : 'info'">{{ user.subscription_status == 2 ? '是' : '否' }}</el-tag>
          </el-descriptions-item>

<!--          <el-descriptions-item label="订阅 等级">{{ user.vip_level || 0 }}</el-descriptions-item>-->

          <el-descriptions-item label="订阅 开始时间">{{ formatDate(user.vip_started_at) }}</el-descriptions-item>
          <el-descriptions-item label="订阅 结束时间">{{ formatDate(user.vip_expires_at) }}</el-descriptions-item>
          <el-descriptions-item label="订阅 积分">{{ user.vip_points || 0 }}</el-descriptions-item>
          <el-descriptions-item label="自购 积分">{{ user.points_balance || 0 }}</el-descriptions-item>
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
              <el-descriptions-item label="设备国家">{{ user.client_country || user.server_country || '-' }}</el-descriptions-item>
              <el-table-column prop="email" label="邮箱" min-width="220" />
              <el-descriptions-item label="最近登录 IP">{{ user.last_login_ip || '-' }}</el-descriptions-item>
              <el-descriptions-item label="最近登录时间">{{ formatDate(user.last_login_at) }}</el-descriptions-item>
<!--              <el-descriptions-item label="积分余额">{{ formatNumber(user.points_balance) }}</el-descriptions-item>-->
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

          <el-tab-pane :label="`积分明细 (${detail.points_ledger_total})`" name="points">
            <div class="table-toolbar">
              <div class="points-summary">
                <el-tag type="success">累计收入 {{ formatNumber(detail.points_summary.income_total) }}</el-tag>
                <el-tag type="danger">累计支出 {{ formatNumber(detail.points_summary.expense_total) }}</el-tag>
              </div>
              <span v-if="detail.points_ledger_total > detail.points_ledgers.length" class="result-hint">
                显示最近 {{ detail.points_ledgers.length }} 条
              </span>
            </div>
            <el-table :data="detail.points_ledgers" border empty-text="暂无积分明细">
              <el-table-column prop="source_type" label="来源" width="130" />
              <el-table-column label="变动" width="110">
                <template #default="{ row }"><span :class="row.points_change >= 0 ? 'income' : 'expense'">{{ row.points_change > 0 ? '+' : '' }}{{ row.points_change }}</span></template>
              </el-table-column>
              <el-table-column prop="balance_after" label="变动后余额" width="130" />
              <el-table-column label="关联业务" min-width="160" show-overflow-tooltip>
                <template #default="{ row }">{{ ledgerBusinessLabel(row) }}</template>
              </el-table-column>
              <el-table-column prop="description" label="说明" min-width="220" show-overflow-tooltip />
              <el-table-column label="发生时间" min-width="180"><template #default="{ row }">{{ formatDate(row.occurred_at) }}</template></el-table-column>
            </el-table>
          </el-tab-pane>

          <el-tab-pane :label="`作品 (${detail.work_total})`" name="works">
            <div v-if="detail.work_total > detail.works.length" class="result-hint table-hint">
              显示最近 {{ detail.works.length }} 条
            </div>
            <el-table :data="detail.works" border empty-text="暂无作品">
              <el-table-column prop="id" label="作品 ID" width="95" />
              <el-table-column prop="model_config_id" label="模型 ID" width="95" />
              <el-table-column prop="external_task_id" label="平台任务 ID" min-width="170" show-overflow-tooltip />
              <el-table-column label="状态" width="110">
                <template #default="{ row }">
                  <el-tag :type="workStatusType(row.status)" size="small">{{ workStatusLabel(row.status) }}</el-tag>
                </template>
              </el-table-column>
              <el-table-column label="进度" width="90"><template #default="{ row }">{{ row.progress }}%</template></el-table-column>
              <el-table-column label="生成耗时" width="110"><template #default="{ row }">{{ formatDuration(row.usage_duration) }}</template></el-table-column>
              <el-table-column label="提交时间" min-width="175"><template #default="{ row }">{{ formatDate(row.submitted_at) }}</template></el-table-column>
              <el-table-column label="完成时间" min-width="175"><template #default="{ row }">{{ formatDate(row.finished_at) }}</template></el-table-column>
              <el-table-column label="创建时间" min-width="175"><template #default="{ row }">{{ formatDate(row.created_at) }}</template></el-table-column>
            </el-table>
          </el-tab-pane>

          <el-tab-pane :label="`订单 (${detail.order_total})`" name="orders">
            <div v-if="detail.order_total > detail.orders.length" class="result-hint table-hint">
              显示最近 {{ detail.orders.length }} 条
            </div>
            <el-table :data="detail.orders" border empty-text="暂无订单">
              <el-table-column prop="order_no" label="订单号" min-width="180" show-overflow-tooltip />
              <el-table-column label="商品" min-width="180">
                <template #default="{ row }">
                  <div class="primary-text">{{ row.product_name || '-' }}</div>
                  <div class="secondary-text">{{ row.product_code || row.product_type || '-' }}</div>
                </template>
              </el-table-column>
              <el-table-column label="订单状态" width="110">
                <template #default="{ row }">
                  <el-tag :type="orderStatusType(row.status)" size="small">{{ orderStatusLabel(row.status) }}</el-tag>
                </template>
              </el-table-column>
              <el-table-column label="应付金额" width="125"><template #default="{ row }">{{ formatMoney(row.payable_amount, row.currency) }}</template></el-table-column>
              <el-table-column label="实付金额" width="125"><template #default="{ row }">{{ formatMoney(row.paid_amount, row.currency) }}</template></el-table-column>
              <el-table-column prop="payment_method" label="支付方式" width="120" />
              <el-table-column label="权益" min-width="135">
                <template #default="{ row }">
                  <div>积分 +{{ formatNumber(row.bonus_points) }}</div>
                  <div class="secondary-text">VIP {{ row.vip_level || 0 }} / {{ row.vip_duration_days || 0 }} 天</div>
                </template>
              </el-table-column>
              <el-table-column label="支付时间" min-width="175"><template #default="{ row }">{{ formatDate(row.paid_at) }}</template></el-table-column>
              <el-table-column label="创建时间" min-width="175"><template #default="{ row }">{{ formatDate(row.created_at) }}</template></el-table-column>
            </el-table>
          </el-tab-pane>
        </el-tabs>
      </div>
    </el-drawer>

    <el-dialog v-model="vipDialogVisible" title="添加 VIP" width="480px">
      <el-form label-width="100px">
<!--        <el-form-item label="VIP 等级" required><el-input-number v-model="vipForm.level" :min="1" :max="999" /></el-form-item>-->
        <el-form-item label="赠送积分">
          <el-input-number v-model="vipForm.vip_points" :min="0" :max="999999999" :precision="0" style="width:100%" />
        </el-form-item>
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
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  bindUserPhone, clearUserDevice, extendUserVIP, getUserCenter, grantUserVIP,
  getAppUserList, setUserBlacklisted, setUserFrozen, terminateUserVIP, transferUserVIP,
  type AppUser, type UserCenterDetail, type UserPointsLedger,
} from '@/api/appUser'
import { useUserStore } from '@/store/user'

const userStore = useUserStore()
const canManage = computed(() => userStore.hasPermission('system:app-user:manage'))
const listLoading = ref(false)
const listData = ref<AppUser[]>([])
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const filters = reactive({ keyword: '', device_country: '', channel_id: '', user_type: undefined as number | undefined, login_type: undefined as number | undefined, status: undefined as number | undefined })
const loadingDetail = ref(false)
const operating = ref(false)
const detail = ref<UserCenterDetail | null>(null)
const user = computed(() => detail.value!.user)
const detailVisible = ref(false)
const activeTab = ref('account')
const vipDialogVisible = ref(false)
const vipForm = reactive({ level: 1, vip_points: 0, started_at: '', expires_at: '' })

async function fetchList() {
  listLoading.value = true
  try {
    const params: Record<string, unknown> = { page: page.value, page_size: pageSize.value }
    for (const [key, value] of Object.entries(filters)) {
      if (value !== '' && value !== undefined && value !== null) params[key] = typeof value === 'string' ? value.trim() : value
    }
    const result: any = await getAppUserList(params)
    listData.value = result.data?.list || []
    total.value = Number(result.data?.total) || 0
  } finally { listLoading.value = false }
}

function handleListSearch() { page.value = 1; fetchList() }
function handlePageSizeChange() { page.value = 1; fetchList() }
function resetListFilters() {
  Object.assign(filters, { keyword: '', device_country: '', channel_id: '', user_type: undefined, login_type: undefined, status: undefined })
  page.value = 1
  fetchList()
}

async function openDetail(row: AppUser) {
  detail.value = null
  activeTab.value = 'account'
  detailVisible.value = true
  await loadDetail(row.id)
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
  vipForm.vip_points = 0
  vipForm.started_at = new Date().toISOString()
  const expires = new Date(); expires.setDate(expires.getDate() + 30)
  vipForm.expires_at = expires.toISOString()
  vipDialogVisible.value = true
}

async function submitVIP() {
  if (!vipForm.expires_at) { ElMessage.warning('请选择 VIP 结束时间'); return }
  await runOperation('VIP 已添加', () => grantUserVIP(user.value.id, {
    level: vipForm.level, vip_points: vipForm.vip_points,
    started_at: vipForm.started_at || null, expires_at: vipForm.expires_at,
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
function subscriptionLabel(value: number) { return value === 2 ? '订阅中' : value === 3 ? '已取消' : value === 4 ? '已过期' : '未订阅' }
function flagActive(value: boolean | number) { return value === true || Number(value) === 1 }
function formatNumber(value: number) { return new Intl.NumberFormat('zh-CN').format(value || 0) }

function ledgerBusinessLabel(ledger: UserPointsLedger) {
  if (ledger.order_id) return `订单 #${ledger.order_id}`
  if (ledger.work_id) return `作品 ${ledger.work_id}`
  if (ledger.business_id) return ledger.business_id
  return ledger.mode_key || '-'
}

type StatusTagType = 'primary' | 'success' | 'info' | 'warning' | 'danger'

function workStatusLabel(status: string) {
  const labels: Record<string, string> = {
    submitting: '提交中', submitted: '已提交', pending: '排队中', running: '生成中',
    downloading: '下载中', success: '成功', failure: '失败',
  }
  return labels[status] || status || '-'
}

function workStatusType(status: string): StatusTagType {
  if (status === 'success') return 'success'
  if (status === 'failure') return 'danger'
  if (status === 'running' || status === 'downloading') return 'warning'
  return 'info'
}

function orderStatusLabel(status: string) {
  const labels: Record<string, string> = {
    pending: '待支付', paid: '已支付', cancelled: '已取消', failed: '失败', refunded: '已退款',
  }
  return labels[status] || status || '-'
}

function orderStatusType(status: string): StatusTagType {
  if (status === 'paid') return 'success'
  if (status === 'pending') return 'warning'
  if (status === 'failed') return 'danger'
  return 'info'
}

function formatDuration(seconds: number) {
  if (!seconds) return '-'
  if (seconds < 60) return `${seconds} 秒`
  const minutes = Math.floor(seconds / 60)
  const remainingSeconds = seconds % 60
  return remainingSeconds ? `${minutes} 分 ${remainingSeconds} 秒` : `${minutes} 分`
}

function formatMoney(value: number, currency: string) {
  try {
    return new Intl.NumberFormat('zh-CN', {
      style: 'currency', currency: currency || 'USD', minimumFractionDigits: 2,
    }).format(value || 0)
  } catch {
    return `${currency || ''} ${Number(value || 0).toFixed(2)}`.trim()
  }
}

function formatDate(value?: string | null) {
  if (!value) return '-'
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString('zh-CN', { hour12: false })
}

onMounted(fetchList)
</script>

<style scoped>
.user-center { min-width: 0; }
.header { display: flex; align-items: center; justify-content: space-between; }
.title { color: #303133; font-size: 18px; font-weight: 600; }
.subtitle { margin-top: 5px; color: #909399; font-size: 12px; }
.list-filters { margin-bottom: 8px; }
.list-filters :deep(.el-input) { width: 190px; }
.primary-text { color: var(--el-text-color-primary); }
.secondary-text { margin-top: 3px; color: var(--el-text-color-secondary); font-size: 12px; }
.status-tag { margin-left: 4px; }
.pagination-wrap { display: flex; justify-content: flex-end; margin-top: 16px; }
.summary { width: 100%; }
.actions { display: flex; flex-wrap: wrap; gap: 10px; margin: 18px 0; }
.actions .el-button { margin-left: 0; }
.detail-tabs { margin-top: 22px; }
.points-summary { display: flex; gap: 10px; }
.table-toolbar { display: flex; align-items: center; justify-content: space-between; gap: 12px; margin-bottom: 12px; }
.result-hint { color: var(--el-text-color-secondary); font-size: 12px; }
.table-hint { margin-bottom: 10px; text-align: right; }
.income { color: #67c23a; font-weight: 600; }
.expense { color: #f56c6c; font-weight: 600; }
@media (max-width: 700px) {
  .summary :deep(.el-descriptions__body) { overflow-x: auto; }
}
</style>
