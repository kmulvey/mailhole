// Add at the top if using Node.js
// const WebSocket = require('ws');

class Mailhole {
  constructor(config) {
    this.baseUrl = config.baseUrl;
  }
  mailbox(email) {
    return new Mailbox(this.baseUrl, email);
  }
}

class Mailbox {
  constructor(baseUrl, email) {
    this.baseUrl = baseUrl;
    this.email = email;
  }
  async all() {
    const res = await fetch(
      `${this.baseUrl}/emails/${encodeURIComponent(this.email)}/messages`
    );
    return res.json();
  }
  first() {
    return new MessageQuery(this.baseUrl, this.email, "first");
  }
  last() {
    return new MessageQuery(this.baseUrl, this.email, "last");
  }
  at(index) {
    return new MessageQuery(this.baseUrl, this.email, index);
  }
  stream() {
    const wsUrl =
      this.baseUrl.replace(/^http/, "ws") +
      `/emails/${encodeURIComponent(this.email)}/stream`;
    return new WebSocket(wsUrl);
  }
}

class MessageQuery {
  constructor(baseUrl, email, which) {
    this.baseUrl = baseUrl;
    this.email = email;
    this.which = which;
  }
  async get() {
    const res = await fetch(
      `${this.baseUrl}/emails/${this.email}/messages/${this.which}`
    );
    return res.json();
  }
  async links() {
    return (await this.get()).links;
  }
  async subject() {
    return (await this.get()).subject;
  }
  async body() {
    return (await this.get()).body;
  }
}

module.exports = { Mailhole };
