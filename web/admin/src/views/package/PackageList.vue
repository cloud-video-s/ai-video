<template>
  <div class="page-wrap">
    <el-card shadow="never">
      <template #header>
        <div class="page-header">
          <div>
            <div class="page-title">安装包管理</div>
            <div class="page-subtitle">维护安装包版本、下载地址和安装使用统计</div>
          </div>
          <el-button v-if="canAdd" type="primary" @click="openCreate">
            <el-icon><Plus /></el-icon>新增包
          </el-button>
        </div>
      </template>

      <div class="filters">
        <el-input v-model="query.keyword" clearable placeholder="包名称、标识、版本或简介" @keyup.enter="handleSearch">
          <template #prefix><el-icon><Search /></el-icon></template>
        </el-input>
        <el-input v-model="query.package_code" clearable placeholder="包标识码" @keyup.enter="handleSearch" />
        <el-input v-model="query.package_version" clearable placeholder="包版本" @keyup.enter="handleSearch" />
        <el-select v-model="query.system_type" clearable filterable placeholder="系统类型">
          <el-option v-for="item in systemTypeOptions" :key="item.value" :label="item.label" :value="item.value" />
        </el-select>
        <el-select v-model="query.status" clearable placeholder="全部状态">
          <el-option label="启用" value="1" />
          <el-option label="禁用" value="0" />
        </el-select>
        <el-button type="primary" plain @click="handleSearch">查询</el-button>
        <el-button @click="handleReset">重置</el-button>
      </div>

      <el-table v-loading="loading" :data="tableData" row-key="id" stripe>
        <el-table-column prop="id" label="ID" width="70" />
        <el-table-column label="包信息" min-width="220">
          <template #default="{ row }">
            <div class="primary-text">{{ row.package_name }}</div>
            <code class="package-code">{{ row.package_code }}</code>
          </template>
        </el-table-column>
        <el-table-column label="版本" width="120" align="center">
          <template #default="{ row }"><el-tag effect="plain">{{ row.package_version }}</el-tag></template>
        </el-table-column>
        <el-table-column label="接口语言" width="120" align="center">
          <template #default="{ row }">{{ languageLabel(row.language) }}</template>
        </el-table-column>
        <el-table-column label="系统类型" min-width="170">
          <template #default="{ row }">
            <div v-if="row.system_types?.length" class="system-tags">
              <el-tag v-for="item in row.system_types" :key="item" size="small" type="info" effect="plain">
                {{ systemTypeLabel(item) }}
              </el-tag>
            </div>
            <span v-else class="secondary-text">未设置</span>
          </template>
        </el-table-column>
        <el-table-column label="包简介" min-width="220">
          <template #default="{ row }">
            <el-tooltip v-if="row.description" :content="row.description" placement="top" :show-after="400">
              <div class="description-text">{{ row.description }}</div>
            </el-tooltip>
            <span v-else class="secondary-text">暂无简介</span>
          </template>
        </el-table-column>
        <el-table-column label="使用统计" width="180">
          <template #default="{ row }">
            <div class="stats-line"><span>安装</span><strong>{{ formatNumber(row.install_count) }}</strong></div>
            <div class="stats-line"><span>下载</span><strong>{{ formatNumber(row.download_count) }}</strong></div>
            <div class="stats-line"><span>设备</span><strong>{{ formatNumber(row.device_count) }}</strong></div>
          </template>
        </el-table-column>
        <el-table-column label="下载" width="100" align="center">
          <template #default="{ row }">
            <el-link :href="row.download_url" type="primary" target="_blank" :underline="false">下载包</el-link>
          </template>
        </el-table-column>
        <el-table-column prop="sort" label="排序" width="70" align="center" />
        <el-table-column label="状态" width="86" align="center">
          <template #default="{ row }">
            <el-tag :type="row.status === 1 ? 'success' : 'danger'">{{ row.status === 1 ? '启用' : '禁用' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="更新时间" width="180">
          <template #default="{ row }">{{ formatDate(row.updated_at) }}</template>
        </el-table-column>
        <el-table-column v-if="canEdit || canDelete" label="操作" width="130" fixed="right" align="center">
          <template #default="{ row }">
            <el-button v-if="canEdit" link type="primary" @click="openEdit(row)">编辑</el-button>
            <el-popconfirm
              v-if="canDelete"
              :title="`确认删除 ${row.package_name} ${row.package_version}？`"
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
          @size-change="handlePageSizeChange"
          @current-change="fetchData"
        />
      </div>
    </el-card>

    <el-dialog v-model="dialogVisible" :title="form.id ? '编辑包' : '新增包'" width="760px" destroy-on-close>
      <el-form ref="formRef" :model="form" :rules="rules" label-width="100px">
        <div class="form-grid">
          <el-form-item label="包名称" prop="package_name">
            <el-input v-model="form.package_name" maxlength="128" placeholder="请输入包名称" />
          </el-form-item>
          <el-form-item label="包标识码" prop="package_code">
            <el-input v-model="form.package_code" maxlength="128" placeholder="例如：com.example.app" />
          </el-form-item>
          <el-form-item label="包版本" prop="package_version">
            <el-input v-model="form.package_version" maxlength="64" placeholder="例如：1.2.0" />
          </el-form-item>
          <el-form-item label="接口语言" prop="language">
            <el-select v-model="form.language" placeholder="请选择错误提示语言" style="width: 100%">
              <el-option v-for="item in languageOptions" :key="item.value" :label="item.label" :value="item.value" />
            </el-select>
          </el-form-item>
          <el-form-item label="系统类型" prop="system_types">
            <el-select
              v-model="form.system_types"
              multiple
              filterable
              allow-create
              default-first-option
              :reserve-keyword="false"
              placeholder="可多选或输入扩展类型"
              style="width: 100%"
            >
              <el-option v-for="item in systemTypeOptions" :key="item.value" :label="item.label" :value="item.value" />
            </el-select>
          </el-form-item>
          <el-form-item label="排序">
            <el-input-number v-model="form.sort" :min="0" :max="999999" controls-position="right" />
          </el-form-item>
        </div>
        <el-form-item label="下载链接" prop="download_url">
          <el-input v-model="form.download_url" maxlength="1024" clearable placeholder="https://... 或 /uploads/...">
            <template #append>
              <el-link v-if="form.download_url" :href="form.download_url" target="_blank" :underline="false">打开</el-link>
              <span v-else>打开</span>
            </template>
          </el-input>
        </el-form-item>
        <div class="form-grid stats-grid">
          <el-form-item label="安装次数">
            <el-input-number v-model="form.install_count" :min="0" :max="999999999999" controls-position="right" />
          </el-form-item>
          <el-form-item label="下载次数">
            <el-input-number v-model="form.download_count" :min="0" :max="999999999999" controls-position="right" />
          </el-form-item>
          <el-form-item label="设备数">
            <el-input-number v-model="form.device_count" :min="0" :max="999999999999" controls-position="right" />
          </el-form-item>
          <el-form-item label="状态">
            <el-radio-group v-model="form.status">
              <el-radio :value="1">启用</el-radio>
              <el-radio :value="0">禁用</el-radio>
            </el-radio-group>
          </el-form-item>
        </div>
        <el-form-item label="包简介" prop="description">
          <el-input v-model="form.description" type="textarea" :rows="4" maxlength="2000" show-word-limit />
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
  createPackage,
  deletePackage,
  getPackageList,
  updatePackage,
  type AppPackage,
  type AppPackagePayload,
} from '@/api/package'
import { useUserStore } from '@/store/user'

const userStore = useUserStore()
const canAdd = computed(() => userStore.hasPermission('package:add'))
const canEdit = computed(() => userStore.hasPermission('package:edit'))
const canDelete = computed(() => userStore.hasPermission('package:delete'))

const loading = ref(false)
const submitting = ref(false)
const dialogVisible = ref(false)
const formRef = ref<FormInstance>()
const tableData = ref<AppPackage[]>([])
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const systemTypeOptions = [
  { value: 'android', label: 'Android' },
  { value: 'ios', label: 'iOS' },
  { value: 'pc', label: 'PC' },
  { value: 'web', label: 'Web' },
]
const languageOptions = [
  { value: 'zh-CN', label: '简体中文' },
  { value: 'en-US', label: 'English' },
  { value: 'ja-JP', label: '日本語' },
  { value: 'ko-KR', label: '한국어' },
  { value: 'es-ES', label: 'Español' },
]
const query = reactive({ keyword: '', package_code: '', package_version: '', system_type: '', status: '' })

const defaultForm: AppPackagePayload & { id: number } = {
  id: 0,
  package_name: '',
  package_code: '',
  package_version: '',
  language: 'zh-CN',
  system_types: [] as string[],
  download_url: '',
  install_count: 0,
  download_count: 0,
  device_count: 0,
  description: '',
  sort: 0,
  status: 1,
}
const form = reactive({ ...defaultForm })
const rules: FormRules = {
  package_name: [{ required: true, message: '请输入包名称', trigger: 'blur' }],
  package_code: [
    { required: true, message: '请输入包标识码', trigger: 'blur' },
    { pattern: /^[A-Za-z0-9._-]+$/, message: '仅支持字母、数字、点、下划线和中划线', trigger: 'blur' },
  ],
  package_version: [{ required: true, message: '请输入包版本', trigger: 'blur' }],
  language: [{ required: true, message: '请选择接口语言', trigger: 'change' }],
  system_types: [{ required: true, type: 'array', min: 1, message: '请至少选择一种系统类型', trigger: 'change' }],
  download_url: [
    { required: true, message: '请输入包下载链接', trigger: 'blur' },
    { pattern: /^(https?:\/\/|\/)/i, message: '请输入 HTTP(S) 地址或站内绝对路径', trigger: 'blur' },
  ],
}

async function fetchData() {
  loading.value = true
  try {
    const params: Record<string, unknown> = { page: page.value, page_size: pageSize.value }
    for (const [key, value] of Object.entries(query)) {
      if (value !== '') params[key] = typeof value === 'string' ? value.trim() : value
    }
    const res: any = await getPackageList(params)
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
  Object.assign(query, { keyword: '', package_code: '', package_version: '', system_type: '', status: '' })
  page.value = 1
  fetchData()
}

function handlePageSizeChange() {
  page.value = 1
  fetchData()
}

function openCreate() {
  Object.assign(form, defaultForm)
  dialogVisible.value = true
}

function openEdit(row: AppPackage) {
  Object.assign(form, {
    id: row.id,
    package_name: row.package_name,
    package_code: row.package_code,
    package_version: row.package_version,
    language: row.language || 'zh-CN',
    system_types: [...(row.system_types || [])],
    download_url: row.download_url,
    install_count: row.install_count,
    download_count: row.download_count,
    device_count: row.device_count,
    description: row.description || '',
    sort: row.sort,
    status: row.status,
  })
  dialogVisible.value = true
}

async function handleSubmit() {
  await formRef.value?.validate()
  submitting.value = true
  try {
    const payload: AppPackagePayload = {
      package_name: form.package_name.trim(),
      package_code: form.package_code.trim(),
      package_version: form.package_version.trim(),
      language: form.language,
      system_types: form.system_types.map((item) => item.trim().toLowerCase()).filter(Boolean),
      download_url: form.download_url.trim(),
      install_count: form.install_count,
      download_count: form.download_count,
      device_count: form.device_count,
      description: form.description.trim(),
      sort: form.sort,
      status: form.status,
    }
    if (form.id) await updatePackage(form.id, payload)
    else await createPackage(payload)
    ElMessage.success('包信息已保存')
    dialogVisible.value = false
    await fetchData()
  } finally {
    submitting.value = false
  }
}

async function handleDelete(id: number) {
  await deletePackage(id)
  ElMessage.success('包已删除')
  if (tableData.value.length === 1 && page.value > 1) page.value--
  await fetchData()
}

function formatNumber(value: number) {
  return new Intl.NumberFormat('zh-CN').format(value || 0)
}

function systemTypeLabel(value: string) {
  return systemTypeOptions.find((item) => item.value === value)?.label || value
}

function languageLabel(value: string) {
  return languageOptions.find((item) => item.value === value)?.label || value || '简体中文'
}

function formatDate(value: string) {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleString('zh-CN', { hour12: false })
}

onMounted(fetchData)
</script>

<style scoped>
.page-wrap { min-width: 0; }
.page-header { display: flex; align-items: center; justify-content: space-between; gap: 16px; }
.page-title { color: #303133; font-size: 17px; font-weight: 600; }
.page-subtitle { margin-top: 4px; color: #909399; font-size: 12px; }
.filters { display: grid; grid-template-columns: minmax(220px, 1.4fr) minmax(170px, 1fr) 140px 140px 120px auto auto; gap: 10px; margin-bottom: 16px; }
.primary-text { color: #303133; font-weight: 600; }
.package-code { display: inline-block; margin-top: 6px; padding: 2px 7px; border-radius: 4px; background: #f2f3f5; color: #606266; font-size: 12px; }
.secondary-text { color: #909399; font-size: 12px; }
.system-tags { display: flex; flex-wrap: wrap; gap: 5px; }
.description-text { display: -webkit-box; overflow: hidden; color: #606266; line-height: 1.5; -webkit-box-orient: vertical; -webkit-line-clamp: 2; }
.stats-line { display: flex; align-items: center; justify-content: space-between; gap: 12px; line-height: 1.7; }
.stats-line span { color: #909399; font-size: 12px; }
.stats-line strong { color: #303133; font-size: 13px; font-variant-numeric: tabular-nums; }
.pagination-wrap { display: flex; justify-content: flex-end; margin-top: 16px; overflow-x: auto; }
.form-grid { display: grid; grid-template-columns: 1fr 1fr; column-gap: 14px; }
.form-grid :deep(.el-input-number) { width: 100%; }
@media (max-width: 1000px) {
  .filters { grid-template-columns: repeat(3, minmax(140px, 1fr)); }
}
@media (max-width: 700px) {
  .page-header { align-items: stretch; flex-direction: column; }
  .filters, .form-grid { grid-template-columns: 1fr; }
  .page-wrap :deep(.el-card__header), .page-wrap :deep(.el-card__body) { padding: 14px; }
}
</style>
