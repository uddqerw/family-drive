import React, { useState, useEffect } from 'react';
import { Button, Upload, List, message, Space, Card } from 'antd';
import { UploadOutlined, DownloadOutlined, DeleteOutlined } from '@ant-design/icons';
import { fileAPI } from '../services/api';

// 文件信息类型定义
interface FileInfo {
  name: string;
  size: number;
  created_at?: string;
  owner_id?: number;
}

const FileManager: React.FC = () => {
  const [files, setFiles] = useState<FileInfo[]>([]);
  const [loading, setLoading] = useState(false);

  // 加载文件列表
  const loadFiles = async () => {
    console.log('=== 开始加载文件列表 ===');
    setLoading(true);
    try {
      const response = await fileAPI.list();
      console.log('文件列表API响应:', response);
      
      // 确保正确设置文件列表
      if (response.data && Array.isArray(response.data)) {
        setFiles(response.data);
        console.log('文件列表已更新:', response.data);
        message.success(`已加载 ${response.data.length} 个文件`);
      } else {
        console.log('文件列表为空或格式错误');
        setFiles([]);
      }
    } catch (error: any) {
      console.error('加载文件列表失败:', error);
      message.error('加载文件列表失败: ' + (error.message || '未知错误'));
    } finally {
      setLoading(false);
    }
  };

  // 初始化加载文件列表
  useEffect(() => {
    loadFiles();
  }, []);

  // 处理文件上传
  const handleUpload = async (file: File) => {
    console.log('=== 开始上传文件 ===');
    console.log('文件信息:', file.name, file.size, file.type);
    
    const formData = new FormData();
    formData.append('file', file);
    
    try {
      console.log('调用上传API...');
      const response = await fileAPI.upload(formData);
      console.log('上传API响应:', response);
      
      message.success(`文件上传成功: ${file.name}`);
      
      // 等待列表刷新完成
      console.log('开始刷新文件列表...');
      await loadFiles();
      console.log('文件列表刷新完成');
      
    } catch (error: any) {
      console.error('上传文件失败:', error);
      if (error.response) {
        console.error('错误响应:', error.response.data);
        message.error('文件上传失败: ' + (error.response.data.error || error.response.status));
      } else {
        message.error('文件上传失败: ' + (error.message || '网络错误'));
      }
    }
    return false; // 阻止默认上传行为
  };

  // 处理文件下载
  const handleDownload = async (filename: string) => {
    console.log('开始下载文件:', filename);
    try {
      const response = await fileAPI.download(filename);
      console.log('下载响应:', response);
      
      // 创建下载链接
      const url = window.URL.createObjectURL(new Blob([response.data]));
      const link = document.createElement('a');
      link.href = url;
      link.setAttribute('download', filename);
      document.body.appendChild(link);
      link.click();
      link.remove();
      window.URL.revokeObjectURL(url);
      
      message.success(`开始下载: ${filename}`);
    } catch (error: any) {
      console.error('下载文件失败:', error);
      message.error('文件下载失败: ' + (error.message || '未知错误'));
    }
  };

  // 处理文件删除
  const handleDelete = async (filename: string) => {
    console.log('开始删除文件:', filename);
    try {
      await fileAPI.delete(filename);
      console.log('删除成功');
      message.success(`文件删除成功: ${filename}`);
      
      // 刷新文件列表
      await loadFiles();
    } catch (error: any) {
      console.error('删除文件失败:', error);
      message.error('文件删除失败: ' + (error.message || '未知错误'));
    }
  };

  return (
    <div style={{ 
      position: 'fixed',
      top: 0,
      left: 0,
      right: 0,
      bottom: 0,
      padding: 20,
      background: '#f0f2f5',
      overflow: 'auto'
    }}>
      <Card title="家庭网盘 - 文件管理" style={{ minHeight: '100%' }}>
        <Space direction="vertical" style={{ width: '100%' }} size="large">
          {/* 文件上传区域 */}
          <div>
            <Upload 
              beforeUpload={handleUpload}
              showUploadList={false}
              accept="*/*"
            >
              <Button type="primary" icon={<UploadOutlined />} size="large">
                上传文件
              </Button>
            </Upload>
            <div style={{ marginTop: 8, color: '#666', fontSize: 12 }}>
              支持所有类型文件，点击或拖拽文件到此处上传
            </div>
          </div>

          {/* 文件列表 */}
          <List
            loading={loading}
            dataSource={files}
            locale={{ emptyText: '暂无文件，请上传文件' }}
            renderItem={(file: FileInfo) => (
              <List.Item
                actions={[
                  <Button 
                    key="download"
                    type="link" 
                    icon={<DownloadOutlined />}
                    onClick={() => handleDownload(file.name)}
                  >
                    下载
                  </Button>,
                  <Button 
                    key="delete"
                    type="link" 
                    danger 
                    icon={<DeleteOutlined />}
                    onClick={() => handleDelete(file.name)}
                  >
                    删除
                  </Button>,
                ]}
              >
                <List.Item.Meta
                  title={file.name}
                  description={
                    <div>
                      <div>大小: {file.size} bytes</div>
                      {file.created_at && <div>上传时间: {file.created_at}</div>}
                    </div>
                  }
                />
              </List.Item>
            )}
          />

          {/* 调试信息（开发时显示） */}
          {process.env.NODE_ENV === 'development' && (
            <div style={{ marginTop: 20, padding: 10, background: '#f5f5f5', borderRadius: 6 }}>
              <h4>调试信息:</h4>
              <div>文件数量: {files.length}</div>
              <div>加载状态: {loading ? '加载中...' : '已完成'}</div>
              <div style={{ fontSize: 12, marginTop: 5 }}>
                文件列表: {JSON.stringify(files, null, 2)}
              </div>
            </div>
          )}
        </Space>
      </Card>
    </div>
  );
};

export default FileManager;