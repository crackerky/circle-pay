import { useState, useEffect } from 'react';
import { useLiff } from '../liff/useLiff';
import { createEvent, getCircleMembers } from '../liff/api';

export default function CreateEvent() {
  const { isLoggedIn, isLoading, error: liffError, accessToken, displayName, closeWindow } = useLiff();
  const [eventName, setEventName] = useState('');
  const [totalAmount, setTotalAmount] = useState('');
  const [members, setMembers] = useState<Array<{ userId: string; name: string }>>([]);
  const [selectedMembers, setSelectedMembers] = useState<Set<string>>(new Set());
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    if (isLoggedIn && accessToken) {
      loadMembers();
    }
  }, [isLoggedIn, accessToken]);

  const loadMembers = async () => {
    if (!accessToken) return;

    try {
      const response = await getCircleMembers(accessToken);
      setMembers(response.members);
    } catch (error) {
      console.error('メンバー取得エラー:', error);
      setError('メンバー一覧の取得に失敗しました');
    }
  };

  const toggleMember = (userId: string) => {
    const newSelected = new Set(selectedMembers);
    if (newSelected.has(userId)) {
      newSelected.delete(userId);
    } else {
      newSelected.add(userId);
    }
    setSelectedMembers(newSelected);
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!accessToken) {
      setError('認証エラー');
      return;
    }

    if (!eventName.trim()) {
      setError('イベント名を入力してください');
      return;
    }

    const amount = parseInt(totalAmount);
    if (isNaN(amount) || amount <= 0) {
      setError('正しい金額を入力してください');
      return;
    }

    if (selectedMembers.size === 0) {
      setError('参加者を選択してください');
      return;
    }

    setIsSubmitting(true);
    setError('');

    try {
      await createEvent(accessToken, {
        eventName: eventName.trim(),
        totalAmount: amount,
        participantIds: Array.from(selectedMembers),
      });

      alert('イベントを作成しました！\n参加者に通知を送信しました。');
      closeWindow();
    } catch (error) {
      console.error('イベント作成エラー:', error);
      setError(error instanceof Error ? error.message : 'イベントの作成に失敗しました');
    } finally {
      setIsSubmitting(false);
    }
  };

  if (isLoading) {
    return (
      <div style={styles.container}>
        <div style={styles.loading}>
          <p>読み込み中...</p>
          <p style={{ fontSize: '12px', color: '#999', marginTop: '10px' }}>
            コンソールログを確認してください
          </p>
        </div>
      </div>
    );
  }

  if (liffError) {
    return (
      <div style={styles.container}>
        <div style={styles.error}>
          <p>LIFF初期化エラー</p>
          <p style={{ fontSize: '14px', marginTop: '10px' }}>{liffError}</p>
          <button
            onClick={() => window.location.reload()}
            style={{ marginTop: '20px', padding: '10px 20px' }}
          >
            再読み込み
          </button>
        </div>
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

  const splitAmount = selectedMembers.size > 0
    ? Math.floor(parseInt(totalAmount) / selectedMembers.size)
    : 0;

  return (
    <div style={styles.container}>
      <h1 style={styles.title}>💰 割り勘イベント作成</h1>

      <div style={styles.userInfo}>
        <p>ユーザー: {displayName}</p>
      </div>

      <form onSubmit={handleSubmit} style={styles.form}>
        <div style={styles.formGroup}>
          <label style={styles.label}>イベント名</label>
          <input
            type="text"
            value={eventName}
            onChange={(e) => setEventName(e.target.value)}
            placeholder="例: 飲み会代"
            style={styles.input}
            required
          />
        </div>

        <div style={styles.formGroup}>
          <label style={styles.label}>合計金額（円）</label>
          <input
            type="number"
            value={totalAmount}
            onChange={(e) => setTotalAmount(e.target.value)}
            placeholder="10000"
            style={styles.input}
            required
            min="1"
          />
        </div>

        <div style={styles.formGroup}>
          <label style={styles.label}>
            参加者を選択 ({selectedMembers.size}人選択中)
          </label>

          {totalAmount && selectedMembers.size > 0 && (
            <div style={styles.splitInfo}>
              1人あたり: <strong>{splitAmount.toLocaleString()}円</strong>
            </div>
          )}

          <div style={styles.memberList}>
            {members.length === 0 ? (
              <p style={styles.noMembers}>同じサークルのメンバーがいません</p>
            ) : (
              members.map((member) => (
                <label key={member.userId} style={styles.memberItem}>
                  <input
                    type="checkbox"
                    checked={selectedMembers.has(member.userId)}
                    onChange={() => toggleMember(member.userId)}
                    style={styles.checkbox}
                  />
                  <span style={styles.memberName}>{member.name}</span>
                </label>
              ))
            )}
          </div>
        </div>

        {error && <div style={styles.errorMessage}>{error}</div>}

        <button
          type="submit"
          disabled={isSubmitting || members.length === 0}
          style={{
            ...styles.submitButton,
            ...(isSubmitting || members.length === 0 ? styles.submitButtonDisabled : {}),
          }}
        >
          {isSubmitting ? '作成中...' : 'イベントを作成して通知'}
        </button>
      </form>
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
  form: {
    display: 'flex',
    flexDirection: 'column',
    gap: '20px',
  },
  formGroup: {
    display: 'flex',
    flexDirection: 'column',
    gap: '8px',
  },
  label: {
    fontSize: '14px',
    fontWeight: 'bold',
    color: '#333',
  },
  input: {
    padding: '12px',
    fontSize: '16px',
    border: '1px solid #ddd',
    borderRadius: '8px',
    outline: 'none',
  },
  splitInfo: {
    backgroundColor: '#e8f5e9',
    padding: '12px',
    borderRadius: '8px',
    fontSize: '16px',
    color: '#2e7d32',
    textAlign: 'center',
  },
  memberList: {
    maxHeight: '300px',
    overflowY: 'auto',
    border: '1px solid #ddd',
    borderRadius: '8px',
    padding: '12px',
  },
  noMembers: {
    textAlign: 'center',
    color: '#999',
    padding: '20px',
  },
  memberItem: {
    display: 'flex',
    alignItems: 'center',
    padding: '12px',
    cursor: 'pointer',
    borderRadius: '6px',
    transition: 'background-color 0.2s',
  },
  checkbox: {
    width: '20px',
    height: '20px',
    marginRight: '12px',
    cursor: 'pointer',
  },
  memberName: {
    fontSize: '16px',
    color: '#333',
  },
  errorMessage: {
    backgroundColor: '#ffebee',
    color: '#c62828',
    padding: '12px',
    borderRadius: '8px',
    fontSize: '14px',
  },
  submitButton: {
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
  submitButtonDisabled: {
    backgroundColor: '#ccc',
    cursor: 'not-allowed',
  },
};
