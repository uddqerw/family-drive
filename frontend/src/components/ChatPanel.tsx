import React, { useState, useEffect, useRef } from 'react';
import { 
  Card, Input, Button, List, Avatar, message as antMessage, 
  Space, Typography, Popconfirm 
} from 'antd';
import { 
  SendOutlined, UserOutlined, MessageOutlined, 
  DeleteOutlined, ExclamationCircleOutlined,
  SyncOutlined
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
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [newMessage, setNewMessage] = useState('');
  const [username, setUsername] = useState(() => {
    return localStorage.getItem('chat_username') || '家庭成员';
  });
  const [loading, setLoading] = useState(false);
  const [syncing, setSyncing] = useState(false);
  const [isClearing, setIsClearing] = useState(false);
  
  const syncIntervalRef = useRef<NodeJS.Timeout>();

  // 同步消息函数
  const syncMessages = async () => {
    if (syncing || isClearing) return;
    
    setSyncing(true);
    try {
      const response = await fetch('http://localhost:8000/api/chat/messages');
      
      if (response.ok) {
        const result = await response.json();
        
        if (result.success && result.data && Array.isArray(result.data)) {
          const formattedMessages = result.data.map((msg: any) => ({
            id: msg.id || Date.now(),
            user_id: msg.user_id || 0,
            username: msg.username || '未知用户',
            content: msg.content || '',
            type: msg.type || 'user',
            timestamp: msg.timestamp || new Date().toLocaleString('zh-CN')
          }));
          
          setMessages(formattedMessages);
          localStorage.setItem('chat_messages', JSON.stringify(formattedMessages));
        }
      }
    } catch (error) {
      console.log('同步失败，使用本地存储');
      try {
        const saved = localStorage.getItem('chat_messages');
        if (saved) {
          const localMessages = JSON.parse(saved);
          if (Array.isArray(localMessages)) {
            setMessages(localMessages);
          }
        }
      } catch (e) {
        // 忽略解析错误
      }
    } finally {
      setSyncing(false);
    }
  };

  // 初始化时同步消息 - 🆕 调整为10秒同步一次
  useEffect(() => {
    syncMessages();
    
    // 🆕 改为10秒同步一次，减少频率
    syncIntervalRef.current = setInterval(syncMessages, 10000);
    
    return () => {
      if (syncIntervalRef.current) {
        clearInterval(syncIntervalRef.current);
      }
    };
  }, []);

  // 保存用户名
  useEffect(() => {
    localStorage.setItem('chat_username', username);
  }, [username]);

  // 发送消息
  const sendMessage = async () => {
    const messageToSend = newMessage.trim();
    if (!messageToSend) {
      antMessage.warning('请输入消息内容');
      return;
    }

    setLoading(true);
    try {
      const messageId = Date.now();
      
      const response = await fetch('http://localhost:8000/api/chat/send', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          username: username,
          content: messageToSend,
          user_id: messageId
        }),
      });

      if (response.ok) {
        setNewMessage('');
        antMessage.success('消息发送成功');
        
        // 发送成功后立即同步一次
        setTimeout(syncMessages, 500);
      } else {
        throw new Error('发送失败');
      }

    } catch (error) {
      console.error('发送消息失败:', error);
      antMessage.error('发送失败，请检查网络连接');
    } finally {
      setLoading(false);
    }
  };

  // 清除消息 - 🆕 修复版本，避免被同步覆盖
  const clearAllMessages = async () => {
    setIsClearing(true); // 🆕 标记正在清除中
    
    try {
      // 🆕 暂停同步
      if (syncIntervalRef.current) {
        clearInterval(syncIntervalRef.current);
      }

      const response = await fetch('http://localhost:8000/api/chat/clear', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
      });

      if (response.ok) {
        // 🆕 立即更新前端显示
        const systemMessage: ChatMessage = {
          id: 1,
          user_id: 1,
          username: '🏠 家庭网盘',
          content: '💬 聊天记录已清空，开始新的对话吧！',
          type: 'system',
          timestamp: new Date().toLocaleString('zh-CN')
        };
        
        setMessages([systemMessage]);
        localStorage.setItem('chat_messages', JSON.stringify([systemMessage]));
        
        antMessage.success('聊天记录已清除');
        
        // 🆕 3秒后恢复同步
        setTimeout(() => {
          syncIntervalRef.current = setInterval(syncMessages, 10000);
          setIsClearing(false);
        }, 3000);
        
      } else {
        throw new Error('清除失败');
      }
    } catch (error) {
      console.error('清除失败:', error);
      antMessage.error('清除失败，请重试');
      
      // 🆕 即使后端失败，也本地清除并恢复同步
      const systemMessage: ChatMessage = {
        id: 1,
        user_id: 1,
        username: '🏠 家庭网盘',
        content: '💬 聊天记录已清空（本地）',
        type: 'system',
        timestamp: new Date().toLocaleString('zh-CN')
      };
      
      setMessages([systemMessage]);
      localStorage.setItem('chat_messages', JSON.stringify([systemMessage]));
      
      // 恢复同步
      setTimeout(() => {
        syncIntervalRef.current = setInterval(syncMessages, 10000);
        setIsClearing(false);
      }, 3000);
    }
  };

  // 处理键盘事件
  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      sendMessage();
    }
  };

  // 手动同步按钮
  const handleManualSync = () => {
    antMessage.info('同步中...');
    syncMessages();
  };

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
            <Button 
              type="text" 
              icon={<SyncOutlined spin={syncing} />} 
              onClick={handleManualSync}
              size="small"
              loading={syncing}
            >
              {syncing ? '同步中' : '同步'}
            </Button>
          </Space>
        }
        className="chat-card"
        extra={
          <Space>
            <Input
              placeholder="你的名字"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              style={{ width: 120 }}
              size="small"
            />
            {messages.length > 0 && (
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
                  loading={isClearing}
                >
                  {isClearing ? '清除中' : '清空'}
                </Button>
              </Popconfirm>
            )}
          </Space>
        }
      >
        <div className="messages-container">
          <List
            dataSource={messages}
            renderItem={renderMessage}
            className="messages-list"
            locale={{ emptyText: '暂无消息，开始聊天吧！' }}
          />
        </div>

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
            💡 10秒自动同步 • 清空时暂停同步 • {messages.length}条消息
          </div>
        </div>
      </Card>
    </div>
  );
};

export default ChatPanel;