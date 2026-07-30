<template>
  <div class="page-wrap">
    <el-card shadow="never">
      <template #header>
        <div class="page-header">
          <div>
            <div class="page-title">视频模板</div>
            <div class="page-subtitle">管理模板自身信息；投放范围由所属模板分类统一控制</div>
          </div>
          <el-button v-if="canAdd" type="primary" :disabled="enabledTypeOptions.length === 0 || enabledModelOptions.length === 0" @click="openCreate">
            <el-icon><Plus /></el-icon>新增模板
          </el-button>
        </div>
      </template>

      <el-alert
        v-if="enabledTypeOptions.length === 0"
        title="请先新增并启用一个模板分类，再创建视频模板。"
        type="warning"
        show-icon
        :closable="false"
        class="type-alert"
      />

      <div class="filters">
        <el-select v-model="query.template_type_id" clearable filterable placeholder="模板分类">
          <el-option v-for="item in typeOptions" :key="item.id" :label="typeLabel(item)" :value="String(item.id)" />
        </el-select>
        <el-select v-model="query.template_type" clearable filterable placeholder="模板类型">
          <el-option v-for="item in templateKinds" :key="item.value" :label="item.label" :value="item.value" />
        </el-select>
        <el-select v-model="query.model_id" clearable filterable placeholder="关联模型">
          <el-option v-for="item in modelOptions" :key="item.id" :label="modelLabel(item)" :value="String(item.id)" />
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
              :src="row.cover_image_url"
              :preview-src-list="[row.cover_image_url]"
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
        <el-table-column label="投放条件" min-width="250">
          <template #default="{ row }">
            <div class="target-tags">
              <el-tag size="small" effect="plain">{{ countrySummary(row.video_template_type?.countries) }}</el-tag>
              <el-tag size="small" type="info" effect="plain">{{ appSummary(row.video_template_type?.apps) }}</el-tag>
              <el-tag size="small" type="info" effect="plain">{{ packageSummary(row.video_template_type?.packages) }}</el-tag>
              <el-tag size="small" type="info" effect="plain">{{ versionSummary(row.video_template_type?.versions) }}</el-tag>
            </div>
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
                @click="previewMedia(mediaKind(row.template_type), row.thumbnail_url, `${row.name} · 缩略资源`)"
              >缩略资源</el-button>
              <span v-else class="secondary-text">无缩略视频</span>
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
        <el-table-column v-if="canEdit || canDelete" label="操作" width="130" fixed="right" align="center">
          <template #default="{ row }">
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

    <el-dialog v-model="dialogVisible" :title="form.id ? '编辑视频模板' : '新增视频模板'" width="820px" destroy-on-close>
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
            <el-select v-model="form.template_type" placeholder="请选择模板类型" style="width: 100%">
              <el-option v-for="item in templateKinds" :key="item.value" :label="item.label" :value="item.value" />
            </el-select>
          </el-form-item>
          <el-form-item label="关联模型" prop="model_id">
            <el-select v-model="form.model_id" filterable placeholder="请选择模型" style="width: 100%" @change="handleModelChange">
              <el-option
                v-for="item in modelOptions"
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
                :src="form.cover_image_url"
                :preview-src-list="[form.cover_image_url]"
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
            :kind="mediaKind(form.template_type)"
            resume-key="template-thumbnail-video"
            placeholder="可选；输入 URL 或选择文件分片上传"
            @preview="(url) => previewMedia(mediaKind(form.template_type), url, '缩略资源预览')"
            @uploading-change="(value) => (mediaUploading.thumbnail = value)"
          />
        </el-form-item>
        <el-form-item label="模型配置">
          <div class="model-parameters" v-loading="parameterLoading">
            <el-empty v-if="!parameterLoading && parameterEditors.length === 0" description="所选模型暂无可配置参数" :image-size="72" />
            <div v-for="editor in parameterEditors" :key="editor.definition.param_key" class="parameter-card">
              <div class="parameter-header">
                <el-checkbox v-model="editor.enabled">
                  <code>{{ editor.definition.param_key }}</code>
                </el-checkbox>
                <div class="parameter-tags">
                  <el-tag size="small" effect="plain">{{ editor.definition.value_type }}</el-tag>
                  <el-tag size="small" :type="editor.definition.parameter_type === 1 ? 'success' : 'warning'">
                    {{ editor.definition.parameter_type === 1 ? '选项' : '请求参数' }}
                  </el-tag>
                </div>
              </div>
              <div v-if="editor.definition.description" class="parameter-description">{{ editor.definition.description }}</div>
              <div v-if="editor.enabled && editor.definition.parameter_type === 1" class="parameter-fields">
                <div>
                  <div class="parameter-label">模板允许值</div>
                  <el-checkbox-group v-model="editor.allowed_values" @change="normalizeOptionDefault(editor)">
                    <el-checkbox
                      v-for="value in editor.definition.allowed_values"
                      :key="valueKey(value)"
                      :value="value"
                    >{{ displayValue(value) }}</el-checkbox>
                  </el-checkbox-group>
                </div>
                <div>
                  <div class="parameter-label">默认值</div>
                  <el-select v-model="editor.default_value" placeholder="请选择默认值" style="width: 100%">
                    <el-option
                      v-for="value in editor.allowed_values"
                      :key="valueKey(value)"
                      :label="displayValue(value)"
                      :value="value"
                    />
                  </el-select>
                </div>
              </div>
              <div v-else-if="editor.enabled" class="parameter-fields request-parameter-fields">
                <div>
                  <div class="parameter-label">默认值（JSON，可留空）</div>
                  <el-input v-model="editor.default_value_text" placeholder='例如："value"、10、true 或 {"key":"value"}' />
                </div>
                <div>
                  <div class="parameter-label">是否必填</div>
                  <el-switch v-model="editor.is_required" :active-value="1" :inactive-value="0" />
                </div>
                <div class="constraints-field">
                  <div class="parameter-label">限制条件（JSON 对象）</div>
                  <el-input v-model="editor.constraints_text" type="textarea" :rows="3" />
                </div>
              </div>
            </div>
          </div>
        </el-form-item>
        <el-form-item label="提示词" prop="prompt">
          <el-input
            v-model="form.prompt"
            type="textarea"
            :rows="4"
            maxlength="65535"
            show-word-limit
            placeholder="输入生成视频所需的提示词"
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
  updateTemplate,
  type TemplateModelParameter,
  type VideoTemplate,
  type VideoTemplatePayload,
  type VideoTemplateType,
} from '@/api/template'
import {
  getModelList,
  getModelParameters,
  type ModelParameter,
  type ModelParameterPayload,
  type VideoModel,
} from '@/api/videoModel'
import { useUserStore } from '@/store/user'
import { getDisplayPositionOptions, type DisplayPosition } from '@/api/displayPosition'
import type { Country } from '@/api/country'
import type { AppPackage } from '@/api/package'
import type { PackageVersion } from '@/api/packageVersion'
import type { VideoApp } from '@/api/videoApp'
import MediaUploader from '@/components/MediaUploader.vue'

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
const dialogVisible = ref(false)
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

interface TemplateParameterEditor {
  definition: ModelParameter
  enabled: boolean
  allowed_values: unknown[]
  default_value: unknown
  default_value_text: string
  constraints_text: string
  is_required: number
  description: string
  sort_order: number
}

const parameterEditors = ref<TemplateParameterEditor[]>([])
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
const rules = {
  name: [{ required: true, message: '请输入模板名称', trigger: 'blur' }],
  template_type_id: [{ required: true, message: '请选择模板分类', trigger: 'change' }],
  model_id: [{ required: true, message: '请选择关联模型', trigger: 'change' }],
  template_type: [{ required: true, message: '请选择模板类型', trigger: 'change' }],
  cover_image_url: [{ required: true, message: '请输入封面图 URL', trigger: 'blur' }],
  original_url: [{ required: true, message: '请输入原始资源 URL', trigger: 'blur' }],
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

function compactSummary(labels: string[], allLabel: string) {
  if (!labels.length) return allLabel
  if (labels.length <= 2) return labels.join('、')
  return `${labels.slice(0, 2).join('、')} 等 ${labels.length} 项`
}

function arrayValue<T>(value: T[] | null | undefined): T[] {
	return Array.isArray(value) ? value : []
}

function countrySummary(items?: Country[] | null) {
	return compactSummary(arrayValue(items).map((item) => `${item.name_zh}·${item.code}`), '全部国家')
}

function packageSummary(items?: AppPackage[] | null) {
	return compactSummary(arrayValue(items).map((item) => item.package_name), '全部安装包')
}

function appSummary(items?: VideoApp[] | null) {
	return compactSummary(arrayValue(items).map((item) => item.name || item.app_code), '全部应用')
}

function versionSummary(items?: PackageVersion[] | null) {
	return compactSummary(arrayValue(items).map((item) => item.version_code), '全部版本')
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

function valueKey(value: unknown) {
  return JSON.stringify(value)
}

function displayValue(value: unknown) {
  return typeof value === 'string' ? value : JSON.stringify(value)
}

function normalizeOptionDefault(editor: TemplateParameterEditor) {
  const allowed = new Set(editor.allowed_values.map(valueKey))
  if (!allowed.has(valueKey(editor.default_value))) {
    editor.default_value = cloneJSON(editor.allowed_values[0] ?? null)
  }
}

async function loadModelParameters(modelID: number, configured?: TemplateModelParameter[]) {
  parameterEditors.value = []
  if (!modelID) return
  parameterLoading.value = true
  try {
    const res: any = await getModelParameters(modelID)
    const definitions = arrayValue<ModelParameter>(res.data)
    const configuredByKey = new Map((configured || []).map((item) => [item.param_key, item]))
    const editingExistingConfiguration = configured !== undefined
    parameterEditors.value = definitions.map((definition) => {
      const configuredItem = configuredByKey.get(definition.param_key)
      const source = configuredItem || definition
      return {
        definition,
        enabled: editingExistingConfiguration ? Boolean(configuredItem) : true,
        allowed_values: cloneJSON(arrayValue(source.allowed_values)),
        default_value: cloneJSON(source.default_value),
        default_value_text: source.default_value === null || source.default_value === undefined
          ? ''
          : JSON.stringify(source.default_value),
        constraints_text: JSON.stringify(source.constraints || {}, null, 2),
        is_required: source.is_required || 0,
        description: source.description || '',
        sort_order: source.sort_order || 0,
      }
    })
  } finally {
    parameterLoading.value = false
  }
}

function buildModelParameterPayloads(): ModelParameterPayload[] {
  return parameterEditors.value.filter((editor) => editor.enabled).map((editor) => {
    const definition = editor.definition
    if (definition.parameter_type === 1) {
      if (editor.allowed_values.length === 0) {
        throw new Error(`模型配置 ${definition.param_key} 至少保留一个允许值`)
      }
      normalizeOptionDefault(editor)
      return {
        param_key: definition.param_key,
        value_type: definition.value_type,
        parameter_type: 1,
        is_required: 0,
        default_value: cloneJSON(editor.default_value),
        allowed_values: cloneJSON(editor.allowed_values),
        constraints: {},
        description: editor.description,
        sort_order: editor.sort_order,
      }
    }

    let constraints: Record<string, unknown>
    let defaultValue: unknown = null
    try {
      constraints = JSON.parse(editor.constraints_text)
      if (!constraints || Array.isArray(constraints) || typeof constraints !== 'object' || Object.keys(constraints).length === 0) {
        throw new Error('限制条件必须是非空 JSON 对象')
      }
      if (editor.default_value_text.trim()) defaultValue = JSON.parse(editor.default_value_text)
    } catch (error) {
      const message = error instanceof Error ? error.message : 'JSON 格式错误'
      throw new Error(`模型配置 ${definition.param_key}：${message}`)
    }
    return {
      param_key: definition.param_key,
      value_type: definition.value_type,
      parameter_type: 2,
      is_required: editor.is_required,
      default_value: defaultValue,
      allowed_values: [],
      constraints,
      description: editor.description,
      sort_order: editor.sort_order,
    }
  })
}

async function fetchTypes() {
	const res: any = await getTemplateTypeOptions()
	typeOptions.value = arrayValue<any>(res.data).map(normalizeTemplateType)
}

async function fetchModels() {
  const res: any = await getModelList({ page: 1, page_size: 100 })
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

async function openCreate() {
  Object.assign(mediaUploading, { cover: false, template: false, thumbnail: false })
  Object.assign(form, defaultForm, {
    template_type_id: enabledTypeOptions.value[0]?.id || 0,
    model_id: enabledModelOptions.value[0]?.id || 0,
  })
  parameterEditors.value = []
  dialogVisible.value = true
  await loadModelParameters(form.model_id)
}

async function openEdit(row: VideoTemplate) {
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
  parameterEditors.value = []
  dialogVisible.value = true
  const res: any = await getTemplateModelParameters(row.id)
  await loadModelParameters(row.model_id, arrayValue<TemplateModelParameter>(res.data?.parameters))
}

async function handleModelChange(modelID: number) {
  await loadModelParameters(modelID)
}

function previewMedia(kind: 'image' | 'video', url: string, title: string) {
  if (!url) return
  Object.assign(preview, { visible: true, kind, url, title })
}

async function handleSubmit() {
  await formRef.value?.validate()
  let modelParameters: ModelParameterPayload[]
  try {
    modelParameters = buildModelParameterPayloads()
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '模型配置格式错误')
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
      model_parameters: modelParameters,
    }
    if (form.id) await updateTemplate(form.id, payload)
    else await createTemplate(payload)
    ElMessage.success('视频模板已保存')
    dialogVisible.value = false
    await fetchData()
  } finally {
    submitting.value = false
  }
}

async function handleDelete(id: number) {
  await deleteTemplate(id)
  ElMessage.success('视频模板已删除')
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
.target-tags { display: flex; align-items: center; flex-wrap: wrap; gap: 5px; }
.media-actions { display: flex; flex-direction: column; align-items: center; }
.media-actions :deep(.el-button + .el-button) { margin-left: 0; }
.prompt-text { display: -webkit-box; overflow: hidden; color: #606266; line-height: 1.5; -webkit-box-orient: vertical; -webkit-line-clamp: 2; }
.pagination-wrap { display: flex; justify-content: flex-end; margin-top: 16px; overflow-x: auto; }
.form-grid { display: grid; grid-template-columns: 1fr 1fr; column-gap: 14px; }
.form-grid :deep(.el-input-number) { width: 100%; }
.cover-field { width: 100%; }
.model-parameters { width: 100%; min-height: 80px; }
.parameter-card { margin-bottom: 10px; padding: 12px; border: 1px solid #ebeef5; border-radius: 8px; background: #fafafa; }
.parameter-card:last-child { margin-bottom: 0; }
.parameter-header { display: flex; align-items: center; justify-content: space-between; gap: 12px; }
.parameter-tags { display: flex; align-items: center; gap: 6px; }
.parameter-description { margin-top: 5px; color: #909399; font-size: 12px; }
.parameter-fields { display: grid; grid-template-columns: minmax(0, 2fr) minmax(160px, 1fr); gap: 12px; margin-top: 12px; padding-top: 12px; border-top: 1px dashed #dcdfe6; }
.request-parameter-fields { grid-template-columns: minmax(0, 2fr) minmax(100px, 1fr); }
.constraints-field { grid-column: 1 / -1; }
.parameter-label { margin-bottom: 5px; color: #606266; font-size: 12px; }
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
  .parameter-fields, .request-parameter-fields { grid-template-columns: 1fr; }
  .page-wrap :deep(.el-card__header), .page-wrap :deep(.el-card__body) { padding: 14px; }
}
</style>
