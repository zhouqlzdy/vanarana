# vanarana

单测报告管理系统 — 集中管理多仓库 Gradle 多模块项目的 JUnit + JaCoCo 报告。

## 架构

```
CI Pipeline (Gradle module)
  │  POST /api/v1/reports  (tar.gz × 2)
  ▼
┌─────────────────────────────────────────┐
│              vanarana                    │
│  ┌─────────┐  ┌────────┐  ┌──────────┐  │
│  │ upload  │──▶ worker  │──▶  MySQL   │  │
│  │ handler │  │ pool   │  │ metrics  │  │
│  └─────────┘  └───┬────┘  └──────────┘  │
│                   │                     │
│  archives/ ───────┤   metrics  ◀── DB   │
│  (tar.gz)    cache/   (JSON)            │
│              (LRU)                       │
│  ┌──────────────────────────────────┐   │
│  │  SSR pages  (Go embed + template) │   │
│  │  /  /repo/{id}  /run/{j}/{b}     │   │
│  │  /report/{id}  /pipeline         │   │
│  └──────────────────────────────────┘   │
└──────────────┬──────────────────────────┘
               │  POST /api/report/:name/link
               ▼
           neutron
```

## 快速开始

```bash
# 1. 启动 MySQL
docker compose up -d mysql

# 2. 配置环境变量
export VANARANA_MYSQL_DSN="root:vanarana@tcp(127.0.0.1:3306)/vanarana?parseTime=true&charset=utf8mb4"
export VANARANA_NEUTRON_URL="http://localhost:8888"   # 可选
export VANARANA_BASE_URL="http://localhost:8080"

# 3. 构建运行
go build -o vanarana ./cmd/server/
./vanarana
```

## 配置

| 环境变量 | 默认值 | 说明 |
|----------|--------|------|
| `VANARANA_PORT` | `8080` | 服务端口 |
| `VANARANA_MYSQL_DSN` | `root:@tcp(127.0.0.1:3306)/vanarana?...` | MySQL 连接串 |
| `VANARANA_DATA_DIR` | `./data` | 归档和缓存目录 |
| `VANARANA_MAX_UPLOAD_MB` | `100` | 上传大小限制 |
| `VANARANA_CACHE_MAX_MB` | `2048` | LRU 缓存上限 |
| `VANARANA_NEUTRON_URL` | （空） | Neutron API 地址，空则不回调 |
| `VANARANA_BASE_URL` | `http://localhost:8080` | 本服务对外 URL |

## API

### 上传报告

```
POST /api/v1/reports
Content-Type: multipart/form-data

repo_url          string  仓库地址
module_name       string  模块名
pipeline_job_name string  作业名
build_id          string  构建号
branch            string  分支（可选）
commit_hash       string  提交 SHA（可选）
jacoco            file    jacocoHtml.tar.gz
junit             file    reports.tar.gz
```

### 查询接口

```
GET /api/v1/repositories/{repoID}/pipeline-runs?days=7
GET /api/v1/repositories/{repoID}/trends?days=7
GET /api/v1/module-reports/{id}
GET /api/v1/module-reports/{id}/{type}/html/
GET /api/v1/trends?repo_url=&module_name=
```

### Neutron 回调

当一次执行的全部模块报告解析完成后，自动回调 Neutron：

```
POST {NEUTRON_URL}/api/report/{jobName}/link
{"report_url": "{BASE_URL}/run/{jobName}/{buildId}"}
```

## 页面

| 路由 | 说明 |
|------|------|
| `/` | 仓库列表 |
| `/repo/{id}` | 仓库概览 — 趋势图表 + 最近执行记录 |
| `/run/{jobName}/{buildId}` | 单次执行详情 — 列出全部模块报告 |
| `/report/{id}` | 模块报告详情 — JUnit + JaCoCo 指标卡片 |
| `/pipeline` | 作业查询（按时间范围和作业名筛选） |

## Gradle 集成

在每个模块的 CI 脚本中：

```bash
MODULE_NAME="my-project-core"
PIPELINE_JOB_NAME="PR-check"

tar czf jacocoHtml.tar.gz -C build/reports jacoco/test/html
tar czf junitReports.tar.gz -C build/reports tests/test

curl -X POST ${VANARANA_URL}/api/v1/reports \
  -F "repo_url=${CI_REPOSITORY_URL}" \
  -F "module_name=${MODULE_NAME}" \
  -F "pipeline_job_name=${PIPELINE_JOB_NAME}" \
  -F "build_id=${CI_BUILD_ID}" \
  -F "branch=${CI_COMMIT_BRANCH}" \
  -F "commit_hash=${CI_COMMIT_SHA}" \
  -F "jacoco=@jacocoHtml.tar.gz" \
  -F "junit=@junitReports.tar.gz"
```

## 项目结构

```
vanarana/
├── cmd/server/main.go          入口
├── internal/
│   ├── config/                 配置
│   ├── model/                  数据模型
│   ├── store/                  MySQL 数据访问
│   ├── archive/                tar.gz 存储与解压
│   ├── cache/                  LRU 解压缓存
│   ├── parser/                 JUnit + JaCoCo HTML 解析
│   ├── worker/                 异步解析 worker pool
│   ├── handler/                HTTP 处理器
│   └── notify/                 Neutron 回调
├── web/
│   ├── templates/              Go SSR 模板
│   ├── static/                 CSS + JS
│   └── embed.go                embed 指令
├── migrations/                 数据库迁移
├── docker-compose.yml         本地 MySQL
└── Makefile
```

## 技术栈

Go · MySQL · Chi Router · html/template · embed · Chart.js · Docker
