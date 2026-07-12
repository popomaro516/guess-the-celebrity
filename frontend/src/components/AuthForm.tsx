import { useEffect, useMemo, useState, type SyntheticEvent } from "react";
import {
  confirmResetPassword,
  confirmSignUp,
  resendSignUpCode,
  resetPassword,
  signIn,
  signUp,
} from "aws-amplify/auth";
import {
  authErrorMessage,
  configureAuth,
  safeNextPath,
} from "../lib/auth";
import "./auth-form.css";

type Mode = "login" | "signup" | "confirm" | "forgot";

interface Props {
  mode: Mode;
}

const passwordRequirements = [
  { label: "8文字以上", test: (value: string) => value.length >= 8 },
  { label: "英大文字を1文字以上", test: (value: string) => /[A-Z]/.test(value) },
  { label: "英小文字を1文字以上", test: (value: string) => /[a-z]/.test(value) },
  { label: "数字を1文字以上", test: (value: string) => /[0-9]/.test(value) },
  { label: "記号を1文字以上", test: (value: string) => /[^A-Za-z0-9]/.test(value) },
] as const;

const content = {
  login: {
    eyebrow: "Welcome back",
    title: "ログイン",
    description: "クイズを作成するにはログインしてください。",
  },
  signup: {
    eyebrow: "Create an account",
    title: "アカウント作成",
    description: "メールアドレスとパスワードで始められます。",
  },
  confirm: {
    eyebrow: "Check your inbox",
    title: "メールアドレスを確認",
    description: "Cognitoから届いた確認コードを入力してください。",
  },
  forgot: {
    eyebrow: "Reset password",
    title: "パスワードを再設定",
    description: "登録済みのメールアドレスへ確認コードを送信します。",
  },
} as const;

export default function AuthForm({ mode }: Props) {
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [passwordConfirmation, setPasswordConfirmation] = useState("");
  const [code, setCode] = useState("");
  const [forgotStep, setForgotStep] = useState<"request" | "confirm">("request");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [message, setMessage] = useState<string | null>(null);

  useEffect(() => {
    setEmail(sessionStorage.getItem("pendingEmail") ?? "");
    const params = new URLSearchParams(window.location.search);
    if (mode === "login" && params.has("confirmed")) {
      setMessage("メールアドレスを確認しました。ログインしてください。");
    } else if (mode === "login" && params.has("reset")) {
      setMessage("パスワードを更新しました。");
    }
  }, [mode]);

  const nextPath = useMemo(() => {
    if (typeof window === "undefined") return "/create/";
    return safeNextPath(new URLSearchParams(window.location.search).get("next"));
  }, []);

  function initializeAuth(): boolean {
    try {
      configureAuth();
      return true;
    } catch (caught) {
      setError(authErrorMessage(caught));
      return false;
    }
  }

  async function submit(event: SyntheticEvent<HTMLFormElement>) {
    event.preventDefault();
    setError(null);
    setMessage(null);
    if (!initializeAuth()) return;

    if ((mode === "signup" || (mode === "forgot" && forgotStep === "confirm")) &&
        password !== passwordConfirmation) {
      setError("パスワードが一致しません。");
      return;
    }

    if (mode === "signup" || (mode === "forgot" && forgotStep === "confirm")) {
      const unmetRequirements = passwordRequirements.filter(({ test }) => !test(password));
      if (unmetRequirements.length > 0) {
        setError(`パスワードには「${unmetRequirements.map(({ label }) => label).join("・")}」が必要です。`);
        return;
      }
    }

    setSubmitting(true);
    try {
      if (mode === "login") {
        const result = await signIn({ username: email.trim(), password });
        if (result.isSignedIn) {
          window.location.assign(nextPath);
          return;
        }
        if (result.nextStep.signInStep === "CONFIRM_SIGN_UP") {
          rememberEmail(email);
          window.location.assign(`/confirm/?next=${encodeURIComponent(nextPath)}`);
          return;
        }
        throw new Error("追加の認証設定が必要です。");
      }

      if (mode === "signup") {
        const result = await signUp({
          username: email.trim(),
          password,
          options: { userAttributes: { email: email.trim() } },
        });
        rememberEmail(email);
        if (result.isSignUpComplete) {
          window.location.assign(`/login/?next=${encodeURIComponent(nextPath)}`);
        } else {
          window.location.assign(`/confirm/?next=${encodeURIComponent(nextPath)}`);
        }
        return;
      }

      if (mode === "confirm") {
        await confirmSignUp({ username: email.trim(), confirmationCode: code.trim() });
        sessionStorage.removeItem("pendingEmail");
        window.location.assign(`/login/?confirmed=1&next=${encodeURIComponent(nextPath)}`);
        return;
      }

      if (forgotStep === "request") {
        await resetPassword({ username: email.trim() });
        rememberEmail(email);
        setForgotStep("confirm");
        setMessage("確認コードを送信しました。");
      } else {
        await confirmResetPassword({
          username: email.trim(),
          confirmationCode: code.trim(),
          newPassword: password,
        });
        window.location.assign("/login/?reset=1");
      }
    } catch (caught) {
      setError(authErrorMessage(caught));
    } finally {
      setSubmitting(false);
    }
  }

  async function resendCode() {
    setError(null);
    setMessage(null);
    if (!initializeAuth()) return;
    setSubmitting(true);
    try {
      await resendSignUpCode({ username: email.trim() });
      setMessage("新しい確認コードを送信しました。");
    } catch (caught) {
      setError(authErrorMessage(caught));
    } finally {
      setSubmitting(false);
    }
  }

  const showPassword =
    mode === "login" || mode === "signup" || (mode === "forgot" && forgotStep === "confirm");
  const showCode = mode === "confirm" || (mode === "forgot" && forgotStep === "confirm");

  return (
    <section className="auth-section container">
      <div className="auth-atmosphere" aria-hidden="true">
        <div className="auth-orb auth-orb-one" />
        <div className="auth-orb auth-orb-two" />
        <p>一問を作ることは、<br />新しい見方を渡すこと。</p>
      </div>

      <div className="auth-card card">
        <div className="auth-heading">
          <p className="eyebrow">{content[mode].eyebrow}</p>
          <h1>{content[mode].title}</h1>
          <p>{content[mode].description}</p>
        </div>

        <form className="auth-form" onSubmit={submit}>
          <div className="field">
            <label htmlFor="email">メールアドレス</label>
            <input
              className="input"
              id="email"
              name="email"
              type="email"
              autoComplete="email"
              required
              value={email}
              onChange={(event) => setEmail(event.target.value)}
            />
          </div>

          {showCode && (
            <div className="field">
              <label htmlFor="code">確認コード</label>
              <input
                className="input code-input"
                id="code"
                name="code"
                type="text"
                inputMode="numeric"
                autoComplete="one-time-code"
                required
                value={code}
                onChange={(event) => setCode(event.target.value)}
              />
            </div>
          )}

          {showPassword && (
            <div className="field">
              <label htmlFor="password">
                {mode === "forgot" ? "新しいパスワード" : "パスワード"}
              </label>
              <input
                className="input"
                id="password"
                name="password"
                type="password"
                autoComplete={mode === "login" ? "current-password" : "new-password"}
                minLength={8}
                required
                value={password}
                onChange={(event) => setPassword(event.target.value)}
              />
              {mode !== "login" && (
                <div className="password-requirements" aria-label="パスワードの要件">
                  <p>次のすべてを含めてください。</p>
                  <ul>
                    {passwordRequirements.map(({ label, test }) => {
                      const isMet = test(password);
                      return (
                        <li className={isMet ? "is-met" : undefined} key={label}>
                          <span aria-hidden="true">{isMet ? "✓" : "○"}</span>
                          {label}
                        </li>
                      );
                    })}
                  </ul>
                </div>
              )}
            </div>
          )}

          {(mode === "signup" || (mode === "forgot" && forgotStep === "confirm")) && (
            <div className="field">
              <label htmlFor="password-confirmation">パスワード（確認）</label>
              <input
                className="input"
                id="password-confirmation"
                name="password-confirmation"
                type="password"
                autoComplete="new-password"
                minLength={8}
                required
                value={passwordConfirmation}
                onChange={(event) => setPasswordConfirmation(event.target.value)}
              />
            </div>
          )}

          {error && <p className="form-error" role="alert">{error}</p>}
          {message && <p className="form-success" role="status">{message}</p>}

          <button className="button button-primary auth-submit" type="submit" disabled={submitting}>
            {submitting && <span className="spinner" aria-hidden="true" />}
            {submitLabel(mode, forgotStep)}
          </button>
        </form>

        <div className="auth-links">
          {mode === "login" && (
            <>
              <a href="/forgot-password/">パスワードを忘れた方</a>
              <span>アカウントをお持ちでない方は <a href="/signup/">新規登録</a></span>
            </>
          )}
          {mode === "signup" && <span>登録済みの方は <a href="/login/">ログイン</a></span>}
          {mode === "confirm" && (
            <button type="button" onClick={resendCode} disabled={submitting}>
              確認コードを再送する
            </button>
          )}
          {mode === "forgot" && <a href="/login/">ログインへ戻る</a>}
        </div>
      </div>
    </section>
  );
}

function rememberEmail(email: string) {
  sessionStorage.setItem("pendingEmail", email.trim());
}

function submitLabel(mode: Mode, forgotStep: "request" | "confirm"): string {
  if (mode === "login") return "ログイン";
  if (mode === "signup") return "アカウントを作成";
  if (mode === "confirm") return "メールアドレスを確認";
  return forgotStep === "request" ? "確認コードを送信" : "パスワードを更新";
}
