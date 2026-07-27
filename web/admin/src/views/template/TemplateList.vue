<template>
  <div class="page-wrap">
    <el-card shadow="never">
      <template #header>
        <div class="page-header">
          <div>
            <div class="page-title">视频模板</div>
            <div class="page-subtitle">管理模板自身信息；投放范围由所属模板分类统一控制</div>
          </div>
          <el-button v-if="canAdd" type="primary" :disabled="enabledTypeOptions.length === 0" @click="openCreate">
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
        <el-select v-model="query.video_template_type_id" clearable filterable placeholder="模板分类">
          <el-option v-for="item in typeOptions" :key="item.id" :label="typeLabel(item)" :value="String(item.id)" />
        </el-select>
        <el-select v-model="query.template_type" clearable filterable allow-create placeholder="模板类型">
          <el-option v-for="item in templateKinds" :key="item.value" :label="item.label" :value="item.value" />
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
              :src="row.cover_image"
              :preview-src-list="[row.cover_image]"
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
              <span class="secondary-text">{{ positionSummary(row.video_template_type?.display_positions) }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="所属分类" min-width="210">
          <template #default="{ row }">
            <div class="primary-text">{{ row.video_template_type?.category_name || `分类 #${row.video_template_type_id}` }}</div>
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
              <el-button link type="primary" @click="previewMedia('video', row.template_video, `${row.name} · 模板视频`)">模板视频</el-button>
              <el-button
                v-if="row.thumbnail_video"
                link
                type="primary"
                @click="previewMedia('video', row.thumbnail_video, `${row.name} · 缩略视频`)"
              >缩略视频</el-button>
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
          <el-form-item label="所属分类" prop="video_template_type_id">
            <el-select v-model="form.video_template_type_id" filterable placeholder="请选择模板分类" style="width: 100%">
              <el-option
                v-for="item in typeOptions"
                :key="item.id"
                :label="typeLabel(item)"
                :value="item.id"
                :disabled="item.status !== 1 && item.id !== form.video_template_type_id"
              />
            </el-select>
          </el-form-item>
          <el-form-item label="模板类型" prop="template_type">
            <el-select v-model="form.template_type" filterable allow-create placeholder="选择或输入模板类型" style="width: 100%">
              <el-option v-for="item in templateKinds" :key="item.value" :label="item.label" :value="item.value" />
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

        <el-form-item label="封面图" prop="cover_image">
          <div class="cover-field">
            <MediaUploader
              v-model="form.cover_image"
              kind="image"
              resume-key="template-cover"
              placeholder="输入图片 URL，或选择图片分片上传"
              @preview="(url) => previewMedia('image', url, '封面图预览')"
              @uploading-change="(value) => (mediaUploading.cover = value)"
            />
            <div v-if="form.cover_image" class="cover-form-preview">
              <el-image
                :src="form.cover_image"
                :preview-src-list="[form.cover_image]"
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
        <el-form-item label="模板视频" prop="template_video">
          <MediaUploader
            v-model="form.template_video"
            kind="video"
            resume-key="template-video"
            placeholder="输入视频 URL，或选择视频分片上传"
            @preview="(url) => previewMedia('video', url, '模板视频预览')"
            @uploading-change="(value) => (mediaUploading.template = value)"
          />
        </el-form-item>
        <el-form-item label="缩略视频" prop="thumbnail_video">
          <MediaUploader
            v-model="form.thumbnail_video"
            kind="video"
            resume-key="template-thumbnail-video"
            placeholder="可选；输入 URL 或选择视频分片上传"
            @preview="(url) => previewMedia('video', url, '缩略视频预览')"
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
  getTemplateTypeOptions,
  updateTemplate,
  type VideoTemplate,
  type VideoTemplatePayload,
  type VideoTemplateType,
} from '@/api/template'
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

const templateKinds = [
  { value: 'action', label: '动作模板' },
  { value: 'face_swap', label: '换脸模板' },
]
const loading = ref(false)
const submitting = ref(false)
const dialogVisible = ref(false)
const formRef = ref<FormInstance>()
const tableData = ref<VideoTemplate[]>([])
const typeOptions = ref<VideoTemplateType[]>([])
const positionOptions = ref<DisplayPosition[]>([])
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const query = reactive({
  video_template_type_id: '', template_type: '', position_key: '', status: '', keyword: '',
})

const defaultForm = {
  id: 0,
  video_template_type_id: 0,
  name: '',
  template_type: 'action',
  sort: 0,
  cover_image: '',
  template_video: '',
  thumbnail_video: '',
  prompt: '',
  status: 1,
  description: '',
}
const form = reactive({ ...defaultForm })
const rules = {
  name: [{ required: true, message: '请输入模板名称', trigger: 'blur' }],
  video_template_type_id: [{ required: true, message: '请选择模板分类', trigger: 'change' }],
  template_type: [{ required: true, message: '请选择或输入模板类型', trigger: 'change' }],
  cover_image: [{ required: true, message: '请输入封面图 URL', trigger: 'blur' }],
  template_video: [{ required: true, message: '请输入模板视频 URL', trigger: 'blur' }],
}
const preview = reactive({ visible: false, kind: 'video' as 'image' | 'video', url: '', title: '' })
const mediaUploading = reactive({ cover: false, template: false, thumbnail: false })
const isMediaUploading = computed(() => mediaUploading.cover || mediaUploading.template || mediaUploading.thumbnail)

function typeLabel(item: VideoTemplateType) {
  return item.category_name
}

function kindLabel(kind: string) {
  return templateKinds.find((item) => item.value === kind)?.label || kind
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

function positionSummary(items?: DisplayPosition[] | null) {
	return compactSummary(arrayValue(items).map((item) => item.position_name), '全部展示位置')
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
		video_template_type: item?.video_template_type
			? normalizeTemplateType(item.video_template_type)
			: undefined,
	}
}

async function fetchTypes() {
	const res: any = await getTemplateTypeOptions()
	typeOptions.value = arrayValue<any>(res.data).map(normalizeTemplateType)
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
    video_template_type_id: '', template_type: '', position_key: '', status: '', keyword: '',
  })
  page.value = 1
  fetchData()
}

function openCreate() {
  Object.assign(mediaUploading, { cover: false, template: false, thumbnail: false })
  Object.assign(form, defaultForm, {
    video_template_type_id: typeOptions.value.find((item) => item.status === 1)?.id || 0,
  })
  dialogVisible.value = true
}

function openEdit(row: VideoTemplate) {
  Object.assign(mediaUploading, { cover: false, template: false, thumbnail: false })
  Object.assign(form, {
    id: row.id,
    video_template_type_id: row.video_template_type_id,
    name: row.name,
    template_type: row.template_type,
    sort: row.sort,
    cover_image: row.cover_image,
    template_video: row.template_video,
    thumbnail_video: row.thumbnail_video || '',
    prompt: row.prompt || '',
    status: row.status,
    description: row.description || '',
  })
  dialogVisible.value = true
}

function previewMedia(kind: 'image' | 'video', url: string, title: string) {
  if (!url) return
  Object.assign(preview, { visible: true, kind, url, title })
}

async function handleSubmit() {
  await formRef.value?.validate()
  submitting.value = true
  try {
    const payload: VideoTemplatePayload = {
      video_template_type_id: form.video_template_type_id,
      name: form.name.trim(),
      template_type: form.template_type.trim(),
      sort: form.sort,
      cover_image: form.cover_image.trim(),
      template_video: form.template_video.trim(),
      thumbnail_video: form.thumbnail_video.trim(),
      prompt: form.prompt.trim(),
      status: form.status,
      description: form.description.trim(),
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

onMounted(() => Promise.all([fetchTypes(), fetchPositions(), fetchData()]))
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
