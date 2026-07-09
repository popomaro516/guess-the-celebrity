import { useEffect, useState } from "react";
import { currentUser, logOut } from "../lib/auth";
import "./account-panel.css";

export default function AccountPanel() {
  const [state, setState] = useState<"loading" | "signed-out" | "signed-in">("loading");
  const [username, setUsername] = useState("");

  useEffect(() => {
    let cancelled = false;

    async function loadUser() {
      try {
        const user = await currentUser();
        if (cancelled) return;
        if (!user) {
          setState("signed-out");
          return;
        }
        setUsername(user.signInDetails?.loginId ?? user.username);
        setState("signed-in");
      } catch {
        if (!cancelled) setState("signed-out");
      }
    }

    void loadUser();
    return () => {
      cancelled = true;
    };
  }, []);

  async function handleLogout() {
    await logOut();
    setState("signed-out");
  }

  if (state === "loading") {
    return <div className="account-card card account-loading" aria-label="アカウント情報を確認中" />;
  }

  if (state === "signed-out") {
    return (
      <div className="account-card card">
        <p className="eyebrow">Your account</p>
        <h1>ログインしていません。</h1>
        <p>クイズの作成と公開にはアカウントが必要です。</p>
        <div className="account-actions">
          <a className="button button-primary" href="/login/">ログイン</a>
          <a className="button button-outline" href="/signup/">アカウント作成</a>
        </div>
      </div>
    );
  }

  return (
    <div className="account-card card">
      <p className="eyebrow">Your account</p>
      <h1>アカウント</h1>
      <p className="account-email">{username}</p>
      <div className="account-actions">
        <a className="button button-primary" href="/create/">クイズを作成</a>
        <a className="button button-outline" href="/my-quizzes/">作成したクイズ</a>
        <button className="button button-danger" type="button" onClick={handleLogout}>
          ログアウト
        </button>
      </div>
    </div>
  );
}
