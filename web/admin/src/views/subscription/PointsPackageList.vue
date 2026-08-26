<template>
  <div class="page-wrap">
    <el-card shadow="never">
      <template #header>
        <div class="page-header">
          <div>
            <div class="page-title">积分套餐</div>
            <div class="page-subtitle">管理积分商品及其应用、安装包、版本、国家和渠道投放范围</div>
          </div>
          <el-button v-if="canAdd" type="primary" @click="openCreate">
            <el-icon><Plus /></el-icon>新增套餐
          </el-button>
        </div>
      </template>

      <div class="filters primary-filters">
        <el-select v-model="query.resource_type" clearable filterable allow-create placeholder="资源类型">
          <el-option v-for="item in resourceTypeOptions" :key="item.value" :label="item.label" :value="item.value" />
        </el-select>
        <el-select v-model="query.status" clearable placeholder="状态">
          <el-option label="显示" value="1" />
          <el-option label="隐藏" value="0" />
        </el-select>
        <el-select v-model="query.app_code" clearable filterable placeholder="应用" @change="handleFilterAppChange">
          <el-option v-for="item in deliveryOptions" :key="item.app_code" :label="appLabel(item)" :value="item.app_code" />
        </el-select>
        <el-select v-model="query.package_code" clearable filterable placeholder="安装包" @change="query.version_code = ''">
          <el-option v-for="item in filterPackageOptions" :key="item.package_code" :label="packageLabel(item)" :value="item.package_code" />
        </el-select>
        <el-input v-model="query.keyword" clearable placeholder="产品 ID、名称或描述" @keyup.enter="handleSearch">
          <template #prefix><el-icon><Search /></el-icon></template>
        </el-input>
      </div>
      <el-collapse-transition>
        <div v-show="advancedSearchVisible" class="filters advanced-filters">
          <el-select v-model="query.version_code" clearable filterable placeholder="版本" :disabled="!query.package_code">
            <el-option v-for="item in filterVersionOptions" :key="item.version_code" :label="item.version_code" :value="item.version_code" />
          </el-select>
          <el-select v-model="query.country_code" clearable filterable placeholder="国家">
            <el-option v-for="item in countryOptions" :key="item.code" :label="countryLabel(item)" :value="item.code" />
          </el-select>
          <el-select v-model="query.channel_code" clearable filterable placeholder="渠道">
            <el-option v-for="item in channelOptions" :key="item.channel_code" :label="channelLabel(item)" :value="item.channel_code" />
          </el-select>
          <el-select v-model="query.system" clearable placeholder="系统">
            <el-option v-for="item in allSystemOptions" :key="item" :label="systemLabel(item)" :value="item" />
          </el-select>
          <el-select v-model="query.user_type" clearable placeholder="用户类型">
            <el-option label="免费用户" value="1" />
            <el-option label="付费用户" value="2" />
          </el-select>
        </div>
      </el-collapse-transition>
      <div class="filter-actions">
        <el-button type="primary" plain @click="handleSearch">查询</el-button>
        <el-button @click="handleReset">重置</el-button>
        <el-button link type="primary" @click="advancedSearchVisible = !advancedSearchVisible">
          {{ advancedSearchVisible ? '收起筛选' : '更多筛选' }}
          <el-icon class="toggle-icon"><ArrowUp v-if="advancedSearchVisible" /><ArrowDown v-else /></el-icon>
        </el-button>
      </div>

      <el-table v-loading="loading" :data="tableData" row-key="id" stripe @sort-change="handleSortChange">
        <el-table-column prop="id" label="ID" width="76" sortable="custom" />
        <el-table-column prop="sort" label="排序" width="76" align="center" sortable="custom" />
        <el-table-column label="默认" width="90" align="center">
          <template #default="{ row }">
            <el-tag v-if="row.is_default" type="success">是</el-tag>
            <el-button v-else-if="canEdit" link type="primary" @click="handleSetDefault(row.id)">设为默认</el-button>
            <span v-else>否</span>
          </template>
        </el-table-column>
        <el-table-column label="积分商品" min-width="220">
          <template #default="{ row }">
            <div class="primary-text">{{ row.name }}</div>
            <code class="product-id">{{ row.product_code }}</code>
          </template>
        </el-table-column>
        <el-table-column label="投放范围" min-width="300">
          <template #default="{ row }">
            <div class="scope-summary"><span>应用</span>{{ targetSummary(row.apps, 'name', '全部应用') }}</div>
            <div class="scope-summary"><span>安装包</span>{{ targetSummary(row.packages, 'package_name', '全部安装包') }}</div>
            <div class="scope-summary"><span>版本</span>{{ targetSummary(row.package_version, 'version_code', '全部版本') }}</div>
            <div class="scope-summary"><span>地区/渠道</span>{{ countryChannelSummary(row) }}</div>
          </template>
        </el-table-column>
        <el-table-column label="系统 / 用户" min-width="170">
          <template #default="{ row }">
            <div class="tag-list">
              <el-tag v-for="item in row.systems" :key="item" size="small" effect="plain">{{ systemLabel(item) }}</el-tag>
            </div>
            <div class="secondary-text">{{ userTypesSummary(row.user_types) }}</div>
          </template>
        </el-table-column>
        <el-table-column label="价格" width="145" align="right">
          <template #default="{ row }">
            <div>{{ money(row.sale_price, row.currency) }}</div>
            <div class="secondary-text">收入 {{ money(row.actual_revenue, row.currency) }}</div>
          </template>
        </el-table-column>
        <el-table-column label="赠送积分" width="145" align="right">
          <template #default="{ row }">
            <strong class="points-value">{{ formatNumber(row.points) }}</strong>
            <div class="secondary-text">{{ resourceTypeLabel(row.resource_type) }}</div>
          </template>
        </el-table-column>
        <el-table-column prop="icon" label="角标文案" min-width="140" align="center">
          <template #default="{ row }">
            <span>{{ row.icon || '-' }}</span>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="90" align="center" fixed="right">
          <template #default="{ row }">
            <el-switch v-if="canEdit" v-model="row.status" :active-value="1" :inactive-value="0" inline-prompt active-text="显" inactive-text="隐" @change="handleStatus(row)" />
            <el-tag v-else :type="row.status === 1 ? 'success' : 'info'">{{ row.status === 1 ? '显示' : '隐藏' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column v-if="canEdit || canDelete" label="操作" width="130" fixed="right" align="center">
          <template #default="{ row }">
            <el-button v-if="canEdit" link type="primary" @click="openEdit(row)">编辑</el-button>
            <el-popconfirm v-if="canDelete" :title="`确认删除 ${row.name}？`" @confirm="handleDelete(row.id)">
              <template #reference><el-button link type="danger">删除</el-button></template>
            </el-popconfirm>
          </template>
        </el-table-column>
      </el-table>

      <div class="pagination-wrap">
        <el-pagination v-model:current-page="page" v-model:page-size="pageSize" :total="total" :page-sizes="[10, 20, 50, 100]" layout="total, sizes, prev, pager, next" @size-change="handlePageSizeChange" @current-change="fetchData" />
      </div>
    </el-card>

    <el-dialog v-model="dialogVisible" :title="form.id ? '编辑积分套餐' : '新增积分套餐'" width="920px" destroy-on-close>
      <el-form ref="formRef" :model="form" :rules="rules" label-width="105px">
        <div class="form-grid">
          <el-form-item label="产品 ID" prop="product_code"><el-input v-model="form.product_code" :disabled="form.id > 0" maxlength="191" placeholder="例如：premium_credits_plan" /></el-form-item>
          <el-form-item label="积分名称" prop="name"><el-input v-model="form.name" maxlength="128" /></el-form-item>
          <el-form-item label="系统" prop="systems" class="full-grid-item">
            <el-checkbox-group v-model="form.systems">
              <el-checkbox-button v-for="item in allSystemOptions" :key="item" :value="item">{{ systemLabel(item) }}</el-checkbox-button>
            </el-checkbox-group>
          </el-form-item>
          <el-form-item label="用户类型" prop="user_types"><el-select v-model="form.user_types" multiple style="width: 100%"><el-option label="免费用户" :value="1" /><el-option label="付费用户" :value="2" /></el-select></el-form-item>
          <el-form-item label="资源类型" prop="resource_type"><el-select v-model="form.resource_type" filterable allow-create style="width: 100%"><el-option v-for="item in resourceTypeOptions" :key="item.value" :label="item.label" :value="item.value" /></el-select></el-form-item>
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
            <div class="scope-tip">全部国家不写入国家关联记录。</div>
          </div>
        </el-form-item>
        <el-form-item label="应用范围">
          <div class="scope-field">
            <el-radio-group v-model="targetModes.apps" @change="handleAppModeChange">
              <el-radio-button value="all">全部应用</el-radio-button>
              <el-radio-button value="selected">指定应用</el-radio-button>
            </el-radio-group>
            <el-select v-if="targetModes.apps === 'selected'" v-model="form.app_codes" multiple collapse-tags collapse-tags-tooltip clearable filterable placeholder="请选择应用" style="width: 100%" @change="handleAppSelectionChange">
              <el-option v-for="item in deliveryOptions" :key="item.app_code" :label="appLabel(item)" :value="item.app_code" />
            </el-select>
            <div class="scope-tip">指定应用后，安装包会按应用联动过滤。</div>
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
            <div class="scope-tip">全部安装包不写入安装包和版本关联记录。</div>
          </div>
        </el-form-item>
        <el-form-item v-if="(targetModes.apps === 'all' || form.app_codes.length) && (targetModes.packages === 'all' || form.package_codes.length)" label="版本">
          <div class="scope-field">
            <el-radio-group v-model="targetModes.versions" @change="handleVersionModeChange">
              <el-radio-button value="all">全部版本</el-radio-button>
              <el-radio-button value="selected" :disabled="targetModes.packages === 'all'">指定版本</el-radio-button>
            </el-radio-group>
            <el-select v-if="targetModes.versions === 'selected'" v-model="form.version_codes" multiple collapse-tags collapse-tags-tooltip clearable filterable placeholder="请选择版本" style="width: 100%">
              <el-option v-for="item in formVersionOptions" :key="item.version_code" :label="item.label" :value="item.version_code" />
            </el-select>
            <div class="scope-tip">全部版本不写入版本关联记录。</div>
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
            <div class="scope-tip">全部渠道不写入渠道关联记录。</div>
          </div>
        </el-form-item>

        <el-divider content-position="left">商品配置</el-divider>
        <div class="form-grid">
          <el-form-item label="赠送积分" prop="points"><el-input-number v-model="form.points" :min="1" :max="999999999999" controls-position="right" /></el-form-item>
          <el-form-item label="币种" prop="currency"><el-input v-model="form.currency" maxlength="3" @input="form.currency = form.currency.toUpperCase()" /></el-form-item>
          <el-form-item label="销售金额" prop="sale_price"><el-input-number v-model="form.sale_price" :min="0" :max="9999999999.99" :precision="2" controls-position="right" /></el-form-item>
          <el-form-item label="实际收入" prop="actual_revenue"><el-input-number v-model="form.actual_revenue" :min="0" :max="9999999999.99" :precision="2" controls-position="right" /></el-form-item>
          <el-form-item label="划线价" prop="original_price"><el-input-number v-model="form.original_price" :min="0" :max="9999999999.99" :precision="2" controls-position="right" /></el-form-item>
          <el-form-item label="角标文案"><el-input v-model="form.icon" maxlength="100" placeholder="例如：限时优惠、最受欢迎" /></el-form-item>
          <el-form-item label="按钮文案"><el-input v-model="form.button_text" maxlength="128" placeholder="例如：获取更多积分" /></el-form-item>
          <el-form-item label="排序"><el-input-number v-model="form.sort" :min="0" :max="999999" controls-position="right" /></el-form-item>
          <el-form-item label="是否默认"><el-switch v-model="form.is_default" active-text="是" inactive-text="否" /></el-form-item>
          <el-form-item label="状态"><el-switch v-model="form.status" :active-value="1" :inactive-value="0" active-text="显示" inactive-text="隐藏" /></el-form-item>
        </div>
        <el-form-item label="描述"><el-input v-model="form.description" type="textarea" :rows="3" maxlength="1000" show-word-limit /></el-form-item>
      </el-form>
      <template #footer><el-button @click="dialogVisible = false">取消</el-button><el-button type="primary" :loading="submitting" @click="handleSubmit">保存</el-button></template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, type FormInstance, type FormRules } from 'element-plus'
import { getBannerDeliveryOptions, type BannerDeliveryApp, type BannerDeliveryPackage, type BannerDeliveryVersion } from '@/api/banner'
import { getChannelOptions, type Channel } from '@/api/channel'
import { getCountryOptions, type Country } from '@/api/country'
import {
  createPointsPackage,
  deletePointsPackage,
  getPointsPackageList,
  setDefaultPointsPackage,
  updatePointsPackage,
  updatePointsPackageStatus,
  type Points,
  type PointsPackagePayload,
} from '@/api/points'
import { useUserStore } from '@/store/user'
import { useRemoteTableSort } from '@/utils/tableSort'

type TargetMode = 'all' | 'selected'
type PointsForm = PointsPackagePayload & { id: number }

const userStore = useUserStore()
const canAdd = computed(() => userStore.hasPermission('subscription:points:add'))
const canEdit = computed(() => userStore.hasPermission('subscription:points:edit'))
const canDelete = computed(() => userStore.hasPermission('subscription:points:delete'))
const allSystemOptions = ['android', 'ios', 'pc', 'harmony', 'web', 'other']
const resourceTypeOptions = [{ value: 'credits', label: '积分包' }, { value: 'word_pack', label: '字数包' }, { value: 'image_pack', label: '图片包' }]
const countryOptions = ref<Country[]>([])
const channelOptions = ref<Channel[]>([])
const deliveryOptions = ref<BannerDeliveryApp[]>([])
const tableData = ref<Points[]>([])
const loading = ref(false)
const submitting = ref(false)
const dialogVisible = ref(false)
const advancedSearchVisible = ref(false)
const formRef = ref<FormInstance>()
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const { sortParams, handleSortChange } = useRemoteTableSort(page, fetchData)
const query = reactive({ app_code: '', package_code: '', version_code: '', country_code: '', channel_code: '', system: '', user_type: '', resource_type: '', status: '', keyword: '' })
const targetModes = reactive<Record<'countries' | 'apps' | 'packages' | 'versions' | 'channels', TargetMode>>({
  countries: 'all', apps: 'all', packages: 'all', versions: 'all', channels: 'all',
})

function createDefaultForm(): PointsForm {
  return {
    id: 0, product_code: '', name: '', app_codes: [], package_codes: [], version_codes: [], country_codes: [], channel_codes: [],
    systems: ['ios'], user_types: [1, 2], resource_type: 'credits', points: 1, currency: 'USD', sale_price: 0,
    actual_revenue: 0, original_price: 0, icon: '', description: '', button_text: '获取积分', is_default: false, status: 1, sort: 0,
  }
}
const form = reactive<PointsForm>(createDefaultForm())
const rules: FormRules = {
  product_code: [{ required: true, message: '请输入产品 ID', trigger: 'blur' }, { pattern: /^[A-Za-z0-9._-]+$/, message: '仅支持字母、数字、点、下划线和中划线', trigger: 'blur' }],
  name: [{ required: true, message: '请输入积分名称', trigger: 'blur' }],
  systems: [{ required: true, type: 'array', min: 1, message: '请至少选择一个系统', trigger: 'change' }],
  user_types: [{ required: true, type: 'array', min: 1, message: '请至少选择一种用户类型', trigger: 'change' }],
  resource_type: [{ required: true, message: '请选择资源类型', trigger: 'change' }],
  points: [{ required: true, type: 'number', min: 1, message: '赠送积分必须大于 0', trigger: 'change' }],
  currency: [{ required: true, pattern: /^[A-Za-z]{3}$/, message: '请输入 3 位币种代码', trigger: 'blur' }],
}

const filterPackageOptions = computed<BannerDeliveryPackage[]>(() => {
  if (!query.app_code) return uniquePackages(deliveryOptions.value.flatMap((item) => item.packages))
  return deliveryOptions.value.find((item) => item.app_code === query.app_code)?.packages || []
})
const filterVersionOptions = computed<BannerDeliveryVersion[]>(() => {
  if (!query.package_code) return []
  return filterPackageOptions.value.find((item) => item.package_code === query.package_code)?.versions || []
})
const formPackageOptions = computed<BannerDeliveryPackage[]>(() => {
  if (targetModes.apps === 'all') return uniquePackages(deliveryOptions.value.flatMap((item) => item.packages))
  const selected = new Set(form.app_codes)
  return uniquePackages(deliveryOptions.value.filter((item) => selected.has(item.app_code)).flatMap((item) => item.packages))
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
  return [...grouped.entries()].map(([version_code, packages]) => ({ version_code, label: `${version_code} · ${packages.join('、')}` }))
})

function arrayValue<T>(value: T[] | null | undefined): T[] { return Array.isArray(value) ? value : [] }
function uniquePackages(items: BannerDeliveryPackage[]) {
  const result = new Map<string, BannerDeliveryPackage>()
  for (const item of items) if (!result.has(item.package_code)) result.set(item.package_code, item)
  return [...result.values()]
}
function appLabel(item: BannerDeliveryApp) { return `${item.app_name} · ${item.app_code}` }
function packageLabel(item: BannerDeliveryPackage) { return `${item.package_name} · ${item.package_code}` }
function channelLabel(item: Channel) { return `${item.channel_name} · ${item.channel_code}` }
function countryLabel(item: Country) { return `${item.name_zh} · ${item.code}` }
function systemLabel(value: string) { return ({ android: 'Android', ios: 'iOS', pc: 'PC', harmony: 'HarmonyOS', web: 'Web', other: '其他' } as Record<string, string>)[value] || value }
function userTypeLabel(value: number) { return value === 1 ? '免费用户' : '付费用户' }
function resourceTypeLabel(value: string) { return resourceTypeOptions.find((item) => item.value === value)?.label || value }
function userTypesSummary(items?: number[] | null) { const values = arrayValue(items); return values.length ? values.map(userTypeLabel).join('、') : '全部用户' }
function formatNumber(value: number) { return Number(value || 0).toLocaleString('zh-CN') }
function money(value: number, currency: string) { return `${currency} ${Number(value || 0).toFixed(2)}` }
function targetSummary<T extends Record<string, any>>(items: T[] | null | undefined, field: keyof T, emptyText: string) {
  const values = arrayValue(items).map((item) => String(item[field] || '')).filter(Boolean)
  return values.length ? `${values.slice(0, 2).join('、')}${values.length > 2 ? ` 等 ${values.length} 项` : ''}` : emptyText
}
function countryChannelSummary(row: Points) {
  const countries = targetSummary(row.country, 'name_zh', '全部国家')
  const channels = targetSummary(row.channels, 'channel_name', '全部渠道')
  return `${countries} / ${channels}`
}

function normalizePoints(item: any): Points {
  return {
    ...item,
    app_codes: arrayValue<string>(item?.app_codes), package_codes: arrayValue<string>(item?.package_codes),
    version_codes: arrayValue<string>(item?.version_codes), country_codes: arrayValue<string>(item?.country_codes),
    channel_codes: arrayValue<string>(item?.channel_codes), systems: arrayValue<string>(item?.systems),
    user_types: arrayValue<number>(item?.user_types), apps: arrayValue(item?.apps), packages: arrayValue(item?.packages),
    package_version: arrayValue(item?.package_version), country: arrayValue(item?.country), channels: arrayValue(item?.channels),
    is_default: item?.is_default === true || item?.is_default === 1,
  }
}

async function fetchOptions() {
  const [countryRes, channelRes, deliveryRes]: any[] = await Promise.all([getCountryOptions(), getChannelOptions(), getBannerDeliveryOptions()])
  countryOptions.value = arrayValue(countryRes.data)
  channelOptions.value = arrayValue(channelRes.data)
  deliveryOptions.value = arrayValue(deliveryRes.data)
}
async function fetchData() {
  loading.value = true
  try {
    const params: Record<string, unknown> = { page: page.value, page_size: pageSize.value, ...sortParams() }
    for (const [key, value] of Object.entries(query)) if (value !== '') params[key] = typeof value === 'string' ? value.trim() : value
    const res: any = await getPointsPackageList(params)
    tableData.value = arrayValue<any>(res.data?.list).map(normalizePoints)
    total.value = Number(res.data?.total) || 0
  } finally { loading.value = false }
}
function handleSearch() { page.value = 1; fetchData() }
function handleReset() {
  Object.assign(query, { app_code: '', package_code: '', version_code: '', country_code: '', channel_code: '', system: '', user_type: '', resource_type: '', status: '', keyword: '' })
  page.value = 1
  fetchData()
}
function handlePageSizeChange() { page.value = 1; fetchData() }
function handleFilterAppChange() { query.package_code = ''; query.version_code = '' }

function openCreate() {
  Object.assign(form, createDefaultForm())
  Object.assign(targetModes, { countries: 'all', apps: 'all', packages: 'all', versions: 'all', channels: 'all' })
  dialogVisible.value = true
}
function openEdit(row: Points) {
  const appCodes = row.app_codes.length ? [...row.app_codes] : arrayValue(row.apps).map((item) => item.app_code)
  const packageCodes = row.package_codes.length ? [...row.package_codes] : arrayValue(row.packages).map((item) => item.package_code)
  const versionCodes = row.version_codes.length ? [...row.version_codes] : arrayValue(row.package_version).map((item) => item.version_code)
  const countryCodes = row.country_codes.length ? [...row.country_codes] : arrayValue(row.country).map((item) => item.code)
  const channelCodes = row.channel_codes.length ? [...row.channel_codes] : arrayValue(row.channels).map((item) => item.channel_code)
  Object.assign(form, {
    id: row.id, product_code: row.product_code, name: row.name, app_codes: appCodes, package_codes: packageCodes,
    version_codes: versionCodes, country_codes: countryCodes, channel_codes: channelCodes, systems: [...row.systems],
    user_types: row.user_types.length ? [...row.user_types] : [1, 2], resource_type: row.resource_type, points: row.points,
    currency: row.currency, sale_price: Number(row.sale_price), actual_revenue: Number(row.actual_revenue),
    original_price: Number(row.original_price), icon: row.icon || '', description: row.description || '',
    is_default: Boolean(row.is_default), status: row.status, sort: row.sort,
  })
  Object.assign(targetModes, {
    countries: countryCodes.length ? 'selected' : 'all', apps: appCodes.length ? 'selected' : 'all',
    packages: packageCodes.length ? 'selected' : 'all', versions: versionCodes.length ? 'selected' : 'all', channels: channelCodes.length ? 'selected' : 'all',
  })
  const availablePackages = formPackageOptions.value.map((item) => item.package_code)
  if (!packageCodes.length || (availablePackages.length && availablePackages.length === packageCodes.length && availablePackages.every((code) => packageCodes.includes(code)))) {
    targetModes.packages = 'all'
    form.package_codes = []
    targetModes.versions = 'all'
    form.version_codes = []
  }
  dialogVisible.value = true
}

function handleCountryModeChange(value: string | number | boolean | undefined) { if (value === 'all') form.country_codes = [] }
function handleChannelModeChange(value: string | number | boolean | undefined) { if (value === 'all') form.channel_codes = [] }
function handleAppModeChange(value: string | number | boolean | undefined) {
  if (value === 'all') {
    form.app_codes = []
    targetModes.packages = 'all'
    form.package_codes = []
    targetModes.versions = 'all'
    form.version_codes = []
  }
  handleAppSelectionChange()
}
function handlePackageModeChange(value: string | number | boolean | undefined) {
  if (value === 'all') {
    form.package_codes = []
    targetModes.versions = 'all'
    form.version_codes = []
  }
  handlePackageSelectionChange()
}
function handleVersionModeChange(value: string | number | boolean | undefined) { if (value === 'all') form.version_codes = [] }
function handleAppSelectionChange() {
  const allowed = new Set(formPackageOptions.value.map((item) => item.package_code))
  if (targetModes.packages === 'all') form.package_codes = []
  else form.package_codes = form.package_codes.filter((code) => allowed.has(code))
  handlePackageSelectionChange()
}
function handlePackageSelectionChange() {
  const allowed = new Set(formVersionOptions.value.map((item) => item.version_code))
  form.version_codes = form.version_codes.filter((code) => allowed.has(code))
  if (!form.package_codes.length) { targetModes.versions = 'all'; form.version_codes = [] }
}

async function handleSubmit() {
  await formRef.value?.validate()
  if (targetModes.countries === 'selected' && !form.country_codes.length) return void ElMessage.warning('请选择国家，或切换为全部国家')
  if (targetModes.apps === 'selected' && !form.app_codes.length) return void ElMessage.warning('请选择应用，或切换为全部应用')
  if (targetModes.packages === 'selected' && !form.package_codes.length) return void ElMessage.warning('请选择安装包，或切换为全部安装包')
  if (targetModes.versions === 'selected' && !form.version_codes.length) return void ElMessage.warning('请选择版本，或切换为全部版本')
  if (targetModes.channels === 'selected' && !form.channel_codes.length) return void ElMessage.warning('请选择渠道，或切换为全部渠道')
  submitting.value = true
  try {
    const payload: PointsPackagePayload = {
      product_code: form.product_code.trim(), name: form.name.trim(), app_codes: targetModes.apps === 'all' ? [] : [...form.app_codes],
      package_codes: targetModes.packages === 'all' ? [] : [...form.package_codes], version_codes: targetModes.versions === 'all' ? [] : [...form.version_codes],
      country_codes: targetModes.countries === 'all' ? [] : [...form.country_codes], channel_codes: targetModes.channels === 'all' ? [] : [...form.channel_codes],
      systems: form.systems.map((value) => value.toLowerCase()), user_types: [...form.user_types], resource_type: form.resource_type.trim().toLowerCase(),
      points: form.points, currency: form.currency.trim().toUpperCase(), sale_price: Number(form.sale_price), actual_revenue: Number(form.actual_revenue),
      original_price: Number(form.original_price), icon: form.icon.trim(), description: form.description.trim(), button_text: form.button_text.trim(),
      is_default: form.is_default, status: form.status, sort: form.sort,
    }
    if (form.id) await updatePointsPackage(form.id, payload)
    else await createPointsPackage(payload)
    ElMessage.success('积分套餐已保存')
    dialogVisible.value = false
    await fetchData()
  } finally { submitting.value = false }
}
async function handleStatus(row: Points) { try { await updatePointsPackageStatus(row.id, row.status); ElMessage.success('状态已更新') } catch { row.status = row.status === 1 ? 0 : 1 } }
async function handleSetDefault(id: number) { await setDefaultPointsPackage(id); ElMessage.success('默认套餐已更新'); await fetchData() }
async function handleDelete(id: number) { await deletePointsPackage(id); ElMessage.success('积分套餐已删除'); if (tableData.value.length === 1 && page.value > 1) page.value--; await fetchData() }

onMounted(() => Promise.all([fetchOptions(), fetchData()]))
</script>

<style scoped>
.page-wrap { min-width: 0; }
.page-header { display: flex; align-items: center; justify-content: space-between; gap: 16px; }
.page-title { color: #303133; font-size: 17px; font-weight: 600; }
.page-subtitle { margin-top: 4px; color: #909399; font-size: 12px; }
.filters { display: grid; grid-template-columns: repeat(5, minmax(150px, 1fr)); gap: 10px; }
.primary-filters { margin-bottom: 10px; }
.advanced-filters { padding: 12px; border: 1px solid #ebeef5; border-radius: 6px; background: #fafafa; }
.filter-actions { display: flex; align-items: center; margin: 10px 0 16px; }
.toggle-icon { margin-left: 4px; }
.primary-text { color: #303133; font-weight: 600; }
.product-id { display: inline-block; margin-top: 5px; padding: 2px 7px; border-radius: 4px; background: #f5f7fa; color: #606266; }
.tag-list { display: flex; flex-wrap: wrap; gap: 5px; }
.secondary-text { margin-top: 4px; color: #909399; font-size: 12px; }
.scope-summary { overflow: hidden; margin: 2px 0; color: #606266; text-overflow: ellipsis; white-space: nowrap; }
.scope-summary span { display: inline-block; width: 58px; color: #909399; }
.points-value { color: #409eff; font-size: 16px; }
.pagination-wrap { display: flex; justify-content: flex-end; margin-top: 16px; overflow-x: auto; }
.form-grid { display: grid; grid-template-columns: 1fr 1fr; column-gap: 16px; }
.full-grid-item { grid-column: 1 / -1; }
.form-grid :deep(.el-input-number) { width: 100%; }
.scope-field { display: flex; width: 100%; flex-direction: column; gap: 9px; }
.scope-tip { color: #909399; font-size: 12px; line-height: 18px; }
@media (max-width: 980px) { .filters { grid-template-columns: repeat(2, minmax(150px, 1fr)); } }
@media (max-width: 720px) {
  .page-header { align-items: stretch; flex-direction: column; }
  .filters, .form-grid { grid-template-columns: 1fr; }
  .full-grid-item { grid-column: auto; }
  .filter-actions { flex-wrap: wrap; }
  .page-wrap :deep(.el-card__header), .page-wrap :deep(.el-card__body) { padding: 14px; }
}
</style>
