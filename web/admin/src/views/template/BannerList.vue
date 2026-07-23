<template>
  <div class="page-wrap">
    <el-card shadow="never">
      <template #header>
        <div class="page-header">
          <div>
            <div class="page-title">Banner 管理</div>
            <div class="page-subtitle">配置 Banner 投放范围、封面与客户端跳转行为</div>
          </div>
          <el-button v-if="canAdd" type="primary" @click="openCreate">
            <el-icon><Plus /></el-icon>新增 Banner
          </el-button>
        </div>
      </template>

      <div class="filters">
        <el-select v-model="query.position_key" clearable filterable placeholder="展示位置">
          <el-option v-for="item in positionOptions" :key="item.id" :label="positionLabel(item)" :value="item.position_key" />
        </el-select>
        <el-select v-model="query.country_code" clearable filterable placeholder="国家">
          <el-option v-for="item in countryOptions" :key="item.id" :label="countryLabel(item)" :value="item.code" />
        </el-select>
        <el-select v-model="query.app_code" clearable filterable placeholder="应用" @change="handleFilterAppChange">
          <el-option v-for="item in deliveryOptions" :key="item.app_code" :label="item.app_name" :value="item.app_code" />
        </el-select>
        <el-select v-model="query.package_code" clearable filterable placeholder="应用包" @change="query.version_code = ''">
          <el-option v-for="item in filterPackageOptions" :key="item.package_code" :label="`${item.package_name} · ${item.package_code}`" :value="item.package_code" />
        </el-select>
        <el-select v-model="query.version_code" clearable filterable placeholder="包版本">
          <el-option v-for="item in filterVersionOptions" :key="item.version_code" :label="item.version_code" :value="item.version_code" />
        </el-select>
        <el-select v-model="query.jump_type" clearable placeholder="跳转方式">
          <el-option v-for="item in jumpTypeOptions" :key="item.value" :label="item.label" :value="item.value" />
        </el-select>
        <el-select v-model="query.status" clearable placeholder="启用状态">
          <el-option label="启用" value="1" />
          <el-option label="禁用" value="0" />
        </el-select>
        <el-input v-model="query.keyword" clearable placeholder="Banner 名称或备注" @keyup.enter="handleSearch">
          <template #prefix><el-icon><Search /></el-icon></template>
        </el-input>
        <el-button type="primary" plain @click="handleSearch">查询</el-button>
        <el-button @click="handleReset">重置</el-button>
      </div>

      <el-table v-loading="loading" :data="tableData" row-key="id" stripe>
        <el-table-column prop="id" label="ID" width="70" />
        <el-table-column label="封面图" width="130" align="center">
          <template #default="{ row }">
            <el-image
              class="cover-image"
              :src="row.cover_image"
              :preview-src-list="[row.cover_image]"
              preview-teleported
              fit="cover"
            >
              <template #error><div class="image-error"><el-icon><Picture /></el-icon></div></template>
            </el-image>
          </template>
        </el-table-column>
        <el-table-column label="Banner" min-width="190">
          <template #default="{ row }">
            <div class="primary-text">{{ row.name }}</div>
            <div class="target-tags position-tags">
              <el-tag v-for="item in row.display_positions" :key="item.id" size="small" type="primary" effect="plain">
                {{ item.position_name }}
              </el-tag>
              <el-tag v-if="!row.display_positions?.length" size="small" type="info" effect="plain">全部位置</el-tag>
            </div>
            <el-tooltip v-if="row.remark" :content="row.remark" placement="top" :show-after="400">
              <div class="remark-text">{{ row.remark }}</div>
            </el-tooltip>
            <span v-else class="secondary-text">暂无备注</span>
          </template>
        </el-table-column>
        <el-table-column label="投放范围" min-width="260">
          <template #default="{ row }">
            <div class="target-tags">
              <el-tag size="small" effect="plain">{{ countrySummary(row.countries) }}</el-tag>
              <el-tag size="small" type="warning" effect="plain">{{ appTargetSummary(row.app_targets) }}</el-tag>
              <el-tag size="small" type="success" effect="plain">{{ subscriptionStatusLabel(row.subscription_status) }}</el-tag>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="跳转方式" min-width="210">
          <template #default="{ row }">
            <el-tag :type="jumpTagType(row.jump_type)" effect="plain">{{ jumpTypeLabel(row.jump_type) }}</el-tag>
            <div v-if="row.jump_type === 1" class="jump-target" :title="row.jump_url">{{ row.jump_url }}</div>
            <div v-else-if="row.jump_type === 2" class="jump-target">
              {{ row.template?.name || `模板 #${row.template_id || '-'}` }}
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="sort" label="排序" width="72" align="center" />
        <el-table-column label="状态" width="82" align="center">
          <template #default="{ row }">
            <el-tag :type="row.status === 1 ? 'success' : 'danger'">{{ row.status === 1 ? '启用' : '禁用' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="updated_at" label="更新时间" width="180" />
        <el-table-column v-if="canEdit || canDelete" label="操作" width="130" fixed="right" align="center">
          <template #default="{ row }">
            <el-button v-if="canEdit" link type="primary" @click="openEdit(row)">编辑</el-button>
            <el-popconfirm v-if="canDelete" title="确认删除该 Banner？" @confirm="handleDelete(row.id)">
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
          @size-change="fetchData"
          @current-change="fetchData"
        />
      </div>
    </el-card>

    <el-dialog v-model="dialogVisible" :title="form.id ? '编辑 Banner' : '新增 Banner'" width="940px" destroy-on-close>
      <el-form ref="formRef" :model="form" :rules="rules" label-width="96px">
        <div class="form-grid">
          <el-form-item label="Banner 名称" prop="name">
            <el-input v-model="form.name" maxlength="128" placeholder="请输入 Banner 名称" />
          </el-form-item>
          <el-form-item label="排序">
            <el-input-number v-model="form.sort" :min="0" :max="999999" controls-position="right" />
          </el-form-item>
          <el-form-item label="国家">
            <div class="target-scope-control">
              <el-radio-group v-model="targetModes.countries" @change="handleCountryModeChange">
                <el-radio-button value="all">全部国家</el-radio-button>
                <el-radio-button value="selected">指定国家</el-radio-button>
              </el-radio-group>
              <el-select
                v-if="targetModes.countries === 'selected'"
                v-model="form.country_ids"
                multiple
                collapse-tags
                collapse-tags-tooltip
                clearable
                filterable
                placeholder="请选择国家"
                style="width: 100%"
              >
                <el-option v-for="item in countryOptions" :key="item.id" :label="countryLabel(item)" :value="item.id" />
              </el-select>
              <div v-else class="secondary-text target-tip">全部国家不写入国家关联数据</div>
            </div>
          </el-form-item>
          <el-form-item label="状态">
            <el-radio-group v-model="form.status">
              <el-radio :value="1">启用</el-radio>
              <el-radio :value="0">禁用</el-radio>
            </el-radio-group>
          </el-form-item>
          <el-form-item label="会员类型">
            <el-select v-model="form.subscription_status" style="width: 100%">
              <el-option label="全部用户" :value="3" />
              <el-option label="会员" :value="2" />
              <el-option label="非会员" :value="1" />
            </el-select>
          </el-form-item>
          <el-form-item label="跳转方式" prop="jump_type">
            <el-select v-model="form.jump_type" style="width: 100%" @change="handleJumpTypeChange">
              <el-option v-for="item in jumpTypeOptions" :key="item.value" :label="item.label" :value="item.value" />
            </el-select>
          </el-form-item>
          <el-form-item v-if="form.jump_type === 2" label="目标模板" prop="template_id">
            <el-select v-model="form.template_id" clearable filterable placeholder="请选择目标模板" style="width: 100%">
              <el-option v-for="item in templateOptions" :key="item.id" :label="`${item.name} · #${item.id}`" :value="item.id" />
            </el-select>
          </el-form-item>
        </div>

        <el-form-item label="应用范围">
          <div class="app-target-field">
            <el-radio-group v-model="targetModes.apps" @change="handleAppModeChange">
              <el-radio-button value="all">全部应用、包和版本</el-radio-button>
              <el-radio-button value="selected">指定应用范围</el-radio-button>
            </el-radio-group>
            <template v-if="targetModes.apps === 'selected'">
              <div v-for="(target, index) in form.app_targets" :key="`${index}-${target.app_code}-${target.package_code}`" class="app-target-row">
                <el-select v-model="target.app_code" filterable placeholder="先选择应用" @change="handleTargetAppChange(target)">
                  <el-option v-for="item in deliveryOptions" :key="item.app_code" :label="item.app_name" :value="item.app_code" />
                </el-select>
                <el-select v-model="target.package_code" filterable placeholder="再选择应用包" :disabled="!target.app_code" @change="target.version_codes = []">
                  <el-option v-for="item in targetPackageOptions(target)" :key="item.package_code" :label="`${item.package_name} · ${item.package_code}`" :value="item.package_code" />
                </el-select>
                <el-select v-model="target.version_codes" multiple collapse-tags collapse-tags-tooltip clearable filterable placeholder="全部版本（不绑定具体版本）" :disabled="!target.package_code">
                  <el-option v-for="item in targetVersionOptions(target)" :key="item.version_code" :label="item.version_code" :value="item.version_code" />
                </el-select>
                <el-button type="danger" plain @click="removeAppTarget(index)">删除</el-button>
              </div>
              <el-button type="primary" plain @click="addAppTarget"><el-icon><Plus /></el-icon>添加应用包范围</el-button>
              <div class="secondary-text target-tip">已选择包但不选版本表示该包全部版本。</div>
            </template>
            <div v-else class="secondary-text target-tip">全部应用、包和版本不写入应用范围关联数据</div>
          </div>
        </el-form-item>

        <el-form-item label="展示区域">
          <div class="target-scope-control">
            <el-radio-group v-model="targetModes.positions" @change="handlePositionModeChange">
              <el-radio-button value="all">全部展示位置</el-radio-button>
              <el-radio-button value="selected">指定展示位置</el-radio-button>
            </el-radio-group>
            <template v-if="targetModes.positions === 'selected'">
              <el-checkbox-group v-if="positionOptions.length" v-model="form.display_position_keys" class="position-card-grid">
                <el-checkbox-button v-for="item in positionOptions" :key="item.position_key" :value="item.position_key" class="position-card-option">
                  <div class="position-card-content">
                    <el-image class="position-card-image" :src="item.cover_image" fit="cover">
                      <template #error><div class="image-error"><el-icon><Picture /></el-icon></div></template>
                    </el-image>
                    <div class="position-card-name">{{ item.position_name }}</div>
                    <div class="position-card-key">{{ item.position_key }}</div>
                  </div>
                </el-checkbox-button>
              </el-checkbox-group>
              <div v-else class="secondary-text target-tip">暂无启用的展示位置，请先到展示位置管理中启用</div>
            </template>
            <div v-else class="secondary-text target-tip">全部展示位置不写入展示位置关联数据</div>
          </div>
        </el-form-item>

        <el-form-item v-if="form.jump_type === 1" label="跳转链接" prop="jump_url">
          <el-input v-model="form.jump_url" maxlength="1024" clearable placeholder="https://...、myapp://... 或 /app/path" />
        </el-form-item>
        <el-alert
          v-else-if="form.jump_type === 3 || form.jump_type === 4"
          :title="form.jump_type === 3 ? '点击后进入文生图功能' : '点击后进入文生视频功能'"
          type="info"
          show-icon
          :closable="false"
          class="jump-alert"
        />

        <el-form-item label="封面图" prop="cover_image">
          <div class="cover-field">
            <MediaUploader
              v-model="form.cover_image"
              kind="image"
              resume-key="banner-cover"
              placeholder="输入图片 URL，或选择图片分片上传"
              @preview="(url) => openPreview(url, form.name || 'Banner 封面')"
              @uploading-change="(value) => (coverUploading = value)"
            />
            <el-image
              v-if="form.cover_image"
              class="cover-form-preview"
              :src="form.cover_image"
              :preview-src-list="[form.cover_image]"
              preview-teleported
              fit="cover"
            >
              <template #error><div class="image-error"><el-icon><Picture /></el-icon></div></template>
            </el-image>
          </div>
        </el-form-item>
        <el-form-item label="备注" prop="remark">
          <el-input v-model="form.remark" type="textarea" :rows="3" maxlength="500" show-word-limit placeholder="请输入 Banner 使用说明或投放备注" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitting" :disabled="coverUploading" @click="handleSubmit">
          {{ coverUploading ? '封面上传中' : '保存' }}
        </el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="preview.visible" :title="preview.title" width="760px" append-to-body destroy-on-close>
      <div class="preview-body"><el-image :src="preview.url" fit="contain" class="preview-image" /></div>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, type FormInstance, type FormRules } from 'element-plus'
import {
  createBanner,
  deleteBanner,
  getBannerDeliveryOptions,
  getBannerList,
  updateBanner,
  type BannerJumpType,
  type BannerAppTarget,
  type BannerDeliveryApp,
  type BannerDeliveryPackage,
  type BannerDeliveryVersion,
  type VideoBanner,
  type VideoBannerPayload,
} from '@/api/banner'
import { getCountryOptions, type Country } from '@/api/country'
import { getTemplateList, type VideoTemplate } from '@/api/template'
import { getDisplayPositionOptions, type DisplayPosition } from '@/api/displayPosition'
import { useUserStore } from '@/store/user'
import MediaUploader from '@/components/MediaUploader.vue'

interface BannerForm extends VideoBannerPayload { id: number }
type TargetMode = 'all' | 'selected'

const userStore = useUserStore()
const canAdd = computed(() => userStore.hasPermission('banner:add'))
const canEdit = computed(() => userStore.hasPermission('banner:edit'))
const canDelete = computed(() => userStore.hasPermission('banner:delete'))
const jumpTypeOptions: { value: BannerJumpType; label: string }[] = [
  { value: 1, label: '链接' },
  { value: 2, label: '视频模板' },
  { value: 3, label: '文生图功能' },
  { value: 4, label: '文生视频功能' },
]

const loading = ref(false)
const submitting = ref(false)
const coverUploading = ref(false)
const dialogVisible = ref(false)
const formRef = ref<FormInstance>()
const tableData = ref<VideoBanner[]>([])
const countryOptions = ref<Country[]>([])
const deliveryOptions = ref<BannerDeliveryApp[]>([])
const templateOptions = ref<VideoTemplate[]>([])
const positionOptions = ref<DisplayPosition[]>([])
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const query = reactive({ position_key: '', country_code: '', app_code: '', package_code: '', version_code: '', jump_type: '', status: '', keyword: '' })
const targetModes = reactive<{ countries: TargetMode; apps: TargetMode; positions: TargetMode }>({
  countries: 'all', apps: 'all', positions: 'all',
})

const filterPackageOptions = computed<BannerDeliveryPackage[]>(() => {
  if (!query.app_code) return deliveryOptions.value.flatMap((item) => item.packages)
  return deliveryOptions.value.find((item) => item.app_code === query.app_code)?.packages || []
})
const filterVersionOptions = computed<BannerDeliveryVersion[]>(() => {
  if (!query.package_code) return []
  return filterPackageOptions.value.find((item) => item.package_code === query.package_code)?.versions || []
})

function createDefaultForm(): BannerForm {
  return {
    id: 0, name: '', cover_image: '', display_position_keys: [], country_ids: [], app_targets: [], remark: '',
    sort: 0, jump_type: 1, jump_url: '', template_id: null, status: 1, subscription_status: 3,
  }
}
const form = reactive<BannerForm>(createDefaultForm())
const rules: FormRules<BannerForm> = {
  name: [{ required: true, message: '请输入 Banner 名称', trigger: 'blur' }],
  cover_image: [{ required: true, message: '请上传或输入封面图', trigger: 'change' }],
  jump_type: [{ required: true, message: '请选择跳转方式', trigger: 'change' }],
  jump_url: [{
    validator: (_rule, value, callback) => {
      if (form.jump_type === 1 && !String(value || '').trim()) callback(new Error('请输入跳转链接'))
      else callback()
    },
    trigger: 'blur',
  }],
  template_id: [{
    validator: (_rule, value, callback) => {
      if (form.jump_type === 2 && !value) callback(new Error('请选择目标模板'))
      else callback()
    },
    trigger: 'change',
  }],
}
const preview = reactive({ visible: false, url: '', title: '' })

function countryLabel(item: Country) { return `${item.name_zh} · ${item.code}` }
function positionLabel(item: DisplayPosition) { return `${item.position_name} · ${item.position_key}` }
function jumpTypeLabel(value: BannerJumpType) { return jumpTypeOptions.find((item) => item.value === value)?.label || value }
function jumpTagType(value: BannerJumpType) {
  return ({ 1: '', 2: 'success', 3: 'warning', 4: 'danger' } as const)[value]
}
function compactSummary(values: string[], allLabel: string) {
  if (!values?.length) return allLabel
  return values.length > 2 ? `${values.slice(0, 2).join('、')} 等 ${values.length} 项` : values.join('、')
}
function countrySummary(items: Country[]) { return compactSummary(items?.map((item) => item.name_zh), '全部国家') }
function appTargetSummary(items: BannerAppTarget[]) {
  return compactSummary(items?.map((item) => `${item.app_name || item.app_code}/${item.package_name || item.package_code}${item.version_codes.length ? `(${item.version_codes.join('、')})` : '(全部版本)'}`), '全部应用版本')
}
function subscriptionStatusLabel(value: number) { return ({ 1: '非会员', 2: '会员', 3: '全部用户' } as Record<number, string>)[value] || '全部用户' }

async function fetchOptions() {
  const [countryRes, deliveryRes, templateRes, positionRes]: any[] = await Promise.all([
    getCountryOptions(), getBannerDeliveryOptions(), getTemplateList({ page: 1, page_size: 100, status: 1 }), getDisplayPositionOptions(),
  ])
  countryOptions.value = countryRes.data || []
  deliveryOptions.value = deliveryRes.data || []
  templateOptions.value = templateRes.data?.list || []
  positionOptions.value = positionRes.data || []
}

async function fetchData() {
  loading.value = true
  try {
    const params: Record<string, unknown> = { page: page.value, page_size: pageSize.value }
    Object.entries(query).forEach(([key, value]) => { if (value !== '') params[key] = value })
    const res: any = await getBannerList(params)
    tableData.value = res.data.list || []
    total.value = res.data.total || 0
  } finally {
    loading.value = false
  }
}

function handleSearch() { page.value = 1; void fetchData() }
function handleReset() {
  Object.assign(query, { position_key: '', country_code: '', app_code: '', package_code: '', version_code: '', jump_type: '', status: '', keyword: '' })
  page.value = 1
  void fetchData()
}
function openCreate() {
  Object.assign(form, createDefaultForm())
  Object.assign(targetModes, { countries: 'all', apps: 'all', positions: 'all' })
  dialogVisible.value = true
}
function openEdit(row: VideoBanner) {
  Object.assign(form, {
    id: row.id,
    name: row.name,
    cover_image: row.cover_image,
    display_position_keys: row.display_positions?.map((item) => item.position_key) || [],
    country_ids: row.countries?.map((item) => item.id) || [],
    app_targets: (row.app_targets || []).map((item) => ({ app_code: item.app_code, package_code: item.package_code, version_codes: [...item.version_codes] })),
    remark: row.remark || '',
    sort: row.sort,
    jump_type: row.jump_type,
    jump_url: row.jump_url || '',
    template_id: row.template_id || null,
    status: row.status,
    subscription_status: row.subscription_status || 3,
  })
  Object.assign(targetModes, {
    countries: form.country_ids.length ? 'selected' : 'all',
    apps: form.app_targets.length ? 'selected' : 'all',
    positions: form.display_position_keys.length ? 'selected' : 'all',
  })
  dialogVisible.value = true
}
function handleJumpTypeChange(value: BannerJumpType) {
  if (value !== 1) form.jump_url = ''
  if (value !== 2) form.template_id = null
  formRef.value?.clearValidate(['jump_url', 'template_id'])
}
function handleFilterAppChange() { query.package_code = ''; query.version_code = '' }
function targetPackageOptions(target: { app_code: string }) { return deliveryOptions.value.find((item) => item.app_code === target.app_code)?.packages || [] }
function targetVersionOptions(target: { app_code: string; package_code: string }) { return targetPackageOptions(target).find((item) => item.package_code === target.package_code)?.versions || [] }
function handleTargetAppChange(target: { package_code: string; version_codes: string[] }) { target.package_code = ''; target.version_codes = [] }
function handleCountryModeChange(value: string | number | boolean | undefined) { if (value === 'all') form.country_ids = [] }
function handleAppModeChange(value: string | number | boolean | undefined) { if (value === 'all') form.app_targets = [] }
function handlePositionModeChange(value: string | number | boolean | undefined) { if (value === 'all') form.display_position_keys = [] }
function addAppTarget() { form.app_targets.push({ app_code: '', package_code: '', version_codes: [] }) }
function removeAppTarget(index: number) { form.app_targets.splice(index, 1) }
function openPreview(url: string, title: string) { preview.url = url; preview.title = title; preview.visible = true }

async function handleSubmit() {
  await formRef.value?.validate()
  if (targetModes.countries === 'selected' && !form.country_ids.length) {
    ElMessage.warning('请选择至少一个国家，或切换为全部国家')
    return
  }
  if (targetModes.positions === 'selected' && !form.display_position_keys.length) {
    ElMessage.warning('请选择至少一个展示位置，或切换为全部展示位置')
    return
  }
  if (targetModes.apps === 'selected' && !form.app_targets.length) {
    ElMessage.warning('请添加至少一个应用包范围，或切换为全部应用、包和版本')
    return
  }
  if (targetModes.apps === 'selected' && form.app_targets.some((item) => !item.app_code || !item.package_code)) {
    ElMessage.warning('请完整选择应用和应用包，版本可以留空表示全部版本')
    return
  }
  submitting.value = true
  try {
    const payload: VideoBannerPayload = {
      name: form.name.trim(), cover_image: form.cover_image.trim(),
      display_position_keys: targetModes.positions === 'all' ? [] : [...form.display_position_keys],
      country_ids: targetModes.countries === 'all' ? [] : [...form.country_ids],
      app_targets: targetModes.apps === 'all' ? [] : form.app_targets.map((item) => ({ app_code: item.app_code, package_code: item.package_code, version_codes: [...item.version_codes] })),
      remark: form.remark.trim(),
      sort: form.sort, jump_type: form.jump_type, jump_url: form.jump_type === 1 ? form.jump_url.trim() : '',
      template_id: form.jump_type === 2 ? form.template_id : null, status: form.status, subscription_status: form.subscription_status,
    }
    if (form.id) await updateBanner(form.id, payload)
    else await createBanner(payload)
    ElMessage.success('Banner 已保存')
    dialogVisible.value = false
    await fetchData()
  } finally {
    submitting.value = false
  }
}

async function handleDelete(id: number) {
  await deleteBanner(id)
  ElMessage.success('Banner 已删除')
  if (tableData.value.length === 1 && page.value > 1) page.value--
  await fetchData()
}

onMounted(async () => { await Promise.all([fetchOptions(), fetchData()]) })
</script>

<style scoped>
.page-wrap { min-width: 0; }
.page-header { display: flex; align-items: center; justify-content: space-between; gap: 16px; }
.page-title { color: #303133; font-size: 17px; font-weight: 600; }
.page-subtitle { margin-top: 4px; color: #909399; font-size: 12px; }
.filters { display: grid; grid-template-columns: repeat(5, minmax(130px, 1fr)) minmax(190px, 1.4fr) auto auto; gap: 10px; margin-bottom: 16px; }
.cover-image { width: 96px; height: 54px; border-radius: 6px; background: #f2f3f5; }
.image-error { width: 100%; height: 100%; display: flex; align-items: center; justify-content: center; color: #c0c4cc; font-size: 24px; }
.primary-text { color: #303133; font-weight: 500; }
.secondary-text { color: #909399; font-size: 12px; }
.remark-text { display: -webkit-box; overflow: hidden; margin-top: 5px; color: #909399; font-size: 12px; line-height: 1.45; -webkit-box-orient: vertical; -webkit-line-clamp: 2; }
.target-tags { display: flex; flex-wrap: wrap; gap: 5px; }
.jump-target { overflow: hidden; margin-top: 6px; color: #606266; font-size: 12px; text-overflow: ellipsis; white-space: nowrap; }
.pagination-wrap { display: flex; justify-content: flex-end; margin-top: 16px; overflow-x: auto; }
.form-grid { display: grid; grid-template-columns: 1fr 1fr; column-gap: 14px; }
.form-grid :deep(.el-input-number) { width: 100%; }
.jump-alert { margin: 0 0 18px 96px; width: calc(100% - 96px); }
.target-scope-control, .app-target-field { display: flex; width: 100%; flex-direction: column; align-items: flex-start; gap: 10px; }
.app-target-row { display: grid; width: 100%; grid-template-columns: 1fr 1.2fr 1.5fr auto; gap: 10px; }
.target-tip { margin-top: 0; line-height: 1.5; }
.position-card-grid { display: grid; width: 100%; grid-template-columns: repeat(auto-fill, minmax(150px, 1fr)); gap: 12px; }
.position-card-option { width: 100%; margin: 0; }
.position-card-option :deep(.el-checkbox-button__inner) { width: 100%; height: 100%; padding: 8px; border: 1px solid #dcdfe6; border-radius: 7px; box-shadow: none; white-space: normal; }
.position-card-option:first-child :deep(.el-checkbox-button__inner) { border-left: 1px solid #dcdfe6; border-radius: 7px; }
.position-card-option.is-checked :deep(.el-checkbox-button__inner) { border-color: var(--el-color-primary); background: var(--el-color-primary-light-9); color: var(--el-text-color-primary); box-shadow: 0 0 0 1px var(--el-color-primary); }
.position-card-content { min-width: 0; text-align: left; }
.position-card-image { width: 100%; height: 82px; border-radius: 5px; background: #f2f3f5; }
.position-card-name { overflow: hidden; margin-top: 7px; color: #303133; font-size: 13px; font-weight: 500; text-overflow: ellipsis; white-space: nowrap; }
.position-card-key { overflow: hidden; margin-top: 3px; color: #909399; font-family: monospace; font-size: 11px; text-overflow: ellipsis; white-space: nowrap; }
.cover-field { width: 100%; }
.cover-form-preview { width: 240px; height: 135px; margin-top: 10px; border: 1px solid #ebeef5; border-radius: 7px; background: #f5f7fa; }
.preview-body { display: flex; align-items: center; justify-content: center; min-height: 280px; border-radius: 8px; background: #0f1115; overflow: hidden; }
.preview-image { max-width: 100%; max-height: 72vh; }
@media (max-width: 1200px) { .filters { grid-template-columns: repeat(3, minmax(150px, 1fr)) auto auto; } }
@media (max-width: 700px) {
  .page-header { align-items: stretch; flex-direction: column; }
  .filters, .form-grid { grid-template-columns: 1fr; }
  .app-target-row { grid-template-columns: 1fr; }
  .jump-alert { margin-left: 0; width: 100%; }
  .page-wrap :deep(.el-card__header), .page-wrap :deep(.el-card__body) { padding: 14px; }
}
</style>
