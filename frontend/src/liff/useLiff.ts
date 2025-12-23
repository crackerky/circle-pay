import { useState, useEffect, useRef } from 'react';
import liff from '@line/liff';

export interface LiffState {
  isLoggedIn: boolean;
  isLoading: boolean;
  error: string | null;
  userId: string | null;
  displayName: string | null;
  accessToken: string | null;
  debugInfo: string | null;
}

export const useLiff = () => {
  const [state, setState] = useState<LiffState>({
    isLoggedIn: false,
    isLoading: true,
    error: null,
    userId: null,
    displayName: null,
    accessToken: null,
    debugInfo: null,
  });

  const initStarted = useRef(false);

  useEffect(() => {
    if (initStarted.current) return;
    initStarted.current = true;

    const initLiff = async () => {
      const debugLines: string[] = [];

      try {
        const liffId = import.meta.env.VITE_LIFF_ID || '2008577348-GDBXaBEr';
        debugLines.push(`LIFF ID: ${liffId}`);
        debugLines.push(`URL: ${window.location.href}`);

        // LIFF初期化
        await liff.init({
          liffId,
          withLoginOnExternalBrowser: false
        });

        debugLines.push('LIFF init完了');

        // liffオブジェクトが正しく初期化されているか確認
        if (!liff) {
          throw new Error('LIFF object is null');
        }

        const isInClient = liff.isInClient?.() ?? false;
        const isLoggedIn = liff.isLoggedIn?.() ?? false;
        const os = liff.getOS?.() ?? 'unknown';

        debugLines.push(`isInClient: ${isInClient}`);
        debugLines.push(`isLoggedIn: ${isLoggedIn}`);
        debugLines.push(`OS: ${os}`);

        if (isLoggedIn) {
          debugLines.push('プロフィール取得中...');

          // getProfileがnullを返す可能性があるのでチェック
          const profile = await liff.getProfile?.();

          if (!profile) {
            throw new Error('プロフィールを取得できませんでした');
          }

          const accessToken = liff.getAccessToken?.() || null;

          debugLines.push(`ユーザー: ${profile.displayName || 'unknown'}`);
          debugLines.push(`トークン: ${accessToken ? 'あり' : 'なし'}`);

          setState({
            isLoggedIn: true,
            isLoading: false,
            error: null,
            userId: profile.userId || null,
            displayName: profile.displayName || null,
            accessToken: accessToken,
            debugInfo: debugLines.join('\n'),
          });
        } else {
          debugLines.push('未ログイン');

          setState({
            isLoggedIn: false,
            isLoading: false,
            error: 'LINEアプリからアクセスしてください',
            userId: null,
            displayName: null,
            accessToken: null,
            debugInfo: debugLines.join('\n'),
          });
        }
      } catch (error) {
        const errorMsg = error instanceof Error ? error.message : String(error);
        debugLines.push(`エラー: ${errorMsg}`);
        console.error('LIFF Error:', error);

        setState({
          isLoggedIn: false,
          isLoading: false,
          error: errorMsg,
          userId: null,
          displayName: null,
          accessToken: null,
          debugInfo: debugLines.join('\n'),
        });
      }
    };

    initLiff();
  }, []);

  const closeWindow = () => {
    try {
      if (liff?.isInClient?.()) {
        liff.closeWindow();
      }
    } catch (e) {
      console.error('closeWindow error:', e);
    }
  };

  return {
    ...state,
    closeWindow,
    liff,
  };
};
