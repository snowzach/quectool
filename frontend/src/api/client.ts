export class ApiError extends Error {
  constructor(public status: number, public code: string, message: string) {
    super(message);
    this.name = "ApiError";
  }
}

export interface ApiOptions extends Omit<RequestInit, "signal"> {
  /** Per-call timeout. Default 10s. Use a larger value for /api/modem/scan. */
  timeoutMs?: number;
  /** If true, a 401 response triggers a redirect to /login. Default true. */
  redirectOn401?: boolean;
}

export async function apiFetch<T = unknown>(
  path: string,
  opts: ApiOptions = {},
): Promise<T> {
  const { timeoutMs = 10_000, redirectOn401 = true, headers, ...rest } = opts;
  const ctrl = new AbortController();
  const t = setTimeout(() => ctrl.abort(), timeoutMs);
  let res: Response;
  try {
    res = await fetch(path, {
      ...rest,
      credentials: "include",
      headers: { Accept: "application/json", ...(headers ?? {}) },
      signal: ctrl.signal,
    });
  } finally {
    clearTimeout(t);
  }

  if (res.status === 401 && redirectOn401 && typeof window !== "undefined") {
    if (window.location.pathname !== "/login") {
      window.location.assign("/login");
    }
  }

  if (res.status === 204) return null as T;

  const contentType = res.headers.get("content-type") ?? "";
  const body = contentType.includes("application/json") ? await res.json() : null;

  if (!res.ok) {
    const code = (body && body.code) || `HTTP_${res.status}`;
    const message = (body && body.error) || res.statusText || "request failed";
    throw new ApiError(res.status, code, message);
  }
  return body as T;
}
