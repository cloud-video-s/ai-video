<template>
  <div class="page-wrap">
    <el-card shadow="never">
      <template #header>
        <div class="page-header">
          <div>
            <div class="page-title">模板分类</div>
            <div class="page-subtitle">维护视频模板使用的基础分类、展示位置、排序和启用状态</div>
          </div>
          <el-button v-if="canAdd" type="primary" @click="openCreate">
            <el-icon><Plus /></el-icon>新增分类
          </el-button>
        </div>
      </template>

      <div class="filters">
        <el-input v-model="query.keyword" clearable placeholder="分类名称、位置标识或描述" @keyup.enter="handleSearch">
          <template #prefix><el-icon><Search /></el-icon></template>
        </el-input>
        <el-select v-model="query.position_key" clearable filterable placeholder="展示位置">
          <el-option
            v-for="item in positionOptions"
            :key="item.id"
            :label="`${item.position_name} · ${item.position_key}`"
            :value="item.position_key"
          />
        </el-select>
        <el-select v-model="query.country_id" clearable filterable placeholder="国家">
          <el-option v-for="item in countryOptions" :key="item.id" :label="`${item.name_zh} · ${item.code}`" :value="String(item.id)" />
        </el-select>
        <el-select v-model="query.channel_id" clearable filterable placeholder="渠道">
          <el-option v-for="item in channelOptions" :key="item.channel_id" :label="`${item.channel_name} · ${item.channel_code}`" :value="String(item.channel_id)" />
        </el-select>
        <el-select v-model="query.package_id" clearable filterable placeholder="安装包">
          <el-option v-for="item in packageOptions" :key="item.id" :label="`${item.package_name} · ${item.package_code}`" :value="String(item.id)" />
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
        <el-table-column prop="category_name" label="分类名称" min-width="260">
          <template #default="{ row }">
            <div class="primary-text">{{ row.category_name }}</div>
            <div class="secondary-text line-clamp">{{ row.description || '暂无描述' }}</div>
          </template>
        </el-table-column>
        <el-table-column label="展示位置" min-width="220">
          <template #default="{ row }">
            <div v-if="positionItems(row).length" class="position-table-list">
              <div v-for="item in positionItems(row)" :key="item.id" class="position-table-cell">
                <el-image class="position-table-image" :src="item.cover_image" fit="cover">
                  <template #error><div class="image-error"><el-icon><Picture /></el-icon></div></template>
                </el-image>
                <div class="position-table-text">
                  <div class="primary-text">{{ item.position_name }}</div>
                  <div class="secondary-text position-key">{{ item.position_key }}</div>
                </div>
              </div>
            </div>
            <span v-else class="secondary-text">未关联</span>
          </template>
        </el-table-column>
        <el-table-column label="投放条件" min-width="260">
          <template #default="{ row }">
            <div class="target-tags">
              <el-tag size="small" effect="plain">{{ countrySummary(row.countries) }}</el-tag>
              <el-tag size="small" type="info" effect="plain">{{ channelSummary(row.channels) }}</el-tag>
              <el-tag size="small" type="info" effect="plain">{{ packageSummary(row.packages) }}</el-tag>
              <el-tag size="small" type="warning" effect="plain">{{ userTypesLabel(row.user_types) }}</el-tag>
              <el-tag size="small" type="success" effect="plain">{{ subscriptionLabel(row.subscription_statuses) }}</el-tag>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="sort" label="排序" width="72" align="center" />
        <el-table-column label="状态" width="86" align="center">
          <template #default="{ row }">
            <el-tag :type="row.status === 1 ? 'success' : 'danger'">
              {{ row.status === 1 ? '启用' : '禁用' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column v-if="canEdit || canDelete" label="操作" width="130" fixed="right" align="center">
          <template #default="{ row }">
            <el-button v-if="canEdit" link type="primary" @click="openEdit(row)">编辑</el-button>
            <el-popconfirm
              v-if="canDelete"
              title="确认删除该分类？分类下存在模板时将无法删除。"
              width="260"
              @confirm="handleDelete(row.id)"
            >
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

    <el-dialog
      v-model="dialogVisible"
      :title="form.id ? '编辑模板分类' : '新增模板分类'"
      width="860px"
      destroy-on-close
    >
      <el-form ref="formRef" :model="form" :rules="rules" label-width="92px">
        <el-form-item label="分类名称" prop="category_name">
          <el-input v-model="form.category_name" maxlength="128" placeholder="例如：热门、节日、商务" />
        </el-form-item>
        <el-form-item label="展示位置" prop="display_position_keys">
          <el-checkbox-group v-model="form.display_position_keys" class="position-card-grid">
            <el-checkbox-button
              v-for="item in positionOptions"
              :key="item.id"
              :value="item.position_key"
              class="position-card-option"
            >
              <div class="position-card-content">
                <el-image class="position-card-image" :src="item.cover_image" fit="cover">
                  <template #error><div class="image-error"><el-icon><Picture /></el-icon></div></template>
                </el-image>
                <div class="position-card-name">{{ item.position_name }}</div>
                <div class="position-card-key">{{ item.position_key }}</div>
              </div>
            </el-checkbox-button>
          </el-checkbox-group>
          <div v-if="!positionOptions.length" class="secondary-text">
            暂无启用的展示位置，请先到展示位置管理中新增或启用
          </div>
        </el-form-item>
        <div class="form-grid">
          <el-form-item label="国家">
            <el-select v-model="form.country_ids" multiple collapse-tags collapse-tags-tooltip clearable filterable placeholder="留空表示全部国家" style="width: 100%">
              <el-option v-for="item in countryOptions" :key="item.id" :label="`${item.name_zh} · ${item.code}`" :value="item.id" />
            </el-select>
          </el-form-item>
          <el-form-item label="渠道">
            <el-select v-model="form.channel_ids" multiple collapse-tags collapse-tags-tooltip clearable filterable placeholder="留空表示全部渠道" style="width: 100%">
              <el-option v-for="item in channelOptions" :key="item.channel_id" :label="`${item.channel_name} · ${item.channel_code}`" :value="item.channel_id" />
            </el-select>
          </el-form-item>
          <el-form-item label="安装包">
            <el-select v-model="form.package_ids" multiple collapse-tags collapse-tags-tooltip clearable filterable placeholder="留空表示全部安装包" style="width: 100%">
              <el-option v-for="item in packageOptions" :key="item.id" :label="`${item.package_name} · ${item.package_code}`" :value="item.id" />
            </el-select>
          </el-form-item>
          <el-form-item label="用户类型" prop="user_types">
            <el-select v-model="form.user_types" multiple style="width: 100%">
              <el-option label="免费用户" :value="1" />
              <el-option label="付费用户" :value="2" />
            </el-select>
          </el-form-item>
          <el-form-item label="订阅状态" prop="subscription_statuses">
            <el-select v-model="form.subscription_statuses" multiple style="width: 100%">
              <el-option label="已订阅" value="subscribed" />
              <el-option label="未订阅" value="unsubscribed" />
            </el-select>
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
import { getChannelOptions, type Channel } from '@/api/channel'
import { getPackageOptions, type AppPackage } from '@/api/package'
import {
  createTemplateType,
  deleteTemplateType,
  getTemplateTypeList,
  updateTemplateType,
  type VideoTemplateType,
  type VideoTemplateTypePayload,
} from '@/api/template'
import { useUserStore } from '@/store/user'

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
const channelOptions = ref<Channel[]>([])
const packageOptions = ref<AppPackage[]>([])
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const query = reactive({ position_key: '', country_id: '', channel_id: '', package_id: '', status: '', keyword: '' })

const defaultForm = {
  id: 0,
  category_name: '',
  display_position_keys: [] as string[],
  country_ids: [] as number[],
  channel_ids: [] as number[],
  package_ids: [] as number[],
  user_types: [1, 2] as number[],
  subscription_statuses: ['subscribed', 'unsubscribed'] as string[],
  sort: 0,
  status: 1,
  description: '',
}
const form = reactive({ ...defaultForm })
const rules = {
  category_name: [{ required: true, message: '请输入分类名称', trigger: 'blur' }],
  display_position_keys: [{ required: true, type: 'array', min: 1, message: '请至少选择一个展示位置', trigger: 'change' }],
  user_types: [{ required: true, type: 'array', min: 1, message: '请选择用户类型', trigger: 'change' }],
  subscription_statuses: [{ required: true, type: 'array', min: 1, message: '请选择订阅状态', trigger: 'change' }],
}

function positionItems(row: VideoTemplateType) {
  return row.display_positions || []
}

function userTypesLabel(values: number[]) { return values?.length === 2 ? '全部用户' : values?.[0] === 2 ? '付费用户' : '免费用户' }
function subscriptionLabel(values: string[]) { return values?.length === 2 ? '全部订阅状态' : values?.[0] === 'subscribed' ? '已订阅' : '未订阅' }
function compactSummary(values: string[], allLabel: string) {
  if (!values?.length) return allLabel
  return values.length > 2 ? `${values.slice(0, 2).join('、')} 等 ${values.length} 项` : values.join('、')
}
function countrySummary(items: Country[]) { return compactSummary(items?.map((item) => item.name_zh), '全部国家') }
function channelSummary(items: Channel[]) { return compactSummary(items?.map((item) => item.channel_name), '全部渠道') }
function packageSummary(items: AppPackage[]) { return compactSummary(items?.map((item) => item.package_name), '全部安装包') }

async function fetchData() {
  loading.value = true
  try {
    const params: Record<string, unknown> = { page: page.value, page_size: pageSize.value }
    for (const [key, value] of Object.entries(query)) {
      if (value !== '') params[key] = value
    }
    const res: any = await getTemplateTypeList(params)
    tableData.value = res.data.list || []
    total.value = res.data.total || 0
  } finally {
    loading.value = false
  }
}

function handleSearch() {
  page.value = 1
  fetchData()
}

function handleReset() {
  Object.assign(query, { position_key: '', country_id: '', channel_id: '', package_id: '', status: '', keyword: '' })
  page.value = 1
  fetchData()
}

function openCreate() {
  Object.assign(form, defaultForm, {
    display_position_keys: [], user_types: [1, 2], subscription_statuses: ['subscribed', 'unsubscribed'],
    country_ids: [], channel_ids: [], package_ids: [],
  })
  dialogVisible.value = true
}

function openEdit(row: VideoTemplateType) {
  Object.assign(form, {
    id: row.id,
    category_name: row.category_name,
    display_position_keys: positionItems(row).map((item) => item.position_key),
    country_ids: row.countries?.map((item) => item.id) || [],
    channel_ids: row.channels?.map((item) => item.channel_id) || [],
    package_ids: row.packages?.map((item) => item.id) || [],
    user_types: row.user_types?.length ? [...row.user_types] : [1, 2],
    subscription_statuses: row.subscription_statuses?.length ? [...row.subscription_statuses] : ['subscribed', 'unsubscribed'],
    sort: row.sort,
    status: row.status,
    description: row.description || '',
  })
  dialogVisible.value = true
}

async function handleSubmit() {
  await formRef.value?.validate()
  submitting.value = true
  try {
    const payload: VideoTemplateTypePayload = {
      category_name: form.category_name.trim(),
      display_position_keys: [...form.display_position_keys],
      country_ids: [...form.country_ids],
      channel_ids: [...form.channel_ids],
      package_ids: [...form.package_ids],
      user_types: [...form.user_types],
      subscription_statuses: [...form.subscription_statuses],
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
  const [positionRes, countryRes, channelRes, packageRes]: any[] = await Promise.all([
    getDisplayPositionOptions(), getCountryOptions(), getChannelOptions(), getPackageOptions(),
  ])
  positionOptions.value = positionRes.data || []
  countryOptions.value = countryRes.data || []
  channelOptions.value = channelRes.data || []
  packageOptions.value = packageRes.data || []
}

onMounted(() => Promise.all([fetchOptions(), fetchData()]))
</script>

<style scoped>
.page-wrap { min-width: 0; }
.page-header { display: flex; align-items: center; justify-content: space-between; gap: 16px; }
.page-title { color: #303133; font-size: 17px; font-weight: 600; }
.page-subtitle { margin-top: 4px; color: #909399; font-size: 12px; }
.filters { display: grid; grid-template-columns: repeat(3, minmax(150px, 1fr)) minmax(180px, 1fr) 150px 150px auto auto; gap: 10px; margin-bottom: 16px; }
.primary-text { color: #303133; font-weight: 500; }
.secondary-text { margin-top: 4px; color: #909399; font-size: 12px; }
.line-clamp { max-width: 300px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.position-table-list { display: flex; flex-wrap: wrap; gap: 8px; }
.position-table-cell { display: flex; align-items: center; min-width: 170px; gap: 8px; }
.position-table-image { flex: 0 0 auto; width: 72px; height: 44px; border-radius: 5px; background: #f2f3f5; }
.position-table-text { min-width: 0; }
.target-tags { display: flex; flex-wrap: wrap; gap: 5px; }
.position-key { overflow: hidden; max-width: 120px; font-family: monospace; text-overflow: ellipsis; white-space: nowrap; }
.pagination-wrap { display: flex; justify-content: flex-end; margin-top: 16px; overflow-x: auto; }
.form-grid { display: grid; grid-template-columns: 1fr 1fr; column-gap: 12px; }
.form-grid :deep(.el-input-number) { width: 100%; }
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
@media (max-width: 900px) {
  .filters { grid-template-columns: repeat(2, minmax(140px, 1fr)); }
}
@media (max-width: 620px) {
  .page-header { align-items: stretch; flex-direction: column; }
  .filters, .form-grid { grid-template-columns: 1fr; }
  .page-wrap :deep(.el-card__header), .page-wrap :deep(.el-card__body) { padding: 14px; }
}
</style>
