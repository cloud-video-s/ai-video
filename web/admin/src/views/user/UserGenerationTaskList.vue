<template>
  <div class="page-wrap">
    <el-card shadow="never">
      <template #header>
        <div class="page-heading">
          <div>
            <div class="page-title">任务管理</div>
            <div class="page-subtitle">查看客户端用户的图片、视频生成任务及完整处理详情</div>
          </div>
          <el-tag effect="plain">共 {{ total.toLocaleString('zh-CN') }} 条</el-tag>
        </div>
      </template>

      <div class="filters">
        <el-input
          v-model="query.keyword"
          clearable
          placeholder="用户、模型、提示词、客户端或第三方任务号"
          @keyup.enter="handleSearch"
        >
          <template #prefix><el-icon><Search /></el-icon></template>
        </el-input>
        <el-input-number v-model="query.user_id" :min="1" :controls="false" placeholder="用户 ID" />
        <el-input v-model="query.task_code" clearable placeholder="任务编码（精确）" @keyup.enter="handleSearch" />
        <el-select v-model="query.model_id" clearable filterable placeholder="生成模型">
          <el-option
            v-for="item in modelOptions"
            :key="item.id"
            :label="`${item.name} · ${item.code}`"
            :value="String(item.id)"
          />
        </el-select>
        <el-select v-model="query.media_type" clearable placeholder="媒体类型">
          <el-option label="图片" value="1" />
          <el-option label="视频" value="2" />
        </el-select>
        <el-select v-model="query.status" clearable placeholder="任务状态">
          <el-option v-for="item in statusOptions" :key="item.value" :label="item.label" :value="String(item.value)" />
        </el-select>
        <el-date-picker
          v-model="dateRange"
          type="daterange"
          value-format="YYYY-MM-DD"
          range-separator="至"
          start-placeholder="开始日期"
          end-placeholder="结束日期"
          style="width: 100%"
        />
        <div class="filter-actions">
          <el-button type="primary" plain @click="handleSearch">查询</el-button>
          <el-button @click="handleReset">重置</el-button>
        </div>
      </div>

      <el-table v-loading="loading" :data="tableData" row-key="id" stripe>
        <el-table-column label="任务" min-width="220">
          <template #default="{ row }">
            <div class="primary-text task-code">{{ row.task_code }}</div>
            <div class="secondary-text">ID {{ row.id }} · {{ row.client_request_id || '无客户端请求号' }}</div>
            <div v-if="row.prompt" class="prompt-summary">{{ row.prompt }}</div>
          </template>
        </el-table-column>
        <el-table-column label="用户" min-width="190">
          <template #default="{ row }">
            <div class="primary-text">{{ userName(row) }}</div>
            <div class="secondary-text">ID {{ row.user_id }} · {{ userAccount(row) }}</div>
          </template>
        </el-table-column>
        <el-table-column label="模型 / 类型" min-width="170">
          <template #default="{ row }">
            <div class="primary-text">{{ row.model?.name || `模型 #${row.model_id}` }}</div>
            <div class="secondary-text">{{ row.model?.code || '-' }}</div>
            <el-tag class="media-tag" size="small" effect="plain" :type="row.media_type === 'video' ? 'warning' : 'success'">
              {{ mediaTypeLabel(row.media_type) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="状态 / 进度" width="180">
          <template #default="{ row }">
            <el-tag size="small" :type="statusType(row.status)">{{ statusLabel(row.status) }}</el-tag>
            <el-progress
              class="task-progress"
              :percentage="normalizeProgress(row.progress)"
              :stroke-width="7"
              :status="progressStatus(row.status)"
            />
          </template>
        </el-table-column>
        <el-table-column label="生成结果" width="110" align="center">
          <template #default="{ row }">
            <strong>{{ row.result_count }}</strong>
            <div class="secondary-text">{{ row.local_urls.length ? '本地文件' : row.remote_urls.length ? '远程文件' : '暂无结果' }}</div>
          </template>
        </el-table-column>
        <el-table-column label="耗时" width="95" align="right">
          <template #default="{ row }">{{ formatDuration(row.usage_duration) }}</template>
        </el-table-column>
        <el-table-column label="消耗积分" width="100" align="right">
          <template #default="{ row }">{{ row.score }}</template>
        </el-table-column>
        <el-table-column label="创建时间" width="180">
          <template #default="{ row }">{{ formatDate(row.created_at) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="132" fixed="right" align="center">
          <template #default="{ row }">
            <el-button link type="primary" :disabled="!row.preview_urls.length" @click="openPreview(row)">预览</el-button>
            <el-button link type="primary" @click="openDetail(row.id)">详情</el-button>
          </template>
        </el-table-column>
        <template #empty>
          <el-empty description="暂无符合条件的生成任务" />
        </template>
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

    <el-dialog v-model="preview.visible" :title="preview.title" width="860px" destroy-on-close append-to-body>
      <div v-if="preview.urls.length" class="preview-grid" :class="{ single: preview.urls.length === 1 }">
        <div v-for="(item, index) in preview.urls" :key="`${item}-${index}`" class="preview-item">
          <video
            v-if="isVideoPreview(item, preview.mediaType)"
            :src="item"
            controls
            playsinline
            preload="metadata"
            class="preview-media"
          />
          <el-image
            v-else
            :src="item"
            :preview-src-list="preview.urls"
            :initial-index="index"
            fit="contain"
            hide-on-click-modal
            preview-teleported
            class="preview-media"
          >
            <template #error><div class="media-error">图片加载失败</div></template>
          </el-image>
          <a :href="toMediaURL(item)" target="_blank" rel="noopener noreferrer" class="source-link">在新窗口打开</a>
        </div>
      </div>
      <el-empty v-else description="该任务暂无可预览结果" />
    </el-dialog>

    <el-drawer v-model="detailVisible" title="任务详情" size="min(900px, 92vw)" destroy-on-close>
      <el-skeleton v-if="detailLoading" :rows="12" animated />
      <div v-else-if="detail" class="detail-wrap">
        <el-alert
          v-if="detail.error_message"
          title="任务失败信息"
          :description="detail.error_message"
          type="error"
          :closable="false"
          show-icon
        />

        <el-descriptions :column="2" border>
          <el-descriptions-item label="任务 ID">{{ detail.id }}</el-descriptions-item>
          <el-descriptions-item label="状态">
            <el-tag size="small" :type="statusType(detail.status)">{{ statusLabel(detail.status) }}</el-tag>
            <span class="detail-progress">{{ normalizeProgress(detail.progress) }}%</span>
          </el-descriptions-item>
          <el-descriptions-item label="任务编码">{{ detail.task_code }}</el-descriptions-item>
          <el-descriptions-item label="第三方任务号">{{ detail.third_task_code || '-' }}</el-descriptions-item>
          <el-descriptions-item label="客户端请求号" :span="2">{{ detail.client_request_id || '-' }}</el-descriptions-item>
          <el-descriptions-item label="用户">{{ userName(detail) }}（ID {{ detail.user_id }}）</el-descriptions-item>
          <el-descriptions-item label="用户账号">{{ userAccount(detail) }}</el-descriptions-item>
          <el-descriptions-item label="模型">
            {{ detail.model?.name || `模型 #${detail.model_id}` }}
            <span v-if="detail.model?.code" class="secondary-inline">（{{ detail.model.code }}{{ detail.model.version ? ` · ${detail.model.version}` : '' }}）</span>
          </el-descriptions-item>
          <el-descriptions-item label="任务类型">{{ mediaTypeLabel(detail.media_type) }}（{{ detail.task_type }}）</el-descriptions-item>
          <el-descriptions-item label="模板 ID">{{ detail.template_id || '-' }}</el-descriptions-item>
          <el-descriptions-item label="消耗积分">{{ detail.score }}</el-descriptions-item>
          <el-descriptions-item label="处理耗时">{{ formatDuration(detail.usage_duration) }}</el-descriptions-item>
          <el-descriptions-item label="结果数量">{{ detail.result_count }}</el-descriptions-item>
          <el-descriptions-item label="提交时间">{{ formatDate(detail.submitted_at) }}</el-descriptions-item>
          <el-descriptions-item label="开始时间">{{ formatDate(detail.started_at) }}</el-descriptions-item>
          <el-descriptions-item label="完成时间">{{ formatDate(detail.finished_at) }}</el-descriptions-item>
          <el-descriptions-item label="最后轮询">{{ formatDate(detail.last_polled_at) }}</el-descriptions-item>
          <el-descriptions-item label="创建时间">{{ formatDate(detail.created_at) }}</el-descriptions-item>
          <el-descriptions-item label="更新时间">{{ formatDate(detail.updated_at) }}</el-descriptions-item>
        </el-descriptions>

        <section v-if="detail.cover_image_url" class="detail-section">
          <div class="section-heading">封面图</div>
          <el-image
            :src="toMediaURL(detail.cover_image_url)"
            :preview-src-list="[toMediaURL(detail.cover_image_url)]"
            fit="contain"
            preview-teleported
            class="cover-image"
          />
        </section>

        <section class="detail-section">
          <div class="section-heading">
            <span>生成结果</span>
            <el-button v-if="detail.preview_urls.length" link type="primary" @click="openPreview(detail)">预览全部</el-button>
          </div>
          <div v-if="detail.preview_urls.length" class="result-links">
            <a
              v-for="(item, index) in detail.preview_urls"
              :key="item"
              :href="toMediaURL(item)"
              target="_blank"
              rel="noopener noreferrer"
            >结果 {{ index + 1 }}</a>
          </div>
          <el-empty v-else :image-size="56" description="暂无生成结果" />
        </section>

        <section class="detail-section">
          <div class="section-heading">提示词</div>
          <div class="text-block">{{ detail.prompt || '-' }}</div>
        </section>

        <section class="detail-section">
          <div class="section-heading">请求参数</div>
          <pre class="json-block">{{ formatJSON(detail.request_payload) }}</pre>
        </section>

        <section class="detail-section">
          <div class="section-heading">提供商响应</div>
          <pre class="json-block">{{ formatJSON(detail.provider_response) }}</pre>
        </section>

        <section class="detail-section">
          <div class="section-heading">媒体地址</div>
          <div class="url-group">
            <strong>本地地址</strong>
            <span v-if="!detail.local_urls.length" class="secondary-text">无</span>
            <a v-for="item in detail.local_urls" :key="item" :href="toMediaURL(item)" target="_blank" rel="noopener noreferrer">{{ item }}</a>
          </div>
          <div class="url-group">
            <strong>远程地址</strong>
            <span v-if="!detail.remote_urls.length" class="secondary-text">无</span>
            <a v-for="item in detail.remote_urls" :key="item" :href="toMediaURL(item)" target="_blank" rel="noopener noreferrer">{{ item }}</a>
          </div>
        </section>
      </div>
    </el-drawer>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { getModelList, type VideoModel } from '@/api/videoModel'
import {
  getUserGenerationTask,
  getUserGenerationTaskList,
  type GenerationTaskMediaType,
  type UserGenerationTask,
} from '@/api/userGenerationTask'
import { toMediaURL } from '@/utils/mediaUrl'

const statusOptions = [
  { value: 1, label: '提交中' },
  { value: 2, label: '已提交' },
  { value: 3, label: '等待处理' },
  { value: 4, label: '生成中' },
  { value: 5, label: '下载中' },
  { value: 6, label: '成功' },
  { value: 7, label: '失败' },
]

const loading = ref(false)
const detailLoading = ref(false)
const detailVisible = ref(false)
const tableData = ref<UserGenerationTask[]>([])
const detail = ref<UserGenerationTask | null>(null)
const modelOptions = ref<VideoModel[]>([])
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const dateRange = ref<string[]>([])
const query = reactive({
  keyword: '',
  user_id: undefined as number | undefined,
  task_code: '',
  model_id: '',
  media_type: '',
  status: '',
})
const preview = reactive({
  visible: false,
  title: '',
  mediaType: 'unknown' as GenerationTaskMediaType,
  urls: [] as string[],
})

function statusLabel(value: number) {
  return statusOptions.find((item) => item.value === value)?.label || `未知（${value}）`
}

function statusType(value: number) {
  if (value === 6) return 'success'
  if (value === 7) return 'danger'
  if (value === 4 || value === 5) return 'warning'
  return 'info'
}

function progressStatus(value: number) {
  if (value === 6) return 'success'
  if (value === 7) return 'exception'
  return undefined
}

function normalizeProgress(value: number) {
  return Math.min(100, Math.max(0, Number(value || 0)))
}

function mediaTypeLabel(value: GenerationTaskMediaType) {
  if (value === 'image') return '图片'
  if (value === 'video') return '视频'
  return '未知'
}

function userName(row: UserGenerationTask) {
  return row.user?.username || row.user?.email || row.user?.login_account || `用户 #${row.user_id}`
}

function userAccount(row: UserGenerationTask) {
  return row.user?.email || row.user?.login_account || row.user?.imei || row.user?.device_code || '-'
}

function formatDate(value?: string | null) {
  if (!value) return '-'
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString('zh-CN', { hour12: false })
}

function formatDuration(value: number) {
  const seconds = Number(value || 0)
  if (seconds < 60) return `${seconds} 秒`
  const minutes = Math.floor(seconds / 60)
  const rest = seconds % 60
  return rest ? `${minutes} 分 ${rest} 秒` : `${minutes} 分`
}

function isVideoPreview(rawURL: string, mediaType: GenerationTaskMediaType) {
  if (mediaType === 'video') return true
  if (mediaType === 'image') return false
  return /\.(mp4|webm|mov|m4v|avi|mkv)(?:[?#]|$)/i.test(rawURL)
}

function formatJSON(value: unknown) {
  if (value === undefined || value === null || value === '') return '-'
  if (typeof value === 'string') return value
  try {
    return JSON.stringify(value, null, 2)
  } catch {
    return String(value)
  }
}

async function fetchModels() {
  try {
    const res: any = await getModelList({ page: 1, page_size: 100 })
    modelOptions.value = res.data.list || []
  } catch {
    modelOptions.value = []
  }
}

async function fetchData() {
  loading.value = true
  try {
    const params: Record<string, unknown> = { page: page.value, page_size: pageSize.value }
    for (const [key, value] of Object.entries(query)) {
      if (value !== '' && value !== undefined) {
        params[key] = typeof value === 'string' ? value.trim() : value
      }
    }
    if (dateRange.value.length === 2) {
      params.date_from = dateRange.value[0]
      params.date_to = dateRange.value[1]
    }
    const res: any = await getUserGenerationTaskList(params)
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

function handlePageSizeChange() {
  page.value = 1
  fetchData()
}

function handleReset() {
  Object.assign(query, {
    keyword: '', user_id: undefined, task_code: '', model_id: '', media_type: '', status: '',
  })
  dateRange.value = []
  page.value = 1
  fetchData()
}

function openPreview(row: UserGenerationTask) {
  preview.title = `${mediaTypeLabel(row.media_type)}生成结果 · ${row.task_code}`
  preview.mediaType = row.media_type
  preview.urls = (row.preview_urls || []).map(toMediaURL)
  preview.visible = true
}

async function openDetail(id: number) {
  detailVisible.value = true
  detailLoading.value = true
  detail.value = null
  try {
    const res: any = await getUserGenerationTask(id)
    detail.value = res.data
  } finally {
    detailLoading.value = false
  }
}

onMounted(async () => {
  await Promise.all([fetchModels(), fetchData()])
})
</script>

<style scoped>
.page-wrap { min-width: 0; }
.page-heading { display: flex; align-items: center; justify-content: space-between; gap: 16px; }
.page-title { color: #303133; font-size: 17px; font-weight: 600; }
.page-subtitle { margin-top: 4px; color: #909399; font-size: 12px; }
.filters { display: grid; grid-template-columns: minmax(260px, 1.6fr) minmax(120px, .7fr) minmax(170px, 1fr) minmax(190px, 1fr) minmax(120px, .7fr) minmax(130px, .8fr) minmax(240px, 1.2fr) auto; gap: 10px; margin-bottom: 16px; }
.filters :deep(.el-input-number) { width: 100%; }
.filter-actions { display: flex; white-space: nowrap; }
.primary-text { color: #303133; font-weight: 600; }
.secondary-text { margin-top: 4px; color: #909399; font-size: 12px; }
.task-code { font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace; }
.prompt-summary { display: -webkit-box; margin-top: 7px; overflow: hidden; color: #606266; font-size: 12px; line-height: 1.45; -webkit-box-orient: vertical; -webkit-line-clamp: 2; }
.media-tag { margin-top: 7px; }
.task-progress { margin-top: 8px; }
.task-progress :deep(.el-progress__text) { min-width: 38px; font-size: 12px !important; }
.pagination-wrap { display: flex; justify-content: flex-end; margin-top: 16px; overflow-x: auto; }
.preview-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 16px; max-height: 70vh; overflow-y: auto; }
.preview-grid.single { grid-template-columns: 1fr; }
.preview-item { min-width: 0; padding: 10px; border: 1px solid #ebeef5; border-radius: 8px; background: #f7f8fa; text-align: center; }
.preview-media { display: block; width: 100%; height: 320px; border-radius: 5px; background: #111; object-fit: contain; }
.media-error { display: flex; width: 100%; height: 100%; align-items: center; justify-content: center; color: #909399; background: #f5f7fa; }
.source-link { display: inline-block; margin-top: 9px; color: #409eff; font-size: 12px; text-decoration: none; }
.detail-wrap { padding-bottom: 24px; }
.detail-wrap > .el-alert { margin-bottom: 16px; }
.detail-progress { margin-left: 8px; color: #606266; }
.secondary-inline { color: #909399; font-size: 12px; }
.detail-section { margin-top: 20px; }
.cover-image { width: min(360px, 100%); height: 220px; margin-top: 8px; border: 1px solid #ebeef5; border-radius: 6px; background: #f7f8fa; }
.section-heading { display: flex; min-height: 28px; align-items: center; justify-content: space-between; color: #303133; font-size: 14px; font-weight: 600; }
.text-block, .json-block { margin: 8px 0 0; padding: 13px 15px; border: 1px solid #ebeef5; border-radius: 6px; background: #f7f8fa; color: #303133; font-size: 13px; line-height: 1.65; white-space: pre-wrap; overflow-wrap: anywhere; }
.json-block { max-height: 360px; overflow: auto; font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace; }
.result-links { display: flex; flex-wrap: wrap; gap: 10px; margin-top: 8px; }
.result-links a { color: #409eff; text-decoration: none; }
.url-group { display: grid; grid-template-columns: 90px minmax(0, 1fr); gap: 8px 12px; margin-top: 10px; }
.url-group a { min-width: 0; overflow-wrap: anywhere; color: #409eff; text-decoration: none; }
@media (max-width: 1500px) {
  .filters { grid-template-columns: repeat(4, minmax(160px, 1fr)); }
}
@media (max-width: 900px) {
  .filters { grid-template-columns: repeat(2, minmax(160px, 1fr)); }
  .preview-grid { grid-template-columns: 1fr; }
  .preview-media { height: 260px; }
}
@media (max-width: 560px) {
  .filters { grid-template-columns: 1fr; }
  .page-heading { align-items: flex-start; }
  .filter-actions .el-button { flex: 1; }
  .preview-media { height: 220px; }
}
</style>
