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

        // LIFF初期化 - 自動ログインを無効化
        await liff.init({
          liffId,
          withLoginOnExternalBrowser: false // 外部ブラウザでの自動ログインを無効化
        });

        debugLines.push('LIFF init完了');

        const isInClient = liff.isInClient();
        const isLoggedIn = liff.isLoggedIn();
        const os = liff.getOS();

        debugLines.push(`isInClient: ${isInClient}`);
        debugLines.push(`isLoggedIn: ${isLoggedIn}`);
        debugLines.push(`OS: ${os}`);

        if (isLoggedIn) {
          const profile = await liff.getProfile();
          const accessToken = liff.getAccessToken();

          debugLines.push(`ユーザー: ${profile.displayName}`);

          setState({
            isLoggedIn: true,
            isLoading: false,
            error: null,
            userId: profile.userId,
            displayName: profile.displayName,
            accessToken: accessToken || null,
            debugInfo: debugLines.join('\n'),
          });
        } else {
          debugLines.push('未ログイン - ログインリダイレクトはしない');

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
    if (liff.isInClient()) {
      liff.closeWindow();
    }
  };

  return {
    ...state,
    closeWindow,
    liff,
  };
};
