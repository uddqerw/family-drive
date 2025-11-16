import React, { useState, useEffect } from 'react';
import { Button, Form, Input, Card, message } from 'antd';
import { UserOutlined, LockOutlined } from '@ant-design/icons';
import { authAPI } from '../services/api';
import FileManager from './FileManager';

const Login: React.FC = () => {
  const [loading, setLoading] = useState(false);
  const [isLoggedIn, setIsLoggedIn] = useState(false);

  // 检查是否已登录
  useEffect(() => {
    const token = localStorage.getItem('access_token');
    console.log('启动时检查Token:', token);
    if (token) {
      console.log('发现已保存的Token，自动登录');
      setIsLoggedIn(true);
    }
  }, []);

  const onFinish = async (values: any) => {
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

  // 如果已登录，显示文件管理界面
  if (isLoggedIn) {
    return <FileManager />;
  }

  // 显示登录界面
  return (
    <div style={{ 
      display: 'flex', 
      justifyContent: 'center', 
      alignItems: 'center', 
      height: '100vh',
      background: '#f0f2f5'
    }}>
      <Card title="家庭网盘登录" style={{ width: 400 }}>
        <Form name="login" onFinish={onFinish} autoComplete="off">
          <Form.Item name="email" rules={[{ required: true, message: '请输入邮箱!' }]}>
            <Input prefix={<UserOutlined />} placeholder="邮箱" />
          </Form.Item>
          
          <Form.Item name="password" rules={[{ required: true, message: '请输入密码!' }]}>
            <Input.Password prefix={<LockOutlined />} placeholder="密码" />
          </Form.Item>
          
          <Form.Item>
            <Button type="primary" htmlType="submit" loading={loading} style={{ width: '100%' }}>
              登录
            </Button>
          </Form.Item>
        </Form>
      </Card>
    </div>
  );
};

export default Login;