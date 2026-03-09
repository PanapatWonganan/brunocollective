<template>
  <div class="login-wrapper">
    <div class="login-left d-none d-md-flex">
      <div class="login-left-content">
        <img src="/brunocollective_logo.jpg" alt="Bruno Collective" class="login-logo" />
        <p class="text-body-1 mt-6" style="color: rgba(255,255,255,0.75); max-width: 300px;">
          Inventory management system for Bruno Collective.
        </p>
      </div>
      <!-- Decorative elements -->
      <div class="deco-circle deco-1" />
      <div class="deco-circle deco-2" />
      <div class="deco-line" />
    </div>

    <div class="login-right">
      <div class="login-form-wrapper">
        <div class="d-md-none text-center mb-8">
          <img src="/brunocollective_logo.jpg" alt="Bruno Collective" style="height: 60px;" />
        </div>

        <h2 class="text-h5 font-weight-bold mb-1" style="color: #1A1714;">Welcome back</h2>
        <p class="text-body-2 mb-8" style="color: #8C8478;">Sign in to your admin account</p>

        <v-form @submit.prevent="handleLogin" ref="form">
          <label class="input-label">USERNAME</label>
          <v-text-field
            v-model="username"
            placeholder="Enter your username"
            prepend-inner-icon="mdi-account-outline"
            :rules="[v => !!v || 'Required']"
            :error-messages="error ? ' ' : ''"
            class="mb-1"
          />
          <label class="input-label">PASSWORD</label>
          <v-text-field
            v-model="password"
            placeholder="Enter your password"
            prepend-inner-icon="mdi-lock-outline"
            :type="showPass ? 'text' : 'password'"
            :append-inner-icon="showPass ? 'mdi-eye-off-outline' : 'mdi-eye-outline'"
            @click:append-inner="showPass = !showPass"
            :rules="[v => !!v || 'Required']"
            :error-messages="error"
            class="mb-2"
          />
          <v-btn
            type="submit"
            block
            size="x-large"
            :loading="loading"
            class="mt-4 login-btn text-none font-weight-bold"
          >
            Sign In
            <v-icon icon="mdi-arrow-right" class="ml-2" />
          </v-btn>
        </v-form>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
const username = ref('')
const password = ref('')
const showPass = ref(false)
const loading = ref(false)
const error = ref('')

async function handleLogin() {
  error.value = ''
  loading.value = true
  try {
    await auth.login(username.value, password.value)
  } catch (e: any) {
    error.value = e.response?.data?.error || 'Login failed'
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.login-wrapper {
  display: flex;
  min-height: 100vh;
}

.login-left {
  flex: 1;
  background: linear-gradient(135deg, #1A1714 0%, #2C2620 40%, #3D3428 100%);
  display: flex;
  align-items: center;
  justify-content: center;
  position: relative;
  overflow: hidden;
}

.login-left-content {
  position: relative;
  z-index: 1;
  text-align: center;
  padding: 2rem;
}

.login-logo {
  height: 100px;
  filter: invert(1);
  mix-blend-mode: screen;
}

/* Decorative */
.deco-circle {
  position: absolute;
  border-radius: 50%;
  border: 1px solid rgba(196, 162, 77, 0.15);
}

.deco-1 {
  width: 400px;
  height: 400px;
  top: -120px;
  right: -120px;
}

.deco-2 {
  width: 300px;
  height: 300px;
  bottom: -80px;
  left: -80px;
}

.deco-line {
  position: absolute;
  bottom: 60px;
  left: 50%;
  transform: translateX(-50%);
  width: 60px;
  height: 2px;
  background: linear-gradient(90deg, transparent, #C4A24D, transparent);
}

.login-right {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 2rem;
  background: #F7F3EE;
}

.login-form-wrapper {
  width: 100%;
  max-width: 400px;
}

.input-label {
  display: block;
  font-size: 11px;
  font-weight: 600;
  letter-spacing: 1px;
  color: #8C8478;
  margin-bottom: 6px;
}

.login-btn {
  background: linear-gradient(135deg, #C4A24D, #D4B96A) !important;
  color: #1A1714 !important;
  letter-spacing: 0.5px;
  box-shadow: 0 4px 16px rgba(196, 162, 77, 0.3) !important;
}

.login-btn:hover {
  box-shadow: 0 6px 24px rgba(196, 162, 77, 0.4) !important;
}
</style>
