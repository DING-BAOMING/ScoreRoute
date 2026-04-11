import axios from 'axios'

const api = axios.create({
  baseURL: '/api',
  timeout: 30000
})

api.interceptors.request.use(config => {
  const token = localStorage.getItem('token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

api.interceptors.response.use(
  response => response.data,
  error => {
    if (error.response?.status === 401) {
      localStorage.removeItem('token')
      window.location.href = '/login'
    }
    return Promise.reject(error)
  }
)

export const authAPI = {
  login: (data) => api.post('/auth/login', data),
  validate: () => api.get('/auth/validate')
}

export const channelAPI = {
  list: (params) => api.get('/channels', { params }),
  get: (id) => api.get(`/channels/${id}`),
  create: (data) => api.post('/channels', data),
  update: (id, data) => api.put(`/channels/${id}`, data),
  delete: (id) => api.delete(`/channels/${id}`),
  setEnabled: (id, enabled) => api.put(`/channels/${id}/enabled`, { enabled }),
  fetchModels: (id) => api.get(`/channels/${id}/models`),
  testCredentials: (data) => api.post('/channels/test-credentials', data)
}

export const modelAPI = {
  list: (params) => api.get('/models', { params }),
  create: (data) => api.post('/models', data),
  batchCreate: (data) => api.post('/models/batch', data),
  test: (id) => api.post(`/models/test/${id}`),
  update: (id, data) => api.put(`/models/${id}`, data),
  delete: (id) => api.delete(`/models/${id}`),
  setEnabled: (id, enabled) => api.put(`/models/${id}/enabled`, { enabled }),
  listByChannel: (channelId) => api.get(`/models/channel/${channelId}`)
}

export const tokenAPI = {
  list: (params) => api.get('/tokens', { params }),
  create: (data) => api.post('/tokens', data),
  update: (id, data) => api.put(`/tokens/${id}`, data),
  delete: (id) => api.delete(`/tokens/${id}`),
  setEnabled: (id, enabled) => api.put(`/tokens/${id}/enabled`, { enabled }),
  regenerate: (id) => api.post(`/tokens/${id}/regenerate`)
}

export const logAPI = {
  list: (params) => api.get('/logs', { params }),
  stats: () => api.get('/logs/stats'),
  dashboard: () => api.get('/logs/dashboard'),
  cleanup: (days) => api.delete('/logs/cleanup', { data: { days } }),
  modelStats: () => api.get('/logs/model-stats')
}

export const userRatingAPI = {
  list: () => api.get('/user-ratings'),
  listDeduplicated: () => api.get('/user-ratings?deduplicated=true'),
  upsert: (data) => api.post('/user-ratings', data),
  delete: (id) => api.delete(`/user-ratings/${id}`)
}

export const sampleAPI = {
  list: (params) => api.get('/samples', { params }),
  get: (id) => api.get(`/samples/${id}`),
  stats: () => api.get('/samples/stats'),
  delete: (id) => api.delete(`/samples/${id}`),
  cleanup: () => api.post('/samples/cleanup')
}

export default api
