<template>
  <div class="app-container">
    <!-- 搜索栏 -->
    <div class="filter-container">
      <el-input
        v-model="listQuery.keyword"
        :placeholder="t('role.please_enter_role_name')"
        style="width: 200px;"
        class="filter-item"
        @keyup.enter="handleFilter"
      />
      <el-button class="filter-item" type="primary" icon="Search" @click="handleFilter">
        {{ t('common.search') }}
      </el-button>
      <el-button class="filter-item" style="margin-left: 10px;" type="primary" icon="Plus" @click="handleCreate">
        {{ t('role.add_role') }}
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
    >
      <el-table-column align="center" label="ID" width="95">
        <template #default="{ row }">
          {{ row.id }}
        </template>
      </el-table-column>
      
      <el-table-column :label="t('role.role_name')" width="300">
        <template #default="{ row }">
          {{ row.name }}
        </template>
      </el-table-column>
      
      <el-table-column :label="t('role.role_code')" width="150">
        <template #default="{ row }">
          <el-tag>{{ row.code }}</el-tag>
        </template>
      </el-table-column>
      
      <el-table-column :label="t('role.description')" width="200">
        <template #default="{ row }">
          {{ row.description }}
        </template>
      </el-table-column>
      
      <el-table-column :label="t('role.user_count')" width="100" align="center">
        <template #default="{ row }">
          <el-tag type="info">{{ row.user_count || 0 }}</el-tag>
        </template>
      </el-table-column>
      
      <el-table-column align="center" prop="created_at" :label="t('common.created_at')" width="160">
        <template #default="{ row }">
          <span>{{ formatDate(row.created_at) }}</span>
        </template>
      </el-table-column>
      
      <el-table-column align="center" :label="t('common.actions')" width="200">
        <template #default="{ row }">
          <el-button 
            type="primary" 
            size="small" 
            @click="handleUpdate(row)"
            :disabled="row.code === 'admin'"
          >
            {{ t('common.edit') }}
          </el-button>
          <el-button 
            type="warning" 
            size="small" 
            @click="handlePermission(row)"
            :disabled="row.code === 'admin'"
          >
            {{ t('role.permissions') }}
          </el-button>
          <el-button 
            size="small" 
            type="danger" 
            @click="handleDelete(row)"
            :disabled="row.code === 'admin'"
          >
            {{ t('common.delete') }}
          </el-button>
        </template>
      </el-table-column>
    </el-table>

    <!-- 分页 -->
    <pagination
      v-show="total > 0"
      :total="total"
      :page.sync="listQuery.page"
      :limit.sync="listQuery.limit"
      @pagination="getList"
    />

    <!-- 添加/编辑对话框 -->
    <el-dialog
      :title="dialogType === 'create' ? t('role.add_role') : t('role.edit_role')"
      v-model="dialogFormVisible"
      width="600px"
    >
      <el-form
        ref="dataForm"
        :rules="rules"
        :model="temp"
        label-position="left"
        label-width="100px"
        style="width: 400px; margin-left:50px;"
      >
        <el-form-item :label="t('role.role_name')" prop="name">
          <el-input v-model="temp.name" />
        </el-form-item>
        
        <el-form-item :label="t('role.role_code')" prop="code">
          <el-input v-model="temp.code" :disabled="dialogType === 'update'" />
        </el-form-item>
        
        <el-form-item :label="t('role.description')" prop="description">
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

    <!-- 权限分配对话框 -->
    <el-dialog
      title="分配权限"
      v-model="permissionDialogVisible"
      width="800px"
    >
      <el-tree
        ref="permissionTree"
        :data="permissionTreeData"
        :default-checked-keys="checkedPermissions"
        node-key="id"
        show-checkbox
        default-expand-all
        :props="{ children: 'children', label: 'name' }"
      />
      
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="permissionDialogVisible = false">
            取消
          </el-button>
          <el-button type="primary" @click="handleSavePermissions">
            确认
          </el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script>
import { getRoleList, createRole, updateRole, deleteRole, getRoleDetail } from '@/api/role'
import { getPermissionTree, updateRolePermissions } from '@/api/permission'
import Pagination from '@/components/Pagination/index.vue'
import dayjs from 'dayjs'
import { useI18n } from '@/composables/useI18n'

export default {
  name: 'RoleManagement',
  components: {
    Pagination
  },
  setup() {
    const { t } = useI18n()
    const dataForm = ref()
    const permissionTree = ref()
    
    const list = ref([])
    const total = ref(0)
    const listLoading = ref(true)
    const dialogFormVisible = ref(false)
    const permissionDialogVisible = ref(false)
    const dialogType = ref('')
    const currentRole = ref(null)
    const permissionTreeData = ref([])
    const checkedPermissions = ref([])
    
    const listQuery = reactive({
      page: 1,
      limit: 20,
      keyword: ''
    })
    
    const temp = reactive({
      id: undefined,
      name: '',
      code: '',
      description: ''
    })

    const rules = {
      name: [{ required: true, message: t('validation.required', { s: t('role.role_name') }), trigger: 'blur' }],
      code: [{ required: true, message: t('validation.required', { s: t('role.role_code') }), trigger: 'blur' }]
    }

    const getList = async () => {
      listLoading.value = true
      try {
        const { data } = await getRoleList(listQuery)
        list.value = data.items || []
        total.value = data.total || 0
      } catch (error) {
        console.error('Failed to get role list:', error)
      } finally {
        listLoading.value = false
      }
    }

    const handleFilter = () => {
      listQuery.page = 1
      getList()
    }

    const resetTemp = () => {
      temp.id = undefined
      temp.name = ''
      temp.code = ''
      temp.description = ''
    }

    const handleCreate = () => {
      resetTemp()
      dialogType.value = 'create'
      dialogFormVisible.value = true
      nextTick(() => {
        dataForm.value.clearValidate()
      })
    }

    const createData = async () => {
      await dataForm.value.validate(async (valid) => {
        if (valid) {
          try {
            await createRole(temp)
            dialogFormVisible.value = false
            ElMessage.success('创建成功')
            getList()
          } catch (error) {
            console.error('Failed to create role:', error)
          }
        }
      })
    }

    const handleUpdate = (row) => {
      Object.assign(temp, row)
      dialogType.value = 'update'
      dialogFormVisible.value = true
      nextTick(() => {
        dataForm.value.clearValidate()
      })
    }

    const updateData = async () => {
      await dataForm.value.validate(async (valid) => {
        if (valid) {
          try {
            await updateRole(temp.id, temp)
            const index = list.value.findIndex(v => v.id === temp.id)
            list.value.splice(index, 1, { ...temp })
            dialogFormVisible.value = false
            ElMessage.success('更新成功')
          } catch (error) {
            console.error('Failed to update role:', error)
          }
        }
      })
    }

    const handleDelete = async (row) => {
      // 超级管理员不允许删除
      if (row.code === 'admin') {
        ElMessage.warning('超级管理员角色不允许删除')
        return
      }
      
      await ElMessageBox.confirm('此操作将永久删除该角色, 是否继续?', '提示', {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      })
      
      try {
        await deleteRole(row.id)
        const index = list.value.findIndex(v => v.id === row.id)
        list.value.splice(index, 1)
        ElMessage.success('删除成功')
      } catch (error) {
        console.error('Failed to delete role:', error)
      }
    }

    const handlePermission = async (row) => {
      // 超级管理员不允许修改权限
      if (row.code === 'admin') {
        ElMessage.warning('超级管理员默认拥有所有权限，无需修改')
        return
      }
      
      currentRole.value = row
      try {
        // 获取权限树
        const { data: treeData } = await getPermissionTree()
        permissionTreeData.value = treeData
        
        // 获取角色已有权限
        const { data: roleData } = await getRoleDetail(row.id)
        checkedPermissions.value = roleData.permissions?.map(p => p.id) || []
        
        // 重置树的选中状态
        nextTick(() => {
          if (permissionTree.value) {
            permissionTree.value.setCheckedKeys([])
            permissionTree.value.setCheckedKeys(checkedPermissions.value)
          }
        })
        
        permissionDialogVisible.value = true
      } catch (error) {
        console.error('Failed to get permissions:', error)
      }
    }

    const handleSavePermissions = async () => {
      try {
        const checkedNodes = permissionTree.value.getCheckedNodes(false, true) // 只获取叶子节点
        const halfCheckedNodes = permissionTree.value.getHalfCheckedNodes()
        
        // 过滤并合并权限节点
        const permissionIds = [
          ...checkedNodes.filter(node => node.id && node.action), // 确保是权限节点（有action属性）
          ...halfCheckedNodes.filter(node => node.id && node.action)
        ].map(node => node.id)
        
        // 确保有权限被选中
        if (permissionIds.length === 0) {
          ElMessage.warning('请至少选择一个权限')
          return
        }
        
        await updateRolePermissions(currentRole.value.id, { permission_ids: permissionIds })
        
        // 更新当前角色的权限数据
        const { data: roleData } = await getRoleDetail(currentRole.value.id)
        const index = list.value.findIndex(v => v.id === currentRole.value.id)
        if (index !== -1) {
          list.value[index] = roleData
        }
        
        permissionDialogVisible.value = false
        ElMessage.success('权限更新成功')
      } catch (error) {
        console.error('Failed to update permissions:', error)
      }
    }

    const formatDate = (date) => {
      return dayjs(date).format('YYYY-MM-DD HH:mm:ss')
    }

    onMounted(() => {
      getList()
    })

    return {
      dataForm,
      permissionTree,
      list,
      total,
      listLoading,
      dialogFormVisible,
      permissionDialogVisible,
      dialogType,
      currentRole,
      permissionTreeData,
      checkedPermissions,
      listQuery,
      temp,
      rules,
      getList,
      handleFilter,
      handleCreate,
      createData,
      handleUpdate,
      updateData,
      handleDelete,
      handlePermission,
      handleSavePermissions,
      formatDate
    }
  }
}
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