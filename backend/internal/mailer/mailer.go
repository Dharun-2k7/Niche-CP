package mailer

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/resend/resend-go/v3"
)

// brandedEmailWrapper wraps content in a consistent NicheCP branded email template
func brandedEmailWrapper(title string, bodyContent string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>%s</title>
</head>
<body style="margin:0; padding:0; background-color:#060606; font-family:-apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;">
<table role="presentation" cellpadding="0" cellspacing="0" width="100%%" style="background-color:#060606;">
<tr>
<td align="center" style="padding: 40px 20px;">
<table role="presentation" cellpadding="0" cellspacing="0" width="100%%" style="max-width:520px; background:#101010; border:1px solid rgba(255,255,255,0.08); border-radius:16px; overflow:hidden;">

<!-- Logo Header -->
<tr>
<td style="padding:32px 40px 24px; text-align:center; border-bottom:1px solid rgba(255,255,255,0.06);">
<div style="font-size:24px; font-weight:700; letter-spacing:-0.5px;">
<span style="color:#3B82F6;">Niche</span><span style="color:#FAFAFA;">CP</span>
</div>
<div style="font-size:12px; color:#6B7280; margin-top:4px; text-transform:uppercase; letter-spacing:1.5px;">Competitive Programming Platform</div>
</td>
</tr>

<!-- Main Content -->
<tr>
<td style="padding:32px 40px;">
%s
</td>
</tr>

<!-- Footer -->
<tr>
<td style="padding:24px 40px 32px; border-top:1px solid rgba(255,255,255,0.06); text-align:center;">
<div style="font-size:12px; color:#4B5563; line-height:1.6;">
Amrita Vishwa Vidyapeetham — Nagercoil Campus<br>
ICPC Competitive Programming Club<br>
<span style="color:#6B7280;">This is an automated message. Do not reply.</span>
</div>
</td>
</tr>

</table>
</td>
</tr>
</table>
</body>
</html>`, title, bodyContent)
}

// SendEmailVerification sends a branded OTP verification email
func SendEmailVerification(to string, otpCode string, expiryMinutes string) error {
	apiKey := os.Getenv("RESEND_API_KEY")
	from := os.Getenv("RESEND_FROM_EMAIL")

	if apiKey == "" || from == "" {
		log.Printf("\n========== MOCK EMAIL ==========\nTo: %s\nOTP Code: %s\nExpiry: %s minutes\n================================\n",
			to, otpCode, expiryMinutes)
		return nil
	}

	subject := fmt.Sprintf("NicheCP — Your Verification Code: %s", otpCode)

	bodyContent := fmt.Sprintf(`
<h2 style="color:#FAFAFA; font-size:20px; font-weight:600; margin:0 0 8px; letter-spacing:-0.3px;">Verify Your Email</h2>
<p style="color:#9CA3AF; font-size:14px; line-height:1.6; margin:0 0 28px;">Use the verification code below to complete your registration on NicheCP.</p>

<!-- OTP Code Block -->
<div style="background:#0A0A0A; border:1px solid rgba(59,130,246,0.2); border-radius:12px; padding:24px; text-align:center; margin-bottom:24px;">
<div style="font-size:36px; font-weight:800; letter-spacing:8px; color:#FAFAFA; font-family:'Courier New', monospace;">%s</div>
<div style="font-size:12px; color:#6B7280; margin-top:8px;">Verification Code</div>
</div>

<div style="background:rgba(245,158,11,0.08); border:1px solid rgba(245,158,11,0.15); border-radius:8px; padding:12px 16px; margin-bottom:24px;">
<div style="font-size:13px; color:#F59E0B; display:flex; align-items:center;">
⏱ This code expires in <strong style="margin-left:4px;">%s minutes</strong>
</div>
</div>

<p style="color:#6B7280; font-size:13px; line-height:1.6; margin:0;">
If you did not request this code, you can safely ignore this email. Someone may have entered your email address by mistake.
</p>`, otpCode, expiryMinutes)

	htmlBody := brandedEmailWrapper("NicheCP Email Verification", bodyContent)

	client := resend.NewClient(apiKey)
	params := &resend.SendEmailRequest{
		From:    from,
		To:      []string{to},
		Subject: subject,
		Html:    htmlBody,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := client.Emails.SendWithContext(ctx, params)
	if err != nil {
		log.Printf("[Mailer Error] Failed to send verification email to %s: %v", to, err)
		return err
	}

	log.Printf("[Mailer] Verification email sent to %s (Resend ID: %s)", to, resp.Id)
	return nil
}

// SendEmail sends a generic email using Resend or logs to console if not configured
func SendEmail(to string, subject string, body string) error {
	apiKey := os.Getenv("RESEND_API_KEY")
	from := os.Getenv("RESEND_FROM_EMAIL")

	if apiKey == "" || from == "" {
		log.Printf("\n========== MOCK EMAIL ==========\nTo: %s\nSubject: %s\n\n%s\n================================\n",
			to, subject, body)
		return nil
	}

	client := resend.NewClient(apiKey)

	params := &resend.SendEmailRequest{
		From:    from,
		To:      []string{to},
		Subject: subject,
		Html:    body,
		Text:    body,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := client.Emails.SendWithContext(ctx, params)
	if err != nil {
		log.Printf("[Mailer Error] Failed to send email to %s: %v", to, err)
		return err
	}

	log.Printf("[Mailer] Email sent to %s (Resend ID: %s)", to, resp.Id)
	return nil
}

// SendPasswordReset sends a branded password reset email with a clickable link
func SendPasswordReset(to string, resetUrl string, expiryMinutes string) error {
	apiKey := os.Getenv("RESEND_API_KEY")
	from := os.Getenv("RESEND_FROM_EMAIL")

	if apiKey == "" || from == "" {
		log.Printf("\n========== MOCK EMAIL ==========\nTo: %s\nURL: %s\nExpiry: %s minutes\n================================\n",
			to, resetUrl, expiryMinutes)
		return nil
	}

	subject := "NicheCP — Reset Your Password"

	bodyContent := fmt.Sprintf(`
<h2 style="color:#FAFAFA; font-size:20px; font-weight:600; margin:0 0 8px; letter-spacing:-0.3px;">Reset Your Password</h2>
<p style="color:#9CA3AF; font-size:14px; line-height:1.6; margin:0 0 28px;">We received a request to reset the password for your NicheCP account. Click the button below to set a new password.</p>

<!-- Reset Button -->
<div style="text-align:center; margin-bottom:24px;">
<a href="%s" target="_blank" style="display:inline-block; background:#3B82F6; color:#FFFFFF; text-decoration:none; padding:14px 32px; border-radius:10px; font-size:15px; font-weight:600; letter-spacing:0.3px;">Reset Password</a>
</div>

<div style="background:rgba(245,158,11,0.08); border:1px solid rgba(245,158,11,0.15); border-radius:8px; padding:12px 16px; margin-bottom:24px;">
<div style="font-size:13px; color:#F59E0B;">
⏱ This link expires in <strong>%s minutes</strong>
</div>
</div>

<p style="color:#6B7280; font-size:13px; line-height:1.6; margin:0 0 12px;">
If you did not request a password reset, you can safely ignore this email. Your password will remain unchanged.
</p>

<div style="background:#0A0A0A; border:1px solid rgba(255,255,255,0.06); border-radius:8px; padding:12px 16px;">
<div style="font-size:11px; color:#4B5563; word-break:break-all;">
If the button doesn't work, copy and paste this link:<br>
<span style="color:#6B7280;">%s</span>
</div>
</div>`, resetUrl, expiryMinutes, resetUrl)

	htmlBody := brandedEmailWrapper("NicheCP Password Reset", bodyContent)

	client := resend.NewClient(apiKey)
	params := &resend.SendEmailRequest{
		From:    from,
		To:      []string{to},
		Subject: subject,
		Html:    htmlBody,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := client.Emails.SendWithContext(ctx, params)
	if err != nil {
		log.Printf("[Mailer Error] Failed to send reset email to %s: %v", to, err)
		return err
	}

	log.Printf("[Mailer] Reset email sent to %s (Resend ID: %s)", to, resp.Id)
	return nil
}
