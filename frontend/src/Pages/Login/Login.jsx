// src/Pages/Login.jsx
import React, { useEffect } from "react";
import { fetchWithAuth } from "Global/utils/Auth";
import "./Login.css";

export default function LoginPage() {
  useEffect(() => {
    (async () => {
      // try to hit an auth‐protected endpoint; fetchWithAuth returns null on 401/403
      const res = await fetch("/api/v1/users/me");
      if (res.status === 200) {
        // we got a 200 back → already logged in
        window.location.href = "/modules";
      }
    })();
  }, []); // empty deps → run once on mount

  const handleLogin = () => {
    window.location.href = "/auth/42/login";
  };

  return (
    <div className="login-page">
      <h1>Pan Bagnat</h1>
      <p>Sign in to continue</p>
      <button className="login-42-btn" onClick={handleLogin}>
        🔐 Sign in with 42 Intranet
      </button>
    </div>
  );
}
