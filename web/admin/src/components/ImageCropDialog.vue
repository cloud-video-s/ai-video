<template>
  <el-dialog
    :model-value="modelValue"
    title="裁剪图片"
    width="820px"
    append-to-body
    destroy-on-close
    :close-on-click-modal="false"
    @update:model-value="emit('update:modelValue', $event)"
    @closed="reset"
  >
    <div class="crop-editor">
      <div
        ref="stageRef"
        class="crop-stage"
        @pointermove="handlePointerMove"
        @pointerup="endPointer"
        @pointercancel="endPointer"
      >
        <img
          v-if="sourceURL"
          ref="imageRef"
          :src="sourceURL"
          class="source-image"
          draggable="false"
          alt="待裁剪图片"
          @load="initialize"
        />
        <div
          v-if="ready"
          class="crop-box"
          :style="cropBoxStyle"
          @pointerdown="startMove"
        >
          <div class="crop-grid"><i /><i /><i /><i /></div>
          <button class="resize-handle" type="button" aria-label="调整裁剪区域大小" @pointerdown.stop="startResize" />
        </div>
      </div>

      <div class="crop-toolbar">
        <div class="ratio-group">
          <span>裁剪比例</span>
          <el-radio-group v-model="selectedRatio" size="small" @change="resetCropArea">
            <el-radio-button :value="0">自由</el-radio-button>
            <el-radio-button :value="1">1:1</el-radio-button>
            <el-radio-button :value="4 / 3">4:3</el-radio-button>
            <el-radio-button :value="16 / 9">16:9</el-radio-button>
            <el-radio-button :value="3 / 4">3:4</el-radio-button>
            <el-radio-button :value="9 / 16">9:16</el-radio-button>
          </el-radio-group>
        </div>
        <el-button @click="resetCropArea">最大化裁剪区域</el-button>
      </div>

      <div class="crop-meta">
        <span>原图 {{ naturalWidth }} × {{ naturalHeight }} px</span>
        <span>输出 {{ outputWidth }} × {{ outputHeight }} px</span>
        <span>输出 PNG · 原始像素 · 不缩放</span>
      </div>
      <el-alert
        v-if="file?.type === 'image/gif'"
        title="GIF 裁剪后会输出当前首帧的无损 PNG；如需保留动画，请选择“使用原图上传”。"
        type="warning"
        :closable="false"
        show-icon
      />
    </div>

    <template #footer>
      <el-button :disabled="processing" @click="useOriginal">使用原图上传</el-button>
      <el-button @click="emit('update:modelValue', false)">取消</el-button>
      <el-button type="primary" :loading="processing" :disabled="!ready" @click="confirmCrop">确认裁剪</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, reactive, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'

const props = withDefaults(defineProps<{
  modelValue: boolean
  file?: File
  defaultAspectRatio?: number
}>(), {
  file: undefined,
  defaultAspectRatio: 0,
})

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  confirm: [file: File]
}>()

type Interaction = 'move' | 'resize' | ''
type CropRect = { x: number; y: number; width: number; height: number }

const stageRef = ref<HTMLDivElement>()
const imageRef = ref<HTMLImageElement>()
const sourceURL = ref('')
const naturalWidth = ref(0)
const naturalHeight = ref(0)
const ready = ref(false)
const processing = ref(false)
const selectedRatio = ref(props.defaultAspectRatio)
const crop = reactive<CropRect>({ x: 0, y: 0, width: 0, height: 0 })
const imageBounds = reactive({ left: 0, top: 0, width: 0, height: 0, scale: 1 })
let interaction: Interaction = ''
let pointerID = -1
let startClientX = 0
let startClientY = 0
let initialCrop: CropRect = { x: 0, y: 0, width: 0, height: 0 }

const outputWidth = computed(() => Math.max(1, Math.round(crop.width)))
const outputHeight = computed(() => Math.max(1, Math.round(crop.height)))
const cropBoxStyle = computed(() => ({
  left: `${imageBounds.left + crop.x * imageBounds.scale}px`,
  top: `${imageBounds.top + crop.y * imageBounds.scale}px`,
  width: `${crop.width * imageBounds.scale}px`,
  height: `${crop.height * imageBounds.scale}px`,
}))

watch(() => props.modelValue, async (visible) => {
  if (!visible || !props.file) return
  resetSourceURL()
  selectedRatio.value = props.defaultAspectRatio
  sourceURL.value = URL.createObjectURL(props.file)
  await nextTick()
}, { immediate: true })

watch(() => props.file, async (file) => {
  if (!props.modelValue || !file) return
  resetSourceURL()
  sourceURL.value = URL.createObjectURL(file)
  await nextTick()
})

async function initialize() {
  await nextTick()
  const stage = stageRef.value
  const image = imageRef.value
  if (!stage || !image || !image.naturalWidth || !image.naturalHeight) return
  naturalWidth.value = image.naturalWidth
  naturalHeight.value = image.naturalHeight
  const scale = Math.min(stage.clientWidth / image.naturalWidth, stage.clientHeight / image.naturalHeight)
  imageBounds.scale = scale
  imageBounds.width = image.naturalWidth * scale
  imageBounds.height = image.naturalHeight * scale
  imageBounds.left = (stage.clientWidth - imageBounds.width) / 2
  imageBounds.top = (stage.clientHeight - imageBounds.height) / 2
  image.style.left = `${imageBounds.left}px`
  image.style.top = `${imageBounds.top}px`
  image.style.width = `${imageBounds.width}px`
  image.style.height = `${imageBounds.height}px`
  ready.value = true
  resetCropArea()
}

function resetCropArea() {
  if (!naturalWidth.value || !naturalHeight.value) return
  const ratio = Number(selectedRatio.value)
  let width = naturalWidth.value
  let height = naturalHeight.value
  if (ratio > 0) {
    if (width / height > ratio) width = height * ratio
    else height = width / ratio
  }
  crop.width = width
  crop.height = height
  crop.x = (naturalWidth.value - width) / 2
  crop.y = (naturalHeight.value - height) / 2
}

function beginInteraction(event: PointerEvent, mode: Interaction) {
  if (!ready.value) return
  interaction = mode
  pointerID = event.pointerId
  startClientX = event.clientX
  startClientY = event.clientY
  initialCrop = { ...crop }
  ;(event.currentTarget as HTMLElement).setPointerCapture(event.pointerId)
}

function startMove(event: PointerEvent) {
  beginInteraction(event, 'move')
}

function startResize(event: PointerEvent) {
  beginInteraction(event, 'resize')
}

function handlePointerMove(event: PointerEvent) {
  if (!interaction || event.pointerId !== pointerID) return
  const dx = (event.clientX - startClientX) / imageBounds.scale
  const dy = (event.clientY - startClientY) / imageBounds.scale
  if (interaction === 'move') {
    crop.x = clamp(initialCrop.x + dx, 0, naturalWidth.value - crop.width)
    crop.y = clamp(initialCrop.y + dy, 0, naturalHeight.value - crop.height)
    return
  }

  const maxWidth = naturalWidth.value - initialCrop.x
  const maxHeight = naturalHeight.value - initialCrop.y
  const minSourceSize = Math.max(1, 48 / imageBounds.scale)
  let width = clamp(initialCrop.width + dx, Math.min(minSourceSize, maxWidth), maxWidth)
  let height = clamp(initialCrop.height + dy, Math.min(minSourceSize, maxHeight), maxHeight)
  const ratio = Number(selectedRatio.value)
  if (ratio > 0) {
    if (Math.abs(dx) >= Math.abs(dy)) height = width / ratio
    else width = height * ratio
    if (width > maxWidth) {
      width = maxWidth
      height = width / ratio
    }
    if (height > maxHeight) {
      height = maxHeight
      width = height * ratio
    }
  }
  crop.width = width
  crop.height = height
}

function endPointer(event: PointerEvent) {
  if (event.pointerId !== pointerID) return
  interaction = ''
  pointerID = -1
}

function clamp(value: number, min: number, max: number) {
  return Math.min(max, Math.max(min, value))
}

function useOriginal() {
  if (!props.file) return
  emit('confirm', props.file)
  emit('update:modelValue', false)
}

async function confirmCrop() {
  const image = imageRef.value
  const source = props.file
  if (!image || !source || !ready.value) return
  processing.value = true
  try {
    const canvas = document.createElement('canvas')
    canvas.width = outputWidth.value
    canvas.height = outputHeight.value
    const context = canvas.getContext('2d', { alpha: true })
    if (!context) throw new Error('当前浏览器不支持图片裁剪')
    context.imageSmoothingEnabled = false
    context.drawImage(
      image,
      Math.round(crop.x), Math.round(crop.y), outputWidth.value, outputHeight.value,
      0, 0, outputWidth.value, outputHeight.value,
    )
    const blob = await new Promise<Blob | null>((resolve) => canvas.toBlob(resolve, 'image/png'))
    if (!blob) throw new Error('图片裁剪失败，请更换图片后重试')
    const baseName = source.name.replace(/\.[^.]+$/, '').replace(/[^A-Za-z0-9_-]+/g, '-') || 'image'
    const x = Math.round(crop.x)
    const y = Math.round(crop.y)
    const cropped = new File(
      [blob],
      `${baseName}-crop-${x}-${y}-${outputWidth.value}x${outputHeight.value}.png`,
      { type: 'image/png', lastModified: source.lastModified },
    )
    emit('confirm', cropped)
    emit('update:modelValue', false)
  } catch (error: any) {
    ElMessage.error(error?.message || '图片裁剪失败')
  } finally {
    processing.value = false
  }
}

function resetSourceURL() {
  if (sourceURL.value) URL.revokeObjectURL(sourceURL.value)
  sourceURL.value = ''
  ready.value = false
}

function reset() {
  resetSourceURL()
  naturalWidth.value = 0
  naturalHeight.value = 0
  interaction = ''
}

onBeforeUnmount(resetSourceURL)
</script>

<style scoped>
.crop-editor { min-width: 0; }
.crop-stage { position: relative; width: 100%; height: min(56vh, 520px); min-height: 320px; overflow: hidden; border-radius: 8px; background: #15171b; touch-action: none; user-select: none; }
.source-image { position: absolute; max-width: none; pointer-events: none; }
.crop-box { position: absolute; cursor: move; box-sizing: border-box; border: 2px solid #fff; box-shadow: 0 0 0 9999px rgb(0 0 0 / 58%), 0 0 0 1px rgb(0 0 0 / 40%); touch-action: none; }
.crop-grid { position: absolute; inset: 0; pointer-events: none; }
.crop-grid i { position: absolute; background: rgb(255 255 255 / 45%); }
.crop-grid i:nth-child(1), .crop-grid i:nth-child(2) { top: 0; bottom: 0; width: 1px; }
.crop-grid i:nth-child(1) { left: 33.333%; }
.crop-grid i:nth-child(2) { left: 66.666%; }
.crop-grid i:nth-child(3), .crop-grid i:nth-child(4) { left: 0; right: 0; height: 1px; }
.crop-grid i:nth-child(3) { top: 33.333%; }
.crop-grid i:nth-child(4) { top: 66.666%; }
.resize-handle { position: absolute; right: -7px; bottom: -7px; width: 16px; height: 16px; padding: 0; border: 2px solid #fff; border-radius: 50%; background: var(--el-color-primary); cursor: nwse-resize; }
.crop-toolbar { display: flex; align-items: center; justify-content: space-between; gap: 16px; margin-top: 16px; }
.ratio-group { display: flex; align-items: center; gap: 10px; }
.ratio-group > span { flex: 0 0 auto; color: #606266; font-size: 13px; }
.crop-meta { display: flex; flex-wrap: wrap; gap: 8px 18px; margin: 12px 0; color: #909399; font-size: 12px; }
@media (max-width: 680px) {
  .crop-stage { min-height: 260px; }
  .crop-toolbar, .ratio-group { align-items: flex-start; flex-direction: column; }
}
</style>
