import axios from 'axios'
import { clearAuthSession, isTokenValid } from '../utils/auth'

const api = axios.create({
  baseURL: 'http://localhost:5000',
})

api.interceptors.request.use((config) => {
  const token = localStorage.getItem('token')
  if (token && isTokenValid(token)) {
    config.headers.Authorization = `Bearer ${token}`
  } else if (token) {
    clearAuthSession()
  }
  const isFormData =
    typeof FormData !== 'undefined' && config.data instanceof FormData
  if (isFormData) {
    delete config.headers['Content-Type']
  } else if (!config.headers['Content-Type']) {
    config.headers['Content-Type'] = 'application/json'
  }
  return config
})

api.interceptors.response.use(
  (response) => response,
  (error) => {
    const status = error?.response?.status
    if (status === 401) {
      clearAuthSession()
      if (window.location.pathname !== '/login') {
        window.location.assign('/login')
      }
    }
    return Promise.reject(error)
  },
)

export default api
