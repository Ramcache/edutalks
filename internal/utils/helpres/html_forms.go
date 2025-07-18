package helpers

import (
	"fmt"
)

func BuildNewsHTML(title, content, url string) string {
	return fmt.Sprintf(`
<html>
  <body style="font-family:Arial,sans-serif;background:#f7f7f7;padding:0;margin:0;">
    <table width="100%%" bgcolor="#f7f7f7" cellpadding="0" cellspacing="0" style="padding:30px 0;">
      <tr>
        <td align="center">
          <table width="600" bgcolor="#fff" cellpadding="24" cellspacing="0" style="border-radius:10px;box-shadow:0 2px 8px #eee;">
            <tr>
              <td>
                <h2 style="color:#2d74da;margin-top:0;">%s</h2>
                <p style="font-size:16px;color:#333;">%s</p>
                <p>
                  <a href="%s" style="display:inline-block;padding:12px 24px;background:#2d74da;color:#fff;text-decoration:none;border-radius:5px;font-weight:bold;margin-top:16px;">
                    Читать новость
                  </a>
                </p>
                <hr style="border:none;border-top:1px solid #eee;margin:32px 0 12px 0;">
                <p style="font-size:12px;color:#999;margin:0;">
                  Вы получили это письмо, потому что подписаны на уведомления Edutalks.<br>
                  <i>Если вы не хотите получать такие письма — отпишитесь в настройках профиля.</i>
                </p>
              </td>
            </tr>
          </table>
        </td>
      </tr>
    </table>
  </body>
</html>
`, title, content, url)
}

func BuildSimpleHTML(title, body string) string {
	return fmt.Sprintf(`
<html>
  <body style="font-family:Arial,sans-serif; background:#f9f9f9;">
    <table width="100%%" cellpadding="0" cellspacing="0" bgcolor="#f9f9f9">
      <tr>
        <td align="center" style="padding:32px 0;">
          <table width="500" bgcolor="#fff" cellpadding="24" cellspacing="0" style="border-radius:8px; box-shadow:0 1px 6px #eee;">
            <tr>
              <td>
                <h2 style="color:#2d74da; margin-top:0;">%s</h2>
                <div style="font-size:16px; color:#222;">%s</div>
                <hr style="margin:32px 0 16px 0; border:0; border-top:1px solid #eee;">
                <div style="font-size:12px; color:#999;">Письмо сгенерировано автоматически. Не отвечайте на него.</div>
              </td>
            </tr>
          </table>
        </td>
      </tr>
    </table>
  </body>
</html>
`, title, body)
}

func BuildVerificationHTML(name, link string) string {
	return fmt.Sprintf(`
<html>
  <body style="font-family:Arial,sans-serif; background:#f9f9f9;">
    <table width="100%%" cellpadding="0" cellspacing="0" bgcolor="#f9f9f9">
      <tr>
        <td align="center" style="padding:32px 0;">
          <table width="500" bgcolor="#fff" cellpadding="24" cellspacing="0" style="border-radius:8px; box-shadow:0 1px 6px #eee;">
            <tr>
              <td>
                <h2 style="color:#2d74da; margin-top:0;">Подтверждение почты</h2>
                <div style="font-size:16px; color:#222;">Здравствуйте, %s!</div>
                <p style="margin:24px 0;">
                  Для подтверждения вашей электронной почты нажмите кнопку ниже:
                </p>
                <p>
                  <a href="%s" style="display:inline-block;padding:12px 24px;background:#2d74da;color:#fff;text-decoration:none;border-radius:5px;font-weight:bold;">
                    Подтвердить почту
                  </a>
                </p>
                <hr style="margin:32px 0 16px 0; border:0; border-top:1px solid #eee;">
                <div style="font-size:12px; color:#999;">Если вы не регистрировались на сайте, просто проигнорируйте это письмо.</div>
              </td>
            </tr>
          </table>
        </td>
      </tr>
    </table>
  </body>
</html>
`, name, link)
}
