// src/components/Login.tsx
import React, { useState } from 'react';
import { Button, Form, Input, Card, message } from 'antd';
import { UserOutlined, LockOutlined } from '@ant-design/icons';
import { authAPI } from '../services/api';
import FileManager from './FileManager'; // 导入文件管理组件

const Login: React.FC = () => {
  const [loading, setLoading] = useState(false);
  const [isLoggedIn, setIsLoggedIn] = useState(false); // 新增登录状态

  const onFinish = async (values: any) => {
    setLoading(true);
    try {
      const response = await authAPI.login(values.email, values.password);
      localStorage.setItem('access_token', response.data.access_token);
      message.success('登录成功！');
      setIsLoggedIn(true); // 更新登录状态
    } catch (error) {
      message.error('登录失败，请检查邮箱和密码');
    } finally {
      setLoading(false);
    }
  };

  // 如果已登录，显示文件管理界面
  if (isLoggedIn) {
    return <FileManager />;
  }

  // 否则显示登录界面
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