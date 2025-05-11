// Admin API functions

const API_URL = '/api';

/**
 * Toggle route logging (admin only)
 * @param {string} token - JWT token
 * @returns {Promise} - Response with logging status
 */
export const toggleRouteLogging = async (token) => {
  const response = await fetch(`${API_URL}/admin/debug/log-all-routes`, {
    method: 'POST',
    headers: {
      'Authorization': `Bearer ${token}`,
      'Content-Type': 'application/json',
    },
  });

  if (!response.ok) {
    if (response.status === 403) {
      throw new Error('Admin access required');
    }
    throw new Error('Failed to toggle route logging');
  }
  
  return response.json();
};