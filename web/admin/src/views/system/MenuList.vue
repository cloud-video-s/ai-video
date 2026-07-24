<template>
  <div>
    <el-card>
      <template #header>
        <div style="display: flex; justify-content: space-between; align-items: center">
          <span>菜单管理</span>
          <el-button v-if="userStore.hasPermission('system:menu:add')" type="primary" @click="openDialog()">新增菜单</el-button>
        </div>
      </template>
      <el-table :data="treeData" v-loading="loading" row-key="id" :tree-props="{ children: 'children' }" stripe>
        <el-table-column prop="name" label="菜单名称" width="200" />
        <el-table-column prop="icon" label="图标" width="80">
          <template #default="{ row }">
            <el-icon v-if="row.icon"><component :is="row.icon" /></el-icon>
          </template>
        </el-table-column>
        <el-table-column prop="path" label="路由路径" />
        <el-table-column prop="component" label="组件路径" />
        <el-table-column prop="permission" label="权限标识" width="160" />
        <el-table-column label="关联接口" width="100" align="center">
          <template #default="{ row }">
            <el-tag v-if="row.apis?.length" size="small" type="info">{{ row.apis.length }} 个</el-tag>
            <span v-else class="empty-text">未关联</span>
          </template>
        </el-table-column>
        <el-table-column label="类型" width="80">
          <template #default="{ row }">
            <el-tag v-if="row.type === 0" type="warning">目录</el-tag>
            <el-tag v-else-if="row.type === 1">菜单</el-tag>
            <el-tag v-else type="info">按钮</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="sort" label="排序" width="70" />
        <el-table-column label="状态" width="80">
          <template #default="{ row }">
            <el-tag :type="row.status === 1 ? 'success' : 'danger'">{{ row.status === 1 ? '正常' : '禁用' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="220" fixed="right">
          <template #default="{ row }">
            <el-button v-if="userStore.hasPermission('system:menu:add')" link type="primary" @click="openDialog(undefined, row.id)">新增子项</el-button>
            <el-button v-if="userStore.hasPermission('system:menu:edit')" link type="primary" @click="openDialog(row)">编辑</el-button>
            <el-popconfirm v-if="userStore.hasPermission('system:menu:delete')" title="确认删除？" @confirm="handleDelete(row.id)">
              <template #reference>
                <el-button link type="danger">删除</el-button>
              </template>
            </el-popconfirm>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-dialog v-model="dialogVisible" :title="form.id ? '编辑菜单' : '新增菜单'" width="720px" destroy-on-close>
      <el-form ref="formRef" :model="form" :rules="rules" label-width="90px">
        <el-form-item label="上级菜单">
          <el-tree-select
            v-model="form.parent_id"
            :data="parentOptions"
            :props="{ label: 'name', value: 'id', children: 'children' }"
            check-strictly
            clearable
            placeholder="留空为顶级菜单"
            style="width: 100%"
          />
        </el-form-item>
        <el-form-item label="菜单类型">
          <el-radio-group v-model="form.type">
            <el-radio :value="0">目录</el-radio>
            <el-radio :value="1">菜单</el-radio>
            <el-radio :value="2">按钮</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="菜单名称" prop="name">
          <el-input v-model="form.name" />
        </el-form-item>
        <el-form-item label="路由路径">
          <el-input v-model="form.path" />
        </el-form-item>
        <el-form-item label="组件路径" v-if="form.type === 1">
          <el-input v-model="form.component" />
        </el-form-item>
        <el-form-item label="图标">
          <el-input v-model="form.icon" />
        </el-form-item>
        <el-form-item label="权限标识" v-if="form.type !== 0">
          <el-input v-model="form.permission" />
        </el-form-item>
        <el-form-item label="关联接口">
          <el-select
            v-model="form.api_ids"
            multiple
            filterable
            collapse-tags
            collapse-tags-tooltip
            clearable
            placeholder="选择该菜单或按钮允许访问的接口"
            style="width: 100%"
          >
            <el-option-group v-for="group in apiGroups" :key="group.name" :label="group.name">
              <el-option
                v-for="api in group.items"
                :key="api.id"
                :label="`${api.method} ${api.path}${api.description ? ` · ${api.description}` : ''}`"
                :value="api.id"
              />
            </el-option-group>
          </el-select>
          <div class="form-tip">角色勾选该菜单后，将自动获得这里绑定的接口权限。</div>
        </el-form-item>
        <el-form-item label="排序">
          <el-input-number v-model="form.sort" :min="0" />
        </el-form-item>
        <el-row>
          <el-col :span="12">
            <el-form-item label="显示状态">
              <el-radio-group v-model="form.visible">
                <el-radio :value="1">显示</el-radio>
                <el-radio :value="0">隐藏</el-radio>
              </el-radio-group>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="菜单状态">
              <el-radio-group v-model="form.status">
                <el-radio :value="1">正常</el-radio>
                <el-radio :value="0">禁用</el-radio>
              </el-radio-group>
            </el-form-item>
          </el-col>
        </el-row>
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
import { getAllAPIs, getMenuById, getMenuTree, createMenu, updateMenu, deleteMenu } from '@/api/menu'
import { useUserStore } from '@/store/user'

const userStore = useUserStore()

const loading = ref(false)
const submitLoading = ref(false)
const dialogVisible = ref(false)
const formRef = ref<FormInstance>()
const treeData = ref<any[]>([])
const apiOptions = ref<any[]>([])

const parentOptions = computed(() => {
  return [{ id: 0, name: '顶级菜单', children: excludeMenu(treeData.value, form.id) }]
})

const apiGroups = computed(() => {
  const groups = new Map<string, any[]>()
  for (const api of apiOptions.value) {
    const name = api.group || '其他'
    if (!groups.has(name)) groups.set(name, [])
    groups.get(name)!.push(api)
  }
  return [...groups.entries()].map(([name, items]) => ({ name, items }))
})

const defaultForm = {
  id: 0, parent_id: 0, name: '', path: '', component: '',
  icon: '', sort: 0, type: 1, permission: '', visible: 1, status: 1, api_ids: [] as number[],
}
const form = reactive({ ...defaultForm })

const rules = {
  name: [{ required: true, message: '请输入菜单名称', trigger: 'blur' }],
}

async function fetchData() {
  loading.value = true
  try {
    const res: any = await getMenuTree()
    treeData.value = res.data || []
  } finally {
    loading.value = false
  }
}

async function fetchAPIs() {
  const res: any = await getAllAPIs()
  apiOptions.value = res.data || []
}

function excludeMenu(items: any[], excludedId: number): any[] {
  return items
    .filter((item) => item.id !== excludedId)
    .map((item) => ({ ...item, children: excludeMenu(item.children || [], excludedId) }))
}

async function openDialog(row?: any, parentId?: number) {
  Object.assign(form, { ...defaultForm, api_ids: [] })
  if (row) {
    const res: any = await getMenuById(row.id)
    const item = res.data || row
    Object.assign(form, {
      id: item.id, parent_id: item.parent_id, name: item.name, path: item.path,
      component: item.component, icon: item.icon, sort: item.sort, type: item.type,
      permission: item.permission, visible: item.visible, status: item.status,
      api_ids: (item.apis || []).map((api: any) => api.id),
    })
  } else if (parentId !== undefined) {
    form.parent_id = parentId
  }
  dialogVisible.value = true
}

async function handleSubmit() {
  await formRef.value?.validate()
  submitLoading.value = true
  try {
    const payload = {
      parent_id: form.parent_id || 0,
      name: form.name.trim(),
      path: form.path.trim(),
      component: form.type === 1 ? form.component.trim() : '',
      icon: form.icon.trim(),
      sort: form.sort,
      type: form.type,
      permission: form.type === 0 ? '' : form.permission.trim(),
      visible: form.visible,
      status: form.status,
      api_ids: [...form.api_ids],
    }
    if (form.id) {
      await updateMenu(form.id, payload)
    } else {
      await createMenu(payload)
    }
    ElMessage.success('操作成功')
    dialogVisible.value = false
    fetchData()
  } finally {
    submitLoading.value = false
  }
}

async function handleDelete(id: number) {
  await deleteMenu(id)
  ElMessage.success('删除成功')
  fetchData()
}

onMounted(() => Promise.all([fetchData(), fetchAPIs()]))
</script>

<style scoped>
.empty-text { color: var(--el-text-color-placeholder); font-size: 12px; }
.form-tip { margin-top: 6px; color: var(--el-text-color-secondary); font-size: 12px; line-height: 1.5; }
</style>
