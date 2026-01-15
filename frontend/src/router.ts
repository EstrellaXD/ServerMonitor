import { createRouter, createWebHistory } from 'vue-router'
import Dashboard from './views/Dashboard.vue'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/',
      name: 'dashboard',
      component: Dashboard,
    },
    {
      path: '/system/:id',
      name: 'system-detail',
      component: () => import('./views/SystemDetail.vue'),
    },
  ],
})

export default router
