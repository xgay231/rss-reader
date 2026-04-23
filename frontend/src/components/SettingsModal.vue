<script setup>
import { defineProps, defineEmits, ref, watch } from "vue";
import { fetchWithAuth } from "../utils/api";

const props = defineProps({
  show: {
    type: Boolean,
    default: false,
  },
});

const emit = defineEmits(["close"]);

const intervalOption = ref("15");
const customInterval = ref(15);
const autoSummary = ref(true);
const error = ref("");
const loading = ref(false);
const saving = ref(false);

watch(
  () => props.show,
  async (newVal) => {
    if (newVal) {
      await fetchSettings();
    }
  }
);

const fetchSettings = async () => {
  loading.value = true;
  error.value = "";

  try {
    const response = await fetchWithAuth("/api/settings");
    if (response.ok) {
      const data = await response.json();
      const interval = data.feedUpdateInterval;

      if (interval === 15) {
        intervalOption.value = "15";
      } else if (interval === 60) {
        intervalOption.value = "60";
      } else {
        intervalOption.value = "custom";
        customInterval.value = interval;
      }
      autoSummary.value = data.autoSummary;
    } else {
      error.value = "Failed to load settings";
    }
  } catch (e) {
    error.value = "Failed to load settings";
    console.error(e);
  } finally {
    loading.value = false;
  }
};

const handleSave = async () => {
  let feedUpdateInterval;

  if (intervalOption.value === "custom") {
    if (customInterval.value < 1 || customInterval.value > 1440) {
      error.value = "自定义间隔必须在 1-1440 分钟之间";
      return;
    }
    feedUpdateInterval = customInterval.value;
  } else {
    feedUpdateInterval = parseInt(intervalOption.value);
  }

  saving.value = true;
  error.value = "";

  try {
    const response = await fetchWithAuth("/api/settings", {
      method: "PUT",
      body: JSON.stringify({
        feedUpdateInterval,
        autoSummary: autoSummary.value,
      }),
    });

    if (response.ok) {
      emit("close");
    } else {
      error.value = "Failed to save settings";
    }
  } catch (e) {
    error.value = "Failed to save settings";
    console.error(e);
  } finally {
    saving.value = false;
  }
};

const handleClose = () => {
  emit("close");
};
</script>

<template>
  <div v-if="show" class="modal-overlay" @click.self="handleClose">
    <div class="modal-content">
      <div class="modal-header">
        <h2>设置</h2>
        <button class="close-btn" @click="handleClose">&times;</button>
      </div>

      <div class="modal-body">
        <div v-if="loading" class="loading">加载中...</div>

        <div v-else>
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

            <div v-if="intervalOption === 'custom'" class="custom-input">
              <label>输入自定义分钟数:</label>
              <input
                type="number"
                v-model="customInterval"
                min="1"
                max="1440"
                class="interval-input"
              />
              <span>分钟</span>
              <p class="hint">最大值: 1440 分钟</p>
            </div>
          </div>

          <hr class="divider" />

          <div class="setting-group">
            <label class="setting-label">自动摘要</label>
            <div class="checkbox-option">
              <label>
                <input type="checkbox" v-model="autoSummary" />
                <span>对新文章自动生成摘要</span>
              </label>
            </div>
          </div>

          <p v-if="error" class="error">{{ error }}</p>

          <div class="modal-actions">
            <button class="btn-cancel" @click="handleClose" :disabled="saving">
              取消
            </button>
            <button class="btn-save" @click="handleSave" :disabled="saving">
              {{ saving ? "保存中..." : "保存" }}
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  background-color: rgba(0, 0, 0, 0.5);
  display: flex;
  justify-content: center;
  align-items: center;
  z-index: 1000;
}

.modal-content {
  background: white;
  border-radius: 8px;
  width: 100%;
  max-width: 480px;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.15);
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 1rem 1.5rem;
  border-bottom: 1px solid #eee;
}

.modal-header h2 {
  margin: 0;
  font-size: 1.25rem;
  color: #333;
}

.close-btn {
  background: none;
  border: none;
  font-size: 1.5rem;
  cursor: pointer;
  color: #666;
  padding: 0;
  line-height: 1;
}

.close-btn:hover {
  color: #333;
}

.modal-body {
  padding: 1.5rem;
}

.loading {
  text-align: center;
  padding: 2rem;
  color: #666;
}

.setting-group {
  margin-bottom: 1rem;
}

.setting-label {
  display: block;
  font-weight: 500;
  color: #333;
  margin-bottom: 0.75rem;
}

.interval-options {
  display: flex;
  gap: 1rem;
}

.radio-option {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  cursor: pointer;
}

.radio-option input {
  cursor: pointer;
}

.radio-option span {
  color: #333;
}

.custom-input {
  margin-top: 0.75rem;
  display: flex;
  align-items: center;
  gap: 0.5rem;
  flex-wrap: wrap;
}

.custom-input label {
  color: #666;
  font-size: 0.9rem;
}

.interval-input {
  width: 80px;
  padding: 0.5rem;
  border: 1px solid #ddd;
  border-radius: 4px;
  font-size: 1rem;
}

.hint {
  width: 100%;
  margin: 0.25rem 0 0 0;
  font-size: 0.85rem;
  color: #999;
}

.divider {
  border: none;
  border-top: 1px solid #eee;
  margin: 1.5rem 0;
}

.checkbox-option {
  display: flex;
  align-items: center;
}

.checkbox-option label {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  cursor: pointer;
}

.checkbox-option input {
  cursor: pointer;
}

.error {
  color: #e74c3c;
  margin: 1rem 0;
  font-size: 0.9rem;
}

.modal-actions {
  display: flex;
  justify-content: flex-end;
  gap: 0.75rem;
  margin-top: 1.5rem;
}

.btn-cancel,
.btn-save {
  padding: 0.6rem 1.2rem;
  border-radius: 4px;
  font-size: 0.95rem;
  cursor: pointer;
  border: none;
}

.btn-cancel {
  background: #e0e0e0;
  color: #333;
}

.btn-cancel:hover:not(:disabled) {
  background: #d0d0d0;
}

.btn-save {
  background: #4a90d9;
  color: white;
}

.btn-save:hover:not(:disabled) {
  background: #3a7bc8;
}

.btn-cancel:disabled,
.btn-save:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}
</style>