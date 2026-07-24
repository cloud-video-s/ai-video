<template>
  <div class="page-wrap">
    <el-card shadow="never">
      <template #header>
        <div class="page-header">
          <div>
            <div class="page-title">VIP 等级管理</div>
            <div class="page-subtitle">维护订阅套餐使用的 VIP 等级、说明和排序</div>
          </div>
          <el-button v-if="canAdd" type="primary" @click="openCreate">
            <el-icon><Plus /></el-icon>新增 VIP 等级
          </el-button>
        </div>
      </template>

      <div class="filters">
        <el-input v-model="query.keyword" clearable placeholder="等级名称或描述" @keyup.enter="handleSearch">
          <template #prefix><el-icon><Search /></el-icon></template>
        </el-input>
        <el-select v-model="query.status" clearable placeholder="启用状态">
          <el-option label="启用" value="1" />
          <el-option label="禁用" value="0" />
        </el-select>
        <el-button type="primary" plain @click="handleSearch">查询</el-button>
        <el-button @click="handleReset">重置</el-button>
      </div>

      <el-table v-loading="loading" :data="tableData" row-key="id" stripe>
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="level" label="VIP 等级" min-width="180">
          <template #default="{ row }"><span class="level-name">{{ row.level }}</span></template>
        </el-table-column>
        <el-table-column label="等级说明" min-width="300">
          <template #default="{ row }">
            <el-tooltip v-if="row.description" :content="row.description" placement="top" :show-after="400">
              <div class="description-text">{{ row.description }}</div>
            </el-tooltip>
            <span v-else class="empty-text">暂无说明</span>
          </template>
        </el-table-column>
        <el-table-column prop="sort" label="排序" width="100" align="center" />
        <el-table-column label="状态" width="150" align="center">
          <template #default="{ row }">
            <el-switch
              v-if="canEdit"
              v-model="row.status"
              :active-value="1"
              :inactive-value="0"
              inline-prompt
              active-text="启用"
              inactive-text="禁用"
              :loading="updatingIds.includes(row.id)"
              @change="handleStatusChange(row)"
            />
            <el-tag v-else :type="row.status === 1 ? 'success' : 'danger'">
              {{ row.status === 1 ? '启用' : '禁用' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="更新时间" width="180">
          <template #default="{ row }">{{ formatDate(row.updated_at) }}</template>
        </el-table-column>
        <el-table-column v-if="canEdit || canDelete" label="操作" width="140" fixed="right" align="center">
          <template #default="{ row }">
            <el-button v-if="canEdit" link type="primary" @click="openEdit(row)">编辑</el-button>
            <el-popconfirm
              v-if="canDelete"
              title="确认删除该 VIP 等级？被订阅套餐使用时无法删除。"
              width="280"
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

    <el-dialog v-model="dialogVisible" :title="form.id ? '编辑 VIP 等级' : '新增 VIP 等级'" width="560px" destroy-on-close>
      <el-form ref="formRef" :model="form" :rules="rules" label-width="96px">
        <el-form-item label="等级名称" prop="level">
          <el-input v-model="form.level" maxlength="255" show-word-limit placeholder="例如：黄金会员" />
        </el-form-item>
        <el-form-item label="排序">
          <el-input-number v-model="form.sort" :min="0" :max="999999999999" controls-position="right" />
        </el-form-item>
        <el-form-item label="状态">
          <el-radio-group v-model="form.status">
            <el-radio :value="1">启用</el-radio>
            <el-radio :value="0">禁用</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="等级说明">
          <el-input v-model="form.description" type="textarea" :rows="5" placeholder="请输入等级权益或用途说明" />
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
  createVIPSubscriptionLevel,
  deleteVIPSubscriptionLevel,
  getVIPSubscriptionLevelList,
  updateVIPSubscriptionLevel,
  updateVIPSubscriptionLevelStatus,
  type VIPSubscriptionLevel,
  type VIPSubscriptionLevelPayload,
} from '@/api/vipSubscriptionLevel'
import { useUserStore } from '@/store/user'

const userStore = useUserStore()
const canAdd = computed(() => userStore.hasPermission('subscription:vip-level:add'))
const canEdit = computed(() => userStore.hasPermission('subscription:vip-level:edit'))
const canDelete = computed(() => userStore.hasPermission('subscription:vip-level:delete'))

const loading = ref(false)
const submitting = ref(false)
const dialogVisible = ref(false)
const formRef = ref<FormInstance>()
const tableData = ref<VIPSubscriptionLevel[]>([])
const updatingIds = ref<number[]>([])
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const query = reactive({ keyword: '', status: '' })
const defaultForm = { id: 0, level: '', description: '', status: 1, sort: 0 }
const form = reactive({ ...defaultForm })
const rules: FormRules = {
  level: [{ required: true, whitespace: true, message: '请输入 VIP 等级名称', trigger: 'blur' }],
}

async function fetchData() {
  loading.value = true
  try {
    const params: Record<string, unknown> = { page: page.value, page_size: pageSize.value }
    if (query.keyword.trim()) params.keyword = query.keyword.trim()
    if (query.status !== '') params.status = query.status
    const res: any = await getVIPSubscriptionLevelList(params)
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

function openEdit(row: VIPSubscriptionLevel) {
  Object.assign(form, {
    id: row.id,
    level: row.level,
    description: row.description || '',
    status: row.status,
    sort: Number(row.sort || 0),
  })
  dialogVisible.value = true
}

async function handleSubmit() {
  await formRef.value?.validate()
  submitting.value = true
  try {
    const payload: VIPSubscriptionLevelPayload = {
      level: form.level.trim(),
      description: form.description.trim(),
      status: form.status,
      sort: Number(form.sort),
    }
    if (form.id) await updateVIPSubscriptionLevel(form.id, payload)
    else await createVIPSubscriptionLevel(payload)
    ElMessage.success('VIP 等级已保存')
    dialogVisible.value = false
    await fetchData()
  } finally {
    submitting.value = false
  }
}

async function handleStatusChange(row: VIPSubscriptionLevel) {
  updatingIds.value.push(row.id)
  try {
    await updateVIPSubscriptionLevelStatus(row.id, row.status)
    ElMessage.success(`VIP 等级已${row.status === 1 ? '启用' : '禁用'}`)
  } catch {
    row.status = row.status === 1 ? 0 : 1
  } finally {
    updatingIds.value = updatingIds.value.filter((id) => id !== row.id)
  }
}

async function handleDelete(id: number) {
  await deleteVIPSubscriptionLevel(id)
  ElMessage.success('VIP 等级已删除')
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
.filters { display: grid; grid-template-columns: minmax(260px, 1fr) 180px auto auto; gap: 10px; max-width: 820px; margin-bottom: 16px; }
.level-name { color: #303133; font-weight: 600; }
.description-text { display: -webkit-box; overflow: hidden; color: #606266; line-height: 1.5; -webkit-box-orient: vertical; -webkit-line-clamp: 2; }
.empty-text { color: #909399; font-size: 12px; }
.pagination-wrap { display: flex; justify-content: flex-end; margin-top: 16px; overflow-x: auto; }
:deep(.el-input-number) { width: 100%; }
@media (max-width: 700px) {
  .page-header { align-items: stretch; flex-direction: column; }
  .filters { grid-template-columns: 1fr; max-width: none; }
  .page-wrap :deep(.el-card__header), .page-wrap :deep(.el-card__body) { padding: 14px; }
}
</style>
