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
const { retryWithBackoff } = require('./utils/retry.js');
const http = require('http');



const instance = process.env.INSTANCE_NAME;
const targetService = process.env.TICKET_SERVICE_URL;
const targetUserService = process.env.USER_SERVICE_URL;
const targetAuthService = process.env.AUTH_SERVICE_URL;
const targetEventService = process.env.EVENT_SERVICE_URL;
const targetReserveService = process.env.RESERVE_SERVICE_URL;
const targetVendorService = process.env.VENDOR_SERVICE_URL;

const app = express();
const server = http.createServer(app);
const PORT = process.env.PORT || 8083;
app.set('trust proxy', true);
app.use(bodyParser.json());
app.use(bodyParser.urlencoded({ extended: true }));

const proxy = httpProxy.createProxyServer({ changeOrigin: true });

// Create WebSocket proxy
const wsProxy = httpProxy.createProxyServer({
  target: targetService,
  ws: true,
  changeOrigin: true
});

const retryOptions = {
  maxRetries: 3,
  initialDelay: 100,
  maxDelay: 10000,
  shouldRetry: (error) => {
    return error.code === 'ECONNREFUSED' || 
           error.code === 'ETIMEDOUT' || 
           (error.statusCode && error.statusCode >= 500); // we're checking if we even have an error status code. Without it, it would sometiems fail because there is no status code to read.
  } // this ensures we do not retry errors that are related to misinputs like 404 but server timeouts, or database errors
};

proxy.on( 'proxyReq', ( proxyReq, req, res, options ) => {
  const authHeader = req.headers['authorization'];
  if (authHeader) {
    proxyReq.setHeader('authorization', authHeader);
  }
  if (req.user) {
    try {
      proxyReq.setHeader('email', req.user.email || '');
      proxyReq.setHeader('username', req.user.username || req.user.name || '');
      
      const userData = JSON.stringify(req.user);
    
      if (userData.length < 8000) {
        proxyReq.setHeader('userData', userData);
      } else {
        console.warn(`User data too large for header: ${userData.length} bytes`);
      }
    } catch (error) {
      console.error('JWT Failed to forward user data:', error);
    }
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

proxy.on('error', (err, req, res) => {
  console.error(`[Proxy Error] ${req.method} ${req.path}:`, err);
  
  if (!res.headersSent) {
    res.status(502).json({
      error: 'Bad Gateway',
      message: 'The server encountered a temporary error and could not complete your request'
    });
  }
});

// Handle WebSocket upgrade
server.on('upgrade', (req, socket, head) => {
  const path = req.url;
  
  // Check if this is a WebSocket request for tickets service
  if (path.startsWith('/api/tickets/v1/ws/')) {
    // Verify JWT for WebSocket connections
    const token = req.headers['authorization'];
    if (!token) {
      socket.write('HTTP/1.1 401 Unauthorized\r\n\r\n');
      socket.destroy();
      return;
    }

    verification.verifyToken(token.replace('Bearer ', ''))
      .then(decoded => {
        // Store the decoded JWT in the request for potential use by the backend
        req.user = decoded;
        
        // Modify the URL to match the internal service path
        req.url = req.url.replace(/^\/api\/tickets/, '');
        wsProxy.ws(req, socket, head);
      })
      .catch(err => {
        console.error('WebSocket authentication failed:', err);
        socket.write('HTTP/1.1 401 Unauthorized\r\n\r\n');
        socket.destroy();
      });
  }
});

// Handle proxy errors
wsProxy.on('error', (err, req, socket) => {
  console.error('WebSocket proxy error:', err);
  if (socket.writable) {
    socket.write('HTTP/1.1 502 Bad Gateway\r\n\r\n');
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
    '/api/test',
    '/api/vendor/v1',
    '/api/vendor/v1/authenticate'
  ];
  const isOpen = openPaths.some(path => req.path.startsWith(path) && (req.method === 'POST' || req.method === 'GET'));
  if (isOpen) return next();
  return verification(req, res, next);
});


const vendorProtectedRoutes = [
  { path: '/api/event/v1/:id/events', methods: ['POST'] }
];

// Middleware to check if user is a vendor or not 
const isVendor = (req, res, next) => {
  const needsVendorAccess = vendorProtectedRoutes.some(route => {
    const pathPattern = route.path.replace(/:\w+/g, '[^/]+');
    const regex = new RegExp(`^${pathPattern}`);
    return regex.test(req.path) && route.methods.includes(req.method);
  });

  if (!needsVendorAccess) {
    return next();
  }

  // Log for debugging purposes
  console.log(`[VENDOR-CHECK] Checking vendor access for ${req.method} ${req.path}`);
  
  if (!req.user) {
    console.warn(`[VENDOR-CHECK] No user object found in request for ${req.path}`);
    return res.status(403).json({
      error: 'Forbidden',
      message: 'Authentication required'
    });
  }
  
  if (!req.user.role) {
    console.warn(`[VENDOR-CHECK] User ${req.user.username || 'unknown'} has no role specified in JWT`);
    return res.status(403).json({
      error: 'Forbidden',
      message: 'Missing role information'
    });
  }

  if (req.user.role !== 'vendor') {
    console.warn(`[VENDOR-CHECK] User ${req.user.username || 'unknown'} with role ${req.user.role} attempted to access vendor endpoint`);
    return res.status(403).json({
      error: 'Forbidden',
      message: 'This route is only accessible to vendors'
    });
  }
  console.log(`[VENDOR-CHECK] Vendor access granted for ${req.user.username || 'unknown'}`);
  next();
};
app.use(isVendor);

const proxyWithRetry = (req, res, target) => {
  retryWithBackoff(
    () => {
      return new Promise((resolve, reject) => {
        proxy.web(req, res, { target }, err => {
          if (err) {
            reject(err);
          } else {
            resolve();
          }
        });
      });
    },
    retryOptions
  ).catch(err => {
    console.error(`[Retry Failed] ${req.method} ${req.path} to ${target}:`, err);
    if (!res.headersSent) {
      res.status(503).json({
        error: 'Service Unavailable',
        message: 'The service is currently unavailable. Please try again later'
      });
    }
  });
};

app.use('/api/user', (req, res) => {
  req.url = req.url.replace(/^\/api\/user/, '');
  proxyWithRetry(req, res, targetUserService);
});

app.use('/api/auth', (req, res) => {
  req.url = req.url.replace(/^\/api\/auth/, '');
  proxyWithRetry(req, res, targetAuthService);
});

app.use('/api/tickets', (req, res) => {
  req.url = req.url.replace(/^\/api\/tickets/, '');
  proxyWithRetry(req, res, targetService);
});

app.use('/api/event', (req, res) => {
  req.url = req.url.replace(/^\/api\/event/, '');
  proxyWithRetry(req, res, targetEventService);
});

app.use('/api/reserve', (req, res) => {
  req.url = req.url.replace(/^\/api\/reserve/, '');
  proxyWithRetry(req, res, targetReserveService);
});

app.use('/api/vendor', (req, res) => {
  req.url = req.url.replace(/^\/api\/vendor/, '');
  proxyWithRetry(req, res, targetReserveService);
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

// Start server using the HTTP server instead of app.listen
server.listen(PORT, () => {
  console.log(`API Gateway running on port ${PORT} with WebSocket support`);
});