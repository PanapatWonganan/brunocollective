<template>
  <v-layout>
    <v-navigation-drawer
      v-model="drawer"
      permanent
      class="sidebar"
      :width="collapsed ? 76 : 260"
    >
      <div class="sidebar-inner d-flex flex-column h-100">
        <!-- Logo -->
        <div class="sidebar-logo" :class="collapsed ? 'pa-4 justify-center' : 'px-5 py-4'">
          <img
            v-if="!collapsed"
            src="/brunocollective_logo.jpg"
            alt="Bruno Collective"
            class="logo-img"
          />
          <div v-else class="logo-mark">BC</div>
        </div>

        <!-- Menu Label -->
        <div v-if="!collapsed" class="menu-label">MENU</div>

        <!-- Nav Items -->
        <div class="flex-grow-1 px-3">
          <template v-for="item in navItems" :key="item.to">
            <v-tooltip :text="item.title" location="end" :disabled="!collapsed">
              <template v-slot:activator="{ props }">
                <router-link :to="item.to" custom v-slot="{ navigate }">
                  <div
                    v-bind="props"
                    class="nav-btn"
                    :class="[
                      isActive(item.to) ? 'nav-btn--active' : '',
                      collapsed ? 'nav-btn--collapsed' : ''
                    ]"
                    @click="navigate"
                  >
                    <v-icon :icon="item.icon" :size="collapsed ? 22 : 20" />
                    <span v-if="!collapsed" class="nav-btn-label">{{ item.title }}</span>
                  </div>
                </router-link>
              </template>
            </v-tooltip>
          </template>
        </div>

        <!-- Bottom Section -->
        <div :class="collapsed ? 'px-3 pb-3' : 'px-3 pb-4'">
          <div class="divider-line mb-3" />

          <!-- User Card (expanded) -->
          <div v-if="!collapsed" class="user-card mb-3">
            <v-avatar color="rgba(196,162,77,0.15)" size="34">
              <v-icon icon="mdi-account" size="18" style="color: #C4A24D;" />
            </v-avatar>
            <div class="ml-3">
              <div class="text-body-2 font-weight-medium text-white">{{ auth.username }}</div>
              <div class="text-caption" style="color: rgba(255,255,255,0.45);">Administrator</div>
            </div>
          </div>

          <!-- User Avatar (collapsed) -->
          <div v-else class="d-flex justify-center mb-3">
            <v-tooltip :text="auth.username" location="end">
              <template v-slot:activator="{ props }">
                <v-avatar v-bind="props" color="rgba(196,162,77,0.15)" size="38" style="cursor: pointer;">
                  <v-icon icon="mdi-account" size="20" style="color: #C4A24D;" />
                </v-avatar>
              </template>
            </v-tooltip>
          </div>

          <!-- Collapse Toggle -->
          <v-tooltip :text="collapsed ? 'Expand' : ''" location="end" :disabled="!collapsed">
            <template v-slot:activator="{ props }">
              <div
                v-bind="props"
                class="nav-btn"
                :class="collapsed ? 'nav-btn--collapsed' : ''"
                @click="collapsed = !collapsed"
              >
                <v-icon :icon="collapsed ? 'mdi-arrow-right' : 'mdi-arrow-left'" :size="collapsed ? 22 : 20" />
                <span v-if="!collapsed" class="nav-btn-label">Collapse</span>
              </div>
            </template>
          </v-tooltip>

          <!-- Logout -->
          <v-tooltip text="Logout" location="end" :disabled="!collapsed">
            <template v-slot:activator="{ props }">
              <div
                v-bind="props"
                class="nav-btn"
                :class="collapsed ? 'nav-btn--collapsed' : ''"
                @click="auth.logout()"
              >
                <v-icon icon="mdi-logout" :size="collapsed ? 22 : 20" />
                <span v-if="!collapsed" class="nav-btn-label">Logout</span>
              </div>
            </template>
          </v-tooltip>
        </div>
      </div>
    </v-navigation-drawer>

    <v-app-bar flat color="transparent" :elevation="0">
      <div class="d-flex align-center px-2 w-100">
        <div>
          <div class="text-h6 font-weight-bold" style="color: #1A1714;">{{ currentPageTitle }}</div>
          <div class="text-caption" style="color: #8C8478;">{{ greeting }}</div>
        </div>
        <v-spacer />
        <v-btn icon variant="text" class="mr-1">
          <v-badge :content="pendingCount" :model-value="pendingCount > 0" color="secondary" floating>
            <v-icon icon="mdi-bell-outline" />
          </v-badge>
        </v-btn>
      </div>
    </v-app-bar>

    <v-main style="background: #F7F3EE;">
      <v-container fluid class="pa-6">
        <router-view />
      </v-container>
    </v-main>
  </v-layout>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import api from '@/services/api'

const auth = useAuthStore()
const route = useRoute()
const drawer = ref(true)
const collapsed = ref(false)
const pendingCount = ref(0)

const navItems = [
  { title: 'Dashboard', icon: 'mdi-view-dashboard-outline', to: '/' },
  { title: 'Products', icon: 'mdi-package-variant', to: '/products' },
  { title: 'Customers', icon: 'mdi-account-group-outline', to: '/customers' },
  { title: 'Orders', icon: 'mdi-receipt-text-outline', to: '/orders' },
]

const currentPageTitle = computed(() => route.name?.toString() || 'Dashboard')

const greeting = computed(() => {
  const hour = new Date().getHours()
  if (hour < 12) return 'Good morning'
  if (hour < 18) return 'Good afternoon'
  return 'Good evening'
})

function isActive(to: string) {
  if (to === '/') return route.path === '/'
  return route.path.startsWith(to)
}

onMounted(async () => {
  try {
    const { data } = await api.get('/dashboard')
    pendingCount.value = data.pending_order_count || 0
  } catch {}
})
</script>

<style scoped>
.sidebar {
  background: linear-gradient(180deg, #1A1714 0%, #2C2620 100%) !important;
  border: none !important;
}

.sidebar-inner {
  overflow-x: hidden;
}

/* ── Logo ── */
.sidebar-logo {
  display: flex;
  align-items: center;
  white-space: nowrap;
  overflow: hidden;
}

.logo-img {
  height: 48px;
  width: auto;
  object-fit: contain;
  filter: invert(1);
  mix-blend-mode: screen;
}

.logo-mark {
  width: 40px;
  height: 40px;
  border-radius: 10px;
  background: linear-gradient(135deg, #C4A24D, #D4B96A);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 14px;
  font-weight: 800;
  letter-spacing: 1px;
  color: #1A1714;
}

/* ── Menu Label ── */
.menu-label {
  font-size: 11px;
  font-weight: 600;
  letter-spacing: 1.2px;
  color: rgba(255,255,255,0.3);
  padding: 0 24px;
  margin-bottom: 8px;
}

/* ── Nav Button ── */
.nav-btn {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 14px;
  margin-bottom: 4px;
  border-radius: 10px;
  color: rgba(255,255,255,0.5);
  cursor: pointer;
  transition: all 0.15s ease;
  white-space: nowrap;
  overflow: hidden;
  user-select: none;
}

.nav-btn:hover {
  color: rgba(255,255,255,0.85);
  background: rgba(255,255,255,0.06);
}

.nav-btn--active {
  color: #1A1714 !important;
  background: linear-gradient(135deg, #C4A24D, #D4B96A) !important;
  box-shadow: 0 2px 10px rgba(196, 162, 77, 0.35);
}

.nav-btn--collapsed {
  justify-content: center;
  padding: 10px;
}

.nav-btn-label {
  font-size: 14px;
  font-weight: 500;
}

/* ── Divider ── */
.divider-line {
  height: 1px;
  background: rgba(255,255,255,0.06);
}

/* ── User Card ── */
.user-card {
  display: flex;
  align-items: center;
  padding: 10px 12px;
  border-radius: 10px;
  background: rgba(255,255,255,0.04);
  overflow: hidden;
  white-space: nowrap;
}
</style>
