import { getVerifyStatus } from "$lib/utils/api";

export async function load({ fetch }) {
  try {
    const status = await getVerifyStatus(fetch);
    if (!status.verified) {
      return { status, apps: [] };
    }
    // listApps uses client-side auth token (not SSR fetch) — only available in browser
    return { status, apps: [] };
  } catch {
    return { status: null, apps: [] };
  }
}
