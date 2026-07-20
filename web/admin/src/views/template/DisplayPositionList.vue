<template>
  <div class="page-wrap">
    <el-card shadow="never">
      <template #header>
        <div class="page-header">
          <div>
            <div class="page-title">展示位置</div>
            <div class="page-subtitle">配置视频模板在客户端中的展示入口、封面与启用状态</div>
          </div>
          <el-button v-if="canAdd" type="primary" @click="openCreate">
            <el-icon><Plus /></el-icon>新增展示位置
          </el-button>
        </div>
      </template>

      <div class="filters">
        <el-input v-model="query.keyword" clearable placeholder="位置名称、标识或描述" @keyup.enter="handleSearch">
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
        <el-table-column prop="id" label="ID" width="72" />
        <el-table-column label="封面图" width="132" align="center">
          <template #default="{ row }">
            <el-image
              class="cover-image"
              :src="row.cover_image"
              :preview-src-list="[row.cover_image]"
              preview-teleported
              fit="cover"
            >
              <template #error><div class="image-error"><el-icon><Picture /></el-icon></div></template>
            </el-image>
          </template>
        </el-table-column>
        <el-table-column label="展示位置" min-width="190">
          <template #default="{ row }">
            <div class="primary-text">{{ row.position_name }}</div>
            <el-tag class="position-key" size="small" type="info" effect="plain">{{ row.position_key }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="位置描述" min-width="260">
          <template #default="{ row }">
            <el-tooltip v-if="row.description" :content="row.description" placement="top" :show-after="400">
              <div class="description-text">{{ row.description }}</div>
            </el-tooltip>
            <span v-else class="secondary-text">暂无描述</span>
          </template>
        </el-table-column>
        <el-table-column prop="sort" label="排序" width="80" align="center" />
        <el-table-column label="状态" width="88" align="center">
          <template #default="{ row }">
            <el-tag :type="row.status === 1 ? 'success' : 'danger'">
              {{ row.status === 1 ? '启用' : '禁用' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="updated_at" label="更新时间" width="180" />
        <el-table-column v-if="canEdit || canDelete" label="操作" width="130" fixed="right" align="center">
          <template #default="{ row }">
            <el-button v-if="canEdit" link type="primary" @click="openEdit(row)">编辑</el-button>
            <el-popconfirm
              v-if="canDelete"
              title="确认删除该展示位置？被视频模板使用时将无法删除。"
              width="270"
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
          @size-change="fetchData"
          @current-change="fetchData"
        />
      </div>
    </el-card>

    <el-dialog
      v-model="dialogVisible"
      :title="form.id ? '编辑展示位置' : '新增展示位置'"
      width="680px"
      destroy-on-close
    >
      <el-form ref="formRef" :model="form" :rules="rules" label-width="92px">
        <div class="form-grid">
          <el-form-item label="展示位置" prop="position_name">
            <el-input v-model="form.position_name" maxlength="128" placeholder="例如：首页热门" />
          </el-form-item>
          <el-form-item label="位置标识" prop="position_key">
            <el-input v-model="form.position_key" maxlength="64" placeholder="例如：home_hot" />
          </el-form-item>
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
        <el-form-item label="封面图" prop="cover_image">
          <el-input v-model="form.cover_image" maxlength="1024" clearable placeholder="https://... 或 /storage/...">
            <template #append>
              <el-button :disabled="!form.cover_image" @click="previewVisible = true">预览</el-button>
            </template>
          </el-input>
          <cover-image-uploader v-model="form.cover_image" class="cover-upload" />
        </el-form-item>
        <el-form-item label="位置描述" prop="description">
          <el-input
            v-model="form.description"
            type="textarea"
            :rows="4"
            maxlength="500"
            show-word-limit
            placeholder="说明该位置在客户端的用途和展示场景"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="handleSubmit">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="previewVisible" title="封面图预览" width="680px" append-to-body>
      <div class="preview-body">
        <el-image :src="form.cover_image" fit="contain" class="preview-image" />
      </div>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, type FormInstance, type FormRules } from 'element-plus'
import {
  createDisplayPosition,
  deleteDisplayPosition,
  getDisplayPositionList,
  updateDisplayPosition,
  type DisplayPosition,
  type DisplayPositionPayload,
} from '@/api/displayPosition'
import { useUserStore } from '@/store/user'
import CoverImageUploader from '@/components/CoverImageUploader.vue'

const userStore = useUserStore()
const canAdd = computed(() => userStore.hasPermission('template:position:add'))
const canEdit = computed(() => userStore.hasPermission('template:position:edit'))
const canDelete = computed(() => userStore.hasPermission('template:position:delete'))

const loading = ref(false)
const submitting = ref(false)
const dialogVisible = ref(false)
const previewVisible = ref(false)
const formRef = ref<FormInstance>()
const tableData = ref<DisplayPosition[]>([])
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const query = reactive({ keyword: '', status: '' })

const defaultForm = {
  id: 0,
  position_name: '',
  position_key: '',
  description: '',
  cover_image: '',
  sort: 0,
  status: 1,
}
const form = reactive({ ...defaultForm })
const rules: FormRules = {
  position_name: [{ required: true, message: '请输入展示位置', trigger: 'blur' }],
  position_key: [
    { required: true, message: '请输入位置标识', trigger: 'blur' },
    { pattern: /^[A-Za-z0-9_-]+$/, message: '仅支持字母、数字、下划线和中划线', trigger: 'blur' },
  ],
  cover_image: [{ required: true, message: '请输入封面图 URL', trigger: 'blur' }],
}

async function fetchData() {
  loading.value = true
  try {
    const params: Record<string, unknown> = { page: page.value, page_size: pageSize.value }
    if (query.keyword) params.keyword = query.keyword
    if (query.status !== '') params.status = query.status
    const res: any = await getDisplayPositionList(params)
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

function openCreate() {
  Object.assign(form, defaultForm)
  dialogVisible.value = true
}

function openEdit(row: DisplayPosition) {
  Object.assign(form, {
    id: row.id,
    position_name: row.position_name,
    position_key: row.position_key,
    description: row.description || '',
    cover_image: row.cover_image,
    sort: row.sort,
    status: row.status,
  })
  dialogVisible.value = true
}

async function handleSubmit() {
  await formRef.value?.validate()
  submitting.value = true
  try {
    const payload: DisplayPositionPayload = {
      position_name: form.position_name.trim(),
      position_key: form.position_key.trim(),
      description: form.description.trim(),
      cover_image: form.cover_image.trim(),
      sort: form.sort,
      status: form.status,
    }
    if (form.id) await updateDisplayPosition(form.id, payload)
    else await createDisplayPosition(payload)
    ElMessage.success('展示位置已保存')
    dialogVisible.value = false
    await fetchData()
  } finally {
    submitting.value = false
  }
}

async function handleDelete(id: number) {
  await deleteDisplayPosition(id)
  ElMessage.success('展示位置已删除')
  if (tableData.value.length === 1 && page.value > 1) page.value--
  await fetchData()
}

onMounted(fetchData)
</script>

<style scoped>
.page-wrap { min-width: 0; }
.page-header { display: flex; align-items: center; justify-content: space-between; gap: 16px; }
.page-title { color: #303133; font-size: 17px; font-weight: 600; }
.page-subtitle { margin-top: 4px; color: #909399; font-size: 12px; }
.filters { display: grid; grid-template-columns: minmax(240px, 1fr) 180px auto auto; gap: 10px; max-width: 760px; margin-bottom: 16px; }
.cover-image { width: 96px; height: 60px; border-radius: 6px; background: #f2f3f5; }
.image-error { width: 100%; height: 100%; display: flex; align-items: center; justify-content: center; color: #c0c4cc; font-size: 24px; }
.primary-text { color: #303133; font-weight: 500; }
.position-key { margin-top: 6px; font-family: ui-monospace, SFMono-Regular, Menlo, monospace; }
.secondary-text { color: #909399; font-size: 12px; }
.description-text { display: -webkit-box; overflow: hidden; color: #606266; line-height: 1.5; -webkit-box-orient: vertical; -webkit-line-clamp: 2; }
.pagination-wrap { display: flex; justify-content: flex-end; margin-top: 16px; overflow-x: auto; }
.form-grid { display: grid; grid-template-columns: 1fr 1fr; column-gap: 12px; }
.form-grid :deep(.el-input-number) { width: 100%; }
.cover-upload { margin-top: 10px; }
.preview-body { display: flex; align-items: center; justify-content: center; min-height: 260px; background: #0f1115; border-radius: 8px; overflow: hidden; }
.preview-image { max-width: 100%; max-height: 70vh; }
@media (max-width: 700px) {
  .page-header { align-items: stretch; flex-direction: column; }
  .filters, .form-grid { grid-template-columns: 1fr; max-width: none; }
  .page-wrap :deep(.el-card__header), .page-wrap :deep(.el-card__body) { padding: 14px; }
}
</style>
