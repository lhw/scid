import { getVerifyStatus, getApp } from "$lib/utils/api";

export async function load({ params, fetch }) {
  try {
    const status = await getVerifyStatus(fetch);
    return { status, app: null, id: params.id };
  } catch {
    return { status: null, app: null, id: params.id };
  }
}
