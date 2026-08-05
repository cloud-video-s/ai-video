<template>
  <div class="page-wrap">
    <el-card shadow="never">
      <template #header>
        <div class="page-header">
          <div>
            <div class="page-title">VIP 订阅管理</div>
            <div class="page-subtitle">维护 VIP 等级、商店 SKU、价格权益及应用、安装包、版本投放范围</div>
          </div>
          <el-button v-if="canAdd" type="primary" @click="openCreate">
            <el-icon><Plus /></el-icon>新增套餐
          </el-button>
        </div>
      </template>

      <div class="filters">
        <el-select v-model="query.vip_type" clearable placeholder="套餐类型">
          <el-option v-for="item in vipTypeOptions" :key="item.value" :label="item.label" :value="String(item.value)" />
        </el-select>
        <el-select v-model="query.level_id" clearable filterable placeholder="VIP 等级">
          <el-option v-for="item in levelOptions" :key="item.id" :label="item.level" :value="String(item.id)" />
        </el-select>
        <el-select v-model="query.app_code" clearable filterable placeholder="应用" @change="handleFilterAppChange">
          <el-option v-for="item in deliveryOptions" :key="item.app_code" :label="appLabel(item)" :value="item.app_code" />
        </el-select>
        <el-select v-model="query.package_code" clearable filterable placeholder="安装包" @change="handleFilterPackageChange">
          <el-option v-for="item in filterPackageOptions" :key="item.package_code" :label="packageLabel(item)" :value="item.package_code" />
        </el-select>
        <el-select v-model="query.version_code" clearable filterable placeholder="版本">
          <el-option v-for="item in filterVersionOptions" :key="item.version_code" :label="item.version_code" :value="item.version_code" />
        </el-select>
        <el-select v-model="query.status" clearable placeholder="状态">
          <el-option label="启用" value="1" />
          <el-option label="禁用" value="0" />
        </el-select>
        <el-select v-model="query.display_mode" clearable placeholder="显示模式">
          <el-option label="正常显示" value="1" />
          <el-option label="隐藏" value="0" />
        </el-select>
        <el-select v-model="query.is_subscription" clearable placeholder="订阅方式">
          <el-option label="循环订阅" value="1" />
          <el-option label="一次性" value="0" />
        </el-select>
        <el-input v-model="query.keyword" clearable placeholder="产品 SKU、名称或描述" @keyup.enter="handleSearch" />
        <el-button type="primary" plain @click="handleSearch">查询</el-button>
        <el-button @click="handleReset">重置</el-button>
      </div>

      <el-table v-loading="loading" :data="tableData" row-key="id" stripe>
        <el-table-column prop="id" label="ID" width="70" />
        <el-table-column prop="sort" label="排序" width="68" align="center" />
        <el-table-column label="套餐" min-width="240">
          <template #default="{ row }">
            <div class="primary-text">{{ row.name }}</div>
            <div class="secondary-text mono">{{ row.suk_code }}</div>
            <div class="tag-row">
              <el-tag size="small" effect="plain">{{ vipTypeLabel(row.vip_type) }}</el-tag>
              <el-tag size="small" type="warning" effect="plain">{{ levelLabel(row) }}</el-tag>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="应用 / 安装包 / 版本" min-width="310">
          <template #default="{ row }">
            <div class="target-tags">
              <el-tag size="small" type="info" effect="plain">{{ appSummary(row.apps) }}</el-tag>
              <el-tag size="small" type="info" effect="plain">{{ packageSummary(row.packages) }}</el-tag>
              <el-tag size="small" type="info" effect="plain">{{ versionSummary(row.package_version) }}</el-tag>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="国家 / 渠道" min-width="230">
          <template #default="{ row }">
            <div class="secondary-text">国家：{{ countrySummary(row.country) }}</div>
            <div class="secondary-text">渠道：{{ channelSummary(row.channels) }}</div>
          </template>
        </el-table-column>
        <el-table-column label="价格" width="155">
          <template #default="{ row }">
            <div class="price">{{ row.currency }} {{ formatMoney(row.subscription_price) }}</div>
            <div v-if="row.original_price" class="original-price">{{ row.currency }} {{ formatMoney(row.original_price) }}</div>
            <div class="secondary-text">{{ subscriptionPeriodLabel(row.subscription_period) }}</div>
          </template>
        </el-table-column>
        <el-table-column label="属性" width="155">
          <template #default="{ row }">
            <div class="tag-row">
              <el-tag v-if="row.is_default" size="small" type="success">默认</el-tag>
              <el-tag v-if="row.free_trial" size="small" type="warning">{{ row.trial_days }}天试用</el-tag>
              <el-tag size="small" :type="row.is_subscription ? '' : 'info'">{{ row.is_subscription ? '循环订阅' : '一次性' }}</el-tag>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="150" align="center">
          <template #default="{ row }">
            <el-switch
              v-if="canEdit"
              v-model="row.status"
              :active-value="1"
              :inactive-value="0"
              inline-prompt
              active-text="启用"
              inactive-text="禁用"
              :loading="updatingIds.includes(row.id)"
              @change="handleStatusChange(row)"
            />
            <el-tag v-else :type="row.status === 1 ? 'success' : 'danger'">{{ row.status === 1 ? '启用' : '禁用' }}</el-tag>
            <div class="display-state">{{ row.display_mode === 1 ? '正常显示' : '已隐藏' }}</div>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="260" fixed="right" align="center">
          <template #default="{ row }">
            <el-button v-if="canEdit" link type="primary" @click="openEdit(row)">编辑</el-button>
            <el-button v-if="canEdit" link :type="row.display_mode === 1 ? 'warning' : 'success'" @click="toggleDisplay(row)">
              {{ row.display_mode === 1 ? '隐藏' : '显示' }}
            </el-button>
            <el-button v-if="canEdit && !row.is_default" link type="primary" @click="setDefault(row)">设为默认</el-button>
            <el-button v-if="canAdd" link type="primary" @click="clonePlan(row)">复制</el-button>
            <el-popconfirm v-if="canDelete" title="确认删除该 VIP 套餐？" @confirm="handleDelete(row.id)">
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

    <el-dialog
      v-model="dialogVisible"
      :title="form.id ? '编辑 VIP 订阅套餐' : '新增 VIP 订阅套餐'"
      width="1040px"
      top="4vh"
      class="subscription-edit-dialog"
      destroy-on-close
      :close-on-click-modal="false"
    >
      <el-form ref="formRef" :model="form" :rules="rules" label-width="124px">
        <div class="form-sections">
          <section class="form-section">
            <div class="section-heading">
              <div class="section-title">基础配置</div>
              <div class="section-description">设置套餐身份、类型与投放范围</div>
            </div>
            <div class="form-grid">
              <el-form-item label="产品 SKU" prop="suk_code">
                <el-input v-model="form.suk_code" maxlength="191" placeholder="应用商店订阅 SKU" />
              </el-form-item>
              <el-form-item label="VIP 名称" prop="name">
                <el-input v-model="form.name" maxlength="128" />
              </el-form-item>
              <el-form-item label="VIP 等级" prop="level_id">
                <el-select v-model="form.level_id" filterable placeholder="请选择 VIP 等级" style="width: 100%">
                  <el-option v-for="item in levelOptions" :key="item.id" :label="levelOptionLabel(item)" :value="item.id" />
                </el-select>
              </el-form-item>
              <el-form-item label="套餐类型" prop="vip_type">
                <el-select v-model="form.vip_type" style="width: 100%">
                  <el-option v-for="item in vipTypeOptions" :key="item.value" :label="item.label" :value="item.value" />
                </el-select>
              </el-form-item>
              <el-form-item label="排序">
                <el-input-number v-model="form.sort" :min="0" :max="999999" controls-position="right" />
              </el-form-item>
            </div>

            <el-divider content-position="left">投放范围</el-divider>
            <el-form-item label="国家">
              <div class="scope-field">
                <el-radio-group v-model="targetModes.countries" @change="handleCountryModeChange">
                  <el-radio-button value="all">全部国家</el-radio-button>
                  <el-radio-button value="selected">指定国家</el-radio-button>
                </el-radio-group>
                <el-select v-if="targetModes.countries === 'selected'" v-model="form.country_codes" multiple collapse-tags collapse-tags-tooltip clearable filterable placeholder="请选择国家" style="width: 100%">
                  <el-option v-for="item in countryOptions" :key="item.code" :label="countryLabel(item)" :value="item.code" />
                </el-select>
                <div class="scope-tip">全部国家不会写入国家关联数据。</div>
              </div>
            </el-form-item>

            <el-form-item label="应用范围">
              <div class="scope-field">
                <el-radio-group v-model="targetModes.apps" @change="handleAppModeChange">
                  <el-radio-button value="all">全部应用</el-radio-button>
                  <el-radio-button value="selected">指定应用</el-radio-button>
                </el-radio-group>
                <el-select v-if="targetModes.apps === 'selected'" v-model="form.app_codes" multiple collapse-tags collapse-tags-tooltip clearable filterable placeholder="请先选择应用" style="width: 100%" @change="handleAppSelectionChange">
                  <el-option v-for="item in deliveryOptions" :key="item.app_code" :label="appLabel(item)" :value="item.app_code" />
                </el-select>
                <div class="scope-tip">指定应用时，安装包选项将按已选应用联动过滤。</div>
              </div>
            </el-form-item>

            <el-form-item v-if="targetModes.apps === 'all' || form.app_codes.length" label="安装包" prop="package_codes">
              <div class="scope-field">
                <el-radio-group v-model="targetModes.packages" @change="handlePackageModeChange">
                  <el-radio-button value="all">全部安装包</el-radio-button>
                  <el-radio-button value="selected">指定安装包</el-radio-button>
                </el-radio-group>
                <el-select v-if="targetModes.packages === 'selected'" v-model="form.package_codes" multiple collapse-tags collapse-tags-tooltip clearable filterable placeholder="请选择安装包" style="width: 100%" @change="handlePackageSelectionChange">
                  <el-option v-for="item in formPackageOptions" :key="item.package_code" :label="packageLabel(item)" :value="item.package_code" />
                </el-select>
                <div class="scope-tip">全部安装包会关联当前应用范围内的所有安装包；版本选项随安装包联动。</div>
              </div>
            </el-form-item>

            <el-form-item v-if="form.package_codes.length" label="版本">
              <div class="scope-field">
                <el-radio-group v-model="targetModes.versions" @change="handleVersionModeChange">
                  <el-radio-button value="all">全部版本</el-radio-button>
                  <el-radio-button value="selected">指定版本</el-radio-button>
                </el-radio-group>
                <el-select v-if="targetModes.versions === 'selected'" v-model="form.version_codes" multiple collapse-tags collapse-tags-tooltip clearable filterable placeholder="请选择版本" style="width: 100%">
                  <el-option v-for="item in formVersionOptions" :key="item.version_code" :label="item.label" :value="item.version_code" />
                </el-select>
                <div class="scope-tip">全部版本不会写入版本关联数据。</div>
              </div>
            </el-form-item>

            <el-form-item label="渠道">
              <div class="scope-field">
                <el-radio-group v-model="targetModes.channels" @change="handleChannelModeChange">
                  <el-radio-button value="all">全部渠道</el-radio-button>
                  <el-radio-button value="selected">指定渠道</el-radio-button>
                </el-radio-group>
                <el-select v-if="targetModes.channels === 'selected'" v-model="form.channel_codes" multiple collapse-tags collapse-tags-tooltip clearable filterable placeholder="请选择渠道" style="width: 100%">
                  <el-option v-for="item in channelOptions" :key="item.channel_code" :label="channelLabel(item)" :value="item.channel_code" />
                </el-select>
                <div class="scope-tip">全部渠道不会写入渠道关联数据。</div>
              </div>
            </el-form-item>

            <el-form-item label="套餐描述">
              <el-input v-model="form.description" type="textarea" :rows="3" maxlength="1000" show-word-limit />
            </el-form-item>
          </section>

          <section class="form-section">
            <div class="section-heading">
              <div class="section-title">价格与权益</div>
              <div class="section-description">配置首次订阅、续订、积分和试用规则</div>
            </div>
            <div class="form-grid">
              <el-form-item label="币种" prop="currency">
                <el-input v-model="form.currency" maxlength="3" @input="form.currency = form.currency.toUpperCase()" />
              </el-form-item>
              <el-form-item label="划线金额"><el-input-number v-model="form.original_price" :min="0" :precision="2" :step="0.01" controls-position="right" /></el-form-item>
              <el-form-item label="首次订阅金额"><el-input-number v-model="form.first_subscription_price" :min="0" :precision="2" :step="0.01" controls-position="right" /></el-form-item>
              <el-form-item label="首次实际收入"><el-input-number v-model="form.first_subscription_revenue" :min="0" :precision="2" :step="0.01" controls-position="right" /></el-form-item>
              <el-form-item label="首次赠送积分"><el-input-number v-model="form.first_bonus_points" :min="0" controls-position="right" /></el-form-item>
              <el-form-item label="VIP 时长（天）"><el-input-number v-model="form.vip_duration_days" :min="0" controls-position="right" /></el-form-item>
              <el-form-item label="续订金额"><el-input-number v-model="form.subscription_price" :min="0" :precision="2" :step="0.01" controls-position="right" /></el-form-item>
              <el-form-item label="续订实际收入"><el-input-number v-model="form.subscription_revenue" :min="0" :precision="2" :step="0.01" controls-position="right" /></el-form-item>
              <el-form-item label="续订积分"><el-input-number v-model="form.subscription_points" :min="0" controls-position="right" /></el-form-item>
              <el-form-item label="订阅周期" prop="subscription_period">
                <el-select v-model="form.subscription_period" style="width: 100%">
                  <el-option v-for="item in subscriptionPeriodOptions" :key="item.value" :label="item.label" :value="item.value" />
                </el-select>
              </el-form-item>
              <el-form-item label="试用天数"><el-input-number v-model="form.trial_days" :min="0" :max="3650" controls-position="right" /></el-form-item>
              <el-form-item label="免费体验"><el-switch v-model="form.free_trial" /></el-form-item>
            </div>
            <el-form-item label="订阅说明"><el-input v-model="form.subscription_description" type="textarea" :rows="3" maxlength="500" show-word-limit /></el-form-item>
          </section>

          <section class="form-section">
            <div class="section-heading">
              <div class="section-title">展示与状态</div>
              <div class="section-description">设置前台文案、默认项与上架状态</div>
            </div>
            <div class="form-grid">
              <el-form-item label="续费文案"><el-input v-model="form.renewal_text" maxlength="255" /></el-form-item>
              <el-form-item label="角标文案"><el-input v-model="form.badge_text" maxlength="64" placeholder="例如：最受欢迎、限时优惠" /></el-form-item>
              <el-form-item label="默认勾选协议"><el-switch v-model="form.agreement_default_checked" /></el-form-item>
              <el-form-item label="是否订阅"><el-switch v-model="form.is_subscription" /></el-form-item>
              <el-form-item label="显示模式"><el-radio-group v-model="form.display_mode"><el-radio :value="1">正常显示</el-radio><el-radio :value="0">隐藏</el-radio></el-radio-group></el-form-item>
              <el-form-item label="状态"><el-radio-group v-model="form.status"><el-radio :value="1">启用</el-radio><el-radio :value="0">禁用</el-radio></el-radio-group></el-form-item>
              <el-form-item label="默认套餐"><el-switch v-model="form.is_default" /><span class="form-tip">同一安装包和套餐类型仅保留一个默认套餐</span></el-form-item>
            </div>
            <el-form-item label="内部备注"><el-input v-model="form.remark" type="textarea" :rows="4" maxlength="1000" show-word-limit /></el-form-item>
          </section>
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
import { computed, nextTick, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox, type FormInstance, type FormRules } from 'element-plus'
import { getBannerDeliveryOptions, type BannerDeliveryApp, type BannerDeliveryPackage, type BannerDeliveryVersion } from '@/api/banner'
import { getChannelOptions, type Channel } from '@/api/channel'
import { getCountryOptions, type Country } from '@/api/country'
import type { AppPackage } from '@/api/package'
import type { PackageVersion } from '@/api/packageVersion'
import type { VideoApp } from '@/api/videoApp'
import {
  cloneVIPSubscription,
  createVIPSubscription,
  deleteVIPSubscription,
  getVIPSubscriptionList,
  setDefaultVIPSubscription,
  updateVIPSubscription,
  updateVIPSubscriptionDisplay,
  updateVIPSubscriptionStatus,
  type VIPSubscription,
  type VIPSubscriptionPayload,
  type VIPSubscriptionPeriod,
} from '@/api/vipSubscription'
import { getVIPSubscriptionLevelOptions, type VIPSubscriptionLevel } from '@/api/vipSubscriptionLevel'
import { useUserStore } from '@/store/user'

type TargetMode = 'all' | 'selected'
type VIPSubscriptionForm = VIPSubscriptionPayload & { id: number }

const vipTypeOptions = [
  { value: 1, label: 'OB' },
  { value: 2, label: 'OB 拦截' },
  { value: 3, label: '老用户启动' },
  { value: 4, label: '老用户返回拦截' },
  { value: 5, label: '默认付费页' },
  { value: 6, label: '默认付费页拦截' },
  { value: 7, label: '卸载拦截' },
  { value: 8, label: '默认订阅套餐界面' },
]
const subscriptionPeriodOptions: Array<{ value: VIPSubscriptionPeriod; label: string }> = [
  { value: 1, label: '周' },
  { value: 2, label: '月' },
  { value: 3, label: '季' },
  { value: 4, label: '年' },
]

const userStore = useUserStore()
const canAdd = computed(() => userStore.hasPermission('subscription:vip:add'))
const canEdit = computed(() => userStore.hasPermission('subscription:vip:edit'))
const canDelete = computed(() => userStore.hasPermission('subscription:vip:delete'))
const loading = ref(false)
const submitting = ref(false)
const dialogVisible = ref(false)
const updatingIds = ref<number[]>([])
const formRef = ref<FormInstance>()
const tableData = ref<VIPSubscription[]>([])
const levelOptions = ref<VIPSubscriptionLevel[]>([])
const countryOptions = ref<Country[]>([])
const channelOptions = ref<Channel[]>([])
const deliveryOptions = ref<BannerDeliveryApp[]>([])
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const query = reactive({ vip_type: '', level_id: '', app_code: '', package_code: '', version_code: '', status: '', display_mode: '', is_subscription: '', keyword: '' })
const targetModes = reactive<Record<'countries' | 'apps' | 'packages' | 'versions' | 'channels', TargetMode>>({
  countries: 'all', apps: 'selected', packages: 'selected', versions: 'all', channels: 'all',
})

function createDefaultForm(): VIPSubscriptionForm {
  return {
    id: 0,
    app_codes: [], package_codes: [], version_codes: [], country_codes: [], channel_codes: [],
    level_id: 0, vip_type: 1, suk_code: '', name: '', currency: 'USD',
    first_subscription_price: 0, first_subscription_revenue: 0, first_bonus_points: 0, original_price: 0,
    vip_duration_days: 30, trial_days: 0, renewal_text: '', badge_text: '', agreement_default_checked: true,
    display_mode: 1, status: 1, free_trial: false, is_subscription: true, is_default: false,
    subscription_description: '', subscription_price: 0, subscription_revenue: 0, subscription_points: 0,
    subscription_period: 2, sort: 0, description: '', remark: '',
  }
}
const form = reactive<VIPSubscriptionForm>(createDefaultForm())
const rules: FormRules = {
  suk_code: [{ required: true, message: '请输入产品 SKU', trigger: 'blur' }],
  name: [{ required: true, message: '请输入 VIP 名称', trigger: 'blur' }],
  level_id: [{ required: true, message: '请选择 VIP 等级', trigger: 'change' }],
  vip_type: [{ required: true, message: '请选择套餐类型', trigger: 'change' }],
  package_codes: [{ type: 'array', required: true, min: 1, message: '请至少选择一个安装包', trigger: 'change' }],
  currency: [{ required: true, pattern: /^[A-Za-z]{3}$/, message: '请输入三位币种代码', trigger: 'blur' }],
  subscription_period: [{ required: true, message: '请选择订阅周期', trigger: 'change' }],
}

const filterPackageOptions = computed<BannerDeliveryPackage[]>(() => {
  if (!query.app_code) return deliveryOptions.value.flatMap((item) => item.packages)
  return deliveryOptions.value.find((item) => item.app_code === query.app_code)?.packages || []
})
const filterVersionOptions = computed<BannerDeliveryVersion[]>(() => {
  if (!query.package_code) return []
  return filterPackageOptions.value.find((item) => item.package_code === query.package_code)?.versions || []
})
const formPackageOptions = computed<BannerDeliveryPackage[]>(() => {
  if (targetModes.apps === 'all') return deliveryOptions.value.flatMap((item) => item.packages)
  const selected = new Set(form.app_codes)
  return deliveryOptions.value.filter((item) => selected.has(item.app_code)).flatMap((item) => item.packages)
})
const formVersionOptions = computed(() => {
  const selectedPackages = new Set(form.package_codes)
  const grouped = new Map<string, string[]>()
  for (const item of formPackageOptions.value) {
    if (!selectedPackages.has(item.package_code)) continue
    for (const version of item.versions) {
      const labels = grouped.get(version.version_code) || []
      labels.push(item.package_name || item.package_code)
      grouped.set(version.version_code, labels)
    }
  }
  return [...grouped.entries()].map(([version_code, packages]) => ({
    version_code,
    label: `${version_code} · ${packages.join('、')}`,
  }))
})

function arrayValue<T>(value: T[] | null | undefined): T[] { return Array.isArray(value) ? value : [] }
function booleanValue(value: unknown) { return value === true || value === 1 }
function normalizePeriod(value: unknown): VIPSubscriptionPeriod {
  if (value === 1 || value === 2 || value === 3 || value === 4) return value
  const legacy: Record<string, VIPSubscriptionPeriod> = { P1W: 1, P1M: 2, P3M: 3, P1Y: 4 }
  return legacy[String(value)] || 2
}
function appLabel(item: BannerDeliveryApp) { return `${item.app_name} · ${item.app_code}` }
function packageLabel(item: BannerDeliveryPackage) { return `${item.package_name} · ${item.package_code}` }
function countryLabel(item: Country) { return `${item.name_zh} · ${item.code}` }
function channelLabel(item: Channel) { return `${item.channel_name} · ${item.channel_code}` }
function levelOptionLabel(item: VIPSubscriptionLevel) { return item.description ? `${item.level} · ${item.description}` : item.level }
function compactSummary(labels: string[], allLabel: string) {
  if (!labels.length) return allLabel
  return labels.length > 2 ? `${labels.slice(0, 2).join('、')} 等 ${labels.length} 项` : labels.join('、')
}
function appSummary(items?: VideoApp[]) { return compactSummary(arrayValue(items).map((item) => item.name || item.app_code), '全部应用') }
function packageSummary(items?: AppPackage[]) { return compactSummary(arrayValue(items).map((item) => item.package_name || item.package_code), '未关联安装包') }
function versionSummary(items?: PackageVersion[]) { return compactSummary(arrayValue(items).map((item) => item.version_code), '全部版本') }
function countrySummary(items?: Country[]) { return compactSummary(arrayValue(items).map((item) => item.name_zh), '全部国家') }
function channelSummary(items?: Channel[]) { return compactSummary(arrayValue(items).map((item) => item.channel_name), '全部渠道') }
function vipTypeLabel(value: number) { return vipTypeOptions.find((item) => item.value === Number(value))?.label || `类型 ${value}` }
function levelLabel(row: VIPSubscription) { return row.subscription_level?.level || levelOptions.value.find((item) => item.id === row.level_id)?.level || `等级 #${row.level_id}` }
function subscriptionPeriodLabel(value: number) { return subscriptionPeriodOptions.find((item) => item.value === Number(value))?.label || '-' }
function formatMoney(value: number) { return Number(value || 0).toFixed(2) }

async function fetchData() {
  loading.value = true
  try {
    const params: Record<string, unknown> = { page: page.value, page_size: pageSize.value }
    Object.entries(query).forEach(([key, value]) => { if (value !== '') params[key] = value })
    const res: any = await getVIPSubscriptionList(params)
    tableData.value = arrayValue<VIPSubscription>(res.data?.list)
    total.value = Number(res.data?.total) || 0
  } finally {
    loading.value = false
  }
}
function handleSearch() { page.value = 1; void fetchData() }
function handleReset() {
  Object.assign(query, { vip_type: '', level_id: '', app_code: '', package_code: '', version_code: '', status: '', display_mode: '', is_subscription: '', keyword: '' })
  page.value = 1
  void fetchData()
}
function handlePageSizeChange() { page.value = 1; void fetchData() }
function handleFilterAppChange() { query.package_code = ''; query.version_code = '' }
function handleFilterPackageChange() { query.version_code = '' }

function openCreate() {
  Object.assign(form, createDefaultForm(), { level_id: levelOptions.value[0]?.id || 0 })
  Object.assign(targetModes, { countries: 'all', apps: 'selected', packages: 'selected', versions: 'all', channels: 'all' })
  dialogVisible.value = true
}
function ensureCurrentLevelOption(row: VIPSubscription) {
  const current = row.subscription_level
  if (current?.id && !levelOptions.value.some((item) => item.id === current.id)) levelOptions.value.push(current)
}
function openEdit(row: VIPSubscription) {
  ensureCurrentLevelOption(row)
  const packageCodes = arrayValue(row.package_codes).length ? [...row.package_codes] : arrayValue(row.packages).map((item) => item.package_code)
  let appCodes = arrayValue(row.app_codes).length ? [...row.app_codes] : arrayValue(row.apps).map((item) => item.app_code)
  if (!appCodes.length && packageCodes.length) {
    const selectedPackages = new Set(packageCodes)
    appCodes = deliveryOptions.value
      .filter((item) => item.packages.some((appPackage) => selectedPackages.has(appPackage.package_code)))
      .map((item) => item.app_code)
  }
  const versionCodes = arrayValue(row.version_codes).length ? [...row.version_codes] : arrayValue(row.package_version).map((item) => item.version_code)
  const countryCodes = arrayValue(row.country_codes).length ? [...row.country_codes] : arrayValue(row.country).map((item) => item.code)
  const channelCodes = arrayValue(row.channel_codes).length ? [...row.channel_codes] : arrayValue(row.channels).map((item) => item.channel_code)
  Object.assign(form, {
    id: row.id,
    app_codes: appCodes,
    package_codes: packageCodes,
    version_codes: versionCodes,
    country_codes: countryCodes,
    channel_codes: channelCodes,
    level_id: Number(row.level_id || row.subscription_level?.id || 0),
    vip_type: Number(row.vip_type) || 1,
    suk_code: row.suk_code || '',
    name: row.name || '',
    currency: row.currency || 'USD',
    first_subscription_price: Number(row.first_subscription_price) || 0,
    first_subscription_revenue: Number(row.first_subscription_revenue) || 0,
    first_bonus_points: Number(row.first_bonus_points) || 0,
    original_price: Number(row.original_price) || 0,
    vip_duration_days: Number(row.vip_duration_days) || 0,
    trial_days: Number(row.trial_days) || 0,
    renewal_text: row.renewal_text || '',
    badge_text: row.badge_text || '',
    agreement_default_checked: booleanValue(row.agreement_default_checked),
    display_mode: Number(row.display_mode) === 0 ? 0 : 1,
    status: Number(row.status) === 0 ? 0 : 1,
    free_trial: booleanValue(row.free_trial),
    is_subscription: booleanValue(row.is_subscription),
    is_default: booleanValue(row.is_default),
    subscription_description: row.subscription_description || '',
    subscription_price: Number(row.subscription_price) || 0,
    subscription_revenue: Number(row.subscription_revenue) || 0,
    subscription_points: Number(row.subscription_points) || 0,
    subscription_period: normalizePeriod(row.subscription_period),
    sort: Number(row.sort) || 0,
    description: row.description || '',
    remark: row.remark || '',
  })
  Object.assign(targetModes, {
    countries: countryCodes.length ? 'selected' : 'all',
    apps: appCodes.length ? 'selected' : 'all',
    packages: 'selected',
    versions: versionCodes.length ? 'selected' : 'all',
    channels: channelCodes.length ? 'selected' : 'all',
  })
  const availablePackages = formPackageOptions.value.map((item) => item.package_code)
  if (availablePackages.length && availablePackages.every((code) => packageCodes.includes(code)) && packageCodes.length === availablePackages.length) {
    targetModes.packages = 'all'
  }
  dialogVisible.value = true
}
function handleCountryModeChange(value: string | number | boolean | undefined) { if (value === 'all') form.country_codes = [] }
function handleChannelModeChange(value: string | number | boolean | undefined) { if (value === 'all') form.channel_codes = [] }
function handleAppModeChange(value: string | number | boolean | undefined) {
  if (value === 'all') form.app_codes = []
  handleAppSelectionChange()
}
function handlePackageModeChange(value: string | number | boolean | undefined) {
  if (value === 'all') syncAllPackageCodes()
  handlePackageSelectionChange()
  clearPackageValidation()
}
function handleVersionModeChange(value: string | number | boolean | undefined) { if (value === 'all') form.version_codes = [] }
function syncAllPackageCodes() {
  if (targetModes.packages !== 'all') return
  form.package_codes = [...new Set(formPackageOptions.value.map((item) => item.package_code))]
}
function clearPackageValidation() {
  void nextTick(() => {
    if (form.package_codes.length) formRef.value?.clearValidate('package_codes')
  })
}
function handleAppSelectionChange() {
  const allowed = new Set(formPackageOptions.value.map((item) => item.package_code))
  if (targetModes.packages === 'all') syncAllPackageCodes()
  else form.package_codes = form.package_codes.filter((code) => allowed.has(code))
  handlePackageSelectionChange()
  clearPackageValidation()
}
function handlePackageSelectionChange() {
  const allowed = new Set(formVersionOptions.value.map((item) => item.version_code))
  form.version_codes = form.version_codes.filter((code) => allowed.has(code))
  if (!form.package_codes.length) {
    targetModes.versions = 'all'
    form.version_codes = []
  }
}

async function handleSubmit() {
  syncAllPackageCodes()
  await nextTick()
  await formRef.value?.validate()
  if (targetModes.countries === 'selected' && !form.country_codes.length) return void ElMessage.warning('请选择国家，或切换为全部国家')
  if (targetModes.apps === 'selected' && !form.app_codes.length) return void ElMessage.warning('请选择应用，或切换为全部应用')
  if (!form.package_codes.length) return void ElMessage.warning('请至少选择一个安装包')
  if (targetModes.versions === 'selected' && !form.version_codes.length) return void ElMessage.warning('请选择版本，或切换为全部版本')
  if (targetModes.channels === 'selected' && !form.channel_codes.length) return void ElMessage.warning('请选择渠道，或切换为全部渠道')
  if (form.free_trial && form.trial_days <= 0) return void ElMessage.warning('开启免费体验时，试用天数必须大于 0')
  submitting.value = true
  try {
    const payload: VIPSubscriptionPayload = {
      app_codes: targetModes.apps === 'all' ? [] : [...form.app_codes],
      package_codes: [...form.package_codes],
      version_codes: targetModes.versions === 'all' ? [] : [...form.version_codes],
      country_codes: targetModes.countries === 'all' ? [] : [...form.country_codes],
      channel_codes: targetModes.channels === 'all' ? [] : [...form.channel_codes],
      level_id: form.level_id,
      vip_type: form.vip_type,
      suk_code: form.suk_code.trim(),
      name: form.name.trim(),
      currency: form.currency.trim().toUpperCase(),
      first_subscription_price: form.first_subscription_price,
      first_subscription_revenue: form.first_subscription_revenue,
      first_bonus_points: form.first_bonus_points,
      original_price: form.original_price,
      vip_duration_days: form.vip_duration_days,
      trial_days: form.trial_days,
      renewal_text: form.renewal_text.trim(),
      badge_text: form.badge_text.trim(),
      agreement_default_checked: form.agreement_default_checked,
      display_mode: form.display_mode,
      status: form.status,
      free_trial: form.free_trial,
      is_subscription: form.is_subscription,
      is_default: form.is_default,
      subscription_description: form.subscription_description.trim(),
      subscription_price: form.subscription_price,
      subscription_revenue: form.subscription_revenue,
      subscription_points: form.subscription_points,
      subscription_period: form.subscription_period,
      sort: form.sort,
      description: form.description.trim(),
      remark: form.remark.trim(),
    }
    if (form.id) await updateVIPSubscription(form.id, payload)
    else await createVIPSubscription(payload)
    ElMessage.success('VIP 订阅套餐已保存')
    dialogVisible.value = false
    await fetchData()
  } finally {
    submitting.value = false
  }
}
async function handleStatusChange(row: VIPSubscription) {
  updatingIds.value.push(row.id)
  try {
    await updateVIPSubscriptionStatus(row.id, row.status)
    ElMessage.success(`套餐已${row.status === 1 ? '启用' : '禁用'}`)
  } catch {
    row.status = row.status === 1 ? 0 : 1
  } finally {
    updatingIds.value = updatingIds.value.filter((id) => id !== row.id)
  }
}
async function toggleDisplay(row: VIPSubscription) {
  const mode = row.display_mode === 1 ? 0 : 1
  await updateVIPSubscriptionDisplay(row.id, mode)
  row.display_mode = mode
  ElMessage.success(mode === 1 ? '套餐已显示' : '套餐已隐藏')
}
async function setDefault(row: VIPSubscription) {
  await setDefaultVIPSubscription(row.id)
  ElMessage.success('已设为默认套餐')
  await fetchData()
}
async function clonePlan(row: VIPSubscription) {
  const { value } = await ElMessageBox.prompt('请输入复制后套餐的新产品 SKU', '复制 VIP 套餐', {
    inputPlaceholder: `${row.suk_code}_copy`, inputPattern: /\S+/, inputErrorMessage: '产品 SKU 不能为空',
  })
  await cloneVIPSubscription(row.id, value.trim())
  ElMessage.success('套餐已复制')
  await fetchData()
}
async function handleDelete(id: number) {
  await deleteVIPSubscription(id)
  ElMessage.success('VIP 套餐已删除')
  if (tableData.value.length === 1 && page.value > 1) page.value--
  await fetchData()
}
async function fetchOptions() {
  const [levelRes, countryRes, channelRes, deliveryRes]: any[] = await Promise.all([
    getVIPSubscriptionLevelOptions(), getCountryOptions(), getChannelOptions(), getBannerDeliveryOptions(),
  ])
  levelOptions.value = arrayValue(levelRes.data)
  countryOptions.value = arrayValue(countryRes.data)
  channelOptions.value = arrayValue(channelRes.data)
  deliveryOptions.value = arrayValue(deliveryRes.data)
  syncAllPackageCodes()
  clearPackageValidation()
}
onMounted(() => Promise.all([fetchOptions(), fetchData()]))
</script>

<style scoped>
.page-wrap { min-width: 0; }
.page-header { display: flex; align-items: center; justify-content: space-between; gap: 16px; }
.page-title { color: #303133; font-size: 17px; font-weight: 600; }
.page-subtitle { margin-top: 4px; color: #909399; font-size: 12px; }
.filters { display: grid; grid-template-columns: repeat(5, minmax(130px, 1fr)) auto auto; gap: 10px; margin-bottom: 16px; }
.primary-text { color: #303133; font-weight: 600; }
.secondary-text { margin-top: 4px; color: #909399; font-size: 12px; }
.mono { font-family: ui-monospace, SFMono-Regular, Menlo, monospace; }
.tag-row, .target-tags { display: flex; flex-wrap: wrap; gap: 5px; margin-top: 6px; }
.price { color: #f56c6c; font-size: 15px; font-weight: 600; }
.original-price { color: #909399; font-size: 12px; text-decoration: line-through; }
.display-state { margin-top: 6px; color: #909399; font-size: 12px; }
.pagination-wrap { display: flex; justify-content: flex-end; margin-top: 16px; overflow-x: auto; }
.form-sections { display: flex; flex-direction: column; gap: 18px; }
.form-section { padding: 18px 20px 2px; border: 1px solid #e4e7ed; border-radius: 8px; background: #fff; }
.section-heading { display: flex; align-items: baseline; gap: 12px; margin-bottom: 10px; }
.section-title { color: #303133; font-size: 16px; font-weight: 600; }
.section-description { color: #909399; font-size: 12px; }
.form-grid { display: grid; grid-template-columns: 1fr 1fr; column-gap: 18px; padding-top: 8px; }
.form-grid :deep(.el-input-number) { width: 100%; }
.form-tip { margin-left: 10px; color: #909399; font-size: 12px; }
.scope-field { display: flex; width: 100%; flex-direction: column; align-items: flex-start; gap: 10px; }
.scope-tip { color: #909399; font-size: 12px; line-height: 1.5; }
:global(.subscription-edit-dialog) { max-width: calc(100vw - 32px); }
:global(.subscription-edit-dialog .el-dialog__body) { max-height: calc(92vh - 132px); overflow-y: auto; }
@media (max-width: 1200px) { .filters { grid-template-columns: repeat(4, minmax(140px, 1fr)); } }
@media (max-width: 700px) {
  .page-header { align-items: stretch; flex-direction: column; }
  .filters, .form-grid { grid-template-columns: 1fr; }
  .form-section { padding: 14px 12px 0; }
  .section-heading { align-items: flex-start; flex-direction: column; gap: 4px; }
  .page-wrap :deep(.el-card__header), .page-wrap :deep(.el-card__body) { padding: 14px; }
}
</style>
