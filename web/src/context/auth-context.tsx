import {
  createContext,
  useCallback,
  useContext,
  useMemo,
  useState,
  type ReactNode,
} from "react";
import {
  setTokens as setApiTokens,
  clearTokens as clearApiTokens,
  getAccessToken,
} from "../lib/api-client";

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export type UserRole = "patient" | "doctor" | "pharmacy_staff" | "admin";

export interface AuthUser {
  id: string;
  email: string;
  role: UserRole;
}

interface AuthContextValue {
  /** Current authenticated user, null if not logged in */
  user: AuthUser | null;
  /** Whether user is authenticated */
  isAuthenticated: boolean;
  /** Set tokens after login/register and decode user from JWT */
  login: (accessToken: string, refreshToken: string) => void;
  /** Clear tokens and user state */
  logout: () => void;
}

// ---------------------------------------------------------------------------
// JWT decode (payload only — tidak verifikasi signature di client)
// ---------------------------------------------------------------------------

function decodeJwtPayload(token: string): AuthUser | null {
  try {
    const parts = token.split(".");
    const payload = parts[1];
    if (!payload) return null;

    const decoded = JSON.parse(atob(payload)) as {
      sub?: string;
      user_id?: string;
      email?: string;
      role?: string;
    };

    const id = decoded.sub ?? decoded.user_id;
    if (!id || !decoded.email || !decoded.role) return null;

    return {
      id,
      email: decoded.email,
      role: decoded.role as UserRole,
    };
  } catch {
    return null;
  }
}

// ---------------------------------------------------------------------------
// Context
// ---------------------------------------------------------------------------

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  // Coba restore dari access token yang masih ada di memori (page belum di-refresh)
  const [user, setUser] = useState<AuthUser | null>(() => {
    const token = getAccessToken();
    return token ? decodeJwtPayload(token) : null;
  });

  const login = useCallback((accessToken: string, refreshToken: string) => {
    setApiTokens(accessToken, refreshToken);
    const decoded = decodeJwtPayload(accessToken);
    setUser(decoded);
  }, []);

  const logout = useCallback(() => {
    clearApiTokens();
    setUser(null);
  }, []);

  const value = useMemo<AuthContextValue>(
    () => ({
      user,
      isAuthenticated: user !== null,
      login,
      logout,
    }),
    [user, login, logout],
  );

  return <AuthContext value={value}>{children}</AuthContext>;
}

export function useAuth(): AuthContextValue {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error("useAuth must be used within an AuthProvider");
  }
  return context;
}
