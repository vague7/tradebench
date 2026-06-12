import type {
  ApiErrorShape,
  LeaderboardEntry,
  Submission,
  SubmissionResults,
  SubmissionUploadResponse,
} from '../types/api';

const API_BASE = import.meta.env.VITE_API_BASE ?? '';

export class ApiError extends Error {
  public readonly code: string;
  public readonly status: number;

  constructor(status: number, payload: ApiErrorShape) {
    super(payload.error);
    this.name = 'ApiError';
    this.code = payload.code;
    this.status = status;
  }
}

async function parseResponse<T>(response: Response): Promise<T> {
  const text = await response.text();

  if (text.length === 0 && response.ok) {
    return {} as T;
  }

  let parsed: unknown;
  try {
    parsed = JSON.parse(text);
  } catch {
    throw new ApiError(response.status, {
      error: 'Invalid JSON response from server',
      code: 'PARSE_ERROR',
    });
  }

  if (!response.ok) {
    const errorBody = parsed as Partial<ApiErrorShape>;
    throw new ApiError(response.status, {
      error: errorBody.error ?? 'Request failed',
      code: errorBody.code ?? 'REQUEST_FAILED',
    });
  }

  return parsed as T;
}

function buildUrl(path: string): string {
  return `${API_BASE}${path}`;
}

export async function uploadSubmission(input: {
  teamName: string;
  token: string;
  zipFile: File;
}): Promise<SubmissionUploadResponse> {
  const form = new FormData();
  form.append('teamName', input.teamName);
  form.append('file', input.zipFile);

  const response = await fetch(buildUrl('/api/submissions'), {
    method: 'POST',
    headers: { Authorization: `Bearer ${input.token}` },
    body: form,
  });

  return parseResponse<SubmissionUploadResponse>(response);
}

export async function getSubmissionStatus(
  submissionId: string,
  token: string,
): Promise<Submission> {
  const response = await fetch(
    buildUrl(`/api/submissions/${submissionId}/status`),
    { headers: { Authorization: `Bearer ${token}` } },
  );

  return parseResponse<Submission>(response);
}

export async function getSubmissionResults(
  submissionId: string,
  token: string,
): Promise<SubmissionResults> {
  const response = await fetch(
    buildUrl(`/api/submissions/${submissionId}/results`),
    { headers: { Authorization: `Bearer ${token}` } },
  );

  return parseResponse<SubmissionResults>(response);
}

export async function getLeaderboard(): Promise<LeaderboardEntry[]> {
  const response = await fetch(buildUrl('/api/leaderboard'));
  return parseResponse<LeaderboardEntry[]>(response);
}

export async function startBenchmark(
  submissionId: string,
  token: string,
): Promise<{ ok: boolean }> {
  const response = await fetch(
    buildUrl(`/api/admin/benchmark/${submissionId}/start`),
    {
      method: 'POST',
      headers: { Authorization: `Bearer ${token}` },
    },
  );
  return parseResponse<{ ok: boolean }>(response);
}

export async function stopBenchmark(
  submissionId: string,
  token: string,
): Promise<{ ok: boolean }> {
  const response = await fetch(
    buildUrl(`/api/admin/benchmark/${submissionId}/stop`),
    {
      method: 'POST',
      headers: { Authorization: `Bearer ${token}` },
    },
  );
  return parseResponse<{ ok: boolean }>(response);
}
