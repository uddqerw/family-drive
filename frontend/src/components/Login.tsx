import React, { useState, useEffect } from 'react';
import { Button, Form, Input, Card, message, Tabs } from 'antd';
import { UserOutlined, LockOutlined, MailOutlined } from '@ant-design/icons';
import { authAPI } from '../services/api';

// 定义props接口
interface LoginProps {
  children?: React.ReactNode;
}

const Login: React.FC<LoginProps> = (props) => {
  const [loading, setLoading] = useState(false);
  const [isLoggedIn, setIsLoggedIn] = useState(false);
  const [activeTab, setActiveTab] = useState('login');

  // 检查是否已登录
  useEffect(() => {
    const token = localStorage.getItem('access_token');
    console.log('启动时检查Token:', token);
    if (token) {
      console.log('发现已保存的Token，自动登录');
      setIsLoggedIn(true);
    }
  }, []);

  const onLoginFinish = async (values: any) => {
    setLoading(true);
    try {
      const response = await authAPI.login(values.email, values.password);
      const token = response.data.access_token;
      console.log('登录成功，Token:', token);

      // 保存到localStorage
      localStorage.setItem('access_token', token);
      console.log('Token已保存到localStorage');

      message.success('登录成功！');
      setIsLoggedIn(true);

    } catch (error: any) {
      console.error('登录失败:', error);
      message.error('登录失败: ' + (error.response?.data?.error || '未知错误'));
    } finally {
      setLoading(false);
    }
  };

  const onRegisterFinish = async (values: any) => {
    setLoading(true);
    try {
      // 检查密码确认
      if (values.password !== values.confirmPassword) {
        message.error('两次输入的密码不一致');
        setLoading(false);
        return;
      }

      // 调用注册API
      const response = await authAPI.register(values.username, values.email, values.password);
      
      console.log('注册成功:', response);
      message.success('注册成功！请登录');

      // 注册成功后切换到登录标签
      setActiveTab('login');

    } catch (error: any) {
      console.error('注册失败:', error);
      message.error('注册失败: ' + (error.response?.data?.error || '未知错误'));
    } finally {
      setLoading(false);
    }
  };

  // 如果已登录，显示子组件（标签页）
  if (isLoggedIn) {
    return <>{props.children}</>;
  }

  // 显示登录/注册界面
  return (
    <div style={{
      display: 'flex',
      justifyContent: 'center',
      alignItems: 'center',
      height: '100vh',
      background: '#f0f2f5'
    }}>
      <Card title="🏠 家庭网盘" style={{ width: 400 }}>
        <Tabs 
          activeKey={activeTab} 
          onChange={setActiveTab}
          items={[
            {
              key: 'login',
              label: '登录',
              children: (
                <Form name="login" onFinish={onLoginFinish} autoComplete="off">
                  <Form.Item 
                    name="email" 
                    rules={[
                      { required: true, message: '请输入邮箱!' },
                      { type: 'email', message: '请输入有效的邮箱地址!' }
                    ]}
                  >
                    <Input 
                      prefix={<MailOutlined />} 
                      placeholder="邮箱" 
                      size="large" 
                    />
                  </Form.Item>

                  <Form.Item 
                    name="password" 
                    rules={[{ required: true, message: '请输入密码!' }]}
                  >
                    <Input.Password 
                      prefix={<LockOutlined />} 
                      placeholder="密码" 
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
                      登录
                    </Button>
                  </Form.Item>
                </Form>
              ),
            },
            {
              key: 'register',
              label: '注册',
              children: (
                <Form name="register" onFinish={onRegisterFinish} autoComplete="off">
                  <Form.Item 
                    name="username" 
                    rules={[
                      { required: true, message: '请输入用户名!' },
                      { min: 2, message: '用户名至少2个字符!' }
                    ]}
                  >
                    <Input 
                      prefix={<UserOutlined />} 
                      placeholder="用户名" 
                      size="large" 
                    />
                  </Form.Item>

                  <Form.Item 
                    name="email" 
                    rules={[
                      { required: true, message: '请输入邮箱!' },
                      { type: 'email', message: '请输入有效的邮箱地址!' }
                    ]}
                  >
                    <Input 
                      prefix={<MailOutlined />} 
                      placeholder="邮箱" 
                      size="large" 
                    />
                  </Form.Item>

                  <Form.Item 
                    name="password" 
                    rules={[
                      { required: true, message: '请输入密码!' },
                      { min: 6, message: '密码至少6位!' }
                    ]}
                  >
                    <Input.Password 
                      prefix={<LockOutlined />} 
                      placeholder="密码" 
                      size="large" 
                    />
                  </Form.Item>

                  <Form.Item 
                    name="confirmPassword" 
                    rules={[{ required: true, message: '请确认密码!' }]}
                  >
                    <Input.Password 
                      prefix={<LockOutlined />} 
                      placeholder="确认密码" 
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
                      注册
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
            style={{ padding: '0 4px', height: 'auto' }}
          >
            {activeTab === 'login' ? '立即注册' : '立即登录'}
          </Button>
        </div>
      </Card>
    </div>
  );
};

export default Login;