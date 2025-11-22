import { useState, useEffect } from 'react';
import { Layout, Tabs } from 'antd';
import { FileTextOutlined, MessageOutlined } from '@ant-design/icons';
import FileManager from './components/FileManager';
import ChatPanel from './components/ChatPanel';
import Login from './components/Login';

const { Header, Content } = Layout;
const { TabPane } = Tabs;

function App() {
  const [activeTab, setActiveTab] = useState('files');
  const [isLoggedIn, setIsLoggedIn] = useState(false);

  // 检查登录状态
  useEffect(() => {
    const token = localStorage.getItem('access_token');
    const userInfo = localStorage.getItem('user_info');
    if (token && userInfo) {
      setIsLoggedIn(true);
    }
  }, []);

  // 登录成功回调
  const handleLoginSuccess = () => {
    setIsLoggedIn(true);
  };

  // 退出登录回调
  const handleLogout = () => {
    setIsLoggedIn(false);
  };

  // 如果未登录，显示登录界面
  if (!isLoggedIn) {
    return (
      <Login onLoginSuccess={handleLoginSuccess} onLogout={handleLogout}>
        {/* 登录时不会显示这个内容 */}
        <div>加载中...</div>
      </Login>
    );
  }

  // 已登录，显示主应用
  return (
    <Layout style={{ height: '100vh' }}>
      <Header style={{
        background: '#001529',
        color: 'white',
        fontSize: '18px',
        fontWeight: 'bold',
        height: '48px',
        lineHeight: '48px'
      }}>
        🏠 家庭网盘 & 聊天室
      </Header>

      <Content style={{
        height: 'calc(100vh - 48px)',
        padding: '0'
      }}>
        <Tabs
          activeKey={activeTab}
          onChange={setActiveTab}
          style={{
            height: '100%',
            display: 'flex',
            flexDirection: 'column'
          }}
          tabBarStyle={{
            margin: 0,
            padding: '0 16px',
            background: 'white'
          }}
        >
          <TabPane
            tab={
              <span>
                <FileTextOutlined />
                文件管理
              </span>
            }
            key="files"
          >
            <div style={{ height: '100%' }}>
              <FileManager />
            </div>
          </TabPane>

          <TabPane
            tab={
              <span>
                <MessageOutlined />
                家庭聊天
              </span>
            }
            key="chat"
          >
            <div style={{ height: '100%' }}>
              <ChatPanel />
            </div>
          </TabPane>
        </Tabs>
      </Content>
    </Layout>
  );
}

export default App;