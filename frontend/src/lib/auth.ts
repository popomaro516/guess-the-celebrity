import { Amplify } from "aws-amplify";
import {
  fetchAuthSession,
  getCurrentUser,
  signOut,
  type AuthUser,
} from "aws-amplify/auth";

let configured = false;

export function configureAuth(): void {
  if (configured) return;

  const region = import.meta.env.PUBLIC_AWS_REGION;
  const userPoolId = import.meta.env.PUBLIC_COGNITO_USER_POOL_ID;
  const userPoolClientId = import.meta.env.PUBLIC_COGNITO_USER_POOL_CLIENT_ID;

  if (!region || !userPoolId || !userPoolClientId) {
    throw new Error("Cognitoの環境変数が設定されていません。");
  }

  Amplify.configure({
    Auth: {
      Cognito: {
        userPoolId,
        userPoolClientId,
        loginWith: { email: true },
        signUpVerificationMethod: "code",
      },
    },
  });
  configured = true;
}

export async function currentUser(): Promise<AuthUser | null> {
  configureAuth();
  try {
    return await getCurrentUser();
  } catch {
    return null;
  }
}

export async function accessToken(): Promise<string> {
  configureAuth();
  const session = await fetchAuthSession();
  const token = session.tokens?.accessToken?.toString();
  if (!token) throw new Error("ログインが必要です。");
  return token;
}

export async function logOut(): Promise<void> {
  configureAuth();
  await signOut();
}

export function safeNextPath(value: string | null, fallback = "/create/"): string {
  return value?.startsWith("/") && !value.startsWith("//") ? value : fallback;
}

export function authErrorMessage(error: unknown): string {
  if (!(error instanceof Error)) return "認証処理に失敗しました。";

  const messages: Record<string, string> = {
    UsernameExistsException: "このメールアドレスは既に登録されています。",
    UserNotFoundException: "メールアドレスまたはパスワードが正しくありません。",
    NotAuthorizedException: "メールアドレスまたはパスワードが正しくありません。",
    CodeMismatchException: "確認コードが正しくありません。",
    ExpiredCodeException: "確認コードの有効期限が切れています。再送してください。",
    LimitExceededException: "試行回数が上限に達しました。時間をおいてお試しください。",
    InvalidPasswordException: "パスワードが要件を満たしていません。",
    UserNotConfirmedException: "メールアドレスの確認が完了していません。",
  };

  return messages[error.name] ?? error.message ?? "認証処理に失敗しました。";
}
