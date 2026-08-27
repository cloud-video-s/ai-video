<template>
  <div class="page-wrap">
    <el-card shadow="never">
      <template #header>
        <div class="page-header">
          <div>
            <div class="page-title">工具配置</div>
            <div class="page-subtitle">维护工具的生成类型、关联模型、图片素材、可选配置与上下线状态</div>
          </div>
          <el-button v-if="canAdd" type="primary" @click="openCreate">
            <el-icon><Plus /></el-icon>
            新增工具
          </el-button>
        </div>
      </template>

      <div class="filters">
        <el-input v-model="query.keyword" clearable placeholder="搜索工具名称" @keyup.enter="handleSearch">
          <template #prefix><el-icon><Search /></el-icon></template>
        </el-input>
        <el-select v-model="query.status" clearable placeholder="上下线状态">
          <el-option label="在线" :value="1" />
          <el-option label="下线" :value="0" />
        </el-select>
        <el-button type="primary" plain @click="handleSearch">查询</el-button>
        <el-button @click="handleReset">重置</el-button>
      </div>

      <el-table v-loading="loading" :data="tableData" row-key="id" stripe @sort-change="handleSortChange">
        <el-table-column label="序号" width="76" align="center">
          <template #default="{ $index }">{{ (page - 1) * pageSize + $index + 1 }}</template>
        </el-table-column>
        <el-table-column prop="sort" label="排序" width="96" align="center" sortable="custom" />
        <el-table-column label="图标" width="112" align="center">
          <template #default="{ row }">
            <el-image
              class="tool-icon"
              :src="toMediaURL(row.icon)"
              :preview-src-list="[toMediaURL(row.icon)]"
              preview-teleported
              fit="contain"
            >
              <template #error><div class="image-error"><el-icon><Picture /></el-icon></div></template>
            </el-image>
          </template>
        </el-table-column>
        <el-table-column label="背景图" width="164" align="center">
          <template #default="{ row }">
            <el-image
              class="tool-background"
              :src="toMediaURL(row.background_image)"
              :preview-src-list="[toMediaURL(row.background_image)]"
              preview-teleported
              fit="cover"
            >
              <template #error><div class="image-error"><el-icon><Picture /></el-icon></div></template>
            </el-image>
          </template>
        </el-table-column>
        <el-table-column prop="name" label="工具名称" min-width="220">
          <template #default="{ row }">
            <div class="tool-name">{{ row.name }}</div>
            <div class="tool-meta">
              <el-tag size="small" effect="plain">{{ toolTypeLabel(row.tool_type) }}</el-tag>
              <el-tag size="small" type="info" effect="plain">{{ configTypeLabel(row.config_type) }}</el-tag>
              <el-tag size="small" type="info" effect="plain">{{ toolsTypeLabel(row.tools_type) }}</el-tag>
              <span>{{ row.model_name || `模型 #${row.model_id}` }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="status" label="状态" width="124" align="center" sortable="custom">
          <template #default="{ row }">
            <el-switch
              v-if="canStatus"
              v-model="row.status"
              :active-value="1"
              :inactive-value="0"
              active-text="在线"
              inactive-text="下线"
              inline-prompt
              :loading="statusChangingID === row.id"
              :before-change="() => handleStatusChange(row)"
            />
            <el-tag v-else :type="row.status === 1 ? 'success' : 'info'">
              {{ row.status === 1 ? '在线' : '下线' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="updated_at" label="更新时间" width="188" sortable="custom" />
        <el-table-column v-if="canEdit || canDelete" label="操作" width="138" fixed="right" align="center">
          <template #default="{ row }">
            <el-button v-if="canEdit" link type="primary" @click="openEdit(row)">编辑</el-button>
            <el-popconfirm
              v-if="canDelete"
              title="确认删除该工具配置？删除后列表不再展示。"
              width="260"
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
          @size-change="fetchData"
          @current-change="fetchData"
        />
      </div>
    </el-card>

    <el-dialog
      v-model="dialogVisible"
      :title="form.id ? '编辑工具配置' : '新增工具配置'"
      width="920px"
      destroy-on-close
    >
      <el-form ref="formRef" :model="form" :rules="rules" label-width="96px">
        <div class="form-grid">
          <el-form-item label="名称" prop="name">
            <el-input v-model="form.name" maxlength="128" show-word-limit placeholder="例如：Dance" />
          </el-form-item>
          <el-form-item label="排序" prop="sort">
            <el-input-number v-model="form.sort" :min="0" :max="999999" controls-position="right" />
            <span class="form-tip">数值越小越靠前</span>
          </el-form-item>
          <el-form-item label="类型" prop="tool_type">
            <el-select v-model="form.tool_type" style="width: 100%" @change="handleToolTypeChange">
              <el-option label="图片生成" :value="1" />
              <el-option label="视频生成" :value="2" />
            </el-select>
          </el-form-item>
          <el-form-item label="模型" prop="model_id">
            <el-select v-model="form.model_id" filterable style="width: 100%" :loading="modelsLoading" placeholder="请选择启用模型">
              <el-option v-for="item in filteredModelOptions" :key="item.id" :label="item.name" :value="item.id" />
            </el-select>
          </el-form-item>
          <el-form-item label="配置类型" prop="config_type">
            <el-select v-model="form.config_type" style="width: 100%" @change="handleConfigTypeChange">
              <el-option label="无" :value="1" />
              <el-option label="图片" :value="2" />
              <el-option label="年龄" :value="3" />
              <el-option label="比例" :value="4" />
            </el-select>
          </el-form-item>
          <el-form-item label="状态" prop="status">
            <el-radio-group v-model="form.status">
              <el-radio :value="1">在线</el-radio>
              <el-radio :value="0">下线</el-radio>
            </el-radio-group>
          </el-form-item>
        </div>

        <el-form-item v-if="form.config_type === 2" label="参考图片" required>
          <div class="config-editor">
            <div class="config-actions">
              <input
                ref="referenceImageInput"
                class="file-input"
                type="file"
                accept="image/jpeg,image/png,image/webp,image/gif"
                multiple
                @change="handleReferenceFiles"
              />
              <el-button type="primary" plain :loading="uploadingReferenceImages" @click="referenceImageInput?.click()">
                批量上传
              </el-button>
              <el-button @click="addReferenceImage">添加一项</el-button>
              <span class="config-tip">支持多张图片，每张可配置别名和排序</span>
            </div>
            <el-table :data="form.reference_images" border size="small">
              <el-table-column label="别名" width="150">
                <template #default="{ row }"><el-input v-model="row.name" maxlength="128" placeholder="请输入别名" /></template>
              </el-table-column>
              <el-table-column label="图片" min-width="430">
                <template #default="{ row }">
                  <logo-image-uploader v-model="row.image" image-name="参考图片" placeholder="图片 URL 或选择上传" />
                </template>
              </el-table-column>
              <el-table-column label="排序" width="110">
                <template #default="{ row }"><el-input-number v-model="row.sort" :min="0" :max="999999" controls-position="right" /></template>
              </el-table-column>
              <el-table-column label="操作" width="72" align="center">
                <template #default="{ $index }"><el-button link type="danger" @click="removeReferenceImage($index)">删除</el-button></template>
              </el-table-column>
            </el-table>
          </div>
        </el-form-item>

        <el-form-item v-if="form.config_type === 3" label="年龄区间" required>
          <div class="config-actions">
            <el-button type="primary" plain @click="addRatioOption">添加比例</el-button>
            <span class="config-tip">年龄参数需要添加到提示词中:例如 18岁 要写成： {age}岁 生成任务时将此标识替换成对应的年龄</span>
          </div>
          <div class="age-range">
            <span>最小值</span>
            <el-input-number v-model="form.age_min" :min="0" :max="200" controls-position="right" />
            <span>最大值</span>
            <el-input-number v-model="form.age_max" :min="0" :max="200" controls-position="right" />
          </div>
        </el-form-item>

        <el-form-item v-if="form.config_type === 4" label="比例选项" required>
          <div class="config-editor">
            <div class="config-actions">
              <el-button type="primary" plain @click="addRatioOption">添加比例</el-button>
              <span class="config-tip">比例参数按字符串保存，例如 4:3、1:1、9:16 (下方提示词中添加 {scale} 生成时将此标识替换成对应的比例)</span>
            </div>
            <el-table :data="form.ratio_options" border size="small">
              <el-table-column label="别名" min-width="180">
                <template #default="{ row }"><el-input v-model="row.name" maxlength="128" placeholder="例如：竖屏" /></template>
              </el-table-column>
              <el-table-column label="比例参数" min-width="180">
                <template #default="{ row }"><el-input v-model="row.value" maxlength="64" placeholder="例如：9:16" /></template>
              </el-table-column>
              <el-table-column label="排序" width="140">
                <template #default="{ row }"><el-input-number v-model="row.sort" :min="0" :max="999999" controls-position="right" /></template>
              </el-table-column>
              <el-table-column label="操作" width="72" align="center">
                <template #default="{ $index }"><el-button link type="danger" @click="removeRatioOption($index)">删除</el-button></template>
              </el-table-column>
            </el-table>
          </div>
        </el-form-item>
        <el-form-item label="所属功能" prop="tools_type">
          <el-select v-model="form.tools_type" placeholder="请选择所属功能" style="width: 100%">
            <el-option v-for="item in toolsTypeOptions" :key="item.value" :label="item.label" :value="item.value" />
          </el-select>
        </el-form-item>
        <el-form-item label="图标" prop="icon">
          <logo-image-uploader
            v-model="form.icon"
            image-name="工具图标"
            placeholder="请输入图标 URL 或选择图片上传"
            :crop-aspect-ratio="1"
          />
        </el-form-item>
        <el-form-item label="背景图" prop="background_image">
          <div class="background-field">
            <el-input
              v-model="form.background_image"
              maxlength="1024"
              clearable
              placeholder="请输入背景图 URL 或选择图片上传"
            />
            <cover-image-uploader v-model="form.background_image" class="background-upload" />
            <div v-if="form.background_image" class="background-form-preview">
              <el-image
                class="background-preview-image"
                :src="toMediaURL(form.background_image)"
                :preview-src-list="[toMediaURL(form.background_image)]"
                preview-teleported
                fit="cover"
              >
                <template #error>
                  <div class="background-preview-error">
                    <el-icon><Picture /></el-icon>
                    <span>背景图加载失败</span>
                  </div>
                </template>
              </el-image>
              <div class="background-preview-meta">
                <span>背景图预览</span>
                <span>点击图片查看大图</span>
              </div>
            </div>
          </div>
        </el-form-item>

        <el-form-item label="角标">
          <logo-image-uploader
            v-model="form.badge_image"
            image-name="工具角标"
            placeholder="可选，输入角标 URL 或选择上传"
            :crop-aspect-ratio="1"
          />
        </el-form-item>
        <el-form-item label="提示词" prop="prompt">
          <el-input v-model="form.prompt" type="textarea" :rows="5" maxlength="10000" show-word-limit placeholder="请输入工具提示词" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="handleSubmit">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, type FormInstance, type FormRules } from 'element-plus'
import {
  createToolConfig,
  deleteToolConfig,
  getToolConfigList,
  getToolModelOptions,
  updateToolConfig,
  updateToolConfigStatus,
  type ToolConfig,
  type ToolConfigData,
  type ToolConfigPayload,
  type ToolModelOption,
  type ToolRatioOption,
  type ToolReferenceImageOption,
  type ToolsType,
} from '@/api/toolConfig'
import { uploadImage } from '@/api/upload'
import LogoImageUploader from '@/components/LogoImageUploader.vue'
import CoverImageUploader from '@/components/CoverImageUploader.vue'
import { useUserStore } from '@/store/user'
import { toMediaURL } from '@/utils/mediaUrl'
import { useRemoteTableSort } from '@/utils/tableSort'

const userStore = useUserStore()
const canAdd = computed(() => userStore.hasPermission('tool:config:add'))
const canEdit = computed(() => userStore.hasPermission('tool:config:edit'))
const canStatus = computed(() => userStore.hasPermission('tool:config:status'))
const canDelete = computed(() => userStore.hasPermission('tool:config:delete'))

const loading = ref(false)
const submitting = ref(false)
const dialogVisible = ref(false)
const statusChangingID = ref(0)
const modelsLoading = ref(false)
const uploadingReferenceImages = ref(false)
const formRef = ref<FormInstance>()
const referenceImageInput = ref<HTMLInputElement>()
const tableData = ref<ToolConfig[]>([])
const modelOptions = ref<ToolModelOption[]>([])
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const query = reactive<{ keyword: string; status: '' | number }>({ keyword: '', status: '' })
const { sortParams, handleSortChange } = useRemoteTableSort(page, fetchData)

interface ToolForm {
  id: number
  name: string
  icon: string
  background_image: string
  tool_type: 1 | 2
  tools_type: ToolsType | ''
  model_id: number
  config_type: 1 | 2 | 3 | 4
  reference_images: ToolReferenceImageOption[]
  ratio_options: ToolRatioOption[]
  age_min: number
  age_max: number
  badge_image: string
  sort: number
  prompt: string
  status: number
}

function createDefaultForm(): ToolForm {
  return {
    id: 0,
    name: '',
    icon: '',
    background_image: '',
    tool_type: 1,
    model_id: 0,
    config_type: 1,
    tools_type: '',
    reference_images: [],
    ratio_options: [],
    age_min: 18,
    age_max: 50,
    badge_image: '',
    sort: 0,
    prompt: '',
    status: 1,
  }
}

const toolsTypeOptions: Array<{ value: ToolsType; label: string }> = [
  { value: 'enhance', label: '图片变清晰' },
  { value: 'outpaint', label: '扩图' },
  { value: 'hairstyle', label: '换发型' },
  { value: 'age_transform', label: '年龄变换' },
  { value: 'body_reshape', label: '完美身材' },
  { value: 'colorful', label: '老照片上色' },
  { value: 'makeup', label: '换妆' },
  { value: 'outfit', label: '换装' },
  { value: 'pose_transfer', label: '动作模仿' },
]

const form = reactive<ToolForm>(createDefaultForm())
const filteredModelOptions = computed(() => modelOptions.value.filter((item) => item.model_type === form.tool_type))
const rules: FormRules = {
  name: [{ required: true, message: '请输入工具名称', trigger: 'blur' }],
  icon: [{ required: true, message: '请输入或上传工具图标', trigger: 'change' }],
  background_image: [{ required: true, message: '请输入或上传工具背景图', trigger: 'change' }],
  tool_type: [{ required: true, message: '请选择工具类型', trigger: 'change' }],
  tools_type: [{ required: true, message: '请选择所属功能', trigger: 'change' }],
  model_id: [{ required: true, message: '请选择启用模型', trigger: 'change' }],
}

async function fetchData() {
  loading.value = true
  try {
    const params: Record<string, unknown> = {
      page: page.value,
      page_size: pageSize.value,
      ...sortParams(),
    }
    if (query.keyword.trim()) params.keyword = query.keyword.trim()
    if (query.status !== '') params.status = query.status
    const res: any = await getToolConfigList(params)
    tableData.value = res.data.list || []
    total.value = res.data.total || 0
  } finally {
    loading.value = false
  }
}

async function fetchModelOptions(toolType: 1 | 2 = form.tool_type) {
  modelsLoading.value = true
  try {
    const res: any = await getToolModelOptions(toolType)
    modelOptions.value = res.data || []
  } finally {
    modelsLoading.value = false
  }
}

function handleSearch() {
  page.value = 1
  fetchData()
}

function handleReset() {
  Object.assign(query, { keyword: '', status: '' })
  page.value = 1
  fetchData()
}

function openCreate() {
  Object.assign(form, createDefaultForm())
  fetchModelOptions(form.tool_type)
  dialogVisible.value = true
}

function openEdit(row: ToolConfig) {
  const data = row.config_data || {}
  Object.assign(form, {
    id: row.id,
    name: row.name,
    icon: row.icon,
    background_image: row.background_image,
    tool_type: row.tool_type,
    tools_type: row.tools_type,
    model_id: row.model_id,
    config_type: row.config_type,
    reference_images: (data.reference_images || []).map((item) => ({ ...item })),
    ratio_options: (data.ratio_options || []).map((item) => ({ ...item })),
    age_min: data.age_range?.min ?? 18,
    age_max: data.age_range?.max ?? 50,
    badge_image: row.badge_image || '',
    sort: row.sort,
    prompt: row.prompt || '',
    status: row.status,
  })
  fetchModelOptions(row.tool_type)
  dialogVisible.value = true
}

async function handleSubmit() {
  await formRef.value?.validate()
  const configData = buildConfigData()
  if (!configData) return
  submitting.value = true
  try {
    const payload: ToolConfigPayload = {
      name: form.name.trim(),
      icon: form.icon.trim(),
      background_image: form.background_image.trim(),
      tool_type: form.tool_type,
      tools_type: form.tools_type as ToolsType,
      model_id: form.model_id,
      config_type: form.config_type,
      config_data: configData,
      badge_image: form.badge_image.trim(),
      sort: form.sort,
      prompt: form.prompt.trim(),
      status: form.status,
    }
    if (form.id) await updateToolConfig(form.id, payload)
    else await createToolConfig(payload)
    ElMessage.success('工具配置已保存')
    dialogVisible.value = false
    await fetchData()
  } finally {
    submitting.value = false
  }
}

async function handleToolTypeChange() {
  form.model_id = 0
  await fetchModelOptions(form.tool_type)
}

function handleConfigTypeChange() {
  if (form.config_type === 2 && form.reference_images.length === 0) addReferenceImage()
  if (form.config_type === 4 && form.ratio_options.length === 0) {
    form.ratio_options.push(
      { name: '9:16', value: '9:16', sort: 1 },
    )
  }
}

function addReferenceImage() {
  form.reference_images.push({ name: '', image: '', sort: form.reference_images.length + 1 })
}

function removeReferenceImage(index: number) {
  form.reference_images.splice(index, 1)
}

function addRatioOption() {
  form.ratio_options.push({ name: '', value: '', sort: form.ratio_options.length + 1 })
}

function removeRatioOption(index: number) {
  form.ratio_options.splice(index, 1)
}

async function handleReferenceFiles(event: Event) {
  const input = event.target as HTMLInputElement
  const files = Array.from(input.files || [])
  input.value = ''
  if (files.length === 0) return
  uploadingReferenceImages.value = true
  try {
    for (const file of files) {
      if (!['image/jpeg', 'image/png', 'image/webp', 'image/gif'].includes(file.type)) {
        ElMessage.warning(`${file.name} 不是支持的图片格式`)
        continue
      }
      if (file.size > 20 * 1024 * 1024) {
        ElMessage.warning(`${file.name} 不能超过 20 MB`)
        continue
      }
      const image = await uploadImage(file)
      const name = file.name.replace(/\.[^.]+$/, '')
      form.reference_images.push({ name, image, sort: form.reference_images.length + 1 })
    }
  } finally {
    uploadingReferenceImages.value = false
  }
}

function buildConfigData(): ToolConfigData | null {
  if (form.config_type === 1) return {}
  if (form.config_type === 2) {
    const items = form.reference_images.map((item) => ({
      name: item.name.trim(), image: item.image.trim(), sort: item.sort,
    }))
    if (items.length === 0 || items.some((item) => !item.name || !item.image)) {
      ElMessage.warning('请至少配置一张参考图片，并填写每张图片的别名')
      return null
    }
    return { reference_images: items }
  }
  if (form.config_type === 3) {
    if (form.age_min < 0 || form.age_max > 200 || form.age_min > form.age_max) {
      ElMessage.warning('年龄区间必须满足 0 ≤ 最小值 ≤ 最大值 ≤ 200')
      return null
    }
    return { age_range: { min: form.age_min, max: form.age_max } }
  }
  const items = form.ratio_options.map((item) => ({
    name: item.name.trim(), value: item.value.trim(), sort: item.sort,
  }))
  if (items.length === 0 || items.some((item) => !item.name || !item.value)) {
    ElMessage.warning('请至少配置一个比例，并填写别名和比例参数')
    return null
  }
  return { ratio_options: items }
}

function toolTypeLabel(value: number) {
  return value === 2 ? '视频生成' : '图片生成'
}

function configTypeLabel(value: number) {
  return ({ 1: '无配置', 2: '图片', 3: '年龄', 4: '比例' } as Record<number, string>)[value] || '未知配置'
}

function toolsTypeLabel(value: string) {
  return toolsTypeOptions.find((item) => item.value === value)?.label || '未知功能'
}

async function handleStatusChange(row: ToolConfig) {
  const nextStatus = row.status === 1 ? 0 : 1
  statusChangingID.value = row.id
  try {
    await updateToolConfigStatus(row.id, nextStatus)
    ElMessage.success(nextStatus === 1 ? '工具已上线' : '工具已下线')
    return true
  } catch {
    return false
  } finally {
    statusChangingID.value = 0
  }
}

async function handleDelete(id: number) {
  await deleteToolConfig(id)
  ElMessage.success('工具配置已删除')
  if (tableData.value.length === 1 && page.value > 1) page.value--
  await fetchData()
}

onMounted(() => Promise.all([fetchData(), fetchModelOptions(1)]))
</script>

<style scoped>
.page-wrap { min-width: 0; }
.page-header { display: flex; align-items: center; justify-content: space-between; gap: 16px; }
.page-title { color: #303133; font-size: 17px; font-weight: 600; }
.page-subtitle { margin-top: 4px; color: #909399; font-size: 12px; }
.filters { display: grid; grid-template-columns: minmax(240px, 1fr) 180px auto auto; gap: 10px; max-width: 760px; margin-bottom: 16px; }
.tool-icon { width: 56px; height: 56px; padding: 6px; box-sizing: border-box; border: 1px solid #ebeef5; border-radius: 12px; background: #f7f9fc; }
.tool-background { width: 128px; height: 72px; border: 1px solid #ebeef5; border-radius: 8px; background: #f7f9fc; }
.image-error { display: grid; width: 100%; height: 100%; place-items: center; color: #c0c4cc; font-size: 22px; }
.tool-name { color: #303133; font-size: 14px; font-weight: 600; }
.tool-meta { display: flex; align-items: center; flex-wrap: wrap; gap: 6px; margin-top: 7px; color: #909399; font-size: 12px; }
.pagination-wrap { display: flex; justify-content: flex-end; margin-top: 16px; overflow-x: auto; }
.form-grid { display: grid; grid-template-columns: 1fr 1fr; column-gap: 18px; }
.page-wrap :deep(.el-input-number) { width: 180px; }
.form-tip { margin-left: 10px; color: #909399; font-size: 12px; }
.background-field { width: 100%; }
.background-upload { margin-top: 10px; }
.background-form-preview { display: flex; align-items: center; gap: 12px; margin-top: 10px; padding: 10px; border: 1px solid #ebeef5; border-radius: 8px; background: #fafafa; }
.background-preview-image { width: 240px; height: 135px; flex: 0 0 auto; border-radius: 6px; background: #f0f2f5; cursor: zoom-in; }
.background-preview-error { display: flex; align-items: center; justify-content: center; flex-direction: column; gap: 5px; width: 100%; height: 100%; color: #a8abb2; font-size: 12px; }
.background-preview-error .el-icon { font-size: 24px; }
.background-preview-meta { display: flex; flex-direction: column; gap: 4px; color: #606266; font-size: 13px; }
.background-preview-meta span:last-child { color: #a8abb2; font-size: 12px; }
.config-editor { width: 100%; min-width: 0; }
.config-actions { display: flex; align-items: center; flex-wrap: wrap; gap: 8px; margin-bottom: 10px; }
.config-tip { color: #909399; font-size: 12px; }
.config-editor :deep(.el-input-number) { width: 100%; }
.age-range { display: flex; align-items: center; flex-wrap: wrap; gap: 10px; }
.age-range span { color: #606266; font-size: 13px; }
.file-input { display: none; }
@media (max-width: 700px) {
  .page-header { align-items: stretch; flex-direction: column; }
  .filters, .form-grid { grid-template-columns: 1fr; max-width: none; }
  .background-form-preview { align-items: flex-start; flex-direction: column; }
  .background-preview-image { width: 100%; height: auto; aspect-ratio: 16 / 9; }
  .page-wrap :deep(.el-card__header), .page-wrap :deep(.el-card__body) { padding: 14px; }
  .page-wrap :deep(.el-dialog) { width: calc(100% - 24px) !important; }
}
</style>
