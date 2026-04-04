# 开发计划

## 1. 概述

本文档基于 [PLAN.md](docs/PLAN.md) 的功能规划，结合当前项目现有进度，制定后续开发计划。

## 2. 当前项目进度

### 已完成功能

| 功能模块     | 完成情况    | 备注                                                           |
| ------------ | ----------- | -------------------------------------------------------------- |
| 三栏布局     | ✅ 完成     | 左栏订阅源(280px) + 中栏文章列表(350px) + 右栏文章内容(自适应) |
| 订阅源管理   | ✅ 完成     | 添加、删除订阅源，显示名称和 URL                               |
| 文章列表     | ✅ 部分完成 | 显示标题和描述                                                 |
| 文章阅读     | ✅ 完成     | Markdown 渲染，标题链接指向原文                                |
| AI 摘要      | ✅ 完成     | 调用 OpenAI API 生成摘要                                       |
| 本地摘要     | ✅ 完成     | TextRank + MMR 算法                                            |
| RSS 定时更新 | ✅ 完成     | 每分钟自动更新                                                 |

### 待完善功能

| 功能模块           | 状态      | 备注                         |
| ------------------ | --------- | ---------------------------- |
| 文章发布时间显示   | ❌ 未完成 | 需要在文章列表中显示发布时间 |
| 订阅源分组         | ❌ 未完成 | 需要添加分组功能             |
| 收藏夹（独立分组） | ❌ 未完成 | 收藏的文章需要置顶显示       |
| 文章收藏功能       | ❌ 未完成 | 用户可以收藏喜欢的文章       |

## 3. 未来开发计划

### Phase 1: 文章列表完善 (优先级：高)

**目标**：完善文章列表显示，包括发布时间

**任务**：

- [ ] 1.1 后端：Article 模型添加发布时间字段 (`PublishedAt`)
- [ ] 1.2 后端：解析 RSS 时提取文章发布时间 (`item.Published`)
- [ ] 1.3 前端：ArticleList 组件显示发布时间
- [ ] 1.4 前端：时间格式化显示 (如 "2 小时前"、"昨天" 等)

**相关文件**：

- `backend/main.go` - 修改 Article 结构体和解析逻辑
- `frontend/src/components/ArticleList.vue` - 添加发布时间显示

---

### Phase 2: 订阅源分组功能 (优先级：中)

**目标**：支持用户将订阅源分组管理

**任务**：

- [ ] 2.1 后端：创建 Group 模型 (分组表)

  ```go
  type Group struct {
      ID        primitive.ObjectID `json:"id" bson:"_id,omitempty"`
      Name      string             `json:"name" bson:"name"`
      UserID    string             `json:"userId" bson:"userId"` // 预留多用户支持
      SortOrder int                `json:"sortOrder" bson:"sortOrder"`
      CreatedAt time.Time          `json:"createdAt" bson:"createdAt"`
  }
  ```

- [ ] 2.2 后端：修改 FeedSource 模型，添加 GroupID 字段

  ```go
  type FeedSource struct {
      // ... 现有字段
      GroupID   primitive.ObjectID `json:"groupId" bson:"groupId"`
  }
  ```

- [ ] 2.3 后端：添加分组 API

  - `POST /api/groups` - 创建分组
  - `GET /api/groups` - 获取所有分组
  - `PUT /api/groups/:id` - 更新分组
  - `DELETE /api/groups/:id` - 删除分组
  - `PUT /api/sources/:id/group` - 将订阅源分配到分组

- [ ] 2.4 前端：SourceList 组件添加分组 UI
  - 分组列表展示
  - 添加/编辑/删除分组
  - 拖拽订阅源到分组

**相关文件**：

- `backend/main.go` - 添加分组 API
- `backend/db/db.go` - 添加 groups 集合
- `frontend/src/components/SourceList.vue` - 分组 UI

---

### Phase 3: 收藏夹功能 (优先级：中)

**目标**：实现文章收藏功能，收藏夹作为独立分组置顶显示

**任务**：

- [ ] 3.1 后端：Article 模型添加收藏字段

  ```go
  type Article struct {
      // ... 现有字段
      IsStarred  bool      `json:"isStarred" bson:"isStarred"`
      StarredAt  time.Time `json:"starredAt" bson:"starredAt"`
  }
  ```

- [ ] 3.2 后端：添加收藏 API

  - `POST /api/articles/:id/star` - 收藏文章
  - `DELETE /api/articles/:id/star` - 取消收藏
  - `GET /api/articles/starred` - 获取所有收藏的文章

- [ ] 3.3 前端：ArticleView 组件添加收藏按钮

  - 按钮位置：文章标题旁边
  - 点击切换收藏状态
  - 已收藏状态显示不同图标

- [ ] 3.4 前端：SourceList 添加收藏夹分组
  - 固定置顶显示
  - 显示收藏数量角标
  - 点击显示收藏文章列表

**相关文件**：

- `backend/main.go` - 添加收藏 API
- `frontend/src/components/ArticleView.vue` - 收藏按钮
- `frontend/src/components/SourceList.vue` - 收藏夹分组

---

### Phase 4: 响应式布局优化 (优先级：低)

**目标**：确保在不同设备上良好显示

**任务**：

- [ ] 4.1 分析移动端场景下的三栏布局适配方案
- [ ] 4.2 考虑使用抽屉式导航或标签页切换
- [ ] 4.3 添加响应式断点样式

**相关文件**：

- `frontend/src/App.vue` - 调整布局
- `frontend/src/style.css` - 添加响应式样式

---

## 4. 任务依赖关系

```
Phase 1 (文章列表)
    ↓
Phase 2 (订阅源分组)  ←→  Phase 3 (收藏夹)
            ↓                    ↓
        Phase 4 (响应式优化)
```

- Phase 2 和 Phase 3 可以并行开发
- Phase 4 依赖于 Phase 2、3 完成后进行

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

| 里程碑 | 目标功能             | 预计阶段 |
| ------ | -------------------- | -------- |
| M1     | 文章列表显示发布时间 | Phase 1  |
| M2     | 订阅源分组管理       | Phase 2  |
| M3     | 收藏夹功能           | Phase 3  |
| M4     | 响应式布局优化       | Phase 4  |

---

## 7. 待办清单

### 短期 (当前迭代)

- [ ] 添加 Article.PublishedAt 字段
- [ ] 解析 RSS 发布时间
- [ ] 文章列表显示发布时间

### 中期 (下个迭代)

- [ ] 创建 Group 数据模型
- [ ] 实现分组 CRUD API
- [ ] 修改订阅源添加/编辑支持分组
- [ ] 前端分组 UI

### 长期 (后续迭代)

- [ ] 收藏功能后端 API
- [ ] 收藏夹前端 UI
- [ ] 响应式布局适配
- [ ] 用户认证 (预留)
