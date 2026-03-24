package mailer

// newAppTmplSrc is the HTML source for the new-app admin notification email.
const newAppTmplSrc = `<!DOCTYPE html>
<html lang="en">
<head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>New app pending review</title>
</head>
<body style="margin:0;padding:0;background:#0a0e1a;font-family:system-ui,sans-serif;color:#e2e8f0;">
<table width="100%" cellpadding="0" cellspacing="0" style="background:#0a0e1a;padding:32px 16px;">
  <tr><td align="center">
    <table width="560" cellpadding="0" cellspacing="0" style="max-width:560px;width:100%;background:#0d1526;border:1px solid #1e3a5f;border-radius:8px;overflow:hidden;">

      <!-- header -->
      <tr>
        <td style="padding:24px 28px 20px;border-bottom:1px solid #1e3a5f;">
          <table cellpadding="0" cellspacing="0">
            <tr>
              <td style="padding-right:12px;vertical-align:middle;">
                <svg xmlns="http://www.w3.org/2000/svg" width="28" height="28" viewBox="0 0 32 32" fill="none">
                  <rect width="32" height="32" rx="6" fill="#0d1526"/>
                  <polygon points="16,4 26,12 22,28 10,28 6,12" fill="none" stroke="#00d4ff" stroke-width="1.5" stroke-linejoin="round"/>
                  <polygon points="16,8 23,13 20,26 12,26 9,13" fill="#00d4ff" fill-opacity="0.08"/>
                  <text x="16" y="22" text-anchor="middle" font-family="Verdana,Arial,sans-serif" font-size="13" font-weight="700" fill="#00d4ff">S</text>
                </svg>
              </td>
              <td style="vertical-align:middle;">
                <span style="font-size:17px;font-weight:700;color:#e2e8f0;letter-spacing:0.4px;">SCID</span>
                <span style="font-size:13px;color:#64748b;margin-left:6px;">Admin Notification</span>
              </td>
            </tr>
          </table>
        </td>
      </tr>

      <!-- body -->
      <tr>
        <td style="padding:28px 28px 0;">
          <p style="margin:0 0 6px;font-size:18px;font-weight:600;color:#e2e8f0;">New app pending review</p>
          <p style="margin:0 0 24px;font-size:14px;color:#94a3b8;">A verified user submitted a new OIDC application that requires your approval.</p>

          <!-- detail card -->
          <table width="100%" cellpadding="0" cellspacing="0" style="background:#0a0e1a;border:1px solid #1e3a5f;border-radius:6px;margin-bottom:24px;">
            <tr>
              <td style="padding:18px 20px;">
                <table width="100%" cellpadding="0" cellspacing="0">
                  {{template "row" list "App name" .AppName}}
                  {{template "row" list "App ID" .AppID}}
                  {{template "row" list "Submitted by" .OwnerHandle}}
                  {{template "row" list "Submitted at" (formatTime .CreatedAt)}}
                  {{template "row" list "Verified-only" (yesNo .VerifiedOnly)}}
                  {{template "row" list "Listed in directory" (yesNo .Listed)}}
                </table>
                {{if .RedirectURIs}}
                <p style="margin:12px 0 4px;font-size:11px;text-transform:uppercase;letter-spacing:0.6px;color:#64748b;">Redirect URIs</p>
                {{range .RedirectURIs}}<p style="margin:2px 0;font-size:12px;color:#00d4ff;font-family:monospace,monospace;word-break:break-all;">{{.}}</p>{{end}}
                {{end}}
              </td>
            </tr>
          </table>

          {{if .AdminURL}}
          <table cellpadding="0" cellspacing="0" style="margin-bottom:28px;">
            <tr>
              <td style="background:#00d4ff;border-radius:6px;">
                <a href="{{.AdminURL}}" style="display:inline-block;padding:11px 24px;font-size:14px;font-weight:600;color:#0a0e1a;text-decoration:none;letter-spacing:0.2px;">Review application &#8594;</a>
              </td>
            </tr>
          </table>
          {{end}}
        </td>
      </tr>

      <!-- footer -->
      <tr>
        <td style="padding:16px 28px 22px;border-top:1px solid #1e3a5f;">
          <p style="margin:0;font-size:12px;color:#475569;">This is an automated message from SCID. Do not reply to this email.</p>
        </td>
      </tr>

    </table>
  </td></tr>
</table>
</body>
</html>
`
