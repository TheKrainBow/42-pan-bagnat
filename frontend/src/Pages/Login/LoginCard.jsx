// src/Components/LoginCard.jsx
import React, { useState, useRef } from "react";
import "./LoginCard.css";
import Button from "Global/Button/Button";
import Field from "Global/Field/Field";
import { toast } from "react-toastify";

export default function LoginCard({ onLogin }) {
  const [email, setEmail] = useState("");
  const emailFieldRef = useRef(null);

  const handleEmailSubmit = () => {
    const emailOk = emailFieldRef.current?.isValid(true);
    if (!emailOk) {
      emailFieldRef.current?.triggerShake();
      return;
    }
    toast.info("Email/password login isn't enabled yet. Please keep using OAuth.");
  };

  return (
    <div className="login-card">
      <span className="card-glow" aria-hidden />
      <div className="card-body">
        <div className="card-meta">
          <h1>Sign in</h1>
          {/* <span className="card-pill">ADM</span> */}
          <p className="card-subtitle">
            Bienvenue sur Pan Bagnat
          </p>
        </div>

        <div className="oauth-section">
          <div className="oauth-button">
            <Button
              label="Sign in with 42 OAuth"
              icon="/icons/42.svg"
              color="black"
              onClick={onLogin}
            />
          </div>
        </div>

        <div className="card-divider">
          <span>or</span>
        </div>

        <div className="credential-form">
          <Field
            ref={emailFieldRef}
            label="Email"
            type="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            placeholder="login@student.42nice.fr"
            required
          />
          {/* <Field
            ref={passwordFieldRef}
            label="Password"
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            placeholder="••••••••"
            required
          /> */}
          <div className="card-actions">
            {/* <button type="button" className="link-btn" onClick={handleNeedHelp}>
              Forgot password?
            </button> */}
            <div className="email-button">
              <Button
                label="Continue with email"
                color="green"
                onClick={handleEmailSubmit}
              />
            </div>
          </div>
        </div>

      </div>
    </div>
  );
}
