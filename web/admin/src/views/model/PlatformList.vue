<template>
  <div class="page-wrap">
    <el-card shadow="never">
      <template #header>
        <div class="page-header">
          <div>
            <div class="page-title">平台管理</div>
            <div class="page-subtitle">维护视频生成服务平台及默认 API 域名</div>
          </div>
          <el-button v-if="canAdd" type="primary" @click="openCreate">
            <el-icon><Plus /></el-icon>新增平台
          </el-button>
        </div>
      </template>

      <div class="filters">
        <el-input v-model="query.keyword" clearable placeholder="平台名称、编码或 API 域名" @keyup.enter="handleSearch">
          <template #prefix><el-icon><Search /></el-icon></template>
        </el-input>
        <el-select v-model="query.status" clearable placeholder="全部状态">
          <el-option label="启用" value="1" />
          <el-option label="禁用" value="0" />
        </el-select>
        <el-button type="primary" plain @click="handleSearch">查询</el-button>
        <el-button @click="handleReset">重置</el-button>
      </div>

      <el-table v-loading="loading" :data="tableData" row-key="id" stripe>
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column label="平台" min-width="210">
          <template #default="{ row }">
            <div class="primary-text">{{ row.name }}</div>
            <code class="code-label">{{ row.code }}</code>
          </template>
        </el-table-column>
        <el-table-column label="API 基础域名" min-width="290">
          <template #default="{ row }">
            <el-link :href="row.base_url" target="_blank" type="primary" :underline="false">{{ row.base_url }}</el-link>
          </template>
        </el-table-column>
        <el-table-column prop="description" label="说明" min-width="220" show-overflow-tooltip>
          <template #default="{ row }">{{ row.description || '-' }}</template>
        </el-table-column>
        <el-table-column label="状态" width="100" align="center">
          <template #default="{ row }">
            <el-tag :type="row.status === 1 ? 'success' : 'info'">{{ row.status === 1 ? '启用' : '禁用' }}</el-tag>
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
              :title="`确认软删除平台 ${row.name}？`"
              width="250"
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

    <el-dialog v-model="dialogVisible" :title="form.id ? '编辑平台' : '新增平台'" width="680px" destroy-on-close>
      <el-form ref="formRef" :model="form" :rules="rules" label-width="105px">
        <div class="form-grid">
          <el-form-item label="平台名称" prop="name">
            <el-input v-model="form.name" maxlength="64" placeholder="例如：ModelVerse" />
          </el-form-item>
          <el-form-item label="平台编码" prop="code">
            <el-input v-model="form.code" maxlength="32" placeholder="例如：modelverse" />
          </el-form-item>
        </div>
        <el-form-item label="基础域名" prop="base_url">
          <el-input v-model="form.base_url" maxlength="255" placeholder="https://api.example.com" />
        </el-form-item>
        <el-form-item label="状态">
          <el-radio-group v-model="form.status">
            <el-radio :value="1">启用</el-radio>
            <el-radio :value="0">禁用</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="说明">
          <el-input v-model="form.description" type="textarea" :rows="3" maxlength="255" show-word-limit placeholder="可选" />
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
  createPlatform,
  deletePlatform,
  getPlatformList,
  updatePlatform,
  type VideoPlatform,
  type VideoPlatformPayload,
} from '@/api/videoModel'
import { useUserStore } from '@/store/user'

const userStore = useUserStore()
const canAdd = computed(() => userStore.hasPermission('platform:add'))
const canEdit = computed(() => userStore.hasPermission('platform:edit'))
const canDelete = computed(() => userStore.hasPermission('platform:delete'))

const loading = ref(false)
const submitting = ref(false)
const dialogVisible = ref(false)
const formRef = ref<FormInstance>()
const tableData = ref<VideoPlatform[]>([])
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const query = reactive({ keyword: '', status: '' })

const defaultForm: VideoPlatformPayload & { id: number } = {
  id: 0,
  name: '',
  code: '',
  base_url: '',
  description: '',
  status: 1,
}
const form = reactive({ ...defaultForm })
const rules: FormRules = {
  name: [{ required: true, message: '请输入平台名称', trigger: 'blur' }],
  code: [
    { required: true, message: '请输入平台编码', trigger: 'blur' },
    { pattern: /^[A-Za-z0-9._-]+$/, message: '仅支持字母、数字、点、下划线和中划线', trigger: 'blur' },
  ],
  base_url: [
    { required: true, message: '请输入平台基础域名', trigger: 'blur' },
    { pattern: /^https?:\/\/[^\s]+$/i, message: '请输入有效的 HTTP(S) 地址', trigger: 'blur' },
  ],
}

async function fetchData() {
  loading.value = true
  try {
    const params: Record<string, unknown> = { page: page.value, page_size: pageSize.value }
    if (query.keyword.trim()) params.keyword = query.keyword.trim()
    if (query.status !== '') params.status = query.status
    const res: any = await getPlatformList(params)
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
  Object.assign(query, { keyword: '', status: '' })
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

function openEdit(row: VideoPlatform) {
  Object.assign(form, {
    id: row.id,
    name: row.name,
    code: row.code,
    base_url: row.base_url,
    description: row.description || '',
    status: row.status,
  })
  dialogVisible.value = true
}

async function handleSubmit() {
  await formRef.value?.validate()
  submitting.value = true
  try {
    const payload: VideoPlatformPayload = {
      name: form.name.trim(),
      code: form.code.trim(),
      base_url: form.base_url.trim().replace(/\/+$/, ''),
      description: form.description.trim(),
      status: form.status,
    }
    if (form.id) await updatePlatform(form.id, payload)
    else await createPlatform(payload)
    ElMessage.success('平台信息已保存')
    dialogVisible.value = false
    await fetchData()
  } finally {
    submitting.value = false
  }
}

async function handleDelete(id: number) {
  await deletePlatform(id)
  ElMessage.success('平台已软删除')
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
.filters { display: grid; grid-template-columns: minmax(280px, 1fr) 150px auto auto; gap: 10px; margin-bottom: 16px; }
.primary-text { color: #303133; font-weight: 600; }
.code-label { display: inline-block; margin-top: 5px; padding: 2px 7px; border-radius: 4px; background: #f5f7fa; color: #606266; font-size: 12px; }
.pagination-wrap { display: flex; justify-content: flex-end; margin-top: 16px; overflow-x: auto; }
.form-grid { display: grid; grid-template-columns: 1fr 1fr; column-gap: 16px; }
@media (max-width: 720px) {
  .page-header { align-items: stretch; flex-direction: column; }
  .filters, .form-grid { grid-template-columns: 1fr; }
}
</style>
