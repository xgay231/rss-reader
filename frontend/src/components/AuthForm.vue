<template>
  <div class="auth-container">
    <div class="auth-form">
      <h2>{{ isLogin ? '登录' : '注册' }}</h2>
      <form @submit.prevent="handleSubmit">
        <div class="form-group">
          <input
            v-model="form.email"
            type="email"
            placeholder="邮箱"
            required
          />
        </div>
        <div v-if="!isLogin" class="form-group">
          <input
            v-model="form.username"
            type="text"
            placeholder="用户名"
            required
            minlength="2"
          />
        </div>
        <div class="form-group">
          <input
            v-model="form.password"
            type="password"
            placeholder="密码"
            required
            minlength="6"
          />
        </div>
        <button type="submit" :disabled="loading">
          {{ loading ? '处理中...' : (isLogin ? '登录' : '注册') }}
        </button>
      </form>
      <p class="toggle-mode" @click="isLogin = !isLogin">
        {{ isLogin ? '没有账号？立即注册' : '已有账号？立即登录' }}
      </p>
      <p v-if="error" class="error">{{ error }}</p>
      <p v-if="success" class="success">{{ success }}</p>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive } from 'vue'
import { useAuth } from '../composables/useAuth'

const auth = useAuth()

const isLogin = ref(true)
const loading = ref(false)
const error = ref('')
const success = ref('')

const form = reactive({
  email: '',
  username: '',
  password: ''
})

const handleSubmit = async () => {
  error.value = ''
  success.value = ''
  loading.value = true

  try {
    if (isLogin.value) {
      await auth.login(form.email, form.password)
    } else {
      await auth.register(form.email, form.username, form.password)
      success.value = '注册成功！请登录'
      isLogin.value = true
      form.email = ''
      form.username = ''
      form.password = ''
    }
  } catch (e) {
    error.value = e.message
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.auth-container {
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: 100vh;
  background: #f5f5f5;
}

.auth-form {
  background: white;
  padding: 2rem;
  border-radius: 8px;
  box-shadow: 0 2px 10px rgba(0, 0, 0, 0.1);
  width: 100%;
  max-width: 360px;
}

.auth-form h2 {
  text-align: center;
  margin-bottom: 1.5rem;
  color: #333;
}

.form-group {
  margin-bottom: 1rem;
}

.form-group input {
  width: 100%;
  padding: 0.75rem;
  border: 1px solid #ddd;
  border-radius: 4px;
  font-size: 1rem;
  box-sizing: border-box;
}

.form-group input:focus {
  outline: none;
  border-color: #4a90d9;
}

button {
  width: 100%;
  padding: 0.75rem;
  background: #4a90d9;
  color: white;
  border: none;
  border-radius: 4px;
  font-size: 1rem;
  cursor: pointer;
  margin-top: 0.5rem;
}

button:hover:not(:disabled) {
  background: #3a7bc8;
}

button:disabled {
  background: #ccc;
  cursor: not-allowed;
}

.toggle-mode {
  text-align: center;
  margin-top: 1rem;
  color: #4a90d9;
  cursor: pointer;
  font-size: 0.9rem;
}

.toggle-mode:hover {
  text-decoration: underline;
}

.error {
  color: #e74c3c;
  text-align: center;
  margin-top: 1rem;
  font-size: 0.9rem;
}

.success {
  color: #27ae60;
  text-align: center;
  margin-top: 1rem;
  font-size: 0.9rem;
}
</style>
