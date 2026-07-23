<template>
  <div class="page-wrap">
    <el-card shadow="never">
      <template #header>
        <div class="page-header">
          <div>
            <div class="page-title">应用管理</div>
            <div class="page-subtitle">维护应用名称、应用标识、状态及排序</div>
          </div>
          <el-button v-if="canAdd" type="primary" @click="openCreate">新增应用</el-button>
        </div>
      </template>

      <div class="filters">
        <el-input v-model="query.keyword" clearable placeholder="名称、标识或描述" @keyup.enter="handleSearch" />
        <el-input v-model="query.app_code" clearable maxlength="60" placeholder="应用标识" @keyup.enter="handleSearch" />
        <el-select v-model="query.status" clearable placeholder="全部状态">
          <el-option label="启用" value="1" />
          <el-option label="禁用" value="0" />
        </el-select>
        <el-button type="primary" plain @click="handleSearch">查询</el-button>
        <el-button @click="handleReset">重置</el-button>
      </div>

      <el-table v-loading="loading" :data="tableData" row-key="id" stripe>
        <el-table-column prop="id" label="ID" width="90" />
        <el-table-column prop="name" label="应用名称" min-width="180">
          <template #default="{ row }"><span class="app-name">{{ row.name }}</span></template>
        </el-table-column>
        <el-table-column label="应用标识" min-width="180">
          <template #default="{ row }"><code class="app-code">{{ row.app_code }}</code></template>
        </el-table-column>
        <el-table-column label="描述" min-width="240">
          <template #default="{ row }">
            <el-tooltip v-if="row.description" :content="row.description" placement="top" :show-after="400">
              <div class="description-text">{{ row.description }}</div>
            </el-tooltip>
            <span v-else class="secondary-text">暂无描述</span>
          </template>
        </el-table-column>
        <el-table-column prop="sort" label="排序" width="90" align="center" />
        <el-table-column label="状态" width="110" align="center">
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
            <el-popconfirm v-if="canDelete" :title="`确认删除应用“${row.name}”？`" width="240" @confirm="handleDelete(row.id)">
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

    <el-dialog v-model="dialogVisible" :title="form.id ? '编辑应用' : '新增应用'" width="600px" destroy-on-close>
      <el-form ref="formRef" :model="form" :rules="rules" label-width="90px">
        <el-form-item label="应用名称" prop="name">
          <el-input v-model="form.name" maxlength="255" show-word-limit placeholder="请输入应用名称" />
        </el-form-item>
        <el-form-item label="应用标识" prop="app_code">
          <el-input v-model="form.app_code" maxlength="60" show-word-limit placeholder="例如：ai.video" />
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
        <el-form-item label="应用描述" prop="description">
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
import { ElMessage, type FormInstance, type FormRules } from 'element-plus'
import { createVideoApp, deleteVideoApp, getVideoAppList, updateVideoApp, type VideoApp, type VideoAppPayload } from '@/api/videoApp'
import { useUserStore } from '@/store/user'

const userStore = useUserStore()
const canAdd = computed(() => userStore.hasPermission('app:add'))
const canEdit = computed(() => userStore.hasPermission('app:edit'))
const canDelete = computed(() => userStore.hasPermission('app:delete'))

const loading = ref(false)
const submitting = ref(false)
const dialogVisible = ref(false)
const formRef = ref<FormInstance>()
const tableData = ref<VideoApp[]>([])
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const query = reactive({ keyword: '', app_code: '', status: '' })
const defaultForm: VideoAppPayload & { id: number } = {
  id: 0,
  name: '',
  app_code: '',
  status: 1,
  sort: 0,
  description: '',
}
const form = reactive({ ...defaultForm })
const rules: FormRules = {
  name: [{ required: true, message: '请输入应用名称', trigger: 'blur' }],
  app_code: [
    { required: true, message: '请输入应用标识', trigger: 'blur' },
    { pattern: /^[A-Za-z0-9._-]+$/, message: '仅支持字母、数字、点、下划线和中划线', trigger: 'blur' },
  ],
}

async function fetchData() {
  loading.value = true
  try {
    const params: Record<string, unknown> = { page: page.value, page_size: pageSize.value }
    if (query.keyword.trim()) params.keyword = query.keyword.trim()
    if (query.app_code.trim()) params.app_code = query.app_code.trim()
    if (query.status !== '') params.status = query.status
    const response: any = await getVideoAppList(params)
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
  Object.assign(query, { keyword: '', app_code: '', status: '' })
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

function openEdit(row: VideoApp) {
  Object.assign(form, {
    id: row.id,
    name: row.name,
    app_code: row.app_code,
    status: row.status,
    sort: row.sort,
    description: row.description || '',
  })
  dialogVisible.value = true
}

async function handleSubmit() {
  await formRef.value?.validate()
  submitting.value = true
  try {
    const payload: VideoAppPayload = {
      name: form.name.trim(),
      app_code: form.app_code.trim(),
      status: form.status,
      sort: form.sort,
      description: form.description.trim(),
    }
    if (form.id) await updateVideoApp(form.id, payload)
    else await createVideoApp(payload)
    ElMessage.success('应用信息已保存')
    dialogVisible.value = false
    await fetchData()
  } finally {
    submitting.value = false
  }
}

async function handleDelete(id: number) {
  await deleteVideoApp(id)
  ElMessage.success('应用已删除')
  if (tableData.value.length === 1 && page.value > 1) page.value--
  await fetchData()
}

function formatDate(value: string) {
  if (!value) return '-'
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString('zh-CN', { hour12: false })
}

onMounted(fetchData)
</script>

<style scoped>
.page-wrap { min-width: 0; }
.page-header { display: flex; align-items: center; justify-content: space-between; gap: 16px; }
.page-title { color: #303133; font-size: 17px; font-weight: 600; }
.page-subtitle { margin-top: 4px; color: #909399; font-size: 12px; }
.filters { display: grid; grid-template-columns: minmax(240px, 1fr) 180px 140px auto auto; gap: 10px; margin-bottom: 16px; }
.app-name { color: #303133; font-weight: 600; }
.app-code { display: inline-block; padding: 2px 7px; border-radius: 4px; background: #f2f3f5; color: #606266; font-size: 12px; }
.description-text { display: -webkit-box; overflow: hidden; color: #606266; line-height: 1.5; -webkit-box-orient: vertical; -webkit-line-clamp: 2; }
.secondary-text { color: #909399; font-size: 12px; }
.pagination-wrap { display: flex; justify-content: flex-end; margin-top: 16px; overflow-x: auto; }
.page-wrap :deep(.el-input-number) { width: 100%; }
@media (max-width: 900px) {
  .filters { grid-template-columns: repeat(2, minmax(160px, 1fr)); }
}
@media (max-width: 700px) {
  .page-header { align-items: stretch; flex-direction: column; }
  .filters { grid-template-columns: 1fr; }
  .page-wrap :deep(.el-card__header), .page-wrap :deep(.el-card__body) { padding: 14px; }
}
</style>
