package api

const sessionAccessTokenKey = "access_token"

func sessionCookieName(secure bool) string {
	if secure {
		return "__Host-scid_session"
	}
	return "scid_session"
}
