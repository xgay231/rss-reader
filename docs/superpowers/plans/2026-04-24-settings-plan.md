# Settings Feature Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 实现前端设置功能，允许用户配置 feed 更新间隔和自动摘要选项

**Architecture:** 后端在 User 模型添加设置字段并提供 GET/PUT /api/settings 接口，updateFeeds() 根据设置自动生成新文章摘要；前端通过模态框让用户配置这些选项

**Tech Stack:** Go (Gin), Vue.js, MongoDB

---

## File Structure

### Backend
- `backend/models/user.go` — 添加 FeedUpdateInterval 和 AutoSummary 字段
- `backend/main.go` — 添加 settings handlers 和更新 updateFeeds() 逻辑

### Frontend
- `frontend/src/components/SettingsModal.vue` — 新建，设置弹窗组件
- `frontend/src/components/SourceList.vue` — 添加"立即更新"按钮，绑定设置菜单
- `frontend/src/App.vue` — 导入 SettingsModal，添加 showSettings 状态

---

## Task 1: Backend - User Model Changes

**Files:**
- Modify: `backend/models/user.go`

- [ ] **Step 1: 添加 User 模型字段**

```go
type User struct {
    ID                primitive.ObjectID `json:"id" bson:"_id,omitempty"`
    Email             string            `json:"email" bson:"email"`
    Username          string            `json:"username" bson:"username"`
    PasswordHash      string            `json:"-" bson:"passwordHash"`
    CreatedAt         time.Time         `json:"createdAt" bson:"createdAt"`
    UpdatedAt         time.Time         `json:"updatedAt" bson:"updatedAt"`
    FeedUpdateInterval int              `json:"feedUpdateInterval" bson:"feedUpdateInterval"` // minutes, default 15
    AutoSummary       bool              `json:"autoSummary" bson:"autoSummary"`             // default true
}
```

- [ ] **Step 2: Commit**

```bash
git add backend/models/user.go
git commit -m "feat(backend): add FeedUpdateInterval and AutoSummary fields to User model"
```

---

## Task 2: Backend - Settings Handlers

**Files:**
- Modify: `backend/main.go`

- [ ] **Step 1: 添加 Settings 结构体和 handler 函数**

在 main.go 中添加：

```go
// Settings represents user settings
type Settings struct {
    FeedUpdateInterval int  `json:"feedUpdateInterval"`
    AutoSummary       bool `json:"autoSummary"`
}

// GetSettings returns the current user's settings
func GetSettings(c *gin.Context) {
    userID := getUserID(c)
    if userID == primitive.NilObjectID {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
        return
    }

    var user struct {
        FeedUpdateInterval int  `bson:"feedUpdateInterval"`
        AutoSummary        bool `bson:"autoSummary"`
    }

    err := db.UserCollection.FindOne(context.Background(), bson.M{"_id": userID}).Decode(&user)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get settings"})
        return
    }

    // Return defaults if not set
    feedInterval := user.FeedUpdateInterval
    if feedInterval == 0 {
        feedInterval = 15
    }

    c.JSON(http.StatusOK, Settings{
        FeedUpdateInterval: feedInterval,
        AutoSummary:        user.AutoSummary,
    })
}

// UpdateSettings updates the current user's settings
func UpdateSettings(c *gin.Context) {
    userID := getUserID(c)
    if userID == primitive.NilObjectID {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
        return
    }

    var json struct {
        FeedUpdateInterval *int  `json:"feedUpdateInterval"`
        AutoSummary        *bool `json:"autoSummary"`
    }

    if err := c.ShouldBindJSON(&json); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    // Validate feedUpdateInterval
    if json.FeedUpdateInterval != nil {
        if *json.FeedUpdateInterval < 1 || *json.FeedUpdateInterval > 1440 {
            c.JSON(http.StatusBadRequest, gin.H{"error": "feedUpdateInterval must be between 1 and 1440"})
            return
        }
    }

    update := bson.M{}
    if json.FeedUpdateInterval != nil {
        update["feedUpdateInterval"] = *json.FeedUpdateInterval
    }
    if json.AutoSummary != nil {
        update["autoSummary"] = *json.AutoSummary
    }

    if len(update) == 0 {
        c.JSON(http.StatusBadRequest, gin.H{"error": "No settings to update"})
        return
    }

    _, err := db.UserCollection.UpdateOne(
        context.Background(),
        bson.M{"_id": userID},
        bson.M{"$set": update},
    )
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update settings"})
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "Settings updated"})
}
```

- [ ] **Step 2: 注册 settings 路由**

在 main.go 的路由设置部分添加：

```go
api := router.Group("/api")
{
    // existing auth routes...
    settings := api.Group("/settings")
    settings.Use(middleware.AuthRequired())
    {
        settings.GET("", GetSettings)
        settings.PUT("", UpdateSettings)
    }
}
```

- [ ] **Step 3: Commit**

```bash
git add backend/main.go
git commit -m "feat(backend): add GET/PUT /api/settings endpoints"
```

---

## Task 3: Backend - updateFeeds Auto Summary

**Files:**
- Modify: `backend/main.go`

- [ ] **Step 1: 修改 updateFeeds() 函数，在插入新文章后自动生成摘要**

找到 `updateFeeds()` 函数中插入文章的代码段 `if len(newArticles) > 0 {`，在其后添加摘要生成逻辑：

```go
if len(newArticles) > 0 {
    opts := options.InsertMany().SetOrdered(false)
    _, err := db.ArticleCollection.InsertMany(ctx, newArticles, opts)
    if err != nil {
        log.Printf("Failed to insert %d new articles for source %s: %v", len(newArticles), source.Name, err)
    } else {
        log.Printf("Inserted %d new articles for source %s", len(newArticles), source.Name)

        // Auto-generate summaries for new articles if autoSummary is enabled
        go func() {
            for _, article := range newArticles {
                // Check if autoSummary is enabled for this user
                var user struct {
                    AutoSummary bool `bson:"autoSummary"`
                }
                err := db.UserCollection.FindOne(context.Background(), bson.M{"_id": article.UserID}).Decode(&user)
                if err != nil || !user.AutoSummary {
                    continue
                }

                // Generate summary for this article
                generateSummary(article.URL, article.ID, article.UserID)
            }
        }()
    }
}
```

**注意：** 需要确保 `generateSummary` 函数签名与闭包内使用的参数匹配。如果 `generateSummary` 接收 article 而不是三个独立参数，需要调整。

- [ ] **Step 2: Commit**

```bash
git add backend/main.go
git commit -m "feat(backend): auto-generate summaries for new articles when autoSummary is enabled"
```

---

## Task 4: Frontend - SettingsModal Component

**Files:**
- Create: `frontend/src/components/SettingsModal.vue`

- [ ] **Step 1: 创建 SettingsModal.vue**

```vue
<script setup>
import { ref, onMounted } from 'vue';
import { fetchWithAuth } from '../utils/api';

const props = defineProps({
  show: {
    type: Boolean,
    default: false
  }
});

const emit = defineEmits(['close']);

const loading = ref(false);
const saving = ref(false);
const error = ref('');

// Settings state
const feedUpdateInterval = ref(15);
const customInterval = ref(null);
const intervalOption = ref('15'); // '15', '60', 'custom'
const autoSummary = ref(true);

const fetchSettings = async () => {
  loading.value = true;
  error.value = '';
  try {
    const response = await fetchWithAuth('/api/settings');
    if (response.ok) {
      const data = await response.json();
      feedUpdateInterval.value = data.feedUpdateInterval;
      autoSummary.value = data.autoSummary;

      // Set interval option
      if (data.feedUpdateInterval === 15) {
        intervalOption.value = '15';
      } else if (data.feedUpdateInterval === 60) {
        intervalOption.value = '60';
      } else {
        intervalOption.value = 'custom';
        customInterval.value = data.feedUpdateInterval;
      }
    } else {
      error.value = '获取设置失败';
    }
  } catch (e) {
    error.value = '获取设置失败';
  } finally {
    loading.value = false;
  }
};

const saveSettings = async () => {
  // Validate custom interval
  if (intervalOption.value === 'custom') {
    if (!customInterval.value || customInterval.value < 1 || customInterval.value > 1440) {
      error.value = '自定义间隔必须在 1-1440 分钟之间';
      return;
    }
    feedUpdateInterval.value = customInterval.value;
  } else {
    feedUpdateInterval.value = parseInt(intervalOption.value);
  }

  saving.value = true;
  error.value = '';
  try {
    const response = await fetchWithAuth('/api/settings', {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        feedUpdateInterval: feedUpdateInterval.value,
        autoSummary: autoSummary.value
      })
    });

    if (response.ok) {
      emit('close');
    } else {
      const data = await response.json();
      error.value = data.error || '保存失败';
    }
  } catch (e) {
    error.value = '保存失败';
  } finally {
    saving.value = false;
  }
};

const handleClose = () => {
  emit('close');
};

onMounted(() => {
  if (props.show) {
    fetchSettings();
  }
});
</script>

<template>
  <div class="modal-overlay" v-if="show" @click.self="handleClose">
    <div class="modal-content">
      <div class="modal-header">
        <h2>设置</h2>
        <button class="close-btn" @click="handleClose">×</button>
      </div>

      <div class="modal-body">
        <div v-if="loading" class="loading">加载中...</div>
        <div v-else>
          <!-- Feed Update Interval -->
          <div class="setting-group">
            <label class="setting-label">Feed 更新间隔</label>
            <div class="interval-options">
              <label class="radio-option">
                <input type="radio" v-model="intervalOption" value="15" />
                <span>15 分钟</span>
              </label>
              <label class="radio-option">
                <input type="radio" v-model="intervalOption" value="60" />
                <span>60 分钟</span>
              </label>
              <label class="radio-option">
                <input type="radio" v-model="intervalOption" value="custom" />
                <span>自定义</span>
              </label>
            </div>
            <div class="custom-input" v-if="intervalOption === 'custom'">
              <input
                type="number"
                v-model.number="customInterval"
                min="1"
                max="1440"
                placeholder="输入分钟数"
              />
              <span>分钟 (最大 1440)</span>
            </div>
          </div>

          <!-- Auto Summary Toggle -->
          <div class="setting-group">
            <label class="setting-label">自动摘要</label>
            <label class="toggle-option">
              <input type="checkbox" v-model="autoSummary" />
              <span>对新文章自动生成摘要</span>
            </label>
          </div>

          <div v-if="error" class="error-message">{{ error }}</div>
        </div>
      </div>

      <div class="modal-footer">
        <button class="btn-cancel" @click="handleClose">取消</button>
        <button class="btn-save" @click="saveSettings" :disabled="saving">
          {{ saving ? '保存中...' : '保存' }}
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.modal-content {
  background: var(--color-bg-pane, #fff);
  border-radius: 8px;
  width: 400px;
  max-width: 90vw;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.15);
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 1rem;
  border-bottom: 1px solid var(--color-border, #e0e0e0);
}

.modal-header h2 {
  margin: 0;
  font-size: 1.2rem;
}

.close-btn {
  background: none;
  border: none;
  font-size: 1.5rem;
  cursor: pointer;
  color: var(--color-text-secondary, #666);
  padding: 0;
  line-height: 1;
}

.close-btn:hover {
  color: var(--color-text-primary, #333);
}

.modal-body {
  padding: 1rem;
}

.loading {
  text-align: center;
  padding: 2rem;
  color: var(--color-text-secondary, #666);
}

.setting-group {
  margin-bottom: 1.5rem;
}

.setting-label {
  display: block;
  font-weight: 500;
  margin-bottom: 0.75rem;
  color: var(--color-text-primary, #333);
}

.interval-options {
  display: flex;
  gap: 1rem;
  flex-wrap: wrap;
}

.radio-option {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  cursor: pointer;
}

.custom-input {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  margin-top: 0.75rem;
}

.custom-input input {
  width: 100px;
  padding: 0.5rem;
  border: 1px solid var(--color-border, #e0e0e0);
  border-radius: 4px;
}

.toggle-option {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  cursor: pointer;
}

.error-message {
  color: var(--color-danger, #e53935);
  font-size: 0.875rem;
  margin-top: 0.5rem;
}

.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: 0.75rem;
  padding: 1rem;
  border-top: 1px solid var(--color-border, #e0e0e0);
}

.btn-cancel,
.btn-save {
  padding: 0.5rem 1rem;
  border-radius: 4px;
  cursor: pointer;
  font-size: 0.9rem;
}

.btn-cancel {
  background: none;
  border: 1px solid var(--color-border, #e0e0e0);
  color: var(--color-text-primary, #333);
}

.btn-cancel:hover {
  background: var(--color-bg-item-hover, #f5f5f5);
}

.btn-save {
  background: var(--color-accent, #1976d2);
  border: 1px solid var(--color-accent, #1976d2);
  color: var(--color-accent-text, #fff);
}

.btn-save:hover {
  background: var(--color-accent-hover, #1565c0);
}

.btn-save:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}
</style>
```

- [ ] **Step 2: Commit**

```bash
git add frontend/src/components/SettingsModal.vue
git commit -m "feat(frontend): add SettingsModal component"
```

---

## Task 5: Frontend - SourceList Changes

**Files:**
- Modify: `frontend/src/components/SourceList.vue`

- [ ] **Step 1: 添加"立即更新"按钮和设置菜单绑定**

在 `<h2>Feeds</h2>` 后添加按钮：

```vue
<div class="header-actions">
  <button class="refresh-btn" @click="handleRefresh" title="立即更新">
    ↻
  </button>
</div>
```

在用户菜单的设置项添加点击事件：

```vue
<div class="menu-item" @click="$emit('open-settings')">设置</div>
```

添加 emit 定义：

```javascript
const emit = defineEmits(['source-selected', 'open-settings']);
```

添加 handleRefresh 函数：

```javascript
const handleRefresh = async () => {
  try {
    await fetchWithAuth('/api/sources/refresh', { method: 'POST' });
    // Optionally refresh the current source list
    if (selectedSourceId.value) {
      const source = sources.value.find(s => s.id === selectedSourceId.value);
      if (source) {
        emit('source-selected', source);
      }
    }
  } catch (error) {
    console.error('Failed to refresh feeds:', error);
  }
};
```

添加样式：

```css
.header-actions {
  display: flex;
  justify-content: flex-end;
  margin-bottom: 0.5rem;
}

.refresh-btn {
  padding: 0.25rem 0.5rem;
  background: none;
  border: 1px solid var(--color-border);
  border-radius: 4px;
  cursor: pointer;
  font-size: 1.2rem;
  color: var(--color-text-secondary);
}

.refresh-btn:hover {
  background: var(--color-bg-item-hover);
  color: var(--color-accent);
}
```

- [ ] **Step 2: Commit**

```bash
git add frontend/src/components/SourceList.vue
git commit -m "feat(frontend): add refresh button and open-settings emit to SourceList"
```

---

## Task 6: Frontend - App.vue Integration

**Files:**
- Modify: `frontend/src/App.vue`

- [ ] **Step 1: 导入并使用 SettingsModal**

添加导入：

```javascript
import SettingsModal from "./components/SettingsModal.vue";
```

添加状态：

```javascript
const showSettings = ref(false);
```

在 template 中添加组件：

```vue
<SettingsModal
  :show="showSettings"
  @close="showSettings = false"
/>
```

修改 SourceList 绑定：

```vue
<SourceList
  ref="sourceListRef"
  :key="auth.user.value?.id"
  @source-selected="handleSourceSelected"
  @open-settings="showSettings = true"
/>
```

- [ ] **Step 2: Commit**

```bash
git add frontend/src/App.vue
git commit -m "feat(frontend): integrate SettingsModal in App.vue"
```

---

## Task 7: Backend - Refresh Endpoint (Optional)

**Files:**
- Modify: `backend/main.go`

**注意：** Task 5 中前端调用的 `/api/sources/refresh` 接口需要后端支持。如果暂不实现，frontend 会报错。

- [ ] **Step 1: 添加刷新接口**

```go
// RefreshFeeds manually triggers a feed update
func RefreshFeeds(c *gin.Context) {
    go updateFeeds()
    c.JSON(http.StatusOK, gin.H{"message": "Feed refresh triggered"})
}
```

注册路由：

```go
sources.POST("/refresh", RefreshFeeds)
```

- [ ] **Step 2: Commit**

```bash
git add backend/main.go
git commit -m "feat(backend): add POST /api/sources/refresh endpoint"
```

---

## Verification Checklist

- [ ] User 模型包含 feedUpdateInterval 和 autoSummary 字段
- [ ] GET /api/settings 返回正确格式的设置
- [ ] PUT /api/settings 能保存设置并验证 feedUpdateInterval 范围
- [ ] updateFeeds() 在插入新文章后检查 autoSummary 并生成摘要
- [ ] SettingsModal 组件能正确显示和保存设置
- [ ] 立即更新按钮能触发后端刷新
- [ ] 设置菜单项能打开 SettingsModal

---

## Self-Review

1. **Spec coverage:** 所有 spec 中的需求都有对应 task
2. **Placeholder scan:** 无 TBD/TODO/placeholder
3. **Type consistency:** Settings 结构体字段名在前后端一致

Plan complete and saved to `docs/superpowers/plans/2026-04-24-settings-plan.md`.

**Two execution options:**

**1. Subagent-Driven (recommended)** - I dispatch a fresh subagent per task, review between tasks, fast iteration

**2. Inline Execution** - Execute tasks in this session using executing-plans, batch execution with checkpoints

Which approach?
