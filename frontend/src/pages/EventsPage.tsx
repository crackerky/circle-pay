import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { useLiff } from '../liff/useLiff';
import { getMyEvents, type Event } from '../liff/api';

export default function EventsPage() {
  const navigate = useNavigate();
  const { isLoggedIn, isLoading, accessToken, displayName } = useLiff();
  const [events, setEvents] = useState<Event[]>([]);
  const [isLoadingData, setIsLoadingData] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    if (isLoggedIn && accessToken) {
      loadEvents();
    }
  }, [isLoggedIn, accessToken]);

  const loadEvents = async () => {
    if (!accessToken) return;

    setIsLoadingData(true);
    try {
      const response = await getMyEvents(accessToken);
      setEvents(response.events || []);
    } catch (error) {
      console.error('イベント取得エラー:', error);
      setError('イベント一覧の取得に失敗しました');
    } finally {
      setIsLoadingData(false);
    }
  };

  if (isLoading || isLoadingData) {
    return (
      <div style={styles.container}>
        <div style={styles.loading}>読み込み中...</div>
      </div>
    );
  }

  if (!isLoggedIn) {
    return (
      <div style={styles.container}>
        <div style={styles.error}>ログインが必要です</div>
      </div>
    );
  }

  return (
    <div style={styles.container}>
      <h1 style={styles.title}>📊 イベント管理</h1>

      <div style={styles.userInfo}>
        <p>ユーザー: {displayName}</p>
      </div>

      <div style={styles.actions}>
        <button onClick={() => navigate('/create')} style={styles.createButton}>
          ➕ 新しいイベントを作成
        </button>
        <button onClick={() => navigate('/approve')} style={styles.approveButton}>
          ✅ 支払いを承認
        </button>
      </div>

      {error && <div style={styles.errorMessage}>{error}</div>}

      {events.length === 0 ? (
        <div style={styles.emptyState}>
          <p style={styles.emptyText}>作成したイベントはありません</p>
          <button onClick={() => navigate('/create')} style={styles.emptyButton}>
            最初のイベントを作成
          </button>
        </div>
      ) : (
        <div style={styles.eventList}>
          {events.map((event) => (
            <div key={event.id} style={styles.eventCard}>
              <div style={styles.eventHeader}>
                <span style={styles.eventName}>{event.name}</span>
                <span
                  style={{
                    ...styles.statusBadge,
                    ...(event.status === 'confirmed'
                      ? styles.statusConfirmed
                      : event.status === 'completed'
                      ? styles.statusCompleted
                      : styles.statusSelecting),
                  }}
                >
                  {event.status === 'confirmed'
                    ? '進行中'
                    : event.status === 'completed'
                    ? '完了'
                    : '選択中'}
                </span>
              </div>

              <div style={styles.eventDetails}>
                <div style={styles.detailRow}>
                  <span style={styles.detailLabel}>合計金額:</span>
                  <span style={styles.detailValue}>
                    {event.totalAmount.toLocaleString()}円
                  </span>
                </div>
                <div style={styles.detailRow}>
                  <span style={styles.detailLabel}>1人あたり:</span>
                  <span style={styles.detailValue}>
                    {event.splitAmount.toLocaleString()}円
                  </span>
                </div>
                <div style={styles.detailRow}>
                  <span style={styles.detailLabel}>作成日時:</span>
                  <span style={styles.detailValue}>
                    {new Date(event.createdAt).toLocaleString('ja-JP')}
                  </span>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

const styles: { [key: string]: React.CSSProperties } = {
  container: {
    maxWidth: '600px',
    margin: '0 auto',
    padding: '20px',
    fontFamily: 'sans-serif',
  },
  loading: {
    textAlign: 'center',
    padding: '40px',
    fontSize: '16px',
    color: '#666',
  },
  error: {
    textAlign: 'center',
    padding: '40px',
    fontSize: '16px',
    color: '#e74c3c',
  },
  title: {
    fontSize: '24px',
    fontWeight: 'bold',
    marginBottom: '20px',
    textAlign: 'center',
  },
  userInfo: {
    backgroundColor: '#f8f9fa',
    padding: '12px',
    borderRadius: '8px',
    marginBottom: '20px',
    fontSize: '14px',
    color: '#666',
  },
  actions: {
    display: 'flex',
    gap: '12px',
    marginBottom: '24px',
  },
  createButton: {
    flex: 1,
    padding: '14px',
    fontSize: '16px',
    fontWeight: 'bold',
    color: '#fff',
    backgroundColor: '#06c755',
    border: 'none',
    borderRadius: '8px',
    cursor: 'pointer',
  },
  approveButton: {
    flex: 1,
    padding: '14px',
    fontSize: '16px',
    fontWeight: 'bold',
    color: '#fff',
    backgroundColor: '#00b0ff',
    border: 'none',
    borderRadius: '8px',
    cursor: 'pointer',
  },
  errorMessage: {
    backgroundColor: '#ffebee',
    color: '#c62828',
    padding: '12px',
    borderRadius: '8px',
    fontSize: '14px',
    marginBottom: '16px',
  },
  emptyState: {
    textAlign: 'center',
    padding: '60px 20px',
  },
  emptyText: {
    fontSize: '16px',
    color: '#999',
    marginBottom: '24px',
  },
  emptyButton: {
    padding: '14px 28px',
    fontSize: '16px',
    fontWeight: 'bold',
    color: '#fff',
    backgroundColor: '#06c755',
    border: 'none',
    borderRadius: '8px',
    cursor: 'pointer',
  },
  eventList: {
    display: 'flex',
    flexDirection: 'column',
    gap: '16px',
  },
  eventCard: {
    padding: '16px',
    border: '1px solid #ddd',
    borderRadius: '8px',
    backgroundColor: '#fff',
  },
  eventHeader: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: '12px',
  },
  eventName: {
    fontSize: '18px',
    fontWeight: 'bold',
    color: '#333',
  },
  statusBadge: {
    padding: '4px 12px',
    fontSize: '12px',
    fontWeight: 'bold',
    borderRadius: '12px',
  },
  statusConfirmed: {
    backgroundColor: '#e3f2fd',
    color: '#1976d2',
  },
  statusCompleted: {
    backgroundColor: '#e8f5e9',
    color: '#388e3c',
  },
  statusSelecting: {
    backgroundColor: '#fff3e0',
    color: '#f57c00',
  },
  eventDetails: {
    display: 'flex',
    flexDirection: 'column',
    gap: '8px',
  },
  detailRow: {
    display: 'flex',
    justifyContent: 'space-between',
    fontSize: '14px',
  },
  detailLabel: {
    color: '#666',
  },
  detailValue: {
    fontWeight: '500',
    color: '#333',
  },
};
