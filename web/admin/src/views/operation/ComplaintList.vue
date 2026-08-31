<template>
  <div class="page-wrap">
    <el-card shadow="never">
      <template #header>
        <div class="page-heading">
          <div>
            <div class="page-title">投诉管理</div>
            <div class="page-subtitle">查看用户提交的模板投诉及关联用户、模板信息</div>
          </div>
          <el-tag effect="plain">共 {{ total.toLocaleString('zh-CN') }} 条</el-tag>
        </div>
      </template>

      <div class="filters">
        <el-input
          v-model="query.keyword"
          clearable
          placeholder="投诉内容、类型、用户或模板"
          @keyup.enter="handleSearch"
        >
          <template #prefix><el-icon><Search /></el-icon></template>
        </el-input>
        <el-input-number v-model="query.user_id" :min="1" :controls="false" placeholder="用户 ID" />
        <el-input-number v-model="query.template_id" :min="1" :controls="false" placeholder="模板 ID" />
        <el-input
          v-model="query.complaint_type"
          clearable
          placeholder="投诉类型（精确）"
          @keyup.enter="handleSearch"
        />
        <el-date-picker
          v-model="dateRange"
          type="daterange"
          value-format="YYYY-MM-DD"
          range-separator="至"
          start-placeholder="开始日期"
          end-placeholder="结束日期"
          style="width: 100%"
        />
        <div class="filter-actions">
          <el-button type="primary" plain @click="handleSearch">查询</el-button>
          <el-button @click="handleReset">重置</el-button>
        </div>
      </div>

      <el-table v-loading="loading" :data="tableData" row-key="id" stripe>
        <el-table-column label="投诉" min-width="260">
          <template #default="{ row }">
            <div class="complaint-title">
              <el-tag size="small" type="danger" effect="plain">{{ row.complaint_type }}</el-tag>
              <span class="secondary-inline">#{{ row.id }}</span>
            </div>
            <div class="content-preview" :class="{ empty: !row.content }">
              {{ row.content || '未填写补充内容' }}
            </div>
          </template>
        </el-table-column>

        <el-table-column label="投诉用户" min-width="210">
          <template #default="{ row }">
            <div class="relation-title">
              <span class="primary-text">{{ userName(row) }}</span>
              <el-tag v-if="row.user?.deleted" size="small" type="info">已删除</el-tag>
            </div>
            <div class="secondary-text">ID {{ row.user_id }} · {{ userAccount(row) }}</div>
          </template>
        </el-table-column>

        <el-table-column label="被投诉模板" min-width="240">
          <template #default="{ row }">
            <div class="template-cell">
              <el-image
                v-if="row.template?.cover_image_url"
                :src="toMediaURL(row.template.cover_image_url)"
                :preview-src-list="[toMediaURL(row.template.cover_image_url)]"
                preview-teleported
                fit="cover"
                class="template-cover"
              >
                <template #error><div class="cover-fallback"><el-icon><Picture /></el-icon></div></template>
              </el-image>
              <div v-else class="cover-fallback"><el-icon><Picture /></el-icon></div>
              <div class="template-copy">
                <div class="relation-title">
                  <span class="primary-text">{{ row.template?.name || `模板 #${row.template_id}` }}</span>
                  <el-tag v-if="row.template?.deleted" size="small" type="info">已删除</el-tag>
                </div>
                <div class="secondary-text">
                  ID {{ row.template_id }} · {{ templateTypeLabel(row.template?.template_type) }}
                </div>
              </div>
            </div>
          </template>
        </el-table-column>

        <el-table-column label="投诉时间" width="180">
          <template #default="{ row }">{{ formatDate(row.created_at) }}</template>
        </el-table-column>

        <el-table-column label="操作" width="80" fixed="right" align="center">
          <template #default="{ row }">
            <el-button link type="primary" @click="openDetail(row.id)">详情</el-button>
          </template>
        </el-table-column>

        <template #empty>
          <el-empty description="暂无符合条件的投诉记录" />
        </template>
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

    <el-drawer v-model="detailVisible" title="投诉详情" size="min(820px, 94vw)" destroy-on-close>
      <el-skeleton v-if="detailLoading" :rows="10" animated />
      <div v-else-if="detail" class="detail-wrap">
        <section>
          <h3>投诉信息</h3>
          <el-descriptions :column="2" border>
            <el-descriptions-item label="投诉 ID">{{ detail.id }}</el-descriptions-item>
            <el-descriptions-item label="投诉时间">{{ formatDate(detail.created_at) }}</el-descriptions-item>
            <el-descriptions-item label="投诉类型" :span="2">
              <el-tag size="small" type="danger" effect="plain">{{ detail.complaint_type }}</el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="更新时间" :span="2">{{ formatDate(detail.updated_at) }}</el-descriptions-item>
          </el-descriptions>
          <div class="section-label">投诉内容</div>
          <div class="content-block" :class="{ empty: !detail.content }">
            {{ detail.content || '未填写补充内容' }}
          </div>
        </section>

        <section>
          <h3>投诉用户</h3>
          <el-alert
            v-if="!detail.user"
            title="未找到关联用户，以下仅保留投诉记录中的用户 ID"
            type="warning"
            :closable="false"
          />
          <el-descriptions :column="2" border>
            <el-descriptions-item label="用户 ID">{{ detail.user_id }}</el-descriptions-item>
            <el-descriptions-item label="昵称">
              {{ detail.user?.username || '-' }}
              <el-tag v-if="detail.user?.deleted" class="inline-tag" size="small" type="info">已删除</el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="登录账号">{{ detail.user?.login_account || '-' }}</el-descriptions-item>
            <el-descriptions-item label="邮箱">{{ detail.user?.email || '-' }}</el-descriptions-item>
            <el-descriptions-item label="手机号">{{ detail.user?.phone || '-' }}</el-descriptions-item>
            <el-descriptions-item label="IMEI">{{ detail.user?.imei || '-' }}</el-descriptions-item>
            <el-descriptions-item label="设备编号" :span="2">{{ detail.user?.device_code || '-' }}</el-descriptions-item>
          </el-descriptions>
        </section>

        <section>
          <h3>被投诉模板</h3>
          <el-alert
            v-if="!detail.template"
            title="未找到关联模板，以下仅保留投诉记录中的模板 ID"
            type="warning"
            :closable="false"
          />
          <div class="template-detail">
            <el-image
              v-if="detail.template?.cover_image_url"
              :src="toMediaURL(detail.template.cover_image_url)"
              :preview-src-list="[toMediaURL(detail.template.cover_image_url)]"
              preview-teleported
              fit="cover"
              class="detail-cover"
            />
            <el-descriptions :column="2" border class="template-descriptions">
              <el-descriptions-item label="模板 ID">{{ detail.template_id }}</el-descriptions-item>
              <el-descriptions-item label="模板类型">{{ templateTypeLabel(detail.template?.template_type) }}</el-descriptions-item>
              <el-descriptions-item label="模板名称" :span="2">
                {{ detail.template?.name || '-' }}
                <el-tag v-if="detail.template?.deleted" class="inline-tag" size="small" type="info">已删除</el-tag>
              </el-descriptions-item>
              <el-descriptions-item label="当前状态" :span="2">
                {{ templateStatusLabel(detail.template?.status) }}
              </el-descriptions-item>
            </el-descriptions>
          </div>
        </section>
      </div>
    </el-drawer>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import {
  getTemplateComplaint,
  getTemplateComplaintList,
  type TemplateComplaint,
} from '@/api/templateComplaint'
import { toMediaURL } from '@/utils/mediaUrl'

const loading = ref(false)
const detailLoading = ref(false)
const detailVisible = ref(false)
const tableData = ref<TemplateComplaint[]>([])
const detail = ref<TemplateComplaint | null>(null)
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const dateRange = ref<string[]>([])
const query = reactive({
  keyword: '',
  user_id: undefined as number | undefined,
  template_id: undefined as number | undefined,
  complaint_type: '',
})

function userName(row: TemplateComplaint) {
  return row.user?.username || row.user?.email || row.user?.login_account || `用户 #${row.user_id}`
}

function userAccount(row: TemplateComplaint) {
  return row.user?.email || row.user?.phone || row.user?.login_account || row.user?.imei || '-'
}

function templateTypeLabel(value?: number) {
  if (value === 1) return '图片模板'
  if (value === 2) return '视频模板'
  return value ? `类型 ${value}` : '-'
}

function templateStatusLabel(value?: number) {
  if (value === 1) return '启用'
  if (value === 0) return '禁用'
  return '-'
}

function formatDate(value?: string | null) {
  if (!value) return '-'
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString('zh-CN', { hour12: false })
}

async function fetchData() {
  loading.value = true
  try {
    const params: Record<string, unknown> = { page: page.value, page_size: pageSize.value }
    for (const [key, value] of Object.entries(query)) {
      if (value !== '' && value !== undefined) {
        params[key] = typeof value === 'string' ? value.trim() : value
      }
    }
    if (dateRange.value.length === 2) {
      params.date_from = dateRange.value[0]
      params.date_to = dateRange.value[1]
    }
    const res: any = await getTemplateComplaintList(params)
    tableData.value = res.data.list || []
    total.value = res.data.total || 0
  } finally {
    loading.value = false
  }
}

function handleSearch() {
  page.value = 1
  fetchData()
}

function handlePageSizeChange() {
  page.value = 1
  fetchData()
}

function handleReset() {
  Object.assign(query, {
    keyword: '', user_id: undefined, template_id: undefined, complaint_type: '',
  })
  dateRange.value = []
  page.value = 1
  fetchData()
}

async function openDetail(id: number) {
  detailVisible.value = true
  detailLoading.value = true
  detail.value = null
  try {
    const res: any = await getTemplateComplaint(id)
    detail.value = res.data
  } finally {
    detailLoading.value = false
  }
}

onMounted(fetchData)
</script>

<style scoped>
.page-wrap { min-width: 0; }
.page-heading { display: flex; align-items: center; justify-content: space-between; gap: 16px; }
.page-title { color: #303133; font-size: 17px; font-weight: 600; }
.page-subtitle { margin-top: 4px; color: #909399; font-size: 12px; }
.filters { display: grid; grid-template-columns: minmax(240px, 1.7fr) repeat(3, minmax(140px, 1fr)) minmax(250px, 1.5fr) auto; gap: 10px; margin-bottom: 16px; }
.filters :deep(.el-input-number) { width: 100%; }
.filter-actions { display: flex; white-space: nowrap; }
.complaint-title, .relation-title { display: flex; align-items: center; gap: 7px; min-width: 0; }
.complaint-title { margin-bottom: 8px; }
.primary-text { min-width: 0; overflow: hidden; color: #303133; font-weight: 600; text-overflow: ellipsis; white-space: nowrap; }
.secondary-text { margin-top: 4px; color: #909399; font-size: 12px; }
.secondary-inline { color: #909399; font-size: 12px; }
.content-preview { display: -webkit-box; overflow: hidden; color: #606266; line-height: 1.55; overflow-wrap: anywhere; -webkit-box-orient: vertical; -webkit-line-clamp: 2; }
.empty { color: #a8abb2; }
.template-cell { display: flex; align-items: center; gap: 10px; min-width: 0; }
.template-cover, .cover-fallback { width: 54px; height: 54px; flex: 0 0 54px; border-radius: 7px; }
.cover-fallback { display: grid; place-items: center; background: #f2f4f7; color: #b2bac6; font-size: 20px; }
.template-copy { min-width: 0; }
.pagination-wrap { display: flex; justify-content: flex-end; margin-top: 16px; overflow-x: auto; }
.detail-wrap { padding: 0 4px 24px; }
.detail-wrap section + section { margin-top: 24px; }
.detail-wrap h3 { margin: 0 0 10px; color: #303133; font-size: 15px; }
.section-label { margin: 14px 0 8px; color: #606266; font-size: 14px; font-weight: 600; }
.content-block { min-height: 70px; padding: 14px; border: 1px solid #ebeef5; border-radius: 6px; background: #f8fafc; color: #303133; line-height: 1.7; white-space: pre-wrap; overflow-wrap: anywhere; }
.inline-tag { margin-left: 8px; }
.template-detail { display: flex; align-items: stretch; gap: 14px; }
.detail-cover { width: 150px; min-height: 150px; flex: 0 0 150px; border-radius: 7px; background: #f4f6f8; }
.template-descriptions { flex: 1; min-width: 0; }
@media (max-width: 1280px) {
  .filters { grid-template-columns: repeat(3, minmax(180px, 1fr)); }
}
@media (max-width: 760px) {
  .filters { grid-template-columns: 1fr; }
  .template-detail { flex-direction: column; }
  .detail-cover { width: 100%; max-height: 280px; }
  .detail-wrap :deep(.el-descriptions__body) { overflow-x: auto; }
  .page-wrap :deep(.el-card__header), .page-wrap :deep(.el-card__body) { padding: 14px; }
}
</style>
