# Settings Feature Design

## Overview

实现前端设置功能，允许用户配置 feed 更新间隔和自动摘要选项。

## Backend Changes

### Model Changes

**File:** `backend/models/user.go`

新增字段：
```go
type User struct {
    // existing fields...
    FeedUpdateInterval int  `json:"feedUpdateInterval" bson:"feedUpdateInterval"` // minutes, default 15
    AutoSummary        bool `json:"autoSummary" bson:"autoSummary"`               // default true
}
```

### API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/settings` | 获取当前用户设置 |
| PUT | `/api/settings` | 更新用户设置 |

**GET /api/settings Response:**
```json
{
  "feedUpdateInterval": 15,
  "autoSummary": true
}
```

**PUT /api/settings Request:**
```json
{
  "feedUpdateInterval": 60,
  "autoSummary": false
}
```

**Validation:**
- `feedUpdateInterval`: 1-1440 分钟
- `autoSummary`: boolean

### Feed Update Logic

**File:** `backend/main.go`

修改 `updateFeeds()` 函数：
- 读取全局变量或定时器配置的间隔（暂时保持硬编码 1 分钟）
- 新文章插入后，如果 `autoSummary = true`，自动调用摘要生成接口

**注意：** 当前定时器间隔硬编码为 1 分钟，设置变更后需要：
- 方案 A: 重启服务生效（简单）
- 方案 B: 动态修改定时器间隔（复杂，需要 chan 信号）

**建议：本次采用方案 A，后续优化可做动态调整。**

### Summary Generation

在 `updateFeeds()` 中，新文章插入后检查 `autoSummary` 设置，若为 true，调用摘要生成逻辑。

## Frontend Changes

### Settings Modal

**New File:** `frontend/src/components/SettingsModal.vue`

**UI Layout:**
```
┌─────────────────────────────────────┐
│  设置                            ×  │
├─────────────────────────────────────┤
│                                     │
│  Feed 更新间隔                      │
│  ┌─────────────────────────────┐   │
│  │ [15] [60] [自定义 ▼]         │   │
│  └─────────────────────────────┘   │
│  输入自定义分钟数: [____] 分钟      │
│  最大值: 1440 分钟                  │
│                                     │
│  ─────────────────────────────────  │
│                                     │
│  自动摘要                          │
│  ┌─────────────────────────────┐   │
│  │ [✓] 对新文章自动生成摘要     │   │
│  └─────────────────────────────┘   │
│                                     │
│           [取消]  [保存]            │
│                                     │
└─────────────────────────────────────┘
```

**Components:**
- Radio buttons: 15 min, 60 min, 自定义
- When "自定义" selected: show number input with validation (1-1440)
- Toggle switch for auto-summary
- Cancel / Save buttons

### SourceList Changes

**File:** `frontend/src/components/SourceList.vue`

1. 在 "Feeds" 标题下方，添加"立即更新"按钮
2. 点击设置菜单项时，打开 SettingsModal

```vue
<!-- In template, after <h2>Feeds</h2> -->
<button class="refresh-btn" @click="handleRefresh">立即更新</button>
```

### App.vue Changes

**File:** `frontend/src/App.vue`

1. 导入 SettingsModal 组件
2. 添加 `showSettings` ref
3. 在 SourceList 的用户菜单点击"设置"时设置 `showSettings = true`

## Data Flow

1. 用户点击"设置" → 打开 SettingsModal
2. SettingsModal 加载时调用 `GET /api/settings` 获取当前设置
3. 用户修改设置后点击"保存" → 调用 `PUT /api/settings`
4. 成功后关闭弹窗，Settings 数据缓存在前端

## API Utility

**File:** `frontend/src/utils/api.ts`

Add `fetchSettings` function if needed.

## Error Handling

- 网络错误：显示 toast 提示"保存失败，请重试"
- 验证错误：前端实时校验，显示错误提示
- 加载失败：显示默认设置值

## Testing Checklist

- [ ] 点击"设置"菜单能打开设置弹窗
- [ ] 获取设置接口能正确获取默认值
- [ ] 保存设置后重新打开显示新值
- [ ] feed 更新间隔校验 1-1440
- [ ] 立即更新按钮能触发后端更新
- [ ] auto-summary 开关能正确保存
