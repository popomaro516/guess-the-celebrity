import { accessToken } from "./auth";

const API_BASE = (import.meta.env.PUBLIC_API_BASE_URL ?? "/api").replace(/\/$/, "");

export interface PublicQuiz {
  quiz_id: string;
  question: string;
  cropped_image_url: string;
  choices: string[];
  difficulty: "easy" | "normal" | "hard";
}

export interface AnswerResult {
  correct: boolean;
  correct_answer?: string;
  original_image_url?: string;
}

export interface CropRatio {
  x: number;
  y: number;
  width: number;
  height: number;
}

export interface CreateQuizInput {
  image_id: string;
  question: string;
  answer: string;
  choices: string[];
  difficulty: "easy" | "normal" | "hard";
  crop: CropRatio;
}

export interface CreatedQuiz {
  quiz_id: string;
  status: "processing";
}

export interface OwnedQuiz {
  quiz_id: string;
  question: string;
  difficulty: "easy" | "normal" | "hard";
  status: "processing" | "ready" | "published" | "failed";
  cropped_image_url?: string;
  created_at?: string;
}

export class ApiError extends Error {
  constructor(
    message: string,
    readonly status: number,
  ) {
    super(message);
    this.name = "ApiError";
  }
}

async function request<T>(
  path: string,
  init?: RequestInit,
  authenticated = false,
): Promise<T> {
  const headers = new Headers(init?.headers);
  if (init?.body && !headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json");
  }
  if (authenticated) {
    headers.set("Authorization", `Bearer ${await accessToken()}`);
  }

  const response = await fetch(`${API_BASE}${path}`, { ...init, headers });
  if (!response.ok) {
    let message = "通信に失敗しました。";
    try {
      const body = (await response.json()) as { error?: string };
      if (body.error) message = body.error;
    } catch {
      // Keep the user-facing fallback for non-JSON responses.
    }
    throw new ApiError(message, response.status);
  }
  if (response.status === 204) {
    return undefined as T;
  }

  return response.json() as Promise<T>;
}

export async function getRandomQuizzes(count = 4): Promise<PublicQuiz[]> {
  const response = await request<{ quizzes: PublicQuiz[] }>(`/quizzes/random?count=${count}`);
  return response.quizzes;
}

export function answerQuiz(quizId: string, answer: string): Promise<AnswerResult> {
  return request<AnswerResult>(`/quizzes/${encodeURIComponent(quizId)}/answer`, {
    method: "POST",
    body: JSON.stringify({ answer }),
  });
}

export async function uploadImage(file: File): Promise<string> {
  const presign = await request<{
    image_id: string;
    upload_url: string;
  }>(
    "/uploads/presign",
    {
      method: "POST",
      body: JSON.stringify({
        filename: file.name,
        content_type: file.type,
        size: file.size,
      }),
    },
    true,
  );

  const uploadResponse = await fetch(presign.upload_url, {
    method: "PUT",
    headers: { "Content-Type": file.type },
    body: file,
  });
  if (!uploadResponse.ok) {
    throw new ApiError("画像のアップロードに失敗しました。", uploadResponse.status);
  }

  await request(
    `/images/${encodeURIComponent(presign.image_id)}/complete`,
    { method: "POST" },
    true,
  );
  return presign.image_id;
}

export function createQuiz(input: CreateQuizInput): Promise<CreatedQuiz> {
  return request<CreatedQuiz>(
    "/quizzes",
    { method: "POST", body: JSON.stringify(input) },
    true,
  );
}

export async function getMyQuizzes(): Promise<OwnedQuiz[]> {
  const response = await request<OwnedQuiz[] | { quizzes: OwnedQuiz[] }>(
    "/quizzes/mine",
    undefined,
    true,
  );
  return Array.isArray(response) ? response : response.quizzes;
}

export function publishQuiz(quizId: string): Promise<{ quiz_id: string; status: "published" }> {
  return request(
    `/quizzes/${encodeURIComponent(quizId)}/publish`,
    { method: "POST" },
    true,
  );
}

export function deleteQuiz(quizId: string): Promise<void> {
  return request(
    `/quizzes/${encodeURIComponent(quizId)}`,
    { method: "DELETE" },
    true,
  );
}
