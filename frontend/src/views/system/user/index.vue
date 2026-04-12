<template>
  <div class="app-container">
    <!-- 搜索栏 -->
    <div class="filter-container">
      <el-input
        v-model="listQuery.username"
        :placeholder="t('user.please_enter_username')"
        style="width: 150px;"
        class="filter-item"
        @keyup.enter="handleFilter"
      />
      <el-input
        v-model="listQuery.email"
        :placeholder="t('user.please_enter_email')"
        style="width: 150px;"
        class="filter-item"
        @keyup.enter="handleFilter"
      />
      <el-select v-model="listQuery.status" :placeholder="t('user.status')" clearable style="width: 120px" class="filter-item">
        <el-option :label="t('common.enabled')" :value="1" />
        <el-option :label="t('common.disabled')" :value="0" />
      </el-select>
      <el-select v-model="listQuery.role_id" :placeholder="t('user.role')" clearable style="width: 120px" class="filter-item">
        <el-option
          v-for="role in roleList"
          :key="role.id"
          :label="role.name"
          :value="role.id"
        />
      </el-select>
      <el-button class="filter-item" type="primary" icon="Search" @click="handleFilter">
        {{ t('common.search') }}
      </el-button>
      <el-button class="filter-item" type="default" icon="Refresh" @click="resetFilter">
        {{ t('common.reset') }}
      </el-button>
      <el-button class="filter-item" style="margin-left: 10px;" type="primary" icon="Plus" @click="handleCreate">
        {{ t('user.add_user') }}
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
      
      <el-table-column :label="t('user.username')" prop="username">
        <template #default="{ row }">
          {{ row.username }}
          <el-tag v-if="row.is_super_admin" type="danger" size="small" effect="dark" style="margin-left: 8px;">
            {{ t('user.super_admin') }}
          </el-tag>
        </template>
      </el-table-column>
      
      <el-table-column :label="t('user.email')" prop="email" />
      
      <el-table-column :label="t('user.nickname')" prop="nickname" />
      
      <el-table-column :label="t('user.role')" width="120">
        <template #default="{ row }">
          <el-tag v-for="role in row.roles" :key="role.id" size="small" style="margin-right: 5px;">
            {{ role.name }}
          </el-tag>
        </template>
      </el-table-column>
      
      <el-table-column :label="t('user.status')" prop="status">
        <template #default="{ row }">
          <el-tag :type="row.status === 1 ? 'success' : 'danger'">
            {{ row.status === 1 ? t('common.enabled') : t('common.disabled') }}
          </el-tag>
        </template>
      </el-table-column>
      
      <el-table-column align="center" prop="created_at" :label="t('user.created_at')" width="160">
        <template #default="{ row }">
          <span>{{ formatDate(row.created_at) }}</span>
        </template>
      </el-table-column>
      
      <el-table-column align="center" :label="t('common.actions')" width="250">
        <template #default="{ row }">
          <!-- 如果是超级管理员，只显示标签 -->
          <template v-if="row.is_super_admin">
            <el-tag type="warning">{{ t('user.super_admin_account') }}</el-tag>
          </template>
          <!-- 非超级管理员显示操作按钮 -->
          <template v-else>
            <el-button type="primary" size="small" @click="handleUpdate(row)">
              {{ t('common.edit') }}
            </el-button>
            <el-button
              v-if="row.status === 1"
              size="small"
              type="warning"
              @click="handleModifyStatus(row, 0)"
            >
              {{ t('common.disabled') }}
            </el-button>
            <el-button
              v-else
              size="small"
              type="success"
              @click="handleModifyStatus(row, 1)"
            >
              {{ t('common.enabled') }}
            </el-button>
            <el-button size="small" type="danger" @click="handleDelete(row)">
              {{ t('common.delete') }}
            </el-button>
          </template>
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
      :title="dialogType === 'create' ? t('user.add_user') : t('user.edit_user')"
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
        <el-form-item :label="t('user.username')" prop="username">
          <el-input v-model="temp.username" :disabled="dialogType === 'update'" />
        </el-form-item>
        
        <el-form-item v-if="dialogType === 'create'" :label="t('user.password')" prop="password">
          <el-input v-model="temp.password" type="password" show-password />
        </el-form-item>
        
        <el-form-item :label="t('user.email')" prop="email">
          <el-input v-model="temp.email" />
        </el-form-item>
        
        <el-form-item :label="t('user.nickname')" prop="nickname">
          <el-input v-model="temp.nickname" />
        </el-form-item>
        
        <el-form-item :label="t('user.status')" prop="status">
          <el-select v-model="temp.status" :placeholder="t('common.please_select')">
            <el-option :label="t('common.enabled')" :value="1" />
            <el-option :label="t('common.disabled')" :value="0" />
          </el-select>
        </el-form-item>
        
        <el-form-item :label="t('user.role')" prop="role_ids">
          <el-select
            v-model="temp.role_ids"
            multiple
            :placeholder="t('user.please_select_role')"
            style="width: 100%"
          >
            <el-option
              v-for="role in roleList"
              :key="role.id"
              :label="role.name"
              :value="role.id"
              :disabled="role.code === 'admin' && dialogType === 'create'"
            />
          </el-select>
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

<script>
import { ref, reactive, nextTick, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { getUserList, createUser, updateUser, deleteUser, updateUserStatus, updateUserRoles } from '@/api/user'
import { getRoleList } from '@/api/role'
import Pagination from '@/components/Pagination/index.vue'
import dayjs from 'dayjs'
import { useI18n } from '@/composables/useI18n'

export default {
  name: 'UserManagement',
  components: {
    Pagination
  },
  setup() {
    const { t } = useI18n()
    const dataForm = ref()
    
    const list = ref([])
    const total = ref(0)
    const listLoading = ref(true)
    const dialogFormVisible = ref(false)
    const dialogType = ref('')
    const roleList = ref([])
    
    const listQuery = reactive({
      page: 1,
      limit: 20,
      username: '',
      email: '',
      status: '',
      role_id: ''
    })
    
    const temp = reactive({
      id: undefined,
      username: '',
      email: '',
      nickname: '',
      password: '',
      phone: '',
      status: 1,
      role_ids: []
    })
    
    const rules = {
      username: [{ required: true, message: t('validation.required', { s: t('user.username') }), trigger: 'blur' }],
      password: [{ required: true, message: t('validation.required', { s: t('user.password') }), trigger: 'blur' }],
      email: [
        { required: true, message: t('validation.required', { s: t('user.email') }), trigger: 'blur' },
        { type: 'email', message: t('validation.invalid_email'), trigger: 'blur' }
      ],
      role_ids: [{ required: true, message: t('validation.required', { s: t('user.role') }), trigger: 'change' }]
    }

    const getList = async () => {
      listLoading.value = true
      try {
        const params = {
          page: listQuery.page,
          page_size: listQuery.limit
        }
        
        // 添加搜索参数
        if (listQuery.username) params.username = listQuery.username
        if (listQuery.email) params.email = listQuery.email
        if (listQuery.status !== '') params.status = listQuery.status
        if (listQuery.role_id) params.role_id = listQuery.role_id
        
        const { data } = await getUserList(params)
        
        // 确保每个用户都有 is_super_admin 字段
        const userList = (data.items || []).map(user => ({
          ...user,
          is_super_admin: Boolean(user.is_super_admin)
        }))
        
        list.value = userList
        total.value = data.pagination?.total || 0
      } catch (error) {
        console.error('获取用户列表失败:', error)
        ElMessage.error('获取用户列表失败')
      } finally {
        listLoading.value = false
      }
    }

    const getRoles = async () => {
      try {
        const { data } = await getRoleList({ page: 1, page_size: 100 })
        roleList.value = data.items || []
      } catch (error) {
        console.error('获取角色列表失败:', error)
      }
    }

    const handleFilter = () => {
      listQuery.page = 1
      getList()
    }

    const resetFilter = () => {
      listQuery.username = ''
      listQuery.email = ''
      listQuery.status = ''
      listQuery.role_id = ''
      listQuery.page = 1
      getList()
    }

    const resetTemp = () => {
      temp.id = undefined
      temp.username = ''
      temp.password = ''
      temp.email = ''
      temp.nickname = ''
      temp.phone = ''
      temp.status = 1
      temp.role_ids = []
    }

    const handleCreate = () => {
      resetTemp()
      dialogType.value = 'create'
      dialogFormVisible.value = true
      nextTick(() => {
        dataForm.value.clearValidate()
      })
    }

    const createData = () => {
      dataForm.value.validate(async (valid) => {
        if (valid) {
          try {
            // 创建用户
            const { data } = await createUser(temp)
            // 分配角色
            if (temp.role_ids.length > 0) {
              await updateUserRoles(data.id, temp.role_ids)
            }
            dialogFormVisible.value = false
            ElMessage.success('创建成功')
            getList()
          } catch (error) {
            console.error('创建用户失败:', error)
            ElMessage.error('创建用户失败')
          }
        }
      })
    }

    const handleUpdate = (row) => {
      if (row.is_super_admin) {
        ElMessage.warning('超级管理员账户不能修改')
        return
      }
      Object.assign(temp, row)
      temp.role_ids = row.roles?.map(role => role.id) || []
      dialogType.value = 'update'
      dialogFormVisible.value = true
      nextTick(() => {
        dataForm.value.clearValidate()
      })
    }

    const updateData = () => {
      dataForm.value.validate(async (valid) => {
        if (valid) {
          try {
            const updateData = { ...temp }
            delete updateData.password // 移除密码字段
            // 更新用户基本信息
            await updateUser(temp.id, updateData)
            // 更新用户角色
            await updateUserRoles(temp.id, temp.role_ids)
            dialogFormVisible.value = false
            ElMessage.success('更新成功')
            getList()
          } catch (error) {
            console.error('更新用户失败:', error)
            ElMessage.error('更新用户失败')
          }
        }
      })
    }

    const handleDelete = (row) => {
      if (row.is_super_admin) {
        ElMessage.warning('超级管理员账户不能删除')
        return
      }
      
      ElMessageBox.confirm('此操作将永久删除该用户, 是否继续?', '提示', {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }).then(async () => {
        try {
          await deleteUser(row.id)
          const index = list.value.findIndex(v => v.id === row.id)
          list.value.splice(index, 1)
          ElMessage.success('删除成功')
          getList()
        } catch (error) {
          console.error('删除用户失败:', error)
          ElMessage.error(error.response?.data?.message || '删除用户失败')
        }
      })
    }

    const handleModifyStatus = async (row, status) => {
      if (row.is_super_admin) {
        ElMessage.warning('超级管理员账户状态不能修改')
        return
      }
      
      console.log('handleModifyStatus called:', { userId: row.id, newStatus: status, statusType: typeof status })
      
      try {
        await updateUserStatus(row.id, status)
        row.status = status
        ElMessage.success('状态更新成功')
        getList()
      } catch (error) {
        console.error('更新状态失败:', error)
        ElMessage.error(error.response?.data?.message || '更新状态失败')
      }
    }

    const formatDate = (date) => {
      if (!date) return ''
      return dayjs(date).format('YYYY-MM-DD HH:mm:ss')
    }

    onMounted(() => {
      getList()
      getRoles()
    })

    return {
      t,
      list,
      total,
      listLoading,
      listQuery,
      dialogFormVisible,
      dialogType,
      temp,
      rules,
      dataForm,
      roleList,
      getList,
      handleFilter,
      handleCreate,
      createData,
      handleUpdate,
      updateData,
      handleDelete,
      handleModifyStatus,
      formatDate,
      resetFilter
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