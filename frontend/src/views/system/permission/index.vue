<template>
  <div class="app-container">
    <!-- 搜索栏 -->
    <div class="filter-container">
      <el-input
        v-model="listQuery.keyword"
        :placeholder="t('permission.please_enter_permission_name')"
        style="width: 200px;"
        class="filter-item"
        @keyup.enter="handleFilter"
      />
      <el-button class="filter-item" type="primary" icon="Search" @click="handleFilter">
        {{ t('common.search') }}
      </el-button>
      <el-button class="filter-item" style="margin-left: 10px;" type="primary" icon="Plus" @click="handleCreate">
        {{ t('permission.add_permission') }}
      </el-button>
    </div>

    <!-- 表格 -->
    <el-table
      v-loading="listLoading"
      :data="list"
      :element-loading-text="t('common.loading')"
      border
      fit
      highlight-current-row
      row-key="id"
    >
      <el-table-column :label="t('permission.permission_name')" min-width="150">
        <template #default="{ row }">
          <span>{{ row.name }}</span>
        </template>
      </el-table-column>
      
      <el-table-column :label="t('permission.display_name')" min-width="150">
        <template #default="{ row }">
          <span>{{ row.display_name }}</span>
        </template>
      </el-table-column>
      
      <el-table-column :label="t('permission.module')" width="120">
        <template #default="{ row }">
          <el-tag type="info">{{ row.module }}</el-tag>
        </template>
      </el-table-column>
      
      <el-table-column :label="t('permission.action_type')" width="100">
        <template #default="{ row }">
          <el-tag :type="getActionTagType(row.action)">{{ row.action }}</el-tag>
        </template>
      </el-table-column>
      
      <el-table-column :label="t('permission.resource_type')" width="120">
        <template #default="{ row }">
          <span>{{ row.resource }}</span>
        </template>
      </el-table-column>
      
      <el-table-column :label="t('permission.status')" width="80" align="center">
        <template #default="{ row }">
          <el-tag :type="row.status === 1 ? 'success' : 'danger'">
            {{ row.status === 1 ? t('common.enabled') : t('common.disabled') }}
          </el-tag>
        </template>
      </el-table-column>
      
      <el-table-column :label="t('permission.description')" min-width="200">
        <template #default="{ row }">
          <span>{{ row.description }}</span>
        </template>
      </el-table-column>
      
      <el-table-column align="center" :label="t('common.actions')" width="150">
        <template #default="{ row }">
          <el-button type="primary" size="small" @click="handleUpdate(row)">
            {{ t('common.edit') }}
          </el-button>
          <el-button size="small" type="danger" @click="handleDelete(row)">
            {{ t('common.delete') }}
          </el-button>
        </template>
      </el-table-column>
    </el-table>

    <!-- 添加/编辑对话框 -->
    <el-dialog
      :title="dialogType === 'create' ? t('permission.add_permission') : t('permission.edit_permission')"
      v-model="dialogFormVisible"
      width="600px"
    >
      <el-form
        ref="dataForm"
        :rules="rules"
        :model="temp"
        label-position="left"
        label-width="100px"
        style="width: 500px; margin-left:20px;"
      >
        <el-form-item :label="t('permission.permission_name')" prop="name">
          <el-input v-model="temp.name" :placeholder="t('permission.permission_name_example')" />
        </el-form-item>
        
        <el-form-item :label="t('permission.display_name')" prop="display_name">
          <el-input v-model="temp.display_name" :placeholder="t('permission.display_name_example')" />
        </el-form-item>
        
        <el-form-item :label="t('permission.module')" prop="module">
          <el-input v-model="temp.module" :placeholder="t('permission.module_example')" />
        </el-form-item>
        
        <el-form-item :label="t('permission.action_type')" prop="action">
          <el-select v-model="temp.action" :placeholder="t('common.please_select')" style="width: 100%">
            <el-option :label="t('permission.action_view')" value="view" />
            <el-option :label="t('permission.action_create')" value="create" />
            <el-option :label="t('permission.action_edit')" value="edit" />
            <el-option :label="t('permission.action_delete')" value="delete" />
            <el-option :label="t('permission.action_import')" value="import" />
            <el-option :label="t('permission.action_export')" value="export" />
          </el-select>
        </el-form-item>
        
        <el-form-item :label="t('permission.resource_type')" prop="resource">
          <el-input v-model="temp.resource" :placeholder="t('permission.resource_type_example')" />
        </el-form-item>
        
        <el-form-item :label="t('permission.status')" prop="status">
          <el-radio-group v-model="temp.status">
            <el-radio :label="1">{{ t('common.enabled') }}</el-radio>
            <el-radio :label="0">{{ t('common.disabled') }}</el-radio>
          </el-radio-group>
        </el-form-item>
        
        <el-form-item :label="t('permission.description')" prop="description">
          <el-input v-model="temp.description" type="textarea" rows="3" />
        </el-form-item>
      </el-form>
      
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="dialogFormVisible = false">
            {{ t('common.cancel') }}
          </el-button>
          <el-button type="primary" @click="dialogType === 'create' ? createData() : updateData()">
            {{ t('common.confirm') }}
          </el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, computed, nextTick } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { getPermissionList, createPermission, updatePermission, deletePermission } from '@/api/permission'
import { useI18n } from '@/composables/useI18n'

const { t } = useI18n()

const dataForm = ref()
const list = ref([])
const listLoading = ref(true)
const dialogFormVisible = ref(false)
const dialogType = ref('create')

const listQuery = reactive({
  page: 1,
  limit: 20,
  keyword: ''
})

const temp = reactive({
  id: undefined,
  name: '',
  display_name: '',
  description: '',
  module: '',
  action: 'view',
  resource: '',
  status: 1
})

const rules = {
  name: [{ required: true, message: t('validation.required', { s: t('permission.permission_name') }), trigger: 'blur' }],
  display_name: [{ required: true, message: t('validation.required', { s: t('permission.display_name') }), trigger: 'blur' }],
  module: [{ required: true, message: t('validation.required', { s: t('permission.module') }), trigger: 'blur' }],
  action: [{ required: true, message: t('validation.required', { s: t('permission.action_type') }), trigger: 'blur' }],
  resource: [{ required: true, message: t('validation.required', { s: t('permission.resource_type') }), trigger: 'blur' }]
}

const dialogTitle = computed(() => {
  return dialogType.value === 'create' ? t('permission.add_permission') : t('permission.edit_permission')
})

// 获取操作类型对应的标签类型
const getActionTagType = (action) => {
  const types = {
    view: '',
    create: 'success',
    edit: 'warning',
    delete: 'danger',
    import: 'info',
    export: 'info'
  }
  return types[action] || ''
}

const getList = async () => {
  listLoading.value = true
  try {
    const { data } = await getPermissionList(listQuery)
    list.value = data.items
  } catch (error) {
    console.error('获取权限列表失败:', error)
  } finally {
    listLoading.value = false
  }
}

const resetTemp = () => {
  Object.assign(temp, {
    id: undefined,
    name: '',
    display_name: '',
    description: '',
    module: '',
    action: 'view',
    resource: '',
    status: 1
  })
}

const handleCreate = () => {
  resetTemp()
  dialogType.value = 'create'
  dialogFormVisible.value = true
  nextTick(() => {
    dataForm.value?.clearValidate()
  })
}

const createData = async () => {
  try {
    await dataForm.value?.validate()
    await createPermission(temp)
    dialogFormVisible.value = false
    ElMessage({
      message: '创建成功',
      type: 'success'
    })
    getList()
  } catch (error) {
    console.error('创建权限失败:', error)
  }
}

const handleUpdate = (row) => {
  Object.assign(temp, row)
  dialogType.value = 'update'
  dialogFormVisible.value = true
  nextTick(() => {
    dataForm.value?.clearValidate()
  })
}

const updateData = async () => {
  try {
    await dataForm.value?.validate()
    await updatePermission(temp.id, temp)
    dialogFormVisible.value = false
    ElMessage({
      message: '更新成功',
      type: 'success'
    })
    getList()
  } catch (error) {
    console.error('更新权限失败:', error)
  }
}

const handleDelete = (row) => {
  ElMessageBox.confirm(
    '确定要删除这个权限吗？',
    '警告',
    {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning'
    }
  ).then(async () => {
    try {
      await deletePermission(row.id)
      ElMessage({
        message: '删除成功',
        type: 'success'
      })
      getList()
    } catch (error) {
      console.error('删除权限失败:', error)
    }
  })
}

const handleFilter = () => {
  getList()
}

// 初始化
getList()
</script>

<style lang="scss" scoped>
.app-container {
  padding: 20px;
}

.filter-container {
  padding-bottom: 10px;
  
  .filter-item {
    display: inline-block;
    vertical-align: middle;
    margin-bottom: 10px;
    margin-right: 10px;
  }
}
</style> 