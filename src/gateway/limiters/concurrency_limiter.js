// src/gateway/limiters/concurrencyLimiter.js
const redis = require('../redis/r_client');
// this is a tighter limiter than rate limiter as it checks how many requests are being processed at the same time from a single user
const MAX_CONCURRENT = 3;
const EXPIRATION = 15000; // also for testiing purposes

const concurrencyLimiter= async (req, res, next) => {
  const key = `concurrency:${req.ip}`;

  try {
    const active = await redis.incr(key);

    if (active === 1) {
      await redis.pexpire(key, EXPIRATION);
    }

    if (active > MAX_CONCURRENT) {
      await redis.decr(key); // Rollback the increment
      console.log(`[Concurrency] Too many concurrent requests from ${req.ip}: ${active}`);
      return res.status(429).json({ error: 'Too many concurrent requests' });
    }

    // Decrement when request ends
    res.on('finish', async () => { // this is an emitted flag 
      const remaining = await redis.decr(key);
      if (remaining <= 0) await redis.del(key);
    });

    next();
  } catch (err) {
    console.error('[Concurrency] Error:', err);
    next(); // Pass the error to the next middleware
  }
};

module.exports = concurrencyLimiter;