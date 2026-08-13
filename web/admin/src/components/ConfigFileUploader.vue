<template>
  <div class="config-file-uploader">
    <el-input
      :model-value="modelValue"
      clearable
      maxlength="2048"
      placeholder="可上传文件或填写 HTTP/HTTPS 地址"
      :disabled="disabled || uploading"
      @update:model-value="emit('update:modelValue', String($event ?? ''))"
    />
    <input ref="fileInput" class="file-input" type="file" :accept="accept" @change="handleFileChange" />
    <el-tooltip :content="uploadDisabled ? '没有文件上传权限' : '上传配置文件（最大 5 MB）'">
      <el-button
        :icon="Upload"
        circle
        :loading="uploading"
        :disabled="disabled || uploadDisabled || !configKey"
        @click="selectFile"
      />
    </el-tooltip>
    <el-tooltip content="打开文件">
      <el-button :icon="View" circle :disabled="!modelValue || uploading" @click="openFile" />
    </el-tooltip>
    <el-tooltip content="清除文件地址">
      <el-button
        :icon="Delete"
        circle
        type="danger"
        plain
        :disabled="disabled || !modelValue || uploading"
        @click="emit('update:modelValue', '')"
      />
    </el-tooltip>
    <el-progress v-if="uploading" class="upload-progress" :percentage="progress" :stroke-width="6" />
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { Delete, Upload, View } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { uploadConfigFile } from '@/api/upload'
import { toMediaURL } from '@/utils/mediaUrl'

const props = withDefaults(defineProps<{
  modelValue: string
  configKey: string
  disabled?: boolean
  uploadDisabled?: boolean
}>(), {
  disabled: false,
  uploadDisabled: false,
})

const emit = defineEmits<{ (event: 'update:modelValue', value: string): void }>()
const fileInput = ref<HTMLInputElement>()
const uploading = ref(false)
const progress = ref(0)
const maxFileSize = 5 * 1024 * 1024
const extensions = ['.txt', '.html', '.htm', '.md', '.json', '.xml', '.pdf']
const accept = [...extensions, 'text/plain', 'text/html', 'text/markdown', 'application/json', 'application/xml', 'application/pdf'].join(',')

function selectFile() {
  if (!props.disabled && !props.uploadDisabled && props.configKey && !uploading.value) fileInput.value?.click()
}

async function handleFileChange(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  input.value = ''
  if (!file) return
  const extension = file.name.includes('.') ? `.${file.name.split('.').pop()?.toLowerCase()}` : ''
  if (!extensions.includes(extension)) {
    ElMessage.warning('仅支持 TXT、HTML、Markdown、JSON、XML 或 PDF 文件')
    return
  }
  if (file.size <= 0 || file.size > maxFileSize) {
    ElMessage.warning('配置文件必须大于 0 且不能超过 5 MB')
    return
  }

  uploading.value = true
  progress.value = 0
  try {
    const result = await uploadConfigFile(props.configKey, file, (value) => { progress.value = value })
    if (!result.file_url) throw new Error('上传完成后未返回文件地址')
    emit('update:modelValue', result.file_url)
    ElMessage.success(`${result.original_name || file.name} 上传完成，请点击“保存”发布配置`)
  } finally {
    uploading.value = false
  }
}

function openFile() {
  if (!props.modelValue) return
  window.open(toMediaURL(props.modelValue), '_blank', 'noopener,noreferrer')
}
</script>

<style scoped>
.config-file-uploader {
  display: grid;
  grid-template-columns: minmax(240px, 1fr) 32px 32px 32px;
  align-items: center;
  gap: 8px;
  width: 100%;
}
.file-input { display: none; }
.upload-progress { grid-column: 1 / -1; }
@media (max-width: 720px) {
  .config-file-uploader { grid-template-columns: minmax(160px, 1fr) 32px 32px 32px; }
}
</style>
