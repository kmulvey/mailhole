const fs = require("fs");
const path = require("path");
const sendMail = require("./sendmail");
const { Mailhole } = require("./client");

// Add at the top if running in Node.js and not in a browser
const WebSocket = require("ws");

const testEmailBody = fs.readFileSync(
  path.resolve(__dirname, "../testdata/email.txt"),
  "utf8"
);

const recipient = "test@recipient.com";
const wsRecipient = "test+ws@recipient.com";
const baseUrl = "http://localhost:8080";
const smtpHost = "localhost";
const smtpPort = 2525;

test("can send and fetch multiple emails, and test all client methods", async () => {
  // Send 5 emails with different subjects
  for (let i = 0; i < 5; i++) {
    await sendMail({
      smtpHost,
      smtpPort,
      from: "test@sender.com",
      to: recipient,
      subject: `Hello ${i}`,
      text: testEmailBody,
    });
  }

  // Wait for processing
  await new Promise((r) => setTimeout(r, 1000));

  const mail = new Mailhole({ baseUrl });
  const mailbox = mail.mailbox(recipient);

  // Test .all()
  const messages = await mailbox.all();
  expect(Array.isArray(messages)).toBe(true);
  expect(messages.length).toBeGreaterThanOrEqual(5);

  // Test .first()
  const first = await mailbox.first().get();
  expect(first.subject).toBe("Hello 0");

  // Test .last()
  const last = await mailbox.last().get();
  expect(last.subject).toBe("Hello 4");

  // Test .at(index)
  for (let i = 0; i < 5; i++) {
    const msg = await mailbox.at(i).get();
    expect(msg.subject).toBe(`Hello ${i}`);
    expect(msg.body).toContain("Please click here:");
    expect(msg.links).toBeDefined();
    expect(msg.links.length).toBeGreaterThan(0);
  }
});

// TODO: this test passes but does not complete. Something about closing the WebSocket connection is not handled properly.
test("can receive notifications via websocket", async () => {
  const mail = new Mailhole({ baseUrl });
  const mailbox = mail.mailbox(wsRecipient);
  const ws = mailbox.stream();

  try {
    await new Promise((resolve, reject) => {
      ws.onopen = resolve;
      ws.onerror = reject;
    });

    const notificationPromise = new Promise((resolve, reject) => {
      ws.onmessage = (event) => {
        console.log("WebSocket message:", event.data);
        try {
          const data = JSON.parse(event.data);
          resolve(data);
        } catch (err) {
          reject(err);
        }
      };
      ws.onclose = () => reject(new Error("WebSocket closed unexpectedly."));
      ws.onerror = reject;
    });

    await sendMail({
      smtpHost,
      smtpPort,
      from: "test@sender.com",
      to: wsRecipient,
      subject: "WebSocket Test",
      text: "This is a test.",
    });

    const notification = await notificationPromise;
    expect(notification.subject).toBe("WebSocket Test");
    expect(notification.recipient).toBe(wsRecipient);
  } finally {
    if (ws && ws.readyState !== WebSocket.CLOSED) {
      console.log("Closing WebSocket connection.");
      ws.close();
      console.log("WebSocket connection closed.");
    }
  }
});
