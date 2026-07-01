// src/context/AuthContext.tsx
import { createContext, useContext, useEffect, useState, type ReactNode } from 'react';
import { useNavigate } from 'react-router-dom';
interface PlayerPageData {
  username: string;
  steamLinked: boolean;
  hasMatches: boolean;
  steamId?: number;
  playerName?: string;
  teamName?: string;
  profilePic?: string;
  profilePicfull?: string;
  // these get added later once you build out the stats endpoint
}

interface AuthContextValue {
  user: PlayerPageData | null;
  loading: boolean;
  refreshUser: () => Promise<void>;
}

const AuthContext = createContext<AuthContextValue | undefined>(undefined);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<PlayerPageData | null>(null);
  const [loading, setLoading] = useState(true);
  const navigate = useNavigate()
  const fetchUser = async () => {
  try {
        const res = await fetch('http://localhost:4000/api/player/me', {
        credentials: 'include',
        });

        if (!res.ok) {
            if (res.status === 401) {
                navigate('/')
            }
        setUser(null);
        return; // don't attempt to parse a non-JSON error body
        }

        const data = await res.json();
        setUser(data);
    } catch {
        setUser(null);
    } finally {
        setLoading(false);
    }
  };

  useEffect(() => {
    fetchUser();
  }, []);

  return (
    <AuthContext.Provider value={{ user, loading, refreshUser: fetchUser }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error('useAuth must be used within AuthProvider');
  return ctx;
}