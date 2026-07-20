<template>
  <div class="cover-uploader">
    <input ref="fileInput" class="file-input" type="file" accept="image/jpeg,image/png,image/webp,image/gif" @change="handleFileChange" />
    <el-button type="primary" plain :loading="uploading" @click="selectFile">
      {{ uploading ? `上传中 ${uploadProgress}%` : '选择图片并裁剪' }}
    </el-button>
    <span class="upload-tip">默认比例 {{ ratioLabel }}，按原图像素输出无损 PNG，不缩放</span>
  </div>

  <ImageCropDialog
    v-model="cropVisible"
    :file="pendingImage"
    :default-aspect-ratio="aspectRatio"
    @confirm="uploadSelectedImage"
  />
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { uploadImage } from '@/api/upload'
import ImageCropDialog from '@/components/ImageCropDialog.vue'

const props = withDefaults(defineProps<{
  modelValue: string
  targetWidth?: number
  targetHeight?: number
  maxFileSize?: number
}>(), {
  targetWidth: 1200,
  targetHeight: 675,
  maxFileSize: 20 * 1024 * 1024,
})

const emit = defineEmits<{ (event: 'update:modelValue', value: string): void }>()
const fileInput = ref<HTMLInputElement>()
const pendingImage = ref<File>()
const cropVisible = ref(false)
const uploading = ref(false)
const uploadProgress = ref(0)
const aspectRatio = computed(() => props.targetWidth > 0 && props.targetHeight > 0 ? props.targetWidth / props.targetHeight : 0)
const ratioLabel = computed(() => {
  const divisor = greatestCommonDivisor(props.targetWidth, props.targetHeight)
  return divisor > 0 ? `${props.targetWidth / divisor}:${props.targetHeight / divisor}` : '自由'
})

function selectFile() {
  if (!uploading.value) fileInput.value?.click()
}

function handleFileChange(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  input.value = ''
  if (!file) return
  if (!['image/jpeg', 'image/png', 'image/webp', 'image/gif'].includes(file.type)) {
    ElMessage.warning('仅支持 JPG、PNG、WebP 或 GIF 图片')
    return
  }
  if (file.size > props.maxFileSize) {
    ElMessage.warning(`图片不能超过 ${Math.round(props.maxFileSize / 1024 / 1024)} MB`)
    return
  }
  pendingImage.value = file
  cropVisible.value = true
}

async function uploadSelectedImage(file: File) {
  pendingImage.value = undefined
  uploading.value = true
  uploadProgress.value = 0
  try {
    const imageURL = await uploadImage(file, (percentage) => { uploadProgress.value = percentage })
    if (!imageURL) throw new Error('上传完成后未返回文件地址')
    emit('update:modelValue', imageURL)
    ElMessage.success(file.type === 'image/png' && file.name.includes('-crop-') ? '图片已无损裁剪并上传' : '原图已上传')
  } finally {
    uploading.value = false
  }
}

function greatestCommonDivisor(a: number, b: number): number {
  let x = Math.abs(Math.round(a))
  let y = Math.abs(Math.round(b))
  while (y) [x, y] = [y, x % y]
  return x
}
</script>

<style scoped>
.cover-uploader { display: flex; align-items: center; flex-wrap: wrap; gap: 10px; }
.file-input { display: none; }
.upload-tip { color: #909399; font-size: 12px; }
</style>
