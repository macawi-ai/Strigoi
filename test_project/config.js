// Example configuration file with security issues

const config = {
  // Hardcoded API key - security issue!
  apiKey: 'sk_test_4eC39HqLyjWDarjtT1zdp7dc',
  
  // AWS credentials - another security issue!
  aws: {
    accessKey: 'AKIAIOSFODNN7EXAMPLE',
    secretKey: 'wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY'
  },
  
  // External API endpoints
  apiEndpoints: {
    payment: 'https://api.stripe.com/v1/charges',
    analytics: 'https://api.analytics.example.com/v2/events',
    auth: 'https://auth.example.com/oauth/token'
  },
  
  // Debug mode - should not be enabled in production
  debug: true,
  debugEndpoint: '/debug/status',
  
  // Verbose error handling
  errorHandler: function(err) {
    console.error('Stack trace:', err.stack);
    return {
      error: err.message,
      stack: err.stack,
      internal: err.internal
    };
  }
};

module.exports = config;