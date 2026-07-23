<template>
  <div class="page-wrap">
    <el-card shadow="never">
      <template #header>
        <div class="page-header">
          <div>
            <div class="page-title">VIP 订阅管理</div>
            <div class="page-subtitle">维护多平台订阅 SKU、价格权益、展示区域和渠道投放规则</div>
          </div>
          <el-button v-if="canAdd" type="primary" @click="openCreate"><el-icon><Plus /></el-icon>新增套餐</el-button>
        </div>
      </template>

      <div class="filters">
        <el-select v-model="query.plan_type" clearable placeholder="套餐类型"><el-option v-for="item in planTypeOptions" :key="item.value" :label="item.label" :value="item.value" /></el-select>
        <el-select v-model="query.status" clearable placeholder="状态"><el-option label="启用" value="1" /><el-option label="禁用" value="0" /></el-select>
        <el-select v-model="query.package_id" clearable filterable placeholder="应用包"><el-option v-for="item in packageOptions" :key="item.id" :label="packageLabel(item)" :value="String(item.id)" /></el-select>
        <el-select v-model="query.display_mode" clearable placeholder="显示模式"><el-option label="正常显示" value="1" /><el-option label="隐藏" value="0" /></el-select>
        <el-select v-model="query.is_subscription" clearable placeholder="是否订阅"><el-option label="订阅" value="true" /><el-option label="非订阅" value="false" /></el-select>
        <el-select v-model="query.platform" clearable placeholder="平台"><el-option v-for="item in platformOptions" :key="item.value" :label="item.label" :value="item.value" /></el-select>
        <el-select v-model="query.channel_id" clearable filterable placeholder="渠道"><el-option v-for="item in channelOptions" :key="item.channel_id" :label="channelLabel(item)" :value="String(item.channel_id)" /></el-select>
        <el-select v-model="query.excluded_channel_id" clearable filterable placeholder="排除渠道"><el-option v-for="item in channelOptions" :key="item.channel_id" :label="channelLabel(item)" :value="String(item.channel_id)" /></el-select>
        <el-input v-model="query.keyword" clearable placeholder="产品ID、名称、VIP等级" @keyup.enter="handleSearch" />
        <el-button type="primary" plain @click="handleSearch">查询</el-button>
        <el-button @click="handleReset">重置</el-button>
      </div>

      <el-table v-loading="loading" :data="tableData" row-key="id" stripe>
        <el-table-column prop="id" label="ID" width="70" />
        <el-table-column prop="sort" label="排序" width="68" align="center" />
        <el-table-column label="套餐" min-width="220">
          <template #default="{ row }"><div class="primary-text">{{ row.name }}</div><div class="secondary-text mono">{{ row.product_id }}</div><div class="tag-row"><el-tag size="small" effect="plain">{{ planTypeLabel(row.plan_type) }}</el-tag><el-tag size="small" type="warning" effect="plain">{{ row.vip_level }}</el-tag></div></template>
        </el-table-column>
        <el-table-column label="平台 / 应用包" min-width="210">
          <template #default="{ row }"><div><el-tag size="small" type="info">{{ platformLabel(row.platform) }}</el-tag></div><div class="secondary-text">{{ row.package?.package_name || `安装包 #${row.package_id}` }}</div><div class="secondary-text mono">{{ row.package?.package_code }}</div></template>
        </el-table-column>
        <el-table-column label="价格" width="150">
          <template #default="{ row }"><div class="price">{{ row.currency }} {{ formatMoney(row.subscription_price) }}</div><div v-if="row.original_price" class="original-price">{{ row.currency }} {{ formatMoney(row.original_price) }}</div><div class="secondary-text">周期 {{ row.subscription_period || '-' }}</div></template>
        </el-table-column>
        <el-table-column label="展示区域" min-width="180"><template #default="{ row }"><div class="target-tags"><el-tag v-for="item in row.display_positions || []" :key="item.id" size="small" effect="plain">{{ item.position_name }}</el-tag><span v-if="!row.display_positions?.length" class="secondary-text">全部区域</span></div></template></el-table-column>
        <el-table-column label="渠道" min-width="190"><template #default="{ row }"><div class="secondary-text">包含：{{ channelNames(row.channels) }}</div><div class="secondary-text danger-text">排除：{{ channelNames(row.excluded_channels) }}</div></template></el-table-column>
        <el-table-column label="属性" width="150"><template #default="{ row }"><div class="tag-row"><el-tag v-if="row.is_default" size="small" type="success">默认</el-tag><el-tag v-if="row.free_trial" size="small" type="warning">{{ row.trial_days }}天试用</el-tag><el-tag size="small" :type="row.is_subscription ? '' : 'info'">{{ row.is_subscription ? '订阅' : '一次性' }}</el-tag></div></template></el-table-column>
        <el-table-column label="状态" width="150" align="center"><template #default="{ row }"><el-switch v-if="canEdit" v-model="row.status" :active-value="1" :inactive-value="0" inline-prompt active-text="启用" inactive-text="禁用" :loading="updatingIds.includes(row.id)" @change="handleStatusChange(row)" /><el-tag v-else :type="row.status === 1 ? 'success' : 'danger'">{{ row.status === 1 ? '启用' : '禁用' }}</el-tag><div class="display-state">{{ row.display_mode === 1 ? '正常显示' : '已隐藏' }}</div></template></el-table-column>
        <el-table-column label="操作" width="260" fixed="right" align="center">
          <template #default="{ row }">
            <el-button v-if="canEdit" link type="primary" @click="openEdit(row)">编辑</el-button>
            <el-button v-if="canEdit" link :type="row.display_mode === 1 ? 'warning' : 'success'" @click="toggleDisplay(row)">{{ row.display_mode === 1 ? '隐藏' : '显示' }}</el-button>
            <el-button v-if="canEdit && !row.is_default" link type="primary" @click="setDefault(row)">设为默认</el-button>
            <el-button v-if="canAdd" link type="primary" @click="clonePlan(row)">复制</el-button>
            <el-popconfirm v-if="canDelete" title="确认删除该 VIP 套餐？" @confirm="handleDelete(row.id)"><template #reference><el-button link type="danger">删除</el-button></template></el-popconfirm>
          </template>
        </el-table-column>
      </el-table>
      <div class="pagination-wrap"><el-pagination v-model:current-page="page" v-model:page-size="pageSize" :total="total" :page-sizes="[10, 20, 50, 100]" layout="total, sizes, prev, pager, next" @size-change="handlePageSizeChange" @current-change="fetchData" /></div>
    </el-card>

    <el-dialog v-model="dialogVisible" :title="form.id ? '编辑 VIP 订阅套餐' : '新增 VIP 订阅套餐'" width="940px" destroy-on-close :close-on-click-modal="false">
      <el-form ref="formRef" :model="form" :rules="rules" label-width="124px">
        <el-tabs v-model="formTab">
          <el-tab-pane label="基础配置" name="base">
            <div class="form-grid">
              <el-form-item label="产品 ID" prop="product_id"><el-input v-model="form.product_id" maxlength="191" placeholder="应用商店订阅 SKU" /></el-form-item>
              <el-form-item label="VIP 名称" prop="name"><el-input v-model="form.name" maxlength="128" /></el-form-item>
              <el-form-item label="VIP 等级" prop="vip_level"><el-input v-model="form.vip_level" maxlength="64" placeholder="例如：月度会员、年度会员" /></el-form-item>
              <el-form-item label="套餐类型" prop="plan_type"><el-select v-model="form.plan_type" style="width: 100%"><el-option v-for="item in planTypeOptions" :key="item.value" :label="item.label" :value="item.value" /></el-select></el-form-item>
              <el-form-item label="应用包" prop="package_id"><el-select v-model="form.package_id" filterable style="width: 100%" @change="handlePackageChange"><el-option v-for="item in packageOptions" :key="item.id" :label="packageLabel(item)" :value="item.id" /></el-select></el-form-item>
              <el-form-item label="平台" prop="platform"><el-select v-model="form.platform" style="width: 100%"><el-option v-for="item in availablePlatforms" :key="item.value" :label="item.label" :value="item.value" /></el-select></el-form-item>
              <el-form-item label="目标版本"><el-input v-model="form.app_version" maxlength="32" placeholder="留空表示全部版本" /></el-form-item>
              <el-form-item label="排序"><el-input-number v-model="form.sort" :min="0" :max="999999" controls-position="right" /></el-form-item>
            </div>
            <el-form-item label="展示区域">
              <div class="position-field">
                <el-checkbox-group v-model="form.display_position_ids" class="position-grid">
                  <el-checkbox v-for="item in positionOptions" :key="item.id" :value="item.id" class="position-card">
                    <el-image v-if="item.cover_image" :src="item.cover_image" fit="cover" class="position-cover" lazy />
                    <div v-else class="position-cover position-cover-empty">无封面</div>
                    <span class="position-name">{{ item.position_name }}</span>
                    <span class="position-key mono">{{ item.position_key }}</span>
                  </el-checkbox>
                </el-checkbox-group>
                <div class="form-tip position-tip">不选表示适用于全部展示区域</div>
              </div>
            </el-form-item>
            <el-form-item label="渠道"><el-select v-model="form.channel_ids" multiple filterable clearable placeholder="不选表示全部渠道" style="width: 100%" @change="handleChannelsChange"><el-option v-for="item in channelOptions" :key="item.channel_id" :label="channelLabel(item)" :value="item.channel_id" /></el-select></el-form-item>
            <el-form-item label="排除渠道"><el-select v-model="form.excluded_channel_ids" multiple filterable clearable placeholder="选择明确排除的渠道" style="width: 100%"><el-option v-for="item in channelOptions" :key="item.channel_id" :label="channelLabel(item)" :value="item.channel_id" :disabled="form.channel_ids.includes(item.channel_id)" /></el-select></el-form-item>
            <el-form-item label="套餐描述"><el-input v-model="form.description" type="textarea" :rows="3" maxlength="1000" show-word-limit /></el-form-item>
          </el-tab-pane>

          <el-tab-pane label="价格与权益" name="pricing">
            <div class="form-grid">
              <el-form-item label="币种" prop="currency"><el-input v-model="form.currency" maxlength="3" @input="form.currency = form.currency.toUpperCase()" /></el-form-item>
              <el-form-item label="划线金额"><el-input-number v-model="form.original_price" :min="0" :precision="2" :step="0.01" controls-position="right" /></el-form-item>
              <el-form-item label="首次订阅金额"><el-input-number v-model="form.first_subscription_price" :min="0" :precision="2" :step="0.01" controls-position="right" /></el-form-item>
              <el-form-item label="首次实际收入"><el-input-number v-model="form.first_subscription_revenue" :min="0" :precision="2" :step="0.01" controls-position="right" /></el-form-item>
              <el-form-item label="首次赠送积分"><el-input-number v-model="form.first_bonus_points" :min="0" controls-position="right" /></el-form-item>
              <el-form-item label="VIP 时长（天）"><el-input-number v-model="form.vip_duration_days" :min="0" controls-position="right" /></el-form-item>
              <el-form-item label="续订金额"><el-input-number v-model="form.subscription_price" :min="0" :precision="2" :step="0.01" controls-position="right" /></el-form-item>
              <el-form-item label="续订实际收入"><el-input-number v-model="form.subscription_revenue" :min="0" :precision="2" :step="0.01" controls-position="right" /></el-form-item>
              <el-form-item label="续订积分"><el-input-number v-model="form.subscription_points" :min="0" controls-position="right" /></el-form-item>
              <el-form-item label="订阅周期"><el-input v-model="form.subscription_period" maxlength="64" placeholder="例如：P1W、P1M、P1Y" /></el-form-item>
              <el-form-item label="试用天数"><el-input-number v-model="form.trial_days" :min="0" :max="3650" controls-position="right" /></el-form-item>
              <el-form-item label="免费体验"><el-switch v-model="form.free_trial" /></el-form-item>
            </div>
            <el-form-item label="订阅说明"><el-input v-model="form.subscription_description" type="textarea" :rows="3" maxlength="500" show-word-limit /></el-form-item>
          </el-tab-pane>

          <el-tab-pane label="展示与状态" name="presentation">
            <div class="form-grid">
              <el-form-item label="续费文案"><el-input v-model="form.renewal_text" maxlength="255" /></el-form-item>
              <el-form-item label="角标文案"><el-input v-model="form.badge_text" maxlength="64" placeholder="例如：最受欢迎、限时优惠" /></el-form-item>
              <el-form-item label="默认勾选协议"><el-switch v-model="form.agreement_default_checked" /></el-form-item>
              <el-form-item label="是否订阅"><el-switch v-model="form.is_subscription" /></el-form-item>
              <el-form-item label="显示模式"><el-radio-group v-model="form.display_mode"><el-radio :value="1">正常显示</el-radio><el-radio :value="0">隐藏</el-radio></el-radio-group></el-form-item>
              <el-form-item label="状态"><el-radio-group v-model="form.status"><el-radio :value="1">启用</el-radio><el-radio :value="0">禁用</el-radio></el-radio-group></el-form-item>
              <el-form-item label="默认套餐"><el-switch v-model="form.is_default" /><span class="form-tip">同一应用包和平台仅保留一个默认套餐</span></el-form-item>
            </div>
            <el-form-item label="内部备注"><el-input v-model="form.remark" type="textarea" :rows="4" maxlength="1000" show-word-limit /></el-form-item>
          </el-tab-pane>
        </el-tabs>
      </el-form>
      <template #footer><el-button @click="dialogVisible = false">取消</el-button><el-button type="primary" :loading="submitting" @click="handleSubmit">保存</el-button></template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox, type FormInstance, type FormRules } from 'element-plus'
import { getPackageOptions, type AppPackage } from '@/api/package'
import { getDisplayPositionOptions, type DisplayPosition } from '@/api/displayPosition'
import { getChannelOptions, type Channel } from '@/api/channel'
import { createVIPSubscription, deleteVIPSubscription, getVIPSubscriptionList, updateVIPSubscription, updateVIPSubscriptionDisplay, updateVIPSubscriptionStatus, setDefaultVIPSubscription, cloneVIPSubscription, type VIPSubscription, type VIPSubscriptionPayload } from '@/api/vipSubscription'
import { useUserStore } from '@/store/user'

const platformOptions = [{ value: 'android', label: 'Android' }, { value: 'ios', label: 'iOS' }, { value: 'pc', label: 'PC' }, { value: 'web', label: 'Web' }]
const planTypeOptions = [{ value: 'normal', label: '普通套餐' }, { value: 'trial', label: '体验套餐' }, { value: 'paywall', label: '支付页配置套餐' }]
const userStore = useUserStore()
const canAdd = computed(() => userStore.hasPermission('subscription:vip:add'))
const canEdit = computed(() => userStore.hasPermission('subscription:vip:edit'))
const canDelete = computed(() => userStore.hasPermission('subscription:vip:delete'))
const loading = ref(false), submitting = ref(false), dialogVisible = ref(false)
const updatingIds = ref<number[]>([]), formRef = ref<FormInstance>(), formTab = ref('base')
const tableData = ref<VIPSubscription[]>([]), packageOptions = ref<AppPackage[]>([]), positionOptions = ref<DisplayPosition[]>([]), channelOptions = ref<Channel[]>([])
const page = ref(1), pageSize = ref(20), total = ref(0)
const query = reactive({ plan_type: '', status: '', package_id: '', display_mode: '', is_subscription: '', platform: '', channel_id: '', excluded_channel_id: '', keyword: '' })
const defaultForm: VIPSubscriptionPayload & { id: number } = { id: 0, package_id: 0, platform: '', product_id: '', name: '', vip_level: '', plan_type: 'normal', display_position_ids: [], channel_ids: [], excluded_channel_ids: [], app_version: '', currency: 'USD', first_subscription_price: 0, first_subscription_revenue: 0, first_bonus_points: 0, original_price: 0, vip_duration_days: 30, trial_days: 0, renewal_text: '', badge_text: '', agreement_default_checked: true, display_mode: 1, status: 1, free_trial: false, is_subscription: true, is_default: false, subscription_description: '', subscription_price: 0, subscription_revenue: 0, subscription_points: 0, subscription_period: 'P1M', sort: 0, description: '', remark: '' }
const form = reactive({ ...defaultForm })
const rules: FormRules = { product_id: [{ required: true, message: '请输入产品 ID', trigger: 'blur' }], name: [{ required: true, message: '请输入 VIP 名称', trigger: 'blur' }], vip_level: [{ required: true, message: '请输入 VIP 等级', trigger: 'blur' }], plan_type: [{ required: true, message: '请选择套餐类型', trigger: 'change' }], package_id: [{ required: true, message: '请选择应用包', trigger: 'change' }], platform: [{ required: true, message: '请选择平台', trigger: 'change' }], currency: [{ required: true, pattern: /^[A-Za-z]{3}$/, message: '请输入三位币种代码', trigger: 'blur' }] }
const selectedPackage = computed(() => packageOptions.value.find((item) => item.id === form.package_id))
const availablePlatforms = computed(() => {
  const systemType = selectedPackage.value?.system_type
  const platform = systemType === 1 ? 'ios' : systemType === 2 ? 'android' : ''
  return platform ? platformOptions.filter((item) => item.value === platform) : platformOptions
})

async function fetchOptions() { const [packages, positions, channels]: any[] = await Promise.all([getPackageOptions(), getDisplayPositionOptions(), getChannelOptions()]); packageOptions.value = packages.data || []; positionOptions.value = positions.data || []; channelOptions.value = channels.data || [] }
async function fetchData() { loading.value = true; try { const params: Record<string, unknown> = { page: page.value, page_size: pageSize.value }; for (const [key, value] of Object.entries(query)) if (value !== '') params[key] = value; const res: any = await getVIPSubscriptionList(params); tableData.value = res.data.list || []; total.value = res.data.total || 0 } finally { loading.value = false } }
function handleSearch() { page.value = 1; fetchData() }
function handleReset() { Object.assign(query, { plan_type: '', status: '', package_id: '', display_mode: '', is_subscription: '', platform: '', channel_id: '', excluded_channel_id: '', keyword: '' }); page.value = 1; fetchData() }
function handlePageSizeChange() { page.value = 1; fetchData() }
function openCreate() { Object.assign(form, defaultForm, { display_position_ids: [], channel_ids: [], excluded_channel_ids: [], package_id: packageOptions.value.find((item) => item.status === 1)?.id || 0 }); handlePackageChange(); formTab.value = 'base'; dialogVisible.value = true }
function openEdit(row: VIPSubscription) {
  const values = Object.fromEntries(Object.keys(defaultForm).map((key) => [key, (row as unknown as Record<string, unknown>)[key] ?? (defaultForm as unknown as Record<string, unknown>)[key]]))
  Object.assign(form, values, {
    package_id: row.package_id || row.package?.id || row.packages?.[0]?.id || 0,
    display_position_ids: (row.display_positions || []).map((item) => item.id),
    channel_ids: (row.channels || []).map((item) => item.channel_id),
    excluded_channel_ids: (row.excluded_channels || []).map((item) => item.channel_id),
  })
  handlePackageChange(); formTab.value = 'base'; dialogVisible.value = true
}
function handlePackageChange() { if (form.platform && !availablePlatforms.value.some((item) => item.value === form.platform)) form.platform = ''; if (!form.platform && availablePlatforms.value.length) form.platform = availablePlatforms.value[0].value }
function handleChannelsChange() { form.excluded_channel_ids = form.excluded_channel_ids.filter((id) => !form.channel_ids.includes(id)) }
async function handleSubmit() { await formRef.value?.validate(); submitting.value = true; try { const payload = Object.fromEntries(Object.keys(defaultForm).filter((key) => key !== 'id').map((key) => [key, (form as unknown as Record<string, unknown>)[key]])) as unknown as VIPSubscriptionPayload; if (form.id) await updateVIPSubscription(form.id, payload); else await createVIPSubscription(payload); ElMessage.success('VIP 订阅套餐已保存'); dialogVisible.value = false; await fetchData() } finally { submitting.value = false } }
async function handleStatusChange(row: VIPSubscription) { updatingIds.value.push(row.id); try { await updateVIPSubscriptionStatus(row.id, row.status); ElMessage.success(`套餐已${row.status === 1 ? '启用' : '禁用'}`) } catch { row.status = row.status === 1 ? 0 : 1 } finally { updatingIds.value = updatingIds.value.filter((id) => id !== row.id) } }
async function toggleDisplay(row: VIPSubscription) { const mode = row.display_mode === 1 ? 0 : 1; await updateVIPSubscriptionDisplay(row.id, mode); row.display_mode = mode; ElMessage.success(mode === 1 ? '套餐已显示' : '套餐已隐藏') }
async function setDefault(row: VIPSubscription) { await setDefaultVIPSubscription(row.id); ElMessage.success('已设为默认套餐'); await fetchData() }
async function clonePlan(row: VIPSubscription) { const { value } = await ElMessageBox.prompt('请输入复制后套餐的新产品 ID（SKU）', '复制 VIP 套餐', { inputPlaceholder: `${row.product_id}_copy`, inputPattern: /\S+/, inputErrorMessage: '产品 ID 不能为空' }); await cloneVIPSubscription(row.id, value.trim()); ElMessage.success('套餐已复制'); await fetchData() }
async function handleDelete(id: number) { await deleteVIPSubscription(id); ElMessage.success('VIP 套餐已删除'); if (tableData.value.length === 1 && page.value > 1) page.value--; await fetchData() }
function packageLabel(item: AppPackage) { return `${item.package_name} · ${item.package_code}` }
function channelLabel(item: Channel) { return `${item.channel_name} · ${item.channel_code}` }
function channelNames(items?: Channel[]) { return items?.length ? items.map((item) => item.channel_name).join('、') : '全部' }
function planTypeLabel(value: string) { return planTypeOptions.find((item) => item.value === value)?.label || value }
function platformLabel(value: string) { return platformOptions.find((item) => item.value === value)?.label || value }
function formatMoney(value: number) { return Number(value || 0).toFixed(2) }
onMounted(() => Promise.all([fetchOptions(), fetchData()]))
</script>

<style scoped>
.page-wrap { min-width: 0; }.page-header { display: flex; align-items: center; justify-content: space-between; gap: 16px; }.page-title { color: #303133; font-size: 17px; font-weight: 600; }.page-subtitle { margin-top: 4px; color: #909399; font-size: 12px; }
.filters { display: grid; grid-template-columns: repeat(4, minmax(130px, 1fr)) repeat(4, minmax(130px, 1fr)) auto auto auto; gap: 10px; margin-bottom: 16px; }.primary-text { color: #303133; font-weight: 600; }.secondary-text { margin-top: 4px; color: #909399; font-size: 12px; }.mono { font-family: ui-monospace, SFMono-Regular, Menlo, monospace; }.tag-row,.target-tags { display: flex; flex-wrap: wrap; gap: 5px; margin-top: 6px; }.danger-text { color: #e6a23c; }.price { color: #f56c6c; font-size: 15px; font-weight: 600; }.original-price { color: #909399; font-size: 12px; text-decoration: line-through; }.display-state { margin-top: 6px; color: #909399; font-size: 12px; }.pagination-wrap { display: flex; justify-content: flex-end; margin-top: 16px; overflow-x: auto; }.form-grid { display: grid; grid-template-columns: 1fr 1fr; column-gap: 18px; padding-top: 8px; }.form-grid :deep(.el-input-number) { width: 100%; }.form-tip { margin-left: 10px; color: #909399; font-size: 12px; }
.position-field { width: 100%; }.position-grid { display: grid; grid-template-columns: repeat(5, minmax(0, 1fr)); gap: 10px; width: 100%; }.position-card { box-sizing: border-box; display: flex; height: auto; margin: 0; padding: 8px; border: 1px solid var(--el-border-color); border-radius: 6px; background: var(--el-fill-color-blank); }.position-card.is-checked { border-color: var(--el-color-primary); background: var(--el-color-primary-light-9); }.position-card :deep(.el-checkbox__label) { display: grid; min-width: 0; padding-left: 7px; }.position-cover { width: 100%; height: 62px; margin-bottom: 6px; border-radius: 4px; background: var(--el-fill-color-light); }.position-cover-empty { display: flex; align-items: center; justify-content: center; color: var(--el-text-color-placeholder); font-size: 12px; }.position-name,.position-key { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }.position-name { color: var(--el-text-color-primary); font-size: 12px; }.position-key { margin-top: 2px; color: var(--el-text-color-secondary); font-size: 10px; }.position-tip { margin: 8px 0 0; }
@media (max-width: 1200px) { .filters { grid-template-columns: repeat(4, minmax(130px, 1fr)); }.position-grid { grid-template-columns: repeat(4, minmax(0, 1fr)); } } @media (max-width: 700px) { .page-header { align-items: stretch; flex-direction: column; }.filters,.form-grid { grid-template-columns: 1fr; }.position-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); }.page-wrap :deep(.el-card__header),.page-wrap :deep(.el-card__body) { padding: 14px; } }
</style>
