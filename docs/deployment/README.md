# 部署指南

本文档详细说明了 Go Admin Scaffold 的部署方法和最佳实践。

## 部署方式

### 1. 二进制部署

直接使用编译好的二进制文件部署，适用于大多数场景。

#### 准备工作

1. 下载二进制文件
```bash
# 下载最新发布版本
wget https://github.com/1768177868/go-admin-scaffold/releases/latest/download/go-admin-scaffold.zip

# 解压文件
unzip go-admin-scaffold.zip
```

2. 准备配置文件
```bash
# 复制配置文件
cp configs/config.example.yaml configs/config.prod.yaml

# 修改配置
vim configs/config.prod.yaml
```

3. 准备数据库
```bash
# 创建数据库
mysql -u root -p
CREATE DATABASE go_admin CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

# 运行迁移
./migrate.exe
```

#### 启动服务

1. 启动主服务
```bash
# 直接运行
./server.exe

# 后台运行
nohup ./server.exe > server.log 2>&1 &

# 使用 systemd (Linux)
sudo systemctl start go-admin
```

2. 启动队列服务
```bash
# 直接运行
./worker.exe

# 后台运行
nohup ./worker.exe > worker.log 2>&1 &

# 使用 systemd (Linux)
sudo systemctl start go-admin-queue
```

### 2. Docker 部署

使用 Docker 容器化部署，便于环境一致性管理。

#### 准备工作

1. 安装 Docker
```bash
# Ubuntu/Debian
sudo apt update
sudo apt install docker.io docker-compose

# CentOS
sudo yum install docker docker-compose
sudo systemctl start docker
```

2. 准备配置文件
```bash
# 复制配置文件
cp configs/config.example.yaml configs/config.prod.yaml
cp deploy/docker-compose.example.yml docker-compose.yml

# 修改配置
vim configs/config.prod.yaml
vim docker-compose.yml
```

#### 启动服务

```bash
# 构建并启动所有服务
docker-compose up -d

# 查看服务状态
docker-compose ps

# 查看服务日志
docker-compose logs -f
```

### 3. Kubernetes 部署

使用 Kubernetes 进行容器编排，适用于大规模部署。

#### 准备工作

1. 准备 Kubernetes 集群
2. 安装 kubectl
3. 配置 kubeconfig

#### 部署步骤

1. 创建命名空间
```bash
kubectl create namespace go-admin
```

2. 创建配置
```bash
# 创建 ConfigMap
kubectl create configmap go-admin-config \
    --from-file=configs/config.prod.yaml \
    -n go-admin

# 创建 Secret
kubectl create secret generic go-admin-secrets \
    --from-literal=db-password=your-password \
    --from-literal=redis-password=your-password \
    -n go-admin
```

3. 部署应用
```bash
# 部署主服务
kubectl apply -f deploy/kubernetes/server.yaml

# 部署队列服务
kubectl apply -f deploy/kubernetes/worker.yaml

# 部署数据库
kubectl apply -f deploy/kubernetes/mysql.yaml

# 部署 Redis
kubectl apply -f deploy/kubernetes/redis.yaml
```

## 环境配置

### 1. 生产环境

#### 系统要求

- CPU: 2核或以上
- 内存: 4GB或以上
- 磁盘: 50GB或以上
- 操作系统: Ubuntu 20.04/CentOS 8/Windows Server 2019

#### 依赖服务

- MySQL 5.7+
- Redis 6.0+
- Nginx (可选，用于反向代理)

#### 安全配置

1. 防火墙设置
```bash
# 开放必要端口
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw allow 8080/tcp
```

2. SSL 配置
```nginx
# Nginx SSL 配置
server {
    listen 443 ssl;
    server_name your-domain.com;

    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;

    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

### 2. 开发环境

#### 本地开发

1. 安装依赖
```bash
# 安装 Go
go install

# 安装 MySQL
brew install mysql  # Mac
sudo apt install mysql-server  # Ubuntu

# 安装 Redis
brew install redis  # Mac
sudo apt install redis-server  # Ubuntu
```

2. 配置开发环境
```bash
# 复制开发配置
cp configs/config.example.yaml configs/config.dev.yaml

# 修改配置
vim configs/config.dev.yaml
```

3. 启动服务
```bash
# 启动主服务（带热重载）
air

# 启动队列服务
go run cmd/worker/main.go
```

#### Docker 开发环境

```bash
# 启动开发环境
docker-compose -f docker-compose.dev.yml up -d

# 查看日志
docker-compose -f docker-compose.dev.yml logs -f
```

## 监控和维护

### 1. 日志管理

#### 日志配置

```yaml
# configs/config.prod.yaml
logger:
  level: "info"
  filename: "logs/app.log"
  max_size: 100
  max_age: 30
  max_backups: 10
  compress: true
```

#### 日志收集

1. 使用 ELK 栈
```yaml
# deploy/elk/filebeat.yml
filebeat.inputs:
- type: log
  paths:
    - /path/to/logs/*.log
  fields:
    app: go-admin
  fields_under_root: true

output.elasticsearch:
  hosts: ["elasticsearch:9200"]
```

2. 使用 Prometheus + Grafana
```yaml
# deploy/prometheus/prometheus.yml
scrape_configs:
  - job_name: 'go-admin'
    static_configs:
      - targets: ['localhost:8080']
```

### 2. 性能监控

#### 系统监控

1. 使用 Node Exporter
```bash
# 安装 Node Exporter
docker run -d \
  --name node-exporter \
  -p 9100:9100 \
  prom/node-exporter
```

2. 使用 cAdvisor
```bash
# 安装 cAdvisor
docker run -d \
  --name=cadvisor \
  -p 8080:8080 \
  -v /:/rootfs:ro \
  -v /var/run:/var/run:ro \
  -v /sys:/sys:ro \
  -v /var/lib/docker/:/var/lib/docker:ro \
  google/cadvisor:latest
```

#### 应用监控

1. 使用 Prometheus 客户端
```go
import "github.com/prometheus/client_golang/prometheus"

// 注册指标
var (
    httpRequests = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "http_requests_total",
            Help: "Total number of HTTP requests",
        },
        []string{"method", "endpoint", "status"},
    )
)

func init() {
    prometheus.MustRegister(httpRequests)
}
```

2. 使用 Grafana 仪表板
```bash
# 安装 Grafana
docker run -d \
  --name grafana \
  -p 3000:3000 \
  grafana/grafana
```

### 3. 备份策略

#### 数据库备份

1. 自动备份脚本
```bash
#!/bin/bash
# backup.sh
BACKUP_DIR="/path/to/backups"
DATE=$(date +%Y%m%d)
mysqldump -u root -p go_admin > $BACKUP_DIR/go_admin_$DATE.sql
```

2. 定时任务
```bash
# 添加到 crontab
0 2 * * * /path/to/backup.sh
```

#### 文件备份

1. 配置文件备份
```bash
# 备份配置文件
tar -czf configs_$(date +%Y%m%d).tar.gz configs/
```

2. 上传到对象存储
```bash
# 使用 AWS CLI
aws s3 cp configs_20240310.tar.gz s3://your-bucket/backups/
```

## 故障排除

### 1. 常见问题

#### 服务无法启动

检查：
- 配置文件是否正确
- 端口是否被占用
- 依赖服务是否正常
- 日志中是否有错误

#### 性能问题

检查：
- 系统资源使用情况
- 数据库连接池配置
- Redis 连接配置
- 队列处理状态

#### 连接问题

检查：
- 网络连接状态
- 防火墙配置
- 服务健康状态
- 日志中的错误信息

### 2. 维护命令

#### 服务管理

```bash
# 查看服务状态
systemctl status go-admin
systemctl status go-admin-queue

# 重启服务
systemctl restart go-admin
systemctl restart go-admin-queue

# 查看日志
journalctl -u go-admin -f
journalctl -u go-admin-queue -f
```

#### 数据库维护

```bash
# 检查数据库状态
mysql -u root -p -e "SHOW STATUS;"

# 优化数据库
mysql -u root -p -e "OPTIMIZE TABLE table_name;"

# 修复数据库
mysqlcheck -u root -p --repair go_admin
```

#### 队列维护

```bash
# 查看队列任务规模（Asynq）
./queue-status -all

# 清空某队列（慎用）
./queue -clear -queue=default

# 可视化与归档任务管理可使用 asynqmon（需单独部署）
```

## 相关文档

- [配置说明](../getting-started/configuration.md)
- [开发环境配置](../advanced/development.md)
- [API 文档](../api/README.md)
- [队列系统](../features/queue.md) 