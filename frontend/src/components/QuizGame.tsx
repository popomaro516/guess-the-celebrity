import { useCallback, useEffect, useRef, useState } from "react";
import {
  ApiError,
  answerQuiz,
  getRandomQuiz,
  type AnswerResult,
  type PublicQuiz,
} from "../lib/api";
import "./quiz-game.css";

const difficultyLabels = {
  easy: "かんたん",
  normal: "ふつう",
  hard: "むずかしい",
} as const;

export default function QuizGame() {
  const [quiz, setQuiz] = useState<PublicQuiz | null>(null);
  const [selected, setSelected] = useState<string | null>(null);
  const [result, setResult] = useState<AnswerResult | null>(null);
  const [loading, setLoading] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const prefetchedQuiz = useRef<Promise<PublicQuiz> | null>(null);

  const loadQuiz = useCallback(async () => {
    setLoading(true);
    setError(null);
    setSelected(null);
    setResult(null);

    try {
      const nextQuiz = prefetchedQuiz.current
        ? await prefetchedQuiz.current
        : await getRandomQuiz();
      prefetchedQuiz.current = null;
      setQuiz(nextQuiz);
    } catch (caught) {
      setQuiz(null);
      setError(
        caught instanceof ApiError && caught.status === 404
          ? "公開中のクイズはまだありません。"
          : "クイズを読み込めませんでした。時間をおいてお試しください。",
      );
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void loadQuiz();
  }, [loadQuiz]);

  async function submitAnswer() {
    if (!quiz || !selected || submitting) return;
    setSubmitting(true);
    setError(null);
    try {
      const answer = await answerQuiz(quiz.quiz_id, selected);
      setResult(answer);
      prefetchedQuiz.current = getRandomQuiz();
      void prefetchedQuiz.current.catch(() => {
        prefetchedQuiz.current = null;
      });
    } catch {
      setError("回答を送信できませんでした。もう一度お試しください。");
    } finally {
      setSubmitting(false);
    }
  }

  if (loading) {
    return (
      <section className="quiz-shell container" aria-live="polite">
        <div className="quiz-card card quiz-loading">
          <div className="image-skeleton" />
          <div className="line-skeleton wide" />
          <div className="line-skeleton" />
        </div>
      </section>
    );
  }

  if (!quiz) {
    return (
      <section className="quiz-shell container">
        <div className="empty-card card">
          <p className="eyebrow">No quiz found</p>
          <h2>次の一問を準備しています。</h2>
          <p>{error}</p>
          <button className="button button-primary" type="button" onClick={loadQuiz}>
            もう一度読み込む
          </button>
        </div>
      </section>
    );
  }

  if (result) {
    const hasReveal = result.correct_answer && result.original_image_url;
    return (
      <section className="quiz-shell container" aria-live="polite">
        <article className="result-card card">
          <div className="result-image">
            {result.original_image_url ? (
              <img src={result.original_image_url} alt="クイズの正解となる元画像" />
            ) : (
              <div className="result-image-missing">
                <span>正解画像を取得できませんでした</span>
              </div>
            )}
          </div>
          <div className="result-copy">
            <p className="eyebrow">{result.correct ? "Correct answer" : "The answer is"}</p>
            <h2>{result.correct ? "正解です。" : "答えはこちら。"}</h2>
            {hasReveal ? (
              <p className="correct-answer">{result.correct_answer}</p>
            ) : (
              <p className="result-note">
                正解情報のAPI対応後、この場所に正解と元画像が表示されます。
              </p>
            )}
            <p className="your-answer">あなたの回答：{selected}</p>
            <button className="button button-primary" type="button" onClick={loadQuiz}>
              次のクイズへ <span aria-hidden="true">→</span>
            </button>
          </div>
        </article>
      </section>
    );
  }

  return (
    <section className="quiz-shell container">
      <article className="quiz-card card">
        <div className="quiz-image-wrap">
          <img
            className="quiz-image"
            src={quiz.cropped_image_url}
            alt="クイズのヒントとなる切り抜き画像"
            fetchPriority="high"
          />
          <span className={`difficulty difficulty-${quiz.difficulty}`}>
            {difficultyLabels[quiz.difficulty]}
          </span>
        </div>

        <div className="quiz-content">
          <p className="quiz-count">Question</p>
          <h2>{quiz.question}</h2>
          <div className="answer-grid" role="radiogroup" aria-label="回答の選択肢">
            {quiz.choices.map((choice, index) => (
              <button
                className={`answer-choice ${selected === choice ? "selected" : ""}`}
                type="button"
                role="radio"
                aria-checked={selected === choice}
                key={choice}
                onClick={() => setSelected(choice)}
              >
                <span className="choice-letter">{String.fromCharCode(65 + index)}</span>
                <span>{choice}</span>
              </button>
            ))}
          </div>
          {error && <p className="form-error">{error}</p>}
          <div className="quiz-actions">
            <button
              className="button button-primary"
              type="button"
              disabled={!selected || submitting}
              onClick={submitAnswer}
            >
              {submitting && <span className="spinner" aria-hidden="true" />}
              回答する
            </button>
          </div>
        </div>
      </article>
    </section>
  );
}
