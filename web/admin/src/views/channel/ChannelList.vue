<template>
  <div class="page-wrap">
    <el-card shadow="never">
      <template #header>
        <div class="page-header">
          <div>
            <div class="page-title">渠道管理</div>
            <div class="page-subtitle">维护投放渠道、归因信息及事件回传规则</div>
          </div>
          <el-button v-if="canAdd" type="primary" @click="openCreate">
            <el-icon><Plus /></el-icon>创建渠道
          </el-button>
        </div>
      </template>

      <div class="filters">
        <div class="filter-item">
          <span>代理公司</span>
          <el-input v-model="query.agency_company" clearable placeholder="请输入" @keyup.enter="handleSearch" />
        </div>
        <div class="filter-item">
          <span>投放包</span>
          <el-select v-model="query.delivery_package" clearable filterable placeholder="请选择">
            <el-option
              v-for="item in enabledPackages"
              :key="item.id"
              :label="item.package_name"
              :value="item.package_code"
            />
          </el-select>
        </div>
        <div class="filter-item">
          <span>投放平台</span>
          <el-select v-model="query.ad_platform" clearable filterable placeholder="请选择">
            <el-option
              v-for="item in platformOptions"
              :key="item"
              :label="item"
              :value="item"
            />
          </el-select>
<!--          <el-input v-model="query.ad_platform" clearable placeholder="请输入" @keyup.enter="handleSearch" />-->
        </div>
        <div class="filter-item keyword-filter">
          <span>渠道名称/唯一码/ID</span>
          <el-input v-model="query.keyword" clearable placeholder="请输入" @keyup.enter="handleSearch" />
        </div>
        <div class="filter-actions">
          <el-button type="primary" @click="handleSearch">搜索</el-button>
          <el-button @click="handleReset">重置</el-button>
        </div>
      </div>

      <el-table v-loading="loading" :data="tableData" row-key="channel_id" stripe @sort-change="handleSortChange">
        <el-table-column prop="channel_id" label="渠道ID" width="88" fixed="left" sortable="custom" />
        <el-table-column prop="channel_code" label="唯一码" min-width="160" show-overflow-tooltip>
          <template #default="{ row }"><code class="channel-code">{{ row.channel_code }}</code></template>
        </el-table-column>
        <el-table-column prop="channel_name" label="渠道名称" min-width="130" show-overflow-tooltip />
        <el-table-column prop="agency_company" label="代理公司" min-width="130" show-overflow-tooltip>
          <template #default="{ row }">{{ row.agency_company || '-' }}</template>
        </el-table-column>
        <el-table-column prop="ad_platform" label="投放平台" min-width="110" />
        <el-table-column prop="ad_media" label="投放媒体" min-width="105" />
        <el-table-column label="投放包名" min-width="140" show-overflow-tooltip>
          <template #default="{ row }">{{ row.delivery_package_name || row.delivery_package || '-' }}</template>
        </el-table-column>
        <el-table-column label="监测链接" width="96" align="center">
          <template #default="{ row }">
            <el-link v-if="row.tracking_url" :href="row.tracking_url" type="primary" target="_blank" :underline="false">打开</el-link>
            <span v-else>-</span>
          </template>
        </el-table-column>
        <el-table-column prop="landing_page" label="落地页" min-width="110" show-overflow-tooltip>
          <template #default="{ row }">{{ row.landing_page || '-' }}</template>
        </el-table-column>
        <el-table-column label="端口返点" width="100" align="right">
          <template #default="{ row }">{{ formatRebate(row.port_rebate) }}%</template>
        </el-table-column>
        <el-table-column label="服务单费" width="100" align="right">
          <template #default="{ row }">¥{{ formatMoney(row.service_order_fee) }}</template>
        </el-table-column>
        <el-table-column prop="upload_method" label="上传方式" width="96" align="center">
          <template #default="{ row }"><el-tag effect="plain" type="info">{{ row.upload_method }}</el-tag></template>
        </el-table-column>
        <el-table-column v-if="canEdit || canDelete" label="操作" width="120" align="center">
          <template #default="{ row }">
            <el-button v-if="canEdit" link type="primary" @click="openEdit(row)">编辑</el-button>
            <el-popconfirm
              v-if="canDelete"
              :title="`确认删除渠道 ${row.channel_name}？`"
              width="240"
              @confirm="handleDelete(row.channel_id)"
            >
              <template #reference><el-button link type="danger">删除</el-button></template>
            </el-popconfirm>
          </template>
        </el-table-column>
        <el-table-column label="创建时间" width="172">
          <template #default="{ row }">{{ formatDate(row.created_at) }}</template>
        </el-table-column>
        <el-table-column label="最近更新时间" width="172">
          <template #default="{ row }">{{ formatDate(row.updated_at) }}</template>
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

    <el-dialog
      v-model="dialogVisible"
      :title="form.channel_id ? '修改渠道' : '创建渠道'"
      width="980px"
      top="4vh"
      destroy-on-close
    >
      <el-alert
        v-if="form.channel_id"
        class="code-alert"
        type="info"
        :closable="false"
        show-icon
        :title="`渠道唯一码：${form.channel_code}（系统生成，不可修改）`"
      />
      <el-form ref="formRef" :model="form" :rules="rules" label-width="96px">
        <div class="form-grid">
          <el-form-item label="渠道名称" prop="channel_name">
            <el-input v-model="form.channel_name" maxlength="128" placeholder="请输入" />
          </el-form-item>
          <el-form-item label="开户渠道" prop="account_channel">
            <el-input v-model="form.account_channel" maxlength="128" placeholder="请输入（选填）" />
          </el-form-item>
          <el-form-item label="代理公司" prop="agency_company">
            <el-input v-model="form.agency_company" maxlength="128" placeholder="请输入（选填）" />
          </el-form-item>
          <el-form-item label="归因平台" prop="ad_platform">
            <el-select v-model="form.ad_platform" placeholder="请选择" style="width: 100%">
              <el-option v-for="item in platformOptions" :key="item" :label="item" :value="item" />
            </el-select>
          </el-form-item>
          <el-form-item label="投放媒体" prop="ad_media">
            <el-select v-model="form.ad_media" placeholder="请选择" style="width: 100%">
              <el-option
                v-for="item in mediaOptions"
                :key="item.id"
                :label="item.name"
                :value="item.name"
              />
            </el-select>
          </el-form-item>
          <el-form-item label="包名" prop="delivery_package">
            <el-select v-model="form.delivery_package" filterable placeholder="请选择" style="width: 100%">
              <el-option
                v-for="item in enabledPackages"
                :key="item.id"
                :label="`${item.package_name} · ${item.package_code}`"
                :value="item.package_code"
              />
            </el-select>
          </el-form-item>
          <el-form-item label="系统" prop="system_type">
            <el-select v-model="form.system_type" placeholder="请选择" style="width: 100%">
              <el-option v-for="item in systemOptions" :key="item" :label="item" :value="item" />
            </el-select>
          </el-form-item>
          <el-form-item label="所属用户" prop="owner_admin_id">
            <el-select v-model="form.owner_admin_id" filterable placeholder="请选择" style="width: 100%">
              <el-option
                v-for="item in ownerOptions"
                :key="item.id"
                :label="ownerLabel(item)"
                :value="item.id"
              />
            </el-select>
          </el-form-item>
          <el-form-item label="上传方式" prop="upload_method">
            <el-select v-model="form.upload_method" placeholder="请选择" style="width: 100%">
              <el-option v-for="item in uploadMethodOptions" :key="item" :label="item" :value="item" />
            </el-select>
          </el-form-item>
          <el-form-item label="投放账户" prop="ad_account">
            <el-input v-model="form.ad_account" maxlength="128" placeholder="请输入" />
          </el-form-item>
          <el-form-item label="落地页" prop="landing_page">
            <el-select v-model="form.landing_page" placeholder="请选择" style="width: 100%">
              <el-option v-for="item in landingPageOptions" :key="item" :label="item" :value="item" />
            </el-select>
          </el-form-item>
          <el-form-item label="开户返点" prop="port_rebate">
            <div class="number-with-unit">
              <el-input-number
                v-model="form.port_rebate"
                :min="0"
                :max="100"
                :precision="4"
                :step="0.1"
                controls-position="right"
              />
              <span>%</span>
            </div>
            <div class="field-tip">用于后续成本计算，实际成本 = 前端成本 ÷ (1 + 返点)</div>
          </el-form-item>
        </div>

        <el-form-item label="回传配置" class="callback-form-item">
          <div class="callback-wrap">
            <el-checkbox-group
              v-model="selectedCallbackTypes"
              class="callback-event-group"
              @change="handleCallbackTypesChange"
            >
              <el-checkbox-button
                v-for="item in callbackEventOptions"
                :key="item.value"
                :value="item.value"
              >{{ item.label }}</el-checkbox-button>
            </el-checkbox-group>
            <div class="field-tip">最多同时选择 3 个触发类型；每个类型单独配置需要回传的事件和条件。</div>

            <div v-if="form.callback_config.rules.length" class="callback-rule-list">
              <section
                v-for="(rule, index) in form.callback_config.rules"
                :key="rule.trigger_event"
                class="callback-rule-card"
              >
                <div class="callback-rule-heading">
                  <span class="rule-index">{{ index + 1 }}</span>
                  <div>
                    <strong>{{ callbackEventLabel(rule.trigger_event) }}配置</strong>
                    <span>触发事件：{{ callbackEventDescription(rule.trigger_event) }}</span>
                  </div>
                </div>

                <el-checkbox-group
                  v-model="rule.callback_events"
                  class="rule-callback-options"
                  @change="handleRuleCallbacksChange(rule)"
                >
                  <div class="simple-event-options">
                    <el-checkbox value="activation">激活</el-checkbox>
                    <el-checkbox value="login">登陆</el-checkbox>
                  </div>
                  <div class="rule-condition-grid">
                    <div class="rule-condition-item">
                      <el-checkbox value="order_created">创建订单</el-checkbox>
                      <span>次数</span>
                      <el-input-number
                        v-model="rule.order_count_threshold"
                        :disabled="!hasRuleCallback(rule, 'order_created')"
                        :min="1"
                        :max="999999"
                        :precision="0"
                        controls-position="right"
                      />
                    </div>
                    <div class="rule-condition-item">
                      <el-checkbox value="payment">付费</el-checkbox>
                      <span>最低金额</span>
                      <el-input-number
                        v-model="rule.payment_minimum_amount"
                        :disabled="!hasRuleCallback(rule, 'payment')"
                        :min="0"
                        :max="9999999999.99"
                        :precision="2"
                        controls-position="right"
                      />
                    </div>
                    <div class="rule-condition-item">
                      <el-checkbox value="subscription">订阅</el-checkbox>
                      <span>延时</span>
                      <el-input-number
                        v-model="rule.subscription_delay_minutes"
                        :disabled="!hasRuleCallback(rule, 'subscription')"
                        :min="0"
                        :max="525600"
                        :precision="0"
                        controls-position="right"
                      />
                      <span class="unit-label">分钟</span>
                    </div>
                  </div>
                </el-checkbox-group>

                <div class="rule-deduction-row">
                  <el-checkbox
                    v-model="rule.amount_deduction_enabled"
                    :disabled="!hasRuleRevenueCallback(rule)"
                  >实际金额扣量</el-checkbox>
                  <span>扣量比例</span>
                  <el-input-number
                    v-model="rule.amount_deduction_percent"
                    :disabled="!rule.amount_deduction_enabled"
                    :min="0.01"
                    :max="100"
                    :precision="2"
                    :step="1"
                    controls-position="right"
                  />
                  <span class="unit-label">%</span>
                </div>
              </section>
            </div>
          </div>
        </el-form-item>
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
import { ElMessage, type FormInstance, type FormRules } from 'element-plus'
import {
  createChannel,
  deleteChannel,
  getChannelList,
  getMediaOptions,
  updateChannel,
  type Channel,
  type ChannelCallbackConfig,
  type ChannelCallbackEvent,
  type ChannelCallbackRule,
  type ChannelListParams,
  type ChannelPayload,
  type MediaOption,
} from '@/api/channel'
import { getPackageOptions, type AppPackage } from '@/api/package'
import { getUserOptions, type AdminOption } from '@/api/user'
import { useUserStore } from '@/store/user'
import { useRemoteTableSort } from '@/utils/tableSort'

interface ChannelForm extends ChannelPayload {
  channel_id: number
  channel_code: string
}

const userStore = useUserStore()
const canAdd = computed(() => userStore.hasPermission('channel:add'))
const canEdit = computed(() => userStore.hasPermission('channel:edit'))
const canDelete = computed(() => userStore.hasPermission('channel:delete'))

const platformOptions = ['Adjust', '热力引擎', 'AppsFlyer']
const systemOptions = ['iOS', 'Android', 'web']
const uploadMethodOptions = ['API', 'SDK']
const landingPageOptions = ['API']
const callbackEventOptions: Array<{ label: string; value: ChannelCallbackEvent }> = [
  { label: '创建订单', value: 'order_created' },
  { label: '付费', value: 'payment' },
  { label: '订阅', value: 'subscription' },
  { label: '激活', value: 'activation' },
  { label: '登陆', value: 'login' },
]

const loading = ref(false)
const submitting = ref(false)
const dialogVisible = ref(false)
const formRef = ref<FormInstance>()
const tableData = ref<Channel[]>([])
const packageOptions = ref<AppPackage[]>([])
const ownerOptions = ref<AdminOption[]>([])
const mediaOptions = ref<MediaOption[]>([])
const selectedCallbackTypes = ref<ChannelCallbackEvent[]>([])
const previousCallbackTypes = ref<ChannelCallbackEvent[]>([])
const page = ref(1)
const { sortParams, handleSortChange } = useRemoteTableSort(page, fetchData, { channel_id: 'id' })
const pageSize = ref(20)
const total = ref(0)
const query = reactive({ agency_company: '', delivery_package: '', ad_platform: '', keyword: '' })

const enabledPackages = computed(() => packageOptions.value.filter((item) => item.status === 1))

function defaultCallbackConfig(): ChannelCallbackConfig {
  return { rules: [] }
}

function defaultCallbackRule(triggerEvent: ChannelCallbackEvent): ChannelCallbackRule {
  return {
    trigger_event: triggerEvent,
    callback_events: [triggerEvent],
    order_count_threshold: 1,
    payment_minimum_amount: 0,
    subscription_delay_minutes: 0,
    amount_deduction_enabled: false,
    amount_deduction_percent: 1,
  }
}

function createDefaultForm(): ChannelForm {
  return {
    channel_id: 0,
    channel_code: '',
    channel_name: '',
    account_channel: '',
    agency_company: '',
    ad_platform: 'Adjust',
    ad_media: '',
    delivery_package: '',
    system_type: 'iOS',
    owner_admin_id: userStore.userInfo?.id || 0,
    ad_account: '',
    tracking_url: '',
    landing_page: 'API',
    port_rebate: 0,
    service_order_fee: 0,
    upload_method: 'API',
    callback_config: defaultCallbackConfig(),
    status: 1,
  }
}

const form = reactive<ChannelForm>(createDefaultForm())
const rules: FormRules = {
  channel_name: [{ required: true, message: '请输入渠道名称', trigger: 'blur' }],
  ad_platform: [{ required: true, message: '请选择归因平台', trigger: 'change' }],
  ad_media: [{ required: true, message: '请选择投放媒体', trigger: 'change' }],
  delivery_package: [{ required: true, message: '请选择投放包', trigger: 'change' }],
  system_type: [{ required: true, message: '请选择系统', trigger: 'change' }],
  owner_admin_id: [{ required: true, message: '请选择所属用户', trigger: 'change' }],
  upload_method: [{ required: true, message: '请选择上传方式', trigger: 'change' }],
  ad_account: [{ required: true, message: '请输入投放账户', trigger: 'blur' }],
  landing_page: [{ required: true, message: '请选择落地页', trigger: 'change' }],
  port_rebate: [{ type: 'number', min: 0, max: 100, message: '开户返点必须在 0 到 100 之间', trigger: 'change' }],
}

async function fetchData() {
  loading.value = true
  try {
    const params: ChannelListParams = { page: page.value, page_size: pageSize.value }
    Object.assign(params, sortParams())
    if (query.agency_company.trim()) params.agency_company = query.agency_company.trim()
    if (query.delivery_package.trim()) params.delivery_package = query.delivery_package.trim()
    if (query.ad_platform.trim()) params.ad_platform = query.ad_platform.trim()
    if (query.keyword.trim()) params.keyword = query.keyword.trim()
    const res: any = await getChannelList(params)
    tableData.value = res.data.list || []
    total.value = res.data.total || 0
  } finally {
    loading.value = false
  }
}

async function fetchFormOptions() {
  const [packageResult, ownerResult, mediaResult] = await Promise.allSettled([
    getPackageOptions(),
    getUserOptions(),
    getMediaOptions(),
  ])
  packageOptions.value = packageResult.status === 'fulfilled' ? ((packageResult.value as any).data || []) : []
  ownerOptions.value = ownerResult.status === 'fulfilled' ? ((ownerResult.value as any).data || []) : []
  mediaOptions.value = mediaResult.status === 'fulfilled' ? ((mediaResult.value as any).data || []) : []
  if (!ownerOptions.value.length && userStore.userInfo) {
    ownerOptions.value = [{
      id: userStore.userInfo.id,
      username: userStore.userInfo.username,
      nickname: userStore.userInfo.nickname,
    }]
  }
}

function handleSearch() {
  page.value = 1
  fetchData()
}

function handleReset() {
  Object.assign(query, { agency_company: '', delivery_package: '', ad_platform: '', keyword: '' })
  page.value = 1
  fetchData()
}

function handlePageSizeChange() {
  page.value = 1
  fetchData()
}

function openCreate() {
  Object.assign(form, createDefaultForm())
  form.callback_config = defaultCallbackConfig()
  selectedCallbackTypes.value = []
  previousCallbackTypes.value = []
  dialogVisible.value = true
}

function openEdit(row: Channel) {
  const callbackRules = (row.callback_config?.rules || []).slice(0, 3).map(cloneCallbackRule)
  Object.assign(form, {
    channel_id: row.channel_id,
    channel_code: row.channel_code,
    channel_name: row.channel_name,
    account_channel: row.account_channel || '',
    agency_company: row.agency_company || '',
    ad_platform: row.ad_platform || 'Adjust',
    ad_media: row.ad_media || '',
    delivery_package: row.delivery_package || '',
    system_type: row.system_type || 'iOS',
    owner_admin_id: row.owner_admin_id || userStore.userInfo?.id || 0,
    ad_account: row.ad_account || '',
    tracking_url: row.tracking_url || '',
    landing_page: row.landing_page || 'API',
    port_rebate: Number(row.port_rebate || 0),
    service_order_fee: Number(row.service_order_fee || 0),
    upload_method: row.upload_method || 'API',
    callback_config: { rules: callbackRules },
    status: row.status,
  })
  selectedCallbackTypes.value = callbackRules.map((rule) => rule.trigger_event)
  previousCallbackTypes.value = [...selectedCallbackTypes.value]
  dialogVisible.value = true
}

function cloneCallbackRule(rule: ChannelCallbackRule): ChannelCallbackRule {
  return {
    ...defaultCallbackRule(rule.trigger_event),
    ...rule,
    callback_events: [...(rule.callback_events || [])],
  }
}

function handleCallbackTypesChange() {
  const types = [...selectedCallbackTypes.value]
  if (types.length > 3) {
    selectedCallbackTypes.value = [...previousCallbackTypes.value]
    ElMessage.warning('回传配置最多同时选择 3 个')
    return
  }
  const existingRules = new Map(form.callback_config.rules.map((rule) => [rule.trigger_event, rule]))
  form.callback_config.rules = types.map((type) => existingRules.get(type) || defaultCallbackRule(type))
  previousCallbackTypes.value = types
}

function hasRuleCallback(rule: ChannelCallbackRule, event: ChannelCallbackEvent) {
  return rule.callback_events.includes(event)
}

function hasRuleRevenueCallback(rule: ChannelCallbackRule) {
  return hasRuleCallback(rule, 'payment') || hasRuleCallback(rule, 'subscription')
}

function handleRuleCallbacksChange(rule: ChannelCallbackRule) {
  if (!hasRuleRevenueCallback(rule)) {
    rule.amount_deduction_enabled = false
    rule.amount_deduction_percent = 0
  }
}

function callbackEventLabel(event: ChannelCallbackEvent) {
  return callbackEventOptions.find((item) => item.value === event)?.label || event
}

function callbackEventDescription(event: ChannelCallbackEvent) {
  const descriptions: Record<ChannelCallbackEvent, string> = {
    activation: 'App 启动并激活',
    login: '用户登陆或绑定邮箱',
    order_created: '创建订单',
    payment: '一次性付费成功',
    subscription: '订阅成功',
  }
  return descriptions[event]
}

function validateCallbackConfig() {
  if (form.callback_config.rules.length > 3) {
    ElMessage.warning('回传配置最多同时选择 3 个')
    return false
  }
  for (const rule of form.callback_config.rules) {
    const prefix = `${callbackEventLabel(rule.trigger_event)}配置：`
    if (hasRuleCallback(rule, 'order_created') && rule.order_count_threshold < 1) {
      ElMessage.warning(`${prefix}创建订单回传次数必须大于 0`)
      return false
    }
    if (rule.amount_deduction_enabled) {
      if (!hasRuleRevenueCallback(rule)) {
        ElMessage.warning(`${prefix}金额扣量仅适用于付费或订阅回传`)
        return false
      }
      if (rule.amount_deduction_percent <= 0 || rule.amount_deduction_percent > 100) {
        ElMessage.warning(`${prefix}金额扣量比例必须大于 0 且不超过 100%`)
        return false
      }
    }
  }
  return true
}

function normalizedCallbackConfig(): ChannelCallbackConfig {
  return {
    rules: form.callback_config.rules.map((source) => {
      const rule = cloneCallbackRule(source)
      if (!rule.callback_events.includes('order_created')) rule.order_count_threshold = 0
      if (!rule.callback_events.includes('payment')) rule.payment_minimum_amount = 0
      if (!rule.callback_events.includes('subscription')) rule.subscription_delay_minutes = 0
      if (!rule.callback_events.includes('payment') && !rule.callback_events.includes('subscription')) {
        rule.amount_deduction_enabled = false
        rule.amount_deduction_percent = 0
      } else if (!rule.amount_deduction_enabled) {
        rule.amount_deduction_percent = 0
      }
      return rule
    }),
  }
}

async function handleSubmit() {
  await formRef.value?.validate()
  if (!validateCallbackConfig()) return
  submitting.value = true
  try {
    const payload: ChannelPayload = {
      channel_code: form.channel_code || undefined,
      channel_name: form.channel_name.trim(),
      account_channel: form.account_channel.trim(),
      agency_company: form.agency_company.trim(),
      ad_platform: form.ad_platform,
      ad_media: form.ad_media,
      delivery_package: form.delivery_package,
      system_type: form.system_type,
      owner_admin_id: Number(form.owner_admin_id),
      ad_account: form.ad_account.trim(),
      tracking_url: form.tracking_url.trim(),
      landing_page: form.landing_page,
      port_rebate: Number(form.port_rebate),
      service_order_fee: Number(form.service_order_fee),
      upload_method: form.upload_method,
      callback_config: normalizedCallbackConfig(),
      status: form.status,
    }
    if (form.channel_id) await updateChannel(form.channel_id, payload)
    else await createChannel(payload)
    ElMessage.success('渠道信息已保存')
    dialogVisible.value = false
    await fetchData()
  } finally {
    submitting.value = false
  }
}

async function handleDelete(id: number) {
  await deleteChannel(id)
  ElMessage.success('渠道已删除')
  if (tableData.value.length === 1 && page.value > 1) page.value--
  await fetchData()
}

function ownerLabel(item: AdminOption) {
  return item.nickname ? `${item.nickname} · ${item.username}` : item.username
}

function formatRebate(value: number) {
  return Number(value || 0).toLocaleString('zh-CN', { maximumFractionDigits: 4 })
}

function formatMoney(value: number) {
  return Number(value || 0).toLocaleString('zh-CN', { minimumFractionDigits: 2, maximumFractionDigits: 2 })
}

function formatDate(value: string) {
  if (!value) return '-'
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString('zh-CN', { hour12: false })
}

onMounted(() => {
  fetchData()
  fetchFormOptions()
})
</script>

<style scoped>
.page-wrap { min-width: 0; }
.page-header { display: flex; align-items: center; justify-content: space-between; gap: 16px; }
.page-title { color: #303133; font-size: 17px; font-weight: 600; }
.page-subtitle { margin-top: 4px; color: #909399; font-size: 12px; }
.filters { display: grid; grid-template-columns: minmax(190px, .9fr) minmax(190px, 1fr) minmax(190px, .9fr) minmax(250px, 1.25fr) auto; gap: 12px; margin-bottom: 16px; }
.filter-item { display: grid; grid-template-columns: auto minmax(0, 1fr); align-items: center; gap: 9px; min-width: 0; }
.filter-item > span { color: #566276; font-size: 12px; font-weight: 600; white-space: nowrap; }
.filter-actions { display: flex; align-items: center; gap: 4px; }
.channel-code { padding: 2px 7px; border-radius: 4px; background: #f5f7fa; color: #606266; font-size: 12px; }
.pagination-wrap { display: flex; justify-content: flex-end; margin-top: 16px; overflow-x: auto; }
.code-alert { margin-bottom: 18px; }
.form-grid { display: grid; grid-template-columns: 1fr 1fr; column-gap: 24px; }
.number-with-unit { display: flex; align-items: center; gap: 8px; width: 100%; color: #606266; }
.number-with-unit :deep(.el-input-number) { flex: 1; width: 100%; }
.field-tip { width: 100%; margin-top: 5px; color: #909399; font-size: 12px; line-height: 18px; }
.callback-form-item :deep(.el-form-item__content) { min-width: 0; }
.callback-wrap { width: 100%; }
.callback-event-group { display: flex; flex-wrap: wrap; gap: 10px; }
.callback-event-group :deep(.el-checkbox-button__inner) { min-width: 96px; border: 1px solid var(--el-border-color) !important; border-radius: 7px !important; box-shadow: none !important; }
.callback-rule-list { display: flex; flex-direction: column; gap: 16px; margin-top: 16px; }
.callback-rule-card { overflow: hidden; border: 1px solid #d8e0e9; border-radius: 11px; background: #fbfcfe; }
.callback-rule-heading { display: flex; align-items: center; gap: 11px; padding: 12px 15px; border-bottom: 1px solid #e8edf3; background: #f4f8fd; }
.callback-rule-heading > div { display: flex; align-items: baseline; gap: 10px; min-width: 0; }
.callback-rule-heading strong { color: #2d3a4f; font-size: 14px; }
.callback-rule-heading span:not(.rule-index) { overflow: hidden; color: #8792a4; font-size: 12px; text-overflow: ellipsis; white-space: nowrap; }
.rule-index { display: inline-flex; align-items: center; justify-content: center; width: 24px; height: 24px; border-radius: 7px; background: #3479ed; color: #fff; font-size: 12px; font-weight: 700; }
.rule-callback-options { display: block; padding: 14px 15px 0; }
.simple-event-options { display: flex; gap: 28px; padding: 0 4px 12px; border-bottom: 1px dashed #e2e8f0; }
.rule-condition-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 10px; padding-top: 12px; }
.rule-condition-item { display: grid; grid-template-columns: auto auto minmax(120px, 1fr) auto; align-items: center; gap: 9px; min-width: 0; padding: 10px 11px; border: 1px solid #e4eaf1; border-radius: 8px; background: #fff; }
.rule-condition-item > span:not(.unit-label) { color: #657187; font-size: 12px; font-weight: 600; white-space: nowrap; }
.rule-condition-item :deep(.el-input-number) { width: 100%; }
.rule-deduction-row { display: grid; grid-template-columns: auto auto minmax(160px, 240px) auto; align-items: center; gap: 10px; margin: 12px 15px 15px; padding: 11px 12px; border: 1px solid #e4eaf1; border-radius: 8px; background: #fff; }
.rule-deduction-row > span:not(.unit-label) { color: #657187; font-size: 12px; font-weight: 600; }
.rule-deduction-row :deep(.el-input-number) { width: 100%; }
.unit-label { color: #606266; font-size: 12px; white-space: nowrap; }

@media (max-width: 1280px) {
  .filters { grid-template-columns: repeat(2, minmax(220px, 1fr)); }
  .filter-actions { justify-content: flex-end; }
}
@media (max-width: 760px) {
  .page-header { align-items: stretch; flex-direction: column; }
  .filters, .form-grid, .rule-condition-grid { grid-template-columns: 1fr; }
  .filter-actions { justify-content: stretch; }
  .filter-actions .el-button { flex: 1; }
  .callback-rule-heading > div { align-items: flex-start; flex-direction: column; gap: 2px; }
  .rule-condition-item, .rule-deduction-row { grid-template-columns: 1fr; }
  .page-wrap :deep(.el-card__header), .page-wrap :deep(.el-card__body) { padding: 14px; }
}
</style>
