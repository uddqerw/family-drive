// src/services/api.tsx
import axios from 'axios';

const API_BASE = 'http://localhost:8000/api';

const api = axios.create({
  baseURL: API_BASE,
});

// 请求拦截器 - 自动添加token
api.interceptors.request.use((config) => {
  const token = localStorage.getItem('access_token');
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

export const authAPI = {
  login: (email: string, password: string) => 
    api.post('/auth/login', { email, password }),
  
  register: (name: string, email: string, password: string) =>
    api.post('/auth/register', { name, email, password }),
  
  getMe: () => api.get('/auth/me'),
};

export const fileAPI = {
  upload: (formData: FormData) =>
    api.post('/files/upload', formData, {
      headers: { 'Content-Type': 'multipart/form-data' },
    }),
  
  list: () => api.get('/files/list'),
  
  download: (filename: string) =>
    api.get(`/files/download/${filename}`, { responseType: 'blob' }),
  
  delete: (filename: string) =>
    api.delete(`/files/delete/${filename}`),
};