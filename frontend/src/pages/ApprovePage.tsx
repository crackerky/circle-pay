import { useState, useEffect } from 'react';
import { useLiff } from '../liff/useLiff';
import { getPendingApprovals, approvePayments, type Approval } from '../liff/api';

export default function ApprovePage() {
  const { isLoggedIn, isLoading, accessToken, displayName, closeWindow } = useLiff();
  const [approvals, setApprovals] = useState<Approval[]>([]);
  const [selectedApprovals, setSelectedApprovals] = useState<Set<number>>(new Set());
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [error, setError] = useState('');
  const [isLoadingData, setIsLoadingData] = useState(true);

  useEffect(() => {
    if (isLoggedIn && accessToken) {
      loadApprovals();
    }
  }, [isLoggedIn, accessToken]);

  const loadApprovals = async () => {
    if (!accessToken) return;

    setIsLoadingData(true);
    try {
      const response = await getPendingApprovals(accessToken);
      setApprovals(response.approvals || []);
    } catch (error) {
      console.error('承認一覧取得エラー:', error);
      setError('承認一覧の取得に失敗しました');
    } finally {
      setIsLoadingData(false);
    }
  };

  const toggleApproval = (id: number) => {
    const newSelected = new Set(selectedApprovals);
    if (newSelected.has(id)) {
      newSelected.delete(id);
    } else {
      newSelected.add(id);
    }
    setSelectedApprovals(newSelected);
  };

  const selectAll = () => {
    if (selectedApprovals.size === approvals.length) {
      setSelectedApprovals(new Set());
    } else {
      setSelectedApprovals(new Set(approvals.map((a) => a.id)));
    }
  };

  const handleApprove = async () => {
    if (!accessToken) {
      setError('認証エラー');
      return;
    }

    if (selectedApprovals.size === 0) {
      setError('承認する支払いを選択してください');
      return;
    }

    if (!confirm(`${selectedApprovals.size}件の支払いを承認しますか？`)) {
      return;
    }

    setIsSubmitting(true);
    setError('');

    try {
      await approvePayments(accessToken, Array.from(selectedApprovals));
      alert('承認しました！\n参加者に通知を送信しました。');

      // リロードまたは画面を閉じる
      await loadApprovals();
      setSelectedApprovals(new Set());

      if (approvals.length - selectedApprovals.size === 0) {
        closeWindow();
      }
    } catch (error) {
      console.error('承認エラー:', error);
      setError(error instanceof Error ? error.message : '承認に失敗しました');
    } finally {
      setIsSubmitting(false);
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
      <h1 style={styles.title}>✅ 支払い承認</h1>

      <div style={styles.userInfo}>
        <p>ユーザー: {displayName}</p>
      </div>

      {approvals.length === 0 ? (
        <div style={styles.emptyState}>
          <p style={styles.emptyText}>承認待ちの支払いはありません</p>
          <button onClick={closeWindow} style={styles.closeButton}>
            閉じる
          </button>
        </div>
      ) : (
        <>
          <div style={styles.header}>
            <p style={styles.count}>
              承認待ち: {approvals.length}件 / 選択中: {selectedApprovals.size}件
            </p>
            <button onClick={selectAll} style={styles.selectAllButton}>
              {selectedApprovals.size === approvals.length ? '全て解除' : '全て選択'}
            </button>
          </div>

          <div style={styles.approvalList}>
            {approvals.map((approval) => (
              <label key={approval.id} style={styles.approvalItem}>
                <input
                  type="checkbox"
                  checked={selectedApprovals.has(approval.id)}
                  onChange={() => toggleApproval(approval.id)}
                  style={styles.checkbox}
                />
                <div style={styles.approvalContent}>
                  <div style={styles.approvalHeader}>
                    <span style={styles.participantName}>{approval.participantName}</span>
                    <span style={styles.amount}>{approval.amount.toLocaleString()}円</span>
                  </div>
                  <div style={styles.eventName}>{approval.eventName}</div>
                  <div style={styles.reportedAt}>
                    報告日時: {new Date(approval.reportedAt).toLocaleString('ja-JP')}
                  </div>
                </div>
              </label>
            ))}
          </div>

          {error && <div style={styles.errorMessage}>{error}</div>}

          <div style={styles.buttonGroup}>
            <button
              onClick={handleApprove}
              disabled={isSubmitting || selectedApprovals.size === 0}
              style={{
                ...styles.approveButton,
                ...(isSubmitting || selectedApprovals.size === 0 ? styles.buttonDisabled : {}),
              }}
            >
              {isSubmitting ? '承認中...' : `${selectedApprovals.size}件を承認`}
            </button>
          </div>
        </>
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
  emptyState: {
    textAlign: 'center',
    padding: '40px 20px',
  },
  emptyText: {
    fontSize: '16px',
    color: '#999',
    marginBottom: '20px',
  },
  closeButton: {
    padding: '12px 24px',
    fontSize: '16px',
    color: '#666',
    backgroundColor: '#f5f5f5',
    border: 'none',
    borderRadius: '8px',
    cursor: 'pointer',
  },
  header: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: '16px',
  },
  count: {
    fontSize: '14px',
    color: '#666',
  },
  selectAllButton: {
    padding: '8px 16px',
    fontSize: '14px',
    color: '#06c755',
    backgroundColor: '#fff',
    border: '1px solid #06c755',
    borderRadius: '6px',
    cursor: 'pointer',
  },
  approvalList: {
    display: 'flex',
    flexDirection: 'column',
    gap: '12px',
    marginBottom: '20px',
  },
  approvalItem: {
    display: 'flex',
    alignItems: 'flex-start',
    padding: '16px',
    border: '1px solid #ddd',
    borderRadius: '8px',
    cursor: 'pointer',
    transition: 'background-color 0.2s',
    backgroundColor: '#fff',
  },
  checkbox: {
    width: '20px',
    height: '20px',
    marginRight: '12px',
    marginTop: '4px',
    cursor: 'pointer',
    flexShrink: 0,
  },
  approvalContent: {
    flex: 1,
    display: 'flex',
    flexDirection: 'column',
    gap: '6px',
  },
  approvalHeader: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
  },
  participantName: {
    fontSize: '16px',
    fontWeight: 'bold',
    color: '#333',
  },
  amount: {
    fontSize: '18px',
    fontWeight: 'bold',
    color: '#06c755',
  },
  eventName: {
    fontSize: '14px',
    color: '#666',
  },
  reportedAt: {
    fontSize: '12px',
    color: '#999',
  },
  errorMessage: {
    backgroundColor: '#ffebee',
    color: '#c62828',
    padding: '12px',
    borderRadius: '8px',
    fontSize: '14px',
    marginBottom: '16px',
  },
  buttonGroup: {
    display: 'flex',
    gap: '12px',
  },
  approveButton: {
    flex: 1,
    padding: '16px',
    fontSize: '16px',
    fontWeight: 'bold',
    color: '#fff',
    backgroundColor: '#06c755',
    border: 'none',
    borderRadius: '8px',
    cursor: 'pointer',
    transition: 'background-color 0.2s',
  },
  buttonDisabled: {
    backgroundColor: '#ccc',
    cursor: 'not-allowed',
  },
};
