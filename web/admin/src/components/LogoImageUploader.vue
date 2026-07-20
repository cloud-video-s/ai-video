<template>
  <div class="logo-uploader">
    <div class="logo-preview">
      <el-image v-if="modelValue" :src="modelValue" fit="contain" preview-teleported :preview-src-list="[modelValue]">
        <template #error><el-icon><Picture /></el-icon></template>
      </el-image>
      <el-icon v-else><Picture /></el-icon>
    </div>
    <el-input
      :model-value="modelValue"
      clearable
      placeholder="Logo URL"
      :disabled="disabled"
      @update:model-value="emit('update:modelValue', String($event ?? ''))"
    />
    <input ref="fileInput" class="file-input" type="file" accept="image/jpeg,image/png,image/webp,image/gif" @change="handleFileChange" />
    <el-tooltip :content="uploading ? `上传中 ${progress}%` : '选择并裁剪 Logo'">
      <el-button :icon="Upload" circle :loading="uploading" :disabled="disabled || uploadDisabled" @click="selectFile" />
    </el-tooltip>
    <el-tooltip content="清除 Logo">
      <el-button
        :icon="Delete"
        circle
        type="danger"
        plain
        :disabled="disabled || !modelValue || uploading"
        @click="emit('update:modelValue', '')"
      />
    </el-tooltip>
  </div>
  <ImageCropDialog v-model="cropVisible" :file="pendingImage" @confirm="uploadSelectedImage" />
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { Delete, Picture, Upload } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { uploadImage } from '@/api/upload'
import ImageCropDialog from '@/components/ImageCropDialog.vue'

const props = withDefaults(defineProps<{
  modelValue: string
  disabled?: boolean
  uploadDisabled?: boolean
  maxFileSize?: number
}>(), {
  disabled: false,
  uploadDisabled: false,
  maxFileSize: 20 * 1024 * 1024,
})

const emit = defineEmits<{ (event: 'update:modelValue', value: string): void }>()
const fileInput = ref<HTMLInputElement>()
const uploading = ref(false)
const progress = ref(0)
const pendingImage = ref<File>()
const cropVisible = ref(false)

function selectFile() {
  if (!props.disabled && !props.uploadDisabled && !uploading.value) fileInput.value?.click()
}

async function handleFileChange(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  input.value = ''
  if (!file) return
  if (!['image/jpeg', 'image/png', 'image/webp', 'image/gif'].includes(file.type)) {
    ElMessage.warning('仅支持 JPG、PNG、WebP 或 GIF 图片')
    return
  }
  if (file.size > props.maxFileSize) {
    ElMessage.warning(`Logo 不能超过 ${Math.round(props.maxFileSize / 1024 / 1024)} MB`)
    return
  }

  pendingImage.value = file
  cropVisible.value = true
}

async function uploadSelectedImage(file: File) {
  pendingImage.value = undefined
  uploading.value = true
  progress.value = 0
  try {
    const url = await uploadImage(file, (value) => { progress.value = value })
    if (!url) throw new Error('上传完成后未返回文件地址')
    emit('update:modelValue', url)
    ElMessage.success(file.type === 'image/png' && file.name.includes('-crop-') ? 'Logo 已无损裁剪并上传' : 'Logo 原图已上传')
  } finally {
    uploading.value = false
  }
}
</script>

<style scoped>
.logo-uploader {
  display: grid;
  grid-template-columns: 72px minmax(180px, 1fr) 32px 32px;
  align-items: center;
  gap: 8px;
  width: 100%;
}
.logo-preview {
  display: grid;
  place-items: center;
  width: 72px;
  height: 44px;
  overflow: hidden;
  color: #909399;
  background: #f5f7fa;
  border: 1px solid #dcdfe6;
  border-radius: 4px;
}
.logo-preview .el-image {
  width: 100%;
  height: 100%;
}
.file-input {
  display: none;
}
@media (max-width: 720px) {
  .logo-uploader {
    grid-template-columns: 64px minmax(120px, 1fr) 32px 32px;
  }
}
</style>
