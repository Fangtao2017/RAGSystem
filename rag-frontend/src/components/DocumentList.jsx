import React, { useState, useEffect } from "react";
import { List, Button, message, Spin } from "antd";
import axios from 'axios';
import './DocumentList.css';

const DocumentList = ({ refreshTrigger, onRefresh }) => {
  const [documents, setDocuments] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [clearingVectors, setClearingVectors] = useState(false);
  const [clearingDocuments, setClearingDocuments] = useState(false);
  const [cleaningInvalidDocs, setCleaningInvalidDocs] = useState(false);
  const [processingDocs, setProcessingDocs] = useState({});

  useEffect(() => {
    fetchDocuments();
  }, [refreshTrigger]);

  const fetchDocuments = async () => {
    setLoading(true);
    try {
      const response = await axios.get('http://localhost:8080/documents');
      console.log('Documents API response:', response.data);
      
      // 确保documents是一个数组
      const docs = Array.isArray(response.data) ? response.data : 
                  (response.data.documents ? response.data.documents : []);
      
      console.log('Processed documents:', docs);
      setDocuments(docs);
      setError(null);
    } catch (err) {
      console.error('Error fetching documents:', err);
      setError('Failed to load documents');
    } finally {
      setLoading(false);
    }
  };

  const handleDelete = async (docId) => {
    try {
      await axios.delete(`http://localhost:8080/document/${docId}`);
      fetchDocuments();
      if (onRefresh) onRefresh();
      message.success("文档删除成功");
    } catch (err) {
      console.error('Error deleting document:', err);
      message.error(err.response?.data?.error || "删除文档失败");
    }
  };

  const handleReprocess = async (docId) => {
    setProcessingDocs(prev => ({ ...prev, [docId]: true }));
    try {
      await axios.post(`http://localhost:8080/document/${docId}/reprocess`);
      message.success("文档重新处理已开始");
      // 刷新文档列表以显示更新的状态
      fetchDocuments();
    } catch (err) {
      console.error('Error reprocessing document:', err);
      message.error(err.response?.data?.error || "重新处理文档失败");
    } finally {
      setProcessingDocs(prev => ({ ...prev, [docId]: false }));
    }
  };

  const handleClearVectors = async () => {
    if (window.confirm('确定要清空向量数据库吗？这将删除所有向量数据，但保留文档记录。')) {
      setClearingVectors(true);
      try {
        await axios.post('http://localhost:8080/clear-vectors');
        alert('向量数据库已清空。您可以点击文档旁边的"重新处理"按钮来重新生成向量数据。');
        fetchDocuments();
      } catch (err) {
        console.error('Error clearing vectors:', err);
        alert('清空向量数据库失败: ' + err.message);
      } finally {
        setClearingVectors(false);
      }
    }
  };

  const handleClearAllDocuments = async () => {
    if (window.confirm('确定要清空所有文档吗？这将删除所有文档记录和文件，此操作不可恢复！')) {
      setClearingDocuments(true);
      try {
        await axios.post('http://localhost:8080/clear-all-documents');
        message.success("所有文档已清空");
        fetchDocuments();
        if (onRefresh) onRefresh();
      } catch (err) {
        console.error('Error clearing all documents:', err);
        message.error(err.response?.data?.error || "清空文档失败");
      } finally {
        setClearingDocuments(false);
      }
    }
  };

  const handleCleanupInvalidDocuments = async () => {
    setCleaningInvalidDocs(true);
    try {
      const response = await axios.post('http://localhost:8080/cleanup-invalid-documents');
      message.success(`已清理 ${response.data.count} 个无效文档记录`);
      fetchDocuments();
    } catch (err) {
      console.error('Error cleaning up invalid documents:', err);
      message.error(err.response?.data?.error || "清理无效文档记录失败");
    } finally {
      setCleaningInvalidDocs(false);
    }
  };

  // 格式化日期显示
  const formatDate = (dateString) => {
    if (!dateString) return 'Unknown';
    
    try {
      const date = new Date(dateString);
      // 检查日期是否有效
      if (isNaN(date.getTime())) {
        return 'Invalid Date';
      }
      return date.toLocaleString();
    } catch (err) {
      console.error('Error formatting date:', err);
      return 'Invalid Date';
    }
  };

  if (loading) return <div>加载中...</div>;
  if (error) return <div className="error">{error}</div>;

  return (
    <div className="document-list">
      <h2>已上传文档</h2>
      {documents.length === 0 ? (
        <p>暂无文档</p>
      ) : (
        <div>
          <ul>
            {documents.map((doc) => (
              <li key={doc._id}>
                <div className="doc-info">
                  <span className="doc-name">{doc.name}</span>
                  <span className={`doc-status ${doc.status}`}>
                    状态: {doc.status}
                  </span>
                  <span className="doc-date">
                    上传时间: {formatDate(doc.uploadedAt)}
                  </span>
                </div>
                <div className="doc-actions">
                  <button
                    className="reprocess-btn"
                    onClick={() => handleReprocess(doc._id)}
                    disabled={processingDocs[doc._id] || doc.status === 'processing'}
                  >
                    {processingDocs[doc._id] ? '处理中...' : '重新处理'}
                  </button>
                  <button
                    className="delete-btn"
                    onClick={() => handleDelete(doc._id)}
                  >
                    删除
                  </button>
                </div>
              </li>
            ))}
          </ul>
          <div className="admin-actions">
            <div className="admin-buttons">
              <button 
                className="clear-vectors-btn" 
                onClick={handleClearVectors}
                disabled={clearingVectors}
              >
                {clearingVectors ? '清空中...' : '清空向量数据库'}
              </button>
              <button 
                className="clear-all-btn" 
                onClick={handleClearAllDocuments}
                disabled={clearingDocuments}
              >
                {clearingDocuments ? '清空中...' : '清空所有文档'}
              </button>
              <button 
                className="cleanup-btn" 
                onClick={handleCleanupInvalidDocuments}
                disabled={cleaningInvalidDocs}
              >
                {cleaningInvalidDocs ? '清理中...' : '清理无效记录'}
              </button>
            </div>
            <p className="help-text">
              <strong>清空向量数据库</strong>: 删除所有向量数据，但保留文档记录。清空后，您可以点击文档旁边的"重新处理"按钮来重新生成向量数据。
            </p>
            <p className="help-text warning">
              <strong>清空所有文档</strong>: 删除所有文档记录和文件，此操作不可恢复！
            </p>
            <p className="help-text">
              <strong>清理无效记录</strong>: 删除数据库中的无效文档记录，修复界面显示问题。
            </p>
          </div>
        </div>
      )}
    </div>
  );
};

export default DocumentList;