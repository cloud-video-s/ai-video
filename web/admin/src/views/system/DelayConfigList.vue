<template>
  <div class="delay-config-page">
    <el-card shadow="never">
      <template #header>
        <div class="page-header">
          <div class="page-title">OB 延迟配置</div>
          <div class="header-actions">
            <el-button v-if="canSync" :loading="syncing" @click="handleSync">
              <el-icon><Refresh /></el-icon>
              从 YAML 同步
            </el-button>
            <el-button v-if="canAdd" type="primary" @click="openCreate">
              <el-icon><Plus /></el-icon>
              新增配置
            </el-button>
          </div>
        </div>
      </template>

      <div class="toolbar">
        <div class="filters">
          <el-select v-model="query.group" clearable placeholder="全部分组" class="group-filter" @change="handleSearch">
            <el-option v-for="group in groups" :key="group" :label="group" :value="group" />
          </el-select>
          <el-input
            v-model="query.keyword"
            clearable
            placeholder="搜索配置键或说明"
            class="keyword-filter"
            @keyup.enter="handleSearch"
            @clear="handleSearch"
          >
            <template #prefix><el-icon><Search /></el-icon></template>
          </el-input>
          <el-button type="primary" plain @click="handleSearch">查询</el-button>
          <el-button @click="handleReset">重置</el-button>
        </div>
        <el-button
          v-if="canEdit"
          type="primary"
          :disabled="dirtyKeys.length === 0"
          :loading="saving"
          @click="handleSaveValues"
        >
          <el-icon><Check /></el-icon>
          保存修改<span v-if="dirtyKeys.length">（{{ dirtyKeys.length }}）</span>
        </el-button>
      </div>

      <el-table :data="tableData" v-loading="loading" row-key="id" stripe class="config-table">
        <el-table-column prop="sort" label="排序" width="72" align="center" />
        <el-table-column label="配置项" min-width="250">
          <template #default="{ row }">
            <div class="config-name">
              <code>{{ row.key }}</code>
              <el-tag v-if="dirtyKeys.includes(row.key)" size="small" type="warning" effect="plain">待保存</el-tag>
            </div>
            <div class="config-remark">{{ row.remark || '暂无说明' }}</div>
          </template>
        </el-table-column>
        <el-table-column prop="group" label="分组" width="110">
          <template #default="{ row }"><el-tag size="small" type="info">{{ row.group }}</el-tag></template>
        </el-table-column>
        <el-table-column prop="type" label="类型" width="86" align="center">
          <template #default="{ row }"><el-tag size="small" :type="typeTag(row.type)">{{ row.type }}</el-tag></template>
        </el-table-column>
        <el-table-column label="配置值" min-width="270">
          <template #default="{ row }">
            <el-radio-group
              v-if="row.type === 'bool' && optionValues(row).length"
              :model-value="row.value"
              :disabled="!canEdit"
              size="small"
              class="value-control bool-control"
              @update:model-value="(value: string | number | boolean | undefined) => setRowValue(row, String(value ?? ''))"
            >
              <el-radio-button v-for="option in optionValues(row)" :key="option" :value="option">
                {{ option }}
              </el-radio-button>
            </el-radio-group>
            <el-select
              v-else-if="optionValues(row).length"
              :model-value="row.value"
              :disabled="!canEdit"
              class="value-control"
              @update:model-value="(value: string) => setRowValue(row, value)"
            >
              <el-option v-for="option in optionValues(row)" :key="option" :label="option" :value="option" />
            </el-select>
            <el-input-number
              v-else-if="row.type === 'int'"
              :model-value="Number(row.value)"
              :min="0"
              :max="86400"
              :disabled="!canEdit"
              controls-position="right"
              class="value-control"
              @update:model-value="(value: number | undefined) => setRowValue(row, String(value ?? 0))"
            />
            <el-input
              v-else
              :model-value="row.value"
              :disabled="!canEdit"
              class="value-control"
              @update:model-value="(value: string) => setRowValue(row, value)"
            />
          </template>
        </el-table-column>
        <el-table-column label="更新时间" width="170">
          <template #default="{ row }">{{ formatTime(row.updated_at) }}</template>
        </el-table-column>
        <el-table-column v-if="canEdit || canDelete" label="操作" width="130" fixed="right">
          <template #default="{ row }">
            <el-tooltip content="编辑配置" placement="top">
              <el-button v-if="canEdit" link type="primary" aria-label="编辑配置" @click="openEdit(row)">
                <el-icon><Edit /></el-icon>
              </el-button>
            </el-tooltip>
            <el-popconfirm v-if="canDelete" title="确认删除该延迟配置？" @confirm="handleDelete(row.id)">
              <template #reference>
                <el-tooltip content="删除配置" placement="top">
                  <el-button link type="danger" aria-label="删除配置"><el-icon><Delete /></el-icon></el-button>
                </el-tooltip>
              </template>
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
      :title="dialogMode === 'create' ? '新增延迟配置' : '编辑延迟配置'"
      width="560px"
      destroy-on-close
    >
      <el-form ref="formRef" :model="form" :rules="rules" label-width="88px">
        <el-form-item label="分组" prop="group"><el-input v-model="form.group" maxlength="64" /></el-form-item>
        <el-form-item label="配置键" prop="key">
          <el-input v-model="form.key" maxlength="128" :disabled="dialogMode === 'edit'" />
        </el-form-item>
        <el-form-item label="类型" prop="type"><el-segmented v-model="form.type" :options="typeOptions" /></el-form-item>
        <el-form-item label="配置值" prop="value">
          <el-select v-if="formOptions.length" v-model="form.value" style="width: 100%">
            <el-option v-for="option in formOptions" :key="option" :label="option" :value="option" />
          </el-select>
          <el-input-number
            v-else-if="form.type === 'int'"
            :model-value="Number(form.value || 0)"
            :min="0"
            :max="86400"
            controls-position="right"
            style="width: 100%"
            @update:model-value="(value: number | undefined) => (form.value = String(value ?? 0))"
          />
          <el-input v-else v-model="form.value" maxlength="64" />
        </el-form-item>
        <el-form-item label="可选值"><el-input v-model="form.options" placeholder='JSON 数组，例如 ["0","1"]' maxlength="255" /></el-form-item>
        <el-form-item label="排序"><el-input-number v-model="form.sort" :min="0" :max="9999" /></el-form-item>
        <el-form-item label="说明"><el-input v-model="form.remark" type="textarea" :rows="3" maxlength="255" show-word-limit /></el-form-item>
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
import { ElMessage, ElMessageBox, type FormInstance } from 'element-plus'
import {
  batchUpdateDelayConfigValues, createDelayConfig, deleteDelayConfig, getDelayConfigGroups,
  getDelayConfigList, syncDelayConfigs, updateDelayConfig,
  type DelayConfig, type DelayConfigPayload,
} from '@/api/delayConfig'
import { useUserStore } from '@/store/user'

const userStore = useUserStore()
const canAdd = computed(() => userStore.hasPermission('system:delay-config:add'))
const canEdit = computed(() => userStore.hasPermission('system:delay-config:edit'))
const canDelete = computed(() => userStore.hasPermission('system:delay-config:delete'))
const canSync = computed(() => userStore.hasPermission('system:delay-config:sync'))

const loading = ref(false)
const saving = ref(false)
const syncing = ref(false)
const submitting = ref(false)
const tableData = ref<DelayConfig[]>([])
const groups = ref<string[]>([])
const dirtyKeys = ref<string[]>([])
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const query = reactive({ group: '', keyword: '' })

async function fetchData() {
  loading.value = true
  try {
    const res: any = await getDelayConfigList({
      page: page.value, page_size: pageSize.value, group: query.group || undefined,
      keyword: query.keyword.trim() || undefined,
    })
    tableData.value = res.data.list || []
    total.value = res.data.total || 0
    dirtyKeys.value = []
  } finally {
    loading.value = false
  }
}

async function fetchGroups() {
  const res: any = await getDelayConfigGroups()
  groups.value = res.data || []
}

function handleSearch() { page.value = 1; fetchData() }
function handleReset() { query.group = ''; query.keyword = ''; page.value = 1; fetchData() }

function setRowValue(row: DelayConfig, value: string) {
  row.value = value
  if (!dirtyKeys.value.includes(row.key)) dirtyKeys.value = [...dirtyKeys.value, row.key]
}

async function handleSaveValues() {
  const items = tableData.value.filter((row) => dirtyKeys.value.includes(row.key)).map((row) => ({ key: row.key, value: String(row.value) }))
  if (!items.length) return
  saving.value = true
  try {
    await batchUpdateDelayConfigValues(items)
    ElMessage.success('延迟配置已保存')
    await fetchData()
  } finally { saving.value = false }
}

async function handleSync() {
  await ElMessageBox.confirm('同步后将以 YAML 文件内容覆盖同名配置，确认继续？', '同步配置', { type: 'warning' })
  syncing.value = true
  try {
    await syncDelayConfigs()
    ElMessage.success('已从 YAML 同步')
    await Promise.all([fetchData(), fetchGroups()])
  } finally { syncing.value = false }
}

async function handleDelete(id: number) {
  await deleteDelayConfig(id)
  ElMessage.success('配置已删除')
  await Promise.all([fetchData(), fetchGroups()])
}

function optionValues(row: DelayConfig): string[] { return parseOptions(row.options) }
function parseOptions(raw: string): string[] {
  if (!raw) return []
  try { const values = JSON.parse(raw); return Array.isArray(values) ? values.map(String) : [] } catch { return [] }
}
function typeTag(type: DelayConfig['type']): 'success' | 'warning' | 'info' {
  if (type === 'bool') return 'success'
  if (type === 'int') return 'warning'
  return 'info'
}
function formatTime(value: string) { return value ? new Date(value).toLocaleString('zh-CN', { hour12: false }) : '-' }

const dialogVisible = ref(false)
const dialogMode = ref<'create' | 'edit'>('create')
const editingID = ref(0)
const formRef = ref<FormInstance>()
const typeOptions = [
  { label: '字符串', value: 'string' }, { label: '整数', value: 'int' }, { label: '布尔', value: 'bool' },
]
const defaultForm: DelayConfigPayload & { key: string } = {
  group: 'config', key: '', value: '0', type: 'int', options: '', remark: '', sort: 0,
}
const form = reactive({ ...defaultForm })
const formOptions = computed(() => parseOptions(form.options))
const rules = {
  group: [{ required: true, message: '请输入分组', trigger: 'blur' }],
  key: [{ required: true, message: '请输入配置键', trigger: 'blur' }],
  type: [{ required: true, message: '请选择类型', trigger: 'change' }],
  value: [{ required: true, message: '请输入配置值', trigger: 'blur' }],
}

function openCreate() { dialogMode.value = 'create'; editingID.value = 0; Object.assign(form, defaultForm); dialogVisible.value = true }
function openEdit(row: DelayConfig) {
  dialogMode.value = 'edit'; editingID.value = row.id
  Object.assign(form, { group: row.group, key: row.key, value: row.value, type: row.type, options: row.options || '', remark: row.remark || '', sort: row.sort || 0 })
  dialogVisible.value = true
}

async function handleSubmit() {
  await formRef.value?.validate()
  submitting.value = true
  try {
    const payload: DelayConfigPayload = {
      group: form.group.trim(), value: String(form.value).trim(), type: form.type,
      options: form.options.trim(), remark: form.remark.trim(), sort: form.sort,
    }
    if (dialogMode.value === 'create') await createDelayConfig({ ...payload, key: form.key.trim() })
    else await updateDelayConfig(editingID.value, payload)
    ElMessage.success('配置已保存')
    dialogVisible.value = false
    await Promise.all([fetchData(), fetchGroups()])
  } finally { submitting.value = false }
}

onMounted(() => Promise.all([fetchData(), fetchGroups()]))
</script>

<style scoped>
.delay-config-page { min-width: 0; }
.page-header, .toolbar, .filters, .header-actions, .config-name { display: flex; align-items: center; }
.page-header, .toolbar { justify-content: space-between; gap: 16px; }
.page-title { font-size: 16px; font-weight: 600; color: #303133; }
.header-actions, .filters, .config-name { gap: 8px; }
.toolbar { margin-bottom: 16px; flex-wrap: wrap; }
.filters { flex-wrap: wrap; }
.group-filter { width: 150px; }
.keyword-filter { width: 260px; }
.config-name code { color: #303133; font-size: 13px; overflow-wrap: anywhere; }
.config-remark { margin-top: 5px; color: #909399; font-size: 12px; line-height: 1.45; }
.value-control { width: min(220px, 100%); }
.bool-control { width: auto; }
.pagination-wrap { display: flex; justify-content: flex-end; margin-top: 16px; overflow-x: auto; }
.delay-config-page :deep(.el-dialog) { max-width: calc(100vw - 32px); }
@media (max-width: 760px) {
  .page-header, .toolbar { align-items: stretch; flex-direction: column; }
  .header-actions, .filters { width: 100%; }
  .group-filter, .keyword-filter { width: 100%; }
  .delay-config-page :deep(.el-card__header),
  .delay-config-page :deep(.el-card__body) { padding: 14px; }
}
</style>
