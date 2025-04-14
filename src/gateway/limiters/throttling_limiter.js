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
// set expiration for cleanup for keys without an already made timer or option 'PX'.
// there were others but this seemed the most suitable
    await redis.set(key, now, 'PX', WINDOW_MS); 
    next();
  } catch (err) {
    console.error('[THROTTLING ERROR]', err);
    next(); // Passing the error to the next middleware since they work in sequence
  }
};

module.exports = throttlingMiddleware;
