// express/server.js
const express = require('express');
const { createProxyMiddleware } = require('http-proxy-middleware');

const app = express();
const modules = {};

app.use(express.json());

app.use('/modules', (req, res, next) => {
  // CSP: only allow framing by your frontend (adjust origin as needed)
  res.setHeader('Content-Security-Policy', "frame-ancestors 'self' http://localhost");

  const dest = req.get('sec-fetch-dest');
  const ref  = req.get('referer') || '';

  // If browser doesn’t declare iframe, or referer isn’t your origin → block
  if (dest !== 'iframe' && !ref.startsWith('http://localhost/')) {
    return res.status(403).send('Forbidden: module must be embedded in iframe');
  }
  next();
});

// 1) Module registration
app.post('/__register', (req, res) => {
  const { name, url } = req.body;
  if (!name || !url) {
    return res.status(400).send('Must provide { name, url }');
  }
  modules[name] = url;
  console.log(`Registered module "${name}" → ${url}`);
  res.sendStatus(200);
});

// 2) Proxy any /modules/:mod/* to the registered URL
app.use('/modules/:mod/*', (req, res, next) => {
  const mod = req.params.mod;
  const target = modules[mod];
  if (!target) return res.status(404).send(`Module "${mod}" not found`);

  createProxyMiddleware({
    target,
    changeOrigin: true,
    pathRewrite: {
      // strip off /modules/{mod}
      [`^/modules/${mod}`]: ''
    },
  })(req, res, next);
});

app.delete('/__register/:mod', (req, res) => {
  const mod = req.params.mod;
  if (!modules[mod]) {
    return res.status(404).send(`Module "${mod}" not found`);
  }

  delete modules[mod];
  console.log(`Unregistered module "${mod}"`);
  res.sendStatus(204);  // No Content
});

// 3) Also catch the root of a module: /modules/:mod → / on the target
app.use('/modules/:mod', (req, res, next) => {
  const mod = req.params.mod;
  const target = modules[mod];
  if (!target) return res.status(404).send(`Module "${mod}" not found`);

  createProxyMiddleware({
    target,
    changeOrigin: true,
    pathRewrite: {
      [`^/modules/${mod}`]: ''
    },
  })(req, res, next);
});

const PORT = process.env.PORT || 3000;
app.listen(PORT, () => {
  console.log(`Proxy listening on port ${PORT}`);
});
