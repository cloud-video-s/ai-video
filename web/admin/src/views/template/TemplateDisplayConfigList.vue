<template>
  <div class="page-wrap">
    <el-card shadow="never">
      <template #header>
        <div class="page-header">
          <div>
            <div class="page-title">模板展示配置</div>
            <div class="page-subtitle">把具体视频模板配置到客户端展示位置，并独立控制排序和启用状态</div>
          </div>
          <el-button v-if="canAdd" type="primary" @click="openCreate">
            <el-icon><Plus /></el-icon>新增配置
          </el-button>
        </div>
      </template>

      <div class="filters">
        <el-input v-model="query.keyword" clearable placeholder="模板名称或备注" @keyup.enter="handleSearch">
          <template #prefix><el-icon><Search /></el-icon></template>
        </el-input>
        <el-select v-model="query.template_id" clearable filterable placeholder="视频模板">
          <el-option v-for="item in templateOptions" :key="item.id" :label="templateLabel(item)" :value="String(item.id)" />
        </el-select>
        <el-select v-model="query.position_key" clearable filterable placeholder="展示位置">
          <el-option v-for="item in positionOptions" :key="item.position_key" :label="positionLabel(item)" :value="item.position_key" />
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
        <el-table-column label="视频模板" min-width="270">
          <template #default="{ row }">
            <div class="template-cell">
              <el-image class="template-cover" :src="row.template?.cover_image" fit="cover" preview-teleported>
                <template #error><div class="image-error"><el-icon><Picture /></el-icon></div></template>
              </el-image>
              <div class="template-info">
                <div class="primary-text">{{ row.template?.name || `模板 #${row.template_id}` }}</div>
                <div class="secondary-text">ID: {{ row.template_id }}</div>
              </div>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="展示位置" min-width="230">
          <template #default="{ row }">
            <div class="primary-text">{{ row.display_position?.position_name || row.position_key }}</div>
            <el-tag class="position-key" size="small" type="info" effect="plain">{{ row.position_key }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="sort" label="排序" width="80" align="center" />
        <el-table-column label="状态" width="88" align="center">
          <template #default="{ row }">
            <el-tag :type="row.status === 1 ? 'success' : 'danger'">{{ row.status === 1 ? '启用' : '禁用' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="备注" min-width="220">
          <template #default="{ row }">
            <el-tooltip v-if="row.remark" :content="row.remark" placement="top" :show-after="400">
              <div class="remark-text">{{ row.remark }}</div>
            </el-tooltip>
            <span v-else class="secondary-text">暂无备注</span>
          </template>
        </el-table-column>
        <el-table-column prop="updated_at" label="更新时间" width="180" />
        <el-table-column v-if="canEdit || canDelete" label="操作" width="130" fixed="right" align="center">
          <template #default="{ row }">
            <el-button v-if="canEdit" link type="primary" @click="openEdit(row)">编辑</el-button>
            <el-popconfirm v-if="canDelete" title="确认删除该模板展示配置？" width="230" @confirm="handleDelete(row.id)">
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

    <el-dialog v-model="dialogVisible" :title="form.id ? '编辑模板展示配置' : '新增模板展示配置'" width="640px" destroy-on-close>
      <el-form ref="formRef" :model="form" :rules="rules" label-width="96px">
        <el-form-item label="视频模板" prop="template_id">
          <el-select v-model="form.template_id" filterable placeholder="请选择视频模板" style="width: 100%">
            <el-option v-for="item in templateOptions" :key="item.id" :label="templateLabel(item)" :value="item.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="展示位置" prop="position_key">
          <el-select v-model="form.position_key" filterable placeholder="请选择展示位置" style="width: 100%">
            <el-option
              v-for="item in positionOptions"
              :key="item.position_key"
              :label="positionLabel(item)"
              :value="item.position_key"
              :disabled="item.status !== 1 && form.status === 1"
            />
          </el-select>
        </el-form-item>
        <div class="form-grid">
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
        <el-form-item label="备注" prop="remark">
          <el-input v-model="form.remark" type="textarea" :rows="4" maxlength="500" show-word-limit placeholder="填写该展示配置的用途或说明" />
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
import { getDisplayPositionOptions, type DisplayPosition } from '@/api/displayPosition'
import {
  createTemplateDisplayConfig,
  deleteTemplateDisplayConfig,
  getTemplateDisplayConfigList,
  getTemplateOptions,
  updateTemplateDisplayConfig,
  type TemplateDisplayConfig,
  type TemplateDisplayConfigPayload,
  type VideoTemplate,
} from '@/api/template'
import { useUserStore } from '@/store/user'

const userStore = useUserStore()
const canAdd = computed(() => userStore.hasPermission('template:display-config:add'))
const canEdit = computed(() => userStore.hasPermission('template:display-config:edit'))
const canDelete = computed(() => userStore.hasPermission('template:display-config:delete'))

const loading = ref(false)
const submitting = ref(false)
const dialogVisible = ref(false)
const formRef = ref<FormInstance>()
const tableData = ref<TemplateDisplayConfig[]>([])
const templateOptions = ref<VideoTemplate[]>([])
const positionOptions = ref<DisplayPosition[]>([])
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const query = reactive({ keyword: '', template_id: '', position_key: '', status: '' })

const defaultForm: TemplateDisplayConfigPayload & { id: number } = {
  id: 0,
  template_id: 0,
  position_key: '',
  sort: 0,
  status: 1,
  remark: '',
}
const form = reactive({ ...defaultForm })
const rules: FormRules = {
  template_id: [{ required: true, type: 'number', min: 1, message: '请选择视频模板', trigger: 'change' }],
  position_key: [{ required: true, message: '请选择展示位置', trigger: 'change' }],
  remark: [{ max: 500, message: '备注不能超过 500 个字符', trigger: 'blur' }],
}

function templateLabel(item: VideoTemplate) {
  const category = item.video_template_type?.category_name
  return `${item.name}${category ? ` · ${category}` : ''} · #${item.id}`
}

function positionLabel(item: DisplayPosition) {
  return `${item.position_name} · ${item.position_key}${item.status === 1 ? '' : '（已禁用）'}`
}

async function fetchData() {
  loading.value = true
  try {
    const params: Record<string, unknown> = { page: page.value, page_size: pageSize.value }
    for (const [key, value] of Object.entries(query)) {
      if (value !== '') params[key] = value
    }
    const res: any = await getTemplateDisplayConfigList(params)
    tableData.value = res.data?.list || []
    total.value = res.data?.total || 0
  } finally {
    loading.value = false
  }
}

function handleSearch() {
  page.value = 1
  fetchData()
}

function handleReset() {
  Object.assign(query, { keyword: '', template_id: '', position_key: '', status: '' })
  page.value = 1
  fetchData()
}

function openCreate() {
  Object.assign(form, defaultForm)
  dialogVisible.value = true
}

function openEdit(row: TemplateDisplayConfig) {
  Object.assign(form, {
    id: row.id,
    template_id: row.template_id,
    position_key: row.position_key,
    sort: row.sort,
    status: row.status,
    remark: row.remark || '',
  })
  dialogVisible.value = true
}

async function handleSubmit() {
  await formRef.value?.validate()
  submitting.value = true
  try {
    const payload: TemplateDisplayConfigPayload = {
      template_id: form.template_id,
      position_key: form.position_key,
      sort: form.sort,
      status: form.status,
      remark: form.remark.trim(),
    }
    if (form.id) await updateTemplateDisplayConfig(form.id, payload)
    else await createTemplateDisplayConfig(payload)
    ElMessage.success('模板展示配置已保存')
    dialogVisible.value = false
    await fetchData()
  } finally {
    submitting.value = false
  }
}

async function handleDelete(id: number) {
  await deleteTemplateDisplayConfig(id)
  ElMessage.success('模板展示配置已删除')
  if (tableData.value.length === 1 && page.value > 1) page.value--
  await fetchData()
}

async function fetchOptions() {
  const [templateRes, positionRes]: any[] = await Promise.all([getTemplateOptions(), getDisplayPositionOptions()])
  templateOptions.value = Array.isArray(templateRes.data) ? templateRes.data : templateRes.data?.list || []
  positionOptions.value = Array.isArray(positionRes.data) ? positionRes.data : positionRes.data?.list || []
}

onMounted(() => Promise.all([fetchOptions(), fetchData()]))
</script>

<style scoped>
.page-wrap { min-width: 0; }
.page-header { display: flex; align-items: center; justify-content: space-between; gap: 16px; }
.page-title { color: #303133; font-size: 17px; font-weight: 600; }
.page-subtitle { margin-top: 4px; color: #909399; font-size: 12px; }
.filters { display: grid; grid-template-columns: minmax(190px, 1fr) minmax(190px, 1fr) minmax(190px, 1fr) 130px auto auto; gap: 10px; margin-bottom: 16px; }
.template-cell { display: flex; align-items: center; gap: 12px; }
.template-cover { flex: 0 0 auto; width: 88px; height: 54px; border-radius: 6px; background: #f2f3f5; }
.template-info { min-width: 0; }
.image-error { display: flex; align-items: center; justify-content: center; width: 100%; height: 100%; color: #c0c4cc; font-size: 22px; }
.primary-text { overflow: hidden; color: #303133; font-weight: 500; text-overflow: ellipsis; white-space: nowrap; }
.secondary-text { margin-top: 4px; color: #909399; font-size: 12px; }
.position-key { margin-top: 6px; font-family: ui-monospace, SFMono-Regular, Menlo, monospace; }
.remark-text { display: -webkit-box; overflow: hidden; color: #606266; line-height: 1.5; -webkit-box-orient: vertical; -webkit-line-clamp: 2; }
.pagination-wrap { display: flex; justify-content: flex-end; margin-top: 16px; overflow-x: auto; }
.form-grid { display: grid; grid-template-columns: 1fr 1fr; column-gap: 12px; }
.form-grid :deep(.el-input-number) { width: 100%; }
@media (max-width: 960px) {
  .filters { grid-template-columns: repeat(2, minmax(160px, 1fr)); }
}
@media (max-width: 620px) {
  .page-header { align-items: stretch; flex-direction: column; }
  .filters, .form-grid { grid-template-columns: 1fr; }
  .page-wrap :deep(.el-card__header), .page-wrap :deep(.el-card__body) { padding: 14px; }
}
</style>
