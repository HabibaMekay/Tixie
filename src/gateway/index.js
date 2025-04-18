const express = require('express');
const cors = require('cors');
const { createProxyMiddleware } = require('http-proxy-middleware');
const redis = require('./redis/r_client.js');
const rateLimiter = require('./limiters/rate_limiter.js');
const concurrencyLimiter = require('./limiters/concurrency_limiter.js');
const throttlingLimiter = require('./limiters/throttling_limiter.js');
const verification = require('./jwt/jwt.js');

const instance = process.env.INSTANCE_NAME;
const targetService = process.env.TICKET_SERVICE_URL;
const targetUserService = process.env.USER_SERVICE_URL;
const targetAuthService = process.env.AUTH_SERVICE_URL;

const app = express();
const PORT = process.env.PORT || 8083;

app.use(cors());
app.use(express.json());

// Header for gateway instance
app.use((req, res, next) => {
  res.setHeader('X-Gateway-Instance', instance);
  next();
});

// app.use('/api', concurrencyLimiter);
// app.use('/api', throttlingLimiter);
// app.use('/api', rateLimiter);

// Apply JWT verification only to protected endpoints
app.use((req, res, next) => {
  const openPaths = [
    '/api/v1/user',         
    '/api/v1/auth/login',
    '/api/v1/auth/oauth2-login',
    '/api/v1/auth/callback',
    '/api/test'
  ];
  const isOpen = openPaths.some(path => req.path.startsWith(path) && (req.method === 'POST' || req.method === 'GET'));
  if (isOpen) return next();
  return verification(req, res, next);
});


app.use('/api/v1/tickets', createProxyMiddleware({
  target: targetService,
  changeOrigin: true,
  pathRewrite: { '^/api/v1/tickets': '' },
  onProxyReq: (proxyReq, req) => {
    console.log(`[PROXY] ${req.method} ${req.originalUrl} → ${targetService}/tickets${req.url}`);
  },
}));


app.use('/api/v1/user', createProxyMiddleware({
  target: targetUserService,
  changeOrigin: true,
  secure: true,
  pathRewrite: { ['/api/v1/user']: '' },
  onProxyReq: (proxyReq, req) => {
    console.log(`[PROXY] ${req.method} ${req.originalUrl} → ${targetUserService}${req.url}`);
  },
}));


app.use('/api/v1/auth', createProxyMiddleware({
  target: targetAuthService,
  changeOrigin: true,
  pathRewrite: { '^/api/v1/auth': '' },
  onProxyReq: (proxyReq, req) => {
    console.log(`[PROXY] ${req.method} ${req.originalUrl} → ${targetAuthService}/auth${req.url}`);
  },
}));


app.get('/api/test', (req, res) => {
  res.json({ message: 'Success! You have not hit the rate limit.' ,
    targetU : targetUserService,
    targetT : targetService,
  });
});

app.get('/debug/redis-keys', async (req, res) => {
  try {
    const keys = await redis.keys('*');
    res.json({ keys });
  } catch (err) {
    console.error('[DEBUG] Error fetching Redis keys:', err);
    res.status(500).json({ error: 'Failed to fetch Redis keys' });
  }
});

// Start server
app.listen(PORT, () => {
  console.log(`API Gateway running on port ${PORT}`);
});
