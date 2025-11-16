import React, { useState, useEffect, useRef } from 'react';
import { Card, Input, Button, List, Avatar, message as antMessage, Space, Typography } from 'antd';
import { SendOutlined, UserOutlined, MessageOutlined } from '@ant-design/icons';
import './ChatPanel.css';

const { TextArea } = Input;
const { Text } = Typography;

interface ChatMessage {
  id: number;
  user_id: number;
  username: string;
  content: string;
  type: 'system' | 'user';
  timestamp: string;
}

const ChatPanel: React.FC = () => {
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [newMessage, setNewMessage] = useState('');
  const [username, setUsername] = useState('家庭成员');
  const [loading, setLoading] = useState(false);
  const messagesEndRef = useRef<HTMLDivElement>(null);

  // 滚动到最新消息
  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  };

  // 加载消息
  const loadMessages = async () => {
    try {
      const response = await fetch('http://localhost:8000/api/chat/messages');
      const data = await response.json();
      if (data.success) {
        setMessages(data.data);
      }
    } catch (error) {
      console.error('加载消息失败:', error);
    }
  };

  // 发送消息
  const sendMessage = async () => {
    if (!newMessage.trim()) {
      antMessage.warning('请输入消息内容');
      return;
    }

    setLoading(true);
    try {
      const response = await fetch('http://localhost:8000/api/chat/send', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          username: username,
          content: newMessage,
          user_id: Date.now()
        }),
      });

      const data = await response.json();
      if (data.success) {
        setNewMessage('');
        await loadMessages();
        antMessage.success('消息发送成功');
      }
    } catch (error) {
      console.error('发送消息失败:', error);
      antMessage.error('发送失败，请重试');
    } finally {
      setLoading(false);
    }
  };

  // 处理回车发送
  const handleKeyPress = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      sendMessage();
    }
  };

  // 初始化
  useEffect(() => {
    loadMessages();
    const interval = setInterval(loadMessages, 3000);
    return () => clearInterval(interval);
  }, []);

  useEffect(() => {
    scrollToBottom();
  }, [messages]);

  const renderMessage = (msg: ChatMessage) => (
    <List.Item className={`message-item ${msg.type === 'system' ? 'system-message' : 'user-message'}`}>
      <List.Item.Meta
        avatar={
          <Avatar 
            icon={msg.type === 'system' ? <MessageOutlined /> : <UserOutlined />}
            style={{
              backgroundColor: msg.type === 'system' ? '#52c41a' : '#1890ff'
            }}
          />
        }
        title={
          <Space>
            <Text strong>{msg.username}</Text>
            <Text type="secondary" style={{ fontSize: '12px' }}>
              {msg.timestamp}
            </Text>
          </Space>
        }
        description={msg.content}
      />
    </List.Item>
  );

  return (
    <div style={{ height: '100%', padding: '16px' }}>
      <Card 
        title={
          <Space>
            <MessageOutlined />
            家庭聊天室
            <Text type="secondary" style={{ fontSize: '12px' }}>
              {messages.length} 条消息
            </Text>
          </Space>
        }
        style={{ height: '100%' }}
        extra={
          <Input
            placeholder="你的名字"
            value={username}
            onChange={(e) => setUsername(e.target.value)}
            style={{ width: 120 }}
            size="small"
          />
        }
      >
        <div style={{ height: 'calc(100% - 120px)', overflow: 'auto', marginBottom: '16px' }}>
          <List
            dataSource={messages}
            renderItem={renderMessage}
            locale={{ emptyText: '暂无消息，开始聊天吧！' }}
          />
          <div ref={messagesEndRef} />
        </div>

        <Space.Compact style={{ width: '100%' }}>
          <TextArea
            value={newMessage}
            onChange={(e) => setNewMessage(e.target.value)}
            onKeyPress={handleKeyPress}
            placeholder="输入消息... (Enter发送)"
            autoSize={{ minRows: 1, maxRows: 4 }}
          />
          <Button 
            type="primary" 
            icon={<SendOutlined />}
            onClick={sendMessage}
            loading={loading}
            style={{ height: 'auto' }}
          >
            发送
          </Button>
        </Space.Compact>
      </Card>
    </div>
  );
};

export default ChatPanel;