<template>
  <div class="page-wrap">
    <div class="summary-grid">
      <el-card shadow="never">
        <div class="summary-label">筛选订单</div>
        <div class="summary-value">{{ formatInteger(total) }} 笔</div>
      </el-card>
      <el-card shadow="never">
        <div class="summary-label">支付成功</div>
        <div class="summary-value paid">{{ formatInteger(summary.paid_order_count) }} 笔</div>
      </el-card>
      <el-card shadow="never">
        <div class="summary-label">应付合计</div>
        <div class="summary-value compact">{{ formatSummaryAmounts('payable_total') }}</div>
      </el-card>
      <el-card shadow="never">
        <div class="summary-label">实付 / 已退款</div>
        <div class="summary-value compact summary-money-lines">
          <span class="paid">{{ formatSummaryAmounts('paid_total') }}</span>
          <span class="refunded">退款 {{ formatSummaryAmounts('refunded_total') }}</span>
        </div>
      </el-card>
    </div>

    <el-card shadow="never">
      <template #header>
        <div>
          <div class="page-title">订单管理</div>
          <div class="page-subtitle">查询订单、支付结果、权益内容及关联下单人；订单数据仅供运营查看</div>
        </div>
      </template>

      <div class="filters">
        <el-input
          v-model="query.keyword"
          clearable
          placeholder="订单号、交易号、商品或下单人"
          @keyup.enter="handleSearch"
        >
          <template #prefix><el-icon><Search /></el-icon></template>
        </el-input>
        <el-input-number v-model="query.user_id" :min="1" :controls="false" placeholder="下单人 ID" />
        <el-select v-model="query.product_type" clearable placeholder="产品类型">
          <el-option label="VIP" :value="1" />
          <el-option label="积分" :value="2" />
        </el-select>
        <el-select v-model="query.status" clearable placeholder="订单状态">
          <el-option v-for="item in statusOptions" :key="item.value" :label="item.label" :value="item.value" />
        </el-select>
        <el-select v-model="query.pay_type" clearable placeholder="支付方式">
          <el-option v-for="item in paymentOptions" :key="item.value" :label="item.label" :value="item.value" />
        </el-select>
        <el-input v-model="query.product_code" clearable placeholder="产品编码" @keyup.enter="handleSearch" />
        <el-date-picker
          v-model="dateRange"
          type="daterange"
          value-format="YYYY-MM-DD"
          range-separator="至"
          start-placeholder="创建开始日期"
          end-placeholder="创建结束日期"
          style="width: 100%"
        />
        <el-button type="primary" plain @click="handleSearch">查询</el-button>
        <el-button @click="handleReset">重置</el-button>
      </div>

      <el-table v-loading="loading" :data="tableData" row-key="id" stripe>
        <el-table-column label="订单" min-width="220" fixed="left">
          <template #default="{ row }">
            <div class="primary-text mono">{{ row.order_no }}</div>
            <div class="secondary-text">ID {{ row.id }} · {{ row.client_request_id }}</div>
          </template>
        </el-table-column>
        <el-table-column label="下单人" min-width="210">
          <template #default="{ row }">
            <div class="user-line">
              <span class="primary-text">{{ purchaserName(row) }}</span>
              <el-tag v-if="row.user?.deleted" size="small" type="info">已删除</el-tag>
            </div>
            <div class="secondary-text">ID {{ row.user_id }} · {{ purchaserContact(row) }}</div>
          </template>
        </el-table-column>
        <el-table-column label="产品" min-width="210">
          <template #default="{ row }">
            <div class="product-line">
              <el-tag size="small" effect="plain" :type="row.product_type === 1 ? 'warning' : 'success'">
                {{ productTypeLabel(row.product_type) }}
              </el-tag>
              <span class="primary-text">{{ row.product_name }}</span>
            </div>
            <div class="secondary-text">{{ row.product_code }} · ID {{ row.product_id }}</div>
          </template>
        </el-table-column>
        <el-table-column label="金额" min-width="155" align="right">
          <template #default="{ row }">
            <div class="primary-text money">实付 {{ formatMoney(row.paid_amount, row.currency) }}</div>
            <div class="secondary-text">应付 {{ formatMoney(row.payable_amount, row.currency) }}</div>
            <div v-if="row.refunded_amount > 0" class="secondary-text refunded">
              已退 {{ formatMoney(row.refunded_amount, row.currency) }}
            </div>
          </template>
        </el-table-column>
        <el-table-column label="权益" min-width="145">
          <template #default="{ row }">
            <template v-if="row.product_type === 1">
              <div class="primary-text">VIP {{ row.vip_level || '-' }}</div>
              <div class="secondary-text">{{ row.vip_duration_days || 0 }} 天</div>
            </template>
            <template v-else>
              <div class="primary-text">{{ formatInteger(row.bonus_points) }} 积分</div>
            </template>
          </template>
        </el-table-column>
        <el-table-column label="状态 / 支付" min-width="145">
          <template #default="{ row }">
            <el-tag size="small" :type="statusType(row.status)">{{ statusLabel(row.status) }}</el-tag>
            <div class="secondary-text">{{ paymentMethodLabel(row.pay_type) }}</div>
          </template>
        </el-table-column>
        <el-table-column label="交易号" min-width="185" show-overflow-tooltip>
          <template #default="{ row }">
            <span class="mono">{{ row.third_order_no || '-' }}</span>
          </template>
        </el-table-column>
        <el-table-column label="时间" width="180">
          <template #default="{ row }">
            <div>{{ formatDate(row.created_at) }}</div>
            <div class="secondary-text">支付 {{ formatDate(row.pay_time) }}</div>
          </template>
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

    <el-drawer v-model="detailVisible" title="订单详情" size="min(920px, 94vw)" destroy-on-close>
      <el-skeleton v-if="detailLoading" :rows="12" animated />
      <div v-else-if="detail" class="detail-wrap">
        <section>
          <h3>订单信息</h3>
          <el-descriptions :column="2" border>
            <el-descriptions-item label="订单 ID">{{ detail.id }}</el-descriptions-item>
            <el-descriptions-item label="订单状态">
              <el-tag size="small" :type="statusType(detail.status)">{{ statusLabel(detail.status) }}</el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="订单编号" :span="2"><span class="mono">{{ detail.order_no }}</span></el-descriptions-item>
            <el-descriptions-item label="客户端请求标识" :span="2"><span class="mono">{{ detail.client_request_id }}</span></el-descriptions-item>
            <el-descriptions-item label="产品类型">{{ productTypeLabel(detail.product_type) }}</el-descriptions-item>
            <el-descriptions-item label="产品 ID">{{ detail.product_id }}</el-descriptions-item>
            <el-descriptions-item label="产品名称">{{ detail.product_name }}</el-descriptions-item>
            <el-descriptions-item label="产品编码"><span class="mono">{{ detail.product_code }}</span></el-descriptions-item>
          </el-descriptions>
        </section>

        <section>
          <h3>下单人</h3>
          <el-alert v-if="!detail.user" type="warning" :closable="false" title="未找到关联用户，以下仅保留订单中的用户 ID" />
          <el-descriptions :column="2" border>
            <el-descriptions-item label="用户 ID">{{ detail.user_id }}</el-descriptions-item>
            <el-descriptions-item label="昵称">
              {{ detail.user?.username || '-' }}
              <el-tag v-if="detail.user?.deleted" class="inline-tag" size="small" type="info">已删除</el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="登录账号">{{ detail.user?.login_account || '-' }}</el-descriptions-item>
            <el-descriptions-item label="邮箱">{{ detail.user?.email || '-' }}</el-descriptions-item>
            <el-descriptions-item label="手机号">{{ detail.user?.phone || '-' }}</el-descriptions-item>
            <el-descriptions-item label="用户类型">{{ userTypeLabel(detail.user?.user_type) }}</el-descriptions-item>
            <el-descriptions-item label="设备编号">{{ detail.user?.device_code || '-' }}</el-descriptions-item>
            <el-descriptions-item label="IMEI">{{ detail.user?.imei || '-' }}</el-descriptions-item>
          </el-descriptions>
        </section>

        <section>
          <h3>金额与权益</h3>
          <el-descriptions :column="3" border>
            <el-descriptions-item label="商品原价">{{ formatMoney(detail.product_amount, detail.currency) }}</el-descriptions-item>
            <el-descriptions-item label="优惠金额">{{ formatMoney(detail.discount_amount, detail.currency) }}</el-descriptions-item>
            <el-descriptions-item label="应付金额">{{ formatMoney(detail.payable_amount, detail.currency) }}</el-descriptions-item>
            <el-descriptions-item label="实付金额"><strong class="paid">{{ formatMoney(detail.paid_amount, detail.currency) }}</strong></el-descriptions-item>
            <el-descriptions-item label="已退款"><strong :class="detail.refunded_amount > 0 ? 'refunded' : ''">{{ formatMoney(detail.refunded_amount, detail.currency) }}</strong></el-descriptions-item>
            <el-descriptions-item label="货币">{{ detail.currency }}</el-descriptions-item>
            <el-descriptions-item label="赠送积分">{{ formatInteger(detail.bonus_points) }}</el-descriptions-item>
            <el-descriptions-item label="VIP 等级">{{ detail.vip_level || '-' }}</el-descriptions-item>
            <el-descriptions-item label="VIP 天数">{{ detail.vip_duration_days || 0 }}</el-descriptions-item>
          </el-descriptions>
        </section>

        <section>
          <h3>支付信息</h3>
          <el-descriptions :column="2" border>
            <el-descriptions-item label="支付方式">{{ paymentMethodLabel(detail.pay_type) }}</el-descriptions-item>
            <el-descriptions-item label="平台交易 ID"><span class="mono">{{ detail.third_order_no || '-' }}</span></el-descriptions-item>
            <el-descriptions-item label="原始交易 ID" :span="2"><span class="mono">{{ detail.original_transaction_id || '-' }}</span></el-descriptions-item>
            <el-descriptions-item label="失败错误码">{{ detail.failure_code || '-' }}</el-descriptions-item>
            <el-descriptions-item label="失败描述">{{ detail.failure_message || '-' }}</el-descriptions-item>
            <el-descriptions-item label="取消原因" :span="2">{{ detail.cancel_reason || '-' }}</el-descriptions-item>
          </el-descriptions>
          <div class="evidence-title">支付凭证</div>
          <pre class="evidence-block">{{ formatEvidence(detail.payment_evidence) }}</pre>
        </section>

        <section>
          <h3>时间信息</h3>
          <el-descriptions :column="2" border>
            <el-descriptions-item label="创建时间">{{ formatDate(detail.created_at) }}</el-descriptions-item>
            <el-descriptions-item label="更新时间">{{ formatDate(detail.updated_at) }}</el-descriptions-item>
            <el-descriptions-item label="支付完成时间">{{ formatDate(detail.pay_time) }}</el-descriptions-item>
            <el-descriptions-item label="订单过期时间">{{ formatDate(detail.expires_at) }}</el-descriptions-item>
            <el-descriptions-item label="取消时间">{{ formatDate(detail.cancelled_at) }}</el-descriptions-item>
            <el-descriptions-item label="软删时间">{{ formatDate(detail.deleted_at) }}</el-descriptions-item>
          </el-descriptions>
        </section>
      </div>
    </el-drawer>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { getOrder, getOrderList, type OrderSummary, type VideoOrder } from '@/api/order'
import { orderPaymentLabel as paymentMethodLabel, orderPaymentOptions as paymentOptions } from '@/utils/orderPayment'
import { orderStatusLabel as statusLabel, orderStatusOptions as statusOptions, orderStatusType as statusType } from '@/utils/orderStatus'

const loading = ref(false)
const detailLoading = ref(false)
const detailVisible = ref(false)
const tableData = ref<VideoOrder[]>([])
const detail = ref<VideoOrder | null>(null)
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const dateRange = ref<string[]>([])
const summary = reactive<OrderSummary>({ paid_order_count: 0, amounts: [] })
const query = reactive({
  keyword: '',
  user_id: undefined as number | undefined,
  product_type: undefined as number | undefined,
  status: undefined as number | undefined,
  pay_type: undefined as number | undefined,
  product_code: '',
})

function productTypeLabel(value: number) {
  if (value === 1) return 'VIP'
  if (value === 2) return '积分'
  return `类型 ${value}`
}

function userTypeLabel(value?: number) {
  if (value === 1) return '免费用户'
  if (value === 2) return '付费用户'
  return '-'
}

function purchaserName(row: VideoOrder) {
  return row.user?.username || row.user?.email || row.user?.login_account || `用户 #${row.user_id}`
}

function purchaserContact(row: VideoOrder) {
  return row.user?.email || row.user?.phone || row.user?.login_account || row.user?.imei || '-'
}

function formatInteger(value: number) {
  return Number(value || 0).toLocaleString('zh-CN')
}

function formatMoney(value: number, currency: string) {
  return `${currency || '-'} ${Number(value || 0).toFixed(2)}`
}

function formatSummaryAmounts(field: 'payable_total' | 'paid_total' | 'refunded_total') {
  if (!summary.amounts?.length) return '0.00'
  return summary.amounts
    .map((item) => `${item.currency || '-'} ${Number(item[field] || 0).toLocaleString('zh-CN', { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`)
    .join(' / ')
}

function formatDate(value?: string | null) {
  if (!value) return '-'
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString('zh-CN', { hour12: false })
}

function formatEvidence(value?: string) {
  if (!value) return '无支付凭证'
  try {
    return JSON.stringify(JSON.parse(value), null, 2)
  } catch {
    return value
  }
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
    const res: any = await getOrderList(params)
    tableData.value = res.data.list || []
    total.value = res.data.total || 0
    Object.assign(summary, res.data.summary || { paid_order_count: 0, amounts: [] })
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
    keyword: '', user_id: undefined, product_type: undefined,
    status: undefined, pay_type: undefined, product_code: '',
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
    const res: any = await getOrder(id)
    detail.value = res.data
  } finally {
    detailLoading.value = false
  }
}

onMounted(fetchData)
</script>

<style scoped>
.page-wrap { min-width: 0; }
.summary-grid { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 14px; margin-bottom: 14px; }
.summary-label { color: #909399; font-size: 13px; }
.summary-value { margin-top: 8px; color: #303133; font-size: 22px; font-weight: 700; font-variant-numeric: tabular-nums; }
.summary-value.compact { font-size: 18px; }
.summary-money-lines { display: flex; flex-direction: column; gap: 4px; }
.paid { color: #67c23a; }
.refunded { color: #e6a23c; }
.page-title { color: #303133; font-size: 17px; font-weight: 600; }
.page-subtitle { margin-top: 4px; color: #909399; font-size: 12px; }
.filters { display: grid; grid-template-columns: minmax(250px, 1.6fr) repeat(5, minmax(135px, 1fr)) minmax(250px, 1.5fr) auto auto; gap: 10px; margin-bottom: 16px; }
.filters :deep(.el-input-number) { width: 100%; }
.primary-text { color: #303133; font-weight: 600; }
.secondary-text { margin-top: 4px; color: #909399; font-size: 12px; }
.mono { font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace; }
.money { font-variant-numeric: tabular-nums; }
.user-line, .product-line { display: flex; align-items: center; gap: 7px; min-width: 0; }
.pagination-wrap { display: flex; justify-content: flex-end; margin-top: 16px; overflow-x: auto; }
.detail-wrap { padding: 0 4px 24px; }
.detail-wrap section + section { margin-top: 24px; }
.detail-wrap h3 { margin: 0 0 10px; color: #303133; font-size: 15px; }
.inline-tag { margin-left: 8px; }
.evidence-title { margin: 14px 0 8px; color: #606266; font-size: 14px; font-weight: 600; }
.evidence-block { max-height: 360px; margin: 0; padding: 14px; overflow: auto; border: 1px solid #ebeef5; border-radius: 6px; background: #f7f8fa; color: #303133; font: 12px/1.6 ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace; white-space: pre-wrap; overflow-wrap: anywhere; }
@media (max-width: 1350px) {
  .filters { grid-template-columns: repeat(3, minmax(180px, 1fr)); }
}
@media (max-width: 900px) {
  .summary-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  .detail-wrap :deep(.el-descriptions__body) { overflow-x: auto; }
}
@media (max-width: 680px) {
  .filters, .summary-grid { grid-template-columns: 1fr; }
  .page-wrap :deep(.el-card__header), .page-wrap :deep(.el-card__body) { padding: 14px; }
}
</style>
