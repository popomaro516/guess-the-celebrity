import { useEffect, useState, type SyntheticEvent } from "react";
import ReactCrop, { type PercentCrop } from "react-image-crop";
import "react-image-crop/dist/ReactCrop.css";
import { createQuiz, uploadImage, type CreatedQuiz } from "../lib/api";
import { authErrorMessage, currentUser } from "../lib/auth";
import "./quiz-creator.css";

const MAX_FILE_SIZE = 10 * 1024 * 1024;
const ACCEPTED_TYPES = ["image/jpeg", "image/png", "image/webp"];
const initialCrop: PercentCrop = { unit: "%", x: 20, y: 20, width: 60, height: 60 };

type Progress = "idle" | "uploading" | "creating";

export default function QuizCreator() {
  const [authState, setAuthState] = useState<"loading" | "authenticated" | "error">("loading");
  const [authError, setAuthError] = useState<string | null>(null);
  const [file, setFile] = useState<File | null>(null);
  const [previewUrl, setPreviewUrl] = useState<string | null>(null);
  const [crop, setCrop] = useState<PercentCrop>(initialCrop);
  const [question, setQuestion] = useState("");
  const [choices, setChoices] = useState(["", "", "", ""]);
  const [answerIndex, setAnswerIndex] = useState(0);
  const [difficulty, setDifficulty] = useState<"easy" | "normal" | "hard">("normal");
  const [progress, setProgress] = useState<Progress>("idle");
  const [error, setError] = useState<string | null>(null);
  const [created, setCreated] = useState<CreatedQuiz | null>(null);

  useEffect(() => {
    let cancelled = false;

    async function requireUser() {
      try {
        const user = await currentUser();
        if (cancelled) return;
        if (!user) {
          window.location.replace("/login/?next=/create/");
          return;
        }
        setAuthState("authenticated");
      } catch (caught) {
        if (cancelled) return;
        setAuthError(authErrorMessage(caught));
        setAuthState("error");
      }
    }

    void requireUser();
    return () => {
      cancelled = true;
    };
  }, []);

  useEffect(() => {
    return () => {
      if (previewUrl) URL.revokeObjectURL(previewUrl);
    };
  }, [previewUrl]);

  function chooseFile(nextFile: File | undefined) {
    setError(null);
    if (!nextFile) return;
    if (!ACCEPTED_TYPES.includes(nextFile.type)) {
      setError("JPEG、PNG、WebP形式の画像を選択してください。");
      return;
    }
    if (nextFile.size > MAX_FILE_SIZE) {
      setError("画像サイズは10 MiB以下にしてください。");
      return;
    }
    if (previewUrl) URL.revokeObjectURL(previewUrl);
    setFile(nextFile);
    setPreviewUrl(URL.createObjectURL(nextFile));
    setCrop(initialCrop);
  }

  function updateChoice(index: number, value: string) {
    setChoices((current) => current.map((choice, choiceIndex) =>
      choiceIndex === index ? value : choice
    ));
  }

  async function submit(event: SyntheticEvent<HTMLFormElement>) {
    event.preventDefault();
    setError(null);

    const normalizedChoices = choices.map((choice) => choice.trim());
    if (!file) {
      setError("画像を選択してください。");
      return;
    }
    if (!crop.width || !crop.height) {
      setError("画像上でcrop範囲を選択してください。");
      return;
    }
    if (new Set(normalizedChoices).size !== 4 || normalizedChoices.some((choice) => !choice)) {
      setError("重複しない4つの選択肢を入力してください。");
      return;
    }

    try {
      setProgress("uploading");
      const imageId = await uploadImage(file);
      setProgress("creating");
      const result = await createQuiz({
        image_id: imageId,
        question: question.trim(),
        answer: normalizedChoices[answerIndex],
        choices: normalizedChoices,
        difficulty,
        crop: {
          x: roundRatio(crop.x / 100),
          y: roundRatio(crop.y / 100),
          width: roundRatio(crop.width / 100),
          height: roundRatio(crop.height / 100),
        },
      });
      setCreated(result);
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : "クイズを作成できませんでした。");
    } finally {
      setProgress("idle");
    }
  }

  if (authState === "loading") {
    return (
      <section className="creator-shell container">
        <div className="creator-loading card" aria-label="ログイン状態を確認中" />
      </section>
    );
  }

  if (authState === "error") {
    return (
      <section className="creator-shell container">
        <div className="creator-message card">
          <p className="eyebrow">Configuration required</p>
          <h2>認証設定を確認してください。</h2>
          <p>{authError}</p>
        </div>
      </section>
    );
  }

  if (created) {
    return (
      <section className="creator-shell container">
        <div className="creator-message card">
          <div className="success-mark" aria-hidden="true">✓</div>
          <p className="eyebrow">Quiz created</p>
          <h2>クイズを受け付けました。</h2>
          <p>
            画像を処理しています。処理が完了すると、作成したクイズの一覧から公開できます。
          </p>
          <p className="quiz-id">Quiz ID: {created.quiz_id}</p>
          <div className="creator-message-actions">
            <a className="button button-primary" href="/my-quizzes/">作成したクイズを見る</a>
            <button className="button button-outline" type="button" onClick={() => setCreated(null)}>
              続けて作成する
            </button>
          </div>
        </div>
      </section>
    );
  }

  return (
    <section className="creator-shell container">
      <form className="creator-layout" onSubmit={submit}>
        <div className="creator-image-card card">
          <div className="creator-section-heading">
            <span className="step-number">01</span>
            <div>
              <h2>画像とcrop範囲</h2>
              <p>ドラッグして、回答者に見せる範囲を選択します。</p>
            </div>
          </div>

          {!previewUrl ? (
            <label className="upload-dropzone">
              <input
                type="file"
                accept={ACCEPTED_TYPES.join(",")}
                onChange={(event) => chooseFile(event.target.files?.[0])}
              />
              <span className="upload-icon" aria-hidden="true">↑</span>
              <strong>画像を選択</strong>
              <span>JPEG・PNG・WebP、10 MiBまで</span>
            </label>
          ) : (
            <>
              <div className="crop-stage">
                <ReactCrop
                  crop={crop}
                  onChange={(_, percentCrop) => setCrop(percentCrop)}
                  minWidth={32}
                  minHeight={32}
                  keepSelection
                >
                  <img src={previewUrl} alt="crop範囲を指定するアップロード画像" />
                </ReactCrop>
              </div>
              <div className="file-meta">
                <div>
                  <strong>{file?.name}</strong>
                  <span>{file ? formatBytes(file.size) : ""}</span>
                </div>
                <label className="replace-file">
                  画像を変更
                  <input
                    type="file"
                    accept={ACCEPTED_TYPES.join(",")}
                    onChange={(event) => chooseFile(event.target.files?.[0])}
                  />
                </label>
              </div>
            </>
          )}
        </div>

        <div className="creator-form-card card">
          <div className="creator-section-heading">
            <span className="step-number">02</span>
            <div>
              <h2>問題の内容</h2>
              <p>問題文と4つの選択肢を入力します。</p>
            </div>
          </div>

          <div className="creator-fields">
            <div className="field">
              <label htmlFor="question">問題文</label>
              <textarea
                className="textarea"
                id="question"
                required
                maxLength={200}
                placeholder="この画像に写っている人物は誰？"
                value={question}
                onChange={(event) => setQuestion(event.target.value)}
              />
            </div>

            <fieldset className="choice-fields">
              <legend>選択肢と正解</legend>
              <p className="field-help">左の丸を選択した項目が正解になります。</p>
              {choices.map((choice, index) => (
                <div className="choice-field" key={index}>
                  <input
                    type="radio"
                    name="correct-answer"
                    aria-label={`選択肢${index + 1}を正解にする`}
                    checked={answerIndex === index}
                    onChange={() => setAnswerIndex(index)}
                  />
                  <input
                    className="input"
                    type="text"
                    required
                    maxLength={100}
                    aria-label={`選択肢${index + 1}`}
                    placeholder={`選択肢 ${index + 1}`}
                    value={choice}
                    onChange={(event) => updateChoice(index, event.target.value)}
                  />
                </div>
              ))}
            </fieldset>

            <div className="field">
              <label htmlFor="difficulty">難易度</label>
              <select
                className="select"
                id="difficulty"
                value={difficulty}
                onChange={(event) => setDifficulty(event.target.value as typeof difficulty)}
              >
                <option value="easy">かんたん</option>
                <option value="normal">ふつう</option>
                <option value="hard">むずかしい</option>
              </select>
            </div>

            {error && <p className="form-error" role="alert">{error}</p>}

            <button
              className="button button-primary create-submit"
              type="submit"
              disabled={progress !== "idle"}
            >
              {progress !== "idle" && <span className="spinner" aria-hidden="true" />}
              {progress === "uploading"
                ? "画像をアップロード中"
                : progress === "creating"
                  ? "クイズを作成中"
                  : "クイズを作成"}
            </button>
          </div>
        </div>
      </form>
    </section>
  );
}

function roundRatio(value: number): number {
  return Math.round(value * 10000) / 10000;
}

function formatBytes(bytes: number): string {
  return bytes < 1024 * 1024
    ? `${Math.round(bytes / 1024)} KB`
    : `${(bytes / 1024 / 1024).toFixed(1)} MB`;
}
