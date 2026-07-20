<template>
  <div class="page-wrap">
    <el-card shadow="never">
      <template #header>
        <div class="page-header">
          <div>
            <div class="page-title">全球国家管理</div>
            <div class="page-subtitle">维护 ISO 3166-1 两位国家/地区代码、中文名称与可用状态</div>
          </div>
          <el-button v-if="canAdd" type="primary" :icon="Plus" @click="openCreate">新增国家</el-button>
        </div>
      </template>

      <div class="toolbar">
        <el-input
          v-model="query.keyword"
          clearable
          :prefix-icon="Search"
          placeholder="搜索国家代码或中文名称"
          @keyup.enter="handleSearch"
          @clear="handleSearch"
        />
        <el-select v-model="query.status" clearable placeholder="全部状态" @change="handleSearch">
          <el-option label="启用" :value="1" />
          <el-option label="禁用" :value="0" />
        </el-select>
        <el-button type="primary" @click="handleSearch">查询</el-button>
        <el-button @click="handleReset">重置</el-button>
        <div class="record-count">共 {{ total }} 个国家及地区</div>
      </div>

      <el-table v-loading="loading" :data="tableData" row-key="id" stripe>
        <el-table-column label="国家标识" width="170">
          <template #default="{ row }">
            <div class="country-identity">
              <span class="country-flag" aria-hidden="true">{{ countryFlag(row.code) }}</span>
              <span class="country-code">{{ row.code }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="name_zh" label="中文名称" min-width="240">
          <template #default="{ row }"><span class="country-name">{{ row.name_zh }}</span></template>
        </el-table-column>
        <el-table-column label="状态" width="150" align="center">
          <template #default="{ row }">
            <el-switch
              v-if="canEdit"
              v-model="row.status"
              :active-value="1"
              :inactive-value="0"
              active-text="启用"
              inactive-text="禁用"
              inline-prompt
              :loading="updatingIds.includes(row.id)"
              @change="handleStatusChange(row)"
            />
            <el-tag v-else :type="row.status === 1 ? 'success' : 'info'" effect="light">
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
              :title="`确认删除 ${row.code} / ${row.name_zh}？`"
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

    <el-dialog
      v-model="dialogVisible"
      :title="form.id ? '编辑国家' : '新增国家'"
      width="500px"
      destroy-on-close
      @closed="formRef?.clearValidate()"
    >
      <el-form ref="formRef" :model="form" :rules="rules" label-width="92px">
        <el-form-item label="国家代码" prop="code">
          <el-input
            v-model="form.code"
            maxlength="2"
            show-word-limit
            placeholder="例如：CN、US"
            @input="normalizeFormCode"
          >
            <template #prefix><span class="input-flag">{{ countryFlag(form.code) }}</span></template>
          </el-input>
          <div class="form-tip">采用 ISO 3166-1 alpha-2 两位大写代码</div>
        </el-form-item>
        <el-form-item label="中文名称" prop="name_zh">
          <el-input v-model="form.name_zh" maxlength="100" placeholder="例如：中国、美国" />
        </el-form-item>
        <el-form-item label="状态">
          <el-radio-group v-model="form.status">
            <el-radio :value="1">启用</el-radio>
            <el-radio :value="0">禁用</el-radio>
          </el-radio-group>
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
import { Plus, Search } from '@element-plus/icons-vue'
import {
  createCountry,
  deleteCountry,
  getCountryList,
  updateCountry,
  updateCountryStatus,
  type Country,
  type CountryPayload,
} from '@/api/country'
import { useUserStore } from '@/store/user'

const userStore = useUserStore()
const canAdd = computed(() => userStore.hasPermission('system:country:add'))
const canEdit = computed(() => userStore.hasPermission('system:country:edit'))
const canDelete = computed(() => userStore.hasPermission('system:country:delete'))

const loading = ref(false)
const submitting = ref(false)
const dialogVisible = ref(false)
const updatingIds = ref<number[]>([])
const formRef = ref<FormInstance>()
const tableData = ref<Country[]>([])
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const query = reactive<{ keyword: string; status: '' | number }>({ keyword: '', status: '' })

const defaultForm: CountryPayload & { id: number } = { id: 0, code: '', name_zh: '', status: 1 }
const form = reactive({ ...defaultForm })
const rules: FormRules = {
  code: [
    { required: true, message: '请输入国家代码', trigger: 'blur' },
    { pattern: /^[A-Z]{2}$/, message: '请输入 2 位大写英文字母', trigger: 'blur' },
  ],
  name_zh: [{ required: true, message: '请输入中文名称', trigger: 'blur' }],
}

async function fetchData() {
  loading.value = true
  try {
    const params: Record<string, unknown> = { page: page.value, page_size: pageSize.value }
    if (query.keyword.trim()) params.keyword = query.keyword.trim()
    if (query.status !== '') params.status = query.status
    const res: any = await getCountryList(params)
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

function openEdit(row: Country) {
  Object.assign(form, { id: row.id, code: row.code, name_zh: row.name_zh, status: row.status })
  dialogVisible.value = true
}

function normalizeFormCode(value: string) {
  form.code = value.replace(/[^a-zA-Z]/g, '').toUpperCase().slice(0, 2)
}

async function handleSubmit() {
  await formRef.value?.validate()
  submitting.value = true
  try {
    const payload: CountryPayload = {
      code: form.code.trim().toUpperCase(),
      name_zh: form.name_zh.trim(),
      status: form.status,
    }
    if (form.id) await updateCountry(form.id, payload)
    else await createCountry(payload)
    ElMessage.success('国家信息已保存')
    dialogVisible.value = false
    await fetchData()
  } finally {
    submitting.value = false
  }
}

async function handleStatusChange(row: Country) {
  updatingIds.value.push(row.id)
  try {
    await updateCountryStatus(row.id, row.status)
    ElMessage.success(`${row.code} 已${row.status === 1 ? '启用' : '禁用'}`)
  } catch {
    row.status = row.status === 1 ? 0 : 1
  } finally {
    updatingIds.value = updatingIds.value.filter((id) => id !== row.id)
  }
}

async function handleDelete(id: number) {
  await deleteCountry(id)
  ElMessage.success('国家已删除')
  if (tableData.value.length === 1 && page.value > 1) page.value--
  await fetchData()
}

function countryFlag(code: string) {
  const normalized = code.trim().toUpperCase()
  if (!/^[A-Z]{2}$/.test(normalized)) return '🌐'
  return [...normalized].map((letter) => String.fromCodePoint(127397 + letter.charCodeAt(0))).join('')
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
.toolbar { display: grid; grid-template-columns: minmax(220px, 360px) 150px auto auto 1fr; gap: 10px; align-items: center; margin-bottom: 16px; }
.record-count { justify-self: end; color: #909399; font-size: 13px; }
.country-identity { display: flex; align-items: center; gap: 10px; }
.country-flag { width: 30px; font-size: 24px; line-height: 1; text-align: center; }
.country-code { min-width: 44px; padding: 4px 10px; border: 1px solid #dcdfe6; border-radius: 6px; background: #f5f7fa; color: #303133; font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace; font-size: 13px; font-weight: 700; letter-spacing: 1px; text-align: center; }
.country-name { color: #303133; font-weight: 500; }
.pagination-wrap { display: flex; justify-content: flex-end; margin-top: 16px; overflow-x: auto; }
.form-tip { margin-top: 5px; color: #909399; font-size: 12px; line-height: 1.4; }
.input-flag { display: inline-block; min-width: 20px; font-size: 18px; }
@media (max-width: 900px) {
  .toolbar { grid-template-columns: 1fr 140px auto auto; }
  .record-count { display: none; }
}
@media (max-width: 620px) {
  .page-header { align-items: stretch; flex-direction: column; }
  .toolbar { grid-template-columns: 1fr 1fr; }
  .toolbar :deep(.el-input) { grid-column: 1 / -1; }
  .page-wrap :deep(.el-card__header), .page-wrap :deep(.el-card__body) { padding: 14px; }
}
</style>
