import { createContext, useContext, useState, useEffect, useCallback } from 'react';
import { login as apiLogin, getCurrentUser } from '@/api/auth';

const AuthContext = createContext();

export const useAuth = () => useContext(AuthContext);

export function AuthProvider({ children }) {
  const [currentUser, setCurrentUser] = useState(null);
  const [token, setToken] = useState(localStorage.getItem('token'));
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    const initAuth = async () => {
      setLoading(true);
      if (token) {
        console.log('Token found, initializing auth state...');
        try {
          const userData = await getCurrentUser(token);
          console.log('Found user data during initialization:', userData);
          setCurrentUser(userData);
        } catch (err) {
          console.error('Failed to get user data during init:', err);
          logout(); // Token invalid, so logout
        }
      } else {
        console.log('No token found, user is not authenticated');
      }
      setLoading(false);
    };

    initAuth();
  }, [token]);

  const fetchUserData = useCallback(async (authToken) => {
    try {
      const userData = await getCurrentUser(authToken);
      setCurrentUser(userData);
      console.log('User data fetched successfully:', userData);
      return userData;
    } catch (err) {
      console.error('Error fetching user data:', err);
      return null;
    }
  }, []);
  
  const login = async (email, password) => {
    try {
      setError('');
      setLoading(true);
      console.log('Attempting login for:', email);
      
      const data = await apiLogin(email, password);
      console.log('Login API response:', data);
      
      if (data.success && data.token) {
        localStorage.setItem('token', data.token);
        setToken(data.token);
        
        // Immediately fetch user data after successful login
        const userData = await fetchUserData(data.token);
        if (userData) {
          console.log('Login complete with user data');
        } else {
          console.warn('Login succeeded but user data fetch failed');
        }
        
        return true;
      } else {
        setError(data.message || 'Login failed');
        return false;
      }
    } catch (err) {
      console.error('Login error:', err);
      setError(err.message || 'Login failed');
      return false;
    } finally {
      setLoading(false);
    }
  };

  const logout = () => {
    localStorage.removeItem('token');
    setToken(null);
    setCurrentUser(null);
  };

  const value = {
    currentUser,
    token,
    loading,
    error,
    login,
    logout,
    isAuthenticated: !!currentUser,
  };

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}