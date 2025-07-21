// src/socketService.js
class SocketService {
  constructor() {
    this.listeners = new Set();
    this.sendQueue = [];

    this._connect();
  }

  _connect() {
    const protocol = window.location.protocol === 'https:' ? 'wss' : 'ws';
    const endpoint = `${protocol}://localhost:8080/ws`;
    this.socket = new WebSocket(endpoint);

    this.socket.addEventListener('open', () => {
      console.log('⚡ WebSocket connected to', endpoint);
      // flush any queued messages
      this.sendQueue.forEach(msg => {
        console.log('⚡ flushing queued msg', msg);
        this.socket.send(JSON.stringify(msg));
      });
      this.sendQueue = [];
    });

    this.socket.addEventListener('message', ev => {
      let msg;
      try { msg = JSON.parse(ev.data) } catch { return }
      this.listeners.forEach(fn => fn(msg));
    });

    this.socket.addEventListener('close', () => {
      console.warn('⚡ WebSocket disconnected—reconnecting in 3s');
      setTimeout(() => this._connect(), 3000);
    });

    this.socket.addEventListener('error', err => {
      console.error('⚡ WebSocket error', err);
      this.socket.close();
    });
  }

  send(msg) {
    const data = JSON.stringify(msg);
    if (this.socket.readyState === WebSocket.OPEN) {
      console.log('⚡ sent msg to WS:', msg);
      this.socket.send(data);
    } else {
      console.warn('⚡ queueing msg (socket not open yet):', msg);
      this.sendQueue.push(msg);
    }
  }

  subscribe(fn) {
    this.listeners.add(fn);
    return () => this.listeners.delete(fn);
  }

  subscribeModule(moduleId) {
    console.log('⚡ subscribeModule(', moduleId, ')');
    this.send({ action: 'subscribe', module_id: moduleId });
  }

  unsubscribeModule(moduleId) {
    console.log('⚡ unsubscribeModule(', moduleId, ')');
    this.send({ action: 'unsubscribe', module_id: moduleId });
  }
}

export const socketService = new SocketService();
