const redis = require('../redis/r_client');
const WINDOW_SIZE = 60000;
const MAX_REQUESTS = 5;

const slidingWindowRateLimiter = async (req, res, next) => {
  const userKey = req.user?.username ;
  const key = `rate_limiter:${userKey}`;
  const now = Date.now();
  const windowStart = now - WINDOW_SIZE;

  try {
    // Remove timestamps older than the window
    await redis.zremrangebyscore(key, 0, windowStart);

    // Count current requests in window
    const requestCount = await redis.zcard(key);

    if (requestCount >= MAX_REQUESTS) {
      return res.status(429).json({ error: 'Too many requests. Rate limit reached' });
    }
    // Add current request with timestamp
    await redis.zadd(key, now, `${now}-${Math.random()}`); // use unique member
    await redis.pexpire(key, WINDOW_SIZE); // expire the key after the window has passed

    next();
  } catch (err) {
    console.error('[SlidingWindowLimiter] Redis error:', err);
    res.status(500).json({ error: 'Internal server error' }); // this is the last middleware in the chain so we can just return a result
  }
};

module.exports = slidingWindowRateLimiter;
