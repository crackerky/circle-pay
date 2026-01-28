import { useState, useEffect } from 'react';
import { useLiff } from '../liff/useLiff';
import {
  getMyCircles,
  createCircle,
  joinCircle,
  searchCircles,
  leaveCircle,
  setPrimaryCircle,
  Circle
} from '../liff/api';

export default function CirclesPage() {
  const { isLoggedIn, accessToken } = useLiff();
  const [circles, setCircles] = useState<Circle[]>([]);
  const [primaryCircleId, setPrimaryCircleIdState] = useState<number | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // モーダル状態
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [showJoinModal, setShowJoinModal] = useState(false);
  const [newCircleName, setNewCircleName] = useState('');
  const [searchQuery, setSearchQuery] = useState('');
  const [searchResults, setSearchResults] = useState<Circle[]>([]);
  const [searching, setSearching] = useState(false);

  useEffect(() => {
    if (isLoggedIn && accessToken) {
      loadCircles();
    }
  }, [isLoggedIn, accessToken]);

  const loadCircles = async () => {
    if (!accessToken) return;
    setLoading(true);
    setError(null);
    try {
      const response = await getMyCircles(accessToken);
      setCircles(response.circles || []);
      setPrimaryCircleIdState(response.primaryCircleId);
    } catch (err) {
      console.error('サークル取得エラー:', err);
      setError('サークル一覧の取得に失敗しました');
    } finally {
      setLoading(false);
    }
  };

  const handleCreateCircle = async () => {
    if (!accessToken || !newCircleName.trim()) return;
    try {
      await createCircle(accessToken, newCircleName.trim());
      setShowCreateModal(false);
      setNewCircleName('');
      loadCircles();
    } catch (err: any) {
      alert(err.message || 'サークルの作成に失敗しました');
    }
  };

  const handleSearch = async () => {
    if (!accessToken || !searchQuery.trim()) return;
    setSearching(true);
    try {
      const response = await searchCircles(accessToken, searchQuery.trim());
      setSearchResults(response.circles || []);
    } catch (err) {
      console.error('検索エラー:', err);
    } finally {
      setSearching(false);
    }
  };

  const handleJoinCircle = async (circleId: number) => {
    if (!accessToken) return;
    try {
      await joinCircle(accessToken, undefined, circleId);
      setShowJoinModal(false);
      setSearchQuery('');
      setSearchResults([]);
      loadCircles();
    } catch (err: any) {
      alert(err.message || 'サークルへの参加に失敗しました');
    }
  };

  const handleLeaveCircle = async (circleId: number, circleName: string) => {
    if (!accessToken) return;
    if (!confirm(`「${circleName}」から退出しますか？`)) return;
    try {
      await leaveCircle(accessToken, circleId);
      loadCircles();
    } catch (err: any) {
      alert(err.message || 'サークルからの退出に失敗しました');
    }
  };

  const handleSetPrimary = async (circleId: number) => {
    if (!accessToken) return;
    try {
      await setPrimaryCircle(accessToken, circleId);
      setPrimaryCircleIdState(circleId);
    } catch (err: any) {
      alert(err.message || '主サークルの設定に失敗しました');
    }
  };

  if (!isLoggedIn) {
    return <div style={styles.container}><p>ログインしてください</p></div>;
  }

  if (loading) {
    return <div style={styles.container}><p>読み込み中...</p></div>;
  }

  return (
    <div style={styles.container}>
      <h1 style={styles.title}>サークル管理</h1>

      {error && <p style={styles.error}>{error}</p>}

      <div style={styles.buttonGroup}>
        <button style={styles.primaryButton} onClick={() => setShowCreateModal(true)}>
          + 新規作成
        </button>
        <button style={styles.secondaryButton} onClick={() => setShowJoinModal(true)}>
          🔍 参加する
        </button>
      </div>

      <h2 style={styles.subtitle}>所属サークル ({circles.length})</h2>

      {circles.length === 0 ? (
        <p style={styles.noData}>サークルに所属していません</p>
      ) : (
        <ul style={styles.list}>
          {circles.map((circle) => (
            <li key={circle.id} style={styles.listItem}>
              <div style={styles.circleInfo}>
                <span style={styles.circleName}>
                  {circle.name}
                  {primaryCircleId === circle.id && <span style={styles.primaryBadge}>メイン</span>}
                </span>
              </div>
              <div style={styles.circleActions}>
                {primaryCircleId !== circle.id && (
                  <button
                    style={styles.smallButton}
                    onClick={() => handleSetPrimary(circle.id)}
                  >
                    メインに設定
                  </button>
                )}
                <button
                  style={styles.dangerButton}
                  onClick={() => handleLeaveCircle(circle.id, circle.name)}
                >
                  退出
                </button>
              </div>
            </li>
          ))}
        </ul>
      )}

      {/* 新規作成モーダル */}
      {showCreateModal && (
        <div style={styles.modalOverlay} onClick={() => setShowCreateModal(false)}>
          <div style={styles.modal} onClick={(e) => e.stopPropagation()}>
            <h3 style={styles.modalTitle}>新規サークル作成</h3>
            <input
              type="text"
              placeholder="サークル名"
              value={newCircleName}
              onChange={(e) => setNewCircleName(e.target.value)}
              style={styles.input}
            />
            <div style={styles.modalButtons}>
              <button style={styles.secondaryButton} onClick={() => setShowCreateModal(false)}>
                キャンセル
              </button>
              <button
                style={styles.primaryButton}
                onClick={handleCreateCircle}
                disabled={!newCircleName.trim()}
              >
                作成
              </button>
            </div>
          </div>
        </div>
      )}

      {/* 参加モーダル */}
      {showJoinModal && (
        <div style={styles.modalOverlay} onClick={() => setShowJoinModal(false)}>
          <div style={styles.modal} onClick={(e) => e.stopPropagation()}>
            <h3 style={styles.modalTitle}>サークルに参加</h3>
            <div style={styles.searchBox}>
              <input
                type="text"
                placeholder="サークル名で検索"
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                style={styles.input}
                onKeyDown={(e) => e.key === 'Enter' && handleSearch()}
              />
              <button style={styles.primaryButton} onClick={handleSearch} disabled={searching}>
                {searching ? '...' : '検索'}
              </button>
            </div>

            {searchResults.length > 0 && (
              <ul style={styles.searchResults}>
                {searchResults.map((circle) => (
                  <li key={circle.id} style={styles.searchResultItem}>
                    <span>{circle.name}</span>
                    <button
                      style={styles.smallButton}
                      onClick={() => handleJoinCircle(circle.id)}
                    >
                      参加
                    </button>
                  </li>
                ))}
              </ul>
            )}

            <button style={styles.secondaryButton} onClick={() => setShowJoinModal(false)}>
              閉じる
            </button>
          </div>
        </div>
      )}
    </div>
  );
}

const styles: { [key: string]: React.CSSProperties } = {
  container: {
    padding: '20px',
    maxWidth: '600px',
    margin: '0 auto',
  },
  title: {
    fontSize: '24px',
    fontWeight: 'bold',
    marginBottom: '20px',
    textAlign: 'center',
  },
  subtitle: {
    fontSize: '18px',
    fontWeight: 'bold',
    marginTop: '24px',
    marginBottom: '12px',
  },
  error: {
    color: '#dc3545',
    marginBottom: '16px',
  },
  buttonGroup: {
    display: 'flex',
    gap: '12px',
    justifyContent: 'center',
  },
  primaryButton: {
    backgroundColor: '#06C755',
    color: 'white',
    border: 'none',
    padding: '12px 24px',
    borderRadius: '8px',
    fontSize: '16px',
    cursor: 'pointer',
  },
  secondaryButton: {
    backgroundColor: '#6c757d',
    color: 'white',
    border: 'none',
    padding: '12px 24px',
    borderRadius: '8px',
    fontSize: '16px',
    cursor: 'pointer',
  },
  smallButton: {
    backgroundColor: '#06C755',
    color: 'white',
    border: 'none',
    padding: '6px 12px',
    borderRadius: '4px',
    fontSize: '12px',
    cursor: 'pointer',
  },
  dangerButton: {
    backgroundColor: '#dc3545',
    color: 'white',
    border: 'none',
    padding: '6px 12px',
    borderRadius: '4px',
    fontSize: '12px',
    cursor: 'pointer',
  },
  noData: {
    color: '#6c757d',
    textAlign: 'center',
    marginTop: '20px',
  },
  list: {
    listStyle: 'none',
    padding: 0,
    margin: 0,
  },
  listItem: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    padding: '12px 16px',
    backgroundColor: '#f8f9fa',
    borderRadius: '8px',
    marginBottom: '8px',
  },
  circleInfo: {
    flex: 1,
  },
  circleName: {
    fontWeight: 'bold',
    display: 'flex',
    alignItems: 'center',
    gap: '8px',
  },
  primaryBadge: {
    backgroundColor: '#ffc107',
    color: '#000',
    padding: '2px 8px',
    borderRadius: '4px',
    fontSize: '10px',
    fontWeight: 'bold',
  },
  circleActions: {
    display: 'flex',
    gap: '8px',
  },
  modalOverlay: {
    position: 'fixed',
    top: 0,
    left: 0,
    right: 0,
    bottom: 0,
    backgroundColor: 'rgba(0, 0, 0, 0.5)',
    display: 'flex',
    justifyContent: 'center',
    alignItems: 'center',
    zIndex: 1000,
  },
  modal: {
    backgroundColor: 'white',
    padding: '24px',
    borderRadius: '12px',
    width: '90%',
    maxWidth: '400px',
  },
  modalTitle: {
    fontSize: '18px',
    fontWeight: 'bold',
    marginBottom: '16px',
    textAlign: 'center',
  },
  modalButtons: {
    display: 'flex',
    gap: '12px',
    justifyContent: 'center',
    marginTop: '16px',
  },
  input: {
    width: '100%',
    padding: '12px',
    fontSize: '16px',
    border: '1px solid #ddd',
    borderRadius: '8px',
    boxSizing: 'border-box',
    marginBottom: '12px',
  },
  searchBox: {
    display: 'flex',
    gap: '8px',
    marginBottom: '16px',
  },
  searchResults: {
    listStyle: 'none',
    padding: 0,
    margin: '0 0 16px 0',
    maxHeight: '200px',
    overflowY: 'auto',
  },
  searchResultItem: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    padding: '8px 12px',
    backgroundColor: '#f8f9fa',
    borderRadius: '4px',
    marginBottom: '4px',
  },
};
