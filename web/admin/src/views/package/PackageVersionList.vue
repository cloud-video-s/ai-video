<template>
  <div class="page-wrap">
    <el-card shadow="never">
      <template #header>
        <div class="page-header">
          <div>
            <div class="page-title">版本管理</div>
            <div class="page-subtitle">维护安装包版本、下载地址以及安装、下载和设备统计</div>
          </div>
          <el-button v-if="canAdd" type="primary" @click="openCreate">新增版本</el-button>
        </div>
      </template>

      <div class="filters">
        <el-input v-model="query.keyword" clearable placeholder="版本、包标识或描述" @keyup.enter="handleSearch" />
        <el-select v-model="query.package_code" clearable filterable placeholder="所属安装包">
          <el-option v-for="item in packageOptions" :key="item.id" :label="packageLabel(item)" :value="item.package_code" />
        </el-select>
        <el-input v-model="query.version_code" clearable maxlength="50" placeholder="版本号" @keyup.enter="handleSearch" />
        <el-select v-model="query.status" clearable placeholder="全部状态">
          <el-option label="启用" value="1" />
          <el-option label="禁用" value="2" />
        </el-select>
        <el-button type="primary" plain @click="handleSearch">查询</el-button>
        <el-button @click="handleReset">重置</el-button>
      </div>

      <el-table v-loading="loading" :data="tableData" row-key="id" stripe>
        <el-table-column prop="id" label="ID" width="70" />
        <el-table-column label="安装包" min-width="220">
          <template #default="{ row }">
            <div class="primary-text">{{ packageName(row.package_code) }}</div>
            <code class="code-text">{{ row.package_code }}</code>
          </template>
        </el-table-column>
        <el-table-column label="版本" width="150" align="center">
          <template #default="{ row }"><el-tag effect="plain">{{ row.version_code }}</el-tag></template>
        </el-table-column>
        <el-table-column label="下载地址" min-width="220">
          <template #default="{ row }">
            <el-link :href="row.download_url" type="primary" target="_blank" :underline="false" class="download-link">
              {{ row.download_url }}
            </el-link>
          </template>
        </el-table-column>
        <el-table-column label="使用统计" width="180">
          <template #default="{ row }">
            <div class="stats-line"><span>安装</span><strong>{{ formatNumber(row.install_count) }}</strong></div>
            <div class="stats-line"><span>下载</span><strong>{{ formatNumber(row.download_count) }}</strong></div>
            <div class="stats-line"><span>设备</span><strong>{{ formatNumber(row.device_count) }}</strong></div>
          </template>
        </el-table-column>
        <el-table-column label="描述" min-width="210">
          <template #default="{ row }">
            <el-tooltip v-if="row.description" :content="row.description" placement="top" :show-after="400">
              <div class="description-text">{{ row.description }}</div>
            </el-tooltip>
            <span v-else class="secondary-text">暂无描述</span>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="90" align="center">
          <template #default="{ row }">
            <el-tag :type="row.status === 1 ? 'success' : 'danger'">{{ row.status === 1 ? '启用' : '禁用' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="更新时间" width="180">
          <template #default="{ row }">{{ formatDate(row.updated_at) }}</template>
        </el-table-column>
        <el-table-column v-if="canEdit || canDelete" label="操作" width="140" fixed="right" align="center">
          <template #default="{ row }">
            <el-button v-if="canEdit" link type="primary" @click="openEdit(row)">编辑</el-button>
            <el-popconfirm v-if="canDelete" :title="`确认删除版本“${row.version_code}”？`" width="230" @confirm="handleDelete(row.id)">
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

    <el-dialog v-model="dialogVisible" :title="form.id ? '编辑版本' : '新增版本'" width="760px" destroy-on-close>
      <el-form ref="formRef" :model="form" :rules="rules" label-width="100px">
        <div class="form-grid">
          <el-form-item label="所属安装包" prop="package_code">
            <el-select v-model="form.package_code" filterable placeholder="请选择安装包" style="width: 100%">
              <el-option v-for="item in packageOptions" :key="item.id" :label="packageLabel(item)" :value="item.package_code" />
            </el-select>
          </el-form-item>
          <el-form-item label="版本号" prop="version_code">
            <el-input v-model="form.version_code" maxlength="50" placeholder="例如：1.2.0" />
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
          <el-form-item label="状态" prop="status">
            <el-radio-group v-model="form.status">
              <el-radio :value="1">启用</el-radio>
              <el-radio :value="2">禁用</el-radio>
            </el-radio-group>
          </el-form-item>
        </div>
        <el-form-item label="版本描述" prop="description">
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
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage, type FormInstance, type FormRules } from 'element-plus'
import { getPackageOptions, type AppPackage } from '@/api/package'
import {
  createPackageVersion,
  deletePackageVersion,
  getPackageVersionList,
  updatePackageVersion,
  type PackageVersion,
  type PackageVersionPayload,
} from '@/api/packageVersion'
import { useUserStore } from '@/store/user'

const route = useRoute()
const userStore = useUserStore()
const canAdd = computed(() => userStore.hasPermission('package:version:add'))
const canEdit = computed(() => userStore.hasPermission('package:version:edit'))
const canDelete = computed(() => userStore.hasPermission('package:version:delete'))
const packageOptions = ref<AppPackage[]>([])
const loading = ref(false)
const submitting = ref(false)
const dialogVisible = ref(false)
const formRef = ref<FormInstance>()
const tableData = ref<PackageVersion[]>([])
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const query = reactive({ keyword: '', package_code: '', version_code: '', status: '' })
const defaultForm: PackageVersionPayload & { id: number } = {
  id: 0,
  package_code: '',
  version_code: '',
  download_url: '',
  install_count: 0,
  download_count: 0,
  device_count: 0,
  description: '',
  status: 1,
}
const form = reactive({ ...defaultForm })
const rules: FormRules = {
  package_code: [{ required: true, message: '请选择所属安装包', trigger: 'change' }],
  version_code: [
    { required: true, message: '请输入版本号', trigger: 'blur' },
    { pattern: /^[A-Za-z0-9._+-]+$/, message: '仅支持字母、数字、点、下划线、中划线和加号', trigger: 'blur' },
  ],
  download_url: [
    { required: true, message: '请输入下载链接', trigger: 'blur' },
    { pattern: /^(https?:\/\/|\/)/i, message: '请输入 HTTP(S) 地址或站内绝对路径', trigger: 'blur' },
  ],
}

async function fetchOptions() {
  const response: any = await getPackageOptions()
  packageOptions.value = response.data || []
}

async function fetchData() {
  loading.value = true
  try {
    const params: Record<string, unknown> = { page: page.value, page_size: pageSize.value }
    for (const [key, value] of Object.entries(query)) {
      if (value !== '') params[key] = value.trim()
    }
    const response: any = await getPackageVersionList(params)
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
  Object.assign(query, { keyword: '', package_code: '', version_code: '', status: '' })
  page.value = 1
  fetchData()
}

function handlePageSizeChange() {
  page.value = 1
  fetchData()
}

function openCreate() {
  Object.assign(form, defaultForm, {
    package_code: query.package_code || packageOptions.value.find((item) => item.status === 1)?.package_code || '',
  })
  dialogVisible.value = true
}

function openEdit(row: PackageVersion) {
  Object.assign(form, {
    id: row.id,
    package_code: row.package_code,
    version_code: row.version_code,
    download_url: row.download_url,
    install_count: row.install_count,
    download_count: row.download_count,
    device_count: row.device_count,
    description: row.description || '',
    status: row.status,
  })
  dialogVisible.value = true
}

async function handleSubmit() {
  await formRef.value?.validate()
  submitting.value = true
  try {
    const payload: PackageVersionPayload = {
      package_code: form.package_code.trim(),
      version_code: form.version_code.trim(),
      download_url: form.download_url.trim(),
      install_count: form.install_count,
      download_count: form.download_count,
      device_count: form.device_count,
      description: form.description.trim(),
      status: form.status,
    }
    if (form.id) await updatePackageVersion(form.id, payload)
    else await createPackageVersion(payload)
    ElMessage.success('版本信息已保存')
    dialogVisible.value = false
    await fetchData()
  } finally {
    submitting.value = false
  }
}

async function handleDelete(id: number) {
  await deletePackageVersion(id)
  ElMessage.success('版本已删除')
  if (tableData.value.length === 1 && page.value > 1) page.value--
  await fetchData()
}

function packageLabel(item: AppPackage) {
  return `${item.package_name} · ${item.package_code}`
}

function packageName(packageCode: string) {
  return packageOptions.value.find((item) => item.package_code === packageCode)?.package_name || '未知安装包'
}

function formatNumber(value: number) {
  return Number(value || 0).toLocaleString('zh-CN')
}

function formatDate(value: string) {
  if (!value) return '-'
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString('zh-CN', { hour12: false })
}

watch(
  () => route.query.package_code,
  (value) => {
    const packageCode = typeof value === 'string' ? value : ''
    if (query.package_code !== packageCode) {
      query.package_code = packageCode
      page.value = 1
      fetchData()
    }
  },
)

onMounted(async () => {
  query.package_code = typeof route.query.package_code === 'string' ? route.query.package_code : ''
  await Promise.all([fetchOptions(), fetchData()])
})
</script>

<style scoped>
.page-wrap { min-width: 0; }
.page-header { display: flex; align-items: center; justify-content: space-between; gap: 16px; }
.page-title { color: #303133; font-size: 17px; font-weight: 600; }
.page-subtitle { margin-top: 4px; color: #909399; font-size: 12px; }
.filters { display: grid; grid-template-columns: minmax(220px, 1fr) minmax(220px, 1fr) 150px 120px auto auto; gap: 10px; margin-bottom: 16px; }
.primary-text { color: #303133; font-weight: 600; }
.code-text { display: inline-block; margin-top: 6px; padding: 2px 7px; border-radius: 4px; background: #f2f3f5; color: #606266; font-size: 12px; }
.secondary-text { color: #909399; font-size: 12px; }
.download-link { display: block; overflow: hidden; max-width: 100%; text-overflow: ellipsis; white-space: nowrap; }
.stats-line { display: flex; align-items: center; justify-content: space-between; gap: 12px; line-height: 1.7; }
.stats-line span { color: #909399; font-size: 12px; }
.stats-line strong { color: #303133; font-size: 13px; font-variant-numeric: tabular-nums; }
.description-text { display: -webkit-box; overflow: hidden; color: #606266; line-height: 1.5; -webkit-box-orient: vertical; -webkit-line-clamp: 2; }
.pagination-wrap { display: flex; justify-content: flex-end; margin-top: 16px; overflow-x: auto; }
.form-grid { display: grid; grid-template-columns: 1fr 1fr; column-gap: 16px; }
.form-grid :deep(.el-input-number) { width: 100%; }
@media (max-width: 950px) { .filters { grid-template-columns: repeat(2, minmax(160px, 1fr)); } }
@media (max-width: 700px) {
  .page-header { align-items: stretch; flex-direction: column; }
  .filters, .form-grid { grid-template-columns: 1fr; }
  .page-wrap :deep(.el-card__header), .page-wrap :deep(.el-card__body) { padding: 14px; }
}
</style>
