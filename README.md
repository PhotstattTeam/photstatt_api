# ConsiliumOus 后端

项目计划看板（ConsiliumOus）的 Web 后端，Go + Gin + GORM 实现。

## 技术栈

| 组件 | 选型 | 说明 |
| --- | --- | --- |
| HTTP 框架 | Gin | 用户指定 |
| ORM | GORM | 读写数据库 |
| 数据库 | SQLite（本地）/ PostgreSQL（生产） | 见下 |
| 认证 | JWT + bcrypt | 密码哈希、令牌签发 |

## 为什么本地用 SQLite

本地没有 PostgreSQL / Docker，而 SQLite 是**单文件数据库，零安装**，跑起来就能用，最适合学习和开发。
GORM 屏蔽了 SQL 方言差异，上服务器切 PostgreSQL 时只需：

1. `go get gorm.io/driver/postgres`
2. 把 [internal/database/database.go](internal/database/database.go) 里的 `sqlite.Open` 换成 `postgres.Open`
3. 把连接串换成 `postgres://user:pass@host:5432/dbname`

表结构由 [internal/model/models.go](internal/model/models.go) 里的 Go 结构体定义，`AutoMigrate` 自动建表。

## 运行

```bash
go mod tidy   # 首次运行前拉取依赖
go run ./cmd/server
```

默认监听 `:8081`（避开本机已占用的 8080），首次运行会在当前目录生成 `consiliumous.db` 并自动建表。

## 已实现的接口

- `GET /health`、`GET /version` — 健康检查与版本
- `POST /api/auth/register` — 注册（返回 token）
- `POST /api/auth/login` — 登录（返回 token）
- `GET /api/me` — 当前用户
- `GET /api/users/search?q=` — 按姓名/邮箱搜索用户（管理员加入项目用）
- `POST /api/projects` — 创建项目（创建者自动成为管理员）
- `GET /api/projects` — 我参与的项目（跨项目隔离）
- `GET /api/projects/:id` — 项目详情
- `GET /api/projects/:id/members` — 成员列表
- `POST /api/projects/:id/members` — 添加成员
- `DELETE /api/projects/:id/members/:userId` — 移除成员
- `POST /api/projects/:id/admins/:userId/revoke?successor_id=` — 撤销管理员（含"最后一名管理员"保护）
- `POST /api/projects/:id/archive` — 归档项目（管理员）
- `POST /api/projects/:id/unarchive` — 取消归档（管理员）
- `DELETE /api/projects/:id` — 永久删除项目（仅系统管理员）

### 子项目（两级）

- `GET /api/projects/:id/subprojects` — 列表
- `POST /api/projects/:id/subprojects` — 创建（`parent_id` 空=一级，否则挂到一级下）
- `PUT /api/projects/:id/subprojects/:spId` — 更新（全量）
- `DELETE /api/projects/:id/subprojects/:spId` — 删除（其下有 Issue 时禁止）

### Issue 池

- `GET /api/projects/:id/issues` — 列表，支持 `status`/`type`/`priority`/`sub_project_id`/`creator_id`/`keyword`/`sort` 筛选；默认只看"待处理"，`status=all` 看全部
- `POST /api/projects/:id/issues` — 新建（标题/类型/优先级/子项目必填）
- `GET /api/projects/:id/issues/:issueId` — 详情
- `PUT /api/projects/:id/issues/:issueId` — 编辑（管理员，或本人提出且待处理）
- `DELETE /api/projects/:id/issues/:issueId` — 删除（同编辑权限）
- `POST /api/projects/:id/issues/:issueId/status` — 关闭/拒绝（`action`=close|reject，需 `reason`）
- `POST /api/projects/:id/issues/:issueId/comments` — 评论
- `GET /api/projects/:id/issues/:issueId/comments` — 评论列表

### 排期与计划

- `POST /api/projects/:id/schedule` — 勾选若干 Issue 批量排期（n 个 Issue → n 个计划，Issue 置为"已排期"）
- `GET /api/projects/:id/plans` — 计划列表，支持 `sub_project_id`/`owner_id`/`issue_id`/`status`/`keyword` 筛选
- `POST /api/projects/:id/plans` — 直接新建计划（不经过 Issue）
- `GET /api/projects/:id/plans/:planId` — 详情
- `PUT /api/projects/:id/plans/:planId` — 编辑（标题/提示/起止时间/子项目，管理员）
- `PUT /api/projects/:id/plans/:planId/owner` — 分配/变更负责人（管理员）
- `PUT /api/projects/:id/plans/:planId/issue` — 改挂到另一个 Issue
- `DELETE /api/projects/:id/plans/:planId/issue` — 解除与 Issue 的关联
- `PATCH /api/projects/:id/plans/:planId/state` — 负责人更新状态/进度
- `POST /api/projects/:id/plans/:planId/progress/restore-auto` — 恢复进度自动计算
- `POST /api/projects/:id/plans/:planId/cancel` — 取消计划
- `DELETE /api/projects/:id/plans/:planId` — 删除计划

> 删除或取消计划后，若该 Issue 名下已无存活计划，Issue 自动回退"待处理"并留痕（事务内完成）。

### 计划任务

- `GET /api/projects/:id/plans/:planId/tasks` — 任务列表
- `POST /api/projects/:id/plans/:planId/tasks` — 新增（管理员/计划负责人）
- `PUT /api/projects/:id/plans/:planId/tasks/:taskId` — 编辑
- `DELETE /api/projects/:id/plans/:planId/tasks/:taskId` — 删除
- `PATCH /api/projects/:id/plans/:planId/tasks/:taskId/status` — 更新状态（管理员/计划负责人/任务负责人）

> 任务状态变化会自动重算计划进度（已完成/总数）；若进度被人工改过（`progress_manual=true`）则不再自动覆盖，直到"恢复自动"。

### 视图数据接口

- `GET /api/projects/:id/mindmap` — 项目思维导图（项目 → 子项目 → 计划 → 任务），支持 `from`/`to` 阶段过滤
- 计划/Issue 列表已支持 `from`/`to` 时间区间过滤，供甘特图阶段选择器使用

### 个人空间

- `GET /api/me/plans` — 我的个人计划（可筛 `plan_group_id`/`status`/`from`/`to`）
- `POST /api/me/plans` — 新建个人计划（来源=个人自建）
- `GET /api/me/plans/:planId` / `PUT` / `DELETE` — 详情/编辑/删除
- `PATCH /api/me/plans/:planId/state` — 更新状态/进度
- `GET /api/me/assigned-plans` — 虚拟"项目计划组"：分配给我的项目计划（已归档项目不出现）
- `GET/POST/PUT/DELETE /api/me/plan-groups` — 个人计划组（删除组时组内计划变"未分组"）
- `GET /api/me/preferences`、`PUT /api/me/preferences/:key` — 用户偏好（虚拟项目计划组的折叠/排序，仅本人生效）

### 移动端同步

- `POST /api/sync/plans` — 一次性批量上传（按 `client_id` 幂等；`last_synced_at` 冲突检测，策略 `keep_local`/`keep_cloud`/`save_as_new`，默认另存为新计划）
- `POST /api/sync/bind` — 记录绑定后端域名事件（审计）

### 系统管理员

通过环境变量 `SYSTEM_ADMIN_EMAIL`（可配 `SYSTEM_ADMIN_PASSWORD`）在启动时指定，全局兜底角色：可查看/管理任何项目与 Issue，不需加入项目。

- `GET /api/admin/users/:userId/plans` — 查看任意用户的个人计划
- `PUT /api/admin/users/:userId/plans/:planId` — 修改任意用户的个人计划
- `DELETE /api/admin/users/:userId/plans/:planId` — 删除任意用户的个人计划

## 目录结构

```
cmd/server/        程序入口
internal/config/   配置加载
internal/database/ 数据库连接与建表
internal/model/    数据模型（= 表结构）
internal/auth/     JWT + 密码
internal/middleware/ 认证、跨域中间件
internal/service/  业务逻辑与权限校验
internal/handler/  HTTP 处理
internal/router/   路由注册
```

## 当前状态

PRD 首版（MVP）后端已全部实现：认证、项目/子项目/成员/管理员、Issue 池、排期、计划/任务/进度、个人空间、视图数据接口、移动端同步、审计留痕。后续版本内容（计划依赖、审批流、多级子项目、甘特图进度条等）见 PRD 第八节。
