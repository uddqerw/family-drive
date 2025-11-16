import axios from 'axios';

// 创建axios实例
const api = axios.create({
  baseURL: 'http://localhost:8000', // 后端基础URL
  timeout: 10000, // 10秒超时
});

// 请求拦截器 - 自动添加Token
api.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('access_token');
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    
    // 对于文件上传，使用multipart/form-data
    if (config.data instanceof FormData) {
      config.headers['Content-Type'] = 'multipart/form-data';
    } else {
      config.headers['Content-Type'] = 'application/json';
    }
    
    console.log(`🚀 发送请求: ${config.method?.toUpperCase()} ${config.url}`);
    return config;
  },
  (error) => {
    console.error('❌ 请求配置错误:', error);
    return Promise.reject(error);
  }
);

// 响应拦截器 - 处理Token过期和错误
api.interceptors.response.use(
  (response) => {
    console.log(`✅ 请求成功: ${response.status} ${response.config.url}`);
    return response;
  },
  (error) => {
    console.error('❌ 请求失败:', {
      url: error.config?.url,
      status: error.response?.status,
      message: error.response?.data?.error || error.message
    });
    
    if (error.response?.status === 401) {
      // Token过期或无效
      console.log('🔐 Token已过期，清除本地存储');
      localStorage.removeItem('access_token');
      // 可以在这里跳转到登录页
      window.location.reload();
    }
    
    return Promise.reject(error);
  }
);

// 文件相关API
export const fileAPI = {
  // 上传文件
  upload: (formData: FormData, config?: any) => 
    api.post('/api/files/upload', formData, config),
  
  // 获取文件列表
  list: () => api.get('/api/files/list'),
  
  // 下载文件
  download: (filename: string) => 
    api.get(`/api/files/download/${filename}`, { 
      responseType: 'blob',
      timeout: 30000 // 下载大文件需要更长时间
    }),
  
  // 删除文件
  delete: (filename: string) => 
    api.delete(`/api/files/delete/${filename}`),
};

// 认证相关API
export const authAPI = {
  // 用户登录
  login: (email: string, password: string) => 
    api.post('/api/auth/login', { email, password }),
  
  // 用户注册
  register: (email: string, password: string) => 
    api.post('/api/auth/register', { email, password }),
  
  // 刷新Token
  refresh: (refreshToken: string) => 
    api.post('/api/auth/refresh', { refresh_token: refreshToken }),
  
  // 获取当前用户信息
  getMe: () => api.get('/api/auth/me'),
  
  // 用户登出
  logout: () => api.post('/api/auth/logout'),
  
  // 健康检查
  healthCheck: () => api.get('/api/auth/me').catch(() => {
    // 如果认证检查失败，尝试基础健康检查
    return api.get('/');
  }),
};

// 工具函数：测试所有API连接
export const testAPIConnection = async () => {
  const results = {
    backend: false,
    auth: false,
    files: false,
  };

  try {
    // 测试基础连接
    await api.get('/');
    results.backend = true;
    console.log('✅ 后端服务连接正常');
  } catch (error) {
    console.error('❌ 后端服务连接失败');
  }

  try {
    // 测试认证API
    await api.get('/api/auth/me').catch(() => {}); // 即使401也算连接成功
    results.auth = true;
    console.log('✅ 认证API连接正常');
  } catch (error) {
    console.error('❌ 认证API连接失败');
  }

  try {
    // 测试文件API
    await api.get('/api/files/list');
    results.files = true;
    console.log('✅ 文件API连接正常');
  } catch (error) {
    console.log('⚠️ 文件API连接测试失败（可能是认证问题）');
  }

  return results;
};

export default api;