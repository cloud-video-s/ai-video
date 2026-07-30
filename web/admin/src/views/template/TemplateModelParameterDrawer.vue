<template>
  <el-drawer v-model="visible" size="min(960px, 92vw)" append-to-body destroy-on-close>
    <template #header>
      <div>
        <div class="drawer-title">模板模型配置</div>
        <div class="drawer-subtitle">{{ model?.name }} · {{ model?.code }} · {{ model?.version }}</div>
      </div>
    </template>

    <div class="drawer-toolbar">
      <el-alert type="info" :closable="false" show-icon>
        <template #title>配置方式与模型管理一致；模板只能从关联模型已有参数中选择，选项值只能收窄，不能超出模型允许范围。</template>
      </el-alert>
      <el-button type="primary" :disabled="availableDefinitions.length === 0" @click="openCreate">
        <el-icon><Plus /></el-icon>新增配置
      </el-button>
    </div>

    <el-table v-loading="loading" :data="parameters" row-key="param_key" stripe>
      <el-table-column prop="sort_order" label="排序" width="70" align="center" />
      <el-table-column label="字段" min-width="160">
        <template #default="{ row }">
          <code class="param-key">{{ row.param_key }}</code>
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
      <el-table-column label="配置内容" min-width="290">
        <template #default="{ row }">
          <template v-if="row.parameter_type === 1">
            <div class="value-tags">
              <el-tag
                v-for="(value, index) in row.allowed_values"
                :key="`${row.param_key}-${index}`"
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
          <el-popconfirm :title="`确认移除配置 ${row.param_key}？`" width="250" @confirm="handleDelete(row.param_key)">
            <template #reference><el-button link type="danger">删除</el-button></template>
          </el-popconfirm>
        </template>
      </el-table-column>
    </el-table>

    <el-empty v-if="!loading && parameters.length === 0" description="该模板尚未配置模型参数" :image-size="90" />

    <el-dialog
      v-model="dialogVisible"
      :title="form.editing ? '编辑模板模型配置' : '新增模板模型配置'"
      width="720px"
      append-to-body
      destroy-on-close
    >
      <el-form label-width="105px">
        <el-form-item label="参数字段" required>
          <el-select
            v-model="form.param_key"
            filterable
            :disabled="form.editing"
            placeholder="请选择关联模型的参数字段"
            style="width: 100%"
            @change="handleDefinitionChange"
          >
            <el-option
              v-for="item in selectableDefinitions"
              :key="item.param_key"
              :label="`${item.param_key} · ${item.description || item.value_type}`"
              :value="item.param_key"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="配置类型" required>
          <el-radio-group v-model="form.parameter_type" disabled>
            <el-radio-button :value="1">选项参数</el-radio-button>
            <el-radio-button :value="2">请求参数</el-radio-button>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="值类型" required>
          <el-select v-model="form.value_type" disabled style="width: 100%">
            <el-option label="string（字符串）" value="string" />
            <el-option label="integer（整数）" value="integer" />
            <el-option label="number（数字）" value="number" />
            <el-option label="boolean（布尔值）" value="boolean" />
            <el-option label="object（对象）" value="object" />
            <el-option label="array（数组）" value="array" />
          </el-select>
        </el-form-item>

        <template v-if="form.parameter_type === 1">
          <el-form-item label="选择值" required>
            <el-select
              v-model="form.option_values"
              multiple
              filterable
              style="width: 100%"
              placeholder="请从模型允许值中选择"
            >
              <el-option v-for="value in baseOptionValues" :key="value" :label="value" :value="value" />
            </el-select>
            <div class="form-help">模板可以收窄模型选项范围，但不能新增模型未定义的值。</div>
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
            <div class="form-help">字段和格式与模型管理中的请求参数配置一致。</div>
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
        <el-button type="primary" @click="handleSubmit">保存</el-button>
      </template>
    </el-dialog>
  </el-drawer>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import type { ModelParameter, ModelParameterPayload, VideoModel } from '@/api/videoModel'

type ValueType = ModelParameter['value_type']

const props = defineProps<{
  modelValue: boolean
  model: VideoModel | null
  definitions: ModelParameter[]
  parameters: ModelParameterPayload[]
  loading?: boolean
}>()
const emit = defineEmits<{
  (event: 'update:modelValue', value: boolean): void
  (event: 'update:parameters', value: ModelParameterPayload[]): void
}>()

const visible = computed({
  get: () => props.modelValue,
  set: (value: boolean) => emit('update:modelValue', value),
})
const dialogVisible = ref(false)

interface ParameterForm {
  editing: boolean
  param_key: string
  value_type: ValueType
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
  editing: false,
  param_key: '',
  value_type: 'string',
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

const configuredKeys = computed(() => new Set(props.parameters.map((item) => item.param_key)))
const availableDefinitions = computed(() => props.definitions.filter((item) => !configuredKeys.value.has(item.param_key)))
const selectableDefinitions = computed(() => {
  if (!form.editing) return availableDefinitions.value
  return props.definitions.filter((item) => item.param_key === form.param_key)
})
const currentDefinition = computed(() => props.definitions.find((item) => item.param_key === form.param_key) || null)
const baseOptionValues = computed(() => (currentDefinition.value?.allowed_values || []).map(displayValue))
const defaultPlaceholder = computed(() => {
  if (form.value_type === 'object') return '{"key":"value"}'
  if (form.value_type === 'array') return '["value1","value2"]'
  if (form.value_type === 'boolean') return 'true 或 false'
  return '请输入默认值'
})

function resetForm() {
  Object.assign(form, { ...defaultForm, option_values: [] })
}

function openCreate() {
  resetForm()
  const first = availableDefinitions.value[0]
  if (!first) {
    ElMessage.warning('关联模型没有其他可配置参数')
    return
  }
  form.param_key = first.param_key
  applyDefinition(first)
  dialogVisible.value = true
}

function openEdit(row: ModelParameterPayload) {
  const definition = props.definitions.find((item) => item.param_key === row.param_key)
  if (!definition) {
    ElMessage.error(`模型参数 ${row.param_key} 已不存在，请移除该模板配置`)
    return
  }
  Object.assign(form, {
    editing: true,
    param_key: row.param_key,
    value_type: row.value_type,
    parameter_type: row.parameter_type,
    is_required: row.is_required,
    option_values: (row.allowed_values || []).map(displayValue),
    option_default: row.default_value === null ? '' : displayValue(row.default_value),
    has_default: row.default_value !== null,
    default_input: row.default_value === null ? '' : inputValue(row.default_value, row.value_type),
    constraints_text: JSON.stringify(row.constraints || {}, null, 2),
    description: row.description || '',
    sort_order: row.sort_order,
  })
  dialogVisible.value = true
}

function handleDefinitionChange(paramKey: string) {
  const definition = props.definitions.find((item) => item.param_key === paramKey)
  if (definition) applyDefinition(definition)
}

function applyDefinition(definition: ModelParameter) {
  Object.assign(form, {
    value_type: definition.value_type,
    parameter_type: definition.parameter_type,
    is_required: definition.is_required,
    option_values: (definition.allowed_values || []).map(displayValue),
    option_default: definition.default_value === null ? '' : displayValue(definition.default_value),
    has_default: definition.default_value !== null,
    default_input: definition.default_value === null ? '' : inputValue(definition.default_value, definition.value_type),
    constraints_text: JSON.stringify(definition.constraints || {}, null, 2),
    description: definition.description || '',
    sort_order: definition.sort_order,
  })
}

function handleSubmit() {
  const definition = currentDefinition.value
  if (!definition) {
    ElMessage.error('请选择有效的模型参数字段')
    return
  }

  let defaultValue: unknown = null
  let allowedValues: unknown[] = []
  let constraints: Record<string, unknown> = {}
  try {
    if (form.parameter_type === 1) {
      if (form.option_values.length === 0) throw new Error('请至少选择一个模型允许值')
      if (!form.option_values.includes(form.option_default)) throw new Error('请从选择值中指定一个默认选择')
      allowedValues = form.option_values.map((value) => rawOptionValue(definition, value))
      defaultValue = rawOptionValue(definition, form.option_default)
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
    param_key: definition.param_key,
    value_type: definition.value_type,
    parameter_type: definition.parameter_type,
    is_required: definition.parameter_type === 2 ? form.is_required : 0,
    default_value: defaultValue,
    allowed_values: allowedValues,
    constraints,
    description: form.description.trim(),
    sort_order: form.sort_order,
  }
  const next = props.parameters.filter((item) => item.param_key !== payload.param_key)
  next.push(payload)
  next.sort((left, right) => left.parameter_type - right.parameter_type || left.sort_order - right.sort_order || left.param_key.localeCompare(right.param_key))
  emit('update:parameters', next)
  dialogVisible.value = false
}

function handleDelete(paramKey: string) {
  emit('update:parameters', props.parameters.filter((item) => item.param_key !== paramKey))
}

function rawOptionValue(definition: ModelParameter, display: string): unknown {
  const value = (definition.allowed_values || []).find((item) => displayValue(item) === display)
  if (value === undefined) throw new Error(`选项值 ${display} 不属于模型允许范围`)
  return cloneJSON(value)
}

function cloneJSON<T>(value: T): T {
  if (value === undefined || value === null) return value
  return JSON.parse(JSON.stringify(value)) as T
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
  return typeof value === 'string' ? value : JSON.stringify(value)
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
.drawer-toolbar { display: flex; align-items: flex-start; justify-content: space-between; gap: 12px; margin-bottom: 14px; }
.drawer-toolbar .el-alert { flex: 1; }
.param-key { color: #303133; font-weight: 600; }
.description { margin-top: 5px; color: #909399; font-size: 12px; }
.value-tags { display: flex; flex-wrap: wrap; gap: 5px; }
.muted { color: #909399; }
.default-line { margin-top: 6px; }
.form-grid { display: grid; grid-template-columns: 1fr 1fr; column-gap: 14px; }
.form-grid :deep(.el-input-number) { width: 100%; }
.form-help { margin-top: 5px; color: #909399; font-size: 12px; line-height: 1.5; }
.default-value-row { display: flex; width: 100%; flex-direction: column; gap: 10px; }
@media (max-width: 720px) {
  .drawer-toolbar { align-items: stretch; flex-direction: column; }
  .form-grid { grid-template-columns: 1fr; }
}
</style>
