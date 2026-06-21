import { createRouter, createWebHistory } from 'vue-router'

const router = createRouter({
  // Base path comes from Vite's `base` config (set to /admin/ for production),
  // so the admin app lives under /admin while the storefront owns the root.
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/login',
      name: 'Login',
      component: () => import('@/views/LoginView.vue'),
      meta: { public: true }
    },
    {
      path: '/',
      component: () => import('@/layouts/DefaultLayout.vue'),
      children: [
        { path: '', name: 'Dashboard', component: () => import('@/views/DashboardView.vue') },
        { path: 'products', name: 'Products', component: () => import('@/views/ProductsView.vue') },
        { path: 'customers', name: 'Customers', component: () => import('@/views/CustomersView.vue') },
        { path: 'orders', name: 'Orders', component: () => import('@/views/OrdersView.vue') },
      ]
    }
  ]
})

router.beforeEach((to) => {
  const token = localStorage.getItem('token')
  if (!to.meta.public && !token) {
    return '/login'
  }
})

export default router
