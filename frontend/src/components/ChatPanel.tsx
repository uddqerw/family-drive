import React, { useState, useEffect, useRef } from 'react';
import {
  Card, Input, Button, List, Avatar, message as antMessage,
  Space, Typography, Popconfirm
} from 'antd';
import {
  SendOutlined, UserOutlined, MessageOutlined,
  DeleteOutlined, ExclamationCircleOutlined,
  SyncOutlined, AudioOutlined, StopOutlined, PlayCircleOutlined
} from '@ant-design/icons';
import './ChatPanel.css';

const { TextArea } = Input;
const { Text } = Typography;

interface ChatMessage {
  id: number;
  user_id: number;
  username: string;
  content: string;
  type: 'system' | 'user' | 'voice';
  timestamp: string;
  voice_url?: string;
  duration?: number;
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
  const [isRecording, setIsRecording] = useState(false);
  const [recordingTime, setRecordingTime] = useState(0);
  const [mediaRecorder, setMediaRecorder] = useState<MediaRecorder | null>(null);
  const [, setAudioChunks] = useState<Blob[]>([]);
  
  const syncIntervalRef = useRef<NodeJS.Timeout | null>(null);
  const recordingIntervalRef = useRef<NodeJS.Timeout | null>(null);

  // 时间格式化函数
  const formatTime = (timestamp: string) => {
    try {
      const date = new Date(timestamp);
      const now = new Date();
      const diffMs = now.getTime() - date.getTime();
      const diffMins = Math.floor(diffMs / 60000);
      const diffHours = Math.floor(diffMs / 3600000);
      
      // 如果是今天
      if (date.toDateString() === now.toDateString()) {
        if (diffMins < 1) return '刚刚';
        if (diffMins < 60) return `${diffMins}分钟前`;
        return `${diffHours}小时前`;
      }
      
      // 如果是昨天
      const yesterday = new Date(now);
      yesterday.setDate(yesterday.getDate() - 1);
      if (date.toDateString() === yesterday.toDateString()) {
        return `昨天 ${date.getHours().toString().padStart(2, '0')}:${date.getMinutes().toString().padStart(2, '0')}`;
      }
      
      // 其他情况
      return `${date.getMonth() + 1}-${date.getDate()} ${date.getHours().toString().padStart(2, '0')}:${date.getMinutes().toString().padStart(2, '0')}`;
    } catch (error) {
      // 如果已经是格式化好的时间，直接返回
      if (timestamp.includes('-') && timestamp.includes(':')) {
        return timestamp;
      }
      return '未知时间';
    }
  };

  // 同步消息函数
  const syncMessages = async () => {
    if (syncing || isClearing) return;

    setSyncing(true);
    try {
      const response = await fetch('https://localhost:8000/api/chat/messages');

      if (response.ok) {
        const result = await response.json();

        if (result.success && result.data && Array.isArray(result.data)) {
          const formattedMessages = result.data.map((msg: any) => ({
            id: msg.id || Date.now(),
            user_id: msg.user_id || 0,
            username: msg.username || '未知用户',
            content: msg.content || '',
            type: msg.type || 'user',
            timestamp: msg.timestamp || new Date().toISOString(),
            voice_url: msg.voice_url,
            duration: msg.duration || 0  // 确保 duration 有默认值
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

  // 初始化时同步消息
  useEffect(() => {
    syncMessages();
    syncIntervalRef.current = setInterval(syncMessages, 5000);

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

  // 开始录音
  const startRecording = async () => {
    try {
      const stream = await navigator.mediaDevices.getUserMedia({ 
        audio: {
          echoCancellation: true,
          noiseSuppression: true,
          sampleRate: 44100,
        } 
      });
      
      const recorder = new MediaRecorder(stream, {
        mimeType: 'audio/webm;codecs=opus'
      });
      
      const chunks: Blob[] = [];
      recorder.ondataavailable = (event) => {
        if (event.data.size > 0) {
          chunks.push(event.data);
        }
      };
      
      recorder.onstop = () => {
        const audioBlob = new Blob(chunks, { type: 'audio/webm' });
        sendVoiceMessage(audioBlob);
        stream.getTracks().forEach(track => track.stop());
      };
      
      recorder.start();
      setMediaRecorder(recorder);
      setAudioChunks(chunks);
      setIsRecording(true);
      setRecordingTime(0);
      
      // 录音计时器
      recordingIntervalRef.current = setInterval(() => {
        setRecordingTime(prev => prev + 1);
      }, 1000);
      
    } catch (error) {
      console.error('无法访问麦克风:', error);
      antMessage.error('无法访问麦克风，请检查权限设置');
    }
  };

  // 停止录音
  const stopRecording = () => {
    if (mediaRecorder && isRecording) {
      mediaRecorder.stop();
      setIsRecording(false);
      if (recordingIntervalRef.current) {
        clearInterval(recordingIntervalRef.current);
      }
    }
  };

  // 发送语音消息
  const sendVoiceMessage = async (audioBlob: Blob) => {
    setLoading(true);
    try {
      const formData = new FormData();
      formData.append('audio', audioBlob, `voice_${Date.now()}.webm`);
      formData.append('username', username);
      formData.append('user_id', Date.now().toString());
      formData.append('duration', recordingTime.toString());

      console.log('🎤 发送语音消息，时长:', recordingTime);

      const response = await fetch('https://localhost:8000/api/chat/voice', {
        method: 'POST',
        body: formData,
      });

      if (response.ok) {
        const result = await response.json();
        console.log('✅ 语音发送成功:', result);
        antMessage.success('语音发送成功');
        setTimeout(syncMessages, 500);
      } else {
        console.error('❌ 语音发送失败，状态码:', response.status);
        throw new Error('发送失败');
      }
    } catch (error) {
      console.error('发送语音失败:', error);
      antMessage.error('语音发送失败，后端服务可能未就绪');
      
      // 降级为文本消息
      const voiceMessage: ChatMessage = {
        id: Date.now(),
        user_id: Date.now(),
        username: username,
        content: `[语音消息 ${recordingTime}秒]`,
        type: 'user',
        timestamp: new Date().toISOString()
      };
      
      setMessages(prev => [...prev, voiceMessage]);
      localStorage.setItem('chat_messages', JSON.stringify([...messages, voiceMessage]));
    } finally {
      setLoading(false);
      setRecordingTime(0);
    }
  };

  // 播放语音消息
  const playVoiceMessage = (audioUrl: string) => {
    const fullUrl = audioUrl.startsWith('http') ? audioUrl : `https://localhost:8000${audioUrl}`;
    const audio = new Audio(fullUrl);
    audio.play().catch(error => {
      console.error('播放失败:', error);
      antMessage.error('播放失败，请检查语音文件');
    });
  };

  // 发送文本消息
  const sendMessage = async () => {
    const messageToSend = newMessage.trim();
    if (!messageToSend) {
      antMessage.warning('请输入消息内容');
      return;
    }

    setLoading(true);
    try {
      const messageId = Date.now();

      const response = await fetch('https://localhost:8000/api/chat/send', {
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

  // 清除消息
  const clearAllMessages = async () => {
    setIsClearing(true);

    try {
      if (syncIntervalRef.current) {
        clearInterval(syncIntervalRef.current);
      }

      const response = await fetch('https://localhost:8000/api/chat/clear', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
      });

      if (response.ok) {
        const systemMessage: ChatMessage = {
          id: 1,
          user_id: 1,
          username: '🏠 家庭网盘',
          content: '💬 聊天记录已清空，开始新的对话吧！',
          type: 'system',
          timestamp: new Date().toISOString()
        };

        setMessages([systemMessage]);
        localStorage.setItem('chat_messages', JSON.stringify([systemMessage]));
        antMessage.success('聊天记录已清除');

        setTimeout(() => {
          syncIntervalRef.current = setInterval(syncMessages, 5000);
          setIsClearing(false);
        }, 3000);

      } else {
        throw new Error('清除失败');
      }
    } catch (error) {
      console.error('清除失败:', error);
      antMessage.error('清除失败，请重试');

      const systemMessage: ChatMessage = {
        id: 1,
        user_id: 1,
        username: '🏠 家庭网盘',
        content: '💬 聊天记录已清空（本地）',
        type: 'system',
        timestamp: new Date().toISOString()
      };

      setMessages([systemMessage]);
      localStorage.setItem('chat_messages', JSON.stringify([systemMessage]));

      setTimeout(() => {
        syncIntervalRef.current = setInterval(syncMessages, 5000);
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
    <List.Item className={`message-item ${msg.type === 'system' ? 'system-message' : 'user-message'} ${msg.type === 'voice' ? 'voice-message' : ''}`}>
      <List.Item.Meta
        avatar={
          <Avatar
            icon={msg.type === 'system' ? <MessageOutlined /> : 
                  msg.type === 'voice' ? <AudioOutlined /> : <UserOutlined />}
            style={{
              backgroundColor: msg.type === 'system' ? '#52c41a' : 
                             msg.type === 'voice' ? '#722ed1' : '#1890ff'
            }}
          />
        }
        title={
          <Space>
            <Text strong>{msg.username}</Text>
            <Text type="secondary" style={{ fontSize: '12px' }}>
              {formatTime(msg.timestamp)}
              {msg.type === 'voice' && msg.duration && msg.duration > 0 && ` • ${msg.duration}秒`}
            </Text>
          </Space>
        }
        description={
          msg.type === 'voice' ? (
            <div className="voice-message-content">
              <Button
                type="text"
                icon={<PlayCircleOutlined />}
                onClick={() => {
                  if (msg.voice_url) {
                    playVoiceMessage(msg.voice_url);
                  } else {
                    antMessage.warning('语音文件不存在');
                  }
                }}
                style={{ color: '#722ed1' }}
                disabled={!msg.voice_url}
              >
                播放语音 {msg.duration && msg.duration > 0 ? `(${msg.duration}秒)` : '(语音)'}
              </Button>
            </div>
          ) : (
            <div style={{
              whiteSpace: 'pre-wrap',
              wordBreak: 'break-word',
              lineHeight: '1.5'
            }}>
              {msg.content}
            </div>
          )
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
          {isRecording ? (
            <div style={{ textAlign: 'center' }}>
              <Button
                danger
                icon={<StopOutlined />}
                onClick={stopRecording}
                size="large"
                className="recording-indicator"
              >
                停止录音 ({recordingTime}秒)
              </Button>
              <div style={{ marginTop: 8, color: '#ff4d4f' }}>
                🎤 录音中... 点击停止按钮结束录音
              </div>
            </div>
          ) : (
            <Space.Compact style={{ width: '100%' }}>
              <Button
                type="default"
                icon={<AudioOutlined />}
                onClick={startRecording}
                style={{ height: 'auto' }}
              >
                录音
              </Button>
              <TextArea
                value={newMessage}
                onChange={(e) => setNewMessage(e.target.value)}
                onKeyDown={handleKeyDown}
                placeholder="输入消息... (Enter发送，Shift+Enter换行)"
                autoSize={{ minRows: 1, maxRows: 4 }}
                style={{ resize: 'none' }}
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
          )}
          
          <div style={{
            fontSize: '12px',
            color: '#999',
            marginTop: '8px',
            textAlign: 'center'
          }}>
            {!isRecording && `💡 5秒自动同步 • 清空时暂停同步 • ${messages.length}条消息`}
          </div>
        </div>
      </Card>
    </div>
  );
};

export default ChatPanel;