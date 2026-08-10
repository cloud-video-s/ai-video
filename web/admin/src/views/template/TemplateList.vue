<template>
  <div class="page-wrap">
    <el-card shadow="never">
      <template #header>
        <div class="page-header">
          <div>
            <div class="page-title">模板管理</div>
            <div class="page-subtitle">管理模板自身信息；投放范围由所属模板分类统一控制</div>
          </div>
          <el-button v-if="canAdd" type="primary" :disabled="enabledTypeOptions.length === 0 || enabledModelOptions.length === 0" @click="openCreate">
            <el-icon><Plus /></el-icon>新增模板
          </el-button>
        </div>
      </template>

      <el-alert
        v-if="enabledTypeOptions.length === 0"
        title="请先新增并启用一个模板分类，再创建模板。"
        type="warning"
        show-icon
        :closable="false"
        class="type-alert"
      />
      <el-alert
        v-else-if="enabledModelOptions.length === 0"
        title="请先在模型管理中新增并启用图片或视频模型，再创建模板。"
        type="warning"
        show-icon
        :closable="false"
        class="type-alert"
      />

      <div class="filters">
        <el-select v-model="query.template_type_id" clearable filterable placeholder="模板分类">
          <el-option v-for="item in typeOptions" :key="item.id" :label="typeLabel(item)" :value="String(item.id)" />
        </el-select>
        <el-select v-model="query.template_type" clearable filterable placeholder="模板类型" @change="handleQueryTemplateTypeChange">
          <el-option v-for="item in templateKinds" :key="item.value" :label="item.label" :value="item.value" />
        </el-select>
        <el-select v-model="query.model_id" clearable filterable placeholder="关联模型">
          <el-option v-for="item in queryModelOptions" :key="item.id" :label="modelLabel(item)" :value="String(item.id)" />
        </el-select>
        <el-select v-model="query.position_key" clearable filterable placeholder="展示位置">
          <el-option v-for="item in positionOptions" :key="item.id" :label="positionLabel(item)" :value="item.position_key" />
        </el-select>
        <el-select v-model="query.status" clearable placeholder="启用状态">
          <el-option label="启用" value="1" />
          <el-option label="禁用" value="0" />
        </el-select>
        <el-input v-model="query.keyword" clearable placeholder="模板名称、提示词或描述" @keyup.enter="handleSearch">
          <template #prefix><el-icon><Search /></el-icon></template>
        </el-input>
        <el-button type="primary" plain @click="handleSearch">查询</el-button>
        <el-button @click="handleReset">重置</el-button>
      </div>

      <el-table v-loading="loading" :data="tableData" row-key="id" stripe>
        <el-table-column prop="id" label="ID" width="68" />
        <el-table-column label="封面" width="92" align="center">
          <template #default="{ row }">
            <el-image
              class="cover-image"
              :src="toMediaURL(row.cover_image_url)"
              :preview-src-list="[toMediaURL(row.cover_image_url)]"
              preview-teleported
              fit="cover"
            >
              <template #error><div class="image-error"><el-icon><Picture /></el-icon></div></template>
            </el-image>
          </template>
        </el-table-column>
        <el-table-column label="模板" min-width="180">
          <template #default="{ row }">
            <div class="primary-text">{{ row.name }}</div>
            <div class="tag-line">
              <el-tag size="small" effect="plain">{{ kindLabel(row.template_type) }}</el-tag>
              <span class="secondary-text">{{ modelName(row.model_id) }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="所属分类" min-width="210">
          <template #default="{ row }">
            <div class="primary-text">{{ row.video_template_type?.category_name || `分类 #${row.template_type_id}` }}</div>
          </template>
        </el-table-column>
        <el-table-column label="媒体" width="150" align="center">
          <template #default="{ row }">
            <div class="media-actions">
              <el-button link type="primary" @click="previewMedia(mediaKind(row.template_type), row.original_url, `${row.name} · 原始资源`)">原始资源</el-button>
              <el-button
                v-if="row.thumbnail_url"
                link
                type="primary"
                @click="previewMedia(mediaKindFromURL(row.thumbnail_url), row.thumbnail_url, `${row.name} · 缩略资源`)"
              >缩略资源</el-button>
              <span v-else class="secondary-text">无缩略资源</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="提示词" min-width="220">
          <template #default="{ row }">
            <el-tooltip v-if="row.prompt" :content="row.prompt" placement="top" :show-after="400">
              <div class="prompt-text">{{ row.prompt }}</div>
            </el-tooltip>
            <span v-else class="secondary-text">暂无提示词</span>
          </template>
        </el-table-column>
        <el-table-column prop="sort" label="排序" width="68" align="center" />
        <el-table-column label="状态" width="82" align="center">
          <template #default="{ row }">
            <el-tag :type="row.status === 1 ? 'success' : 'danger'">
              {{ row.status === 1 ? '启用' : '禁用' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column v-if="canEdit || canDelete" label="操作" width="205" fixed="right" align="center">
          <template #default="{ row }">
            <el-button v-if="canEdit" link type="success" @click="openTemplateParameters(row)">模型配置</el-button>
            <el-button v-if="canEdit" link type="primary" @click="openEdit(row)">编辑</el-button>
            <el-popconfirm v-if="canDelete" title="确认删除该模板？" @confirm="handleDelete(row.id)">
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

    <el-dialog v-model="dialogVisible" :title="form.id ? '编辑模板' : '新增模板'" width="820px" destroy-on-close>
      <el-form ref="formRef" :model="form" :rules="rules" label-width="108px">
        <div class="form-grid">
          <el-form-item label="模板名称" prop="name">
            <el-input v-model="form.name" maxlength="128" />
          </el-form-item>
          <el-form-item label="所属分类" prop="template_type_id">
            <el-select v-model="form.template_type_id" filterable placeholder="请选择模板分类" style="width: 100%">
              <el-option
                v-for="item in typeOptions"
                :key="item.id"
                :label="typeLabel(item)"
                :value="item.id"
                :disabled="item.status !== 1 && item.id !== form.template_type_id"
              />
            </el-select>
          </el-form-item>
          <el-form-item label="模板类型" prop="template_type">
            <el-select v-model="form.template_type" placeholder="请选择模板类型" style="width: 100%" @change="handleTemplateTypeChange">
              <el-option v-for="item in templateKinds" :key="item.value" :label="item.label" :value="item.value" />
            </el-select>
          </el-form-item>
          <el-form-item label="关联模型" prop="model_id">
            <el-select v-model="form.model_id" filterable placeholder="请选择模型" style="width: 100%" @change="handleModelChange">
              <el-option
                v-for="item in matchingModelOptions"
                :key="item.id"
                :label="modelLabel(item)"
                :value="item.id"
                :disabled="item.status !== 1 && item.id !== form.model_id"
              />
            </el-select>
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

        <el-form-item label="封面图" prop="cover_image_url">
          <div class="cover-field">
            <MediaUploader
              v-model="form.cover_image_url"
              kind="image"
              resume-key="template-cover"
              placeholder="输入图片 URL，或选择图片分片上传"
              @preview="(url) => previewMedia('image', url, '封面图预览')"
              @uploading-change="(value) => (mediaUploading.cover = value)"
            />
            <div v-if="form.cover_image_url" class="cover-form-preview">
              <el-image
                :src="toMediaURL(form.cover_image_url)"
                :preview-src-list="[toMediaURL(form.cover_image_url)]"
                preview-teleported
                fit="cover"
                class="cover-preview-image"
              >
                <template #error>
                  <div class="cover-preview-error">
                    <el-icon><Picture /></el-icon>
                    <span>封面加载失败</span>
                  </div>
                </template>
              </el-image>
              <div class="cover-preview-meta">
                <span>封面预览</span>
                <span>点击图片查看大图</span>
              </div>
            </div>
          </div>
        </el-form-item>
        <el-form-item :label="form.template_type === 1 ? '原始图片' : '原始视频'" prop="original_url">
          <MediaUploader
            v-model="form.original_url"
            :kind="mediaKind(form.template_type)"
            resume-key="template-video"
            placeholder="输入原始资源 URL，或选择文件分片上传"
            @preview="(url) => previewMedia(mediaKind(form.template_type), url, '原始资源预览')"
            @uploading-change="(value) => (mediaUploading.template = value)"
          />
        </el-form-item>
        <el-form-item label="缩略资源" prop="thumbnail_url">
          <MediaUploader
            v-model="form.thumbnail_url"
            kind="media"
            resume-key="template-thumbnail"
            placeholder="输入缩略资源 URL，或选择图片/视频分片上传"
            @preview="(url) => previewMedia(mediaKindFromURL(url), url, '缩略资源预览')"
            @uploading-change="(value) => (mediaUploading.thumbnail = value)"
          />
        </el-form-item>
        <el-form-item label="提示词" prop="prompt">
          <el-input
            v-model="form.prompt"
            type="textarea"
            :rows="4"
            maxlength="65535"
            show-word-limit
            placeholder="输入生成资源所需的提示词"
          />
        </el-form-item>
        <el-form-item label="描述" prop="description">
          <el-input v-model="form.description" type="textarea" :rows="2" maxlength="500" show-word-limit />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitting" :disabled="isMediaUploading" @click="handleSubmit">
          {{ isMediaUploading ? '媒体上传中' : '保存' }}
        </el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="preview.visible" :title="preview.title" width="760px" destroy-on-close append-to-body>
      <div class="preview-body">
        <el-image v-if="preview.kind === 'image'" :src="preview.url" fit="contain" class="preview-image" />
        <video v-else-if="preview.url" :src="preview.url" controls playsinline class="preview-video" />
      </div>
    </el-dialog>

    <TemplateModelParameterDrawer
      v-model="parameterDrawerVisible"
      :parameters="modelParameterPayloads"
      :model="parameterDrawerModel"
      :definitions="modelParameterDefinitions"
      :loading="parameterLoading || parameterSaving"
      @update:parameters="handleModelParameterUpdate"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, type FormInstance } from 'element-plus'
import {
  createTemplate,
  deleteTemplate,
  getTemplateList,
  getTemplateModelParameters,
  getTemplateTypeOptions,
  replaceTemplateModelParameters,
  updateTemplate,
  type TemplateModelParameter,
  type TemplateModelParameterPayload,
  type VideoTemplate,
  type VideoTemplatePayload,
  type VideoTemplateType,
} from '@/api/template'
import {
  getModelList,
  getModelParameters,
  type ModelParameter,
  type VideoModel,
} from '@/api/videoModel'
import { useUserStore } from '@/store/user'
import { getDisplayPositionOptions, type DisplayPosition } from '@/api/displayPosition'
import type { Country } from '@/api/country'
import type { AppPackage } from '@/api/package'
import type { PackageVersion } from '@/api/packageVersion'
import type { VideoApp } from '@/api/videoApp'
import MediaUploader from '@/components/MediaUploader.vue'
import { toMediaURL } from '@/utils/mediaUrl'
import TemplateModelParameterDrawer from './TemplateModelParameterDrawer.vue'

const userStore = useUserStore()
const canAdd = computed(() => userStore.hasPermission('template:add'))
const canEdit = computed(() => userStore.hasPermission('template:edit'))
const canDelete = computed(() => userStore.hasPermission('template:delete'))
const enabledTypeOptions = computed(() => typeOptions.value.filter((item) => item.status === 1))
const enabledModelOptions = computed(() => modelOptions.value.filter((item) => item.status === 1))

const templateKinds = [
  { value: 1 as const, label: '图片模板' },
  { value: 2 as const, label: '视频模板' },
]
const loading = ref(false)
const submitting = ref(false)
const parameterLoading = ref(false)
const parameterSaving = ref(false)
const dialogVisible = ref(false)
const parameterDrawerVisible = ref(false)
const configuredTemplate = ref<VideoTemplate | null>(null)
const formRef = ref<FormInstance>()
const tableData = ref<VideoTemplate[]>([])
const typeOptions = ref<VideoTemplateType[]>([])
const modelOptions = ref<VideoModel[]>([])
const positionOptions = ref<DisplayPosition[]>([])
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const query = reactive({
  template_type_id: '', model_id: '', template_type: '', position_key: '', status: '', keyword: '',
})

interface TemplateForm {
  id: number
  template_type_id: number
  model_id: number
  name: string
  template_type: 1 | 2
  sort: number
  cover_image_url: string
  original_url: string
  thumbnail_url: string
  prompt: string
  status: number
  description: string
}

const modelParameterDefinitions = ref<ModelParameter[]>([])
const modelParameterPayloads = ref<TemplateModelParameterPayload[]>([])
const defaultForm: TemplateForm = {
  id: 0,
  template_type_id: 0,
  model_id: 0,
  name: '',
  template_type: 2,
  sort: 0,
  cover_image_url: '',
  original_url: '',
  thumbnail_url: '',
  prompt: '',
  status: 1,
  description: '',
}
const form = reactive<TemplateForm>({ ...defaultForm })
const matchingModelOptions = computed(() => modelOptions.value.filter((item) => item.model_type === form.template_type))
const currentModel = computed(() => modelOptions.value.find((item) => item.id === form.model_id) || null)
const parameterDrawerModel = computed(() => {
  const modelID = configuredTemplate.value?.model_id
  return modelOptions.value.find((item) => item.id === modelID) || null
})
const queryModelOptions = computed(() => {
  const kind = Number(query.template_type)
  return kind === 1 || kind === 2 ? modelOptions.value.filter((item) => item.model_type === kind) : modelOptions.value
})
const imageExtensions = ['.jpg', '.jpeg', '.png', '.webp', '.gif']
const videoExtensions = ['.mp4', '.mov', '.webm', '.mkv']
type ValidationCallback = (error?: Error) => void

function mediaURLMatches(value: string, kind: 'image' | 'video') {
  try {
    const parsed = new URL(value.trim(), window.location.origin)
    const path = parsed.pathname.toLowerCase()
    const extensions = kind === 'image' ? imageExtensions : videoExtensions
    return extensions.some((extension) => path.endsWith(extension))
  } catch {
    return false
  }
}

function validateCoverImageURL(_rule: unknown, value: string, callback: ValidationCallback) {
  if (!value?.trim()) {
    callback(new Error('请输入封面图 URL'))
  } else if (!mediaURLMatches(value, 'image')) {
    callback(new Error('封面图仅支持 JPG、PNG、WebP 或 GIF 图片'))
  } else {
    callback()
  }
}

function validateOriginalURL(_rule: unknown, value: string, callback: ValidationCallback) {
  if (!value?.trim()) {
    callback(new Error('请输入原始资源 URL'))
    return
  }
  const kind = mediaKind(form.template_type)
  if (!mediaURLMatches(value, kind)) {
    callback(new Error(kind === 'image'
      ? '图片模板的原始资源仅支持 JPG、PNG、WebP 或 GIF 图片'
      : '视频模板的原始资源仅支持 MP4、MOV、WebM 或 MKV 视频'))
    return
  }
  callback()
}

function validateThumbnailURL(_rule: unknown, value: string, callback: ValidationCallback) {
  if (!value?.trim()) {
    callback(new Error('请输入缩略资源 URL'))
    return
  }
  if (!mediaURLMatches(value, 'image') && !mediaURLMatches(value, 'video')) {
    callback(new Error('缩略资源仅支持 JPG、PNG、WebP、GIF 图片或 MP4、MOV、WebM、MKV 视频'))
    return
  }
  callback()
}

function validatePrompt(_rule: unknown, value: string, callback: ValidationCallback) {
  if (!value?.trim()) callback(new Error('请输入提示词'))
  else callback()
}

const rules = {
  name: [{ required: true, message: '请输入模板名称', trigger: 'blur' }],
  template_type_id: [{ required: true, message: '请选择模板分类', trigger: 'change' }],
  model_id: [{ required: true, message: '请选择关联模型', trigger: 'change' }],
  template_type: [{ required: true, message: '请选择模板类型', trigger: 'change' }],
  cover_image_url: [{ required: true, validator: validateCoverImageURL, trigger: ['blur', 'change'] }],
  original_url: [{ required: true, validator: validateOriginalURL, trigger: ['blur', 'change'] }],
  thumbnail_url: [{ required: true, validator: validateThumbnailURL, trigger: ['blur', 'change'] }],
  prompt: [{ required: true, validator: validatePrompt, trigger: ['blur', 'change'] }],
}
const preview = reactive({ visible: false, kind: 'video' as 'image' | 'video', url: '', title: '' })
const mediaUploading = reactive({ cover: false, template: false, thumbnail: false })
const isMediaUploading = computed(() => mediaUploading.cover || mediaUploading.template || mediaUploading.thumbnail)

function typeLabel(item: VideoTemplateType) {
  return item.category_name
}

function kindLabel(kind: number) {
  return templateKinds.find((item) => item.value === kind)?.label || kind
}

function mediaKind(kind: number): 'image' | 'video' {
  return kind === 1 ? 'image' : 'video'
}

function mediaKindFromURL(url: string): 'image' | 'video' {
  return mediaURLMatches(url, 'image') ? 'image' : 'video'
}

function modelLabel(item: VideoModel) {
  return `${item.name} · ${item.code}${item.version ? ` · ${item.version}` : ''}`
}

function modelName(modelID: number) {
  const item = modelOptions.value.find((model) => model.id === modelID)
  return item ? modelLabel(item) : `模型 #${modelID}`
}

function positionLabel(item: DisplayPosition) {
  return `${item.position_name} · ${item.position_key}`
}

function arrayValue<T>(value: T[] | null | undefined): T[] {
	return Array.isArray(value) ? value : []
}

function normalizeTemplateType(item: any): VideoTemplateType {
	return {
		...item,
		display_positions: arrayValue<DisplayPosition>(item?.display_positions),
		countries: arrayValue<Country>(item?.countries),
		apps: arrayValue<VideoApp>(item?.apps),
		packages: arrayValue<AppPackage>(item?.packages),
		versions: arrayValue<PackageVersion>(item?.versions),
	}
}

function normalizeTemplate(item: any): VideoTemplate {
	return {
		...item,
		template_type_id: Number(item?.template_type_id ?? item?.video_template_type_id) || 0,
		model_id: Number(item?.model_id) || 0,
		template_type: Number(item?.template_type) as 1 | 2,
		cover_image_url: item?.cover_image_url ?? item?.cover_image ?? '',
		original_url: item?.original_url ?? item?.template_video ?? '',
		thumbnail_url: item?.thumbnail_url ?? item?.thumbnail_video ?? '',
		video_template_type: item?.video_template_type
			? normalizeTemplateType(item.video_template_type)
			: undefined,
	}
}

function cloneJSON<T>(value: T): T {
  if (value === undefined || value === null) return value
  return JSON.parse(JSON.stringify(value)) as T
}

async function loadPersistedTemplateParameters(row: VideoTemplate) {
  modelParameterDefinitions.value = []
  modelParameterPayloads.value = []
  parameterLoading.value = true
  try {
    const [definitionRes, configurationRes]: any[] = await Promise.all([
      getModelParameters(row.model_id),
      getTemplateModelParameters(row.id),
    ])
    modelParameterDefinitions.value = arrayValue<ModelParameter>(definitionRes.data)
    modelParameterPayloads.value = arrayValue<TemplateModelParameter>(configurationRes.data?.parameters)
      .map(templateModelParameterPayload)
  } finally {
    parameterLoading.value = false
  }
}

function templateModelParameterPayload(item: TemplateModelParameter): TemplateModelParameterPayload {
  return {
    param_key: item.param_key,
    value_type: item.value_type,
    parameter_type: item.parameter_type,
    is_required: item.is_required,
    default_value: cloneJSON(item.default_value),
    allowed_values: cloneJSON(arrayValue(item.allowed_values)),
    constraints: cloneJSON(item.constraints || {}),
    description: item.description || '',
    sort_order: item.sort_order || 0,
  }
}

async function fetchTypes() {
	const res: any = await getTemplateTypeOptions()
	typeOptions.value = arrayValue<any>(res.data).map(normalizeTemplateType)
}

async function fetchModels() {
  const res: any = await getModelList({ page: 1, page_size: 100, model_features: '1,2' })
  modelOptions.value = arrayValue<VideoModel>(res.data?.list)
}

async function fetchPositions() {
  const res: any = await getDisplayPositionOptions()
  positionOptions.value = res.data || []
}

async function fetchData() {
  loading.value = true
  try {
    const params: Record<string, unknown> = { page: page.value, page_size: pageSize.value }
    for (const [key, value] of Object.entries(query)) {
      if (value !== '') params[key] = value
	}
	const res: any = await getTemplateList(params)
	tableData.value = arrayValue<any>(res.data?.list).map(normalizeTemplate)
	total.value = Number(res.data?.total) || 0
  } finally {
    loading.value = false
  }
}

function handleSearch() {
  page.value = 1
  fetchData()
}

function handleReset() {
  Object.assign(query, {
    template_type_id: '', model_id: '', template_type: '', position_key: '', status: '', keyword: '',
  })
  page.value = 1
  fetchData()
}

function openCreate() {
  Object.assign(mediaUploading, { cover: false, template: false, thumbnail: false })
  const initialModel = modelOptions.value.find((item) => item.status === 1 && item.model_type === defaultForm.template_type)
  Object.assign(form, defaultForm, {
    template_type_id: enabledTypeOptions.value[0]?.id || 0,
    model_id: initialModel?.id || 0,
  })
  dialogVisible.value = true
}

function openEdit(row: VideoTemplate) {
  Object.assign(mediaUploading, { cover: false, template: false, thumbnail: false })
  Object.assign(form, {
    id: row.id,
    template_type_id: row.template_type_id,
    model_id: row.model_id,
    name: row.name,
    template_type: row.template_type,
    sort: row.sort,
    cover_image_url: row.cover_image_url,
    original_url: row.original_url,
    thumbnail_url: row.thumbnail_url || '',
    prompt: row.prompt || '',
    status: row.status,
    description: row.description || '',
  })
  dialogVisible.value = true
}

async function openTemplateParameters(row: VideoTemplate) {
  configuredTemplate.value = row
  parameterDrawerVisible.value = true
  await loadPersistedTemplateParameters(row)
}

async function handleModelParameterUpdate(next: TemplateModelParameterPayload[]) {
  const template = configuredTemplate.value
  if (!template) return
  parameterSaving.value = true
  try {
    const res: any = await replaceTemplateModelParameters(template.id, next)
    const saved = arrayValue<TemplateModelParameter>(res.data?.parameters)
    modelParameterPayloads.value = saved.length > 0 || next.length === 0
      ? saved.map(templateModelParameterPayload)
      : next
    ElMessage.success('模板模型配置已保存')
  } finally {
    parameterSaving.value = false
  }
}

function handleModelChange(modelID: number) {
  const selected = modelOptions.value.find((item) => item.id === modelID)
  if (selected && selected.model_type !== form.template_type) {
    form.model_id = 0
    ElMessage.error('关联模型类型必须与模板种类一致')
    return
  }
}

async function handleTemplateTypeChange(templateType: 1 | 2) {
  const selected = modelOptions.value.find((item) => item.id === form.model_id)
  if (selected?.model_type !== templateType) {
    const next = modelOptions.value.find((item) => item.status === 1 && item.model_type === templateType)
    form.model_id = next?.id || 0
  }
  try {
    await formRef.value?.validateField(['original_url', 'thumbnail_url'])
  } catch {
    // 类型切换时只刷新校验提示，最终仍由提交校验拦截。
  }
}

function handleQueryTemplateTypeChange(value: string | number) {
  const kind = Number(value)
  const selected = modelOptions.value.find((item) => String(item.id) === query.model_id)
  if ((kind === 1 || kind === 2) && selected?.model_type !== kind) query.model_id = ''
}

function previewMedia(kind: 'image' | 'video', url: string, title: string) {
  if (!url) return
  Object.assign(preview, { visible: true, kind, url: toMediaURL(url), title })
}

async function handleSubmit() {
  await formRef.value?.validate()
  if (!currentModel.value || currentModel.value.model_type !== form.template_type) {
    ElMessage.error('关联模型类型必须与模板种类一致')
    return
  }
  submitting.value = true
  try {
    const payload: VideoTemplatePayload = {
      template_type_id: form.template_type_id,
      model_id: form.model_id,
      name: form.name.trim(),
      template_type: form.template_type,
      sort: form.sort,
      cover_image_url: form.cover_image_url.trim(),
      original_url: form.original_url.trim(),
      thumbnail_url: form.thumbnail_url.trim(),
      prompt: form.prompt.trim(),
      status: form.status,
      description: form.description.trim(),
    }
    if (form.id) await updateTemplate(form.id, payload)
    else await createTemplate(payload)
    ElMessage.success('模板已保存')
    dialogVisible.value = false
    await fetchData()
  } finally {
    submitting.value = false
  }
}

async function handleDelete(id: number) {
  await deleteTemplate(id)
  ElMessage.success('模板已删除')
  if (tableData.value.length === 1 && page.value > 1) page.value--
  await fetchData()
}

onMounted(() => Promise.all([fetchTypes(), fetchModels(), fetchPositions(), fetchData()]))
</script>

<style scoped>
.page-wrap { min-width: 0; }
.page-header { display: flex; align-items: center; justify-content: space-between; gap: 16px; }
.page-title { color: #303133; font-size: 17px; font-weight: 600; }
.page-subtitle { margin-top: 4px; color: #909399; font-size: 12px; }
.type-alert { margin-bottom: 16px; }
.filters { display: grid; grid-template-columns: repeat(auto-fit, minmax(150px, 1fr)); gap: 10px; margin-bottom: 16px; }
.cover-image { width: 62px; height: 82px; border-radius: 6px; background: #f2f3f5; }
.image-error { width: 100%; height: 100%; display: flex; align-items: center; justify-content: center; color: #c0c4cc; font-size: 24px; }
.primary-text { color: #303133; font-weight: 500; }
.secondary-text { color: #909399; font-size: 12px; }
.tag-line { display: flex; align-items: center; gap: 8px; margin-top: 6px; }
.media-actions { display: flex; flex-direction: column; align-items: center; }
.media-actions :deep(.el-button + .el-button) { margin-left: 0; }
.prompt-text { display: -webkit-box; overflow: hidden; color: #606266; line-height: 1.5; -webkit-box-orient: vertical; -webkit-line-clamp: 2; }
.pagination-wrap { display: flex; justify-content: flex-end; margin-top: 16px; overflow-x: auto; }
.form-grid { display: grid; grid-template-columns: 1fr 1fr; column-gap: 14px; }
.form-grid :deep(.el-input-number) { width: 100%; }
.cover-field { width: 100%; }
.cover-form-preview { display: flex; align-items: center; gap: 12px; margin-top: 10px; padding: 10px; border: 1px solid #ebeef5; border-radius: 8px; background: #fafafa; }
.cover-preview-image { width: 160px; height: 90px; flex: 0 0 auto; border-radius: 6px; background: #f0f2f5; cursor: zoom-in; }
.cover-preview-error { width: 100%; height: 100%; display: flex; align-items: center; justify-content: center; flex-direction: column; gap: 5px; color: #a8abb2; font-size: 12px; }
.cover-preview-error .el-icon { font-size: 24px; }
.cover-preview-meta { display: flex; flex-direction: column; gap: 4px; color: #606266; font-size: 13px; }
.cover-preview-meta span:last-child { color: #a8abb2; font-size: 12px; }
.preview-body { display: flex; align-items: center; justify-content: center; min-height: 240px; background: #0f1115; border-radius: 8px; overflow: hidden; }
.preview-image, .preview-video { display: block; max-width: 100%; max-height: 70vh; }
@media (max-width: 1100px) {
  .filters { grid-template-columns: repeat(3, minmax(140px, 1fr)); }
}
@media (max-width: 700px) {
  .page-header { align-items: stretch; flex-direction: column; }
  .filters, .form-grid { grid-template-columns: 1fr; }
  .cover-form-preview { align-items: flex-start; flex-direction: column; }
  .cover-preview-image { width: 100%; height: auto; aspect-ratio: 16 / 9; }
  .page-wrap :deep(.el-card__header), .page-wrap :deep(.el-card__body) { padding: 14px; }
}
</style>
