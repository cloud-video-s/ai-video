<template>
  <div class="page-wrap">
    <el-card shadow="never">
      <template #header>
        <div class="page-header">
          <div>
            <div class="page-title">渠道管理</div>
            <div class="page-subtitle">维护投放渠道、代理关系、归因监测与结算配置</div>
          </div>
          <el-button v-if="canAdd" type="primary" @click="openCreate">
            <el-icon><Plus /></el-icon>新增渠道
          </el-button>
        </div>
      </template>

      <div class="filters">
        <el-input v-model="query.keyword" clearable placeholder="渠道码、名称、代理公司或投放包" @keyup.enter="handleSearch">
          <template #prefix><el-icon><Search /></el-icon></template>
        </el-input>
        <el-select v-model="query.ad_platform" clearable filterable placeholder="投放平台">
          <el-option v-for="item in platformOptions" :key="item" :label="item" :value="item" />
        </el-select>
        <el-select v-model="query.upload_method" clearable placeholder="上传方式">
          <el-option v-for="item in uploadMethodOptions" :key="item" :label="item" :value="item" />
        </el-select>
        <el-select v-model="query.status" clearable placeholder="全部状态">
          <el-option label="启用" value="1" />
          <el-option label="禁用" value="0" />
        </el-select>
        <el-button type="primary" plain @click="handleSearch">查询</el-button>
        <el-button @click="handleReset">重置</el-button>
      </div>

      <el-table v-loading="loading" :data="tableData" row-key="channel_id" stripe>
        <el-table-column prop="channel_id" label="渠道ID" width="85" />
        <el-table-column label="渠道信息" min-width="210">
          <template #default="{ row }">
            <div class="primary-text">{{ row.channel_name }}</div>
            <code class="channel-code">{{ row.channel_code }}</code>
          </template>
        </el-table-column>
        <el-table-column prop="agency_company" label="代理公司" min-width="150">
          <template #default="{ row }">{{ row.agency_company || '-' }}</template>
        </el-table-column>
        <el-table-column label="投放配置" min-width="210">
          <template #default="{ row }">
            <div><el-tag size="small" effect="plain">{{ row.ad_platform }}</el-tag></div>
            <div class="secondary-text package-text">{{ row.delivery_package }}</div>
          </template>
        </el-table-column>
        <el-table-column label="监测链接" width="100" align="center">
          <template #default="{ row }">
            <el-link v-if="row.tracking_url" :href="row.tracking_url" type="primary" target="_blank" :underline="false">打开链接</el-link>
            <span v-else class="secondary-text">未配置</span>
          </template>
        </el-table-column>
        <el-table-column label="结算配置" width="155">
          <template #default="{ row }">
            <div class="settlement-line"><span>返点</span><strong>{{ formatRebate(row.port_rebate) }}%</strong></div>
            <div class="settlement-line"><span>单费</span><strong>¥{{ formatMoney(row.service_order_fee) }}</strong></div>
          </template>
        </el-table-column>
        <el-table-column label="上传方式" width="105" align="center">
          <template #default="{ row }"><el-tag type="info" effect="plain">{{ row.upload_method }}</el-tag></template>
        </el-table-column>
        <el-table-column label="状态" width="100" align="center">
          <template #default="{ row }">
            <el-switch
              v-if="canEdit"
              v-model="row.status"
              :active-value="1"
              :inactive-value="0"
              :loading="updatingIds.includes(row.channel_id)"
              @change="handleStatusChange(row)"
            />
            <el-tag v-else :type="row.status === 1 ? 'success' : 'info'">{{ row.status === 1 ? '启用' : '禁用' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="更新时间" width="180">
          <template #default="{ row }">{{ formatDate(row.updated_at) }}</template>
        </el-table-column>
        <el-table-column v-if="canEdit || canDelete" label="操作" width="130" fixed="right" align="center">
          <template #default="{ row }">
            <el-button v-if="canEdit" link type="primary" @click="openEdit(row)">编辑</el-button>
            <el-popconfirm
              v-if="canDelete"
              :title="`确认删除渠道 ${row.channel_name}？`"
              width="240"
              @confirm="handleDelete(row.channel_id)"
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
          @size-change="handlePageSizeChange"
          @current-change="fetchData"
        />
      </div>
    </el-card>

    <el-dialog v-model="dialogVisible" :title="form.channel_id ? '编辑渠道' : '新增渠道'" width="780px" destroy-on-close>
      <el-form ref="formRef" :model="form" :rules="rules" label-width="105px">
        <div class="form-grid">
          <el-form-item label="唯一识别码" prop="channel_code">
            <el-input v-model="form.channel_code" maxlength="64" placeholder="例如：META_US_001" />
          </el-form-item>
          <el-form-item label="渠道名称" prop="channel_name">
            <el-input v-model="form.channel_name" maxlength="128" placeholder="请输入渠道名称" />
          </el-form-item>
          <el-form-item label="代理公司" prop="agency_company">
            <el-input v-model="form.agency_company" maxlength="128" placeholder="可选" />
          </el-form-item>
          <el-form-item label="投放平台" prop="ad_platform">
            <el-select
              v-model="form.ad_platform"
              filterable
              allow-create
              default-first-option
              :reserve-keyword="false"
              placeholder="请选择或输入平台"
              style="width: 100%"
            >
              <el-option v-for="item in platformOptions" :key="item" :label="item" :value="item" />
            </el-select>
          </el-form-item>
          <el-form-item label="投放包" prop="delivery_package">
            <el-select
              v-model="form.delivery_package"
              filterable
              allow-create
              default-first-option
              :reserve-keyword="false"
              placeholder="请选择或输入包标识"
              style="width: 100%"
            >
              <el-option
                v-for="item in packageOptions"
                :key="item.id"
                :label="`${item.package_name} · ${item.package_code} · ${item.package_version}`"
                :value="item.package_code"
              />
            </el-select>
          </el-form-item>
          <el-form-item label="上传方式" prop="upload_method">
            <el-select
              v-model="form.upload_method"
              filterable
              allow-create
              default-first-option
              :reserve-keyword="false"
              placeholder="请选择或输入方式"
              style="width: 100%"
              @change="normalizeUploadMethod"
            >
              <el-option v-for="item in uploadMethodOptions" :key="item" :label="item" :value="item" />
            </el-select>
          </el-form-item>
          <el-form-item label="端口返点" prop="port_rebate">
            <div class="number-with-unit">
              <el-input-number v-model="form.port_rebate" :min="0" :max="100" :precision="4" :step="0.1" controls-position="right" />
              <span>%</span>
            </div>
          </el-form-item>
          <el-form-item label="服务单费" prop="service_order_fee">
            <div class="number-with-unit">
              <span>¥</span>
              <el-input-number v-model="form.service_order_fee" :min="0" :max="9999999999.99" :precision="2" :step="0.01" controls-position="right" />
            </div>
          </el-form-item>
          <el-form-item label="状态">
            <el-radio-group v-model="form.status">
              <el-radio :value="1">启用</el-radio>
              <el-radio :value="0">禁用</el-radio>
            </el-radio-group>
          </el-form-item>
        </div>
        <el-form-item label="监测链接" prop="tracking_url">
          <el-input v-model="form.tracking_url" maxlength="1024" clearable placeholder="https://tracker.example.com/click?...">
            <template #append>
              <el-link v-if="form.tracking_url" :href="form.tracking_url" target="_blank" :underline="false">测试</el-link>
              <span v-else>测试</span>
            </template>
          </el-input>
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
  createChannel,
  deleteChannel,
  getChannelList,
  updateChannel,
  updateChannelStatus,
  type Channel,
  type ChannelPayload,
} from '@/api/channel'
import { getPackageOptions, type AppPackage } from '@/api/package'
import { useUserStore } from '@/store/user'

const userStore = useUserStore()
const canAdd = computed(() => userStore.hasPermission('channel:add'))
const canEdit = computed(() => userStore.hasPermission('channel:edit'))
const canDelete = computed(() => userStore.hasPermission('channel:delete'))

const platformOptions = ['Google Ads', 'Meta Ads', 'TikTok Ads', 'Unity Ads', 'AppLovin', 'ironSource', 'Mintegral']
const uploadMethodOptions = ['API', 'SFTP', 'MANUAL']
const loading = ref(false)
const submitting = ref(false)
const dialogVisible = ref(false)
const formRef = ref<FormInstance>()
const tableData = ref<Channel[]>([])
const packageOptions = ref<AppPackage[]>([])
const updatingIds = ref<number[]>([])
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const query = reactive({ keyword: '', ad_platform: '', upload_method: '', status: '' })

const defaultForm: ChannelPayload & { channel_id: number } = {
  channel_id: 0,
  channel_code: '',
  channel_name: '',
  agency_company: '',
  ad_platform: '',
  delivery_package: '',
  tracking_url: '',
  port_rebate: 0,
  service_order_fee: 0,
  upload_method: 'API',
  status: 1,
}
const form = reactive({ ...defaultForm })
const rules: FormRules = {
  channel_code: [
    { required: true, message: '请输入渠道唯一识别码', trigger: 'blur' },
    { pattern: /^[A-Za-z0-9._-]+$/, message: '仅支持字母、数字、点、下划线和中划线', trigger: 'blur' },
  ],
  channel_name: [{ required: true, message: '请输入渠道名称', trigger: 'blur' }],
  ad_platform: [{ required: true, message: '请选择或输入投放平台', trigger: 'change' }],
  delivery_package: [{ required: true, message: '请选择或输入投放包', trigger: 'change' }],
  upload_method: [
    { required: true, message: '请选择或输入上传方式', trigger: 'change' },
    { pattern: /^[A-Z][A-Z0-9_-]{0,31}$/, message: '仅支持大写字母、数字、下划线和中划线', trigger: 'change' },
  ],
  tracking_url: [
    { pattern: /^(https?:\/\/.*)?$/i, message: '请输入有效的 HTTP(S) 地址', trigger: 'blur' },
  ],
  port_rebate: [{ type: 'number', min: 0, max: 100, message: '返点必须在 0 到 100 之间', trigger: 'change' }],
  service_order_fee: [{ type: 'number', min: 0, message: '服务单费不能小于 0', trigger: 'change' }],
}

async function fetchData() {
  loading.value = true
  try {
    const params: Record<string, unknown> = { page: page.value, page_size: pageSize.value }
    for (const [key, value] of Object.entries(query)) {
      if (value !== '') params[key] = value.trim()
    }
    const res: any = await getChannelList(params)
    tableData.value = res.data.list || []
    total.value = res.data.total || 0
  } finally {
    loading.value = false
  }
}

async function fetchPackageOptions() {
  try {
    const res: any = await getPackageOptions()
    packageOptions.value = (res.data || []).filter((item: AppPackage) => item.status === 1)
  } catch {
    packageOptions.value = []
  }
}

function handleSearch() {
  page.value = 1
  fetchData()
}

function handleReset() {
  Object.assign(query, { keyword: '', ad_platform: '', upload_method: '', status: '' })
  page.value = 1
  fetchData()
}

function handlePageSizeChange() {
  page.value = 1
  fetchData()
}

function openCreate() {
  Object.assign(form, defaultForm)
  dialogVisible.value = true
}

function openEdit(row: Channel) {
  Object.assign(form, {
    channel_id: row.channel_id,
    channel_code: row.channel_code,
    channel_name: row.channel_name,
    agency_company: row.agency_company || '',
    ad_platform: row.ad_platform,
    delivery_package: row.delivery_package,
    tracking_url: row.tracking_url || '',
    port_rebate: Number(row.port_rebate || 0),
    service_order_fee: Number(row.service_order_fee || 0),
    upload_method: row.upload_method,
    status: row.status,
  })
  dialogVisible.value = true
}

function normalizeUploadMethod(value: string) {
  form.upload_method = String(value || '').trim().toUpperCase()
}

async function handleSubmit() {
  normalizeUploadMethod(form.upload_method)
  await formRef.value?.validate()
  submitting.value = true
  try {
    const payload: ChannelPayload = {
      channel_code: form.channel_code.trim(),
      channel_name: form.channel_name.trim(),
      agency_company: form.agency_company.trim(),
      ad_platform: form.ad_platform.trim(),
      delivery_package: form.delivery_package.trim(),
      tracking_url: form.tracking_url.trim(),
      port_rebate: Number(form.port_rebate),
      service_order_fee: Number(form.service_order_fee),
      upload_method: form.upload_method,
      status: form.status,
    }
    if (form.channel_id) await updateChannel(form.channel_id, payload)
    else await createChannel(payload)
    ElMessage.success('渠道信息已保存')
    dialogVisible.value = false
    await fetchData()
  } finally {
    submitting.value = false
  }
}

async function handleStatusChange(row: Channel) {
  updatingIds.value.push(row.channel_id)
  try {
    await updateChannelStatus(row.channel_id, row.status)
    ElMessage.success(`${row.channel_name} 已${row.status === 1 ? '启用' : '禁用'}`)
  } catch {
    row.status = row.status === 1 ? 0 : 1
  } finally {
    updatingIds.value = updatingIds.value.filter((id) => id !== row.channel_id)
  }
}

async function handleDelete(id: number) {
  await deleteChannel(id)
  ElMessage.success('渠道已删除')
  if (tableData.value.length === 1 && page.value > 1) page.value--
  await fetchData()
}

function formatRebate(value: number) {
  return Number(value || 0).toLocaleString('zh-CN', { maximumFractionDigits: 4 })
}

function formatMoney(value: number) {
  return Number(value || 0).toLocaleString('zh-CN', { minimumFractionDigits: 2, maximumFractionDigits: 2 })
}

function formatDate(value: string) {
  if (!value) return '-'
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString('zh-CN', { hour12: false })
}

onMounted(() => {
  fetchData()
  fetchPackageOptions()
})
</script>

<style scoped>
.page-wrap { min-width: 0; }
.page-header { display: flex; align-items: center; justify-content: space-between; gap: 16px; }
.page-title { color: #303133; font-size: 17px; font-weight: 600; }
.page-subtitle { margin-top: 4px; color: #909399; font-size: 12px; }
.filters { display: grid; grid-template-columns: minmax(240px, 1.6fr) repeat(3, minmax(130px, 0.8fr)) auto auto; gap: 10px; margin-bottom: 16px; }
.primary-text { color: #303133; font-weight: 600; }
.secondary-text { margin-top: 5px; color: #909399; font-size: 12px; }
.channel-code { display: inline-block; margin-top: 5px; padding: 2px 7px; border-radius: 4px; background: #f5f7fa; color: #606266; font-size: 12px; }
.package-text { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.settlement-line { display: flex; justify-content: space-between; gap: 10px; line-height: 24px; }
.settlement-line span { color: #909399; }
.settlement-line strong { color: #303133; font-variant-numeric: tabular-nums; }
.pagination-wrap { display: flex; justify-content: flex-end; margin-top: 16px; overflow-x: auto; }
.form-grid { display: grid; grid-template-columns: 1fr 1fr; column-gap: 16px; }
.number-with-unit { display: flex; align-items: center; gap: 8px; width: 100%; color: #606266; }
.number-with-unit :deep(.el-input-number) { width: 100%; }
@media (max-width: 1000px) {
  .filters { grid-template-columns: repeat(2, minmax(160px, 1fr)); }
}
@media (max-width: 680px) {
  .page-header { align-items: stretch; flex-direction: column; }
  .filters, .form-grid { grid-template-columns: 1fr; }
  .page-wrap :deep(.el-card__header), .page-wrap :deep(.el-card__body) { padding: 14px; }
}
</style>
