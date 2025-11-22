import { authAPI } from './api';

class AuthService {
  private token: string | null = null;
  private user: any = null;

  // 登录
  async login(email: string, password: string) {
    try {
      const response = await authAPI.login(email, password);
      const data = response.data;
      
      if (data.success) {
        this.token = data.data.access_token;
        this.user = data.data.user;
        
        // 存储到 localStorage
        localStorage.setItem('access_token', this.token);
        localStorage.setItem('user', JSON.stringify(this.user));
        
        console.log('✅ 登录成功:', this.user);
        
        // 🔥 重要：触发页面跳转
        window.location.href = '/'; // 跳转到首页
        
        return data;
      } else {
        throw new Error(data.message);
      }
    } catch (error: any) {
      console.error('❌ 登录失败:', error);
      throw new Error('登录失败: ' + (error.response?.data?.message || error.message));
    }
  }

  // 注册
  async register(username: string, email: string, password: string) {
    try {
      const response = await authAPI.register(email, password);
      return response.data;
    } catch (error: any) {
      throw new Error('注册失败: ' + (error.response?.data?.message || error.message));
    }
  }

  // 获取当前用户
  getCurrentUser() {
    if (!this.user) {
      const storedUser = localStorage.getItem('user');
      if (storedUser) {
        this.user = JSON.parse(storedUser);
      }
    }
    return this.user;
  }

  // 获取 Token
  getToken() {
    if (!this.token) {
      this.token = localStorage.getItem('access_token');
    }
    return this.token;
  }

  // 检查是否已登录
  isLoggedIn() {
    return !!this.getToken();
  }

  // 登出
  logout() {
    this.token = null;
    this.user = null;
    localStorage.removeItem('access_token');
    localStorage.removeItem('user');
    window.location.href = '/login'; // 跳转到登录页
  }
}

export const authService = new AuthService();