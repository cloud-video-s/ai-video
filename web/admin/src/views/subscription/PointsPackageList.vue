<template>
  <div class="page-wrap">
    <el-card shadow="never">
      <template #header>
        <div class="page-header">
          <div>
            <div class="page-title">积分套餐</div>
            <div class="page-subtitle">配置一次性积分商品、目标用户类型、适用系统及渠道</div>
          </div>
          <el-button v-if="canAdd" type="primary" @click="openCreate"><el-icon><Plus /></el-icon>新增套餐</el-button>
        </div>
      </template>

      <div class="filters">
        <el-select v-model="query.package_id" clearable filterable placeholder="安装包">
          <el-option v-for="item in packageOptions" :key="item.id" :label="packageLabel(item)" :value="String(item.id)" />
        </el-select>
        <el-select v-model="query.system" clearable placeholder="系统">
          <el-option v-for="item in allSystemOptions" :key="item" :label="systemLabel(item)" :value="item" />
        </el-select>
        <el-select v-model="query.user_type" clearable placeholder="用户类型">
          <el-option label="免费用户" value="1" /><el-option label="付费用户" value="2" />
        </el-select>
        <el-select v-model="query.channel_id" clearable filterable placeholder="渠道">
          <el-option v-for="item in channelOptions" :key="item.channel_id" :label="channelLabel(item)" :value="String(item.channel_id)" />
        </el-select>
        <el-select v-model="query.resource_type" clearable filterable allow-create placeholder="资源类型">
          <el-option v-for="item in resourceTypeOptions" :key="item.value" :label="item.label" :value="item.value" />
        </el-select>
        <el-select v-model="query.status" clearable placeholder="状态">
          <el-option label="启用" value="1" /><el-option label="禁用" value="0" />
        </el-select>
        <el-input v-model="query.keyword" clearable placeholder="产品 ID、名称、角标或描述" @keyup.enter="handleSearch">
          <template #prefix><el-icon><Search /></el-icon></template>
        </el-input>
        <el-button type="primary" plain @click="handleSearch">查询</el-button>
        <el-button @click="handleReset">重置</el-button>
      </div>

      <el-table v-loading="loading" :data="tableData" row-key="id" stripe>
        <el-table-column prop="id" label="套餐ID" width="80" />
        <el-table-column prop="sort" label="排序" width="68" align="center" />
        <el-table-column label="默认" width="78" align="center">
          <template #default="{ row }">
            <el-tag v-if="row.is_default" type="success">默认</el-tag>
            <el-button v-else-if="canEdit" link type="primary" @click="handleSetDefault(row.id)">设默认</el-button>
            <span v-else>-</span>
          </template>
        </el-table-column>
        <el-table-column label="商品" min-width="230">
          <template #default="{ row }"><div class="primary-text">{{ row.name }}</div><code class="product-id">{{ row.product_id }}</code></template>
        </el-table-column>
        <el-table-column label="安装包 / 系统" min-width="220">
          <template #default="{ row }">
            <div>{{ row.package ? row.package.package_name : `安装包 #${row.package_id}` }}</div>
            <div class="tag-list"><el-tag v-for="item in row.systems" :key="item" size="small" effect="plain">{{ systemLabel(item) }}</el-tag></div>
          </template>
        </el-table-column>
        <el-table-column label="用户 / 渠道" min-width="190">
          <template #default="{ row }">
            <div class="tag-list"><el-tag v-for="item in row.user_types" :key="item" size="small" type="warning" effect="plain">{{ userTypeLabel(item) }}</el-tag></div>
            <div class="secondary-text">{{ channelSummary(row.channels) }}</div>
          </template>
        </el-table-column>
        <el-table-column label="积分" width="135">
          <template #default="{ row }"><strong class="points-value">{{ formatNumber(row.points) }}</strong><div class="secondary-text">{{ resourceTypeLabel(row.resource_type) }}</div></template>
        </el-table-column>
        <el-table-column label="价格" width="165">
          <template #default="{ row }">
            <div class="price-line"><span>销售</span><strong>{{ money(row.sale_price, row.currency) }}</strong></div>
            <div class="price-line"><span>收入</span><strong>{{ money(row.actual_revenue, row.currency) }}</strong></div>
            <div v-if="row.original_price" class="original-price">{{ money(row.original_price, row.currency) }}</div>
          </template>
        </el-table-column>
        <el-table-column label="角标 / 按钮" min-width="150">
          <template #default="{ row }"><el-tag v-if="row.badge_text" size="small" type="danger">{{ row.badge_text }}</el-tag><div class="secondary-text">{{ row.button_text || '-' }}</div></template>
        </el-table-column>
        <el-table-column label="状态" width="90" align="center">
          <template #default="{ row }"><el-switch v-if="canEdit" v-model="row.status" :active-value="1" :inactive-value="0" @change="handleStatus(row)" /><el-tag v-else :type="row.status === 1 ? 'success' : 'info'">{{ row.status === 1 ? '启用' : '禁用' }}</el-tag></template>
        </el-table-column>
        <el-table-column v-if="canEdit || canDelete" label="操作" width="130" fixed="right" align="center">
          <template #default="{ row }">
            <el-button v-if="canEdit" link type="primary" @click="openEdit(row)">编辑</el-button>
            <el-popconfirm v-if="canDelete" :title="`确认删除 ${row.name}？`" @confirm="handleDelete(row.id)"><template #reference><el-button link type="danger">删除</el-button></template></el-popconfirm>
          </template>
        </el-table-column>
      </el-table>

      <div class="pagination-wrap"><el-pagination v-model:current-page="page" v-model:page-size="pageSize" :total="total" :page-sizes="[10,20,50,100]" layout="total, sizes, prev, pager, next" @size-change="handlePageSizeChange" @current-change="fetchData" /></div>
    </el-card>

    <el-dialog v-model="dialogVisible" :title="form.id ? '编辑积分套餐' : '新增积分套餐'" width="840px" destroy-on-close>
      <el-form ref="formRef" :model="form" :rules="rules" label-width="105px">
        <div class="form-grid">
          <el-form-item label="产品 ID" prop="product_id"><el-input v-model="form.product_id" maxlength="191" placeholder="例如：premium_credits_plan" /></el-form-item>
          <el-form-item label="积分名称" prop="name"><el-input v-model="form.name" maxlength="128" /></el-form-item>
          <el-form-item label="安装包" prop="package_id">
            <el-select v-model="form.package_id" filterable placeholder="请选择安装包" style="width:100%" @change="handlePackageChange"><el-option v-for="item in packageOptions" :key="item.id" :label="packageLabel(item)" :value="item.id" /></el-select>
          </el-form-item>
          <el-form-item label="系统" prop="systems"><el-select v-model="form.systems" multiple filterable placeholder="可多选" style="width:100%"><el-option v-for="item in formSystemOptions" :key="item" :label="systemLabel(item)" :value="item" /></el-select></el-form-item>
          <el-form-item label="用户类型" prop="user_types"><el-select v-model="form.user_types" multiple style="width:100%"><el-option label="免费用户" :value="1" /><el-option label="付费用户" :value="2" /></el-select></el-form-item>
          <el-form-item label="渠道"><el-select v-model="form.channel_ids" multiple filterable collapse-tags collapse-tags-tooltip clearable placeholder="留空表示全部渠道" style="width:100%"><el-option v-for="item in channelOptions" :key="item.channel_id" :label="channelLabel(item)" :value="item.channel_id" /></el-select></el-form-item>
          <el-form-item label="资源类型" prop="resource_type"><el-select v-model="form.resource_type" filterable allow-create style="width:100%"><el-option v-for="item in resourceTypeOptions" :key="item.value" :label="item.label" :value="item.value" /></el-select></el-form-item>
          <el-form-item label="赠送积分" prop="points"><el-input-number v-model="form.points" :min="1" :max="999999999999" controls-position="right" /></el-form-item>
          <el-form-item label="币种" prop="currency"><el-input v-model="form.currency" maxlength="3" @input="form.currency = form.currency.toUpperCase()" /></el-form-item>
          <el-form-item label="销售金额" prop="sale_price"><el-input-number v-model="form.sale_price" :min="0" :max="9999999999.99" :precision="2" controls-position="right" /></el-form-item>
          <el-form-item label="实际收入" prop="actual_revenue"><el-input-number v-model="form.actual_revenue" :min="0" :max="9999999999.99" :precision="2" controls-position="right" /></el-form-item>
          <el-form-item label="划线价" prop="original_price"><el-input-number v-model="form.original_price" :min="0" :max="9999999999.99" :precision="2" controls-position="right" /></el-form-item>
          <el-form-item label="角标"><el-input v-model="form.badge_text" maxlength="64" placeholder="例如：Most Popular" /></el-form-item>
          <el-form-item label="按钮文案"><el-input v-model="form.button_text" maxlength="128" placeholder="例如：获取更多积分" /></el-form-item>
          <el-form-item label="排序"><el-input-number v-model="form.sort" :min="0" :max="999999" controls-position="right" /></el-form-item>
          <el-form-item label="是否默认"><el-switch v-model="form.is_default" active-text="是" inactive-text="否" /></el-form-item>
          <el-form-item label="状态"><el-radio-group v-model="form.status"><el-radio :value="1">启用</el-radio><el-radio :value="0">禁用</el-radio></el-radio-group></el-form-item>
        </div>
        <el-form-item label="描述"><el-input v-model="form.description" type="textarea" :rows="3" maxlength="1000" show-word-limit /></el-form-item>
      </el-form>
      <template #footer><el-button @click="dialogVisible=false">取消</el-button><el-button type="primary" :loading="submitting" @click="handleSubmit">保存</el-button></template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, type FormInstance, type FormRules } from 'element-plus'
import { useUserStore } from '@/store/user'
import { getPackageOptions, type AppPackage } from '@/api/package'
import { getChannelOptions, type Channel } from '@/api/channel'
import { createPointsPackage, deletePointsPackage, getPointsPackageList, setDefaultPointsPackage, updatePointsPackage, updatePointsPackageStatus, type PointsPackage, type PointsPackagePayload } from '@/api/pointsPackage'

const userStore = useUserStore()
const canAdd = computed(() => userStore.hasPermission('subscription:points:add'))
const canEdit = computed(() => userStore.hasPermission('subscription:points:edit'))
const canDelete = computed(() => userStore.hasPermission('subscription:points:delete'))
const allSystemOptions = ['android','ios','pc','harmony','web']
const resourceTypeOptions = [{ value:'credits', label:'积分包' }, { value:'word_pack', label:'字数包' }, { value:'image_pack', label:'图片包' }]
const packageOptions = ref<AppPackage[]>([])
const channelOptions = ref<Channel[]>([])
const tableData = ref<PointsPackage[]>([])
const loading = ref(false), submitting = ref(false), dialogVisible = ref(false)
const formRef = ref<FormInstance>()
const page = ref(1), pageSize = ref(20), total = ref(0)
const query = reactive({ package_id:'', system:'', user_type:'', channel_id:'', resource_type:'', status:'', keyword:'' })
const defaultForm: PointsPackagePayload & { id:number } = { id:0, product_id:'', name:'', package_id:0, systems:[], user_types:[1,2], channel_ids:[], resource_type:'credits', points:1, currency:'USD', sale_price:0, actual_revenue:0, original_price:0, badge_text:'', description:'', button_text:'获取积分', is_default:false, status:1, sort:0 }
const form = reactive({ ...defaultForm })
const rules: FormRules = {
  product_id:[{required:true,message:'请输入产品 ID',trigger:'blur'},{pattern:/^[A-Za-z0-9._-]+$/,message:'仅支持字母、数字、点、下划线和中划线',trigger:'blur'}],
  name:[{required:true,message:'请输入积分名称',trigger:'blur'}], package_id:[{required:true,message:'请选择安装包',trigger:'change'}],
  systems:[{required:true,type:'array',min:1,message:'请至少选择一个系统',trigger:'change'}], user_types:[{required:true,type:'array',min:1,message:'请至少选择一种用户类型',trigger:'change'}],
  resource_type:[{required:true,message:'请选择资源类型',trigger:'change'}], points:[{required:true,type:'number',min:1,message:'赠送积分必须大于 0',trigger:'change'}],
  currency:[{required:true,pattern:/^[A-Za-z]{3}$/,message:'请输入 3 位币种代码',trigger:'blur'}],
}
const formSystemOptions = computed(() => packageOptions.value.find((item)=>item.id===form.package_id)?.system_types || allSystemOptions)

function packageLabel(item:AppPackage){ return `${item.package_name} · ${item.package_code} · ${item.package_version}` }
function channelLabel(item:Channel){ return `${item.channel_name} · ${item.channel_code}` }
function systemLabel(value:string){ return ({android:'Android',ios:'iOS',pc:'PC',harmony:'HarmonyOS',web:'Web'} as Record<string,string>)[value] || value }
function userTypeLabel(value:number){ return value===1?'免费用户':'付费用户' }
function resourceTypeLabel(value:string){ return resourceTypeOptions.find((item)=>item.value===value)?.label || value }
function channelSummary(items:Channel[]=[]){ return items.length ? items.map((item)=>item.channel_name).slice(0,2).join('、')+(items.length>2?` 等 ${items.length} 项`:'') : '全部渠道' }
function formatNumber(value:number){ return Number(value||0).toLocaleString('zh-CN') }
function money(value:number,currency:string){ return `${currency} ${Number(value||0).toFixed(2)}` }

async function fetchOptions(){ const [packages,channels]:any[]=await Promise.all([getPackageOptions(),getChannelOptions()]); packageOptions.value=packages.data||[]; channelOptions.value=channels.data||[] }
async function fetchData(){ loading.value=true; try { const params:Record<string,unknown>={page:page.value,page_size:pageSize.value}; for(const [key,value] of Object.entries(query)) if(value!=='') params[key]=typeof value==='string'?value.trim():value; const res:any=await getPointsPackageList(params); tableData.value=res.data.list||[]; total.value=res.data.total||0 } finally { loading.value=false } }
function handleSearch(){ page.value=1; fetchData() }
function handleReset(){ Object.assign(query,{package_id:'',system:'',user_type:'',channel_id:'',resource_type:'',status:'',keyword:''}); page.value=1; fetchData() }
function handlePageSizeChange(){ page.value=1; fetchData() }
function handlePackageChange(){ form.systems=form.systems.filter((item)=>formSystemOptions.value.includes(item)) }
function openCreate(){ Object.assign(form,defaultForm,{systems:[],user_types:[1,2],channel_ids:[]}); dialogVisible.value=true }
function openEdit(row:PointsPackage){ Object.assign(form,{id:row.id,product_id:row.product_id,name:row.name,package_id:row.package_id,systems:[...(row.systems||[])],user_types:[...(row.user_types||[1,2])],channel_ids:(row.channels||[]).map((item)=>item.channel_id),resource_type:row.resource_type,points:row.points,currency:row.currency,sale_price:Number(row.sale_price),actual_revenue:Number(row.actual_revenue),original_price:Number(row.original_price),badge_text:row.badge_text||'',description:row.description||'',button_text:row.button_text||'',is_default:row.is_default,status:row.status,sort:row.sort}); dialogVisible.value=true }
async function handleSubmit(){ await formRef.value?.validate(); submitting.value=true; try { const payload:PointsPackagePayload={product_id:form.product_id.trim(),name:form.name.trim(),package_id:form.package_id,systems:form.systems.map((v)=>v.toLowerCase()),user_types:[...form.user_types],channel_ids:[...form.channel_ids],resource_type:form.resource_type.trim().toLowerCase(),points:form.points,currency:form.currency.trim().toUpperCase(),sale_price:Number(form.sale_price),actual_revenue:Number(form.actual_revenue),original_price:Number(form.original_price),badge_text:form.badge_text.trim(),description:form.description.trim(),button_text:form.button_text.trim(),is_default:form.is_default,status:form.status,sort:form.sort}; if(form.id) await updatePointsPackage(form.id,payload); else await createPointsPackage(payload); ElMessage.success('积分套餐已保存'); dialogVisible.value=false; await fetchData() } finally { submitting.value=false } }
async function handleStatus(row:PointsPackage){ try { await updatePointsPackageStatus(row.id,row.status); ElMessage.success('状态已更新') } catch { row.status=row.status===1?0:1 } }
async function handleSetDefault(id:number){ await setDefaultPointsPackage(id); ElMessage.success('默认套餐已更新'); await fetchData() }
async function handleDelete(id:number){ await deletePointsPackage(id); ElMessage.success('积分套餐已删除'); if(tableData.value.length===1&&page.value>1) page.value--; await fetchData() }
onMounted(()=>Promise.all([fetchOptions(),fetchData()]))
</script>

<style scoped>
.page-wrap{min-width:0}.page-header{display:flex;align-items:center;justify-content:space-between;gap:16px}.page-title{color:#303133;font-size:17px;font-weight:600}.page-subtitle{margin-top:4px;color:#909399;font-size:12px}.filters{display:grid;grid-template-columns:repeat(auto-fit,minmax(145px,1fr));gap:10px;margin-bottom:16px}.primary-text{font-weight:600;color:#303133}.product-id{display:inline-block;margin-top:5px;padding:2px 7px;border-radius:4px;background:#f5f7fa;color:#606266}.tag-list{display:flex;flex-wrap:wrap;gap:5px;margin-top:5px}.secondary-text{margin-top:5px;color:#909399;font-size:12px}.points-value{font-size:16px;color:#409eff}.price-line{display:flex;justify-content:space-between;line-height:22px}.price-line span{color:#909399}.original-price{text-align:right;color:#a8abb2;font-size:12px;text-decoration:line-through}.pagination-wrap{display:flex;justify-content:flex-end;margin-top:16px;overflow-x:auto}.form-grid{display:grid;grid-template-columns:1fr 1fr;column-gap:16px}.form-grid :deep(.el-input-number){width:100%}@media(max-width:720px){.page-header{align-items:stretch;flex-direction:column}.form-grid{grid-template-columns:1fr}.page-wrap :deep(.el-card__header),.page-wrap :deep(.el-card__body){padding:14px}}
</style>
