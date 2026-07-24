<template>
  <div class="page-wrap">
    <el-card shadow="never">
      <template #header>
        <div class="page-header">
          <div>
            <div class="page-title">API 管理</div>
            <div class="page-subtitle">维护后台接口元数据，菜单绑定接口后会自动同步角色访问策略。</div>
          </div>
          <el-button v-if="canAdd" type="primary" @click="openCreate">新增 API</el-button>
        </div>
      </template>

      <div class="filters">
        <el-input v-model="query.keyword" clearable placeholder="路径、分组或说明" @keyup.enter="handleSearch" />
        <el-select v-model="query.group" clearable filterable allow-create placeholder="接口分组">
          <el-option v-for="group in groupOptions" :key="group" :label="group" :value="group" />
        </el-select>
        <el-select v-model="query.method" clearable placeholder="请求方法">
          <el-option v-for="method in methods" :key="method" :label="method" :value="method" />
        </el-select>
        <el-button type="primary" plain @click="handleSearch">查询</el-button>
        <el-button @click="handleReset">重置</el-button>
      </div>

      <el-table v-loading="loading" :data="tableData" row-key="id" stripe>
        <el-table-column prop="id" label="ID" width="76" />
        <el-table-column label="方法" width="92" align="center">
          <template #default="{ row }"><el-tag :type="methodTag(row.method)">{{ row.method }}</el-tag></template>
        </el-table-column>
        <el-table-column prop="path" label="接口路径" min-width="300">
          <template #default="{ row }"><span class="mono">{{ row.path }}</span></template>
        </el-table-column>
        <el-table-column prop="group" label="分组" width="150" />
        <el-table-column prop="description" label="说明" min-width="220" show-overflow-tooltip />
        <el-table-column prop="updated_at" label="更新时间" width="180" />
        <el-table-column v-if="canEdit || canDelete" label="操作" width="130" fixed="right" align="center">
          <template #default="{ row }">
            <el-button v-if="canEdit" link type="primary" @click="openEdit(row)">编辑</el-button>
            <el-popconfirm v-if="canDelete" title="删除后会解除所有菜单绑定，确认删除？" width="260" @confirm="handleDelete(row.id)">
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

    <el-dialog v-model="dialogVisible" :title="form.id ? '编辑 API' : '新增 API'" width="620px" destroy-on-close>
      <el-form ref="formRef" :model="form" :rules="rules" label-width="92px">
        <el-form-item label="请求方法" prop="method">
          <el-select v-model="form.method" style="width: 100%">
            <el-option v-for="method in methods" :key="method" :label="method" :value="method" />
          </el-select>
        </el-form-item>
        <el-form-item label="接口路径" prop="path">
          <el-input v-model="form.path" maxlength="255" placeholder="例如 /admin/users/:id" />
        </el-form-item>
        <el-form-item label="接口分组" prop="group">
          <el-select v-model="form.group" filterable allow-create clearable style="width: 100%">
            <el-option v-for="group in groupOptions" :key="group" :label="group" :value="group" />
          </el-select>
        </el-form-item>
        <el-form-item label="接口说明" prop="description">
          <el-input v-model="form.description" type="textarea" :rows="3" maxlength="255" show-word-limit />
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
import { createAPI, deleteAPI, getAPIList, updateAPI, type AdminAPI, type AdminAPIPayload } from '@/api/api'
import { useUserStore } from '@/store/user'

const methods = ['GET', 'POST', 'PUT', 'PATCH', 'DELETE', 'OPTIONS', 'HEAD']
const userStore = useUserStore()
const canAdd = computed(() => userStore.hasPermission('system:api:add'))
const canEdit = computed(() => userStore.hasPermission('system:api:edit'))
const canDelete = computed(() => userStore.hasPermission('system:api:delete'))

const loading = ref(false)
const submitting = ref(false)
const dialogVisible = ref(false)
const formRef = ref<FormInstance>()
const tableData = ref<AdminAPI[]>([])
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const query = reactive({ keyword: '', group: '', method: '' })
const defaultForm: AdminAPIPayload & { id: number } = { id: 0, path: '', method: 'GET', group: '', description: '' }
const form = reactive({ ...defaultForm })

const groupOptions = computed(() => [...new Set(tableData.value.map((item) => item.group).filter(Boolean))].sort())
const rules: FormRules = {
  path: [
    { required: true, message: '请输入接口路径', trigger: 'blur' },
    { pattern: /^\//, message: '接口路径必须以 / 开头', trigger: 'blur' },
  ],
  method: [{ required: true, message: '请选择请求方法', trigger: 'change' }],
}

async function fetchData() {
  loading.value = true
  try {
    const res: any = await getAPIList({ page: page.value, page_size: pageSize.value, ...query })
    tableData.value = res.data?.list || []
    total.value = Number(res.data?.total) || 0
  } finally {
    loading.value = false
  }
}

function handleSearch() { page.value = 1; fetchData() }
function handleReset() { Object.assign(query, { keyword: '', group: '', method: '' }); page.value = 1; fetchData() }
function handlePageSizeChange() { page.value = 1; fetchData() }
function openCreate() { Object.assign(form, defaultForm); dialogVisible.value = true }
function openEdit(row: AdminAPI) {
  Object.assign(form, { id: row.id, path: row.path, method: row.method, group: row.group || '', description: row.description || '' })
  dialogVisible.value = true
}

async function handleSubmit() {
  await formRef.value?.validate()
  submitting.value = true
  try {
    const payload: AdminAPIPayload = {
      path: form.path.trim(), method: form.method.toUpperCase(),
      group: form.group.trim(), description: form.description.trim(),
    }
    if (form.id) await updateAPI(form.id, payload)
    else await createAPI(payload)
    ElMessage.success('API 已保存')
    dialogVisible.value = false
    await fetchData()
  } finally {
    submitting.value = false
  }
}

async function handleDelete(id: number) {
  await deleteAPI(id)
  ElMessage.success('API 已删除')
  if (tableData.value.length === 1 && page.value > 1) page.value--
  await fetchData()
}

function methodTag(method: string) {
  if (method === 'GET') return 'success'
  if (method === 'DELETE') return 'danger'
  if (method === 'POST') return 'primary'
  if (method === 'PATCH') return 'warning'
  return 'info'
}

onMounted(fetchData)
</script>

<style scoped>
.page-header { display: flex; align-items: center; justify-content: space-between; gap: 16px; }
.page-title { color: var(--el-text-color-primary); font-size: 17px; font-weight: 600; }
.page-subtitle { margin-top: 4px; color: var(--el-text-color-secondary); font-size: 12px; }
.filters { display: grid; grid-template-columns: minmax(220px, 2fr) minmax(150px, 1fr) 130px auto auto; gap: 10px; margin-bottom: 16px; }
.mono { font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace; }
.pagination-wrap { display: flex; justify-content: flex-end; margin-top: 16px; }
@media (max-width: 800px) { .filters { grid-template-columns: 1fr; }.page-header { align-items: stretch; flex-direction: column; } }
</style>
