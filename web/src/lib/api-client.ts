/**
 * API Client — satu-satunya tempat yang memanggil `fetch` langsung.
 *
 * Sesuai docs/16-frontend-architecture.md & docs/07-api-design.md:
 * - Auto-attach Authorization header
 * - Parse error envelope { error: { code, message, details } }
 * - Auto-retry dengan refresh token saat 401
 */

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export interface ApiErrorDetail {
  field: string;
  issue: string;
}

export interface ApiErrorBody {
  code: string;
  message: string;
  details?: ApiErrorDetail[];
}

export class ApiError extends Error {
  public readonly status: number;
  public readonly code: string;
  public readonly details: ApiErrorDetail[];

  constructor(status: number, body: ApiErrorBody) {
    super(body.message);
    this.name = "ApiError";
    this.status = status;
    this.code = body.code;
    this.details = body.details ?? [];
  }
}

// ---------------------------------------------------------------------------
// Token storage (in-memory — bukan localStorage, sesuai 16-frontend-architecture.md)
// ---------------------------------------------------------------------------

let accessToken: string | null = null;
let refreshToken: string | null = null;

export function setTokens(access: string, refresh: string): void {
  accessToken = access;
  refreshToken = refresh;
}

export function getAccessToken(): string | null {
  return accessToken;
}

export function clearTokens(): void {
  accessToken = null;
  refreshToken = null;
}

// ---------------------------------------------------------------------------
// Base URL
// ---------------------------------------------------------------------------

const BASE_URL = import.meta.env.VITE_API_BASE_URL as string | undefined ?? "http://localhost:8080/api/v1";

// ---------------------------------------------------------------------------
// Core fetch wrapper
// ---------------------------------------------------------------------------

interface RequestOptions extends Omit<RequestInit, "body"> {
  body?: unknown;
}

async function parseErrorResponse(res: Response): Promise<ApiError> {
  try {
    const json = (await res.json()) as { error?: ApiErrorBody };
    if (json.error) {
      return new ApiError(res.status, json.error);
    }
  } catch {
    // Response bukan JSON — buat error generik
  }
  return new ApiError(res.status, {
    code: "UNKNOWN_ERROR",
    message: `Request failed with status ${res.status.toString()}`,
  });
}

async function refreshAccessToken(): Promise<boolean> {
  if (!refreshToken) return false;

  try {
    const res = await fetch(`${BASE_URL}/auth/refresh`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ refresh_token: refreshToken }),
    });

    if (!res.ok) return false;

    const data = (await res.json()) as {
      data: { access_token: string; refresh_token: string };
    };
    setTokens(data.data.access_token, data.data.refresh_token);
    return true;
  } catch {
    return false;
  }
}

async function request<T>(
  endpoint: string,
  options: RequestOptions = {},
): Promise<T> {
  const { body, headers: customHeaders, ...rest } = options;

  const headers = new Headers(customHeaders);
  headers.set("Content-Type", "application/json");

  if (accessToken) {
    headers.set("Authorization", `Bearer ${accessToken}`);
  }

  const config: RequestInit = {
    ...rest,
    headers,
    body: body !== undefined ? JSON.stringify(body) : undefined,
  };

  let res = await fetch(`${BASE_URL}${endpoint}`, config);

  // Auto-retry dengan refresh token saat 401
  if (res.status === 401 && accessToken) {
    const refreshed = await refreshAccessToken();
    if (refreshed) {
      headers.set("Authorization", `Bearer ${accessToken!}`);
      const retryConfig: RequestInit = { ...config, headers };
      res = await fetch(`${BASE_URL}${endpoint}`, retryConfig);
    }
  }

  if (!res.ok) {
    throw await parseErrorResponse(res);
  }

  // 204 No Content
  if (res.status === 204) {
    return undefined as T;
  }

  return (await res.json()) as T;
}

// ---------------------------------------------------------------------------
// Public API methods
// ---------------------------------------------------------------------------

export const apiClient = {
  get<T>(endpoint: string, options?: RequestOptions): Promise<T> {
    return request<T>(endpoint, { ...options, method: "GET" });
  },

  post<T>(endpoint: string, body?: unknown, options?: RequestOptions): Promise<T> {
    return request<T>(endpoint, { ...options, method: "POST", body });
  },

  put<T>(endpoint: string, body?: unknown, options?: RequestOptions): Promise<T> {
    return request<T>(endpoint, { ...options, method: "PUT", body });
  },

  patch<T>(endpoint: string, body?: unknown, options?: RequestOptions): Promise<T> {
    return request<T>(endpoint, { ...options, method: "PATCH", body });
  },

  delete<T>(endpoint: string, options?: RequestOptions): Promise<T> {
    return request<T>(endpoint, { ...options, method: "DELETE" });
  },
};
