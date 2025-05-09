async function retryWithBackoff(fn, options = {}) {
    const maxRetries = options.maxRetries || 5;
    const initialDelay = options.initialDelay || 3000;
    const maxDelay = options.maxDelay || 20000;
    const shouldRetry = options.shouldRetry || (() => true);
    
    let attempt = 0;
    let lastError;
  
    while (attempt <= maxRetries) {
      try {
        return await fn();
      } catch (error) {
        lastError = error;
        attempt++;
        
        // If we've reached max retries or shouldn't retry this error anymore
        if (attempt > maxRetries || !shouldRetry(error)) {
          throw error;
        }
        
        // Calculate delay with exponential backoff and jitter
        const delay = Math.min(
          maxDelay, 
          initialDelay * Math.pow(2, attempt - 1) * (0.5 + Math.random() * 0.5)
        );
        
        await new Promise(resolve => setTimeout(resolve, delay));
      }
    }
    
    throw lastError;
  }
  
  module.exports = {retryWithBackoff};