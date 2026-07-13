// src/Components/LoginCard.jsx
import React, { useState, useRef } from "react";
import "./LoginCard.css";
import Button from "Global/Button/Button";
import Field from "Global/Field/Field";
import { toast } from "react-toastify";

export default function LoginCard({ onLogin, onMagicLink }) {
  const [email, setEmail] = useState("");
  const [sendingEmail, setSendingEmail] = useState(false);
  const emailFieldRef = useRef(null);

  const handleEmailSubmit = async () => {
    const emailOk = emailFieldRef.current?.isValid(true);
    if (!emailOk) {
      emailFieldRef.current?.triggerShake();
      return;
    }
    if (sendingEmail) {
      return;
    }
    setSendingEmail(true);
    try {
      await onMagicLink?.(email);
      toast.info("If an account exists for this email, a sign-in link has been sent.");
    } catch (err) {
      toast.error(err.message || "Unable to send sign-in link.");
    } finally {
      setSendingEmail(false);
    }
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
                label={sendingEmail ? "Sending..." : "Continue with email"}
                color="green"
                onClick={handleEmailSubmit}
                disabled={sendingEmail}
              />
            </div>
          </div>
        </div>

      </div>
    </div>
  );
}
