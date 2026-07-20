<template>
  <div class="page-wrap">
    <el-card shadow="never">
      <template #header>
        <div class="page-header">
          <div>
            <div class="page-title">客户端用户</div>
            <div class="page-subtitle">管理客户端用户资料、应用设备、订阅资产、付费及行为指标</div>
          </div>
          <el-button v-if="canAdd" type="primary" @click="openCreate">
            <el-icon><Plus /></el-icon>新增用户
          </el-button>
        </div>
      </template>

      <div class="filters">
        <el-input v-model="query.keyword" clearable placeholder="昵称、账号、设备号、邮箱或应用" @keyup.enter="handleSearch">
          <template #prefix><el-icon><Search /></el-icon></template>
        </el-input>
        <el-input v-model="query.app_name" clearable placeholder="应用名称" @keyup.enter="handleSearch" />
        <el-input v-model="query.device_country" clearable placeholder="设备国家" @keyup.enter="handleSearch" />
        <el-input v-model="query.channel_id" clearable placeholder="渠道 ID" @keyup.enter="handleSearch" />
        <el-select v-model="query.login_type" clearable placeholder="登录方式">
          <el-option v-for="item in loginTypeOptions" :key="item.value" :label="item.label" :value="String(item.value)" />
        </el-select>
        <el-select v-model="query.user_type" clearable placeholder="用户类型">
          <el-option v-for="item in userTypeOptions" :key="item.value" :label="item.label" :value="String(item.value)" />
        </el-select>
        <el-select v-model="query.subscription_status" clearable placeholder="订阅状态">
          <el-option v-for="item in subscriptionOptions" :key="item.value" :label="item.label" :value="String(item.value)" />
        </el-select>
        <el-select v-model="query.status" clearable placeholder="账号状态">
          <el-option label="正常" value="1" />
          <el-option label="禁用" value="0" />
        </el-select>
        <el-button type="primary" plain @click="handleSearch">查询</el-button>
        <el-button @click="handleReset">重置</el-button>
      </div>

      <el-table v-loading="loading" :data="tableData" row-key="id" stripe>
        <el-table-column prop="id" label="ID" width="72" />
        <el-table-column label="用户" min-width="220">
          <template #default="{ row }">
            <div class="primary-text">{{ row.username }}</div>
            <div class="secondary-text mono">{{ row.login_account || row.google_email || row.appid_email || row.imei }}</div>
            <div class="secondary-text">设备号：{{ row.imei }}</div>
          </template>
        </el-table-column>
        <el-table-column label="应用与设备" min-width="220">
          <template #default="{ row }">
            <div>{{ row.app_name || '未记录应用' }} <span class="secondary-text">{{ row.app_version }}</span></div>
            <div class="secondary-text">{{ row.device_country || '-' }} · {{ row.phone_model || '未知设备' }}</div>
            <div class="secondary-text">渠道：{{ row.channel_id || '-' }}</div>
          </template>
        </el-table-column>
        <el-table-column label="账号属性" width="160">
          <template #default="{ row }">
            <div class="tag-row">
              <el-tag size="small" effect="plain">{{ optionLabel(loginTypeOptions, row.login_type) }}</el-tag>
              <el-tag size="small" :type="row.user_type === 2 ? 'warning' : 'info'">{{ optionLabel(userTypeOptions, row.user_type) }}</el-tag>
            </div>
            <el-tag class="subscription-tag" size="small" :type="subscriptionTagType(row.subscription_status)">
              {{ optionLabel(subscriptionOptions, row.subscription_status) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="活跃与资产" width="170">
          <template #default="{ row }">
            <div class="metric-line"><span>活跃天数</span><strong>{{ row.active_days }}</strong></div>
            <div class="metric-line"><span>日均使用</span><strong>{{ formatDuration(row.avg_daily_usage_seconds) }}</strong></div>
            <div class="metric-line"><span>积分</span><strong>{{ formatNumber(row.points_balance) }}</strong></div>
          </template>
        </el-table-column>
        <el-table-column label="付费" width="180">
          <template #default="{ row }">
            <div class="metric-line"><span>实付金额</span><strong>¥{{ formatMoney(row.actual_amount_money) }}</strong></div>
            <div class="metric-line"><span>付费次数</span><strong>{{ formatNumber(row.payment_count) }}</strong></div>
            <div class="metric-line"><span>订单数</span><strong>{{ formatNumber(row.order_count) }}</strong></div>
          </template>
        </el-table-column>
        <el-table-column label="行为指标" width="190">
          <template #default="{ row }">
            <div class="behavior-tags">
              <el-tag size="small" :type="row.activated ? 'success' : 'info'">激活{{ row.activated ? '达标' : '未达标' }}</el-tag>
              <el-tag size="small" :type="row.registered ? 'success' : 'info'">注册{{ row.registered ? '达标' : '未达标' }}</el-tag>
              <el-tag size="small" :type="row.payment_met ? 'success' : 'info'">付费{{ row.payment_met ? '达标' : '未达标' }}</el-tag>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="118" align="center">
          <template #default="{ row }">
            <el-switch
              v-if="canEdit"
              v-model="row.status"
              :active-value="1"
              :inactive-value="0"
              inline-prompt
              active-text="正常"
              inactive-text="禁用"
              :loading="updatingIds.includes(row.id)"
              @change="handleStatusChange(row)"
            />
            <el-tag v-else :type="row.status === 1 ? 'success' : 'danger'">{{ row.status === 1 ? '正常' : '禁用' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="最近活动" width="180">
          <template #default="{ row }">
            <div>{{ formatDate(row.last_opened_at || row.last_login_at) }}</div>
            <div class="secondary-text">{{ row.last_login_ip || '-' }}</div>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="170" fixed="right" align="center">
          <template #default="{ row }">
            <el-button link type="primary" @click="openDetail(row.id)">详情</el-button>
            <el-button v-if="canEdit" link type="primary" @click="openEdit(row)">编辑</el-button>
            <el-popconfirm v-if="canDelete" title="确认删除该客户端用户？" @confirm="handleDelete(row.id)">
              <template #reference><el-button link type="danger">删除</el-button></template>
            </el-popconfirm>
          </template>
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
          @current-change="fetchData"
        />
      </div>
    </el-card>

    <el-dialog v-model="dialogVisible" :title="form.id ? '编辑客户端用户' : '新增客户端用户'" width="920px" destroy-on-close>
      <el-form ref="formRef" :model="form" :rules="rules" label-width="112px">
        <el-tabs v-model="formTab">
          <el-tab-pane label="账号资料" name="account">
            <div class="form-grid">
              <el-form-item label="用户昵称" prop="username"><el-input v-model="form.username" maxlength="128" /></el-form-item>
              <el-form-item label="设备编号" prop="imei"><el-input v-model="form.imei" maxlength="128" /></el-form-item>
              <el-form-item label="登录方式"><el-select v-model="form.login_type" style="width: 100%"><el-option v-for="item in loginTypeOptions" :key="item.value" :label="item.label" :value="item.value" /></el-select></el-form-item>
              <el-form-item label="登录账号"><el-input v-model="form.login_account" maxlength="255" /></el-form-item>
              <el-form-item label="Google 邮箱" prop="google_email"><el-input v-model="form.google_email" maxlength="50" /></el-form-item>
              <el-form-item label="Google 唯一码"><el-input v-model="form.google_third_code" maxlength="50" /></el-form-item>
              <el-form-item label="Apple 邮箱" prop="appid_email"><el-input v-model="form.appid_email" maxlength="50" /></el-form-item>
              <el-form-item label="Apple 唯一码"><el-input v-model="form.appid_third_code" maxlength="50" /></el-form-item>
              <el-form-item label="用户类型"><el-select v-model="form.user_type" style="width: 100%"><el-option v-for="item in userTypeOptions" :key="item.value" :label="item.label" :value="item.value" /></el-select></el-form-item>
              <el-form-item label="账号状态"><el-radio-group v-model="form.status"><el-radio :value="1">正常</el-radio><el-radio :value="0">禁用</el-radio></el-radio-group></el-form-item>
              <el-form-item label="重注册来源ID"><el-input-number v-model="form.re_registered_from_id" :min="0" :max="999999999999" controls-position="right" /></el-form-item>
            </div>
          </el-tab-pane>

          <el-tab-pane label="应用与设备" name="device">
            <div class="form-grid">
              <el-form-item label="应用名称"><el-input v-model="form.app_name" maxlength="255" /></el-form-item>
              <el-form-item label="应用版本"><el-input v-model="form.app_version" maxlength="32" /></el-form-item>
              <el-form-item label="设备国家"><el-input v-model="form.device_country" maxlength="64" /></el-form-item>
              <el-form-item label="渠道 ID"><el-input v-model="form.channel_id" maxlength="64" /></el-form-item>
              <el-form-item label="手机型号"><el-input v-model="form.phone_model" maxlength="128" /></el-form-item>
              <el-form-item label="首次打开"><el-date-picker v-model="form.first_opened_at" type="datetime" value-format="YYYY-MM-DDTHH:mm:ssZ" clearable style="width: 100%" /></el-form-item>
              <el-form-item label="上次打开"><el-date-picker v-model="form.last_opened_at" type="datetime" value-format="YYYY-MM-DDTHH:mm:ssZ" clearable style="width: 100%" /></el-form-item>
              <el-form-item label="最近登录"><el-date-picker v-model="form.last_login_at" type="datetime" value-format="YYYY-MM-DDTHH:mm:ssZ" clearable style="width: 100%" /></el-form-item>
              <el-form-item label="最近登录 IP"><el-input v-model="form.last_login_ip" maxlength="64" /></el-form-item>
            </div>
          </el-tab-pane>

          <el-tab-pane label="订阅与活跃" name="subscription">
            <div class="form-grid">
              <el-form-item label="订阅状态"><el-select v-model="form.subscription_status" style="width: 100%"><el-option v-for="item in subscriptionOptions" :key="item.value" :label="item.label" :value="item.value" /></el-select></el-form-item>
              <el-form-item label="VIP 到期时间"><el-date-picker v-model="form.vip_expires_at" type="datetime" value-format="YYYY-MM-DDTHH:mm:ssZ" clearable style="width: 100%" /></el-form-item>
              <el-form-item label="积分余额"><el-input-number v-model="form.points_balance" :min="0" :max="999999999999" controls-position="right" /></el-form-item>
              <el-form-item label="积分成本"><el-input-number v-model="form.points_money" :min="0" :precision="2" :step="0.01" controls-position="right" /></el-form-item>
              <el-form-item label="活跃天数"><el-input-number v-model="form.active_days" :min="0" :max="999999" controls-position="right" /></el-form-item>
              <el-form-item label="日均使用秒数"><el-input-number v-model="form.avg_daily_usage_seconds" :min="0" :max="999999999" controls-position="right" /></el-form-item>
            </div>
          </el-tab-pane>

          <el-tab-pane label="付费数据" name="payment">
            <div class="form-grid">
              <el-form-item label="订单创建次数"><el-input-number v-model="form.order_count" :min="0" controls-position="right" /></el-form-item>
              <el-form-item label="付费次数"><el-input-number v-model="form.payment_count" :min="0" controls-position="right" /></el-form-item>
              <el-form-item label="订阅付费次数"><el-input-number v-model="form.subscription_payment_count" :min="0" controls-position="right" /></el-form-item>
              <el-form-item label="单次付费次数"><el-input-number v-model="form.one_time_payment_count" :min="0" controls-position="right" /></el-form-item>
              <el-form-item label="累计订单金额"><el-input-number v-model="form.order_amount_money" :min="0" :precision="2" :step="0.01" controls-position="right" /></el-form-item>
              <el-form-item label="累计实付金额"><el-input-number v-model="form.actual_amount_money" :min="0" :precision="2" :step="0.01" controls-position="right" /></el-form-item>
              <el-form-item label="累计退款金额"><el-input-number v-model="form.refund_amount_money" :min="0" :precision="2" :step="0.01" controls-position="right" /></el-form-item>
              <el-form-item label="累计 AI 成本"><el-input-number v-model="form.ai_cots_money" :min="0" :precision="2" :step="0.01" controls-position="right" /></el-form-item>
              <el-form-item label="首单创建时间"><el-date-picker v-model="form.first_order_created_at" type="datetime" value-format="YYYY-MM-DDTHH:mm:ssZ" clearable style="width: 100%" /></el-form-item>
              <el-form-item label="首次付费时间"><el-date-picker v-model="form.first_paid_at" type="datetime" value-format="YYYY-MM-DDTHH:mm:ssZ" clearable style="width: 100%" /></el-form-item>
              <el-form-item label="最后付费时间"><el-date-picker v-model="form.last_paid_at" type="datetime" value-format="YYYY-MM-DDTHH:mm:ssZ" clearable style="width: 100%" /></el-form-item>
            </div>
          </el-tab-pane>

          <el-tab-pane label="行为指标" name="behavior">
            <div class="form-grid behavior-form">
              <el-form-item label="激活达标"><el-switch v-model="form.activated" :active-value="1" :inactive-value="0" /></el-form-item>
              <el-form-item label="关键行为达标"><el-switch v-model="form.key_behavior_met" :active-value="1" :inactive-value="0" /></el-form-item>
              <el-form-item label="付费达标"><el-switch v-model="form.payment_met" /></el-form-item>
              <el-form-item label="首次付费达标"><el-switch v-model="form.first_payment_met" /></el-form-item>
              <el-form-item label="注册达标"><el-switch v-model="form.registered" /></el-form-item>
              <el-form-item label="归因点击时间"><el-date-picker v-model="form.attribution_clicked_at" type="datetime" value-format="YYYY-MM-DDTHH:mm:ssZ" clearable style="width: 100%" /></el-form-item>
            </div>
          </el-tab-pane>
        </el-tabs>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="handleSubmit">保存</el-button>
      </template>
    </el-dialog>

    <el-drawer v-model="detailVisible" title="客户端用户详情" size="760px" destroy-on-close>
      <div v-loading="detailLoading" class="detail-body">
        <template v-if="detailUser">
          <div class="detail-title">账号资料</div>
          <el-descriptions :column="2" border>
            <el-descriptions-item label="用户 ID">{{ detailUser.id }}</el-descriptions-item>
            <el-descriptions-item label="昵称">{{ detailUser.username }}</el-descriptions-item>
            <el-descriptions-item label="设备编号">{{ detailUser.imei }}</el-descriptions-item>
            <el-descriptions-item label="重注册来源 ID">{{ detailUser.re_registered_from_id || '-' }}</el-descriptions-item>
            <el-descriptions-item label="登录方式">{{ optionLabel(loginTypeOptions, detailUser.login_type) }}</el-descriptions-item>
            <el-descriptions-item label="登录账号">{{ detailUser.login_account || '-' }}</el-descriptions-item>
            <el-descriptions-item label="Google 邮箱">{{ detailUser.google_email || '-' }}</el-descriptions-item>
            <el-descriptions-item label="Google 唯一码">{{ detailUser.google_third_code || '-' }}</el-descriptions-item>
            <el-descriptions-item label="Apple 邮箱">{{ detailUser.appid_email || '-' }}</el-descriptions-item>
            <el-descriptions-item label="Apple 唯一码">{{ detailUser.appid_third_code || '-' }}</el-descriptions-item>
            <el-descriptions-item label="用户类型">{{ optionLabel(userTypeOptions, detailUser.user_type) }}</el-descriptions-item>
            <el-descriptions-item label="状态">{{ detailUser.status === 1 ? '正常' : '禁用' }}</el-descriptions-item>
          </el-descriptions>

          <div class="detail-title">应用、订阅与资产</div>
          <el-descriptions :column="2" border>
            <el-descriptions-item label="应用">{{ detailUser.app_name || '-' }} {{ detailUser.app_version }}</el-descriptions-item>
            <el-descriptions-item label="渠道">{{ detailUser.channel_id || '-' }}</el-descriptions-item>
            <el-descriptions-item label="设备国家">{{ detailUser.device_country || '-' }}</el-descriptions-item>
            <el-descriptions-item label="设备型号">{{ detailUser.phone_model || '-' }}</el-descriptions-item>
            <el-descriptions-item label="订阅状态">{{ optionLabel(subscriptionOptions, detailUser.subscription_status) }}</el-descriptions-item>
            <el-descriptions-item label="VIP 到期">{{ formatDate(detailUser.vip_expires_at) }}</el-descriptions-item>
            <el-descriptions-item label="积分余额">{{ formatNumber(detailUser.points_balance) }}</el-descriptions-item>
            <el-descriptions-item label="活跃天数">{{ detailUser.active_days }}</el-descriptions-item>
            <el-descriptions-item label="日均使用">{{ formatDuration(detailUser.avg_daily_usage_seconds) }}</el-descriptions-item>
          </el-descriptions>

          <div class="detail-title">付费与成本</div>
          <el-descriptions :column="2" border>
            <el-descriptions-item label="订单/付费次数">{{ detailUser.order_count }} / {{ detailUser.payment_count }}</el-descriptions-item>
            <el-descriptions-item label="订阅/单次付费">{{ detailUser.subscription_payment_count }} / {{ detailUser.one_time_payment_count }}</el-descriptions-item>
            <el-descriptions-item label="订单金额">¥{{ formatMoney(detailUser.order_amount_money) }}</el-descriptions-item>
            <el-descriptions-item label="实付金额">¥{{ formatMoney(detailUser.actual_amount_money) }}</el-descriptions-item>
            <el-descriptions-item label="退款金额">¥{{ formatMoney(detailUser.refund_amount_money) }}</el-descriptions-item>
            <el-descriptions-item label="AI 成本">¥{{ formatMoney(detailUser.ai_cots_money) }}</el-descriptions-item>
            <el-descriptions-item label="首次付费">{{ formatDate(detailUser.first_paid_at) }}</el-descriptions-item>
            <el-descriptions-item label="最后付费">{{ formatDate(detailUser.last_paid_at) }}</el-descriptions-item>
          </el-descriptions>

          <div class="detail-title">时间与登录</div>
          <el-descriptions :column="2" border>
            <el-descriptions-item label="首次打开">{{ formatDate(detailUser.first_opened_at) }}</el-descriptions-item>
            <el-descriptions-item label="上次打开">{{ formatDate(detailUser.last_opened_at) }}</el-descriptions-item>
            <el-descriptions-item label="最近登录">{{ formatDate(detailUser.last_login_at) }}</el-descriptions-item>
            <el-descriptions-item label="最近登录 IP">{{ detailUser.last_login_ip || '-' }}</el-descriptions-item>
            <el-descriptions-item label="创建时间">{{ formatDate(detailUser.created_at) }}</el-descriptions-item>
            <el-descriptions-item label="更新时间">{{ formatDate(detailUser.updated_at) }}</el-descriptions-item>
          </el-descriptions>
        </template>
      </div>
    </el-drawer>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, type FormInstance, type FormRules } from 'element-plus'
import {
  createAppUser,
  deleteAppUser,
  getAppUser,
  getAppUserList,
  updateAppUser,
  type AppUser,
  type AppUserPayload,
} from '@/api/appUser'
import { useUserStore } from '@/store/user'

const loginTypeOptions = [{ value: 1, label: '游客' }, { value: 2, label: 'Google' }, { value: 3, label: 'App ID' }]
const userTypeOptions = [{ value: 1, label: '免费用户' }, { value: 2, label: '付费用户' }]
const subscriptionOptions = [{ value: 1, label: '未订阅' }, { value: 2, label: '订阅中' }, { value: 3, label: '已取消' }]

const userStore = useUserStore()
const canAdd = computed(() => userStore.hasPermission('system:app-user:add'))
const canEdit = computed(() => userStore.hasPermission('system:app-user:edit'))
const canDelete = computed(() => userStore.hasPermission('system:app-user:delete'))
const loading = ref(false)
const submitting = ref(false)
const dialogVisible = ref(false)
const detailVisible = ref(false)
const detailLoading = ref(false)
const detailUser = ref<AppUser | null>(null)
const updatingIds = ref<number[]>([])
const formRef = ref<FormInstance>()
const formTab = ref('account')
const tableData = ref<AppUser[]>([])
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const query = reactive({ keyword: '', app_name: '', device_country: '', channel_id: '', login_type: '', user_type: '', subscription_status: '', status: '' })

const defaultForm: AppUserPayload & { id: number } = {
  id: 0, imei: '', username: '', device_country: '', channel_id: '', app_version: '', app_name: '',
  first_opened_at: null, last_opened_at: null, login_type: 1, user_type: 1, active_days: 0,
  avg_daily_usage_seconds: 0, vip_expires_at: null, points_balance: 0, subscription_status: 1,
  first_order_created_at: null, first_paid_at: null, order_count: 0, payment_count: 0,
  subscription_payment_count: 0, one_time_payment_count: 0, order_amount_money: 0, actual_amount_money: 0,
  last_paid_at: null, refund_amount_money: 0, points_money: 0, ai_cots_money: 0, activated: 0,
  key_behavior_met: 0, payment_met: false, first_payment_met: false, registered: false,
  attribution_clicked_at: null, phone_model: '', re_registered_from_id: null,
  appid_email: '', appid_third_code: '', google_email: '', google_third_code: '',
  status: 1, last_login_at: null, last_login_ip: '', login_account: '',
}
const form = reactive({ ...defaultForm })
const rules: FormRules = {
  username: [{ required: true, message: '请输入用户昵称', trigger: 'blur' }],
  imei: [{ required: true, message: '请输入设备编号', trigger: 'blur' }],
  google_email: [{ type: 'email', message: 'Google 邮箱格式不正确', trigger: 'blur' }],
  appid_email: [{ type: 'email', message: 'Apple 邮箱格式不正确', trigger: 'blur' }],
}

async function fetchData() {
  loading.value = true
  try {
    const params: Record<string, unknown> = { page: page.value, page_size: pageSize.value }
    for (const [key, value] of Object.entries(query)) if (value !== '') params[key] = typeof value === 'string' ? value.trim() : value
    const res: any = await getAppUserList(params)
    tableData.value = res.data.list || []
    total.value = res.data.total || 0
  } finally { loading.value = false }
}

function handleSearch() { page.value = 1; fetchData() }
function handleReset() {
  Object.assign(query, { keyword: '', app_name: '', device_country: '', channel_id: '', login_type: '', user_type: '', subscription_status: '', status: '' })
  page.value = 1
  fetchData()
}
function handlePageSizeChange() { page.value = 1; fetchData() }

function openCreate() {
  Object.assign(form, defaultForm)
  formTab.value = 'account'
  dialogVisible.value = true
}

function openEdit(row: AppUser) {
  Object.assign(form, defaultForm, row, {
    appid_email: row.appid_email || '', appid_third_code: row.appid_third_code || '',
    google_email: row.google_email || '', google_third_code: row.google_third_code || '',
  })
  formTab.value = 'account'
  dialogVisible.value = true
}

async function openDetail(id: number) {
  detailVisible.value = true
  detailLoading.value = true
  try {
    const res: any = await getAppUser(id)
    detailUser.value = res.data
  } finally { detailLoading.value = false }
}

async function handleSubmit() {
  await formRef.value?.validate()
  submitting.value = true
  try {
    const { id, ...values } = form
    const payload = {
      ...values,
      username: values.username.trim(), imei: values.imei.trim(),
      appid_email: values.appid_email?.trim() || '', appid_third_code: values.appid_third_code?.trim() || '',
      google_email: values.google_email?.trim() || '', google_third_code: values.google_third_code?.trim() || '',
    }
    if (id) await updateAppUser(id, payload)
    else await createAppUser(payload)
    ElMessage.success('客户端用户已保存')
    dialogVisible.value = false
    await fetchData()
  } finally { submitting.value = false }
}

async function handleStatusChange(row: AppUser) {
  updatingIds.value.push(row.id)
  try {
    await updateAppUser(row.id, { status: row.status })
    ElMessage.success(`用户已${row.status === 1 ? '启用' : '禁用'}`)
  } catch {
    row.status = row.status === 1 ? 0 : 1
  } finally { updatingIds.value = updatingIds.value.filter((id) => id !== row.id) }
}

async function handleDelete(id: number) {
  await deleteAppUser(id)
  ElMessage.success('客户端用户已删除')
  if (tableData.value.length === 1 && page.value > 1) page.value--
  await fetchData()
}

function optionLabel(options: Array<{ value: number; label: string }>, value: number) {
  return options.find((item) => item.value === value)?.label || String(value || '-')
}
function subscriptionTagType(value: number) { return value === 2 ? 'success' : value === 3 ? 'danger' : 'info' }
function formatNumber(value: number) { return new Intl.NumberFormat('zh-CN').format(value || 0) }
function formatMoney(value: number) { return Number(value || 0).toFixed(2) }
function formatDuration(seconds: number) {
  if (!seconds) return '0秒'
  if (seconds < 60) return `${seconds}秒`
  const hours = Math.floor(seconds / 3600)
  const minutes = Math.floor((seconds % 3600) / 60)
  return hours ? `${hours}小时${minutes}分` : `${minutes}分钟`
}
function formatDate(value?: string | null) {
  if (!value) return '-'
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString('zh-CN', { hour12: false })
}

onMounted(fetchData)
</script>

<style scoped>
.page-wrap { min-width: 0; }
.page-header { display: flex; align-items: center; justify-content: space-between; gap: 16px; }
.page-title { color: #303133; font-size: 17px; font-weight: 600; }
.page-subtitle { margin-top: 4px; color: #909399; font-size: 12px; }
.filters { display: grid; grid-template-columns: minmax(230px, 1.5fr) repeat(3, minmax(130px, 1fr)) repeat(4, 125px) auto auto; gap: 10px; margin-bottom: 16px; }
.primary-text { color: #303133; font-weight: 600; }
.secondary-text { margin-top: 3px; color: #909399; font-size: 12px; }
.mono { font-family: ui-monospace, SFMono-Regular, Menlo, monospace; }
.tag-row, .behavior-tags { display: flex; flex-wrap: wrap; gap: 5px; }
.subscription-tag { margin-top: 6px; }
.metric-line { display: flex; align-items: center; justify-content: space-between; gap: 10px; line-height: 1.75; }
.metric-line span { color: #909399; font-size: 12px; }
.metric-line strong { color: #303133; font-size: 13px; font-variant-numeric: tabular-nums; }
.pagination-wrap { display: flex; justify-content: flex-end; margin-top: 16px; overflow-x: auto; }
.form-grid { display: grid; grid-template-columns: 1fr 1fr; column-gap: 18px; padding-top: 8px; }
.form-grid :deep(.el-input-number) { width: 100%; }
.behavior-form { align-items: center; }
.detail-body { min-height: 240px; padding: 0 4px 30px; }
.detail-title { margin: 22px 0 10px; color: #303133; font-weight: 600; }
.detail-title:first-child { margin-top: 0; }
@media (max-width: 1200px) { .filters { grid-template-columns: repeat(4, minmax(130px, 1fr)); } }
@media (max-width: 700px) {
  .page-header { align-items: stretch; flex-direction: column; }
  .filters, .form-grid { grid-template-columns: 1fr; }
  .page-wrap :deep(.el-card__header), .page-wrap :deep(.el-card__body) { padding: 14px; }
}
</style>
