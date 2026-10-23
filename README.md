# AbingBlog-Backend

个人博客后端 —— Go 语言练手项目，已跑通 **本地开发 → 分层架构 → Docker → CI → CD → 公网访问** 完整闭环。

技术栈横跨 Gin + GORM + MySQL + Redis + JWT，已覆盖文章/分类/标签/作品墙 CRUD、JWT 单管理员鉴权、友链（访客自助提交 + 管理员审核）、访客记录（中间件自动采集 + 统计）、Docker 多阶段构建、GitHub Actions CI/CD，配套一个 Vue 3 + TypeScript 前端（含后台管理）。

## 线上地址

- 站点：**https://abing3952.cn**（终端风前台 + 后台管理）
- API Base URL：`https://abing3952.cn/api/v1`
- 健康检查：`curl https://abing3952.cn/api/v1/healthz`

部署形态（阿里云 ECS，详见 [`docs/deployment.md`](docs/deployment.md)）：

```
Internet
   │ 80/443（certbot HTTPS）
   ▼
宿主机 Nginx ──/api/──► 127.0.0.1:8080  server 容器（Go 后端）
              ──/────► 127.0.0.1:8081  frontend 容器（Vue SPA）
server 容器 ────────► mysql 容器 / redis 容器（docker-compose.prod.yml 组网）
```

> 实际生产中统一入口由 `deploy/gateway/` 的 Nginx 网关容器承担（host 网络），server / frontend / mysql / redis 四个容器只绑 `127.0.0.1`，不直接对外。

## 技术栈

| 层 | 选型 | 用途 |
|---|---|---|
| HTTP 框架 | Gin | 路由、中间件、请求解析 |
| ORM | GORM | 数据库操作、自动迁移、软删除 |
| 配置 | YAML + 环境变量覆盖 | `configs/config.yaml` 兜底，部署时靠环境变量注入 |
| 鉴权 | JWT（golang-jwt/v5）+ bcrypt | 单管理员登录，后台接口鉴权 |
| 数据库 | MySQL 8.0 | 持久化存储 |
| 缓存 | Redis（客户端已接入，缓存策略待实现） | 文章列表/详情缓存，Cache Aside |
| 消息队列 | 待定（计划中） | 浏览量异步落库 |
| 容器 | Docker + Docker Compose | 本地开发环境 + 生产部署 |
| 反向代理 | Nginx（`deploy/gateway/`） | 统一入口，`/` → 前端，`/api` → 后端 |
| CI/CD | GitHub Actions | CI：格式 + vet + test + build；CD：推 GHCR 镜像 + SSH 部署 |

## 功能清单

| 模块 | 状态 | 说明 |
|---|---|---|
| 文章 CRUD | ✅ | 分类/标签关联、草稿/发布状态机、软删除、分页 + 关键词/分类/标签筛选 |
| 分类 / 标签管理 | ✅ | 唯一性校验；分类下有文章禁止删除；删标签清关联 |
| 作品墙（projects） | ✅ | slug 唯一性、排序权重、状态约定同文章 |
| JWT 鉴权 | ✅ | 登录签发、`AuthRequired` 中间件验签（校验 HMAC 算法，防 `alg=none`） |
| 站点指标 | ✅ | 独立访客数 + 累计启动次数（原子自增） |
| 友链 | ✅ | 访客凭 `X-Visitor-ID` 自助提交/改/删，管理员审核（上架/驳回）后才前台可见 |
| 访客记录 | ✅ | `VisitTracker` 中间件自动记录访问明细，后台可查汇总 |
| Docker 化 | ✅ | 多阶段构建（builder + alpine 运行时），非 root 用户运行 |
| CI / CD | ✅ | CI 四件套；推 `v*.*.*` 标签自动构建镜像 → GHCR → SSH 部署 → 健康检查 |
| Redis 缓存 | 🟡 | 客户端已接入探活，Cache Aside 策略待实现 |
| MQ 异步 | 🔲 | 计划中：浏览量攒批落库 |

## 项目结构

```
Blog-Backend/
├── cmd/
│   ├── server/
│   │   └── main.go              # 程序入口：装配依赖，启动服务（启动时自动 seed 管理员）
│   └── seed/
│       └── main.go              # 独立的管理员初始化命令（可重复执行，已存在则跳过）
├── configs/
│   ├── config.yaml              # 配置文件（端口、数据库、Redis、JWT、CORS）
│   └── config_test.go           # 配置解析测试
├── internal/                    # 业务代码（Go 惯例：internal 对外不可 import）
│   ├── config/
│   │   └── config.go            # 读取 config.yaml → Config 结构体，再用环境变量覆盖
│   ├── database/
│   │   └── database.go          # 连接 MySQL，自动建库建表（AutoMigrate）
│   ├── cache/
│   │   └── cache.go             # 连接 Redis（go-redis/v9）并启动探活
│   ├── auth/
│   │   ├── jwt.go               # GenerateToken / ParseToken（HS256，校验签名算法）
│   │   └── password.go          # bcrypt 哈希与校验
│   ├── model/
│   │   └── model.go             # 数据库表 → Go struct（User/Category/Tag/Article/Project/SiteMetric/FriendLink/VisitLog）
│   ├── repository/              # 数据访问层 —— 只写 SQL，不写业务逻辑
│   │   ├── article.go           # ArticleRepo：文章 CRUD + 分页查询 + 标签关联
│   │   ├── category.go          # CategoryRepo：分类 CRUD + 分组统计文章数
│   │   ├── tag.go               # TagRepo：标签 CRUD + 分组统计文章数
│   │   ├── user.go              # UserRepo：按用户名查管理员
│   │   ├── project.go           # ProjectRepo：作品墙 CRUD + 分页
│   │   ├── site_metric.go       # SiteMetricRepo：启动次数原子自增
│   │   ├── FriendLink.go        # FriendLinkRepo：友链 CRUD + 按 owner 查
│   │   └── visit_log.go         # VisitLogRepo：访问明细写入 + 汇总统计
│   ├── service/                 # 业务逻辑层 —— 校验、状态规则、数据组装
│   │   ├── article.go           # ArticleService：创建/更新/发布/撤回/删除
│   │   ├── category.go          # CategoryService：唯一性校验、删除保护
│   │   ├── tag.go               # TagService：唯一性校验、删除时清理关联
│   │   ├── user.go              # UserService：SeedAdmin + Login 校验密码签发 token
│   │   ├── project.go           # ProjectService：slug 唯一性、状态规则
│   │   ├── site_metric.go       # SiteMetricService：累计启动次数 / 独立访客
│   │   ├── FriendLink.go        # FriendLinkService：审核状态机 + 归属校验
│   │   └── visit_log.go         # VisitLogService：访客汇总查询
│   ├── handler/                 # HTTP 接口层 —— 解析请求 → 调 service → 封装响应
│   │   ├── handler.go           # Handler 聚合结构体（所有 handler 的集合）
│   │   ├── article.go           # ArticleHandler：文章列表/详情/创建/更新/删除/发布
│   │   ├── category.go          # CategoryHandler：分类 CRUD
│   │   ├── tag.go               # TagHandler：标签 CRUD
│   │   ├── project.go           # ProjectHandler：项目 CRUD
│   │   ├── user.go              # UserHandler：登录
│   │   ├── site_metric.go       # SiteMetricHandler：/visits 读写
│   │   ├── health.go            # HealthHandler：健康检查（DB 连通性）
│   │   ├── FriendLink.go        # FriendLinkHandler：友链表单/审核
│   │   ├── visit_log.go         # VisitLogHandler：后台访客记录/汇总
│   │   └── dto.go               # 返回给前端的 JSON 结构（与数据库 model 解耦）
│   ├── middleware/
│   │   ├── auth.go              # AuthRequired：解析 Bearer token，失败 401
│   │   ├── cors.go              # CORS 白名单，放行前端 dev server
│   │   └── visit.go             # VisitTracker：公共接口自动记录访客行为
│   ├── router/
│   │   └── router.go            # 路由注册 + 权限分组（公共 / 后台）
│   ├── response/
│   │   └── response.go          # 统一响应格式 {code, message, data}
│   └── errcode/
│       └── errcode.go           # 业务错误码定义 + BizError 类型
├── deploy/
│   └── gateway/                 # 统一入口 Nginx 网关容器：/ → 前端，/api → 后端
│       ├── docker-compose.yml
│       └── nginx.conf
├── docs/
│   ├── api-v1.md                # API 文档（前后端契约）
│   └── deployment.md            # 部署与调试手册（ECS + Nginx + HTTPS + 回滚）
├── docker-compose.yml           # 本地 MySQL + Redis 开发环境
├── Dockerfile                   # 生产镜像，多阶段构建
├── docker-compose.prod.yml      # 生产服务编排（mysql + redis + server）
├── .github/workflows/
│   ├── ci.yml                   # 格式 + vet + test(-race) + build
│   └── deploy.yml               # 发布 GHCR 镜像并通过 SSH 部署
├── go.mod
└── go.sum
```

## 快速开始

完整的阿里云 ECS、Docker、Nginx、HTTPS、CI/CD、回滚和故障排查说明见 [`docs/deployment.md`](docs/deployment.md)。

### 1. 前置条件

- Go 1.27+
- Docker Desktop（提供本地 MySQL + Redis）

### 2. 启动 MySQL 与 Redis

```bash
# 启动依赖容器（后台运行）
docker compose up -d

# 确认已就绪
docker compose ps
# 看到 blog-mysql / blog-redis 状态为 healthy 即可

# 停止（数据不丢失，存在 Docker 命名卷里）
docker compose down
```

### 3. 启动后端服务

```bash
# 安装依赖
go mod tidy

# 运行（首次启动会自动创建初始管理员 admin/admin123456）
go run ./cmd/server/

# 看到以下输出表示启动成功：
# Redis connected at 127.0.0.1:6379 (db=0)
# server running at http://localhost:8080
```

需要手动重置/新建管理员时用独立命令（已存在则跳过，不会覆盖密码）：

```bash
ABINGBLOG_ADMIN_USERNAME=admin ABINGBLOG_ADMIN_PASSWORD=你的密码 go run ./cmd/seed/
```

### 4. 验证

```bash
# 健康检查
curl http://localhost:8080/api/v1/healthz

# 文章列表（此时为空）
curl http://localhost:8080/api/v1/articles

# 登录拿 token（返回值里的 token 给下面所有 admin 请求用）
curl -X POST http://localhost:8080/api/v1/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123456"}'

TOKEN=<上一步返回的 token>

# 创建分类
curl -X POST http://localhost:8080/api/v1/admin/categories \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"name":"Go","slug":"go"}'

# 创建标签
curl -X POST http://localhost:8080/api/v1/admin/tags \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"name":"后端"}'

# 创建文章
curl -X POST http://localhost:8080/api/v1/admin/articles \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"title":"第一篇文章","content":"Hello World","status":"published","category_id":1,"tag_ids":[1]}'

# 查看文章列表 / 详情
curl http://localhost:8080/api/v1/articles
curl http://localhost:8080/api/v1/articles/1

# 项目列表（作品墙）
curl http://localhost:8080/api/v1/projects

# 站点指标：独立访客数 + 累计启动次数（只读 / 自增一次）
curl http://localhost:8080/api/v1/visits
curl -X POST http://localhost:8080/api/v1/visits/start

# 友链：访客自助提交（X-Visitor-ID 认领归属，待审核）
curl -X POST http://localhost:8080/api/v1/friend-links/mine \
  -H "Content-Type: application/json" \
  -H "X-Visitor-ID: 550e8400-e29b-41d4-a716-446655440000" \
  -d '{"name":"某某博客","url":"https://example.com","description":"Go / Vue"}'

# 后台审核友链（status=1 上架 / 2 驳回）
curl -X PUT http://localhost:8080/api/v1/admin/friend-links/1 \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"status":1}'

# 后台看访客记录
curl http://localhost:8080/api/v1/admin/visits/summary \
  -H "Authorization: Bearer $TOKEN"
```

> 未带 token 访问 `/api/v1/admin/*` 会返回 401，这是 JWT 中间件在起作用——可以顺手验证一下鉴权链路。

## 架构设计

### 分层架构（三层）

```
HTTP 请求
  │
  ▼
┌──────────┐
│  router   │  路由层：URL → handler 映射，分组控制权限边界
└────┬─────┘
     │
     ▼
┌──────────┐
│  handler  │  接口层：解析请求参数、调 service、封装 JSON 响应
└────┬─────┘  不写业务逻辑，不直接操作数据库
     │
     ▼
┌──────────┐
│  service  │  业务逻辑层：参数校验、状态规则、数据组装、抛出业务错误
└────┬─────┘  不碰 HTTP 的东西（gin.Context），不拼 SQL
     │
     ▼
┌──────────┐
│ repository│  数据访问层：只写 GORM/SQL 查询，不写业务逻辑
└────┬─────┘
     │
     ▼
┌──────────┐
│  model    │  数据模型：Go struct = 数据库表结构
└────┬─────┘
     │
     ▼
  MySQL
```

### 为什么要分层？

每一层只做一件事，改一个地方不影响其他地方：

- **改数据库表结构** → 只改 `model/` 和 `repository/`，handler 和 service 不用动
- **改业务规则**（比如"草稿不能超过 10 篇"）→ 只改 `service/`，handler 和 repo 不用动
- **改接口返回格式** → 只改 `handler/dto.go`，service 和 repo 不用动
- **换数据库**（MySQL → PostgreSQL）→ 只改 `repository/` 和 `database/`，上层完全无感

### 依赖注入（手写，不用框架）

```go
// cmd/server/main.go — 依赖装配顺序：底层 → 上层
cfg := config.Load()
setTimeZone(cfg.Server.Timezone)  // 容器默认 UTC，先校准时区（影响时间写入与「今日」统计边界）
cache.Init(cfg)          // Redis 探活，连不上直接 Fatal，避免带病启动
db := database.Init(cfg) // 连接 MySQL + AutoMigrate

// 1. repo 依赖 db
articleRepo  := repository.NewArticleRepo(db)
categoryRepo := repository.NewCategoryRepo(db)
tagRepo      := repository.NewTagRepo(db)
userRepo     := repository.NewUserRepo(db)
metricRepo   := repository.NewSiteMetricRepo(db)
projectRepo  := repository.NewProjectRepo(db)
friendLinkRepo := repository.NewFriendLinkRepo(db)
visitLogRepo := repository.NewVisitLogRepo(db)

// 2. service 依赖 repo（UserService 额外依赖 cfg，用来取 JWT 密钥和初始管理员）
userSvc := service.NewUserService(userRepo, cfg)
userSvc.SeedAdmin()      // 启动时确保初始管理员存在，已存在则跳过

// 3. handler 依赖 service
h := &handler.Handler{
    Article:    handler.NewArticleHandler(service.NewArticleService(articleRepo, categoryRepo, tagRepo)),
    Category:   handler.NewCategoryHandler(service.NewCategoryService(categoryRepo)),
    Tag:        handler.NewTagHandler(service.NewTagService(tagRepo)),
    User:       handler.NewUserHandler(userSvc),
    Health:     handler.NewHealthHandler(db),
    Metric:     handler.NewSiteMetricHandler(service.NewSiteMetricService(metricRepo, visitLogRepo)),
    Project:    handler.NewProjectHandler(service.NewProjectService(projectRepo)),
    FriendLink: handler.NewFriendLinkHandler(service.NewFriendLinkService(friendLinkRepo)),
    Visit:      handler.NewVisitLogHandler(service.NewVisitLogService(visitLogRepo)),
}

// 4. router 依赖 handler + JWT 密钥 + CORS 白名单 + 访客记录 repo
r := router.Setup(h, cfg.JWT.Secret, cfg.CORS.AllowOrigins, visitLogRepo)
```

这种手写方式的好处：**显式、可读、不需要学 wire 之类的工具**。依赖关系一目了然。

### 配置加载：YAML 兜底 + 环境变量覆盖

`config.Load()` 分两步（`internal/config/config.go`）：

1. 读 `configs/config.yaml` 得到本地开发默认值
2. 逐个字段检查对应的大写环境变量是否设置，设置了就覆盖（空字符串视为未设置，避免容器里漏传的变量把默认值冲成空）

这样同一份二进制在本地和容器里跑同一套代码路径：本地靠 YAML，生产靠 `MYSQL_HOST`、`REDIS_HOST`、`JWT_SECRET` 等环境变量（见 `docker-compose.prod.yml`）。

> ⚠️ `configs/config.yaml` 里的 `jwt.secret` 只是占位值，`JWTConfig.Validate()` 要求密钥至少 32 字节。生产**必须**用 `JWT_SECRET` 环境变量注入强密钥。

### 统一响应格式

所有接口返回同一个 JSON 结构：

```json
{
  "code": 0,
  "message": "success",
  "data": { ... }
}
```

- `code=0` 成功，`code≠0` 失败（具体含义见 `docs/api-v1.md` 的错误码表）
- handler 层统一用 `response.OK()` / `response.Fail()` / `response.Error()` 三个函数
- service 层抛出 `errcode.BizError`，handler 层自动映射为 HTTP 状态码

### Model vs DTO

数据库的 Go struct（`model.Article`）和返回给前端的 JSON 结构（`articleDTO`）是两套：

- 数据库加了字段，不自动暴露给前端
- 列表接口不返回 `content`（Markdown 正文很大），详情接口才返回
- `handler/dto.go` 里的 `toArticleDTO()` 负责转换，控制哪些字段暴露

### 路由权限模型

```
/api/v1/*          → 公共接口（游客可访问，只返回 status=published 的内容）
/api/v1/admin/*    → 后台接口（必须携带 Authorization: Bearer <token>）
```

完整路由（`internal/router/router.go`）：

| 分组 | 方法 | 路由 |
|---|---|---|
| 公共 | GET | `/articles`、`/articles/:id`、`/categories`、`/tags`、`/projects`、`/healthz`、`/visits`、`/friend-links` |
| 公共 | POST | `/login`、`/visits/start` |
| 公共 | GET/POST/PUT/DELETE | `/friend-links/mine`（系列）——访客凭 `X-Visitor-ID` 管理自己提交的友链 |
| 后台 | GET | `/admin/articles`、`/admin/articles/:id`、`/admin/projects`、`/admin/friend-links`、`/admin/visits`、`/admin/visits/summary` |
| 后台 | POST | `/admin/articles`、`/admin/categories`、`/admin/tags`、`/admin/projects` |
| 后台 | PUT | `/admin/articles/:id`、`/admin/articles/:id/status`、`/admin/categories/:id`、`/admin/tags/:id`、`/admin/projects/:id`、`/admin/friend-links/:id`（审核） |
| 后台 | DELETE | `/admin/articles/:id`、`/admin/categories/:id`、`/admin/tags/:id`、`/admin/projects/:id`、`/admin/friend-links/:id` |

鉴权链路：`POST /login` → `UserService.Login` 查用户 → `bcrypt` 比对密码 → `auth.GenerateToken` 签发 JWT → 前端存 token → 请求 admin 接口带 `Authorization` 头 → `middleware.AuthRequired` 调 `auth.ParseToken` 验签 → 通过则 `c.Set("user_id", ...)` 并放行，失败 401。

> `ParseToken` 的 keyFunc 里**必须**校验签名算法是 HMAC，否则会被 `alg=none` 攻击绕过验签。

### 中间件：CORS + VisitTracker

两个中间件，各管一件事：

- **Cors**（`middleware/cors.go`）：挂在最外层，按白名单放行前端来源，允许 `Authorization` 头。本地开发白名单是 `localhost:5173`，生产用环境变量注入真实域名。
- **VisitTracker**（`middleware/visit.go`）：挂在 `/api/v1` 公共组，每个公共请求自动落一条访问明细。访客身份两级识别：前端生成的 UUID（`X-Visitor-ID` 头）优先，没有则降级 `sha256(IP + UserAgent)`；同时记录昵称（`X-Visitor-Name`）和设备串（`X-Visitor-Device`）。`/admin/*` 和 `/healthz` 在中间件内部跳过——后台流量和探针不统计。

### 友链审核流

友链不是"管理员单向增删改"，而是一个带审核状态机的小流程：

```
访客提交（POST /friend-links/mine）
   │  owner_id = X-Visitor-ID（认领归属）
   ▼
status = 0 待审核 ──管理员 PUT 审核──► status = 1 已上架（前台 /friend-links 可见）
   ▲                                └─► status = 2 已驳回（前台不可见）
   │
访客修改自己的友链 ──► 打回 0 重新审核（防绕过审核）
```

归属校验在 service 层：访客只能改/删 `owner_id` 与自己 `X-Visitor-ID` 一致的记录，改别人的直接拒。

## 数据库表关系

```
users (管理员，单用户)          site_metrics (站点指标，单行 id=1)

categories (分类)
  │
  ├── articles (文章) ──多对多── tags (标签)
  │                        │
  │                   article_tags (中间表，GORM 自动生成)
  │
projects (作品墙，独立表)
friend_links (友链，owner_id 认领归属，status 审核状态机)
visit_logs (访客记录，中间件自动写入的访问明细)
```

| 表 | 说明 |
|---|---|
| `users` | 管理员账号，密码 bcrypt 入库，绝不存明文 |
| `categories` | 文章分类，`slug` 供前端筛选，`sort` 控制展示顺序 |
| `tags` | 文章标签，删除时清理 `article_tags` 关联 |
| `articles` | 文章，正文为 Markdown 原文（前端负责渲染），`url` 字段支持外链（原文在掘金/CSDN 等平台时只存跳转地址） |
| `projects` | 作品墙条目，字段对齐前端 `ProjectSummary` |
| `site_metrics` | 站点指标，固定 `id=1`，用 `UPDATE ... + 1` 原子自增 |
| `friend_links` | 友链，`owner_id` 记录提交者，`status`：0 待审核 / 1 上架 / 2 驳回 |
| `visit_logs` | 访客明细，`visitor_key` 两级身份识别的结果 |

- 一篇文章属于一个分类（`category_id`，0 表示未分类），可以有多个标签（多对多，通过 `article_tags` 中间表）
- 文章、项目、友链都支持软删除（`deleted_at`）：DELETE 只打标记，数据可恢复
- 分类下有文章时不允许删除分类（防止文章变成"无分类"状态）
- 文章和项目共用 `status` 约定：`draft` 草稿 / `published` 已发布，公共接口只放行 `published`

## 如何添加新功能

标准流程，以「评论」为例：

```
1. internal/model/model.go       → 加 Comment struct 定义表结构
2. internal/repository/comment.go → NewCommentRepo，写 CRUD 方法
3. internal/service/comment.go    → NewCommentService，写业务校验
4. internal/handler/comment.go    → NewCommentHandler，写 HTTP 接口
5. internal/handler/handler.go    → 在 Handler 结构体里加 Comment 字段
6. internal/router/router.go      → 注册路由
7. cmd/server/main.go             → 装配依赖链：repo → service → handler
8. docs/api-v1.md                 → 更新 API 文档
```

每一步都是**复制已有的模式**——`FriendLink`（handler/service/repository/model 四件套 + 审核状态机）和 `VisitLog`（中间件 + 后台查询）是最新的两个完整参考实现，照着写评论就行。

## 开发命令速查

```bash
# 运行
go run ./cmd/server/

# 构建
go build -o server.exe ./cmd/server/

# 静态检查
go vet ./...

# 测试
go test ./...

# 格式化检查（CI 会跑，格式不对直接失败）
gofmt -l .

# 安装/更新依赖
go mod tidy

# 查看依赖关系
go mod graph
```

## 部署

推标签即触发 CD（`.github/workflows/deploy.yml`）：

```bash
git tag v1.0.0
git push origin v1.0.0
```

流水线做的事：构建多阶段镜像推 GHCR（同时把 mysql/redis/nginx 基础镜像中转进 GHCR，绕开国内服务器连不上 Docker Hub 的问题）→ SCP 同步 compose 文件 → SSH 登录服务器拉镜像起容器 → 循环探测 `/api/v1/healthz` 直到就绪（首启要跑 AutoMigrate + seed，直接 curl 会撞上空回复）。镜像按 commit SHA 保存，回滚只需把 `IMAGE` 换成旧 SHA 重新 `up -d`。

仓库需要的 Secrets：`DEPLOY_HOST`、`DEPLOY_USER`、`DEPLOY_SSH_KEY`、`DEPLOY_PATH`、`MYSQL_ROOT_PASSWORD`、`JWT_SECRET`。完整步骤、HTTPS、备份与回滚见 [`docs/deployment.md`](docs/deployment.md)。

## 路线图

| 阶段 | 内容 | 状态 |
|---|---|---|
| 1 | 项目骨架 + 分层架构 + 文章/分类/标签 CRUD | ✅ 已完成 |
| 2 | JWT 登录认证 + 后台接口鉴权 + 环境变量覆盖配置 | ✅ 已完成 |
| 3 | 作品墙（projects）+ 站点指标（独立访客/启动次数） | ✅ 已完成 |
| 4 | Docker 化（多阶段构建）+ 生产 compose | ✅ 已完成 |
| 5 | CI（格式/vet/test/build）+ CD（GHCR + SSH 部署） | ✅ 已完成 |
| 6 | 前后端联调 + 网关编排（Nginx）+ 域名 HTTPS 上线 | ✅ 已完成 |
| 7 | 友链（访客自助提交 + 审核）+ 访客记录 | ✅ 已完成 |
| 8 | Redis 缓存（Cache Aside） | 🟡 客户端已接入探活，缓存策略待实现 |
| 9 | MQ 异步（浏览量落库） | 🔲 待开发 |

已完成的能力链路：**本地开发 → 分层架构 → 容器化 → CI → CD → 公网访问**。线上地址 https://abing3952.cn 。

接口契约见 [`docs/api-v1.md`](docs/api-v1.md)，部署与排查见 [`docs/deployment.md`](docs/deployment.md)。

## 常见问题

**Q: 为什么 `internal/` 下分这么多包，不能全写在一个文件里吗？**

A: 可以，但分层后每一层职责单一，改 bug 时能快速定位。面试官看到分层清晰的项目会比看到一个大文件印象好得多。这也是 Go 社区的主流做法。

**Q: 为什么不用 wire 之类的依赖注入框架？**

A: 这个项目依赖少，手动装配不到 10 行代码，引入框架反而增加学习成本。等依赖多到手动装配变痛苦时再考虑也不迟。

**Q: 如何部署生产环境？**

A: 准备一台装了 Docker 的 Linux 服务器，在 GitHub 配置 `DEPLOY_HOST`、`DEPLOY_USER`、`DEPLOY_SSH_KEY`、`DEPLOY_PATH`、`MYSQL_ROOT_PASSWORD` 和 `JWT_SECRET`（长度 ≥32 字节）。推送版本标签（如 `v1.0.0`）或手动运行 `Deploy` workflow，流水线会构建镜像推到 GHCR、SSH 到服务器拉起 `docker-compose.prod.yml` 并做健康检查。前端容器和网关（`deploy/gateway/`）另需部署。完整步骤见 [`docs/deployment.md`](docs/deployment.md)。

**Q: 为什么 Redis 接进来了却没看到缓存代码？**

A: `internal/cache/cache.go` 目前只做「连上 Redis + 启动探活」这一件事。具体的缓存策略——读时 Cache Aside、写时如何让缓存失效、key 怎么设计、TTL 多久、穿透/雪崩怎么防——属于 service 层的业务决策，还没写。这是面试高频点，故意留着自己实现而不是让 AI 代劳。

**Q: 友链功能怎么防止被恶意刷 / 乱改？**

A: 三层防护：①访客自助提交默认 `待审核`，管理员点通过才前台可见，审核前的内容伤害不到站点；②每条友链绑定提交者的 `X-Visitor-ID`，service 层校验归属，改不了别人的；③软删除，误删可恢复。这是"开放提交 + 人工审核"的常见形态，比纯后台 CRUD 多了一层产品视角。

**Q: 数据库表结构改了怎么办？**

A: 开发期用 `AutoMigrate` 自动同步表结构（`database/database.go`）。上线后应换成正式的迁移工具（如 `golang-migrate`）。
