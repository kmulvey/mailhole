// test/sendMail.js
const nodemailer = require("nodemailer");

async function sendMail({ smtpHost, smtpPort, from, to, subject, text }) {
  let transporter = nodemailer.createTransport({
    host: smtpHost,
    port: smtpPort,
    secure: false, // true for 465, false for other ports
    tls: { rejectUnauthorized: false },
  });

  await transporter.sendMail({
    from,
    to,
    subject,
    text,
  });
}

module.exports = sendMail;
