# 开发计划

## 1. 概述

本文档基于 [PLAN.md](docs/PLAN.md) 的功能规划，结合当前项目现有进度，制定后续开发计划。

## 2. 当前项目进度

### 已完成功能

| 功能模块       | 完成情况 | 备注                                                           |
| -------------- | -------- | -------------------------------------------------------------- |
| 三栏布局       | ✅ 完成  | 左栏订阅源(280px) + 中栏文章列表(350px) + 右栏文章内容(自适应) |
| 订阅源管理     | ✅ 完成  | 添加、删除订阅源，显示名称和 URL                               |
| 文章列表       | ✅ 完成  | 显示标题、发布时间和描述                                       |
| 文章阅读       | ✅ 完成  | Markdown/HTML/纯文本渲染，标题链接指向原文                      |
| AI 摘要        | ✅ 完成  | 调用 OpenAI API 生成摘要                                       |
| 本地摘要       | ✅ 完成  | TextRank + MMR 算法                                            |
| RSS 定时更新   | ✅ 完成  | 每分钟自动更新                                                 |
| 文章发布时间   | ✅ 完成  | Article 模型添加 PublishedAt，支持相对时间显示                 |
| 订阅源分组     | ✅ 完成  | 创建/编辑/删除分组，拖拽订阅源到分组                           |
| 收藏夹         | ✅ 完成  | 收藏/取消收藏，显示收藏数量角标                                |
| 响应式布局     | ✅ 完成  | 移动端适配                                                     |
| 内容格式检测   | ✅ 完成  | 自动检测 HTML/Markdown/纯文本                                  |
| XSS 安全防护   | ✅ 完成  | DOMPurify 消毒 HTML 内容                                       |
| AI 摘要持久化   | ✅ 完成  | 摘要保存到数据库，重新生成功能                                 |

### 待完善功能

| 功能模块 | 状态 | 备注 |
| -------- | ---- | ---- |
| 用户认证 | ❌ 未完成 | 多用户支持预留 |

## 3. 未来开发计划

### Phase 1: 文章列表完善 ✅ 已完成

**目标**：完善文章列表显示，包括发布时间

**任务**：

- [x] 1.1 后端：Article 模型添加发布时间字段 (`PublishedAt`)
- [x] 1.2 后端：解析 RSS 时提取文章发布时间 (`item.Published`)
- [x] 1.3 前端：ArticleList 组件显示发布时间
- [x] 1.4 前端：时间格式化显示 (如 "2 小时前"、"昨天" 等)

**相关文件**：

- `backend/main.go` - 修改 Article 结构体和解析逻辑
- `frontend/src/components/ArticleList.vue` - 添加发布时间显示

---

### Phase 2: 订阅源分组功能 ✅ 已完成

**目标**：支持用户将订阅源分组管理

**任务**：

- [x] 2.1 后端：创建 Group 模型 (分组表)

  ```go
  type Group struct {
      ID        primitive.ObjectID `json:"id" bson:"_id,omitempty"`
      Name      string             `json:"name" bson:"name"`
      CreatedAt time.Time         `json:"createdAt" bson:"createdAt"`
  }
  ```

- [x] 2.2 后端：修改 FeedSource 模型，添加 GroupID 字段

  ```go
  type FeedSource struct {
      // ... 现有字段
      GroupID   primitive.ObjectID `json:"groupId" bson:"groupId"`
  }
  ```

- [x] 2.3 后端：添加分组 API

  - `POST /api/groups` - 创建分组
  - `GET /api/groups` - 获取所有分组
  - `PUT /api/groups/:id` - 更新分组
  - `DELETE /api/groups/:id` - 删除分组
  - `PUT /api/sources/:id/group` - 将订阅源分配到分组

- [x] 2.4 前端：SourceList 组件添加分组 UI
  - 分组列表展示
  - 添加/编辑/删除分组
  - 拖拽订阅源到分组

**相关文件**：

- `backend/main.go` - 添加分组 API
- `backend/db/db.go` - 添加 groups 集合
- `frontend/src/components/SourceList.vue` - 分组 UI

---

### Phase 3: 收藏夹功能 ✅ 已完成

**目标**：实现文章收藏功能，收藏夹作为独立分组置顶显示

**任务**：

- [x] 3.1 后端：Article 模型添加收藏字段

  ```go
  type Article struct {
      // ... 现有字段
      IsStarred  bool      `json:"isStarred" bson:"isStarred"`
      StarredAt  time.Time `json:"starredAt" bson:"starredAt"`
  }
  ```

- [x] 3.2 后端：添加收藏 API

  - `POST /api/articles/:id/star` - 收藏文章
  - `DELETE /api/articles/:id/star` - 取消收藏
  - `GET /api/articles/starred` - 获取所有收藏的文章

- [x] 3.3 前端：ArticleView 组件添加收藏按钮

  - 按钮位置：文章标题旁边
  - 点击切换收藏状态
  - 已收藏状态显示不同图标

- [x] 3.4 前端：SourceList 添加收藏夹分组
  - 固定置顶显示
  - 显示收藏数量角标
  - 点击显示收藏文章列表

**相关文件**：

- `backend/main.go` - 添加收藏 API
- `frontend/src/components/ArticleView.vue` - 收藏按钮
- `frontend/src/components/SourceList.vue` - 收藏夹分组

---

### Phase 4: 响应式布局优化 ✅ 已完成

**目标**：确保在不同设备上良好显示

**任务**：

- [x] 4.1 分析移动端场景下的三栏布局适配方案
- [x] 4.2 使用抽屉式导航和标签页切换
- [x] 4.3 添加响应式断点样式

**相关文件**：

- `frontend/src/App.vue` - 调整布局
- `frontend/src/style.css` - 添加响应式样式

---

### Phase 5: 用户认证系统 ❌ 进行中

**目标**：实现 JWT + bcrypt 多用户认证系统，为现有数据模型添加用户隔离

**技术方案**：

- **密码加密**: bcrypt (cost 12)
- **Token**: JWT access token (15分钟) + refresh token (7天)
- **认证架构**: 公开路由 `/api/auth/*`，保护路由需携带有效 JWT

**任务**：

#### 5.1 后端 - User 模型与数据库

- [ ] 5.1.1 创建 `backend/models/user.go` - User 模型定义

  ```go
  type User struct {
      ID           primitive.ObjectID `json:"id" bson:"_id,omitempty"`
      Email        string             `json:"email" bson:"email"`
      Username     string             `json:"username" bson:"username"`
      PasswordHash string             `json:"-" bson:"passwordHash"`
      CreatedAt    time.Time          `json:"createdAt" bson:"createdAt"`
      UpdatedAt    time.Time          `json:"updatedAt" bson:"updatedAt"`
  }
  ```

- [ ] 5.1.2 修改 `backend/db/db.go` - 添加 UserCollection 和 email 唯一索引

- [ ] 5.1.3 修改 `backend/main.go` - 为 Group、FeedSource、Article 添加 UserID 字段

  ```go
  type Group struct {
      // ... 现有字段
      UserID primitive.ObjectID `json:"userId" bson:"userId"`
  }
  type FeedSource struct {
      // ... 现有字段
      UserID primitive.ObjectID `json:"userId" bson:"userId"`
  }
  type Article struct {
      // ... 现有字段
      UserID primitive.ObjectID `json:"userId" bson:"userId"`
  }
  ```

#### 5.2 后端 - 认证中间件与 Handler

- [ ] 5.2.1 创建 `backend/middleware/auth.go` - JWT 验证中间件

  - 从 `Authorization: Bearer <token>` 提取 token
  - 验证 token 签名和过期时间
  - 验证通过后将 userID 注入 `c.Set("userID", userID)`
  - 验证失败返回 401

- [ ] 5.2.2 创建 `backend/handlers/auth.go` - 认证 API Handler

  | 端点 | 方法 | 说明 |
  |------|------|------|
  | `/api/auth/register` | POST | 注册 (email, username, password) |
  | `/api/auth/login` | POST | 登录，返回 accessToken + refreshToken |
  | `/api/auth/refresh` | POST | 刷新令牌 |
  | `/api/auth/logout` | POST | 登出 |
  | `/api/auth/me` | GET | 获取当前用户信息 |

- [ ] 5.2.3 修改 `backend/main.go` - 添加认证路由和保护现有 API

  - 添加公开路由 `/api/auth/*` (无需认证)
  - 现有 `/api/sources`, `/api/groups`, `/api/articles` 应用 AuthMiddleware
  - 所有数据库查询添加 `userId` 过滤条件

- [ ] 5.2.4 添加依赖

  ```bash
  go get github.com/golang-jwt/jwt/v5
  go get golang.org/x/crypto/bcrypt
  ```

#### 5.3 前端 - 认证状态与 UI

- [ ] 5.3.1 创建 `frontend/src/composables/useAuth.js` - 认证状态管理

  - 使用 Vue 3 Composition API `provide/inject` 模式
  - 管理 user, accessToken, refreshToken 状态
  - 提供 login, logout, fetchCurrentUser, refreshAccessToken 方法
  - Token 存储在 localStorage

- [ ] 5.3.2 创建 `frontend/src/utils/api.js` - 带认证的 fetch 封装

  - `fetchWithAuth()` 自动携带 Authorization 头
  - 401 响应时自动刷新 token 并重试

- [ ] 5.3.3 创建 `frontend/src/components/AuthForm.vue` - 登录/注册组件

  - 登录/注册表单切换
  - 表单验证 (email 格式, 密码最少 6 位)

- [ ] 5.3.4 修改 `frontend/src/main.js` - 导入并调用 `provideAuth()`

- [ ] 5.3.5 修改 `frontend/src/App.vue`

  - 未登录时显示 AuthForm
  - 已登录显示主界面三栏布局
  - `onMounted` 时自动检查登录状态

**相关文件**：

- `backend/models/user.go` - 新建
- `backend/middleware/auth.go` - 新建
- `backend/handlers/auth.go` - 新建
- `backend/db/db.go` - 修改
- `backend/main.go` - 修改
- `frontend/src/composables/useAuth.js` - 新建
- `frontend/src/utils/api.js` - 新建
- `frontend/src/components/AuthForm.vue` - 新建
- `frontend/src/App.vue` - 修改
- `frontend/src/main.js` - 修改

**数据迁移**：

现有数据需要一次性迁移添加 userId：

```javascript
// MongoDB shell
db.groups.updateMany({}, { $set: { userId: ObjectId("目标用户ID") } })
db.sources.updateMany({}, { $set: { userId: ObjectId("目标用户ID") } })
db.articles.updateMany({}, { $set: { userId: ObjectId("目标用户ID") } })
```

---

## 4. 任务依赖关系

```
Phase 1 (文章列表) ✅
Phase 2 (订阅源分组) ✅  ←→  Phase 3 (收藏夹) ✅
            ↓                    ↓
        Phase 4 (响应式优化) ✅
```

- Phase 1, 2, 3, 4 均已完成
- 后续可进行多用户认证等高级功能

## 5. 开发建议

### 代码组织建议

1. **后端**：将分组和收藏相关逻辑拆分到独立文件

   ```
   backend/
   ├── main.go
   ├── db/db.go
   ├── handlers/
   │   ├── source.go      # 订阅源相关
   │   ├── group.go       # 分组相关
   │   ├── article.go     # 文章相关
   │   └── starred.go    # 收藏相关
   ```

2. **前端**：考虑将 SourceList 拆分为组件
   ```
   frontend/src/components/
   ├── SourceList.vue        # 主组件
   ├── GroupList.vue         # 分组列表
   ├── SourceItem.vue        # 订阅源项
   └── StarredList.vue       # 收藏夹
   ```

### 测试建议

- 每个 API 端点编写单元测试
- 前端组件添加 E2E 测试
- 特别关注：RSS 解析、数据库操作、收藏状态同步

## 6. 里程碑

| 里程碑 | 目标功能             | 状态 |
| ------ | -------------------- | ---- |
| M1     | 文章列表显示发布时间 | ✅ 完成 |
| M2     | 订阅源分组管理       | ✅ 完成 |
| M3     | 收藏夹功能           | ✅ 完成 |
| M4     | 响应式布局优化       | ✅ 完成 |

---

## 7. 待办清单

### 短期 (当前迭代)

- [x] 添加 Article.PublishedAt 字段
- [x] 解析 RSS 发布时间
- [x] 文章列表显示发布时间

### 中期 (下个迭代)

- [x] 创建 Group 数据模型
- [x] 实现分组 CRUD API
- [x] 修改订阅源添加/编辑支持分组
- [x] 前端分组 UI

### 长期 (后续迭代)

- [ ] 用户认证 (预留)
- [ ] 多用户支持
- [ ] 文章标签和分类
- [ ] 全文搜索
- [ ] OPML 导入/导出
