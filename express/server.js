// express/server.js
const express = require('express');
const { createProxyMiddleware } = require('http-proxy-middleware');

const app = express();
const modules = {};

app.use(express.json());

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
app.use('/mymodules/:mod/*', (req, res, next) => {
  const mod = req.params.mod;
  const target = modules[mod];
  if (!target) return res.status(404).send(`Module "${mod}" not found`);

  createProxyMiddleware({
    target,
    changeOrigin: true,
    pathRewrite: {
      // strip off /mymodules/{mod}
      [`^/mymodules/${mod}`]: ''
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
app.use('/mymodules/:mod', (req, res, next) => {
  const mod = req.params.mod;
  const target = modules[mod];
  if (!target) return res.status(404).send(`Module "${mod}" not found`);

  createProxyMiddleware({
    target,
    changeOrigin: true,
    pathRewrite: {
      [`^/mymodules/${mod}`]: ''
    },
  })(req, res, next);
});

const PORT = process.env.PORT || 3000;
app.listen(PORT, () => {
  console.log(`Proxy listening on port ${PORT}`);
});
