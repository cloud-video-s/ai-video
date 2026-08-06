<template>
  <div class="dashboard-page">
    <section class="hero-panel">
      <div class="hero-glow hero-glow-one" />
      <div class="hero-glow hero-glow-two" />
      <div class="hero-copy">
        <div class="hero-kicker"><span class="live-dot" /> BUSINESS OVERVIEW</div>
        <h1>经营数据总览 <span>{{ monthLabel }}</span></h1>
        <p>自然月统计 · {{ statistics.timezone || 'Asia/Shanghai' }} 时区 · 数据实时更新</p>
      </div>
      <el-button class="refresh-button" :loading="loading" @click="fetchData">
        <el-icon><Refresh /></el-icon>
        刷新数据
      </el-button>
    </section>

    <div v-loading="loading && !initialized" class="metric-grid">
      <el-card shadow="never" class="metric-card users-card">
        <div class="metric-top">
          <div class="metric-label-wrap">
            <span class="metric-icon"><el-icon><UserFilled /></el-icon></span>
            <span class="metric-label">客户端注册用户</span>
          </div>
<!--          <span class="metric-index">01</span>-->
        </div>
        <div class="metric-value">{{ formatInteger(statistics.registered_user_count) }}</div>
        <div class="metric-note"><span />本月新增注册用户</div>
      </el-card>

      <el-card shadow="never" class="metric-card subscriptions-card">
        <div class="metric-top">
          <div class="metric-label-wrap">
            <span class="metric-icon"><el-icon><ShoppingBag /></el-icon></span>
            <span class="metric-label">成功订阅套餐</span>
          </div>
<!--          <span class="metric-index">02</span>-->
        </div>
        <div class="metric-value">{{ formatInteger(statistics.successful_subscription_count) }}</div>
        <div class="metric-note"><span />按支付完成时间统计</div>
      </el-card>

      <el-card shadow="never" class="metric-card amount-card">
        <div class="metric-top">
          <div class="metric-label-wrap">
            <span class="metric-icon"><el-icon><Wallet /></el-icon></span>
            <span class="metric-label">交易金额</span>
          </div>
<!--          <span class="metric-index">03</span>-->
        </div>
        <div v-if="statistics.transaction_amounts.length" class="amount-list">
          <div v-for="item in statistics.transaction_amounts" :key="item.currency" class="amount-row">
            <span>{{ item.currency || '-' }}</span>
            <strong>{{ formatMoneyValue(item.amount) }}</strong>
          </div>
        </div>
        <div v-else class="metric-value">0.00</div>
        <div class="metric-note"><span />全部已支付订单，按币种统计</div>
      </el-card>

      <el-card shadow="never" class="metric-card tasks-card">
        <div class="metric-top">
          <div class="metric-label-wrap">
            <span class="metric-icon"><el-icon><MagicStick /></el-icon></span>
            <span class="metric-label">成功生成任务</span>
          </div>
<!--          <span class="metric-index">04</span>-->
        </div>
        <div class="metric-value">{{ formatInteger(statistics.successful_generation_task_count) }}</div>
        <div class="metric-note"><span />按任务完成时间统计</div>
      </el-card>
    </div>

    <el-card shadow="never" class="orders-card">
      <template #header>
        <div class="card-heading">
          <div class="card-title-wrap">
            <span class="section-icon"><el-icon><Tickets /></el-icon></span>
            <div>
              <div class="card-title">订阅订单</div>
              <div class="card-subtitle">查看套餐购买记录与支付状态</div>
            </div>
          </div>
          <div class="total-badge"><span>{{ formatInteger(total) }}</span> 条订单</div>
        </div>
      </template>

      <div class="filter-toolbar">
        <el-input
          v-model="query.keyword"
          class="filter-item keyword-filter"
          clearable
          placeholder="搜索订单号、交易号、用户或套餐"
          @keyup.enter="handleSearch"
        >
          <template #prefix><el-icon><Search /></el-icon></template>
        </el-input>
        <el-input v-model="query.user_id" class="filter-item user-filter" clearable placeholder="用户 ID" @keyup.enter="handleSearch" />
        <el-select v-model="query.status" class="filter-item status-filter" clearable placeholder="订单状态">
          <el-option v-for="item in statusOptions" :key="item.value" :label="item.label" :value="item.value" />
        </el-select>
        <el-select v-model="query.pay_type" class="filter-item payment-filter" clearable placeholder="支付方式">
          <el-option v-for="item in paymentOptions" :key="item.value" :label="item.label" :value="item.value" />
        </el-select>
        <el-input v-model="query.product_code" class="filter-item product-filter" clearable placeholder="套餐编码" @keyup.enter="handleSearch" />
        <el-date-picker
          v-model="dateRange"
          class="filter-item date-filter"
          type="daterange"
          value-format="YYYY-MM-DD"
          range-separator="至"
          start-placeholder="下单开始日期"
          end-placeholder="下单结束日期"
        />
        <div class="filter-actions">
          <el-button type="primary" @click="handleSearch"><el-icon><Search /></el-icon>查询</el-button>
          <el-button @click="handleReset"><el-icon><RefreshLeft /></el-icon>重置</el-button>
        </div>
      </div>

      <el-table v-loading="loading" :data="tableData" row-key="id" empty-text="暂无订阅订单">
        <el-table-column label="订单信息" min-width="220" fixed="left">
          <template #default="{ row }">
            <div class="order-number mono">{{ row.order_no }}</div>
            <div class="secondary-text">内部 ID · {{ row.id }}</div>
          </template>
        </el-table-column>
        <el-table-column label="下单用户" min-width="225">
          <template #default="{ row }">
            <div class="user-cell">
              <span class="mini-avatar">{{ purchaserInitial(row) }}</span>
              <div class="user-meta">
                <div class="primary-text">
                  {{ purchaserName(row) }}
                  <el-tag v-if="row.user?.deleted" size="small" type="info">已删除</el-tag>
                </div>
                <div class="secondary-text">ID {{ row.user_id }} · {{ purchaserContact(row) }}</div>
              </div>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="订阅套餐" min-width="180">
          <template #default="{ row }">
            <div class="primary-text">{{ row.product_name || '-' }}</div>
            <div class="product-code mono">{{ row.product_code || '-' }}</div>
          </template>
        </el-table-column>
        <el-table-column label="实付金额" min-width="145" align="right">
          <template #default="{ row }">
            <div class="money-text"><span>{{ row.currency || '-' }}</span>{{ formatMoneyValue(row.paid_amount) }}</div>
            <div class="secondary-text">应付 {{ formatMoney(row.payable_amount, row.currency) }}</div>
          </template>
        </el-table-column>
        <el-table-column label="订单状态" width="115" align="center">
          <template #default="{ row }">
            <el-tag class="status-tag" size="small" :type="statusType(row.status)" effect="light">
              <span class="status-dot" />{{ statusLabel(row.status) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="支付渠道" min-width="130">
          <template #default="{ row }">
            <div class="payment-method"><span class="payment-mark" />{{ paymentMethodLabel(row.pay_type) }}</div>
          </template>
        </el-table-column>
        <el-table-column label="平台交易号" min-width="190" show-overflow-tooltip>
          <template #default="{ row }"><span class="transaction-number mono">{{ row.third_order_no || '-' }}</span></template>
        </el-table-column>
        <el-table-column label="下单 / 支付时间" width="190">
          <template #default="{ row }">
            <div class="time-line">{{ formatDate(row.created_at) }}</div>
            <div class="secondary-text">支付 · {{ formatDate(row.pay_time) }}</div>
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
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { getDashboard, type DashboardData, type DashboardStatistics } from '@/api/dashboard'
import type { VideoOrder } from '@/api/order'
import { orderPaymentLabel as paymentMethodLabel, orderPaymentOptions as paymentOptions } from '@/utils/orderPayment'
import { orderStatusLabel as statusLabel, orderStatusOptions as statusOptions, orderStatusType as statusType } from '@/utils/orderStatus'

const emptyStatistics = (): DashboardStatistics => ({
  month: '', period_start: '', period_end: '', timezone: '',
  registered_user_count: 0, successful_subscription_count: 0,
  transaction_amounts: [], successful_generation_task_count: 0,
})

const loading = ref(false)
const initialized = ref(false)
const statistics = reactive<DashboardStatistics>(emptyStatistics())
const tableData = ref<VideoOrder[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const dateRange = ref<string[]>([])
const query = reactive({
  keyword: '', user_id: '', status: undefined as number | undefined,
  pay_type: undefined as number | undefined, product_code: '',
})

const monthLabel = computed(() => {
  const [year, month] = statistics.month.split('-')
  if (year && month) return `${year}年${Number(month)}月`
  const now = new Date()
  return `${now.getFullYear()}年${now.getMonth() + 1}月`
})

function formatInteger(value: number) {
  return Number(value || 0).toLocaleString('zh-CN')
}

function formatMoneyValue(value: number) {
  return Number(value || 0).toLocaleString('zh-CN', { minimumFractionDigits: 2, maximumFractionDigits: 2 })
}

function formatMoney(value: number, currency: string) {
  return `${currency || '-'} ${formatMoneyValue(value)}`
}

function formatDate(value?: string | null) {
  if (!value) return '-'
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString('zh-CN', { hour12: false })
}

function purchaserName(row: VideoOrder) {
  return row.user?.username || row.user?.email || row.user?.login_account || `用户 #${row.user_id}`
}

function purchaserContact(row: VideoOrder) {
  return row.user?.email || row.user?.phone || row.user?.login_account || row.user?.imei || '-'
}

function purchaserInitial(row: VideoOrder) {
  return purchaserName(row).trim().charAt(0).toUpperCase() || 'U'
}

async function fetchData() {
  loading.value = true
  try {
    const params: Record<string, unknown> = { page: page.value, page_size: pageSize.value }
    for (const [key, value] of Object.entries(query)) {
      if (value === '' || value === undefined) continue
      const normalized = typeof value === 'string' ? value.trim() : value
      if (normalized !== '') params[key] = normalized
    }
    if (dateRange.value.length === 2) {
      params.date_from = dateRange.value[0]
      params.date_to = dateRange.value[1]
    }
    const res: any = await getDashboard(params)
    const data = res.data as DashboardData
    Object.assign(statistics, emptyStatistics(), data.statistics || {})
    tableData.value = data.subscription_orders?.list || []
    total.value = data.subscription_orders?.total || 0
    initialized.value = true
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
  Object.assign(query, { keyword: '', user_id: '', status: undefined, pay_type: undefined, product_code: '' })
  dateRange.value = []
  page.value = 1
  fetchData()
}

onMounted(fetchData)
</script>

<style scoped>
.dashboard-page {
  min-width: 0;
  min-height: calc(100% + 40px);
  margin: -20px;
  padding: 22px;
  box-sizing: border-box;
  background: #f4f7fb;
  color: #172033;
}
.hero-panel {
  position: relative;
  display: flex;
  min-height: 116px;
  align-items: center;
  justify-content: space-between;
  gap: 24px;
  margin-bottom: 16px;
  padding: 24px 28px;
  overflow: hidden;
  box-sizing: border-box;
  border: 1px solid rgba(255, 255, 255, .08);
  border-radius: 18px;
  background: linear-gradient(118deg, #111b32 0%, #172b4d 58%, #21486a 100%);
  box-shadow: 0 16px 38px rgba(25, 50, 86, .16);
  color: #fff;
}
.hero-copy { position: relative; z-index: 2; }
.hero-kicker { display: flex; align-items: center; gap: 8px; margin-bottom: 7px; color: #9fb8d6; font-size: 10px; font-weight: 700; letter-spacing: 1.6px; }
.live-dot { width: 7px; height: 7px; border-radius: 50%; background: #48d9a0; box-shadow: 0 0 0 5px rgba(72, 217, 160, .12); }
.hero-panel h1 { margin: 0; font-size: 25px; font-weight: 700; letter-spacing: -.5px; }
.hero-panel h1 span { margin-left: 8px; color: #86d9ff; font-size: 15px; font-weight: 600; letter-spacing: 0; }
.hero-panel p { margin: 7px 0 0; color: #b6c7da; font-size: 12px; }
.hero-glow { position: absolute; border-radius: 50%; pointer-events: none; filter: blur(1px); }
.hero-glow-one { right: 8%; bottom: -105px; width: 250px; height: 250px; background: radial-gradient(circle, rgba(48, 196, 255, .2), transparent 68%); }
.hero-glow-two { right: 29%; top: -125px; width: 220px; height: 220px; border: 1px solid rgba(133, 218, 255, .14); }
.refresh-button { position: relative; z-index: 2; height: 38px; border-color: rgba(255, 255, 255, .2); background: rgba(255, 255, 255, .1); color: #fff; backdrop-filter: blur(10px); }
.refresh-button:hover, .refresh-button:focus { border-color: rgba(255, 255, 255, .4); background: rgba(255, 255, 255, .18); color: #fff; }

.metric-grid { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 14px; min-height: 150px; margin-bottom: 16px; }
.metric-card { position: relative; min-width: 0; overflow: hidden; border: 1px solid #e8edf4; border-radius: 16px; background: #fff; box-shadow: 0 7px 24px rgba(32, 56, 85, .055); transition: transform .2s ease, box-shadow .2s ease; }
.metric-card::before { position: absolute; top: 0; right: 0; left: 0; height: 3px; background: var(--metric-color); content: ''; opacity: .9; }
.metric-card::after { position: absolute; right: -34px; bottom: -45px; width: 120px; height: 120px; border-radius: 50%; background: var(--metric-soft); content: ''; }
.metric-card:hover { transform: translateY(-2px); box-shadow: 0 12px 30px rgba(32, 56, 85, .09); }
.metric-card :deep(.el-card__body) { position: relative; z-index: 1; height: 100%; padding: 19px 19px 16px; box-sizing: border-box; }
.users-card { --metric-color: #3c8cff; --metric-soft: rgba(60, 140, 255, .08); }
.subscriptions-card { --metric-color: #22b991; --metric-soft: rgba(34, 185, 145, .08); }
.amount-card { --metric-color: #f19b42; --metric-soft: rgba(241, 155, 66, .09); }
.tasks-card { --metric-color: #8c6bea; --metric-soft: rgba(140, 107, 234, .08); }
.metric-top { display: flex; align-items: center; justify-content: space-between; gap: 10px; }
.metric-label-wrap { display: flex; min-width: 0; align-items: center; gap: 9px; }
.metric-icon { display: grid; flex: 0 0 34px; width: 34px; height: 34px; place-items: center; border-radius: 10px; background: var(--metric-soft); color: var(--metric-color); font-size: 17px; }
.metric-label { overflow: hidden; color: #657187; font-size: 13px; font-weight: 600; text-overflow: ellipsis; white-space: nowrap; }
.metric-index { color: #d8dee8; font: 700 11px/1 ui-monospace, SFMono-Regular, Menlo, monospace; letter-spacing: .6px; }
.metric-value { margin-top: 13px; color: #182235; font-size: 29px; font-weight: 750; line-height: 1; letter-spacing: -.8px; font-variant-numeric: tabular-nums; }
.metric-note { position: absolute; bottom: 16px; display: flex; align-items: center; gap: 6px; color: #9ba5b5; font-size: 11px; }
.metric-note > span { width: 5px; height: 5px; border-radius: 50%; background: var(--metric-color); opacity: .75; }
.amount-list { display: flex; flex-direction: column; gap: 3px; margin-top: 9px; }
.amount-row { display: flex; align-items: baseline; gap: 7px; line-height: 1.05; }
.amount-row span { min-width: 28px; color: #9a6b32; font-size: 10px; font-weight: 700; }
.amount-row strong { color: #182235; font-size: 20px; font-weight: 750; letter-spacing: -.4px; font-variant-numeric: tabular-nums; }

.orders-card { overflow: hidden; border: 1px solid #e6ecf3; border-radius: 16px; background: #fff; box-shadow: 0 8px 28px rgba(32, 56, 85, .055); }
.orders-card :deep(.el-card__header) { padding: 18px 20px; border-bottom: 1px solid #eef2f6; }
.orders-card :deep(.el-card__body) { padding: 16px 20px 18px; }
.card-heading { display: flex; align-items: center; justify-content: space-between; gap: 16px; }
.card-title-wrap { display: flex; align-items: center; gap: 12px; }
.section-icon { display: grid; width: 38px; height: 38px; place-items: center; border-radius: 11px; background: #edf4ff; color: #397cf4; font-size: 18px; }
.card-title { color: #202a3c; font-size: 16px; font-weight: 700; }
.card-subtitle { margin-top: 3px; color: #98a2b2; font-size: 11px; }
.total-badge { padding: 7px 11px; border: 1px solid #e4eaf1; border-radius: 9px; background: #f8fafc; color: #8b96a6; font-size: 11px; }
.total-badge span { margin-right: 3px; color: #2f6fd8; font-size: 13px; font-weight: 700; }
.filter-toolbar { display: flex; align-items: center; gap: 9px; margin-bottom: 16px; padding: 12px; border: 1px solid #edf1f6; border-radius: 12px; background: #f8fafc; flex-wrap: wrap; }
.filter-item { flex: 0 1 auto; }
.keyword-filter { flex: 1 1 250px; min-width: 220px; }
.user-filter { width: 112px; }
.status-filter { width: 128px; }
.payment-filter { width: 142px; }
.product-filter { width: 145px; }
.date-filter { width: 278px !important; }
.filter-actions { display: flex; gap: 1px; white-space: nowrap; }
.filter-toolbar :deep(.el-input__wrapper), .filter-toolbar :deep(.el-select__wrapper) { min-height: 36px; border-radius: 8px; background: #fff; box-shadow: 0 0 0 1px #e3e8ef inset; }
.filter-toolbar :deep(.el-input__wrapper:hover), .filter-toolbar :deep(.el-select__wrapper:hover) { box-shadow: 0 0 0 1px #b9c8db inset; }
.filter-toolbar :deep(.el-button) { height: 36px; border-radius: 8px; }
.filter-toolbar :deep(.el-button--primary) { border-color: #3479ed; background: #3479ed; box-shadow: 0 5px 12px rgba(52, 121, 237, .18); }

.orders-card :deep(.el-table) { --el-table-border-color: #edf1f5; --el-table-row-hover-bg-color: #f8fbff; color: #4f5c70; }
.orders-card :deep(.el-table th.el-table__cell) { height: 45px; padding: 0; background: #f7f9fc; color: #738096; font-size: 11px; font-weight: 650; }
.orders-card :deep(.el-table td.el-table__cell) { height: 62px; padding: 6px 0; border-bottom-color: #eef2f6; }
.orders-card :deep(.el-table__inner-wrapper::before) { display: none; }
.primary-text { overflow: hidden; color: #354155; font-size: 12px; font-weight: 650; text-overflow: ellipsis; white-space: nowrap; }
.secondary-text { margin-top: 4px; overflow: hidden; color: #9aa4b3; font-size: 10px; text-overflow: ellipsis; white-space: nowrap; }
.mono { font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace; }
.order-number { color: #2e6bc7; font-size: 12px; font-weight: 650; }
.user-cell { display: flex; align-items: center; gap: 9px; min-width: 0; }
.mini-avatar { display: grid; flex: 0 0 30px; width: 30px; height: 30px; place-items: center; border-radius: 9px; background: linear-gradient(135deg, #e8f1ff, #dce9fb); color: #3975cf; font-size: 12px; font-weight: 700; }
.user-meta { min-width: 0; }
.product-code { display: inline-flex; max-width: 100%; margin-top: 4px; padding: 2px 6px; overflow: hidden; border-radius: 5px; background: #f2f5f8; color: #8894a6; font-size: 9px; text-overflow: ellipsis; white-space: nowrap; }
.money-text { color: #1f2a3d; font-size: 15px; font-weight: 750; font-variant-numeric: tabular-nums; }
.money-text span { margin-right: 5px; color: #8995a6; font-size: 9px; font-weight: 700; }
.status-tag { border: none; border-radius: 999px; font-weight: 600; }
.status-dot { display: inline-block; width: 5px; height: 5px; margin-right: 5px; border-radius: 50%; background: currentColor; vertical-align: middle; }
.payment-method { display: flex; align-items: center; gap: 7px; color: #596579; font-size: 11px; }
.payment-mark { width: 7px; height: 7px; border: 2px solid #8ba2bd; border-radius: 50%; }
.transaction-number { color: #657187; font-size: 10px; }
.time-line { color: #4f5c70; font-size: 11px; font-variant-numeric: tabular-nums; }
.pagination-wrap { display: flex; justify-content: flex-end; margin-top: 16px; overflow-x: auto; }
.pagination-wrap :deep(.el-pagination) { --el-pagination-button-bg-color: #f5f7fa; --el-pagination-hover-color: #3479ed; }

@media (max-width: 1199px) {
  .metric-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  .metric-card { min-height: 145px; }
}
@media (max-width: 760px) {
  .dashboard-page { margin: -14px; padding: 14px; }
  .hero-panel { min-height: 140px; align-items: flex-start; padding: 21px; }
  .hero-panel h1 { font-size: 21px; }
  .hero-panel h1 span { display: block; margin: 5px 0 0; }
  .refresh-button { position: absolute; right: 16px; bottom: 16px; }
  .metric-grid { grid-template-columns: 1fr; }
  .filter-toolbar { align-items: stretch; }
  .filter-item, .keyword-filter, .user-filter, .status-filter, .payment-filter, .product-filter, .date-filter { width: 100% !important; min-width: 0; }
  .filter-actions { width: 100%; }
  .filter-actions :deep(.el-button) { flex: 1; }
  .orders-card :deep(.el-card__header), .orders-card :deep(.el-card__body) { padding-right: 14px; padding-left: 14px; }
}
</style>
