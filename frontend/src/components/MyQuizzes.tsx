import { useCallback, useEffect, useState } from "react";
import {
  ApiError,
  deleteQuiz,
  getMyQuizzes,
  publishQuiz,
  type OwnedQuiz,
} from "../lib/api";
import { authErrorMessage, currentUser } from "../lib/auth";
import "./my-quizzes.css";

const statusLabels = {
  processing: "画像を処理中",
  ready: "公開できます",
  published: "公開済み",
  failed: "処理に失敗",
} as const;

export default function MyQuizzes() {
  const [quizzes, setQuizzes] = useState<OwnedQuiz[]>([]);
  const [loading, setLoading] = useState(true);
  const [publishingId, setPublishingId] = useState<string | null>(null);
  const [deletingId, setDeletingId] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [notImplemented, setNotImplemented] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const user = await currentUser();
      if (!user) {
        window.location.replace("/login/?next=/my-quizzes/");
        return;
      }
      setQuizzes(await getMyQuizzes());
    } catch (caught) {
      if (caught instanceof ApiError && caught.status === 501) {
        setNotImplemented(true);
      } else {
        setError(authErrorMessage(caught));
      }
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  async function publish(quizId: string) {
    setPublishingId(quizId);
    setError(null);
    try {
      await publishQuiz(quizId);
      setQuizzes((items) => items.map((item) =>
        item.quiz_id === quizId ? { ...item, status: "published" } : item
      ));
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : "公開できませんでした。");
    } finally {
      setPublishingId(null);
    }
  }

  async function remove(quiz: OwnedQuiz) {
    if (!window.confirm(`「${quiz.question}」を削除しますか？`)) {
      return;
    }
    setDeletingId(quiz.quiz_id);
    setError(null);
    try {
      await deleteQuiz(quiz.quiz_id);
      setQuizzes((items) => items.filter((item) => item.quiz_id !== quiz.quiz_id));
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : "削除できませんでした。");
    } finally {
      setDeletingId(null);
    }
  }

  if (loading) {
    return <div className="quiz-list-loading card" aria-label="クイズ一覧を読み込み中" />;
  }

  if (notImplemented) {
    return (
      <div className="list-message card">
        <p className="eyebrow">Coming soon</p>
        <h2>一覧APIの準備中です。</h2>
        <p>APIが利用可能になると、ここから処理状況の確認と公開ができます。</p>
        <a className="button button-primary" href="/create/">クイズを作成</a>
      </div>
    );
  }

  if (quizzes.length === 0 && !error) {
    return (
      <div className="list-message card">
        <p className="eyebrow">No quizzes yet</p>
        <h2>最初のクイズを作りましょう。</h2>
        <p>作成したクイズの処理状況と公開状態がここに表示されます。</p>
        <a className="button button-primary" href="/create/">クイズを作成</a>
      </div>
    );
  }

  return (
    <>
      {error && <p className="form-error quiz-list-error" role="alert">{error}</p>}
      <div className="quiz-list">
        {quizzes.map((quiz) => (
          <article className="owned-quiz card" key={quiz.quiz_id}>
            <div className="owned-quiz-image">
              {quiz.cropped_image_url ? (
                <img src={quiz.cropped_image_url} alt="" />
              ) : (
                <span aria-hidden="true">⌗</span>
              )}
            </div>
            <div className="owned-quiz-content">
              <div className="owned-quiz-meta">
                <span className={`status status-${quiz.status}`}>{statusLabels[quiz.status]}</span>
                <span>{difficultyLabel(quiz.difficulty)}</span>
              </div>
              <h2>{quiz.question}</h2>
              {quiz.created_at && (
                <time dateTime={quiz.created_at}>
                  {new Intl.DateTimeFormat("ja-JP", { dateStyle: "medium" }).format(new Date(quiz.created_at))}
                </time>
              )}
              <div className="owned-quiz-actions">
                {quiz.status === "ready" && (
                  <button
                    className="button button-primary"
                    type="button"
                    disabled={publishingId === quiz.quiz_id || deletingId === quiz.quiz_id}
                    onClick={() => publish(quiz.quiz_id)}
                  >
                    {publishingId === quiz.quiz_id && <span className="spinner" aria-hidden="true" />}
                    公開する
                  </button>
                )}
                <button
                  className="button button-danger"
                  type="button"
                  disabled={deletingId === quiz.quiz_id || publishingId === quiz.quiz_id}
                  onClick={() => remove(quiz)}
                >
                  {deletingId === quiz.quiz_id && <span className="spinner" aria-hidden="true" />}
                  消去
                </button>
              </div>
            </div>
          </article>
        ))}
      </div>
    </>
  );
}

function difficultyLabel(value: OwnedQuiz["difficulty"]): string {
  return { easy: "かんたん", normal: "ふつう", hard: "むずかしい" }[value];
}
