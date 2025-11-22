import React, { useState, useEffect } from 'react';
import { Button, Form, Input, Card, message, Tabs, Space, Typography } from 'antd';
import { UserOutlined, LockOutlined, MailOutlined, SafetyCertificateOutlined } from '@ant-design/icons';
import { authAPI } from '../services/api';

const { Text } = Typography;

// 定义props接口
interface LoginProps {
  children?: React.ReactNode;
  onLoginSuccess?: () => void;
  onLogout?: () => void;
}

const Login: React.FC<LoginProps> = (props) => {
  const [loading, setLoading] = useState(false);
  const [isLoggedIn, setIsLoggedIn] = useState(false);
  const [activeTab, setActiveTab] = useState('login');

  // 检查是否已登录
  useEffect(() => {
    const token = localStorage.getItem('access_token');
    const userInfo = localStorage.getItem('user_info');
    console.log('启动时检查登录状态:', { token, userInfo });

    if (token && userInfo) {
      try {
        const user = JSON.parse(userInfo);
        console.log('发现已保存的用户信息，自动登录:', user);
        setIsLoggedIn(true);
        // 通知父组件登录状态
        if (props.onLoginSuccess) {
          props.onLoginSuccess();
        }
      } catch (error) {
        console.error('解析用户信息失败:', error);
        localStorage.removeItem('access_token');
        localStorage.removeItem('user_info');
      }
    }
  }, [props.onLoginSuccess]);

  // 登录处理
  const onLoginFinish = async (values: any) => {
    setLoading(true);
    try {
      console.log('开始登录:', values.email);

      const response = await authAPI.login(values.email, values.password);
      const data = response.data;
      
      console.log('登录API响应:', data);

      if (data.success) {
        const { access_token, user } = data.data;

        console.log('登录成功，用户信息:', user);

        // 保存到localStorage
        localStorage.setItem('access_token', access_token);
        localStorage.setItem('user_info', JSON.stringify(user));

        message.success(`欢迎回来，${user.username}！`);
        setIsLoggedIn(true);
        
        // 🔥 调用父组件的登录成功回调
        if (props.onLoginSuccess) {
          props.onLoginSuccess();
        }

      } else {
        throw new Error(data.message);
      }

    } catch (error: any) {
      console.error('登录失败:', error);
      const errorMessage = error.response?.data?.message || error.response?.data?.error || error.message || '登录失败，请检查网络连接';
      message.error(errorMessage);
    } finally {
      setLoading(false);
    }
  };

  // 注册处理
  const onRegisterFinish = async (values: any) => {
    // 🔍 添加详细调试信息
    console.log('🔍 注册表单完整数据:', JSON.stringify(values, null, 2));
    console.log('🔍 各个字段值:', {
      username: values.username,
      email: values.email, 
      password: values.password,
      confirmPassword: values.confirmPassword
    });
    console.log('🔍 字段类型:', {
      username_type: typeof values.username,
      email_type: typeof values.email,
      password_type: typeof values.password
    });

    setLoading(true);
    try {
      // 检查密码确认
      if (values.password !== values.confirmPassword) {
        message.error('两次输入的密码不一致');
        setLoading(false);
        return;
      }

      console.log('开始注册 - 用户名:', values.username, '邮箱:', values.email);

      // 🔍 调试API调用参数
      console.log('🔍 调用authAPI.register参数:', {
        username: values.username,
        email: values.email,
        password: values.password
      });

      const response = await authAPI.register(values.username, values.email, values.password);
      const data = response.data;

      console.log('注册API响应:', data);

      if (data.success) {
        message.success('注册成功！请登录');

        // 注册成功后切换到登录标签
        setActiveTab('login');

        // 自动填充登录表单（可选）
        const loginForm = document.querySelector('form[name="login"]') as HTMLFormElement;
        if (loginForm) {
          const emailInput = loginForm.querySelector('input[name="email"]') as HTMLInputElement;
          if (emailInput) {
            emailInput.value = values.email;
          }
        }
      } else {
        throw new Error(data.message);
      }

    } catch (error: any) {
      console.error('注册失败:', error);
      const errorMessage = error.response?.data?.message || error.response?.data?.error || error.message || '注册失败，请重试';
      message.error(errorMessage);
    } finally {
      setLoading(false);
    }
  };

  // 退出登录
  const handleLogout = () => {
    localStorage.removeItem('access_token');
    localStorage.removeItem('user_info');
    setIsLoggedIn(false);
    
    // 🔥 调用父组件的退出回调
    if (props.onLogout) {
      props.onLogout();
    }
    
    message.success('已退出登录');
  };

  // 如果已登录，显示子组件和用户信息
  if (isLoggedIn) {
    const userInfo = JSON.parse(localStorage.getItem('user_info') || '{}');

    return (
      <div>
        {/* 用户信息栏 */}
        <div style={{
          padding: '8px 16px',
          background: '#f0f2f5',
          borderBottom: '1px solid #d9d9d9',
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center'
        }}>
          <Space>
            <UserOutlined />
            <Text strong>欢迎，{userInfo.username || '家庭成员'}</Text>
            <Text type="secondary">{userInfo.email}</Text>
          </Space>
          <Button type="link" onClick={handleLogout} size="small">
            退出登录
          </Button>
        </div>

        {/* 主内容 */}
        {props.children}
      </div>
    );
  }

  // 显示登录/注册界面
  return (
    <div style={{
      display: 'flex',
      justifyContent: 'center',
      alignItems: 'center',
      height: '100vh',
      background: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)'
    }}>
      <Card
        title={
          <Space>
            <SafetyCertificateOutlined />
            <span>🏠 家庭网盘</span>
          </Space>
        }
        style={{
          width: 420,
          boxShadow: '0 8px 32px rgba(0, 0, 0, 0.1)'
        }}
        headStyle={{
          textAlign: 'center',
          fontSize: '20px',
          fontWeight: 'bold'
        }}
      >
        <Tabs
          activeKey={activeTab}
          onChange={setActiveTab}
          items={[
            {
              key: 'login',
              label: '登录',
              children: (
                <Form
                  name="login"
                  onFinish={onLoginFinish}
                  autoComplete="off"
                  layout="vertical"
                >
                  <Form.Item
                    name="email"
                    label="邮箱"
                    rules={[
                      { required: true, message: '请输入邮箱!' },
                      { type: 'email', message: '请输入有效的邮箱地址!' }
                    ]}
                  >
                    <Input
                      prefix={<MailOutlined />}
                      placeholder="请输入邮箱"
                      size="large"
                    />
                  </Form.Item>

                  <Form.Item
                    name="password"
                    label="密码"
                    rules={[{ required: true, message: '请输入密码!' }]}
                  >
                    <Input.Password
                      prefix={<LockOutlined />}
                      placeholder="请输入密码"
                      size="large"
                    />
                  </Form.Item>

                  <Form.Item>
                    <Button
                      type="primary"
                      htmlType="submit"
                      loading={loading}
                      style={{ width: '100%' }}
                      size="large"
                    >
                      {loading ? '登录中...' : '登录'}
                    </Button>
                  </Form.Item>
                </Form>
              ),
            },
            {
              key: 'register',
              label: '注册',
              children: (
                <Form
                  name="register"
                  onFinish={onRegisterFinish}
                  autoComplete="off"
                  layout="vertical"
                  initialValues={{ 
                    username: '',
                    email: '',
                    password: '',
                    confirmPassword: ''
                  }}
                >
                  <Form.Item
                    name="username"
                    label="用户名"
                    rules={[
                      { required: true, message: '请输入用户名!' },
                      { min: 2, message: '用户名至少2个字符!' },
                      { max: 20, message: '用户名不能超过20个字符!' }
                    ]}
                  >
                    <Input
                      prefix={<UserOutlined />}
                      placeholder="请输入用户名"
                      size="large"
                      autoComplete="username"
                    />
                  </Form.Item>

                  <Form.Item
                    name="email"
                    label="邮箱"
                    rules={[
                      { required: true, message: '请输入邮箱!' },
                      { type: 'email', message: '请输入有效的邮箱地址!' }
                    ]}
                  >
                    <Input
                      prefix={<MailOutlined />}
                      placeholder="请输入邮箱"
                      size="large"
                      autoComplete="email"
                    />
                  </Form.Item>

                  <Form.Item
                    name="password"
                    label="密码"
                    rules={[
                      { required: true, message: '请输入密码!' },
                      { min: 6, message: '密码至少6位!' },
                      { pattern: /^(?=.*[A-Za-z])(?=.*\d)/, message: '密码必须包含字母和数字!' }
                    ]}
                  >
                    <Input.Password
                      prefix={<LockOutlined />}
                      placeholder="请输入密码"
                      size="large"
                      autoComplete="new-password"
                    />
                  </Form.Item>

                  <Form.Item
                    name="confirmPassword"
                    label="确认密码"
                    rules={[
                      { required: true, message: '请确认密码!' },
                      ({ getFieldValue }) => ({
                        validator(_, value) {
                          if (!value || getFieldValue('password') === value) {
                            return Promise.resolve();
                          }
                          return Promise.reject(new Error('两次输入的密码不一致!'));
                        },
                      }),
                    ]}
                  >
                    <Input.Password
                      prefix={<LockOutlined />}
                      placeholder="请再次输入密码"
                      size="large"
                      autoComplete="new-password"
                    />
                  </Form.Item>

                  <Form.Item>
                    <Button
                      type="primary"
                      htmlType="submit"
                      loading={loading}
                      style={{ width: '100%' }}
                      size="large"
                    >
                      {loading ? '注册中...' : '注册'}
                    </Button>
                  </Form.Item>
                </Form>
              ),
            },
          ]}
        />

        <div style={{
          textAlign: 'center',
          marginTop: 16,
          color: '#666',
          fontSize: '14px'
        }}>
          {activeTab === 'login' ? '还没有账号？' : '已有账号？'}
          <Button
            type="link"
            onClick={() => setActiveTab(activeTab === 'login' ? 'register' : 'login')}
            style={{ padding: '0 4px', height: 'auto', fontWeight: 'bold' }}
          >
            {activeTab === 'login' ? '立即注册' : '立即登录'}
          </Button>
        </div>

        {/* 演示账号提示 */}
        <div style={{
          marginTop: 16,
          padding: '12px',
          background: '#f6ffed',
          border: '1px solid #b7eb8f',
          borderRadius: '6px',
          fontSize: '12px',
          color: '#52c41a'
        }}>
          <div><strong>演示账号：</strong></div>
          <div>邮箱: test@example.com | 密码: 123456</div>
        </div>
      </Card>
    </div>
  );
};

export default Login;