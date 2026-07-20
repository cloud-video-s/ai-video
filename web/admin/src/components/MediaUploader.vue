<template>
  <div class="media-uploader">
    <div class="input-row">
      <el-input
        :model-value="modelValue"
        maxlength="1024"
        clearable
        :placeholder="placeholder"
        @update:model-value="(value: string) => emit('update:modelValue', value)"
      />
      <el-upload
        ref="uploadRef"
        :auto-upload="false"
        :show-file-list="false"
        :accept="accept"
        :disabled="busy"
        :on-change="handleFileChange"
      >
        <el-button :loading="state === 'preparing'" :disabled="busy">
          <el-icon><Upload /></el-icon>{{ kind === 'image' ? (modelValue ? '重新选择并裁剪' : '选择并裁剪') : (modelValue ? '重新上传' : '选择文件') }}
        </el-button>
      </el-upload>
      <el-button v-if="modelValue" @click="emit('preview', modelValue)">预览</el-button>
    </div>

    <div v-if="fileName || active" class="upload-status">
      <div class="status-head">
        <span class="file-name">{{ fileName || '上传任务' }}</span>
        <span class="status-text">{{ statusText }}</span>
      </div>
      <el-progress :percentage="progress" :status="progressStatus" :stroke-width="8" />
      <div class="status-actions">
        <span class="resume-tip">分片上传；中断后重新选择同一文件即可续传</span>
        <div>
          <el-button v-if="state === 'uploading'" link type="warning" @click="pauseUpload">暂停</el-button>
          <el-button v-if="state === 'paused' || state === 'error'" link type="primary" @click="resumeUpload">继续</el-button>
          <el-button v-if="active" link type="danger" @click="removeTask">移除任务</el-button>
        </div>
      </div>
    </div>

    <ImageCropDialog
      v-if="kind === 'image'"
      v-model="cropVisible"
      :file="pendingImage"
      @confirm="handleCroppedImage"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { ElMessage, type UploadFile, type UploadInstance } from 'element-plus'
import {
  completeUpload,
  getUploadStatus,
  initiateUpload,
  uploadChunk,
  type MediaKind,
  type UploadSession,
} from '@/api/upload'
import { useUserStore } from '@/store/user'
import ImageCropDialog from '@/components/ImageCropDialog.vue'

const props = defineProps<{
  modelValue: string
  kind: MediaKind
  resumeKey: string
  placeholder?: string
}>()
const emit = defineEmits<{
  'update:modelValue': [value: string]
  'uploading-change': [value: boolean]
  preview: [value: string]
}>()

type UploadState = 'idle' | 'preparing' | 'uploading' | 'paused' | 'merging' | 'done' | 'error'
type ResumeRecord = {
  upload_id: string
  file_name: string
  file_size: number
  last_modified: number
  expires_at: string
}

const userStore = useUserStore()
const uploadRef = ref<UploadInstance>()
const file = ref<File>()
const fileName = ref('')
const session = ref<UploadSession>()
const progress = ref(0)
const state = ref<UploadState>('idle')
const paused = ref(false)
const controller = ref<AbortController>()
const storageKey = ref('')
const pendingImage = ref<File>()
const cropVisible = ref(false)
let disposed = false

const accept = computed(() => props.kind === 'image' ? 'image/jpeg,image/png,image/gif,image/webp' : 'video/mp4,video/quicktime,video/webm,video/x-matroska')
const busy = computed(() => ['preparing', 'uploading', 'merging'].includes(state.value))
const active = computed(() => !['idle', 'done'].includes(state.value))
const progressStatus = computed(() => state.value === 'done' ? 'success' : state.value === 'error' ? 'exception' : undefined)
const statusText = computed(() => ({
  idle: '', preparing: '正在检查断点…', uploading: `上传中 ${progress.value}%`, paused: `已暂停 ${progress.value}%`,
  merging: '正在合并文件…', done: '上传完成', error: '上传中断，可继续',
})[state.value])

watch(busy, (value) => emit('uploading-change', value), { immediate: true })

function buildStorageKey(selected: File) {
  const owner = userStore.userInfo?.id || 'unknown'
  return `template-media-upload:${owner}:${props.resumeKey}:${props.kind}:${selected.name}:${selected.size}:${selected.lastModified}`
}

function loadResumeRecord(): ResumeRecord | undefined {
  if (!storageKey.value) return undefined
  try {
    const raw = localStorage.getItem(storageKey.value)
    if (!raw) return undefined
    const record = JSON.parse(raw) as ResumeRecord
    if (new Date(record.expires_at).getTime() <= Date.now()) {
      localStorage.removeItem(storageKey.value)
      return undefined
    }
    return record
  } catch {
    localStorage.removeItem(storageKey.value)
    return undefined
  }
}

function saveResumeRecord(current: UploadSession) {
  if (!file.value || !storageKey.value) return
  const record: ResumeRecord = {
    upload_id: current.upload_id,
    file_name: file.value.name,
    file_size: file.value.size,
    last_modified: file.value.lastModified,
    expires_at: current.expires_at,
  }
  localStorage.setItem(storageKey.value, JSON.stringify(record))
}

async function handleFileChange(uploadFile: UploadFile) {
  const selected = uploadFile.raw
  uploadRef.value?.clearFiles()
  if (!selected) return
  if (props.kind === 'image') {
    if (!['image/jpeg', 'image/png', 'image/webp', 'image/gif'].includes(selected.type)) {
      ElMessage.warning('仅支持 JPG、PNG、WebP 或 GIF 图片')
      return
    }
    pendingImage.value = selected
    cropVisible.value = true
    return
  }
  await prepareUpload(selected)
}

function handleCroppedImage(selected: File) {
  pendingImage.value = undefined
  void prepareUpload(selected)
}

async function prepareUpload(selected: File) {
  controller.value?.abort()
  file.value = selected
  fileName.value = selected.name
  storageKey.value = buildStorageKey(selected)
  progress.value = 0
  paused.value = false
  state.value = 'preparing'

  try {
    const record = loadResumeRecord()
    let current: UploadSession | undefined
    if (record) {
      try {
        const statusResponse: any = await getUploadStatus(props.kind, record.upload_id)
        current = statusResponse.data as UploadSession
      } catch (error: any) {
        if (error?.response?.status === 404 || error?.response?.status === 410) {
          localStorage.removeItem(storageKey.value)
        } else {
          throw error
        }
      }
    }
    if (!current) {
      const response: any = await initiateUpload(props.kind, selected)
      current = response.data.uploads?.[0] as UploadSession | undefined
    }
    if (disposed) return
    if (!current) throw new Error('上传初始化失败')
    session.value = current
    saveResumeRecord(current)
    if (current.completed) {
      finishUpload(current)
      return
    }
    await runUpload()
  } catch (error: any) {
    if (error?.code === 'ERR_CANCELED') return
    state.value = 'error'
    if (!error?.response) ElMessage.error(error?.message || '上传初始化失败')
  }
}

function chunkSizeAt(current: UploadSession, index: number) {
  const start = index * current.chunk_size
  return Math.max(0, Math.min(current.chunk_size, current.total_size - start))
}

function uploadedBytes(current: UploadSession, uploaded: Set<number>) {
  let total = 0
  for (const index of uploaded) total += chunkSizeAt(current, index)
  return total
}

function setProgress(bytes: number, total: number) {
  progress.value = total > 0 ? Math.min(99, Math.round(bytes * 100 / total)) : 0
}

async function chunkSHA256(chunk: Blob) {
  if (!globalThis.crypto?.subtle) return ''
  const digest = await crypto.subtle.digest('SHA-256', await chunk.arrayBuffer())
  return Array.from(new Uint8Array(digest), (byte) => byte.toString(16).padStart(2, '0')).join('')
}

async function runUpload() {
  const selected = file.value
  const current = session.value
  if (!selected || !current) return
  paused.value = false
  state.value = 'uploading'
  const uploaded = new Set(current.uploaded_chunks || [])
  let completedBytes = uploadedBytes(current, uploaded)
  setProgress(completedBytes, current.total_size)

  try {
    for (let index = 0; index < current.total_chunks; index++) {
      if (uploaded.has(index)) continue
      if (paused.value) {
        state.value = 'paused'
        return
      }
      const start = index * current.chunk_size
      const chunk = selected.slice(start, Math.min(start + current.chunk_size, selected.size))
      const checksum = await chunkSHA256(chunk)
      if (paused.value) {
        state.value = 'paused'
        return
      }
      controller.value = new AbortController()
      await uploadChunk(props.kind, current.upload_id, index, chunk, checksum, controller.value.signal, (loaded) => {
        setProgress(completedBytes + loaded, current.total_size)
      })
      uploaded.add(index)
      current.uploaded_chunks = [...uploaded].sort((a, b) => a - b)
      completedBytes += chunk.size
      saveResumeRecord(current)
      setProgress(completedBytes, current.total_size)
    }

    state.value = 'merging'
    const response: any = await completeUpload(props.kind, current.upload_id)
    finishUpload(response.data as UploadSession)
  } catch (error: any) {
    if (error?.code === 'ERR_CANCELED' && paused.value) {
      state.value = 'paused'
      return
    }
    state.value = 'error'
  }
}

function finishUpload(completed: UploadSession) {
  session.value = completed
  progress.value = 100
  state.value = 'done'
  paused.value = false
  if (storageKey.value) localStorage.removeItem(storageKey.value)
  const url = completed.file_url || (completed.file_path ? `/uploads/${completed.file_path}` : '')
  if (url) emit('update:modelValue', url)
  ElMessage.success(`${fileName.value || '文件'}上传完成`)
}

function pauseUpload() {
  paused.value = true
  controller.value?.abort()
  state.value = 'paused'
}

function resumeUpload() {
  if (!file.value || !session.value) {
    ElMessage.warning('请重新选择同一文件以恢复上传')
    return
  }
  void runUpload()
}

function removeTask() {
  paused.value = true
  controller.value?.abort()
  if (storageKey.value) localStorage.removeItem(storageKey.value)
  file.value = undefined
  fileName.value = ''
  session.value = undefined
  progress.value = 0
  state.value = 'idle'
}

onBeforeUnmount(() => {
  disposed = true
  paused.value = true
  controller.value?.abort()
  emit('uploading-change', false)
})
</script>

<style scoped>
.media-uploader { width: 100%; }
.input-row { display: flex; align-items: center; gap: 8px; }
.input-row :deep(.el-input) { flex: 1; }
.input-row :deep(.el-upload) { display: block; }
.upload-status { margin-top: 8px; padding: 10px 12px; border: 1px solid #ebeef5; border-radius: 6px; background: #fafafa; }
.status-head, .status-actions { display: flex; align-items: center; justify-content: space-between; gap: 12px; }
.status-head { margin-bottom: 6px; font-size: 12px; }
.file-name { min-width: 0; overflow: hidden; color: #606266; text-overflow: ellipsis; white-space: nowrap; }
.status-text { flex: 0 0 auto; color: #909399; }
.status-actions { margin-top: 4px; }
.resume-tip { color: #a8abb2; font-size: 11px; }
@media (max-width: 680px) {
  .input-row { align-items: stretch; flex-wrap: wrap; }
  .input-row :deep(.el-input) { flex-basis: 100%; }
  .status-actions { align-items: flex-start; flex-direction: column; }
}
</style>
