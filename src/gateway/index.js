const express = require('express');
const cors = require('cors');
const { createProxyMiddleware } = require('http-proxy-middleware');
const redis = require('./redis/r_client.js');
const rateLimiter = require('./limiters/rate_limiter.js');
const concurrencyLimiter = require('./limiters/concurrency_limiter.js');
const throttlingLimiter = require('./limiters/throttling_limiter.js');

const instance = process.env.INSTANCE_NAME;
const targetService = process.env.TICKET_SERVICE_URL;

const app = express();
const PORT = process.env.PORT || 8083;

app.use(cors());
app.use(express.json());

app.use((req, res, next) => {
  res.setHeader('X-Gateway-Instance', process.env.INSTANCE_NAME);
  next();
});

app.use('/api', concurrencyLimiter);
app.use('/api', throttlingLimiter);
app.use('/api', rateLimiter);

console.log(`[PROXY] Proxying request to ${targetService}:8082`),
app.use('/api/v1/tickets', createProxyMiddleware({
  target: targetService,
  changeOrigin: true,
  pathRewrite: {
    '^/api/v1': '',
  },
  onProxyReq: (proxyReq, req, res) => {
    console.log(`[PROXY] ${req.method} ${req.originalUrl} → ${targetService}/tickets${req.url}`);
  },
}));

// Sample route
app.get('/api/test', (req, res) => {
  res.json({ message: 'Success! You have not hit the rate limit.' });
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
  
  app.listen(PORT, () => {
    console.log(`API Gateway running on port ${PORT}`);
  });