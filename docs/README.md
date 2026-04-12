# Go Admin Scaffold 文档中心

## 文档目录

### 1. 快速入门
- [项目介绍](getting-started/introduction.md)
- [快速开始](getting-started/quick-start.md)
- [项目结构](getting-started/structure.md)
- [配置指南](getting-started/configuration.md)
- [开发环境搭建](getting-started/development.md)

### 2. 核心功能
- [认证系统](features/authentication.md)
  - JWT 认证
  - 登录/登出
  - 密码重置
  - 会话管理
- [RBAC 权限系统](features/rbac.md)
  - 角色管理
  - 权限管理
  - 用户角色分配
  - 权限检查
- [操作日志](features/operation-log.md)
  - 日志记录
  - 日志查询
  - 日志分析
- [国际化](features/i18n.md)
  - 语言配置
  - 翻译管理
  - 动态切换

### 3. 系统组件
- [队列系统](features/queue.md)
  - Redis + Asynq
  - 入队与消费者（`worker` / `QueueService`）
  - 唯一任务与 CLI 工具
- [缓存系统](features/cache.md)
  - Redis 缓存
  - 内存缓存
  - 缓存策略
- [存储系统](features/storage.md)
  - 本地存储
  - AWS S3 集成
  - 文件管理
- [数据库](database/README.md)
  - 数据库设计
  - 迁移管理
  - 数据填充
  - 查询优化

### 4. API 文档
- [API 概述](api/README.md)
- [认证 API](api/authentication.md)
- [用户管理 API](api/users.md)
- [角色管理 API](api/roles.md)
- [权限管理 API](api/permissions.md)
- [系统管理 API](api/system.md)

### 5. 部署指南
- [部署概述](deployment/README.md)
- [本地开发部署](deployment/local.md)
- [生产环境部署](deployment/production.md)
  - Linux 部署
  - Windows 部署
  - Docker 部署
- [多租户部署](deployment/multi-tenant.md)
- [性能优化](deployment/performance.md)
- [安全配置](deployment/security.md)

### 6. 开发指南
- [开发规范](advanced/development-guide.md)
- [测试指南](testing.md)
  - 单元测试
  - 集成测试
  - 性能测试
- [错误处理](advanced/error-handling.md)
- [日志管理](advanced/logging.md)
- [性能优化](advanced/performance.md)
- [安全最佳实践](advanced/security.md)

### 7. 示例教程
- [基础示例](examples/basic/README.md)
  - 用户管理
  - 角色管理
  - 权限管理
- [高级示例](examples/advanced/README.md)
  - 自定义中间件
  - 自定义验证器
  - 自定义响应
- [集成示例](examples/integration/README.md)
  - 第三方登录
  - 支付集成
  - 消息推送

### 8. 常见问题
- [FAQ](faq/README.md)
- [故障排除](faq/troubleshooting.md)
- [更新日志](changelog.md)

## 文档更新日志

### 2024-03-xx
- 重新组织文档结构
- 添加队列系统详细文档
- 更新部署指南
- 补充开发环境配置说明

### 2024-03-xx
- 添加 RBAC 系统更新说明
- 更新测试文档
- 补充 API 文档

## 贡献指南

欢迎为文档做出贡献！如果您发现任何问题或有改进建议，请：

1. 提交 Issue 描述问题或建议
2. 提交 Pull Request 进行修改
3. 在提交 PR 时，请确保：
   - 更新文档目录
   - 添加适当的标签
   - 提供清晰的修改说明

## 文档维护

- 文档使用 Markdown 格式编写
- 图片等资源文件存放在 `docs/assets` 目录
- 代码示例应包含完整的上下文
- 保持文档的及时更新
- 定期检查文档的准确性

## 联系方式

如有文档相关的问题或建议，请：

- 提交 Issue
- 发送邮件至：your-email@example.com
- 加入技术交流群：xxx 