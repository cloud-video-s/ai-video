<template>
  <el-drawer v-model="visible" size="min(960px, 92vw)" destroy-on-close>
    <template #header>
      <div>
        <div class="drawer-title">模型配置</div>
        <div class="drawer-subtitle">{{ model?.name }} · {{ model?.code }} · {{ model?.version }}</div>
      </div>
    </template>

    <div class="drawer-toolbar">
      <el-alert type="info" :closable="false" show-icon>
        <template #title>选项参数必须配置多个候选值中的一个默认值；请求参数使用 JSON 对象描述限制条件。</template>
      </el-alert>
      <el-button type="primary" @click="openCreate"><el-icon><Plus /></el-icon>新增配置</el-button>
    </div>

    <el-table v-loading="loading" :data="items" row-key="id" stripe>
      <el-table-column prop="sort_order" label="排序" width="70" align="center" />
      <el-table-column label="字段" min-width="160">
        <template #default="{ row }">
          <code class="param-key">{{ row.param_key }}</code>
          <div class="alias">{{ row.alias }}</div>
          <div class="description">{{ row.description || '暂无说明' }}</div>
        </template>
      </el-table-column>
      <el-table-column label="配置类型" width="115" align="center">
        <template #default="{ row }">
          <el-tag :type="row.parameter_type === 1 ? 'success' : 'warning'" effect="plain">
            {{ row.parameter_type === 1 ? '选项参数' : '请求参数' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="值类型" width="105" align="center">
        <template #default="{ row }"><el-tag type="info">{{ row.value_type }}</el-tag></template>
      </el-table-column>
      <el-table-column label="展示类型" width="105" align="center">
        <template #default="{ row }"><el-tag effect="plain">{{ row.display_type }}</el-tag></template>
      </el-table-column>
      <el-table-column label="配置内容" min-width="290">
        <template #default="{ row }">
          <template v-if="row.parameter_type === 1">
            <div class="value-tags">
              <el-tag
                v-for="(value, index) in row.allowed_values"
                :key="`${row.id}-${index}`"
                :type="sameValue(value, row.default_value) ? 'primary' : 'info'"
                :effect="sameValue(value, row.default_value) ? 'dark' : 'plain'"
              >
                {{ displayValue(value) }}<span v-if="sameValue(value, row.default_value)">（默认）</span>
              </el-tag>
            </div>
          </template>
          <template v-else>
            <div><span class="muted">限制：</span><code>{{ displayJSON(row.constraints) }}</code></div>
            <div class="default-line"><span class="muted">默认：</span>{{ row.default_value === null ? '无' : displayValue(row.default_value) }}</div>
          </template>
        </template>
      </el-table-column>
      <el-table-column label="必填" width="70" align="center">
        <template #default="{ row }">{{ row.parameter_type === 2 && row.is_required === 1 ? '是' : '否' }}</template>
      </el-table-column>
      <el-table-column label="操作" width="120" fixed="right" align="center">
        <template #default="{ row }">
          <el-button link type="primary" @click="openEdit(row)">编辑</el-button>
          <el-popconfirm :title="`确认软删除配置 ${row.param_key}？`" width="250" @confirm="handleDelete(row.id)">
            <template #reference><el-button link type="danger">删除</el-button></template>
          </el-popconfirm>
        </template>
      </el-table-column>
    </el-table>

    <el-empty v-if="!loading && items.length === 0" description="该模型尚未配置参数" :image-size="90" />

    <el-dialog
      v-model="dialogVisible"
      :title="form.id ? '编辑模型配置' : '新增模型配置'"
      width="720px"
      append-to-body
      destroy-on-close
    >
      <el-form label-width="105px">
        <el-form-item label="配置类型" required>
          <el-radio-group v-model="form.parameter_type" @change="handleParameterTypeChange">
            <el-radio-button :value="1">选项参数</el-radio-button>
            <el-radio-button :value="2">请求参数</el-radio-button>
          </el-radio-group>
        </el-form-item>
        <div class="form-grid">
          <el-form-item label="参数字段" required>
            <el-input v-model="form.param_key" maxlength="64" placeholder="例如：aspect_ratio" />
          </el-form-item>
          <el-form-item label="值类型" required>
            <el-select v-model="form.value_type" style="width: 100%" @change="handleValueTypeChange">
              <el-option label="string（字符串）" value="string" />
              <el-option label="integer（整数）" value="integer" />
              <el-option label="number（数字）" value="number" />
              <el-option label="boolean（布尔值）" value="boolean" />
              <el-option v-if="form.parameter_type === 2" label="object（对象）" value="object" />
              <el-option v-if="form.parameter_type === 2" label="array（数组）" value="array" />
            </el-select>
          </el-form-item>
        </div>
        <div class="form-grid">
          <el-form-item label="别名" required>
            <el-input v-model="form.alias" maxlength="255" show-word-limit placeholder="例如：画面比例" />
          </el-form-item>
          <el-form-item label="展示类型" required>
            <el-select v-model="form.display_type" style="width: 100%">
              <el-option label="string（字符串）" value="string" />
              <el-option label="integer（整数）" value="integer" />
              <el-option label="boolean（布尔值）" value="boolean" />
              <el-option label="object（对象）" value="object" />
              <el-option label="array（数组）" value="array" />
              <el-option label="select（选择器）" value="select" />
              <el-option label="time（时间）" value="time" />
            </el-select>
          </el-form-item>
        </div>

        <template v-if="form.parameter_type === 1">
          <el-form-item label="选择值" required>
            <el-select
              v-model="form.option_values"
              multiple
              filterable
              allow-create
              default-first-option
              :reserve-keyword="false"
              style="width: 100%"
              placeholder="输入后回车，可添加多个值"
            >
              <el-option v-for="value in form.option_values" :key="value" :label="value" :value="value" />
            </el-select>
            <div class="form-help">数值类型请输入数字；boolean 仅支持 true 或 false。</div>
          </el-form-item>
          <el-form-item label="默认选择" required>
            <el-select v-model="form.option_default" style="width: 100%" placeholder="必须从选择值中指定一个">
              <el-option v-for="value in form.option_values" :key="value" :label="value" :value="value" />
            </el-select>
          </el-form-item>
        </template>

        <template v-else>
          <el-form-item label="是否必填">
            <el-switch v-model="form.is_required" :active-value="1" :inactive-value="0" />
          </el-form-item>
          <el-form-item label="限制条件" required>
            <el-input
              v-model="form.constraints_text"
              type="textarea"
              :rows="4"
              placeholder='JSON 对象，例如：{"min":1,"max":10} 或 {"max_length":500,"pattern":"..."}'
            />
            <div class="form-help">支持 min、max、min_length、max_length、pattern，也可保存平台自定义限制字段。</div>
          </el-form-item>
          <el-form-item label="默认值">
            <div class="default-value-row">
              <el-switch v-model="form.has_default" active-text="配置默认值" />
              <el-input
                v-if="form.has_default"
                v-model="form.default_input"
                :type="form.value_type === 'object' || form.value_type === 'array' ? 'textarea' : 'text'"
                :rows="3"
                :placeholder="defaultPlaceholder"
              />
            </div>
          </el-form-item>
        </template>

        <div class="form-grid">
          <el-form-item label="排序">
            <el-input-number v-model="form.sort_order" :min="0" :max="999999" controls-position="right" />
          </el-form-item>
        </div>
        <el-form-item label="参数说明">
          <el-input v-model="form.description" type="textarea" :rows="2" maxlength="255" show-word-limit placeholder="可选" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="handleSubmit">保存</el-button>
      </template>
    </el-dialog>
  </el-drawer>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import {
  createModelParameter,
  deleteModelParameter,
  getModelParameters,
  updateModelParameter,
  type ModelParameter,
  type ModelParameterPayload,
  type VideoModel,
} from '@/api/videoModel'

type ValueType = ModelParameter['value_type']
type DisplayType = ModelParameter['display_type']

const props = defineProps<{ modelValue: boolean; model: VideoModel | null }>()
const emit = defineEmits<{ (event: 'update:modelValue', value: boolean): void }>()

const visible = computed({
  get: () => props.modelValue,
  set: (value: boolean) => emit('update:modelValue', value),
})
const loading = ref(false)
const submitting = ref(false)
const dialogVisible = ref(false)
const items = ref<ModelParameter[]>([])

interface ParameterForm {
  id: number
  param_key: string
  value_type: ValueType
  alias: string
  display_type: DisplayType
  parameter_type: 1 | 2
  is_required: number
  option_values: string[]
  option_default: string
  has_default: boolean
  default_input: string
  constraints_text: string
  description: string
  sort_order: number
}

const defaultForm: ParameterForm = {
  id: 0,
  param_key: '',
  value_type: 'string',
  alias: '',
  display_type: 'string',
  parameter_type: 1,
  is_required: 0,
  option_values: [],
  option_default: '',
  has_default: false,
  default_input: '',
  constraints_text: '{\n  "max_length": 1000\n}',
  description: '',
  sort_order: 0,
}
const form = reactive<ParameterForm>({ ...defaultForm, option_values: [] })

const defaultPlaceholder = computed(() => {
  if (form.value_type === 'object') return '{"key":"value"}'
  if (form.value_type === 'array') return '["value1","value2"]'
  if (form.value_type === 'boolean') return 'true 或 false'
  return '请输入默认值'
})

async function fetchParameters() {
  if (!props.model?.id) return
  loading.value = true
  try {
    const res: any = await getModelParameters(props.model.id)
    items.value = res.data || []
  } finally {
    loading.value = false
  }
}

watch(visible, (open) => {
  if (open) fetchParameters()
})
watch(() => props.model?.id, () => {
  if (visible.value) fetchParameters()
})

function resetForm() {
  Object.assign(form, { ...defaultForm, option_values: [] })
}

function openCreate() {
  resetForm()
  dialogVisible.value = true
}

function openEdit(row: ModelParameter) {
  const optionValues = (row.allowed_values || []).map(displayValue)
  Object.assign(form, {
    id: row.id,
    param_key: row.param_key,
    value_type: row.value_type,
    alias: row.alias,
    display_type: row.display_type,
    parameter_type: row.parameter_type,
    is_required: row.is_required,
    option_values: optionValues,
    option_default: row.default_value === null ? '' : displayValue(row.default_value),
    has_default: row.default_value !== null,
    default_input: row.default_value === null ? '' : inputValue(row.default_value, row.value_type),
    constraints_text: JSON.stringify(row.constraints || {}, null, 2),
    description: row.description || '',
    sort_order: row.sort_order,
  })
  dialogVisible.value = true
}

function handleParameterTypeChange() {
  if (form.parameter_type === 1 && (form.value_type === 'object' || form.value_type === 'array')) {
    form.value_type = 'string'
  }
}

function handleValueTypeChange() {
  form.option_values = []
  form.option_default = ''
  form.default_input = ''
  form.has_default = false
}

async function handleSubmit() {
  if (!props.model?.id) return
  if (!/^[A-Za-z_][A-Za-z0-9_.-]{0,63}$/.test(form.param_key.trim())) {
    ElMessage.error('参数字段格式不正确')
    return
  }
  if (!form.alias.trim()) {
    ElMessage.error('请填写别名')
    return
  }

  let defaultValue: unknown = null
  let allowedValues: unknown[] = []
  let constraints: Record<string, unknown> = {}
  try {
    if (form.parameter_type === 1) {
      if (form.option_values.length === 0) throw new Error('请至少添加一个选择值')
      if (!form.option_values.includes(form.option_default)) throw new Error('请从选择值中指定一个默认选择')
      allowedValues = form.option_values.map((value) => parseTypedValue(form.value_type, value))
      defaultValue = parseTypedValue(form.value_type, form.option_default)
      const unique = new Set(allowedValues.map((value) => JSON.stringify(value)))
      if (unique.size !== allowedValues.length) throw new Error('转换后的选择值不能重复')
    } else {
      constraints = JSON.parse(form.constraints_text)
      if (!constraints || Array.isArray(constraints) || typeof constraints !== 'object' || Object.keys(constraints).length === 0) {
        throw new Error('限制条件必须是非空 JSON 对象')
      }
      defaultValue = form.has_default ? parseTypedValue(form.value_type, form.default_input) : null
    }
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '配置内容格式错误')
    return
  }

  const payload: ModelParameterPayload = {
    param_key: form.param_key.trim(),
    value_type: form.value_type,
    alias: form.alias.trim(),
    display_type: form.display_type,
    parameter_type: form.parameter_type,
    is_required: form.parameter_type === 2 ? form.is_required : 0,
    default_value: defaultValue,
    allowed_values: allowedValues,
    constraints,
    description: form.description.trim(),
    sort_order: form.sort_order,
  }
  submitting.value = true
  try {
    if (form.id) await updateModelParameter(props.model.id, form.id, payload)
    else await createModelParameter(props.model.id, payload)
    ElMessage.success('模型配置已保存')
    dialogVisible.value = false
    await fetchParameters()
  } finally {
    submitting.value = false
  }
}

async function handleDelete(id: number) {
  if (!props.model?.id) return
  await deleteModelParameter(props.model.id, id)
  ElMessage.success('模型配置已软删除')
  await fetchParameters()
}

function parseTypedValue(type: ValueType, raw: string): unknown {
  if (type === 'string') return raw
  if (type === 'integer') {
    const value = Number(raw)
    if (raw.trim() === '' || !Number.isInteger(value)) throw new Error(`“${raw}”不是有效整数`)
    return value
  }
  if (type === 'number') {
    const value = Number(raw)
    if (raw.trim() === '' || !Number.isFinite(value)) throw new Error(`“${raw}”不是有效数字`)
    return value
  }
  if (type === 'boolean') {
    if (raw === 'true') return true
    if (raw === 'false') return false
    throw new Error('布尔值只能填写 true 或 false')
  }
  const value = JSON.parse(raw)
  if (type === 'object' && (!value || Array.isArray(value) || typeof value !== 'object')) throw new Error('默认值必须是 JSON 对象')
  if (type === 'array' && !Array.isArray(value)) throw new Error('默认值必须是 JSON 数组')
  return value
}

function displayValue(value: unknown): string {
  if (typeof value === 'string') return value
  return JSON.stringify(value)
}

function inputValue(value: unknown, type: ValueType): string {
  return type === 'string' ? String(value) : JSON.stringify(value)
}

function displayJSON(value: unknown): string {
  return JSON.stringify(value || {})
}

function sameValue(left: unknown, right: unknown): boolean {
  return JSON.stringify(left) === JSON.stringify(right)
}
</script>

<style scoped>
.drawer-title { color: #303133; font-size: 17px; font-weight: 600; }
.drawer-subtitle { margin-top: 4px; color: #909399; font-size: 12px; }
.drawer-toolbar { display: grid; grid-template-columns: 1fr auto; align-items: center; gap: 14px; margin-bottom: 16px; }
.param-key { color: #409eff; font-weight: 600; }
.alias { margin-top: 4px; color: #606266; }
.description { margin-top: 5px; color: #909399; font-size: 12px; }
.value-tags { display: flex; flex-wrap: wrap; gap: 6px; }
.muted { color: #909399; }
.default-line { margin-top: 7px; }
.form-grid { display: grid; grid-template-columns: 1fr 1fr; column-gap: 16px; }
.form-help { margin-top: 5px; color: #909399; font-size: 12px; line-height: 18px; }
.default-value-row { display: flex; flex-direction: column; align-items: flex-start; gap: 10px; width: 100%; }
.default-value-row :deep(.el-input), .default-value-row :deep(.el-textarea) { width: 100%; }
@media (max-width: 720px) {
  .drawer-toolbar, .form-grid { grid-template-columns: 1fr; }
}
</style>
