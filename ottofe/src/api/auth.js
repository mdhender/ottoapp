// Authentication API functions

const API_URL = '/api';

/**
 * Login with username and password
 * @param {string} email - User email
 * @param {string} password - User password
 * @returns {Promise} - Response with token and user info
 */
export const login = async (email, password) => {
  const response = await fetch(`${API_URL}/auth/login`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ email, password }),
  });

  if (!response.ok) {
    try {
      const error = await response.json();
      throw new Error(error.message || `Login failed with status: ${response.status}`);
    } catch (parseError) {
      throw new Error(`Login failed with status: ${response.status}`);
    }
  }
  
  return response.json();
};

/**
 * Get current user info
 * @param {string} token - JWT token
 * @returns {Promise} - Response with user info
 */
export const getCurrentUser = async (token) => {
  const response = await fetch(`${API_URL}/auth/user`, {
    method: 'GET',
    headers: {
      'Authorization': `Bearer ${token}`,
    },
  });

  if (!response.ok) {
    throw new Error('Failed to get user info');
  }
  
  return response.json();
};