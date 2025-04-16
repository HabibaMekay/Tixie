const Redis = require('ioredis');

const redis = new Redis(process.env.REDIS_URL);

redis.on('connect', () => {
  console.log('[Redis] Connected successfully');
});

redis.on('error', (err) => {
  console.error('[Redis] Connection error:', err);
});

module.exports = redis;
