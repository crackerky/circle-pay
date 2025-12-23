import { useState, useEffect } from 'react';
import './App.css';
import type { User, ReceivedMessage } from './types';

function App() {
  const [users, setUsers] = useState<User[]>([]);
  const [messages, setMessages] = useState<ReceivedMessage[]>([]);
  const [selectedUserId, setSelectedUserId] = useState<string>('');
  const [messageText, setMessageText] = useState<string>('');

  const loadUsers = async () => {
    try {
      const response = await fetch('/api/users');
      const data = await response.json();
      setUsers(data || []);
    } catch (error) {
      console.error('ユーザー取得エラー:', error);
    }
  };

  const loadMessages = async () => {
    try {
      const response = await fetch('/messages');
      const data = await response.json();
      setMessages(data || []);
    } catch (error) {
      console.error('メッセージ取得エラー:', error);
    }
  };

  const sendMessage = async () => {
    if (!selectedUserId || !messageText) {
      alert('ユーザーとメッセージを入力してください');
      return;
    }

    try {
      const response = await fetch('/send', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          userId: selectedUserId,
          message: messageText,
        }),
      });

      if (response.ok) {
        alert('送信しました！');
        setMessageText('');
      } else {
        alert('送信に失敗しました');
      }
    } catch (error) {
      console.error('送信エラー:', error);
      alert('送信に失敗しました');
    }
  };

  useEffect(() => {
    loadUsers();
    loadMessages();

    const messageInterval = setInterval(loadMessages, 5000);
    const userInterval = setInterval(loadUsers, 10000);

    return () => {
      clearInterval(messageInterval);
      clearInterval(userInterval);
    };
  }, []);

  return (
    <div className="container">
      <h1>LINE Bot Manager</h1>

      <div className="section">
        <h2>メッセージ送信</h2>
        <select
          value={selectedUserId}
          onChange={(e) => setSelectedUserId(e.target.value)}
        >
          <option value="">ユーザーを選択してください</option>
          {users.map((user) => (
            <option key={user.UserID} value={user.UserID}>
              {user.Name || 'Unknown'} ({user.Circle || '未登録'})
            </option>
          ))}
        </select>
        <textarea
          placeholder="メッセージ"
          rows={3}
          value={messageText}
          onChange={(e) => setMessageText(e.target.value)}
        />
        <button onClick={sendMessage}>送信</button>
      </div>

      <div className="section">
        <h2>受信メッセージ</h2>
        <button onClick={loadMessages}>更新</button>
        <div className="messages">
          {[...messages].reverse().map((msg, index) => (
            <div key={index} className="message">
              <div className="timestamp">
                {new Date(msg.timestamp).toLocaleString()}
              </div>
              <div>
                <strong>User:</strong> {msg.userID}
              </div>
              <div>{msg.text}</div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}

export default App;
