<template>
  <div class="page-wrap">
    <el-card shadow="never">
      <template #header>
        <div class="page-header">
          <div>
            <div class="page-title">模板分类</div>
            <div class="page-subtitle">分类统一控制展示位置、国家、应用、安装包、版本和用户范围</div>
          </div>
          <el-button v-if="canAdd" type="primary" @click="openCreate">
            <el-icon><Plus /></el-icon>新增分类
          </el-button>
        </div>
      </template>

      <div class="filters">
        <el-input v-model="query.keyword" clearable placeholder="分类名称或描述" @keyup.enter="handleSearch">
          <template #prefix><el-icon><Search /></el-icon></template>
        </el-input>
        <el-select v-model="query.position_key" clearable filterable placeholder="展示位置">
          <el-option v-for="item in positionOptions" :key="item.id" :label="positionLabel(item)" :value="item.position_key" />
        </el-select>
        <el-select v-model="query.country_id" clearable filterable placeholder="国家">
          <el-option v-for="item in countryOptions" :key="item.id" :label="countryLabel(item)" :value="String(item.id)" />
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
        <el-select v-model="query.status" clearable placeholder="启用状态">
          <el-option label="启用" value="1" />
          <el-option label="禁用" value="0" />
        </el-select>
        <el-button type="primary" plain @click="handleSearch">查询</el-button>
        <el-button @click="handleReset">重置</el-button>
      </div>

      <el-table v-loading="loading" :data="tableData" row-key="id" stripe>
        <el-table-column prop="id" label="ID" width="72" />
        <el-table-column prop="category_name" label="分类名称" min-width="210">
          <template #default="{ row }">
            <div class="primary-text">{{ row.category_name }}</div>
            <div class="secondary-text line-clamp">{{ row.description || '暂无描述' }}</div>
          </template>
        </el-table-column>
        <el-table-column label="投放范围" min-width="380">
          <template #default="{ row }">
            <div class="target-tags">
              <el-tag size="small" effect="plain">{{ positionSummary(row.display_positions) }}</el-tag>
              <el-tag size="small" effect="plain">{{ countrySummary(row.countries) }}</el-tag>
              <el-tag size="small" type="info" effect="plain">{{ appSummary(row.apps) }}</el-tag>
              <el-tag size="small" type="info" effect="plain">{{ packageSummary(row.packages) }}</el-tag>
              <el-tag size="small" type="info" effect="plain">{{ versionSummary(row.versions) }}</el-tag>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="sort" label="排序" width="72" align="center" />
        <el-table-column label="状态" width="86" align="center">
          <template #default="{ row }">
            <el-tag :type="row.status === 1 ? 'success' : 'danger'">{{ row.status === 1 ? '启用' : '禁用' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column v-if="canEdit || canDelete" label="操作" width="130" fixed="right" align="center">
          <template #default="{ row }">
            <el-button v-if="canEdit" link type="primary" @click="openEdit(row)">编辑</el-button>
            <el-popconfirm v-if="canDelete" title="确认删除该分类？分类下存在模板时无法删除。" width="260" @confirm="handleDelete(row.id)">
              <template #reference><el-button link type="danger">删除</el-button></template>
            </el-popconfirm>
          </template>
        </el-table-column>
      </el-table>

      <div class="pagination-wrap">
        <el-pagination v-model:current-page="page" v-model:page-size="pageSize" :total="total" :page-sizes="[10, 20, 50, 100]" layout="total, sizes, prev, pager, next" @size-change="fetchData" @current-change="fetchData" />
      </div>
    </el-card>

    <el-dialog v-model="dialogVisible" :title="form.id ? '编辑模板分类' : '新增模板分类'" width="920px" destroy-on-close>
      <el-form ref="formRef" :model="form" :rules="rules" label-width="100px">
        <div class="form-grid">
          <el-form-item label="分类名称" prop="category_name">
            <el-input v-model="form.category_name" maxlength="128" placeholder="例如：热门、节日、商务" />
          </el-form-item>
          <el-form-item label="排序">
            <el-input-number v-model="form.sort" :min="0" :max="999999" controls-position="right" />
          </el-form-item>
          <el-form-item label="状态">
            <el-radio-group v-model="form.status">
              <el-radio :value="1">启用</el-radio>
              <el-radio :value="0">禁用</el-radio>
            </el-radio-group>
          </el-form-item>
        </div>

        <el-form-item label="分类图标" prop="icon">
          <LogoImageUploader
            v-model="form.icon"
            image-name="分类图标"
            placeholder="上传或输入图标 URL"
            :crop-aspect-ratio="1"
          />
        </el-form-item>
        <el-divider content-position="left">投放范围</el-divider>
        <el-form-item label="展示位置">
          <div class="scope-field">
            <el-radio-group v-model="targetModes.positions" @change="handlePositionModeChange">
              <el-radio-button value="all">全部位置</el-radio-button>
              <el-radio-button value="selected">指定位置</el-radio-button>
            </el-radio-group>
            <el-checkbox-group v-if="targetModes.positions === 'selected'" v-model="form.display_position_keys" class="position-card-grid">
              <el-checkbox-button v-for="item in positionOptions" :key="item.id" :value="item.position_key" class="position-card-option">
                <div class="position-card-content">
                  <el-image class="position-card-image" :src="toMediaURL(item.cover_image)" fit="cover">
                    <template #error><div class="image-error"><el-icon><Picture /></el-icon></div></template>
                  </el-image>
                  <div class="position-card-name">{{ item.position_name }}</div>
                  <div class="position-card-key">{{ item.position_key }}</div>
                </div>
              </el-checkbox-button>
            </el-checkbox-group>
            <div class="scope-tip">全部位置不会写入展示位置关联数据。</div>
          </div>
        </el-form-item>

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
            <div class="scope-tip">全部应用不会写入应用、安装包和版本关联数据。</div>
          </div>
        </el-form-item>

        <el-form-item v-if="targetModes.apps === 'selected' && form.app_codes.length" label="安装包">
          <div class="scope-field">
            <el-radio-group v-model="targetModes.packages" @change="handlePackageModeChange">
              <el-radio-button value="all">全部安装包</el-radio-button>
              <el-radio-button value="selected">指定安装包</el-radio-button>
            </el-radio-group>
            <el-select v-if="targetModes.packages === 'selected'" v-model="form.package_codes" multiple collapse-tags collapse-tags-tooltip clearable filterable placeholder="请选择安装包" style="width: 100%" @change="handlePackageSelectionChange">
              <el-option v-for="item in formPackageOptions" :key="item.package_code" :label="packageLabel(item)" :value="item.package_code" />
            </el-select>
            <div class="scope-tip">全部安装包不会写入安装包和版本关联数据。</div>
          </div>
        </el-form-item>

        <el-form-item v-if="targetModes.packages === 'selected' && form.package_codes.length" label="版本">
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

        <el-form-item label="描述" prop="description">
          <el-input v-model="form.description" type="textarea" :rows="3" maxlength="500" show-word-limit />
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
import { ElMessage, type FormInstance } from 'element-plus'
import { getDisplayPositionOptions, type DisplayPosition } from '@/api/displayPosition'
import { getCountryOptions, type Country } from '@/api/country'
import {
  getBannerDeliveryOptions,
  type BannerDeliveryApp,
  type BannerDeliveryPackage,
  type BannerDeliveryVersion,
} from '@/api/banner'
import {
  createTemplateType,
  deleteTemplateType,
  getTemplateTypeList,
  updateTemplateType,
  type VideoTemplateType,
  type VideoTemplateTypePayload,
} from '@/api/template'
import type { VideoApp } from '@/api/videoApp'
import type { AppPackage } from '@/api/package'
import type { PackageVersion } from '@/api/packageVersion'
import { useUserStore } from '@/store/user'
import { toMediaURL } from '@/utils/mediaUrl'
import LogoImageUploader from '@/components/LogoImageUploader.vue'

type TargetMode = 'all' | 'selected'
interface TemplateTypeForm {
  id: number
  category_name: string
  icon: string
  display_position_keys: string[]
  country_codes: string[]
  app_codes: string[]
  package_codes: string[]
  version_codes: string[]
  sort: number
  status: number
  description: string
}

const userStore = useUserStore()
const canAdd = computed(() => userStore.hasPermission('template:type:add'))
const canEdit = computed(() => userStore.hasPermission('template:type:edit'))
const canDelete = computed(() => userStore.hasPermission('template:type:delete'))
const loading = ref(false)
const submitting = ref(false)
const dialogVisible = ref(false)
const formRef = ref<FormInstance>()
const tableData = ref<VideoTemplateType[]>([])
const positionOptions = ref<DisplayPosition[]>([])
const countryOptions = ref<Country[]>([])
const deliveryOptions = ref<BannerDeliveryApp[]>([])
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const query = reactive({ position_key: '', country_id: '', app_code: '', package_code: '', version_code: '', status: '', keyword: '' })
const targetModes = reactive<Record<'positions' | 'countries' | 'apps' | 'packages' | 'versions', TargetMode>>({
  positions: 'all', countries: 'all', apps: 'all', packages: 'all', versions: 'all',
})

function createDefaultForm(): TemplateTypeForm {
  return {
    id: 0, category_name: '', icon: '', display_position_keys: [], country_codes: [], app_codes: [], package_codes: [],
    version_codes: [], sort: 0, status: 1, description: '',
  }
}
const form = reactive<TemplateTypeForm>(createDefaultForm())
const rules = { category_name: [{ required: true, message: '请输入分类名称', trigger: 'blur' }] }

const filterPackageOptions = computed<BannerDeliveryPackage[]>(() => {
  if (!query.app_code) return deliveryOptions.value.flatMap((item) => item.packages)
  return deliveryOptions.value.find((item) => item.app_code === query.app_code)?.packages || []
})
const filterVersionOptions = computed<BannerDeliveryVersion[]>(() => {
  if (!query.package_code) return []
  return filterPackageOptions.value.find((item) => item.package_code === query.package_code)?.versions || []
})
const formPackageOptions = computed<BannerDeliveryPackage[]>(() => {
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
function positionLabel(item: DisplayPosition) { return `${item.position_name} · ${item.position_key}` }
function countryLabel(item: Country) { return `${item.name_zh} · ${item.code}` }
function appLabel(item: BannerDeliveryApp) { return `${item.app_name} · ${item.app_code}` }
function packageLabel(item: BannerDeliveryPackage) { return `${item.package_name} · ${item.package_code}` }
function compactSummary(labels: string[], allLabel: string) {
  if (!labels.length) return allLabel
  return labels.length > 2 ? `${labels.slice(0, 2).join('、')} 等 ${labels.length} 项` : labels.join('、')
}
function positionSummary(items?: DisplayPosition[]) { return compactSummary(arrayValue(items).map((item) => item.position_name), '全部位置') }
function countrySummary(items?: Country[]) { return compactSummary(arrayValue(items).map((item) => item.name_zh), '全部国家') }
function appSummary(items?: VideoApp[]) { return compactSummary(arrayValue(items).map((item) => item.name || item.app_code), '全部应用') }
function packageSummary(items?: AppPackage[]) { return compactSummary(arrayValue(items).map((item) => item.package_name), '全部安装包') }
function versionSummary(items?: PackageVersion[]) { return compactSummary(arrayValue(items).map((item) => item.version_code), '全部版本') }

async function fetchData() {
  loading.value = true
  try {
    const params: Record<string, unknown> = { page: page.value, page_size: pageSize.value }
    Object.entries(query).forEach(([key, value]) => { if (value !== '') params[key] = value })
    const res: any = await getTemplateTypeList(params)
    tableData.value = arrayValue<VideoTemplateType>(res.data?.list)
    total.value = Number(res.data?.total) || 0
  } finally {
    loading.value = false
  }
}
function handleSearch() { page.value = 1; void fetchData() }
function handleReset() {
  Object.assign(query, { position_key: '', country_id: '', app_code: '', package_code: '', version_code: '', status: '', keyword: '' })
  page.value = 1
  void fetchData()
}
function handleFilterAppChange() { query.package_code = ''; query.version_code = '' }
function handleFilterPackageChange() { query.version_code = '' }

function openCreate() {
  Object.assign(form, createDefaultForm())
  Object.assign(targetModes, { positions: 'all', countries: 'all', apps: 'all', packages: 'all', versions: 'all' })
  dialogVisible.value = true
}
function openEdit(row: VideoTemplateType) {
  Object.assign(form, {
    id: row.id,
    category_name: row.category_name,
    icon: row.icon || '',
    display_position_keys: arrayValue(row.display_positions).map((item) => item.position_key),
    country_codes: arrayValue(row.countries).map((item) => item.code),
    app_codes: arrayValue(row.apps).map((item) => item.app_code),
    package_codes: arrayValue(row.packages).map((item) => item.package_code),
    version_codes: arrayValue(row.versions).map((item) => item.version_code),
    sort: row.sort,
    status: row.status,
    description: row.description || '',
  })
  Object.assign(targetModes, {
    positions: form.display_position_keys.length ? 'selected' : 'all',
    countries: form.country_codes.length ? 'selected' : 'all',
    apps: form.app_codes.length ? 'selected' : 'all',
    packages: form.package_codes.length ? 'selected' : 'all',
    versions: form.version_codes.length ? 'selected' : 'all',
  })
  dialogVisible.value = true
}
function handlePositionModeChange(value: string | number | boolean | undefined) { if (value === 'all') form.display_position_keys = [] }
function handleCountryModeChange(value: string | number | boolean | undefined) { if (value === 'all') form.country_codes = [] }
function handleAppModeChange(value: string | number | boolean | undefined) {
  if (value === 'all') {
    form.app_codes = []; form.package_codes = []; form.version_codes = []
    targetModes.packages = 'all'; targetModes.versions = 'all'
  }
}
function handlePackageModeChange(value: string | number | boolean | undefined) {
  if (value === 'all') { form.package_codes = []; form.version_codes = []; targetModes.versions = 'all' }
}
function handleVersionModeChange(value: string | number | boolean | undefined) { if (value === 'all') form.version_codes = [] }
function handleAppSelectionChange() {
  const allowed = new Set(formPackageOptions.value.map((item) => item.package_code))
  form.package_codes = form.package_codes.filter((code) => allowed.has(code))
  if (!form.package_codes.length) { targetModes.packages = 'all'; form.version_codes = []; targetModes.versions = 'all' }
  else handlePackageSelectionChange()
}
function handlePackageSelectionChange() {
  const allowed = new Set(formVersionOptions.value.map((item) => item.version_code))
  form.version_codes = form.version_codes.filter((code) => allowed.has(code))
  if (!form.package_codes.length) { targetModes.versions = 'all'; form.version_codes = [] }
}

async function handleSubmit() {
  await formRef.value?.validate()
  if (targetModes.positions === 'selected' && !form.display_position_keys.length) return void ElMessage.warning('请选择展示位置，或切换为全部位置')
  if (targetModes.countries === 'selected' && !form.country_codes.length) return void ElMessage.warning('请选择国家，或切换为全部国家')
  if (targetModes.apps === 'selected' && !form.app_codes.length) return void ElMessage.warning('请选择应用，或切换为全部应用')
  if (targetModes.packages === 'selected' && !form.package_codes.length) return void ElMessage.warning('请选择安装包，或切换为全部安装包')
  if (targetModes.versions === 'selected' && !form.version_codes.length) return void ElMessage.warning('请选择版本，或切换为全部版本')
  submitting.value = true
  try {
    const payload: VideoTemplateTypePayload = {
      category_name: form.category_name.trim(),
      icon: form.icon.trim(),
      display_position_keys: targetModes.positions === 'all' ? [] : [...form.display_position_keys],
      country_codes: targetModes.countries === 'all' ? [] : [...form.country_codes],
      app_rules: targetModes.apps === 'all' ? [] : form.app_codes.map((app_code) => ({ app_code })),
      package_codes: targetModes.apps === 'all' || targetModes.packages === 'all' ? [] : [...form.package_codes],
      version_codes: targetModes.apps === 'all' || targetModes.packages === 'all' || targetModes.versions === 'all' ? [] : [...form.version_codes],
      sort: form.sort,
      status: form.status,
      description: form.description.trim(),
    }
    if (form.id) await updateTemplateType(form.id, payload)
    else await createTemplateType(payload)
    ElMessage.success('模板分类已保存')
    dialogVisible.value = false
    await fetchData()
  } finally {
    submitting.value = false
  }
}
async function handleDelete(id: number) {
  await deleteTemplateType(id)
  ElMessage.success('模板分类已删除')
  if (tableData.value.length === 1 && page.value > 1) page.value--
  await fetchData()
}
async function fetchOptions() {
  const [positionRes, countryRes, deliveryRes]: any[] = await Promise.all([
    getDisplayPositionOptions(), getCountryOptions(), getBannerDeliveryOptions(),
  ])
  positionOptions.value = arrayValue(positionRes.data)
  countryOptions.value = arrayValue(countryRes.data)
  deliveryOptions.value = arrayValue(deliveryRes.data)
}
onMounted(() => Promise.all([fetchOptions(), fetchData()]))
</script>

<style scoped>
.page-wrap { min-width: 0; }
.page-header { display: flex; align-items: center; justify-content: space-between; gap: 16px; }
.page-title { color: #303133; font-size: 17px; font-weight: 600; }
.page-subtitle { margin-top: 4px; color: #909399; font-size: 12px; }
.filters { display: grid; grid-template-columns: minmax(180px, 1.4fr) repeat(6, minmax(125px, 1fr)) auto auto; gap: 10px; margin-bottom: 16px; }
.primary-text { color: #303133; font-weight: 500; }
.secondary-text { margin-top: 4px; color: #909399; font-size: 12px; }
.line-clamp { max-width: 300px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.target-tags { display: flex; flex-wrap: wrap; gap: 5px; }
.pagination-wrap { display: flex; justify-content: flex-end; margin-top: 16px; overflow-x: auto; }
.form-grid { display: grid; grid-template-columns: 1fr 1fr; column-gap: 14px; }
.form-grid :deep(.el-input-number) { width: 100%; }
.scope-field { display: flex; width: 100%; flex-direction: column; align-items: flex-start; gap: 10px; }
.scope-tip { color: #909399; font-size: 12px; line-height: 1.5; }
.position-card-grid { display: grid; width: 100%; grid-template-columns: repeat(auto-fill, minmax(150px, 1fr)); gap: 12px; }
.position-card-option { width: 100%; margin: 0; }
.position-card-option :deep(.el-checkbox-button__inner) { width: 100%; height: 100%; padding: 8px; border: 1px solid #dcdfe6; border-radius: 7px; box-shadow: none; white-space: normal; }
.position-card-option:first-child :deep(.el-checkbox-button__inner) { border-left: 1px solid #dcdfe6; border-radius: 7px; }
.position-card-option.is-checked :deep(.el-checkbox-button__inner) { border-color: var(--el-color-primary); background: var(--el-color-primary-light-9); color: var(--el-text-color-primary); box-shadow: 0 0 0 1px var(--el-color-primary); }
.position-card-content { min-width: 0; text-align: left; }
.position-card-image { width: 100%; height: 82px; border-radius: 5px; background: #f2f3f5; }
.position-card-name { overflow: hidden; margin-top: 7px; color: #303133; font-size: 13px; font-weight: 500; text-overflow: ellipsis; white-space: nowrap; }
.position-card-key { overflow: hidden; margin-top: 3px; color: #909399; font-family: monospace; font-size: 11px; text-overflow: ellipsis; white-space: nowrap; }
.image-error { display: flex; align-items: center; justify-content: center; width: 100%; height: 100%; color: #a8abb2; font-size: 24px; }
@media (max-width: 1200px) { .filters { grid-template-columns: repeat(4, minmax(140px, 1fr)); } }
@media (max-width: 700px) {
  .page-header { align-items: stretch; flex-direction: column; }
  .filters, .form-grid { grid-template-columns: 1fr; }
  .page-wrap :deep(.el-card__header), .page-wrap :deep(.el-card__body) { padding: 14px; }
}
</style>
