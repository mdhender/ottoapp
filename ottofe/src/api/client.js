/**
 * API client for communicating with the backend
 */

const API_BASE_URL = '/api';

// Helper for making fetch requests with appropriate headers
async function fetchJSON(endpoint, options = {}) {
  const headers = {
    'Content-Type': 'application/json',
    ...options.headers,
  };

  const response = await fetch(`${API_BASE_URL}${endpoint}`, {
    ...options,
    headers,
  });

  if (!response.ok) {
    const error = await response.json().catch(() => ({
      message: response.statusText,
    }));
    throw new Error(error.message || 'API request failed');
  }

  return response.json();
}

// API client with methods for different endpoints
const apiClient = {
  // Auth methods
  login: (credentials) => fetchJSON('/auth/login', {
    method: 'POST',
    body: JSON.stringify(credentials),
  }),
  logout: () => fetchJSON('/auth/logout', { method: 'POST' }),
  
  // Example data methods
  getData: () => fetchJSON('/data'),
};

export default apiClient;