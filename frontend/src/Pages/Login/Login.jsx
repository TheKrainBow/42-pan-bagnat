// src/Pages/Login.jsx
import React from "react";
import "./Login.css";

export default function LoginPage() {
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
