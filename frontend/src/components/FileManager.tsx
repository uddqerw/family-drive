import React, { useState, useEffect } from 'react';
import { Button, Upload, List, message, Space, Card } from 'antd';
import { UploadOutlined, DownloadOutlined, DeleteOutlined } from '@ant-design/icons';
import { fileAPI } from '../services/api';

const FileManager: React.FC = () => {
  const [files, setFiles] = useState<any[]>([]);
  const [loading, setLoading] = useState(false);

  const loadFiles = async () => {
    try {
      const response = await fileAPI.list();
      setFiles(response.data || []);
    } catch (error) {
      message.error('加载文件列表失败');
    }
  };

  useEffect(() => {
    loadFiles();
  }, []);

  const handleUpload = async (file: File) => {
    const formData = new FormData();
    formData.append('file', file);
    
    try {
      await fileAPI.upload(formData);
      message.success('文件上传成功');
      loadFiles(); // 刷新列表
    } catch (error) {
      message.error('文件上传失败');
    }
    return false; // 阻止默认上传
  };

  const handleDownload = async (filename: string) => {
    try {
      const response = await fileAPI.download(filename);
      const url = window.URL.createObjectURL(new Blob([response.data]));
      const link = document.createElement('a');
      link.href = url;
      link.setAttribute('download', filename);
      document.body.appendChild(link);
      link.click();
      link.remove();
      message.success('开始下载文件');
    } catch (error) {
      message.error('文件下载失败');
    }
  };

  const handleDelete = async (filename: string) => {
    try {
      await fileAPI.delete(filename);
      message.success('文件删除成功');
      loadFiles(); // 刷新列表
    } catch (error) {
      message.error('文件删除失败');
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
      <Card title="文件管理" style={{ minHeight: '100%' }}>
        <Space direction="vertical" style={{ width: '100%' }} size="large">
          <Upload 
            beforeUpload={handleUpload}
            showUploadList={false}
          >
            <Button type="primary" icon={<UploadOutlined />}>
              上传文件
            </Button>
          </Upload>

          <List
            loading={loading}
            dataSource={files}
            renderItem={(file: any) => (
              <List.Item
                actions={[
                  <Button 
                    type="link" 
                    icon={<DownloadOutlined />}
                    onClick={() => handleDownload(file.name)}
                  >
                    下载
                  </Button>,
                  <Button 
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
                  description={`大小: ${file.size} bytes`}
                />
              </List.Item>
            )}
          />
        </Space>
      </Card>
    </div>
  );
};

export default FileManager;