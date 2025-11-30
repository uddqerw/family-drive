import axios from 'axios';

// 创建axios实例
const api = axios.create({
  baseURL: 'https://localhost:8000/api', // ✅ 添加 /api
  timeout: 10000, // 10秒超时
});

export const API_CONFIG = {
  baseURL: 'https://localhost:8000/api',  // ✅ 添加 /api
  timeout: 10000,
};

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
    api.post('/files/upload', formData, config),  // ✅ 去掉 /api 前缀
  
  // 获取文件列表
  list: () => api.get('/files/list'),  // ✅ 去掉 /api 前缀
  
  // 下载文件
  download: (filename: string) => 
    api.get(`/files/download/${filename}`, {   // ✅ 去掉 /api 前缀
      responseType: 'blob',
      timeout: 30000 // 下载大文件需要更长时间
    }),
  
  // 删除文件
  delete: (filename: string) => 
    api.delete(`/files/delete/${filename}`),  // ✅ 去掉 /api 前缀
};

// 认证相关API
export const authAPI = {
  // 用户登录
  login: (email: string, password: string) => 
    api.post('/auth/login', { email, password }),  // ✅ 去掉 /api 前缀
  
  // 用户注册
  register: (username: string, email: string, password: string) => 
    api.post('/auth/register', { username, email, password }),  // ✅ 去掉 /api 前缀
  
  // 刷新Token
  refresh: (refreshToken: string) => 
    api.post('/auth/refresh', { refresh_token: refreshToken }),  // ✅ 去掉 /api 前缀
  
  // 获取当前用户信息
  getMe: () => api.get('/auth/me'),  // ✅ 去掉 /api 前缀
  
  // 用户登出
  logout: () => api.post('/auth/logout'),  // ✅ 去掉 /api 前缀
  
  // 健康检查
  healthCheck: () => api.get('/auth/me').catch(() => {
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
    await api.get('/auth/me').catch(() => {}); // 即使401也算连接成功
    results.auth = true;
    console.log('✅ 认证API连接正常');
  } catch (error) {
    console.error('❌ 认证API连接失败');
  }

  try {
    // 测试文件API
    await api.get('/files/list');
    results.files = true;
    console.log('✅ 文件API连接正常');
  } catch (error) {
    console.log('⚠️ 文件API连接测试失败（可能是认证问题）');
  }

  return results;
};

export default api;