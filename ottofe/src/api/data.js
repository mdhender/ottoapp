// Data API functions

const API_URL = '/api';

/**
 * Get user data info
 * @param {string} token - JWT token
 * @returns {Promise} - Response with user data path
 */
export const getUserData = async (token) => {
  const response = await fetch(`${API_URL}/data`, {
    method: 'GET',
    headers: {
      'Authorization': `Bearer ${token}`,
    },
  });

  if (!response.ok) {
    throw new Error('Failed to get user data');
  }
  
  return response.json();
};

/**
 * Get turn data
 * @param {string} token - JWT token
 * @param {number} year - Turn year
 * @param {number} month - Turn month
 * @returns {Promise} - Response with turn data info
 */
export const getTurnData = async (token, year, month) => {
  const response = await fetch(`${API_URL}/data/turn?year=${year}&month=${month}`, {
    method: 'GET',
    headers: {
      'Authorization': `Bearer ${token}`,
    },
  });

  if (!response.ok) {
    throw new Error('Failed to get turn data');
  }
  
  return response.json();
};