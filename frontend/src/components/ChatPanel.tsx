import React, { useState, useEffect, useRef } from 'react';
import { 
  Card, Input, Button, List, Avatar, message as antMessage, 
  Space, Typography, Popconfirm 
} from 'antd';
import { 
  SendOutlined, UserOutlined, MessageOutlined, 
  DeleteOutlined, ExclamationCircleOutlined 
} from '@ant-design/icons';
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
  const [messages, setMessages] = useState<ChatMessage[]>(() => {
    // 从localStorage初始化消息，如果没有则使用空数组
    const saved = localStorage.getItem('chat_messages');
    if (saved) {
      try {
        return JSON.parse(saved);
      } catch (e) {
        return [];
      }
    }
    return [];
  });
  const [newMessage, setNewMessage] = useState('');
  const [username, setUsername] = useState(() => {
    return localStorage.getItem('chat_username') || '家庭成员';
  });
  const [loading, setLoading] = useState(false);

  // 当消息或用户名改变时保存到localStorage
  useEffect(() => {
    localStorage.setItem('chat_messages', JSON.stringify(messages));
  }, [messages]);

  useEffect(() => {
    localStorage.setItem('chat_username', username);
  }, [username]);

  // 发送消息 - 完全前端处理
  const sendMessage = async () => {
    const messageToSend = newMessage.trim();
    if (!messageToSend) {
      antMessage.warning('请输入消息内容');
      return;
    }

    setLoading(true);
    try {
      // 先在前端添加消息（立即显示）
      const newMsg: ChatMessage = {
        id: Date.now(),
        user_id: Date.now(),
        username: username,
        content: messageToSend,
        type: 'user',
        timestamp: new Date().toLocaleString('zh-CN')
      };

      setMessages(prev => [...prev, newMsg]);
      setNewMessage('');
      antMessage.success('消息发送成功');

      // 可选：同时发送到后端保存（如果需要多设备同步）
      try {
        await fetch('http://localhost:8000/api/chat/send', {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
          },
          body: JSON.stringify({
            username: username,
            content: messageToSend,
            user_id: Date.now()
          }),
        });
      } catch (error) {
        console.log('后端保存失败，但前端已显示');
      }

    } catch (error) {
      console.error('发送消息失败:', error);
      antMessage.error('发送失败，请重试');
    } finally {
      setLoading(false);
    }
  };

  // 清除所有消息 - 完全前端处理
  const clearAllMessages = () => {
    // 只保留一条系统消息
    const systemMessage: ChatMessage = {
      id: 1,
      user_id: 1,
      username: '🏠 家庭网盘',
      content: '💬 聊天记录已清空，开始新的对话吧！',
      type: 'system',
      timestamp: new Date().toLocaleString('zh-CN')
    };
    
    setMessages([systemMessage]);
    antMessage.success('聊天记录已清除');
  };

  // 处理键盘事件
  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      sendMessage();
    }
  };

  // 初始化 - 不再从后端加载
  useEffect(() => {
    // 如果没有消息，添加一条欢迎消息
    if (messages.length === 0) {
      const welcomeMessage: ChatMessage = {
        id: 1,
        user_id: 1,
        username: '🏠 家庭网盘',
        content: '🎉 欢迎来到家庭聊天室！',
        type: 'system',
        timestamp: new Date().toLocaleString('zh-CN')
      };
      setMessages([welcomeMessage]);
    }
  }, []);

  // 渲染消息项
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
        description={
          <div style={{ 
            whiteSpace: 'pre-wrap',
            wordBreak: 'break-word',
            lineHeight: '1.5'
          }}>
            {msg.content}
          </div>
        }
      />
    </List.Item>
  );

  return (
    <div className="chat-panel">
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
        className="chat-card"
        extra={
          <Space>
            <Input
              placeholder="你的名字"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              onBlur={(e) => {
                localStorage.setItem('chat_username', e.target.value);
              }}
              style={{ width: 120 }}
              size="small"
            />
            {messages.length > 1 && ( // 至少有系统消息+用户消息时才显示清除
              <Popconfirm
                title="清除聊天记录"
                description="确定要清除所有聊天消息吗？此操作不可撤销。"
                icon={<ExclamationCircleOutlined style={{ color: '#ff4d4f' }} />}
                onConfirm={clearAllMessages}
                okText="确定清除"
                cancelText="取消"
                okType="danger"
              >
                <Button 
                  type="default" 
                  icon={<DeleteOutlined />}
                  size="small"
                  danger
                >
                  清空
                </Button>
              </Popconfirm>
            )}
          </Space>
        }
      >
        {/* 消息列表 */}
        <div className="messages-container">
          <List
            dataSource={messages}
            renderItem={renderMessage}
            className="messages-list"
            locale={{ emptyText: '暂无消息，开始聊天吧！' }}
          />
        </div>

        {/* 消息输入框 */}
        <div className="message-input">
          <Space.Compact style={{ width: '100%' }}>
            <TextArea
              value={newMessage}
              onChange={(e) => setNewMessage(e.target.value)}
              onKeyDown={handleKeyDown}
              placeholder="输入消息... (Enter发送，Shift+Enter换行)"
              autoSize={{ minRows: 1, maxRows: 4 }}
              style={{ 
                resize: 'none',
              }}
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
          <div style={{ 
            fontSize: '12px', 
            color: '#999', 
            marginTop: '8px',
            textAlign: 'center'
          }}>
            💡 提示: Enter发送 • Shift+Enter换行 • 消息自动保存
          </div>
        </div>
      </Card>
    </div>
  );
};

export default ChatPanel;