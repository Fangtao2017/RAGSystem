import React, { useState } from 'react';
import axios from 'axios';
import ReactMarkdown from 'react-markdown';
import './QueryComponent.css';

const QueryComponent = () => {
  const [query, setQuery] = useState('');
  const [answer, setAnswer] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);

  const handleSubmit = async (e) => {
    e.preventDefault();
    if (!query.trim()) return;

    setLoading(true);
    setError(null);
    try {
      const response = await axios.post('http://localhost:8080/query', { query });
      setAnswer(response.data.answer || '没有找到相关信息');
    } catch (err) {
      console.error('Error querying:', err);
      setError('查询失败: ' + (err.response?.data?.error || err.message));
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="query-component">
      <h2>提问问题</h2>
      <form onSubmit={handleSubmit}>
        <textarea
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder="输入您的问题..."
          rows={4}
          className="query-input"
        />
        <button 
          type="submit" 
          className="query-button"
          disabled={loading || !query.trim()}
        >
          {loading ? '查询中...' : '查询'}
        </button>
      </form>
      
      {error && <div className="error-message">{error}</div>}
      
      {answer && (
        <div className="answer-container">
          <h3>答案:</h3>
          <div className="answer-content">
            <ReactMarkdown>{answer}</ReactMarkdown>
          </div>
        </div>
      )}
    </div>
  );
};

export default QueryComponent;