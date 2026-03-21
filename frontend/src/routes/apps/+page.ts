import { getVerifyStatus, listApps } from "$lib/utils/api";
import { getAccessToken } from "$lib/utils/auth";

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
