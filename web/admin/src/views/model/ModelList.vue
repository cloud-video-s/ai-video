<template>
  <div class="page-wrap">
    <el-card shadow="never">
      <template #header>
        <div class="page-header">
          <div>
            <div class="page-title">模型管理</div>
            <div class="page-subtitle">维护平台关联、API 路由、版本、密钥及模型参数配置</div>
          </div>
          <el-button v-if="canAdd" type="primary" :disabled="platformOptions.length === 0" @click="openCreate">
            <el-icon><Plus /></el-icon>新增模型
          </el-button>
        </div>
      </template>

      <el-alert v-if="platformOptions.length === 0" type="warning" :closable="false" show-icon class="platform-warning">
        <template #title>请先新增并启用一个平台，再创建模型。</template>
      </el-alert>

      <div class="filters">
        <el-input v-model="query.keyword" clearable placeholder="模型名称、编码或版本" @keyup.enter="handleSearch">
          <template #prefix><el-icon><Search /></el-icon></template>
        </el-input>
        <el-select v-model="query.platform_id" clearable filterable placeholder="全部平台">
          <el-option v-for="item in platformOptions" :key="item.id" :label="item.name" :value="String(item.id)" />
        </el-select>
        <el-select v-model="query.model_type" clearable placeholder="全部类型">
          <el-option v-for="item in modelTypeOptions" :key="item.value" :label="item.key" :value="item.value" />
        </el-select>
        <el-select v-model="query.status" clearable placeholder="全部状态">
          <el-option label="启用" value="1" />
          <el-option label="禁用" value="0" />
        </el-select>
        <el-button type="primary" plain @click="handleSearch">查询</el-button>
        <el-button @click="handleReset">重置</el-button>
      </div>

      <el-table v-loading="loading" :data="tableData" row-key="id" stripe @sort-change="handleSortChange">
        <el-table-column prop="id" label="ID" width="70" sortable="custom" />
        <el-table-column label="模型" min-width="190">
          <template #default="{ row }">
            <div class="model-cell">
              <el-image
                v-if="row.icon"
                :src="toMediaURL(row.icon)"
                :preview-src-list="[toMediaURL(row.icon)]"
                preview-teleported
                fit="contain"
                class="model-icon"
              />
              <div>
                <div class="primary-text">{{ row.name }}</div>
                <div class="model-meta"><code>{{ row.code }}</code><el-tag size="small" type="info">{{ row.version }}</el-tag></div>
              </div>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="平台 / 生成类型 / 模型类型" min-width="180">
          <template #default="{ row }">
            <div>{{ row.platform?.name || `平台 #${row.platform_id}` }}</div>
            <el-tag class="type-tag" size="small" effect="plain">
              {{ getmodelType(row.model_type) }}
            </el-tag>
            <el-tag class="type-tag" size="small" effect="plain">
              {{ getmodelFeatures(row.model_features) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="API 配置" min-width="310">
          <template #default="{ row }">
            <div class="endpoint host-line">{{ row.host_url }}</div>
            <div class="endpoint"><span>生成</span><code>{{ row.submit_endpoint }}</code></div>
            <div class="endpoint"><span>查询</span><code>{{ row.status_endpoint }}</code></div>
          </template>
        </el-table-column>
        <el-table-column label="认证类型" width="125" align="center">
          <template #default="{ row }">
            <el-tag :type="row.api_key_configured ? 'success' : 'warning'" effect="plain">
              {{ authTypeLabel(row.auth_type) }}
            </el-tag>
            <div v-if="!row.api_key_configured" class="secondary-text">密钥未配置</div>
          </template>
        </el-table-column>
        <el-table-column label="积分" width="80" align="right">
          <template #default="{ row }">{{ row.score }}</template>
        </el-table-column>
        <el-table-column label="状态" width="85" align="center">
          <template #default="{ row }">
            <el-tag :type="row.status === 1 ? 'success' : 'info'">{{ row.status === 1 ? '启用' : '禁用' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="更新时间" width="165">
          <template #default="{ row }">{{ formatDate(row.updated_at) }}</template>
        </el-table-column>
        <el-table-column v-if="canConfig || canEdit || canDelete" label="操作" width="205" fixed="right" align="center">
          <template #default="{ row }">
            <el-button v-if="canConfig" link type="success" @click="openParameters(row)">模型配置</el-button>
            <el-button v-if="canEdit" link type="primary" @click="openEdit(row)">编辑</el-button>
            <el-popconfirm
              v-if="canDelete"
              :title="`确认软删除模型 ${row.name} 及其配置？`"
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

    <el-dialog v-model="dialogVisible" :title="form.id ? '编辑模型' : '新增模型'" width="880px" destroy-on-close>
      <el-form ref="formRef" :model="form" :rules="rules" label-width="110px">
        <div class="form-grid">
          <el-form-item label="所属平台" prop="platform_id">
            <el-select v-model="form.platform_id" filterable style="width: 100%" placeholder="请选择平台" @change="handlePlatformChange">
              <el-option v-for="item in platformOptions" :key="item.id" :label="`${item.name} · ${item.code}`" :value="item.id" />
            </el-select>
          </el-form-item>
          <el-form-item label="模型名称" prop="name">
            <el-input v-model="form.name" maxlength="64" placeholder="例如：Kling v3" />
          </el-form-item>
          <el-form-item label="模型编码" prop="code">
            <el-input v-model="form.code" maxlength="32" placeholder="例如：kling-v3" />
          </el-form-item>
          <el-form-item label="模型版本" prop="version">
            <el-input v-model="form.version" maxlength="16" placeholder="例如：v3.0" />
          </el-form-item>
          <el-form-item label="模型类型" prop="model_features">
            <el-select v-model="form.model_features" style="width: 100%" placeholder="请选择模型类型">
              <el-option v-for="item in modelFeaturesOptions" :key="item.value" :label="item.key" :value="item.value" />
            </el-select>
          </el-form-item>
          <el-form-item label="生成类型" prop="model_type">
            <el-select v-model="form.model_type" filterable allow-create default-first-option style="width: 100%">
              <el-option v-for="item in modelTypeOptions" :key="item.value" :label="item.key" :value="item.value" />
            </el-select>
          </el-form-item>
          <el-form-item label="模型积分" prop="score">
            <el-input-number v-model="form.score" :min="0" :max="999999999" controls-position="right" style="width: 100%" />
          </el-form-item>
          <el-form-item label="模型图标" prop="icon" class="form-grid-wide">
            <LogoImageUploader
              v-model="form.icon"
              image-name="模型图标"
              placeholder="上传或输入图标 URL"
              :crop-aspect-ratio="1"
            />
          </el-form-item>
        </div>

        <el-divider content-position="left">接口与认证</el-divider>
        <el-form-item label="API 域名" prop="host_url">
          <el-input v-model="form.host_url" maxlength="255" placeholder="https://api.example.com" />
        </el-form-item>
        <div class="form-grid">
          <el-form-item label="生成地址路由" prop="submit_endpoint">
            <el-input v-model="form.submit_endpoint" maxlength="255" placeholder="/v1/tasks/submit" />
          </el-form-item>
          <el-form-item label="查询地址路由" prop="status_endpoint">
            <el-input v-model="form.status_endpoint" maxlength="255" placeholder="/v1/tasks/status" />
          </el-form-item>
          <el-form-item label="认证类型" prop="auth_type">
            <el-select v-model="form.auth_type" style="width: 100%">
              <el-option v-for="item in authTypeOptions" :key="item.value" :label="item.key" :value="item.value" />
            </el-select>
          </el-form-item>
        </div>
        <el-form-item :label="form.id ? '更换密钥' : 'API 密钥'" :required="!form.id">
          <el-input
            v-model="form.api_key"
            type="password"
            maxlength="2048"
            show-password
            autocomplete="new-password"
            :placeholder="form.id ? '留空表示保留原密钥' : '请输入模型 API 密钥'"
          />
        </el-form-item>

        <el-divider content-position="left">其他信息</el-divider>
        <el-form-item label="状态">
          <el-radio-group v-model="form.status">
            <el-radio :value="1">启用</el-radio>
            <el-radio :value="0">禁用</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="模型说明">
          <el-input v-model="form.description" type="textarea" :rows="3" maxlength="255" show-word-limit placeholder="可选" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="handleSubmit">保存</el-button>
      </template>
    </el-dialog>

    <ModelParameterDrawer v-model="parameterDrawerVisible" :model="selectedModel" />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, type FormInstance, type FormRules } from 'element-plus'
import {
  createModel,
  deleteModel,
  getModelList,
  getPlatformOptions,
  updateModel,
  type VideoModel,
  type VideoModelPayload,
  type VideoPlatform,
} from '@/api/videoModel'
import { useUserStore } from '@/store/user'
import { toMediaURL } from '@/utils/mediaUrl'
import LogoImageUploader from '@/components/LogoImageUploader.vue'
import { useRemoteTableSort } from '@/utils/tableSort'
import ModelParameterDrawer from './ModelParameterDrawer.vue'

const userStore = useUserStore()
const canAdd = computed(() => userStore.hasPermission('model:add'))
const canEdit = computed(() => userStore.hasPermission('model:edit'))
const canDelete = computed(() => userStore.hasPermission('model:delete'))
const canConfig = computed(() => userStore.hasPermission('model:config'))

const modelTypeOptions = [
  {
    key: "生成图片",
    value: 1,
  },
  {
    key: "生成视频",
    value: 2,
  }
  ]
const authTypeOptions = [
  {
    key: "Bearer Token",
    value: 1,
  },
  {
    key: "API-Key",
    value: 2,
  }
]
const modelFeaturesOptions = [
  {
    key: "通用",
    value: 1,
  },
  {
    key: "模板",
    value: 2,
  },
  {
    key: "生成模型",
    value: 3,
  },
  {
    key: "工具",
    value: 4,
  }
]
const loading = ref(false)
const submitting = ref(false)
const dialogVisible = ref(false)
const parameterDrawerVisible = ref(false)
const selectedModel = ref<VideoModel | null>(null)
const formRef = ref<FormInstance>()
const tableData = ref<VideoModel[]>([])
const platformOptions = ref<VideoPlatform[]>([])
const page = ref(1)
const { sortParams, handleSortChange } = useRemoteTableSort(page, fetchData)
const pageSize = ref(20)
const total = ref(0)
const query = reactive({ keyword: '', platform_id: '', model_type: '', status: '' })

type ModelForm = VideoModelPayload & { id: number }
const defaultForm: ModelForm = {
  id: 0,
  platform_id: 0,
  name: '',
  code: '',
  model_type: 1,
  model_features: 1,
  version: '',
  host_url: '',
  submit_endpoint: '',
  status_endpoint: '',
  request_method: 'POST',
  auth_type: 1,
  api_key: '',
  score: 0,
  icon: '',
  description: '',
  status: 1,
}
const form = reactive<ModelForm>({ ...defaultForm })
const rules: FormRules = {
  platform_id: [{ required: true, type: 'number', min: 1, message: '请选择所属平台', trigger: 'change' }],
  name: [{ required: true, message: '请输入模型名称', trigger: 'blur' }],
  code: [
    { required: true, message: '请输入模型编码', trigger: 'blur' },
    { pattern: /^[A-Za-z0-9._-]+$/, message: '仅支持字母、数字、点、下划线和中划线', trigger: 'blur' },
  ],
  model_type: [{ required: true, message: '请输入模型类型', trigger: 'change' }],
  model_features: [{ required: true, message: '请选择模型类型', trigger: 'change' }],
  version: [{ required: true, message: '请输入模型版本', trigger: 'blur' }],
  host_url: [
    { required: true, message: '请输入 API 域名', trigger: 'blur' },
    { pattern: /^https?:\/\/[^\s]+$/i, message: '请输入有效的 HTTP(S) 地址', trigger: 'blur' },
  ],
  submit_endpoint: [
    { required: true, message: '请输入生成地址路由', trigger: 'blur' },
    { pattern: /^\//, message: '路由必须以 / 开头', trigger: 'blur' },
  ],
  status_endpoint: [
    { required: false, message: '请输入查询地址路由', trigger: 'blur' },
    // { pattern: /^\//, message: '路由必须以 / 开头', trigger: 'blur' },
  ],
  auth_type: [{ required: true, type: 'number', message: '请选择认证类型', trigger: 'change' }],
}

async function fetchPlatforms() {
  try {
    const res: any = await getPlatformOptions()
    platformOptions.value = res.data || []
  } catch {
    platformOptions.value = []
  }
}

async function fetchData() {
  loading.value = true
  try {
    const params: Record<string, unknown> = { page: page.value, page_size: pageSize.value, ...sortParams() }
    if (query.keyword.trim()) params.keyword = query.keyword.trim()
    if (query.platform_id) params.platform_id = query.platform_id
    if (query.model_type) params.model_type = query.model_type
    if (query.status !== '') params.status = query.status
    const res: any = await getModelList(params)
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
  Object.assign(query, { keyword: '', platform_id: '', model_type: '', status: '' })
  page.value = 1
  fetchData()
}

function handlePageSizeChange() {
  page.value = 1
  fetchData()
}

function openCreate() {
  Object.assign(form, defaultForm)
  if (platformOptions.value.length === 1) {
    form.platform_id = platformOptions.value[0].id
    form.host_url = platformOptions.value[0].base_url
  }
  dialogVisible.value = true
}

function openEdit(row: VideoModel) {
  Object.assign(form, {
    id: row.id,
    platform_id: row.platform_id,
    name: row.name,
    code: row.code,
    model_type: row.model_type,
    model_features: row.model_features,
    version: row.version,
    host_url: row.host_url,
    submit_endpoint: row.submit_endpoint,
    status_endpoint: row.status_endpoint,
    request_method: row.request_method,
    auth_type: row.auth_type,
    api_key: '',
    score: row.score,
    icon: row.icon || '',
    description: row.description || '',
    status: row.status,
  })
  dialogVisible.value = true
}

function handlePlatformChange(id: number) {
  const platform = platformOptions.value.find((item) => item.id === id)
  if (platform && (!form.host_url || !form.id)) form.host_url = platform.base_url
}

async function handleSubmit() {
  await formRef.value?.validate()
  if (!form.id && !form.api_key.trim()) {
    ElMessage.error('新建模型必须配置 API 密钥')
    return
  }
  submitting.value = true
  try {
    const payload: VideoModelPayload = {
      platform_id: form.platform_id,
      name: form.name.trim(),
      code: form.code.trim(),
      model_type: form.model_type,
      model_features: form.model_features,
      version: form.version.trim(),
      host_url: form.host_url.trim().replace(/\/+$/, ''),
      submit_endpoint: form.submit_endpoint.trim(),
      status_endpoint: form.status_endpoint.trim(),
      request_method: form.request_method,
      auth_type: form.auth_type,
      api_key: form.api_key.trim(),
      score: Number(form.score),
      icon: form.icon.trim(),
      description: form.description.trim(),
      status: form.status,
    }
    if (form.id) await updateModel(form.id, payload)
    else await createModel(payload)
    ElMessage.success('模型信息已保存')
    dialogVisible.value = false
    await fetchData()
  } finally {
    submitting.value = false
  }
}

function openParameters(row: VideoModel) {
  selectedModel.value = row
  parameterDrawerVisible.value = true
}

async function handleDelete(id: number) {
  await deleteModel(id)
  ElMessage.success('模型及其配置已软删除')
  if (tableData.value.length === 1 && page.value > 1) page.value--
  await fetchData()
}

function formatDate(value: string) {
  if (!value) return '-'
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString('zh-CN', { hour12: false })
}

function getmodelType(modelType: number){
  for (let i = 0; i <= modelTypeOptions.length; i++){
    if (modelTypeOptions[i].value === modelType){
      return modelTypeOptions[i].key
    }
  }
  return ''
}

function getmodelFeatures(modelFeatures: number) {
  return modelFeaturesOptions.find((item) => item.value === modelFeatures)?.key || `未知（${modelFeatures}）`
}

function authTypeLabel(authType: number) {
  return authTypeOptions.find((item) => item.value === authType)?.key || `未知（${authType}）`
}

onMounted(async () => {
  await fetchPlatforms()
  await fetchData()
})
</script>

<style scoped>
.page-wrap { min-width: 0; }
.page-header { display: flex; align-items: center; justify-content: space-between; gap: 16px; }
.page-title { color: #303133; font-size: 17px; font-weight: 600; }
.page-subtitle { margin-top: 4px; color: #909399; font-size: 12px; }
.platform-warning { margin-bottom: 16px; }
.filters { display: grid; grid-template-columns: minmax(220px, 1.2fr) repeat(3, minmax(130px, 0.8fr)) auto auto; gap: 10px; margin-bottom: 16px; }
.primary-text { color: #303133; font-weight: 600; }
.model-cell { display: flex; align-items: center; gap: 10px; min-width: 0; }
.model-icon { width: 42px; height: 42px; flex: 0 0 42px; border: 1px solid #ebeef5; border-radius: 8px; background: #f5f7fa; cursor: zoom-in; }
.model-meta { display: flex; align-items: center; flex-wrap: wrap; gap: 6px; margin-top: 5px; }
.model-meta code { color: #606266; }
.type-tag { margin-top: 6px; }
.endpoint { display: flex; align-items: center; gap: 7px; margin-top: 4px; overflow: hidden; }
.endpoint span { flex: 0 0 auto; color: #909399; font-size: 12px; }
.endpoint code { overflow: hidden; color: #606266; text-overflow: ellipsis; white-space: nowrap; }
.host-line { margin-top: 0; color: #303133; }
.pagination-wrap { display: flex; justify-content: flex-end; margin-top: 16px; overflow-x: auto; }
.form-grid { display: grid; grid-template-columns: 1fr 1fr; column-gap: 16px; }
.form-grid-wide { grid-column: 1 / -1; }
@media (max-width: 1080px) {
  .filters { grid-template-columns: repeat(2, minmax(170px, 1fr)); }
}
@media (max-width: 720px) {
  .page-header { align-items: stretch; flex-direction: column; }
  .filters, .form-grid { grid-template-columns: 1fr; }
}
</style>
