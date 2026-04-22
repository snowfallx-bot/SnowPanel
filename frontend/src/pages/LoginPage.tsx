import { FormEvent, useEffect, useState } from "react";
import { useLocation, useNavigate } from "react-router-dom";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { login } from "@/api/auth";
import { AUTH_REDIRECT_MESSAGE_KEY } from "@/lib/http";
import { useAuthStore } from "@/store/auth-store";

export function LoginPage() {
  const navigate = useNavigate();
  const location = useLocation();
  const setAuth = useAuthStore((state) => state.setAuth);

  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  const locationState = (location.state as { from?: string; message?: string } | null) ?? null;
  const from = locationState?.from ?? "/dashboard";

  useEffect(() => {
    const redirectMessage = locationState?.message;
    const storedMessage =
      typeof window !== "undefined" ? window.sessionStorage.getItem(AUTH_REDIRECT_MESSAGE_KEY) : null;

    if (storedMessage) {
      window.sessionStorage.removeItem(AUTH_REDIRECT_MESSAGE_KEY);
    }

    const nextError = redirectMessage || storedMessage;
    if (nextError) {
      setError(nextError);
    }
  }, [locationState?.message]);

  async function onSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError("");
    setLoading(true);
    try {
      const result = await login({ username, password });
      setAuth(result.access_token, result.user);
      navigate(from, { replace: true });
    } catch (err) {
      setError(err instanceof Error ? err.message : "Login failed");
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-gradient-to-br from-panel-900 via-panel-800 to-slate-900 px-4">
      <Card className="w-full max-w-md border-panel-700 bg-white/95">
        <CardHeader>
          <CardTitle>Sign In To SnowPanel</CardTitle>
        </CardHeader>
        <CardContent>
          <form className="space-y-4" onSubmit={onSubmit}>
            <div className="space-y-1">
              <label className="text-sm font-medium text-slate-700">Username</label>
              <Input
                placeholder="Enter your username"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
              />
            </div>
            <div className="space-y-1">
              <label className="text-sm font-medium text-slate-700">Password</label>
              <Input
                type="password"
                placeholder="Enter your password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
              />
            </div>
            {error && <p className="text-sm text-rose-600">{error}</p>}
            <Button className="w-full" disabled={loading} type="submit">
              {loading ? "Signing in..." : "Sign In"}
            </Button>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}
