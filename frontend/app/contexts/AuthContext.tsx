'use client';

import { createContext, useContext, useState, useEffect, ReactNode } from 'react';
import { useRouter } from 'next/navigation';

interface User {
  user_id: string;
  email: string;
  name: string;
  picture?: string;
}

interface AuthContextType {
  user: User | null;
  loading: boolean;
  login: () => void;
  logout: () => void;
  checkAuth: () => Promise<void>;
  handleUnauthorized: () => void;
  authenticatedFetch: (input: RequestInfo | URL, init?: RequestInit) => Promise<Response>;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [loading, setLoading] = useState(true);
  const router = useRouter();

  const handleUnauthorized = () => {
    setUser(null);
    if (typeof window !== 'undefined' && window.location.pathname !== '/login') {
      router.push('/login');
    }
  };

  const checkAuth = async () => {
    try {
      const response = await fetch('/api/auth/user', {
        credentials: 'include',
      });

      if (response.ok) {
        const userData = await response.json();
        setUser(userData);
      } else {
        handleUnauthorized();
      }
    } catch (error) {
      console.error('Failed to check auth:', error);
      handleUnauthorized();
    } finally {
      setLoading(false);
    }
  };

  const login = async () => {
    try {
      const response = await fetch('/api/auth/login/google', {
        credentials: 'include',
      });

      if (response.ok) {
        const data = await response.json();
        // Redirect to Google OAuth URL
        window.location.href = data.url;
      } else {
        console.error('Failed to get login URL');
      }
    } catch (error) {
      console.error('Failed to login:', error);
    }
  };

  const logout = async () => {
    try {
      await fetch('/api/auth/logout', {
        method: 'POST',
        credentials: 'include',
      });
      setUser(null);
    } catch (error) {
      console.error('Failed to logout:', error);
    }
  };

  const authenticatedFetch = async (input: RequestInfo | URL, init?: RequestInit) => {
    const response = await fetch(input, init);
    if (response.status === 401) {
      handleUnauthorized();
    }
    return response;
  };

  useEffect(() => {
    checkAuth();
  }, []);

  return (
    <AuthContext.Provider value={{ user, loading, login, logout, checkAuth, handleUnauthorized, authenticatedFetch }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const context = useContext(AuthContext);
  if (context === undefined) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
}
