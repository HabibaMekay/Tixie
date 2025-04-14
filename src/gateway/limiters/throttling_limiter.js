const redis = require('../redis/r_client'); // assume Redis is set up

const WINDOW_MS = 10 * 1000; // 10 seconds window
const MIN_INTERVAL = 1000;   // Min 2 seconds between requests per IP

const throttlingMiddleware = async (req, res, next) => {
  const ip = req.ip;
  const key = `throttle:${ip}`;
  const now = Date.now();

  try {
    const lastRequest = await redis.get(key);

    if (lastRequest && (now - parseInt(lastRequest)) < MIN_INTERVAL) {
      const wait = MIN_INTERVAL - (now - parseInt(lastRequest));
      return res.status(429).json({
        error: 'Too many requests – you are being throttled.',
        retryIn: `${wait}ms`,
      });
    }

    await redis.set(key, now, 'PX', WINDOW_MS); // set expiration for cleanup
    next();
  } catch (err) {
    console.error('[THROTTLING ERROR]', err);
    next(); // fail open
  }
};

module.exports = throttlingMiddleware;
