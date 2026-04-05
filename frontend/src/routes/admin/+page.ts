import { getVerifyStatus } from '$lib/utils/api';

export async function load({ fetch }) {
  try {
    const status = await getVerifyStatus(fetch);
    return { status };
  } catch {
    return { status: null };
  }
}
