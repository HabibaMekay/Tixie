const express = require('express');
const httpProxy = require('http-proxy');
const { URL } = require('url');
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

app.use(express.json());
app.use(express.urlencoded({ extended: true }));

// app.use('/api', concurrencyLimiter);
// app.use('/api', throttlingLimiter);
// app.use('/api', rateLimiter);

// Apply JWT verification only to protected endpoints
// app.use((req, res, next) => {
//   const openPaths = [
//     '/api/v1/user',
//     '/api/v1/tickets',         
//     '/api/v1/auth/login',
//     '/api/v1/auth/oauth2-login',
//     '/api/v1/auth/callback',
//     '/api/test'
//   ];
//   const isOpen = openPaths.some(path => req.path.startsWith(path) && (req.method === 'POST' || req.method === 'GET'));
//   if (isOpen) return next();
//   return verification(req, res, next);
// });


const proxyOptions = (target) => ({
  target,
  changeOrigin: true,
  pathRewrite: (path, req) => path.replace(/^\/api\/v1\/(user|auth|tickets)/, ''),
  selfHandleResponse: false,
  logLevel: 'debug',
  onProxyReq: (proxyReq, req, res) => {
    if (req.body && Object.keys(req.body).length) {
      const bodyData = JSON.stringify(req.body);

      proxyReq.setHeader('Content-Type', 'application/json');
      proxyReq.setHeader('Content-Length', Buffer.byteLength(bodyData));
      proxyReq.write(bodyData);
    }
  }
});

app.use('/api/v1/user', createProxyMiddleware(proxyOptions(targetUserService)));
app.use('/api/v1/auth', createProxyMiddleware(proxyOptions(targetAuthService)));
app.use('/api/v1/tickets', createProxyMiddleware(proxyOptions(targetService)));


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
