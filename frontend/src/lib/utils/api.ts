export interface VerifyStatus {
  verified: boolean;
  handle: string;
  verified_at: string;
  pending_handle: string;
  pending_expires_at: string;
}

export interface StartVerifyResponse {
  token: string;
  expires_at: string;
}

export interface ConfirmVerifyResponse {
  verified: boolean;
  handle?: string;
  message?: string;
}

// Fetch current verification status for the logged-in user
export async function getVerifyStatus(
  fetch: typeof globalThis.fetch,
): Promise<VerifyStatus> {
  const res = await fetch("/api/verify/status");
  if (!res.ok) throw new Error(`Status ${res.status}`);
  return res.json();
}

// Start a new verification for the given RSI handle
export async function startVerify(
  handle: string,
): Promise<StartVerifyResponse> {
  const res = await fetch("/api/verify/start", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ handle }),
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: "Unknown error" }));
    throw new Error(err.error ?? `Status ${res.status}`);
  }
  return res.json();
}

// Confirm verification (check that the token is in the RSI bio)
export async function confirmVerify(): Promise<ConfirmVerifyResponse> {
  const res = await fetch("/api/verify/confirm", { method: "POST" });
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: "Unknown error" }));
    throw new Error(err.error ?? `Status ${res.status}`);
  }
  return res.json();
}
