import * as staticPublic from '$env/static/public';

const env = staticPublic as Record<string, string | undefined>;

export const PUBLIC_POCKET_ID_URL = env.PUBLIC_POCKET_ID_URL ?? 'https://id.scid.my';
export const PUBLIC_OIDC_CLIENT_ID = env.PUBLIC_OIDC_CLIENT_ID ?? 'scid-frontend';
export const PUBLIC_TURNSTILE_SITE_KEY = env.PUBLIC_TURNSTILE_SITE_KEY ?? '';