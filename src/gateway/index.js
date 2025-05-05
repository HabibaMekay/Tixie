const express = require('express');
const httpProxy = require('http-proxy');
const { URL } = require('url');
const redis = require('./redis/r_client.js');
const rateLimiter = require('./limiters/rate_limiter.js');
const concurrencyLimiter = require('./limiters/concurrency_limiter.js');
const verification = require('./jwt/jwt.js');
const getRawBody = require('raw-body');
const bodyParser = require('body-parser');
const queryString = require('querystring');

const instance = process.env.INSTANCE_NAME;
const targetService = process.env.TICKET_SERVICE_URL;
const targetUserService = process.env.USER_SERVICE_URL;
const targetAuthService = process.env.AUTH_SERVICE_URL;
const targetEventService = process.env.EVENT_SERVICE_URL;
const targetReserveService = process.env.RESERVE_SERVICE_URL;

const app = express();
const PORT = process.env.PORT || 8083;
app.set('trust proxy', true);
app.use(bodyParser.json());
app.use(bodyParser.urlencoded({ extended: true }));

const proxy = httpProxy.createProxyServer({ changeOrigin: true });

proxy.on( 'proxyReq', ( proxyReq, req, res, options ) => {
  const authHeader = req.headers['Authorization'];
  if (authHeader) {
    proxyReq.setHeader('Authorization', authHeader);
  }
  if ( !req.body || !Object.keys( req.body ).length ) {
    return;
  }
  let contentType = proxyReq.getHeader( 'Content-Type' );
  let bodyData;
  if ( contentType.includes( 'application/json' ) ) {
    bodyData = JSON.stringify( req.body );
  }

  if ( contentType.includes( 'application/x-www-form-urlencoded' ) ) {
    bodyData = queryString.stringify( req.body );
  }

  if ( bodyData ) {
    proxyReq.setHeader( 'Content-Length', Buffer.byteLength( bodyData ) );
    proxyReq.write( bodyData );
  }
});


app.use('/api', concurrencyLimiter);
app.use('/api', rateLimiter);

// Apply JWT verification only to protected endpoints
app.use((req, res, next) => {
  const openPaths = [
    '/api/user', // user signup     
    '/api/auth/login', //login duhh
    '/api/auth/oauth2-login', //logging in using oauth2
    '/api/auth/callback', //oauth callback
    '/api/test'
  ];
  const isOpen = openPaths.some(path => req.path.startsWith(path) && (req.method === 'POST' || req.method === 'GET'));
  if (isOpen) return next();
  return verification(req, res, next);
});


app.use('/api/user', (req, res) => {
  req.url = req.url.replace(/^\/api\/user/, '');
  proxy.web(req, res, { target: targetUserService });
});

app.use('/api/auth', (req, res) => {
  req.url = req.url.replace(/^\/api\/auth/, '');
  proxy.web(req, res, { target: targetAuthService });
});

app.use('/api/tickets', (req, res) => {
  req.url = req.url.replace(/^\/api\/tickets/, '');
  proxy.web(req, res, { target: targetService });
});

app.use('/api/event', (req, res) => {
  req.url = req.url.replace(/^\/api\/event/, '');
  proxy.web(req, res, { target: targetEventService });
});

app.use('/api/reserve', (req, res) => {
  req.url = req.url.replace(/^\/api\/reserve/, '');
  proxy.web(req, res, { target: targetReserveService });
});

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
