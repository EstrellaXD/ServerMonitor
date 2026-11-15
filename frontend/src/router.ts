import { createRouter, createWebHistory } from 'vue-router'
import Overview from './views/Overview.vue'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/',
      name: 'dashboard',
      component: Overview,
    },
    {
      path: '/system/:id',
      name: 'system-detail',
      component: () => import('./views/SystemDetail.vue'),
    },
  ],
})

export default router
