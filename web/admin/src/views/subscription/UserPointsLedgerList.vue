<template>
  <div class="page-wrap">
    <div class="summary-grid">
      <el-card shadow="never">
        <div class="summary-label">筛选结果</div>
        <div class="summary-value">{{ formatNumber(total) }} 条</div>
      </el-card>
      <el-card shadow="never">
        <div class="summary-label">积分收入</div>
        <div class="summary-value income">+{{ formatNumber(summary.income_total) }}</div>
      </el-card>
      <el-card shadow="never">
        <div class="summary-label">积分支出</div>
        <div class="summary-value expense">-{{ formatNumber(summary.expense_total) }}</div>
      </el-card>
      <el-card shadow="never">
        <div class="summary-label">净变动</div>
        <div class="summary-value" :class="netPoints >= 0 ? 'income' : 'expense'">
          {{ netPoints >= 0 ? '+' : '' }}{{ formatNumber(netPoints) }}
        </div>
      </el-card>
    </div>

    <el-card shadow="never">
      <template #header>
        <div>
          <div class="page-title">用户积分明细</div>
          <div class="page-subtitle">积分流水仅供查询；如需更正，请新增反向流水，保留完整审计记录</div>
        </div>
      </template>

      <div class="filters">
        <el-input
          v-model="query.keyword"
          clearable
          placeholder="昵称、设备号、账号、业务单号或说明"
          @keyup.enter="handleSearch"
        >
          <template #prefix><el-icon><Search /></el-icon></template>
        </el-input>
        <el-input-number v-model="query.user_id" :min="1" :controls="false" placeholder="用户 ID" />
        <el-select v-model="query.direction" clearable placeholder="收支方向">
          <el-option label="收入" value="1" />
          <el-option label="支出" value="2" />
        </el-select>
        <el-select v-model="query.source_type" clearable filterable allow-create placeholder="来源类型">
          <el-option v-for="item in sourceTypeOptions" :key="item.value" :label="item.label" :value="item.value" />
        </el-select>
        <el-select v-model="query.points_package_id" clearable filterable placeholder="积分套餐">
          <el-option
            v-for="item in packageOptions"
            :key="item.id"
            :label="`${item.name} · ${item.product_id}`"
            :value="String(item.id)"
          />
        </el-select>
        <el-input v-model="query.business_id" clearable placeholder="业务单号" @keyup.enter="handleSearch" />
        <el-date-picker
          v-model="dateRange"
          type="daterange"
          value-format="YYYY-MM-DD"
          range-separator="至"
          start-placeholder="开始日期"
          end-placeholder="结束日期"
          style="width: 100%"
        />
        <el-button type="primary" plain @click="handleSearch">查询</el-button>
        <el-button @click="handleReset">重置</el-button>
      </div>

      <el-table v-loading="loading" :data="tableData" row-key="id" stripe>
        <el-table-column prop="id" label="流水 ID" width="90" />
        <el-table-column label="用户" min-width="200">
          <template #default="{ row }">
            <div class="primary-text">{{ row.user?.username || `用户 #${row.user_id}` }}</div>
            <div class="secondary-text">ID {{ row.user_id }} · {{ row.user?.imei || '-' }}</div>
          </template>
        </el-table-column>
        <el-table-column label="积分变动" width="135" align="right">
          <template #default="{ row }">
            <strong class="change-value" :class="row.points_change >= 0 ? 'income' : 'expense'">
              {{ row.points_change >= 0 ? '+' : '' }}{{ formatNumber(row.points_change) }}
            </strong>
            <div class="secondary-text">{{ directionLabel(row.direction) }}</div>
          </template>
        </el-table-column>
        <el-table-column label="余额" width="185">
          <template #default="{ row }">
            <span>{{ formatNumber(row.balance_before) }}</span>
            <span class="balance-arrow">→</span>
            <strong>{{ formatNumber(row.balance_after) }}</strong>
          </template>
        </el-table-column>
        <el-table-column label="来源" min-width="185">
          <template #default="{ row }">
            <el-tag size="small" effect="plain">{{ sourceTypeLabel(row.source_type) }}</el-tag>
            <div class="secondary-text">{{ row.business_id || '无业务单号' }}</div>
          </template>
        </el-table-column>
        <el-table-column label="积分套餐" min-width="190">
          <template #default="{ row }">
            <span v-if="row.points_package">{{ row.points_package.name }}</span>
            <span v-else class="secondary-text">未关联套餐</span>
          </template>
        </el-table-column>
        <el-table-column prop="description" label="说明" min-width="220" show-overflow-tooltip />
        <el-table-column label="发生时间" width="180">
          <template #default="{ row }">{{ formatDate(row.occurred_at) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="80" fixed="right" align="center">
          <template #default="{ row }">
            <el-button link type="primary" @click="openDetail(row.id)">详情</el-button>
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

    <el-dialog v-model="detailVisible" title="积分流水详情" width="720px" destroy-on-close>
      <el-skeleton v-if="detailLoading" :rows="7" animated />
      <el-descriptions v-else-if="detail" :column="2" border>
        <el-descriptions-item label="流水 ID">{{ detail.id }}</el-descriptions-item>
        <el-descriptions-item label="发生时间">{{ formatDate(detail.occurred_at) }}</el-descriptions-item>
        <el-descriptions-item label="用户">{{ detail.user?.username || `用户 #${detail.user_id}` }}</el-descriptions-item>
        <el-descriptions-item label="用户类型">{{ userTypeLabel(detail.user?.user_type) }}</el-descriptions-item>
        <el-descriptions-item label="方向">{{ directionLabel(detail.direction) }}</el-descriptions-item>
        <el-descriptions-item label="积分变动">
          <strong :class="detail.points_change >= 0 ? 'income' : 'expense'">
            {{ detail.points_change >= 0 ? '+' : '' }}{{ formatNumber(detail.points_change) }}
          </strong>
        </el-descriptions-item>
        <el-descriptions-item label="变动前余额">{{ formatNumber(detail.balance_before) }}</el-descriptions-item>
        <el-descriptions-item label="变动后余额">{{ formatNumber(detail.balance_after) }}</el-descriptions-item>
        <el-descriptions-item label="来源类型">{{ sourceTypeLabel(detail.source_type) }}</el-descriptions-item>
        <el-descriptions-item label="业务单号">{{ detail.business_id || '-' }}</el-descriptions-item>
        <el-descriptions-item label="积分套餐" :span="2">
          {{ detail.points_package ? `${detail.points_package.name} · ${detail.points_package.product_id}` : '-' }}
        </el-descriptions-item>
        <el-descriptions-item label="后台操作员">{{ detail.operator_admin_id || '-' }}</el-descriptions-item>
        <el-descriptions-item label="入库时间">{{ formatDate(detail.created_at) }}</el-descriptions-item>
        <el-descriptions-item label="说明" :span="2">{{ detail.description || '-' }}</el-descriptions-item>
      </el-descriptions>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { getPointsPackageOptions, type PointsPackage } from '@/api/pointsPackage'
import {
  getUserPointsLedger,
  getUserPointsLedgerList,
  type UserPointsLedger,
  type UserPointsLedgerSummary,
} from '@/api/userPointsLedger'

const sourceTypeOptions = [
  { value: 'purchase', label: '购买套餐' },
  { value: 'consume', label: '积分消费' },
  { value: 'reward', label: '活动奖励' },
  { value: 'refund', label: '退款返还' },
  { value: 'subscription', label: '订阅赠送' },
  { value: 'admin', label: '后台调整' },
  { value: 'other', label: '其他' },
]

const loading = ref(false)
const detailLoading = ref(false)
const detailVisible = ref(false)
const tableData = ref<UserPointsLedger[]>([])
const detail = ref<UserPointsLedger | null>(null)
const packageOptions = ref<PointsPackage[]>([])
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const dateRange = ref<string[]>([])
const summary = reactive<UserPointsLedgerSummary>({ income_total: 0, expense_total: 0 })
const query = reactive({
  keyword: '',
  user_id: undefined as number | undefined,
  direction: '',
  source_type: '',
  points_package_id: '',
  business_id: '',
})

const netPoints = computed(() => Number(summary.income_total || 0) - Number(summary.expense_total || 0))

function directionLabel(value: number) {
  return value === 1 ? '收入' : '支出'
}

function sourceTypeLabel(value: string) {
  return sourceTypeOptions.find((item) => item.value === value)?.label || value
}

function userTypeLabel(value?: number) {
  if (value === 1) return '免费用户'
  if (value === 2) return '付费用户'
  return '-'
}

function formatNumber(value: number) {
  return Number(value || 0).toLocaleString('zh-CN')
}

function formatDate(value: string) {
  if (!value) return '-'
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString('zh-CN', { hour12: false })
}

async function fetchData() {
  loading.value = true
  try {
    const params: Record<string, unknown> = { page: page.value, page_size: pageSize.value }
    for (const [key, value] of Object.entries(query)) {
      if (value !== '' && value !== undefined) {
        params[key] = typeof value === 'string' ? value.trim() : value
      }
    }
    if (dateRange.value.length === 2) {
      params.date_from = dateRange.value[0]
      params.date_to = dateRange.value[1]
    }
    const res: any = await getUserPointsLedgerList(params)
    tableData.value = res.data.list || []
    total.value = res.data.total || 0
    Object.assign(summary, res.data.summary || { income_total: 0, expense_total: 0 })
  } finally {
    loading.value = false
  }
}

function handleSearch() {
  page.value = 1
  fetchData()
}

function handlePageSizeChange() {
  page.value = 1
  fetchData()
}

function handleReset() {
  Object.assign(query, {
    keyword: '',
    user_id: undefined,
    direction: '',
    source_type: '',
    points_package_id: '',
    business_id: '',
  })
  dateRange.value = []
  page.value = 1
  fetchData()
}

async function openDetail(id: number) {
  detailVisible.value = true
  detailLoading.value = true
  detail.value = null
  try {
    const res: any = await getUserPointsLedger(id)
    detail.value = res.data
  } finally {
    detailLoading.value = false
  }
}

onMounted(async () => {
  const options: any = await getPointsPackageOptions()
  packageOptions.value = options.data || []
  await fetchData()
})
</script>

<style scoped>
.page-wrap { min-width: 0; }
.summary-grid { display: grid; grid-template-columns: repeat(4, 1fr); gap: 14px; margin-bottom: 14px; }
.summary-label { color: #909399; font-size: 13px; }
.summary-value { margin-top: 8px; color: #303133; font-size: 22px; font-weight: 700; font-variant-numeric: tabular-nums; }
.income { color: #67c23a; }
.expense { color: #f56c6c; }
.page-title { color: #303133; font-size: 17px; font-weight: 600; }
.page-subtitle { margin-top: 4px; color: #909399; font-size: 12px; }
.filters { display: grid; grid-template-columns: minmax(240px, 1.5fr) repeat(5, minmax(130px, 1fr)) minmax(240px, 1.4fr) auto auto; gap: 10px; margin-bottom: 16px; }
.filters :deep(.el-input-number) { width: 100%; }
.primary-text { color: #303133; font-weight: 600; }
.secondary-text { margin-top: 4px; color: #909399; font-size: 12px; }
.change-value { font-size: 16px; }
.balance-arrow { margin: 0 8px; color: #c0c4cc; }
.pagination-wrap { display: flex; justify-content: flex-end; margin-top: 16px; overflow-x: auto; }
@media (max-width: 1200px) {
  .filters { grid-template-columns: repeat(3, minmax(160px, 1fr)); }
  .summary-grid { grid-template-columns: repeat(2, 1fr); }
}
@media (max-width: 680px) {
  .filters, .summary-grid { grid-template-columns: 1fr; }
  .page-wrap :deep(.el-card__header), .page-wrap :deep(.el-card__body) { padding: 14px; }
}
</style>
