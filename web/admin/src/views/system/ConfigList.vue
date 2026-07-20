<template>
  <div>
    <el-card>
      <template #header>
        <div style="display: flex; justify-content: space-between; align-items: center">
          <span>系统配置</span>
          <div>
            <el-button v-if="canAdd" @click="openCreate">新增配置</el-button>
            <el-button v-if="canEdit" :loading="refreshing" @click="handleRefreshAll">刷新全部缓存</el-button>
            <el-button v-if="canEdit" type="primary" :loading="saving" @click="handleSave">保存</el-button>
          </div>
        </div>
      </template>

      <el-tabs v-if="groups.length" v-model="activeGroup" v-loading="loading">
        <el-tab-pane v-for="g in groups" :key="g" :label="g" :name="g">
          <el-table :data="grouped[g]" row-key="id">
            <el-table-column label="名称" width="200">
              <template #default="{ row }">
                <div>{{ row.name || row.key }}</div>
                <div v-if="row.remark" style="color: #909399; font-size: 12px">{{ row.remark }}</div>
              </template>
            </el-table-column>
            <el-table-column label="键" width="210">
              <template #default="{ row }"><el-tag size="small" type="info">{{ row.key }}</el-tag></template>
            </el-table-column>
            <el-table-column label="值" min-width="360">
              <template #default="{ row }">
                <!-- 按类型给合理宽高：数字短、文本中等、select 中等，text/json 宽且高 -->
                <el-checkbox-group
                  v-if="isUploadExtensionConfig(row.key)"
                  :model-value="extensionValues(row.value)"
                  :disabled="!canEdit"
                  class="extension-options"
                  @change="setExtensionValue(row, $event)"
                >
                  <el-checkbox-button
                    v-for="opt in parseOptions(row.options)"
                    :key="opt.value"
                    :value="opt.value"
                  >{{ opt.label }}</el-checkbox-button>
                </el-checkbox-group>
                <div v-else-if="isUploadSizeConfig(row.key)" class="file-size-editor">
                  <el-input-number
                    :model-value="bytesToMB(row.value)"
                    :min="1"
                    :max="102400"
                    :step="1"
                    :disabled="!canEdit"
                    controls-position="right"
                    @update:model-value="setFileSizeMB(row, $event)"
                  />
                  <span>MB / 单文件</span>
                </div>
                <LogoImageUploader
                  v-else-if="row.key === 'site.logo'"
                  v-model="row.value"
                  :disabled="!canEdit"
                  :upload-disabled="!canUpload"
                />
                <el-switch v-else-if="row.type === 'bool'" v-model="row.value" active-value="true" inactive-value="false" />
                <el-select v-else-if="row.type === 'select'" v-model="row.value" style="width: 320px" placeholder="请选择">
                  <el-option v-for="opt in parseOptions(row.options)" :key="opt.value" :label="opt.label" :value="opt.value" />
                </el-select>
                <div v-else-if="row.type === 'color'" class="color-editor">
                  <el-color-picker v-model="row.value" :disabled="!canEdit" />
                  <el-input v-model="row.value" :disabled="!canEdit" maxlength="7" style="width: 130px" placeholder="#409EFF" />
                </div>
                <el-input
                  v-else-if="row.type === 'json'"
                  v-model="row.value"
                  type="textarea"
                  :autosize="{ minRows: 3, maxRows: 12 }"
                  style="width: 100%; font-family: monospace"
                />
                <el-input
                  v-else-if="row.type === 'text'"
                  v-model="row.value"
                  type="textarea"
                  :autosize="{ minRows: 2, maxRows: 6 }"
                  style="width: 100%"
                />
                <el-input
                  v-else-if="row.type === 'int' || row.type === 'float'"
                  v-model="row.value"
                  style="width: 180px"
                  placeholder="数字"
                />
                <el-input
                  v-else-if="row.type === 'password' || row.sensitive"
                  v-model="row.value"
                  type="password"
                  show-password
                  autocomplete="new-password"
                  style="width: 360px"
                />
                <el-input v-else v-model="row.value" style="width: 360px" />
              </template>
            </el-table-column>
            <el-table-column label="操作" width="190" fixed="right">
              <template #default="{ row }">
                <el-button v-if="canEdit" link type="primary" @click="openEdit(row)">编辑</el-button>
                <el-button v-if="canEdit" link type="primary" @click="handleRefreshKey(row.key)">刷新</el-button>
                <el-popconfirm
                  v-if="canDelete && !row.builtin"
                  title="确认删除该配置？"
                  @confirm="handleDelete(row.id)"
                >
                  <template #reference><el-button link type="danger">删除</el-button></template>
                </el-popconfirm>
              </template>
            </el-table-column>
          </el-table>
        </el-tab-pane>
      </el-tabs>
      <el-empty v-else description="暂无配置" />
    </el-card>

    <!-- 新增 / 编辑 -->
    <el-dialog v-model="dialogVisible" :title="dialogMode === 'edit' ? '编辑配置' : '新增配置'" width="560px" destroy-on-close>
      <el-form ref="formRef" :model="form" :rules="formRules" label-width="90px">
        <el-form-item label="分组" prop="group"><el-input v-model="form.group" placeholder="如：站点" /></el-form-item>
        <el-form-item label="键" prop="key">
          <el-input v-model="form.key" placeholder="如：site.name" :disabled="dialogMode === 'edit'" />
        </el-form-item>
        <el-form-item label="名称" prop="name"><el-input v-model="form.name" /></el-form-item>
        <el-form-item label="类型">
          <el-select v-model="form.type" style="width: 100%">
            <el-option v-for="t in types" :key="t" :label="t" :value="t" />
          </el-select>
        </el-form-item>

        <!-- select 选项编辑器：键值对，避免手写 JSON -->
        <el-form-item v-if="form.type === 'select'" label="选项">
          <div style="width: 100%">
            <div v-for="(opt, i) in optionRows" :key="i" style="display: flex; gap: 8px; margin-bottom: 6px">
              <el-input v-model="opt.label" placeholder="显示文字" />
              <el-input v-model="opt.value" placeholder="存储值" />
              <el-button link type="danger" @click="optionRows.splice(i, 1)">删除</el-button>
            </div>
            <el-button size="small" @click="optionRows.push({ label: '', value: '' })">+ 添加选项</el-button>
          </div>
        </el-form-item>

        <el-form-item label="值">
          <el-checkbox-group
            v-if="isUploadExtensionConfig(form.key)"
            :model-value="extensionValues(form.value)"
            class="extension-options"
            @change="setFormExtensionValue"
          >
            <el-checkbox-button
              v-for="opt in optionRows"
              :key="opt.value"
              :value="opt.value"
            >{{ opt.label || opt.value }}</el-checkbox-button>
          </el-checkbox-group>
          <div v-else-if="isUploadSizeConfig(form.key)" class="file-size-editor">
            <el-input-number
              :model-value="bytesToMB(form.value)"
              :min="1"
              :max="102400"
              :step="1"
              controls-position="right"
              @update:model-value="setFormFileSizeMB"
            />
            <span>MB / 单文件</span>
          </div>
          <LogoImageUploader
            v-else-if="form.key === 'site.logo'"
            v-model="form.value"
            :disabled="!canEdit"
            :upload-disabled="!canUpload"
          />
          <el-switch v-else-if="form.type === 'bool'" v-model="form.value" active-value="true" inactive-value="false" />
          <el-select v-else-if="form.type === 'select'" v-model="form.value" style="width: 100%" placeholder="默认值（从选项中选）">
            <el-option v-for="(o, i) in validOptionRows" :key="i" :label="o.label || o.value" :value="o.value" />
          </el-select>
          <div v-else-if="form.type === 'color'" class="color-editor">
            <el-color-picker v-model="form.value" />
            <el-input v-model="form.value" maxlength="7" placeholder="#409EFF" />
          </div>
          <el-input v-else-if="form.type === 'text' || form.type === 'json'" v-model="form.value" type="textarea" :rows="3" />
          <el-input
            v-else
            v-model="form.value"
            :type="form.type === 'password' ? 'password' : 'text'"
            :show-password="form.type === 'password'"
            autocomplete="new-password"
            :placeholder="form.type === 'int' || form.type === 'float' ? '数字' : ''"
          />
        </el-form-item>

        <el-form-item label="公开读"><el-switch v-model="form.is_public" /></el-form-item>
        <el-form-item label="排序"><el-input-number v-model="form.sort" :min="0" /></el-form-item>
        <el-form-item label="备注"><el-input v-model="form.remark" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitLoading" @click="handleSubmit">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { ElMessage, type FormInstance } from 'element-plus'
import { getConfigList, batchUpdateConfig, createConfig, updateConfig, deleteConfig, refreshConfig } from '@/api/config'
import { useUserStore } from '@/store/user'
import LogoImageUploader from '@/components/LogoImageUploader.vue'

const userStore = useUserStore()
const canEdit = computed(() => userStore.hasPermission('system:config:edit'))
const canAdd = computed(() => userStore.hasPermission('system:config:add'))
const canDelete = computed(() => userStore.hasPermission('system:config:delete'))
const canUpload = computed(() => userStore.hasPermission('system:upload'))

const loading = ref(false)
const saving = ref(false)
const refreshing = ref(false)
const submitLoading = ref(false)
const configs = ref<any[]>([])
const activeGroup = ref('')

const types = ['string', 'int', 'float', 'bool', 'text', 'json', 'select', 'password', 'color']
const bytesPerMB = 1024 * 1024
const uploadExtensionKeys = new Set(['upload.image_extensions', 'upload.video_extensions'])
const uploadSizeKeys = new Set(['upload.image_max_file_size', 'upload.video_max_file_size'])

const grouped = computed<Record<string, any[]>>(() => {
  const m: Record<string, any[]> = {}
  for (const c of configs.value) {
    const g = c.group || '其它'
    if (!m[g]) m[g] = []
    m[g].push(c)
  }
  return m
})
const groups = computed(() => Object.keys(grouped.value))

async function fetchData() {
  loading.value = true
  try {
    const res: any = await getConfigList()
    configs.value = res.data || []
    if ((!activeGroup.value || !groups.value.includes(activeGroup.value)) && groups.value.length) {
      activeGroup.value = groups.value[0]
    }
  } finally {
    loading.value = false
  }
}

async function handleSave() {
  if (!validateUploadConfigs(configs.value)) return
  saving.value = true
  try {
    const items = configs.value.map((c) => ({ key: c.key, value: String(c.value ?? '') }))
    await batchUpdateConfig(items)
    ElMessage.success('保存成功')
    fetchData()
  } finally {
    saving.value = false
  }
}

async function handleRefreshAll() {
  refreshing.value = true
  try {
    await refreshConfig()
    ElMessage.success('已刷新全部缓存')
  } finally {
    refreshing.value = false
  }
}

async function handleRefreshKey(key: string) {
  await refreshConfig(key)
  ElMessage.success(`已刷新：${key}`)
}

async function handleDelete(id: number) {
  await deleteConfig(id)
  ElMessage.success('删除成功')
  fetchData()
}

function parseOptions(options: string): { label: string; value: string }[] {
  if (!options) return []
  try {
    const arr = JSON.parse(options)
    return arr.map((o: any) =>
      typeof o === 'object' ? { label: o.label ?? o.value, value: String(o.value) } : { label: String(o), value: String(o) },
    )
  } catch {
    return []
  }
}

function isUploadExtensionConfig(key: string) {
  return uploadExtensionKeys.has(key)
}

function isUploadSizeConfig(key: string) {
  return uploadSizeKeys.has(key)
}

function extensionValues(value: string) {
  return String(value || '').split(',').map((item) => item.trim().toLowerCase()).filter(Boolean)
}

function normalizeExtensions(values: unknown) {
  return Array.isArray(values) ? [...new Set(values.map((value) => String(value).trim().toLowerCase()).filter(Boolean))] : []
}

function setExtensionValue(target: { value: string }, values: unknown) {
  target.value = normalizeExtensions(values).join(',')
}

function setFormExtensionValue(values: unknown) {
  form.value = normalizeExtensions(values).join(',')
}

function bytesToMB(value: unknown) {
  const bytes = Number(value)
  if (!Number.isFinite(bytes) || bytes <= 0) return 1
  return Math.max(1, Math.round(bytes / bytesPerMB))
}

function setFileSizeMB(target: { value: string }, value: number | undefined) {
  target.value = String(Math.max(1, Math.round(Number(value) || 1)) * bytesPerMB)
}

function setFormFileSizeMB(value: number | undefined) {
  setFileSizeMB(form, value)
}

function validateUploadConfigs(rows: { key: string; value: string }[]) {
  for (const row of rows) {
    if (isUploadExtensionConfig(row.key) && extensionValues(row.value).length === 0) {
      ElMessage.warning(row.key.includes('image') ? '请至少选择一种图片格式' : '请至少选择一种视频格式')
      return false
    }
    if (isUploadSizeConfig(row.key)) {
		  const bytes = Number(row.value)
		  const sizeMB = bytes / bytesPerMB
		  if (!Number.isFinite(bytes) || !Number.isInteger(bytes) || sizeMB < 1 || sizeMB > 102400) {
        ElMessage.warning('单文件大小必须在 1 MB 到 102400 MB 之间')
        return false
      }
    }
  }
  return true
}

// ── 新增 / 编辑弹窗 ──
const dialogVisible = ref(false)
const dialogMode = ref<'create' | 'edit'>('create')
const editingId = ref(0)
const formRef = ref<FormInstance>()
const optionRows = ref<{ label: string; value: string }[]>([])
const defaultForm = { group: '', key: '', name: '', type: 'string', value: '', is_public: false, sort: 0, remark: '' }
const form = reactive({ ...defaultForm })
const formRules = {
  key: [{ required: true, message: '请输入配置键', trigger: 'blur' }],
  name: [{ required: true, message: '请输入名称', trigger: 'blur' }],
}

const validOptionRows = computed(() => optionRows.value.filter((o) => o.value !== ''))

function openCreate() {
  dialogMode.value = 'create'
  editingId.value = 0
  Object.assign(form, defaultForm)
  optionRows.value = []
  dialogVisible.value = true
}

function openEdit(row: any) {
  dialogMode.value = 'edit'
  editingId.value = row.id
  Object.assign(form, {
    group: row.group,
    key: row.key,
    name: row.name,
    type: row.type || 'string',
    value: row.value,
    is_public: row.is_public,
    sort: row.sort,
    remark: row.remark,
  })
  optionRows.value = parseOptions(row.options)
  dialogVisible.value = true
}

async function handleSubmit() {
  await formRef.value?.validate()

  if (!validateUploadConfigs([{ key: form.key, value: form.value }])) return

  let options = ''
  if (form.type === 'select') {
    const rows = optionRows.value.filter((o) => o.value !== '')
    if (rows.length === 0) {
      ElMessage.warning('请至少添加一个选项')
      return
    }
    options = JSON.stringify(rows.map((o) => ({ label: o.label || o.value, value: o.value })))
  }

  const payload = { ...form, options }
  submitLoading.value = true
  try {
    if (dialogMode.value === 'edit') {
      await updateConfig(editingId.value, payload)
    } else {
      await createConfig(payload)
    }
    ElMessage.success('保存成功')
    dialogVisible.value = false
    fetchData()
  } finally {
    submitLoading.value = false
  }
}

onMounted(fetchData)
</script>

<style scoped>
.extension-options { display: flex; flex-wrap: wrap; gap: 6px; }
.extension-options :deep(.el-checkbox-button__inner) { border-left: 1px solid var(--el-border-color); border-radius: 4px; box-shadow: none; }
.file-size-editor { display: flex; align-items: center; gap: 10px; color: #606266; }
.file-size-editor :deep(.el-input-number) { width: 180px; }
.color-editor { display: flex; align-items: center; gap: 10px; }
</style>
