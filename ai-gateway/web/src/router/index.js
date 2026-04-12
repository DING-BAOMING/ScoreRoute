import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '../stores/auth'

const routes = [
  {
    path: '/login',
    name: 'Login',
    component: () => import('../views/Login.vue'),
    meta: { requiresAuth: false }
  },
  {
    path: '/',
    component: () => import('../views/Layout.vue'),
    meta: { requiresAuth: true },
    children: [
      {
        path: '',
        name: 'Dashboard',
        component: () => import('../views/Dashboard.vue')
      },
      {
        path: 'channels',
        name: 'Channels',
        component: () => import('../views/Channels.vue')
      },
      {
        path: 'models',
        name: 'Models',
        component: () => import('../views/Models.vue')
      },
      {
        path: 'tokens',
        name: 'Tokens',
        component: () => import('../views/Tokens.vue')
      },
      {
        path: 'logs',
        name: 'Logs',
        component: () => import('../views/Logs.vue')
      },
      {
        path: 'invocation',
        name: 'Invocation',
        component: () => import('../views/Invocation.vue')
      },
      {
        path: 'model-rating',
        name: 'ModelRating',
        component: () => import('../views/ModelRating.vue')
      },
      {
        path: 'user-rating',
        name: 'UserRating',
        component: () => import('../views/UserRating.vue')
      },
      {
        path: 'sample-analysis',
        name: 'SampleAnalysis',
        component: () => import('../views/SampleAnalysis.vue')
      },
      {
        path: 'extra-rating',
        name: 'ExtraRating',
        component: () => import('../views/ExtraRating.vue')
      },
      {
        path: 'docs',
        name: 'Docs',
        component: () => import('../views/Docs.vue')
      }
    ]
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

router.beforeEach((to, from, next) => {
  const hasToken = !!localStorage.getItem('token')
  
  if (to.meta.requiresAuth !== false && !hasToken) {
    next('/login')
  } else if (to.path === '/login' && hasToken) {
    next('/')
  } else {
    next()
  }
})

export default router
