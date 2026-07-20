<template>
  <div class="page-wrap">
    <el-card shadow="never">
      <template #header>
        <div class="page-header">
          <div>
            <div class="page-title">用户归因</div>
            <div class="page-subtitle">查看渠道归因、设备标识及用户行为事件的回传与扣除记录</div>
          </div>
          <el-button v-if="canSync" :icon="Refresh" :loading="syncing" @click="handleSync">同步用户</el-button>
        </div>
      </template>

      <div class="filters">
        <el-input v-model="query.keyword" clearable placeholder="用户、设备编号、OAID、IMEI、Android ID 或 IP" @keyup.enter="handleSearch">
          <template #prefix><el-icon><Search /></el-icon></template>
        </el-input>
        <el-select v-model="query.channel_code" clearable filterable placeholder="渠道">
          <el-option
            v-for="item in channelOptions"
            :key="item.channel_id"
            :label="`${item.channel_name} · ${item.channel_code}`"
            :value="item.channel_code"
          />
        </el-select>
        <el-select v-model="query.event" clearable placeholder="事件">
          <el-option v-for="item in events" :key="item.key" :label="item.label" :value="item.key" />
        </el-select>
        <el-select v-model="query.reached" clearable :disabled="!query.event" placeholder="达标状态">
          <el-option label="已达标" value="true" />
          <el-option label="未达标" value="false" />
        </el-select>
        <el-date-picker v-model="dateRange" type="daterange" value-format="YYYY-MM-DD" start-placeholder="开始日期" end-placeholder="结束日期" />
        <el-button type="primary" plain @click="handleSearch">查询</el-button>
        <el-button @click="handleReset">重置</el-button>
      </div>

      <el-table v-loading="loading" :data="tableData" row-key="id" stripe>
        <el-table-column prop="id" label="归因 ID" width="88" fixed="left" />
        <el-table-column label="用户 / 渠道" min-width="190" fixed="left">
          <template #default="{ row }">
            <div class="primary-text">{{ row.user?.username || `用户 #${row.user_id}` }}</div>
            <div class="secondary-text">{{ row.user?.imei || '-' }}</div>
            <el-tag size="small" effect="plain">{{ channelLabel(row) }}</el-tag>
          </template>
        </el-table-column>

        <el-table-column v-for="event in events" :key="event.key" :label="event.label" width="158" align="center">
          <template #default="{ row }">
            <div class="event-cell">
              <el-tag size="small" :type="isReached(row, event.key) ? 'success' : 'info'">
                {{ isReached(row, event.key) ? '已达标' : '未达标' }}
              </el-tag>
              <div class="event-counts">
                <span>回传 <strong>{{ eventCount(row, event.key, 'callback') }}</strong></span>
                <span>扣除 <strong>{{ eventCount(row, event.key, 'deduct') }}</strong></span>
              </div>
              <div v-if="canEdit" class="event-actions">
                <el-button
                  link
                  type="primary"
                  :loading="isOperating(row.id, event.key, 'callback')"
                  :disabled="!isReached(row, event.key)"
                  @click="handleEvent(row, event.key, 'callback')"
                >回传</el-button>
                <el-button
                  link
                  type="danger"
                  :loading="isOperating(row.id, event.key, 'deduct')"
                  :disabled="eventCount(row, event.key, 'deduct') >= eventCount(row, event.key, 'callback')"
                  @click="handleEvent(row, event.key, 'deduct')"
                >扣除</el-button>
              </div>
            </div>
          </template>
        </el-table-column>

        <el-table-column label="设备标识" min-width="250">
          <template #default="{ row }">
            <div class="identifier"><span>OAID</span><code>{{ row.oaid || '-' }}</code></div>
            <div class="identifier"><span>IMEI</span><code>{{ row.imei || '-' }}</code></div>
            <div class="identifier"><span>Android</span><code>{{ row.android_id || '-' }}</code></div>
          </template>
        </el-table-column>
        <el-table-column label="网络" min-width="210">
          <template #default="{ row }">
            <div>{{ row.ip || row.user?.last_login_ip || '-' }}</div>
            <el-tooltip v-if="row.user_agent" :content="row.user_agent" placement="top">
              <div class="ua-text">{{ row.user_agent }}</div>
            </el-tooltip>
            <span v-else class="secondary-text">无 UA</span>
          </template>
        </el-table-column>
        <el-table-column label="归因时间" width="172">
          <template #default="{ row }">
            <div>{{ formatDate(row.attributed_at || row.created_at) }}</div>
            <div v-if="row.last_operated_at" class="secondary-text">操作 {{ formatDate(row.last_operated_at) }}</div>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="82" fixed="right" align="center">
          <template #default="{ row }">
            <el-button v-if="canEdit" link type="primary" @click="openEdit(row)">编辑</el-button>
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
          @size-change="fetchData"
          @current-change="fetchData"
        />
      </div>
    </el-card>

    <el-dialog v-model="dialogVisible" title="编辑归因信息" width="760px" destroy-on-close>
      <el-form ref="formRef" :model="form" label-width="104px">
        <div class="form-grid">
          <el-form-item label="渠道">
            <el-select v-model="form.channel_code" clearable filterable style="width: 100%">
              <el-option
                v-for="item in channelOptions"
                :key="item.channel_id"
                :label="`${item.channel_name} · ${item.channel_code}`"
                :value="item.channel_code"
              />
            </el-select>
          </el-form-item>
          <el-form-item label="归因时间">
            <el-date-picker v-model="form.attributed_at" type="datetime" value-format="YYYY-MM-DDTHH:mm:ssZ" clearable style="width: 100%" />
          </el-form-item>
          <el-form-item label="OAID"><el-input v-model="form.oaid" maxlength="128" /></el-form-item>
          <el-form-item label="IMEI"><el-input v-model="form.imei" maxlength="128" /></el-form-item>
          <el-form-item label="Android ID"><el-input v-model="form.android_id" maxlength="128" /></el-form-item>
          <el-form-item label="IP"><el-input v-model="form.ip" maxlength="64" /></el-form-item>
          <el-form-item label="UA" class="full-width"><el-input v-model="form.user_agent" type="textarea" :rows="3" maxlength="1024" show-word-limit /></el-form-item>
          <el-form-item label="备注" class="full-width"><el-input v-model="form.remark" maxlength="255" /></el-form-item>
        </div>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="handleSubmit">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { Refresh, Search } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  getAttributionList,
  recordAttributionEvent,
  syncAttributionUsers,
  updateAttribution,
  type AttributionAction,
  type AttributionEvent,
  type AttributionRecord,
} from '@/api/attribution'
import { getChannelOptions, type Channel } from '@/api/channel'
import { useUserStore } from '@/store/user'

const userStore = useUserStore()
const canEdit = computed(() => userStore.hasPermission('attribution:edit'))
const canSync = computed(() => userStore.hasPermission('attribution:sync'))
const events: { key: AttributionEvent; label: string }[] = [
  { key: 'activation', label: '激活' },
  { key: 'key_behavior', label: '关键行为' },
  { key: 'payment', label: '付费' },
  { key: 'first_payment', label: '首次付费' },
  { key: 'registration', label: '注册' },
]

const query = reactive({ keyword: '', channel_code: '', event: '', reached: '' })
const dateRange = ref<string[]>([])
const tableData = ref<AttributionRecord[]>([])
const channelOptions = ref<Channel[]>([])
const loading = ref(false)
const syncing = ref(false)
const submitting = ref(false)
const operatingKey = ref('')
const page = ref(1)
const pageSize = ref(10)
const total = ref(0)

const dialogVisible = ref(false)
const form = reactive({
  id: 0,
  channel_code: '',
  oaid: '',
  imei: '',
  android_id: '',
  ip: '',
  user_agent: '',
  attributed_at: '',
  remark: '',
})

async function fetchOptions() {
  const response: any = await getChannelOptions()
  channelOptions.value = response.data || []
}

async function fetchData() {
  loading.value = true
  try {
    const params: Record<string, unknown> = { page: page.value, page_size: pageSize.value }
    for (const [key, value] of Object.entries(query)) if (value !== '') params[key] = value
    if (dateRange.value?.length === 2) {
      params.started_at = dateRange.value[0]
      params.ended_at = dateRange.value[1]
    }
    const response: any = await getAttributionList(params)
    tableData.value = response.data.list || []
    total.value = response.data.total || 0
  } finally {
    loading.value = false
  }
}

function handleSearch() {
  page.value = 1
  fetchData()
}

function handleReset() {
  Object.assign(query, { keyword: '', channel_code: '', event: '', reached: '' })
  dateRange.value = []
  handleSearch()
}

function channelLabel(row: AttributionRecord) {
  return row.channel ? `${row.channel.channel_name} · ${row.channel.channel_code}` : (row.channel_code || row.user?.channel_id || '未归因渠道')
}

function isReached(row: AttributionRecord, event: AttributionEvent) {
  const user = row.user
  if (!user) return false
  if (event === 'activation') return Boolean(user.activated)
  if (event === 'key_behavior') return Boolean(user.key_behavior_met)
  if (event === 'payment') return Boolean(user.payment_met)
  if (event === 'first_payment') return Boolean(user.first_payment_met)
  return Boolean(user.registered)
}

function eventCount(row: AttributionRecord, event: AttributionEvent, action: AttributionAction) {
  return Number((row as any)[`${event}_${action}_count`] || 0)
}

function isOperating(id: number, event: AttributionEvent, action: AttributionAction) {
  return operatingKey.value === `${id}:${event}:${action}`
}

async function handleEvent(row: AttributionRecord, event: AttributionEvent, action: AttributionAction) {
  const eventLabel = events.find((item) => item.key === event)?.label || event
  if (action === 'deduct') {
    await ElMessageBox.confirm(`确认扣除一次“${eventLabel}”回传记录？`, '确认扣除', { type: 'warning' })
  }
  operatingKey.value = `${row.id}:${event}:${action}`
  try {
    const response: any = await recordAttributionEvent(row.id, event, action)
    const index = tableData.value.findIndex((item) => item.id === row.id)
    if (index >= 0) tableData.value[index] = response.data
    ElMessage.success(action === 'callback' ? '已记录回传' : '已记录扣除')
  } finally {
    operatingKey.value = ''
  }
}

async function handleSync() {
  syncing.value = true
  try {
    const response: any = await syncAttributionUsers()
    ElMessage.success(`同步完成，新增 ${response.data.created || 0} 条`)
    fetchData()
  } finally {
    syncing.value = false
  }
}

function openEdit(row: AttributionRecord) {
  Object.assign(form, {
    id: row.id,
    channel_code: row.channel_code || row.user?.channel_id || '',
    oaid: row.oaid || '',
    imei: row.imei || '',
    android_id: row.android_id || '',
    ip: row.ip || '',
    user_agent: row.user_agent || '',
    attributed_at: row.attributed_at || '',
    remark: row.remark || '',
  })
  dialogVisible.value = true
}

async function handleSubmit() {
  submitting.value = true
  try {
    const { id, ...payload } = form
    await updateAttribution(id, payload)
    ElMessage.success('保存成功')
    dialogVisible.value = false
    fetchData()
  } finally {
    submitting.value = false
  }
}

function formatDate(value?: string) {
  if (!value) return '-'
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString('zh-CN', { hour12: false })
}

onMounted(() => {
  fetchOptions()
  fetchData()
})
</script>

<style scoped>
.page-wrap { min-width: 0; }
.page-header { display: flex; align-items: center; justify-content: space-between; gap: 16px; }
.page-title { color: #303133; font-size: 18px; font-weight: 600; }
.page-subtitle { margin-top: 4px; color: #909399; font-size: 13px; }
.filters { display: grid; grid-template-columns: minmax(280px, 1.6fr) 190px 150px 130px 250px auto auto; gap: 10px; margin-bottom: 16px; }
.primary-text { color: #303133; font-weight: 600; }
.secondary-text { margin: 3px 0; color: #909399; font-size: 12px; }
.event-cell { display: grid; justify-items: center; gap: 6px; }
.event-counts { display: flex; gap: 10px; color: #606266; font-size: 12px; }
.event-counts strong { color: #303133; }
.event-actions { display: flex; min-height: 22px; }
.identifier { display: grid; grid-template-columns: 58px minmax(0, 1fr); gap: 6px; margin: 3px 0; }
.identifier span { color: #909399; font-size: 12px; }
.identifier code, .ua-text { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.ua-text { max-width: 190px; margin-top: 4px; color: #909399; font-size: 12px; }
.pagination-wrap { display: flex; justify-content: flex-end; margin-top: 18px; }
.form-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 0 16px; }
.full-width { grid-column: 1 / -1; }
@media (max-width: 1280px) {
  .filters { grid-template-columns: repeat(3, minmax(150px, 1fr)); }
}
@media (max-width: 720px) {
  .page-header { align-items: stretch; flex-direction: column; }
  .filters, .form-grid { grid-template-columns: 1fr; }
  .full-width { grid-column: auto; }
  .page-wrap :deep(.el-card__header), .page-wrap :deep(.el-card__body) { padding: 14px; }
}
</style>
