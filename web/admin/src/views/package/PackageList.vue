<template>
  <div class="page-wrap">
    <el-card shadow="never">
      <template #header>
        <div class="page-header">
          <div>
            <div class="page-title">安装包管理</div>
            <div class="page-subtitle">维护应用下的 iOS、Android 安装包；具体版本和下载信息在版本管理中维护</div>
          </div>
          <el-button v-if="canAdd" type="primary" @click="openCreate">新增安装包</el-button>
        </div>
      </template>

      <div class="filters">
        <el-input v-model="query.keyword" clearable placeholder="包名称、标识、应用或描述" @keyup.enter="handleSearch" />
        <el-select v-model="query.app_code" clearable filterable placeholder="所属应用">
          <el-option v-for="item in appOptions" :key="item.id" :label="appLabel(item)" :value="item.app_code" />
        </el-select>
        <el-input v-model="query.package_code" clearable placeholder="包标识码" @keyup.enter="handleSearch" />
        <el-select v-model="query.system_type" clearable placeholder="系统类型">
          <el-option v-for="item in systemTypeOptions" :key="item.value" :label="item.label" :value="String(item.value)" />
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
        <el-table-column label="安装包" min-width="220">
          <template #default="{ row }">
            <div class="primary-text">{{ row.package_name }}</div>
            <code class="code-text">{{ row.package_code }}</code>
          </template>
        </el-table-column>
        <el-table-column label="所属应用" min-width="190">
          <template #default="{ row }">
            <div>{{ appName(row.app_code) }}</div>
            <code class="secondary-text">{{ row.app_code }}</code>
          </template>
        </el-table-column>
        <el-table-column label="系统" width="110" align="center">
          <template #default="{ row }"><el-tag effect="plain">{{ systemTypeLabel(row.system_type) }}</el-tag></template>
        </el-table-column>
        <el-table-column label="描述" min-width="230">
          <template #default="{ row }">
            <el-tooltip v-if="row.description" :content="row.description" placement="top" :show-after="400">
              <div class="description-text">{{ row.description }}</div>
            </el-tooltip>
            <span v-else class="secondary-text">暂无描述</span>
          </template>
        </el-table-column>
        <el-table-column prop="sort" label="排序" width="75" align="center" />
        <el-table-column label="状态" width="90" align="center">
          <template #default="{ row }">
            <el-tag :type="row.status === 1 ? 'success' : 'danger'">{{ row.status === 1 ? '启用' : '禁用' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="更新时间" width="180">
          <template #default="{ row }">{{ formatDate(row.updated_at) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="205" fixed="right" align="center">
          <template #default="{ row }">
            <el-button v-if="canViewVersions" link type="primary" @click="openVersions(row)">版本</el-button>
            <el-button v-if="canEdit" link type="primary" @click="openEdit(row)">编辑</el-button>
            <el-popconfirm v-if="canDelete" :title="`确认删除安装包“${row.package_name}”？`" width="250" @confirm="handleDelete(row.id)">
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

    <el-dialog v-model="dialogVisible" :title="form.id ? '编辑安装包' : '新增安装包'" width="680px" destroy-on-close>
      <el-form ref="formRef" :model="form" :rules="rules" label-width="100px">
        <div class="form-grid">
          <el-form-item label="所属应用" prop="app_code">
            <el-select v-model="form.app_code" filterable placeholder="请选择应用" style="width: 100%">
              <el-option v-for="item in appOptions" :key="item.id" :label="appLabel(item)" :value="item.app_code" />
            </el-select>
          </el-form-item>
          <el-form-item label="系统类型" prop="system_type">
            <el-select v-model="form.system_type" style="width: 100%">
              <el-option v-for="item in systemTypeOptions" :key="item.value" :label="item.label" :value="item.value" />
            </el-select>
          </el-form-item>
          <el-form-item label="包名称" prop="package_name">
            <el-input v-model="form.package_name" maxlength="128" placeholder="请输入包名称" />
          </el-form-item>
          <el-form-item label="包标识码" prop="package_code">
            <el-input v-model="form.package_code" :disabled="form.id > 0" maxlength="128" placeholder="例如：com.example.app" />
          </el-form-item>
          <el-form-item label="排序" prop="sort">
            <el-input-number v-model="form.sort" :min="0" :max="999999" controls-position="right" />
          </el-form-item>
          <el-form-item label="状态" prop="status">
            <el-radio-group v-model="form.status">
              <el-radio :value="1">启用</el-radio>
              <el-radio :value="0">禁用</el-radio>
            </el-radio-group>
          </el-form-item>
        </div>
        <el-form-item label="包描述" prop="description">
          <el-input v-model="form.description" type="textarea" :rows="4" maxlength="10000" show-word-limit />
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
import { useRouter } from 'vue-router'
import { ElMessage, type FormInstance, type FormRules } from 'element-plus'
import { createPackage, deletePackage, getPackageList, updatePackage, type AppPackage, type AppPackagePayload } from '@/api/package'
import { getVideoAppOptions, type VideoApp } from '@/api/videoApp'
import { useUserStore } from '@/store/user'

const router = useRouter()
const userStore = useUserStore()
const canAdd = computed(() => userStore.hasPermission('package:add'))
const canEdit = computed(() => userStore.hasPermission('package:edit'))
const canDelete = computed(() => userStore.hasPermission('package:delete'))
const canViewVersions = computed(() => userStore.hasPermission('package:version:list'))
const systemTypeOptions = [
  { value: 1, label: 'iOS' },
  { value: 2, label: 'Android' },
]
const appOptions = ref<VideoApp[]>([])
const loading = ref(false)
const submitting = ref(false)
const dialogVisible = ref(false)
const formRef = ref<FormInstance>()
const tableData = ref<AppPackage[]>([])
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const query = reactive({ keyword: '', app_code: '', package_code: '', system_type: '', status: '' })
const defaultForm: AppPackagePayload & { id: number } = {
  id: 0,
  package_name: '',
  package_code: '',
  app_code: '',
  description: '',
  sort: 0,
  status: 1,
  system_type: 2,
}
const form = reactive({ ...defaultForm })
const rules: FormRules = {
  app_code: [{ required: true, message: '请选择所属应用', trigger: 'change' }],
  package_name: [{ required: true, message: '请输入包名称', trigger: 'blur' }],
  package_code: [
    { required: true, message: '请输入包标识码', trigger: 'blur' },
    { pattern: /^[A-Za-z0-9._-]+$/, message: '仅支持字母、数字、点、下划线和中划线', trigger: 'blur' },
  ],
  system_type: [{ required: true, type: 'number', message: '请选择系统类型', trigger: 'change' }],
}

async function fetchOptions() {
  const response: any = await getVideoAppOptions()
  appOptions.value = response.data || []
}

async function fetchData() {
  loading.value = true
  try {
    const params: Record<string, unknown> = { page: page.value, page_size: pageSize.value }
    for (const [key, value] of Object.entries(query)) {
      if (value !== '') params[key] = value.trim()
    }
    const response: any = await getPackageList(params)
    tableData.value = response.data.list || []
    total.value = response.data.total || 0
  } finally {
    loading.value = false
  }
}

function handleSearch() {
  page.value = 1
  fetchData()
}

function handleReset() {
  Object.assign(query, { keyword: '', app_code: '', package_code: '', system_type: '', status: '' })
  page.value = 1
  fetchData()
}

function handlePageSizeChange() {
  page.value = 1
  fetchData()
}

function openCreate() {
  Object.assign(form, defaultForm, { app_code: appOptions.value.find((item) => item.status === 1)?.app_code || '' })
  dialogVisible.value = true
}

function openEdit(row: AppPackage) {
  Object.assign(form, {
    id: row.id,
    package_name: row.package_name,
    package_code: row.package_code,
    app_code: row.app_code,
    description: row.description || '',
    sort: row.sort,
    status: row.status,
    system_type: row.system_type,
  })
  dialogVisible.value = true
}

function openVersions(row: AppPackage) {
  router.push({ path: '/package/versions', query: { package_code: row.package_code } })
}

async function handleSubmit() {
  await formRef.value?.validate()
  submitting.value = true
  try {
    const payload: AppPackagePayload = {
      package_name: form.package_name.trim(),
      package_code: form.package_code.trim(),
      app_code: form.app_code.trim(),
      description: form.description.trim(),
      sort: form.sort,
      status: form.status,
      system_type: form.system_type,
    }
    if (form.id) await updatePackage(form.id, payload)
    else await createPackage(payload)
    ElMessage.success('安装包信息已保存')
    dialogVisible.value = false
    await fetchData()
  } finally {
    submitting.value = false
  }
}

async function handleDelete(id: number) {
  await deletePackage(id)
  ElMessage.success('安装包已删除')
  if (tableData.value.length === 1 && page.value > 1) page.value--
  await fetchData()
}

function appLabel(item: VideoApp) {
  return `${item.name} · ${item.app_code}`
}

function appName(appCode: string) {
  return appOptions.value.find((item) => item.app_code === appCode)?.name || '未知应用'
}

function systemTypeLabel(value: number) {
  return systemTypeOptions.find((item) => item.value === value)?.label || `未知（${value}）`
}

function formatDate(value: string) {
  if (!value) return '-'
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString('zh-CN', { hour12: false })
}

onMounted(() => Promise.all([fetchOptions(), fetchData()]))
</script>

<style scoped>
.page-wrap { min-width: 0; }
.page-header { display: flex; align-items: center; justify-content: space-between; gap: 16px; }
.page-title { color: #303133; font-size: 17px; font-weight: 600; }
.page-subtitle { margin-top: 4px; color: #909399; font-size: 12px; }
.filters { display: grid; grid-template-columns: minmax(220px, 1fr) minmax(190px, 1fr) 180px 130px 120px auto auto; gap: 10px; margin-bottom: 16px; }
.primary-text { color: #303133; font-weight: 600; }
.code-text { display: inline-block; margin-top: 6px; padding: 2px 7px; border-radius: 4px; background: #f2f3f5; color: #606266; font-size: 12px; }
.secondary-text { margin-top: 4px; color: #909399; font-size: 12px; }
.description-text { display: -webkit-box; overflow: hidden; color: #606266; line-height: 1.5; -webkit-box-orient: vertical; -webkit-line-clamp: 2; }
.pagination-wrap { display: flex; justify-content: flex-end; margin-top: 16px; overflow-x: auto; }
.form-grid { display: grid; grid-template-columns: 1fr 1fr; column-gap: 16px; }
.form-grid :deep(.el-input-number) { width: 100%; }
@media (max-width: 1100px) { .filters { grid-template-columns: repeat(3, minmax(150px, 1fr)); } }
@media (max-width: 700px) {
  .page-header { align-items: stretch; flex-direction: column; }
  .filters, .form-grid { grid-template-columns: 1fr; }
  .page-wrap :deep(.el-card__header), .page-wrap :deep(.el-card__body) { padding: 14px; }
}
</style>
