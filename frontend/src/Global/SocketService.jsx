// src/socketService.js
import { toast } from 'react-toastify';

let moduleStatusUpdater = null;

export const setModuleStatusUpdater = (fn) => {
  moduleStatusUpdater = fn;
};

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

      console.log(msg)
      if (msg?.eventType === "module_status_changed" && msg?.payload) {
        const { module_id, module_name, new_status } = msg.payload;

        const status = new_status?.toLowerCase();
        const url = `/admin/modules/${module_id}`;

        let message = '';
        let type = 'info';

        switch (status) {
          case 'enabled':
            message = `Module ${module_name} is now enabled`;
            type = 'success';
            break;
          case 'disabled':
            message = `Module ${module_name} is now disabled`;
            type = 'error';
            break;
          case 'downloading':
            message = `⬇Module ${module_name} is downloading`;
            type = 'warning';
            break;
          case 'waitingforaction':
          case 'waiting_for_action':
            message = `Module ${module_name} is waiting for an action`;
            type = 'warning';
            break;
          default:
            message = `Module ${module_name} status changed to ${status}`;
            type = 'info';
        }

        toast(message, {
          type,
          onClick: () => window.location.href = url,
          className: 'toast-simple',
        });
        if (moduleStatusUpdater) {
          console.log("thing shoulv updated!")
          moduleStatusUpdater(module_id, new_status);
        } else {
          console.log("update function not set!")
        }
      }
      if (msg?.eventType === "module_deleted" && msg?.payload) {
        const { module_id, module_name } = msg.payload;

        toast(`🗑️ Module ${module_name} was deleted`, {
          className: 'toast-simple',
          autoClose: 10000,
          onClick: () => {
            window.location.href = '/admin/modules';
          },
          onClose: () => {
            if (window.location.pathname === `/admin/modules/${module_id}`) {
              window.location.href = '/admin/modules';
            }
          }
        });
      }

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
