import { useState } from 'react';
import { Layout, Tabs } from 'antd';
import { FileTextOutlined, MessageOutlined } from '@ant-design/icons';
import FileManager from './components/FileManager';
import ChatPanel from './components/ChatPanel';
import Login from './components/Login';

const { Header, Content } = Layout;
const { TabPane } = Tabs;

function App() {
  const [activeTab, setActiveTab] = useState('files');

  // 直接渲染内容，绕过Login组件测试
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
        {/* 取消注释Login组件测试 */}
        <Login>
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
         </Login>
      </Content>
    </Layout>
  );
}

export default App;