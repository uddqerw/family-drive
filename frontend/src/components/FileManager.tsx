import React, { useState, useEffect } from 'react';
import {
  Button, Upload, message, Card,
  Row, Col, Tag, Progress
} from 'antd';
import {
  UploadOutlined, DownloadOutlined, DeleteOutlined,
  FileOutlined, FileImageOutlined, FilePdfOutlined,
  FileWordOutlined, FileExcelOutlined, FileZipOutlined
} from '@ant-design/icons';
import { fileAPI } from '../services/api';
import './FileManager.css';

// 文件类型图标映射
const fileIcons = {
  'pdf': <FilePdfOutlined style={{ color: '#ff4d4f' }} />,
  'jpg': <FileImageOutlined style={{ color: '#52c41a' }} />,
  'jpeg': <FileImageOutlined style={{ color: '#52c41a' }} />,
  'png': <FileImageOutlined style={{ color: '#52c41a' }} />,
  'doc': <FileWordOutlined style={{ color: '#1890ff' }} />,
  'docx': <FileWordOutlined style={{ color: '#1890ff' }} />,
  'xls': <FileExcelOutlined style={{ color: '#52c41a' }} />,
  'xlsx': <FileExcelOutlined style={{ color: '#52c41a' }} />,
  'zip': <FileZipOutlined style={{ color: '#faad14' }} />,
  'rar': <FileZipOutlined style={{ color: '#faad14' }} />,
  'default': <FileOutlined style={{ color: '#666' }} />
};

interface FileManagerProps {
  onLogout?: () => void;
}

const FileManager: React.FC<FileManagerProps> = () => {
  const [files, setFiles] = useState<any[]>([]);
  const [uploading, setUploading] = useState(false);
  const [downloading, setDownloading] = useState<string | null>(null);

  // 获取文件图标
  const getFileIcon = (filename: string) => {
    const ext = filename.split('.').pop()?.toLowerCase();
    return fileIcons[ext as keyof typeof fileIcons] || fileIcons.default;
  };

  // 格式化文件大小
  const formatFileSize = (bytes: number) => {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
  };

  // 加载文件列表
  const loadFiles = async () => {
    try {
      const response = await fileAPI.list();
      setFiles(response.data || []);
    } catch (error) {
      message.error('加载文件列表失败');
    }
  };

  // 文件上传
  const handleUpload = async (file: File) => {
    setUploading(true);
    const formData = new FormData();
    formData.append('file', file);

    try {
      await fileAPI.upload(formData);
      message.success(`文件 "${file.name}" 上传成功`);
      await loadFiles();
    } catch (error) {
      message.error('文件上传失败');
    } finally {
      setUploading(false);
    }
    return false;
  };

  // 文件下载 - 确保提示显示
  const handleDownload = async (filename: string) => {
    console.log('🚀 开始下载:', filename);
    setDownloading(filename);
    
    // 显示开始下载提示
    message.loading({
      content: `📥 准备下载: ${filename}`,
      key: 'download',
      duration: 0, // 持续显示
    });

    try {
      // 短暂延迟确保用户看到提示
      await new Promise(resolve => setTimeout(resolve, 500));
      
      const response = await fileAPI.download(filename);
      
      // 更新提示为下载中
      message.loading({
        content: `📥 下载中: ${filename}`,
        key: 'download',
        duration: 0,
      });

      const url = window.URL.createObjectURL(new Blob([response.data]));
      const link = document.createElement('a');
      link.href = url;
      link.setAttribute('download', filename);
      document.body.appendChild(link);
      link.click();
      link.remove();
      
      // 清理URL对象
      setTimeout(() => window.URL.revokeObjectURL(url), 100);
      
      // 下载完成 - 显示成功提示
      message.success({
        content: `✅ 下载完成: ${filename}`,
        key: 'download',
        duration: 3,
      });
      
      console.log('✅ 下载完成:', filename);
      
    } catch (error: any) {
      console.error('❌ 下载失败:', error);
      
      // 下载失败提示
      message.error({
        content: `❌ 下载失败: ${filename}`,
        key: 'download',
        duration: 3,
      });
    } finally {
      setDownloading(null);
    }
  };

  // 文件删除
  const handleDelete = async (filename: string) => {
    if (!window.confirm(`确定要删除文件 "${filename}" 吗？此操作不可撤销。`)) {
      return;
    }

    try {
      await fileAPI.delete(filename);
      message.success(`✅ 文件 "${filename}" 删除成功`);
      await loadFiles();
    } catch (error: any) {
      console.error('删除失败:', error);
      message.error('文件删除失败');
    }
  };

  useEffect(() => {
    loadFiles();
  }, []);

  return (
    <div className="enhanced-file-manager">
      <Card
        title={
          <div className="card-header">
            <span>🏠 家庭网盘</span>
            <Tag color="blue">{files.length} 个文件</Tag>
          </div>
        }
        className="file-manager-card"
      >
        {/* 上传区域 */}
        <div className="upload-section">
          <Upload.Dragger
            multiple
            showUploadList={false}
            beforeUpload={handleUpload}
            className="upload-dragger"
          >
            <div className="upload-content">
              <UploadOutlined className="upload-icon" />
              <div className="upload-text">
                <div>点击或拖拽文件到此处上传</div>
                <div className="upload-hint">支持单个或批量上传</div>
              </div>
            </div>
          </Upload.Dragger>
          {uploading && (
            <div className="upload-progress">
              <Progress percent={50} status="active" showInfo={false} />
              <div>上传中...</div>
            </div>
          )}
        </div>

        {/* 文件列表 */}
        <div className="file-list-section">
          {files.length === 0 ? (
            <div className="empty-state">
              <FileOutlined className="empty-icon" />
              <div className="empty-text">暂无文件</div>
              <div className="empty-hint">上传第一个文件开始使用家庭网盘</div>
            </div>
          ) : (
            <Row gutter={[16, 16]} className="file-grid">
              {files.map((file, index) => (
                <Col xs={24} sm={12} md={8} lg={6} key={index}>
                  <div className="file-card">
                    <div className="file-header">
                      {getFileIcon(file.name)}
                      <span className="file-name" title={file.name}>
                        {file.name}
                      </span>
                    </div>
                    <div className="file-info">
                      <div className="file-size">
                        {formatFileSize(file.size)}
                      </div>
                      <div className="file-actions">
                        <Button
                          type="link"
                          icon={<DownloadOutlined />}
                          onClick={() => handleDownload(file.name)}
                          title="下载"
                          loading={downloading === file.name}
                          disabled={!!downloading}
                        />
                        <Button
                          type="link"
                          danger
                          icon={<DeleteOutlined />}
                          onClick={() => handleDelete(file.name)}
                          title="删除"
                          disabled={!!downloading}
                        />
                      </div>
                    </div>
                  </div>
                </Col>
              ))}
            </Row>
          )}
        </div>
      </Card>
    </div>
  );
};

export default FileManager;