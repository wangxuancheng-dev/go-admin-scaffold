<template>
  <div class="dashboard">
    <div class="dashboard-header">
      <h1>{{ t('dashboard.welcome') }}</h1>
      <p>{{ t('dashboard.welcome_description') }}</p>
    </div>

    <!-- 统计卡片 -->
    <el-row :gutter="20" class="stats-row">
      <el-col :xs="24" :sm="12" :lg="6">
        <div class="stats-card">
          <div class="stats-icon user">
            <el-icon><User /></el-icon>
          </div>
          <div class="stats-content">
            <div class="stats-number">{{ stats.users }}</div>
            <div class="stats-label">{{ t('dashboard.total_users') }}</div>
          </div>
        </div>
      </el-col>
      
      <el-col :xs="24" :sm="12" :lg="6">
        <div class="stats-card">
          <div class="stats-icon role">
            <el-icon><UserFilled /></el-icon>
          </div>
          <div class="stats-content">
            <div class="stats-number">{{ stats.roles }}</div>
            <div class="stats-label">{{ t('dashboard.total_roles') }}</div>
          </div>
        </div>
      </el-col>
      
      <el-col :xs="24" :sm="12" :lg="6">
        <div class="stats-card">
          <div class="stats-icon login">
            <el-icon><Key /></el-icon>
          </div>
          <div class="stats-content">
            <div class="stats-number">{{ stats.logins }}</div>
            <div class="stats-label">{{ t('dashboard.total_logins') }}</div>
          </div>
        </div>
      </el-col>
      
      <el-col :xs="24" :sm="12" :lg="6">
        <div class="stats-card">
          <div class="stats-icon operation">
            <el-icon><Document /></el-icon>
          </div>
          <div class="stats-content">
            <div class="stats-number">{{ stats.operations }}</div>
            <div class="stats-label">{{ t('dashboard.total_operations') }}</div>
          </div>
        </div>
      </el-col>
    </el-row>

    <!-- 快捷入口 -->
    <div class="quick-actions">
      <h2>{{ t('dashboard.quick_actions') }}</h2>
      <el-row :gutter="20">
        <el-col :xs="24" :sm="12" :lg="6">
          <div class="action-card" @click="$router.push('/system/user')">
            <el-icon><User /></el-icon>
            <span>{{ t('user.user_management') }}</span>
          </div>
        </el-col>
        
        <el-col :xs="24" :sm="12" :lg="6">
          <div class="action-card" @click="$router.push('/system/role')">
            <el-icon><UserFilled /></el-icon>
            <span>{{ t('role.role_management') }}</span>
          </div>
        </el-col>
        
        <el-col :xs="24" :sm="12" :lg="6">
          <div class="action-card" @click="$router.push('/log/login')">
            <el-icon><Key /></el-icon>
            <span>{{ t('log.login_log') }}</span>
          </div>
        </el-col>
        
        <el-col :xs="24" :sm="12" :lg="6">
          <div class="action-card" @click="$router.push('/log/operation')">
            <el-icon><Document /></el-icon>
            <span>{{ t('log.operation_log') }}</span>
          </div>
        </el-col>
      </el-row>
    </div>

    <!-- 系统信息 -->
    <div class="system-info">
      <h2>{{ t('dashboard.system_info') }}</h2>
      <el-row :gutter="20">
        <el-col :span="12">
          <el-card>
            <template #header>
              <span>{{ t('dashboard.server_info') }}</span>
            </template>
            <div class="info-item">
              <span>{{ t('dashboard.os') }}:</span>
              <span>{{ systemInfo.os }}</span>
            </div>
            <div class="info-item">
              <span>{{ t('dashboard.go_version') }}:</span>
              <span>{{ systemInfo.goVersion }}</span>
            </div>
            <div class="info-item">
              <span>{{ t('dashboard.uptime') }}:</span>
              <span>{{ systemInfo.uptime }}</span>
            </div>
            <div class="info-item">
              <span>{{ t('dashboard.memory_usage') }}:</span>
              <span>{{ systemInfo.memory }}</span>
            </div>
          </el-card>
        </el-col>
        
        <el-col :span="12">
          <el-card>
            <template #header>
              <span>{{ t('dashboard.project_info') }}</span>
            </template>
            <div class="info-item">
              <span>{{ t('dashboard.project_name') }}:</span>
              <span>Go Admin</span>
            </div>
            <div class="info-item">
              <span>{{ t('dashboard.project_version') }}:</span>
              <span>v1.0.0</span>
            </div>
            <div class="info-item">
              <span>{{ t('dashboard.tech_stack') }}:</span>
              <span>Go + Gin + Vue3 + Element Plus</span>
            </div>
            <div class="info-item">
              <span>{{ t('dashboard.update_time') }}:</span>
              <span>{{ new Date().toLocaleDateString() }}</span>
            </div>
          </el-card>
        </el-col>
      </el-row>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { User, UserFilled, Key, Document } from '@element-plus/icons-vue'
import { useI18n } from '@/composables/useI18n'

const { t } = useI18n()

const stats = reactive({
  users: 0,
  roles: 0,
  logins: 0,
  operations: 0
})

const systemInfo = reactive({
  os: 'Linux',
  goVersion: 'Go 1.21',
  uptime: '15天 6小时',
  memory: '256MB / 2GB'
})

// 模拟加载统计数据
const loadStats = () => {
  // 这里应该调用实际的API
  setTimeout(() => {
    stats.users = 128
    stats.roles = 8
    stats.logins = 45
    stats.operations = 156
  }, 1000)
}

onMounted(() => {
  loadStats()
})
</script>

<style lang="scss" scoped>
.dashboard {
  padding: 20px;
}

.dashboard-header {
  text-align: center;
  margin-bottom: 30px;
  
  h1 {
    font-size: 28px;
    color: #303133;
    margin-bottom: 10px;
  }
  
  p {
    font-size: 16px;
    color: #909399;
  }
}

.stats-row {
  margin-bottom: 30px;
}

.stats-card {
  background: #fff;
  border-radius: 8px;
  padding: 20px;
  display: flex;
  align-items: center;
  box-shadow: 0 2px 12px 0 rgba(0, 0, 0, 0.1);
  transition: transform 0.3s;
  
  &:hover {
    transform: translateY(-5px);
  }
}

.stats-icon {
  width: 60px;
  height: 60px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-right: 20px;
  
  .el-icon {
    font-size: 24px;
    color: #fff;
  }
  
  &.user {
    background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  }
  
  &.role {
    background: linear-gradient(135deg, #f093fb 0%, #f5576c 100%);
  }
  
  &.login {
    background: linear-gradient(135deg, #4facfe 0%, #00f2fe 100%);
  }
  
  &.operation {
    background: linear-gradient(135deg, #43e97b 0%, #38f9d7 100%);
  }
}

.stats-content {
  flex: 1;
}

.stats-number {
  font-size: 24px;
  font-weight: 600;
  color: #303133;
  margin-bottom: 5px;
}

.stats-label {
  font-size: 14px;
  color: #909399;
}

.quick-actions, .system-info {
  margin-bottom: 30px;
  
  h2 {
    font-size: 20px;
    color: #303133;
    margin-bottom: 20px;
  }
}

.action-card {
  background: #fff;
  border-radius: 8px;
  padding: 30px 20px;
  text-align: center;
  cursor: pointer;
  transition: all 0.3s;
  box-shadow: 0 2px 12px 0 rgba(0, 0, 0, 0.1);
  
  &:hover {
    transform: translateY(-5px);
    box-shadow: 0 8px 25px 0 rgba(0, 0, 0, 0.15);
  }
  
  .el-icon {
    font-size: 32px;
    color: #409eff;
    margin-bottom: 10px;
  }
  
  span {
    display: block;
    font-size: 16px;
    color: #303133;
  }
}

.info-item {
  display: flex;
  justify-content: space-between;
  margin-bottom: 15px;
  
  &:last-child {
    margin-bottom: 0;
  }
}

:deep(.el-card__header) {
  background: #f8f9fa;
  border-bottom: 1px solid #ebeef5;
}
</style> 