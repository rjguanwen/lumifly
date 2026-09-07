# 飞光 Lumifly

以**人生日历**为核心的生活记录与规划应用。将你的一生可视化为网格，记录每一天的点滴、重要里程碑、未来目标和灵光一现的想法。

> 本项目由旧版 **Life Recorder**（Nuxt 3 全栈）重构而来：前端改为 Vue 3 + Element Plus，后端改为 Go（Gin + GORM），数据库仍为 SQLite。**旧版功能全部保留，历史数据（数据库 + 上传的图片/视频）可一键迁移**。

## 功能特性

- **人生日历** - 以周/月为单位的生命网格，颜色编码展示记录密度、重大事件和未来规划
- **日常记录** - 支持富文本编辑、心情选择、标签分类和图片/视频附件
- **人生大事记** - 按时间线展示重要事件，8 大分类 + 5 星重要度评级
- **想法灵感** - 瀑布流卡片布局，快速捕捉灵感
- **速记语录** - 简短文字，快速记录美好词句
- **读书记录** - 阅读记录与书籍推介
- **多用户认证** - JWT + Bearer Token 认证，注册/登录/初始设置流程
- **用户角色与权限** - 管理员可停用/启用普通用户、控制系统注册开关；所有用户可自行修改密码
- **文件上传** - 支持图片（jpg/png/gif/webp）和视频（mp4/webm）本地上传

## 用户与权限

- 账号分为 **管理员** 与 **普通用户**（`users.role`：admin/user）
- 管理员（通过 `.env` 的 `ADMIN_EMAILS` 指定，如 `ADMIN_EMAILS=rjguanwen001@163.com`，启动时自动提升）：
  - 在「用户管理」页停用/启用普通用户（被停用的账号立即无法登录与使用）
  - 开启/关闭系统注册功能（关闭后新用户注册将被拒绝）
- 所有用户可在「设置 → 修改密码」中自行修改登录密码
- 登录接口与每次请求都会校验账号启用状态；管理员专属接口有权限拦截

## 忘记密码的三种找回方式

1. **邮箱自助找回**：登录页「忘记密码」→「通过邮箱找回」。系统向注册邮箱发送含重置链接的邮件（30 分钟有效），点击后设置新密码。
   - 需在 `.env` 配置 SMTP（`SMTP_HOST/PORT/USER/PASS/FROM`）与 `APP_BASE_URL`；
   - 未配置 SMTP 时进入**开发模式**：不真实发信，页面直接给出重置入口，便于本地联调。
2. **安全问答找回**：预先在「设置 → 安全设置」里设置密码提示词与安全问题；忘记密码时回答正确即可重置。
3. **联系管理员**：未设置安全问答且邮箱无法接收邮件时，需管理员协助。

## 技术架构

| 层级 | 技术 |
|------|------|
| 前端 | Vue 3 + Element Plus + Vite + Pinia + TipTap |
| 后端 | Go（Gin + GORM） |
| 数据库 | SQLite |
| 认证 | JWT + bcrypt |

## 目录结构

```
lumifly/
├── backend-go/                 # Go 后端（端口 8004）
│   ├── cmd/server/main.go      # 入口
│   ├── internal/
│   │   ├── config/             # 环境配置
│   │   ├── model/              # GORM 模型（与旧库表结构一致）
│   │   ├── database/           # 数据库连接 + 旧数据自动迁移
│   │   ├── middleware/         # JWT 认证
│   │   └── handler/            # RESTful API
│   ├── data/                   # SQLite 数据库（自动创建）
│   ├── uploads/                # 上传文件（自动创建）
│   └── .env                    # 本地配置
├── frontend/                   # Vue 前端（端口 5176）
│   └── src/
│       ├── api/                # Axios 接口封装
│       ├── router/             # 路由与守卫
│       ├── stores/             # Pinia 状态管理
│       ├── layout/             # 主布局
│       ├── views/              # 页面
│       └── components/         # 共享组件
└── README.md
```

## 快速开始

### 环境要求

- Go >= 1.24
- Node.js >= 18

### 1. 启动后端

```bash
cd backend-go
cp .env.example .env   # 按需修改
go run ./cmd/server
```

### 2. 启动前端

```bash
cd frontend
npm install
npm run dev
```

浏览器访问 `http://localhost:5176`。

## 历史数据迁移

新版应用在**首次启动**时自动完成旧数据迁移，旧版文件不会被修改：

1. **数据库迁移**：在 `backend-go/.env` 中配置：

   ```env
   # 旧版 life-recorder 数据库文件路径
   LEGACY_DB_PATH=D:/path/to/life-recorder/data/life-recorder.db
   # 旧版上传目录（图片/视频）
   LEGACY_UPLOADS_DIR=D:/path/to/life-recorder/uploads
   ```

2. **首次启动后**：
   - 旧数据库被复制为新应用的 `data/lumifly.db`（原库保持不变）
   - 旧上传目录中的图片/视频被复制到 `backend-go/uploads/`
   - 原有账号与密码**直接可用**（bcrypt 哈希兼容），如 `rjguanwen001@163.com`

3. **手动迁移**（可选）：把旧 `life-recorder.db` 复制为 `backend-go/data/lumifly.db`，把旧 `uploads/` 目录复制为 `backend-go/uploads/`，效果相同。

## 数据备份

```bash
# 数据库
cp backend-go/data/lumifly.db backup/

# 上传文件
cp -r backend-go/uploads backup/
```

## API 一览（均以 /api 为前缀）

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /auth/login / /auth/register | 登录 / 注册 |
| GET | /auth/me | 当前用户 |
| PUT | /profile | 更新个人资料 |
| GET/POST | /records、/milestones、/plans、/ideas | 各实体列表 / 新建 |
| GET/PUT/DELETE | /records/:id 等 | 各实体详情 / 更新 / 删除 |
| GET/POST | /tags | 标签列表 / 创建 |
| GET | /calendar/summary | 人生日历汇总数据 |
| POST | /upload | 文件上传 |
| GET | /uploads/*filepath | 上传文件访问 |

## License

GPL
